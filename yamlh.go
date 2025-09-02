//
// Copyright (c) 2011-2019 Canonical Ltd
// Copyright (c) 2006-2010 Kirill Simonov
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package yaml

import (
	"fmt"
	"io"
)

// The version directive data.
type yamlVersionDirective struct {
	major int8 // The major version number.
	minor int8 // The minor version number.
}

// The tag directive data.
type yamlTagDirective struct {
	handle []byte // The tag handle.
	prefix []byte // The tag prefix.
}

type yamlEncoding int

// The stream encoding.
const (
	// Let the parser choose the encoding.
	yaml_ANY_ENCODING yamlEncoding = iota

	yaml_UTF8_ENCODING    // The default UTF-8 encoding.
	yaml_UTF16LE_ENCODING // The UTF-16-LE encoding with BOM.
	yaml_UTF16BE_ENCODING // The UTF-16-BE encoding with BOM.
)

type yamlLineBreak int

// Line break types.
const (
	// Let the parser choose the break type.
	yaml_ANY_BREAK yamlLineBreak = iota

	yaml_CR_BREAK   // Use CR for line breaks (Mac style).
	yaml_LN_BREAK   // Use LN for line breaks (Unix style).
	yaml_CRLN_BREAK // Use CR LN for line breaks (DOS style).
)

type yamlErrorType int

// Many bad things could happen with the parser and emitter.
const (
	// No error is produced.
	yaml_NO_ERROR yamlErrorType = iota

	yaml_MEMORY_ERROR   // Cannot allocate or reallocate a block of memory.
	yaml_READER_ERROR   // Cannot read or decode the input stream.
	yaml_SCANNER_ERROR  // Cannot scan the input stream.
	yaml_PARSER_ERROR   // Cannot parse the input stream.
	yaml_COMPOSER_ERROR // Cannot compose a YAML document.
	yaml_WRITER_ERROR   // Cannot write to the output stream.
	yaml_EMITTER_ERROR  // Cannot emit a YAML stream.
)

// The pointer position.
type yamlMark struct {
	index  int // The position index.
	line   int // The position line.
	column int // The position column.
}

// Node Styles

type yamlStyle int8

type yamlScalarStyle yamlStyle

// Scalar styles.
const (
	// Let the emitter choose the style.
	yaml_ANY_SCALAR_STYLE yamlScalarStyle = 0

	yaml_PLAIN_SCALAR_STYLE         yamlScalarStyle = 1 << iota // The plain scalar style.
	yaml_SINGLE_QUOTED_SCALAR_STYLE                             // The single-quoted scalar style.
	yaml_DOUBLE_QUOTED_SCALAR_STYLE                             // The double-quoted scalar style.
	yaml_LITERAL_SCALAR_STYLE                                   // The literal scalar style.
	yaml_FOLDED_SCALAR_STYLE                                    // The folded scalar style.
)

type yamlSequenceStyle yamlStyle

// Sequence styles.
const (
	// Let the emitter choose the style.
	yaml_ANY_SEQUENCE_STYLE yamlSequenceStyle = iota

	yaml_BLOCK_SEQUENCE_STYLE // The block sequence style.
	yaml_FLOW_SEQUENCE_STYLE  // The flow sequence style.
)

type yamlMappingStyle yamlStyle

// Mapping styles.
const (
	// Let the emitter choose the style.
	yaml_ANY_MAPPING_STYLE yamlMappingStyle = iota

	yaml_BLOCK_MAPPING_STYLE // The block mapping style.
	yaml_FLOW_MAPPING_STYLE  // The flow mapping style.
)

// Tokens

type yamlTokenType int

