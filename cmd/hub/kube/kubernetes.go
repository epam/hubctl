// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package kube

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/manifest"
	"github.com/epam/hubctl/cmd/hub/parameters"
	"github.com/epam/hubctl/cmd/hub/util"
)

const (
	stackNameOutput               = "dns.name"
	stackDomainOutput             = "dns.domain"
	kubernetesFlavorOutput        = "kubernetes.flavor"
	kubernetesContext             = "kubernetes.context"
	kubernetesApiEndpointOutput   = "kubernetes.api.endpoint"
	kubernetesApiTokenOutput      = "kubernetes.api.token"
	kubernetesApiCaCertOutput     = "kubernetes.api.caCert"
	kubernetesApiClientCertOutput = "kubernetes.api.clientCert"
	kubernetesApiClientKeyOutput  = "kubernetes.api.clientKey"
	kubernetesEksClusterOutput    = "kubernetes.eks.cluster"
	kubernetesGkeClusterOutput    = "kubernetes.gke.cluster"
	kubernetesFileRefPrefix       = "file:"
)

var (
	KubernetesDefaultProviders = []string{
		"kubernetes", "eks", "gke", "aks",
		"stack-k8s-aws", "stack-k8s-eks", "stack-k8s-gke", "stack-k8s-aks",
		"k8s-aws", "k8s-eks", "k8s-gke", "k8s-aks", "k8s-hybrid",
		"k8s-metal", "k8s-openshift"}
	kubernetesApiKeysFileSuf = map[string]string{
		kubernetesApiCaCertOutput:     "-ca.pem",
		kubernetesApiClientCertOutput: "-client.pem",
		kubernetesApiClientKeyOutput:  "-client-key.pem",
	}
	KubernetesParameters = []string{stackDomainOutput, kubernetesFlavorOutput, kubernetesApiEndpointOutput,
		kubernetesApiTokenOutput, kubernetesApiCaCertOutput, kubernetesApiClientCertOutput, kubernetesApiClientKeyOutput,
		kubernetesEksClusterOutput, kubernetesGkeClusterOutput}
	KubernetesKeysParameters = []string{
		kubernetesApiCaCertOutput, kubernetesApiClientCertOutput, kubernetesApiClientKeyOutput}
	KubernetesSecretParameters = []string{
		kubernetesApiTokenOutput, kubernetesApiCaCertOutput, kubernetesApiClientCertOutput, kubernetesApiClientKeyOutput}
)

func CaptureKubernetes(component *manifest.ComponentRef, stackBaseDir string, componentsBaseDir string,
	componentOutputs parameters.CapturedOutputs) parameters.CapturedOutputs {

	outputs := make(parameters.CapturedOutputs)

	componentName := manifest.ComponentQualifiedNameFromRef(component)
	flavor := "k8s-aws"
	if o, exist := componentOutputs[parameters.OutputQualifiedName(kubernetesFlavorOutput, componentName)]; exist {
		flavor = util.String(o.Value)
	}

	domainQName := parameters.OutputQualifiedName(stackDomainOutput, componentName)
	if _, exists := componentOutputs[domainQName]; !exists {
		util.Warn("Component `%s` declared to provide Kubernetes but no `%s` output found", componentName, domainQName)
		if len(componentOutputs) > 0 {
			log.Print("Outputs:")
			parameters.PrintCapturedOutputs(componentOutputs)
		}
		return outputs
	}

	switch flavor {
	case "k8s-aws":
		var missing []string
		for _, outputName := range KubernetesKeysParameters {
			outputQName := parameters.OutputQualifiedName(outputName, componentName)
			if _, exists := componentOutputs[outputQName]; !exists {
				missing = append(missing, outputName)
			}
		}
		if len(missing) > 0 {
			util.Warn("Component `%s` declared to provide Kubernetes but some key/certs output(s) are missing: %v",
				componentName, missing)
		}

	case "eks":
		caQName := parameters.OutputQualifiedName(kubernetesApiCaCertOutput, componentName)
		_, exists := componentOutputs[caQName]
		if !exists {
			util.Warn("Component `%s` declared to provide EKS Kubernetes but no `%s` output found", componentName, caQName)
			if config.Debug && len(componentOutputs) > 0 {
				log.Print("Outputs:")
				parameters.PrintCapturedOutputs(componentOutputs)
			}
		}

	case "openshift":
		tokenQName := parameters.OutputQualifiedName(kubernetesApiTokenOutput, componentName)
		_, exists := componentOutputs[tokenQName]
		if !exists {
			util.Warn("Component `%s` declared to provide OpenShift Kubernetes but no `%s` output found", componentName, tokenQName)
			if config.Debug && len(componentOutputs) > 0 {
				log.Print("Outputs:")
				parameters.PrintCapturedOutputs(componentOutputs)
			}
		}

	case "gke":
		caQName := parameters.OutputQualifiedName(kubernetesApiCaCertOutput, componentName)
		tokenQName := parameters.OutputQualifiedName(kubernetesApiTokenOutput, componentName)
		_, caExists := componentOutputs[caQName]
		_, tokenExists := componentOutputs[tokenQName]
		if !caExists || !tokenExists {
			util.Warn("Component `%s` declared to provide GKE Kubernetes but no `%s` or `%s` output found", componentName, tokenQName, caQName)
			if config.Debug && len(componentOutputs) > 0 {
				log.Print("Outputs:")
				parameters.PrintCapturedOutputs(componentOutputs)
			}
		}

	case "aks":
		caQName := parameters.OutputQualifiedName(kubernetesApiCaCertOutput, componentName)
		tokenQName := parameters.OutputQualifiedName(kubernetesApiTokenOutput, componentName)
		_, caExists := componentOutputs[caQName]
		_, tokenExists := componentOutputs[tokenQName]
		if !caExists || !tokenExists {
			util.Warn("Component `%s` declared to provide AKS Kubernetes but no `%s` or `%s` output found", componentName, tokenQName, caQName)
			if config.Debug && len(componentOutputs) > 0 {
				log.Print("Outputs:")
				parameters.PrintCapturedOutputs(componentOutputs)
			}
		}
	}

	return outputs
}

