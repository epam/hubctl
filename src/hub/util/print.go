package util

import (
	"log"
	"strings"
)

func PrintDeps(deps map[string][]string) {
	for _, name := range SortedKeys2(deps) {
		log.Printf("\t%s => %s", name, strings.Join(deps[name], ", "))
	}
}

func PrintMap(m map[string]string) {
	for _, k := range SortedKeys(m) {
		log.Printf("\t%s => `%s`", k, m[k])
	}
}

func PrintMap2(m map[string][]string) {
	for _, k := range SortedKeys2(m) {
		log.Printf("\t%s => `%s`", k, strings.Join(m[k], ", "))
	}
}
