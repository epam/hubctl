package lifecycle

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/manifest"
	"github.com/agilestacks/hub/cmd/hub/parameters"
	"github.com/agilestacks/hub/cmd/hub/state"
	"github.com/agilestacks/hub/cmd/hub/util"
)

// check there is a perfect match between components in Stack manifest and accompanied Components manifests
func checkComponentsManifests(components []manifest.ComponentRef, componentsManifests []manifest.Manifest) {
	refs := manifest.ComponentsNamesFromRefs(components)
	docs := manifest.ComponentsNamesFromManifests(componentsManifests)
	sort.Strings(refs)
	sort.Strings(docs)
	if !util.Equal(refs, docs) {
		log.Fatalf("Components in stack manifest: %s;\n\tdoes not match components manifest documents: %s\n\t%s",
			strings.Join(refs, ", "), strings.Join(docs, ", "),
			"(do you have YAML file(s) with \\r\\n line endings?)")
	}
}

// check sources is an accessible local directory
func checkComponentsSourcesExist(components []manifest.ComponentRef, stackBaseDir, componentsBaseDir string) {
	for _, component := range components {
		compName := manifest.ComponentQualifiedNameFromRef(&component)
		dir := manifest.ComponentSourceDirFromRef(&component, stackBaseDir, componentsBaseDir)
		info, err := os.Stat(dir)
		if err != nil {
			log.Fatalf("`%s` source directory for component `%s` not found: %v", dir, compName, err)
		}
		if !info.IsDir() {
			log.Fatalf("`%s` source for component `%s` is not a directory", dir, compName)
		}
	}
}

// check manifest.Lifecycle verbs
func checkLifecycleVerbs(components []manifest.ComponentRef, componentsManifests []manifest.Manifest,
	verbs []string, stackBaseDir, componentsBaseDir string) {

	optionalVerbs := []string{"backup"}
	for _, component := range components {
		dir := manifest.ComponentSourceDirFromRef(&component, stackBaseDir, componentsBaseDir)
		for _, verb := range verbs {
			if util.Contains(optionalVerbs, verb) {
				continue
			}
			impl, err := probeImplementation(dir, verb)
			if !impl {
				msg := fmt.Sprintf("`%s` component in `%s` has no `%s` implementation: %v",
					manifest.ComponentQualifiedNameFromRef(&component), dir, verb, err)
				manifest := manifest.ComponentManifestByRef(componentsManifests, &component)
				if manifest.Lifecycle.Bare == "allow" {
					if config.Debug {
						log.Print(msg)
					}
				} else {
					log.Fatalf("%s;\n\tTry setting `lifecycle.bare: allow` in component's manifest if it's your intention", msg)
				}
			}
		}
	}
}

func checkLifecycleOrder(components []manifest.ComponentRef, lifecycle manifest.Lifecycle) {
	refs := manifest.ComponentsNamesFromRefs(components)
	sorted := make([]string, len(refs))
	copy(sorted, refs)
	order := make([]string, len(lifecycle.Order))
	copy(order, lifecycle.Order)
	sort.Strings(sorted)
	sort.Strings(order)
	if !util.Equal(sorted, order) {
		lifecycleOrder := "(not specified)"
		if len(lifecycle.Order) > 0 {
			lifecycleOrder = strings.Join(lifecycle.Order, ", ")
		}
		log.Fatalf("Components: %s;\n\tdoes not match `lifecycle.order`: %s",
			strings.Join(refs, ", "), lifecycleOrder)
	}

	if len(lifecycle.Mandatory) > 0 && len(lifecycle.Optional) > 0 {
		util.Warn("Both\n\tlifecycle.mandatory = %v and\n\tlifecycle.optional = %v are set",
			lifecycle.Mandatory, lifecycle.Optional)
	}
	if !util.ContainsAll(sorted, lifecycle.Mandatory) {
		log.Fatalf("lifecycle.mandatory = %v does not match component list:\n\t%v", lifecycle.Mandatory, sorted)
	}
	if !util.ContainsAll(sorted, lifecycle.Optional) {
		log.Fatalf("lifecycle.optional = %v does not match component list:\n\t%v", lifecycle.Optional, sorted)
	}
}

