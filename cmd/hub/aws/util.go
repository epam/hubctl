// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package aws

import (
	"strings"

	"github.com/agilestacks/hub/cmd/hub/config"
)

func IsNotFound(err error) bool {
	str := err.Error()
	return strings.HasPrefix(str, "NotFound:") &&
		strings.Contains(str, "status code: 404,")
}

func IsSlowDown(err error) bool {
	str := err.Error()
	return strings.Contains(str, "SlowDown: Please reduce your request rate.")
}

func arnRegion(arn string) string {
	region := config.AwsRegion
	parts := strings.Split(arn, ":")
	if len(parts) > 5 && parts[3] != "" { // full ARN
		region = parts[3]
	}
	return region
}
