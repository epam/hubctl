// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gcp

import "cloud.google.com/go/storage"

func IsNotFound(err error) bool {
	return err == storage.ErrObjectNotExist
}
