package lifecycle

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/epam/hubctl/cmd/hub/util"
	"gopkg.in/yaml.v2"
)

func isSecure(url *url.URL) bool {
	return url.Scheme == "https"
}

func toURL(iface interface{}) (*url.URL, error) {
	switch v := iface.(type) {
	case string:
		return parseURL(v)
	case *url.URL:
		return v, nil
	default:
		return nil, fmt.Errorf("invalid type %T", iface)
	}
}

func parseURL(urlStr string) (*url.URL, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	if u.Port() == "" {
		if isSecure(u) {
			u.Host = fmt.Sprintf("%s:443", u.Host)
		} else if u.Scheme == "http" {
			u.Host = fmt.Sprintf("%s:80", u.Host)
		}
	}
	return u, nil
}

func tplFunctionBase64(substitution interface{}) (interface{}, error) {
	return base64.StdEncoding.EncodeToString([]byte(util.String(substitution))), nil
}

func tplFunctionUnbase64(substitution interface{}) (interface{}, error) {
	decoded, err := base64.StdEncoding.DecodeString(util.String(substitution))
	if err != nil {
		return nil, err
	}
	return string(decoded), nil
}

func tplFunctionJson(substitution interface{}) (interface{}, error) {
	jsonBytes, err := json.Marshal(substitution)
	if err != nil {
		return nil, err
	}
	return string(jsonBytes), nil
}

func tplFunctionYaml(substitution interface{}) (interface{}, error) {
	yamlBytes, err := yaml.Marshal(substitution)
	if err != nil {
		return nil, err
	}
	return string(yamlBytes), nil
}

func tplFunctionFirst(substitution interface{}) (interface{}, error) {
	str := util.String(substitution)
	if strings.Contains(str, " ") {
		return strings.Split(str, " ")[0], nil
	}
	return str, nil
}

func tplFunctionParseUrl(substitution interface{}) (interface{}, error) {
	str := util.String(substitution)
	url, err := parseURL(str)
	if err != nil {
		return nil, err
	}
	return url, nil
}

func tplFunctionIsUrlSecure(substitution interface{}) (interface{}, error) {
	url, err := toURL(substitution)
	if err != nil {
		return nil, err
	}
	return isSecure(url), nil
}

func tplFunctionIsUrlInsecure(substitution interface{}) (interface{}, error) {
	url, err := toURL(substitution)
	if err != nil {
		return nil, err
	}
	return !isSecure(url), nil
}
func tplFunctionGetUrlHostname(substitution interface{}) (interface{}, error) {
	url, err := toURL(substitution)
	if err != nil {
		return nil, err
	}
	return url.Hostname(), nil
}
func tplFunctionGetUrlPort(substitution interface{}) (interface{}, error) {
	url, err := toURL(substitution)
	if err != nil {
		return nil, err
	}
	return url.Port(), nil
}
func tplFunctionGetUrlScheme(substitution interface{}) (interface{}, error) {
	url, err := toURL(substitution)
	if err != nil {
		return nil, err
	}
	return url.Scheme, nil
}

var tplFunctionsMap = map[string]func(interface{}) (interface{}, error){
	"base64":   tplFunctionBase64,
	"unbase64": tplFunctionUnbase64,
	"json":     tplFunctionJson,
	"yaml":     tplFunctionYaml,
	"first":    tplFunctionFirst,
	"parseURL": tplFunctionParseUrl,
	"isSecure": tplFunctionIsUrlSecure,
	"insecure": tplFunctionIsUrlInsecure,
	"hostname": tplFunctionGetUrlHostname,
	"port":     tplFunctionGetUrlPort,
	"scheme":   tplFunctionGetUrlScheme,
}

var supportedTplFunctions = util.Keys(tplFunctionsMap)