// Token types.
const (
	// An empty token.
	yaml_NO_TOKEN yamlTokenType = iota

	yaml_STREAM_START_TOKEN // A STREAM-START token.
	yaml_STREAM_END_TOKEN   // A STREAM-END token.

	yaml_VERSION_DIRECTIVE_TOKEN // A VERSION-DIRECTIVE token.
	yaml_TAG_DIRECTIVE_TOKEN     // A TAG-DIRECTIVE token.
	yaml_DOCUMENT_START_TOKEN    // A DOCUMENT-START token.
	yaml_DOCUMENT_END_TOKEN      // A DOCUMENT-END token.

	yaml_BLOCK_SEQUENCE_START_TOKEN // A BLOCK-SEQUENCE-START token.
	yaml_BLOCK_MAPPING_START_TOKEN  // A BLOCK-SEQUENCE-END token.
	yaml_BLOCK_END_TOKEN            // A BLOCK-END token.

	yaml_FLOW_SEQUENCE_START_TOKEN // A FLOW-SEQUENCE-START token.
	yaml_FLOW_SEQUENCE_END_TOKEN   // A FLOW-SEQUENCE-END token.
	yaml_FLOW_MAPPING_START_TOKEN  // A FLOW-MAPPING-START token.
	yaml_FLOW_MAPPING_END_TOKEN    // A FLOW-MAPPING-END token.

	yaml_BLOCK_ENTRY_TOKEN // A BLOCK-ENTRY token.
	yaml_FLOW_ENTRY_TOKEN  // A FLOW-ENTRY token.
	yaml_KEY_TOKEN         // A KEY token.
	yaml_VALUE_TOKEN       // A VALUE token.

	yaml_ALIAS_TOKEN  // An ALIAS token.
	yaml_ANCHOR_TOKEN // An ANCHOR token.
	yaml_TAG_TOKEN    // A TAG token.
	yaml_SCALAR_TOKEN // A SCALAR token.
)

func (tt yamlTokenType) String() string {
	switch tt {
	case yaml_NO_TOKEN:
		return "yaml_NO_TOKEN"
	case yaml_STREAM_START_TOKEN:
		return "yaml_STREAM_START_TOKEN"
	case yaml_STREAM_END_TOKEN:
		return "yaml_STREAM_END_TOKEN"
	case yaml_VERSION_DIRECTIVE_TOKEN:
		return "yaml_VERSION_DIRECTIVE_TOKEN"
	case yaml_TAG_DIRECTIVE_TOKEN:
		return "yaml_TAG_DIRECTIVE_TOKEN"
	case yaml_DOCUMENT_START_TOKEN:
		return "yaml_DOCUMENT_START_TOKEN"
	case yaml_DOCUMENT_END_TOKEN:
		return "yaml_DOCUMENT_END_TOKEN"
	case yaml_BLOCK_SEQUENCE_START_TOKEN:
		return "yaml_BLOCK_SEQUENCE_START_TOKEN"
	case yaml_BLOCK_MAPPING_START_TOKEN:
		return "yaml_BLOCK_MAPPING_START_TOKEN"
	case yaml_BLOCK_END_TOKEN:
		return "yaml_BLOCK_END_TOKEN"
	case yaml_FLOW_SEQUENCE_START_TOKEN:
		return "yaml_FLOW_SEQUENCE_START_TOKEN"
	case yaml_FLOW_SEQUENCE_END_TOKEN:
		return "yaml_FLOW_SEQUENCE_END_TOKEN"
	case yaml_FLOW_MAPPING_START_TOKEN:
		return "yaml_FLOW_MAPPING_START_TOKEN"
	case yaml_FLOW_MAPPING_END_TOKEN:
		return "yaml_FLOW_MAPPING_END_TOKEN"
	case yaml_BLOCK_ENTRY_TOKEN:
		return "yaml_BLOCK_ENTRY_TOKEN"
	case yaml_FLOW_ENTRY_TOKEN:
		return "yaml_FLOW_ENTRY_TOKEN"
	case yaml_KEY_TOKEN:
		return "yaml_KEY_TOKEN"
	case yaml_VALUE_TOKEN:
		return "yaml_VALUE_TOKEN"
	case yaml_ALIAS_TOKEN:
		return "yaml_ALIAS_TOKEN"
	case yaml_ANCHOR_TOKEN:
		return "yaml_ANCHOR_TOKEN"
	case yaml_TAG_TOKEN:
		return "yaml_TAG_TOKEN"
	case yaml_SCALAR_TOKEN:
		return "yaml_SCALAR_TOKEN"
	}
	return "<unknown token>"
}

