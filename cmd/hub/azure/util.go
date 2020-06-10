package azure

import "strings"

func IsNotFound(err error) bool {
	str := err.Error()
	return strings.HasPrefix(str, "storage:") &&
		strings.Contains(str, "StatusCode=404")
}
