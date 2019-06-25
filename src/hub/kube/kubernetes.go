package kube

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hub/config"
	"hub/manifest"
	"hub/parameters"
	"hub/util"
)

const (
	kubernetesDomainOutput        = "dns.domain"
	kubernetesFlavorOutput        = "kubernetes.flavor"
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
		"kubernetes",
		"stack-k8s-aws", "stack-k8s-eks", "stack-k8s-gke", "stack-k8s-aks",
		"k8s-aws", "k8s-eks", "k8s-gke", "k8s-aks",
		"k8s-metal", "k8s-openshift"}
	kubernetesApiKeysFileSuf = map[string]string{
		kubernetesApiCaCertOutput:     "-ca.pem",
		kubernetesApiClientCertOutput: "-client.pem",
		kubernetesApiClientKeyOutput:  "-client-key.pem",
	}
	KubernetesParameters = []string{kubernetesDomainOutput, kubernetesFlavorOutput, kubernetesApiEndpointOutput,
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
		flavor = o.Value
	}

	domainQName := parameters.OutputQualifiedName(kubernetesDomainOutput, componentName)
	if _, exists := componentOutputs[domainQName]; !exists {
		util.Warn("Component `%s` declared to provide Kubernetes but no `%s` output found", componentName, domainQName)
		if len(componentOutputs) > 0 {
			log.Print("Outputs:")
			parameters.PrintCapturedOutputs(componentOutputs)
		}
		return outputs
	}

	kubernetesApiKeysFiles := make(map[string]string)
	for _, outputName := range KubernetesKeysParameters {
		outputQName := parameters.OutputQualifiedName(outputName, componentName)
		output, exists := componentOutputs[outputQName]
		if !exists {
			continue
		}
		kubernetesApiKeysFiles[outputName] = output.Value
	}

	switch flavor {
	case "k8s-aws":
		if len(kubernetesApiKeysFiles) != len(KubernetesKeysParameters) {
			util.Warn("Component `%s` declared to provide Kubernetes but some key/certs output(s) are missing", componentName)
			if config.Debug && len(kubernetesApiKeysFiles) > 0 {
				log.Print("Required key/certs outputs found:")
				util.PrintMap(kubernetesApiKeysFiles)
			}
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

	// TODO deprecate
	for key, apiKey := range kubernetesApiKeysFiles {
		if strings.HasPrefix(apiKey, kubernetesFileRefPrefix) {
			filename := apiKey[len(kubernetesFileRefPrefix):]
			content := captureFile(filename)
			// replace file:// with actual content
			parameters.MergeOutput(outputs,
				parameters.CapturedOutput{
					Component: componentName,
					Name:      key,
					Value:     content,
					Kind:      "secret",
				})
		}
	}

	return outputs
}

// TODO deprecate
func captureFile(filename string) string {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Unable to read %s: %v", filename, err)
	}
	return string(bytes)
}

func SetupKubernetes(params parameters.LockedParameters,
	provider string, outputs parameters.CapturedOutputs,
	context string, overwrite, keepPems bool) {

	kubectl := "kubectl"

	domain := mustOutput(params, outputs, provider, kubernetesDomainOutput)
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
			log.Printf("Changing Kubeconfig context to %s", context)
		}
		configCmd = "use-context"
	} else {
		if config.Verbose {
			log.Printf("Checking Kubeconfig context %s", context)
		}
		configCmd = "get-contexts"
	}
	cmd := exec.Command(kubectl, "config", configCmd, context)
	outBytes, err := cmd.CombinedOutput()
	if err != nil || config.Debug {
		log.Printf("kubectl output:\n%s", outBytes)
	}
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
	if util.Contains([]string{"k8s-aws", "metal"}, flavor) {
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
	case "k8s-aws", "metal":
		user = "admin@" + domain
		mustExec(kubectl, "config", "set-credentials", user,
			"--embed-certs=true",
			"--client-key="+clientKeyFile,
			"--client-certificate="+clientCertFile)

	case "eks":
		user = "eks-" + eksClusterName
		addHeptioUser(configFilename, user, eksClusterName)
		// TODO
		// Add cli support to "exec" auth plugin
		// https://github.com/kubernetes/kubernetes/issues/64751
		// https://github.com/kubernetes/kubernetes/pull/73230
		/*
			mustExec(kubectl, "config", "set-credentials", user,
				"--exec-command=heptio-authenticator-aws",
				"--exec-arg=token",
				"--exec-arg=-i",
				"--exec-arg="+eksClusterName)
		*/

	case "openshift", "gke", "aks":
		user = fmt.Sprintf("%s-%s", flavor, domain)
		mustExec(kubectl, "config", "set-credentials", user,
			"--token="+bearerToken)
	}
	mustExec(kubectl, "config", "set-context", context,
		"--cluster="+domain,
		"--user="+user,
		"--namespace=kube-system")
	if config.SwitchKubeconfigContext {
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
				return output.Value, true
			}
		}
		param, exist := params[name]
		if exist {
			return param.Value, true
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

func mustExec(name string, args ...string) {
	cmd := exec.Command(name, args...)
	outBytes, err := cmd.CombinedOutput()
	if err != nil || config.Debug {
		log.Printf("%s output:\n%s", name, outBytes)
	}
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
	You may want to 'kubectl config delete-context %s' for Hub CLI to re-create it`,
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
