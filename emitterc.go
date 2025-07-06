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
	"bytes"
	"fmt"
)

// Flush the buffer if needed.
func (emitter *yamlEmitter) flushIfNeeded() bool {
	if emitter.buffer_pos+5 >= len(emitter.buffer) {
		return emitter.flush()
	}
	return true
}

// Put a character to the output buffer.
func (emitter *yamlEmitter) put(value byte) bool {
	if emitter.buffer_pos+5 >= len(emitter.buffer) && !emitter.flush() {
		return false
	}
	emitter.buffer[emitter.buffer_pos] = value
	emitter.buffer_pos++
	emitter.column++
	return true
}

// Put a line break to the output buffer.
func (emitter *yamlEmitter) putLineBreak() bool {
	if emitter.buffer_pos+5 >= len(emitter.buffer) && !emitter.flush() {
		return false
	}
	switch emitter.line_break {
	case yaml_CR_BREAK:
		emitter.buffer[emitter.buffer_pos] = '\r'
		emitter.buffer_pos += 1
	case yaml_LN_BREAK:
		emitter.buffer[emitter.buffer_pos] = '\n'
		emitter.buffer_pos += 1
	case yaml_CRLN_BREAK:
		emitter.buffer[emitter.buffer_pos+0] = '\r'
		emitter.buffer[emitter.buffer_pos+1] = '\n'
		emitter.buffer_pos += 2
	default:
		panic("unknown line break setting")
	}
	if emitter.column == 0 {
		emitter.space_above = true
	}
	emitter.column = 0
	emitter.line++
	// [Go] Do this here and below and drop from everywhere else (see commented lines).
	emitter.indention = true
	return true
}

// Copy a character from a string into buffer.
func (emitter *yamlEmitter) write(s []byte, i *int) bool {
	if emitter.buffer_pos+5 >= len(emitter.buffer) && !emitter.flush() {
		return false
	}
	p := emitter.buffer_pos
	w := width(s[*i])
	switch w {
	case 4:
		emitter.buffer[p+3] = s[*i+3]
		fallthrough
	case 3:
		emitter.buffer[p+2] = s[*i+2]
		fallthrough
	case 2:
		emitter.buffer[p+1] = s[*i+1]
		fallthrough
	case 1:
		emitter.buffer[p+0] = s[*i+0]
	default:
		panic("unknown character width")
	}
	emitter.column++
	emitter.buffer_pos += w
	*i += w
	return true
}

// Write a whole string into buffer.
func (emitter *yamlEmitter) writeAll(s []byte) bool {
	for i := 0; i < len(s); {
		if !emitter.write(s, &i) {
			return false
		}
	}
	return true
}

// Copy a line break character from a string into buffer.
func (emitter *yamlEmitter) writeLineBreak(s []byte, i *int) bool {
	if s[*i] == '\n' {
		if !emitter.putLineBreak() {
			return false
		}
		*i++
	} else {
		if !emitter.write(s, i) {
			return false
		}
		if emitter.column == 0 {
			emitter.space_above = true
		}
		emitter.column = 0
		emitter.line++
		// [Go] Do this here and above and drop from everywhere else (see commented lines).
		emitter.indention = true
	}
	return true
}

// Set an emitter error and return false.
func (emitter *yamlEmitter) setEmitterError(problem string) bool {
	emitter.error = yaml_EMITTER_ERROR
	emitter.problem = problem
	return false
}

// Emit an event.
func (emitter *yamlEmitter) emit(event *yamlEvent) bool {
	emitter.events = append(emitter.events, *event)
	for !emitter.needMoreEvents() {
		event := &emitter.events[emitter.events_head]
		if !emitter.analyzeEvent(event) {
			return false
		}
		if !emitter.stateMachine(event) {
			return false
		}
		event.delete()
		emitter.events_head++
	}
	return true
}

// Check if we need to accumulate more events before emitting.
//
// We accumulate extra
//   - 1 event for DOCUMENT-START
//   - 2 events for SEQUENCE-START
//   - 3 events for MAPPING-START
func (emitter *yamlEmitter) needMoreEvents() bool {
	if emitter.events_head == len(emitter.events) {
		return true
	}
	var accumulate int
	switch emitter.events[emitter.events_head].typ {
	case yaml_DOCUMENT_START_EVENT:
		accumulate = 1
	case yaml_SEQUENCE_START_EVENT:
		accumulate = 2
	case yaml_MAPPING_START_EVENT:
		accumulate = 3
	default:
		return false
	}
	if len(emitter.events)-emitter.events_head > accumulate {
		return false
	}
	var level int
	for i := emitter.events_head; i < len(emitter.events); i++ {
		switch emitter.events[i].typ {
		case yaml_STREAM_START_EVENT, yaml_DOCUMENT_START_EVENT, yaml_SEQUENCE_START_EVENT, yaml_MAPPING_START_EVENT:
			level++
		case yaml_STREAM_END_EVENT, yaml_DOCUMENT_END_EVENT, yaml_SEQUENCE_END_EVENT, yaml_MAPPING_END_EVENT:
			level--
		}
		if level == 0 {
			return false
		}
	}
	return true
}

// Append a directive to the directives stack.
func (emitter *yamlEmitter) appendTagDirective(value *yamlTagDirective, allow_duplicates bool) bool {
	for i := 0; i < len(emitter.tag_directives); i++ {
		if bytes.Equal(value.handle, emitter.tag_directives[i].handle) {
			if allow_duplicates {
				return true
			}
			return emitter.setEmitterError("duplicate %TAG directive")
		}
	}

	// [Go] Do we actually need to copy this given garbage collection
	// and the lack of deallocating destructors?
	tag_copy := yamlTagDirective{
		handle: make([]byte, len(value.handle)),
		prefix: make([]byte, len(value.prefix)),
	}
	copy(tag_copy.handle, value.handle)
	copy(tag_copy.prefix, value.prefix)
	emitter.tag_directives = append(emitter.tag_directives, tag_copy)
	return true
}

// Increase the indentation level.
func (emitter *yamlEmitter) increaseIndentCompact(flow, indentless bool, compact_seq bool) bool {
	emitter.indents = append(emitter.indents, emitter.indent)
	if emitter.indent < 0 {
		if flow {
			emitter.indent = emitter.best_indent
		} else {
			emitter.indent = 0
		}
	} else if !indentless {
		// [Go] This was changed so that indentations are more regular.
		if emitter.states[len(emitter.states)-1] == yaml_EMIT_BLOCK_SEQUENCE_ITEM_STATE {
			// The first indent inside a sequence will just skip the "- " indicator.
			emitter.indent += 2
		} else {
			// Everything else aligns to the chosen indentation.
			emitter.indent = emitter.best_indent * ((emitter.indent + emitter.best_indent) / emitter.best_indent)
			if compact_seq {
				// The value compact_seq passed in is almost always set to `false` when this function is called,
				// except when we are dealing with sequence nodes. So this gets triggered to subtract 2 only when we
				// are increasing the indent to account for sequence nodes, which will be correct because we need to
				// subtract 2 to account for the - at the beginning of the sequence node.
				emitter.indent = emitter.indent - 2
			}
		}
	}
	return true
}

