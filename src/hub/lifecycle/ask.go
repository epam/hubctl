package lifecycle

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mattn/go-isatty"

	"hub/api"
	"hub/config"
	"hub/manifest"
	"hub/util"
)

func AskParameter(parameter *manifest.Parameter,
	environment map[string]string, hubEnvironment, hubStackInstance, hubApplication string,
	isDeploy bool) (retErr error) {

	if parameter.FromEnv != "" {
		key := parameter.FromEnv
		if environment != nil {
			if v, exist := environment[key]; exist {
				parameter.Value = v
				return
			}
		}
		if v, exist := os.LookupEnv(key); exist {
			parameter.Value = v
			return
		}
	}

	qName := parameter.QName()

	if hubEnvironment != "" || hubStackInstance != "" || hubApplication != "" {
		found, v, errs := api.GetParameterOrMaybeCreateSecret(hubEnvironment, hubStackInstance, hubApplication,
			parameter.Name, parameter.Component, isDeploy)
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
			parameter.Value = v
			return
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
		if parameter.Default != "" {
			prompt = fmt.Sprintf("%s [%s]", prompt, parameter.Default)
		}
		fmt.Printf("%s: ", prompt)
		read, err := fmt.Scanln(&parameter.Value)
		if read > 0 {
			if err != nil {
				log.Fatalf("Error reading input: %v (read %d items)", err, read)
			} else {
				return
			}
		}
	}

	if parameter.Default == "" {
		if parameter.Env != "" && parameter.FromEnv == "" {
			util.Warn("Parameter `%s` has `env = %s` assigned. Did you mean `fromEnv`?", qName, parameter.Env)
		}
		if parameter.Empty == "allow" {
			if config.Debug {
				log.Printf("Empty parameter `%s` value allowed", qName)
			}
			return
		}
		retErr = fmt.Errorf("Parameter `%s` has no value nor default assigned", qName)
		parameter.Value = "unknown"
	} else {
		parameter.Value = parameter.Default
	}

	return
}
