// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package datatest

import (
	"fmt"
	"testing"
)

// TestHandler is a function that runs a single test case.
// The test case is passed as a map[string]interface{}.
type TestHandler func(t *testing.T, tc map[string]any)

// TestRunner manages test execution with handlers for different test types.
type TestRunner struct {
	handlers map[string]TestHandler
}

// NewTestRunner creates a new test runner.
func NewTestRunner() *TestRunner {
	return &TestRunner{
		handlers: make(map[string]TestHandler),
	}
}

// RegisterHandler registers a handler for a specific test type.
func (r *TestRunner) RegisterHandler(testType string, handler TestHandler) {
	r.handlers[testType] = handler
}

// RunWithCases executes test cases loaded from a slice of maps.
func (r *TestRunner) RunWithCases(t *testing.T, cases []map[string]any) {
	t.Helper()

	for _, tc := range cases {
		tc := tc // capture loop variable

		// Extract test name and type
		name, _ := tc["name"].(string)
		if name == "" {
			name = "unnamed"
		}

		testType, _ := tc["type"].(string)
		if testType == "" {
			t.Fatalf("Test case %q missing 'type' field", name)
		}

		t.Run(name, func(t *testing.T) {
			handler, ok := r.handlers[testType]
			if !ok {
				t.Fatalf("Unknown test type: %s", testType)
			}
			handler(t, tc)
		})
	}
}

// RunTestCases is a convenience function that creates a runner and executes test cases.
// The loadFunc should be provided by the calling package to load test data using its preferred YAML parser.
func RunTestCases(t *testing.T, loadFunc func() ([]map[string]any, error), handlers map[string]TestHandler) {
	t.Helper()

	cases, err := loadFunc()
	if err != nil {
		t.Fatalf("Failed to load test cases: %v", err)
	}

	runner := NewTestRunner()
	for testType, handler := range handlers {
		runner.RegisterHandler(testType, handler)
	}
	runner.RunWithCases(t, cases)
}

// GetString extracts a string field from a test case map.
func GetString(tc map[string]any, key string) (string, bool) {
	val, ok := tc[key]
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt extracts an int field from a test case map.
func GetInt(tc map[string]any, key string) (int, bool) {
	val, ok := tc[key]
	if !ok {
		return 0, false
	}
	intVal, ok := val.(int)
	return intVal, ok
}

// GetBool extracts a bool field from a test case map.
func GetBool(tc map[string]any, key string) (bool, bool) {
	val, ok := tc[key]
	if !ok {
		return false, false
	}
	boolVal, ok := val.(bool)
	return boolVal, ok
}

// GetSlice extracts a slice field from a test case map.
func GetSlice(tc map[string]any, key string) ([]any, bool) {
	val, ok := tc[key]
	if !ok {
		return nil, false
	}
	slice, ok := val.([]any)
	return slice, ok
}

// GetMap extracts a map field from a test case map.
func GetMap(tc map[string]any, key string) (map[string]any, bool) {
	val, ok := tc[key]
	if !ok {
		return nil, false
	}
	m, ok := val.(map[string]any)
	return m, ok
}

// RequireString extracts a string field, failing the test if not present.
func RequireString(t *testing.T, tc map[string]any, key string) string {
	t.Helper()
	val, ok := GetString(tc, key)
	if !ok {
		t.Fatalf("Required field %q missing or not a string", key)
	}
	return val
}

// RequireInt extracts an int field, failing the test if not present.
func RequireInt(t *testing.T, tc map[string]any, key string) int {
	t.Helper()
	val, ok := GetInt(tc, key)
	if !ok {
		t.Fatalf("Required field %q missing or not an int", key)
	}
	return val
}

// RequireSlice extracts a slice field, failing the test if not present.
func RequireSlice(t *testing.T, tc map[string]any, key string) []any {
	t.Helper()
	val, ok := GetSlice(tc, key)
	if !ok {
		t.Fatalf("Required field %q missing or not a slice", key)
	}
	return val
}

// UnmarshalTestCase unmarshals a test case map into a struct.
func UnmarshalTestCase(tc map[string]any, target any) error {
	return UnmarshalStruct(target, tc)
}

// AssertEqual is a helper for comparing expected and actual values in test handlers.
func AssertEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if fmt.Sprintf("%v", expected) != fmt.Sprintf("%v", actual) {
		t.Errorf("Expected:\n%v\nGot:\n%v", expected, actual)
	}
}
