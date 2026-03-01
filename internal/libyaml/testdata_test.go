// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for YAML test data loading.
// Verifies test data loading utilities and scalar coercion functions.

package libyaml

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
	"go.yaml.in/yaml/v4/internal/testutil/datatest"
)

// NodeSpec describes an input node for pipeline stage tests.
// Used in nested test format for representer, desolver, and serializer tests.
type NodeSpec struct {
	Tag     string `yaml:"tag"`     // YAML tag (e.g., "!!int", "!!str")
	Value   string `yaml:"value"`   // Scalar value
	Kind    string `yaml:"kind"`    // Node kind: Scalar, Mapping, Sequence, Document
	Style   string `yaml:"style"`   // Style: Tagged, SingleQuoted, DoubleQuoted, Flow
	Content any    `yaml:"content"` // Nested content for collections
}

// WantSpec describes expected test results for pipeline stage tests.
// Used in nested test format for representer, desolver, and serializer tests.
type WantSpec struct {
	Tag          string `yaml:"tag"`           // Expected tag
	Value        string `yaml:"value"`         // Expected scalar value
	Kind         string `yaml:"kind"`          // Expected node kind
	Quoted       bool   `yaml:"quoted"`        // Whether scalar should be quoted
	ContentCount int    `yaml:"content_count"` // Expected number of content children
	Yaml         string `yaml:"yaml"`          // Expected YAML output
}

