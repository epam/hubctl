package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"hub/api"
)

var componentCmd = &cobra.Command{
	Use:   "component <get | create | delete> ...",
	Short: "Create and manage (custom) Component Registration",
}

var componentGetCmd = &cobra.Command{
	Use:   "get [id | qname | name]",
	Short: "Show a list of Components or details about the Component",
	Long: `Show a list of all user accessible Components or details about
the particular Component (specify Id or search by name)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return component(args)
	},
}

var componentCreateCmd = &cobra.Command{
	Use:   "create < component.json",
	Short: "Create Component Registration",
	Long: `Create Component Registration by sending JSON via stdin, for example:
	{
		"name": "kube-dashboard", // generated qname = "org:kube-dashboard/kube-dashboard-456",
		"title": "Dashboard",
		"brief": "Kubernetes Dashboard",
		"description": "...",
		"tags": [],
		"ui": {},
		"category": "Kubernetes Tools",
		"license": "Apache 2.0",
		"icon": "data:image/png;base64",
		"template": "456", // https://git.agilestacks.io/repo/org/kube-dashboard-456
		"git": {
			"subDir": "optional"
		},
		"version": "1.10.1",
		"maturity": "ga",
		"requires": [
			"kubernetes"
		],
		"provides": [
			"kubernetes-dashboard"
		],
		"parameters": [ // optional defaults for UI custom component settings form
			{"name": "...", "value": "...", "brief": "UI label"}
		],
		"teamsPermissions": []
	}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createComponent(args)
	},
}

var componentPatchCmd = &cobra.Command{
	Use:   "patch <id | qname | name> < component-patch.json",
	Short: "Patch Component Registration",
	Long: `Patch Component Registration by sending JSON via stdin, for example:
	{
		"title": "Dashboard",
		"brief": "Kubernetes Dashboard",
		"description": "...",
		"tags": [],
		"ui": {},
		"category": "Kubernetes Tools",
		"license": "Apache 2.0",
		"icon": "data:image/png;base64",
		"version": "1.10.1",
		"maturity": "ga",
		"requires": [
			"kubernetes"
		],
		"provides": [
			"kubernetes-dashboard"
		],
		"teamsPermissions": []
	}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return patchComponent(args)
	},
}

var componentDeleteCmd = &cobra.Command{
	Use:   "delete <id | qname | name>",
	Short: "Delete Component Registration by Id or name",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteComponent(args)
	},
}

func component(args []string) error {
	if len(args) > 1 {
		return errors.New("Component command has one optional argument - id or name of the component")
	}

	selector := ""
	if len(args) > 0 {
		selector = args[0]
	}
	api.Components(selector, jsonFormat)

	return nil
}

func createComponent(args []string) error {
	if len(args) > 0 {
		return errors.New("Create Component Registration command has no arguments")
	}

	api.CreateComponent(os.Stdin)

	return nil
}

func patchComponent(args []string) error {
	if len(args) != 1 {
		return errors.New("Patch Component Registration command has one mandatory argument - id or name of the component")
	}

	selector := args[0]
	if patchRaw {
		api.RawPatchComponent(selector, os.Stdin)
	} else {
		patchBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil || len(patchBytes) < 3 {
			return fmt.Errorf("Unable to read patch data (read %d bytes): %v", len(patchBytes), err)
		}
		var patch api.ComponentPatch
		err = json.Unmarshal(patchBytes, &patch)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal patch data: %v", err)
		}
		api.PatchComponent(selector, patch)
	}

	return nil
}

func deleteComponent(args []string) error {
	if len(args) != 1 {
		return errors.New("Delete Component Registration command has one mandatory argument - id or name of the component")
	}

	api.DeleteComponent(args[0])

	return nil
}

func init() {
	componentGetCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false,
		"JSON output")
	componentPatchCmd.Flags().BoolVarP(&patchRaw, "raw", "r", false,
		"Send patch data as is, do not trim non-PATCH-able API object fields")
	componentCmd.AddCommand(componentGetCmd)
	componentCmd.AddCommand(componentCreateCmd)
	componentCmd.AddCommand(componentPatchCmd)
	componentCmd.AddCommand(componentDeleteCmd)
	apiCmd.AddCommand(componentCmd)
}
