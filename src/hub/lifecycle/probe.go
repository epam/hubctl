package lifecycle

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"hub/config"
	"hub/util"
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
	skaffold, err3 := probeSkaffold(dir, verb)
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
		verb, dir, util.Errors("; ", err, err2, err3))
}

func probeImplementation(dir string, verb string) bool {
	makefile, err := probeMakefile(dir, verb)
	if makefile {
		return true
	}
	script, err2 := probeScript(dir, verb)
	if script != "" {
		return true
	}
	skaffold, err3 := probeSkaffold(dir, verb)
	if skaffold {
		return true
	}
	if config.Debug {
		log.Printf("Found no `%s` implementations in `%s`: %s",
			verb, dir, util.Errors("; ", err, err2, err3))
	}
	return false
}

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
	scripts := []string{verb + ".sh", "bin/" + verb + ".sh", "_" + verb + ".sh"}
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
		if mode.IsRegular() { // TODO check exec bits
			return script, nil
		}
	}
	return "", lastErr
}

func probeSkaffold(dir string, verb string) (bool, error) {
	yamls := []string{"skaffold.yaml", "skaffold.yaml.template"}
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
