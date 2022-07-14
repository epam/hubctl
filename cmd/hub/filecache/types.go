// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package filecache

type AccessTokenBox struct {
	ApiBaseUrl     string
	LoginTokenHash uint64
	AccessToken    string
	RefreshToken   string
}

type Metrics struct {
	Disabled bool
	Host     *string `yaml:",omitempty"`
}

type FileCache struct {
	Version      int
	AccessTokens []AccessTokenBox
	Metrics      Metrics
}
