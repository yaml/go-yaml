// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"bytes"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func parseEvents(input string) ([]EventType, bool) {
	parser := NewParser()
	parser.SetInputString([]byte(input))

	var types []EventType
	for {
		var event Event
		if !parser.Parse(&event) {
			if parser.ErrorType != NO_ERROR {
				return nil, false
			}
			return types, true
		}
		types = append(types, event.Type)
		if event.Type == STREAM_END_EVENT {
			break
		}
	}
	return types, true
}

func TestParserSimpleScalar(t *testing.T) {
	input := "hello"
	types, ok := parseEvents(input)
	assert.Truef(t, ok, "parseEvents() = %v, want true", ok)

	expected := []EventType{
		STREAM_START_EVENT,
		DOCUMENT_START_EVENT,
		SCALAR_EVENT,
		DOCUMENT_END_EVENT,
		STREAM_END_EVENT,
	}

	assert.Equalf(t, len(expected), len(types), "parseEvents() types length = %d, want %d", len(types), len(expected))
	for i, et := range expected {
		assert.Equalf(t, et, types[i], "parseEvents() types[%d] = %v, want %v", i, types[i], et)
	}
}

func TestParserSimpleMapping(t *testing.T) {
	input := "key: value"
	types, ok := parseEvents(input)
	assert.Truef(t, ok, "parseEvents() = %v, want true", ok)

	expected := []EventType{
		STREAM_START_EVENT,
		DOCUMENT_START_EVENT,
		MAPPING_START_EVENT,
		SCALAR_EVENT,
		SCALAR_EVENT,
		MAPPING_END_EVENT,
		DOCUMENT_END_EVENT,
		STREAM_END_EVENT,
	}

	assert.Equalf(t, len(expected), len(types), "parseEvents() types length = %d, want %d", len(types), len(expected))
	for i, et := range expected {
		assert.Equalf(t, et, types[i], "parseEvents() types[%d] = %v, want %v", i, types[i], et)
	}
}

func TestParserBlockSequence(t *testing.T) {
	input := "- item1\n- item2"
	types, ok := parseEvents(input)
	assert.Truef(t, ok, "parseEvents() = %v, want true", ok)

	expected := []EventType{
		STREAM_START_EVENT,
		DOCUMENT_START_EVENT,
		SEQUENCE_START_EVENT,
		SCALAR_EVENT,
		SCALAR_EVENT,
		SEQUENCE_END_EVENT,
		DOCUMENT_END_EVENT,
		STREAM_END_EVENT,
	}

	assert.Equalf(t, len(expected), len(types), "parseEvents() types length = %d, want %d", len(types), len(expected))
	for i, et := range expected {
		assert.Equalf(t, et, types[i], "event[%d] = %v, want %v", i, types[i], et)
	}
}

func TestParserFlowSequence(t *testing.T) {
	input := "[1, 2, 3]"
	types, ok := parseEvents(input)
	assert.Truef(t, ok, "parseEvents() = %v, want true", ok)

	expected := []EventType{
		STREAM_START_EVENT,
		DOCUMENT_START_EVENT,
		SEQUENCE_START_EVENT,
		SCALAR_EVENT,
		SCALAR_EVENT,
		SCALAR_EVENT,
		SEQUENCE_END_EVENT,
		DOCUMENT_END_EVENT,
		STREAM_END_EVENT,
	}

	assert.Equalf(t, len(expected), len(types), "parseEvents() types length = %d, want %d", len(types), len(expected))
	for i, et := range expected {
		assert.Equalf(t, et, types[i], "event[%d] = %v, want %v", i, types[i], et)
	}
}

func TestParserFlowMapping(t *testing.T) {
	input := "{a: 1, b: 2}"
	types, ok := parseEvents(input)
	assert.Truef(t, ok, "parseEvents() = %v, want true", ok)

	expected := []EventType{
		STREAM_START_EVENT,
		DOCUMENT_START_EVENT,
		MAPPING_START_EVENT,
		SCALAR_EVENT,
		SCALAR_EVENT,
		SCALAR_EVENT,
		SCALAR_EVENT,
		MAPPING_END_EVENT,
		DOCUMENT_END_EVENT,
		STREAM_END_EVENT,
	}

	assert.Equalf(t, len(expected), len(types), "parseEvents() types length = %d, want %d", len(types), len(expected))
	for i, et := range expected {
		assert.Equalf(t, et, types[i], "event[%d] = %v, want %v", i, types[i], et)
	}
}

