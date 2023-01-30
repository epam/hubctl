// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/ext"
	"github.com/epam/hubctl/cmd/hub/manifest"
	"github.com/epam/hubctl/cmd/hub/util"
)

var hubToSkaffoldVerbs = map[string]string{
	"deploy":   "run",
	"undeploy": "delete",
}

func findImplementation(dir string, verb string, component *manifest.Manifest) (*exec.Cmd, error) {
	makefile, err := probeMakefile(dir, verb)
	if makefile {
		binMake, err := exec.LookPath("make")
		if err != nil {
			binMake = "/usr/bin/make"
			util.WarnOnce("Unable to lookup `make` in PATH: %v; trying `%s`", err, binMake)
		}
		return &exec.Cmd{Path: binMake, Args: []string{"make", verb}, Dir: dir}, nil
	}
	script, err2 := probeScript(dir, verb)
	if script != "" {
		return &exec.Cmd{Path: script, Dir: dir}, nil
	}
	helm, args, err3 := probeHelm(dir, verb)
	if helm != "" {
		return &exec.Cmd{Path: helm, Args: append([]string{helm}, args...), Dir: dir}, nil
	}
	kustomize, args, err4 := probeKustomize(dir, verb)
	if kustomize != "" {
		return &exec.Cmd{Path: kustomize, Args: append([]string{kustomize}, args...), Dir: dir}, nil
	}
	skaffold, err5 := probeSkaffold(dir, verb)
	if skaffold {
		binSkaffold, err := exec.LookPath("skaffold")
		if err != nil {
			binSkaffold = "/usr/local/bin/skaffold"
			util.WarnOnce("Unable to lookup `skaffold` in PATH: %v; trying `%s`", err, binSkaffold)
		}
		if translatedVerb, translated := hubToSkaffoldVerbs[verb]; translated {
			verb = translatedVerb
		}
		return &exec.Cmd{Path: binSkaffold, Args: []string{"skaffold", verb}, Dir: dir}, nil
	}
	terraform, args, err6 := probeTerraform(dir, verb)
	if terraform != "" {
		return &exec.Cmd{Path: terraform, Args: append([]string{terraform}, args...), Dir: dir}, nil
	}
	arm, args, err7 := probeArm(dir, verb, component)
	if arm != "" {
		return &exec.Cmd{Path: arm, Args: append([]string{arm}, args...), Dir: dir}, nil
	}

	return nil, fmt.Errorf("No `%s` implementation found in `%s`: %s",
		verb, dir, util.Errors("; ", err, err2, err3, err4, err5, err6, err7))
}

func probeArm(dir string, verb string, component *manifest.Manifest) (string, []string, error) {
	if util.Contains(component.Requires, "arm") {
		return ext.ExtensionPath([]string{"component", "arm", verb}, nil)
	}
	return "", []string{verb}, nil
}

func probeImplementation(dir string, verb string, component *manifest.Manifest) (bool, error) {
	makefile, err := probeMakefile(dir, verb)
	if makefile {
		return true, nil
	}
	script, err2 := probeScript(dir, verb)
	if script != "" {
		return true, nil
	}
	helm, _, err3 := probeHelm(dir, verb)
	if helm != "" {
		return true, nil
	}
	helm, _, err4 := probeKustomize(dir, verb)
	if helm != "" {
		return true, nil
	}
	skaffold, err5 := probeSkaffold(dir, verb)
	if skaffold {
		return true, nil
	}
	terraform, _, err6 := probeTerraform(dir, verb)
	if terraform != "" {
		return true, nil
	}
	arm, _, err7 := probeArm(dir, verb, component)
	if arm != "" {
		return true, nil
	}
	allErrs := util.Errors("; ", err, err2, err3, err4, err5, err6, err7)
	if config.Debug {
		log.Printf("Found no `%s` implementations in `%s`: %s",
			verb, dir, allErrs)
	}
	return false, errors.New(allErrs)
}

// TODO folowing probes relies on impl file presence
// that may not be the case if the impl is templated

func probeMakefile(dir string, verb string) (bool, error) {
	filename := fmt.Sprintf("%s/Makefile", dir)
	makefile, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("%s: %v", filename, err)
	}
	bytes, err := ioutil.ReadAll(makefile)
	if err != nil {
		return false, fmt.Errorf("%s: %v", filename, err)
	}
	text := string(bytes)
	if strings.HasPrefix(text, verb+":") || strings.Contains(text, "\n"+verb+":") {
		return true, nil
	}
	return false, nil
}

func probeScript(dir string, verb string) (string, error) {
	scripts := []string{verb, "bin/" + verb, "_" + verb,
		verb + ".sh", "bin/" + verb + ".sh", "_" + verb + ".sh"}
	var lastErr error = nil
	for _, script := range scripts {
		filename := fmt.Sprintf("%s/%s", dir, script)
		info, err := os.Stat(filename)
		if err != nil {
			if !os.IsNotExist(err) {
				lastErr = err
			}
			continue
		}
		mode := info.Mode()
		if mode.IsRegular() { // TODO check exec bits?
			return script, nil
		}
	}
	return "", lastErr
}

func probeHelm(dir string, verb string) (string, []string, error) {
	return probeExtension(dir, verb, "helm",
		[]string{"values.yaml", "values.yaml.template", "values.yaml.gotemplate"})
}

func probeKustomize(dir string, verb string) (string, []string, error) {
	return probeExtension(dir, verb, "kustomize",
		[]string{"kustomization.yaml", "kustomization.yaml.template", "kustomization.yaml.gotemplate"})
}

func probeExtension(dir, verb, extension string, files []string) (string, []string, error) {
	var lastErr error = nil
	found := false
	for _, yaml := range files {
		filename := fmt.Sprintf("%s/%s", dir, yaml)
		info, err := os.Stat(filename)
		if err != nil {
			if !os.IsNotExist(err) {
				lastErr = err
			}
			continue
		}
		mode := info.Mode()
		if mode.IsRegular() {
			found = true
			break
		}
	}
	if found {
		return ext.ExtensionPath([]string{"component", extension, verb}, nil)
	}
	return "", nil, lastErr
}

func probeSkaffold(dir string, verb string) (bool, error) {
	yamls := []string{"skaffold.yaml", "skaffold.yaml.template", "skaffold.yaml.gotemplate"}
	var lastErr error = nil
	for _, yaml := range yamls {
		filename := fmt.Sprintf("%s/%s", dir, yaml)
		info, err := os.Stat(filename)
		if err != nil {
			if !os.IsNotExist(err) {
				lastErr = err
			}
			continue
		}
		mode := info.Mode()
		if mode.IsRegular() {
			return true, nil
		}
	}
	return false, lastErr
}

func probeTerraform(dir string, verb string) (string, []string, error) {
	globs := []string{"*.tf", "*.tf.*"}
	var lastErr error = nil
	found := false
	for _, glob := range globs {
		pattern := fmt.Sprintf("%s/%s", dir, glob)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			lastErr = err
			continue
		}
		if len(matches) > 0 {
			found = true
			break
		}
	}
	if found {
		path, args, err := ext.ExtensionPath([]string{"component", "terraform", verb}, nil)
		return path, args, err
	}
	return "", nil, lastErr
}
