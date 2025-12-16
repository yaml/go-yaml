// SPDX-License-Identifier: Apache-2.0

package datatest

import "fmt"

// ConstantRegistry maps constant names (strings) to their integer values.
// This allows test data to use symbolic constant names instead of magic numbers.
type ConstantRegistry struct {
	constants map[string]int
}

// NewConstantRegistry creates a new constant registry.
func NewConstantRegistry() *ConstantRegistry {
	return &ConstantRegistry{
		constants: make(map[string]int),
	}
}

// Register registers a constant name and its integer value.
//
// Example:
//
//	registry.Register("STREAM_START_EVENT", 1)
//	registry.Register("PLAIN_SCALAR_STYLE", 2)
func (r *ConstantRegistry) Register(name string, value int) {
	r.constants[name] = value
}

// Resolve looks up a constant name and returns its value.
// Returns (value, true) if found, (0, false) if not found.
func (r *ConstantRegistry) Resolve(name string) (int, bool) {
	val, ok := r.constants[name]
	return val, ok
}

// MustResolve looks up a constant name and returns its value.
// Panics if the constant is not registered.
func (r *ConstantRegistry) MustResolve(name string) int {
	val, ok := r.Resolve(name)
	if !ok {
		panic(fmt.Sprintf("constant %q not registered", name))
	}
	return val
}

// Has checks if a constant is registered.
func (r *ConstantRegistry) Has(name string) bool {
	_, ok := r.constants[name]
	return ok
}

// MergeFrom merges constants from another registry into this one.
// If there are conflicts, values from the other registry take precedence.
func (r *ConstantRegistry) MergeFrom(other *ConstantRegistry) {
	for name, value := range other.constants {
		r.constants[name] = value
	}
}

// ListConstants returns a list of all registered constant names.
func (r *ConstantRegistry) ListConstants() []string {
	names := make([]string, 0, len(r.constants))
	for name := range r.constants {
		names = append(names, name)
	}
	return names
}

// ResolveIntOrString attempts to parse a value as either:
// 1. An integer constant name (returns the resolved value)
// 2. A direct integer value
// Returns an error if neither works.
func (r *ConstantRegistry) ResolveIntOrString(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case string:
		if resolved, ok := r.Resolve(v); ok {
			return resolved, nil
		}
		return 0, fmt.Errorf("constant %q not found", v)
	default:
		return 0, fmt.Errorf("expected int or string, got %T", value)
	}
}
