package api

import (
	"fmt"
)

const userResource = "hub/api/v1/user"

func userDeploymentKey() (string, error) {
	path := fmt.Sprintf("%s/deployment-key", userResource)
	var jsResp DeploymentKey
	code, err := get(hubApi, path, &jsResp)
	if err != nil {
		return "", fmt.Errorf("Error querying Hub Service User Deployment Key: %v", err)
	}
	if code != 200 {
		return "", fmt.Errorf("Got %d HTTP querying Hub Service User Deployment Key, expected 200 HTTP", code)
	}
	key := jsResp.DeploymentKey
	if key == "" {
		return "", fmt.Errorf("Got empty User Deployment Key")
	}
	return key, nil
}
