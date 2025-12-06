// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestYAML(t *testing.T) {
	RunTestCases(t, "yaml_test.yaml", map[string]TestHandler{
		"enum-string":    runEnumStringTest,
		"style-accessor": runStyleAccessorTest,
	})
}

func runEnumStringTest(t *testing.T, tc TestCase) {
	var got string

	switch tc.EnumType {
	case "ScalarStyle":
		got = ScalarStyle(tc.Value.Value).String()
	case "TokenType":
		got = TokenType(tc.Value.Value).String()
	case "EventType":
		got = EventType(tc.Value.Value).String()
	case "ParserState":
		got = ParserState(tc.Value.Value).String()
	default:
		t.Fatalf("unknown enum type: %s", tc.EnumType)
	}

	// Want can be either a string or a single-element sequence
	var want string
	switch v := tc.Want.(type) {
	case string:
		want = v
	case []interface{}:
		want = v[0].(string)
	default:
		t.Fatalf("want must be a string or sequence, got %T", tc.Want)
	}
	assert.Equalf(t, want, got, "%s(%d).String() = %q, want %q", tc.EnumType, tc.Value.Value, got, want)
}

func runStyleAccessorTest(t *testing.T, tc TestCase) {
	event := Event{Style: Style(tc.StyleValue.Value)}

	var got int
	switch tc.Accessor {
	case "ScalarStyle":
		got = int(event.ScalarStyle())
	case "SequenceStyle":
		got = int(event.SequenceStyle())
	case "MappingStyle":
		got = int(event.MappingStyle())
	default:
		t.Fatalf("unknown accessor: %s", tc.Accessor)
	}

	wantValue := tc.StyleValue.Value // Expected result is the style value itself
	assert.Equalf(t, wantValue, got, "Event.%s() = %v, want %v", tc.Accessor, got, wantValue)
}
