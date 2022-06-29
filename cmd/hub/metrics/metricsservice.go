// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/agilestacks/hub/cmd/hub/config"
)

const metricsSeriesResource = "metrics/api/v1/series"

func putMetricsServiceMetric(cmd, host string, additionaTags []string) error {
	tags := map[string]string{
		"command": cmd,
	}
	if host != "" {
		tags["machine-id"] = host
	}
	if len(additionaTags) > 0 {
		for _, t := range additionaTags {
			kv := strings.SplitN(t, ":", 2)
			k := kv[0]
			v := "true"
			if len(kv) > 1 {
				v = kv[1]
			}
			tags[k] = v
		}
	}
	series := Series{
		Metric{
			Metric:    "hubcli.commands.usage",
			Kind:      "count",
			Tags:      tags,
			Value:     1,
			Timestamp: time.Now().Unix(),
		},
	}
	reqBody, err := json.Marshal(series)
	if err != nil {
		return err
	}
	method := "POST"
	addr := fmt.Sprintf("%s/%s", config.ApiBaseUrl, metricsSeriesResource)
	req, err := http.NewRequest(method, addr, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	if config.Trace {
		log.Printf(">>> %s %s", req.Method, req.URL.String())
		log.Printf("%s", string(reqBody))
	}
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("X-API-Secret", MetricsServiceKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error during HTTP request: %v", err)
	}
	if config.Trace {
		log.Printf("<<< %s %s: %s", req.Method, req.URL.String(), resp.Status)
	}
	if resp.StatusCode != 202 {
		return fmt.Errorf("Metrics Service returned HTTP status %d; expected 202", resp.StatusCode)
	}
	var body bytes.Buffer
	read, err := body.ReadFrom(resp.Body)
	resp.Body.Close()
	bResp := body.Bytes()
	if read == 0 {
		return errors.New("Empty response")
	}
	var jsResp SeriesResponse
	err = json.Unmarshal(bResp, &jsResp)
	if err != nil {
		return fmt.Errorf("Error unmarshalling HTTP response: %v", err)
	}
	if jsResp.Status != "ok" {
		return fmt.Errorf("Metrics Service returned status `%s`; expected `ok`", jsResp.Status)
	}
	return nil
}
