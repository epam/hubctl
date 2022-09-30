// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build api

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	gosocketio "github.com/arkadijs/golang-socketio"
	gosocketiotransport "github.com/arkadijs/golang-socketio/transport"
	"github.com/gorilla/websocket"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/util"
)

const (
	requestsBindata  = "cmd/hub/api/requests"
	socketIoResource = "hub/socket.io/"
)

var (
	MethodsWithJsonBody = []string{"POST", "PUT", "PATCH"}
	error404            = errors.New("404 HTTP")
	_hubApi             *http.Client
	_hubApiLongWait     *http.Client
	//lint:ignore U1000 still needed?
	wsApi = &websocket.Dialer{HandshakeTimeout: 10 * time.Second}
)

func Invoke(method, path string, body io.Reader) {
	method = strings.ToUpper(method)

	path = strings.TrimPrefix(path, "/")
	code, resp, err := doWithAuthorization(hubApi(), method, path, body, nil)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if config.Debug {
		log.Printf("HTTP %d", code)
	}
	if config.Verbose {
		os.Stdout.Write(resp)
	}
	if code >= 300 {
		os.Exit(code)
	}
}

func hubApi() *http.Client {
	if _hubApi == nil {
		if config.Trace {
			log.Printf("API timeout: %ds", config.ApiTimeout)
		}
		_hubApi = util.RobustHttpClient(time.Duration(config.ApiTimeout)*time.Second, false)
	}
	return _hubApi
}

func hubApiLongWait() *http.Client {
	if _hubApiLongWait == nil {
		if config.Trace {
			log.Printf("API long timeout: %ds", config.ApiTimeout)
		}
		_hubApiLongWait = util.RobustHttpClient(time.Duration(config.ApiTimeout)*time.Second, false)
	}
	return _hubApiLongWait
}

func hubRequestWithBearerToken(method, path string, body io.Reader) (*http.Request, error) {
	return hubRequest(method, path, bearerToken(), body)
}

func hubRequest(method, path, token string, body io.Reader) (*http.Request, error) {
	addr := fmt.Sprintf("%s/%s", config.ApiBaseUrl, path)
	if config.Trace {
		log.Printf(">>> %s %s", method, addr)
	}
	req, err := http.NewRequest(method, addr, body)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}
	if body != nil && util.Contains(MethodsWithJsonBody, method) {
		req.Header.Add("Content-type", "application/json")
	}
	return req, nil
}

//lint:ignore U1000 still needed?
func hubWs() (*websocket.Conn, *http.Response, error) {
	if !strings.HasPrefix(config.ApiBaseUrl, "http://") && !strings.HasPrefix(config.ApiBaseUrl, "https://") {
		log.Fatalf("Unable to construct Hub WebSocket URL from `%s`", config.ApiBaseUrl)
	}
	token := bearerToken()
	addr := fmt.Sprintf("%s/%s?accessToken=%s&EIO=3&transport=websocket", "ws"+config.ApiBaseUrl[4:], socketIoResource, url.QueryEscape(token))
	if config.Trace {
		log.Printf(">>> WS %s", addr)
	}
	ws, resp, err := wsApi.Dial(addr, nil)
	if config.Trace {
		if resp != nil {
			log.Printf("<<< WS %s", resp.Status)
		}
	}
	return ws, resp, err
}

