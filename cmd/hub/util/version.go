// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package util

import "fmt"

var (
	ref     = "master"
	commit  = "HEAD"
	buildAt = "now"
)

func Version() string {
	return fmt.Sprintf("%s %s build at %s", ref, commit, buildAt)
}
