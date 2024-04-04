// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package storage

import (
	"log"

	"github.com/epam/hubctl/cmd/hub/util"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func printFiles(files []File, kind string) {
	log.Printf("%s %s:", cases.Title(language.Und).String(kind), util.Plural(len(files), "file"))
	for _, file := range files {
		locked := ""
		if file.Locked {
			locked = " [locked]"
		}
		if file.Exist {
			log.Printf("\t%s%s; size = %d; mod time = %v", file.Path, locked, file.Size, file.ModTime)
		} else {
			log.Printf("\t%s (not found)%s", file.Path, locked)
		}
	}
}
