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

package libyaml

import (
	"bytes"
)

// The parser implements the following grammar:
//
// stream               ::= STREAM-START implicit_document? explicit_document* STREAM-END
// implicit_document    ::= block_node DOCUMENT-END*
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
// block_node_or_indentless_sequence    ::=
//                          ALIAS
//                          | properties (block_content | indentless_block_sequence)?
//                          | block_content
//                          | indentless_block_sequence
// block_node           ::= ALIAS
//                          | properties block_content?
//                          | block_content
// flow_node            ::= ALIAS
//                          | properties flow_content?
//                          | flow_content
// properties           ::= TAG ANCHOR? | ANCHOR TAG?
// block_content        ::= block_collection | flow_collection | SCALAR
// flow_content         ::= flow_collection | SCALAR
// block_collection     ::= block_sequence | block_mapping
// flow_collection      ::= flow_sequence | flow_mapping
// block_sequence       ::= BLOCK-SEQUENCE-START (BLOCK-ENTRY block_node?)* BLOCK-END
// indentless_sequence  ::= (BLOCK-ENTRY block_node?)+
// block_mapping        ::= BLOCK-MAPPING_START
//                          ((KEY block_node_or_indentless_sequence?)?
//                          (VALUE block_node_or_indentless_sequence?)?)*
//                          BLOCK-END
// flow_sequence        ::= FLOW-SEQUENCE-START
//                          (flow_sequence_entry FLOW-ENTRY)*
//                          flow_sequence_entry?
//                          FLOW-SEQUENCE-END
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
// flow_mapping         ::= FLOW-MAPPING-START
//                          (flow_mapping_entry FLOW-ENTRY)*
//                          flow_mapping_entry?
//                          FLOW-MAPPING-END
// flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?

// Peek the next token in the token queue.
func (parser *Parser) peekToken() *Token {
	if parser.token_available || parser.fetchMoreTokens() {
		token := &parser.tokens[parser.tokens_head]
		parser.unfoldComments(token)
		return token
	}
	return nil
}

// unfoldComments walks through the comments queue and joins all
// comments behind the position of the provided token into the respective
// top-level comment slices in the parser.
func (parser *Parser) unfoldComments(token *Token) {
	for parser.comments_head < len(parser.comments) && token.start_mark.Index >= parser.comments[parser.comments_head].token_mark.Index {
		comment := &parser.comments[parser.comments_head]
		if len(comment.head) > 0 {
			if token.typ == BLOCK_END_TOKEN {
				// No heads on ends, so keep comment.head for a follow up token.
				break
			}
			if len(parser.head_comment) > 0 {
				parser.head_comment = append(parser.head_comment, '\n')
			}
			parser.head_comment = append(parser.head_comment, comment.head...)
		}
		if len(comment.foot) > 0 {
			if len(parser.foot_comment) > 0 {
				parser.foot_comment = append(parser.foot_comment, '\n')
			}
			parser.foot_comment = append(parser.foot_comment, comment.foot...)
		}
		if len(comment.line) > 0 {
			if len(parser.line_comment) > 0 {
				parser.line_comment = append(parser.line_comment, '\n')
			}
			parser.line_comment = append(parser.line_comment, comment.line...)
		}
		*comment = Comment{}
		parser.comments_head++
	}
}

// Remove the next token from the queue (must be called after peek_token).
func (parser *Parser) skipToken() {
	parser.token_available = false
	parser.tokens_parsed++
	parser.stream_end_produced = parser.tokens[parser.tokens_head].typ == STREAM_END_TOKEN
	parser.tokens_head++
}

// Get the next event.
func (parser *Parser) Parse(event *Event) bool {
	// Erase the event object.
	*event = Event{}

	// No events after the end of the stream or error.
	if parser.stream_end_produced || parser.Err != NO_ERROR || parser.state == PARSE_END_STATE {
		return true
	}

	// Generate the next event.
	return parser.stateMachine(event)
}

// Set parser error.
func (parser *Parser) setParserError(problem string, problem_mark Mark) bool {
	parser.Err = PARSER_ERROR
	parser.Problem = problem
	parser.Problem_mark = problem_mark
	return false
}

func (parser *Parser) setParserErrorContext(context string, context_mark Mark, problem string, problem_mark Mark) bool {
	parser.Err = PARSER_ERROR
	parser.Context = context
	parser.Context_mark = context_mark
	parser.Problem = problem
	parser.Problem_mark = problem_mark
	return false
}

