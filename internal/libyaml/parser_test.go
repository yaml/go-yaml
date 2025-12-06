// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"bytes"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestParser(t *testing.T) {
	RunTestCases(t, "parser.yaml", map[string]TestHandler{
		"parse-events":          runParseEventsTest,
		"parse-events-detailed": runParseEventsDetailedTest,
		"parse-error":           runParseErrorTest,
	})
}

func runParseEventsTest(t *testing.T, tc TestCase) {
	types, ok := parseEvents(tc.Yaml)
	assert.Truef(t, ok, "parseEvents() = %v, want true", ok)

	// Convert Want from interface{} to []string
	wantSlice, ok := tc.Want.([]interface{})
	assert.Truef(t, ok, "Want should be []interface{}")

	var expected []EventType
	for _, item := range wantSlice {
		eventStr, ok := item.(string)
		assert.Truef(t, ok, "Want item should be string")
		expected = append(expected, ParseEventType(t, eventStr))
	}

	assert.Equalf(t, len(expected), len(types), "parseEvents() types length = %d, want %d", len(types), len(expected))
	for i, et := range expected {
		assert.Equalf(t, et, types[i], "parseEvents() types[%d] = %v, want %v", i, types[i], et)
	}
}

func runParseEventsDetailedTest(t *testing.T, tc TestCase) {
	events, ok := parseEventsDetailed(tc.Yaml)
	assert.Truef(t, ok, "parseEventsDetailed() = %v, want true", ok)

	assert.Equalf(t, len(tc.WantSpecs), len(events), "parseEventsDetailed() events length = %d, want %d", len(events), len(tc.WantSpecs))

	for i, wantSpec := range tc.WantSpecs {
		event := events[i]
		wantType := ParseEventType(t, wantSpec.Type)

		assert.Equalf(t, wantType, event.Type, "event[%d].Type = %v, want %v", i, event.Type, wantType)

		// Check specific event properties if specified
		if wantSpec.Anchor != "" {
			assert.Truef(t, bytes.Equal(event.Anchor, []byte(wantSpec.Anchor)),
				"event[%d].Anchor = %q, want %q", i, event.Anchor, wantSpec.Anchor)
		}

		if wantSpec.Tag != "" {
			assert.Truef(t, bytes.Equal(event.Tag, []byte(wantSpec.Tag)),
				"event[%d].Tag = %q, want %q", i, event.Tag, wantSpec.Tag)
		}

		if wantSpec.Value != "" {
			assert.Truef(t, bytes.Equal(event.Value, []byte(wantSpec.Value)),
				"event[%d].Value = %q, want %q", i, event.Value, wantSpec.Value)
		}

		if wantSpec.VersionDirective != nil {
			assert.NotNilf(t, event.version_directive, "event[%d].version_directive should not be nil", i)
			assert.Equalf(t, wantSpec.VersionDirective.Major, int(event.version_directive.major),
				"event[%d].version_directive.major = %d, want %d", i, event.version_directive.major, wantSpec.VersionDirective.Major)
			assert.Equalf(t, wantSpec.VersionDirective.Minor, int(event.version_directive.minor),
				"event[%d].version_directive.minor = %d, want %d", i, event.version_directive.minor, wantSpec.VersionDirective.Minor)
		}

		if len(wantSpec.TagDirectives) > 0 {
			assert.Equalf(t, len(wantSpec.TagDirectives), len(event.tag_directives),
				"event[%d].tag_directives length = %d, want %d", i, len(event.tag_directives), len(wantSpec.TagDirectives))
			for j, wantTd := range wantSpec.TagDirectives {
				assert.Truef(t, bytes.Equal(event.tag_directives[j].handle, []byte(wantTd.Handle)),
					"event[%d].tag_directives[%d].handle = %q, want %q", i, j, event.tag_directives[j].handle, wantTd.Handle)
				assert.Truef(t, bytes.Equal(event.tag_directives[j].prefix, []byte(wantTd.Prefix)),
					"event[%d].tag_directives[%d].prefix = %q, want %q", i, j, event.tag_directives[j].prefix, wantTd.Prefix)
			}
		}
	}
}

func runParseErrorTest(t *testing.T, tc TestCase) {
	parser := NewParser()
	parser.SetInputString([]byte(tc.Yaml))

	var event Event
	for parser.Parse(&event) && event.Type != STREAM_END_EVENT {
	}

	// Convert Want from interface{} to check for error
	// Want can be either a boolean or a single-element sequence
	// Defaults to true if not specified
	wantError := true
	if tc.Want != nil {
		switch v := tc.Want.(type) {
		case bool:
			wantError = v
		case []interface{}:
			if len(v) > 0 {
				if boolVal, ok := v[0].(bool); ok {
					wantError = boolVal
				}
			}
		}
	}
	if wantError {
		assert.Truef(t, parser.ErrorType != NO_ERROR, "Expected parser error, but got none")
	} else {
		assert.Truef(t, parser.ErrorType == NO_ERROR, "Expected no parser error, but got %v", parser.ErrorType)
	}
}
