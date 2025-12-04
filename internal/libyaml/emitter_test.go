// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func emitEvents(events []Event) (string, error) {
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)

	for i := range events {
		if err := emitter.Emit(&events[i]); !errors.Is(err, nil) {
			return "", err
		}
	}

	return string(output), nil
}

func TestEmitterSimpleScalar(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewScalarEvent(nil, nil, []byte("hello"), true, false, PLAIN_SCALAR_STYLE),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)

	assert.Truef(t, strings.Contains(output, "hello"), "emitEvents() output = %q, should contain 'hello'", output)
}

func TestEmitterSimpleMapping(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewMappingStartEvent(nil, nil, true, BLOCK_MAPPING_STYLE),
		NewScalarEvent(nil, nil, []byte("key"), true, false, PLAIN_SCALAR_STYLE),
		NewScalarEvent(nil, nil, []byte("value"), true, false, PLAIN_SCALAR_STYLE),
		NewMappingEndEvent(),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)

	assert.Truef(t, strings.Contains(output, "key"), "emitEvents() output = %q, should contain 'key'", output)
	assert.Truef(t, strings.Contains(output, "value"), "emitEvents() output = %q, should contain 'value'", output)
}

func TestEmitterBlockSequence(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewSequenceStartEvent(nil, nil, true, BLOCK_SEQUENCE_STYLE),
		NewScalarEvent(nil, nil, []byte("item1"), true, false, PLAIN_SCALAR_STYLE),
		NewScalarEvent(nil, nil, []byte("item2"), true, false, PLAIN_SCALAR_STYLE),
		NewSequenceEndEvent(),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)
	assert.Truef(t, strings.Contains(output, "item1"), "emitEvents() output = %q, should contain 'item1'", output)
	assert.Truef(t, strings.Contains(output, "item2"), "emitEvents() output = %q, should contain 'item2'", output)
	assert.Truef(t, strings.Contains(output, "-"), "emitEvents() output = %q, should contain sequence indicator '-'", output)
}

func TestEmitterFlowSequence(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewSequenceStartEvent(nil, nil, true, FLOW_SEQUENCE_STYLE),
		NewScalarEvent(nil, nil, []byte("1"), true, false, PLAIN_SCALAR_STYLE),
		NewScalarEvent(nil, nil, []byte("2"), true, false, PLAIN_SCALAR_STYLE),
		NewScalarEvent(nil, nil, []byte("3"), true, false, PLAIN_SCALAR_STYLE),
		NewSequenceEndEvent(),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)
	assert.Truef(t, strings.Contains(output, "["), "emitEvents() output = %q, should contain '['", output)
	assert.Truef(t, strings.Contains(output, "]"), "emitEvents() output = %q, should contain ']'", output)
}

func TestEmitterFlowMapping(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewMappingStartEvent(nil, nil, true, FLOW_MAPPING_STYLE),
		NewScalarEvent(nil, nil, []byte("a"), true, false, PLAIN_SCALAR_STYLE),
		NewScalarEvent(nil, nil, []byte("1"), true, false, PLAIN_SCALAR_STYLE),
		NewScalarEvent(nil, nil, []byte("b"), true, false, PLAIN_SCALAR_STYLE),
		NewScalarEvent(nil, nil, []byte("2"), true, false, PLAIN_SCALAR_STYLE),
		NewMappingEndEvent(),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)
	assert.Truef(t, strings.Contains(output, "{"), "emitEvents() output = %q, should contain '{'", output)
	assert.Truef(t, strings.Contains(output, "}"), "emitEvents() output = %q, should contain '}'", output)
}

func TestEmitterExplicitDocument(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, false),
		NewScalarEvent(nil, nil, []byte("value"), true, false, PLAIN_SCALAR_STYLE),
		NewDocumentEndEvent(false),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)
	assert.Truef(t, strings.Contains(output, "---"), "emitEvents() output = %q, should contain '---'", output)
	assert.Truef(t, strings.Contains(output, "..."), "emitEvents() output = %q, should contain '...'", output)
}

