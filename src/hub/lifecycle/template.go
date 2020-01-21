package lifecycle

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	gotemplate "text/template"

	"github.com/Masterminds/sprig"
	"github.com/alexkappa/mustache"
	"gopkg.in/yaml.v2"

	"hub/config"
	"hub/manifest"
	"hub/parameters"
	"hub/util"
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
		kv = parameters.ParametersAndOutputsKV(params, outputs, depends)
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
	curlyReplacement    = regexp.MustCompile("\\$\\{[a-zA-Z0-9_\\.\\|:/-]+\\}")
	mustacheReplacement = regexp.MustCompile("\\{\\{[a-zA-Z0-9_\\.\\|:/-]+\\}\\}")

	templateSubstitutionSupportedEncodings = []string{"base64", "unbase64", "json", "yaml"}
)

func stripCurly(match string) string {
	return match[2 : len(match)-1]
}

func stripMustache(match string) string {
	return match[2 : len(match)-2]
}

func valueEncodings(variable string) (string, []string) {
	if strings.Contains(variable, "/") {
		parts := strings.Split(variable, "/")
		return parts[0], parts[1:]
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
			variable, encodings := valueEncodings(variable)
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
				if util.Contains(encodings, "json") {
					jsonBytes, err := json.Marshal(substitution)
					if err != nil {
						errs = append(errs, fmt.Errorf("Unable to marshal JSON from %v while processing template `%s` substitution `%s`: %v",
							substitution, filename, variable, err))
					} else {
						substitution = string(jsonBytes)
					}
				} else if util.Contains(encodings, "yaml") {
					// TODO YAML fragment on a single line
					yamlBytes, err := yaml.Marshal(substitution)
					if err != nil {
						errs = append(errs, fmt.Errorf("Unable to marshal YAML from %v while processing template `%s` substitution `%s`: %v",
							substitution, filename, variable, err))
					} else {
						substitution = string(yamlBytes)
					}
				}
				strSubstitution := util.String(substitution)
				if util.Contains(encodings, "base64") {
					strSubstitution = base64.StdEncoding.EncodeToString([]byte(strSubstitution))
				} else if util.Contains(encodings, "unbase64") {
					bytes, err := base64.StdEncoding.DecodeString(strSubstitution)
					if err != nil {
						util.Warn("Unable to decode `%s` base64 value `%s`: %v", variable, util.Trim(strSubstitution), err)
					} else {
						strSubstitution = string(bytes)
					}
				}
				return strSubstitution
			}
			return strings.TrimSpace(util.String(substitution))
		})

	if !replaced {
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
			leaf := i == len(parts)-1
			if leaf {
				innerkv[part] = v
			} else {
				ref, exist := innerkv[part]
				if exist {
					innerkv = ref.(map[string]interface{})
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

func processGo(content, filename, componentName string, kv map[string]interface{}) (string, error) {
	tmpl, err := gotemplate.New(filepath.Base(filename)).Funcs(sprig.TxtFuncMap()).Parse(content)
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
