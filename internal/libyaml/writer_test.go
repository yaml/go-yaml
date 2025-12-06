// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"bytes"
	"errors"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestWriter(t *testing.T) {
	RunTestCases(t, "writer.yaml", map[string]TestHandler{
		"writer-flush": runWriterFlushTest,
		"writer-error": runWriterErrorTest,
		"writer-panic": runWriterPanicTest,
	})
}

func runWriterFlushTest(t *testing.T, tc TestCase) {
	t.Helper()

	emitter := NewEmitter()
	var output []byte
	var buf bytes.Buffer

	// Setup output handler
	if tc.Output == "string" {
		emitter.SetOutputString(&output)
	} else {
		emitter.SetOutputWriter(&buf)
	}

	// Helper to write data and flush
	writeAndFlush := func(data string) {
		if len(data) > 0 {
			copy(emitter.buffer[:], []byte(data))
			emitter.buffer_pos = len(data)
		}
		err := emitter.flush()
		assert.NoErrorf(t, err, "flush() error: %v", err)
	}

	// Write and flush data (once or twice)
	writeAndFlush(string(tc.Input))
	if len(tc.Data2) > 0 {
		writeAndFlush(tc.Data2)
	}

	// Check output
	actual := output
	if tc.Output == "writer" {
		actual = buf.Bytes()
	}
	want, ok := tc.Want.(string)
	assert.Truef(t, ok, "Want should be string, got %T", tc.Want)
	assert.Equalf(t, want, string(actual), "flush() output = %q, want %q", string(actual), want)

	// Run field checks
	if tc.Checks != nil {
		runFieldChecks(t, &emitter, tc.Checks)
	}
}

func runWriterErrorTest(t *testing.T, tc TestCase) {
	t.Helper()

	emitter := NewEmitter()
	emitter.SetOutputWriter(&errorWriter{})

	if len(tc.Input) > 0 {
		copy(emitter.buffer[:], tc.Input)
		emitter.buffer_pos = len(tc.Input)
	}

	err := emitter.flush()
	want, ok := tc.Want.(string)
	assert.Truef(t, ok, "Want should be string, got %T", tc.Want)
	assert.ErrorMatchesf(t, want, err, "flush() should return error matching %q, got %v", want, err)
}

func runWriterPanicTest(t *testing.T, tc TestCase) {
	t.Helper()

	emitter := NewEmitter()

	if len(tc.Input) > 0 {
		copy(emitter.buffer[:], tc.Input)
		emitter.buffer_pos = len(tc.Input)
	}

	want, ok := tc.Want.(string)
	assert.Truef(t, ok, "Want should be string, got %T", tc.Want)
	assert.PanicMatchesf(t, want, func() {
		_ = emitter.flush()
	}, "Expected panic: %s", want)
}

// errorWriter is a writer that always returns an error
type errorWriter struct{}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write error")
}