// The token structure.
type yamlToken struct {
	// The token type.
	typ yamlTokenType

	// The start/end of the token.
	start_mark, end_mark yamlMark

	// The stream encoding (for yaml_STREAM_START_TOKEN).
	encoding yamlEncoding

	// The alias/anchor/scalar value or tag/tag directive handle
	// (for yaml_ALIAS_TOKEN, yaml_ANCHOR_TOKEN, yaml_SCALAR_TOKEN, yaml_TAG_TOKEN, yaml_TAG_DIRECTIVE_TOKEN).
	value []byte

	// The tag suffix (for yaml_TAG_TOKEN).
	suffix []byte

	// The tag directive prefix (for yaml_TAG_DIRECTIVE_TOKEN).
	prefix []byte

	// The scalar style (for yaml_SCALAR_TOKEN).
	style yamlScalarStyle

	// The version directive major/minor (for yaml_VERSION_DIRECTIVE_TOKEN).
	major, minor int8
}

// Events

type yamlEventType int8

// Event types.
const (
	// An empty event.
	yaml_NO_EVENT yamlEventType = iota

	yaml_STREAM_START_EVENT   // A STREAM-START event.
	yaml_STREAM_END_EVENT     // A STREAM-END event.
	yaml_DOCUMENT_START_EVENT // A DOCUMENT-START event.
	yaml_DOCUMENT_END_EVENT   // A DOCUMENT-END event.
	yaml_ALIAS_EVENT          // An ALIAS event.
	yaml_SCALAR_EVENT         // A SCALAR event.
	yaml_SEQUENCE_START_EVENT // A SEQUENCE-START event.
	yaml_SEQUENCE_END_EVENT   // A SEQUENCE-END event.
	yaml_MAPPING_START_EVENT  // A MAPPING-START event.
	yaml_MAPPING_END_EVENT    // A MAPPING-END event.
	yaml_TAIL_COMMENT_EVENT
)

var eventStrings = []string{
	yaml_NO_EVENT:             "none",
	yaml_STREAM_START_EVENT:   "stream start",
	yaml_STREAM_END_EVENT:     "stream end",
	yaml_DOCUMENT_START_EVENT: "document start",
	yaml_DOCUMENT_END_EVENT:   "document end",
	yaml_ALIAS_EVENT:          "alias",
	yaml_SCALAR_EVENT:         "scalar",
	yaml_SEQUENCE_START_EVENT: "sequence start",
	yaml_SEQUENCE_END_EVENT:   "sequence end",
	yaml_MAPPING_START_EVENT:  "mapping start",
	yaml_MAPPING_END_EVENT:    "mapping end",
	yaml_TAIL_COMMENT_EVENT:   "tail comment",
}

func (e yamlEventType) String() string {
	if e < 0 || int(e) >= len(eventStrings) {
		return fmt.Sprintf("unknown event %d", e)
	}
	return eventStrings[e]
}

// The event structure.
type yamlEvent struct {

	// The event type.
	typ yamlEventType

	// The start and end of the event.
	start_mark, end_mark yamlMark

	// The document encoding (for yaml_STREAM_START_EVENT).
	encoding yamlEncoding

	// The version directive (for yaml_DOCUMENT_START_EVENT).
	version_directive *yamlVersionDirective

	// The list of tag directives (for yaml_DOCUMENT_START_EVENT).
	tag_directives []yamlTagDirective

	// The comments
	head_comment []byte
	line_comment []byte
	foot_comment []byte
	tail_comment []byte

	// The anchor (for yaml_SCALAR_EVENT, yaml_SEQUENCE_START_EVENT, yaml_MAPPING_START_EVENT, yaml_ALIAS_EVENT).
	anchor []byte

	// The tag (for yaml_SCALAR_EVENT, yaml_SEQUENCE_START_EVENT, yaml_MAPPING_START_EVENT).
	tag []byte

	// The scalar value (for yaml_SCALAR_EVENT).
	value []byte

	// Is the document start/end indicator implicit, or the tag optional?
	// (for yaml_DOCUMENT_START_EVENT, yaml_DOCUMENT_END_EVENT, yaml_SEQUENCE_START_EVENT, yaml_MAPPING_START_EVENT, yaml_SCALAR_EVENT).
	implicit bool

	// Is the tag optional for any non-plain style? (for yaml_SCALAR_EVENT).
	quoted_implicit bool

	// The style (for yaml_SCALAR_EVENT, yaml_SEQUENCE_START_EVENT, yaml_MAPPING_START_EVENT).
	style yamlStyle
}

