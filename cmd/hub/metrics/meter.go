package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/filecache"
	"github.com/agilestacks/hub/cmd/hub/util"
)

const (
	ddSeriesApi        = "https://api.datadoghq.com/api/v1/series"
	ddApiKeyEnvVarName = "DD_CLIENT_API_KEY"
)

var (
	dd           = util.RobustHttpClient(0, false)
	ddKey        string
	cachedConfig *filecache.Metrics
)

func MeterCommand(cmd *cobra.Command) {
	if ddKey == "" {
		return
	}
	err := meterCommand(cmd)
	if err != nil {
		util.Warn("Unable to send usage metrics: %v", err)
	}
}

func meterCommand(cmd *cobra.Command) error {
	enabled, _, err := meteringConfig()
	if err != nil {
		return fmt.Errorf("Unable to load metrics config: %v", err)
	}
	if !enabled {
		if config.Trace {
			log.Print("Usage metering is not enabled")
		}
		return nil
	}
	bin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("Unable to determine path to Hub CLI executable: %v", err)
	}
	os.Setenv(ddApiKeyEnvVarName, ddKey)
	hub := exec.Cmd{
		Path: bin,
		Args: []string{"hub", "util", "metrics", commandStr(cmd)},
	}
	if config.Trace {
		hub.Stdout = os.Stdout
		hub.Stderr = os.Stderr
	}
	err = hub.Start()
	if err != nil {
		return err
	}
	go hub.Wait()
	return nil
}

func PutMetrics(cmd string) error {
	enabled, host, err := meteringConfig()
	if config.Debug && !enabled {
		log.Print("Usage metering is not enabled; continuing as requested")
	}
	tags := make([]string, 0, 1)
	tags = append(tags, "command:"+cmd)
	series := DDSeries{
		[]DDMetric{{
			Metric: "hubcli.commands.usage",
			Type:   "count",
			Host:   host,
			Tags:   tags,
			Points: [][]int64{{time.Now().Unix(), 1}},
		}},
	}
	reqBody, err := json.Marshal(series)
	if err != nil {
		return err
	}
	method := "POST"
	addr := fmt.Sprintf("%s?api_key=%s", ddSeriesApi, ddKey)
	req, err := http.NewRequest(method, addr, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	if config.Trace {
		log.Printf(">>> %s %s", req.Method, req.URL.String())
		log.Printf("%s", string(reqBody))
	}
	req.Header.Add("Content-type", "application/json")
	resp, err := dd.Do(req)
	if err != nil {
		return fmt.Errorf("Error during HTTP request: %v", err)
	}
	if config.Trace {
		log.Printf("<<< %s %s: %s", req.Method, req.URL.String(), resp.Status)
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
		return fmt.Errorf("Status `%s`", jsResp.Status)
	}
	return nil
}

func commandStr(cmd *cobra.Command) string {
	var parts []string
	for cmd2 := cmd; cmd2 != nil; cmd2 = cmd2.Parent() {
		use := cmd2.Use
		i := strings.Index(use, " ")
		if i > 0 {
			use = use[:i]
		}
		parts = append(parts, use)
	}
	return strings.Join(util.Reverse(parts), "-")
}

func meteringConfig() (bool, string, error) {
	var cache *filecache.FileCache
	conf := cachedConfig
	if conf == nil {
		file, cache2, err := filecache.ReadCache(os.O_RDONLY)
		cache = cache2
		if err != nil {
			return false, "", err
		}
		if file != nil {
			file.Close()
		}
		if cache != nil {
			conf = &cache.Metrics
		}
	}
	// metrics disabled
	if conf != nil && conf.Disabled {
		cachedConfig = conf
		return !conf.Disabled, "", nil
	}
	if conf == nil {
		conf = &filecache.Metrics{}
	}
	// generate and save host uuid if interactive session
	var writeErr error
	if conf.Host == nil && util.IsTerminal() {
		u, err := uuid.NewRandom()
		if err != nil {
			util.Warn("Unable to generate host random v4 UUID: %v", err)
		}
		uuidStr := u.String()
		conf.Host = &uuidStr

		file, cache, err := filecache.ReadCache(os.O_RDWR | os.O_CREATE)
		if err != nil {
			return !conf.Disabled, "", err
		}
		if file == nil {
			return !conf.Disabled, "", errors.New("No cache file created")
		}
		defer file.Close()
		_, err = file.Seek(0, os.SEEK_SET)
		if err != nil {
			return !conf.Disabled, "", err
		}
		cache.Metrics = *conf
		writeErr = filecache.WriteCache(file, cache)
	}
	cachedConfig = conf
	host := ""
	if conf.Host != nil {
		host = *conf.Host
	}
	return !conf.Disabled, host, writeErr
}

func init() {
	ddKey = DDKey
	if ddKey == "" {
		ddKey = os.Getenv(ddApiKeyEnvVarName)
	}
}