// State dispatcher.
func (parser *Parser) stateMachine(event *Event) bool {
	// trace("yaml_parser_state_machine", "state:", parser.state.String())

	switch parser.state {
	case PARSE_STREAM_START_STATE:
		return parser.parseStreamStart(event)

	case PARSE_IMPLICIT_DOCUMENT_START_STATE:
		return parser.parseDocumentStart(event, true)

	case PARSE_DOCUMENT_START_STATE:
		return parser.parseDocumentStart(event, false)

	case PARSE_DOCUMENT_CONTENT_STATE:
		return parser.parseDocumentContent(event)

	case PARSE_DOCUMENT_END_STATE:
		return parser.parseDocumentEnd(event)

	case PARSE_BLOCK_NODE_STATE:
		return parser.parseNode(event, true, false)

	case PARSE_BLOCK_NODE_OR_INDENTLESS_SEQUENCE_STATE:
		return parser.parseNode(event, true, true)

	case PARSE_FLOW_NODE_STATE:
		return parser.parseNode(event, false, false)

	case PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE:
		return parser.parseBlockSequenceEntry(event, true)

	case PARSE_BLOCK_SEQUENCE_ENTRY_STATE:
		return parser.parseBlockSequenceEntry(event, false)

	case PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE:
		return parser.parseIndentlessSequenceEntry(event)

	case PARSE_BLOCK_MAPPING_FIRST_KEY_STATE:
		return parser.parseBlockMappingKey(event, true)

	case PARSE_BLOCK_MAPPING_KEY_STATE:
		return parser.parseBlockMappingKey(event, false)

	case PARSE_BLOCK_MAPPING_VALUE_STATE:
		return parser.parseBlockMappingValue(event)

	case PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE:
		return parser.parseFlowSequenceEntry(event, true)

	case PARSE_FLOW_SEQUENCE_ENTRY_STATE:
		return parser.parseFlowSequenceEntry(event, false)

	case PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE:
		return parser.parseFlowSequenceEntryMappingKey(event)

	case PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE:
		return parser.parseFlowSequenceEntryMappingValue(event)

	case PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE:
		return parser.parseFlowSequenceEntryMappingEnd(event)

	case PARSE_FLOW_MAPPING_FIRST_KEY_STATE:
		return parser.parseFlowMappingKey(event, true)

	case PARSE_FLOW_MAPPING_KEY_STATE:
		return parser.parseFlowMappingKey(event, false)

	case PARSE_FLOW_MAPPING_VALUE_STATE:
		return parser.parseFlowMappingValue(event, false)

	case PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE:
		return parser.parseFlowMappingValue(event, true)

	default:
		panic("invalid parser state")
	}
}

// Parse the production:
// stream   ::= STREAM-START implicit_document? explicit_document* STREAM-END
//
//	************
func (parser *Parser) parseStreamStart(event *Event) bool {
	token := parser.peekToken()
	if token == nil {
		return false
	}
	if token.typ != STREAM_START_TOKEN {
		return parser.setParserError("did not find expected <stream-start>", token.start_mark)
	}
	parser.state = PARSE_IMPLICIT_DOCUMENT_START_STATE
	*event = Event{
		Typ:        STREAM_START_EVENT,
		Start_mark: token.start_mark,
		End_mark:   token.end_mark,
		encoding:   token.encoding,
	}
	parser.skipToken()
	return true
}