func TestParserExplicitDocument(t *testing.T) {
	input := "---\nkey: value\n..."
	parser := NewParser()
	parser.SetInputString([]byte(input))

	events := []Event{}
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

	assert.Truef(t, len(events) >= 2, "parseEvents() events length = %d, want >= 2", len(events))

	docStartEvent := events[1]
	assert.Equalf(t, DOCUMENT_START_EVENT, docStartEvent.Type, "event[1] type = %v, want DOCUMENT_START_EVENT", docStartEvent.Type)
	assert.Falsef(t, docStartEvent.Implicit, "event[1] implicit = %v, want false", docStartEvent.Implicit)
}

func TestParserAnchorAndAlias(t *testing.T) {
	input := "- &anchor value\n- *anchor"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	var anchorValue []byte
	foundAlias := false

	for {
		var event Event
		if !parser.Parse(&event) {
			break
		}

		if event.Type == SCALAR_EVENT && len(event.Anchor) > 0 {
			anchorValue = event.Anchor
		}

		if event.Type == ALIAS_EVENT {
			foundAlias = true
			if len(anchorValue) > 0 {
				assert.DeepEqualf(t, anchorValue, event.Anchor, "ALIAS_EVENT Anchor = %q, want %q", event.Anchor, anchorValue)
			}
		}

		if event.Type == STREAM_END_EVENT {
			break
		}
	}

	assert.Truef(t, foundAlias, "Expected ALIAS_EVENT not found")
}

func TestParserTag(t *testing.T) {
	input := "!!str value"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	foundTag := false
	for {
		var event Event
		if !parser.Parse(&event) {
			break
		}

		if event.Type == SCALAR_EVENT && len(event.Tag) > 0 {
			foundTag = true
			expectedTag := []byte("tag:yaml.org,2002:str")
			assert.DeepEqualf(t, expectedTag, event.Tag, "SCALAR_EVENT Tag = %q, want %q", event.Tag, expectedTag)
		}

		if event.Type == STREAM_END_EVENT {
			break
		}
	}

	assert.Truef(t, foundTag, "Expected tag on SCALAR_EVENT not found")
}

func TestParserNestedStructures(t *testing.T) {
	input := `
parent:
  - item1
  - item2:
      nested: value
`
	types, ok := parseEvents(input)
	assert.Truef(t, ok, "parseEvents() failed")

	assert.Equalf(t, STREAM_START_EVENT, types[0], "First event should be STREAM_START_EVENT, got %v", types[0])

	hasMapping := false
	hasSequence := false
	for _, et := range types {
		if et == MAPPING_START_EVENT {
			hasMapping = true
		}
		if et == SEQUENCE_START_EVENT {
			hasSequence = true
		}
	}

	assert.Truef(t, hasMapping, "Expected MAPPING_START_EVENT not found")
	assert.Truef(t, hasSequence, "Expected SEQUENCE_START_EVENT not found")
}

func TestParserMultipleDocuments(t *testing.T) {
	input := "---\ndoc1\n---\ndoc2"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	docCount := 0
	for {
		var event Event
		if !parser.Parse(&event) {
			break
		}
		if event.Type == DOCUMENT_START_EVENT {
			docCount++
		}
		if event.Type == STREAM_END_EVENT {
			break
		}
	}

	assert.Equalf(t, 2, docCount, "Expected 2 documents, got %d", docCount)
}

func TestParserScalarValue(t *testing.T) {
	input := "key: hello world"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	foundValue := false
	for {
		var event Event
		if !parser.Parse(&event) {
			break
		}

		if event.Type == SCALAR_EVENT && bytes.Equal(event.Value, []byte("hello world")) {
			foundValue = true
		}

		if event.Type == STREAM_END_EVENT {
			break
		}
	}

	assert.Truef(t, foundValue, "Expected scalar value 'hello world' not found")
}

func TestParserEmptyInput(t *testing.T) {
	input := ""
	types, ok := parseEvents(input)
	assert.Truef(t, ok, "parseEvents() failed")

	expected := []EventType{
		STREAM_START_EVENT,
		STREAM_END_EVENT,
	}

	assert.Equalf(t, len(expected), len(types), "parseEvents() got %d events, want %d", len(types), len(expected))
	for i, et := range expected {
		assert.Equalf(t, et, types[i], "parseEvents() event[%d] = %v, want %v", i, types[i], et)
	}
}

