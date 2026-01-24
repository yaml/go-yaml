// Copyright 2006-2010 Kirill Simonov
// Copyright 2011-2019 Canonical Ltd
// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0 AND MIT

// High-level API helpers for parser and emitter initialization and
// configuration.
// Provides convenience functions for token insertion and stream management.

package libyaml

import (
	"io"
)

func (parser *Parser) insertToken(pos int, token *Token) {
	// fmt.Println("yaml_insert_token", "pos:", pos, "typ:", token.typ, "head:", parser.tokens_head, "len:", len(parser.tokens))

	// Check if we can move the queue at the beginning of the buffer.
	if parser.tokens_head > 0 && len(parser.tokens) == cap(parser.tokens) {
		if parser.tokens_head != len(parser.tokens) {
			copy(parser.tokens, parser.tokens[parser.tokens_head:])
		}
		parser.tokens = parser.tokens[:len(parser.tokens)-parser.tokens_head]
		parser.tokens_head = 0
	}
	parser.tokens = append(parser.tokens, *token)
	if pos < 0 {
		return
	}
	copy(parser.tokens[parser.tokens_head+pos+1:], parser.tokens[parser.tokens_head+pos:])
	parser.tokens[parser.tokens_head+pos] = *token
}

// NewParser creates a new parser object.
func NewParser() Parser {
	return Parser{
		raw_buffer: make([]byte, 0, input_raw_buffer_size),
		buffer:     make([]byte, 0, input_buffer_size),
	}
}

// Delete a parser object.
func (parser *Parser) Delete() {
	*parser = Parser{}
}

// String read handler.
func yamlStringReadHandler(parser *Parser, buffer []byte) (n int, err error) {
	if parser.input_pos == len(parser.input) {
		return 0, io.EOF
	}
	n = copy(buffer, parser.input[parser.input_pos:])
	parser.input_pos += n
	return n, nil
}

// Reader read handler.
func yamlReaderReadHandler(parser *Parser, buffer []byte) (n int, err error) {
	return parser.input_reader.Read(buffer)
}

// SetInputString sets a string input.
func (parser *Parser) SetInputString(input []byte) {
	if parser.read_handler != nil {
		panic("must set the input source only once")
	}
	parser.read_handler = yamlStringReadHandler
	parser.input = input
	parser.input_pos = 0
}

// SetInputReader sets a file input.
func (parser *Parser) SetInputReader(r io.Reader) {
	if parser.read_handler != nil {
		panic("must set the input source only once")
	}
	parser.read_handler = yamlReaderReadHandler
	parser.input_reader = r
}

// SetEncoding sets the source encoding.
func (parser *Parser) SetEncoding(encoding Encoding) {
	if parser.encoding != ANY_ENCODING {
		panic("must set the encoding only once")
	}
	parser.encoding = encoding
}

// GetPendingComments returns the parser's comment queue for CLI access.
func (parser *Parser) GetPendingComments() []Comment {
	return parser.comments
}

// GetCommentsHead returns the current position in the comment queue.
func (parser *Parser) GetCommentsHead() int {
	return parser.comments_head
}

// NewEmitter creates a new emitter object.
func NewEmitter() Emitter {
	return Emitter{
		buffer:     make([]byte, output_buffer_size),
		states:     make([]EmitterState, 0, initial_stack_size),
		events:     make([]Event, 0, initial_queue_size),
		best_width: -1,
	}
}

// Delete an emitter object.
func (emitter *Emitter) Delete() {
	*emitter = Emitter{}
}

// String write handler.
func yamlStringWriteHandler(emitter *Emitter, buffer []byte) error {
	*emitter.output_buffer = append(*emitter.output_buffer, buffer...)
	return nil
}

// yamlWriterWriteHandler uses emitter.output_writer to write the
// emitted text.
func yamlWriterWriteHandler(emitter *Emitter, buffer []byte) error {
	_, err := emitter.output_writer.Write(buffer)
	return err
}

// SetOutputString sets a string output.
func (emitter *Emitter) SetOutputString(output_buffer *[]byte) {
	if emitter.write_handler != nil {
		panic("must set the output target only once")
	}
	emitter.write_handler = yamlStringWriteHandler
	emitter.output_buffer = output_buffer
}