// Parse the productions:
// implicit_document    ::= block_node DOCUMENT-END*
//
//	*
//
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
//
//	*************************
func (parser *Parser) parseDocumentStart(event *Event, implicit bool) bool {
	token := parser.peekToken()
	if token == nil {
		return false
	}

	// Parse extra document end indicators.
	if !implicit {
		for token.typ == DOCUMENT_END_TOKEN {
			parser.skipToken()
			token = parser.peekToken()
			if token == nil {
				return false
			}
		}
	}

	if implicit && token.typ != VERSION_DIRECTIVE_TOKEN &&
		token.typ != TAG_DIRECTIVE_TOKEN &&
		token.typ != DOCUMENT_START_TOKEN &&
		token.typ != STREAM_END_TOKEN {
		// Parse an implicit document.
		if !parser.processDirectives(nil, nil) {
			return false
		}
		parser.states = append(parser.states, PARSE_DOCUMENT_END_STATE)
		parser.state = PARSE_BLOCK_NODE_STATE

		var head_comment []byte
		if len(parser.head_comment) > 0 {
			// [Go] Scan the header comment backwards, and if an empty line is found, break
			//      the header so the part before the last empty line goes into the
			//      document header, while the bottom of it goes into a follow up event.
			for i := len(parser.head_comment) - 1; i > 0; i-- {
				if parser.head_comment[i] == '\n' {
					if i == len(parser.head_comment)-1 {
						head_comment = parser.head_comment[:i]
						parser.head_comment = parser.head_comment[i+1:]
						break
					} else if parser.head_comment[i-1] == '\n' {
						head_comment = parser.head_comment[:i-1]
						parser.head_comment = parser.head_comment[i+1:]
						break
					}
				}
			}
		}

		*event = Event{
			Typ:        DOCUMENT_START_EVENT,
			Start_mark: token.start_mark,
			End_mark:   token.end_mark,

			Head_comment: head_comment,
		}

	} else if token.typ != STREAM_END_TOKEN {
		// Parse an explicit document.
		var version_directive *VersionDirective
		var tag_directives []TagDirective
		start_mark := token.start_mark
		if !parser.processDirectives(&version_directive, &tag_directives) {
			return false
		}
		token = parser.peekToken()
		if token == nil {
			return false
		}
		if token.typ != DOCUMENT_START_TOKEN {
			parser.setParserError(
				"did not find expected <document start>", token.start_mark)
			return false
		}
		parser.states = append(parser.states, PARSE_DOCUMENT_END_STATE)
		parser.state = PARSE_DOCUMENT_CONTENT_STATE
		end_mark := token.end_mark

		*event = Event{
			Typ:               DOCUMENT_START_EVENT,
			Start_mark:        start_mark,
			End_mark:          end_mark,
			version_directive: version_directive,
			tag_directives:    tag_directives,
			Implicit:          false,
		}
		parser.skipToken()

	} else {
		// Parse the stream end.
		parser.state = PARSE_END_STATE
		*event = Event{
			Typ:        STREAM_END_EVENT,
			Start_mark: token.start_mark,
			End_mark:   token.end_mark,
		}
		parser.skipToken()
	}

	return true
}

// Parse the productions:
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
//
//	***********
func (parser *Parser) parseDocumentContent(event *Event) bool {
	token := parser.peekToken()
	if token == nil {
		return false
	}

	if token.typ == VERSION_DIRECTIVE_TOKEN ||
		token.typ == TAG_DIRECTIVE_TOKEN ||
		token.typ == DOCUMENT_START_TOKEN ||
		token.typ == DOCUMENT_END_TOKEN ||
		token.typ == STREAM_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		return parser.processEmptyScalar(event,
			token.start_mark)
	}
	return parser.parseNode(event, true, false)
}

// Parse the productions:
// implicit_document    ::= block_node DOCUMENT-END*
//
//	*************
//
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
func (parser *Parser) parseDocumentEnd(event *Event) bool {
	token := parser.peekToken()
	if token == nil {
		return false
	}

	start_mark := token.start_mark
	end_mark := token.start_mark

	implicit := true
	if token.typ == DOCUMENT_END_TOKEN {
		end_mark = token.end_mark
		parser.skipToken()
		implicit = false
	}

	parser.tag_directives = parser.tag_directives[:0]

	parser.state = PARSE_DOCUMENT_START_STATE
	*event = Event{
		Typ:        DOCUMENT_END_EVENT,
		Start_mark: start_mark,
		End_mark:   end_mark,
		Implicit:   implicit,
	}
	parser.setEventComments(event)
	if len(event.Head_comment) > 0 && len(event.Foot_comment) == 0 {
		event.Foot_comment = event.Head_comment
		event.Head_comment = nil
	}
	return true
}

func (parser *Parser) setEventComments(event *Event) {
	event.Head_comment = parser.head_comment
	event.Line_comment = parser.line_comment
	event.Foot_comment = parser.foot_comment
	parser.head_comment = nil
	parser.line_comment = nil
	parser.foot_comment = nil
	parser.tail_comment = nil
	parser.stem_comment = nil
}

