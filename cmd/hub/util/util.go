// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/mattn/go-isatty"

	"github.com/epam/hubctl/cmd/hub/config"
)

var (
	warnings        = make([]string, 0)
	warningsEmitted = make(map[string]struct{})
	HighlightColor  = maybeHighlight(aurora.BrightCyan)
	WarnColor       = maybeHighlight(aurora.BrightMagenta)
	logTerminal     *bool
)

func maybeHighlight(color func(interface{}) aurora.Value) func(string) string {
	return func(str string) string {
		if config.Tty && (IsLogTerminal() || config.TtyForced) {
			str = color(str).String()
		}
		return str
	}
}

func IsLogTerminal() bool {
	if logTerminal != nil {
		return *logTerminal
	}
	fd := os.Stderr.Fd()
	if config.LogDestination == "stdout" {
		fd = os.Stdout.Fd()
	}
	tty := isatty.IsTerminal(fd)
	logTerminal = &tty
	return tty
}

func Warn(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf(WarnColor("WARN: %s"), msg)
	if config.AggWarnings {
		warnings = append(warnings, msg)
	}
}

func Coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func WarnOnce(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if _, emitted := warningsEmitted[msg]; emitted {
		return
	}
	warningsEmitted[msg] = struct{}{}
	log.Printf(WarnColor("WARN: %s"), msg)
	if config.AggWarnings {
		warnings = append(warnings, msg)
	}
}

func PrintAllWarnings() {
	if !config.AggWarnings || len(warnings) == 0 {
		return
	}
	if config.Verbose {
		log.Print(WarnColor("All warnings combined:"))
	}
	uniq := make([]string, 0, len(warnings))
	seen := make(map[string]struct{})
	for _, msg := range warnings {
		if _, emitted := seen[msg]; emitted {
			continue
		}
		seen[msg] = struct{}{}
		uniq = append(uniq, msg)
	}
	io.WriteString(os.Stderr, strings.Join(uniq, "\n"))
	io.WriteString(os.Stderr, "\n")
}

func Errors(sep string, maybeErrors ...error) string {
	if sep == "" {
		sep = ", "
	}
	errs := make([]string, 0, len(maybeErrors))
	for _, err := range maybeErrors {
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return "(no errors)"
	}
	return strings.Join(UniqInOrder(errs), sep)
}

func Errors2(maybeErrors ...error) string {
	return Errors("", maybeErrors...)
}

// TODO honour `secure`
func askOnTerminal(prompt string, secure bool) string {
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		if config.Verbose {
			log.Printf("Stdin is not a terminal, not asking for `%s`", prompt)
		}
		return ""
	}
	var input string
	fmt.Printf("%s: ", prompt)
	read, err := fmt.Scanln(&input)
	if read > 0 {
		if err != nil {
			log.Fatalf("Error reading input for `%s`: %v (read %d bytes)", prompt, err, read)
		} else {
			return input
		}
	}
	return ""
}

func maybeAskInput(input string, prompt string, secure bool) string {
	if input != "" {
		return input
	}
	return askOnTerminal(prompt, secure)
}

func AskInput(input string, prompt string) string {
	return maybeAskInput(input, prompt, false)
}

func AskPassword(input string, prompt string) string {
	return maybeAskInput(input, prompt, true)
}

func CopyMap2(m map[string][]string) map[string][]string {
	res := make(map[string][]string)
	for k, v := range m {
		v2 := make([]string, len(v))
		copy(v2, v)
		res[k] = v2
	}
	return res
}

func AppendMapList(m map[string][]string, key, value string) {
	if list, exist := m[key]; exist {
		m[key] = append(list, value)
	} else {
		m[key] = []string{value}
	}
}

func ConcatMaps(m1, m2 map[string]string) map[string]string {
	if len(m1) == 0 && len(m2) > 0 {
		return m2
	}
	if len(m2) == 0 && len(m1) > 0 {
		return m1
	}
	res := make(map[string]string)
	for k, v := range m1 {
		res[k] = v
	}
	for k, v := range m2 {
		res[k] = v
	}
	return res
}

