// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/util"
)

type CreateSecretResponse struct {
	Id string
}

func CreateSecret(entityKind, selector, name, component, kind string, values map[string]string) {
	id := ""
	resource := ""
	var parameters []Parameter
	var qErr error

	switch entityKind {
	case "environment":
		env, err := environmentBy(selector)
		qErr = err
		if err == nil && env != nil {
			id = env.Id
			resource = environmentsResource
			parameters = env.Parameters
		}
	case "template":
		template, err := templateBy(selector)
		qErr = err
		if err == nil && template != nil {
			id = template.Id
			resource = templatesResource
			parameters = template.Parameters
		}
	case "instance":
		instance, err := stackInstanceBy(selector)
		qErr = err
		if err == nil && instance != nil {
			id = instance.Id
			resource = stackInstancesResource
			parameters = instance.Parameters
		}
	default:
		log.Fatalf("Unknown entity kind `%s`", entityKind)
	}
	if id == "" && qErr == nil {
		qErr = errors.New("Not Found")
	}
	if qErr != nil {
		log.Fatalf("Unable to query %s: %v", entityKind, qErr)
	}

	for _, existing := range parameters {
		if !util.Empty(existing.Value) && existing.Name == name &&
			((existing.Component == "" && component == "") || existing.Component == component) {
			qname := name
			if component != "" {
				qname = fmt.Sprintf("%s|%s", name, component)
			}
			log.Fatalf("Parameter `%s` already exist in %s `%s` and is not empty", qname, entityKind, selector)
		}
	}

	secretId, err := createSecret(resource, id, name, component, kind, values)
	if err != nil {
		log.Fatalf("Unable to create %s secret: %v", entityKind, err)
	}
	if config.Verbose {
		log.Printf("Secret `%s` created in %s `%s` with id `%s`",
			name, entityKind, selector, secretId)
	}
	if config.Verbose {
		switch entityKind {
		case "environment":
			env, err := environmentById(id)
			if err == nil {
				formatEnvironment(env)
			}
		case "template":
			template, err := templateById(id)
			if err == nil {
				formatTemplate(template)
			}
		case "instance":
			instance, err := stackInstanceBy(id)
			if err == nil {
				formatStackInstance(instance)
			}
		}
	}
}

func createSecret(resource, id, name, component, kind string, values map[string]string) (string, error) {
	values["name"] = name
	if component != "" {
		values["component"] = component
	}
	values["kind"] = kind
	path := fmt.Sprintf("%s/%s/secrets", resource, url.PathEscape(id))
	var jsResp CreateSecretResponse
	code, err := post(hubApi(), path, values, &jsResp)
	if err != nil {
		return "", fmt.Errorf("Error creating SuperHub `%s/%s` Secret `%s`: %v",
			resource, id, name, err)
	}
	if code != 201 {
		return "", fmt.Errorf("Got %d HTTP creating SuperHub `%s` Secret `%s`, expected 201 HTTP",
			code, id, name)
	}
	return jsResp.Id, nil
}

func GetSecret(entityKind, selector, uuid string, jsonFormat bool) {
	if config.Debug {
		log.Printf("Getting %s/%s secret `%s`", entityKind, selector, uuid)
	}

	id := ""
	resource := ""
	var qErr error

	switch entityKind {
	case "cloudaccount":
		cloudAccount, err := cloudAccountBy(selector)
		qErr = err
		if err == nil && cloudAccount != nil {
			id = cloudAccount.Id
		}
		resource = cloudAccountsResource
	case "environment":
		env, err := environmentBy(selector)
		qErr = err
		if err == nil && env != nil {
			id = env.Id
		}
		resource = environmentsResource
	case "template":
		template, err := templateBy(selector)
		qErr = err
		if err == nil && template != nil {
			id = template.Id
		}
		resource = templatesResource
	case "instance":
		instance, err := stackInstanceBy(selector)
		qErr = err
		if err == nil && instance != nil {
			id = instance.Id
		}
		resource = stackInstancesResource
	case "application":
		application, err := applicationBy(selector)
		qErr = err
		if err == nil && application != nil {
			id = application.Id
		}
		resource = applicationsResource
	default:
		log.Fatalf("Unknown entity kind `%s`", entityKind)
	}
	if id == "" && qErr == nil {
		qErr = errors.New("Not Found")
	}
	if qErr != nil {
		msg := fmt.Sprintf("Unable to query %s %s: %v", entityKind, selector, qErr)
		if config.Force && util.IsUint(selector) {
			util.Warn("%s", msg)
			id = selector
		} else {
			log.Fatal(msg)
		}
	}

	resource = fmt.Sprintf("%s/%s", resource, id)

	resp, err := secret(resource, uuid)
	if err != nil {
		log.Fatalf("Unable to get secret: %v", err)
	}

	if jsonFormat {
		out, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			log.Fatalf("Error marshalling JSON response for output: %v", err)
		}
		os.Stdout.Write(out)
		os.Stdout.Write([]byte("\n"))
	} else {
		str, kind := formatSecret(resp)
		if config.Debug {
			log.Printf("Secret kind: %s", kind)
		}
		fmt.Println(str)
	}
}

func secret(resource, id string) (map[string]string, error) {
	path := fmt.Sprintf("%s/secrets/%s", resource, url.PathEscape(id))
	var jsResp map[string]string
	code, err := get(hubApi(), path, &jsResp)
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub `%s` Secret `%s`: %v",
			resource, id, err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub `%s` Secret `%s`, expected 200 HTTP",
			code, resource, id)
	}
	return jsResp, nil
}

func formatSecret(s map[string]string) (string, string) {
	str := ""
	k := ""
	if kind, ok := s["kind"]; ok {
		k = kind
		switch kind {
		case "text", "password", "certificate", "sshKey", "privateKey",
			"token", "bearerToken", "accessToken", "refreshToken", "loginToken":
			str = s[kind]
		case "license":
			str = s["licenseKey"]
		case "gitAccessToken": // legacy
			str = s["loginToken"]
		case "usernamePassword":
			str = fmt.Sprintf("%s/%s", s["username"], s["password"])
		case "cloudAccessKeys":
			str = fmt.Sprintf("%s:%s", s["accessKey"], s["secretKey"])
		case "cloudAccount":
			str = fmt.Sprintf("%s/%s", s["roleArn"], s["externalId"])
		}
	}
	if str == "" {
		str = fmt.Sprintf("%+v", s)
	}
	return str, k
}
