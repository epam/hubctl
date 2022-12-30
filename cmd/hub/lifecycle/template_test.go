// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	result, _ := split("")
	assert.Equal(t, 0, len(result))

	result, _ = split("a/b/c", "/")
	expected := []string{"a", "b", "c"}
	assert.EqualValues(t, expected, result)

	result, _ = split("a b c")
	assert.EqualValues(t, expected, result)
}

func TestCompact(t *testing.T) {
	sample := []string{}
	result, _ := compact(sample)
	assert.Equal(t, 0, len(result))

	sample = []string{"a", "b", "c"}
	result, _ = compact(sample)
	assert.EqualValues(t, []string{"a", "b", "c"}, result)

	sample = []string{"a", "", "c"}
	result, _ = compact(sample)
	assert.EqualValues(t, []string{"a", "c"}, result)

	result, _ = compact("abc")
	assert.EqualValues(t, []string{"abc"}, result)

	result, _ = compact("a", "", "c")
	assert.EqualValues(t, []string{"a", "c"}, result)
}

func TestFirst(t *testing.T) {
	sample := []string{}
	result, _ := first(sample)
	assert.Equal(t, "", result)

	sample = []string{"a", "b", "c"}
	result, _ = first(sample)
	assert.Equal(t, "a", result)
}

func TestJoin(t *testing.T) {
	sample := []string{}
	result, _ := join(sample)
	assert.Equal(t, "", result)

	sample = []string{"a", "b", "c"}
	result, _ = join(sample)
	assert.Equal(t, "a b c", result)

	sample = []string{"a", "b", "c"}
	result, _ = join(sample, "/")
	assert.Equal(t, "a/b/c", result)

	result, _ = join("a", "b", "c", "/")
	assert.Equal(t, "a/b/c", result)
}

func TestFormatSubdomain(t *testing.T) {
	result, _ := formatSubdomain("")
	assert.Equal(t, "", result)

	result, _ = formatSubdomain("a")
	assert.Equal(t, "a", result)

	result, _ = formatSubdomain("a b")
	assert.Equal(t, "a-b", result)

	// dashes cannot repeat
	result, _ = formatSubdomain("A B  c")
	assert.Equal(t, "a-b-c", result)

	// cannot start and finish with dash
	result, _ = formatSubdomain("--a b c--")
	assert.Equal(t, "a-b-c", result)

	// cannot start but can finish with digit
	result, _ = formatSubdomain("12a3 b c4")
	assert.Equal(t, "a3-b-c4", result)

	// max length
	result, _ = formatSubdomain("a b c", 3)
	assert.Equal(t, "a-b", result)

	// second param may be string
	result, _ = formatSubdomain("a b c", "3")
	assert.Equal(t, "a-b", result)

	// max length and delimiter
	result, _ = formatSubdomain("a b c", 3, "_")
	assert.Equal(t, "a_b", result)
}

func TestUnquote(t *testing.T) {
	result, _ := unquote("")
	assert.Equal(t, "", result)

	result, _ = unquote("a")
	assert.Equal(t, "a", result)

	result, _ = unquote("'a'")
	assert.Equal(t, "a", result)

	result, _ = unquote("\"a\"")
	assert.Equal(t, "a", result)

	result, _ = unquote("\"a")
	assert.Equal(t, "\"a", result)

	result, _ = unquote("a\"")
	assert.Equal(t, "a\"", result)

	result, err := unquote("'a")
	assert.Equal(t, "'a", result)
	assert.EqualError(t, err, "invalid syntax")

	result, err = unquote("a'")
	assert.Equal(t, "a'", result)
	assert.EqualError(t, err, "invalid syntax")

	result, _ = unquote("\"a'b\"")
	assert.Equal(t, "a'b", result)

	_, err = unquote("'a\"b'")
	assert.EqualError(t, err, "invalid syntax")
}
