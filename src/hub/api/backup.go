package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"hub/config"
	"hub/util"
)

const backupsResource = "hub/api/v1/backups"

func Backups(selector string, showLogs, jsonFormat bool) {
	backups, err := backupsBy(selector)
	if err != nil {
		log.Fatalf("Unable to query for Backups(s): %v", err)
	}
	if len(backups) == 0 {
		if jsonFormat {
			log.Print("No Backups")
		} else {
			fmt.Print("No Backups\n")
		}
	} else {
		if jsonFormat {
			var toMarshal interface{}
			if len(backups) == 1 {
				toMarshal = &backups[0]
			} else {
				toMarshal = backups
			}
			out, err := json.MarshalIndent(toMarshal, "", "  ")
			if err != nil {
				log.Fatalf("Error marshalling JSON response for output: %v", err)
			}
			os.Stdout.Write(out)
			os.Stdout.Write([]byte("\n"))
		} else {
			fmt.Print("Backups:\n")
			errors := make([]error, 0)
			for _, backup := range backups {
				errors = formatBackupEntity(&backup, showLogs, errors)
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

func formatBackupEntity(backup *Backup, showLogs bool, errors []error) []error {
	title := formatBackupTitle(backup)
	if backup.Description != "" {
		title = fmt.Sprintf("%s - %s", title, backup.Description)
	}
	fmt.Printf("\n\t%s @ %v\n", title, backup.Timestamp)
	if len(backup.Tags) > 0 {
		fmt.Printf("\t\tTags: %s\n", strings.Join(backup.Tags, ", "))
	}
	fmt.Printf("\t\tStatus: %s\n", backup.Status)
	if len(backup.Components) > 0 {
		fmt.Printf("\t\tComponents: %s\n", strings.Join(backup.Components, ", "))
	}
	if backup.Environment.Name != "" {
		fmt.Printf("\t\tEnvironment: %s\n", formatEnvironmentRef(&backup.Environment))
	}
	if backup.StackInstance.Name != "" {
		resource := fmt.Sprintf("%s/%s", backupsResource, backup.Id)
		instance, errs := formatStackInstanceRef(&backup.StackInstance, resource)
		fmt.Printf("\t\tStack Instance: %s\n", instance)
		errors = append(errors, errs...)
	}
	if backup.Bundle.Kind != "" {
		bundle := backup.Bundle
		fmt.Printf("\t\tBundle: %s @ %v\n", bundle.Kind, bundle.Timestamp)
		fmt.Printf("\t\t\tStatus: %s\n", bundle.Status)
		if len(bundle.Components) > 0 {
			fmt.Print("\t\t\tComponents:\n")
			for name, comp := range bundle.Components {
				fmt.Printf("\t\t\t\t%s [%s] @ %v\n", name, comp.Kind, comp.Timestamp)
				fmt.Printf("\t\t\t\tStatus: %s\n", comp.Status)
				if len(comp.Outputs) > 0 {
					fmt.Print("\t\t\t\tOutputs:\n")
					for _, output := range comp.Outputs {
						fmt.Printf("\t\t\t\t\t%s: %s\n", output.Name, output.Value)
					}
				}
			}
		}
	}
	if showLogs && backup.Logs != "" {
		fmt.Printf("\t\tLogs:\n\t\t\t%s\n", strings.Join(strings.Split(backup.Logs, "\n"), "\n\t\t\t"))
	}
	return errors
}

func formatBackup(backup *Backup) {
	errors := formatBackupEntity(backup, false, make([]error, 0))
	if len(errors) > 0 {
		fmt.Print("Errors encountered formatting response:\n")
		for _, err := range errors {
			fmt.Printf("\t%v\n", err)
		}
	}
}

func showBackup(id string) {
	backup, err := backupById(id)
	if err != nil {
		fmt.Printf("%v", err)
	}
	if backup != nil {
		formatBackup(backup)
	}
}

func backupBy(selector string) (*Backup, error) {
	if !util.IsUint(selector) {
		return backupByName(selector)
	}
	return backupById(selector)
}

func backupsBy(selector string) ([]Backup, error) {
	if !util.IsUint(selector) {
		return backupsByName(selector)
	}
	backup, err := backupById(selector)
	if err != nil {
		return nil, err
	}
	if backup != nil {
		return []Backup{*backup}, nil
	}
	return nil, nil
}

func backupById(id string) (*Backup, error) {
	path := fmt.Sprintf("%s/%s", backupsResource, url.PathEscape(id))
	var jsResp Backup
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Backups: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Backups, expected 200 HTTP", code)
	}
	return &jsResp, nil
}

func backupByName(name string) (*Backup, error) {
	backups, err := backupsByName(name)
	if err != nil {
		return nil, fmt.Errorf("Unable to query for Backup `%s`: %v", name, err)
	}
	if len(backups) == 0 {
		return nil, fmt.Errorf("No Backup `%s` found", name)
	}
	if len(backups) > 1 {
		return nil, fmt.Errorf("More than one Backup returned by name `%s`", name)
	}
	backup := backups[0]
	return &backup, nil
}

func backupsByName(name string) ([]Backup, error) {
	return backupsByFilter("name", name)
}

func backupsByEnvironmentId(id string) ([]Backup, error) {
	return backupsByFilter("environment", id)
}

func backupsByInstanceId(id string) ([]Backup, error) {
	return backupsByFilter("instance", id)
}

func backupsByPlatformId(id string) ([]Backup, error) {
	return backupsByFilter("platform", id)
}

func backupsByFilter(field, value string) ([]Backup, error) {
	path := backupsResource
	if field != "" && value != "" {
		path += fmt.Sprintf("?%s=%s", url.QueryEscape(field), url.QueryEscape(value))
	}
	var jsResp []Backup
	code, err := get(hubApi(), path, &jsResp)
	if code == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub Backups: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub Backups, expected 200 HTTP", code)
	}
	return jsResp, nil
}

func formatBackupTitle(backup *Backup) string {
	return fmt.Sprintf("%s [%s]", backup.Name, backup.Id)
}

func DeleteBackup(selector string) {
	err := deleteBackup(selector)
	if err != nil {
		log.Fatalf("Unable to delete SuperHub Backup: %v", err)
	}
}

func deleteBackup(selector string) error {
	backup, err := backupBy(selector)
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
	} else if backup == nil {
		return error404
	} else {
		id = backup.Id
	}
	force := ""
	if config.Force {
		force = "?force=true"
	}
	path := fmt.Sprintf("%s/%s%s", backupsResource, url.PathEscape(id), force)
	code, err := delete(hubApi(), path)
	if err != nil {
		return err
	}
	if code != 202 && code != 204 {
		return fmt.Errorf("Got %d HTTP deleting SuperHub Backup, expected [202, 204] HTTP", code)
	}
	return nil
}
