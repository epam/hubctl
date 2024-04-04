package lifecycle

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Converts the string into kubernetes acceptable name
// which consist of kebab lower case with alphanumeric characters.
// '.' is not allowed
//
// Arguments:
//
//	First argument is a text to convert
//	Second optional argument is a size of the name (default is 63)
//	Third optional argument is a delimiter (default is '-')
func formatSubdomain(args ...interface{}) (string, error) {
	if len(args) == 0 {
		return "", errors.New("hostname expects at least one argument")
	}
	arg0 := reflect.ValueOf(args[0])
	if arg0.Kind() != reflect.String {
		return "", errors.New("hostname expects string as first argument")
	}
	text := strings.TrimSpace(arg0.String())
	if text == "" {
		return "", nil
	}

	size := 63
	if len(args) > 1 {
		arg1 := reflect.ValueOf(args[1])
		if arg1.Kind() == reflect.Int {
			size = int(reflect.ValueOf(args[1]).Int())
		} else if arg1.Kind() == reflect.String {
			size, _ = strconv.Atoi(arg1.String())
		} else {
			return "", fmt.Errorf("argument type %T not yet supported", args[1])
		}
	}

	var del = "-"
	if len(args) > 2 {
		del = fmt.Sprintf("%v", args[2])
	}

	var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")
	var matchNonAlphanumericEnd = regexp.MustCompile("[^a-zA-Z0-9]+$")
	var matchNonLetterStart = regexp.MustCompile("^[^a-zA-Z]+")
	var matchNonAnumericOrDash = regexp.MustCompile("[^a-zA-Z0-9-]+")
	var matchTwoOrMoreDashes = regexp.MustCompile("-{2,}")

	text = matchNonLetterStart.ReplaceAllString(text, "")
	text = matchAllCap.ReplaceAllString(text, "${1}-${2}")
	text = matchNonAnumericOrDash.ReplaceAllString(text, "-")
	text = matchTwoOrMoreDashes.ReplaceAllString(text, "-")
	text = strings.ToLower(text)
	if len(text) > size {
		text = text[:size]
	}
	text = matchNonAlphanumericEnd.ReplaceAllString(text, "")
	if del != "-" {
		text = strings.ReplaceAll(text, "-", del)
	}
	return text, nil
}

// Removes single or double or back quotes from the string
func unquote(str string) (string, error) {
	result, err := strconv.Unquote(str)
	if err != nil && err.Error() == "invalid syntax" {
		return str, err
	}
	return result, err
}

var hubGoTemplateFuncMap = map[string]interface{}{
	"formatSubdomain": formatSubdomain,
	"unquote":         unquote,
	"uquote":          unquote,
}
