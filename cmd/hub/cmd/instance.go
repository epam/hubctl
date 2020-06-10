package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/api"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var (
	kubeconfigOutput         string
	logsOutput               string
	workerpoolSpotPrice      float32
	workerpoolPreemptibleVMs bool
	workerpoolAutoscale      bool
	workerpoolVolumeSize     int
	workerpoolDelete         bool
	workerpoolInstanceType   string
)

var instanceCmd = &cobra.Command{
	Use:   "instance <get | create | patch | deploy | undeploy | delete> ...",
	Short: "Create and manage Stack Instances",
}

var instanceGetCmd = &cobra.Command{
	Use:   "get [id | domain]",
	Short: "Show a list of stack instances or details about the instance",
	Long: `Show a list of all user accessible stack instances or details about
the particular instance (specify Id or search by full domain name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return instance(args)
	},
}

var instanceCreateCmd = &cobra.Command{
	Use:   "create < instance.json",
	Short: "Create Stack Instance",
	Long: fmt.Sprintf(`Create Stack Instance by sending JSON via stdin, for example:
%[1]s
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
	}
%[1]s`, mdpre),
	RunE: func(cmd *cobra.Command, args []string) error {
		return createInstance(args)
	},
}

var instancePatchCmd = &cobra.Command{
	Use:   "patch <id | domain> < instance-patch.json",
	Short: "Patch Stack Instance",
	Long: fmt.Sprintf(`Patch Stack Instance by sending JSON via stdin, for example:
%[1]s
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
	}
%[1]s`, mdpre),
	RunE: func(cmd *cobra.Command, args []string) error {
		return patchInstance(args)
	},
}

var instanceDeployCmd = &cobra.Command{
	Use:   "deploy <id | domain>",
	Short: "Deploy Stack Instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deployInstance(args)
	},
}

var instanceUndeployCmd = &cobra.Command{
	Use:   "undeploy <id | domain>",
	Short: "Undeploy Stack Instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		return undeployInstance(args)
	},
}

var instanceBackupCmd = &cobra.Command{
	Use:   "backup <id | domain> <backup name>",
	Short: "Backup Stack Instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		return backupInstance(args)
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
	Short: "Delete Stack Instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteInstance(args)
	},
}

var instanceKubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig <id | domain>",
	Short: "Download Stack Instance Kubeconfig",
	RunE: func(cmd *cobra.Command, args []string) error {
		return kubeconfigInstance(args)
	},
}

var instanceLogsCmd = &cobra.Command{
	Use:   "logs <id | domain> [operation-id]",
	Short: "Download Stack Instance lifecycle operation logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		return logsInstance(args)
	},
}

var instanceWorkerpoolCmd = &cobra.Command{
	Use:   "workerpool <get | create | scale | undeploy | deploy | delete>",
	Short: "Create and manage platform stack instance Kubernetes worker pools",
}

var instanceWorkerpoolGetCmd = &cobra.Command{
	Use:   "get [id | domain | pool@domain]",
	Short: "Show a list of worker pools or details about the worker pool",
	Long: `Show a list of all platform stack instance worker pools or details about
the particular worker pool (specify Id or search by full domain name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return workerpool(args)
	},
}

var instanceWorkerpoolCreateCmd = &cobra.Command{
	Use:   "create <id | domain> <name> <instance type> <count> [max]",
	Short: "Create worker pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		return createWorkerpool(args)
	},
}

var instanceWorkerpoolScaleCmd = &cobra.Command{
	Use:   "scale <id | name@domain> <count> [max]",
	Short: "Scale worker pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		return scaleWorkerpool(args)
	},
}

var instanceWorkerpoolDeployCmd = &cobra.Command{
	Use:   "deploy <id | name@domain>",
	Short: "Deploy worker pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deployWorkerpool(args)
	},
}

var instanceWorkerpoolUndeployCmd = &cobra.Command{
	Use:   "undeploy <id | name@domain>",
	Short: "Undeploy worker pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		return undeployWorkerpool(args)
	},
}

var instanceWorkerpoolDeleteCmd = &cobra.Command{
	Use:   "delete <id | name@domain>",
	Short: "Delete worker pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteWorkerpool(args)
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
	api.StackInstances(selector, environmentSelector, showSecrets, showLogs, showBackups, jsonFormat)

	return nil
}

func createInstance(args []string) error {
	if len(args) > 0 {
		return errors.New("Create Instance command has no arguments")
	}

	if createRaw {
		api.RawCreateStackInstance(os.Stdin)
	} else {
		createBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil || len(createBytes) < 3 {
			return fmt.Errorf("Unable to read data (read %d bytes): %v", len(createBytes), err)
		}
		var req api.StackInstanceRequest
		err = json.Unmarshal(createBytes, &req)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal data: %v", err)
		}
		api.CreateStackInstance(req)
	}

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
	components := util.SplitPaths(componentName)
	api.DeployStackInstance(args[0], components, waitAndTailDeployLogs, dryRun)

	return nil
}

