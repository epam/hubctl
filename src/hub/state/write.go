package state

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"hub/config"
	"hub/manifest"
	"hub/parameters"
	"hub/storage"
	"hub/util"
)

const (
	stateUpdateSettleInterval = time.Duration(2 * time.Second)
)

func InitWriter(stateFiles *storage.Files) func(interface{}) {
	ch := make(chan interface{}, 2)
	done := make(chan struct{})
	ticker := time.NewTicker(1 * time.Second)
	go writer(ch, done, ticker.C, stateFiles)
	update := func(v interface{}) {
		ch <- v
		if cmd, ok := v.(string); ok && cmd == "done" {
			ticker.Stop()
		}
	}
	util.AtDone(func() <-chan struct{} {
		update("done")
		return done
	})
	return update
}

func writer(ch <-chan interface{}, done chan<- struct{}, ticker <-chan time.Time, files *storage.Files) {
	pending := false
	var updated time.Time
	var state *StateManifest

	maybeWrite := func() {
		if pending && state != nil {
			WriteState(state, files)
			pending = false
		}
	}

	atExit := func() {
		done <- struct{}{}
	}
	defer atExit()

	for {
		select {
		case m := <-ch:
			switch v := m.(type) {
			case string:
				switch v {
				case "sync":
					maybeWrite()
				case "done":
					maybeWrite()
					return
				default:
					log.Fatalf("Unknown command `%s` received by state writer", v)
				}

			case *StateManifest:
				state = v
				pending = true
				updated = time.Now()

			default:
				log.Fatalf("Unknown type received by state writer: %+v", m)
			}

		case now := <-ticker:
			if updated.Add(stateUpdateSettleInterval).Before(now) {
				maybeWrite()
			}
		}
	}
}

func UpdateState(manifest *StateManifest, componentName string,
	stackParameters parameters.LockedParameters, componentParameters []parameters.LockedParameter,
	rawOutputs parameters.RawOutputs, outputs parameters.CapturedOutputs,
	requestedOutputs []manifest.Output,
	provides map[string][]string,
	final bool) *StateManifest {

	now := time.Now()

	manifest = maybeInitState(manifest)
	componentState := maybeInitComponentState(manifest, componentName)
	componentState.Timestamp = now
	componentState.Parameters = componentParameters
	componentState.CapturedOutputs = parameters.CapturedOutputsToList(outputs)
	if len(rawOutputs) > 0 {
		componentState.RawOutputs = parameters.RawOutputsToList(rawOutputs)
	}

	manifest.Timestamp = now
	if final {
		manifest.CapturedOutputs = componentState.CapturedOutputs
	}
	manifest.StackParameters = parameters.LockedParametersToList(stackParameters)
	expandedOutputs := parameters.ExpandRequestedOutputs(stackParameters, outputs, requestedOutputs, final)
	manifest.StackOutputs = mergeExpandedOutputs(manifest.StackOutputs, expandedOutputs, requestedOutputs)
	manifest.Provides = provides

	return manifest
}

func UpdateStackStatus(manifest *StateManifest, status, message string) *StateManifest {
	manifest = maybeInitState(manifest)
	if status != "" {
		manifest.Timestamp = time.Now()
		manifest.Status = status
		manifest.Message = message
		if config.Debug {
			log.Printf("State stack status: %s", status)
			if message != "" && config.Trace {
				log.Printf("State stack message: %s", message)
			}
		}
	}
	return manifest
}

func UpdateComponentStartTimestamp(manifest *StateManifest, name string) *StateManifest {
	manifest = maybeInitState(manifest)
	componentState := maybeInitComponentState(manifest, name)
	componentState.Timestamps.Start = time.Now()
	return manifest
}

func UpdateComponentStatus(manifest *StateManifest, name, version, status, message string) *StateManifest {
	manifest = maybeInitState(manifest)
	if name != "" && status != "" {
		componentState := maybeInitComponentState(manifest, name)
		componentState.Version = version
		now := time.Now()
		componentState.Timestamp = now
		componentState.Timestamps.End = now
		componentState.Status = status
		componentState.Message = message
		if config.Debug {
			log.Printf("State component `%s` status: %s", name, status)
			if message != "" && config.Trace {
				log.Printf("State component `%s` message: %s", name, message)
			}
		}
	}
	return manifest
}

