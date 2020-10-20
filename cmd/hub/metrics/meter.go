package metrics

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/filecache"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var (
	httpClient   = util.RobustHttpClient(0, false)
	cachedConfig *filecache.Metrics
)

func MeterCommand(cmd *cobra.Command, connectStdin bool) io.WriteCloser {
	if ddKey == "" && MetricsServiceKey == "" {
		return nil
	}
	stdin, err := meterCommand(cmd, connectStdin)
	if err != nil {
		util.Warn("Unable to send usage metrics: %v", err)
	}
	return stdin
}

func meterCommand(cmd *cobra.Command, connectStdin bool) (io.WriteCloser, error) {
	enabled, _, err := meteringConfig()
	if err != nil {
		return nil, fmt.Errorf("Unable to load metrics config: %v", err)
	}
	if !enabled {
		if config.Trace {
			log.Print("Usage metering is not enabled")
		}
		return nil, nil
	}
	bin, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("Unable to determine path to Hub CLI executable: %v", err)
	}
	args := []string{"hub", "util", "metrics"}
	if connectStdin {
		args = append(args, "--tags-stdin")
	}
	args = append(args, commandStr(cmd))
	hub := exec.Cmd{
		Path: bin,
		Args: args,
	}
	if config.Trace {
		hub.Stdout = os.Stdout
		hub.Stderr = os.Stderr
	}
	var stdin io.WriteCloser
	if connectStdin {
		stdin, err = hub.StdinPipe()
		if err != nil {
			return nil, err
		}
	}
	err = hub.Start()
	if err != nil {
		if stdin != nil {
			stdin.Close()
		}
		return nil, err
	}
	go hub.Wait()
	return stdin, nil
}

func PutMetrics(cmd string, tags []string) {
	err := putMetrics(cmd, tags)
	if err != nil {
		log.Fatalf("Unable to send usage metrics: %v", err)
	}
}

func putMetrics(cmd string, tags []string) error {
	enabled, host, err := meteringConfig()
	if err != nil {
		return fmt.Errorf("Unable to load metrics config: %v", err)
	}
	if config.Debug && !enabled && (ddKey != "" || MetricsServiceKey != "") {
		log.Print("Usage metering is not enabled; continuing as requested")
	}
	var err1, err2 error
	if MetricsServiceKey != "" {
		err1 = putMetricsServiceMetric(cmd, host, tags)
	}
	if ddKey != "" {
		err2 = putDDMetric(cmd, host, tags)
	}
	if err1 != nil || err2 != nil {
		err = errors.New(util.Errors2(err1, err2))
	}
	return err
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
	if conf.Host == nil && (config.Tty && !config.TtyForced) {
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
		if cache == nil {
			cache = &filecache.FileCache{}
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
