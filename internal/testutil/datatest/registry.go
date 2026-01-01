// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package datatest

import (
	"fmt"
	"reflect"
)

// TypeRegistry maps type names (strings) to Go types.
// This allows test data in YAML to reference types by name.
type TypeRegistry struct {
	types     map[string]reflect.Type
	factories map[string]func() any
}

// NewTypeRegistry creates a new type registry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types:     make(map[string]reflect.Type),
		factories: make(map[string]func() any),
	}
}

// Register registers a type by providing an exemplar value.
// The type name can be used in YAML test data to instantiate values of this type.
//
// Example:
//
//	registry.Register("string", "")
//	registry.Register("int", 0)
//	registry.Register("yaml.Node", yaml.Node{})
func (r *TypeRegistry) Register(name string, exemplar any) {
	r.types[name] = reflect.TypeOf(exemplar)
}

// RegisterFactory registers a factory function for creating instances of a type.
// This is useful for parameterized types like maps and slices.
//
// Example:
//
//	registry.RegisterFactory("map[string]any", func() interface{} {
//	    return make(map[string]interface{})
//	})
func (r *TypeRegistry) RegisterFactory(name string, factory func() any) {
	r.factories[name] = factory
	// Also register the type by calling the factory once
	if instance := factory(); instance != nil {
		r.types[name] = reflect.TypeOf(instance)
	}
}

// NewInstance creates a new zero-value instance of the registered type.
// Returns an error if the type is not registered.
func (r *TypeRegistry) NewInstance(name string) (any, error) {
	// Check if there's a factory first
	if factory, ok := r.factories[name]; ok {
		return factory(), nil
	}

	// Otherwise, create zero value from type
	typ, ok := r.types[name]
	if !ok {
		return nil, fmt.Errorf("type %q not registered", name)
	}

	// Create a new instance
	return reflect.New(typ).Elem().Interface(), nil
}

// NewPointerInstance creates a new pointer to a zero-value instance of the registered type.
// This is useful for unmarshaling into struct pointers.
func (r *TypeRegistry) NewPointerInstance(name string) (any, error) {
	// Check if there's a factory first
	if factory, ok := r.factories[name]; ok {
		instance := factory()
		// Return pointer to the instance
		ptr := reflect.New(reflect.TypeOf(instance))
		ptr.Elem().Set(reflect.ValueOf(instance))
		return ptr.Interface(), nil
	}

	// Otherwise, create pointer from type
	typ, ok := r.types[name]
	if !ok {
		return nil, fmt.Errorf("type %q not registered", name)
	}

	// Create a new pointer instance
	return reflect.New(typ).Interface(), nil
}

// GetType returns the reflect.Type for a registered type name.
func (r *TypeRegistry) GetType(name string) (reflect.Type, bool) {
	typ, ok := r.types[name]
	return typ, ok
}

// Has checks if a type is registered.
func (r *TypeRegistry) Has(name string) bool {
	_, ok := r.types[name]
	if !ok {
		_, ok = r.factories[name]
	}
	return ok
}

// ListTypes returns a list of all registered type names.
func (r *TypeRegistry) ListTypes() []string {
	types := make([]string, 0, len(r.types)+len(r.factories))
	seen := make(map[string]bool)

	for name := range r.types {
		types = append(types, name)
		seen[name] = true
	}

	for name := range r.factories {
		if !seen[name] {
			types = append(types, name)
		}
	}

	return types
}

// ValueRegistry maps string names to arbitrary constant values for test data.
// This allows YAML test files to reference constants by name (e.g., "+Inf", "NaN", "MaxInt32").
// Unlike ConstantRegistry which only handles ints, this handles any type.
type ValueRegistry struct {
	values map[string]any
}

// NewValueRegistry creates a new empty ValueRegistry.
func NewValueRegistry() *ValueRegistry {
	return &ValueRegistry{
		values: make(map[string]any),
	}
}

// Register registers a constant value with a name.
func (r *ValueRegistry) Register(name string, value any) {
	r.values[name] = value
}

// Get retrieves a constant value by name.
// Returns (value, true) if found, (nil, false) if not found.
func (r *ValueRegistry) Get(name string) (any, bool) {
	value, ok := r.values[name]
	return value, ok
}

// Resolve recursively resolves constant names in a value.
// If the value is a string that matches a registered constant name, it's replaced.
// If the value is a map or slice, it recursively resolves all elements.
func (r *ValueRegistry) Resolve(value any) any {
	// Check if it's a string constant name
	if str, ok := value.(string); ok {
		if constVal, found := r.Get(str); found {
			return constVal
		}
		return value
	}

	// Recursively resolve maps
	if m, ok := value.(map[string]any); ok {
		result := make(map[string]any, len(m))
		for k, v := range m {
			result[k] = r.Resolve(v)
		}
		return result
	}

	if m, ok := value.(map[any]any); ok {
		result := make(map[any]any, len(m))
		for k, v := range m {
			result[r.Resolve(k)] = r.Resolve(v)
		}
		return result
	}

	// Recursively resolve slices
	if s, ok := value.([]any); ok {
		result := make([]any, len(s))
		for i, v := range s {
			result[i] = r.Resolve(v)
		}
		return result
	}

	return value
}
