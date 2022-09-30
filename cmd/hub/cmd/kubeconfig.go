// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/kube"
)

var (
	kubeconfigProviderComponent string
	kubeconfigContextName       string
	kubeconfigKeepPems          bool
)

var kubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig hub.yaml.state[,s3://bucket/hub.yaml.state] [more.yaml.state]",
	Short: "Create ~/.kube/config entry from state file",
	Long: `Create or update kubectl context with Kubernetes key files stored in state file.
The context name is stack FQDN, ie. dns.domain.
State files are separate command line arguments or separated by comma - to match
deploy -s / --state syntax.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return kubeconfig(args)
	},
}

func kubeconfig(args []string) error {
	if len(args) < 1 {
		return errors.New("Kubeconfig command has one or more arguments - paths to State files, possibly on S3")
	}

	stateFiles := make([]string, 0, len(args))
	for _, arg := range args {
		files := strings.Split(arg, ",")
		stateFiles = append(stateFiles, files...)
	}
	providerComponents := kube.KubernetesDefaultProviders
	if kubeconfigProviderComponent != "" {
		providerComponents = strings.Split(kubeconfigProviderComponent, ",")
	}

	kube.Kubeconfig(stateFiles, providerComponents, kubeconfigContextName, kubeconfigKeepPems)

	return nil
}

func init() {
	providers := strings.Join(kube.KubernetesDefaultProviders, ", ")
	kubeconfigCmd.Flags().StringVarP(&kubeconfigProviderComponent, "providers", "p", "",
		fmt.Sprintf("Component name(s) providing Kubernetes outputs (default to %s)", providers))
	kubeconfigCmd.Flags().StringVarP(&kubeconfigContextName, "context", "c", "",
		"Context name (default to dns.domain output of kubernetes component)")
	kubeconfigCmd.Flags().BoolVarP(&kubeconfigKeepPems, "keep-tls-pems", "", false,
		"Keep TLS PEM files")
	kubeconfigCmd.Flags().BoolVarP(&config.SwitchKubeconfigContext, "switch-kube-context", "k", false,
		"Switch current Kubeconfig context to new context")
	RootCmd.AddCommand(kubeconfigCmd)
}