// State dispatcher.
func (emitter *yamlEmitter) stateMachine(event *yamlEvent) bool {
	switch emitter.state {
	default:
	case yaml_EMIT_STREAM_START_STATE:
		return emitter.emitStreamStart(event)

	case yaml_EMIT_FIRST_DOCUMENT_START_STATE:
		return emitter.emitDocumentStart(event, true)

	case yaml_EMIT_DOCUMENT_START_STATE:
		return emitter.emitDocumentStart(event, false)

	case yaml_EMIT_DOCUMENT_CONTENT_STATE:
		return emitter.emitDocumentContent(event)

	case yaml_EMIT_DOCUMENT_END_STATE:
		return emitter.emitDocumentEnd(event)

	case yaml_EMIT_FLOW_SEQUENCE_FIRST_ITEM_STATE:
		return emitter.emitFlowSequenceItem(event, true, false)

	case yaml_EMIT_FLOW_SEQUENCE_TRAIL_ITEM_STATE:
		return emitter.emitFlowSequenceItem(event, false, true)

	case yaml_EMIT_FLOW_SEQUENCE_ITEM_STATE:
		return emitter.emitFlowSequenceItem(event, false, false)

	case yaml_EMIT_FLOW_MAPPING_FIRST_KEY_STATE:
		return emitter.emitFlowMappingKey(event, true, false)

	case yaml_EMIT_FLOW_MAPPING_TRAIL_KEY_STATE:
		return emitter.emitFlowMappingKey(event, false, true)

	case yaml_EMIT_FLOW_MAPPING_KEY_STATE:
		return emitter.emitFlowMappingKey(event, false, false)

	case yaml_EMIT_FLOW_MAPPING_SIMPLE_VALUE_STATE:
		return emitter.emitFlowMappingValue(event, true)

	case yaml_EMIT_FLOW_MAPPING_VALUE_STATE:
		return emitter.emitFlowMappingValue(event, false)

	case yaml_EMIT_BLOCK_SEQUENCE_FIRST_ITEM_STATE:
		return emitter.emitBlockSequenceItem(event, true)

	case yaml_EMIT_BLOCK_SEQUENCE_ITEM_STATE:
		return emitter.emitBlockSequenceItem(event, false)

	case yaml_EMIT_BLOCK_MAPPING_FIRST_KEY_STATE:
		return emitter.emitBlockMappingKey(event, true)

	case yaml_EMIT_BLOCK_MAPPING_KEY_STATE:
		return emitter.emitBlockMappingKey(event, false)

	case yaml_EMIT_BLOCK_MAPPING_SIMPLE_VALUE_STATE:
		return emitter.emitBlockMappingValue(event, true)

	case yaml_EMIT_BLOCK_MAPPING_VALUE_STATE:
		return emitter.emitBlockMappingValue(event, false)

	case yaml_EMIT_END_STATE:
		return emitter.setEmitterError("expected nothing after STREAM-END")
	}
	panic("invalid emitter state")
}

// Expect STREAM-START.
func (emitter *yamlEmitter) emitStreamStart(event *yamlEvent) bool {
	if event.typ != yaml_STREAM_START_EVENT {
		return emitter.setEmitterError("expected STREAM-START")
	}
	if emitter.encoding == yaml_ANY_ENCODING {
		emitter.encoding = event.encoding
		if emitter.encoding == yaml_ANY_ENCODING {
			emitter.encoding = yaml_UTF8_ENCODING
		}
	}
	if emitter.best_indent < 2 || emitter.best_indent > 9 {
		emitter.best_indent = 2
	}
	if emitter.best_width >= 0 && emitter.best_width <= emitter.best_indent*2 {
		emitter.best_width = 80
	}
	if emitter.best_width < 0 {
		emitter.best_width = 1<<31 - 1
	}
	if emitter.line_break == yaml_ANY_BREAK {
		emitter.line_break = yaml_LN_BREAK
	}

	emitter.indent = -1
	emitter.line = 0
	emitter.column = 0
	emitter.whitespace = true
	emitter.indention = true
	emitter.space_above = true
	emitter.foot_indent = -1

	if emitter.encoding != yaml_UTF8_ENCODING {
		if !emitter.writeBom() {
			return false
		}
	}
	emitter.state = yaml_EMIT_FIRST_DOCUMENT_START_STATE
	return true
}

// Expect DOCUMENT-START or STREAM-END.
func (emitter *yamlEmitter) emitDocumentStart(event *yamlEvent, first bool) bool {
	if event.typ == yaml_DOCUMENT_START_EVENT {

		if event.version_directive != nil {
			if !emitter.analyzeVersionDirective(event.version_directive) {
				return false
			}
		}

		for i := 0; i < len(event.tag_directives); i++ {
			tag_directive := &event.tag_directives[i]
			if !emitter.analyzeTagDirective(tag_directive) {
				return false
			}
			if !emitter.appendTagDirective(tag_directive, false) {
				return false
			}
		}

		for i := 0; i < len(default_tag_directives); i++ {
			tag_directive := &default_tag_directives[i]
			if !emitter.appendTagDirective(tag_directive, true) {
				return false
			}
		}

		implicit := event.implicit
		if !first || emitter.canonical {
			implicit = false
		}

		if emitter.open_ended && (event.version_directive != nil || len(event.tag_directives) > 0) {
			if !emitter.writeIndicator([]byte("..."), true, false, false) {
				return false
			}
			if !emitter.writeIndent() {
				return false
			}
		}

		if event.version_directive != nil {
			implicit = false
			if !emitter.writeIndicator([]byte("%YAML"), true, false, false) {
				return false
			}
			if !emitter.writeIndicator([]byte("1.1"), true, false, false) {
				return false
			}
			if !emitter.writeIndent() {
				return false
			}
		}

		if len(event.tag_directives) > 0 {
			implicit = false
			for i := 0; i < len(event.tag_directives); i++ {
				tag_directive := &event.tag_directives[i]
				if !emitter.writeIndicator([]byte("%TAG"), true, false, false) {
					return false
				}
				if !emitter.writeTagHandle(tag_directive.handle) {
					return false
				}
				if !emitter.writeTagContent(tag_directive.prefix, true) {
					return false
				}
				if !emitter.writeIndent() {
					return false
				}
			}
		}

		if emitter.checkEmptyDocument() {
			implicit = false
		}
		if !implicit {
			if !emitter.writeIndent() {
				return false
			}
			if !emitter.writeIndicator([]byte("---"), true, false, false) {
				return false
			}
			if emitter.canonical || true {
				if !emitter.writeIndent() {
					return false
				}
			}
		}

		if len(emitter.head_comment) > 0 {
			if !emitter.processHeadComment() {
				return false
			}
			if !emitter.putLineBreak() {
				return false
			}
		}

		emitter.state = yaml_EMIT_DOCUMENT_CONTENT_STATE
		return true
	}

	if event.typ == yaml_STREAM_END_EVENT {
		if emitter.open_ended {
			if !emitter.writeIndicator([]byte("..."), true, false, false) {
				return false
			}
			if !emitter.writeIndent() {
				return false
			}
		}
		if !emitter.flush() {
			return false
		}
		emitter.state = yaml_EMIT_END_STATE
		return true
	}

	return emitter.setEmitterError("expected DOCUMENT-START or STREAM-END")
}

// emitter preserves the original signature and delegates to
// increaseIndentCompact without compact-sequence indentation
func (emitter *yamlEmitter) increaseIndent(flow, indentless bool) bool {
	return emitter.increaseIndentCompact(flow, indentless, false)
}

// processLineComment preserves the original signature and delegates to
// processLineCommentLinebreak passing false for linebreak
func (emitter *yamlEmitter) processLineComment() bool {
	return emitter.processLineCommentLinebreak(false)
}

// Expect the root node.
func (emitter *yamlEmitter) emitDocumentContent(event *yamlEvent) bool {
	emitter.states = append(emitter.states, yaml_EMIT_DOCUMENT_END_STATE)

	if !emitter.processHeadComment() {
		return false
	}
	if !emitter.emitNode(event, true, false, false, false) {
		return false
	}
	if !emitter.processLineComment() {
		return false
	}
	if !emitter.processFootComment() {
		return false
	}
	return true
}

