// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package datatest provides utilities for data-driven testing with YAML test files.
// It extracts and generalizes the testing infrastructure originally from internal/libyaml.
package datatest

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// LoadYAMLFunc is a function type for loading YAML data.
// Different packages can provide their own YAML loading implementation.
type LoadYAMLFunc func([]byte) (any, error)

// LoadTestCasesFromFile loads and normalizes test cases from a YAML file.
// It reads the file, parses it with the provided loadYAML function, and normalizes
// the type-as-key format to standard format.
func LoadTestCasesFromFile(filename string, loadYAML LoadYAMLFunc) ([]map[string]any, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	rawData, err := loadYAML(data)
	if err != nil {
		return nil, err
	}

	rawCases, ok := rawData.([]any)
	if !ok {
		return nil, fmt.Errorf("expected []interface{}, got %T", rawData)
	}

	result := make([]map[string]any, 0, len(rawCases))
	for _, item := range rawCases {
		rawCase, ok := item.(map[string]any)
		if !ok {
			continue
		}

		// Normalize type-as-key format: {test-type: {...}} -> {type: test-type, ...}
		normalized := NormalizeTypeAsKey(rawCase)
		result = append(result, normalized)
	}

	return result, nil
}

// UnmarshalStruct populates a struct from a map using reflection and yaml tags.
// This function is exported so it can be used by other packages for data-driven testing.
//
// Example:
//
//	type TestCase struct {
//	    Name string `yaml:"name"`
//	    Data string `yaml:"data"`
//	}
//	var tc TestCase
//	err := datatest.UnmarshalStruct(&tc, map[string]interface{}{
//	    "name": "test1",
//	    "data": "hello",
//	})
func UnmarshalStruct(target any, data map[string]any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be pointer to struct, got %T", target)
	}

	v = v.Elem()
	t := v.Type()

	// Build map of yaml tag names to field indices (support multiple fields per tag)
	fieldMap := make(map[string][]int)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		yamlTag := field.Tag.Get("yaml")
		if yamlTag != "" && yamlTag != "-" {
			// Remove options like ",omitempty"
			if idx := strings.Index(yamlTag, ","); idx != -1 {
				yamlTag = yamlTag[:idx]
			}
			fieldMap[yamlTag] = append(fieldMap[yamlTag], i)
		}
	}

	// Populate fields from map
	for key, value := range data {
		fieldIndices, ok := fieldMap[key]
		if !ok {
			// Skip unknown fields
			continue
		}

		// Try each field with this tag until one succeeds
		var lastErr error
		for _, fieldIdx := range fieldIndices {
			field := v.Field(fieldIdx)
			if !field.CanSet() {
				continue
			}

			if err := setField(field, value); err != nil {
				lastErr = err
				continue
			}
			// Success, move to next key
			lastErr = nil
			break
		}

		// If all fields failed, return the last error
		if lastErr != nil {
			return fmt.Errorf("field %s: %w", key, lastErr)
		}
	}

	return nil
}

