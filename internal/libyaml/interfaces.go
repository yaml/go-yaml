// Copyright 2011-2019 Canonical Ltd
// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Interfaces for custom YAML marshaling and unmarshaling behavior.
//
// This file defines interfaces that types can implement to customize
// how they are converted to and from YAML.

package libyaml

import "reflect"

// Marshaler interface may be implemented by types to customize their
// behavior when being marshaled into a YAML document.
type Marshaler interface {
	MarshalYAML() (any, error)
}

// IsZeroer is used to check whether an object is zero to determine whether
// it should be omitted when marshaling with the ,omitempty flag. One notable
// implementation is time.Time.
type IsZeroer interface {
	IsZero() bool
}

// FromYAMLNode is a new interface that types can implement to customize
// their unmarshaling behavior. It receives a Node directly and modifies
// the receiver in place.
// This is the preferred interface for new code.
type FromYAMLNode interface {
	FromYAMLNode(*Node) error
}

// ToYAMLNode is a new interface that types can implement to customize
// their marshaling behavior. It returns a Node directly.
// This is the preferred interface for new code.
type ToYAMLNode interface {
	ToYAMLNode() (*Node, error)
}

// isZero reports whether v represents the zero value for its type.
// If v implements the IsZeroer interface, IsZero() is called.
// Otherwise, zero is determined by checking type-specific conditions.
// This is used to determine omitempty behavior when marshaling.
func isZero(v reflect.Value) bool {
	kind := v.Kind()
	if z, ok := v.Interface().(IsZeroer); ok {
		if (kind == reflect.Pointer || kind == reflect.Interface) && v.IsNil() {
			return true
		}
		return z.IsZero()
	}
	switch kind {
	case reflect.String:
		return len(v.String()) == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	case reflect.Slice:
		return v.Len() == 0
	case reflect.Map:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Struct:
		vt := v.Type()
		for i := v.NumField() - 1; i >= 0; i-- {
			if vt.Field(i).PkgPath != "" {
				continue // Private field
			}
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	}
	return false
}
