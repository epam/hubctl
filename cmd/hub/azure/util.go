// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package azure

import "strings"

func IsNotFound(err error) bool {
	str := err.Error()
	return strings.HasPrefix(str, "storage:") &&
		strings.Contains(str, "StatusCode=404")
}
