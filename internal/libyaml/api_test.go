// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
	"go.yaml.in/yaml/v3"
)

func TestAPI(t *testing.T) {
	RunTestCases(t, "api_test.yaml", map[string]TestHandler{
		"api-new":       runAPINewTest,
		"api-method":    runAPIMethodTest,
		"api-panic":     runAPIPanicTest,
		"api-delete":    runAPIDeleteTest,
		"api-new-event": runAPINewEventTest,
	})
}

func runAPINewTest(t *testing.T, tc TestCase) {
	t.Helper()

	obj := createObject(tc.Constructor)
	runFieldChecks(t, obj, tc.Checks)
}

func runAPIMethodTest(t *testing.T, tc TestCase) {
	t.Helper()

	obj := createObject(tc.Constructor)

	// Run setup if any
	if tc.Setup != nil {
		if setupList, ok := tc.Setup.([]interface{}); ok && len(setupList) > 0 {
			callMethodFromList(obj, setupList, tc.Bytes)
		}
	}

	// Call the main method
	callMethodFromList(obj, tc.Method, tc.Bytes)

	// Run checks
	runFieldChecks(t, obj, tc.Checks)
}

func runAPIPanicTest(t *testing.T, tc TestCase) {
	t.Helper()

	obj := createObject(tc.Constructor)

	// Run setup if any
	if tc.Setup != nil {
		if setupList, ok := tc.Setup.([]interface{}); ok && len(setupList) > 0 {
			callMethodFromList(obj, setupList, tc.Bytes)
		}
	}

	// The main method call should panic
	// Want can be either a string or a single-element sequence
	var wantMsg string
	switch v := tc.Want.(type) {
	case string:
		wantMsg = v
	case []interface{}:
		wantMsg = v[0].(string)
	default:
		t.Fatalf("want must be a string or sequence, got %T", tc.Want)
	}
	assert.PanicMatchesf(t, wantMsg, func() {
		callMethodFromList(obj, tc.Method, tc.Bytes)
	}, "Expected panic: %s", wantMsg)
}

func runAPIDeleteTest(t *testing.T, tc TestCase) {
	t.Helper()

	obj := createObject(tc.Constructor)

	// Run setup if any
	if tc.Setup != nil {
		if setupList, ok := tc.Setup.([]interface{}); ok && len(setupList) > 0 {
			callMethodFromList(obj, setupList, tc.Bytes)
		}
	}

	// Call Delete method
	callMethodFromList(obj, []interface{}{"Delete"}, false)

	// Run checks after delete
	runFieldChecks(t, obj, tc.Checks)
}

func runAPINewEventTest(t *testing.T, tc TestCase) {
	t.Helper()

	event := createEventFromList(tc.Method, tc.Bytes)
	runFieldChecks(t, &event, tc.Checks)
}

// createObject creates a Parser or Emitter based on constructor name
func createObject(constructor string) interface{} {
	switch constructor {
	case "NewParser":
		p := NewParser()
		return &p
	case "NewEmitter":
		e := NewEmitter()
		return &e
	default:
		panic("unknown constructor: " + constructor)
	}
}

// createEventFromList creates an Event from a method list [constructor, args...]
func createEventFromList(methodList []interface{}, useBytes bool) Event {
	if len(methodList) == 0 {
		panic("empty method list")
	}

	constructor := methodList[0].(string)
	args := methodList[1:]

	switch constructor {
	case "NewStreamStartEvent":
		encoding := parseArg(args[0])
		return NewStreamStartEvent(Encoding(encoding))
	case "NewStreamEndEvent":
		return NewStreamEndEvent()
	case "NewDocumentEndEvent":
		implicit := parseBoolArg(args[0])
		return NewDocumentEndEvent(implicit)
	case "NewAliasEvent":
		anchor := parseStringArg(args[0], useBytes)
		return NewAliasEvent([]byte(anchor))
	case "NewSequenceEndEvent":
		return NewSequenceEndEvent()
	case "NewMappingEndEvent":
		return NewMappingEndEvent()
	default:
		panic("unknown event constructor: " + constructor)
	}
}

// callMethodFromList calls a method from a list [methodName, args...]
func callMethodFromList(obj interface{}, methodList []interface{}, useBytes bool) {
	if len(methodList) == 0 {
		panic("empty method list")
	}

	methodName := methodList[0].(string)
	args := methodList[1:]

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	m := v.Addr().MethodByName(methodName)
	if !m.IsValid() {
		panic("method not found: " + methodName)
	}

	methodType := m.Type()

	// Build argument list
	var callArgs []reflect.Value
	for i, arg := range args {
		paramType := methodType.In(i)

		// Handle different parameter types
		if paramType.Kind() == reflect.Bool {
			val := parseBoolArg(arg)
			callArgs = append(callArgs, reflect.ValueOf(val))
		} else if paramType.Kind() == reflect.Slice && paramType.Elem().Kind() == reflect.Uint8 {
			// Byte slice parameter
			str := parseStringArg(arg, useBytes)
			callArgs = append(callArgs, reflect.ValueOf([]byte(str)))
		} else {
			// Try parsing as constant/int
			val := parseArg(arg)
			convertedVal := reflect.ValueOf(val).Convert(paramType)
			callArgs = append(callArgs, convertedVal)
		}
	}

	// Call the method
	m.Call(callArgs)
}

