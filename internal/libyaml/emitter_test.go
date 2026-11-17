// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for the emitter stage.
// Verifies YAML output generation from events.

package libyaml

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml/internal/testutil/assert"
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
			events := []Event{
				NewStreamStartEvent(UTF8_ENCODING),
				NewDocumentStartEvent(nil, nil, true),
				NewScalarEvent(nil, nil, []byte(tc.value), true, true, FOLDED_SCALAR_STYLE),
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
			assert.Equal(t, tc.want, string(output))
		})
	}
}

// TestEmitAlias verifies that emitting an alias only needs the anchor
// data that analyzeEvent extracted from the event: the alias event
// itself carries nothing else the emitter has to consult.
func TestEmitAlias(t *testing.T) {
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)

	event := NewAliasEvent([]byte("anchor"))
	err := emitter.analyzeEvent(&event)
	assert.NoErrorf(t, err, "analyzeEvent() error: %v", err)

	// The state machine reaches emitAlias right after a token separator
	// (an indent or a mapping value indicator), with one state to pop.
	emitter.whitespace = true
	emitter.states = append(emitter.states, EMIT_DOCUMENT_CONTENT_STATE)

	err = emitter.emitAlias()
	assert.NoErrorf(t, err, "emitAlias() error: %v", err)
	err = emitter.flush()
	assert.NoErrorf(t, err, "flush() error: %v", err)

	assert.Equal(t, "*anchor", string(output))
	assert.Equal(t, EMIT_DOCUMENT_CONTENT_STATE, emitter.state)
	assert.Equal(t, 0, len(emitter.states))
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

// emitMeasuringQueue emits events and returns the output and the largest
// capacity the event queue reached.
func emitMeasuringQueue(tb testing.TB, events []Event) (string, int) {
	tb.Helper()
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)
	queueCap := 0
	for i := range events {
		err := emitter.Emit(&events[i])
		assert.NoErrorf(tb, err, "Emit() error: %v", err)
		if cap(emitter.events) > queueCap {
			queueCap = cap(emitter.events)
		}
	}
	return string(output), queueCap
}

func documentEvents(body ...Event) []Event {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
	}
	events = append(events, body...)
	return append(events, NewDocumentEndEvent(true), NewStreamEndEvent())
}

func scalarEvent(value string) Event {
	return NewScalarEvent(nil, nil, []byte(value), true, true, PLAIN_SCALAR_STYLE)
}

// The emitter keeps only a short lookahead of events, however long the
// document.
func TestEmitterReclaimsWrittenEvents(t *testing.T) {
	sequence := func(n int) []Event {
		body := []Event{
			NewSequenceStartEvent(nil, nil, true, BLOCK_SEQUENCE_STYLE),
		}
		for i := 0; i < n; i++ {
			body = append(body, scalarEvent("item"))
		}
		return documentEvents(append(body, NewSequenceEndEvent())...)
	}

	output, queueCap := emitMeasuringQueue(t, sequence(10000))
	assert.Equal(t, strings.Repeat("- item\n", 10000), output)
	assert.Equal(t, initial_queue_size, queueCap)
}

// Nested collections hold events back for lookahead while the head moves on,
// so written events are compacted out from in front of the pending ones.
func TestEmitterCompactsQueueBehindLookahead(t *testing.T) {
	nested := func(depth int) ([]Event, string) {
		var body []Event
		for i := 0; i < depth; i++ {
			body = append(body,
				NewSequenceStartEvent(nil, nil, true, FLOW_SEQUENCE_STYLE))
		}
		body = append(body, scalarEvent("deep"))
		for i := 0; i < depth; i++ {
			body = append(body, NewSequenceEndEvent())
		}
		want := strings.Repeat("[", depth) + "deep" +
			strings.Repeat("]", depth) + "\n"
		return documentEvents(body...), want
	}

	for _, depth := range []int{5 * initial_queue_size, 50 * initial_queue_size} {
		events, want := nested(depth)
		output, queueCap := emitMeasuringQueue(t, events)
		assert.Equal(t, want, output)
		assert.Equal(t, initial_queue_size, queueCap)
	}
}

func BenchmarkEmitLongSequence(b *testing.B) {
	body := []Event{NewSequenceStartEvent(nil, nil, true, BLOCK_SEQUENCE_STYLE)}
	for i := 0; i < 100000; i++ {
		body = append(body, scalarEvent("item"))
	}
	body = append(body, NewSequenceEndEvent())
	events := documentEvents(body...)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		batch := make([]Event, len(events))
		copy(batch, events)
		emitMeasuringQueue(b, batch)
	}
}