// setField sets a reflect.Value from an interface{} value
func setField(field reflect.Value, value any) error {
	if !field.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	if value == nil {
		// Set zero value
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	fieldType := field.Type()

	// Handle custom types with FromValue method (IntOrStr, ByteInput, Args)
	if field.CanAddr() {
		addr := field.Addr()
		if converter, ok := addr.Interface().(interface{ FromValue(any) error }); ok {
			if err := converter.FromValue(value); err != nil {
				return err
			}
			return nil
		}
	}

	// Handle basic types
	valueRefl := reflect.ValueOf(value)
	valueType := valueRefl.Type()

	// Direct assignment if types match
	if valueType.AssignableTo(fieldType) {
		field.Set(valueRefl)
		return nil
	}

	// Check if conversion is possible for compatible types
	if valueType.ConvertibleTo(fieldType) && valueRefl.CanConvert(fieldType) {
		field.Set(valueRefl.Convert(fieldType))
		return nil
	}

	// Handle conversions
	switch fieldType.Kind() {
	case reflect.String:
		if str, ok := value.(string); ok {
			field.SetString(str)
			return nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !field.CanInt() {
			return fmt.Errorf("field cannot store int values")
		}
		if !valueRefl.CanInt() {
			return fmt.Errorf("field type %v expects an integer value, but got %T", fieldType, value)
		}
		i64 := valueRefl.Int()
		if field.OverflowInt(i64) {
			return fmt.Errorf("value %v overflows %v", value, fieldType)
		}
		field.SetInt(i64)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if !field.CanUint() {
			return fmt.Errorf("field cannot store uint values")
		}
		// Check for negative integers first (most specific error case)
		if valueRefl.CanInt() && valueRefl.Int() < 0 {
			return fmt.Errorf("field type %v expects an unsigned integer, but got negative value %v of type %T", fieldType, value, value)
		}
		// Then check if value can be converted to uint
		if !valueRefl.CanUint() {
			return fmt.Errorf("field type %v expects an unsigned integer value, but got %T", fieldType, value)
		}
		u64 := valueRefl.Uint()
		if field.OverflowUint(u64) {
			return fmt.Errorf("value %v overflows %v", value, fieldType)
		}
		field.SetUint(u64)
		return nil
	case reflect.Bool:
		if valueRefl.Kind() != reflect.Bool {
			// Check if value can be converted to bool
			if valueRefl.Type().ConvertibleTo(fieldType) {
				field.Set(valueRefl.Convert(fieldType))
				return nil
			}
			return fmt.Errorf("field type %v expects a boolean value, but got %T", fieldType, value)
		}
		field.SetBool(valueRefl.Bool())
		return nil
	case reflect.Float32, reflect.Float64:
		if !field.CanFloat() {
			return fmt.Errorf("field cannot store float values")
		}
		if !valueRefl.CanFloat() {
			return fmt.Errorf("field type %v expects a floating-point value, but got %T", fieldType, value)
		}
		f64 := valueRefl.Float()
		if field.OverflowFloat(f64) {
			return fmt.Errorf("value %f overflows %v", f64, fieldType)
		}
		field.SetFloat(f64)
		return nil
	case reflect.Slice:
		return setSliceField(field, value)
	case reflect.Map:
		return setMapField(field, value)
	case reflect.Struct:
		// Recursively unmarshal nested structs
		if m, ok := value.(map[string]any); ok {
			if !field.CanAddr() {
				return fmt.Errorf("cannot take address of field for nested struct unmarshaling")
			}
			return UnmarshalStruct(field.Addr().Interface(), m)
		}
	case reflect.Interface:
		// Just set the value directly
		field.Set(valueRefl)
		return nil
	case reflect.Ptr:
		// Handle pointer types
		if fieldType.Elem().Kind() == reflect.Struct {
			m, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("expected map for struct pointer, got %T", value)
			}
			newStruct := reflect.New(fieldType.Elem())
			if err := UnmarshalStruct(newStruct.Interface(), m); err != nil {
				return err
			}
			field.Set(newStruct)
			return nil
		} else {
			// Handle pointer to basic types (like *bool, *int, *string)
			ptr := reflect.New(fieldType.Elem())
			if err := setField(ptr.Elem(), value); err != nil {
				return err
			}
			field.Set(ptr)
			return nil
		}
	}

	return fmt.Errorf("cannot convert %T to %v", value, fieldType)
}

// NormalizeTypeAsKey converts maps with type as key to standard format.
// Example: {"SCALAR_TOKEN": {"value": "x"}} -> {"type": "SCALAR_TOKEN", "value": "x"}
// This function is exported so it can be used by test code that needs to normalize
// YAML test data with type-as-key format.
func NormalizeTypeAsKey(itemMap map[string]any) map[string]any {
	// Check if map has exactly one key and no "type" field
	if len(itemMap) == 1 {
		_, hasType := itemMap["type"]
		if !hasType {
			// Get the single key
			for key, value := range itemMap {
				// Check if key looks like a type constant (all uppercase with underscores)
				if IsTypeConstant(key) {
					// Check if value is a map
					if subMap, ok := value.(map[string]any); ok {
						// Create new map with "type" field for test type
						newMap := map[string]any{"type": key}
						// Merge in the sub-map fields, but preserve sub-map's "type" as "output_type" if it exists
						for k, v := range subMap {
							if k == "type" {
								// Sub-map has its own "type" field - preserve it as "output_type"
								newMap["output_type"] = v
							} else {
								newMap[k] = v
							}
						}
						return newMap
					}
				}
			}
		}
	}
	return itemMap
}

// IsTypeConstant checks if a string looks like a type constant.
// Accepts: UPPERCASE_WITH_UNDERSCORES or lowercase-with-hyphens
// This function is exported for use in test infrastructure.
func IsTypeConstant(s string) bool {
	if s == "" {
		return false
	}
	hasUpper := false
	hasLower := false
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
		} else if c >= 'a' && c <= 'z' {
			hasLower = true
		} else if !(c == '_' || c == '-' || c >= '0' && c <= '9') {
			return false
		}
	}
	// Either all uppercase (EVENT_TYPE) or has lowercase (test-type)
	return hasUpper || hasLower
}

