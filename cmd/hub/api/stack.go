// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build api

package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
)

const stacksResource = "hub/api/v1/stacks"

func BaseStacks(selector string, jsonFormat bool) {
	stacks, err := stacksBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Stack(s): %v", err)
	}
	if len(stacks) == 0 {
		if jsonFormat {
			log.Print("No Stacks")
		} else {
			fmt.Print("No Stacks\n")
		}
	} else {
		if jsonFormat {
			var toMarshal interface{}
			if len(stacks) == 1 {
				toMarshal = &stacks[0]
			} else {
				toMarshal = stacks
			}
			out, err := json.MarshalIndent(toMarshal, "", "  ")
			if err != nil {
				log.Fatalf("Error marshalling JSON response for output: %v", err)
			}
			os.Stdout.Write(out)
			os.Stdout.Write([]byte("\n"))
		} else {
			errors := make([]error, 0)
			fmt.Print("Stacks:\n")
			for _, stack := range stacks {
				formatBaseStackEntity(&stack, errors)
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

func formatBaseStackEntity(stack *BaseStack, errors []error) []error {
	title := fmt.Sprintf("%s [%s]", stack.Name, stack.Id)
	if stack.Brief != "" {
		title = fmt.Sprintf("%s - %s", title, stack.Brief)
	}
	fmt.Printf("\n\t%s\n", title)
	if len(stack.Tags) > 0 {
		fmt.Printf("\t\tTags: %s\n", strings.Join(stack.Tags, ", "))
	}
	if len(stack.Components) > 0 {
		fmt.Print("\t\tComponents:\n")
		for _, comp := range stack.Components {
			fmt.Printf("\t\t\t%s - %s - %s\n", comp.Name, comp.Brief, comp.Description)
		}
	}
	if len(stack.Parameters) > 0 {
		fmt.Print("\t\tParameters:\n")
	}
	resource := fmt.Sprintf("%s/%s", stacksResource, stack.Id)
	for _, param := range sortParameters(stack.Parameters) {
		formatted, err := formatParameter(resource, param, false)
		fmt.Printf("\t\t%s\n", formatted)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}
func stacksBy(selector string) ([]BaseStack, error) {
	if selector != "" {
		stack, err := stackById(selector)
		if err != nil {
			return nil, err
		} else if stack != nil {
			return []BaseStack{*stack}, nil
		}
		return nil, nil
	} else {
		return stacks()
	}
}

func stackById(id string) (*BaseStack, error) {
	path := fmt.Sprintf("%s/%s", stacksResource, url.PathEscape(id))
	var jsResp BaseStack
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Base Stacks: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Base Stacks, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func stacks() ([]BaseStack, error) {
	var jsResp []BaseStack
	code, err := get(hubApi(), stacksResource, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Base Stacks: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Base Stacks, expected 200 HTTP", code)
	}
	return jsResp, nil
}
