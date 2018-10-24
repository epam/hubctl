package lifecycle

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"hub/api"
	"hub/manifest"
	"hub/parameters"
	"hub/util"
)

func printStartBlurb(request *Request, manifestFilename string, stackManifest *manifest.Manifest) {
	if len(request.Components) == 0 {
		startAt := ""
		if request.OffsetComponent != "" {
			startAt = fmt.Sprintf(" starting with component %s", request.OffsetComponent)
		}
		stopAt := ""
		if request.LimitComponent != "" {
			stopAt = fmt.Sprintf(" stoping at component %s", request.LimitComponent)
		}
		log.Printf("%s %s (%s) with components %s%s%s", request.Verb, stackManifest.Meta.Name, manifestFilename,
			strings.Join(manifest.ComponentsNamesFromRefs(stackManifest.Components), ", "), startAt, stopAt)
	} else {
		log.Printf("%s %s (%s) component %s", request.Verb, stackManifest.Meta.Name, manifestFilename,
			strings.Join(request.Components, ", "))
	}

}

func printEndBlurb(request *Request, stackManifest *manifest.Manifest) {
	component := ""
	if len(request.Components) > 0 {
		component = fmt.Sprintf(" component(s) %s", strings.Join(request.Components, ", "))
	} else if request.OffsetComponent != "" {
		component = fmt.Sprintf(" starting with %s component", request.OffsetComponent)
	}
	log.Printf("Completed %s on %s%s", request.Verb, stackManifest.Meta.Name, component)
}

func printBackupStartBlurb(request *Request, bundles []string) {
	state := ""
	if len(request.StateFilenames) > 0 {
		state = fmt.Sprintf(" with %v state", request.StateFilenames)
	}
	comp := ""
	if request.Component != "" {
		comp = fmt.Sprintf(", just for `%s` component", request.Component)
	}
	bundle := ""
	if len(bundles) > 0 {
		bundle = fmt.Sprintf(", saving bundle into %v", bundles)
	}
	log.Printf("Creating backup for %v%s%s%s", request.ManifestFilenames, state, comp, bundle)
}

func printBackupEndBlurb(request *Request, stackManifest *manifest.Manifest) {
	component := ""
	if request.Component != "" {
		component = fmt.Sprintf(" component %s", request.Component)
	}
	log.Printf("Completed %s on %s%s", request.Verb, stackManifest.Meta.Name, component)
}

func printEnvironment(env []string) {
	for _, v := range env {
		log.Printf("\t%s", v)
	}
}

func printTemplates(templates []TemplateRef) {
	for _, template := range templates {
		log.Printf("\t%s (%s)", template.Filename, template.Kind)
	}
}

func formatStdout(what string, stream bytes.Buffer) string {
	if stream.Len() > 0 {
		str := stream.String()
		nl := "\n"
		if strings.HasSuffix(str, "\n") {
			nl = ""
		}
		return fmt.Sprintf("\n--- %s:\n%s%s---\n", what, str, nl)
	}
	return ""
}

func printExpandedOutputs(outputs []parameters.ExpandedOutput) {
	if len(outputs) > 0 {
		log.Print("Stack outputs:")
		for _, output := range outputs {
			brief := ""
			if output.Brief != "" {
				brief = fmt.Sprintf("[%s] ", output.Brief)
			}
			if strings.Contains(output.Value, "\n") {
				maybeNl := "\n"
				if strings.HasSuffix(output.Value, "\n") {
					maybeNl = ""
				}
				log.Printf("\t%s%s ~>\n%s%s~~", brief, output.Name, output.Value, maybeNl)
			} else {
				log.Printf("\t%s%s => `%s`", brief, output.Name, output.Value)
			}
		}
	}
}

func printStackInstancePatch(patch api.StackInstancePatch) {
	if len(patch.Outputs) > 0 {
		log.Print("Outputs to API:")
		for _, output := range patch.Outputs {
			brief := ""
			if output.Brief != "" {
				brief = fmt.Sprintf("[%s] ", output.Brief)
			}
			component := ""
			if output.Component != "" {
				component = fmt.Sprintf("%s:", output.Component)
			}
			log.Printf("\t%s%s%s => `%v`", brief, component, output.Name, output.Value)
		}
	}
	if len(patch.Provides) > 0 {
		log.Print("Provides to API:")
		util.PrintMap2(patch.Provides)

	}
}
