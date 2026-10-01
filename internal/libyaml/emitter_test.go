// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for the emitter stage.
// Verifies YAML output generation from events.

package libyaml

import (
	"bytes"
	"fmt"
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

// eventQueueDocument returns a mapping with n entries, each holding nested
// mappings, sequences and empty collections, and the exact text a Dumper
// writes for it.
func eventQueueDocument(n int) (map[string]any, string) {
	doc := make(map[string]any, n)
	var want strings.Builder
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("k%06d", i)
		doc[key] = map[string]any{
			"empty": map[string]any{},
			"list":  []any{i, "x", []any{}},
			"name":  key,
			"nested": map[string]any{
				"a": []any{map[string]any{"b": "c"}},
			},
		}
		fmt.Fprintf(&want, "%s:\n", key)
		want.WriteString("  empty: {}\n")
		fmt.Fprintf(&want, "  list:\n  - %d\n  - x\n  - []\n", i)
		fmt.Fprintf(&want, "  name: %s\n", key)
		want.WriteString("  nested:\n    a:\n    - b: c\n")
	}
	return doc, want.String()
}

// The emitter only needs a few events of lookahead, so its event queue
// must not keep every event of the document once they have been written.
func TestEmitterEventQueueStaysSmall(t *testing.T) {
	doc, want := eventQueueDocument(10000)

	var buf bytes.Buffer
	d, err := NewDumper(&buf)
	assert.NoError(t, err)
	assert.NoError(t, d.Dump(doc))
	n := cap(d.serializer.Emitter.events)
	assert.NoError(t, d.Close())

	assert.Truef(t, buf.String() == want,
		"output differs from the expected document")
	assert.Truef(t, n <= initial_queue_size,
		"event queue grew to %d events, want at most %d",
		n, initial_queue_size)
}

// A long-lived Dumper must not keep the events of every document it has
// written.
func TestEmitterEventQueueStaysSmallAcrossDocuments(t *testing.T) {
	doc, one := eventQueueDocument(10)

	var buf bytes.Buffer
	d, err := NewDumper(&buf)
	assert.NoError(t, err)
	for i := 0; i < 1000; i++ {
		assert.NoError(t, d.Dump(doc))
	}
	n := cap(d.serializer.Emitter.events)
	assert.NoError(t, d.Close())

	want := one + strings.Repeat("---\n"+one, 999)
	assert.Truef(t, buf.String() == want,
		"output differs from the expected stream")
	assert.Truef(t, n <= initial_queue_size,
		"event queue grew to %d events, want at most %d",
		n, initial_queue_size)
}

func BenchmarkDumpLargeDocument(b *testing.B) {
	doc, _ := eventQueueDocument(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Dump(doc); err != nil {
			b.Fatal(err)
		}
	}
}
