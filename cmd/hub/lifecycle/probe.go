package lifecycle

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/ext"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var hubToSkaffoldVerbs = map[string]string{
	"deploy":   "run",
	"undeploy": "delete",
}

func findImplementation(dir string, verb string) (*exec.Cmd, error) {
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
	helm, err3 := probeHelm(dir, verb)
	if helm != "" {
		return &exec.Cmd{Path: helm, Dir: dir}, nil
	}
	skaffold, err4 := probeSkaffold(dir, verb)
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
	return nil, fmt.Errorf("No `%s` implementation found in `%s`: %s",
		verb, dir, util.Errors("; ", err, err2, err3, err4))
}

func probeImplementation(dir string, verb string) (bool, error) {
	makefile, err := probeMakefile(dir, verb)
	if makefile {
		return true, nil
	}
	script, err2 := probeScript(dir, verb)
	if script != "" {
		return true, nil
	}
	helm, err3 := probeHelm(dir, verb)
	if helm != "" {
		return true, nil
	}
	skaffold, err4 := probeSkaffold(dir, verb)
	if skaffold {
		return true, nil
	}
	allErrs := util.Errors("; ", err, err2, err3, err4)
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

func probeHelm(dir string, verb string) (string, error) {
	yamls := []string{"values.yaml", "values.yaml.template", "values.yaml.gotemplate"}
	var lastErr error = nil
	found := false
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
			found = true
			break
		}
	}
	if found {
		path, _, err := ext.ExtensionPath([]string{"component", "helm", verb}, nil) // TODO return additional args?
		return path, err
	}
	return "", lastErr
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