// Returns the first non-empty environment variable value and name
func getFirstEnviron(name ...string) (string, string) {
	for _, n := range name {
		if v := os.Getenv(n); v != "" {
			return n, v
		}
	}
	return "", ""
}

func SetupKubernetes(params parameters.LockedParameters,
	provider string, outputs parameters.CapturedOutputs,
	context string, overwrite, keepPems bool) {

	kubectl := "kubectl"
	domain, _ := mayOutput(params, outputs, provider, stackDomainOutput)
	if domain == "" {
		util.Debug("Parameters from %s are not providing: %s", provider, stackDomainOutput) // try to get domain from environment
		name, val := getFirstEnviron("HUB_DOMAIN_NAME", "DOMAIN_NAME")
		if val != "" {
			util.Debug("Using %s from %s variable as %s", val, name, stackDomainOutput)
			domain = val
		} else {
			// Porting fuzzy logic here from extensions, for compatibility
			// When stack doesn't have a ingress domain.
			// In this case user will declare only a stack name
			util.Debug("Cannot find dns.domain from variables")
			util.Debug("Trying %s instead for stack that doesn't use ingress", stackNameOutput)
			stackName, _ := mayOutput(params, outputs, provider, stackNameOutput)
			if stackName != "" {
				domain = stackName
			} else {
				util.Debug("Trying environment variables")
				name, val = getFirstEnviron("HUB_STACK_NAME", "STACK_NAME")
				if val != "" {
					util.Debug("Using %s from %s variable as %s", val, name, stackNameOutput)
					domain = val
				} else {
					util.Warn("Giving up with domain name from %s", provider)
				}
			}
		}
	}
	if domain == "" {
		util.Errors("Unable to setup Kubeconfig: no domain name found")
		os.Exit(1)
		return
	}

	if context != "" {
		util.Debug("Using kube context: %s", context)
	} else {
		context, _ = mayOutput(params, outputs, provider, kubernetesContext)
		if context == "" {
			util.Debug("Parameters from %s are not providing: %s", provider, kubernetesContext)
			util.Debug("Taking kube context from %s parameter %s", provider, stackDomainOutput)
		}
	}
	if context == "" {
		context = domain
	}

	flavor, _ := mayOutput(params, outputs, provider, kubernetesFlavorOutput)
	if flavor == "" {
		flavor = "k8s-aws"
	}

	eksClusterName := ""
	bearerToken := ""
	switch flavor {
	case "eks":
		eksClusterName = mustOutput(params, outputs, provider, kubernetesEksClusterOutput)
		bearerToken, _ = mayOutput(params, outputs, provider, kubernetesApiTokenOutput)
	case "openshift", "gke", "aks":
		bearerToken = mustOutput(params, outputs, provider, kubernetesApiTokenOutput)
	}

	configFilename, err := kubeconfigFilename()
	if err != nil {
		util.WarnOnce("Unable to setup Kubeconfig: %v", err)
		return
	}

	var configCmd string
	if config.SwitchKubeconfigContext {
		if config.Verbose {
			log.Printf("Changing Kubeconfig context to `%s`", context)
		}
		configCmd = "use-context"
	} else {
		if config.Verbose {
			log.Printf("Checking Kubeconfig context `%s`", context)
		}
		configCmd = "get-contexts"
	}

	outBytes, err := execOutput(kubectl, "config", configCmd, context)
	if err == nil {
		if !overwrite {
			// check CA cert match to
			// catch Kubeconfig leftovers from previous deployment to the same domain name
			if ca, exist := mayOutput(params, outputs, provider, kubernetesApiCaCertOutput); exist && ca != "" {
				warnClusterCaCertMismatch(configFilename, context, ca)
			}
			return
		}
	} else {
		out := string(outBytes)
		if !strings.Contains(out, "no context exists") && !strings.Contains(out, "not found") {
			util.MaybeFatalf("kubectl failed: %v", err)
		}
	}

	if config.Verbose {
		if provider != "" {
			log.Printf("Setting up Kubeconfig from `%s` outputs", provider)
			if config.Debug {
				parameters.PrintCapturedOutputsByComponent(outputs, provider)
			}
		} else {
			log.Printf("Setting up Kubeconfig from stack parameters")
		}
	}

	filenameBase := filepath.Join(filepath.Dir(configFilename), strings.Replace(domain, ".", "-", -1))
	caCertFile := filenameBase + kubernetesApiKeysFileSuf[kubernetesApiCaCertOutput]
	clientCertFile := filenameBase + kubernetesApiKeysFileSuf[kubernetesApiClientCertOutput]
	clientKeyFile := filenameBase + kubernetesApiKeysFileSuf[kubernetesApiClientKeyOutput]

	var caCert string
	caCertExist := true
	if flavor != "openshift" {
		caCert = mustOutput(params, outputs, provider, kubernetesApiCaCertOutput)
	} else {
		caCert, caCertExist = mayOutput(params, outputs, provider, kubernetesApiCaCertOutput)
	}
	var pemsWritten []string
	if caCertExist {
		writeFile(caCertFile, caCert)
		pemsWritten = append(pemsWritten, caCertFile)
	}
	if util.Contains([]string{"k8s-aws", "hybrid", "metal"}, flavor) {
		writeFile(clientCertFile,
			mustOutput(params, outputs, provider, kubernetesApiClientCertOutput))
		writeFile(clientKeyFile,
			mustOutput(params, outputs, provider, kubernetesApiClientKeyOutput))
		pemsWritten = append(pemsWritten, clientCertFile, clientKeyFile)
	}

	apiEndpoint := domain
	if endpoint, exist := mayOutput(params, outputs, provider, kubernetesApiEndpointOutput); exist && endpoint != "" {
		apiEndpoint = endpoint
	}

	clusterArgs := []string{"config", "set-cluster", domain, "--server=https://" + apiEndpoint}
	if caCertExist {
		clusterArgs = append(clusterArgs, "--embed-certs=true", "--certificate-authority="+caCertFile)
	}
	mustExec(kubectl, clusterArgs...)
	user := ""
	switch flavor {
	case "k8s-aws", "hybrid", "metal":
		user = "admin@" + domain
		mustExec(kubectl, "config", "set-credentials", user,
			"--embed-certs=true",
			"--client-key="+clientKeyFile,
			"--client-certificate="+clientCertFile)

	case "eks":
		user = "eks-" + eksClusterName
		if bearerToken != "" {
			mustExec(kubectl, "config", "set-credentials", user,
				"--token="+bearerToken)
			mustExec(kubectl, "config", "unset", fmt.Sprintf("users.%s.exec", user))
		} else {
			mustExec(kubectl, "config", "set-credentials", user,
				"--exec-api-version=client.authentication.k8s.io/v1alpha1",
				"--exec-command=aws-iam-authenticator",
				"--exec-arg=token",
				"--exec-arg=-i",
				"--exec-arg="+eksClusterName)
			mustExec(kubectl, "config", "unset", fmt.Sprintf("users.%s.token", user))
		}

	case "openshift", "gke", "aks":
		user = fmt.Sprintf("%s-%s", flavor, domain)
		mustExec(kubectl, "config", "set-credentials", user,
			"--token="+bearerToken)
	}
	mustExec(kubectl, "config", "set-context", context,
		"--cluster="+domain,
		"--user="+user,
		"--namespace=kube-system")
	switchContext := config.SwitchKubeconfigContext
	if os.Getenv("HUB_KUBECONFIG") != "" {
		kubeconfig := os.Getenv("HUB_KUBECONFIG")
		os.Setenv("KUBECONFIG", kubeconfig)
	}

	if !switchContext && os.Getenv("KUBECONFIG") != "" {
		// Hub CTL extensions expects a private Kubeconfig with current-context set
		outBytes, err := execOutput(kubectl, "config", "current-context")
		out := string(outBytes)
		if strings.Contains(out, "current-context is not set") {
			switchContext = true
		}
		if !switchContext && err != nil {
			if processErr, ok := err.(*exec.ExitError); ok && processErr.ExitCode() == 1 {
				switchContext = true
			}
		}
	}
	if switchContext {
		mustExec(kubectl, "config", "use-context", context)
	}
	if !keepPems {
		for _, filename := range pemsWritten {
			if err := os.Remove(filename); err != nil {
				util.WarnOnce("Unable to remove `%s`: %v", filename, err)
			} else if config.Debug {
				log.Printf("Removed `%s`", filename)
			}
		}
	}
}