// SetOutputWriter sets a file output.
func (emitter *Emitter) SetOutputWriter(w io.Writer) {
	if emitter.write_handler != nil {
		panic("must set the output target only once")
	}
	emitter.write_handler = yamlWriterWriteHandler
	emitter.output_writer = w
}

// SetEncoding sets the output encoding.
func (emitter *Emitter) SetEncoding(encoding Encoding) {
	if emitter.encoding != ANY_ENCODING {
		panic("must set the output encoding only once")
	}
	emitter.encoding = encoding
}

// SetCanonical sets the canonical output style.
func (emitter *Emitter) SetCanonical(canonical bool) {
	emitter.canonical = canonical
}

// SetIndent sets the indentation increment.
func (emitter *Emitter) SetIndent(indent int) {
	if indent < 2 || indent > 9 {
		indent = 2
	}
	emitter.BestIndent = indent
}

// SetWidth sets the preferred line width.
func (emitter *Emitter) SetWidth(width int) {
	if width < 0 {
		width = -1
	}
	emitter.best_width = width
}

// SetUnicode sets if unescaped non-ASCII characters are allowed.
func (emitter *Emitter) SetUnicode(unicode bool) {
	emitter.unicode = unicode
}

// SetLineBreak sets the preferred line break character.
func (emitter *Emitter) SetLineBreak(line_break LineBreak) {
	emitter.line_break = line_break
}

// NewStreamStartEvent creates a new STREAM-START event.
func NewStreamStartEvent(encoding Encoding) Event {
	return Event{
		Type:     STREAM_START_EVENT,
		encoding: encoding,
	}
}

// NewStreamEndEvent creates a new STREAM-END event.
func NewStreamEndEvent() Event {
	return Event{
		Type: STREAM_END_EVENT,
	}
}

// NewDocumentStartEvent creates a new DOCUMENT-START event.
func NewDocumentStartEvent(version_directive *VersionDirective, tag_directives []TagDirective, implicit bool) Event {
	return Event{
		Type:             DOCUMENT_START_EVENT,
		versionDirective: version_directive,
		tagDirectives:    tag_directives,
		Implicit:         implicit,
	}
}

// NewDocumentEndEvent creates a new DOCUMENT-END event.
func NewDocumentEndEvent(implicit bool) Event {
	return Event{
		Type:     DOCUMENT_END_EVENT,
		Implicit: implicit,
	}
}

// NewAliasEvent creates a new ALIAS event.
func NewAliasEvent(anchor []byte) Event {
	return Event{
		Type:   ALIAS_EVENT,
		Anchor: anchor,
	}
}

// NewScalarEvent creates a new SCALAR event.
func NewScalarEvent(anchor, tag, value []byte, plain_implicit, quoted_implicit bool, style ScalarStyle) Event {
	return Event{
		Type:            SCALAR_EVENT,
		Anchor:          anchor,
		Tag:             tag,
		Value:           value,
		Implicit:        plain_implicit,
		quoted_implicit: quoted_implicit,
		Style:           Style(style),
	}
}

// NewSequenceStartEvent creates a new SEQUENCE-START event.
func NewSequenceStartEvent(anchor, tag []byte, implicit bool, style SequenceStyle) Event {
	return Event{
		Type:     SEQUENCE_START_EVENT,
		Anchor:   anchor,
		Tag:      tag,
		Implicit: implicit,
		Style:    Style(style),
	}
}

// NewSequenceEndEvent creates a new SEQUENCE-END event.
func NewSequenceEndEvent() Event {
	return Event{
		Type: SEQUENCE_END_EVENT,
	}
}

// NewMappingStartEvent creates a new MAPPING-START event.
func NewMappingStartEvent(anchor, tag []byte, implicit bool, style MappingStyle) Event {
	return Event{
		Type:     MAPPING_START_EVENT,
		Anchor:   anchor,
		Tag:      tag,
		Implicit: implicit,
		Style:    Style(style),
	}
}

// NewMappingEndEvent creates a new MAPPING-END event.
func NewMappingEndEvent() Event {
	return Event{
		Type: MAPPING_END_EVENT,
	}
}

// Delete an event object.
func (e *Event) Delete() {
	*e = Event{}
}