func checkVerbs(componentName string, availableVerbs []string, verb string) {
	for _, supported := range availableVerbs {
		if verb == supported {
			return
		}
	}
	available := "(none)"
	if len(availableVerbs) > 0 {
		available = strings.Join(availableVerbs, ", ")
	}
	util.MaybeFatalf("Verb `%s` is not supported by component `%s`, available verbs: %s",
		verb, componentName, available)
}

func checkLifecycleRequires(components []manifest.ComponentRef, requires manifest.RequiresTuning) {
	refs := manifest.ComponentsNamesFromRefs(components)
	sorted := make([]string, len(refs))
	copy(sorted, refs)
	sort.Strings(sorted)
	for _, req := range requires.Optional {
		i := strings.Index(req, ":")
		if i > 0 && i < len(req)-1 {
			component := req[i+1:]
			if !util.Contains(refs, component) {
				log.Fatalf("lifecycle.requires.optional = %s component `%s` does not match component list:\n\t%v",
					req, component, sorted)
			}
		}
	}
}

func checkComponentsDepends(components []manifest.ComponentRef, order []string) {
	for _, component := range components {
		if len(component.Depends) > 0 && !util.ContainsAll(order, component.Depends) {
			var missing []string
			for _, dep := range component.Depends {
				if !util.Contains(order, dep) {
					missing = append(missing, dep)
				}
			}
			log.Fatalf("Component `%s` `depends: %v` does not match component list",
				manifest.ComponentQualifiedNameFromRef(&component), missing)
		}
	}
}

func checkStateMatch(state *state.StateManifest, elaborate *manifest.Manifest, stackParameters parameters.LockedParameters) {
	errs := make([]error, 0)

	if state.Meta.Name != "" && state.Meta.Name != elaborate.Meta.Name {
		errs = append(errs, fmt.Errorf("State meta.name = `%s` does not match elaborate meta.name = `%s`",
			state.Meta.Name, elaborate.Meta.Name))
	}

	parametersToMatch := []string{"cloud.kind", "cloud.region", "terraform.bucket.name", "dns.name", "dns.domain"}
	for _, name := range parametersToMatch {
		if stackParam, exist := stackParameters[name]; exist {
			for _, stateParam := range state.StackParameters {
				if stateParam.QName() == name {
					if util.String(stateParam.Value) != util.String(stackParam.Value) {
						errs = append(errs, fmt.Errorf("Parameter `%s` state value `%v` does not match stack parameter value `%v`",
							name, stateParam.Value, stackParam.Value))
					}
					break
				}
			}
		}
	}

	if len(state.Components) > 2 {
		stateComponents := make([]string, 0, len(state.Components))
		stackComponents := manifest.ComponentsNamesFromRefs(elaborate.Components)
		found := 0
		for stateComponent := range state.Components {
			stateComponents = append(stateComponents, stateComponent)
			if util.Contains(stackComponents, stateComponent) {
				found++
			}
		}
		if found < len(stateComponents)/2 {
			sort.Strings(stackComponents)
			sort.Strings(stateComponents)
			errs = append(errs, fmt.Errorf("State components:\n\t\t%s\n\t\t\tdoes not match stack components:\n\t\t%s",
				strings.Join(stateComponents, ", "), strings.Join(stackComponents, ", ")))
		}
	}

	if len(errs) > 0 {
		msg := fmt.Sprintf("State file does not match deployment manifest (elaborate):\n\t%s", util.Errors("\n\t", errs...))
		if config.Force {
			util.Warn("%s", msg)
		} else {
			log.Fatal(msg)
		}
	}
}
