// Copyright (c) 2023 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	gotemplate "text/template"

	"github.com/Masterminds/sprig"
	"github.com/alexkappa/mustache"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/manifest"
	"github.com/epam/hubctl/cmd/hub/parameters"
	"github.com/epam/hubctl/cmd/hub/util"
)

const (
	curlyKind        = "curly"
	mustacheKind     = "mustache"
	trueMustacheKind = "_mustache"
	goKind           = "go"
)

var (
	templateSuffices = []string{".template", ".gotemplate"}
	kinds            = []string{curlyKind, mustacheKind, trueMustacheKind, goKind}
)

type TemplateRef struct {
	Filename string
	Kind     string
}

type OpenErr struct {
	Filename string
	Error    error
}

func processTemplates(component *manifest.ComponentRef, templateSetup *manifest.TemplateSetup,
	params parameters.LockedParameters, outputs parameters.CapturedOutputs,
	dir string) []error {

	componentName := manifest.ComponentQualifiedNameFromRef(component)
	kv := parameters.ParametersKV(params)
	templateSetup, err := expandParametersInTemplateSetup(templateSetup, kv)
	if err != nil {
		return []error{err}
	}
	err = checkTemplateSetupKind(templateSetup)
	if err != nil {
		return []error{err}
	}
	templates := scanTemplates(componentName, dir, templateSetup)

	if config.Verbose {
		if len(templates) > 0 {
			log.Print("Component templates:")
			printTemplates(templates)
		} else if len(templateSetup.Files) > 0 || len(templateSetup.Directories) > 0 || len(templateSetup.Extra) > 0 {
			log.Printf("No templates for component `%s`", componentName)
		}
	}

	if len(templates) == 0 {
		return nil
	}

	filenames := make([]string, 0, len(templates))
	hasMustache := false
	hasGo := false
	for _, template := range templates {
		filenames = append(filenames, template.Filename)
		if !hasMustache && template.Kind == trueMustacheKind {
			hasMustache = true
		}
		if !hasGo && template.Kind == goKind {
			hasGo = true
		}
	}
	cannot := checkStat(filenames)
	if len(cannot) > 0 {
		diag := make([]string, 0, len(cannot))
		for _, e := range cannot {
			diag = append(diag, fmt.Sprintf("\t`%s`: %v", e.Filename, e.Error))
		}
		return []error{fmt.Errorf("Unable to open `%s` component template input(s):\n%s", componentName, strings.Join(diag, "\n"))}
	}

	// during lifecycle operation `outputs` is nil - only parameters are available in templates
	// outputs are for `hub render`
	if outputs != nil {
		var depends []string
		if hasMustache || hasGo {
			depends = component.Depends
		}
		kv = parameters.ParametersAndOutputsKVWithDepends(params, outputs, depends)
	}
	if config.Trace {
		log.Printf("Template binding:\n%v", kv)
	}
	var mustacheKV map[string]interface{}
	if hasMustache {
		mustacheKV = mustacheCompatibleBindings(kv)
		if config.Trace {
			log.Printf("Mustache template binding:\n%v", mustacheKV)
		}
	}
	var goKV map[string]interface{}
	if hasGo {
		goKV = goTemplateBindings(kv)
		if config.Trace {
			log.Printf("Go template binding:\n%v", goKV)
		}
	}

	processor := func(content, filename, kind string) (string, []error) {
		var outContent string
		var err error
		var errs []error
		switch kind {
		case "", curlyKind:
			outContent, errs = processReplacement(content, filename, componentName, component.Depends, kv,
				curlyReplacement, stripCurly)
		case mustacheKind:
			outContent, errs = processReplacement(content, filename, componentName, component.Depends, kv,
				mustacheReplacement, stripMustache)
		case trueMustacheKind:
			outContent, err = processMustache(content, filename, componentName, mustacheKV)
		case goKind:
			outContent, err = processGo(content, filename, componentName, goKV)
		}
		if err != nil {
			errs = append(errs, err)
		}
		return outContent, errs
	}

	errs := make([]error, 0)
	for _, template := range templates {
		errs = append(errs, processTemplate(template.Filename, template.Kind, componentName, processor)...)
	}
	return errs
}

