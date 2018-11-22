package api

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
)

const applicationsResource = "hub/api/v1/applications"

var applicationsCache = make(map[string]*Application)

func Applications(selector string, showSecrets bool) {
	applications, err := applicationsBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Application(s): %v", err)
	}
	if len(applications) == 0 {
		fmt.Print("No Applications\n")
	} else {
		fmt.Print("Applications:\n")
		errors := make([]error, 0)
		for _, application := range applications {
			title := fmt.Sprintf("%s [%s]", application.Name, application.Id)
			if application.Description != "" {
				title = fmt.Sprintf("%s - %s", title, application.Description)
			}
			fmt.Printf("\n\t%s\n", title)
			if len(application.Tags) > 0 {
				fmt.Printf("\t\tTags: %s\n", strings.Join(application.Tags, ", "))
			}
			if len(application.Environments) > 0 {
				fmt.Print("\t\tEnvironments:\n")
				for _, environment := range application.Environments {
					fmt.Printf("\t\t\t%s @ %s\n", environment.Name, environment.Domain)
				}
			}
			if len(application.TeamsPermissions) > 0 {
				formatted := formatTeams(application.TeamsPermissions)
				fmt.Printf("\t\tTeams: %s\n", formatted)
			}
			if len(application.Parameters) > 0 {
				fmt.Print("\t\tParameters:\n")
			}
			resource := fmt.Sprintf("%s/%s", applicationsResource, application.Id)
			for _, param := range sortParameters(application.Parameters) {
				formatted, err := formatParameter(resource, param, showSecrets)
				fmt.Printf("\t\t%s\n", formatted)
				if err != nil {
					errors = append(errors, err)
				}
			}
		}
		if len(errors) > 0 {
			fmt.Print("Errors encountered:\n")
			for _, err := range errors {
				fmt.Printf("\t%v\n", err)
			}
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
	_, err := strconv.ParseUint(selector, 10, 32)
	if err != nil {
		return applicationByDomain(selector)
	}
	return applicationById(selector)
}

func applicationsBy(selector string) ([]Application, error) {
	_, err := strconv.ParseUint(selector, 10, 32)
	if err != nil {
		return applicationsByDomain(selector)
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
		return nil, fmt.Errorf("Error querying Hub Service Applications: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Applications, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func applicationByDomain(domain string) (*Application, error) {
	applications, err := applicationsByDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("Unable to query for Application `%s`: %v", domain, err)
	}
	if len(applications) == 0 {
		return nil, fmt.Errorf("No Application `%s` found", domain)
	}
	if len(applications) > 1 {
		return nil, fmt.Errorf("More than one Application returned by domain `%s`", domain)
	}
	instance := applications[0]
	return &instance, nil
}

func applicationsByDomain(domain string) ([]Application, error) {
	path := applicationsResource
	if domain != "" {
		path += "?domain=" + url.QueryEscape(domain)
	}
	var jsResp []Application
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying Hub Service Applications: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Applications, expected 200 HTTP", code)
	}
	return jsResp, nil
}