// Parse the productions:
// block_node_or_indentless_sequence    ::=
//
//	ALIAS
//	*****
//	| properties (block_content | indentless_block_sequence)?
//	  **********  *
//	| block_content | indentless_block_sequence
//	  *
//
// block_node           ::= ALIAS
//
//	*****
//	| properties block_content?
//	  ********** *
//	| block_content
//	  *
//
// flow_node            ::= ALIAS
//
//	*****
//	| properties flow_content?
//	  ********** *
//	| flow_content
//	  *
//
// properties           ::= TAG ANCHOR? | ANCHOR TAG?
//
//	*************************
//
// block_content        ::= block_collection | flow_collection | SCALAR
//
//	******
//
// flow_content         ::= flow_collection | SCALAR
//
//	******
func (parser *Parser) parseNode(event *Event, block, indentless_sequence bool) bool {
	// defer trace("yaml_parser_parse_node", "block:", block, "indentless_sequence:", indentless_sequence)()

	token := parser.peekToken()
	if token == nil {
		return false
	}

	if token.typ == ALIAS_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		*event = Event{
			Typ:        ALIAS_EVENT,
			Start_mark: token.start_mark,
			End_mark:   token.end_mark,
			Anchor:     token.value,
		}
		parser.setEventComments(event)
		parser.skipToken()
		return true
	}

	start_mark := token.start_mark
	end_mark := token.start_mark

	var tag_token bool
	var tag_handle, tag_suffix, anchor []byte
	var tag_mark Mark
	if token.typ == ANCHOR_TOKEN {
		anchor = token.value
		start_mark = token.start_mark
		end_mark = token.end_mark
		parser.skipToken()
		token = parser.peekToken()
		if token == nil {
			return false
		}
		if token.typ == TAG_TOKEN {
			tag_token = true
			tag_handle = token.value
			tag_suffix = token.suffix
			tag_mark = token.start_mark
			end_mark = token.end_mark
			parser.skipToken()
			token = parser.peekToken()
			if token == nil {
				return false
			}
		}
	} else if token.typ == TAG_TOKEN {
		tag_token = true
		tag_handle = token.value
		tag_suffix = token.suffix
		start_mark = token.start_mark
		tag_mark = token.start_mark
		end_mark = token.end_mark
		parser.skipToken()
		token = parser.peekToken()
		if token == nil {
			return false
		}
		if token.typ == ANCHOR_TOKEN {
			anchor = token.value
			end_mark = token.end_mark
			parser.skipToken()
			token = parser.peekToken()
			if token == nil {
				return false
			}
		}
	}

	var tag []byte
	if tag_token {
		if len(tag_handle) == 0 {
			tag = tag_suffix
		} else {
			for i := range parser.tag_directives {
				if bytes.Equal(parser.tag_directives[i].handle, tag_handle) {
					tag = append([]byte(nil), parser.tag_directives[i].prefix...)
					tag = append(tag, tag_suffix...)
					break
				}
			}
			if len(tag) == 0 {
				parser.setParserErrorContext(
					"while parsing a node", start_mark,
					"found undefined tag handle", tag_mark)
				return false
			}
		}
	}

	implicit := len(tag) == 0
	if indentless_sequence && token.typ == BLOCK_ENTRY_TOKEN {
		end_mark = token.end_mark
		parser.state = PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE
		*event = Event{
			Typ:        SEQUENCE_START_EVENT,
			Start_mark: start_mark,
			End_mark:   end_mark,
			Anchor:     anchor,
			Tag:        tag,
			Implicit:   implicit,
			Style:      Style(BLOCK_SEQUENCE_STYLE),
		}
		return true
	}
	if token.typ == SCALAR_TOKEN {
		var plain_implicit, quoted_implicit bool
		end_mark = token.end_mark
		if (len(tag) == 0 && token.style == PLAIN_SCALAR_STYLE) || (len(tag) == 1 && tag[0] == '!') {
			plain_implicit = true
		} else if len(tag) == 0 {
			quoted_implicit = true
		}
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]

		*event = Event{
			Typ:             SCALAR_EVENT,
			Start_mark:      start_mark,
			End_mark:        end_mark,
			Anchor:          anchor,
			Tag:             tag,
			Value:           token.value,
			Implicit:        plain_implicit,
			quoted_implicit: quoted_implicit,
			Style:           Style(token.style),
		}
		parser.setEventComments(event)
		parser.skipToken()
		return true
	}
	if token.typ == FLOW_SEQUENCE_START_TOKEN {
		// [Go] Some of the events below can be merged as they differ only on style.
		end_mark = token.end_mark
		parser.state = PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE
		*event = Event{
			Typ:        SEQUENCE_START_EVENT,
			Start_mark: start_mark,
			End_mark:   end_mark,
			Anchor:     anchor,
			Tag:        tag,
			Implicit:   implicit,
			Style:      Style(FLOW_SEQUENCE_STYLE),
		}
		parser.setEventComments(event)
		return true
	}
	if token.typ == FLOW_MAPPING_START_TOKEN {
		end_mark = token.end_mark
		parser.state = PARSE_FLOW_MAPPING_FIRST_KEY_STATE
		*event = Event{
			Typ:        MAPPING_START_EVENT,
			Start_mark: start_mark,
			End_mark:   end_mark,
			Anchor:     anchor,
			Tag:        tag,
			Implicit:   implicit,
			Style:      Style(FLOW_MAPPING_STYLE),
		}
		parser.setEventComments(event)
		return true
	}
	if block && token.typ == BLOCK_SEQUENCE_START_TOKEN {
		end_mark = token.end_mark
		parser.state = PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE
		*event = Event{
			Typ:        SEQUENCE_START_EVENT,
			Start_mark: start_mark,
			End_mark:   end_mark,
			Anchor:     anchor,
			Tag:        tag,
			Implicit:   implicit,
			Style:      Style(BLOCK_SEQUENCE_STYLE),
		}
		if parser.stem_comment != nil {
			event.Head_comment = parser.stem_comment
			parser.stem_comment = nil
		}
		return true
	}
	if block && token.typ == BLOCK_MAPPING_START_TOKEN {
		end_mark = token.end_mark
		parser.state = PARSE_BLOCK_MAPPING_FIRST_KEY_STATE
		*event = Event{
			Typ:        MAPPING_START_EVENT,
			Start_mark: start_mark,
			End_mark:   end_mark,
			Anchor:     anchor,
			Tag:        tag,
			Implicit:   implicit,
			Style:      Style(BLOCK_MAPPING_STYLE),
		}
		if parser.stem_comment != nil {
			event.Head_comment = parser.stem_comment
			parser.stem_comment = nil
		}
		return true
	}
	if len(anchor) > 0 || len(tag) > 0 {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]

		*event = Event{
			Typ:             SCALAR_EVENT,
			Start_mark:      start_mark,
			End_mark:        end_mark,
			Anchor:          anchor,
			Tag:             tag,
			Implicit:        implicit,
			quoted_implicit: false,
			Style:           Style(PLAIN_SCALAR_STYLE),
		}
		return true
	}

	context := "while parsing a flow node"
	if block {
		context = "while parsing a block node"
	}
	parser.setParserErrorContext(context, start_mark,
		"did not find expected node content", token.start_mark)
	return false
}

