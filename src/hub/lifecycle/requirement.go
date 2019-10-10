package lifecycle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"hub/config"
	"hub/kube"
	"hub/manifest"
	"hub/parameters"
	"hub/util"
)

const (
	providedByEnv          = "*environment*"
	gcpServiceAccountsHelp = "https://cloud.google.com/docs/authentication/getting-started"
	azureGoSdkAuthHelp     = "https://docs.microsoft.com/en-us/go/azure/azure-sdk-go-authorization"
)

var (
	supportedCloudRequires = []string{"aws", "azure", "gcp", "gcs"}
	guessedEnabledClouds   []string
)

func prepareComponentRequires(provided map[string][]string, componentManifest *manifest.Manifest,
	parameters parameters.LockedParameters, outputs parameters.CapturedOutputs,
	maybeOptional map[string][]string, enabledClouds []string) ([]string, error) {

	componentRequires := maybeOmitCloudRequires(componentManifest.Requires, enabledClouds)

	setups := make([]util.Tuple2, 0, len(componentRequires))
	optionalNotProvided := make([]string, 0)

	componentName := manifest.ComponentQualifiedNameFromMeta(&componentManifest.Meta)
	for _, req := range componentRequires {
		by, exist := provided[req]
		if !exist || len(by) == 0 {
			if optionalFor, exist := maybeOptional[req]; exist &&
				(util.Contains(optionalFor, componentName) || util.Contains(optionalFor, "*")) {

				optionalNotProvided = append(optionalNotProvided, req)
				if config.Verbose {
					log.Printf("Optional requirement `%s` is not provided", req)
				}
				continue
			}
			err := fmt.Errorf("Component `%s` requires `%s` but only following provides are currently known:\n%s",
				componentName, strings.Join(componentRequires, ", "), util.SprintDeps(provided))
			return optionalNotProvided, err
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

	if len(optionalNotProvided) == 0 {
		for _, setup := range setups {
			setupRequirement(setup.S1, setup.S2, parameters, outputs)
		}
	}
	return optionalNotProvided, nil
}

func maybeOmitCloudRequires(requires, enabledClouds []string) []string {
	if !util.ContainsAny(supportedCloudRequires, requires) {
		return requires
	}
	if len(enabledClouds) == 0 {
		enabledClouds = guessEnabledClouds()
	}
	if len(enabledClouds) == 0 {
		util.WarnOnce("Unable to autodetect enabled clouds, try `--clouds` if cloud access is failing")
		return requires
	}
	if util.Contains(enabledClouds, "gcp") {
		enabledClouds = append(enabledClouds, "gcs")
	}
	modified := make([]string, 0, len(requires))
	for _, r := range requires {
		if !util.Contains(supportedCloudRequires, r) || util.Contains(enabledClouds, r) {
			modified = append(modified, r)
		}
	}
	return modified
}

func guessEnabledClouds() []string {
	if guessedEnabledClouds != nil {
		return guessedEnabledClouds
	}
	var clouds []string
	// TODO probe meta-data server
	if util.MaybeEnv([]string{"AWS_PROFILE", "AWS_DEFAULT_REGION", "AWS_ACCESS_KEY_ID"}) {
		clouds = append(clouds, "aws")
	}
	if util.MaybeEnv([]string{"AZURE_AUTH_LOCATION", "AZURE_TENANT_ID", "AZURE_RESOURCE_GROUP_NAME"}) {
		clouds = append(clouds, "azure")
	}
	if util.MaybeEnv([]string{"GOOGLE_APPLICATION_CREDENTIALS"}) {
		clouds = append(clouds, "gcp")
	}
	if config.Debug {
		detected := "(none)"
		if len(clouds) > 0 {
			detected = strings.Join(clouds, ", ")
		}
		log.Printf("Autodetected clouds: %s", detected)
	}
	guessedEnabledClouds = clouds
	return clouds
}

func setupRequirement(requirement string, provider string,
	parameters parameters.LockedParameters, outputs parameters.CapturedOutputs) {

	switch requirement {
	case "kubectl", "kubernetes":
		kube.SetupKubernetes(parameters, provider, outputs, "", false, false)

	case "aws", "azure", "gcp", "gcs", "tiller", "helm", "etcd", "vault", "ingress", "tls-ingress":
		wellKnown, err := checkRequire(requirement)
		if wellKnown {
			if err != nil {
				log.Fatalf("`%s` requirement cannot be satisfied: %v", requirement, err)
			}
		} else {
			if config.Verbose {
				log.Printf("Assuming `%s` requirement is setup", requirement)
			}
		}

	default:
		util.WarnOnce("Don't know how to setup requirement `%s`", requirement)
	}
}

var bins = map[string][]string{
	"aws":        {"aws", "s3", "ls", "--page-size", "5"},
	"gcp":        {"gcloud", "version"},
	"gcs":        {"gsutil", "version"},
	"kubectl":    {"kubectl", "version", "--client"},
	"kubernetes": {"kubectl", "version", "--client"},
	"helm":       {"helm", "version", "--client"},
	"etcd":       {"etcdctl", "--version"},
}

type BinVersion struct {
	minVersion    string
	versionRegexp *regexp.Regexp
}

var binVersion = map[string]BinVersion{
	"gcloud":  {"246.0.0", regexp.MustCompile("Google Cloud SDK ([\\d.]+)")},
	"gsutil":  {"4.38", regexp.MustCompile("version: ([\\d.]+)")},
	"vault":   {"1.1.2", regexp.MustCompile("Vault v([\\d.]+)")},
	"kubectl": {"1.14.2", regexp.MustCompile("GitVersion:\"v([\\d.]+)")},
	"helm":    {"2.14.0", regexp.MustCompile("SemVer:\"v([\\d.]+)")},
}

func checkStackRequires(requires []string, optional, requiresOfOptionalComponents map[string][]string) map[string][]string {
	provided := make(map[string][]string, len(requires))
	for _, require := range requires {
		_, err := checkRequire(require)
		if err == nil {
			provided[require] = []string{providedByEnv}
			continue
		}
		if config.Verbose {
			log.Printf("`%s` requirement cannot be satisfied: %v", require, err)
		}
		if requiredByOptional, exist := requiresOfOptionalComponents[require]; exist {
			if config.Verbose {
				log.Printf("Skipping requirement `%s` requested by optional components %v", require, requiredByOptional)
			}
			continue
		}
		if optionalFor, exist := optional[require]; exist {
			if config.Verbose {
				log.Printf("Skipping requirement `%s` as it is optional for %v", require, optionalFor)
			}
			continue
		}
		os.Exit(1)
	}
	return provided
}

var requirementsVerified = make(map[string]struct{})

func checkRequire(require string) (bool, error) {
	if _, exist := requirementsVerified[require]; exist {
		return true, nil
	}
	switch require {
	case "azure":
		err := checkRequiresAzure()
		if err != nil {
			return true, err
		}
		setupTerraformAzureOsEnv()

	case "aws", "gcp", "gcs", "kubectl", "kubernetes", "helm", "vault": // "etcd"
		bin, exist := bins[require]
		if !exist {
			bin = []string{require, "version"}
		}
		out, err := checkRequiresBin(bin...)
		if err != nil {
			return true, err
		}
		verReq, exist := binVersion[bin[0]]
		if exist {
			err := checkRequiresBinVersion(verReq.minVersion, verReq.versionRegexp, out)
			if err != nil {
				util.WarnOnce("`%s` version requirement cannot be satisfied: %s: %v; update `%[1]s` binary?",
					bin[0], require, err)
			}
		}
		if require == "gcp" || require == "gcs" {
			err := checkRequiresGcp()
			if err != nil {
				return true, err
			}
		}

	default:
		return false, errors.New("no implementation")
	}
	requirementsVerified[require] = struct{}{}
	return true, nil
}

func checkRequiresBin(bin ...string) ([]byte, error) {
	if config.Debug {
		printCmd(bin)
	}
	cmd := exec.Command(bin[0], bin[1:]...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v: %v", bin, err)
	}
	if config.Trace && len(out) > 0 {
		fmt.Printf("%s", out)
	}
	return out, err
}

func checkRequiresBinVersion(minVer string, verRegexp *regexp.Regexp, out []byte) error {
	if len(out) == 0 {
		return errors.New("no output")
	}
	match := verRegexp.FindSubmatch(out)
	if len(match) != 2 {
		return errors.New("no version string found")
	}
	ver := string(match[1])
	if ver < minVer {
		return fmt.Errorf("`%s` version detected; should have at least version `%s`", ver, minVer)
	}
	return nil
}

func checkRequiresAzure() error {
	out, err := checkRequiresBin("az", "storage", "account", "list", "-o", "table")
	if err == nil {
		return nil
	}
	if !bytes.Contains(out, []byte("az login")) {
		return err
	}
	tenantId := os.Getenv("AZURE_TENANT_ID")
	if tenantId == "" {
		return fmt.Errorf("AZURE_TENANT_ID is not set, see %s", azureGoSdkAuthHelp)
	}
	clientId := os.Getenv("AZURE_CLIENT_ID")
	if clientId == "" {
		return fmt.Errorf("AZURE_CLIENT_ID is not set, see %s", azureGoSdkAuthHelp)
	}
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = os.Getenv("AZURE_CERTIFICATE_PATH")
		if clientSecret == "" {
			return fmt.Errorf("No AZURE_CLIENT_SECRET, nor AZURE_CERTIFICATE_PATH is set, see %s", azureGoSdkAuthHelp)
		}
	}
	_, err = checkRequiresBin("az", "login", "--service-principal",
		"--tenant", tenantId, "--username", clientId, "--password", clientSecret)
	return err
	// TODO az login --identity
	// https://docs.microsoft.com/en-us/go/azure/azure-sdk-go-authorization
	// Also, SDK supports AZURE_AUTH_LOCATION
}

func setupTerraformAzureOsEnv() {
	if os.Getenv("ARM_CLIENT_ID") != "" {
		return
	}
	if config.Debug {
		log.Print("Setting Terraform ARM_* variables for Azure provider")
	}
	if os.Getenv("ARM_ACCESS_KEY") == "" {
		vars := []string{"AZURE_STORAGE_ACCESS_KEY", "AZURE_STORAGE_KEY"}
		for _, v := range vars {
			key := os.Getenv(v)
			if key != "" {
				if config.Trace {
					log.Printf("Setting ARM_ACCESS_KEY=%s", key)
				}
				os.Setenv("ARM_ACCESS_KEY", key)
				break
			}
		}
	}
	for _, v := range []string{"ARM_CLIENT_ID", "ARM_CLIENT_SECRET", "ARM_SUBSCRIPTION_ID", "ARM_TENANT_ID"} {
		src := "AZURE" + v[3:]
		if value := os.Getenv(src); value != "" {
			if config.Trace {
				log.Printf("Setting %s=%s", v, value)
			}
			os.Setenv(v, value)
		} else {
			util.Warn("Unable to set %s: no %s env var set", v, src)
		}
	}
	// TODO ARM_USE_MSI ARM_ENVIRONMENT?
	// https://www.terraform.io/docs/backends/types/azurerm.html
}

func checkRequiresGcp() error {
	out, err := checkRequiresBin("gcloud", "auth", "list")
	if err != nil {
		return err
	}
	if !bytes.Contains(out, []byte("gcloud auth login")) {
		return nil
	}

	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsFile == "" {
		return fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS is not set, see %s", gcpServiceAccountsHelp)
	}
	_, err = checkRequiresBin("gcloud", "auth", "activate-service-account", "--key-file", credsFile)
	if err != nil {
		return err
	}

	jsonData, err := ioutil.ReadFile(credsFile)
	if err != nil {
		return fmt.Errorf("Unable to read `%s`: %v", credsFile, err)
	}
	var creds map[string]string
	err = json.Unmarshal(jsonData, &creds)
	if err != nil {
		return fmt.Errorf("Unable to unmarshall `%s`: %v", credsFile, err)
	}
	credsProject, existInCredsFile := creds["project_id"]

	out, err = checkRequiresBin("gcloud", "config", "get-value", "project")
	if err != nil {
		return err
	}
	gcloudProject := strings.Trim(string(out), " \r\n")

	projectUnset := "(unset)"
	if !existInCredsFile {
		if gcloudProject == projectUnset {
			util.WarnOnce("No `project_id` found in `%s` and no gcloud project is set in config - gcloud will fail until --project is specifed inline",
				credsFile)
		} else {
			if config.Debug {
				log.Printf("Using gcloud pre-configured `%s` project id", gcloudProject)
			}
		}
		return nil
	}
	if gcloudProject != credsProject {
		if gcloudProject == projectUnset || config.Force {
			if config.Force && gcloudProject != projectUnset {
				util.Warn("Setting gcloud project to `%s`; was `%s`", credsProject, gcloudProject)
			}
			_, err = checkRequiresBin("gcloud", "config", "set", "project", credsProject)
			if err != nil {
				return err
			}
		} else {
			util.WarnOnce("Using gcloud pre-configured `%s` project id that is different from service account credentials `%s` project id (%s)",
				gcloudProject, credsProject, credsFile)
		}
	}
	return nil
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

func calculateRequiresOfOptionalComponents(componentManifests []manifest.Manifest, lifecycle *manifest.Lifecycle, requires []string) map[string][]string {
	var optionalRequirements []string
	for _, requirement := range requires {
		optional := true
		for _, componentManifest := range componentManifests {
			if util.Contains(componentManifest.Requires, requirement) {
				componentName := manifest.ComponentQualifiedNameFromMeta(&componentManifest.Meta)
				if !optionalComponent(lifecycle, componentName) {
					optional = false
					break
				}
			}
		}
		if optional {
			optionalRequirements = append(optionalRequirements, requirement)
		}
	}
	requiredBy := make(map[string][]string)
	for _, componentManifest := range componentManifests {
		for _, requirement := range util.Union(optionalRequirements, componentManifest.Requires) {
			componentName := manifest.ComponentQualifiedNameFromMeta(&componentManifest.Meta)
			util.AppendMapList(requiredBy, requirement, componentName)
		}
	}
	return requiredBy
}