func TestEmitterAnchorAndAlias(t *testing.T) {
	anchor := []byte("myanchor")
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewSequenceStartEvent(nil, nil, true, BLOCK_SEQUENCE_STYLE),
		NewScalarEvent(anchor, nil, []byte("value"), true, false, PLAIN_SCALAR_STYLE),
		NewAliasEvent(anchor),
		NewSequenceEndEvent(),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)
	assert.Truef(t, strings.Contains(output, "&myanchor"), "emitEvents() output = %q, should contain '&myanchor'", output)
	assert.Truef(t, strings.Contains(output, "*myanchor"), "emitEvents() output = %q, should contain '*myanchor'", output)
}

func TestEmitterTag(t *testing.T) {
	tag := []byte("tag:yaml.org,2002:str")
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewScalarEvent(nil, tag, []byte("value"), false, false, PLAIN_SCALAR_STYLE),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)

	assert.Truef(t, strings.Contains(output, "!!str"), "emitEvents() output should contain '!!str', got %q", output)
}

func TestEmitterSingleQuotedScalar(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewScalarEvent(nil, nil, []byte("quoted value"), true, false, SINGLE_QUOTED_SCALAR_STYLE),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)

	assert.Truef(t, strings.Contains(output, "'"), "emitEvents() output should contain single quotes, got %q", output)
}

func TestEmitterDoubleQuotedScalar(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewScalarEvent(nil, nil, []byte("quoted value"), true, false, DOUBLE_QUOTED_SCALAR_STYLE),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)

	assert.Truef(t, strings.Contains(output, "\""), "emitEvents() output should contain double quotes, got %q", output)
}

func TestEmitterLiteralScalar(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewMappingStartEvent(nil, nil, true, BLOCK_MAPPING_STYLE),
		NewScalarEvent(nil, nil, []byte("key"), true, false, PLAIN_SCALAR_STYLE),
		NewScalarEvent(nil, nil, []byte("line1\nline2\n"), true, false, LITERAL_SCALAR_STYLE),
		NewMappingEndEvent(),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)

	assert.Truef(t, strings.Contains(output, "|"), "emitEvents() output should contain '|' for literal scalar, got %q", output)
}

func TestEmitterFoldedScalar(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewMappingStartEvent(nil, nil, true, BLOCK_MAPPING_STYLE),
		NewScalarEvent(nil, nil, []byte("key"), true, false, PLAIN_SCALAR_STYLE),
		NewScalarEvent(nil, nil, []byte("folded text\n"), true, false, FOLDED_SCALAR_STYLE),
		NewMappingEndEvent(),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)

	assert.Truef(t, strings.Contains(output, ">"), "emitEvents() output should contain '>' for folded scalar, got %q", output)
}

func TestEmitterCanonicalMode(t *testing.T) {
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)
	emitter.SetCanonical(true)

	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, false),
		NewScalarEvent(nil, nil, []byte("value"), true, false, PLAIN_SCALAR_STYLE),
		NewDocumentEndEvent(false),
		NewStreamEndEvent(),
	}

	for i := range events {
		err := emitter.Emit(&events[i])
		assert.NoErrorf(t, err, "Emit() error: %v", err)
	}

	result := string(output)
	assert.Truef(t, strings.Contains(result, "---"), "Canonical mode output should contain '---', got %q", result)
}

func TestEmitterCustomIndent(t *testing.T) {
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)
	emitter.SetIndent(4)

	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewMappingStartEvent(nil, nil, true, BLOCK_MAPPING_STYLE),
		NewScalarEvent(nil, nil, []byte("key"), true, false, PLAIN_SCALAR_STYLE),
		NewMappingStartEvent(nil, nil, true, BLOCK_MAPPING_STYLE),
		NewScalarEvent(nil, nil, []byte("nested"), true, false, PLAIN_SCALAR_STYLE),
		NewScalarEvent(nil, nil, []byte("value"), true, false, PLAIN_SCALAR_STYLE),
		NewMappingEndEvent(),
		NewMappingEndEvent(),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	for i := range events {
		err := emitter.Emit(&events[i])
		assert.NoErrorf(t, err, "Emit() error: %v", err)
	}

	result := string(output)
	assert.Equalf(t, emitter.BestIndent, 4, "BestIndent = %d, want 4", emitter.BestIndent)
	assert.Truef(t, strings.Contains(result, "key"), "Output should contain 'key', got %q", result)
}

