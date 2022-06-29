// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package util

import (
	"fmt"
	"log"
	"strings"
)

func PrintDeps(deps map[string][]string) {
	for _, name := range SortedKeys2(deps) {
		log.Printf("\t%s => %s", name, strings.Join(deps[name], ", "))
	}
}

func SprintDeps(deps map[string][]string) string {
	strs := make([]string, 0, len(deps))
	for _, name := range SortedKeys2(deps) {
		strs = append(strs, fmt.Sprintf("\t%s => %s", name, strings.Join(deps[name], ", ")))
	}
	return strings.Join(strs, "\n")
}

func PrintMap(m map[string]string) {
	for _, k := range SortedKeys(m) {
		log.Printf("\t%s => `%s`", k, m[k])
	}
}

func PrintMap2(m map[string][]string) {
	for _, k := range SortedKeys2(m) {
		log.Printf("\t%s => `%s`", k, strings.Join(m[k], ", "))
	}
}
