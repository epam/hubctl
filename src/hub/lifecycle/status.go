package lifecycle

import (
	"fmt"

	"hub/manifest"
	"hub/state"
	"hub/util"
)

func calculateStackStatus(stackManifest *manifest.Manifest, stateManifest *state.StateManifest, verb string) (string, string) {
	statuses := make(map[string][]string)
	mandatoryComponents := 0
	for _, componentName := range stackManifest.Lifecycle.Order {
		if !optionalComponent(&stackManifest.Lifecycle, componentName) {
			mandatoryComponents++
			componentState, exist := stateManifest.Components[componentName]
			if exist {
				componentStatus := componentState.Status
				if componentStatus == "" { // compat
					componentStatus = "deployed"
				}
				util.AppendMapList(statuses, componentStatus, componentName)
			}
		}
	}
	if mandatoryComponents == 0 {
		return verb + "ed", ""
	}
	for _, candidate := range []string{"deployed", "undeployed"} {
		components, exist := statuses[candidate]
		if exist && len(components) == mandatoryComponents {
			return candidate, ""
		}
	}

	// should force-undeployed stack become undeployed?
	return "incomplete",
		fmt.Sprintf("Mandatory components state:\n%s",
			util.SprintDeps(statuses))
}
