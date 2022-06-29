// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package manifest

import (
	"log"
	"sort"
	"strings"
)

func CheckComponentsExist(components []ComponentRef, toCheck ...string) {
	mustExist := make([]string, 0)
	for _, name := range toCheck {
		if name != "" {
			mustExist = append(mustExist, name)
		}
	}
	if len(mustExist) == 0 {
		return
	}
	names := ComponentsNamesFromRefs(components)
	sort.Strings(names)
	for _, nameToCheck := range mustExist {
		found := false
		for _, name := range names {
			if nameToCheck == name {
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("Component `%s` is specified but no such component found in `%s`",
				nameToCheck, strings.Join(names, ", "))
		}
	}
}
