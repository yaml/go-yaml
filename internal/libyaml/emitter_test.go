// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for the emitter stage.
// Verifies YAML output generation from events.

package libyaml

import (
	"bytes"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestEmitter(t *testing.T) {
	RunTestCases(t, "emitter.yaml", map[string]TestHandler{
		"emit":          RunEmitTest,
		"emit-config":   RunEmitTest,
		"roundtrip":     RunRoundTripTest,
		"emit-writer":   runEmitWriterTest,
		"api-new":       runAPINewTest,
		"api-method":    runAPIMethodTest,
		"api-panic":     runAPIPanicTest,
		"api-delete":    runAPIDeleteTest,
		"api-new-event": runAPINewEventTest,
	})
}

func emitFoldedScalar(t *testing.T, value string) string {
	t.Helper()

	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewScalarEvent(nil, nil, []byte(value), true, true, FOLDED_SCALAR_STYLE),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	emitter := NewEmitter()
	emitter.SetIndent(2)
	var output []byte
	emitter.SetOutputString(&output)
	for i := range events {
		err := emitter.Emit(&events[i])
		assert.NoErrorf(t, err, "Emit() error: %v", err)
	}
	return string(output)
}

func TestEmitFoldedScalarNoExtraNewline(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "heading then more-indented block",
			value: "Heading:\n\n  * first item\n  * second item\n",
			want:  ">\n  Heading:\n\n    * first item\n    * second item\n",
		},
		{
			name:  "single newline between plain lines",
			value: "one\ntwo\n",
			want:  ">\n  one\n\n  two\n",
		},
		{
			name:  "trailing more-indented block",
			value: "intro\n\n  indented tail\n",
			want:  ">\n  intro\n\n    indented tail\n",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := emitFoldedScalar(t, tc.value)
			assert.Equal(t, tc.want, got)
		})
	}
}

func runEmitWriterTest(t *testing.T, tc TestCase) {
	t.Helper()

	var events []Event
	for _, eventSpec := range tc.Events {
		events = append(events, CreateEventFromSpec(t, eventSpec))
	}

	emitter := NewEmitter()
	var buf bytes.Buffer
	emitter.SetOutputWriter(&buf)

	for i := range events {
		err := emitter.Emit(&events[i])
		assert.NoErrorf(t, err, "Emit() error: %v", err)
	}

	result := buf.String()
	for _, expected := range tc.WantContains {
		assert.Truef(t, strings.Contains(result, expected),
			"output should contain %q, got %q", expected, result)
	}
}
