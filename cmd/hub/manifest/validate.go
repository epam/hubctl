// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package manifest

import (
	"fmt"
	"log"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"

	"github.com/agilestacks/hub/cmd/hub/bindata"
	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var schemaLoader gojsonschema.JSONLoader

func validateManifest(name string, yamlDocument []byte) {
	err := validate(name, yamlDocument)
	if err != nil {
		util.Warn("Unable to validate `%s`: %v", name, err)
	}
}

func validate(name string, yamlDocument []byte) error {
	schema, err := manifestSchema()
	if err != nil {
		return err
	}

	var manifest interface{}
	err = yaml.Unmarshal(yamlDocument, &manifest)
	if err != nil {
		return err
	}
	converted, err := convertToStringKeysRecursive(manifest, "")
	if err != nil {
		return err
	}

	data := gojsonschema.NewGoLoader(converted)
	result, err := gojsonschema.Validate(schema, data)
	if err != nil {
		return err
	}
	if result.Valid() {
		if config.Trace {
			log.Printf("`%s` schema is valid", name)
		}
	} else {
		sep := "\n\t- "
		var errs []string
		for _, jserr := range result.Errors() {
			errs = append(errs, jserr.String())
		}
		util.Warn("`%s` schema is not valid:%s%s", name, sep, strings.Join(errs, sep))
		if config.Trace {
			log.Printf("Document validated:\n%+v", converted)
		}
	}
	return nil
}

func manifestSchema() (gojsonschema.JSONLoader, error) {
	if schemaLoader == nil {
		jsonBytes, err := bindata.Asset("meta/manifest.schema.json")
		if err != nil {
			return nil, fmt.Errorf("No manifest schema embedded: %v", err)
		}
		schemaLoader = gojsonschema.NewBytesLoader(jsonBytes)
	}
	return schemaLoader, nil
}

func convertForValidator(value interface{}) (interface{}, error) {
	return convertToStringKeysRecursive(value, "")
}

func convertToStringKeysRecursive(value interface{}, keyPrefix string) (interface{}, error) {
	if mapping, ok := value.(map[interface{}]interface{}); ok {
		dict := make(map[string]interface{})
		for key, entry := range mapping {
			str, ok := key.(string)
			if !ok {
				return nil, formatInvalidKeyError(keyPrefix, key)
			}
			var newKeyPrefix string
			if keyPrefix == "" {
				newKeyPrefix = str
			} else {
				newKeyPrefix = fmt.Sprintf("%s.%s", keyPrefix, str)
			}
			convertedEntry, err := convertToStringKeysRecursive(entry, newKeyPrefix)
			if err != nil {
				return nil, err
			}
			dict[str] = convertedEntry
		}
		return dict, nil
	}
	if list, ok := value.([]interface{}); ok {
		var convertedList []interface{}
		for index, entry := range list {
			newKeyPrefix := fmt.Sprintf("%s[%d]", keyPrefix, index)
			convertedEntry, err := convertToStringKeysRecursive(entry, newKeyPrefix)
			if err != nil {
				return nil, err
			}
			convertedList = append(convertedList, convertedEntry)
		}
		return convertedList, nil
	}
	return value, nil
}

func formatInvalidKeyError(keyPrefix string, key interface{}) error {
	var location string
	if keyPrefix == "" {
		location = "at top level"
	} else {
		location = fmt.Sprintf("in %s", keyPrefix)
	}
	return fmt.Errorf("Non-string key %s: %#v", location, key)
}
