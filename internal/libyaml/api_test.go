// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"bytes"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestNewParser(t *testing.T) {
	parser := NewParser()

	assert.NotNilf(t, parser.raw_buffer, "NewParser() should initialize raw_buffer")
	assert.Equalf(t, cap(parser.raw_buffer), input_raw_buffer_size, "NewParser() raw_buffer capacity = %d, want %d", cap(parser.raw_buffer), input_raw_buffer_size)

	assert.NotNilf(t, parser.buffer, "NewParser() should initialize buffer")
	assert.Equalf(t, cap(parser.buffer), input_buffer_size, "NewParser() buffer capacity = %d, want %d", cap(parser.buffer), input_buffer_size)
}

func TestParserDelete(t *testing.T) {
	parser := NewParser()
	parser.SetInputString([]byte("test"))

	parser.Delete()

	assert.Equalf(t, len(parser.input), 0, "Parser.Delete() should clear input")
	assert.Equalf(t, len(parser.buffer), 0, "Parser.Delete() should clear buffer")
}

func TestParserSetInputString(t *testing.T) {
	parser := NewParser()
	input := []byte("key: value")

	parser.SetInputString(input)

	assert.Equalf(t, bytes.Equal(parser.input, input), true, "SetInputString() input = %q, want %q", parser.input, input)
	assert.Equalf(t, parser.input_pos, 0, "SetInputString() input_pos = %d, want 0", parser.input_pos)
	assert.NotNilf(t, parser.read_handler, "SetInputString() should set read_handler")
}

func TestParserSetInputStringPanic(t *testing.T) {
	parser := NewParser()
	parser.SetInputString([]byte("first"))

	assert.PanicMatchesf(t, "must set the input source only once", func() {
		parser.SetInputString([]byte("second"))
	}, "Setting input twice should panic")
}

func TestParserSetInputReader(t *testing.T) {
	parser := NewParser()
	reader := strings.NewReader("key: value")

	parser.SetInputReader(reader)

	assert.NotNilf(t, parser.input_reader, "SetInputReader() should set input_reader")
	assert.NotNilf(t, parser.read_handler, "SetInputReader() should set read_handler")
}

func TestParserSetInputReaderPanic(t *testing.T) {
	parser := NewParser()
	parser.SetInputReader(strings.NewReader("first"))

	assert.PanicMatchesf(t, "must set the input source only once", func() {
		parser.SetInputReader(strings.NewReader("second"))
	}, "Setting input twice should panic")
}

func TestParserSetEncoding(t *testing.T) {
	parser := NewParser()

	parser.SetEncoding(UTF8_ENCODING)

	assert.Equalf(t, parser.encoding, UTF8_ENCODING, "SetEncoding() encoding = %v, want %v", parser.encoding, UTF8_ENCODING)
}

func TestParserSetEncodingPanic(t *testing.T) {
	parser := NewParser()
	parser.SetEncoding(UTF8_ENCODING)

	assert.PanicMatchesf(t, "must set the encoding only once", func() {
		parser.SetEncoding(UTF16LE_ENCODING)
	}, "Setting encoding twice should panic")
}

func TestNewEmitter(t *testing.T) {
	emitter := NewEmitter()

	assert.NotNilf(t, emitter.buffer, "NewEmitter() should initialize buffer")
	assert.Equalf(t, len(emitter.buffer), output_buffer_size, "NewEmitter() buffer length = %d, want %d", len(emitter.buffer), output_buffer_size)
	assert.NotNilf(t, emitter.raw_buffer, "NewEmitter() should initialize raw_buffer")
	assert.NotNilf(t, emitter.states, "NewEmitter() should initialize states")
	assert.NotNilf(t, emitter.events, "NewEmitter() should initialize events")
	assert.Equalf(t, emitter.best_width, -1, "NewEmitter() best_width = %d, want -1", emitter.best_width)
}

func TestEmitterDelete(t *testing.T) {
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)

	emitter.Delete()

	assert.IsNilf(t, emitter.output_buffer, "Emitter.Delete() should clear output_buffer")
	assert.Equalf(t, len(emitter.buffer), 0, "Emitter.Delete() should clear buffer")
}

