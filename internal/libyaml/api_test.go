// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestAPI(t *testing.T) {
	RunTestCases(t, "api.yaml", map[string]TestHandler{
		"api-new":       runAPINewTest,
		"api-method":    runAPIMethodTest,
		"api-panic":     runAPIPanicTest,
		"api-delete":    runAPIDeleteTest,
		"api-new-event": runAPINewEventTest,
	})
}

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
