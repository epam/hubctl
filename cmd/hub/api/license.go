// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package api

import (
	"fmt"
	"net/url"
)

const licensesResource = "hub/api/v1/licenses"

func license(id string) (*License, error) {
	path := fmt.Sprintf("%s/%s", licensesResource, url.PathEscape(id))
	var jsResp License
	code, err := get(hubApi(), path, &jsResp)
	if err != nil {
		return nil, fmt.Errorf("Error querying SuperHub License `%s`: %v",
			id, err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP querying SuperHub License `%s`, expected 200 HTTP",
			code, id)
	}
	return &jsResp, nil
}