// Parse the productions:
// block_sequence ::= BLOCK-SEQUENCE-START (BLOCK-ENTRY block_node?)* BLOCK-END
//
//	********************  *********** *             *********
func (parser *Parser) parseBlockSequenceEntry(event *Event, first bool) bool {
	if first {
		token := parser.peekToken()
		if token == nil {
			return false
		}
		parser.marks = append(parser.marks, token.start_mark)
		parser.skipToken()
	}

	token := parser.peekToken()
	if token == nil {
		return false
	}

	if token.typ == BLOCK_ENTRY_TOKEN {
		mark := token.end_mark
		prior_head_len := len(parser.head_comment)
		parser.skipToken()
		parser.splitStemComment(prior_head_len)
		token = parser.peekToken()
		if token == nil {
			return false
		}
		if token.typ != BLOCK_ENTRY_TOKEN && token.typ != BLOCK_END_TOKEN {
			parser.states = append(parser.states, PARSE_BLOCK_SEQUENCE_ENTRY_STATE)
			return parser.parseNode(event, true, false)
		} else {
			parser.state = PARSE_BLOCK_SEQUENCE_ENTRY_STATE
			return parser.processEmptyScalar(event, mark)
		}
	}
	if token.typ == BLOCK_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		parser.marks = parser.marks[:len(parser.marks)-1]

		*event = Event{
			Typ:        SEQUENCE_END_EVENT,
			Start_mark: token.start_mark,
			End_mark:   token.end_mark,
		}

		parser.skipToken()
		return true
	}

	context_mark := parser.marks[len(parser.marks)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]
	return parser.setParserErrorContext(
		"while parsing a block collection", context_mark,
		"did not find expected '-' indicator", token.start_mark)
}

// Parse the productions:
// indentless_sequence  ::= (BLOCK-ENTRY block_node?)+
//
//	*********** *
func (parser *Parser) parseIndentlessSequenceEntry(event *Event) bool {
	token := parser.peekToken()
	if token == nil {
		return false
	}

	if token.typ == BLOCK_ENTRY_TOKEN {
		mark := token.end_mark
		prior_head_len := len(parser.head_comment)
		parser.skipToken()
		parser.splitStemComment(prior_head_len)
		token = parser.peekToken()
		if token == nil {
			return false
		}
		if token.typ != BLOCK_ENTRY_TOKEN &&
			token.typ != KEY_TOKEN &&
			token.typ != VALUE_TOKEN &&
			token.typ != BLOCK_END_TOKEN {
			parser.states = append(parser.states, PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE)
			return parser.parseNode(event, true, false)
		}
		parser.state = PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE
		return parser.processEmptyScalar(event, mark)
	}
	parser.state = parser.states[len(parser.states)-1]
	parser.states = parser.states[:len(parser.states)-1]

	*event = Event{
		Typ:        SEQUENCE_END_EVENT,
		Start_mark: token.start_mark,
		End_mark:   token.start_mark, // [Go] Shouldn't this be token.end_mark?
	}
	return true
}