// Expect DOCUMENT-END.
func (emitter *yamlEmitter) emitDocumentEnd(event *yamlEvent) bool {
	if event.typ != yaml_DOCUMENT_END_EVENT {
		return emitter.setEmitterError("expected DOCUMENT-END")
	}
	// [Go] Force document foot separation.
	emitter.foot_indent = 0
	if !emitter.processFootComment() {
		return false
	}
	emitter.foot_indent = -1
	if !emitter.writeIndent() {
		return false
	}
	if !event.implicit {
		// [Go] Allocate the slice elsewhere.
		if !emitter.writeIndicator([]byte("..."), true, false, false) {
			return false
		}
		if !emitter.writeIndent() {
			return false
		}
	}
	if !emitter.flush() {
		return false
	}
	emitter.state = yaml_EMIT_DOCUMENT_START_STATE
	emitter.tag_directives = emitter.tag_directives[:0]
	return true
}

// Expect a flow item node.
func (emitter *yamlEmitter) emitFlowSequenceItem(event *yamlEvent, first, trail bool) bool {
	if first {
		if !emitter.writeIndicator([]byte{'['}, true, true, false) {
			return false
		}
		if !emitter.increaseIndent(true, false) {
			return false
		}
		emitter.flow_level++
	}

	if event.typ == yaml_SEQUENCE_END_EVENT {
		if emitter.canonical && !first && !trail {
			if !emitter.writeIndicator([]byte{','}, false, false, false) {
				return false
			}
		}
		emitter.flow_level--
		emitter.indent = emitter.indents[len(emitter.indents)-1]
		emitter.indents = emitter.indents[:len(emitter.indents)-1]
		if emitter.column == 0 || emitter.canonical && !first {
			if !emitter.writeIndent() {
				return false
			}
		}
		if !emitter.writeIndicator([]byte{']'}, false, false, false) {
			return false
		}
		if !emitter.processLineComment() {
			return false
		}
		if !emitter.processFootComment() {
			return false
		}
		emitter.state = emitter.states[len(emitter.states)-1]
		emitter.states = emitter.states[:len(emitter.states)-1]

		return true
	}

	if !first && !trail {
		if !emitter.writeIndicator([]byte{','}, false, false, false) {
			return false
		}
	}

	if !emitter.processHeadComment() {
		return false
	}
	if emitter.column == 0 {
		if !emitter.writeIndent() {
			return false
		}
	}

	if emitter.canonical || emitter.column > emitter.best_width {
		if !emitter.writeIndent() {
			return false
		}
	}
	if len(emitter.line_comment)+len(emitter.foot_comment)+len(emitter.tail_comment) > 0 {
		emitter.states = append(emitter.states, yaml_EMIT_FLOW_SEQUENCE_TRAIL_ITEM_STATE)
	} else {
		emitter.states = append(emitter.states, yaml_EMIT_FLOW_SEQUENCE_ITEM_STATE)
	}
	if !emitter.emitNode(event, false, true, false, false) {
		return false
	}
	if len(emitter.line_comment)+len(emitter.foot_comment)+len(emitter.tail_comment) > 0 {
		if !emitter.writeIndicator([]byte{','}, false, false, false) {
			return false
		}
	}
	if !emitter.processLineComment() {
		return false
	}
	if !emitter.processFootComment() {
		return false
	}
	return true
}

// Expect a flow key node.
func (emitter *yamlEmitter) emitFlowMappingKey(event *yamlEvent, first, trail bool) bool {
	if first {
		if !emitter.writeIndicator([]byte{'{'}, true, true, false) {
			return false
		}
		if !emitter.increaseIndent(true, false) {
			return false
		}
		emitter.flow_level++
	}

	if event.typ == yaml_MAPPING_END_EVENT {
		if (emitter.canonical || len(emitter.head_comment)+len(emitter.foot_comment)+len(emitter.tail_comment) > 0) && !first && !trail {
			if !emitter.writeIndicator([]byte{','}, false, false, false) {
				return false
			}
		}
		if !emitter.processHeadComment() {
			return false
		}
		emitter.flow_level--
		emitter.indent = emitter.indents[len(emitter.indents)-1]
		emitter.indents = emitter.indents[:len(emitter.indents)-1]
		if emitter.canonical && !first {
			if !emitter.writeIndent() {
				return false
			}
		}
		if !emitter.writeIndicator([]byte{'}'}, false, false, false) {
			return false
		}
		if !emitter.processLineComment() {
			return false
		}
		if !emitter.processFootComment() {
			return false
		}
		emitter.state = emitter.states[len(emitter.states)-1]
		emitter.states = emitter.states[:len(emitter.states)-1]
		return true
	}

	if !first && !trail {
		if !emitter.writeIndicator([]byte{','}, false, false, false) {
			return false
		}
	}

	if !emitter.processHeadComment() {
		return false
	}

	if emitter.column == 0 {
		if !emitter.writeIndent() {
			return false
		}
	}

	if emitter.canonical || emitter.column > emitter.best_width {
		if !emitter.writeIndent() {
			return false
		}
	}

	if !emitter.canonical && emitter.checkSimpleKey() {
		emitter.states = append(emitter.states, yaml_EMIT_FLOW_MAPPING_SIMPLE_VALUE_STATE)
		return emitter.emitNode(event, false, false, true, true)
	}
	if !emitter.writeIndicator([]byte{'?'}, true, false, false) {
		return false
	}
	emitter.states = append(emitter.states, yaml_EMIT_FLOW_MAPPING_VALUE_STATE)
	return emitter.emitNode(event, false, false, true, false)
}

// Expect a flow value node.
func (emitter *yamlEmitter) emitFlowMappingValue(event *yamlEvent, simple bool) bool {
	if simple {
		if !emitter.writeIndicator([]byte{':'}, false, false, false) {
			return false
		}
	} else {
		if emitter.canonical || emitter.column > emitter.best_width {
			if !emitter.writeIndent() {
				return false
			}
		}
		if !emitter.writeIndicator([]byte{':'}, true, false, false) {
			return false
		}
	}
	if len(emitter.line_comment)+len(emitter.foot_comment)+len(emitter.tail_comment) > 0 {
		emitter.states = append(emitter.states, yaml_EMIT_FLOW_MAPPING_TRAIL_KEY_STATE)
	} else {
		emitter.states = append(emitter.states, yaml_EMIT_FLOW_MAPPING_KEY_STATE)
	}
	if !emitter.emitNode(event, false, false, true, false) {
		return false
	}
	if len(emitter.line_comment)+len(emitter.foot_comment)+len(emitter.tail_comment) > 0 {
		if !emitter.writeIndicator([]byte{','}, false, false, false) {
			return false
		}
	}
	if !emitter.processLineComment() {
		return false
	}
	if !emitter.processFootComment() {
		return false
	}
	return true
}

