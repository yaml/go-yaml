// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"bytes"
	"errors"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestEmitterFlushEmpty(t *testing.T) {
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)

	err := emitter.flush()
	assert.IsNilf(t, err, "flush() with empty buffer should not error, got %v", err)
	assert.Equalf(t, 0, len(output), "flush() empty buffer produced output %q, want empty", len(output))
}

func TestEmitterFlushWithData(t *testing.T) {
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)

	testData := []byte("test data")
	copy(emitter.buffer, testData)
	emitter.buffer_pos = len(testData)

	err := emitter.flush()
	assert.IsNilf(t, err, "first flush() error: %v", err)
	assert.DeepEqualf(t, testData, output, "flush() output = %q, want %q", output, testData)
	assert.Equalf(t, 0, emitter.buffer_pos, "buffer_pos = %d, want 0", emitter.buffer_pos)
}

func TestEmitterFlushMultipleTimes(t *testing.T) {
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)

	data1 := []byte("first")
	copy(emitter.buffer, data1)
	emitter.buffer_pos = len(data1)

	err := emitter.flush()
	assert.IsNilf(t, err, "first flush() error: %v", err)

	data2 := []byte("second")
	copy(emitter.buffer, data2)
	emitter.buffer_pos = len(data2)

	err = emitter.flush()
	assert.IsNilf(t, err, "second flush() error: %v", err)

	expected := append(data1, data2...)
	assert.DeepEqualf(t, expected, output, "flush() output = %q, want %q", output, expected)
}

func TestEmitterFlushWithWriter(t *testing.T) {
	emitter := NewEmitter()
	var buf bytes.Buffer
	emitter.SetOutputWriter(&buf)

	testData := []byte("test data")
	copy(emitter.buffer, testData)
	emitter.buffer_pos = len(testData)

	err := emitter.flush()
	assert.IsNilf(t, err, "flush() should not error, got %v", err)
	assert.DeepEqualf(t, testData, buf.Bytes(), "flush() output = %q, want %q", buf.Bytes(), testData)
}

type errorWriter struct{}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write error")
}

func TestEmitterFlushWithWriteError(t *testing.T) {
	emitter := NewEmitter()
	emitter.SetOutputWriter(&errorWriter{})

	testData := []byte("test")
	copy(emitter.buffer, testData)
	emitter.buffer_pos = len(testData)

	err := emitter.flush()
	assert.ErrorMatchesf(t, "write error", err, "flush() should return write error, got %v", err)
}

func TestEmitterFlushPanicWithoutHandler(t *testing.T) {
	emitter := NewEmitter()

	testData := []byte("test")
	copy(emitter.buffer, testData)
	emitter.buffer_pos = len(testData)

	assert.PanicMatchesf(t, "write handler not set", func() {
		_ = emitter.flush()
	}, "flush() without write handler should panic")
}