func Reverse(source []string) []string {
	l := len(source)
	if l < 2 {
		return source
	}
	reversed := make([]string, l)
	for i, str := range source {
		reversed[l-i-1] = str
	}
	return reversed
}

func Uniq(source []string) []string {
	sorted := make([]string, len(source))
	copy(sorted, source)
	sort.Strings(sorted)
	dest := make([]string, 0, len(source))
	prev := ""
	for _, str := range sorted {
		if str != prev {
			dest = append(dest, str)
			prev = str
		}
	}
	return dest
}

func UniqInOrder(source []string) []string {
	result := make([]string, 0, len(source))
	seen := make(map[string]struct{})
	for _, str := range source {
		if _, exist := seen[str]; !exist {
			seen[str] = struct{}{}
			result = append(result, str)
		}
	}
	return result
}

func Contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func ContainsSubstring(list []string, substr string) bool {
	for _, v := range list {
		if strings.Contains(v, substr) {
			return true
		}
	}
	return false
}

func ContainsPrefix(prefixes []string, value string) bool {
	for _, prefix := range prefixes {
		if prefix == value ||
			(strings.HasSuffix(prefix, "*") && strings.HasPrefix(value, prefix[:len(prefix)-1])) ||
			strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func ContainsAll(list []string, values []string) bool {
	for _, v := range values {
		if !Contains(list, v) {
			return false
		}
	}
	return true
}

func ContainsAny(list []string, values []string) bool {
	for _, v := range values {
		if Contains(list, v) {
			return true
		}
	}
	return false
}

func Union(list, list2 []string) []string {
	res := make([]string, 0)
	for _, v := range list2 {
		if Contains(list, v) {
			res = append(res, v)
		}
	}
	return res
}

func ContainsAnySubstring(list []string, substrs []string) bool {
	for _, substr := range substrs {
		if ContainsSubstring(list, substr) {
			return true
		}
	}
	return false
}

func Equal(list []string, list2 []string) bool {
	return reflect.DeepEqual(list, list2)
}

func Omit(list []string, value string) []string {
	if !Contains(list, value) {
		return list
	}
	filtered := make([]string, 0, len(list))
	for _, v := range list {
		if v != value {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func OmitAll(list []string, values []string) []string {
	filtered := make([]string, 0, len(list))
	for _, v := range list {
		if !Contains(values, v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func Filter(list []string, patterns []string) []string {
	filtered := make([]string, 0, len(list))
	for _, v := range list {
		if ContainsPrefix(patterns, v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func FilterNot(list []string, patterns []string) []string {
	filtered := make([]string, 0, len(list))
	for _, v := range list {
		if !ContainsPrefix(patterns, v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func Index(list []string, search string) int {
	index := -1
	if search != "" {
		for i, value := range list {
			if search == value {
				index = i
				break
			}
		}
	}
	return index
}

func SortedKeys(m map[string]string) []string {
	if len(m) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for name := range m {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}

func SortedKeys2(m map[string][]string) []string {
	if len(m) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for name := range m {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}

func MergeUnique(lists ...[]string) []string {
	ll := 0
	for _, list := range lists {
		ll += len(list)
	}
	res := make([]string, 0, ll)
	for _, list := range lists {
		for _, value := range list {
			if !Contains(res, value) {
				res = append(res, value)
			}
		}
	}
	return res
}

func Value(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func Wrap(str string) string {
	str = strings.Replace(str, "\n", "\\n", -1)
	if len(str) > 102 {
		str = str[:100] + "..."
	}
	return str
}

func Empty(value interface{}) bool {
	if value == nil {
		return true
	}
	if str, ok := value.(string); ok {
		return str == ""
	}
	return false
}

func String(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}

func MaybeJson(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	valueBytes, err := json.Marshal(value)
	if err != nil {
		if config.Trace {
			return fmt.Sprintf("(%v)", err)
		}
		return "(error)"
	}
	return string(valueBytes)
}

func TrimColor(str string) string {
	colors := "\x1B["
	for strings.Contains(str, colors) {
		start := strings.Index(str, colors)
		end := strings.Index(str[start+1:], "m")
		if end != -1 {
			end += start + 1
		} else {
			end = start + len(colors) - 1
		}
		if end < len(str) {
			str = str[:start] + str[end+1:]
		} else {
			str = str[:start]
		}
	}
	return str
}

func Trim(str string) string {
	cutset := " \r"
	return strings.Trim(str, cutset)
}

func NoSuchFile(err error) bool {
	if err == nil {
		return false
	}
	str := err.Error()
	return err == os.ErrNotExist ||
		str == "file does not exist" ||
		strings.Contains(str, "no such file or directory")
}

func ContextCanceled(err error) bool {
	if err == nil {
		return false
	}
	str := err.Error()
	return err == context.Canceled ||
		str == "context canceled" ||
		strings.Contains(str, ": context canceled") ||
		strings.Contains(str, ": operation was canceled")
}

func Plural(size int, noun ...string) string {
	l := len(noun)
	if l == 0 {
		return ""
	}
	if size > 1 {
		if l > 1 {
			return noun[1]
		}
		return fmt.Sprintf("%ss", noun[0])
	}
	return noun[0]
}

func SplitPaths(paths string) []string {
	if paths == "" {
		return []string{}
	}
	return strings.Split(paths, ",")
}

// strip .hub/ .terraform/ etc.
func StripDotDirs(path string) string {
	for {
		dir := filepath.Base(path)
		if strings.HasPrefix(dir, ".") && dir != "." && dir != ".." {
			path = filepath.Dir(path)
			continue
		}
		return path
	}
}

func MustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Unable to convert `%s` to absolute pathname: %v", path, err)
	}
	return abs
}

func Basedir(paths []string) string {
	for _, path := range paths {
		if !strings.Contains(path, "://") {
			if _, err := os.Stat(path); err == nil {
				return StripDotDirs(filepath.Dir(path))
			}
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Unable to determine current working directory: %v", err)
	}
	return cwd
}

func PlainName(name string) string {
	if name == "" {
		return ""
	}
	i := strings.Index(name, ":")
	if i > 0 {
		return name[0:i]
	}
	return name
}

func SplitQName(qName string) (string, string) {
	name := qName
	component := ""
	parts := strings.SplitN(qName, "|", 2)
	if len(parts) > 1 {
		name = parts[0]
		component = parts[1]
	}
	return name, component
}

func initSecretSuffixes() []string {
	seed := []string{"password", "secret", "key", "cert", "token", "cookie", "kubeconfig"}
	suffixes := make([]string, 0, len(seed)*4)
	for _, suf := range seed {
		suffixes = append(suffixes, "."+suf)
		suffixes = append(suffixes, "_"+suf)
		suffixes = append(suffixes, strings.Title(suf))
		suffixes = append(suffixes, strings.ToUpper(suf))
	}
	return suffixes
}

var secretSuffixes = initSecretSuffixes()
var notASecretWhitelist = []string{"cloud.sshKey"}

func LooksLikeSecret(name string) bool {
	i := strings.Index(name, "|")
	if i > 0 { // a Qname
		name = name[0:i]
	}
	if Contains(notASecretWhitelist, name) {
		return false
	}
	for _, suf := range secretSuffixes {
		if strings.HasSuffix(name, suf) {
			return true
		}
	}
	return false
}

func MaybeMaskedValue(trace bool, name, value string) string {
	if !trace && LooksLikeSecret(name) {
		return "(masked)"
	}
	return value
}

func ParseKvList(list string) (map[string]string, error) {
	parsed := make(map[string]string)
	if list == "" {
		return parsed, nil
	}
	vars := strings.Split(list, ",")
	for _, v := range vars {
		kv := strings.SplitN(v, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("`%s` cannot be split into key/value pair", v)
		}
		parsed[kv[0]] = kv[1]
	}
	if config.Debug {
		log.Print("Parsed:")
		for k, v := range parsed {
			log.Printf("\t%s => %s", k, v)
		}
	}
	return parsed, nil
}

func IsUint(str string) bool {
	if str == "" {
		return false
	}
	_, err := strconv.ParseUint(str, 10, 32)
	return err == nil
}

func MaybeEnv(vars []string) bool {
	for _, v := range vars {
		if _, exist := os.LookupEnv(v); exist {
			return true
		}
	}
	return false
}
