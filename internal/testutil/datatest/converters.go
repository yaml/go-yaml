// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package datatest

import "fmt"

// IntOrStr can be converted from either an int or a string constant name.
// It uses a ConstantRegistry to resolve string names to integer values.
type IntOrStr struct {
	Value    int
	Registry *ConstantRegistry // Required for string name resolution; if nil, returns error
}

// FromValue implements the custom converter interface used by UnmarshalStruct.
func (ios *IntOrStr) FromValue(v any) error {
	switch val := v.(type) {
	case int:
		ios.Value = val
		return nil
	case string:
		registry := ios.Registry
		if registry == nil {
			return fmt.Errorf("no constant registry available for resolving %q", val)
		}
		resolved, ok := registry.Resolve(val)
		if !ok {
			return fmt.Errorf("unknown constant name: %s", val)
		}
		ios.Value = resolved
		return nil
	default:
		return fmt.Errorf("IntOrStr value must be int or string, got %T", v)
	}
}

// ByteInput can be converted from either a string or a sequence of hex bytes.
type ByteInput []byte

// FromValue implements the custom converter interface used by UnmarshalStruct.
func (bi *ByteInput) FromValue(v any) error {
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
	intSlice, ok := v.([]any)
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

// Args can be converted from either a single value or an array of values.
// This is useful for method arguments that can be either scalar or array.
type Args []any

// FromValue implements the custom converter interface used by UnmarshalStruct.
func (a *Args) FromValue(v any) error {
	// Try array first
	if arrVal, ok := v.([]any); ok {
		*a = arrVal
		return nil
	}

	// Otherwise, it's a single scalar value - wrap it in a slice
	*a = []any{v}
	return nil
}

// StringSlice can be converted from either a single string or a slice of strings.
type StringSlice []string

// FromValue implements the custom converter interface used by UnmarshalStruct.
func (ss *StringSlice) FromValue(v any) error {
	// Try string first
	if strVal, ok := v.(string); ok {
		*ss = []string{strVal}
		return nil
	}

	// Try slice of interface{}
	if slice, ok := v.([]any); ok {
		strs := make([]string, len(slice))
		for i, item := range slice {
			str, ok := item.(string)
			if !ok {
				return fmt.Errorf("StringSlice element %d must be string, got %T", i, item)
			}
			strs[i] = str
		}
		*ss = strs
		return nil
	}

	return fmt.Errorf("StringSlice must be string or []string, got %T", v)
}

// IntSlice can be converted from either a single int or a slice of ints.
type IntSlice []int

// FromValue implements the custom converter interface used by UnmarshalStruct.
func (is *IntSlice) FromValue(v any) error {
	// Try int first
	if intVal, ok := v.(int); ok {
		*is = []int{intVal}
		return nil
	}

	// Try slice of interface{}
	if slice, ok := v.([]any); ok {
		ints := make([]int, len(slice))
		for i, item := range slice {
			intVal, ok := item.(int)
			if !ok {
				return fmt.Errorf("IntSlice element %d must be int, got %T", i, item)
			}
			ints[i] = intVal
		}
		*is = ints
		return nil
	}

	return fmt.Errorf("IntSlice must be int or []int, got %T", v)
}
