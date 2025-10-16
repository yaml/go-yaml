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
	"fmt"
)

// Flush the buffer if needed.
func (emitter *Emitter) flushIfNeeded() bool {
	if emitter.buffer_pos+5 >= len(emitter.buffer) {
		return emitter.flush()
	}
	return true
}

// Put a character to the output buffer.
func (emitter *Emitter) put(value byte) bool {
	if emitter.buffer_pos+5 >= len(emitter.buffer) && !emitter.flush() {
		return false
	}
	emitter.buffer[emitter.buffer_pos] = value
	emitter.buffer_pos++
	emitter.column++
	return true
}

// Put a line break to the output buffer.
func (emitter *Emitter) putLineBreak() bool {
	if emitter.buffer_pos+5 >= len(emitter.buffer) && !emitter.flush() {
		return false
	}
	switch emitter.line_break {
	case CR_BREAK:
		emitter.buffer[emitter.buffer_pos] = '\r'
		emitter.buffer_pos += 1
	case LN_BREAK:
		emitter.buffer[emitter.buffer_pos] = '\n'
		emitter.buffer_pos += 1
	case CRLN_BREAK:
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
func (emitter *Emitter) write(s []byte, i *int) bool {
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
func (emitter *Emitter) writeAll(s []byte) bool {
	for i := 0; i < len(s); {
		if !emitter.write(s, &i) {
			return false
		}
	}
	return true
}

// Copy a line break character from a string into buffer.
func (emitter *Emitter) writeLineBreak(s []byte, i *int) bool {
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
func (emitter *Emitter) setEmitterError(problem string) bool {
	emitter.ErrorType = EMITTER_ERROR
	emitter.Problem = problem
	return false
}

// Emit an event.
func (emitter *Emitter) Emit(event *Event) bool {
	emitter.events = append(emitter.events, *event)
	for !emitter.needMoreEvents() {
		event := &emitter.events[emitter.events_head]
		if !emitter.analyzeEvent(event) {
			return false
		}
		if !emitter.stateMachine(event) {
			return false
		}
		event.Delete()
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
func (emitter *Emitter) needMoreEvents() bool {
	if emitter.events_head == len(emitter.events) {
		return true
	}
	var accumulate int
	switch emitter.events[emitter.events_head].Type {
	case DOCUMENT_START_EVENT:
		accumulate = 1
	case SEQUENCE_START_EVENT:
		accumulate = 2
	case MAPPING_START_EVENT:
		accumulate = 3
	default:
		return false
	}
	if len(emitter.events)-emitter.events_head > accumulate {
		return false
	}
	var level int
	for i := emitter.events_head; i < len(emitter.events); i++ {
		switch emitter.events[i].Type {
		case STREAM_START_EVENT, DOCUMENT_START_EVENT, SEQUENCE_START_EVENT, MAPPING_START_EVENT:
			level++
		case STREAM_END_EVENT, DOCUMENT_END_EVENT, SEQUENCE_END_EVENT, MAPPING_END_EVENT:
			level--
		}
		if level == 0 {
			return false
		}
	}
	return true
}

// Append a directive to the directives stack.
func (emitter *Emitter) appendTagDirective(value *TagDirective, allow_duplicates bool) bool {
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
	tag_copy := TagDirective{
		handle: make([]byte, len(value.handle)),
		prefix: make([]byte, len(value.prefix)),
	}
	copy(tag_copy.handle, value.handle)
	copy(tag_copy.prefix, value.prefix)
	emitter.tag_directives = append(emitter.tag_directives, tag_copy)
	return true
}

// Increase the indentation level.
func (emitter *Emitter) increaseIndentCompact(flow, indentless bool, compact_seq bool) bool {
	emitter.indents = append(emitter.indents, emitter.indent)
	if emitter.indent < 0 {
		if flow {
			emitter.indent = emitter.BestIndent
		} else {
			emitter.indent = 0
		}
	} else if !indentless {
		// [Go] This was changed so that indentations are more regular.
		if emitter.states[len(emitter.states)-1] == EMIT_BLOCK_SEQUENCE_ITEM_STATE {
			// The first indent inside a sequence will just skip the "- " indicator.
			emitter.indent += 2
		} else {
			// Everything else aligns to the chosen indentation.
			emitter.indent = emitter.BestIndent * ((emitter.indent + emitter.BestIndent) / emitter.BestIndent)
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
func (emitter *Emitter) stateMachine(event *Event) bool {
	switch emitter.state {
	default:
	case EMIT_STREAM_START_STATE:
		return emitter.emitStreamStart(event)

	case EMIT_FIRST_DOCUMENT_START_STATE:
		return emitter.emitDocumentStart(event, true)

	case EMIT_DOCUMENT_START_STATE:
		return emitter.emitDocumentStart(event, false)

	case EMIT_DOCUMENT_CONTENT_STATE:
		return emitter.emitDocumentContent(event)

	case EMIT_DOCUMENT_END_STATE:
		return emitter.emitDocumentEnd(event)

	case EMIT_FLOW_SEQUENCE_FIRST_ITEM_STATE:
		return emitter.emitFlowSequenceItem(event, true, false)

	case EMIT_FLOW_SEQUENCE_TRAIL_ITEM_STATE:
		return emitter.emitFlowSequenceItem(event, false, true)

	case EMIT_FLOW_SEQUENCE_ITEM_STATE:
		return emitter.emitFlowSequenceItem(event, false, false)

	case EMIT_FLOW_MAPPING_FIRST_KEY_STATE:
		return emitter.emitFlowMappingKey(event, true, false)

	case EMIT_FLOW_MAPPING_TRAIL_KEY_STATE:
		return emitter.emitFlowMappingKey(event, false, true)

	case EMIT_FLOW_MAPPING_KEY_STATE:
		return emitter.emitFlowMappingKey(event, false, false)

	case EMIT_FLOW_MAPPING_SIMPLE_VALUE_STATE:
		return emitter.emitFlowMappingValue(event, true)

	case EMIT_FLOW_MAPPING_VALUE_STATE:
		return emitter.emitFlowMappingValue(event, false)

	case EMIT_BLOCK_SEQUENCE_FIRST_ITEM_STATE:
		return emitter.emitBlockSequenceItem(event, true)

	case EMIT_BLOCK_SEQUENCE_ITEM_STATE:
		return emitter.emitBlockSequenceItem(event, false)

	case EMIT_BLOCK_MAPPING_FIRST_KEY_STATE:
		return emitter.emitBlockMappingKey(event, true)

	case EMIT_BLOCK_MAPPING_KEY_STATE:
		return emitter.emitBlockMappingKey(event, false)

	case EMIT_BLOCK_MAPPING_SIMPLE_VALUE_STATE:
		return emitter.emitBlockMappingValue(event, true)

	case EMIT_BLOCK_MAPPING_VALUE_STATE:
		return emitter.emitBlockMappingValue(event, false)

	case EMIT_END_STATE:
		return emitter.setEmitterError("expected nothing after STREAM-END")
	}
	panic("invalid emitter state")
}

// Expect STREAM-START.
func (emitter *Emitter) emitStreamStart(event *Event) bool {
	if event.Type != STREAM_START_EVENT {
		return emitter.setEmitterError("expected STREAM-START")
	}
	if emitter.encoding == ANY_ENCODING {
		emitter.encoding = event.encoding
		if emitter.encoding == ANY_ENCODING {
			emitter.encoding = UTF8_ENCODING
		}
	}
	if emitter.BestIndent < 2 || emitter.BestIndent > 9 {
		emitter.BestIndent = 2
	}
	if emitter.best_width >= 0 && emitter.best_width <= emitter.BestIndent*2 {
		emitter.best_width = 80
	}
	if emitter.best_width < 0 {
		emitter.best_width = 1<<31 - 1
	}
	if emitter.line_break == ANY_BREAK {
		emitter.line_break = LN_BREAK
	}

	emitter.indent = -1
	emitter.line = 0
	emitter.column = 0
	emitter.whitespace = true
	emitter.indention = true
	emitter.space_above = true
	emitter.foot_indent = -1

	if emitter.encoding != UTF8_ENCODING {
		if !emitter.writeBom() {
			return false
		}
	}
	emitter.state = EMIT_FIRST_DOCUMENT_START_STATE
	return true
}

// Expect DOCUMENT-START or STREAM-END.
func (emitter *Emitter) emitDocumentStart(event *Event, first bool) bool {
	if event.Type == DOCUMENT_START_EVENT {

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

		implicit := event.Implicit
		if !first || emitter.canonical {
			implicit = false
		}

		if emitter.OpenEnded && (event.version_directive != nil || len(event.tag_directives) > 0) {
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

		if len(emitter.HeadComment) > 0 {
			if !emitter.processHeadComment() {
				return false
			}
			if !emitter.putLineBreak() {
				return false
			}
		}

		emitter.state = EMIT_DOCUMENT_CONTENT_STATE
		return true
	}

	if event.Type == STREAM_END_EVENT {
		if emitter.OpenEnded {
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
		emitter.state = EMIT_END_STATE
		return true
	}

	return emitter.setEmitterError("expected DOCUMENT-START or STREAM-END")
}

// emitter preserves the original signature and delegates to
// increaseIndentCompact without compact-sequence indentation
func (emitter *Emitter) increaseIndent(flow, indentless bool) bool {
	return emitter.increaseIndentCompact(flow, indentless, false)
}

// processLineComment preserves the original signature and delegates to
// processLineCommentLinebreak passing false for linebreak
func (emitter *Emitter) processLineComment() bool {
	return emitter.processLineCommentLinebreak(false)
}

// Expect the root node.
func (emitter *Emitter) emitDocumentContent(event *Event) bool {
	emitter.states = append(emitter.states, EMIT_DOCUMENT_END_STATE)

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
func (emitter *Emitter) emitDocumentEnd(event *Event) bool {
	if event.Type != DOCUMENT_END_EVENT {
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
	if !event.Implicit {
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
	emitter.state = EMIT_DOCUMENT_START_STATE
	emitter.tag_directives = emitter.tag_directives[:0]
	return true
}

// Expect a flow item node.
func (emitter *Emitter) emitFlowSequenceItem(event *Event, first, trail bool) bool {
	if first {
		if !emitter.writeIndicator([]byte{'['}, true, true, false) {
			return false
		}
		if !emitter.increaseIndent(true, false) {
			return false
		}
		emitter.flow_level++
	}

	if event.Type == SEQUENCE_END_EVENT {
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
	if len(emitter.LineComment)+len(emitter.FootComment)+len(emitter.TailComment) > 0 {
		emitter.states = append(emitter.states, EMIT_FLOW_SEQUENCE_TRAIL_ITEM_STATE)
	} else {
		emitter.states = append(emitter.states, EMIT_FLOW_SEQUENCE_ITEM_STATE)
	}
	if !emitter.emitNode(event, false, true, false, false) {
		return false
	}
	if len(emitter.LineComment)+len(emitter.FootComment)+len(emitter.TailComment) > 0 {
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
func (emitter *Emitter) emitFlowMappingKey(event *Event, first, trail bool) bool {
	if first {
		if !emitter.writeIndicator([]byte{'{'}, true, true, false) {
			return false
		}
		if !emitter.increaseIndent(true, false) {
			return false
		}
		emitter.flow_level++
	}

	if event.Type == MAPPING_END_EVENT {
		if (emitter.canonical || len(emitter.HeadComment)+len(emitter.FootComment)+len(emitter.TailComment) > 0) && !first && !trail {
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
		emitter.states = append(emitter.states, EMIT_FLOW_MAPPING_SIMPLE_VALUE_STATE)
		return emitter.emitNode(event, false, false, true, true)
	}
	if !emitter.writeIndicator([]byte{'?'}, true, false, false) {
		return false
	}
	emitter.states = append(emitter.states, EMIT_FLOW_MAPPING_VALUE_STATE)
	return emitter.emitNode(event, false, false, true, false)
}

// Expect a flow value node.
func (emitter *Emitter) emitFlowMappingValue(event *Event, simple bool) bool {
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
	if len(emitter.LineComment)+len(emitter.FootComment)+len(emitter.TailComment) > 0 {
		emitter.states = append(emitter.states, EMIT_FLOW_MAPPING_TRAIL_KEY_STATE)
	} else {
		emitter.states = append(emitter.states, EMIT_FLOW_MAPPING_KEY_STATE)
	}
	if !emitter.emitNode(event, false, false, true, false) {
		return false
	}
	if len(emitter.LineComment)+len(emitter.FootComment)+len(emitter.TailComment) > 0 {
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
func (emitter *Emitter) emitBlockSequenceItem(event *Event, first bool) bool {
	if first {
		// emitter.mapping context tells us if we are currently in a mapping context.
		// emitter.column tells us which column we are in the yaml output. 0 is the first char of the column.
		// emitter.indentation tells us if the last character was an indentation character.
		// emitter.compact_sequence_indent tells us if '- ' is considered part of the indentation for sequence elements.
		// So, `seq` means that we are in a mapping context, and we are either at the first char of the column or
		//  the last character was not an indentation character, and we consider '- ' part of the indentation
		//  for sequence elements.
		seq := emitter.mapping_context && (emitter.column == 0 || !emitter.indention) &&
			emitter.CompactSequenceIndent
		if !emitter.increaseIndentCompact(false, false, seq) {
			return false
		}
	}
	if event.Type == SEQUENCE_END_EVENT {
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
	emitter.states = append(emitter.states, EMIT_BLOCK_SEQUENCE_ITEM_STATE)
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
func (emitter *Emitter) emitBlockMappingKey(event *Event, first bool) bool {
	if first {
		if !emitter.increaseIndent(false, false) {
			return false
		}
	}
	if !emitter.processHeadComment() {
		return false
	}
	if event.Type == MAPPING_END_EVENT {
		emitter.indent = emitter.indents[len(emitter.indents)-1]
		emitter.indents = emitter.indents[:len(emitter.indents)-1]
		emitter.state = emitter.states[len(emitter.states)-1]
		emitter.states = emitter.states[:len(emitter.states)-1]
		return true
	}
	if !emitter.writeIndent() {
		return false
	}
	if len(emitter.LineComment) > 0 {
		// [Go] A line comment was provided for the key. That's unusual as the
		//      scanner associates line comments with the value. Either way,
		//      save the line comment and render it appropriately later.
		emitter.key_line_comment = emitter.LineComment
		emitter.LineComment = nil
	}
	if emitter.checkSimpleKey() {
		emitter.states = append(emitter.states, EMIT_BLOCK_MAPPING_SIMPLE_VALUE_STATE)
		if !emitter.emitNode(event, false, false, true, true) {
			return false
		}

		if event.Type == ALIAS_EVENT {
			// make sure there's a space after the alias
			return emitter.put(' ')
		}

		return true
	}
	if !emitter.writeIndicator([]byte{'?'}, true, false, true) {
		return false
	}
	emitter.states = append(emitter.states, EMIT_BLOCK_MAPPING_VALUE_STATE)
	return emitter.emitNode(event, false, false, true, false)
}

// Expect a block value node.
func (emitter *Emitter) emitBlockMappingValue(event *Event, simple bool) bool {
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
		if event.Type == SCALAR_EVENT {
			if len(emitter.LineComment) == 0 {
				// A scalar is coming and it has no line comments by itself yet,
				// so just let it handle the line comment as usual. If it has a
				// line comment, we can't have both so the one from the key is lost.
				emitter.LineComment = emitter.key_line_comment
				emitter.key_line_comment = nil
			}
		} else if event.SequenceStyle() != FLOW_SEQUENCE_STYLE && (event.Type == MAPPING_START_EVENT || event.Type == SEQUENCE_START_EVENT) {
			// An indented block follows, so write the comment right now.
			emitter.LineComment, emitter.key_line_comment = emitter.key_line_comment, emitter.LineComment
			if !emitter.processLineComment() {
				return false
			}
			emitter.LineComment, emitter.key_line_comment = emitter.key_line_comment, emitter.LineComment
		}
	}
	emitter.states = append(emitter.states, EMIT_BLOCK_MAPPING_KEY_STATE)
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

func (emitter *Emitter) silentNilEvent(event *Event) bool {
	return event.Type == SCALAR_EVENT && event.Implicit && !emitter.canonical && len(emitter.scalar_data.value) == 0
}

// Expect a node.
func (emitter *Emitter) emitNode(event *Event,
	root bool, sequence bool, mapping bool, simple_key bool,
) bool {
	emitter.root_context = root
	emitter.sequence_context = sequence
	emitter.mapping_context = mapping
	emitter.simple_key_context = simple_key

	switch event.Type {
	case ALIAS_EVENT:
		return emitter.emitAlias(event)
	case SCALAR_EVENT:
		return emitter.emitScalar(event)
	case SEQUENCE_START_EVENT:
		return emitter.emitSequenceStart(event)
	case MAPPING_START_EVENT:
		return emitter.emitMappingStart(event)
	default:
		return emitter.setEmitterError(
			fmt.Sprintf("expected SCALAR, SEQUENCE-START, MAPPING-START, or ALIAS, but got %v", event.Type))
	}
}

// Expect ALIAS.
func (emitter *Emitter) emitAlias(event *Event) bool {
	if !emitter.processAnchor() {
		return false
	}
	emitter.state = emitter.states[len(emitter.states)-1]
	emitter.states = emitter.states[:len(emitter.states)-1]
	return true
}

// Expect SCALAR.
func (emitter *Emitter) emitScalar(event *Event) bool {
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
func (emitter *Emitter) emitSequenceStart(event *Event) bool {
	if !emitter.processAnchor() {
		return false
	}
	if !emitter.processTag() {
		return false
	}
	if emitter.flow_level > 0 || emitter.canonical || event.SequenceStyle() == FLOW_SEQUENCE_STYLE ||
		emitter.checkEmptySequence() {
		emitter.state = EMIT_FLOW_SEQUENCE_FIRST_ITEM_STATE
	} else {
		emitter.state = EMIT_BLOCK_SEQUENCE_FIRST_ITEM_STATE
	}
	return true
}

// Expect MAPPING-START.
func (emitter *Emitter) emitMappingStart(event *Event) bool {
	if !emitter.processAnchor() {
		return false
	}
	if !emitter.processTag() {
		return false
	}
	if emitter.flow_level > 0 || emitter.canonical || event.MappingStyle() == FLOW_MAPPING_STYLE ||
		emitter.checkEmptyMapping() {
		emitter.state = EMIT_FLOW_MAPPING_FIRST_KEY_STATE
	} else {
		emitter.state = EMIT_BLOCK_MAPPING_FIRST_KEY_STATE
	}
	return true
}

// Check if the document content is an empty scalar.
func (emitter *Emitter) checkEmptyDocument() bool {
	return false // [Go] Huh?
}

// Check if the next events represent an empty sequence.
func (emitter *Emitter) checkEmptySequence() bool {
	if len(emitter.events)-emitter.events_head < 2 {
		return false
	}
	return emitter.events[emitter.events_head].Type == SEQUENCE_START_EVENT &&
		emitter.events[emitter.events_head+1].Type == SEQUENCE_END_EVENT
}

// Check if the next events represent an empty mapping.
func (emitter *Emitter) checkEmptyMapping() bool {
	if len(emitter.events)-emitter.events_head < 2 {
		return false
	}
	return emitter.events[emitter.events_head].Type == MAPPING_START_EVENT &&
		emitter.events[emitter.events_head+1].Type == MAPPING_END_EVENT
}

// Check if the next node can be expressed as a simple key.
func (emitter *Emitter) checkSimpleKey() bool {
	length := 0
	switch emitter.events[emitter.events_head].Type {
	case ALIAS_EVENT:
		length += len(emitter.anchor_data.anchor)
	case SCALAR_EVENT:
		if emitter.scalar_data.multiline {
			return false
		}
		length += len(emitter.anchor_data.anchor) +
			len(emitter.tag_data.handle) +
			len(emitter.tag_data.suffix) +
			len(emitter.scalar_data.value)
	case SEQUENCE_START_EVENT:
		if !emitter.checkEmptySequence() {
			return false
		}
		length += len(emitter.anchor_data.anchor) +
			len(emitter.tag_data.handle) +
			len(emitter.tag_data.suffix)
	case MAPPING_START_EVENT:
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
func (emitter *Emitter) selectScalarStyle(event *Event) bool {
	no_tag := len(emitter.tag_data.handle) == 0 && len(emitter.tag_data.suffix) == 0
	if no_tag && !event.Implicit && !event.quoted_implicit {
		return emitter.setEmitterError("neither tag nor implicit flags are specified")
	}

	style := event.ScalarStyle()
	if style == ANY_SCALAR_STYLE {
		style = PLAIN_SCALAR_STYLE
	}
	if emitter.canonical {
		style = DOUBLE_QUOTED_SCALAR_STYLE
	}
	if emitter.simple_key_context && emitter.scalar_data.multiline {
		style = DOUBLE_QUOTED_SCALAR_STYLE
	}

	if style == PLAIN_SCALAR_STYLE {
		if emitter.flow_level > 0 && !emitter.scalar_data.flow_plain_allowed ||
			emitter.flow_level == 0 && !emitter.scalar_data.block_plain_allowed {
			style = SINGLE_QUOTED_SCALAR_STYLE
		}
		if len(emitter.scalar_data.value) == 0 && (emitter.flow_level > 0 || emitter.simple_key_context) {
			style = SINGLE_QUOTED_SCALAR_STYLE
		}
		if no_tag && !event.Implicit {
			style = SINGLE_QUOTED_SCALAR_STYLE
		}
	}
	if style == SINGLE_QUOTED_SCALAR_STYLE {
		if !emitter.scalar_data.single_quoted_allowed {
			style = DOUBLE_QUOTED_SCALAR_STYLE
		}
	}
	if style == LITERAL_SCALAR_STYLE || style == FOLDED_SCALAR_STYLE {
		if !emitter.scalar_data.block_allowed || emitter.flow_level > 0 || emitter.simple_key_context {
			style = DOUBLE_QUOTED_SCALAR_STYLE
		}
	}

	if no_tag && !event.quoted_implicit && style != PLAIN_SCALAR_STYLE {
		emitter.tag_data.handle = []byte{'!'}
	}
	emitter.scalar_data.style = style
	return true
}

// Write an anchor.
func (emitter *Emitter) processAnchor() bool {
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
func (emitter *Emitter) processTag() bool {
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
func (emitter *Emitter) processScalar() bool {
	switch emitter.scalar_data.style {
	case PLAIN_SCALAR_STYLE:
		return emitter.writePlainScalar(emitter.scalar_data.value, !emitter.simple_key_context)

	case SINGLE_QUOTED_SCALAR_STYLE:
		return emitter.writeSingleQuotedScalar(emitter.scalar_data.value, !emitter.simple_key_context)

	case DOUBLE_QUOTED_SCALAR_STYLE:
		return emitter.writeDoubleQuotedScalar(emitter.scalar_data.value, !emitter.simple_key_context)

	case LITERAL_SCALAR_STYLE:
		return emitter.writeLiteralScalar(emitter.scalar_data.value)

	case FOLDED_SCALAR_STYLE:
		return emitter.writeFoldedScalar(emitter.scalar_data.value)
	}
	panic("unknown scalar style")
}

// Write a head comment.
func (emitter *Emitter) processHeadComment() bool {
	if len(emitter.TailComment) > 0 {
		if !emitter.writeIndent() {
			return false
		}
		if !emitter.writeComment(emitter.TailComment) {
			return false
		}
		emitter.TailComment = emitter.TailComment[:0]
		emitter.foot_indent = emitter.indent
		if emitter.foot_indent < 0 {
			emitter.foot_indent = 0
		}
	}

	if len(emitter.HeadComment) == 0 {
		return true
	}
	if !emitter.writeIndent() {
		return false
	}
	if !emitter.writeComment(emitter.HeadComment) {
		return false
	}
	emitter.HeadComment = emitter.HeadComment[:0]
	return true
}

// Write an line comment.
func (emitter *Emitter) processLineCommentLinebreak(linebreak bool) bool {
	if len(emitter.LineComment) == 0 {
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
	if !emitter.writeComment(emitter.LineComment) {
		return false
	}
	emitter.LineComment = emitter.LineComment[:0]
	return true
}

// Write a foot comment.
func (emitter *Emitter) processFootComment() bool {
	if len(emitter.FootComment) == 0 {
		return true
	}
	if !emitter.writeIndent() {
		return false
	}
	if !emitter.writeComment(emitter.FootComment) {
		return false
	}
	emitter.FootComment = emitter.FootComment[:0]
	emitter.foot_indent = emitter.indent
	if emitter.foot_indent < 0 {
		emitter.foot_indent = 0
	}
	return true
}

// Check if a %YAML directive is valid.
func (emitter *Emitter) analyzeVersionDirective(version_directive *VersionDirective) bool {
	if version_directive.major != 1 || version_directive.minor != 1 {
		return emitter.setEmitterError("incompatible %YAML directive")
	}
	return true
}

// Check if a %TAG directive is valid.
func (emitter *Emitter) analyzeTagDirective(tag_directive *TagDirective) bool {
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
func (emitter *Emitter) analyzeAnchor(anchor []byte, alias bool) bool {
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
func (emitter *Emitter) analyzeTag(tag []byte) bool {
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
func (emitter *Emitter) analyzeScalar(value []byte) bool {
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
func (emitter *Emitter) analyzeEvent(event *Event) bool {
	emitter.anchor_data.anchor = nil
	emitter.tag_data.handle = nil
	emitter.tag_data.suffix = nil
	emitter.scalar_data.value = nil

	if len(event.HeadComment) > 0 {
		emitter.HeadComment = event.HeadComment
	}
	if len(event.LineComment) > 0 {
		emitter.LineComment = event.LineComment
	}
	if len(event.FootComment) > 0 {
		emitter.FootComment = event.FootComment
	}
	if len(event.TailComment) > 0 {
		emitter.TailComment = event.TailComment
	}

	switch event.Type {
	case ALIAS_EVENT:
		if !emitter.analyzeAnchor(event.Anchor, true) {
			return false
		}

	case SCALAR_EVENT:
		if len(event.Anchor) > 0 {
			if !emitter.analyzeAnchor(event.Anchor, false) {
				return false
			}
		}
		if len(event.Tag) > 0 && (emitter.canonical || (!event.Implicit && !event.quoted_implicit)) {
			if !emitter.analyzeTag(event.Tag) {
				return false
			}
		}
		if !emitter.analyzeScalar(event.Value) {
			return false
		}

	case SEQUENCE_START_EVENT:
		if len(event.Anchor) > 0 {
			if !emitter.analyzeAnchor(event.Anchor, false) {
				return false
			}
		}
		if len(event.Tag) > 0 && (emitter.canonical || !event.Implicit) {
			if !emitter.analyzeTag(event.Tag) {
				return false
			}
		}

	case MAPPING_START_EVENT:
		if len(event.Anchor) > 0 {
			if !emitter.analyzeAnchor(event.Anchor, false) {
				return false
			}
		}
		if len(event.Tag) > 0 && (emitter.canonical || !event.Implicit) {
			if !emitter.analyzeTag(event.Tag) {
				return false
			}
		}
	}
	return true
}

// Write the BOM character.
func (emitter *Emitter) writeBom() bool {
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

func (emitter *Emitter) writeIndent() bool {
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

func (emitter *Emitter) writeIndicator(indicator []byte, need_whitespace, is_whitespace, is_indention bool) bool {
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
	emitter.OpenEnded = false
	return true
}

func (emitter *Emitter) writeAnchor(value []byte) bool {
	if !emitter.writeAll(value) {
		return false
	}
	emitter.whitespace = false
	emitter.indention = false
	return true
}

func (emitter *Emitter) writeTagHandle(value []byte) bool {
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

func (emitter *Emitter) writeTagContent(value []byte, need_whitespace bool) bool {
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

func (emitter *Emitter) writePlainScalar(value []byte, allow_breaks bool) bool {
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
		emitter.OpenEnded = true
	}

	return true
}

func (emitter *Emitter) writeSingleQuotedScalar(value []byte, allow_breaks bool) bool {
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

func (emitter *Emitter) writeDoubleQuotedScalar(value []byte, allow_breaks bool) bool {
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

func (emitter *Emitter) writeBlockScalarHints(value []byte) bool {
	if isSpace(value, 0) || isLineBreak(value, 0) {
		indent_hint := []byte{'0' + byte(emitter.BestIndent)}
		if !emitter.writeIndicator(indent_hint, false, false, false) {
			return false
		}
	}

	emitter.OpenEnded = false

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
			emitter.OpenEnded = true
		} else {
			i--
			for value[i]&0xC0 == 0x80 {
				i--
			}
			if isLineBreak(value, i) {
				chomp_hint[0] = '+'
				emitter.OpenEnded = true
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

func (emitter *Emitter) writeLiteralScalar(value []byte) bool {
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

func (emitter *Emitter) writeFoldedScalar(value []byte) bool {
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

func (emitter *Emitter) writeComment(comment []byte) bool {
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