// Expect a block item node.
func (emitter *yamlEmitter) emitBlockSequenceItem(event *yamlEvent, first bool) bool {
	if first {
		// emitter.mapping context tells us if we are currently in a mapping context.
		// emitter.column tells us which column we are in the yaml output. 0 is the first char of the column.
		// emitter.indentation tells us if the last character was an indentation character.
		// emitter.compact_sequence_indent tells us if '- ' is considered part of the indentation for sequence elements.
		// So, `seq` means that we are in a mapping context, and we are either at the first char of the column or
		//  the last character was not an indentation character, and we consider '- ' part of the indentation
		//  for sequence elements.
		seq := emitter.mapping_context && (emitter.column == 0 || !emitter.indention) &&
			emitter.compact_sequence_indent
		if !emitter.increaseIndentCompact(false, false, seq) {
			return false
		}
	}
	if event.typ == yaml_SEQUENCE_END_EVENT {
		emitter.indent = emitter.indents[len(emitter.indents)-1]
		emitter.indents = emitter.indents[:len(emitter.indents)-1]
		emitter.state = emitter.states[len(emitter.states)-1]
		emitter.states = emitter.states[:len(emitter.states)-1]
		return true
	}
	if !emitter.processHeadComment() {
		return false
	}
	if !emitter.writeIndent() {
		return false
	}
	if !emitter.writeIndicator([]byte{'-'}, true, false, true) {
		return false
	}
	emitter.states = append(emitter.states, yaml_EMIT_BLOCK_SEQUENCE_ITEM_STATE)
	if !emitter.emitNode(event, false, true, false, false) {
		return false
	}
	if !emitter.processLineComment() {
		return false
	}
	if !emitter.processFootComment() {
		return false
	}
	return true
}

// Expect a block key node.
func (emitter *yamlEmitter) emitBlockMappingKey(event *yamlEvent, first bool) bool {
	if first {
		if !emitter.increaseIndent(false, false) {
			return false
		}
	}
	if !emitter.processHeadComment() {
		return false
	}
	if event.typ == yaml_MAPPING_END_EVENT {
		emitter.indent = emitter.indents[len(emitter.indents)-1]
		emitter.indents = emitter.indents[:len(emitter.indents)-1]
		emitter.state = emitter.states[len(emitter.states)-1]
		emitter.states = emitter.states[:len(emitter.states)-1]
		return true
	}
	if !emitter.writeIndent() {
		return false
	}
	if len(emitter.line_comment) > 0 {
		// [Go] A line comment was provided for the key. That's unusual as the
		//      scanner associates line comments with the value. Either way,
		//      save the line comment and render it appropriately later.
		emitter.key_line_comment = emitter.line_comment
		emitter.line_comment = nil
	}
	if emitter.checkSimpleKey() {
		emitter.states = append(emitter.states, yaml_EMIT_BLOCK_MAPPING_SIMPLE_VALUE_STATE)
		return emitter.emitNode(event, false, false, true, true)
	}
	if !emitter.writeIndicator([]byte{'?'}, true, false, true) {
		return false
	}
	emitter.states = append(emitter.states, yaml_EMIT_BLOCK_MAPPING_VALUE_STATE)
	return emitter.emitNode(event, false, false, true, false)
}

// Expect a block value node.
func (emitter *yamlEmitter) emitBlockMappingValue(event *yamlEvent, simple bool) bool {
	if simple {
		if !emitter.writeIndicator([]byte{':'}, false, false, false) {
			return false
		}
	} else {
		if !emitter.writeIndent() {
			return false
		}
		if !emitter.writeIndicator([]byte{':'}, true, false, true) {
			return false
		}
	}
	if len(emitter.key_line_comment) > 0 {
		// [Go] Line comments are generally associated with the value, but when there's
		//      no value on the same line as a mapping key they end up attached to the
		//      key itself.
		if event.typ == yaml_SCALAR_EVENT {
			if len(emitter.line_comment) == 0 {
				// A scalar is coming and it has no line comments by itself yet,
				// so just let it handle the line comment as usual. If it has a
				// line comment, we can't have both so the one from the key is lost.
				emitter.line_comment = emitter.key_line_comment
				emitter.key_line_comment = nil
			}
		} else if event.sequenceStyle() != yaml_FLOW_SEQUENCE_STYLE && (event.typ == yaml_MAPPING_START_EVENT || event.typ == yaml_SEQUENCE_START_EVENT) {
			// An indented block follows, so write the comment right now.
			emitter.line_comment, emitter.key_line_comment = emitter.key_line_comment, emitter.line_comment
			if !emitter.processLineComment() {
				return false
			}
			emitter.line_comment, emitter.key_line_comment = emitter.key_line_comment, emitter.line_comment
		}
	}
	emitter.states = append(emitter.states, yaml_EMIT_BLOCK_MAPPING_KEY_STATE)
	if !emitter.emitNode(event, false, false, true, false) {
		return false
	}
	if !emitter.processLineComment() {
		return false
	}
	if !emitter.processFootComment() {
		return false
	}
	return true
}

func (emitter *yamlEmitter) silentNilEvent(event *yamlEvent) bool {
	return event.typ == yaml_SCALAR_EVENT && event.implicit && !emitter.canonical && len(emitter.scalar_data.value) == 0
}

// Expect a node.
func (emitter *yamlEmitter) emitNode(event *yamlEvent,
	root bool, sequence bool, mapping bool, simple_key bool,
) bool {
	emitter.root_context = root
	emitter.sequence_context = sequence
	emitter.mapping_context = mapping
	emitter.simple_key_context = simple_key

	switch event.typ {
	case yaml_ALIAS_EVENT:
		return emitter.emitAlias(event)
	case yaml_SCALAR_EVENT:
		return emitter.emitScalar(event)
	case yaml_SEQUENCE_START_EVENT:
		return emitter.emitSequenceStart(event)
	case yaml_MAPPING_START_EVENT:
		return emitter.emitMappingStart(event)
	default:
		return emitter.setEmitterError(
			fmt.Sprintf("expected SCALAR, SEQUENCE-START, MAPPING-START, or ALIAS, but got %v", event.typ))
	}
}

// Expect ALIAS.
func (emitter *yamlEmitter) emitAlias(event *yamlEvent) bool {
	if !emitter.processAnchor() {
		return false
	}
	emitter.state = emitter.states[len(emitter.states)-1]
	emitter.states = emitter.states[:len(emitter.states)-1]
	return true
}

// Expect SCALAR.
func (emitter *yamlEmitter) emitScalar(event *yamlEvent) bool {
	if !emitter.selectScalarStyle(event) {
		return false
	}
	if !emitter.processAnchor() {
		return false
	}
	if !emitter.processTag() {
		return false
	}
	if !emitter.increaseIndent(true, false) {
		return false
	}
	if !emitter.processScalar() {
		return false
	}
	emitter.indent = emitter.indents[len(emitter.indents)-1]
	emitter.indents = emitter.indents[:len(emitter.indents)-1]
	emitter.state = emitter.states[len(emitter.states)-1]
	emitter.states = emitter.states[:len(emitter.states)-1]
	return true
}

// Expect SEQUENCE-START.
func (emitter *yamlEmitter) emitSequenceStart(event *yamlEvent) bool {
	if !emitter.processAnchor() {
		return false
	}
	if !emitter.processTag() {
		return false
	}
	if emitter.flow_level > 0 || emitter.canonical || event.sequenceStyle() == yaml_FLOW_SEQUENCE_STYLE ||
		emitter.checkEmptySequence() {
		emitter.state = yaml_EMIT_FLOW_SEQUENCE_FIRST_ITEM_STATE
	} else {
		emitter.state = yaml_EMIT_BLOCK_SEQUENCE_FIRST_ITEM_STATE
	}
	return true
}

// Expect MAPPING-START.
func (emitter *yamlEmitter) emitMappingStart(event *yamlEvent) bool {
	if !emitter.processAnchor() {
		return false
	}
	if !emitter.processTag() {
		return false
	}
	if emitter.flow_level > 0 || emitter.canonical || event.mappingStyle() == yaml_FLOW_MAPPING_STYLE ||
		emitter.checkEmptyMapping() {
		emitter.state = yaml_EMIT_FLOW_MAPPING_FIRST_KEY_STATE
	} else {
		emitter.state = yaml_EMIT_BLOCK_MAPPING_FIRST_KEY_STATE
	}
	return true
}

