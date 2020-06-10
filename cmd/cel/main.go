package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/interpreter"
)

var (
	verbose  bool
	autoVars bool
)

func main() {
	flag.BoolVar(&verbose, "v", false, "Print CEL internals if set")
	flag.BoolVar(&autoVars, "a", true, "Auto-resolve variable into <variable name> if not found in binding")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr,
			`Usage:
  cel [-v] [-a] <expression> [bind.some.name=value1,...]

Flags:
`)
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 2 && flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	expression := flag.Arg(0)
	binding := make(map[string]interface{})
	if flag.NArg() == 2 {
		var err error
		binding, err = parseKvList(flag.Arg(1))
		if err != nil {
			fmt.Printf("Unable to parse variable binding: %v\n", err)
			os.Exit(1)
		}
	}

	env, err := cel.NewEnv()
	if err != nil {
		fmt.Printf("Unable to init CEL runtime: %v\n", err)
		os.Exit(1)
	}
	ast, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		fmt.Printf("CEL parse error: %s\n", issues.Err())
		os.Exit(1)
	}
	program, err := env.Program(ast)
	if err != nil {
		fmt.Printf("CEL program construction error: %s\n", err)
		os.Exit(1)
	}
	out, _, err := program.Eval(&verboseActivation{binding})
	if err != nil {
		fmt.Printf("CEL evaluation error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%+v\n", out)
}

func parseKvList(list string) (map[string]interface{}, error) {
	parsed := make(map[string]interface{})
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
	if verbose {
		fmt.Print("Parsed vars binding:\n")
		for k, v := range parsed {
			fmt.Printf("\t%s => %s\n", k, v)
		}
	}
	return parsed, nil
}

type verboseActivation struct {
	bindings map[string]interface{}
}

func (a *verboseActivation) ResolveName(name string) (interface{}, bool) {
	value, exist := a.bindings[name]
	if !exist && autoVars {
		value = fmt.Sprintf("<%s>", name)
		exist = true
	}
	if verbose {
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
