// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/agilestacks/hub/cmd/hub/api"
	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/manifest"
	"github.com/agilestacks/hub/cmd/hub/util"
)

func AskParameter(parameter manifest.Parameter,
	environment map[string]string, hubEnvironment, hubStackInstance, hubApplication string,
	isDeploy bool) (interface{}, error) {

	qName := parameter.QName()

	if parameter.FromEnv != "" {
		key := parameter.FromEnv
		if environment != nil {
			if v, exist := environment[key]; exist {
				return v, nil
			}
		}
		if v, exist := os.LookupEnv(key); exist {
			return v, nil
		}
	}
	if parameter.FromFile != "" {
		filename := parameter.FromFile
		if filename[0] == '$' && len(filename) > 1 {
			key := filename[1:]
			filename = ""
			if environment != nil {
				if v, exist := environment[key]; exist {
					filename = v
				}
			}
			if filename == "" {
				if v, exist := os.LookupEnv(key); exist {
					filename = v
				}
			}
			if filename == "" {
				util.Warn("Parameter `%s` `fromFile = %s` expands to an empty filename", qName, parameter.FromFile)
			}
		}
		if filename != "" {
			bytes, err := ioutil.ReadFile(filename)
			if err != nil {
				return "(error)", fmt.Errorf("Error reading `%s`: %v", filename, err)
			}
			return string(bytes), nil
		}
	}

	if hubEnvironment != "" || hubStackInstance != "" || hubApplication != "" {
		found, v, errs := api.GetParameterOrMaybeCreateSecret(hubEnvironment, hubStackInstance, hubApplication,
			parameter.Name, parameter.Component, isDeploy && parameter.Empty != "allow")
		if len(errs) > 0 {
			where := make([]string, 0, 3)
			if hubEnvironment != "" {
				where = append(where, fmt.Sprintf("environment `%s`", hubEnvironment))
			}
			if hubStackInstance != "" {
				where = append(where, fmt.Sprintf("stack instance `%s`", hubStackInstance))
			}
			if hubApplication != "" {
				where = append(where, fmt.Sprintf("application `%s`", hubApplication))
			}
			util.Warn("Error query parameter `%s` in %s:\n\t%s",
				qName, strings.Join(where, ", "), util.Errors("\n\t", errs...))
		}
		if found && v != "" {
			return v, nil
		}
	}

	// TODO review
	// if parameter with default value is marked empty: allow, then we set value to default without prompt
	if parameter.Empty != "allow" && isatty.IsTerminal(os.Stdin.Fd()) {
		prompt := "Enter value for"
		if parameter.Brief != "" {
			prompt = fmt.Sprintf("%s %s (%s)", prompt, parameter.Brief, qName)
		} else {
			prompt = fmt.Sprintf("%s %s", prompt, qName)
		}
		if !util.Empty(parameter.Default) {
			prompt = fmt.Sprintf("%s [%v]", prompt, parameter.Default)
		}
		fmt.Printf("%s: ", prompt)
		var value string
		read, err := fmt.Scanln(&value)
		if read > 0 {
			if err != nil {
				return "(error)", fmt.Errorf("Error reading input: %v (read %d items)", err, read)
			}
			return value, nil
		}
	}

	if !util.Empty(parameter.Default) {
		return parameter.Default, nil
	}

	if parameter.Env != "" && parameter.FromEnv == "" {
		util.Warn("Parameter `%s` has `env = %s` assigned. Did you mean `fromEnv`?", qName, parameter.Env)
	}
	if parameter.Empty == "allow" {
		if config.Debug {
			log.Printf("Empty parameter `%s` value allowed", qName)
		}
		return "", nil
	}

	return "(unknown)", fmt.Errorf("Parameter `%s` has no value nor default assigned", qName)
}
