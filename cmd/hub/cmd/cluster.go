package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/api"
	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var (
	autoCreateTemplate bool
	createNewTemplate  bool
	knownCreateKinds   = []string{"k8s-aws", "eks", "gke", "aks"}
	knownImportKinds   = []string{"k8s-aws", "eks", "gke", "aks", "metal", "hybrid", "openshift"}
	clusterRegion      string
	clusterZone        string
	k8sEndpoint        string
	eksClusterName     string
	eksEndpoint        string
	eksAdmin           string
	gkeClusterName     string
	aksClusterName     string
	azureResourceGroup string
	metalEndpoint      string
	metalIngress       string
	bearerToken        string

	spotPrice      float32
	preemptibleVMs bool
	volumeSize     int

	acm               bool
	certManager       bool
	autoscaler        bool
	kubeDashboard     bool
	kubeDashboardMode string
)

var clusterCmd = &cobra.Command{
	Use:   "cluster <create | import> ...",
	Short: "Create or import Kubernetes cluster",
}

var createClusterCmd = &cobra.Command{
	Use: fmt.Sprintf("create <%s> <name or FQDN> <instance type> <count> [max count] -e <id | environment name> [-m <id | template name>]",
		strings.Join(knownCreateKinds, " | ")),
	Short: "Create Kubernetes cluster",
	Long: `Create Kubernetes cluster as SuperHub Platform Stack.

Currently supported cluster types are:
- k8s-aws - Agile Stacks Kubernetes on AWS
- eks - AWS EKS
- gke - GCP GKE
- aks - Azure AKS

User-supplied FQDN must match Cloud Account's base domain.
If no FQDN is supplied, then the name is prepended to Environment's Cloud Account base domain name
to construct FQDN.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createKubernetes(args)
	},
}

var importClusterCmd = &cobra.Command{
	Use: fmt.Sprintf("import <%s> <name or FQDN> -e <id | environment name> [-m <id | template name>] < keys.pem",
		strings.Join(knownImportKinds, " | ")),
	Short: "Import Kubernetes cluster",
	Long: `Import Kubernetes cluster into SuperHub to become Platform Stack.

Currently supported cluster types are:
- k8s-aws - Agile Stacks Kubernetes on AWS
- eks - AWS EKS
- gke - GCP GKE
- aks - Azure AKS
- metal - Bare-metal
- hybrid - Hybrid bare-metal
- openshift - OpenShift on AWS

Cluster TLS auth is read from stdin in the order:
- k8s-aws, hybrid, metal - Client cert, Client key, CA cert (optional).
- eks - CA cert, optional if --eks-endpoint is omited, then it will be discovered via AWS API
- openshift - optional CA cert
GKE and AKS certificates are discovered by import adapter component.

User-supplied FQDN must match Cloud Account's base domain.
If no FQDN is supplied, then the name is prepended to Environment's Cloud Account base domain name
to construct FQDN.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return importKubernetes(args)
	},
}

var maybeValidHostname = regexp.MustCompile("^[0-9a-z\\.-]+$")

