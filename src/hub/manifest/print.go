package manifest

import (
	"fmt"
	"log"

	"hub/config"
	"hub/util"
)

func printParameters(parameters []Parameter) {
	for _, p := range parameters {
		def := ""
		if p.Default != "" {
			def = fmt.Sprintf(" [%s]", util.Wrap(p.Default))
		}
		fromEnv := ""
		if p.FromEnv != "" {
			fromEnv = fmt.Sprintf(" (from:%s)", p.FromEnv)
		}
		env := ""
		if p.Env != "" {
			env = fmt.Sprintf(" (env:%s)", p.Env)
		}
		value := p.Value
		if value == "" && p.Kind == "user" {
			value = "*ask*"
		} else {
			if !config.Trace && util.LooksLikeSecret(p.Name) && len(value) > 0 {
				value = "(masked)"
			} else {
				value = fmt.Sprintf("`%s`", util.Wrap(value))
			}
		}
		fqName := p.Name
		if p.Component != "" {
			fqName = fmt.Sprintf("%s|%s", p.Name, p.Component)
		}
		kind := p.Kind
		if kind == "" {
			kind = "    "
		}
		log.Printf("\t%s:%s%s%s => %s%s", kind, fqName, def, fromEnv, value, env)
	}
}