func (e *yamlEvent) scalarStyle() yamlScalarStyle     { return yamlScalarStyle(e.style) }
func (e *yamlEvent) sequenceStyle() yamlSequenceStyle { return yamlSequenceStyle(e.style) }
func (e *yamlEvent) mappingStyle() yamlMappingStyle   { return yamlMappingStyle(e.style) }

// Nodes

const (
	yaml_NULL_TAG      = "tag:yaml.org,2002:null"      // The tag !!null with the only possible value: null.
	yaml_BOOL_TAG      = "tag:yaml.org,2002:bool"      // The tag !!bool with the values: true and false.
	yaml_STR_TAG       = "tag:yaml.org,2002:str"       // The tag !!str for string values.
	yaml_INT_TAG       = "tag:yaml.org,2002:int"       // The tag !!int for integer values.
	yaml_FLOAT_TAG     = "tag:yaml.org,2002:float"     // The tag !!float for float values.
	yaml_TIMESTAMP_TAG = "tag:yaml.org,2002:timestamp" // The tag !!timestamp for date and time values.

	yaml_SEQ_TAG = "tag:yaml.org,2002:seq" // The tag !!seq is used to denote sequences.
	yaml_MAP_TAG = "tag:yaml.org,2002:map" // The tag !!map is used to denote mapping.

	// Not in original libyaml.
	yaml_BINARY_TAG = "tag:yaml.org,2002:binary"
	yaml_MERGE_TAG  = "tag:yaml.org,2002:merge"

	yaml_DEFAULT_SCALAR_TAG   = yaml_STR_TAG // The default scalar tag is !!str.
	yaml_DEFAULT_SEQUENCE_TAG = yaml_SEQ_TAG // The default sequence tag is !!seq.
	yaml_DEFAULT_MAPPING_TAG  = yaml_MAP_TAG // The default mapping tag is !!map.
)

type yamlNodeType int

// Node types.
const (
	// An empty node.
	yaml_NO_NODE yamlNodeType = iota

	yaml_SCALAR_NODE   // A scalar node.
	yaml_SEQUENCE_NODE // A sequence node.
	yaml_MAPPING_NODE  // A mapping node.
)

// An element of a sequence node.
type yamlNodeItem int

// An element of a mapping node.
type yamlNodePair struct {
	key   int // The key of the element.
	value int // The value of the element.
}

// The node structure.
type yamlNode struct {
	typ yamlNodeType // The node type.
	tag []byte       // The node tag.

	// The node data.

	// The scalar parameters (for yaml_SCALAR_NODE).
	scalar struct {
		value  []byte          // The scalar value.
		length int             // The length of the scalar value.
		style  yamlScalarStyle // The scalar style.
	}

	// The sequence parameters (for YAML_SEQUENCE_NODE).
	sequence struct {
		items_data []yamlNodeItem    // The stack of sequence items.
		style      yamlSequenceStyle // The sequence style.
	}

	// The mapping parameters (for yaml_MAPPING_NODE).
	mapping struct {
		pairs_data  []yamlNodePair   // The stack of mapping pairs (key, value).
		pairs_start *yamlNodePair    // The beginning of the stack.
		pairs_end   *yamlNodePair    // The end of the stack.
		pairs_top   *yamlNodePair    // The top of the stack.
		style       yamlMappingStyle // The mapping style.
	}

	start_mark yamlMark // The beginning of the node.
	end_mark   yamlMark // The end of the node.

}