func hubWsSocketIo(connect, disconnect, ex func()) (*gosocketio.Client, error) {
	if !strings.HasPrefix(config.ApiBaseUrl, "http://") && !strings.HasPrefix(config.ApiBaseUrl, "https://") {
		log.Fatalf("Unable to construct Hub WebSocket URL from `%s`", config.ApiBaseUrl)
	}
	token := bearerToken()
	addr := fmt.Sprintf("%s/%s?accessToken=%s&EIO=3&transport=websocket", "ws"+config.ApiBaseUrl[4:], socketIoResource, url.QueryEscape(token))
	if config.Trace {
		log.Printf(">>> WS %s", addr)
	}
	transport := gosocketiotransport.GetDefaultWebsocketTransport()
	transport.PingInterval = 25 * time.Second
	transport.PingTimeout = 5 * time.Second
	ws, err := gosocketio.Dial(addr, transport)
	if err != nil {
		return ws, err
	}
	err = ws.On(gosocketio.OnError, func(ch *gosocketio.Channel) {
		if config.Debug {
			log.Print("Hub WebSocket error")
		}
		if ex != nil {
			ex()
		}
	})
	if err != nil {
		return nil, err
	}
	if config.Verbose {
		err = ws.On(gosocketio.OnConnection, func(ch *gosocketio.Channel) {
			if config.Debug {
				log.Print("Hub WebSocket connected")
			}
			if connect != nil {
				connect()
			}
		})
		if err != nil {
			return nil, err
		}
		err = ws.On(gosocketio.OnDisconnection, func(ch *gosocketio.Channel) {
			if config.Debug {
				log.Print("Hub WebSocket disconnected")
			}
			if disconnect != nil {
				disconnect()
			}
		})
		if err != nil {
			return nil, err
		}
	}
	return ws, err
}

func get(client *http.Client, path string, jsResp interface{}) (int, error) {
	code, _, err := doWithAuthorization(client, "GET", path, nil, jsResp)
	return code, err
}

func get2(client *http.Client, path string) (int, []byte, error) {
	return doWithAuthorization(client, "GET", path, nil, nil)
}

func delete(client *http.Client, path string) (int, error) {
	code, _, err := doWithAuthorization(client, "DELETE", path, nil, nil)
	return code, err
}

func post(client *http.Client, path string, req interface{}, jsResp interface{}) (int, error) {
	return ppp(client, "POST", path, req, jsResp)
}

//lint:ignore U1000 still needed?
func put(client *http.Client, path string, req interface{}, jsResp interface{}) (int, error) {
	return ppp(client, "PUT", path, req, jsResp)
}

func patch(client *http.Client, path string, req interface{}, jsResp interface{}) (int, error) {
	code, err := ppp(client, "PATCH", path, req, jsResp)
	return code, err
}

func ppp(client *http.Client, method, path string, req interface{}, jsResp interface{}) (int, error) {
	var body io.Reader
	if req != nil {
		reqBody, err := json.Marshal(req)
		if err != nil {
			return 0, err
		}
		if config.Trace {
			addr := fmt.Sprintf("%s/%s", config.ApiBaseUrl, path)
			log.Printf(">>> %s %s\n%s", method, addr, identJson(reqBody))
		}
		body = bytes.NewReader(reqBody)
	}
	code, _, err := doWithAuthorization(client, method, path, body, jsResp)
	return code, err
}

// post2() trace no request body as it come from CLI user on stdin
func post2(client *http.Client, path string, req io.Reader, jsResp interface{}) (int, error) {
	code, _, err := doWithAuthorization(client, "POST", path, req, jsResp)
	return code, err
}

func patch2(client *http.Client, path string, req io.Reader, jsResp interface{}) (int, error) {
	code, _, err := doWithAuthorization(client, "PATCH", path, req, jsResp)
	return code, err
}

func doWithAuthorization(client *http.Client, method, path string, reqBody io.Reader, jsResp interface{}) (int, []byte, error) {
	req, err := hubRequestWithBearerToken(method, path, reqBody)
	if err != nil {
		return 0, nil, fmt.Errorf("Error creating HTTP request: %v", err)
	}
	if reqBody != nil && util.Contains(MethodsWithJsonBody, method) {
		req.Header.Add("Content-type", "application/json")
	}
	return do(client, req, jsResp)
}