func maybeExpandParametersInTemplateGlob(glob string, kv map[string]interface{}, section string, index int) (string, error) {
	if !parameters.RequireExpansion(glob) {
		return glob, nil
	}
	value, errs := expandParametersInTemplateGlob(fmt.Sprintf("%s.%d", section, index), glob, kv)
	if len(errs) > 0 {
		return "", fmt.Errorf("Failed to expand template globs:\n\t%s", util.Errors("\n\t", errs...))
	}
	return value, nil
}

func expandParametersInTemplateGlob(what string, value string, kv map[string]interface{}) (string, []error) {
	piggy := manifest.Parameter{Name: fmt.Sprintf("templates.%s", what), Value: value}
	errs := parameters.ExpandParameter(&piggy, []string{}, kv)
	return util.String(piggy.Value), errs
}

func expandParametersInTemplateSetup(templateSetup *manifest.TemplateSetup,
	kv map[string]interface{}) (*manifest.TemplateSetup, error) {

	setup := manifest.TemplateSetup{
		Kind:        templateSetup.Kind,
		Files:       make([]string, 0, len(templateSetup.Files)),
		Directories: make([]string, 0, len(templateSetup.Directories)),
		Extra:       make([]manifest.TemplateTarget, 0, len(templateSetup.Extra)),
	}

	for i, glob := range templateSetup.Files {
		expanded, err := maybeExpandParametersInTemplateGlob(glob, kv, "files", i)
		if err != nil {
			return nil, err
		}
		setup.Files = append(setup.Files, expanded)
	}
	for i, glob := range templateSetup.Directories {
		expanded, err := maybeExpandParametersInTemplateGlob(glob, kv, "directories", i)
		if err != nil {
			return nil, err
		}
		setup.Directories = append(setup.Directories, expanded)
	}
	for j, templateExtra := range templateSetup.Extra {
		extra := manifest.TemplateTarget{
			Kind:        templateExtra.Kind,
			Files:       make([]string, 0, len(templateExtra.Files)),
			Directories: make([]string, 0, len(templateExtra.Directories)),
		}

		prefix := fmt.Sprintf("extra.%d", j)

		prefix2 := fmt.Sprintf("%s.files", prefix)
		for i, glob := range templateExtra.Files {
			expanded, err := maybeExpandParametersInTemplateGlob(glob, kv, prefix2, i)
			if err != nil {
				return nil, err
			}
			extra.Files = append(extra.Files, expanded)
		}
		prefix2 = fmt.Sprintf("%s.directories", prefix)
		for i, glob := range templateExtra.Directories {
			expanded, err := maybeExpandParametersInTemplateGlob(glob, kv, prefix2, i)
			if err != nil {
				return nil, err
			}
			extra.Directories = append(extra.Directories, expanded)
		}

		setup.Extra = append(setup.Extra, extra)
	}

	return &setup, nil
}

