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

const userResource = "hub/api/v1/user"

func userDeploymentKey(subject string) (string, error) {
	if subject != "" {
		subject = "?subject=" + url.QueryEscape(subject)
	}
	path := fmt.Sprintf("%s/deployment-key%s", userResource, subject)
	var jsResp DeploymentKey
	code, err := get(hubApi(), path, &jsResp)
	if err != nil {
		return "", fmt.Errorf("Error querying SuperHub User Deployment Key: %v", err)
	}
	if code != 200 {
		return "", fmt.Errorf("Got %d HTTP querying SuperHub User Deployment Key, expected 200 HTTP", code)
	}
	key := jsResp.DeploymentKey
	if key == "" {
		return "", fmt.Errorf("Got empty User Deployment Key")
	}
	return key, nil
}
