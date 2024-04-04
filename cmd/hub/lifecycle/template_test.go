// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"reflect"
	"testing"
)

func TestProcessGo(t *testing.T) {
	type args struct {
		content string
		kv      map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Should render go template with split", args{`{{range (.a | split " ")}}{{.}}{{end}}`, map[string]interface{}{"a": "a b c d"}}, `abcd`},
		{"Should render go template with join", args{`{{.a | join ","}}`, map[string]interface{}{"a": []string{"a", "b", "c", "d"}}}, `a,b,c,d`},
		{"Should render go template with compact", args{`{{.a | compact | join ""}}`, map[string]interface{}{"a": []string{"a", "", "b", "", "c", "d", ""}}}, `abcd`},
		{"Should render go template with first", args{`{{.a | first}}`, map[string]interface{}{"a": []string{"a", "b", "c", "d"}}}, `a`},
		{"Should render go template with formatSubdomain", args{`{{.a | formatSubdomain}}`, map[string]interface{}{"a": "1a.b_c_d+2--3?"}}, `a-b-c-d-2-3`},
		{"Should render go template with unquote", args{`{{.a | unquote}}`, map[string]interface{}{"a": `"test"`}}, `test`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processGo(tt.args.content, "test.gotemplate", "test-component", tt.args.kv)
			if !reflect.DeepEqual(got, tt.want) || err != nil {
				if err != nil {
					t.Errorf("processGo() error: %v", err)
				} else {
					t.Errorf("processGo() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestProcessReplacement(t *testing.T) {
	type args struct {
		content string
		kv      map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Should render curly template with base64", args{`${a|base64}`, map[string]interface{}{"a": "hubctl"}}, `aHViY3Rs`},
		{"Should render go template with unbase64", args{`${a|unbase64}`, map[string]interface{}{"a": "aHViY3Rs"}}, `hubctl`},
		{"Should render go template with json", args{`${a|json}`, map[string]interface{}{"a": map[string]string{"a": "1", "b": "2", "c": "3"}}}, `{"a":"1","b":"2","c":"3"}`},
		{"Should render go template with yaml", args{`${a|yaml}`, map[string]interface{}{"a": map[string]string{"a": "1", "b": "2", "c": "3"}}}, `a: "1"
b: "2"
c: "3"`},
		{"Should render go template with first", args{`${a|first}`, map[string]interface{}{"a": "a b c"}}, `a`},
		{"Should render go template with parseURL", args{`${a|parseURL}`, map[string]interface{}{"a": "https://hubctl.io"}}, `https://hubctl.io:443`},
		{"Should render go template with parseURL", args{`${a|isSecure}`, map[string]interface{}{"a": "https://hubctl.io"}}, `true`},
		{"Should render go template with insecure", args{`${a|insecure}`, map[string]interface{}{"a": "https://hubctl.io"}}, `false`},
		{"Should render go template with hostname", args{`${a|hostname}`, map[string]interface{}{"a": "https://hubctl.io"}}, `hubctl.io`},
		{"Should render go template with port", args{`${a|port}`, map[string]interface{}{"a": "https://hubctl.io"}}, `443`},
		{"Should render go template with scheme", args{`${a|scheme}`, map[string]interface{}{"a": "https://hubctl.io"}}, `https`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errs := processReplacement(tt.args.content, "test.template", "test-component", []string{}, tt.args.kv, curlyReplacement, stripCurly)
			if !reflect.DeepEqual(got, tt.want) || len(errs) > 0 {
				if len(errs) > 0 {
					t.Errorf("processReplacement() error: %v", errs)
				} else {
					t.Errorf("processReplacement() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
