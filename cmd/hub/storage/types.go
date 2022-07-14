// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package storage

import (
	"time"
)

type File struct {
	Kind    string
	Path    string
	Exist   bool
	Size    int64
	ModTime time.Time
	Locked  bool
}

type Files struct {
	Kind  string
	Files []File
}
