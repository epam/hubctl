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
	return nil, fmt.Errorf("No `%s` implementation found in `%s`: %s",
		verb, dir, util.Errors("; ", err, err2))
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
	if config.Debug {
		log.Printf("Found no `%s` implementations in `%s`: %s",
			verb, dir, util.Errors("; ", err, err2))
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
