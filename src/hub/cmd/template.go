package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"hub/api"
)

var (
	templateShowSecretGitRemote bool
	templateWildcardSecret      bool
	templateShowGitStatus       bool
)

var templateCmd = &cobra.Command{
	Use:   "template <get | create | delete> ...",
	Short: "Create and manage Stack Templates",
}

var templateGetCmd = &cobra.Command{
	Use:   "get [id | name]",
	Short: "Show a list of templates or details about the template",
	Long: `Show a list of all user accessible templates or details about
the particular template (specify Id or search by name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return template(args)
	},
}

var templateCreateCmd = &cobra.Command{
	Use:   "create < template.json",
	Short: "Create Stack Template",
	Long: `Create Stack Template by sending JSON via stdin, for example:
    {
        "name": "EKS",
        "description": "EKS with Terraform",
        "stack": "eks:1",
        "componentsEnabled": ["stack-k8s-eks", "tiller", "traefik", "dex", "kube-dashboard"],
        "verbs": ["deploy", "undeploy"],
        "tags": [],
        "parameters": [{
            "name": "dns.domain"
        }, {
            "name": "component.kubernetes.eks.cluster"
        }, {
            "name": "component.kubernetes.eks.admin"
        }, {
            "name": "component.kubernetes.eks.availabilityZones"
        }, {
            "name": "component.kubernetes.worker.count",
            "value": 3
        }, {
            "name": "component.kubernetes.worker.size",
            "value": "r5a.large"
        }, {
            "name": "component.kubernetes.worker.spotPrice",
            "value": 0.06
        }, {
            "name": "component.ingress.urlPrefix",
            "value": "app"
        }, {
            "name": "component.ingress.ssoUrlPrefix",
            "value": "apps"
        }, {
            "name": "component.ingress.ssl.enabled",
            "value": "false"
        }],
        "teamsPermissions": []
    }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createTemplate(args)
	},
}

var templateInitCmd = &cobra.Command{
	Use:   "init <id | name>",
	Short: "Initialize Stack Template by Id or name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return initTemplate(args)
	},
}

var templateDeleteCmd = &cobra.Command{
	Use:   "delete <id | name>",
	Short: "Delete Stack Template by Id or name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteTemplate(args)
	},
}

func template(args []string) error {
	if len(args) > 1 {
		return errors.New("Template command has one optional argument - id or name of the template")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	if templateWildcardSecret {
		templateShowSecretGitRemote = true
	}
	api.Templates(selector, showSecrets,
		templateShowSecretGitRemote, templateWildcardSecret, templateShowGitStatus, jsonFormat)

	return nil
}

func createTemplate(args []string) error {
	if len(args) > 0 {
		return errors.New("Create Template command has no arguments")
	}

	api.CreateTemplate(os.Stdin)

	return nil
}

func initTemplate(args []string) error {
	if len(args) != 1 {
		return errors.New("Init Template command has one mandator argument - id or name of the template")
	}

	api.InitTemplate(args[0])

	return nil
}

func deleteTemplate(args []string) error {
	if len(args) != 1 {
		return errors.New("Delete Template command has one mandatory argument - id or name of the template")
	}

	api.DeleteTemplate(args[0])

	return nil
}

func init() {
	templateGetCmd.Flags().BoolVarP(&showSecrets, "secrets", "", false,
		"Show secrets")
	templateGetCmd.Flags().BoolVarP(&templateShowSecretGitRemote, "git-secret", "g", false,
		"Output template secret Git remote")
	templateGetCmd.Flags().BoolVarP(&templateWildcardSecret, "git-wildcard-secret", "", false,
		"Request a secret which is not template specific")
	templateGetCmd.Flags().BoolVarP(&templateShowGitStatus, "git-status", "s", false,
		"Output template Git ref/heads/master status")
	templateGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	templateCmd.AddCommand(templateGetCmd)
	templateCmd.AddCommand(templateCreateCmd)
	templateCmd.AddCommand(templateInitCmd)
	templateCmd.AddCommand(templateDeleteCmd)
	apiCmd.AddCommand(templateCmd)
}
