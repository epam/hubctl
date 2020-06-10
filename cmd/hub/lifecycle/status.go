package lifecycle

import (
	"fmt"

	"github.com/agilestacks/hub/cmd/hub/manifest"
	"github.com/agilestacks/hub/cmd/hub/state"
	"github.com/agilestacks/hub/cmd/hub/util"
)

func calculateStackStatus(stackManifest *manifest.Manifest, stateManifest *state.StateManifest, verb string) (string, string) {
	var statuses map[string][]string
	mandatory := make(map[string][]string)
	optional := make(map[string][]string)
	mandatoryCount := 0
	optionalCount := 0
	deployed := "deployed"
	undeployed := "undeployed"

	for _, componentName := range stackManifest.Lifecycle.Order {
		if optionalComponent(&stackManifest.Lifecycle, componentName) {
			optionalCount++
			statuses = optional
		} else {
			mandatoryCount++
			statuses = mandatory
		}
		componentStatus := undeployed
		if componentState, exist := stateManifest.Components[componentName]; exist {
			componentStatus = componentState.Status
			if componentStatus == "" { // compat
				componentStatus = deployed
			}
		}
		util.AppendMapList(statuses, componentStatus, componentName)
	}

	optionalStatus := undeployed
	if optionalCount > 0 {
		if _, exist := optional[deployed]; exist {
			optionalStatus = deployed
		}
	}

	if mandatoryCount > 0 {
		for _, candidate := range []string{deployed, undeployed} {
			if components, exist := mandatory[candidate]; exist && len(components) == mandatoryCount {
				// if all mandatory components are deployed then the stack status is deployed
				// if all mandatory components are undeployed then the stack status is undeployed only if
				// there are no deployed optional components
				if candidate == deployed || optionalStatus == undeployed {
					return candidate, ""
				}
			}
		}
		// should force-undeployed stack become undeployed?
		return "incomplete",
			fmt.Sprintf("Mandatory components state:\n%s",
				util.SprintDeps(statuses))
	}

	return optionalStatus, ""
}
