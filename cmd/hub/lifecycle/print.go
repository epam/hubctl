// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"fmt"
	"log"
	"strings"

	"github.com/agilestacks/hub/cmd/hub/api"
	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/manifest"
	"github.com/agilestacks/hub/cmd/hub/parameters"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var sensitiveCmdArgs = []string{"password", "secret", "key", "token"}

func startStopComponentsBlurb(request *Request, stackManifest *manifest.Manifest) string {
	if len(request.Components) == 0 {
		startAt := ""
		if request.OffsetComponent != "" {
			startAt = fmt.Sprintf("\n\tstarting with component %s", request.OffsetComponent)
		}
		stopAt := ""
		if request.LimitComponent != "" {
			stopAt = fmt.Sprintf("\n\tstopping at component %s", request.LimitComponent)
		}
		return fmt.Sprintf(" with components %s%s%s",
			strings.Join(manifest.ComponentsNamesFromRefs(stackManifest.Components), ", "), startAt, stopAt)
	}
	return fmt.Sprintf(" %s %s", util.Plural(len(request.Components), "component"), strings.Join(request.Components, ", "))
}

func printStartBlurb(request *Request, manifestFilename string, stackManifest *manifest.Manifest) {
	log.Printf("%s %s (%s)%s", request.Verb, stackManifest.Meta.Name, manifestFilename,
		startStopComponentsBlurb(request, stackManifest))
}

func printEndBlurb(request *Request, stackManifest *manifest.Manifest) {
	log.Printf("Completed %s on %s%s", request.Verb, stackManifest.Meta.Name,
		startStopComponentsBlurb(request, stackManifest))
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
		if !config.Trace {
			kv := strings.SplitN(v, "=", 2)
			if len(kv) == 2 && util.LooksLikeSecret(kv[0]) && len(kv[1]) > 0 {
				v = fmt.Sprintf("%s=(masked)", kv[0])
			}
		}
		log.Printf("\t%s", v)
	}
}

func printCmd(cmd []string) {
	if !config.Trace {
		cmd = trimSensitiveCmd(cmd)
	}
	log.Printf("Checking %v", cmd)

}

func trimSensitiveCmd(cmd []string) []string {
	l := len(cmd)
	if l == 0 {
		return cmd
	}

	clean := make([]string, 1, l)
	clean[0] = cmd[0]

	skip := false
	for _, arg := range cmd[1:] {
		if skip {
			arg = "<sensitive>"
			skip = false
		} else {
			arg = strings.ToLower(arg)
			for _, sensitive := range sensitiveCmdArgs {
				if strings.Contains(arg, sensitive) {
					skip = true
				}
			}
		}
		clean = append(clean, arg)
	}
	return clean
}

func printTemplates(templates []TemplateRef) {
	for _, template := range templates {
		log.Printf("\t%s (%s)", template.Filename, template.Kind)
	}
}

func formatBytes(what string, stream []byte) string {
	l := len(stream)
	if l > 0 {
		nl := "\n"
		if stream[l-1] == '\n' {
			nl = ""
		}
		return fmt.Sprintf("\n--- %s:\n%s%s---\n", what, stream, nl)
	}
	return ""
}

func formatStdoutStderr(stdout, stderr []byte) string {
	return fmt.Sprintf("%s%s", formatBytes("stdout", stdout), formatBytes("stderr", stderr))
}

func printStackOutputs(outputs []parameters.ExpandedOutput) {
	if len(outputs) > 0 {
		log.Print("Stack outputs:")
		for _, output := range outputs {
			brief := ""
			if output.Brief != "" {
				brief = fmt.Sprintf(" [%s]", util.HighlightColor(output.Brief))
			}
			value := util.String(output.Value)
			valueMasked := false
			if !config.Trace && strings.HasPrefix(output.Kind, "secret") && len(value) > 0 {
				value = "(masked)"
				valueMasked = true
			}
			if strings.Contains(value, "\n") {
				maybeNl := "\n"
				if strings.HasSuffix(value, "\n") {
					maybeNl = ""
				}
				log.Printf("\t%s%s ~>\n%s%s~~", brief, output.Name, value, maybeNl)
			} else {
				if !valueMasked {
					value = fmt.Sprintf("`%s`", value)
				}
				log.Printf("\t%s%s => %s", output.Name, brief, value)
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
			// this is under Trace, no secret value masking required
			log.Printf("\t%s%s%s => `%v`", brief, component, output.Name, output.Value)
		}
	}
	if len(patch.Provides) > 0 {
		log.Print("Provides to API:")
		util.PrintMap2(patch.Provides)
	}
}
