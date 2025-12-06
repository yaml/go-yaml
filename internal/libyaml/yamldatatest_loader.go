// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"fmt"
	"reflect"
	"strings"
)

// coerceScalar converts a YAML scalar string to an appropriate Go type
func coerceScalar(value string) interface{} {
	// Try bool and null
	switch value {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}

	// Try hex int (0x or 0X prefix) - needed for test data byte arrays
	var intVal int
	if _, err := fmt.Sscanf(strings.ToLower(value), "0x%x", &intVal); err == nil {
		return intVal
	}

	// Try float (must check before int because %d will parse "1.5" as "1")
	if strings.Contains(value, ".") {
		var floatVal float64
		if _, err := fmt.Sscanf(value, "%f", &floatVal); err == nil {
			return floatVal
		}
	}

	// Try decimal int
	if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
		return intVal
	}

	// Default to string
	return value
}

// LoadYAML parses YAML data using the native libyaml Parser.
// This function is exported so it can be used by other packages for data-driven testing.
// It returns a generic interface{} which is typically:
//   - map[string]interface{} for YAML mappings
//   - []interface{} for YAML sequences
//   - scalar values, resolved according to the following rules:
//   - Booleans: "true" and "false" are returned as bool (true/false).
//   - Nulls: "null" is returned as nil.
//   - Floats: values containing "." are parsed as float64.
//   - Decimal integers: values matching integer format are parsed as int.
//   - All other values are returned as string.
//
// This scalar resolution behavior matches the implementation in coerceScalar.
func LoadYAML(data []byte) (interface{}, error) {
	parser := NewParser()
	parser.SetInputString(data)
	defer parser.Delete()

	type stackEntry struct {
		container interface{} // map[string]interface{} or []interface{}
		key       string      // for maps: current key waiting for value
	}

	var stack []stackEntry
	var root interface{}

	for {
		var event Event
		if !parser.Parse(&event) {
			if parser.ErrorType != NO_ERROR {
				return nil, fmt.Errorf("parse error: %s at line %d, column %d",
					parser.Problem, parser.ProblemMark.Line, parser.ProblemMark.Column)
			}
			break
		}

		switch event.Type {
		case STREAM_END_EVENT:
			// End of stream, we're done
			return root, nil

		case STREAM_START_EVENT, DOCUMENT_START_EVENT:
			// Structural markers, no action needed

		case MAPPING_START_EVENT:
			newMap := make(map[string]interface{})
			stack = append(stack, stackEntry{container: newMap})

		case MAPPING_END_EVENT:
			if len(stack) > 0 {
				popped := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				// Add completed map to parent or set as root
				if len(stack) == 0 {
					root = popped.container
				} else {
					parent := &stack[len(stack)-1]
					if m, ok := parent.container.(map[string]interface{}); ok {
						m[parent.key] = popped.container
						parent.key = "" // Reset key after use
					} else if s, ok := parent.container.([]interface{}); ok {
						parent.container = append(s, popped.container)
					}
				}
			}

		case SEQUENCE_START_EVENT:
			newSlice := make([]interface{}, 0)
			stack = append(stack, stackEntry{container: newSlice})

		case SEQUENCE_END_EVENT:
			if len(stack) > 0 {
				popped := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				// Add completed slice to parent or set as root
				if len(stack) == 0 {
					root = popped.container
				} else {
					parent := &stack[len(stack)-1]
					if m, ok := parent.container.(map[string]interface{}); ok {
						m[parent.key] = popped.container
						parent.key = "" // Reset key after use
					} else if s, ok := parent.container.([]interface{}); ok {
						parent.container = append(s, popped.container)
					}
				}
			}

		case SCALAR_EVENT:
			value := string(event.Value)
			// Only coerce plain (unquoted) scalars
			isQuoted := ScalarStyle(event.Style) != PLAIN_SCALAR_STYLE

			if len(stack) == 0 {
				// Scalar at root level
				if isQuoted {
					root = value
				} else {
					root = coerceScalar(value)
				}
			} else {
				parent := &stack[len(stack)-1]
				if m, ok := parent.container.(map[string]interface{}); ok {
					if parent.key == "" {
						// This scalar is a key - keep as string, don't coerce
						parent.key = value
					} else {
						// This scalar is a value
						if isQuoted {
							m[parent.key] = value
						} else {
							m[parent.key] = coerceScalar(value)
						}
						parent.key = ""
					}
				} else if s, ok := parent.container.([]interface{}); ok {
					// Add to sequence
					if isQuoted {
						parent.container = append(s, value)
					} else {
						parent.container = append(s, coerceScalar(value))
					}
				}
			}

		case DOCUMENT_END_EVENT:
			// Document end marker, continue processing

		case ALIAS_EVENT, TAIL_COMMENT_EVENT:
			// For now, skip aliases and comments (not used in test data)
		}
	}

	return root, nil
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
//	err := libyaml.UnmarshalStruct(&tc, map[string]interface{}{
//	    "name": "test1",
//	    "data": "hello",
//	})
func UnmarshalStruct(target interface{}, data map[string]interface{}) error {
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
func setField(field reflect.Value, value interface{}) error {
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
		if converter, ok := addr.Interface().(interface{ FromValue(interface{}) error }); ok {
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
		if !valueRefl.CanUint() {
			if valueRefl.Int() < 0 {
				return fmt.Errorf("cannot convert negative %v to %v", value, fieldType)
			}
			return fmt.Errorf("cannot convert %v to %v", value, fieldType)
		}
		u64 := valueRefl.Uint()
		if field.OverflowUint(u64) {
			return fmt.Errorf("value %v overflows %v", value, fieldType)
		}
		field.SetUint(u64)
		return nil
	case reflect.Bool:
		field.SetBool(valueRefl.Bool())
	case reflect.Float32, reflect.Float64:
		if !field.CanFloat() {
			return fmt.Errorf("field cannot store float values")
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
		if m, ok := value.(map[string]interface{}); ok {
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
			m, ok := value.(map[string]interface{})
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

// normalizeTypeAsKey converts maps with type as key to standard format.
// Example: {"SCALAR_TOKEN": {"value": "x"}} -> {"type": "SCALAR_TOKEN", "value": "x"}
func normalizeTypeAsKey(itemMap map[string]interface{}) map[string]interface{} {
	// Check if map has exactly one key and no "type" field
	if len(itemMap) == 1 {
		_, hasType := itemMap["type"]
		if !hasType {
			// Get the single key
			for key, value := range itemMap {
				// Check if key looks like a type constant (all uppercase with underscores)
				if isTypeConstant(key) {
					// Check if value is a map
					if subMap, ok := value.(map[string]interface{}); ok {
						// Create new map with "type" field for test type
						newMap := map[string]interface{}{"type": key}
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

// isTypeConstant checks if a string looks like a type constant
// Accepts: UPPERCASE_WITH_UNDERSCORES or lowercase-with-hyphens
func isTypeConstant(s string) bool {
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
func setSliceField(field reflect.Value, value interface{}) error {
	sliceVal, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("expected []interface{} for slice, got %T", value)
	}

	elemType := field.Type().Elem()
	newSlice := reflect.MakeSlice(field.Type(), len(sliceVal), len(sliceVal))

	for i, item := range sliceVal {
		elem := newSlice.Index(i)

		// Handle struct elements
		if elemType.Kind() == reflect.Struct {
			var m map[string]interface{}

			// Check if item is a scalar string (simplified format)
			if strVal, ok := item.(string); ok {
				// Convert scalar to map with type field
				m = map[string]interface{}{"type": strVal}
			} else {
				m, ok = item.(map[string]interface{})
				if !ok {
					return fmt.Errorf("slice element %d: expected map for struct, got %T", i, item)
				}
				// Normalize type-as-key format
				m = normalizeTypeAsKey(m)
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
func setMapField(field reflect.Value, value interface{}) error {
	mapVal, ok := value.(map[string]interface{})
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
			m, ok := v.(map[string]interface{})
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
