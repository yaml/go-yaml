// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"bytes"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestEmitter(t *testing.T) {
	RunTestCases(t, "emitter.yaml", map[string]TestHandler{
		"emit":        RunEmitTest,
		"emit-config": RunEmitTest,
		"roundtrip":   RunRoundTripTest,
		"emit-writer": runEmitWriterTest,
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
		assert.Truef(t, bytes.Contains([]byte(result), []byte(expected)),
			"output should contain %q, got %q", expected, result)
	}
}