func mayOutput(params parameters.LockedParameters,
	outputs parameters.CapturedOutputs, component string, candidates ...string) (string, bool) {

	for _, name := range candidates {
		if component != "" {
			qName := parameters.OutputQualifiedName(name, component)
			output, exist := outputs[qName]
			if exist {
				return util.String(output.Value), true
			}
		}
		param, exist := params[name]
		if exist {
			return util.String(param.Value), true
		}
	}
	return "", false
}

func mustOutput(params parameters.LockedParameters,
	outputs parameters.CapturedOutputs, component string, candidates ...string) string {

	if value, exist := mayOutput(params, outputs, component, candidates...); exist {
		return value
	}
	if component != "" {
		log.Printf("Component `%s` provides no %v output(s), nor such stack parameters are found", component, candidates)
		if len(outputs) > 0 {
			log.Print("Outputs:")
			parameters.PrintCapturedOutputs(outputs)
		}
	} else {
		log.Printf("No %v stack parameter(s) are found", candidates)
	}
	os.Exit(1)
	return ""
}

func writeFile(filename string, content string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Unable to open `%s` for write: %v", filename, err)
	}
	wrote, err := strings.NewReader(content).WriteTo(file)
	if err != nil || wrote != int64(len(content)) {
		file.Close()
		log.Fatalf("Unable to write `%s`: %v", filename, err)
	}
	file.Close()
	if config.Debug {
		log.Printf("Wrote `%s`", filename)
	}
}

