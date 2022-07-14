// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/agilestacks/hub/cmd/hub/util"
)

var (
	wellKnownOsEnv = []string{
		"AWS_PROFILE", "AWS_DEFAULT_REGION",
		"AZURE_*", "ARM_*",
		"GOOGLE_APPLICATION_CREDENTIALS",
		"KUBECONFIG",
		"HUB", "HUB_*",
		"GOBIN", "GOPATH", "GOROOT", "GOOS", "GOARCH",
		"DOCKER_*", "NVM_*", "NODE_*", "SDKMAN_*", "GRADLE_HOME", "VIRTUALENVWRAPPER_*", "VIRTUAL_ENV",
		"HOME", "LANG", "LC_*", "LOGNAME", "PATH", "LD_LIBRARY_PATH", "SHELL", "SSH_AUTH_SOCK", "TERM", "TMPDIR", "USER",
	}
	noTfOsEnv = []string{
		"TF_VAR_*",
	}
)

func initOsEnv(mode string) ([]string, error) {
	osEnv := os.Environ()

	switch mode {
	case "everything":
		return osEnv, nil
	case "no-tfvars":
		return filterEnv(osEnv, noTfOsEnv, true), nil
	case "strict":
		return filterEnv(osEnv, wellKnownOsEnv, false), nil
	}

	return nil, fmt.Errorf("`%s` is not recognized as a valid mode", mode)
}

func filterEnv(env, patterns []string, omit bool) []string {
	filtered := make([]string, 0, len(env))
	for _, v := range env {
		kv := strings.SplitN(v, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if omit != util.ContainsPrefix(patterns, kv[0]) {
			filtered = append(filtered, v)
		}
	}
	return filtered

}

func mergeOsEnviron(toMerge ...[]string) []string {
	vars := make(map[string]string)
	for _, varsArray := range toMerge {
		for _, envVar := range varsArray {
			kv := strings.SplitN(envVar, "=", 2)
			if len(kv) != 2 {
				continue
			}
			vars[kv[0]] = kv[1]
		}
	}

	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	res := make([]string, 0, len(vars))
	for _, key := range keys {
		res = append(res, fmt.Sprintf("%s=%s", key, vars[key]))
	}
	return res
}
