// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/agilestacks/hub/cmd/hub/config"
)

const tasksResource = "hub/api/v1/tasks"

func Tasks(environmentSelector string, jsonFormat bool) {
	environmentId := ""
	if environmentSelector != "" {
		environment, err := environmentBy(environmentSelector)
		if err != nil {
			log.Fatalf("Unable to query for Environment: %v", err)
		}
		if environment == nil {
			log.Fatalf("No Environment `%s` found", environmentSelector)
		}
		environmentId = environment.Id
	}

	tasks, err := tasksByEnvironment(environmentId)
	if err != nil {
		log.Fatalf("Unable to query for Tasks: %v", err)
	}
	if len(tasks) == 0 {
		if jsonFormat {
			log.Print("No tasks")
		} else {
			fmt.Print("No tasks\n")
		}
	} else {
		if jsonFormat {
			var toMarshal interface{}
			if len(tasks) == 1 {
				toMarshal = &tasks[0]
			} else {
				toMarshal = tasks
			}
			out, err := json.MarshalIndent(toMarshal, "", "  ")
			if err != nil {
				log.Fatalf("Error marshalling JSON response for output: %v", err)
			}
			os.Stdout.Write(out)
			os.Stdout.Write([]byte("\n"))
		} else {
			errors := make([]error, 0)
			fmt.Print("Tasks:\n")
			for _, task := range tasks {
				formatTaskEntity(&task, errors)
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

func formatTaskEntity(task *Task, errors []error) []error {
	fmt.Printf("\n\t%s - %s\n", task.Id, task.JobId)
	fmt.Printf("\t\t%s: %s\n", task.Operation, task.Status)
	if task.StartTime != nil {
		duration := ""
		if task.CompletionTime != nil {
			duration = fmt.Sprintf("; duration %s", task.CompletionTime.Sub(*task.StartTime).Truncate(time.Second).String())
		}
		fmt.Printf("\t\tstart: %v%s\n", task.StartTime.Truncate(time.Second), duration)
	}
	fmt.Printf("\t\t%s: %s\n", task.EntityType, task.Kind)
	e := task.Entity
	if e.Domain != "" {
		fmt.Printf("\t\t%s\n", e.Domain)
	}
	if e.BaseDomain != "" {
		fmt.Printf("\t\t%s / %s\n", e.Kind, e.BaseDomain)
	}

	return errors
}

func tasksByEnvironment(environmentId string) ([]Task, error) {
	path := tasksResource
	if environmentId != "" {
		path = fmt.Sprintf("%s?environment=%s", path, url.QueryEscape(environmentId))
	}
	var jsResp []Task
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Tasks: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Tasks, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func TerminateTask(id string) {
	err := terminateTask(id)
	if err != nil {
		log.Fatalf("Unable to terminate SuperHub automation task: %v", err)
	}
}

func terminateTask(id string) error {
	req := TaskLifecycleRequest{true}
	path := fmt.Sprintf("%s/%s", tasksResource, id)
	code, err := post(hubApi(), path, req, nil)
	if err != nil {
		return err
	}
	if code != 200 && code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP in response to SuperHub automation task termination request, expected [200, 202, 204] HTTP",
			code)
	}
	if config.Verbose {
		log.Printf("Automation task %s termination request sent", id)
	}
	return nil
}
