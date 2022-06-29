// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package metrics

type Metric struct {
	Metric    string            `json:"metric"`
	Kind      string            `json:"kind,omitempty"`
	Unit      string            `json:"unit,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	Value     int64             `json:"value"`
	Timestamp int64             `json:"timestamp,omitempty"`
}

type Series []Metric

type SeriesResponse struct {
	Status string
}

type DDMetric struct {
	Metric string    `json:"metric"`
	Type   string    `json:"type,omitempty"`
	Host   string    `json:"host,omitempty"`
	Tags   []string  `json:"tags,omitempty"`
	Points [][]int64 `json:"points"`
	// Interval int
}

/*
	"metric": "hubcli.commands.usage",
	"type": "count",
	"host": "714cbf9b-f8df-4362-8aea-b7321ba33a2e",
	"tags": ["command:hub-elaborate", "status:success", "machine-id:714cbf9b-f8df-4362-8aea-b7321ba33a2e"],
	"points": [
	  	[
			$NOW,
			1
		]
	]
*/

type DDSeries struct {
	Series []DDMetric `json:"series,omitempty"`
}

type DDSeriesResponse struct {
	Status string
}