func execOutput(name string, args ...string) ([]byte, error) {
	if config.Trace {
		log.Printf("Executing %s %v", name, args)
	}
	cmd := exec.Command(name, args...)
	outBytes, err := cmd.CombinedOutput()
	if err != nil || config.Debug {
		log.Printf("%s output:\n%s", name, outBytes)
	}
	return outBytes, err
}

func mustExec(name string, args ...string) {
	_, err := execOutput(name, args...)
	if err != nil {
		log.Fatalf("%s failed: %v", name, err)
	}
}

func warnClusterCaCertMismatch(configFilename, searchContext, ca string) {
	kubeconfig, err := readKubeconfig(configFilename)
	if err != nil {
		return
	}
	for _, context := range kubeconfig.Contexts {
		if context.Name == searchContext {
			clusterName := context.Context["cluster"]
			for _, cluster := range kubeconfig.Clusters {
				if cluster.Name == clusterName {
					configCaBase64, exist := cluster.Cluster["certificate-authority-data"]
					if exist && configCaBase64 != "" {
						title := searchContext
						if searchContext != clusterName {
							title = fmt.Sprintf("%s (%s)", searchContext, clusterName)
						}
						configCa, err := base64.StdEncoding.DecodeString(configCaBase64)
						if err != nil {
							break
						}
						if string(configCa) != ca {
							util.WarnOnce(`Kubeconfig %s CA certificate doesn't match stack Kubernetes CA certificate;
	You may want to 'kubectl config delete-context %s' for Hub CTL to re-create it`,
								title, searchContext)
						} else {
							if config.Trace {
								log.Printf("Kubeconfig %s CA certificate do match stack Kubernetes CA - good", title)
							}
						}
					}
					break
				}
			}
			break
		}
	}
}