// Split stem comment from head comment.
//
// When a sequence or map is found under a sequence entry, the former head comment
// is assigned to the underlying sequence or map as a whole, not the individual
// sequence or map entry as would be expected otherwise. To handle this case the
// previous head comment is moved aside as the stem comment.
func (parser *Parser) splitStemComment(stem_len int) {
	if stem_len == 0 {
		return
	}

	token := parser.peekToken()
	if token == nil || token.typ != BLOCK_SEQUENCE_START_TOKEN && token.typ != BLOCK_MAPPING_START_TOKEN {
		return
	}

	parser.stem_comment = parser.head_comment[:stem_len]
	if len(parser.head_comment) == stem_len {
		parser.head_comment = nil
	} else {
		// Copy suffix to prevent very strange bugs if someone ever appends
		// further bytes to the prefix in the stem_comment slice above.
		parser.head_comment = append([]byte(nil), parser.head_comment[stem_len+1:]...)
	}
}

// Parse the productions:
// block_mapping        ::= BLOCK-MAPPING_START
//
//	*******************
//	((KEY block_node_or_indentless_sequence?)?
//	  *** *
//	(VALUE block_node_or_indentless_sequence?)?)*
//
//	BLOCK-END
//	*********
func (parser *Parser) parseBlockMappingKey(event *Event, first bool) bool {
	if first {
		token := parser.peekToken()
		if token == nil {
			return false
		}
		parser.marks = append(parser.marks, token.start_mark)
		parser.skipToken()
	}

	token := parser.peekToken()
	if token == nil {
		return false
	}

	// [Go] A tail comment was left from the prior mapping value processed. Emit an event
	//      as it needs to be processed with that value and not the following key.
	if len(parser.tail_comment) > 0 {
		*event = Event{
			Typ:          TAIL_COMMENT_EVENT,
			Start_mark:   token.start_mark,
			End_mark:     token.end_mark,
			Foot_comment: parser.tail_comment,
		}
		parser.tail_comment = nil
		return true
	}

	if token.typ == KEY_TOKEN {
		mark := token.end_mark
		parser.skipToken()
		token = parser.peekToken()
		if token == nil {
			return false
		}
		if token.typ != KEY_TOKEN &&
			token.typ != VALUE_TOKEN &&
			token.typ != BLOCK_END_TOKEN {
			parser.states = append(parser.states, PARSE_BLOCK_MAPPING_VALUE_STATE)
			return parser.parseNode(event, true, true)
		} else {
			parser.state = PARSE_BLOCK_MAPPING_VALUE_STATE
			return parser.processEmptyScalar(event, mark)
		}
	} else if token.typ == BLOCK_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		parser.marks = parser.marks[:len(parser.marks)-1]
		*event = Event{
			Typ:        MAPPING_END_EVENT,
			Start_mark: token.start_mark,
			End_mark:   token.end_mark,
		}
		parser.setEventComments(event)
		parser.skipToken()
		return true
	}

	context_mark := parser.marks[len(parser.marks)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]
	return parser.setParserErrorContext(
		"while parsing a block mapping", context_mark,
		"did not find expected key", token.start_mark)
}

// Parse the productions:
// block_mapping        ::= BLOCK-MAPPING_START
//
//	((KEY block_node_or_indentless_sequence?)?
//
//	(VALUE block_node_or_indentless_sequence?)?)*
//	 ***** *
//	BLOCK-END
func (parser *Parser) parseBlockMappingValue(event *Event) bool {
	token := parser.peekToken()
	if token == nil {
		return false
	}
	if token.typ == VALUE_TOKEN {
		mark := token.end_mark
		parser.skipToken()
		token = parser.peekToken()
		if token == nil {
			return false
		}
		if token.typ != KEY_TOKEN &&
			token.typ != VALUE_TOKEN &&
			token.typ != BLOCK_END_TOKEN {
			parser.states = append(parser.states, PARSE_BLOCK_MAPPING_KEY_STATE)
			return parser.parseNode(event, true, true)
		}
		parser.state = PARSE_BLOCK_MAPPING_KEY_STATE
		return parser.processEmptyScalar(event, mark)
	}
	parser.state = PARSE_BLOCK_MAPPING_KEY_STATE
	return parser.processEmptyScalar(event, token.start_mark)
}