// Check if the document content is an empty scalar.
func (emitter *yamlEmitter) checkEmptyDocument() bool {
	return false // [Go] Huh?
}

// Check if the next events represent an empty sequence.
func (emitter *yamlEmitter) checkEmptySequence() bool {
	if len(emitter.events)-emitter.events_head < 2 {
		return false
	}
	return emitter.events[emitter.events_head].typ == yaml_SEQUENCE_START_EVENT &&
		emitter.events[emitter.events_head+1].typ == yaml_SEQUENCE_END_EVENT
}

// Check if the next events represent an empty mapping.
func (emitter *yamlEmitter) checkEmptyMapping() bool {
	if len(emitter.events)-emitter.events_head < 2 {
		return false
	}
	return emitter.events[emitter.events_head].typ == yaml_MAPPING_START_EVENT &&
		emitter.events[emitter.events_head+1].typ == yaml_MAPPING_END_EVENT
}

// Check if the next node can be expressed as a simple key.
func (emitter *yamlEmitter) checkSimpleKey() bool {
	length := 0
	switch emitter.events[emitter.events_head].typ {
	case yaml_ALIAS_EVENT:
		length += len(emitter.anchor_data.anchor)
	case yaml_SCALAR_EVENT:
		if emitter.scalar_data.multiline {
			return false
		}
		length += len(emitter.anchor_data.anchor) +
			len(emitter.tag_data.handle) +
			len(emitter.tag_data.suffix) +
			len(emitter.scalar_data.value)
	case yaml_SEQUENCE_START_EVENT:
		if !emitter.checkEmptySequence() {
			return false
		}
		length += len(emitter.anchor_data.anchor) +
			len(emitter.tag_data.handle) +
			len(emitter.tag_data.suffix)
	case yaml_MAPPING_START_EVENT:
		if !emitter.checkEmptyMapping() {
			return false
		}
		length += len(emitter.anchor_data.anchor) +
			len(emitter.tag_data.handle) +
			len(emitter.tag_data.suffix)
	default:
		return false
	}
	return length <= 128
}

// Determine an acceptable scalar style.
func (emitter *yamlEmitter) selectScalarStyle(event *yamlEvent) bool {
	no_tag := len(emitter.tag_data.handle) == 0 && len(emitter.tag_data.suffix) == 0
	if no_tag && !event.implicit && !event.quoted_implicit {
		return emitter.setEmitterError("neither tag nor implicit flags are specified")
	}

	style := event.scalarStyle()
	if style == yaml_ANY_SCALAR_STYLE {
		style = yaml_PLAIN_SCALAR_STYLE
	}
	if emitter.canonical {
		style = yaml_DOUBLE_QUOTED_SCALAR_STYLE
	}
	if emitter.simple_key_context && emitter.scalar_data.multiline {
		style = yaml_DOUBLE_QUOTED_SCALAR_STYLE
	}

	if style == yaml_PLAIN_SCALAR_STYLE {
		if emitter.flow_level > 0 && !emitter.scalar_data.flow_plain_allowed ||
			emitter.flow_level == 0 && !emitter.scalar_data.block_plain_allowed {
			style = yaml_SINGLE_QUOTED_SCALAR_STYLE
		}
		if len(emitter.scalar_data.value) == 0 && (emitter.flow_level > 0 || emitter.simple_key_context) {
			style = yaml_SINGLE_QUOTED_SCALAR_STYLE
		}
		if no_tag && !event.implicit {
			style = yaml_SINGLE_QUOTED_SCALAR_STYLE
		}
	}
	if style == yaml_SINGLE_QUOTED_SCALAR_STYLE {
		if !emitter.scalar_data.single_quoted_allowed {
			style = yaml_DOUBLE_QUOTED_SCALAR_STYLE
		}
	}
	if style == yaml_LITERAL_SCALAR_STYLE || style == yaml_FOLDED_SCALAR_STYLE {
		if !emitter.scalar_data.block_allowed || emitter.flow_level > 0 || emitter.simple_key_context {
			style = yaml_DOUBLE_QUOTED_SCALAR_STYLE
		}
	}

	if no_tag && !event.quoted_implicit && style != yaml_PLAIN_SCALAR_STYLE {
		emitter.tag_data.handle = []byte{'!'}
	}
	emitter.scalar_data.style = style
	return true
}

// Write an anchor.
func (emitter *yamlEmitter) processAnchor() bool {
	if emitter.anchor_data.anchor == nil {
		return true
	}
	c := []byte{'&'}
	if emitter.anchor_data.alias {
		c[0] = '*'
	}
	if !emitter.writeIndicator(c, true, false, false) {
		return false
	}
	return emitter.writeAnchor(emitter.anchor_data.anchor)
}

// Write a tag.
func (emitter *yamlEmitter) processTag() bool {
	if len(emitter.tag_data.handle) == 0 && len(emitter.tag_data.suffix) == 0 {
		return true
	}
	if len(emitter.tag_data.handle) > 0 {
		if !emitter.writeTagHandle(emitter.tag_data.handle) {
			return false
		}
		if len(emitter.tag_data.suffix) > 0 {
			if !emitter.writeTagContent(emitter.tag_data.suffix, false) {
				return false
			}
		}
	} else {
		// [Go] Allocate these slices elsewhere.
		if !emitter.writeIndicator([]byte("!<"), true, false, false) {
			return false
		}
		if !emitter.writeTagContent(emitter.tag_data.suffix, false) {
			return false
		}
		if !emitter.writeIndicator([]byte{'>'}, false, false, false) {
			return false
		}
	}
	return true
}

// Write a scalar.
func (emitter *yamlEmitter) processScalar() bool {
	switch emitter.scalar_data.style {
	case yaml_PLAIN_SCALAR_STYLE:
		return emitter.writePlainScalar(emitter.scalar_data.value, !emitter.simple_key_context)

	case yaml_SINGLE_QUOTED_SCALAR_STYLE:
		return emitter.writeSingleQuotedScalar(emitter.scalar_data.value, !emitter.simple_key_context)

	case yaml_DOUBLE_QUOTED_SCALAR_STYLE:
		return emitter.writeDoubleQuotedScalar(emitter.scalar_data.value, !emitter.simple_key_context)

	case yaml_LITERAL_SCALAR_STYLE:
		return emitter.writeLiteralScalar(emitter.scalar_data.value)

	case yaml_FOLDED_SCALAR_STYLE:
		return emitter.writeFoldedScalar(emitter.scalar_data.value)
	}
	panic("unknown scalar style")
}

// Write a head comment.
func (emitter *yamlEmitter) processHeadComment() bool {
	if len(emitter.tail_comment) > 0 {
		if !emitter.writeIndent() {
			return false
		}
		if !emitter.writeComment(emitter.tail_comment) {
			return false
		}
		emitter.tail_comment = emitter.tail_comment[:0]
		emitter.foot_indent = emitter.indent
		if emitter.foot_indent < 0 {
			emitter.foot_indent = 0
		}
	}

	if len(emitter.head_comment) == 0 {
		return true
	}
	if !emitter.writeIndent() {
		return false
	}
	if !emitter.writeComment(emitter.head_comment) {
		return false
	}
	emitter.head_comment = emitter.head_comment[:0]
	return true
}

