// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestScalarStyleString(t *testing.T) {
	tests := []struct {
		style    ScalarStyle
		expected string
	}{
		{PLAIN_SCALAR_STYLE, "Plain"},
		{SINGLE_QUOTED_SCALAR_STYLE, "Single"},
		{DOUBLE_QUOTED_SCALAR_STYLE, "Double"},
		{LITERAL_SCALAR_STYLE, "Literal"},
		{FOLDED_SCALAR_STYLE, "Folded"},
		{ANY_SCALAR_STYLE, ""},
		{ScalarStyle(99), ""},
	}

	for _, tt := range tests {
		got := tt.style.String()
		assert.Equalf(t, tt.expected, got, "ScalarStyle(%d).String() = %q, want %q", tt.style, got, tt.expected)
	}
}

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		typ      TokenType
		expected string
	}{
		{NO_TOKEN, "NO_TOKEN"},
		{STREAM_START_TOKEN, "STREAM_START_TOKEN"},
		{STREAM_END_TOKEN, "STREAM_END_TOKEN"},
		{VERSION_DIRECTIVE_TOKEN, "VERSION_DIRECTIVE_TOKEN"},
		{TAG_DIRECTIVE_TOKEN, "TAG_DIRECTIVE_TOKEN"},
		{DOCUMENT_START_TOKEN, "DOCUMENT_START_TOKEN"},
		{DOCUMENT_END_TOKEN, "DOCUMENT_END_TOKEN"},
		{BLOCK_SEQUENCE_START_TOKEN, "BLOCK_SEQUENCE_START_TOKEN"},
		{BLOCK_MAPPING_START_TOKEN, "BLOCK_MAPPING_START_TOKEN"},
		{BLOCK_END_TOKEN, "BLOCK_END_TOKEN"},
		{FLOW_SEQUENCE_START_TOKEN, "FLOW_SEQUENCE_START_TOKEN"},
		{FLOW_SEQUENCE_END_TOKEN, "FLOW_SEQUENCE_END_TOKEN"},
		{FLOW_MAPPING_START_TOKEN, "FLOW_MAPPING_START_TOKEN"},
		{FLOW_MAPPING_END_TOKEN, "FLOW_MAPPING_END_TOKEN"},
		{BLOCK_ENTRY_TOKEN, "BLOCK_ENTRY_TOKEN"},
		{FLOW_ENTRY_TOKEN, "FLOW_ENTRY_TOKEN"},
		{KEY_TOKEN, "KEY_TOKEN"},
		{VALUE_TOKEN, "VALUE_TOKEN"},
		{ALIAS_TOKEN, "ALIAS_TOKEN"},
		{ANCHOR_TOKEN, "ANCHOR_TOKEN"},
		{TAG_TOKEN, "TAG_TOKEN"},
		{SCALAR_TOKEN, "SCALAR_TOKEN"},
		{TokenType(99), "<unknown token>"},
	}

	for _, tt := range tests {
		got := tt.typ.String()
		assert.Equalf(t, tt.expected, got, "TokenType(%d).String() = %q, want %q", tt.typ, got, tt.expected)
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		typ      EventType
		expected string
	}{
		{NO_EVENT, "none"},
		{STREAM_START_EVENT, "stream start"},
		{STREAM_END_EVENT, "stream end"},
		{DOCUMENT_START_EVENT, "document start"},
		{DOCUMENT_END_EVENT, "document end"},
		{ALIAS_EVENT, "alias"},
		{SCALAR_EVENT, "scalar"},
		{SEQUENCE_START_EVENT, "sequence start"},
		{SEQUENCE_END_EVENT, "sequence end"},
		{MAPPING_START_EVENT, "mapping start"},
		{MAPPING_END_EVENT, "mapping end"},
		{TAIL_COMMENT_EVENT, "tail comment"},
		{EventType(99), "unknown event 99"},
	}

	for _, tt := range tests {
		got := tt.typ.String()
		assert.Equalf(t, tt.expected, got, "EventType(%d).String() = %q, want %q", tt.typ, got, tt.expected)
	}
}

func TestParserStateString(t *testing.T) {
	tests := []struct {
		state    ParserState
		expected string
	}{
		{PARSE_STREAM_START_STATE, "PARSE_STREAM_START_STATE"},
		{PARSE_IMPLICIT_DOCUMENT_START_STATE, "PARSE_IMPLICIT_DOCUMENT_START_STATE"},
		{PARSE_DOCUMENT_START_STATE, "PARSE_DOCUMENT_START_STATE"},
		{PARSE_DOCUMENT_CONTENT_STATE, "PARSE_DOCUMENT_CONTENT_STATE"},
		{PARSE_DOCUMENT_END_STATE, "PARSE_DOCUMENT_END_STATE"},
		{PARSE_BLOCK_NODE_STATE, "PARSE_BLOCK_NODE_STATE"},
		{PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE, "PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE"},
		{PARSE_BLOCK_SEQUENCE_ENTRY_STATE, "PARSE_BLOCK_SEQUENCE_ENTRY_STATE"},
		{PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE, "PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE"},
		{PARSE_BLOCK_MAPPING_FIRST_KEY_STATE, "PARSE_BLOCK_MAPPING_FIRST_KEY_STATE"},
		{PARSE_BLOCK_MAPPING_KEY_STATE, "PARSE_BLOCK_MAPPING_KEY_STATE"},
		{PARSE_BLOCK_MAPPING_VALUE_STATE, "PARSE_BLOCK_MAPPING_VALUE_STATE"},
		{PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE, "PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE"},
		{PARSE_FLOW_SEQUENCE_ENTRY_STATE, "PARSE_FLOW_SEQUENCE_ENTRY_STATE"},
		{PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE, "PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE"},
		{PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE, "PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE"},
		{PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE, "PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE"},
		{PARSE_FLOW_MAPPING_FIRST_KEY_STATE, "PARSE_FLOW_MAPPING_FIRST_KEY_STATE"},
		{PARSE_FLOW_MAPPING_KEY_STATE, "PARSE_FLOW_MAPPING_KEY_STATE"},
		{PARSE_FLOW_MAPPING_VALUE_STATE, "PARSE_FLOW_MAPPING_VALUE_STATE"},
		{PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE, "PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE"},
		{PARSE_END_STATE, "PARSE_END_STATE"},
		{ParserState(99), "<unknown parser state>"},
	}

	for _, tt := range tests {
		got := tt.state.String()
		assert.Equalf(t, tt.expected, got, "ParserState(%d).String() = %q, want %q", tt.state, got, tt.expected)
	}
}

func TestEventStyleAccessors(t *testing.T) {
	event := Event{Style: Style(DOUBLE_QUOTED_SCALAR_STYLE)}

	got := event.ScalarStyle()
	assert.Equalf(t, DOUBLE_QUOTED_SCALAR_STYLE, got, "Event.ScalarStyle() = %v, want %v", got, DOUBLE_QUOTED_SCALAR_STYLE)

	event.Style = Style(FLOW_SEQUENCE_STYLE)
	got2 := event.SequenceStyle()
	assert.Equalf(t, FLOW_SEQUENCE_STYLE, got2, "Event.SequenceStyle() = %v, want %v", got2, FLOW_SEQUENCE_STYLE)

	event.Style = Style(BLOCK_MAPPING_STYLE)
	got3 := event.MappingStyle()
	assert.Equalf(t, BLOCK_MAPPING_STYLE, got3, "Event.MappingStyle() = %v, want %v", got3, BLOCK_MAPPING_STYLE)
}
