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

// TestEmitterInvalidUTF8DoesNotPanic validates emitter behavior in isolation,
// without involving scanner/parser, when scalar values contain malformed UTF-8.
func TestEmitterInvalidUTF8DoesNotPanic(t *testing.T) {
	for name, malformed := range map[string][]byte{
		"Incomplete 2-byte UTF-8 sequence": {0xC2},
		"Incomplete 3-byte UTF-8 sequence": {0xEF},
		"Incomplete 4-byte UTF-8 sequence": {0xF0},
		"truncated BOM sequence":           {0xEF, 0xBB},
	} {
		t.Run(name, func(t *testing.T) {
			t.Run("plain scalar style", func(t *testing.T) {
				emitter := NewEmitter()
				var out []byte
				emitter.SetOutputString(&out)
				emitter.SetUnicode(true)

				for _, event := range []struct {
					Event                 Event
					ExpectedErrorContains string
				}{
					{Event: NewStreamStartEvent(UTF8_ENCODING)},
					{Event: NewDocumentStartEvent(nil, nil, true)},
					{
						Event:                 NewScalarEvent(nil, nil, malformed, true, false, PLAIN_SCALAR_STYLE),
						ExpectedErrorContains: "incomplete UTF-8 octet sequence",
					},
					{Event: NewDocumentEndEvent(true)},
					{Event: NewStreamEndEvent()},
				} {
					err := emitter.Emit(&event.Event)
					if event.ExpectedErrorContains == "" {
						assert.NoError(t, err)
						continue
					}

					assert.ErrorMatches(t, event.ExpectedErrorContains, err)
					if err != nil {
						break // stop emitting further events after the expected error, all further events would be no-ops anyway due to the error state
					}
				}

				// invalid UTF-8 should not be emitted, output should be empty
				assert.Equal(t, "", string(out))
			})
			t.Run(name, func(t *testing.T) {
				t.Run("double-quoted scalar style", func(t *testing.T) {
					emitter := NewEmitter()
					var out []byte
					emitter.SetOutputString(&out)
					emitter.SetUnicode(true)

					for _, event := range []struct {
						Event                 Event
						ExpectedErrorContains string
					}{
						{Event: NewStreamStartEvent(UTF8_ENCODING)},
						{Event: NewDocumentStartEvent(nil, nil, true)},
						{
							Event:                 NewScalarEvent(nil, nil, malformed, true, false, DOUBLE_QUOTED_SCALAR_STYLE),
							ExpectedErrorContains: "incomplete UTF-8 octet sequence",
						},
						{Event: NewDocumentEndEvent(true)},
						{Event: NewStreamEndEvent()},
					} {
						err := emitter.Emit(&event.Event)
						if event.ExpectedErrorContains == "" {
							assert.NoError(t, err)
							continue
						}

						assert.ErrorMatches(t, event.ExpectedErrorContains, err)
						if err != nil {
							break // stop emitting further events after the expected error, all further events would be no-ops anyway due to the error state
						}
					}

					assert.Equal(t, "", string(out))
				})
			})
		})
	}
}