// The document structure.
type yamlDocument struct {

	// The document nodes.
	nodes []yamlNode

	// The version directive.
	version_directive *yamlVersionDirective

	// The list of tag directives.
	tag_directives_data  []yamlTagDirective
	tag_directives_start int // The beginning of the tag directives list.
	tag_directives_end   int // The end of the tag directives list.

	start_implicit int // Is the document start indicator implicit?
	end_implicit   int // Is the document end indicator implicit?

	// The start/end of the document.
	start_mark, end_mark yamlMark
}

// The prototype of a read handler.
//
// The read handler is called when the parser needs to read more bytes from the
// source. The handler should write not more than size bytes to the buffer.
// The number of written bytes should be set to the size_read variable.
//
// [in,out]   data        A pointer to an application data specified by
//
//	yaml_parser_set_input().
//
// [out]      buffer      The buffer to write the data from the source.
// [in]       size        The size of the buffer.
// [out]      size_read   The actual number of bytes read from the source.
//
// On success, the handler should return 1.  If the handler failed,
// the returned value should be 0. On EOF, the handler should set the
// size_read to 0 and return 1.
type yamlReadHandler func(parser *yamlParser, buffer []byte) (n int, err error)

// This structure holds information about a potential simple key.
type yamlSimpleKey struct {
	possible     bool     // Is a simple key possible?
	required     bool     // Is a simple key required?
	token_number int      // The number of the token.
	mark         yamlMark // The position mark.
}

// The states of the parser.
type yamlParserState int

const (
	yaml_PARSE_STREAM_START_STATE yamlParserState = iota

	yaml_PARSE_IMPLICIT_DOCUMENT_START_STATE           // Expect the beginning of an implicit document.
	yaml_PARSE_DOCUMENT_START_STATE                    // Expect DOCUMENT-START.
	yaml_PARSE_DOCUMENT_CONTENT_STATE                  // Expect the content of a document.
	yaml_PARSE_DOCUMENT_END_STATE                      // Expect DOCUMENT-END.
	yaml_PARSE_BLOCK_NODE_STATE                        // Expect a block node.
	yaml_PARSE_BLOCK_NODE_OR_INDENTLESS_SEQUENCE_STATE // Expect a block node or indentless sequence.
	yaml_PARSE_FLOW_NODE_STATE                         // Expect a flow node.
	yaml_PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE        // Expect the first entry of a block sequence.
	yaml_PARSE_BLOCK_SEQUENCE_ENTRY_STATE              // Expect an entry of a block sequence.
	yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE         // Expect an entry of an indentless sequence.
	yaml_PARSE_BLOCK_MAPPING_FIRST_KEY_STATE           // Expect the first key of a block mapping.
	yaml_PARSE_BLOCK_MAPPING_KEY_STATE                 // Expect a block mapping key.
	yaml_PARSE_BLOCK_MAPPING_VALUE_STATE               // Expect a block mapping value.
	yaml_PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE         // Expect the first entry of a flow sequence.
	yaml_PARSE_FLOW_SEQUENCE_ENTRY_STATE               // Expect an entry of a flow sequence.
	yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE   // Expect a key of an ordered mapping.
	yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE // Expect a value of an ordered mapping.
	yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE   // Expect the and of an ordered mapping entry.
	yaml_PARSE_FLOW_MAPPING_FIRST_KEY_STATE            // Expect the first key of a flow mapping.
	yaml_PARSE_FLOW_MAPPING_KEY_STATE                  // Expect a key of a flow mapping.
	yaml_PARSE_FLOW_MAPPING_VALUE_STATE                // Expect a value of a flow mapping.
	yaml_PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE          // Expect an empty value of a flow mapping.
	yaml_PARSE_END_STATE                               // Expect nothing.
)

