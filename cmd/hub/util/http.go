// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package util

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

func RobustHttpClient(timeout time.Duration, insecureSkipVerify bool) *http.Client {
	if timeout == 0 {
		timeout = time.Duration(10) * time.Second
	}
	transport := &http.Transport{
		ResponseHeaderTimeout: timeout,
		TLSHandshakeTimeout:   timeout,
		DialContext:           (&net.Dialer{Timeout: timeout}).DialContext,
		DisableKeepAlives:     true,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: insecureSkipVerify},
	}
	client := &http.Client{Transport: transport, Timeout: timeout}
	return client
}
