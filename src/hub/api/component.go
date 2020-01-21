package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"

	"hub/config"
	"hub/util"
)

const componentsResource = "hub/api/v1/components"

func Components(selector string, onlyCustomComponents, jsonFormat bool) {
	components, err := componentsBy(selector, onlyCustomComponents)
	if err != nil {
		log.Fatalf("Unable to query for Component(s): %v", err)
	}
	if len(components) == 0 {
		if jsonFormat {
			log.Print("No Components")
		} else {
			fmt.Print("No Components\n")
		}
	} else {
		if jsonFormat {
			var toMarshal interface{}
			if len(components) == 1 {
				toMarshal = &components[0]
			} else {
				toMarshal = components
			}
			out, err := json.MarshalIndent(toMarshal, "", "  ")
			if err != nil {
				log.Fatalf("Error marshalling JSON response for output: %v", err)
			}
			os.Stdout.Write(out)
			os.Stdout.Write([]byte("\n"))
		} else {
			errors := make([]error, 0)
			for _, component := range components {
				errors = formatComponentEntity(&component, errors)
			}
			if len(errors) > 0 {
				fmt.Print("Errors encountered:\n")
				for _, err := range errors {
					fmt.Printf("\t%v\n", err)
				}
			}
		}
	}
}

func formatComponentEntity(component *Component, errors []error) []error {
	fmt.Printf("\n\t%s\n", formatComponentTitle(component))
	if component.Description != "" {
		fmt.Printf("\t\tDescription: %s\n", strings.Trim(component.Description, "\r\n "))
	}
	if len(component.Tags) > 0 {
		fmt.Printf("\t\tTags: %s\n", strings.Join(component.Tags, ", "))
	}
	if component.Category != "" {
		fmt.Printf("\t\tCategory: %s\n", component.Category)
	}
	if component.License != "" {
		fmt.Printf("\t\tLicense: %s\n", component.License)
	}
	if component.Icon != "" {
		fmt.Printf("\t\tIcon: %s\n", util.Wrap(component.Icon))
	}
	if component.Template != nil {
		fmt.Printf("\t\tTemplate: %s\n", formatTemplateRef(component.Template))
	}
	if component.Git != nil {
		fmt.Printf("\t\tGit: %s\n", formatComponentGitRef(component.Git))
	}
	if component.Version != "" {
		fmt.Printf("\t\tVersion: %s\n", component.Version)
	}
	if component.Maturity != "" {
		fmt.Printf("\t\tMaturity: %s\n", component.Maturity)
	}
	if len(component.Requires) > 0 {
		fmt.Printf("\t\tRequires: %s\n", strings.Join(component.Requires, ", "))
	}
	if len(component.Provides) > 0 {
		fmt.Printf("\t\tProvides: %s\n", strings.Join(component.Provides, ", "))
	}
	if len(component.TeamsPermissions) > 0 {
		formatted := formatTeams(component.TeamsPermissions)
		fmt.Printf("\t\tTeams: %s\n", formatted)
	}
	return errors
}

func formatComponentTitle(component *Component) string {
	brief := ""
	if component.Brief != "" {
		brief = " - " + component.Brief
	}
	id := ""
	if component.Id != "" {
		brief = fmt.Sprintf(" [%s]", component.Id)
	}
	return fmt.Sprintf("%s%s%s", component.QName, id, brief)
}

func formatComponentGitRef(ref *ComponentGitRef) string {
	subDir := ""
	if ref.SubDir != "" {
		subDir = "/" + ref.SubDir
	}
	return fmt.Sprintf("%s%s", ref.Remote, subDir)
}

func formatComponent(component *Component) {
	errors := formatComponentEntity(component, make([]error, 0))
	if len(errors) > 0 {
		fmt.Print("Errors encountered formatting response:\n")
		for _, err := range errors {
			fmt.Printf("\t%v\n", err)
		}
	}
}

func componentBy(selector string) (*Component, error) {
	if !util.IsUint(selector) {
		return componentByName(selector)
	}
	return componentById(selector)
}

func componentsBy(selector string, onlyCustomComponents bool) ([]Component, error) {
	if !util.IsUint(selector) {
		return componentsByName(selector, onlyCustomComponents)
	}
	component, err := componentById(selector)
	if err != nil {
		return nil, err
	}
	if component != nil {
		return []Component{*component}, nil
	}
	return nil, nil
}