// TestCase represents a single test case loaded from YAML
type TestCase struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`

	// Common fields
	Yaml       string      `yaml:"yaml"`
	InputHex   string      `yaml:"input_hex"`
	InputBytes string      `yaml:"input_bytes"`
	From       any         `yaml:"from"` // Input data for tests
	Want       any         `yaml:"want"` // Expected output
	Also       string      `yaml:"also"` // Test modifiers (e.g., "unwrap")
	Like       string      `yaml:"like"` // Regex pattern to match error message
	WantSpecs  []EventSpec // Populated from Want for detailed tests

	// scan_tokens_detailed
	WantTokens []TokenSpec // Populated from Want for detailed tests

	// emit tests
	Events       []EventSpec   `yaml:"data"`
	WantContains []string      // Populated from Want for emit tests
	Config       EmitterConfig `yaml:"conf"`

	// style_accessor tests (must come before Checks due to shared yaml:"test" tag)
	StyleTest []any `yaml:"test"` // [Method, STYLE] where Method is string and STYLE is int or string constant

	// api_new tests
	Constructor string       `yaml:"with"`
	Checks      []FieldCheck `yaml:"test"`

	// encoding_detect tests
	WantEncoding string `yaml:"want_encoding"`

	// char_classify tests
	Cases []CharTestCase `yaml:"cases"`

	// yamlprivate tests (char-predicate, char-convert)
	Function string    `yaml:"func"`  // Function to call
	Input    ByteInput `yaml:"data"`  // Can be string or []int (hex bytes)
	Index    int       `yaml:"index"` // Defaults to 0

	// reader tests
	Args Args `yaml:"args"` // Arguments to pass to method (can be scalar or array)

	// writer tests
	Output string `yaml:"output_type"` // Output handler type (string, writer, error-writer)
	Data2  string `yaml:"data2"`       // Second data chunk for multi-flush tests

	// read_handler tests
	Handler  string `yaml:"handler"`
	ReadSize int    `yaml:"read_size"`
	WantData string `yaml:"want_data"`
	WantEOF  bool   `yaml:"want_eof"`

	// enum_string tests
	Enum []any `yaml:"enum"` // [Type, Value] where Type is string and Value is int or string

	// api_method, api_panic, api_delete tests, and reader tests
	Bytes  bool  `yaml:"byte"`
	Method []any `yaml:"call"`
	Setup  any   `yaml:"init"` // Can be []interface{} (api tests) or map[string]interface{} (reader tests)

	// Pipeline stage tests (representer, desolver, serializer) - nested format
	// For representer: use From for input value
	// For desolver: use Node for input node to desolve
	// For serializer: use Node for input node to serialize, Yaml for expected output
	// Note: Want field (type any) is used - cast to map in test handlers for representer/desolver
	Node   NodeSpec `yaml:"node"`   // Input/expected node specification
	Indent int      `yaml:"indent"` // Indentation setting for serializer tests

	// Error test specific fields
	As           string `yaml:"as"`            // Type name for errors.As tests
	Is           string `yaml:"is"`            // Error message for errors.Is tests
	WantAs       bool   `yaml:"want_as"`       // Expected result for errors.As
	WantIs       bool   `yaml:"want_is"`       // Expected result for errors.Is
	WantLine     int    `yaml:"want_line"`     // Expected line for ConstructError
	WantMessage  string `yaml:"want_message"`  // Expected message for ConstructError
	WantMessages []any  `yaml:"want_messages"` // Expected messages for TypeError
}

// constantRegistry holds libyaml-specific constants
var constantRegistry = datatest.NewConstantRegistry()

// constantMap maps constant names to their integer values (for backward compatibility)
var constantMap = map[string]int{
	// ScalarStyle (bit-shifted starting at iota=1)
	"ANY_SCALAR_STYLE":           0,
	"PLAIN_SCALAR_STYLE":         2,
	"SINGLE_QUOTED_SCALAR_STYLE": 4,
	"DOUBLE_QUOTED_SCALAR_STYLE": 8,
	"LITERAL_SCALAR_STYLE":       16,
	"FOLDED_SCALAR_STYLE":        32,

	// TokenType
	"NO_TOKEN":                   0,
	"STREAM_START_TOKEN":         1,
	"STREAM_END_TOKEN":           2,
	"VERSION_DIRECTIVE_TOKEN":    3,
	"TAG_DIRECTIVE_TOKEN":        4,
	"DOCUMENT_START_TOKEN":       5,
	"DOCUMENT_END_TOKEN":         6,
	"BLOCK_SEQUENCE_START_TOKEN": 7,
	"BLOCK_MAPPING_START_TOKEN":  8,
	"BLOCK_END_TOKEN":            9,
	"FLOW_SEQUENCE_START_TOKEN":  10,
	"FLOW_SEQUENCE_END_TOKEN":    11,
	"FLOW_MAPPING_START_TOKEN":   12,
	"FLOW_MAPPING_END_TOKEN":     13,
	"BLOCK_ENTRY_TOKEN":          14,
	"FLOW_ENTRY_TOKEN":           15,
	"KEY_TOKEN":                  16,
	"VALUE_TOKEN":                17,
	"ALIAS_TOKEN":                18,
	"ANCHOR_TOKEN":               19,
	"TAG_TOKEN":                  20,
	"SCALAR_TOKEN":               21,

	// EventType
	"NO_EVENT":             0,
	"STREAM_START_EVENT":   1,
	"STREAM_END_EVENT":     2,
	"DOCUMENT_START_EVENT": 3,
	"DOCUMENT_END_EVENT":   4,
	"ALIAS_EVENT":          5,
	"SCALAR_EVENT":         6,
	"SEQUENCE_START_EVENT": 7,
	"SEQUENCE_END_EVENT":   8,
	"MAPPING_START_EVENT":  9,
	"MAPPING_END_EVENT":    10,
	"TAIL_COMMENT_EVENT":   11,

	// ParserState
	"PARSE_STREAM_START_STATE":                      0,
	"PARSE_IMPLICIT_DOCUMENT_START_STATE":           1,
	"PARSE_DOCUMENT_START_STATE":                    2,
	"PARSE_DOCUMENT_CONTENT_STATE":                  3,
	"PARSE_DOCUMENT_END_STATE":                      4,
	"PARSE_BLOCK_NODE_STATE":                        5,
	"PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE":        6,
	"PARSE_BLOCK_SEQUENCE_ENTRY_STATE":              7,
	"PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE":         8,
	"PARSE_BLOCK_MAPPING_FIRST_KEY_STATE":           9,
	"PARSE_BLOCK_MAPPING_KEY_STATE":                 10,
	"PARSE_BLOCK_MAPPING_VALUE_STATE":               11,
	"PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE":         12,
	"PARSE_FLOW_SEQUENCE_ENTRY_STATE":               13,
	"PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE":   14,
	"PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE": 15,
	"PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE":   16,
	"PARSE_FLOW_MAPPING_FIRST_KEY_STATE":            17,
	"PARSE_FLOW_MAPPING_KEY_STATE":                  18,
	"PARSE_FLOW_MAPPING_VALUE_STATE":                19,
	"PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE":          20,
	"PARSE_END_STATE":                               21,

	// SequenceStyle / MappingStyle
	"ANY_SEQUENCE_STYLE":   0,
	"BLOCK_SEQUENCE_STYLE": 1,
	"FLOW_SEQUENCE_STYLE":  2,
	"ANY_MAPPING_STYLE":    0,
	"BLOCK_MAPPING_STYLE":  1,
	"FLOW_MAPPING_STYLE":   2,

	// Encoding
	"ANY_ENCODING":     0,
	"UTF8_ENCODING":    1,
	"UTF16LE_ENCODING": 2,
	"UTF16BE_ENCODING": 3,

	// LineBreak
	"ANY_BREAK":  0,
	"CR_BREAK":   1,
	"LN_BREAK":   2,
	"CRLN_BREAK": 3,

	// ErrorType
	"NO_ERROR":       0,
	"MEMORY_ERROR":   1,
	"READER_ERROR":   2,
	"SCANNER_ERROR":  3,
	"PARSER_ERROR":   4,
	"COMPOSER_ERROR": 5,
	"WRITER_ERROR":   6,
	"EMITTER_ERROR":  7,
}

func init() {
	// Populate constantRegistry with all constants from constantMap
	for name, value := range constantMap {
		constantRegistry.Register(name, value)
	}
}

// resolveConstant converts a constant name string to its integer value
func resolveConstant(t *testing.T, name string) int {
	t.Helper()
	val, ok := constantMap[name]
	if !ok {
		t.Fatalf("unknown constant name: %s", name)
	}
	return val
}

// IntOrStr wraps the shared datatest.IntOrStr with libyaml's constant registry
type IntOrStr struct {
	datatest.IntOrStr
}

func (ios *IntOrStr) FromValue(v any) error {
	ios.Registry = constantRegistry
	return ios.IntOrStr.FromValue(v)
}

// ByteInput is an alias to the shared datatest.ByteInput
type ByteInput = datatest.ByteInput

// Args is an alias to the shared datatest.Args
type Args = datatest.Args

// EventSpec specifies an event in YAML format
type EventSpec struct {
	Type             string                `yaml:"type"`
	Encoding         string                `yaml:"encoding"`
	Implicit         bool                  `yaml:"implicit"`
	QuotedImplicit   bool                  `yaml:"quoted_implicit"`
	Anchor           string                `yaml:"anchor"`
	Tag              string                `yaml:"tag"`
	Value            string                `yaml:"value"`
	Style            string                `yaml:"style"`
	VersionDirective *VersionDirectiveSpec `yaml:"version-directive"`
	TagDirectives    []TagDirectiveSpec    `yaml:"tag-directives"`
}

// VersionDirectiveSpec specifies a version directive
type VersionDirectiveSpec struct {
	Major int `yaml:"major"`
	Minor int `yaml:"minor"`
}

// TagDirectiveSpec specifies a tag directive
type TagDirectiveSpec struct {
	Handle string `yaml:"handle"`
	Prefix string `yaml:"prefix"`
}

// TokenSpec specifies a token in YAML format
type TokenSpec struct {
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
	Style string `yaml:"style"`
}

// EmitterConfig specifies emitter configuration
type EmitterConfig struct {
	Canonical bool   `yaml:"canonical"`
	Indent    int    `yaml:"indent"`
	Width     int    `yaml:"width"`
	Unicode   bool   `yaml:"unicode"`
	LineBreak string `yaml:"line_break"`
}

// SetupSpec specifies test setup
type SetupSpec struct {
	Constructor string     `yaml:"constructor"`
	Calls       []CallSpec `yaml:"calls"`
}

// CallSpec specifies a method call
type CallSpec struct {
	Method string    `yaml:"method"`
	Args   []ArgSpec `yaml:"args"`
}

// ArgSpec specifies a method argument
type ArgSpec struct {
	Bytes  string `yaml:"bytes"`
	String string `yaml:"string"`
	Int    int    `yaml:"int"`
	Bool   bool   `yaml:"bool"`
	Hex    string `yaml:"hex"`
	Reader bool   `yaml:"reader"` // Creates a strings.Reader from String field
	Writer bool   `yaml:"writer"` // Creates a bytes.Buffer
}

// FieldCheck specifies a field check
type FieldCheck struct {
	Nil   []any `yaml:"nil"`
	Cap   []any `yaml:"cap"`
	Len   []any `yaml:"len"`
	LenGt []any `yaml:"len-gt"` // Length greater than
	Eq    []any `yaml:"eq"`
	Gte   []any `yaml:"gte"` // Greater than or equal
}

// CharTestCase represents a character classification test case
type CharTestCase struct {
	InputHex string `yaml:"input_hex"`
	Pos      int    `yaml:"pos"`
	Want     any    `yaml:"want"` // Can be bool or int
}

// unmarshalTestCases converts raw YAML data to TestCase structs using yamltest
func unmarshalTestCases(data any) ([]TestCase, error) {
	casesSlice, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("expected []interface{}, got %T", data)
	}

	var testCases []TestCase
	for i, item := range casesSlice {
		caseMap, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("test case %d: expected map[string]interface{}, got %T", i, item)
		}

		// Normalize type-as-key format for top-level test cases
		caseMap = datatest.NormalizeTypeAsKey(caseMap)

		var tc TestCase
		if err := datatest.UnmarshalStruct(&tc, caseMap); err != nil {
			return nil, fmt.Errorf("test case %d: %w", i, err)
		}
		testCases = append(testCases, tc)
	}

	return testCases, nil
}

func LoadTestCases(filename string) ([]TestCase, error) {
	// Get the path relative to this file
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile)
	path := filepath.Join(dir, "testdata", filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filename, err)
	}

	// Load YAML using LoadAny from loader.go
	rawData, err := LoadAny(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	// Convert to TestCase structs
	cases, err := unmarshalTestCases(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal test cases from %s: %w", filename, err)
	}

	// Post-process: convert Want to WantSpecs/WantTokens for detailed tests
	for i := range cases {
		// Determine which field to populate based on test type
		switch cases[i].Type {
		case "parse-events-detailed":
			if cases[i].Want != nil {
				// Want should be []interface{} of maps, convert to []EventSpec
				wantSlice, ok := cases[i].Want.([]any)
				if !ok {
					return nil, fmt.Errorf("test %s: want should be a sequence, got %T", cases[i].Name, cases[i].Want)
				}
				cases[i].WantSpecs = make([]EventSpec, len(wantSlice))
				for j, item := range wantSlice {
					var itemMap map[string]any
					// Check if item is a scalar string (simplified format)
					if strVal, ok := item.(string); ok {
						// Convert scalar to map with type field
						itemMap = map[string]any{"type": strVal}
					} else {
						itemMap, ok = item.(map[string]any)
						if !ok {
							return nil, fmt.Errorf("test %s: want[%d] should be a map or string, got %T", cases[i].Name, j, item)
						}
						// Normalize type-as-key format
						itemMap = datatest.NormalizeTypeAsKey(itemMap)
					}
					if err := datatest.UnmarshalStruct(&cases[i].WantSpecs[j], itemMap); err != nil {
						return nil, fmt.Errorf("test %s: want[%d]: %w", cases[i].Name, j, err)
					}
				}
			}
		case "scan-tokens-detailed":
			if cases[i].Want != nil {
				// Want should be []interface{} of maps, convert to []TokenSpec
				wantSlice, ok := cases[i].Want.([]any)
				if !ok {
					return nil, fmt.Errorf("test %s: want should be a sequence, got %T", cases[i].Name, cases[i].Want)
				}
				cases[i].WantTokens = make([]TokenSpec, len(wantSlice))
				for j, item := range wantSlice {
					var itemMap map[string]any
					// Check if item is a scalar string (simplified format)
					if strVal, ok := item.(string); ok {
						// Convert scalar to map with type field
						itemMap = map[string]any{"type": strVal}
					} else {
						itemMap, ok = item.(map[string]any)
						if !ok {
							return nil, fmt.Errorf("test %s: want[%d] should be a map or string, got %T", cases[i].Name, j, item)
						}
						// Normalize type-as-key format
						itemMap = datatest.NormalizeTypeAsKey(itemMap)
					}
					if err := datatest.UnmarshalStruct(&cases[i].WantTokens[j], itemMap); err != nil {
						return nil, fmt.Errorf("test %s: want[%d]: %w", cases[i].Name, j, err)
					}
				}
			}
		}

		// Post-process: convert Want to WantContains for emit tests
		switch cases[i].Type {
		case "emit", "emit-config", "roundtrip", "emit-writer":
			if cases[i].Want != nil {
				switch v := cases[i].Want.(type) {
				case string:
					// Scalar want value
					cases[i].WantContains = []string{v}
				case []any:
					// Sequence want values
					for _, item := range v {
						if str, ok := item.(string); ok {
							cases[i].WantContains = append(cases[i].WantContains, str)
						}
					}
				}
			}
		}
	}

	return cases, nil
}

// ParseEventType converts a string to EventType
func ParseEventType(t *testing.T, s string) EventType {
	t.Helper()
	switch s {
	case "NO_EVENT":
		return NO_EVENT
	case "STREAM_START_EVENT":
		return STREAM_START_EVENT
	case "STREAM_END_EVENT":
		return STREAM_END_EVENT
	case "DOCUMENT_START_EVENT":
		return DOCUMENT_START_EVENT
	case "DOCUMENT_END_EVENT":
		return DOCUMENT_END_EVENT
	case "ALIAS_EVENT":
		return ALIAS_EVENT
	case "SCALAR_EVENT":
		return SCALAR_EVENT
	case "SEQUENCE_START_EVENT":
		return SEQUENCE_START_EVENT
	case "SEQUENCE_END_EVENT":
		return SEQUENCE_END_EVENT
	case "MAPPING_START_EVENT":
		return MAPPING_START_EVENT
	case "MAPPING_END_EVENT":
		return MAPPING_END_EVENT
	default:
		t.Fatalf("unknown event type: %s", s)
		return NO_EVENT
	}
}

// ParseTokenType converts a string to TokenType
func ParseTokenType(t *testing.T, s string) TokenType {
	t.Helper()
	switch s {
	case "NO_TOKEN":
		return NO_TOKEN
	case "STREAM_START_TOKEN":
		return STREAM_START_TOKEN
	case "STREAM_END_TOKEN":
		return STREAM_END_TOKEN
	case "VERSION_DIRECTIVE_TOKEN":
		return VERSION_DIRECTIVE_TOKEN
	case "TAG_DIRECTIVE_TOKEN":
		return TAG_DIRECTIVE_TOKEN
	case "DOCUMENT_START_TOKEN":
		return DOCUMENT_START_TOKEN
	case "DOCUMENT_END_TOKEN":
		return DOCUMENT_END_TOKEN
	case "BLOCK_SEQUENCE_START_TOKEN":
		return BLOCK_SEQUENCE_START_TOKEN
	case "BLOCK_MAPPING_START_TOKEN":
		return BLOCK_MAPPING_START_TOKEN
	case "BLOCK_END_TOKEN":
		return BLOCK_END_TOKEN
	case "FLOW_SEQUENCE_START_TOKEN":
		return FLOW_SEQUENCE_START_TOKEN
	case "FLOW_SEQUENCE_END_TOKEN":
		return FLOW_SEQUENCE_END_TOKEN
	case "FLOW_MAPPING_START_TOKEN":
		return FLOW_MAPPING_START_TOKEN
	case "FLOW_MAPPING_END_TOKEN":
		return FLOW_MAPPING_END_TOKEN
	case "BLOCK_ENTRY_TOKEN":
		return BLOCK_ENTRY_TOKEN
	case "FLOW_ENTRY_TOKEN":
		return FLOW_ENTRY_TOKEN
	case "KEY_TOKEN":
		return KEY_TOKEN
	case "VALUE_TOKEN":
		return VALUE_TOKEN
	case "ALIAS_TOKEN":
		return ALIAS_TOKEN
	case "ANCHOR_TOKEN":
		return ANCHOR_TOKEN
	case "TAG_TOKEN":
		return TAG_TOKEN
	case "SCALAR_TOKEN":
		return SCALAR_TOKEN
	default:
		t.Fatalf("unknown token type: %s", s)
		return NO_TOKEN
	}
}

// ParseEncoding converts a string to Encoding
func ParseEncoding(t *testing.T, s string) Encoding {
	t.Helper()
	switch s {
	case "ANY_ENCODING":
		return ANY_ENCODING
	case "UTF8_ENCODING":
		return UTF8_ENCODING
	case "UTF16LE_ENCODING":
		return UTF16LE_ENCODING
	case "UTF16BE_ENCODING":
		return UTF16BE_ENCODING
	default:
		t.Fatalf("unknown encoding: %s", s)
		return ANY_ENCODING
	}
}

// ParseScalarStyle converts a string to ScalarStyle
func ParseScalarStyle(t *testing.T, s string) ScalarStyle {
	t.Helper()
	switch s {
	case "ANY_SCALAR_STYLE":
		return ANY_SCALAR_STYLE
	case "PLAIN_SCALAR_STYLE":
		return PLAIN_SCALAR_STYLE
	case "SINGLE_QUOTED_SCALAR_STYLE":
		return SINGLE_QUOTED_SCALAR_STYLE
	case "DOUBLE_QUOTED_SCALAR_STYLE":
		return DOUBLE_QUOTED_SCALAR_STYLE
	case "LITERAL_SCALAR_STYLE":
		return LITERAL_SCALAR_STYLE
	case "FOLDED_SCALAR_STYLE":
		return FOLDED_SCALAR_STYLE
	default:
		t.Fatalf("unknown scalar style: %s", s)
		return ANY_SCALAR_STYLE
	}
}

// ParseSequenceStyle converts a string to SequenceStyle
func ParseSequenceStyle(t *testing.T, s string) SequenceStyle {
	t.Helper()
	switch s {
	case "ANY_SEQUENCE_STYLE":
		return ANY_SEQUENCE_STYLE
	case "BLOCK_SEQUENCE_STYLE":
		return BLOCK_SEQUENCE_STYLE
	case "FLOW_SEQUENCE_STYLE":
		return FLOW_SEQUENCE_STYLE
	default:
		t.Fatalf("unknown sequence style: %s", s)
		return ANY_SEQUENCE_STYLE
	}
}

// ParseMappingStyle converts a string to MappingStyle
func ParseMappingStyle(t *testing.T, s string) MappingStyle {
	t.Helper()
	switch s {
	case "ANY_MAPPING_STYLE":
		return ANY_MAPPING_STYLE
	case "BLOCK_MAPPING_STYLE":
		return BLOCK_MAPPING_STYLE
	case "FLOW_MAPPING_STYLE":
		return FLOW_MAPPING_STYLE
	default:
		t.Fatalf("unknown mapping style: %s", s)
		return ANY_MAPPING_STYLE
	}
}

// CreateEventFromSpec creates an Event from an EventSpec
func CreateEventFromSpec(t *testing.T, spec EventSpec) Event {
	t.Helper()
	eventType := ParseEventType(t, spec.Type)

	switch eventType {
	case STREAM_START_EVENT:
		encoding := UTF8_ENCODING
		if spec.Encoding != "" {
			encoding = ParseEncoding(t, spec.Encoding)
		}
		return NewStreamStartEvent(encoding)

	case STREAM_END_EVENT:
		return NewStreamEndEvent()

	case DOCUMENT_START_EVENT:
		var vd *VersionDirective
		if spec.VersionDirective != nil {
			vd = &VersionDirective{
				major: int8(spec.VersionDirective.Major),
				minor: int8(spec.VersionDirective.Minor),
			}
		}
		var td []TagDirective
		for _, tagSpec := range spec.TagDirectives {
			td = append(td, TagDirective{
				handle: []byte(tagSpec.Handle),
				prefix: []byte(tagSpec.Prefix),
			})
		}
		return NewDocumentStartEvent(vd, td, spec.Implicit)

	case DOCUMENT_END_EVENT:
		return NewDocumentEndEvent(spec.Implicit)

	case ALIAS_EVENT:
		return NewAliasEvent([]byte(spec.Anchor))

	case SCALAR_EVENT:
		style := PLAIN_SCALAR_STYLE
		if spec.Style != "" {
			style = ParseScalarStyle(t, spec.Style)
		}
		return NewScalarEvent(
			[]byte(spec.Anchor),
			[]byte(spec.Tag),
			[]byte(spec.Value),
			spec.Implicit,
			spec.QuotedImplicit,
			style,
		)

	case SEQUENCE_START_EVENT:
		style := BLOCK_SEQUENCE_STYLE
		if spec.Style != "" {
			style = ParseSequenceStyle(t, spec.Style)
		}
		return NewSequenceStartEvent(
			[]byte(spec.Anchor),
			[]byte(spec.Tag),
			spec.Implicit,
			style,
		)

	case SEQUENCE_END_EVENT:
		return NewSequenceEndEvent()

	case MAPPING_START_EVENT:
		style := BLOCK_MAPPING_STYLE
		if spec.Style != "" {
			style = ParseMappingStyle(t, spec.Style)
		}
		return NewMappingStartEvent(
			[]byte(spec.Anchor),
			[]byte(spec.Tag),
			spec.Implicit,
			style,
		)

	case MAPPING_END_EVENT:
		return NewMappingEndEvent()

	default:
		t.Fatalf("unsupported event type: %v", eventType)
		return Event{}
	}
}

// HexToBytes converts a hex string to bytes
// HexToBytes is now provided by the shared datatest package
var HexToBytes = datatest.HexToBytes

// GetField is now provided by the shared datatest package
var GetField = datatest.GetField

// CreateArgValue creates a value from an ArgSpec
func CreateArgValue(t *testing.T, spec ArgSpec) any {
	t.Helper()
	if spec.Bytes != "" {
		return []byte(spec.Bytes)
	}
	if spec.String != "" {
		if spec.Reader {
			return bytes.NewReader([]byte(spec.String))
		}
		return spec.String
	}
	if spec.Hex != "" {
		return HexToBytes(t, spec.Hex)
	}
	if spec.Writer {
		return new(bytes.Buffer)
	}
	// Default to the first non-zero value
	if spec.Int != 0 {
		return spec.Int
	}
	if spec.Bool {
		return spec.Bool
	}
	return nil
}

// CallMethod is now provided by the shared datatest package
var CallMethod = datatest.CallMethod

// CreateObject creates an object using a constructor function
func CreateObject(t *testing.T, constructorName string) any {
	t.Helper()
	switch constructorName {
	case "NewParser":
		return NewParser()
	case "NewEmitter":
		return NewEmitter()
	default:
		t.Fatalf("unknown constructor: %s", constructorName)
		return nil
	}
}

// emitEvents is a helper to emit events and return the output
func emitEvents(events []Event) (string, error) {
	emitter := NewEmitter()
	var output []byte
	emitter.SetOutputString(&output)

	for i := range events {
		if err := emitter.Emit(&events[i]); err != nil {
			return "", err
		}
	}

	return string(output), nil
}

// parseEvents is a helper to parse input and return event types
func parseEvents(input string) ([]EventType, bool) {
	parser := NewParser()
	parser.SetInputString([]byte(input))

	var types []EventType
	for {
		var event Event
		if err := parser.Parse(&event); err != nil {
			if errors.Is(err, io.EOF) {
				return types, true
			}
			return nil, false
		}
		types = append(types, event.Type)
		if event.Type == STREAM_END_EVENT {
			break
		}
	}
	return types, true
}

// parseEventsDetailed is a helper to parse input and return full events
func parseEventsDetailed(input string) ([]Event, bool) {
	parser := NewParser()
	parser.SetInputString([]byte(input))

	var events []Event
	for {
		var event Event
		if err := parser.Parse(&event); err != nil {
			if errors.Is(err, io.EOF) {
				return events, true
			}
			return nil, false
		}
		events = append(events, event)
		if event.Type == STREAM_END_EVENT {
			break
		}
	}
	return events, true
}

// scanTokens is a helper to scan input and return token types
func scanTokens(input string) ([]TokenType, bool) {
	parser := NewParser()
	parser.SetInputString([]byte(input))

	var types []TokenType
	for {
		var token Token
		if err := parser.Scan(&token); err != nil {
			if errors.Is(err, io.EOF) {
				return types, true
			}
			return nil, false
		}
		types = append(types, token.Type)
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}
	return types, true
}

// scanTokensDetailed is a helper to scan input and return full tokens
func scanTokensDetailed(input string) ([]Token, bool) {
	parser := NewParser()
	parser.SetInputString([]byte(input))

	var tokens []Token
	for {
		var token Token
		if err := parser.Scan(&token); err != nil {
			if errors.Is(err, io.EOF) {
				return tokens, true
			}
			return nil, false
		}
		tokens = append(tokens, token)
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}
	return tokens, true
}

// ConfigureEmitter configures an emitter from an EmitterConfig
func ConfigureEmitter(emitter *Emitter, config EmitterConfig) {
	if config.Canonical {
		emitter.SetCanonical(true)
	}
	if config.Indent > 0 {
		emitter.SetIndent(config.Indent)
	}
	if config.Width != 0 {
		emitter.SetWidth(config.Width)
	}
	if config.Unicode {
		emitter.SetUnicode(true)
	}
	if config.LineBreak != "" {
		// Parse line break style if needed
		switch config.LineBreak {
		case "LN":
			emitter.SetLineBreak(LN_BREAK)
		case "CR":
			emitter.SetLineBreak(CR_BREAK)
		case "CRLF":
			emitter.SetLineBreak(CRLN_BREAK)
		}
	}
}

// RunEmitTest runs an emit test case
func RunEmitTest(t *testing.T, tc TestCase) {
	t.Helper()

	var events []Event
	for _, eventSpec := range tc.Events {
		events = append(events, CreateEventFromSpec(t, eventSpec))
	}

	var output []byte
	var emitter *Emitter

	if tc.Type == "emit-config" {
		e := NewEmitter()
		emitter = &e
		emitter.SetOutputString(&output)
		ConfigureEmitter(emitter, tc.Config)

		for i := range events {
			err := emitter.Emit(&events[i])
			assert.NoErrorf(t, err, "Emit() error: %v", err)
		}
	} else {
		result, err := emitEvents(events)
		assert.NoErrorf(t, err, "emitEvents() error: %v", err)
		output = []byte(result)
	}

	for _, expected := range tc.WantContains {
		assert.Truef(t, bytes.Contains(output, []byte(expected)),
			"output should contain %q, got %q", expected, string(output))
	}
}

// RunRoundTripTest runs a roundtrip test case
func RunRoundTripTest(t *testing.T, tc TestCase) {
	t.Helper()

	parser := NewParser()
	parser.SetInputString([]byte(tc.Yaml))

	var events []Event
	for {
		var event Event
		if err := parser.Parse(&event); err != nil {
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
	for _, expected := range tc.WantContains {
		assert.Truef(t, bytes.Contains(output, []byte(expected)),
			"output should contain %q, got %q", expected, result)
	}
}

// GetWriter extracts an [io.Writer] from an interface value
func GetWriter(t *testing.T, v any) io.Writer {
	t.Helper()
	if w, ok := v.(io.Writer); ok {
		return w
	}
	t.Fatalf("value is not an io.Writer: %T", v)
	return nil
}

// GetReader extracts an [io.Reader] from an interface value
func GetReader(t *testing.T, v any) io.Reader {
	t.Helper()
	if r, ok := v.(io.Reader); ok {
		return r
	}
	t.Fatalf("value is not an io.Reader: %T", v)
	return nil
}

// TestHandler is a function that runs a specific test type
type TestHandler func(*testing.T, TestCase)

// RunTestCases loads test cases from a YAML file and runs them using the provided handlers
func RunTestCases(t *testing.T, filename string, handlers map[string]TestHandler) {
	t.Helper()
	cases, err := LoadTestCases(filename)
	assert.NoErrorf(t, err, "Failed to load test cases: %v", err)

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			handler, ok := handlers[tc.Type]
			if !ok {
				t.Fatalf("unknown test type: %s", tc.Type)
			}
			handler(t, tc)
		})
	}
}

// WantBool extracts a bool from tc.Want, returning defaultVal if Want is nil
// WantBool is now provided by the shared datatest package
var WantBool = datatest.WantBool

// API test handlers
// These test runner functions are used by both parser and emitter tests

func runAPINewTest(t *testing.T, tc TestCase) {
	t.Helper()

	obj := createObject(t, tc.Constructor)
	runFieldChecks(t, obj, tc.Checks)
}

func runAPIMethodTest(t *testing.T, tc TestCase) {
	t.Helper()

	obj := createObject(t, tc.Constructor)

	// Run setup if any
	if tc.Setup != nil {
		if setupList, ok := tc.Setup.([]any); ok && len(setupList) > 0 {
			callMethodFromList(t, obj, setupList, tc.Bytes)
		}
	}

	// Call the main method
	callMethodFromList(t, obj, tc.Method, tc.Bytes)

	// Run checks
	runFieldChecks(t, obj, tc.Checks)
}

func runAPIPanicTest(t *testing.T, tc TestCase) {
	t.Helper()

	obj := createObject(t, tc.Constructor)

	// Run setup if any
	if tc.Setup != nil {
		if setupList, ok := tc.Setup.([]any); ok && len(setupList) > 0 {
			callMethodFromList(t, obj, setupList, tc.Bytes)
		}
	}

	// The main method call should panic
	// Want can be either a string or a single-element sequence
	var wantMsg string
	switch v := tc.Want.(type) {
	case string:
		wantMsg = v
	case []any:
		if len(v) > 0 {
			msg, ok := v[0].(string)
			assert.Truef(t, ok, "Want[0] should be string, got %T", v[0])
			wantMsg = msg
		} else {
			t.Fatalf("Want slice is empty, expected at least one element")
		}
	default:
		t.Fatalf("want must be a string or sequence, got %T", tc.Want)
	}
	assert.PanicMatchesf(t, wantMsg, func() {
		callMethodFromList(t, obj, tc.Method, tc.Bytes)
	}, "Expected panic: %s", wantMsg)
}

func runAPIDeleteTest(t *testing.T, tc TestCase) {
	t.Helper()

	obj := createObject(t, tc.Constructor)

	// Run setup if any
	if tc.Setup != nil {
		if setupList, ok := tc.Setup.([]any); ok && len(setupList) > 0 {
			callMethodFromList(t, obj, setupList, tc.Bytes)
		}
	}

	// Call Delete method
	callMethodFromList(t, obj, []any{"Delete"}, false)

	// Run checks after delete
	runFieldChecks(t, obj, tc.Checks)
}

func runAPINewEventTest(t *testing.T, tc TestCase) {
	t.Helper()

	event := createEventFromList(t, tc.Method, tc.Bytes)
	runFieldChecks(t, &event, tc.Checks)
}

// API test helper functions
// These functions support both parser and emitter API tests

// createObject creates a Parser or Emitter based on constructor name
func createObject(t *testing.T, constructor string) any {
	t.Helper()
	switch constructor {
	case "NewParser":
		p := NewParser()
		return &p
	case "NewEmitter":
		e := NewEmitter()
		return &e
	default:
		t.Fatalf("unknown constructor: %s", constructor)
	}
	return nil
}

// createEventFromList creates an Event from a method list [constructor, args...]
func createEventFromList(t *testing.T, methodList []any, useBytes bool) Event {
	t.Helper()
	if len(methodList) == 0 {
		t.Fatalf("empty method list")
	}

	constructor, ok := methodList[0].(string)
	if !ok {
		t.Fatalf("constructor should be string, got %T", methodList[0])
	}
	args := methodList[1:]

	switch constructor {
	case "NewStreamStartEvent":
		if len(args) != 1 {
			t.Fatalf("%s expects 1 argument, got %d", constructor, len(args))
		}
		encoding := parseArg(t, args[0])
		return NewStreamStartEvent(Encoding(encoding))
	case "NewStreamEndEvent":
		if len(args) != 0 {
			t.Fatalf("%s expects 0 arguments, got %d", constructor, len(args))
		}
		return NewStreamEndEvent()
	case "NewDocumentEndEvent":
		if len(args) != 1 {
			t.Fatalf("%s expects 1 argument, got %d", constructor, len(args))
		}
		implicit := parseBoolArg(t, args[0])
		return NewDocumentEndEvent(implicit)
	case "NewAliasEvent":
		if len(args) != 1 {
			t.Fatalf("%s expects 1 argument, got %d", constructor, len(args))
		}
		anchor := parseStringArg(t, args[0], useBytes)
		return NewAliasEvent([]byte(anchor))
	case "NewSequenceEndEvent":
		if len(args) != 0 {
			t.Fatalf("%s expects 0 arguments, got %d", constructor, len(args))
		}
		return NewSequenceEndEvent()
	case "NewMappingEndEvent":
		if len(args) != 0 {
			t.Fatalf("%s expects 0 arguments, got %d", constructor, len(args))
		}
		return NewMappingEndEvent()
	default:
		t.Fatalf("unknown event constructor: %s", constructor)
	}
	return Event{}
}

// callMethodFromList calls a method from a list [methodName, args...]
func callMethodFromList(t *testing.T, obj any, methodList []any, useBytes bool) {
	t.Helper()
	if len(methodList) == 0 {
		t.Fatalf("empty method list")
	}

	methodName, ok := methodList[0].(string)
	if !ok {
		t.Fatalf("method name should be string, got %T", methodList[0])
	}
	args := methodList[1:]

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	m := v.Addr().MethodByName(methodName)
	if !m.IsValid() {
		t.Fatalf("method not found: %s", methodName)
	}

	methodType := m.Type()
	numParams := methodType.NumIn()

	// Validate argument count
	if len(args) != numParams {
		t.Fatalf("method %s expects %d arguments, got %d", methodName, numParams, len(args))
	}

	// Build argument list
	var callArgs []reflect.Value
	for i, arg := range args {
		paramType := methodType.In(i)

		// Handle different parameter types
		if paramType.Kind() == reflect.Bool {
			val := parseBoolArg(t, arg)
			callArgs = append(callArgs, reflect.ValueOf(val))
		} else if paramType.Kind() == reflect.Slice && paramType.Elem().Kind() == reflect.Uint8 {
			// Byte slice parameter
			str := parseStringArg(t, arg, useBytes)
			callArgs = append(callArgs, reflect.ValueOf([]byte(str)))
		} else {
			// Try parsing as constant/int
			val := parseArg(t, arg)
			convertedVal := reflect.ValueOf(val).Convert(paramType)
			callArgs = append(callArgs, convertedVal)
		}
	}

	// Call the method
	m.Call(callArgs)
}

// parseArg parses an argument which could be int, bool, or string constant
func parseArg(t *testing.T, arg any) int {
	t.Helper()
	switch v := arg.(type) {
	case int:
		return v
	case string:
		if looksLikeConstant(v) {
			return parseConstant(t, v)
		}
		// Try parsing as int
		if val, err := strconv.Atoi(v); err == nil {
			return val
		}
		t.Fatalf("cannot parse arg as int: %v", arg)
	default:
		t.Fatalf("unsupported arg type: %T", arg)
	}
	return 0
}

// parseBoolArg parses a boolean argument
func parseBoolArg(t *testing.T, arg any) bool {
	t.Helper()
	switch v := arg.(type) {
	case bool:
		return v
	case string:
		switch v {
		case "true":
			return true
		case "false":
			return false
		default:
			t.Fatalf("cannot parse string as bool (expected 'true' or 'false'): %q", v)
		}
	default:
		t.Fatalf("cannot parse arg as bool: %v (type %T)", arg, arg)
	}
	return false
}

// parseStringArg parses a string argument
func parseStringArg(t *testing.T, arg any, useBytes bool) string {
	t.Helper()
	switch v := arg.(type) {
	case string:
		return v
	default:
		t.Fatalf("cannot parse arg as string: %v (type %T)", arg, arg)
	}
	return ""
}

// looksLikeConstant checks if a string looks like a constant name
func looksLikeConstant(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Check if it's all uppercase letters, digits, and underscores
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// parseConstant parses a constant name to its integer value
func parseConstant(t *testing.T, name string) int {
	t.Helper()
	// Handle boolean values
	if name == "true" {
		return 1
	}
	if name == "false" {
		return 0
	}

	// Try parsing as int first
	if val, err := strconv.Atoi(name); err == nil {
		return val
	}

	// Use IntOrStr to parse other constants
	ios := IntOrStr{}
	err := ios.FromValue(name)
	if err != nil {
		t.Fatalf("failed to parse constant %q: %v", name, err)
	}
	return ios.Value
}

// hasLength checks if a slice has exactly the expected length
// Returns true if length matches, false if empty, and fails fatally otherwise
func hasLength(t *testing.T, slice []any, expected int) bool {
	t.Helper()
	if len(slice) == 0 {
		return false
	}
	if len(slice) != expected {
		t.Fatalf("expected exactly %d args, got %d", expected, len(slice))
	}
	return true
}

// runFieldChecks runs field checks on an object
func runFieldChecks(t *testing.T, obj any, checks []FieldCheck) {
	t.Helper()

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for _, check := range checks {
		// Handle nil checks
		if hasLength(t, check.Nil, 2) {
			fieldName, ok := check.Nil[0].(string)
			if !ok {
				t.Fatalf("Nil[0] should be string, got %T", check.Nil[0])
			}
			wantNil, ok := check.Nil[1].(bool)
			if !ok {
				t.Fatalf("Nil[1] should be bool, got %T", check.Nil[1])
			}
			field := getField(t, v, fieldName)
			if field.IsValid() {
				isNil := field.IsNil()
				if wantNil != isNil {
					if wantNil {
						t.Errorf("%s should be nil", fieldName)
					} else {
						t.Errorf("%s should not be nil", fieldName)
					}
				}
			}
		}

		// Handle cap checks
		if hasLength(t, check.Cap, 2) {
			fieldName, ok := check.Cap[0].(string)
			if !ok {
				t.Fatalf("Cap[0] should be string, got %T", check.Cap[0])
			}
			wantCap, ok := check.Cap[1].(int)
			if !ok {
				t.Fatalf("Cap[1] should be int, got %T", check.Cap[1])
			}
			field := getField(t, v, fieldName)
			if field.IsValid() && wantCap > 0 {
				if field.Cap() != wantCap {
					t.Errorf("%s cap = %d, want %d", fieldName, field.Cap(), wantCap)
				}
			}
		}

		// Handle len checks
		if hasLength(t, check.Len, 2) {
			fieldName, ok := check.Len[0].(string)
			if !ok {
				t.Fatalf("Len[0] should be string, got %T", check.Len[0])
			}
			wantLen, ok := check.Len[1].(int)
			if !ok {
				t.Fatalf("Len[1] should be int, got %T", check.Len[1])
			}
			field := getField(t, v, fieldName)
			if field.IsValid() && wantLen > 0 {
				if field.Len() != wantLen {
					t.Errorf("%s len = %d, want %d", fieldName, field.Len(), wantLen)
				}
			}
		}

		// Handle len-gt checks
		if hasLength(t, check.LenGt, 2) {
			fieldName, ok := check.LenGt[0].(string)
			if !ok {
				t.Fatalf("LenGt[0] should be string, got %T", check.LenGt[0])
			}
			minLen, ok := check.LenGt[1].(int)
			if !ok {
				t.Fatalf("LenGt[1] should be int, got %T", check.LenGt[1])
			}
			field := getField(t, v, fieldName)
			if field.IsValid() && minLen > 0 {
				if field.Len() <= minLen {
					t.Errorf("%s len = %d, want > %d", fieldName, field.Len(), minLen)
				}
			}
		}

		// Handle eq checks
		if hasLength(t, check.Eq, 2) {
			fieldName, ok := check.Eq[0].(string)
			if !ok {
				t.Fatalf("Eq[0] should be string, got %T", check.Eq[0])
			}
			expectedValue := check.Eq[1]
			checkEqual(t, v, fieldName, expectedValue)
		}

		// Handle gte checks
		if hasLength(t, check.Gte, 2) {
			fieldName, ok := check.Gte[0].(string)
			if !ok {
				t.Fatalf("Gte[0] should be string, got %T", check.Gte[0])
			}
			minValue, ok := check.Gte[1].(int)
			if !ok {
				t.Fatalf("Gte[1] should be int, got %T", check.Gte[1])
			}
			field := getField(t, v, fieldName)
			if field.IsValid() {
				got := getIntValue(t, field, fieldName)
				if got < minValue {
					t.Errorf("%s = %d, want >= %d", fieldName, got, minValue)
				}
			}
		}
	}
}

// getField retrieves a field from a struct, handling special field names
func getField(t *testing.T, v reflect.Value, fieldName string) reflect.Value {
	t.Helper()

	// Handle special field names like buffer-0, buffer-1
	if strings.HasPrefix(fieldName, "buffer-") {
		var bufferIndex int
		_, err := fmt.Sscanf(fieldName, "buffer-%d", &bufferIndex)
		if err == nil {
			// Validate that buffer index is non-negative
			if bufferIndex < 0 {
				t.Fatalf("invalid buffer index: %s (index must be non-negative)", fieldName)
			}
			// Return invalid value - buffer index checks are handled separately
			return reflect.Value{}
		}
	}

	// Convert hyphenated YAML key to underscored Go field name
	goFieldName := strings.ReplaceAll(fieldName, "-", "_")
	field := v.FieldByName(goFieldName)
	if !field.IsValid() {
		t.Fatalf("field not found: %s (looking for %s)", fieldName, goFieldName)
	}
	return field
}

// getIntValue extracts an integer value from a field
func getIntValue(t *testing.T, field reflect.Value, fieldName string) int {
	t.Helper()

	// Use reflection Kind() instead of type assertions
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(field.Uint())
	default:
		t.Fatalf("%s: expected numeric field, got %s", fieldName, field.Kind())
	}
	return 0
}

// checkEqual performs an equality check on a field
func checkEqual(t *testing.T, v reflect.Value, fieldName string, expectedValue any) {
	t.Helper()

	// Handle buffer-N special case
	var bufferIndex int
	isBufferIndex := false
	if strings.HasPrefix(fieldName, "buffer-") {
		_, err := fmt.Sscanf(fieldName, "buffer-%d", &bufferIndex)
		if err == nil {
			// Validate that buffer index is non-negative
			if bufferIndex < 0 {
				t.Fatalf("invalid buffer index: %s (index must be non-negative)", fieldName)
			}
			isBufferIndex = true
		}
	}

	var field reflect.Value
	if isBufferIndex {
		field = v.FieldByName("buffer")
		if !field.IsValid() {
			t.Fatalf("buffer field not found for %s", fieldName)
		}
		// Check specific byte in buffer
		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
			if bufferIndex >= field.Len() {
				t.Errorf("%s: index %d out of range (buffer len=%d)", fieldName, bufferIndex, field.Len())
				return
			}
			got := int(field.Index(bufferIndex).Uint())
			expected := expectedValue
			if str, ok := expectedValue.(string); ok && looksLikeConstant(str) {
				expected = parseConstant(t, str)
			} else if intVal, ok := expectedValue.(int); ok {
				expected = intVal
			}
			if got != expected {
				t.Errorf("%s = %v, want %v", fieldName, got, expected)
			}
			return
		} else {
			t.Errorf("%s: buffer field is not a byte slice", fieldName)
			return
		}
	}

	field = getField(t, v, fieldName)
	if !field.IsValid() {
		return
	}

	// Parse constant if it's a string that looks like a constant name
	var expectedInt int
	var hasExpectedInt bool
	expected := expectedValue
	if str, ok := expectedValue.(string); ok && looksLikeConstant(str) {
		expectedInt = parseConstant(t, str)
		hasExpectedInt = true
	} else if intVal, ok := expectedValue.(int); ok {
		expectedInt = intVal
		hasExpectedInt = true
	}

	// Get value based on type (handle unexported fields)
	var got any

	if field.CanInterface() {
		// For exported fields, convert expected to field's type
		if hasExpectedInt {
			expected = reflect.ValueOf(expectedInt).Convert(field.Type()).Interface()
		}
		got = field.Interface()

		// Handle byte slice comparison
		if field.Type().Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
			if str, ok := expected.(string); ok {
				expected = []byte(str)
			}
		}
	} else {
		// For unexported fields, use type-specific accessors
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val := field.Int()
			if val > int64(int(^uint(0)>>1)) || val < int64(-int(^uint(0)>>1)-1) {
				t.Errorf("field %s value %d overflows int", fieldName, val)
				return
			}
			got = int(val)
			if hasExpectedInt {
				expected = expectedInt
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val := field.Uint()
			if val > uint64(int(^uint(0)>>1)) {
				t.Errorf("field %s value %d overflows int", fieldName, val)
				return
			}
			got = int(val)
			if hasExpectedInt {
				expected = expectedInt
			}
		case reflect.Bool:
			got = field.Bool()
		case reflect.String:
			got = field.String()
		case reflect.Slice:
			// Handle byte slice comparison
			if field.Type().Elem().Kind() == reflect.Uint8 {
				got = field.Bytes()
				if str, ok := expected.(string); ok {
					expected = []byte(str)
				}
			}
		default:
			t.Errorf("cannot compare unexported field %s of kind %s", fieldName, field.Kind())
			return
		}
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("%s = %v, want %v", fieldName, got, expected)
	}
}