// parseArg parses an argument which could be int, bool, or string constant
func parseArg(arg interface{}) int {
	switch v := arg.(type) {
	case int:
		return v
	case string:
		if looksLikeConstant(v) {
			return parseConstant(v)
		}
		// Try parsing as int
		if val, err := strconv.Atoi(v); err == nil {
			return val
		}
		panic(fmt.Sprintf("cannot parse arg as int: %v", arg))
	default:
		panic(fmt.Sprintf("unsupported arg type: %T", arg))
	}
}

// parseBoolArg parses a boolean argument
func parseBoolArg(arg interface{}) bool {
	switch v := arg.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	default:
		panic(fmt.Sprintf("cannot parse arg as bool: %v", arg))
	}
}

// parseStringArg parses a string argument
func parseStringArg(arg interface{}, useBytes bool) string {
	if str, ok := arg.(string); ok {
		return str
	}
	panic(fmt.Sprintf("cannot parse arg as string: %v", arg))
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
func parseConstant(name string) int {
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
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: name,
	}
	err := ios.UnmarshalYAML(node)
	if err != nil {
		panic(fmt.Sprintf("failed to parse constant %q: %v", name, err))
	}
	return ios.Value
}

// runFieldChecks runs field checks on an object
func runFieldChecks(t *testing.T, obj interface{}, checks map[string]FieldCheck) {
	t.Helper()

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for fieldName, check := range checks {
		// Handle special field names like buffer-0, buffer-1
		var field reflect.Value
		var bufferIndex int
		var isBufferIndex bool

		if strings.HasPrefix(fieldName, "buffer-") {
			// Extract index from buffer-N
			_, err := fmt.Sscanf(fieldName, "buffer-%d", &bufferIndex)
			if err == nil {
				isBufferIndex = true
				// Get the buffer field
				field = v.FieldByName("buffer")
				if !field.IsValid() {
					t.Fatalf("buffer field not found for %s", fieldName)
				}
			}
		}

		if !isBufferIndex {
			// Convert hyphenated YAML key to underscored Go field name
			goFieldName := strings.ReplaceAll(fieldName, "-", "_")
			field = v.FieldByName(goFieldName)
			if !field.IsValid() {
				t.Fatalf("field not found: %s (looking for %s)", fieldName, goFieldName)
			}
		}

		if check.NilCheck != nil {
			wantNil := *check.NilCheck
			isNil := field.IsNil()
			if wantNil != isNil {
				if wantNil {
					t.Errorf("%s should be nil", fieldName)
				} else {
					t.Errorf("%s should not be nil", fieldName)
				}
			}
		}

		if check.Cap > 0 {
			if field.Cap() != check.Cap {
				t.Errorf("%s cap = %d, want %d", fieldName, field.Cap(), check.Cap)
			}
		}

		if check.Len > 0 {
			if field.Len() != check.Len {
				t.Errorf("%s len = %d, want %d", fieldName, field.Len(), check.Len)
			}
		}

		if check.LenGt > 0 {
			if field.Len() <= check.LenGt {
				t.Errorf("%s len = %d, want > %d", fieldName, field.Len(), check.LenGt)
			}
		}

		if check.Equal != nil {
			// Parse constant if it's a string that looks like a constant name
			var expectedInt int
			hasExpectedInt := false
			var expected interface{} = check.Equal
			if str, ok := check.Equal.(string); ok && looksLikeConstant(str) {
				expectedInt = parseConstant(str)
				hasExpectedInt = true
			} else if intVal, ok := check.Equal.(int); ok {
				expectedInt = intVal
				hasExpectedInt = true
			}

			// Get value based on type (handle unexported fields)
			var got interface{}

			// Handle buffer-N special case
			if isBufferIndex {
				// Check specific byte in buffer
				if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
					if bufferIndex >= field.Len() {
						t.Errorf("%s: index %d out of range (buffer len=%d)", fieldName, bufferIndex, field.Len())
						continue
					}
					got = int(field.Index(bufferIndex).Uint())
					expected = expectedInt
				} else {
					t.Errorf("%s: buffer field is not a byte slice", fieldName)
					continue
				}
			} else if field.CanInterface() {
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
					got = int(field.Int())
					if hasExpectedInt {
						expected = expectedInt
					}
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					got = int(field.Uint())
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
					continue
				}
			}

			if !reflect.DeepEqual(got, expected) {
				t.Errorf("%s = %v, want %v", fieldName, got, expected)
			}
		}

		if check.Gte > 0 {
			var got int
			if isBufferIndex {
				t.Errorf("%s: gte check not supported on buffer indices", fieldName)
				continue
			}

			if field.CanInterface() {
				// Try to convert to int
				switch v := field.Interface().(type) {
				case int:
					got = v
				case int64:
					got = int(v)
				case int32:
					got = int(v)
				default:
					t.Errorf("%s: gte check requires int field, got %T", fieldName, v)
					continue
				}
			} else {
				// For unexported fields
				switch field.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					got = int(field.Int())
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					got = int(field.Uint())
				default:
					t.Errorf("%s: gte check requires numeric field, got %s", fieldName, field.Kind())
					continue
				}
			}

			if got < check.Gte {
				t.Errorf("%s = %d, want >= %d", fieldName, got, check.Gte)
			}
		}
	}
}
