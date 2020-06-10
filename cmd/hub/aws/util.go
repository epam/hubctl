package aws

import (
	"strings"
)

func IsNotFound(err error) bool {
	str := err.Error()
	return strings.HasPrefix(str, "NotFound:") &&
		strings.Contains(str, "status code: 404,")
}

func IsSlowDown(err error) bool {
	str := err.Error()
	return strings.Contains(str, "SlowDown: Please reduce your request rate.")
}
