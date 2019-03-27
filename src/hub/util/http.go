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
