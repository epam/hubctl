package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/agilestacks/hub/cmd/hub/config"
)

const (
	ddSeriesApi        = "https://api.datadoghq.com/api/v1/series"
	ddApiKeyEnvVarName = "DD_CLIENT_API_KEY"
)

var ddKey string

func init() {
	ddKey = DDKey
	if ddKey == "" {
		ddKey = os.Getenv(ddApiKeyEnvVarName)
	}
}

func putDDMetric(cmd, host string) error {
	tags := make([]string, 0, 2)
	tags = append(tags, "command:"+cmd)
	if host != "" {
		tags = append(tags, "machine-id:"+host)
	}
	series := DDSeries{
		[]DDMetric{{
			Metric: "hubcli.commands.usage",
			Type:   "count",
			Tags:   tags,
			Points: [][]int64{{time.Now().Unix(), 1}},
		}},
	}
	reqBody, err := json.Marshal(series)
	if err != nil {
		return err
	}
	method := "POST"
	req, err := http.NewRequest(method, ddSeriesApi, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	if config.Trace {
		log.Printf(">>> %s %s", req.Method, req.URL.String())
		log.Printf("%s", string(reqBody))
	}
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("DD-API-KEY", ddKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error during HTTP request: %v", err)
	}
	if config.Trace {
		log.Printf("<<< %s %s: %s", req.Method, req.URL.String(), resp.Status)
	}
	if resp.StatusCode != 202 {
		return fmt.Errorf("Datadog returned HTTP status %d; expected 202", resp.StatusCode)
	}
	var body bytes.Buffer
	read, err := body.ReadFrom(resp.Body)
	resp.Body.Close()
	bResp := body.Bytes()
	if read == 0 {
		return errors.New("Empty response")
	}
	var jsResp DDSeriesResponse
	err = json.Unmarshal(bResp, &jsResp)
	if err != nil {
		return fmt.Errorf("Error unmarshalling HTTP response: %v", err)
	}
	if jsResp.Status != "ok" {
		return fmt.Errorf("Datadog returned status `%s`; expected `ok`", jsResp.Status)
	}
	return nil
}