func checkTemplateSetupKind(templateSetup *manifest.TemplateSetup) error {
	var err error
	templateSetup.Kind, err = checkKind(templateSetup.Kind)
	if err != nil {
		return err
	}
	for i, extra := range templateSetup.Extra {
		templateSetup.Extra[i].Kind, err = checkKind(extra.Kind)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkKind(kind string) (string, error) {
	if kind == "" {
		return curlyKind, nil
	}
	if util.Contains(kinds, kind) {
		return kind, nil
	}
	return "", fmt.Errorf("Template kind `%s` not recognized; supported %v", kind, kinds)
}

func scanTemplates(componentName string, baseDir string, templateSetup *manifest.TemplateSetup) []TemplateRef {
	templates := make([]TemplateRef, 0, 10)

	templates = appendPlainFiles(templates, baseDir, templateSetup.Files, templateSetup.Kind)
	templates = scanDirectories(componentName, templates, baseDir, templateSetup.Directories, templateSetup.Files, templateSetup.Kind)

	for _, extra := range templateSetup.Extra {
		templates = appendPlainFiles(templates, baseDir, extra.Files, extra.Kind)
		templates = scanDirectories(componentName, templates, baseDir, extra.Directories, extra.Files, extra.Kind)
	}
	return templates
}

func appendPlainFiles(acc []TemplateRef, baseDir string, files []string, kind string) []TemplateRef {
	for _, file := range files {
		if !isGlob(file) {
			filePath := path.Join(baseDir, file)
			hasTemplateSuffix := false
			for _, templateSuffix := range templateSuffices {
				hasTemplateSuffix = strings.HasSuffix(file, templateSuffix)
				if hasTemplateSuffix {
					break
				}

			}
			if !hasTemplateSuffix {
				for _, templateSuffix := range templateSuffices {
					info, err := os.Stat(filePath + templateSuffix)
					if err == nil && !info.IsDir() {
						filePath = filePath + templateSuffix
						break
					}
				}
			}
			acc = append(acc, TemplateRef{Filename: filePath, Kind: kind})
		}
	}
	return acc
}

func scanDirectories(componentName string, acc []TemplateRef, baseDir string, directories []string, files []string, kind string) []TemplateRef {
	if len(files) == 0 && len(directories) > 0 {
		files = []string{"*"}
	}
	if len(directories) == 0 {
		directories = []string{""}
	}
	for _, dir := range directories {
		for _, file := range files {
			if isGlob(file) {
				glob := path.Join(baseDir, dir, file)
				if config.Debug {
					log.Printf("Scanning for `%s` templates `%s`", componentName, glob)
				}
				matches, err := filepath.Glob(glob)
				if err != nil {
					util.Warn("Unable to expand `%s` component template glob `%s`: %v", componentName, glob, err)
				}
				if matches != nil {
					for _, file := range matches {
						acc = append(acc, TemplateRef{Filename: file, Kind: kind})
					}
				} else {
					util.Warn("No matches found for `%s` component template glob `%s`", componentName, glob)
				}
			}
		}
	}
	return acc
}

func isGlob(path string) bool {
	return strings.Contains(path, "*") || strings.Contains(path, "[")
}

func checkStat(templates []string) []OpenErr {
	cannot := make([]OpenErr, 0)
	for _, template := range templates {
		info, err := os.Stat(template)
		if err != nil {
			cannot = append(cannot, OpenErr{Filename: template, Error: err})
		} else if info.IsDir() {
			cannot = append(cannot, OpenErr{Filename: template, Error: errors.New("is a directory")})
		}
	}
	if len(cannot) == 0 {
		return nil
	}
	return cannot
}

func processTemplate(filename, kind, componentName string,
	processor func(string, string, string) (string, []error)) []error {

	tmpl, err := os.Open(filename)
	if err != nil {
		return []error{fmt.Errorf("Unable to open `%s` component template input `%s`: %v", componentName, filename, err)}
	}
	byteContent, err := ioutil.ReadAll(tmpl)
	if err != nil {
		return []error{fmt.Errorf("Unable to read `%s` component template content `%s`: %v", componentName, filename, err)}
	}
	statInfo, err := tmpl.Stat()
	if err != nil {
		util.Warn("Unable to stat `%s` component template input `%s`: %v",
			componentName, filename, err)
	}
	tmpl.Close()
	content := string(byteContent)

	outPath := filename
	for _, templateSuffix := range templateSuffices {
		if strings.HasSuffix(outPath, templateSuffix) {
			outPath = outPath[:len(outPath)-len(templateSuffix)]
			break
		}
	}
	out, err := os.Create(outPath)
	if err != nil {
		return []error{fmt.Errorf("Unable to open `%s` component template output `%s`: %v", componentName, outPath, err)}
	}
	defer out.Close()
	if statInfo != nil {
		err = out.Chmod(statInfo.Mode())
		if err != nil {
			util.Warn("Unable to chmod `%s` component template output `%s`: %v", componentName, filename, err)
		}
	}

	outContent, errs := processor(content, filename, kind)
	if len(outContent) > 0 {
		written, err := strings.NewReader(outContent).WriteTo(out)
		if err != nil || written != int64(len(outContent)) {
			errs = append(errs, fmt.Errorf("Error writting `%s` component template output `%s`: %v", componentName, outPath, err))
		}
	}
	return errs
}

var (
	curlyReplacement    = regexp.MustCompile(`\$\{[a-zA-Z0-9_\.\|:/-]+\}`)
	mustacheReplacement = regexp.MustCompile(`\{\{[a-zA-Z0-9_\.\|:/-]+\}\}`)

	templateSubstitutionSupportedEncodings = []string{"base64", "unbase64", "json", "yaml", "first", "parseURL", "isSecure", "insecure", "hostname", "port", "scheme"}
)

func stripCurly(match string) string {
	return match[2 : len(match)-1]
}

func stripMustache(match string) string {
	return match[2 : len(match)-2]
}

// split string by one of the separators and
//
//	returns tuple of head and tail as slice
func head(variable string, sep ...string) (string, []string) {
	for _, s := range sep {
		if strings.Contains(variable, s) {
			parts := strings.Split(variable, s)
			return parts[0], parts[1:]
		}
	}
	return variable, nil
}

func processReplacement(content, filename, componentName string, componentDepends []string,
	kv map[string]interface{}, replacement *regexp.Regexp, strip func(string) string) (string, []error) {

	errs := make([]error, 0)
	replaced := false

	outContent := replacement.ReplaceAllStringFunc(content,
		func(variable string) string {
			variable = strip(variable)
			variable, encodings := head(variable, "/", "|")
			substitution, exist := parameters.FindValue(variable, componentName, componentDepends, kv)
			if !exist {
				errs = append(errs, fmt.Errorf("Template `%s` refer to unknown substitution `%s`", filename, variable))
				return "(unknown)"
			}
			if parameters.RequireExpansion(substitution) {
				util.WarnOnce("Template `%s` substitution `%s` refer to a value `%s` that is not expanded",
					filename, variable, substitution)
			}
			if config.Trace {
				log.Printf("--- %s | %s => %v", variable, componentName, substitution)
			}
			replaced = true
			if len(encodings) > 0 {
				if unknown := util.OmitAll(encodings, templateSubstitutionSupportedEncodings); len(unknown) > 0 {
					errs = append(errs, fmt.Errorf("Unknown encoding(s) %v processing template `%s` substitution `%s`",
						unknown, filename, variable))
				}
				for _, encoding := range encodings {
					switch encoding {
					case "base64":
						substitution = base64.StdEncoding.EncodeToString([]byte(util.String(substitution)))
					case "unbase64":
						decoded, err := base64.StdEncoding.DecodeString(util.String(substitution))
						if err != nil {
							errs = append(errs, fmt.Errorf("Unable to decode base64 from %v while processing template `%s` substitution `%s`: %v",
								substitution, filename, variable, err))
						} else {
							substitution = string(decoded)
						}
					case "json":
						jsonBytes, err := json.Marshal(substitution)
						if err != nil {
							errs = append(errs, fmt.Errorf("Unable to marshal JSON from %v while processing template `%s` substitution `%s`: %v",
								substitution, filename, variable, err))
						} else {
							substitution = string(jsonBytes)
						}
					case "yaml":
						// TODO YAML fragment on a single line
						yamlBytes, err := yaml.Marshal(substitution)
						if err != nil {
							errs = append(errs, fmt.Errorf("Unable to marshal YAML from %v while processing template `%s` substitution `%s`: %v",
								substitution, filename, variable, err))
						} else {
							substitution = string(yamlBytes)
						}
					case "first":
						str := util.String(substitution)
						if strings.Contains(str, " ") {
							substitution = strings.Split(str, " ")[0]
						}
					case "parseURL":
						str := util.String(substitution)
						url, err := parseURL(str)
						if err != nil {
							errs = append(errs, fmt.Errorf("Unable to parse URL from %v while processing template `%s` substitution `%s`: %v",
								substitution, filename, variable, err))
						} else {
							substitution = url
						}
					case "isSecure":
						url, err := toURL(substitution)
						if err != nil {
							errs = append(errs, fmt.Errorf("Unable to parse URL from %v while processing template `%s` substitution `%s`: %v",
								substitution, filename, variable, err))
						} else {
							substitution = isSecure(url)
						}
					case "insecure":
						url, err := toURL(substitution)
						if err != nil {
							errs = append(errs, fmt.Errorf("Unable to parse URL from %v while processing template `%s` substitution `%s`: %v",
								substitution, filename, variable, err))
						} else {
							substitution = !isSecure(url)
						}
					case "hostname":
						url, err := toURL(substitution)
						if err != nil {
							errs = append(errs, fmt.Errorf("Unable to parse URL from %v while processing template `%s` substitution `%s`: %v",
								substitution, filename, variable, err))
						} else {
							substitution = url.Hostname()
						}
					case "port":
						url, err := toURL(substitution)
						if err != nil {
							errs = append(errs, fmt.Errorf("Unable to parse URL from %v while processing template `%s` substitution `%s`: %v",
								substitution, filename, variable, err))
						} else {
							substitution = url.Port()
						}
					case "scheme":
						url, err := toURL(substitution)
						if err != nil {
							errs = append(errs, fmt.Errorf("Unable to parse URL from %v while processing template `%s` substitution `%s`: %v",
								substitution, filename, variable, err))
						} else {
							substitution = url.Scheme
						}
					}
				}
			}
			return strings.TrimSpace(util.String(substitution))
		})

	if !replaced && len(errs) == 0 {
		util.Warn("No substitutions found in template `%s`", filename)
	}
	return outContent, errs
}

func mustacheCompatibleBindings(kv map[string]interface{}) map[string]interface{} {
	mkv := make(map[string]interface{})
	for k, v := range kv {
		mkv[strings.ReplaceAll(k, ".", "_")] = v
	}
	return mkv
}

func processMustache(content, filename, componentName string, kv map[string]interface{}) (string, error) {
	template := mustache.New(mustache.SilentMiss(false))
	err := template.ParseString(content)
	if err != nil {
		return "", fmt.Errorf("Unable to parse mustache template `%s`: %v", filename, err)
	}
	outContent, err := template.RenderString(kv)
	if err != nil {
		return outContent, fmt.Errorf("Unable to render mustache template `%s`: %v", filename, err)
	}
	return outContent, nil
}

func goTemplateBindings(kv map[string]interface{}) map[string]interface{} {
	gkv := make(map[string]interface{})
	for k, v := range kv {
		parts := strings.Split(k, ".")
		innerkv := gkv
		for i, part := range parts {
			part = strings.ReplaceAll(part, "-", "_")
			leaf := i == len(parts)-1
			if leaf {
				_, exist := innerkv[part]
				if exist {
					util.WarnOnce("Template nested values already installed under `%s`, cannot install leaf value `%[1]s`", k)
					break
				}
				if str, ok := v.(string); ok {
					innerkv[part] = strings.TrimSpace(str)
				} else {
					innerkv[part] = v
				}
			} else {
				ref, exist := innerkv[part]
				if exist {
					var ok bool
					innerkv, ok = ref.(map[string]interface{})
					if !ok {
						util.WarnOnce("Template leaf value already installed at `%s`, cannot install nested value `%s`",
							strings.Join(parts[0:i+1], "."), k)
						break
					}
				} else {
					newkv := make(map[string]interface{})
					innerkv[part] = newkv
					innerkv = newkv
				}
			}

		}
	}
	return gkv
}

func bcryptStr(str string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	if err != nil {
		return str, err
	}
	return string(bytes), nil
}

// Splits the string into a list of strings
//
//	First argument is a string to split
//	Second optional argument is a separator (default is space)
//
// Example:
//
//	split "a b c" => ["a", "b", "c"]
//	split "a-b-c", "-" => ["a", "b", "c"]
func split(args ...string) ([]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("split expects one or two arguments")
	}
	if len(args) == 1 {
		return strings.Fields(args[0]), nil
	}
	return strings.Split(args[0], args[1]), nil
}

// Removes empty string from the list of strings
// Accepts variable arguments arguments (easier tolerate template nature):
//
// Example:
//
//	compact "string1" (compatibility with parametersx)
//	compact "string1" "string2" "string3"
//	compact ["string1", "string2", "string3"]
func compact(args ...interface{}) ([]string, error) {
	var results []string
	for _, arg := range args {
		a := reflect.ValueOf(arg)
		if a.Kind() == reflect.Slice {
			if a.Len() == 0 {
				continue
			}
			ret := make([]interface{}, a.Len())
			for i := 0; i < a.Len(); i++ {
				ret[i] = a.Index(i).Interface()
			}
			res, _ := compact(ret...)
			results = append(results, res...)
			continue
		}
		if a.Kind() == reflect.String {
			trimmed := strings.TrimSpace(a.String())
			if trimmed == "" {
				continue
			}
			results = append(results, trimmed)
			continue
		}
		return nil, fmt.Errorf("Argument type %T not yet supported", arg)
	}
	return results, nil
}

// Joins the list of strings into a single string
// Last argument is a delimiter (default is space)
// Accepts variable arguments arguments (easier tolerate template nature)
//
// Example:
//
//	join "string1" "string2" "delimiter"
//	join ["string1", "string2"] "delimiter"
//	join ["string1", "string2"]
//	join "string1"
func join(args ...interface{}) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("join expects at least one argument")
	}
	var del string
	if len(args) > 1 {
		del = fmt.Sprintf("%v", args[len(args)-1])
		args = args[:len(args)-1]
	}
	if del == "" {
		del = " "
	}

	var result []string
	for _, arg := range args {
		a := reflect.ValueOf(arg)
		if a.Kind() == reflect.Slice {
			if a.Len() == 0 {
				continue
			}
			for i := 0; i < a.Len(); i++ {
				result = append(result, fmt.Sprintf("%v", a.Index(i).Interface()))
			}
			continue
		}
		if a.Kind() == reflect.String {
			result = append(result, a.String())
			continue
		}
		return "", fmt.Errorf("Argument type %T not yet supported", arg)
	}

	return strings.Join(result, del), nil
}

// Returns the first argument from list
//
// Example:
//
//	first ["string1" "string2" "string3"] => "string1"
func first(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("first expects at least one argument")
	}
	return args[0], nil
}

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
		return "", fmt.Errorf("hostname expects at least one argument")
	}
	arg0 := reflect.ValueOf(args[0])
	if arg0.Kind() != reflect.String {
		return "", fmt.Errorf("hostname expects string as first argument")
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
			return "", fmt.Errorf("Argument type %T not yet supported", args[1])
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

var hubGoTemplateFuncMap = map[string]interface{}{
	"bcrypt":          bcryptStr,
	"split":           split,
	"compact":         compact,
	"join":            join,
	"first":           first,
	"formatSubdomain": formatSubdomain,
	"unquote":         unquote,
	"uquote":          unquote,
}

func processGo(content, filename, componentName string, kv map[string]interface{}) (string, error) {
	tmpl, err := gotemplate.New(filepath.Base(filename)).Funcs(sprig.TxtFuncMap()).Funcs(hubGoTemplateFuncMap).Parse(content)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, kv)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}
