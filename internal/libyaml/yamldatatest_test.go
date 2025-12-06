// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

// TestCase represents a single test case loaded from YAML
type TestCase struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`

	// Common fields
	Yaml       string      `yaml:"yaml"`
	InputHex   string      `yaml:"input_hex"`
	InputBytes string      `yaml:"input_bytes"`
	Want       interface{} `yaml:"want"`
	WantSpecs  []EventSpec // Populated from Want for detailed tests

	// scan_tokens_detailed
	WantTokens []TokenSpec // Populated from Want for detailed tests

	// emit tests
	Events       []EventSpec   `yaml:"data"`
	WantContains []string      // Populated from Want for emit tests
	Config       EmitterConfig `yaml:"conf"`

	// style_accessor tests (must come before Checks due to shared yaml:"test" tag)
	StyleTest []interface{} `yaml:"test"` // [Method, STYLE] where Method is string and STYLE is int or string constant

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
	Enum []interface{} `yaml:"enum"` // [Type, Value] where Type is string and Value is int or string

	// api_method, api_panic, api_delete tests, and reader tests
	Bytes  bool          `yaml:"byte"`
	Method []interface{} `yaml:"call"`
	Setup  interface{}   `yaml:"init"` // Can be []interface{} (api tests) or map[string]interface{} (reader tests)
}

// IntOrStr can be converted from either an int or a string constant name
// constantMap maps constant names to their integer values
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

// resolveConstant converts a constant name string to its integer value
func resolveConstant(t *testing.T, name string) int {
	t.Helper()
	val, ok := constantMap[name]
	if !ok {
		t.Fatalf("unknown constant name: %s", name)
	}
	return val
}

// IntOrStr can be converted from either an int or a string constant name
type IntOrStr struct {
	Value int
}

func (ios *IntOrStr) FromValue(v interface{}) error {
	// Try int first
	if intVal, ok := v.(int); ok {
		ios.Value = intVal
		return nil
	}

	// Otherwise, it should be a string constant name
	strVal, ok := v.(string)
	if !ok {
		return fmt.Errorf("IntOrStr value must be int or string, got %T", v)
	}

	val, ok := constantMap[strVal]
	if !ok {
		return fmt.Errorf("unknown constant name: %s", strVal)
	}
	ios.Value = val
	return nil
}

// ByteInput can be converted from either a string or a sequence of hex bytes
type ByteInput []byte

func (bi *ByteInput) FromValue(v interface{}) error {
	// Try string first
	if strVal, ok := v.(string); ok {
		*bi = []byte(strVal)
		return nil
	}

	// Try single int (convert to single-byte array)
	if intVal, ok := v.(int); ok {
		if intVal < 0 || intVal > 255 {
			return fmt.Errorf("byte value out of range [0-255]: %d", intVal)
		}
		*bi = []byte{byte(intVal)}
		return nil
	}

	// Otherwise, it should be a sequence of integer bytes
	intSlice, ok := v.([]interface{})
	if !ok {
		return fmt.Errorf("input must be a string, int, or sequence of integers, got %T", v)
	}

	// Convert integers to bytes
	bytes := make([]byte, len(intSlice))
	for i, val := range intSlice {
		intVal, ok := val.(int)
		if !ok {
			return fmt.Errorf("byte array element must be int, got %T", val)
		}
		if intVal < 0 || intVal > 255 {
			return fmt.Errorf("byte value out of range [0-255]: %d", intVal)
		}
		bytes[i] = byte(intVal)
	}
	*bi = bytes
	return nil
}

// Args can be converted from either a single value or an array of values
type Args []interface{}

func (a *Args) FromValue(v interface{}) error {
	// Try array first
	if arrVal, ok := v.([]interface{}); ok {
		*a = arrVal
		return nil
	}

	// Otherwise, it's a single scalar value - wrap it in a slice
	*a = []interface{}{v}
	return nil
}

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
	Nil   []interface{} `yaml:"nil"`
	Cap   []interface{} `yaml:"cap"`
	Len   []interface{} `yaml:"len"`
	LenGt []interface{} `yaml:"len-gt"` // Length greater than
	Eq    []interface{} `yaml:"eq"`
	Gte   []interface{} `yaml:"gte"` // Greater than or equal
}

// CharTestCase represents a character classification test case
type CharTestCase struct {
	InputHex string      `yaml:"input_hex"`
	Pos      int         `yaml:"pos"`
	Want     interface{} `yaml:"want"` // Can be bool or int
}

// unmarshalTestCases converts raw YAML data to TestCase structs using yamltest
func unmarshalTestCases(data interface{}) ([]TestCase, error) {
	casesSlice, ok := data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected []interface{}, got %T", data)
	}

	var testCases []TestCase
	for i, item := range casesSlice {
		caseMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("test case %d: expected map[string]interface{}, got %T", i, item)
		}

		// Normalize type-as-key format for top-level test cases
		caseMap = normalizeTypeAsKey(caseMap)

		var tc TestCase
		if err := UnmarshalStruct(&tc, caseMap); err != nil {
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

	// Load YAML using LoadYAML from testloader.go
	rawData, err := LoadYAML(data)
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
				wantSlice, ok := cases[i].Want.([]interface{})
				if !ok {
					return nil, fmt.Errorf("test %s: want should be a sequence, got %T", cases[i].Name, cases[i].Want)
				}
				cases[i].WantSpecs = make([]EventSpec, len(wantSlice))
				for j, item := range wantSlice {
					var itemMap map[string]interface{}
					// Check if item is a scalar string (simplified format)
					if strVal, ok := item.(string); ok {
						// Convert scalar to map with type field
						itemMap = map[string]interface{}{"type": strVal}
					} else {
						itemMap, ok = item.(map[string]interface{})
						if !ok {
							return nil, fmt.Errorf("test %s: want[%d] should be a map or string, got %T", cases[i].Name, j, item)
						}
						// Normalize type-as-key format
						itemMap = normalizeTypeAsKey(itemMap)
					}
					if err := UnmarshalStruct(&cases[i].WantSpecs[j], itemMap); err != nil {
						return nil, fmt.Errorf("test %s: want[%d]: %w", cases[i].Name, j, err)
					}
				}
			}
		case "scan-tokens-detailed":
			if cases[i].Want != nil {
				// Want should be []interface{} of maps, convert to []TokenSpec
				wantSlice, ok := cases[i].Want.([]interface{})
				if !ok {
					return nil, fmt.Errorf("test %s: want should be a sequence, got %T", cases[i].Name, cases[i].Want)
				}
				cases[i].WantTokens = make([]TokenSpec, len(wantSlice))
				for j, item := range wantSlice {
					var itemMap map[string]interface{}
					// Check if item is a scalar string (simplified format)
					if strVal, ok := item.(string); ok {
						// Convert scalar to map with type field
						itemMap = map[string]interface{}{"type": strVal}
					} else {
						itemMap, ok = item.(map[string]interface{})
						if !ok {
							return nil, fmt.Errorf("test %s: want[%d] should be a map or string, got %T", cases[i].Name, j, item)
						}
						// Normalize type-as-key format
						itemMap = normalizeTypeAsKey(itemMap)
					}
					if err := UnmarshalStruct(&cases[i].WantTokens[j], itemMap); err != nil {
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
				case []interface{}:
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
func HexToBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("invalid hex string: %s: %v", s, err)
	}
	return b
}

// GetField uses reflection to get a field value from a struct
func GetField(t *testing.T, obj interface{}, fieldName string) interface{} {
	t.Helper()
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("field %s not found", fieldName)
	}
	return field.Interface()
}

// CreateArgValue creates a value from an ArgSpec
func CreateArgValue(t *testing.T, spec ArgSpec) interface{} {
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

// CallMethod calls a method on an object using reflection
func CallMethod(t *testing.T, obj interface{}, methodName string, args []interface{}) []reflect.Value {
	t.Helper()
	v := reflect.ValueOf(obj)
	method := v.MethodByName(methodName)
	if !method.IsValid() {
		t.Fatalf("method %s not found on %T", methodName, obj)
	}

	var argValues []reflect.Value
	for _, arg := range args {
		argValues = append(argValues, reflect.ValueOf(arg))
	}

	return method.Call(argValues)
}

// CreateObject creates an object using a constructor function
func CreateObject(t *testing.T, constructorName string) interface{} {
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

// parseEventsDetailed is a helper to parse input and return full events
func parseEventsDetailed(input string) ([]Event, bool) {
	parser := NewParser()
	parser.SetInputString([]byte(input))

	var events []Event
	for {
		var event Event
		if !parser.Parse(&event) {
			if parser.ErrorType != NO_ERROR {
				return nil, false
			}
			return events, true
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
		if !parser.Scan(&token) {
			if parser.ErrorType != NO_ERROR {
				return nil, false
			}
			return types, true
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
		if !parser.Scan(&token) {
			if parser.ErrorType != NO_ERROR {
				return nil, false
			}
			return tokens, true
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
	for _, expected := range tc.WantContains {
		assert.Truef(t, bytes.Contains(output, []byte(expected)),
			"output should contain %q, got %q", expected, result)
	}
}

// GetWriter extracts an io.Writer from an interface value
func GetWriter(t *testing.T, v interface{}) io.Writer {
	t.Helper()
	if w, ok := v.(io.Writer); ok {
		return w
	}
	t.Fatalf("value is not an io.Writer: %T", v)
	return nil
}

// GetReader extracts an io.Reader from an interface value
func GetReader(t *testing.T, v interface{}) io.Reader {
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
func WantBool(t *testing.T, want interface{}, defaultVal bool) bool {
	t.Helper()
	if want == nil {
		return defaultVal
	}
	boolVal, ok := want.(bool)
	if !ok {
		t.Fatalf("Want should be bool, got %T", want)
	}
	return boolVal
}