// setSliceField sets a slice field from a value
func setSliceField(field reflect.Value, value any) error {
	sliceVal, ok := value.([]any)
	if !ok {
		return fmt.Errorf("expected []interface{} for slice, got %T", value)
	}

	elemType := field.Type().Elem()
	newSlice := reflect.MakeSlice(field.Type(), len(sliceVal), len(sliceVal))

	for i, item := range sliceVal {
		elem := newSlice.Index(i)

		// Handle struct elements
		if elemType.Kind() == reflect.Struct {
			var m map[string]any

			// Check if item is a scalar string (simplified format)
			if strVal, ok := item.(string); ok {
				// Convert scalar to map with type field
				m = map[string]any{"type": strVal}
			} else {
				m, ok = item.(map[string]any)
				if !ok {
					return fmt.Errorf("slice element %d: expected map for struct, got %T", i, item)
				}
				// Normalize type-as-key format
				m = NormalizeTypeAsKey(m)
			}

			if !elem.CanAddr() {
				return fmt.Errorf("slice element %d: cannot take address for struct unmarshaling", i)
			}
			if err := UnmarshalStruct(elem.Addr().Interface(), m); err != nil {
				return fmt.Errorf("slice element %d: %w", i, err)
			}
			continue
		}

		// Handle other types
		if err := setField(elem, item); err != nil {
			return fmt.Errorf("slice element %d: %w", i, err)
		}
	}

	field.Set(newSlice)
	return nil
}

// setMapField sets a map field from a value
func setMapField(field reflect.Value, value any) error {
	mapVal, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("expected map[string]interface{}, got %T", value)
	}

	mapType := field.Type()
	if mapType.Key().Kind() != reflect.String {
		return fmt.Errorf("only string keys supported for maps")
	}

	newMap := reflect.MakeMap(mapType)
	valueType := mapType.Elem()

	for k, v := range mapVal {
		keyRefl := reflect.ValueOf(k)
		valueRefl := reflect.New(valueType).Elem()

		// Handle struct values
		if valueType.Kind() == reflect.Struct {
			m, ok := v.(map[string]any)
			if !ok {
				return fmt.Errorf("map value for key %s: expected map for struct, got %T", k, v)
			}
			if !valueRefl.CanAddr() {
				return fmt.Errorf("map value for key %s: cannot take address for struct unmarshaling", k)
			}
			if err := UnmarshalStruct(valueRefl.Addr().Interface(), m); err != nil {
				return fmt.Errorf("map value for key %s: %w", k, err)
			}
		} else {
			if err := setField(valueRefl, v); err != nil {
				return fmt.Errorf("map value for key %s: %w", k, err)
			}
		}

		newMap.SetMapIndex(keyRefl, valueRefl)
	}

	field.Set(newMap)
	return nil
}

// LoadTestCasesFunc is a function type for loading test cases from a file.
// Each package can provide its own implementation that uses its preferred YAML loader.
type LoadTestCasesFunc func(filename string) ([]map[string]any, error)