func TestEmitterCustomWidth(t *testing.T) {
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)
	emitter.SetWidth(40)

	assert.Equalf(t, emitter.best_width, 40, "best_width = %d, want 40", emitter.best_width)

	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewScalarEvent(nil, nil, []byte("short"), true, false, PLAIN_SCALAR_STYLE),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	for i := range events {
		err := emitter.Emit(&events[i])
		assert.NoErrorf(t, err, "Emit() error: %v", err)
	}
}

func TestEmitterUnicodeMode(t *testing.T) {
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)
	emitter.SetUnicode(true)

	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewScalarEvent(nil, nil, []byte("unicode: \u00e9"), true, false, PLAIN_SCALAR_STYLE),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	for i := range events {
		err := emitter.Emit(&events[i])
		assert.NoErrorf(t, err, "Emit() error: %v", err)
	}

	result := string(output)
	assert.Truef(t, strings.Contains(result, "unicode"), "Output should contain 'unicode', got %q", result)
}

func TestEmitterMultipleDocuments(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, false),
		NewScalarEvent(nil, nil, []byte("doc1"), true, false, PLAIN_SCALAR_STYLE),
		NewDocumentEndEvent(false),
		NewDocumentStartEvent(nil, nil, false),
		NewScalarEvent(nil, nil, []byte("doc2"), true, false, PLAIN_SCALAR_STYLE),
		NewDocumentEndEvent(false),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)

	docStartCount := strings.Count(output, "---")
	assert.Truef(t, docStartCount >= 2, "Output should contain at least 2 '---', found %d in %q", docStartCount, output)
}

func TestEmitterNestedStructures(t *testing.T) {
	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewMappingStartEvent(nil, nil, true, BLOCK_MAPPING_STYLE),
		NewScalarEvent(nil, nil, []byte("parent"), true, false, PLAIN_SCALAR_STYLE),
		NewSequenceStartEvent(nil, nil, true, BLOCK_SEQUENCE_STYLE),
		NewMappingStartEvent(nil, nil, true, BLOCK_MAPPING_STYLE),
		NewScalarEvent(nil, nil, []byte("child"), true, false, PLAIN_SCALAR_STYLE),
		NewScalarEvent(nil, nil, []byte("value"), true, false, PLAIN_SCALAR_STYLE),
		NewMappingEndEvent(),
		NewSequenceEndEvent(),
		NewMappingEndEvent(),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	output, err := emitEvents(events)
	assert.NoErrorf(t, err, "emitEvents() error: %v", err)

	assert.Truef(t, strings.Contains(output, "parent"), "emitEvents() output = %q, should contain 'parent'", output)
	assert.Truef(t, strings.Contains(output, "child"), "emitEvents() output = %q, should contain 'child'", output)
}

func TestEmitterRoundTrip(t *testing.T) {
	input := "key: value\nlist:\n  - item1\n  - item2"

	parser := NewParser()
	parser.SetInputString([]byte(input))

	var events []Event
	for {
		var event Event
		if !parser.Parse(&event) {
			break
		}
		events = append(events, event)
		if event.Type == STREAM_END_EVENT {
			break
		}
	}

	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)

	for i := range events {
		err := emitter.Emit(&events[i])
		assert.NoErrorf(t, err, "Emit() error: %v", err)
	}

	result := string(output)
	assert.Truef(t, strings.Contains(result, "key"), "emitEvents() output = %q, should contain 'key'", output)
	assert.Truef(t, strings.Contains(result, "value"), "emitEvents() output = %q, should contain 'value'", output)
	assert.Truef(t, strings.Contains(result, "item1"), "emitEvents() output = %q, should contain 'item1'", output)
}

func TestEmitterWriter(t *testing.T) {
	emitter := NewEmitter()
	var buf bytes.Buffer
	emitter.SetOutputWriter(&buf)

	events := []Event{
		NewStreamStartEvent(UTF8_ENCODING),
		NewDocumentStartEvent(nil, nil, true),
		NewScalarEvent(nil, nil, []byte("test"), true, false, PLAIN_SCALAR_STYLE),
		NewDocumentEndEvent(true),
		NewStreamEndEvent(),
	}

	for i := range events {
		err := emitter.Emit(&events[i])
		assert.NoErrorf(t, err, "Emit() error: %v", err)
	}

	result := buf.String()
	assert.Truef(t, strings.Contains(result, "test"), "emitEvents() output = %q, should contain 'test'", result)
}