func TestEmitterSetOutputString(t *testing.T) {
	emitter := NewEmitter()
	var output []byte

	emitter.SetOutputString(&output)

	assert.Equalf(t, emitter.output_buffer, &output, "SetOutputString() should set output_buffer")
	assert.NotNilf(t, emitter.write_handler, "SetOutputString() should set write_handler")
}

func TestEmitterSetOutputStringPanic(t *testing.T) {
	emitter := NewEmitter()
	var output1, output2 []byte
	emitter.SetOutputString(&output1)

	assert.PanicMatchesf(t, "must set the output target only once", func() {
		emitter.SetOutputString(&output2)
	}, "Setting output twice should panic")
}

func TestEmitterSetOutputWriter(t *testing.T) {
	emitter := NewEmitter()
	var buf bytes.Buffer

	emitter.SetOutputWriter(&buf)

	assert.NotNilf(t, emitter.output_writer, "SetOutputWriter() should set output_writer")
	assert.NotNilf(t, emitter.write_handler, "SetOutputWriter() should set write_handler")
}

func TestEmitterSetOutputWriterPanic(t *testing.T) {
	emitter := NewEmitter()
	var buf1, buf2 bytes.Buffer
	emitter.SetOutputWriter(&buf1)

	assert.PanicMatchesf(t, "must set the output target only once", func() {
		emitter.SetOutputWriter(&buf2)
	}, "Setting output twice should panic")
}

func TestEmitterSetEncodingPanic(t *testing.T) {
	emitter := NewEmitter()
	emitter.SetEncoding(UTF8_ENCODING)

	assert.PanicMatchesf(t, "must set the output encoding only once", func() {
		emitter.SetEncoding(UTF16LE_ENCODING)
	}, "Setting encoding twice should panic")
}

func TestEmitterSetCanonical(t *testing.T) {
	emitter := NewEmitter()

	emitter.SetCanonical(true)

	assert.Truef(t, emitter.canonical, "SetCanonical(true) should set canonical to true")

	emitter.SetCanonical(false)

	assert.Falsef(t, emitter.canonical, "SetCanonical(false) should set canonical to false")
}

func TestEmitterSetIndent(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{2, 2},
		{5, 5},
		{9, 9},
		{1, 2},
		{10, 2},
		{-1, 2},
	}

	for _, tt := range tests {
		emitter := NewEmitter()
		emitter.SetIndent(tt.input)

		assert.Equalf(t, emitter.BestIndent, tt.expected, "SetIndent(%d) BestIndent = %d, want %d", tt.input, emitter.BestIndent, tt.expected)
	}
}

func TestEmitterSetWidth(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{80, 80},
		{100, 100},
		{-1, -1},
		{-10, -1},
	}

	for _, tt := range tests {
		emitter := NewEmitter()
		emitter.SetWidth(tt.input)

		assert.Equalf(t, emitter.best_width, tt.expected, "SetWidth(%d) best_width = %d, want %d", tt.input, emitter.best_width, tt.expected)
	}
}

func TestEmitterSetUnicode(t *testing.T) {
	emitter := NewEmitter()

	emitter.SetUnicode(true)

	assert.Truef(t, emitter.unicode, "SetUnicode(true) should set unicode to true")

	emitter.SetUnicode(false)

	assert.Falsef(t, emitter.unicode, "SetUnicode(false) should set unicode to false")
}

func TestEmitterSetLineBreak(t *testing.T) {
	emitter := NewEmitter()

	emitter.SetLineBreak(LN_BREAK)

	assert.Equalf(t, emitter.line_break, LN_BREAK, "SetLineBreak(LN_BREAK) line_break = %v, want %v", emitter.line_break, LN_BREAK)
}

func TestNewStreamStartEvent(t *testing.T) {
	event := NewStreamStartEvent(UTF8_ENCODING)

	assert.Equalf(t, event.Type, STREAM_START_EVENT, "NewStreamStartEvent() Type = %v, want %v", event.Type, STREAM_START_EVENT)
	assert.Equalf(t, event.encoding, UTF8_ENCODING, "NewStreamStartEvent() encoding = %v, want %v", event.encoding, UTF8_ENCODING)
}

