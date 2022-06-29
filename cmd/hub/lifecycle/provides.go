// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"log"
	"os"
	"strings"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/kube"
	"github.com/agilestacks/hub/cmd/hub/manifest"
	"github.com/agilestacks/hub/cmd/hub/parameters"
	"github.com/agilestacks/hub/cmd/hub/util"
)

func captureProvides(component *manifest.ComponentRef, stackBaseDir string, componentsBaseDir string, provides []string,
	componentOutputs parameters.CapturedOutputs) parameters.CapturedOutputs {

	outputs := make(parameters.CapturedOutputs)
	for _, prov := range provides {
		switch prov {
		case "kubernetes":
			kubernetesParams := kube.CaptureKubernetes(component, stackBaseDir, componentsBaseDir, componentOutputs)
			parameters.MergeOutputs(outputs, kubernetesParams)

		default:
		}
	}
	return outputs
}

func mergePlatformProvides(provides map[string][]string, platformProvides []string) {
	platform := "*platform*"
	for _, provide := range platformProvides {
		providers, exist := provides[provide]
		if exist {
			providers = append(providers, platform)
		} else {
			providers = []string{platform}
		}
		provides[provide] = providers
	}
}

func mergeProvides(provides map[string][]string, componentName string, componentProvides []string,
	componentOutputs parameters.CapturedOutputs) {

	for _, prov := range componentProvides {
		switch prov {
		case "kubernetes":
			for _, reqOutput := range []string{"dns.domain", "kubernetes.api.endpoint"} {
				qName := parameters.OutputQualifiedName(reqOutput, componentName)
				_, exist := componentOutputs[qName]
				if !exist {
					log.Printf("Component `%s` declared to provide `%s` but no `%s` output found",
						componentName, prov, qName)
					log.Print("Outputs:")
					parameters.PrintCapturedOutputs(componentOutputs)
					if !config.Force {
						os.Exit(1)
					}
				}
			}

		default:
		}

		who, exist := provides[prov]
		if !exist {
			who = []string{componentName}
		} else if !util.Contains(who, componentName) { // check because of re-deploy
			if config.Debug {
				log.Printf("`%s` already provides `%s`, but component `%s` also provides `%s`",
					strings.Join(who, ", "), prov, componentName, prov)
			}
			who = append(who, componentName)
		}
		provides[prov] = who
	}
}

func eraseProvides(provides map[string][]string, componentName string) {
	for k, v := range provides {
		provides[k] = util.Omit(v, componentName)
		if len(provides[k]) == 0 {
			delete(provides, k)
		}
	}
}