func createKubernetes(args []string) error {
	// create <name or FQDN> <instance type> <count> [max count]
	if len(args) < 4 || len(args) > 5 {
		return errors.New("Create cluster command has several arguments - type of Kubernetes cluster, desired cluster name/domain, instance type, and node count")
	}

	kind := args[0]
	name := strings.ToLower(args[1])
	instanceType := args[2]
	count, err := strconv.ParseInt(args[3], 10, 32)
	if err != nil {
		return fmt.Errorf("Unable to parse count: %v", err)
	}
	maxCount := int64(0)
	if len(args) > 4 {
		maxCount, err = strconv.ParseInt(args[4], 10, 32)
		if err != nil {
			return fmt.Errorf("Unable to parse max count: %v", err)
		}
	}

	if !util.Contains(knownCreateKinds, kind) {
		return fmt.Errorf("Kubernetes cluster kind must be one of %v", knownCreateKinds)
	}

	nativeClusterName := ""
	switch kind {
	case "eks":
		if eksClusterName == "" {
			if strings.Contains(name, ".") {
				return errors.New("EKS cluster name (--eks-cluster) must be provided")
			} else {
				log.Printf("Setting --eks-cluster=%s", name)
				eksClusterName = name
			}
		}
		nativeClusterName = eksClusterName
		if eksAdmin == "" {
			return errors.New("EKS cluster admin IAM user (--eks-admin) must be provided")
		}

	case "gke":
		if gkeClusterName == "" {
			if strings.Contains(name, ".") {
				return errors.New("GKE cluster name (--gke-cluster) must be provided")
			} else {
				log.Printf("Setting --gke-cluster=%s", name)
				gkeClusterName = name
			}
		}
		nativeClusterName = gkeClusterName

	case "aks":
		if aksClusterName == "" {
			if strings.Contains(name, ".") {
				return errors.New("AKS cluster name (--aks-cluster) must be provided")
			} else {
				log.Printf("Setting --aks-cluster=%s", name)
				aksClusterName = name
			}
		}
		nativeClusterName = aksClusterName
		if azureResourceGroup == "" {
			log.Printf("Azure resource group name (--azure-resource-group) not be provided - using default Cloud Account resource group")
		}
	}

	if !maybeValidHostname.MatchString(name) {
		return fmt.Errorf("`%s` doesn't look like a valid hostname", name)
	}

	if environmentSelector == "" {
		return errors.New("Environment name or id must be specified by --environment / -e")
	}

	if dryRun {
		waitAndTailDeployLogs = false
	}

	config.AggWarnings = false // confusing UIX otherwise

	api.CreateKubernetes(kind, name, environmentSelector, templateSelector,
		autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs, dryRun,
		clusterRegion, clusterZone, nativeClusterName,
		eksAdmin, azureResourceGroup,
		api.ClusterOptions{InstanceType: instanceType, Count: int(count), MaxCount: int(maxCount),
			SpotPrice: spotPrice, PreemptibleVMs: preemptibleVMs, VolumeSize: volumeSize,
			Acm: acm, CertManager: certManager, Autoscaler: autoscaler,
			KubeDashboard: kubeDashboard, KubeDashboardMode: kubeDashboardMode})

	return nil
}

func importKubernetes(args []string) error {
	if len(args) != 2 {
		return errors.New("Import command has two argument - type of imported Kubernetes cluster and desired cluster name/domain")
	}
	kind := args[0]
	name := strings.ToLower(args[1])

	if !util.Contains(knownImportKinds, kind) {
		return fmt.Errorf("Kubernetes cluster kind must be one of %v", knownImportKinds)
	}

	nativeEndpoint := ""
	nativeClusterName := ""
	switch kind {
	case "k8s-aws":
		if k8sEndpoint == "" {
			return errors.New("AgileStacks K8S cluster API endpoint must be specified by --k8s-endpoint")
		}
		nativeEndpoint = k8sEndpoint

	case "hybrid":
		if metalEndpoint == "" {
			return errors.New("Hybrid bare-metal cluster API endpoint must be specified by --metal-endpoint")
		}
		nativeEndpoint = metalEndpoint

	case "metal":
		if metalEndpoint == "" {
			return errors.New("Bare-metal cluster API endpoint must be specified by --metal-endpoint")
		}
		nativeEndpoint = metalEndpoint

	case "openshift":
		if bearerToken == "" {
			return errors.New("OpenShift authentication must be specified with --bearer-token")
		}

	case "eks":
		if eksClusterName == "" {
			if strings.Contains(name, ".") {
				return errors.New("EKS cluster name (--eks-cluster) must be provided")
			} else {
				log.Printf("Setting --eks-cluster=%s", name)
				eksClusterName = name
			}
		}
		nativeEndpoint = eksEndpoint
		nativeClusterName = eksClusterName

	case "gke":
		if gkeClusterName == "" {
			if strings.Contains(name, ".") {
				return errors.New("GKE cluster name (--gke-cluster) must be provided")
			} else {
				log.Printf("Setting --gke-cluster=%s", name)
				gkeClusterName = name
			}
		}
		nativeClusterName = gkeClusterName

	case "aks":
		if aksClusterName == "" {
			if strings.Contains(name, ".") {
				return errors.New("AKS cluster name (--aks-cluster) must be provided")
			} else {
				log.Printf("Setting --aks-cluster=%s", name)
				aksClusterName = name
			}
		}
		nativeClusterName = aksClusterName
		if azureResourceGroup == "" {
			log.Printf("Azure resource group name (--azure-resource-group) not be provided - using default Cloud Account resource group")
		}
	}
	if len(nativeEndpoint) >= 8 && strings.HasPrefix(nativeEndpoint, "https://") {
		nativeEndpoint = nativeEndpoint[8:]
	}

	if !maybeValidHostname.MatchString(name) {
		return fmt.Errorf("`%s` doesn't look like a valid hostname", name)
	}

	if environmentSelector == "" {
		return errors.New("Environment name or id must be specified by --environment / -e")
	}

	// TODO review interaction of these options
	// if templateSelector != "" {
	// 	autoCreateTemplate = false
	// }

	// if createNewTemplate && templateSelector != "" {
	// 	return fmt.Errorf("If --template is specified then omit --create-new-template")
	// }

	if dryRun {
		waitAndTailDeployLogs = false
	}

	config.AggWarnings = false

	api.ImportKubernetes(kind, name, environmentSelector, templateSelector,
		autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs, dryRun,
		os.Stdin, bearerToken,
		clusterRegion, clusterZone, nativeEndpoint, nativeClusterName,
		metalIngress, azureResourceGroup,
		api.ClusterOptions{Acm: acm, CertManager: certManager, Autoscaler: autoscaler,
			KubeDashboard: kubeDashboard, KubeDashboardMode: kubeDashboardMode})

	return nil
}

