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

const applicationsResource = "hub/api/v1/applications"

var applicationsCache = make(map[string]*Application)

func Applications(selector string, showSecrets, showLogs, jsonFormat bool) {
	applications, err := applicationsBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Application(s): %v", err)
	}
	if len(applications) == 0 {
		if jsonFormat {
			log.Print("No Applications")
		} else {
			fmt.Print("No Applications\n")
		}
	} else {
		if jsonFormat {
			var toMarshal interface{}
			if len(applications) == 1 {
				toMarshal = &applications[0]
			} else {
				toMarshal = applications
			}
			out, err := json.MarshalIndent(toMarshal, "", "  ")
			if err != nil {
				log.Fatalf("Error marshalling JSON response for output: %v", err)
			}
			os.Stdout.Write(out)
			os.Stdout.Write([]byte("\n"))
		} else {
			fmt.Print("Applications:\n")
			errors := make([]error, 0)
			for _, application := range applications {
				errors = formatApplicationEntity(&application, showSecrets, showLogs, errors)
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

func formatApplicationEntity(application *Application, showSecrets, showLogs bool, errors []error) []error {
	title := fmt.Sprintf("%s [%s]", application.Name, application.Id)
	if application.Description != "" {
		title = fmt.Sprintf("%s - %s", title, application.Description)
	}
	fmt.Printf("\n\t%s\n", title)
	if len(application.Tags) > 0 {
		fmt.Printf("\t\tTags: %s\n", strings.Join(application.Tags, ", "))
	}
	fmt.Printf("\t\tKind: %s\n", application.Application)
	if application.Environment.Name != "" {
		fmt.Printf("\t\tEnvironment: %s\n", formatEnvironmentRef(&application.Environment))
	}
	if application.Platform.Name != "" {
		fmt.Printf("\t\tPlatform: %s\n", formatPlatformRef(&application.Platform))
	}
	if len(application.Requires) > 0 {
		fmt.Printf("\t\tRequires: %s\n", strings.Join(application.Requires, ", "))
	}
	fmt.Printf("\t\tStatus: %s\n", application.Status)
	if len(application.StateFiles) > 0 {
		fmt.Printf("\t\tState files:\n\t\t\t%s\n", strings.Join(application.StateFiles, "\n\t\t\t"))
	}
	resource := fmt.Sprintf("%s/%s", applicationsResource, application.Id)
	if len(application.Outputs) > 0 {
		formatted, errs := formatStackOutputs(resource, application.Outputs, showSecrets)
		fmt.Printf("\t\tOutputs:\n%s", formatted)
		if len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}
	if len(application.Parameters) > 0 {
		fmt.Print("\t\tParameters:\n")
	}
	for _, param := range sortParameters(application.Parameters) {
		formatted, err := formatParameter(resource, param, showSecrets)
		fmt.Printf("\t\t%s\n", formatted)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(application.InflightOperations) > 0 {
		fmt.Print("\t\tInflight Operations:\n")
		for _, op := range application.InflightOperations {
			fmt.Print(formatInflightOperation(op, showLogs))
		}
	}
	return errors
}

func formatApplication(application *Application) {
	errors := formatApplicationEntity(application, false, false, make([]error, 0))
	if len(errors) > 0 {
		fmt.Print("Errors encountered formatting response:\n")
		for _, err := range errors {
			fmt.Printf("\t%v\n", err)
		}
	}
}

func cachedApplicationBy(selector string) (*Application, error) {
	application, cached := applicationsCache[selector]
	if !cached {
		var err error
		application, err = applicationBy(selector)
		if err != nil {
			return nil, err
		}
		applicationsCache[selector] = application
	}
	return application, nil
}

func applicationBy(selector string) (*Application, error) {
	if !util.IsUint(selector) {
		return applicationByName(selector)
	}
	return applicationById(selector)
}

func applicationsBy(selector string) ([]Application, error) {
	if !util.IsUint(selector) {
		return applicationsByName(selector)
	}
	application, err := applicationById(selector)
	if err != nil {
		return nil, err
	}
	if application != nil {
		return []Application{*application}, nil
	}
	return nil, nil
}

func applicationById(id string) (*Application, error) {
	path := fmt.Sprintf("%s/%s", applicationsResource, url.PathEscape(id))
	var jsResp Application
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Applications: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Applications, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func applicationByName(name string) (*Application, error) {
	applications, err := applicationsByName(name)
	if err != nil {
		return nil, fmt.Errorf("Unable to query for Application `%s`: %v", name, err)
	}
	if len(applications) == 0 {
		return nil, fmt.Errorf("No Application `%s` found", name)
	}
	if len(applications) > 1 {
		return nil, fmt.Errorf("More than one Application returned by name `%s`", name)
	}
	instance := applications[0]
	return &instance, nil
}

func applicationsByName(name string) ([]Application, error) {
	path := applicationsResource
	if name != "" {
		path += "?name=" + url.QueryEscape(name)
	}
	var jsResp []Application
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Applications: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Applications, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func InstallApplication(body io.Reader, waitAndTailDeployLogs bool) {
	application, err := installApplication(body)
	if err != nil {
		log.Fatalf("Unable to install SuperHub Application: %v", err)
	}
	formatApplication(application)
	if waitAndTailDeployLogs {
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		os.Exit(Logs([]string{"application/" + application.Id}, true))
	}
}

func installApplication(body io.Reader) (*Application, error) {
	var jsResp Application
	code, err := post2(hubApi, applicationsResource, body, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 && code != 202 {
		return nil, fmt.Errorf("Got %d HTTP installing SuperHub Application, expected [200, 201, 202] HTTP", code)
	}
	return &jsResp, nil
}

func DeleteApplication(selector string, waitAndTailDeployLogs bool) {
	err := deleteApplication(selector)
	if err != nil {
		log.Fatalf("Unable to delete SuperHub Application: %v", err)
	}
	if waitAndTailDeployLogs {
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		os.Exit(Logs([]string{"application/" + selector}, true))
	}
}

func deleteApplication(selector string) error {
	application, err := applicationBy(selector)
	id := ""
	if err != nil {
		if strings.Contains(err.Error(), "json: cannot unmarshal") && util.IsUint(selector) {
			util.Warn("%v", err)
			id = selector
		} else {
			return err
		}
	} else if application == nil {
		return error404
	} else {
		id = application.Id
	}
	path := fmt.Sprintf("%s/%s", applicationsResource, url.PathEscape(id))
	code, err := delete(hubApi, path)
	if err != nil {
		return err
	}
	if code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP deleting SuperHub Application, expected [202, 204] HTTP", code)
	}
	return nil
}

func PatchApplication(selector string, change ApplicationPatch, waitAndTailDeployLogs bool) {
	application, err := patchApplication(selector, change)
	if err != nil {
		log.Fatalf("Unable to patch SuperHub Application: %v", err)
	}
	formatApplication(application)
	if waitAndTailDeployLogs {
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		os.Exit(Logs([]string{"application/" + application.Id}, true))
	}
}

func patchApplication(selector string, change ApplicationPatch) (*Application, error) {
	application, err := applicationBy(selector)
	if err != nil {
		return nil, err
	}
	if application == nil {
		return nil, error404
	}
	path := fmt.Sprintf("%s/%s", applicationsResource, url.PathEscape(application.Id))
	var jsResp Application
	code, err := patch(hubApi, path, &change, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 202 {
		return nil, fmt.Errorf("Got %d HTTP patching SuperHub Application, expected [200, 202] HTTP", code)
	}
	return &jsResp, nil
}

func RawPatchApplication(selector string, body io.Reader, waitAndTailDeployLogs bool) {
	application, err := rawPatchApplication(selector, body)
	if err != nil {
		log.Fatalf("Unable to patch SuperHub Application: %v", err)
	}
	formatApplication(application)
	if waitAndTailDeployLogs {
		if config.Verbose {
			log.Print("Tailing automation task logs... ^C to interrupt")
		}
		os.Exit(Logs([]string{"application/" + application.Id}, true))
	}
}

func rawPatchApplication(selector string, body io.Reader) (*Application, error) {
	instance, err := applicationBy(selector)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, error404
	}
	path := fmt.Sprintf("%s/%s", applicationsResource, url.PathEscape(instance.Id))
	var jsResp Application
	code, err := patch2(hubApi, path, body, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 202 {
		return nil, fmt.Errorf("Got %d HTTP patching SuperHub Application, expected [200, 204] HTTP", code)
	}
	return &jsResp, nil
}