func componentById(id string) (*Component, error) {
	path := fmt.Sprintf("%s/%s", componentsResource, url.PathEscape(id))
	var jsResp Component
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Components: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Components, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func componentByName(name string) (*Component, error) {
	components, err := componentsByName(name, false)
	if err != nil {
		return nil, fmt.Errorf("Unable to query for Component `%s`: %v", name, err)
	}
	if len(components) == 0 {
		return nil, fmt.Errorf("No Component `%s` found", name)
	}
	if len(components) > 1 {
		return nil, fmt.Errorf("More than one Component returned by name `%s`", name)
	}
	component := components[0]
	return &component, nil
}

func componentsByName(name string, onlyCustomComponents bool) ([]Component, error) {
	path := componentsResource
	var filters []string
	if name != "" {
		filters = append(filters, "qname="+url.QueryEscape(name))
	}
	if onlyCustomComponents {
		filters = append(filters, "kind=custom")
	}
	if len(filters) > 0 {
		path += "?" + strings.Join(filters, "&")
	}
	var jsResp []Component
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Components: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Components, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func CreateComponent(req ComponentRequest) {
	component, err := createComponent(req)
	if err != nil {
		log.Fatalf("Unable to create SuperHub Component: %v", err)
	}
	formatComponent(component)
}

func createComponent(req ComponentRequest) (*Component, error) {
	if req.Template != "" && !util.IsUint(req.Template) {
		template, err := templateByName(req.Template)
		if err != nil {
			return nil, err
		}
		req.Template = template.Id
	}
	var jsResp Component
	code, err := post(hubApi(), componentsResource, &req, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 {
		return nil, fmt.Errorf("Got %d HTTP creating SuperHub Component, expected [200, 201] HTTP", code)
	}
	return &jsResp, nil
}

func RawCreateComponent(body io.Reader) {
	component, err := rawCreateComponent(body)
	if err != nil {
		log.Fatalf("Unable to create SuperHub Component: %v", err)
	}
	formatComponent(component)
}

func rawCreateComponent(body io.Reader) (*Component, error) {
	var jsResp Component
	code, err := post2(hubApi(), componentsResource, body, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 {
		return nil, fmt.Errorf("Got %d HTTP creating SuperHub Component, expected [200, 201] HTTP", code)
	}
	return &jsResp, nil
}

func DeleteComponent(selector string) {
	err := deleteComponent(selector)
	if err != nil {
		log.Fatalf("Unable to delete SuperHub Component: %v", err)
	}
}

func deleteComponent(selector string) error {
	component, err := componentBy(selector)
	id := ""
	if err != nil {
		str := err.Error()
		if util.IsUint(selector) &&
			(strings.Contains(str, "json: cannot unmarshal") || strings.Contains(str, "cannot parse") || config.Force) {
			util.Warn("%v", err)
			id = selector
		} else {
			return err
		}
	} else if component == nil {
		return error404
	} else {
		id = component.Id
	}
	force := ""
	if config.Force {
		force = "?force=true"
	}
	path := fmt.Sprintf("%s/%s%s", componentsResource, url.PathEscape(id), force)
	code, err := delete(hubApi(), path)
	if err != nil {
		return err
	}
	if code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP deleting SuperHub Component, expected [202, 204] HTTP", code)
	}
	return nil
}

func PatchComponent(selector string, change ComponentPatch) {
	component, err := patchComponent(selector, change)
	if err != nil {
		log.Fatalf("Unable to patch SuperHub Component: %v", err)
	}
	formatComponent(component)
}

func patchComponent(selector string, change ComponentPatch) (*Component, error) {
	component, err := componentBy(selector)
	if err != nil {
		return nil, err
	}
	if component == nil {
		return nil, error404
	}
	path := fmt.Sprintf("%s/%s", componentsResource, url.PathEscape(component.Id))
	var jsResp Component
	code, err := patch(hubApi(), path, &change, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP patching SuperHub Component, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func RawPatchComponent(selector string, body io.Reader) {
	component, err := rawPatchComponent(selector, body)
	if err != nil {
		log.Fatalf("Unable to patch SuperHub Component: %v", err)
	}
	formatComponent(component)
}

func rawPatchComponent(selector string, body io.Reader) (*Component, error) {
	component, err := componentBy(selector)
	if err != nil {
		return nil, err
	}
	if component == nil {
		return nil, error404
	}
	path := fmt.Sprintf("%s/%s", componentsResource, url.PathEscape(component.Id))
	var jsResp Component
	code, err := patch2(hubApi(), path, body, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP patching SuperHub Component, expected 200 HTTP", code)
	}
	return &jsResp, nil
}