func TestNewStreamEndEvent(t *testing.T) {
	event := NewStreamEndEvent()

	assert.Equalf(t, event.Type, STREAM_END_EVENT, "NewStreamEndEvent() Type = %v, want %v", event.Type, STREAM_END_EVENT)
}

func TestNewDocumentStartEvent(t *testing.T) {
	vd := &VersionDirective{major: 1, minor: 2}
	td := []TagDirective{{handle: []byte("!"), prefix: []byte("!")}}

	event := NewDocumentStartEvent(vd, td, true)

	assert.Equalf(t, event.Type, DOCUMENT_START_EVENT, "NewDocumentStartEvent() Type = %v, want %v", event.Type, DOCUMENT_START_EVENT)
	assert.Equalf(t, event.version_directive, vd, "NewDocumentStartEvent() version_directive = %v, want %v", event.version_directive, vd)
	assert.Equalf(t, len(event.tag_directives), 1, "NewDocumentStartEvent() tag_directives length = %d, want 1", len(event.tag_directives))
	assert.Truef(t, event.Implicit, "NewDocumentStartEvent() Implicit should be true")
}

func TestNewDocumentEndEvent(t *testing.T) {
	event := NewDocumentEndEvent(false)

	assert.Equalf(t, event.Type, DOCUMENT_END_EVENT, "NewDocumentEndEvent() Type = %v, want %v", event.Type, DOCUMENT_END_EVENT)
	assert.Falsef(t, event.Implicit, "NewDocumentEndEvent() Implicit should be false")
}

func TestNewAliasEvent(t *testing.T) {
	anchor := []byte("myanchor")
	event := NewAliasEvent(anchor)

	assert.Equalf(t, event.Type, ALIAS_EVENT, "NewAliasEvent() Type = %v, want %v", event.Type, ALIAS_EVENT)
	assert.Equalf(t, bytes.Equal(event.Anchor, anchor), true, "NewAliasEvent() Anchor = %q, want %q", event.Anchor, anchor)
}

func TestNewScalarEvent(t *testing.T) {
	anchor := []byte("anchor")
	tag := []byte("tag")
	value := []byte("value")

	event := NewScalarEvent(anchor, tag, value, true, false, PLAIN_SCALAR_STYLE)

	assert.Equalf(t, event.Type, SCALAR_EVENT, "NewScalarEvent() Type = %v, want %v", event.Type, SCALAR_EVENT)
	assert.Equalf(t, bytes.Equal(event.Anchor, anchor), true, "NewScalarEvent() Anchor = %q, want %q", event.Anchor, anchor)
	assert.Equalf(t, bytes.Equal(event.Tag, tag), true, "NewScalarEvent() Tag = %q, want %q", event.Tag, tag)
	assert.Equalf(t, bytes.Equal(event.Value, value), true, "NewScalarEvent() Value = %q, want %q", event.Value, value)
	assert.Truef(t, event.Implicit, "NewScalarEvent() Implicit should be true")
	assert.Falsef(t, event.quoted_implicit, "NewScalarEvent() quoted_implicit should be false")
	assert.Equalf(t, event.ScalarStyle(), PLAIN_SCALAR_STYLE, "NewScalarEvent() Style = %v, want %v", event.Style, PLAIN_SCALAR_STYLE)
}

func TestNewSequenceStartEvent(t *testing.T) {
	anchor := []byte("anchor")
	tag := []byte("tag")

	event := NewSequenceStartEvent(anchor, tag, true, BLOCK_SEQUENCE_STYLE)

	assert.Equalf(t, event.Type, SEQUENCE_START_EVENT, "NewSequenceStartEvent() Type = %v, want %v", event.Type, SEQUENCE_START_EVENT)
	assert.Equalf(t, bytes.Equal(event.Anchor, anchor), true, "NewSequenceStartEvent() Anchor = %q, want %q", event.Anchor, anchor)
	assert.Equalf(t, bytes.Equal(event.Tag, tag), true, "NewSequenceStartEvent() Tag = %q, want %q", event.Tag, tag)
	assert.Truef(t, event.Implicit, "NewSequenceStartEvent() Implicit should be true")
	assert.Equalf(t, event.SequenceStyle(), BLOCK_SEQUENCE_STYLE, "NewSequenceStartEvent() Style = %v, want %v", event.Style, BLOCK_SEQUENCE_STYLE)
}

