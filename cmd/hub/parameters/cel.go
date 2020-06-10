package parameters

import (
	"fmt"
	"log"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/interpreter"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/util"
)

var (
	CEL *cel.Env
)

func init() {
	env, err := cel.NewEnv()
	if err != nil {
		log.Fatalf("Unable to init CEL runtime: %v", err)
	}
	CEL = env
}

func CelEval(expr string, component string, depends []string, kv map[string]interface{}) (string, error) {
	ast, issues := CEL.Parse(expr)
	if issues != nil && issues.Err() != nil {
		return "(parse error)", fmt.Errorf("CEL parse error: %v", issues.Err())
	}
	program, err := CEL.Program(ast)
	if err != nil {
		return "(program error)", fmt.Errorf("CEL program construction error `%s`: %v", expr, err)
	}
	activation := newCelActivation(component, depends, kv)
	out, _, err := program.Eval(activation)
	if err != nil {
		return "(eval error)", fmt.Errorf("CEL evaluation error `%s`: %v", expr, err)
	}
	return fmt.Sprintf("%+v", out), nil
}

type celActivation struct {
	component string
	depends   []string
	kv        map[string]interface{}
}

func (a *celActivation) ResolveName(name string) (interface{}, bool) {
	value, exist := FindValue(name, a.component, a.depends, a.kv)
	if config.Trace {
		print := "(unknown)"
		if exist {
			print = fmt.Sprintf("`%s`", util.Wrap(util.String(value)))
		}
		log.Printf("CEL resolving: %s => %s", name, print)
	}
	if !exist {
		return nil, false
	}
	return value, true
}

func (*celActivation) Parent() interpreter.Activation {
	return nil
}

func newCelActivation(component string, depends []string, kv map[string]interface{}) interpreter.Activation {
	return &celActivation{component, depends, kv}
}
