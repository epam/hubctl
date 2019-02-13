package api

import (
	"fmt"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"hub/config"
	"hub/util"
)

func GetParameterOrMaybeCreateSecret(environment, stackInstance, application,
	name, component string, create bool) (bool, string, []error) {

	found := false
	var value string
	var errors []error

	applicationEnvironmentIds := make([]string, 0)
	stackInstanceEnvironmentId := ""

	if application != "" {
		app, err := cachedApplicationBy(application)
		if err != nil {
			errors = append(errors, err)
		} else {
			for _, env := range app.Environments {
				applicationEnvironmentIds = append(applicationEnvironmentIds, env.Id)
			}
			resource := fmt.Sprintf("%s/%s", applicationsResource, url.PathEscape(app.Id))
			found, value, err = getParameter(resource, app.Parameters, name, component)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	if !found && stackInstance != "" {
		instance, err := cachedStackInstanceBy(stackInstance)
		if err != nil {
			errors = append(errors, err)
		} else {
			stackInstanceEnvironmentId = instance.Environment.Id
			resource := fmt.Sprintf("%s/%s", stackInstancesResource, url.PathEscape(instance.Id))
			found, value, err = getParameter(resource, instance.Parameters, name, component)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	if !found && environment != "" {
		env, err := cachedEnvironmentBy(environment)
		if err != nil {
			errors = append(errors, err)
		} else {
			if stackInstanceEnvironmentId != "" && stackInstanceEnvironmentId != env.Id {
				util.WarnOnce("Environment `%s` (%s) doesn't match Stack Instance Environment `%s`",
					env.Id, env.Name, stackInstanceEnvironmentId)
			}
			if len(applicationEnvironmentIds) > 0 && !util.Contains(applicationEnvironmentIds, env.Id) {
				util.WarnOnce("Environment `%s` (%s) doesn't match Application Environments %v",
					env.Id, env.Name, applicationEnvironmentIds)
			}

			resource := fmt.Sprintf("%s/%s", environmentsResource, url.PathEscape(env.Id))
			found, value, err = getParameter(resource, env.Parameters, name, component)
			if err != nil {
				errors = append(errors, err)
			}
			if !found && create && util.LooksLikeSecret(name) {
				value, err = createSecretParameter(environment, env.Id, name, component)
				if err != nil {
					errors = append(errors, err)
				}
			}
		}
	}
	return found, value, errors
}

func getParameter(resource string, parameters []Parameter, name, component string) (bool, string, error) {
	var param *Parameter
	if component != "" {
		for i, p := range parameters {
			if name == p.Name && component == p.Component {
				param = &parameters[i]
			}
		}
	}
	if param == nil {
		for i, p := range parameters {
			if name == p.Name {
				param = &parameters[i]
			}
		}
	}
	if param != nil {
		switch param.Kind {

		case "secret":
			reallySecretRef := ""
			secretKind := ""
			if maybeMap, ok := param.Value.(map[string]interface{}); ok {
				if maybeSecretRef, ok := maybeMap["secret"]; ok {
					if secretRef, ok := maybeSecretRef.(string); ok {
						reallySecretRef = secretRef
						if maybeKind, ok := maybeMap["kind"]; ok {
							if kind, ok := maybeKind.(string); ok {
								secretKind = kind
							}
						}
					}
				}
			} else if secretRef, ok := param.Value.(string); ok {
				reallySecretRef = secretRef
			}
			if reallySecretRef != "" {
				s, err := secret(resource, reallySecretRef)
				if err != nil {
					return false, "", err
				}
				if kind, ok := s["kind"]; ok {
					if secretKind != "" && kind != secretKind {
						util.Warn("Secret `%s` kind `%s` doesn't match kind `%s` stored in Secrets Service",
							name, secretKind, kind)
					}
					switch kind {
					case "text", "password", "certificate", "sshKey", "privateKey", "loginToken":
						return true, s[kind], nil
					case "gitAccessToken":
						return true, s["loginToken"], nil
					case "usernamePassword":
						return true, fmt.Sprintf("%s/%s", s["username"], s["password"]), nil
					case "cloudAccessKeys":
						return true, fmt.Sprintf("%s:%s", s["accessKey"], s["secretKey"]), nil
					case "cloudAccount":
						return true, fmt.Sprintf("%s/%s", s["roleArn"], s["externalId"]), nil
					}
				}
			}
			return false, "", fmt.Errorf("Unable to retrieve secret `%s`: `%+v` is not a known secret reference",
				name, param.Value)

		case "license":
			if id, ok := param.Value.(string); ok && id != "" {
				l, err := license(id)
				if err != nil {
					return false, "", err
				}
				return true, l.LicenseKey, nil
			}
			return false, "", fmt.Errorf("Unable to retrieve license `%s`: `%+v` is not a license reference",
				name, param.Value)

		default:
			if param.Value == nil {
				return false, "", fmt.Errorf("Unable to retrieve parameter `%s`: `value` not set", name)
			}
			if scalar, ok := param.Value.(string); ok {
				return true, scalar, nil
			}
			if scalar, ok := param.Value.(bool); ok {
				return true, strconv.FormatBool(scalar), nil
			}
			if scalar, ok := param.Value.(int64); ok {
				return true, strconv.FormatInt(scalar, 10), nil
			}
			if scalar, ok := param.Value.(float64); ok {
				return true, strconv.FormatFloat(scalar, 'f', -1, 64), nil
			}
			return false, "", fmt.Errorf("Unable to retrieve parameter `%s`: `%+v` is not a plain scalar value",
				name, param.Value)
		}
	}
	return false, "", nil
}

func createSecretParameter(environment, environmentId, name, component string) (string, error) {
	kind := "password"
	value, err := util.RandomString(8)
	if err != nil {
		return "", err
	}
	secretId, err := createSecret(environmentsResource, environmentId, name, component, kind,
		map[string]string{kind: value})
	if err != nil {
		return "", fmt.Errorf("Unable to create secret `%s` in environment `%s`: %v", name, environment, err)
	}
	if config.Verbose {
		log.Printf("Secret `%s` created in environment `%s` with id `%s`", name, environment, secretId)
	}
	return value, nil
}

func parameterQName(param Parameter) string {
	qName := param.Name
	if param.Component != "" {
		qName = fmt.Sprintf("%s|%s", param.Name, param.Component)
	}
	return qName
}

func sortParameters(params []Parameter) []Parameter {
	keys := make([]string, 0, len(params))
	indx := make(map[string]int)
	for i := range params {
		name := parameterQName(params[i])
		keys = append(keys, name)
		indx[name] = i
	}
	sort.Strings(keys)
	sorted := make([]Parameter, 0, len(params))
	for _, name := range keys {
		sorted = append(sorted, params[indx[name]])
	}
	return sorted
}

func maybePendingSecretCopy(param Parameter) (string, bool) {
	if param.Kind == "secret" && param.Value == nil && param.From != "" {
		return fmt.Sprintf("<pending copy %s>", param.From), true
	}
	return "", false
}

func formatParameter(resource string, param Parameter, showSecret bool) (string, error) {
	var err error
	value, isCopy := maybePendingSecretCopy(param)
	if !isCopy {
		value, err = formatParameterValue(resource, param.Kind, param.Value, showSecret)
	}
	additional := ""
	if param.Origin != "" || param.Messenger != "" {
		if param.Origin != "" {
			additional = fmt.Sprintf("/%s", param.Origin)
		}
		additional = fmt.Sprintf(" *%s%s*", param.Messenger, additional)
	}
	title := fmt.Sprintf("%7s %s:", param.Kind, parameterQName(param))
	if strings.Contains(value, "\n") {
		maybeNl := "\n"
		if strings.HasSuffix(value, "\n") {
			maybeNl = ""
		}
		return fmt.Sprintf("%s ~~%s %s%s~~", title, additional, value, maybeNl), err
	} else {
		return fmt.Sprintf("%s %v%s", title, value, additional), err
	}
}

func formatParameterValue(resource string, kind string, value interface{}, showSecret bool) (string, error) {
	var err error
	switch kind {

	case "license":
		if id, ok := value.(string); ok && id != "" {
			l, err2 := license(id)
			if err2 != nil {
				err = err2
				value = "(error)"
			} else {
				if showSecret {
					value = fmt.Sprintf("%s : %s", l.Component, l.LicenseKey)
				} else {
					value = fmt.Sprintf("[%s] <hidden>", id)
				}
			}
		}

	case "secret":
		reallySecretRef := ""
		annotation := ""
		if maybeMap, ok := value.(map[string]interface{}); ok {
			if maybeSecretRef, ok := maybeMap["secret"]; ok {
				if secretRef, ok := maybeSecretRef.(string); ok {
					annotation = fmt.Sprintf("%s, %s", maybeMap["kind"], secretRef)
					reallySecretRef = secretRef
				}
			}
		} else if secretRef, ok := value.(string); ok {
			annotation = secretRef
			reallySecretRef = secretRef
		}
		if len(reallySecretRef) == 36 { // uuid
			value2 := ""
			s, err2 := secret(resource, reallySecretRef)
			if err2 != nil {
				err = err2
				value2 = "<error>"
			} else if s != nil {
				if showSecret {
					value2 = formatSecret(s)
				} else {
					value2 = "<hidden>"
				}
			} else {
				value2 = "<nil>"
			}
			if strings.Contains(value2, "\n") {
				value = fmt.Sprintf("(%s)\n%s", annotation, value2)
			} else {
				value = fmt.Sprintf("%s (%s)", value2, annotation)
			}
		}
	}

	return fmt.Sprintf("%v", value), err
}