// Write an line comment.
func (emitter *yamlEmitter) processLineCommentLinebreak(linebreak bool) bool {
	if len(emitter.line_comment) == 0 {
		// The next 3 lines are needed to resolve an issue with leading newlines
		// See https://github.com/go-yaml/yaml/issues/755
		// When linebreak is set to true, put_break will be called and will add
		// the needed newline.
		if linebreak && !emitter.putLineBreak() {
			return false
		}
		return true
	}
	if !emitter.whitespace {
		if !emitter.put(' ') {
			return false
		}
	}
	if !emitter.writeComment(emitter.line_comment) {
		return false
	}
	emitter.line_comment = emitter.line_comment[:0]
	return true
}

// Write a foot comment.
func (emitter *yamlEmitter) processFootComment() bool {
	if len(emitter.foot_comment) == 0 {
		return true
	}
	if !emitter.writeIndent() {
		return false
	}
	if !emitter.writeComment(emitter.foot_comment) {
		return false
	}
	emitter.foot_comment = emitter.foot_comment[:0]
	emitter.foot_indent = emitter.indent
	if emitter.foot_indent < 0 {
		emitter.foot_indent = 0
	}
	return true
}

// Check if a %YAML directive is valid.
func (emitter *yamlEmitter) analyzeVersionDirective(version_directive *yamlVersionDirective) bool {
	if version_directive.major != 1 || version_directive.minor != 1 {
		return emitter.setEmitterError("incompatible %YAML directive")
	}
	return true
}

// Check if a %TAG directive is valid.
func (emitter *yamlEmitter) analyzeTagDirective(tag_directive *yamlTagDirective) bool {
	handle := tag_directive.handle
	prefix := tag_directive.prefix
	if len(handle) == 0 {
		return emitter.setEmitterError("tag handle must not be empty")
	}
	if handle[0] != '!' {
		return emitter.setEmitterError("tag handle must start with '!'")
	}
	if handle[len(handle)-1] != '!' {
		return emitter.setEmitterError("tag handle must end with '!'")
	}
	for i := 1; i < len(handle)-1; i += width(handle[i]) {
		if !isAlpha(handle, i) {
			return emitter.setEmitterError("tag handle must contain alphanumerical characters only")
		}
	}
	if len(prefix) == 0 {
		return emitter.setEmitterError("tag prefix must not be empty")
	}
	return true
}

// Check if an anchor is valid.
func (emitter *yamlEmitter) analyzeAnchor(anchor []byte, alias bool) bool {
	if len(anchor) == 0 {
		problem := "anchor value must not be empty"
		if alias {
			problem = "alias value must not be empty"
		}
		return emitter.setEmitterError(problem)
	}
	for i := 0; i < len(anchor); i += width(anchor[i]) {
		if !isAnchorChar(anchor, i) {
			problem := "anchor value must contain valid characters only"
			if alias {
				problem = "alias value must contain valid characters only"
			}
			return emitter.setEmitterError(problem)
		}
	}
	emitter.anchor_data.anchor = anchor
	emitter.anchor_data.alias = alias
	return true
}

// Check if a tag is valid.
func (emitter *yamlEmitter) analyzeTag(tag []byte) bool {
	if len(tag) == 0 {
		return emitter.setEmitterError("tag value must not be empty")
	}
	for i := 0; i < len(emitter.tag_directives); i++ {
		tag_directive := &emitter.tag_directives[i]
		if bytes.HasPrefix(tag, tag_directive.prefix) {
			emitter.tag_data.handle = tag_directive.handle
			emitter.tag_data.suffix = tag[len(tag_directive.prefix):]
			return true
		}
	}
	emitter.tag_data.suffix = tag
	return true
}

