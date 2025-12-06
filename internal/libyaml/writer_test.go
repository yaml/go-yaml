// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"bytes"
	"errors"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestWriter(t *testing.T) {
	RunTestCases(t, "writer_test.yaml", map[string]TestHandler{
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
	writeAndFlush(tc.Data)
	if len(tc.Data2) > 0 {
		writeAndFlush(tc.Data2)
	}

	// Check output
	actual := output
	if tc.Output == "writer" {
		actual = buf.Bytes()
	}
	assert.Equalf(t, tc.Want.(string), string(actual), "flush() output = %q, want %q", string(actual), tc.Want)

	// Run field checks
	if tc.Checks != nil {
		runFieldChecks(t, &emitter, tc.Checks)
	}
}

func runWriterErrorTest(t *testing.T, tc TestCase) {
	t.Helper()

	emitter := NewEmitter()
	emitter.SetOutputWriter(&errorWriter{})

	if len(tc.Data) > 0 {
		copy(emitter.buffer[:], []byte(tc.Data))
		emitter.buffer_pos = len(tc.Data)
	}

	err := emitter.flush()
	assert.ErrorMatchesf(t, tc.Want.(string), err, "flush() should return error matching %q, got %v", tc.Want, err)
}

func runWriterPanicTest(t *testing.T, tc TestCase) {
	t.Helper()

	emitter := NewEmitter()

	if len(tc.Data) > 0 {
		copy(emitter.buffer[:], []byte(tc.Data))
		emitter.buffer_pos = len(tc.Data)
	}

	assert.PanicMatchesf(t, tc.Want.(string), func() {
		_ = emitter.flush()
	}, "Expected panic: %s", tc.Want)
}

// errorWriter is a writer that always returns an error
type errorWriter struct{}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write error")
}