func (ps yamlParserState) String() string {
	switch ps {
	case yaml_PARSE_STREAM_START_STATE:
		return "yaml_PARSE_STREAM_START_STATE"
	case yaml_PARSE_IMPLICIT_DOCUMENT_START_STATE:
		return "yaml_PARSE_IMPLICIT_DOCUMENT_START_STATE"
	case yaml_PARSE_DOCUMENT_START_STATE:
		return "yaml_PARSE_DOCUMENT_START_STATE"
	case yaml_PARSE_DOCUMENT_CONTENT_STATE:
		return "yaml_PARSE_DOCUMENT_CONTENT_STATE"
	case yaml_PARSE_DOCUMENT_END_STATE:
		return "yaml_PARSE_DOCUMENT_END_STATE"
	case yaml_PARSE_BLOCK_NODE_STATE:
		return "yaml_PARSE_BLOCK_NODE_STATE"
	case yaml_PARSE_BLOCK_NODE_OR_INDENTLESS_SEQUENCE_STATE:
		return "yaml_PARSE_BLOCK_NODE_OR_INDENTLESS_SEQUENCE_STATE"
	case yaml_PARSE_FLOW_NODE_STATE:
		return "yaml_PARSE_FLOW_NODE_STATE"
	case yaml_PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE:
		return "yaml_PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE"
	case yaml_PARSE_BLOCK_SEQUENCE_ENTRY_STATE:
		return "yaml_PARSE_BLOCK_SEQUENCE_ENTRY_STATE"
	case yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE:
		return "yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE"
	case yaml_PARSE_BLOCK_MAPPING_FIRST_KEY_STATE:
		return "yaml_PARSE_BLOCK_MAPPING_FIRST_KEY_STATE"
	case yaml_PARSE_BLOCK_MAPPING_KEY_STATE:
		return "yaml_PARSE_BLOCK_MAPPING_KEY_STATE"
	case yaml_PARSE_BLOCK_MAPPING_VALUE_STATE:
		return "yaml_PARSE_BLOCK_MAPPING_VALUE_STATE"
	case yaml_PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE:
		return "yaml_PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE"
	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_STATE:
		return "yaml_PARSE_FLOW_SEQUENCE_ENTRY_STATE"
	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE:
		return "yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE"
	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE:
		return "yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE"
	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE:
		return "yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE"
	case yaml_PARSE_FLOW_MAPPING_FIRST_KEY_STATE:
		return "yaml_PARSE_FLOW_MAPPING_FIRST_KEY_STATE"
	case yaml_PARSE_FLOW_MAPPING_KEY_STATE:
		return "yaml_PARSE_FLOW_MAPPING_KEY_STATE"
	case yaml_PARSE_FLOW_MAPPING_VALUE_STATE:
		return "yaml_PARSE_FLOW_MAPPING_VALUE_STATE"
	case yaml_PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE:
		return "yaml_PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE"
	case yaml_PARSE_END_STATE:
		return "yaml_PARSE_END_STATE"
	}
	return "<unknown parser state>"
}

// This structure holds aliases data.
type yamlAliasData struct {
	anchor []byte   // The anchor.
	index  int      // The node id.
	mark   yamlMark // The anchor mark.
}