func do(client *http.Client, req *http.Request, jsResp interface{}) (int, []byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("Error during HTTP request: %v", err)
	}
	respBody := func() ([]byte, int64, error) {
		var body bytes.Buffer
		read, err := body.ReadFrom(resp.Body)
		resp.Body.Close()
		bResp := body.Bytes()
		if config.Trace {
			pretty := indentIfJson(resp, bResp)
			if pretty != "" {
				log.Printf("<<<\n%s", pretty)
			}
		}
		return bResp, read, err
	}
	if config.Trace {
		log.Printf("<<< %s %s: %s", req.Method, req.URL.String(), resp.Status)
	}
	if resp.StatusCode == 404 {
		return resp.StatusCode, nil, error404
	}
	if resp.StatusCode >= 300 {
		b, _, _ := respBody()
		maybeErrors := decodeIfApiErrorsAndVerbose(resp, b)
		maybeJson := ""
		if maybeErrors == "" {
			maybeJson = indentIfJsonAndDebug(resp, b)
		}
		return resp.StatusCode, nil, fmt.Errorf("%d HTTP%s%s", resp.StatusCode, maybeErrors, maybeJson)
	}
	body, read, err := respBody()
	if err != nil || (read < 2 && resp.StatusCode != 202 && resp.StatusCode != 204 && jsResp != nil) {
		return resp.StatusCode, body, fmt.Errorf("%d HTTP, error reading response (read %d bytes): %s%s",
			resp.StatusCode, read, util.Errors2(err), indentIfJsonAndDebug(resp, body))
	}
	if jsResp != nil && read >= 2 {
		err = json.Unmarshal(body, jsResp)
		if err != nil {
			return resp.StatusCode, body, fmt.Errorf("%d HTTP, error unmarshalling response (read %d bytes): %v%s",
				resp.StatusCode, read, err, indentIfJsonAndDebug(resp, body))
		}
	}
	return resp.StatusCode, body, nil
}

func identJson(in []byte) []byte {
	js := in
	var pretty bytes.Buffer
	err := json.Indent(&pretty, in, "", "  ")
	if err != nil {
		log.Printf("Unable to indent JSON: %v", err)
	} else {
		js = pretty.Bytes()
	}
	return js
}

func indentIfJson(resp *http.Response, in []byte) string {
	ct := resp.Header.Get("Content-type")
	aj := "application/json"
	js := in
	if strings.HasPrefix(ct, aj) {
		js = identJson(in)
	} else {
		if config.Trace && resp.StatusCode != 204 {
			log.Printf("Response Content-type is not `%s`, but `%s`", aj, ct)
		}
	}
	return string(js)
}

func indentIfJsonAndDebug(resp *http.Response, b []byte) string {
	if !config.Debug {
		return ""
	}
	return "\n" + indentIfJson(resp, b)
}

func decodeIfApiErrorsAndVerbose(resp *http.Response, b []byte) string {
	if !config.Verbose {
		return ""
	}
	return "\n\t" + strings.Join(unmarshalAndDecodeApiErrors(resp, b), "\n\t")
}

func unmarshalAndDecodeApiErrors(resp *http.Response, in []byte) []string {
	ct := resp.Header.Get("Content-type")
	aj := "application/json"
	if !strings.HasPrefix(ct, aj) {
		return nil
	}
	var js ApiErrors
	err := json.Unmarshal(in, &js)
	if err != nil {
		if config.Verbose {
			log.Printf("Error response doesn't look like informative API error: %v", err)
		}
		return nil
	}
	return decodeApiErrors(js.Errors, "")
}

func decodeApiErrors(es []ApiError, indent string) []string {
	errs := make([]string, 0, len(es))
	for _, e := range es {
		errs = append(errs, decodeApiError(e, indent))
	}
	return errs
}

func decodeApiError(e ApiError, indent string) string {
	source := ""
	if e.Source != "" {
		source = fmt.Sprintf(" at %s", e.Source)
	}
	meta := e.Meta
	nested := ""
	if len(meta.Data.Errors) > 0 {
		nested = fmt.Sprintf("\n%s\tNested API error:\n%s\t%s",
			indent, indent,
			strings.Join(decodeApiErrors(meta.Data.Errors, indent+"\t"), "\n\t"+indent))
	}
	validation := ""
	if meta.SchemaPath != "" {
		validation = fmt.Sprintf("\n%s\t\tValidation: %s %+v", indent, meta.Message, meta.Params)
	}
	stack := ""
	if meta.Stack != "" && config.Debug {
		frames := strings.Split(meta.Stack, "\n")
		stack = fmt.Sprintf("\n%s\tStack: %s", indent, strings.Join(frames, "\n\t"+indent))
	}
	return fmt.Sprintf("%s%s%s: %s%s%s%s", indent, e.Type, source, e.Detail, stack, validation, nested)
}
