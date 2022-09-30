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
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/util"
)

type AuthUserPass struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginTokenResponse struct {
	LoginToken string `json:"loginToken"`
}

type AuthLoginToken struct {
	LoginToken string `json:"loginToken"`
}

type SigninResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Exp          int64  `json:"exp,omitempty"`
}

type AuthPingResponse struct {
	Exp int64 `json:"exp"`
}

const loginTokenResource = "auth/api/v1/users/credentials/login-token"
const signinResource = "auth/api/v1/signin"
const authPingResource = "auth/api/v1/authenticated-ping"
const refreshResource = "auth/api/v1/refresh"

var accessTokenExpectedLifetime = 600 * time.Second

func Login(apiBaseUrl, username, password string) {
	token, err := loginForToken(apiBaseUrl, username, password)
	if err != nil {
		log.Fatalf("Unable to login: %v", err)
	}
	fmt.Printf(`# eval this in your shell
export HUB_API=%s
export HUB_TOKEN=%s
`, apiBaseUrl, token)
}

func loginForToken(apiBaseUrl, username, password string) (string, error) {
	reqBody, err := json.Marshal(&AuthUserPass{Username: username, Password: password})
	if err != nil {
		return "", fmt.Errorf("Error marshalling sign-in request: %v", err)
	}
	path := fmt.Sprintf("%s/%s", apiBaseUrl, loginTokenResource)
	if config.Trace {
		log.Printf(">>> POST %s\n%s", path, identJson(reqBody))
	}
	req, err := http.NewRequest("POST", path, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("Error creating Auth Service sign-in request: %v", err)
	}
	req.Header.Add("Content-type", "application/json")

	var jsResp LoginTokenResponse
	code, _, err := do(hubApi(), req, &jsResp)
	if code == 404 {
		return "", fmt.Errorf("No user found (404 HTTP)")
	}
	if err != nil {
		return "", fmt.Errorf("Sign-in error: %v", err)
	}
	if code != 200 {
		return "", fmt.Errorf("Got %d HTTP from Auth Service sign-in, expected 200 HTTP", code)
	}
	if jsResp.LoginToken == "" {
		return "", fmt.Errorf("Empty or no `loginToken` in sign-in response")
	}
	return jsResp.LoginToken, nil
}

func loginWithToken(apiBaseUrl, token string) (*SigninResponse, error) {
	reqBody, err := json.Marshal(&AuthLoginToken{token})
	if err != nil {
		return nil, fmt.Errorf("Error marshalling sign-in request: %v", err)
	}
	path := fmt.Sprintf("%s/%s", apiBaseUrl, signinResource)
	if config.Trace {
		log.Printf(">>> POST %s\n%s", path, identJson(reqBody))
	}
	req, err := http.NewRequest("POST", path, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("Error creating Auth Service sign-in request: %v", err)
	}
	req.Header.Add("Content-type", "application/json")

	var jsResp SigninResponse
	code, _, err := do(hubApi(), req, &jsResp)
	if code == 404 {
		return nil, fmt.Errorf("No user found (404 HTTP)")
	}
	if err != nil {
		return nil, fmt.Errorf("Sign-in error: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP from Auth Service sign-in, expected 200 HTTP", code)
	}
	if jsResp.AccessToken == "" {
		return nil, fmt.Errorf("Empty or no `accessToken` in sign-in response")
	}
	return &jsResp, nil
}

var hubApiBearerToken string
var hubApiBearerTokenExpTime time.Time

func tokenTimeValid(tokenTime time.Time) bool {
	return tokenTime.After(time.Now().Add(accessTokenExpectedLifetime))
}

func bearerToken() string {
	if hubApiBearerToken != "" && tokenTimeValid(hubApiBearerTokenExpTime) {
		return hubApiBearerToken
	}
	if config.ApiLoginToken == "" {
		log.Fatalf("Login token is not supplied - use `hub login` to obtain one")
	}

	cachedTokens, err := loadAccessToken(config.ApiBaseUrl, config.ApiLoginToken)
	if err != nil {
		util.Warn("Unable to load cached API access token (requesting new): %v", err)
	}
	if cachedTokens != nil {
		code, resp, err := verifyAccessToken(cachedTokens.AccessToken)
		if err != nil {
			if code != 401 {
				util.Warn("Unable to verify API access token (requesting new): %v", err)
			}
		} else {
			if code == 200 {
				if resp != nil {
					exp := time.Unix(resp.Exp, 0)
					if config.Debug {
						log.Printf("API access token verified; expiry at %v", exp)
					}
					if tokenTimeValid(exp) {
						hubApiBearerToken = cachedTokens.AccessToken
						hubApiBearerTokenExpTime = exp
						return hubApiBearerToken
					}
					if config.Debug {
						log.Print("API access token must be refreshed")
					}
				}
			} else if code != 401 {
				util.Warn("Got %d HTTP while verifying API access token - expected 401 HTTP (requesting new)", code)
			}

			resp, err := refreshAccessToken(cachedTokens)
			if err != nil {
				util.Warn("Unable to refresh API access token (requesting new): %v", err)
			}
			if resp != nil {
				exp := time.Unix(resp.Exp, 0)
				if config.Debug {
					log.Printf("API access token refreshed; expiry at %v", exp)
				}
				if !tokenTimeValid(exp) {
					util.Warn("API refreshed token expiry too short %v", exp)
				}
				storeAccessToken(config.ApiBaseUrl, config.ApiLoginToken, resp)
				if err != nil {
					util.Warn("Unable to store API access token in cache: %v", err)
				}
				hubApiBearerToken = resp.AccessToken
				hubApiBearerTokenExpTime = exp
				return hubApiBearerToken
			}
		}
	}

	resp, err := loginWithToken(config.ApiBaseUrl, config.ApiLoginToken)
	if err != nil {
		log.Fatalf("Unable to login with token: %v", err)
	}
	exp := time.Unix(resp.Exp, 0)
	if config.Debug {
		log.Printf("New API access token obtained; expiry at %v", exp)
	}
	if !tokenTimeValid(exp) {
		util.Warn("API token expiry too short %v", exp)
	}
	storeAccessToken(config.ApiBaseUrl, config.ApiLoginToken, resp)
	if err != nil {
		util.Warn("Unable to store API access token in cache: %v", err)
	}
	hubApiBearerToken = resp.AccessToken
	hubApiBearerTokenExpTime = exp
	return hubApiBearerToken
}

func verifyAccessToken(accessToken string) (int, *AuthPingResponse, error) {
	req, err := hubRequest("GET", authPingResource, accessToken, nil)
	if err != nil {
		return 0, nil, err
	}
	var jsResp AuthPingResponse
	code, _, err := do(hubApi(), req, &jsResp)
	if err != nil {
		return code, nil, err
	}
	return code, &jsResp, nil
}

func refreshAccessToken(tokens *SigninResponse) (*SigninResponse, error) {
	reqBody, err := json.Marshal(tokens)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling sign-in request: %v", err)
	}
	req, err := hubRequest("POST", refreshResource, "", bytes.NewReader(reqBody))
	if config.Trace {
		log.Printf(">>>\n%s", identJson(reqBody))
	}
	if err != nil {
		return nil, err
	}
	var jsResp SigninResponse
	code, _, err := do(hubApi(), req, &jsResp)
	if err != nil {
		return nil, fmt.Errorf("Refresh API token error: %v", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("Got %d HTTP from Auth Service refresh API token, expected 200 HTTP", code)
	}
	if jsResp.AccessToken == "" {
		return nil, fmt.Errorf("Empty or no `accessToken` in refresh API token response")
	}
	return &jsResp, nil
}