// Check if a scalar is valid.
func (emitter *yamlEmitter) analyzeScalar(value []byte) bool {
	var block_indicators,
		flow_indicators,
		line_breaks,
		special_characters,
		tab_characters,

		leading_space,
		leading_break,
		trailing_space,
		trailing_break,
		break_space,
		space_break,

		preceded_by_whitespace,
		followed_by_whitespace,
		previous_space,
		previous_break bool

	emitter.scalar_data.value = value

	if len(value) == 0 {
		emitter.scalar_data.multiline = false
		emitter.scalar_data.flow_plain_allowed = false
		emitter.scalar_data.block_plain_allowed = true
		emitter.scalar_data.single_quoted_allowed = true
		emitter.scalar_data.block_allowed = false
		return true
	}

	if len(value) >= 3 && ((value[0] == '-' && value[1] == '-' && value[2] == '-') || (value[0] == '.' && value[1] == '.' && value[2] == '.')) {
		block_indicators = true
		flow_indicators = true
	}

	preceded_by_whitespace = true
	for i, w := 0, 0; i < len(value); i += w {
		w = width(value[i])
		followed_by_whitespace = i+w >= len(value) || isBlank(value, i+w)

		if i == 0 {
			switch value[i] {
			case '#', ',', '[', ']', '{', '}', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`':
				flow_indicators = true
				block_indicators = true
			case '?', ':':
				flow_indicators = true
				if followed_by_whitespace {
					block_indicators = true
				}
			case '-':
				if followed_by_whitespace {
					flow_indicators = true
					block_indicators = true
				}
			}
		} else {
			switch value[i] {
			case ',', '?', '[', ']', '{', '}':
				flow_indicators = true
			case ':':
				flow_indicators = true
				if followed_by_whitespace {
					block_indicators = true
				}
			case '#':
				if preceded_by_whitespace {
					flow_indicators = true
					block_indicators = true
				}
			}
		}

		if value[i] == '\t' {
			tab_characters = true
		} else if !isPrintable(value, i) || !isASCII(value, i) && !emitter.unicode {
			special_characters = true
		}
		if isSpace(value, i) {
			if i == 0 {
				leading_space = true
			}
			if i+width(value[i]) == len(value) {
				trailing_space = true
			}
			if previous_break {
				break_space = true
			}
			previous_space = true
			previous_break = false
		} else if isLineBreak(value, i) {
			line_breaks = true
			if i == 0 {
				leading_break = true
			}
			if i+width(value[i]) == len(value) {
				trailing_break = true
			}
			if previous_space {
				space_break = true
			}
			previous_space = false
			previous_break = true
		} else {
			previous_space = false
			previous_break = false
		}

		// [Go]: Why 'z'? Couldn't be the end of the string as that's the loop condition.
		preceded_by_whitespace = isBlankOrZero(value, i)
	}

	emitter.scalar_data.multiline = line_breaks
	emitter.scalar_data.flow_plain_allowed = true
	emitter.scalar_data.block_plain_allowed = true
	emitter.scalar_data.single_quoted_allowed = true
	emitter.scalar_data.block_allowed = true

	if leading_space || leading_break || trailing_space || trailing_break {
		emitter.scalar_data.flow_plain_allowed = false
		emitter.scalar_data.block_plain_allowed = false
	}
	if trailing_space {
		emitter.scalar_data.block_allowed = false
	}
	if break_space {
		emitter.scalar_data.flow_plain_allowed = false
		emitter.scalar_data.block_plain_allowed = false
		emitter.scalar_data.single_quoted_allowed = false
	}
	if space_break || tab_characters || special_characters {
		emitter.scalar_data.flow_plain_allowed = false
		emitter.scalar_data.block_plain_allowed = false
		emitter.scalar_data.single_quoted_allowed = false
	}
	if space_break || special_characters {
		emitter.scalar_data.block_allowed = false
	}
	if line_breaks {
		emitter.scalar_data.flow_plain_allowed = false
		emitter.scalar_data.block_plain_allowed = false
	}
	if flow_indicators {
		emitter.scalar_data.flow_plain_allowed = false
	}
	if block_indicators {
		emitter.scalar_data.block_plain_allowed = false
	}
	return true
}

// Check if the event data is valid.
func (emitter *yamlEmitter) analyzeEvent(event *yamlEvent) bool {
	emitter.anchor_data.anchor = nil
	emitter.tag_data.handle = nil
	emitter.tag_data.suffix = nil
	emitter.scalar_data.value = nil

	if len(event.head_comment) > 0 {
		emitter.head_comment = event.head_comment
	}
	if len(event.line_comment) > 0 {
		emitter.line_comment = event.line_comment
	}
	if len(event.foot_comment) > 0 {
		emitter.foot_comment = event.foot_comment
	}
	if len(event.tail_comment) > 0 {
		emitter.tail_comment = event.tail_comment
	}

	switch event.typ {
	case yaml_ALIAS_EVENT:
		if !emitter.analyzeAnchor(event.anchor, true) {
			return false
		}

	case yaml_SCALAR_EVENT:
		if len(event.anchor) > 0 {
			if !emitter.analyzeAnchor(event.anchor, false) {
				return false
			}
		}
		if len(event.tag) > 0 && (emitter.canonical || (!event.implicit && !event.quoted_implicit)) {
			if !emitter.analyzeTag(event.tag) {
				return false
			}
		}
		if !emitter.analyzeScalar(event.value) {
			return false
		}

	case yaml_SEQUENCE_START_EVENT:
		if len(event.anchor) > 0 {
			if !emitter.analyzeAnchor(event.anchor, false) {
				return false
			}
		}
		if len(event.tag) > 0 && (emitter.canonical || !event.implicit) {
			if !emitter.analyzeTag(event.tag) {
				return false
			}
		}

	case yaml_MAPPING_START_EVENT:
		if len(event.anchor) > 0 {
			if !emitter.analyzeAnchor(event.anchor, false) {
				return false
			}
		}
		if len(event.tag) > 0 && (emitter.canonical || !event.implicit) {
			if !emitter.analyzeTag(event.tag) {
				return false
			}
		}
	}
	return true
}

// Write the BOM character.
func (emitter *yamlEmitter) writeBom() bool {
	if !emitter.flushIfNeeded() {
		return false
	}
	pos := emitter.buffer_pos
	emitter.buffer[pos+0] = '\xEF'
	emitter.buffer[pos+1] = '\xBB'
	emitter.buffer[pos+2] = '\xBF'
	emitter.buffer_pos += 3
	return true
}

func (emitter *yamlEmitter) writeIndent() bool {
	indent := emitter.indent
	if indent < 0 {
		indent = 0
	}
	if !emitter.indention || emitter.column > indent || (emitter.column == indent && !emitter.whitespace) {
		if !emitter.putLineBreak() {
			return false
		}
	}
	if emitter.foot_indent == indent {
		if !emitter.putLineBreak() {
			return false
		}
	}
	for emitter.column < indent {
		if !emitter.put(' ') {
			return false
		}
	}
	emitter.whitespace = true
	// emitter.indention = true
	emitter.space_above = false
	emitter.foot_indent = -1
	return true
}

func (emitter *yamlEmitter) writeIndicator(indicator []byte, need_whitespace, is_whitespace, is_indention bool) bool {
	if need_whitespace && !emitter.whitespace {
		if !emitter.put(' ') {
			return false
		}
	}
	if !emitter.writeAll(indicator) {
		return false
	}
	emitter.whitespace = is_whitespace
	emitter.indention = (emitter.indention && is_indention)
	emitter.open_ended = false
	return true
}

func (emitter *yamlEmitter) writeAnchor(value []byte) bool {
	if !emitter.writeAll(value) {
		return false
	}
	emitter.whitespace = false
	emitter.indention = false
	return true
}

func (emitter *yamlEmitter) writeTagHandle(value []byte) bool {
	if !emitter.whitespace {
		if !emitter.put(' ') {
			return false
		}
	}
	if !emitter.writeAll(value) {
		return false
	}
	emitter.whitespace = false
	emitter.indention = false
	return true
}

func (emitter *yamlEmitter) writeTagContent(value []byte, need_whitespace bool) bool {
	if need_whitespace && !emitter.whitespace {
		if !emitter.put(' ') {
			return false
		}
	}
	for i := 0; i < len(value); {
		var must_write bool
		switch value[i] {
		case ';', '/', '?', ':', '@', '&', '=', '+', '$', ',', '_', '.', '~', '*', '\'', '(', ')', '[', ']':
			must_write = true
		default:
			must_write = isAlpha(value, i)
		}
		if must_write {
			if !emitter.write(value, &i) {
				return false
			}
		} else {
			w := width(value[i])
			for k := 0; k < w; k++ {
				octet := value[i]
				i++
				if !emitter.put('%') {
					return false
				}

				c := octet >> 4
				if c < 10 {
					c += '0'
				} else {
					c += 'A' - 10
				}
				if !emitter.put(c) {
					return false
				}

				c = octet & 0x0f
				if c < 10 {
					c += '0'
				} else {
					c += 'A' - 10
				}
				if !emitter.put(c) {
					return false
				}
			}
		}
	}
	emitter.whitespace = false
	emitter.indention = false
	return true
}

func (emitter *yamlEmitter) writePlainScalar(value []byte, allow_breaks bool) bool {
	if len(value) > 0 && !emitter.whitespace {
		if !emitter.put(' ') {
			return false
		}
	}

	spaces := false
	breaks := false
	for i := 0; i < len(value); {
		if isSpace(value, i) {
			if allow_breaks && !spaces && emitter.column > emitter.best_width && !isSpace(value, i+1) {
				if !emitter.writeIndent() {
					return false
				}
				i += width(value[i])
			} else {
				if !emitter.write(value, &i) {
					return false
				}
			}
			spaces = true
		} else if isLineBreak(value, i) {
			if !breaks && value[i] == '\n' {
				if !emitter.putLineBreak() {
					return false
				}
			}
			if !emitter.writeLineBreak(value, &i) {
				return false
			}
			// emitter.indention = true
			breaks = true
		} else {
			if breaks {
				if !emitter.writeIndent() {
					return false
				}
			}
			if !emitter.write(value, &i) {
				return false
			}
			emitter.indention = false
			spaces = false
			breaks = false
		}
	}

	if len(value) > 0 {
		emitter.whitespace = false
	}
	emitter.indention = false
	if emitter.root_context {
		emitter.open_ended = true
	}

	return true
}

func (emitter *yamlEmitter) writeSingleQuotedScalar(value []byte, allow_breaks bool) bool {
	if !emitter.writeIndicator([]byte{'\''}, true, false, false) {
		return false
	}

	spaces := false
	breaks := false
	for i := 0; i < len(value); {
		if isSpace(value, i) {
			if allow_breaks && !spaces && emitter.column > emitter.best_width && i > 0 && i < len(value)-1 && !isSpace(value, i+1) {
				if !emitter.writeIndent() {
					return false
				}
				i += width(value[i])
			} else {
				if !emitter.write(value, &i) {
					return false
				}
			}
			spaces = true
		} else if isLineBreak(value, i) {
			if !breaks && value[i] == '\n' {
				if !emitter.putLineBreak() {
					return false
				}
			}
			if !emitter.writeLineBreak(value, &i) {
				return false
			}
			// emitter.indention = true
			breaks = true
		} else {
			if breaks {
				if !emitter.writeIndent() {
					return false
				}
			}
			if value[i] == '\'' {
				if !emitter.put('\'') {
					return false
				}
			}
			if !emitter.write(value, &i) {
				return false
			}
			emitter.indention = false
			spaces = false
			breaks = false
		}
	}
	if !emitter.writeIndicator([]byte{'\''}, false, false, false) {
		return false
	}
	emitter.whitespace = false
	emitter.indention = false
	return true
}

func (emitter *yamlEmitter) writeDoubleQuotedScalar(value []byte, allow_breaks bool) bool {
	spaces := false
	if !emitter.writeIndicator([]byte{'"'}, true, false, false) {
		return false
	}

	for i := 0; i < len(value); {
		if !isPrintable(value, i) || (!emitter.unicode && !isASCII(value, i)) ||
			isBOM(value, i) || isLineBreak(value, i) ||
			value[i] == '"' || value[i] == '\\' {

			octet := value[i]

			var w int
			var v rune
			switch {
			case octet&0x80 == 0x00:
				w, v = 1, rune(octet&0x7F)
			case octet&0xE0 == 0xC0:
				w, v = 2, rune(octet&0x1F)
			case octet&0xF0 == 0xE0:
				w, v = 3, rune(octet&0x0F)
			case octet&0xF8 == 0xF0:
				w, v = 4, rune(octet&0x07)
			}
			for k := 1; k < w; k++ {
				octet = value[i+k]
				v = (v << 6) + (rune(octet) & 0x3F)
			}
			i += w

			if !emitter.put('\\') {
				return false
			}

			var ok bool
			switch v {
			case 0x00:
				ok = emitter.put('0')
			case 0x07:
				ok = emitter.put('a')
			case 0x08:
				ok = emitter.put('b')
			case 0x09:
				ok = emitter.put('t')
			case 0x0A:
				ok = emitter.put('n')
			case 0x0b:
				ok = emitter.put('v')
			case 0x0c:
				ok = emitter.put('f')
			case 0x0d:
				ok = emitter.put('r')
			case 0x1b:
				ok = emitter.put('e')
			case 0x22:
				ok = emitter.put('"')
			case 0x5c:
				ok = emitter.put('\\')
			case 0x85:
				ok = emitter.put('N')
			case 0xA0:
				ok = emitter.put('_')
			case 0x2028:
				ok = emitter.put('L')
			case 0x2029:
				ok = emitter.put('P')
			default:
				if v <= 0xFF {
					ok = emitter.put('x')
					w = 2
				} else if v <= 0xFFFF {
					ok = emitter.put('u')
					w = 4
				} else {
					ok = emitter.put('U')
					w = 8
				}
				for k := (w - 1) * 4; ok && k >= 0; k -= 4 {
					digit := byte((v >> uint(k)) & 0x0F)
					if digit < 10 {
						ok = emitter.put(digit + '0')
					} else {
						ok = emitter.put(digit + 'A' - 10)
					}
				}
			}
			if !ok {
				return false
			}
			spaces = false
		} else if isSpace(value, i) {
			if allow_breaks && !spaces && emitter.column > emitter.best_width && i > 0 && i < len(value)-1 {
				if !emitter.writeIndent() {
					return false
				}
				if isSpace(value, i+1) {
					if !emitter.put('\\') {
						return false
					}
				}
				i += width(value[i])
			} else if !emitter.write(value, &i) {
				return false
			}
			spaces = true
		} else {
			if !emitter.write(value, &i) {
				return false
			}
			spaces = false
		}
	}
	if !emitter.writeIndicator([]byte{'"'}, false, false, false) {
		return false
	}
	emitter.whitespace = false
	emitter.indention = false
	return true
}

func (emitter *yamlEmitter) writeBlockScalarHints(value []byte) bool {
	if isSpace(value, 0) || isLineBreak(value, 0) {
		indent_hint := []byte{'0' + byte(emitter.best_indent)}
		if !emitter.writeIndicator(indent_hint, false, false, false) {
			return false
		}
	}

	emitter.open_ended = false

	var chomp_hint [1]byte
	if len(value) == 0 {
		chomp_hint[0] = '-'
	} else {
		i := len(value) - 1
		for value[i]&0xC0 == 0x80 {
			i--
		}
		if !isLineBreak(value, i) {
			chomp_hint[0] = '-'
		} else if i == 0 {
			chomp_hint[0] = '+'
			emitter.open_ended = true
		} else {
			i--
			for value[i]&0xC0 == 0x80 {
				i--
			}
			if isLineBreak(value, i) {
				chomp_hint[0] = '+'
				emitter.open_ended = true
			}
		}
	}
	if chomp_hint[0] != 0 {
		if !emitter.writeIndicator(chomp_hint[:], false, false, false) {
			return false
		}
	}
	return true
}

func (emitter *yamlEmitter) writeLiteralScalar(value []byte) bool {
	if !emitter.writeIndicator([]byte{'|'}, true, false, false) {
		return false
	}
	if !emitter.writeBlockScalarHints(value) {
		return false
	}
	if !emitter.processLineCommentLinebreak(true) {
		return false
	}
	// emitter.indention = true
	emitter.whitespace = true
	breaks := true
	for i := 0; i < len(value); {
		if isLineBreak(value, i) {
			if !emitter.writeLineBreak(value, &i) {
				return false
			}
			// emitter.indention = true
			breaks = true
		} else {
			if breaks {
				if !emitter.writeIndent() {
					return false
				}
			}
			if !emitter.write(value, &i) {
				return false
			}
			emitter.indention = false
			breaks = false
		}
	}

	return true
}

func (emitter *yamlEmitter) writeFoldedScalar(value []byte) bool {
	if !emitter.writeIndicator([]byte{'>'}, true, false, false) {
		return false
	}
	if !emitter.writeBlockScalarHints(value) {
		return false
	}
	if !emitter.processLineCommentLinebreak(true) {
		return false
	}

	// emitter.indention = true
	emitter.whitespace = true

	breaks := true
	leading_spaces := true
	for i := 0; i < len(value); {
		if isLineBreak(value, i) {
			if !breaks && !leading_spaces && value[i] == '\n' {
				k := 0
				for isLineBreak(value, k) {
					k += width(value[k])
				}
				if !isBlankOrZero(value, k) {
					if !emitter.putLineBreak() {
						return false
					}
				}
			}
			if !emitter.writeLineBreak(value, &i) {
				return false
			}
			// emitter.indention = true
			breaks = true
		} else {
			if breaks {
				if !emitter.writeIndent() {
					return false
				}
				leading_spaces = isBlank(value, i)
			}
			if !breaks && isSpace(value, i) && !isSpace(value, i+1) && emitter.column > emitter.best_width {
				if !emitter.writeIndent() {
					return false
				}
				i += width(value[i])
			} else {
				if !emitter.write(value, &i) {
					return false
				}
			}
			emitter.indention = false
			breaks = false
		}
	}
	return true
}

func (emitter *yamlEmitter) writeComment(comment []byte) bool {
	breaks := false
	pound := false
	for i := 0; i < len(comment); {
		if isLineBreak(comment, i) {
			if !emitter.writeLineBreak(comment, &i) {
				return false
			}
			// emitter.indention = true
			breaks = true
			pound = false
		} else {
			if breaks && !emitter.writeIndent() {
				return false
			}
			if !pound {
				if comment[i] != '#' && (!emitter.put('#') || !emitter.put(' ')) {
					return false
				}
				pound = true
			}
			if !emitter.write(comment, &i) {
				return false
			}
			emitter.indention = false
			breaks = false
		}
	}
	if !breaks && !emitter.putLineBreak() {
		return false
	}

	emitter.whitespace = true
	// emitter.indention = true
	return true
}
