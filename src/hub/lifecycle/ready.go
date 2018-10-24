package lifecycle

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"hub/config"
	"hub/manifest"
	"hub/parameters"
)

func waitForReadyConditions(conditions []manifest.ReadyCondition,
	parameters parameters.LockedParameters, outputs parameters.CapturedOutputs) error {

	for _, condition := range conditions {
		err := waitForReadyCondition(condition, parameters, outputs)
		if err != nil {
			return err
		}
	}
	return nil
}

func expandReadyConditionParameter(what string, value string, kv map[string]string) string {
	piggy := manifest.Parameter{Name: fmt.Sprintf("lifecycle.readyCondition.%s", what), Value: value}
	parameters.ExpandParameter(&piggy, []string{}, kv)
	return piggy.Value
}

const defaultReadyConditionWaitSeconds = 1200

func waitForReadyCondition(condition manifest.ReadyCondition,
	params parameters.LockedParameters, outputs parameters.CapturedOutputs) error {

	if condition.PauseSeconds > 0 {
		if config.Verbose {
			log.Printf("Sleeping %d seconds", condition.PauseSeconds)
		}
		time.Sleep(time.Duration(condition.PauseSeconds) * time.Second)
	}

	if condition.DNS == "" && condition.URL == "" {
		return nil
	}

	wait := condition.WaitSeconds
	if wait <= 0 {
		wait = defaultReadyConditionWaitSeconds
	}
	kv := parameters.ParametersAndOutputsKV(params, outputs)
	if condition.DNS != "" {
		fqdn := expandReadyConditionParameter("DNS", condition.DNS, kv)
		err := waitForFqdn(maybeStripPort(fqdn), wait)
		if err != nil {
			return err
		}
	}
	if condition.URL != "" {
		url := expandReadyConditionParameter("URL", condition.URL, kv)
		err := waitForUrl(url, wait)
		if err != nil {
			return err
		}
	}
	return nil
}

func maybeStripPort(fqdn string) string {
	i := strings.Index(fqdn, ":")
	if i > 0 {
		return fqdn[0:i]
	}
	return fqdn
}

func waitForFqdn(fqdn string, waitSeconds int) error {
	if config.Verbose {
		log.Printf("Waiting for `%s` in DNS to resolve to an accessible address", fqdn)
	}
	start := time.Now()
	for time.Since(start) < time.Duration(waitSeconds)*time.Second {
		addrs, err := net.LookupHost(fqdn)
		if config.Debug {
			if err != nil {
				log.Printf("%v", err)
			} else {
				log.Printf("Resolved `%s` into: %v", fqdn, addrs)
			}
		}
		if err == nil && len(addrs) > 0 {
			addr := addrs[0]
			if len(addr) >= 7 && addr != "127.0.0.1" {
				return nil
			}
		}
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("Timeout waiting for `%s` to resolve", fqdn)
}

func waitForUrl(url string, waitSeconds int) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("Only HTTP and HTTPS is supported in lifecycle.readyCondition.URL, expanded to `%s`", url)
	}
	if config.Verbose {
		log.Printf("Waiting for `%s` to respond", url)
	}
	interval := time.Duration(10) * time.Second
	transport := &http.Transport{
		ResponseHeaderTimeout: interval,
		TLSHandshakeTimeout:   interval,
		DialContext:           (&net.Dialer{Timeout: interval}).DialContext,
		DisableKeepAlives:     true,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport, Timeout: interval}
	start := time.Now()
	for time.Since(start) < time.Duration(waitSeconds)*time.Second {
		response, err := client.Get(url)
		if config.Debug {
			if err != nil {
				log.Printf("%v", err)
			} else {
				if config.Trace {
					log.Printf("`%s` responded with:\n\t%+v", url, response)
				} else {
					log.Printf("`%s` responded with: %s", url, response.Status)
				}
			}
		}
		if err == nil {
			response.Body.Close()
			if response.StatusCode >= 100 && response.StatusCode < 500 {
				return nil
			}
			if config.Verbose {
				log.Printf("`%s` responded with: %s", url, response.Status)
			}
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("Timeout waiting for `%s` to respond", url)
}