func UpdateOperation(manifest *StateManifest, id, operation, status string, options map[string]interface{}) *StateManifest {
	found := -1
	ops := manifest.Operations
	for i, op := range ops {
		if op.Id == id {
			found = i
			break
		}
	}
	op := LifecycleOperation{
		Id:        id,
		Operation: operation,
		Timestamp: time.Now(),
		Status:    status,
		Options:   options,
		Initiator: os.Getenv("USER"),
	}
	if found >= 0 {
		if op.Options == nil {
			op.Options = ops[found].Options
		}
		op.Logs = ops[found].Logs
		op.Phases = ops[found].Phases
		ops[found] = op
	} else {
		manifest.Operations = append(ops, op)
	}
	if config.Debug {
		log.Printf("State lifecycle operation `%s` status: %s", op.Operation, op.Status)
	}
	return manifest
}

func findOperation(manifest *StateManifest, id string) int {
	found := -1
	ops := manifest.Operations
	for i, op := range ops {
		if op.Id == id {
			found = i
			break
		}
	}
	if found == -1 {
		util.Warn("Internal state error: no lifecycle operation with id `%s` found", id)
	}
	return found
}

func AppendOperationLog(manifest *StateManifest, id, logAdd string) *StateManifest {
	foundOp := findOperation(manifest, id)
	if foundOp == -1 {
		return manifest
	}
	op := manifest.Operations[foundOp]

	sep := "\n"
	if op.Logs == "" || strings.HasSuffix(op.Logs, sep) {
		sep = ""
	}
	manifest.Operations[foundOp].Logs = op.Logs + sep + logAdd

	if config.Debug {
		log.Printf("State lifecycle operation `%s` log appended: %s", op.Operation, util.Wrap(logAdd))
	}
	return manifest
}

func UpdatePhase(manifest *StateManifest, opId, name, status string) *StateManifest {
	foundOp := findOperation(manifest, opId)
	if foundOp == -1 {
		return manifest
	}
	op := manifest.Operations[foundOp]

	foundPhase := -1
	phases := op.Phases
	for i, phase := range phases {
		if phase.Phase == name {
			foundPhase = i
			break
		}
	}
	phase := LifecyclePhase{Phase: name, Status: status}
	if foundPhase >= 0 {
		phases[foundPhase] = phase
	} else {
		manifest.Operations[foundOp].Phases = append(phases, phase)
	}
	if config.Debug {
		log.Printf("State lifecycle phase `%s` status: %s", phase.Phase, phase.Status)
	}
	return manifest
}

func WriteState(manifest *StateManifest, stateFiles *storage.Files) {
	manifest.Version = 1
	manifest.Kind = "state"

	yamlBytes, err := yaml.Marshal(manifest)
	if err != nil {
		log.Fatalf("Unable to marshal state into YAML: %v", err)
	}

	written, errs := storage.Write(yamlBytes, stateFiles)
	if len(errs) > 0 {
		msg := fmt.Sprintf("Unable to write state: %s", util.Errors2(errs...))
		if !written {
			log.Fatal(msg)
		} else {
			util.Warn("%s", msg)
		}
	}
}

func maybeInitState(manifest *StateManifest) *StateManifest {
	if manifest == nil {
		manifest = &StateManifest{}
	}
	if manifest.Components == nil {
		manifest.Components = make(map[string]*StateStep)
	}
	return manifest
}

func maybeInitComponentState(manifest *StateManifest, componentName string) *StateStep {
	componentState, exist := manifest.Components[componentName]
	if !exist {
		componentState = &StateStep{}
		manifest.Components[componentName] = componentState
	}
	return componentState
}

func mergeExpandedOutputs(prev, curr []parameters.ExpandedOutput, requestedOutputs []manifest.Output) []parameters.ExpandedOutput {
	if len(prev) == 0 {
		return curr
	}
	currNames := make([]string, 0, len(curr))
	for _, c := range curr {
		currNames = append(currNames, c.Name)
	}
	reqNames := make([]string, 0, len(requestedOutputs))
	for _, r := range requestedOutputs {
		reqNames = append(reqNames, r.Name)
	}
	for _, p := range prev {
		if !util.Contains(currNames, p.Name) && util.Contains(reqNames, p.Name) {
			curr = append(curr, p)
		}
	}
	return curr
}
