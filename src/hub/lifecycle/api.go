package lifecycle

import (
	"strings"

	"hub/api"
	"hub/kube"
	"hub/parameters"
	"hub/util"
)

func transformStackOutputsToApi(stackOutputs []parameters.ExpandedOutput) []api.Output {
	outputs := make([]api.Output, 0, len(stackOutputs))
	kubeSecretOutputs := kube.RequiredKubernetesKeysParameters()
	for _, o := range stackOutputs {
		name := o.Name
		component := ""
		if strings.Contains(o.Name, ":") {
			parts := strings.SplitN(o.Name, ":", 2)
			if len(parts) > 1 {
				component = parts[0]
				name = parts[1]
			}
		}

		var value interface{}
		kind := ""
		if strings.HasPrefix(o.Kind, "secret") || util.Contains(kubeSecretOutputs, name) {
			secretKind := guessSecretKind(o.Kind, name)
			value = map[string]string{
				"kind":     secretKind,
				secretKind: o.Value,
			}
			kind = "secret"
		} else {
			value = o.Value
		}

		outputs = append(outputs, api.Output{
			Name:      name,
			Component: component,
			Value:     value,
			Kind:      kind,
			Brief:     o.Brief,
			Messenger: "cli",
		})
	}
	return outputs
}

func guessSecretKind(outputKind, name string) string {
	if strings.Contains(outputKind, "/") {
		parts := strings.SplitN(outputKind, "/", 2)
		if len(parts) > 1 && parts[1] != "" {
			return parts[1]
		}
	}

	kind := "text"
	if strings.HasSuffix(name, ".key") || strings.HasSuffix(name, "Key") {
		kind = "privateKey"
	} else if strings.HasSuffix(name, ".cert") || strings.HasSuffix(name, "Cert") {
		kind = "certificate"
	} else if strings.HasSuffix(name, ".password") || strings.HasSuffix(name, "Password") {
		kind = "password"
	}
	return kind
}
