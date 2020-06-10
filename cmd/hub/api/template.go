package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/util"
)

const templatesResource = "hub/api/v1/templates"

var templatesCache = make(map[string]*StackTemplate)

func Templates(selector string, showSecrets, showGitRemote, wildcardSecret, showGitStatus, jsonFormat bool) {
	templates, err := templatesBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Template(s): %v", err)
	}
	if len(templates) == 0 {
		if jsonFormat {
			log.Print("No Templates")
		} else {
			fmt.Print("No Templates\n")
		}
	} else {
		if jsonFormat {
			var toMarshal interface{}
			if len(templates) == 1 {
				toMarshal = &templates[0]
			} else {
				toMarshal = templates
			}
			out, err := json.MarshalIndent(toMarshal, "", "  ")
			if err != nil {
				log.Fatalf("Error marshalling JSON response for output: %v", err)
			}
			os.Stdout.Write(out)
			os.Stdout.Write([]byte("\n"))
		} else {
			deploymentKey := ""
			if showGitRemote {
				if wildcardSecret {
					key, err := userDeploymentKey("")
					if err != nil {
						log.Fatalf("Unable to retrieve deployment key: %v", err)
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
							errors = append(errors, fmt.Errorf("Unable to retrieve deployment key: %v", err))
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
					errors = formatTemplateEntity(&template, showSecrets, showGitStatus, errors)
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
}

func formatGitRemoteWithKey(url, key string) string {
	i := strings.Index(url, "://")
	if i > 0 && i < len(url)-3 {
		return fmt.Sprintf("%s://%s@%s", url[0:i], key, url[i+3:])
	}
	return url
}

func formatTemplateEntity(template *StackTemplate, showSecrets, showGitStatus bool, errors []error) []error {
	title := formatTemplateTitle(template)
	if template.Description != "" {
		title = fmt.Sprintf("%s - %s", title, template.Description)
	}
	fmt.Printf("\n\t%s\n", title)
	if len(template.Tags) > 0 {
		fmt.Printf("\t\tTags: %s\n", strings.Join(template.Tags, ", "))
	}
	fmt.Printf("\t\tStatus: %s\n", template.Status)
	if template.Stack != nil && template.Stack.Name != "" {
		fmt.Printf("\t\tStack: %s\n", formatStackRef(template.Stack))
	}
	if template.Component != nil && template.Component.Name != "" {
		fmt.Printf("\t\tComponent: %s\n", formatComponentRef(template.Component))
	}
	if len(template.Verbs) > 0 {
		fmt.Printf("\t\tVerbs: %s\n", strings.Join(template.Verbs, ", "))
	}
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
		formatted, err := formatParameter(resource, param, showSecrets)
		fmt.Printf("\t\t%s\n", formatted)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func formatTemplateTitle(template *StackTemplate) string {
	return fmt.Sprintf("%s [%s]", template.Name, template.Id)
}

func formatTemplateRef(ref *StackTemplateRef) string {
	return fmt.Sprintf("%s [%s]", ref.Name, ref.Id)
}

func formatStackRef(ref *StackRef) string {
	return fmt.Sprintf("%s [%s]", ref.Name, ref.Id)
}

func formatComponentRef(ref *ComponentRef) string {
	return fmt.Sprintf("%s [%s]", ref.Name, ref.Id)
}

func formatTemplate(template *StackTemplate) {
	errors := formatTemplateEntity(template, false, false, make([]error, 0))
	if len(errors) > 0 {
		fmt.Print("Errors encountered formatting response:\n")
		for _, err := range errors {
			fmt.Printf("\t%v\n", err)
		}
	}
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
	if !util.IsUint(selector) {
		return templateByName(selector)
	}
	return templateById(selector)
}

func templatesBy(selector string) ([]StackTemplate, error) {
	if !util.IsUint(selector) {
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
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Templates: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Templates, expected 200 HTTP", code)
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
	template := templates[0]
	return &template, nil
}

func templatesByName(name string) ([]StackTemplate, error) {
	path := templatesResource
	if name != "" {
		path += "?name=" + url.QueryEscape(name)
	}
	var jsResp []StackTemplate
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Templates: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Templates, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func templateGitStatus(id string) (*TemplateStatus, error) {
	path := fmt.Sprintf("%s/%s/git/status", templatesResource, url.PathEscape(id))
	var jsResp TemplateStatus
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Template Git status: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Template Git status, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func CreateTemplate(req StackTemplateRequest) {
	template, err := createTemplate(req)
	if err != nil {
		log.Fatalf("Unable to create SuperHub Template: %v", err)
	}
	formatTemplate(template)
}

func createTemplate(req StackTemplateRequest) (*StackTemplate, error) {
	var jsResp StackTemplate
	code, err := post(hubApi(), templatesResource, &req, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 {
		return nil, fmt.Errorf("Got %d HTTP creating SuperHub Template, expected [200, 201] HTTP", code)
	}
	return &jsResp, nil
}

func RawCreateTemplate(body io.Reader) {
	template, err := rawCreateTemplate(body)
	if err != nil {
		log.Fatalf("Unable to create SuperHub Template: %v", err)
	}
	formatTemplate(template)
}

func rawCreateTemplate(body io.Reader) (*StackTemplate, error) {
	var jsResp StackTemplate
	code, err := post2(hubApi(), templatesResource, body, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 {
		return nil, fmt.Errorf("Got %d HTTP creating SuperHub Template, expected [200, 201] HTTP", code)
	}
	return &jsResp, nil
}

func InitTemplate(selector string) {
	err := initTemplate(selector)
	if err != nil {
		log.Fatalf("Unable to initialize SuperHub Template: %v", err)
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
	code, err := post2(hubApiLongWait(), path, nil, nil)
	if err != nil {
		return err
	}
	if code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP initializing SuperHub Template, expected [202, 204] HTTP", code)
	}
	return nil
}

func DeleteTemplate(selector string) {
	err := deleteTemplate(selector)
	if err != nil {
		log.Fatalf("Unable to delete SuperHub Template: %v", err)
	}
}

func deleteTemplate(selector string) error {
	template, err := templateBy(selector)
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
	} else if template == nil {
		return error404
	} else {
		id = template.Id
	}
	force := ""
	if config.Force {
		force = "?force=true"
	}
	path := fmt.Sprintf("%s/%s%s", templatesResource, url.PathEscape(id), force)
	code, err := delete(hubApi(), path)
	if err != nil {
		return err
	}
	if code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP deleting SuperHub Template, expected [202, 204] HTTP", code)
	}
	return nil
}

func PatchTemplate(selector string, change StackTemplatePatch) {
	template, err := patchTemplate(selector, change)
	if err != nil {
		log.Fatalf("Unable to patch SuperHub Template: %v", err)
	}
	formatTemplate(template)
}

func patchTemplate(selector string, change StackTemplatePatch) (*StackTemplate, error) {
	template, err := templateBy(selector)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, error404
	}
	path := fmt.Sprintf("%s/%s", templatesResource, url.PathEscape(template.Id))
	var jsResp StackTemplate
	code, err := patch(hubApi(), path, &change, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP patching SuperHub Template, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func RawPatchTemplate(selector string, body io.Reader) {
	template, err := rawPatchTemplate(selector, body)
	if err != nil {
		log.Fatalf("Unable to patch SuperHub Template: %v", err)
	}
	formatTemplate(template)
}

func rawPatchTemplate(selector string, body io.Reader) (*StackTemplate, error) {
	template, err := templateBy(selector)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, error404
	}
	path := fmt.Sprintf("%s/%s", templatesResource, url.PathEscape(template.Id))
	var jsResp StackTemplate
	code, err := patch2(hubApi(), path, body, &jsResp)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP patching SuperHub Template, expected 200 HTTP", code)
	}
	return &jsResp, nil
}
