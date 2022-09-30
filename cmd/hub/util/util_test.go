package util_test

import (
	"errors"
	"reflect"
	"testing"

	. "github.com/epam/hubctl/cmd/hub/util"
	"github.com/stretchr/testify/assert"
)

// func Test_maybeHighlight(t *testing.T) {
// 	type args struct {
// 		color func(interface{}) aurora.Value
// 	}
// 	tests := []struct {
// 		name string
// 		args args
// 		want func(string) string
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := maybeHighlight(tt.args.color); !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("maybeHighlight() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

func TestIsLogTerminal(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Should return False if run not in terminal", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLogTerminal(); got != tt.want {
				t.Errorf("IsLogTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWarn(t *testing.T) {
	type args struct {
		format string
		v      []interface{}
	}

	var msgs = make([]interface{}, 0)
	msgs = append(msgs, "Just string")

	tests := []struct {
		name string
		args args
	}{
		{"Should not fail with empty format", args{"", nil}},
		{"Should not fail with just string", args{"Just string", nil}},
		{"Should not fail with simple format", args{"%s", msgs}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Warn(tt.args.format, tt.args.v...)
		})
	}
}

func TestWarnOnce(t *testing.T) {
	type args struct {
		format string
		v      []interface{}
	}

	var msgs = make([]interface{}, 0)
	msgs = append(msgs, "Just string")

	tests := []struct {
		name string
		args args
	}{
		{"Should not fail with empty format", args{"", nil}},
		{"Should not fail with just string", args{"Just string", nil}},
		{"Should not fail with duplicate just string", args{"Just string", nil}},
		{"Should not fail with simple format", args{"%s", msgs}},
		{"Should not fail with duplicate simple format", args{"%s", msgs}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			WarnOnce(tt.args.format, tt.args.v...)
		})
	}
}

func TestPrintAllWarnings(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"Should not fail"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrintAllWarnings()
		})
	}
}