func TestNewSequenceEndEvent(t *testing.T) {
	event := NewSequenceEndEvent()

	assert.Equalf(t, event.Type, SEQUENCE_END_EVENT, "NewSequenceEndEvent() Type = %v, want %v", event.Type, SEQUENCE_END_EVENT)
}

func TestNewMappingStartEvent(t *testing.T) {
	anchor := []byte("anchor")
	tag := []byte("tag")

	event := NewMappingStartEvent(anchor, tag, false, FLOW_MAPPING_STYLE)

	assert.Equalf(t, event.Type, MAPPING_START_EVENT, "NewMappingStartEvent() Type = %v, want %v", event.Type, MAPPING_START_EVENT)
	assert.Equalf(t, bytes.Equal(event.Anchor, anchor), true, "NewMappingStartEvent() Anchor = %q, want %q", event.Anchor, anchor)
	assert.Equalf(t, bytes.Equal(event.Tag, tag), true, "NewMappingStartEvent() Tag = %q, want %q", event.Tag, tag)
	assert.Falsef(t, event.Implicit, "NewMappingStartEvent() Implicit should be false")
	assert.Equalf(t, event.MappingStyle(), FLOW_MAPPING_STYLE, "NewMappingStartEvent() Style = %v, want %v", event.Style, FLOW_MAPPING_STYLE)
}

func TestNewMappingEndEvent(t *testing.T) {
	event := NewMappingEndEvent()

	assert.Equalf(t, event.Type, MAPPING_END_EVENT, "NewMappingEndEvent() Type = %v, want %v", event.Type, MAPPING_END_EVENT)
}

func TestEventDelete(t *testing.T) {
	event := NewScalarEvent([]byte("a"), []byte("t"), []byte("v"), true, false, PLAIN_SCALAR_STYLE)

	event.Delete()

	assert.Equalf(t, event.Type, NO_EVENT, "Event.Delete() should reset Type to NO_EVENT")
	assert.Equalf(t, len(event.Anchor), 0, "Event.Delete() should clear Anchor")
}

func TestParserInsertToken(t *testing.T) {
	parser := NewParser()
	token := Token{Type: SCALAR_TOKEN, Value: []byte("test")}

	parser.insertToken(-1, &token)

	assert.Equalf(t, len(parser.tokens), 1, "insertToken() tokens length = %d, want 1", len(parser.tokens))
	assert.Equalf(t, parser.tokens[0].Type, SCALAR_TOKEN, "insertToken() token type = %v, want %v", parser.tokens[0].Type, SCALAR_TOKEN)
}

func TestParserInsertTokenAtPosition(t *testing.T) {
	parser := NewParser()
	token1 := Token{Type: KEY_TOKEN}
	token2 := Token{Type: VALUE_TOKEN}
	token3 := Token{Type: SCALAR_TOKEN}

	parser.insertToken(-1, &token1)
	parser.insertToken(-1, &token3)
	parser.insertToken(1, &token2)

	assert.Equalf(t, len(parser.tokens), 3, "insertToken() tokens length = %d, want 3", len(parser.tokens))
	assert.Equalf(t, parser.tokens[0].Type, KEY_TOKEN, "token[0] type = %v, want KEY_TOKEN", parser.tokens[0].Type)
	assert.Equalf(t, parser.tokens[1].Type, VALUE_TOKEN, "token[1] type = %v, want VALUE_TOKEN", parser.tokens[1].Type)
	assert.Equalf(t, parser.tokens[2].Type, SCALAR_TOKEN, "token[2] type = %v, want SCALAR_TOKEN", parser.tokens[2].Type)
}