// The parser structure.
//
// All members are internal. Manage the structure using the
// yaml_parser_ family of functions.
type yamlParser struct {

	// Error handling

	error yamlErrorType // Error type.

	problem string // Error description.

	// The byte about which the problem occurred.
	problem_offset int
	problem_value  int
	problem_mark   yamlMark

	// The error context.
	context      string
	context_mark yamlMark

	// Reader stuff

	read_handler yamlReadHandler // Read handler.

	input_reader io.Reader // File input data.
	input        []byte    // String input data.
	input_pos    int

	eof bool // EOF flag

	buffer     []byte // The working buffer.
	buffer_pos int    // The current position of the buffer.

	unread int // The number of unread characters in the buffer.

	newlines int // The number of line breaks since last non-break/non-blank character

	raw_buffer     []byte // The raw buffer.
	raw_buffer_pos int    // The current position of the buffer.

	encoding yamlEncoding // The input encoding.

	offset int      // The offset of the current position (in bytes).
	mark   yamlMark // The mark of the current position.

	// Comments

	head_comment []byte // The current head comments
	line_comment []byte // The current line comments
	foot_comment []byte // The current foot comments
	tail_comment []byte // Foot comment that happens at the end of a block.
	stem_comment []byte // Comment in item preceding a nested structure (list inside list item, etc)

	comments      []yamlComment // The folded comments for all parsed tokens
	comments_head int

	// Scanner stuff

	stream_start_produced bool // Have we started to scan the input stream?
	stream_end_produced   bool // Have we reached the end of the input stream?

	flow_level int // The number of unclosed '[' and '{' indicators.

	tokens          []yamlToken // The tokens queue.
	tokens_head     int         // The head of the tokens queue.
	tokens_parsed   int         // The number of tokens fetched from the queue.
	token_available bool        // Does the tokens queue contain a token ready for dequeueing.

	indent  int   // The current indentation level.
	indents []int // The indentation levels stack.

	simple_key_allowed bool            // May a simple key occur at the current position?
	simple_keys        []yamlSimpleKey // The stack of simple keys.
	simple_keys_by_tok map[int]int     // possible simple_key indexes indexed by token_number

	// Parser stuff

	state          yamlParserState    // The current parser state.
	states         []yamlParserState  // The parser states stack.
	marks          []yamlMark         // The stack of marks.
	tag_directives []yamlTagDirective // The list of TAG directives.

	// Dumper stuff

	aliases []yamlAliasData // The alias data.

	document *yamlDocument // The currently parsed document.
}

type yamlComment struct {
	scan_mark  yamlMark // Position where scanning for comments started
	token_mark yamlMark // Position after which tokens will be associated with this comment
	start_mark yamlMark // Position of '#' comment mark
	end_mark   yamlMark // Position where comment terminated

	head []byte
	line []byte
	foot []byte
}

// Emitter Definitions

// The prototype of a write handler.
//
// The write handler is called when the emitter needs to flush the accumulated
// characters to the output.  The handler should write @a size bytes of the
// @a buffer to the output.
//
// @param[in,out]   data        A pointer to an application data specified by
//
//	yaml_emitter_set_output().
//
// @param[in]       buffer      The buffer with bytes to be written.
// @param[in]       size        The size of the buffer.
//
// @returns On success, the handler should return @c 1.  If the handler failed,
// the returned value should be @c 0.
type yamlWriteHandler func(emitter *yamlEmitter, buffer []byte) error

type yamlEmitterState int

// The emitter states.
const (
	// Expect STREAM-START.
	yaml_EMIT_STREAM_START_STATE yamlEmitterState = iota

	yaml_EMIT_FIRST_DOCUMENT_START_STATE       // Expect the first DOCUMENT-START or STREAM-END.
	yaml_EMIT_DOCUMENT_START_STATE             // Expect DOCUMENT-START or STREAM-END.
	yaml_EMIT_DOCUMENT_CONTENT_STATE           // Expect the content of a document.
	yaml_EMIT_DOCUMENT_END_STATE               // Expect DOCUMENT-END.
	yaml_EMIT_FLOW_SEQUENCE_FIRST_ITEM_STATE   // Expect the first item of a flow sequence.
	yaml_EMIT_FLOW_SEQUENCE_TRAIL_ITEM_STATE   // Expect the next item of a flow sequence, with the comma already written out
	yaml_EMIT_FLOW_SEQUENCE_ITEM_STATE         // Expect an item of a flow sequence.
	yaml_EMIT_FLOW_MAPPING_FIRST_KEY_STATE     // Expect the first key of a flow mapping.
	yaml_EMIT_FLOW_MAPPING_TRAIL_KEY_STATE     // Expect the next key of a flow mapping, with the comma already written out
	yaml_EMIT_FLOW_MAPPING_KEY_STATE           // Expect a key of a flow mapping.
	yaml_EMIT_FLOW_MAPPING_SIMPLE_VALUE_STATE  // Expect a value for a simple key of a flow mapping.
	yaml_EMIT_FLOW_MAPPING_VALUE_STATE         // Expect a value of a flow mapping.
	yaml_EMIT_BLOCK_SEQUENCE_FIRST_ITEM_STATE  // Expect the first item of a block sequence.
	yaml_EMIT_BLOCK_SEQUENCE_ITEM_STATE        // Expect an item of a block sequence.
	yaml_EMIT_BLOCK_MAPPING_FIRST_KEY_STATE    // Expect the first key of a block mapping.
	yaml_EMIT_BLOCK_MAPPING_KEY_STATE          // Expect the key of a block mapping.
	yaml_EMIT_BLOCK_MAPPING_SIMPLE_VALUE_STATE // Expect a value for a simple key of a block mapping.
	yaml_EMIT_BLOCK_MAPPING_VALUE_STATE        // Expect a value of a block mapping.
	yaml_EMIT_END_STATE                        // Expect nothing.
)

