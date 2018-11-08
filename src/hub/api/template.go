package api

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"strconv"
	"strings"
)

const templatesResource = "hub/api/v1/templates"

var templatesCache = make(map[string]*StackTemplate)

func Templates(selector string, showGitRemote, wildcardSecret, showGitStatus bool) {
	templates, err := templatesBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Template(s): %v", err)
	}
	if len(templates) == 0 {
		fmt.Print("No Templates\n")
	} else {
		deploymentKey := ""
		if showGitRemote {
			if wildcardSecret {
				key, err := userDeploymentKey("")
				if err != nil {
					log.Fatalf("Unable to retrieve deployment key: ", err)
				}
				deploymentKey = key
			}
		} else {
			fmt.Print("Templates:\n")
		}
		errors := make([]error, 0)
		for _, template := range templates {
			if showGitRemote {
				if !wildcardSecret {
					key, err := userDeploymentKey("git:" + template.Id)
					if err != nil {
						errors = append(errors, fmt.Errorf("Unable to retrieve deployment key: ", err))
						deploymentKey = "(error)"
					} else {
						deploymentKey = key
					}
				}
				title := ""
				if len(templates) > 1 {
					title = fmt.Sprintf("%s [%s]: ", template.Name, template.Id)
				}
				fmt.Printf("%s%s\n", title, formatGitRemoteWithKey(template.GitRemote.Public, deploymentKey))
			} else {
				errors = formatTemplateEntity(&template, showGitStatus, errors)
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

func formatGitRemoteWithKey(url, key string) string {
	i := strings.Index(url, "://")
	if i > 0 && i < len(url)-3 {
		return fmt.Sprintf("%s://%s@%s", url[0:i], key, url[i+3:])
	}
	return url
}

func formatTemplateEntity(template *StackTemplate, showGitStatus bool, errors []error) []error {
	title := formatTemplate(template)
	if template.Description != "" {
		title = fmt.Sprintf("%s - %s", title, template.Description)
	}
	fmt.Printf("\n\t%s\n", title)
	if len(template.Tags) > 0 {
		fmt.Printf("\t\tTags: %s\n", strings.Join(template.Tags, ", "))
	}
	fmt.Printf("\t\tStatus: %s\n", template.Status)
	if template.Stack.Name != "" {
		fmt.Printf("\t\tStack: %s\n", formatStackRef(&template.Stack))
	}
	fmt.Printf("\t\tVerbs: %s\n", strings.Join(template.Verbs, ", "))
	if len(template.ComponentsEnabled) > 0 {
		fmt.Printf("\t\tComponents enabled: %s\n", strings.Join(template.ComponentsEnabled, ", "))
	}
	if len(template.TeamsPermissions) > 0 {
		formatted := formatTeams(template.TeamsPermissions)
		fmt.Printf("\t\tTeams: %s\n", formatted)
	}
	if template.GitRemote.Public != "" {
		fmt.Printf("\t\tGit: %s\n", template.GitRemote.Public)
		if showGitStatus {
			g, err := templateGitStatus(template.Id)
			if err != nil {
				errors = append(errors, err)
			} else {
				commit := g.Commit
				if len(commit) > 7 {
					commit = commit[:7]
				}
				fmt.Printf("\t\t\t%s %s %s %s %s\n", commit, g.Ref, g.Author, g.Date, g.Subject)
			}
		}
	}
	if len(template.Parameters) > 0 {
		fmt.Print("\t\tParameters:\n")
	}
	resource := fmt.Sprintf("%s/%s", templatesResource, template.Id)
	for _, param := range sortParameters(template.Parameters) {
		formatted, err := formatParameter(resource, param)
		fmt.Printf("\t\t%s\n", formatted)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func cachedTemplateBy(selector string) (*StackTemplate, error) {
	template, cached := templatesCache[selector]
	if !cached {
		var err error
		template, err = templateBy(selector)
		if err != nil {
			return nil, err
		}
		templatesCache[selector] = template
	}
	return template, nil
}

func templateBy(selector string) (*StackTemplate, error) {
	_, err := strconv.ParseUint(selector, 10, 32)
	if err != nil {
		return templateByName(selector)
	}
	return templateById(selector)
}

func templatesBy(selector string) ([]StackTemplate, error) {
	_, err := strconv.ParseUint(selector, 10, 32)
	if err != nil {
		return templatesByName(selector)
	}
	template, err := templateById(selector)
	if err != nil {
		return nil, err
	}
	if template != nil {
		return []StackTemplate{*template}, nil
	}
	return nil, nil
}

func templateById(id string) (*StackTemplate, error) {
	path := fmt.Sprintf("%s/%s", templatesResource, url.PathEscape(id))
	var jsResp StackTemplate
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying Hub Service Templates: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Templates, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func templateByName(name string) (*StackTemplate, error) {
	templates, err := templatesByName(name)
	if err != nil {
		return nil, fmt.Errorf("Unable to query for Template `%s`: %v", name, err)
	}
	if len(templates) == 0 {
		return nil, fmt.Errorf("No Template `%s` found", name)
	}
	if len(templates) > 1 {
		return nil, fmt.Errorf("More than one Template returned by name `%s`", name)
	}
	instance := templates[0]
	return &instance, nil
}

func templatesByName(name string) ([]StackTemplate, error) {
	path := templatesResource
	if name != "" {
		path += "?name=" + url.QueryEscape(name)
	}
	var jsResp []StackTemplate
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying Hub Service Templates: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Templates, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func templateGitStatus(id string) (*TemplateStatus, error) {
	path := fmt.Sprintf("%s/%s/git/status", templatesResource, url.PathEscape(id))
	var jsResp TemplateStatus
	code, err := get(hubApi, path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying Hub Service Template Git status: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying Hub Service Template Git status, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func formatTemplate(template *StackTemplate) string {
	return fmt.Sprintf("%s [%s]", template.Name, template.Id)
}

func formatTemplateRef(ref *StackTemplateRef) string {
	return fmt.Sprintf("%s [%s]", ref.Name, ref.Id)
}

func formatStackRef(ref *StackRef) string {
	return fmt.Sprintf("%s [%s]", ref.Name, ref.Id)
}

func CreateTemplate(body io.Reader) {
	template, err := createTemplate(body)
	if err != nil {
		log.Fatalf("Unable to create Hub Service Template: %v", err)
	}
	errors := formatTemplateEntity(template, false, make([]error, 0))
	if len(errors) > 0 {
		fmt.Print("Errors encountered formatting response:\n")
		for _, err := range errors {
			fmt.Printf("\t%v\n", err)
		}
	}
}

func createTemplate(body io.Reader) (*StackTemplate, error) {
	var jsResp StackTemplate
	code, err := post2(hubApi, templatesResource, body, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 {
		return nil, fmt.Errorf("Got %d HTTP creating Hub Service Template, expected [200, 201] HTTP", code)
	}
	return &jsResp, nil
}

func InitTemplate(selector string) {
	err := initTemplate(selector)
	if err != nil {
		log.Fatalf("Unable to initialize Hub Service Template: %v", err)
	}
}

func initTemplate(selector string) error {
	template, err := templateBy(selector)
	if err != nil {
		return err
	}
	if template == nil {
		return error404
	}
	path := fmt.Sprintf("%s/%s/git/create", templatesResource, url.PathEscape(template.Id))
	code, err := post2(hubApi, path, nil, nil)
	if err != nil {
		return err
	}
	if code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP initializing	Hub Service Template, expected [202, 204] HTTP", code)
	}
	return nil
}

func DeleteTemplate(selector string) {
	err := deleteTemplate(selector)
	if err != nil {
		log.Fatalf("Unable to delete Hub Service Template: %v", err)
	}
}

func deleteTemplate(selector string) error {
	template, err := templateBy(selector)
	if err != nil {
		return err
	}
	if template == nil {
		return error404
	}
	path := fmt.Sprintf("%s/%s", templatesResource, url.PathEscape(template.Id))
	code, err := delete(hubApi, path)
	if err != nil {
		return err
	}
	if code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP deleting Hub Service Template, expected [202, 204] HTTP", code)
	}
	return nil
}