func init() {
	createClusterCmd.Flags().StringVarP(&environmentSelector, "environment", "e", "",
		"Put cluster in Environment, supply name or id")
	createClusterCmd.Flags().StringVarP(&templateSelector, "template", "m", "",
		"Use specified template, by name or id")
	createClusterCmd.Flags().StringVarP(&clusterRegion, "region", "", "",
		"Cloud region if different from Cloud Account default region")
	createClusterCmd.Flags().StringVarP(&clusterZone, "zone", "", "",
		"Cloud zone if different from Cloud Account default zone")
	createClusterCmd.Flags().StringVarP(&eksAdmin, "eks-admin", "", "",
		"Set AWS EKS cluster admin IAM user https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html")
	createClusterCmd.Flags().StringVarP(&eksClusterName, "eks-cluster", "", "",
		"Set AWS EKS cluster native name")
	createClusterCmd.Flags().StringVarP(&gkeClusterName, "gke-cluster", "", "",
		"Set GCP GKE cluster native name")
	createClusterCmd.Flags().StringVarP(&aksClusterName, "aks-cluster", "", "",
		"Set Azure AKS cluster native name")
	createClusterCmd.Flags().StringVarP(&azureResourceGroup, "azure-resource-group", "", "",
		"Set Azure resource group name")
	createClusterCmd.Flags().BoolVarP(&autoCreateTemplate, "create-template", "", true,
		"Create adapter template if no existing template is found for reuse")
	createClusterCmd.Flags().BoolVarP(&createNewTemplate, "create-new-template", "", false,
		"Do not reuse existing template, always create fresh one (set name via --template)")
	createClusterCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	createClusterCmd.Flags().BoolVarP(&dryRun, "dry", "y", false,
		"Save parameters and envrc to Template's Git but do not start the deployment")

	createClusterCmd.Flags().Float32VarP(&spotPrice, "spot-price", "s", 0,
		"AWS use spot instances at specified spot price")
	createClusterCmd.Flags().BoolVarP(&preemptibleVMs, "preemptible-vms", "p", false,
		"GCP use preemptible VMs")
	createClusterCmd.Flags().IntVarP(&volumeSize, "volume-size", "z", 0,
		"Node root volume size (default 30GB)")

	createClusterCmd.Flags().BoolVarP(&acm, "acm", "", false,
		"Provision ACM for ingress TLS (AWS only)")
	createClusterCmd.Flags().BoolVarP(&certManager, "cert-manager", "", true,
		"Provision Cert-Manager for ingress TLS")
	createClusterCmd.Flags().BoolVarP(&autoscaler, "autoscale", "a", true,
		"Autoscale workers with cluster-autoscaler (Agile Stacks and EKS Kubernetes only)")
	createClusterCmd.Flags().BoolVarP(&kubeDashboard, "kube-dashboard", "", false,
		"Provision Kube Dashboard")
	createClusterCmd.Flags().StringVarP(&kubeDashboardMode, "kube-dashboard-mode", "", "read-only",
		"Kube Dashboard access mode, one of: read-only, admin")

	importClusterCmd.Flags().StringVarP(&environmentSelector, "environment", "e", "",
		"Put cluster in Environment, supply name or id")
	importClusterCmd.Flags().StringVarP(&templateSelector, "template", "m", "",
		"Use specified adapter template, by name or id")
	importClusterCmd.Flags().StringVarP(&clusterRegion, "region", "", "",
		"Cloud region if different from Cloud Account default region")
	importClusterCmd.Flags().StringVarP(&clusterZone, "zone", "", "",
		"Cloud zone if different from Cloud Account default zone")
	importClusterCmd.Flags().StringVarP(&k8sEndpoint, "k8s-endpoint", "", "",
		"Agile Stacks Kubernetes cluster API endpoint, default to api.{domain}")
	importClusterCmd.Flags().StringVarP(&eksClusterName, "eks-cluster", "", "",
		"AWS EKS cluster native name")
	importClusterCmd.Flags().StringVarP(&eksEndpoint, "eks-endpoint", "", "",
		"AWS EKS cluster API endpoint (discovered via AWS EKS API if cluster name is supplied)")
	importClusterCmd.Flags().StringVarP(&gkeClusterName, "gke-cluster", "", "",
		"GCP GKE cluster native name")
	importClusterCmd.Flags().StringVarP(&aksClusterName, "aks-cluster", "", "",
		"Azure AKS cluster native name")
	importClusterCmd.Flags().StringVarP(&azureResourceGroup, "azure-resource-group", "", "",
		"Azure resource group name")
	importClusterCmd.Flags().StringVarP(&metalEndpoint, "metal-endpoint", "", "",
		"Bare-metal cluster Kubernetes API endpoint (IP or hostname [:port])")
	importClusterCmd.Flags().StringVarP(&metalIngress, "metal-ingress", "", "",
		"Bare-metal cluster static ingress (IP or hostname, default to IP or hostname of the API endpoint)")
	importClusterCmd.Flags().StringVarP(&bearerToken, "bearer-token", "b", "",
		"Use Bearer token to authenticate (to the OpenShift cluster)")
	importClusterCmd.Flags().BoolVarP(&autoCreateTemplate, "create-template", "", true,
		"Create adapter template if no existing template is found for reuse")
	importClusterCmd.Flags().BoolVarP(&createNewTemplate, "create-new-template", "", false,
		"Do not reuse existing template, always create fresh one (set name via --template)")
	importClusterCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	importClusterCmd.Flags().BoolVarP(&dryRun, "dry", "y", false,
		"Save parameters and envrc to Template's Git but do not start the import")

	importClusterCmd.Flags().BoolVarP(&acm, "acm", "", false,
		"Provision ACM for ingress TLS (AWS only)")
	importClusterCmd.Flags().BoolVarP(&certManager, "cert-manager", "", true,
		"Provision Cert-Manager for ingress TLS")
	importClusterCmd.Flags().BoolVarP(&kubeDashboard, "kube-dashboard", "", false,
		"Provision Kube Dashboard")
	importClusterCmd.Flags().StringVarP(&kubeDashboardMode, "kube-dashboard-mode", "", "read-only",
		"Kube Dashboard access mode, one of: read-only, admin")

	clusterCmd.AddCommand(createClusterCmd)
	clusterCmd.AddCommand(importClusterCmd)
	apiCmd.AddCommand(clusterCmd)
}