func TestParserImplicitDocument(t *testing.T) {
	input := "value"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	events := []Event{}
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

	assert.Truef(t, len(events) >= 2, "Expected at least 2 events, got %d", len(events))

	docStartEvent := events[1]
	assert.Equalf(t, DOCUMENT_START_EVENT, docStartEvent.Type, "event[1] type = %v, want DOCUMENT_START_EVENT", docStartEvent.Type)
	assert.Truef(t, docStartEvent.Implicit, "event[1] implicit = %v, want true", docStartEvent.Implicit)
}

func TestParserComplexMapping(t *testing.T) {
	input := `
? key1
: value1
? key2
: value2
`
	types, ok := parseEvents(input)
	assert.Truef(t, ok, "parseEvents() failed")

	hasMapping := false
	scalarCount := 0

	for _, et := range types {
		if et == MAPPING_START_EVENT {
			hasMapping = true
		}
		if et == SCALAR_EVENT {
			scalarCount++
		}
	}

	assert.Truef(t, hasMapping, "Expected MAPPING_START_EVENT not found")
	assert.Truef(t, scalarCount >= 4, "Expected at least 4 scalars, got %d", scalarCount)
}

func TestParserFlowSequenceInMapping(t *testing.T) {
	input := "key: [1, 2, 3]"
	types, ok := parseEvents(input)
	assert.Truef(t, ok, "parseEvents() failed")

	hasMappingStart := false
	hasSequenceStart := false

	for _, et := range types {
		if et == MAPPING_START_EVENT {
			hasMappingStart = true
		}
		if et == SEQUENCE_START_EVENT {
			hasSequenceStart = true
		}
	}

	assert.Truef(t, hasMappingStart, "Expected MAPPING_START_EVENT not found")
	assert.Truef(t, hasSequenceStart, "Expected SEQUENCE_START_EVENT not found")
}

func TestParserBlockMappingInSequence(t *testing.T) {
	input := "- key1: value1\n- key2: value2"
	types, ok := parseEvents(input)
	assert.Truef(t, ok, "parseEvents() failed")

	hasSequenceStart := false
	mappingCount := 0

	for _, et := range types {
		if et == SEQUENCE_START_EVENT {
			hasSequenceStart = true
		}
		if et == MAPPING_START_EVENT {
			mappingCount++
		}
	}

	assert.Truef(t, hasSequenceStart, "Expected SEQUENCE_START_EVENT not found")
	assert.Truef(t, mappingCount >= 2, "Expected at least 2 mappings, got %d", mappingCount)
}

func TestParserErrorState(t *testing.T) {
	input := "key: : invalid"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	for {
		var event Event
		if !parser.Parse(&event) {
			if parser.ErrorType != NO_ERROR {
				return
			}
			break
		}
		if event.Type == STREAM_END_EVENT {
			break
		}
	}

	t.Error("Expected parser error for invalid YAML")
}

func TestParserVersionDirective(t *testing.T) {
	input := "%YAML 1.1\n---\nkey: value"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	foundVersionDirective := false
	for {
		var event Event
		if !parser.Parse(&event) {
			break
		}

		if event.Type == DOCUMENT_START_EVENT && event.version_directive != nil {
			foundVersionDirective = true
			assert.Equalf(t, 1, int(event.version_directive.major), "event.version_directive.major = %d, want 1", event.version_directive.major)
			assert.Equalf(t, 1, int(event.version_directive.minor), "event.version_directive.minor = %d, want 1", event.version_directive.minor)
		}

		if event.Type == STREAM_END_EVENT {
			break
		}
	}

	assert.Truef(t, foundVersionDirective, "Expected version directive not found")
}

func TestParserTagDirective(t *testing.T) {
	input := "%TAG !yaml! tag:yaml.org,2002:\n---\nkey: value"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	foundTagDirective := false
	for {
		var event Event
		if !parser.Parse(&event) {
			break
		}

		if event.Type == DOCUMENT_START_EVENT && len(event.tag_directives) > 0 {
			foundTagDirective = true
		}

		if event.Type == STREAM_END_EVENT {
			break
		}
	}

	assert.Truef(t, foundTagDirective, "Expected tag directive not found")
}
