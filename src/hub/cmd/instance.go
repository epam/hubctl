package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"hub/api"
	"hub/util"
)

var (
	instanceShowLogs bool
	kubeconfigOutput string
)

var instanceCmd = &cobra.Command{
	Use:   "instance <get | create | delete> ...",
	Short: "Create and manage Stack Instances",
}

var instanceGetCmd = &cobra.Command{
	Use:   "get [id | domain]",
	Short: "Show a list of instances or details about the instance",
	Long: `Show a list of all user accessible stack instances or details about
the particular instance (specify Id or search by full domain name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return instance(args)
	},
}

var instanceCreateCmd = &cobra.Command{
	Use:   "create < instance.json",
	Short: "Create Stack Instance",
	Long: `Create Stack Instance by sending JSON via stdin, for example:
    {
        "name": "kubernetes",
        "template": "1",
        "environment": "2",
        "tags": [],
        "parameters": [
            {
                "name": "dns.name",
                "value": "kubernetes"
            }, {
                "name": "dns.baseDomain",
                "value": "dev01.superhub.io"
            }, {
                "name": "component.postgresql.password",
                "kind": "secret",
                "value": {
                    "kind": "password",
                    "password": "qwerty123"
                }
            }
        ]
    }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createInstance(args)
	},
}

var instancePatchCmd = &cobra.Command{
	Use:   "patch <id | domain> < instance-patch.json",
	Short: "Patch Stack Instance",
	Long: `Patch Stack Instance by sending JSON via stdin, for example:
    {
        "status": {
            "status": "deployed",
            "components": [],
            "inflightOperations": []
        },
        "parameters": [
            {
                "name": "dns.name",
                "value": "kubernetes"
            }, {
                "name": "component.postgresql.password",
                "kind": "secret",
                "value": {
                    "kind": "password",
                    "password": "qwerty123"
                }
            }
        ],
        "outputs": [
            {
                "name": "component.ingress.fqdn",
                "value": "app.kubernetes.dev01.superhub.io"
            }
        ],
        "provides": {
            "kubernetes": ["stack-k8s-aws"]
        }
    }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return patchInstance(args)
	},
}

var instanceDeployCmd = &cobra.Command{
	Use:   "deploy <id | domain>",
	Short: "Deploy Stack Instance by Id or full domain name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deployInstance(args)
	},
}

var instanceUndeployCmd = &cobra.Command{
	Use:   "undeploy <id | domain>",
	Short: "Undeploy Stack Instance by Id or full domain name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return undeployInstance(args)
	},
}

var instanceSyncCmd = &cobra.Command{
	Use:   "sync <id | domain>",
	Short: "Sync Stack Instance state from state file",
	RunE: func(cmd *cobra.Command, args []string) error {
		return syncInstance(args)
	},
}

var instanceDeleteCmd = &cobra.Command{
	Use:   "delete <id | domain>",
	Short: "Delete Stack Instance by Id or full domain name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteInstance(args)
	},
}

var instanceKubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig <id | domain>",
	Short: "Download Stack Instance Kubeconfig by Id or full domain name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return kubeconfigInstance(args)
	},
}

func instance(args []string) error {
	if len(args) > 1 {
		return errors.New("Instance command has one optional argument - id or domain of the Stack Instance")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	api.StackInstances(selector, showSecrets, instanceShowLogs, jsonFormat)

	return nil
}

func createInstance(args []string) error {
	if len(args) > 0 {
		return errors.New("Create Instance command has no arguments")
	}

	api.CreateStackInstance(os.Stdin)

	return nil
}

func patchInstance(args []string) error {
	if len(args) != 1 {
		return errors.New("Patch Instance command has one mandatory argument - id or full domain name of the Instance")
	}

	selector := args[0]
	if patchRaw {
		api.RawPatchStackInstance(selector, os.Stdin, patchReplace)
	} else {
		patchBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil || len(patchBytes) < 3 {
			return fmt.Errorf("Unable to read patch data (read %d bytes): %v", len(patchBytes), err)
		}
		var patch api.StackInstancePatch
		err = json.Unmarshal(patchBytes, &patch)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal patch data: %v", err)
		}
		api.PatchStackInstanceForCmd(selector, patch, patchReplace)
	}

	return nil
}

func deployInstance(args []string) error {
	if len(args) != 1 {
		return errors.New("Deploy Instance command has one mandatory argument - id or full domain name of the Instance")
	}

	if dryRun {
		waitAndTailDeployLogs = false
	}
	api.DeployStackInstance(args[0], waitAndTailDeployLogs, dryRun)

	return nil
}

func undeployInstance(args []string) error {
	if len(args) != 1 {
		return errors.New("Undeploy Instance command has one mandatory argument - id or full domain name of the Instance")
	}

	api.UndeployStackInstance(args[0], waitAndTailDeployLogs)

	return nil
}

func syncInstance(args []string) error {
	if len(args) != 1 && len(args) != 2 {
		return errors.New("Sync Instance command has one mandatory argument - id or full domain name of the Instance, and optionally Instance status")
	}
	status := ""
	if len(args) == 2 {
		status = args[1]
	}
	if stateManifestExplicit == "" && status == "" {
		return errors.New("Either status or state file to sync must be specified")
	}

	manifests := util.SplitPaths(stateManifestExplicit)

	api.SyncStackInstance(args[0], status, manifests)

	return nil
}

func deleteInstance(args []string) error {
	if len(args) != 1 {
		return errors.New("Delete Instance command has one mandatory argument - id or full domain name of the Instance")
	}

	api.DeleteStackInstance(args[0])

	return nil
}

func kubeconfigInstance(args []string) error {
	if len(args) != 1 {
		return errors.New("Kubeconfig Instance command has one mandatory argument - id or full domain name of the Instance")
	}

	api.KubeconfigStackInstance(args[0], kubeconfigOutput)

	return nil
}

func init() {
	instanceGetCmd.Flags().BoolVarP(&showSecrets, "secrets", "s", false,
		"Show secrets")
	instanceGetCmd.Flags().BoolVarP(&instanceShowLogs, "logs", "l", false,
		"Show logs")
	instanceGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	instancePatchCmd.Flags().BoolVarP(&patchReplace, "replace", "", true,
		"Replace patched fields, do not merge")
	instancePatchCmd.Flags().BoolVarP(&patchRaw, "raw", "r", false,
		"Send patch data as is, do not trim non-PATCH-able API object fields")
	instanceDeployCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	instanceDeployCmd.Flags().BoolVarP(&dryRun, "dry", "y", false,
		"Save parameters and envrc to Template's Git but do not start the deployment")
	instanceUndeployCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	instanceSyncCmd.Flags().StringVarP(&stateManifestExplicit, "state", "s", "",
		"Path to state files")
	instanceKubeconfigCmd.Flags().StringVarP(&kubeconfigOutput, "output", "o", "",
		"Set output filename, `-` for stdout (default to kubeconfig-<domain>.yaml)")
	instanceCmd.AddCommand(instanceGetCmd)
	instanceCmd.AddCommand(instanceCreateCmd)
	instanceCmd.AddCommand(instancePatchCmd)
	instanceCmd.AddCommand(instanceDeployCmd)
	instanceCmd.AddCommand(instanceUndeployCmd)
	instanceCmd.AddCommand(instanceSyncCmd)
	instanceCmd.AddCommand(instanceDeleteCmd)
	instanceCmd.AddCommand(instanceKubeconfigCmd)
	apiCmd.AddCommand(instanceCmd)
}
