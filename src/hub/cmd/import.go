package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"hub/api"
	"hub/config"
	"hub/util"
)

var (
	autoCreateTemplate bool
	createNewTemplate  bool
	knownImportKinds   = []string{"k8s-aws", "eks", "metal"}
	k8sEndpoint        string
	eksClusterName     string
	eksEndpoint        string
	metalEndpoint      string
	metalIngressIp     string
)

var importCmd = &cobra.Command{
	Use:   "import <k8s-aws | eks | metal> <name or FQDN> -e <id | environment name> [-m <id | template name>] < keys.pem",
	Short: "Import Kubernetes cluster",
	Long: `Import Kubernetes cluster into Control Plane to become Platform Stack.
Currently supported cluster types are:
- k8s-aws - AgileStacks Kubernetes on AWS (stack-k8s-aws)
- eks - AWS EKS
- metal - Bare-metal

Cluster TLS auth is read from stdin in the order:
- k8s-aws, metal - Client cert, Client key, CA cert (optional).
- eks - CA cert

User-supplied FQDN must match Cloud Account's base domain.
If no FQDN is supplied, then the name is prepended to Environment's Cloud Account base domain name
to construct FQDN.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return importKubernetes(args)
	},
}

var maybeValidHostname = regexp.MustCompile("^[0-9a-z\\.-]+$")

func importKubernetes(args []string) error {
	if len(args) != 2 {
		return errors.New("Import command has two argument - type of imported Kubernetes cluster and desired cluster name")
	}
	kind := args[0]
	name := strings.ToLower(args[1])

	if !util.Contains(knownImportKinds, kind) {
		return fmt.Errorf("Kubernetes cluster kind must be one of %v", knownImportKinds)
	}

	nativeEndpoint := k8sEndpoint
	switch kind {
	case "metal":
		if metalEndpoint == "" {
			return errors.New("Bare-metal cluster API endpoint must be specified by --metal-endpoint")
		}
		nativeEndpoint = metalEndpoint

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

	config.AggWarnings = false // confusing UIX otherwise

	api.ImportKubernetes(kind, name, environmentSelector, templateSelector,
		autoCreateTemplate, createNewTemplate, waitAndTailDeployLogs,
		os.Stdin,
		nativeEndpoint, eksClusterName, metalIngressIp)

	return nil
}

func init() {
	importCmd.Flags().StringVarP(&environmentSelector, "environment", "e", "",
		"Put cluster in Environment, supply name or id")
	importCmd.Flags().StringVarP(&templateSelector, "template", "m", "",
		"Use specified adapter template, by name or id")
	importCmd.Flags().StringVarP(&k8sEndpoint, "k8s-endpoint", "", "",
		"K8S cluster API endpoint, default to api.{domain}")
	importCmd.Flags().StringVarP(&eksClusterName, "eks-cluster", "", "",
		"EKS cluster native name")
	importCmd.Flags().StringVarP(&eksEndpoint, "eks-endpoint", "", "",
		"EKS cluster API endpoint (discovered via AWS EKS API)")
	importCmd.Flags().StringVarP(&metalEndpoint, "metal-endpoint", "", "",
		"Bare-metal cluster API endpoint (ip[:port])")
	importCmd.Flags().StringVarP(&metalIngressIp, "metal-ingress-ip", "", "",
		"Bare-metal cluster static ingress IP (default to IP of endpoint)")
	importCmd.Flags().BoolVarP(&autoCreateTemplate, "create-template", "", true,
		"Create adapter template if no existing template is found for reuse")
	importCmd.Flags().BoolVarP(&createNewTemplate, "create-new-template", "", false,
		"Do not reuse existing template, always create fresh one")
	importCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	apiCmd.AddCommand(importCmd)
}