// Parse the productions:
// flow_sequence        ::= FLOW-SEQUENCE-START
//
//	*******************
//	(flow_sequence_entry FLOW-ENTRY)*
//	 *                   **********
//	flow_sequence_entry?
//	*
//	FLOW-SEQUENCE-END
//	*****************
//
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//
//	*
func (parser *Parser) parseFlowSequenceEntry(event *Event, first bool) bool {
	if first {
		token := parser.peekToken()
		if token == nil {
			return false
		}
		parser.marks = append(parser.marks, token.start_mark)
		parser.skipToken()
	}
	token := parser.peekToken()
	if token == nil {
		return false
	}
	if token.typ != FLOW_SEQUENCE_END_TOKEN {
		if !first {
			if token.typ == FLOW_ENTRY_TOKEN {
				parser.skipToken()
				token = parser.peekToken()
				if token == nil {
					return false
				}
			} else {
				context_mark := parser.marks[len(parser.marks)-1]
				parser.marks = parser.marks[:len(parser.marks)-1]
				return parser.setParserErrorContext(
					"while parsing a flow sequence", context_mark,
					"did not find expected ',' or ']'", token.start_mark)
			}
		}

		if token.typ == KEY_TOKEN {
			parser.state = PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE
			*event = Event{
				Typ:        MAPPING_START_EVENT,
				Start_mark: token.start_mark,
				End_mark:   token.end_mark,
				Implicit:   true,
				Style:      Style(FLOW_MAPPING_STYLE),
			}
			parser.skipToken()
			return true
		} else if token.typ != FLOW_SEQUENCE_END_TOKEN {
			parser.states = append(parser.states, PARSE_FLOW_SEQUENCE_ENTRY_STATE)
			return parser.parseNode(event, false, false)
		}
	}

	parser.state = parser.states[len(parser.states)-1]
	parser.states = parser.states[:len(parser.states)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]

	*event = Event{
		Typ:        SEQUENCE_END_EVENT,
		Start_mark: token.start_mark,
		End_mark:   token.end_mark,
	}
	parser.setEventComments(event)

	parser.skipToken()
	return true
}

// Parse the productions:
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//
//	*** *
func (parser *Parser) parseFlowSequenceEntryMappingKey(event *Event) bool {
	token := parser.peekToken()
	if token == nil {
		return false
	}
	if token.typ != VALUE_TOKEN &&
		token.typ != FLOW_ENTRY_TOKEN &&
		token.typ != FLOW_SEQUENCE_END_TOKEN {
		parser.states = append(parser.states, PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE)
		return parser.parseNode(event, false, false)
	}
	mark := token.end_mark
	parser.skipToken()
	parser.state = PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE
	return parser.processEmptyScalar(event, mark)
}

// Parse the productions:
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//
//	***** *
func (parser *Parser) parseFlowSequenceEntryMappingValue(event *Event) bool {
	token := parser.peekToken()
	if token == nil {
		return false
	}
	if token.typ == VALUE_TOKEN {
		parser.skipToken()
		token := parser.peekToken()
		if token == nil {
			return false
		}
		if token.typ != FLOW_ENTRY_TOKEN && token.typ != FLOW_SEQUENCE_END_TOKEN {
			parser.states = append(parser.states, PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE)
			return parser.parseNode(event, false, false)
		}
	}
	parser.state = PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE
	return parser.processEmptyScalar(event, token.start_mark)
}

// Parse the productions:
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//
//	*
func (parser *Parser) parseFlowSequenceEntryMappingEnd(event *Event) bool {
	token := parser.peekToken()
	if token == nil {
		return false
	}
	parser.state = PARSE_FLOW_SEQUENCE_ENTRY_STATE
	*event = Event{
		Typ:        MAPPING_END_EVENT,
		Start_mark: token.start_mark,
		End_mark:   token.start_mark, // [Go] Shouldn't this be end_mark?
	}
	return true
}

// Parse the productions:
// flow_mapping         ::= FLOW-MAPPING-START
//
//	******************
//	(flow_mapping_entry FLOW-ENTRY)*
//	 *                  **********
//	flow_mapping_entry?
//	******************
//	FLOW-MAPPING-END
//	****************
//
// flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//   - *** *
func (parser *Parser) parseFlowMappingKey(event *Event, first bool) bool {
	if first {
		token := parser.peekToken()
		parser.marks = append(parser.marks, token.start_mark)
		parser.skipToken()
	}

	token := parser.peekToken()
	if token == nil {
		return false
	}

	if token.typ != FLOW_MAPPING_END_TOKEN {
		if !first {
			if token.typ == FLOW_ENTRY_TOKEN {
				parser.skipToken()
				token = parser.peekToken()
				if token == nil {
					return false
				}
			} else {
				context_mark := parser.marks[len(parser.marks)-1]
				parser.marks = parser.marks[:len(parser.marks)-1]
				return parser.setParserErrorContext(
					"while parsing a flow mapping", context_mark,
					"did not find expected ',' or '}'", token.start_mark)
			}
		}

		if token.typ == KEY_TOKEN {
			parser.skipToken()
			token = parser.peekToken()
			if token == nil {
				return false
			}
			if token.typ != VALUE_TOKEN &&
				token.typ != FLOW_ENTRY_TOKEN &&
				token.typ != FLOW_MAPPING_END_TOKEN {
				parser.states = append(parser.states, PARSE_FLOW_MAPPING_VALUE_STATE)
				return parser.parseNode(event, false, false)
			} else {
				parser.state = PARSE_FLOW_MAPPING_VALUE_STATE
				return parser.processEmptyScalar(event, token.start_mark)
			}
		} else if token.typ != FLOW_MAPPING_END_TOKEN {
			parser.states = append(parser.states, PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE)
			return parser.parseNode(event, false, false)
		}
	}

	parser.state = parser.states[len(parser.states)-1]
	parser.states = parser.states[:len(parser.states)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]
	*event = Event{
		Typ:        MAPPING_END_EVENT,
		Start_mark: token.start_mark,
		End_mark:   token.end_mark,
	}
	parser.setEventComments(event)
	parser.skipToken()
	return true
}

