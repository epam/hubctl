package api

import (
	"fmt"
	"net/url"
)

const userResource = "hub/api/v1/user"

func userDeploymentKey(subject string) (string, error) {
	if subject != "" {
		subject = "?subject=" + url.QueryEscape(subject)
	}
	path := fmt.Sprintf("%s/deployment-key%s", userResource, subject)
	var jsResp DeploymentKey
	code, err := get(hubApi, path, &jsResp)
	if err != nil {
		return "", fmt.Errorf("Error querying SuperHub User Deployment Key: %v", err)
	}
	if code != 200 {
		return "", fmt.Errorf("Got %d HTTP querying SuperHub User Deployment Key, expected 200 HTTP", code)
	}
	key := jsResp.DeploymentKey
	if key == "" {
		return "", fmt.Errorf("Got empty User Deployment Key")
	}
	return key, nil
}
