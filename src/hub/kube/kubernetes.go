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
	kubernetesDomainOutput      = "dns.domain"
	kubernetesFlavorOutput      = "kubernetes.flavor"
	kubernetesApiEndpoint       = "kubernetes.api.endpoint"
	KubernetesApiKeysOutputBase = "kubernetes.api."
	kubernetesEksClusterOutput  = "kubernetes.eks.cluster"
	kubernetesFileRefPrefix     = "file:"
)

var (
	KubernetesDefaultProviders = []string{
		"kubernetes", "stack-k8s-aws", "stack-k8s-eks", "k8s-aws", "k8s-eks", "k8s-gke", "k8s-metal"}
	kubernetesApiKeysOutputs = []string{"caCert", "clientCert", "clientKey"}
	kubernetesApiKeysFileSuf = map[string]string{
		"caCert":     "-ca.pem",
		"clientCert": "-client.pem",
		"clientKey":  "-client-key.pem",
	}
)

func RequiredKubernetesParameters() []string {
	return append(RequiredKubernetesKeysParameters(), kubernetesApiEndpoint)
}

func RequiredKubernetesKeysParameters() []string {
	params := make([]string, 0, len(kubernetesApiKeysOutputs)+1)
	for _, name := range kubernetesApiKeysOutputs {
		params = append(params, KubernetesApiKeysOutputBase+name)
	}
	return params
}

func CaptureKubernetes(component *manifest.ComponentRef, stackBaseDir string, componentsBaseDir string,
	componentOutputs parameters.CapturedOutputs) parameters.CapturedOutputs {

	outputs := make(parameters.CapturedOutputs)

	componentName := manifest.ComponentQualifiedNameFromRef(component)
	flavor := "k8s-aws"
	if o, exist := componentOutputs[parameters.OutputQualifiedName(kubernetesFlavorOutput, componentName)]; exist {
		flavor = o.Value
	}

	kubernetesApiKeysFiles := make(map[string]string)
	for _, outputSuf := range kubernetesApiKeysOutputs {
		outputName := parameters.OutputQualifiedName(KubernetesApiKeysOutputBase+outputSuf, componentName)
		output, exists := componentOutputs[outputName]
		if !exists {
			continue
		}
		kubernetesApiKeysFiles[outputSuf] = output.Value
	}

	if len(kubernetesApiKeysFiles) != len(kubernetesApiKeysOutputs) {
		if flavor == "eks" {
			caQName := parameters.OutputQualifiedName(KubernetesApiKeysOutputBase+"caCert", componentName)
			_, exists := componentOutputs[caQName]
			if !exists {
				log.Printf("Component `%s` declared to provide EKS Kubernetes but no `%s` output found", componentName, caQName)
				if len(componentOutputs) > 0 {
					log.Print("Outputs:")
					parameters.PrintCapturedOutputs(componentOutputs)
				}
				return outputs
			}
		} else {
			if config.Verbose {
				log.Printf("Component `%s` declared to provide Kubernetes but some key/certs output(s) are missing",
					componentName)
				if config.Debug && len(kubernetesApiKeysFiles) > 0 {
					log.Print("Outputs found:")
					util.PrintMap(kubernetesApiKeysFiles)
				}
			}

			domainQName := parameters.OutputQualifiedName(kubernetesDomainOutput, componentName)
			domain, exists := componentOutputs[domainQName]
			if !exists {
				log.Printf("Component `%s` declared to provide Kubernetes but no `%s` output found", componentName, domainQName)
				if len(componentOutputs) > 0 {
					log.Print("Outputs:")
					parameters.PrintCapturedOutputs(componentOutputs)
				}
				return outputs
			}

			dir := manifest.ComponentSourceDirFromRef(component, stackBaseDir, componentsBaseDir)
			dir = filepath.Join(dir, filepath.FromSlash(fmt.Sprintf(".terraform/%s/.terraform/", domain.Value)))
			_, err := os.Stat(dir)
			if err != nil {
				if !util.NoSuchFile(err) {
					util.MaybeFatalf("Unable to capture Kubernetes key and cert files from `%s`: %v", dir, err)
				} else {
					if config.Verbose {
						log.Printf("Kubernetes keys output directory `%s` not found: %v; skipping key and cert files capture",
							dir, err)
					}
				}
				return outputs
			}

			filenameBase := filepath.Join(dir, kubernetesFileRefPrefix+strings.Replace(domain.Value, ".", "-", -1))
			for k, suf := range kubernetesApiKeysFileSuf {
				kubernetesApiKeysFiles[k] = filenameBase + suf
			}
		}
	}

	for k, apiKey := range kubernetesApiKeysFiles {
		content := apiKey
		if strings.HasPrefix(apiKey, kubernetesFileRefPrefix) {
			filename := apiKey[len(kubernetesFileRefPrefix):]
			content = captureFile(filename)
			// replace file:// with actual content
			parameters.MergeOutput(outputs,
				parameters.CapturedOutput{
					Component: componentName,
					Name:      KubernetesApiKeysOutputBase + k,
					Value:     content,
					Kind:      "secret",
				})
		}
	}

	return outputs
}

func captureFile(filename string) string {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Unable to read %s: %v", filename, err)
	}
	return string(bytes)
}

func SetupKubernetes(params parameters.LockedParameters,
	provider string, outputs parameters.CapturedOutputs,
	context string, overwrite bool) {

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
	if flavor == "eks" {
		eksClusterName = mustOutput(params, outputs, provider, kubernetesEksClusterOutput)
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
			if ca, exist := mayOutput(params, outputs, provider, KubernetesApiKeysOutputBase+"caCert"); exist && ca != "" {
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

	caCert := filenameBase + "-ca.pem"
	clientCert := filenameBase + "-client.pem"
	clientKey := filenameBase + "-client-key.pem"
	writeFile(caCert, mustOutput(params, outputs, provider, KubernetesApiKeysOutputBase+"caCert"))
	if flavor != "eks" {
		writeFile(clientCert, mustOutput(params, outputs, provider, KubernetesApiKeysOutputBase+"clientCert"))
		writeFile(clientKey, mustOutput(params, outputs, provider, KubernetesApiKeysOutputBase+"clientKey"))
	}

	apiEndpoint := domain
	if endpoint, exist := mayOutput(params, outputs, provider, kubernetesApiEndpoint); exist && endpoint != "" {
		apiEndpoint = endpoint
	}

	mustExec(kubectl, "config", "set-cluster", domain,
		"--embed-certs=true",
		"--server=https://"+apiEndpoint,
		"--certificate-authority="+caCert)
	user := ""
	if flavor == "eks" {
		user = "eks-" + eksClusterName
		addHeptioUser(configFilename, user, eksClusterName)
		// Pending issue
		// Add cli support to "exec" auth plugin https://github.com/kubernetes/kubernetes/issues/64751
		/*
			mustExec(kubectl, "config", "set-credentials", user,
				"--exec-command=heptio-authenticator-aws",
				"--exec-arg=token",
				"--exec-arg=-i",
				"--exec-arg="+eksClusterName)
		*/
	} else {
		user = "admin@" + domain
		mustExec(kubectl, "config", "set-credentials", user,
			"--embed-certs=true",
			"--client-key="+clientKey,
			"--client-certificate="+clientCert)
	}
	mustExec(kubectl, "config", "set-context", context,
		"--cluster="+domain,
		"--user="+user,
		"--namespace=kube-system")
	if config.SwitchKubeconfigContext {
		mustExec(kubectl, "config", "use-context", context)
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
	if config.Verbose {
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