// The emitter structure.
//
// All members are internal.  Manage the structure using the @c yaml_emitter_
// family of functions.
type yamlEmitter struct {

	// Error handling

	error   yamlErrorType // Error type.
	problem string        // Error description.

	// Writer stuff

	write_handler yamlWriteHandler // Write handler.

	output_buffer *[]byte   // String output data.
	output_writer io.Writer // File output data.

	buffer     []byte // The working buffer.
	buffer_pos int    // The current position of the buffer.

	raw_buffer     []byte // The raw buffer.
	raw_buffer_pos int    // The current position of the buffer.

	encoding yamlEncoding // The stream encoding.

	// Emitter stuff

	canonical   bool          // If the output is in the canonical style?
	best_indent int           // The number of indentation spaces.
	best_width  int           // The preferred width of the output lines.
	unicode     bool          // Allow unescaped non-ASCII characters?
	line_break  yamlLineBreak // The preferred line break.

	state  yamlEmitterState   // The current emitter state.
	states []yamlEmitterState // The stack of states.

	events      []yamlEvent // The event queue.
	events_head int         // The head of the event queue.

	indents []int // The stack of indentation levels.

	tag_directives []yamlTagDirective // The list of tag directives.

	indent int // The current indentation level.

	compact_sequence_indent bool // Is '- ' is considered part of the indentation for sequence elements?

	flow_level int // The current flow level.

	root_context       bool // Is it the document root context?
	sequence_context   bool // Is it a sequence context?
	mapping_context    bool // Is it a mapping context?
	simple_key_context bool // Is it a simple mapping key context?

	line       int  // The current line.
	column     int  // The current column.
	whitespace bool // If the last character was a whitespace?
	indention  bool // If the last character was an indentation character (' ', '-', '?', ':')?
	open_ended bool // If an explicit document end is required?

	space_above bool // Is there's an empty line above?
	foot_indent int  // The indent used to write the foot comment above, or -1 if none.

	// Anchor analysis.
	anchor_data struct {
		anchor []byte // The anchor value.
		alias  bool   // Is it an alias?
	}

	// Tag analysis.
	tag_data struct {
		handle []byte // The tag handle.
		suffix []byte // The tag suffix.
	}

	// Scalar analysis.
	scalar_data struct {
		value                 []byte          // The scalar value.
		multiline             bool            // Does the scalar contain line breaks?
		flow_plain_allowed    bool            // Can the scalar be expessed in the flow plain style?
		block_plain_allowed   bool            // Can the scalar be expressed in the block plain style?
		single_quoted_allowed bool            // Can the scalar be expressed in the single quoted style?
		block_allowed         bool            // Can the scalar be expressed in the literal or folded styles?
		style                 yamlScalarStyle // The output style.
	}

	// Comments
	head_comment []byte
	line_comment []byte
	foot_comment []byte
	tail_comment []byte

	key_line_comment []byte

	// Dumper stuff

	opened bool // If the stream was already opened?
	closed bool // If the stream was already closed?

	// The information associated with the document nodes.
	anchors *struct {
		references int  // The number of references.
		anchor     int  // The anchor id.
		serialized bool // If the node has been emitted?
	}

	last_anchor_id int // The last assigned anchor id.

	document *yamlDocument // The currently emitted document.
}
