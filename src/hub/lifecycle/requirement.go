package lifecycle

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"hub/config"
	"hub/kube"
	"hub/manifest"
	"hub/parameters"
	"hub/util"
)

const providedByEnv = "*environment*"

func prepareComponentRequires(provided map[string][]string, componentManifest *manifest.Manifest,
	parameters parameters.LockedParameters, outputs parameters.CapturedOutputs, maybeOptional map[string][]string) bool {

	setups := make([]util.Tuple2, 0, len(componentManifest.Requires))
	allProvided := true

	componentName := manifest.ComponentQualifiedNameFromMeta(&componentManifest.Meta)
	for _, req := range componentManifest.Requires {
		by, exist := provided[req]
		if !exist || len(by) == 0 {
			if optionalFor, exist := maybeOptional[req]; exist &&
				(util.Contains(optionalFor, componentName) || util.Contains(optionalFor, "*")) {

				allProvided = false
				if config.Verbose {
					log.Printf("Optional requirement `%s` is not provided", req)
				}
				continue
			}
			log.Printf("Component `%s` requires `%s` but only following provides are currently known:",
				componentName, strings.Join(componentManifest.Requires, ", "))
			util.PrintDeps(provided)
			if config.Force {
				continue
			} else {
				os.Exit(1)
			}
		}
		if config.Debug && len(by) == 1 {
			log.Printf("Requirement `%s` provided by `%s`", req, by[0])
		}
		provider := by[len(by)-1]
		if len(by) > 1 {
			util.Warn("Requirement `%s` provided by multiple components `%s`, only `%s` will be used",
				req, strings.Join(by, ", "), provider)
		}

		setups = append(setups, util.Tuple2{req, provider})
	}

	if allProvided {
		for _, setup := range setups {
			setupRequirement(setup.S1, setup.S2, parameters, outputs)
		}
	}
	return allProvided
}

func setupRequirement(requirement string, provider string,
	parameters parameters.LockedParameters, outputs parameters.CapturedOutputs) {

	switch requirement {
	case "kubectl", "kubernetes":
		kube.SetupKubernetes(parameters, provider, outputs, "", false)

	case "aws", "gcp", "gcs", "tiller", "helm", "vault", "ingress":
		if config.Verbose {
			log.Printf("Assuming `%s` requirement is setup", requirement)
		}

	default:
		util.Warn("Don't know how to setup requirement `%s`", requirement)
	}
}

var bins = map[string][]string{
	"gcp":        {"gcloud", "version"},
	"gcs":        {"gsutil", "version"},
	"kubectl":    {"kubectl", "version", "-c"},
	"kubernetes": {"kubectl", "version", "-c"},
	"helm":       {"helm", "version", "-c"},
}

func checkRequires(requires []string, maybeOptional map[string][]string) map[string][]string {
	provided := make(map[string][]string, len(requires))
	for _, require := range requires {
		skip := false
		switch require {
		case "aws":
			hasAws, err := checkRequiresAws()
			if !hasAws {
				log.Fatalf("`aws` requirement cannot be satisfied: %v", err)
			}

		case "gcp", "gcs", "kubectl", "kubernetes", "helm", "vault":
			bin, exist := bins[require]
			if !exist {
				bin = []string{require, "version"}
			}
			hasBin, err := checkRequiresBin(bin...)
			if !hasBin {
				log.Fatalf("`%s` requirement cannot be satisfied: %v", require, err)
			}

		default:
			if optionalFor, exist := maybeOptional[require]; !exist {
				log.Fatalf("Cannot check for `requires: %s`: no implementation", require)
			} else {
				skip = true
				if config.Debug {
					log.Printf("Requirement `%s` is optional for %v", require, optionalFor)
				}
			}
		}
		if !skip {
			provided[require] = []string{providedByEnv}
		}
	}
	return provided
}

func checkRequiresAws() (bool, error) {
	return checkRequiresBin("aws", "s3", "ls", "--page-size", "5")
}

func checkRequiresBin(bin ...string) (bool, error) {
	if config.Trace {
		log.Printf("Checking %v", bin)
	}
	cmd := exec.Command(bin[0], bin[1:]...)
	cmd.Env = os.Environ()
	if config.Trace {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("%v: %v", bin, err)
	}
	return true, nil
}

func noEnvironmentProvides(provides map[string][]string) map[string][]string {
	filtered := make(map[string][]string)
	for p, by := range provides {
		if util.Contains(by, providedByEnv) {
			by = util.Omit(by, providedByEnv)
		}
		if len(by) > 0 {
			filtered[p] = by
		}
	}
	return filtered
}

func parseRequiresTunning(requires manifest.RequiresTuning) map[string][]string {
	optional := make(map[string][]string)
	for _, req := range requires.Optional {
		i := strings.Index(req, ":")
		if i > 0 && i < len(req)-1 {
			component := req[i+1:]
			req = req[:i]
			util.AppendMapList(optional, req, component)
		} else if i == -1 {
			util.AppendMapList(optional, req, "*")
		}
	}
	return optional
}

var falseParameterValues = []string{"", "false", "0", "no", "(unknown)"}

func calculateOptionalFalseParameters(componentName string, params parameters.LockedParameters, optionalRequires map[string][]string) []string {
	falseParameters := make([]string, 0)
	for term, optionalForList := range optionalRequires {
		if strings.Contains(term, ".") { // looks like a parameter
			for _, optionalFor := range optionalForList {
				if optionalFor == "*" || optionalFor == componentName {
					parameterExists := false
					for _, p := range params {
						if p.Name == term && (p.Component == "" || p.Component == componentName) {
							parameterExists = true
							if util.Contains(falseParameterValues, p.Value) {
								falseParameters = append(falseParameters, p.QName())
								if optionalFor == "*" {
									util.WarnOnce("Optional parameter `lifecycle.requires.optional = %s` targets all components as wildcard;\n\tYou may want to narrow specification to `%[1]s:component`",
										term)
								}
							}
						}
					}
					if !parameterExists && optionalFor != "*" {
						falseParameters = append(falseParameters, term)
					}
				}
			}
		}
	}
	return falseParameters
}
