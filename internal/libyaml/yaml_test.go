// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for core libyaml types.
// Verifies Event, Token, and other fundamental type operations.

package libyaml

import (
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestYAML(t *testing.T) {
	RunTestCases(t, "yaml.yaml", map[string]TestHandler{
		"enum-string":    runEnumStringTest,
		"style-accessor": runStyleAccessorTest,
	})
}

// runEnumStringTest tests the String() method of various enum types.
//
//nolint:thelper // because this function is the real test
func runEnumStringTest(t *testing.T, tc TestCase) {
	// Parse enum array: [Type, Value]
	if len(tc.Enum) != 2 {
		t.Fatalf("enum must be [Type, Value], got %v", tc.Enum)
	}
	enumType, ok := tc.Enum[0].(string)
	if !ok {
		t.Fatalf("enum type must be string, got %T", tc.Enum[0])
	}

	// Value can be int or string constant
	var enumValue int
	switch v := tc.Enum[1].(type) {
	case int:
		enumValue = v
	case string:
		// Parse as constant - this will be resolved by the constant lookup
		enumValue = resolveConstant(t, v)
	default:
		t.Fatalf("enum value must be int or string, got %T", tc.Enum[1])
	}

	var got string
	switch enumType {
	case "ScalarStyle":
		got = ScalarStyle(enumValue).String()
	case "TokenType":
		got = TokenType(enumValue).String()
	case "EventType":
		got = EventType(enumValue).String()
	case "ParserState":
		got = ParserState(enumValue).String()
	default:
		t.Fatalf("unknown enum type: %s", enumType)
	}

	// Want can be either a string or a single-element sequence
	var want string
	switch v := tc.Want.(type) {
	case string:
		want = v
	case []any:
		if len(v) > 0 {
			var ok bool
			want, ok = v[0].(string)
			if !ok {
				t.Fatalf("want[0] must be string, got %T", v[0])
			}
		} else {
			t.Fatalf("Want slice is empty, expected at least one element")
		}
	default:
		t.Fatalf("want must be a string or sequence, got %T", tc.Want)
	}
	assert.Equalf(t, want, got, "%s(%d).String() = %q, want %q", enumType, enumValue, got, want)
}

// runStyleAccessorTest tests the style accessor methods of Event.
//
//nolint:thelper // because this function is the real test
func runStyleAccessorTest(t *testing.T, tc TestCase) {
	// Parse test array: [Method, STYLE]
	if len(tc.StyleTest) != 2 {
		t.Fatalf("test must be [Method, STYLE], got %v", tc.StyleTest)
	}
	method, ok := tc.StyleTest[0].(string)
	if !ok {
		t.Fatalf("method must be string, got %T", tc.StyleTest[0])
	}

	// Style value can be int or string constant
	var styleValue int
	switch v := tc.StyleTest[1].(type) {
	case int:
		styleValue = v
	case string:
		styleValue = resolveConstant(t, v)
	default:
		t.Fatalf("style value must be int or string, got %T", tc.StyleTest[1])
	}

	event := Event{Style: Style(styleValue)}

	var got int
	switch method {
	case "ScalarStyle":
		got = int(event.ScalarStyle())
	case "SequenceStyle":
		got = int(event.SequenceStyle())
	case "MappingStyle":
		got = int(event.MappingStyle())
	default:
		t.Fatalf("unknown accessor: %s", method)
	}

	assert.Equalf(t, styleValue, got, "Event.%s() = %v, want %v", method, got, styleValue)
}