func undeployInstance(args []string) error {
	if len(args) != 1 {
		return errors.New("Undeploy Instance command has one mandatory argument - id or full domain name of the Instance")
	}

	components := util.SplitPaths(componentName)
	api.UndeployStackInstance(args[0], components, waitAndTailDeployLogs)

	return nil
}

func backupInstance(args []string) error {
	if len(args) != 2 {
		return errors.New("Backup Instance command has two mandatory arguments - id or full domain name of the Instance, and Backup name")
	}
	components := util.SplitPaths(componentName)
	api.BackupStackInstance(args[0], args[1], components, waitAndTailDeployLogs)

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

func logsInstance(args []string) error {
	if len(args) != 1 && len(args) != 2 {
		return errors.New("Logs Instance command has one mandatory argument - id or full domain name of the Instance")
	}
	selector := args[0]
	operationId := ""
	if len(args) > 1 {
		operationId = args[1]
	}

	api.LogsStackInstance(selector, operationId, logsOutput)

	return nil
}

func workerpool(args []string) error {
	if len(args) > 1 {
		return errors.New("Workerpool command has one optional argument - id or domain of the worker pool")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	api.Workerpools(selector, environmentSelector, showSecrets, showLogs, jsonFormat)

	return nil
}

func createWorkerpool(args []string) error {
	// create <id | domain> <name> <instance type> <count> [max count]
	if len(args) < 4 || len(args) > 5 {
		return errors.New("Create worker pool command has several arguments - id or domain of the platform, name of the worker pool, instance type, and node count")
	}

	selector := args[0]
	name := args[1]
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
	api.CreateWorkerpool(selector, name, instanceType, int(count), int(maxCount),
		workerpoolSpotPrice, workerpoolPreemptibleVMs, workerpoolAutoscale, workerpoolVolumeSize,
		waitAndTailDeployLogs, dryRun)

	return nil
}

func scaleWorkerpool(args []string) error {
	if len(args) < 2 || len(args) > 3 {
		return errors.New("Scale worker pool command has two or three arguments - id or name@domain of the worker pool, node count, and (optionally) node max count")
	}

	selector := args[0]
	count, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return fmt.Errorf("Unable to parse count: %v", err)
	}
	maxCount := int64(0)
	if len(args) > 2 {
		maxCount, err = strconv.ParseInt(args[2], 10, 32)
		if err != nil {
			return fmt.Errorf("Unable to parse max count: %v", err)
		}
	}
	if dryRun {
		waitAndTailDeployLogs = false
	}
	api.ScaleWorkerpool(selector, workerpoolInstanceType, int(count), int(maxCount), waitAndTailDeployLogs, dryRun)

	return nil
}

func deployWorkerpool(args []string) error {
	if len(args) != 1 {
		return errors.New("Deploy worker command has one mandatory argument - id or name@domain of the worker pool")
	}

	if dryRun {
		waitAndTailDeployLogs = false
	}
	api.DeployWorkerpool(args[0], waitAndTailDeployLogs, dryRun)

	return nil
}

func undeployWorkerpool(args []string) error {
	if len(args) != 1 {
		return errors.New("Undeploy worker command has one mandatory argument - id or name@domain of the worker pool")
	}

	api.UndeployWorkerpool(args[0], workerpoolDelete, waitAndTailDeployLogs)

	return nil
}

func deleteWorkerpool(args []string) error {
	if len(args) != 1 {
		return errors.New("Delete worker command has one mandatory argument - id or name@domain of the worker pool")
	}

	api.DeleteWorkerpool(args[0])

	return nil
}

func init() {
	instanceGetCmd.Flags().StringVarP(&environmentSelector, "environment", "e", "",
		"Environment name or Id")
	instanceGetCmd.Flags().BoolVarP(&showSecrets, "secrets", "", false,
		"Show secrets")
	instanceGetCmd.Flags().BoolVarP(&showBackups, "backups", "b", false,
		"Show backups")
	instanceGetCmd.Flags().BoolVarP(&showLogs, "logs", "l", false,
		"Show logs")
	instanceGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	instanceCreateCmd.Flags().BoolVarP(&createRaw, "raw", "r", false,
		"Send data as is, do not trim non-POST-able API object fields")
	instancePatchCmd.Flags().BoolVarP(&patchReplace, "replace", "", true,
		"Replace patched fields, do not merge")
	instancePatchCmd.Flags().BoolVarP(&patchRaw, "raw", "r", false,
		"Send patch data as is, do not trim non-PATCH-able API object fields")
	instanceDeployCmd.Flags().StringVarP(&componentName, "components", "c", "",
		"A list of components to deploy")
	instanceDeployCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	instanceDeployCmd.Flags().BoolVarP(&dryRun, "dry", "y", false,
		"Save parameters and envrc to Template's Git but do not start the deployment")
	instanceUndeployCmd.Flags().StringVarP(&componentName, "components", "c", "",
		"A list of components to undeploy")
	instanceUndeployCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	instanceBackupCmd.Flags().StringVarP(&componentName, "components", "c", "",
		"A list of components to backup")
	instanceBackupCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for backup and tail logs")
	instanceSyncCmd.Flags().StringVarP(&stateManifestExplicit, "state", "s", "",
		"Path to state files")
	instanceKubeconfigCmd.Flags().StringVarP(&kubeconfigOutput, "output", "o", "",
		"Set output filename, \"-\" for stdout (default to kubeconfig.<domain>.yaml)")
	instanceLogsCmd.Flags().StringVarP(&logsOutput, "output", "o", "",
		"Set oustput filename, \"-\" for stdout (default to logs.<domain>.<operation-id>.txt)")
	instanceCmd.AddCommand(instanceGetCmd)
	instanceCmd.AddCommand(instanceCreateCmd)
	instanceCmd.AddCommand(instancePatchCmd)
	instanceCmd.AddCommand(instanceDeployCmd)
	instanceCmd.AddCommand(instanceUndeployCmd)
	instanceCmd.AddCommand(instanceBackupCmd)
	instanceCmd.AddCommand(instanceSyncCmd)
	instanceCmd.AddCommand(instanceDeleteCmd)
	instanceCmd.AddCommand(instanceKubeconfigCmd)
	instanceCmd.AddCommand(instanceLogsCmd)

	instanceWorkerpoolGetCmd.Flags().StringVarP(&environmentSelector, "environment", "e", "",
		"Environment name or Id")
	instanceWorkerpoolGetCmd.Flags().BoolVarP(&showSecrets, "secrets", "", false,
		"Show secrets")
	instanceWorkerpoolGetCmd.Flags().BoolVarP(&showLogs, "logs", "l", false,
		"Show logs")
	instanceWorkerpoolGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	instanceWorkerpoolCreateCmd.Flags().Float32VarP(&workerpoolSpotPrice, "spot-price", "s", 0,
		"AWS use spot instances at specified spot price")
	instanceWorkerpoolCreateCmd.Flags().BoolVarP(&workerpoolPreemptibleVMs, "preemptible-vms", "p", false,
		"GCP use preemptible VMs")
	instanceWorkerpoolCreateCmd.Flags().BoolVarP(&workerpoolAutoscale, "autoscale", "a", false,
		"Autoscale worker pool with cluster-autoscaler (Agile Stacks and EKS Kubernetes only)")
	instanceWorkerpoolCreateCmd.Flags().IntVarP(&workerpoolVolumeSize, "volume-size", "z", 0,
		"Node root volume size (default 50GB)")
	instanceWorkerpoolCreateCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	instanceWorkerpoolCreateCmd.Flags().BoolVarP(&dryRun, "dry", "y", false,
		"Save parameters and envrc to Template's Git but do not start the deployment")
	instanceWorkerpoolScaleCmd.Flags().StringVarP(&workerpoolInstanceType, "instance-type", "m", "",
		"Instance type")
	instanceWorkerpoolScaleCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	instanceWorkerpoolScaleCmd.Flags().BoolVarP(&dryRun, "dry", "y", false,
		"Save parameters and envrc to Template's Git but do not start the deployment")
	instanceWorkerpoolDeployCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	instanceWorkerpoolDeployCmd.Flags().BoolVarP(&dryRun, "dry", "y", false,
		"Save parameters and envrc to Template's Git but do not start the deployment")
	instanceWorkerpoolUndeployCmd.Flags().BoolVarP(&workerpoolDelete, "delete", "", false,
		"Delete worker pool (stack instance) after it is undeployed")
	instanceWorkerpoolUndeployCmd.Flags().BoolVarP(&waitAndTailDeployLogs, "wait", "w", false,
		"Wait for deployment and tail logs")
	instanceWorkerpoolCmd.AddCommand(instanceWorkerpoolGetCmd)
	instanceWorkerpoolCmd.AddCommand(instanceWorkerpoolCreateCmd)
	instanceWorkerpoolCmd.AddCommand(instanceWorkerpoolScaleCmd)
	instanceWorkerpoolCmd.AddCommand(instanceWorkerpoolDeployCmd)
	instanceWorkerpoolCmd.AddCommand(instanceWorkerpoolUndeployCmd)
	instanceWorkerpoolCmd.AddCommand(instanceWorkerpoolDeleteCmd)

	instanceCmd.AddCommand(instanceWorkerpoolCmd)
	apiCmd.AddCommand(instanceCmd)
}