// Parse the productions:
// flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//   - ***** *
func (parser *Parser) parseFlowMappingValue(event *Event, empty bool) bool {
	token := parser.peekToken()
	if token == nil {
		return false
	}
	if empty {
		parser.state = PARSE_FLOW_MAPPING_KEY_STATE
		return parser.processEmptyScalar(event, token.start_mark)
	}
	if token.typ == VALUE_TOKEN {
		parser.skipToken()
		token = parser.peekToken()
		if token == nil {
			return false
		}
		if token.typ != FLOW_ENTRY_TOKEN && token.typ != FLOW_MAPPING_END_TOKEN {
			parser.states = append(parser.states, PARSE_FLOW_MAPPING_KEY_STATE)
			return parser.parseNode(event, false, false)
		}
	}
	parser.state = PARSE_FLOW_MAPPING_KEY_STATE
	return parser.processEmptyScalar(event, token.start_mark)
}

// Generate an empty scalar event.
func (parser *Parser) processEmptyScalar(event *Event, mark Mark) bool {
	*event = Event{
		Typ:        SCALAR_EVENT,
		Start_mark: mark,
		End_mark:   mark,
		Value:      nil, // Empty
		Implicit:   true,
		Style:      Style(PLAIN_SCALAR_STYLE),
	}
	return true
}

var default_tag_directives = []TagDirective{
	{[]byte("!"), []byte("!")},
	{[]byte("!!"), []byte("tag:yaml.org,2002:")},
}

// Parse directives.
func (parser *Parser) processDirectives(version_directive_ref **VersionDirective, tag_directives_ref *[]TagDirective) bool {
	var version_directive *VersionDirective
	var tag_directives []TagDirective

	token := parser.peekToken()
	if token == nil {
		return false
	}

	for token.typ == VERSION_DIRECTIVE_TOKEN || token.typ == TAG_DIRECTIVE_TOKEN {
		if token.typ == VERSION_DIRECTIVE_TOKEN {
			if version_directive != nil {
				parser.setParserError(
					"found duplicate %YAML directive", token.start_mark)
				return false
			}
			if token.major != 1 || token.minor != 1 {
				parser.setParserError(
					"found incompatible YAML document", token.start_mark)
				return false
			}
			version_directive = &VersionDirective{
				major: token.major,
				minor: token.minor,
			}
		} else if token.typ == TAG_DIRECTIVE_TOKEN {
			value := TagDirective{
				handle: token.value,
				prefix: token.prefix,
			}
			if !parser.appendTagDirective(value, false, token.start_mark) {
				return false
			}
			tag_directives = append(tag_directives, value)
		}

		parser.skipToken()
		token = parser.peekToken()
		if token == nil {
			return false
		}
	}

	for i := range default_tag_directives {
		if !parser.appendTagDirective(default_tag_directives[i], true, token.start_mark) {
			return false
		}
	}

	if version_directive_ref != nil {
		*version_directive_ref = version_directive
	}
	if tag_directives_ref != nil {
		*tag_directives_ref = tag_directives
	}
	return true
}

// Append a tag directive to the directives stack.
func (parser *Parser) appendTagDirective(value TagDirective, allow_duplicates bool, mark Mark) bool {
	for i := range parser.tag_directives {
		if bytes.Equal(value.handle, parser.tag_directives[i].handle) {
			if allow_duplicates {
				return true
			}
			return parser.setParserError("found duplicate %TAG directive", mark)
		}
	}

	// [Go] I suspect the copy is unnecessary. This was likely done
	// because there was no way to track ownership of the data.
	value_copy := TagDirective{
		handle: make([]byte, len(value.handle)),
		prefix: make([]byte, len(value.prefix)),
	}
	copy(value_copy.handle, value.handle)
	copy(value_copy.prefix, value.prefix)
	parser.tag_directives = append(parser.tag_directives, value_copy)
	return true
}
