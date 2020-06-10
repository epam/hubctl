package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/api"
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
	Short: "Show a list of Stack  Templates or details about the template",
	Long: `Show a list of all user accessible Stack Templates or details about
the particular Template (specify Id or search by name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return template(args)
	},
}

var templateCreateCmd = &cobra.Command{
	Use:   "create < template.json",
	Short: "Create Stack Template",
	Long: fmt.Sprintf(`Create Stack Template by sending JSON via stdin, for example:
%[1]s
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
	}
%[1]s`, mdpre),
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

var templatePatchCmd = &cobra.Command{
	Use:   "patch <id | name> < template-patch.json",
	Short: "Patch Stack Template",
	Long: fmt.Sprintf(`Patch Template by sending JSON via stdin, for example:
%[1]s
	{
		"description": "",
		"verbs": [
			"deploy",
			"undeploy"
		],
		"tags": [
			"kind=platform"
		],
		"componentsEnabled": [
			"flannel",
			"traefik",
			"dex",
			"cluster-autoscaler",
			"cert-manager",
			"kube-dashboard"
		],
		"parameters": [
			{
				"name": "component.kubernetes.worker.count",
				"value": 1
			}
		],
		"teamsPermissions": []
	}
%[1]s`, mdpre),
	RunE: func(cmd *cobra.Command, args []string) error {
		return patchTemplate(args)
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
		return errors.New("Create Stack Template command has no arguments")
	}

	if createRaw {
		api.RawCreateTemplate(os.Stdin)
	} else {
		createBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil || len(createBytes) < 3 {
			return fmt.Errorf("Unable to read data (read %d bytes): %v", len(createBytes), err)
		}
		var req api.StackTemplateRequest
		err = json.Unmarshal(createBytes, &req)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal data: %v", err)
		}
		api.CreateTemplate(req)
	}

	return nil
}

func initTemplate(args []string) error {
	if len(args) != 1 {
		return errors.New("Init Stack Template command has one mandator argument - id or name of the template")
	}

	api.InitTemplate(args[0])

	return nil
}

func patchTemplate(args []string) error {
	if len(args) != 1 {
		return errors.New("Patch Stack Template command has one mandatory argument - id or name of the Template")
	}

	selector := args[0]
	if patchRaw {
		api.RawPatchTemplate(selector, os.Stdin)
	} else {
		patchBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil || len(patchBytes) < 3 {
			return fmt.Errorf("Unable to read patch data (read %d bytes): %v", len(patchBytes), err)
		}
		var patch api.StackTemplatePatch
		err = json.Unmarshal(patchBytes, &patch)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal patch data: %v", err)
		}
		api.PatchTemplate(selector, patch)
	}

	return nil
}

func deleteTemplate(args []string) error {
	if len(args) != 1 {
		return errors.New("Delete Stack Template command has one mandatory argument - id or name of the template")
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
	templateCreateCmd.Flags().BoolVarP(&createRaw, "raw", "r", false,
		"Send data as is, do not trim non-POST-able API object fields")
	templatePatchCmd.Flags().BoolVarP(&patchRaw, "raw", "r", false,
		"Send patch data as is, do not trim non-PATCH-able API object fields")
	templateCmd.AddCommand(templateGetCmd)
	templateCmd.AddCommand(templateCreateCmd)
	templateCmd.AddCommand(templateInitCmd)
	templateCmd.AddCommand(templatePatchCmd)
	templateCmd.AddCommand(templateDeleteCmd)
	apiCmd.AddCommand(templateCmd)
}
