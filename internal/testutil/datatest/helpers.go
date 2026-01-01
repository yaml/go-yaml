// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package datatest

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// HexToBytes converts a hex string to bytes.
// This is useful for test data that needs to specify binary data.
func HexToBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("invalid hex string: %s: %v", s, err)
	}
	return b
}

// GetField uses reflection to get a field value from a struct or pointer to struct.
// This is useful for test assertions that need to check internal field values.
func GetField(t *testing.T, obj any, fieldName string) any {
	t.Helper()
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("field %s not found in %T", fieldName, obj)
	}
	return field.Interface()
}

// CallMethod calls a method on an object using reflection.
// This is useful for test cases that need to call methods dynamically.
// Returns the slice of return values from the method call as reflect.Value wrappers.
// Callers must extract the actual values using methods like .Interface(), .Int(), .String(), etc.
func CallMethod(t *testing.T, obj any, methodName string, args []any) []reflect.Value {
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

// WantBool extracts a bool value from a test case's Want field.
// If Want is nil, returns the defaultVal.
// This is useful for test cases where the expected result is a boolean.
func WantBool(t *testing.T, want any, defaultVal bool) bool {
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

// WantString extracts a string value from a test case's Want field.
// If Want is nil, returns the defaultVal.
func WantString(t *testing.T, want any, defaultVal string) string {
	t.Helper()
	if want == nil {
		return defaultVal
	}
	strVal, ok := want.(string)
	if !ok {
		t.Fatalf("Want should be string, got %T", want)
	}
	return strVal
}

// WantInt extracts an int value from a test case's Want field.
// If Want is nil, returns the defaultVal.
func WantInt(t *testing.T, want any, defaultVal int) int {
	t.Helper()
	if want == nil {
		return defaultVal
	}
	intVal, ok := want.(int)
	if !ok {
		t.Fatalf("Want should be int, got %T", want)
	}
	return intVal
}

// WantSlice extracts a slice value from a test case's Want field.
// If Want is nil, returns nil.
func WantSlice(t *testing.T, want any) []any {
	t.Helper()
	if want == nil {
		return nil
	}
	sliceVal, ok := want.([]any)
	if !ok {
		t.Fatalf("Want should be []interface{}, got %T", want)
	}
	return sliceVal
}

// SetFieldValue sets a field value on a struct using reflection.
// This is useful for test setup that needs to configure objects dynamically.
func SetFieldValue(t *testing.T, obj any, fieldName string, value any) {
	t.Helper()
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("field %s not found in %T", fieldName, obj)
	}
	if !field.CanSet() {
		t.Fatalf("field %s cannot be set (unexported?)", fieldName)
	}
	field.Set(reflect.ValueOf(value))
}

// TrimTrailingNewline removes a trailing newline character from a string if present.
// This is useful for YAML literal blocks which add a trailing newline.
func TrimTrailingNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		return s[:len(s)-1]
	}
	return s
}

// GenerateData generates test data from a generator specification.
// Supports simple loops, concatenation of parts (join), and nested loops.
//
// Format:
//
//	Simple loop: {loop: ["value", count]}
//	Join parts: {join: [{text: "..."}, {loop: ["...", count]}]}
//	Nested: {join: [...], loop: count}
func GenerateData(spec any) ([]byte, error) {
	specMap, ok := spec.(map[string]any)
	if !ok {
		// If it's just a string, return it as-is
		if str, ok := spec.(string); ok {
			return []byte(str), nil
		}
		return nil, fmt.Errorf("data spec must be map or string, got %T", spec)
	}

	// Check for simple loop: {loop: ["value", count]}
	if loopVal, hasLoop := specMap["loop"]; hasLoop {
		if _, hasJoin := specMap["join"]; !hasJoin {
			return generateSimpleLoop(loopVal)
		}
	}

	// Check for join: {join: [{text: "..."}, {loop: ["...", count]}]}
	if joinVal, hasJoin := specMap["join"]; hasJoin {
		result, err := generateJoin(joinVal)
		if err != nil {
			return nil, err
		}

		// Check for loop: repeat the entire join N times
		if loopVal, hasLoop := specMap["loop"]; hasLoop {
			count, ok := loopVal.(int)
			if !ok {
				return nil, fmt.Errorf("loop count must be int, got %T", loopVal)
			}
			return []byte(strings.Repeat(string(result), count)), nil
		}

		return result, nil
	}

	return nil, fmt.Errorf("data spec must have 'loop' or 'join' field")
}

func generateSimpleLoop(loopVal any) ([]byte, error) {
	loopArr, ok := loopVal.([]any)
	if !ok {
		return nil, fmt.Errorf("loop must be array [value, count], got %T", loopVal)
	}

	if len(loopArr) != 2 {
		return nil, fmt.Errorf("loop must have 2 elements [value, count], got %d", len(loopArr))
	}

	value, ok := loopArr[0].(string)
	if !ok {
		return nil, fmt.Errorf("loop value must be string, got %T", loopArr[0])
	}

	count, ok := loopArr[1].(int)
	if !ok {
		return nil, fmt.Errorf("loop count must be int, got %T", loopArr[1])
	}

	return []byte(strings.Repeat(value, count)), nil
}

func generateJoin(joinVal any) ([]byte, error) {
	joinList, ok := joinVal.([]any)
	if !ok {
		return nil, fmt.Errorf("join must be array, got %T", joinVal)
	}

	var result strings.Builder
	for i, item := range joinList {
		itemMap, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("join item %d must be map, got %T", i, item)
		}

		// Check for text field
		if text, hasText := itemMap["text"]; hasText {
			textStr, ok := text.(string)
			if !ok {
				return nil, fmt.Errorf("join item %d text must be string, got %T", i, text)
			}
			result.WriteString(textStr)
			continue
		}

		// Check for loop field
		if loopVal, hasLoop := itemMap["loop"]; hasLoop {
			loopData, err := generateSimpleLoop(loopVal)
			if err != nil {
				return nil, fmt.Errorf("join item %d: %w", i, err)
			}
			result.Write(loopData)
			continue
		}

		return nil, fmt.Errorf("join item %d must have 'text' or 'loop' field", i)
	}

	return []byte(result.String()), nil
}
