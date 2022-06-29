// Copyright (c) 2022 EPAM Systems, Inc.
// 
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cel

import (
	"fmt"
	"log"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/interpreter"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/parameters"
	"github.com/agilestacks/hub/cmd/hub/util"
)

func Eval(expression, vars string, autoVars, yamlValue bool) {
	bindings, err := util.ParseKvList(vars)
	if err != nil {
		log.Fatalf("Unable to parse variable bindings: %v\n", err)
	}
	activation := &verboseActivation{bindings, autoVars}
	env, err := cel.NewEnv()
	if err != nil {
		log.Fatalf("Unable to init CEL runtime: %v\n", err)
	}
	var out interface{}
	if yamlValue {
		out = yamlExpression(expression, env, activation)
	} else {
		out = plainExpression(expression, env, activation)
	}
	fmt.Printf("%+v\n", out)
}

func plainExpression(expression string, env *cel.Env, activation interpreter.Activation) interface{} {
	ast, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		log.Fatalf("CEL parse error: %s\n", issues.Err())
	}
	program, err := env.Program(ast)
	if err != nil {
		log.Fatalf("CEL program construction error: %s\n", err)
	}
	out, _, err := program.Eval(activation)
	if err != nil {
		log.Fatalf("CEL evaluation error: %v\n", err)
	}
	return out
}

func yamlExpression(yamlExpression string, env *cel.Env, activation interpreter.Activation) string {
	expanded := parameters.CurlyReplacement.ReplaceAllStringFunc(yamlExpression,
		func(match string) string {
			expression, isCel := parameters.StripCurly(match)
			if !isCel {
				util.Warn("`%s` is not a CEL substitution", match)
			}
			result := plainExpression(expression, env, activation)
			return fmt.Sprintf("%+v", result)
		})
	return expanded
}

type verboseActivation struct {
	bindings map[string]string
	autoVars bool
}

func (a *verboseActivation) ResolveName(name string) (interface{}, bool) {
	value, exist := a.bindings[name]
	if !exist && a.autoVars {
		value = fmt.Sprintf("<%s>", name)
		exist = true
	}
	if config.Debug {
		print := "(undefined)"
		if exist {
			print = fmt.Sprintf("`%s`", value)
		}
		fmt.Printf("CEL resolving: %s => %s\n", name, print)

	}
	if !exist {
		return nil, false
	}
	return value, true
}

func (*verboseActivation) Parent() interpreter.Activation {
	return nil
}