func TestErrors(t *testing.T) {
	type args struct {
		sep         string
		maybeErrors []error
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Should return (no errors)", args{"", nil}, "(no errors)"},
		{"Should return (no errors)", args{",", nil}, "(no errors)"},
		{
			"Should return 'error1'",
			args{"", []error{errors.New("error1")}},
			"error1",
		},
		{
			"Should return 'error1, error2'",
			args{"", []error{errors.New("error1"), errors.New("error2")}},
			"error1, error2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Errors(tt.args.sep, tt.args.maybeErrors...); got != tt.want {
				t.Errorf("Errors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrors2(t *testing.T) {
	type args struct {
		maybeErrors []error
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Should return (no errors)", args{nil}, "(no errors)"},
		{"Should return (no errors)", args{nil}, "(no errors)"},
		{
			"Should return 'error1'",
			args{[]error{errors.New("error1")}},
			"error1",
		},
		{
			"Should return 'error1, error2'",
			args{[]error{errors.New("error1"), errors.New("error2")}},
			"error1, error2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Errors2(tt.args.maybeErrors...); got != tt.want {
				t.Errorf("Errors2() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAskInput(t *testing.T) {
	type args struct {
		input  string
		prompt string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Should return empty string", args{"", ""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AskInput(tt.args.input, tt.args.prompt); got != tt.want {
				t.Errorf("AskInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAskPassword(t *testing.T) {
	type args struct {
		input  string
		prompt string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Should return empty string", args{"", ""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AskPassword(tt.args.input, tt.args.prompt); got != tt.want {
				t.Errorf("AskPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyMap2(t *testing.T) {
	type args struct {
		m map[string][]string
	}

	testMap := map[string][]string{
		"test1": nil,
		"test2": {},
		"test3": {"test3"},
	}

	tests := []struct {
		name string
		args args
		want map[string][]string
	}{
		{"Should not fail", args{nil}, map[string][]string{}},
		{"Should equal to", args{testMap}, map[string][]string{
			"test1": {},
			"test2": {},
			"test3": {"test3"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyMap2(tt.args.m); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyMap2() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppendMapList(t *testing.T) {
	type args struct {
		m     map[string][]string
		key   string
		value string
	}
	testMap := map[string][]string{}
	tests := []struct {
		name string
		args args
	}{
		{"Should contain empty value by empty key", args{map[string][]string{}, "", ""}},
		{"Should contain value by key", args{testMap, "key1", ""}},
		{"Should contain value by key", args{testMap, "key1", "value1"}},
		{"Should contain value by key", args{testMap, "key1", "value2"}},
		{"Should contain value by key", args{testMap, "key2", "value1"}},
		{"Should contain value by key", args{testMap, "key2", "value2"}},
		{"Should contain value by key", args{testMap, "", "value1"}},
		{"Should contain value by key", args{testMap, "", "value2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AppendMapList(tt.args.m, tt.args.key, tt.args.value)
			assert.Contains(t, tt.args.m[tt.args.key], tt.args.value)
		})
	}
}

func TestConcatMaps(t *testing.T) {
	type args struct {
		m1 map[string]string
		m2 map[string]string
	}
	firstMap := map[string]string{
		"first": "map",
	}
	secondMap := map[string]string{
		"second": "map",
	}
	emptyMap := map[string]string{}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{"Should return empty map", args{nil, nil}, map[string]string{}},
		{"Should return empty map", args{emptyMap, nil}, map[string]string{}},
		{"Should return empty map", args{nil, emptyMap}, map[string]string{}},
		{
			"Should return first map",
			args{
				firstMap,
				emptyMap,
			},
			map[string]string{
				"first": "map",
			},
		},
		{
			"Should return second map",
			args{
				emptyMap,
				secondMap,
			},
			map[string]string{
				"second": "map",
			},
		},
		{
			"Should concat two maps",
			args{
				firstMap,
				secondMap,
			},
			map[string]string{
				"first":  "map",
				"second": "map",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConcatMaps(tt.args.m1, tt.args.m2); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConcatMaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	type args struct {
		source []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"Should not fail", args{nil}, nil},
		{"Should return same array", args{[]string{"1"}}, []string{"1"}},
		{"Should return reversed array", args{[]string{"1", "2"}}, []string{"2", "1"}},
		{"Should return reversed array", args{[]string{"1", "2", "3", "4", "5"}}, []string{"5", "4", "3", "2", "1"}},
		{"Should return reversed array", args{[]string{"abc", "acb"}}, []string{"acb", "abc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Reverse(tt.args.source); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reverse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUniq(t *testing.T) {
	type args struct {
		source []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"Should return empty array", args{nil}, []string{}},
		{"Should return empty array", args{[]string{}}, []string{}},
		{"Should return uniq array", args{[]string{"1", "1", "2", "3", "3"}}, []string{"1", "2", "3"}},
		{"Should return uniq array", args{[]string{"1", "2", "3", "3", "4", "5"}}, []string{"1", "2", "3", "4", "5"}},
		{"Should return uniq array", args{[]string{"2", "1", "5", "3", "4", "3", "4"}}, []string{"1", "2", "3", "4", "5"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Uniq(tt.args.source); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Uniq() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUniqInOrder(t *testing.T) {
	type args struct {
		source []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"Should return empty array", args{nil}, []string{}},
		{"Should return empty array", args{[]string{}}, []string{}},
		{"Should return uniq array", args{[]string{"1", "1", "2", "3", "3"}}, []string{"1", "2", "3"}},
		{"Should return uniq array", args{[]string{"1", "2", "3", "3", "4", "5"}}, []string{"1", "2", "3", "4", "5"}},
		{"Should return uniq array", args{[]string{"2", "1", "5", "3", "4", "3", "4"}}, []string{"2", "1", "5", "3", "4"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UniqInOrder(tt.args.source); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UniqInOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	type args struct {
		list  []string
		value string
	}
	testArr := []string{"2", "1", "5", "3", "4", "3", "4"}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Should not fail", args{nil, ""}, false},
		{"Should return true", args{testArr, "5"}, true},
		{"Should return false", args{[]string{}, ""}, false},
		{"Should return false", args{[]string{}, "1"}, false},
		{"Should return false", args{testArr, "6"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.args.list, tt.args.value); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsSubstring(t *testing.T) {
	type args struct {
		list   []string
		substr string
	}
	testArr := []string{"2", "1", "5", "3", "4", "3", "4"}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Should not fail", args{nil, ""}, false},
		{"Should return true", args{testArr, "5"}, true},
		{"Should return false", args{[]string{}, ""}, false},
		{"Should return false", args{[]string{}, "1"}, false},
		{"Should return false", args{testArr, "6"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsSubstring(tt.args.list, tt.args.substr); got != tt.want {
				t.Errorf("ContainsSubstring() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsPrefix(t *testing.T) {
	type args struct {
		list  []string
		value string
	}
	testArr := []string{"hub*", "nonhub*", "cli"}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Should return false", args{nil, ""}, false},
		{"Should return false", args{[]string{}, ""}, false},
		{"Should return false", args{testArr, "git"}, false},
		{"Should return false", args{testArr, "cli"}, true},
		{"Should return false", args{testArr, "cliTest"}, true},
		{"Should return true", args{testArr, "hubTest"}, true},
		{"Should return true", args{testArr, "nonhubTest"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsPrefix(tt.args.list, tt.args.value); got != tt.want {
				t.Errorf("ContainsPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsAll(t *testing.T) {
	type args struct {
		list   []string
		values []string
	}
	emptyArr := []string{}
	testArr := []string{"1", "2", "3", "4", "5"}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Should return true if both nil", args{nil, nil}, true},
		{"Should return true if list nil and values empty", args{nil, emptyArr}, true},
		{"Should return true if list empty and values nil", args{emptyArr, nil}, true},
		{"Should return true if list empty and values empty", args{emptyArr, emptyArr}, true},
		{"Should return true if list with testing values and values empty", args{testArr, emptyArr}, true},
		{"Should return false if testing values does not contain one value", args{testArr, []string{"1", "6"}}, false},
		{"Should return true if testing values contains both elements", args{testArr, []string{"1", "5"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsAll(tt.args.list, tt.args.values); got != tt.want {
				t.Errorf("ContainsAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	type args struct {
		list   []string
		values []string
	}
	emptyArr := []string{}
	testArr := []string{"1", "2", "3", "4", "5"}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Should return false if both nil", args{nil, nil}, false},
		{"Should return false if list nil and values empty", args{nil, emptyArr}, false},
		{"Should return false if list empty and values nil", args{emptyArr, nil}, false},
		{"Should return false if list empty and values empty", args{emptyArr, emptyArr}, false},
		{"Should return false if list with testing values and values empty", args{testArr, emptyArr}, false},
		{"Should return true if testing values does not contain one value", args{testArr, []string{"1", "6"}}, true},
		{"Should return true if testing values contains both elements", args{testArr, []string{"1", "5"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsAny(tt.args.list, tt.args.values); got != tt.want {
				t.Errorf("ContainsAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnion(t *testing.T) {
	type args struct {
		list  []string
		list2 []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Union(tt.args.list, tt.args.list2); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Union() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsAnySubstring(t *testing.T) {
	type args struct {
		list    []string
		substrs []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsAnySubstring(tt.args.list, tt.args.substrs); got != tt.want {
				t.Errorf("ContainsAnySubstring() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEqual(t *testing.T) {
	type args struct {
		list  []string
		list2 []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Equal(tt.args.list, tt.args.list2); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOmit(t *testing.T) {
	type args struct {
		list  []string
		value string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Omit(tt.args.list, tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Omit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOmitAll(t *testing.T) {
	type args struct {
		list   []string
		values []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OmitAll(tt.args.list, tt.args.values); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OmitAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter(t *testing.T) {
	type args struct {
		list     []string
		patterns []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Filter(tt.args.list, tt.args.patterns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterNot(t *testing.T) {
	type args struct {
		list     []string
		patterns []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterNot(tt.args.list, tt.args.patterns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterNot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndex(t *testing.T) {
	type args struct {
		list   []string
		search string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Index(tt.args.list, tt.args.search); got != tt.want {
				t.Errorf("Index() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortedKeys(t *testing.T) {
	type args struct {
		m map[string]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortedKeys(tt.args.m); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortedKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortedKeys2(t *testing.T) {
	type args struct {
		m map[string][]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortedKeys2(tt.args.m); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SortedKeys2() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeUnique(t *testing.T) {
	type args struct {
		lists [][]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeUnique(tt.args.lists...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeUnique() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValue(t *testing.T) {
	type args struct {
		values []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Value(tt.args.values...); got != tt.want {
				t.Errorf("Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Wrap(tt.args.str); got != tt.want {
				t.Errorf("Wrap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmpty(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Empty(tt.args.value); got != tt.want {
				t.Errorf("Empty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := String(tt.args.value); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaybeJson(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaybeJson(tt.args.value); got != tt.want {
				t.Errorf("MaybeJson() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrimColor(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimColor(tt.args.str); got != tt.want {
				t.Errorf("TrimColor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrim(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Trim(tt.args.str); got != tt.want {
				t.Errorf("Trim() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoSuchFile(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NoSuchFile(tt.args.err); got != tt.want {
				t.Errorf("NoSuchFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContextCanceled(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContextCanceled(tt.args.err); got != tt.want {
				t.Errorf("ContextCanceled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlural(t *testing.T) {
	type args struct {
		size int
		noun []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Plural(tt.args.size, tt.args.noun...); got != tt.want {
				t.Errorf("Plural() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitPaths(t *testing.T) {
	type args struct {
		paths string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitPaths(tt.args.paths); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitPaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStripDotDirs(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripDotDirs(tt.args.path); got != tt.want {
				t.Errorf("StripDotDirs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMustAbs(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MustAbs(tt.args.path); got != tt.want {
				t.Errorf("MustAbs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasedir(t *testing.T) {
	type args struct {
		paths []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Basedir(tt.args.paths); got != tt.want {
				t.Errorf("Basedir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlainName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PlainName(tt.args.name); got != tt.want {
				t.Errorf("PlainName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitQName(t *testing.T) {
	type args struct {
		qName string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := SplitQName(tt.args.qName)
			if got != tt.want {
				t.Errorf("SplitQName() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("SplitQName() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

// func Test_initSecretSuffixes(t *testing.T) {
// 	tests := []struct {
// 		name string
// 		want []string
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := initSecretSuffixes(); !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("initSecretSuffixes() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

func TestLooksLikeSecret(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LooksLikeSecret(tt.args.name); got != tt.want {
				t.Errorf("LooksLikeSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaybeMaskedValue(t *testing.T) {
	type args struct {
		trace bool
		name  string
		value string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaybeMaskedValue(tt.args.trace, tt.args.name, tt.args.value); got != tt.want {
				t.Errorf("MaybeMaskedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseKvList(t *testing.T) {
	type args struct {
		list string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseKvList(tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKvList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseKvList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUint(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUint(tt.args.str); got != tt.want {
				t.Errorf("IsUint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaybeEnv(t *testing.T) {
	type args struct {
		vars []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaybeEnv(tt.args.vars); got != tt.want {
				t.Errorf("MaybeEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
