// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for the constructor stage.
// Verifies YAML node to Go value conversion and error handling.

package libyaml

import (
	"fmt"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestConstructor(t *testing.T) {
	RunTestCases(t, "constructor.yaml", map[string]TestHandler{
		"scalar-resolution": func(t *testing.T, tc TestCase) {
			t.Helper()

			result, err := LoadAny([]byte(tc.Yaml))
			assert.NoErrorf(t, err, "LoadAny() error: %v", err)

			if !reflect.DeepEqual(result, tc.Want) {
				t.Errorf("LoadAny() = %v (type: %T), want %v (type: %T)",
					result, result, tc.Want, tc.Want)
			}
		},
	})
}

type wrappingLegacyConstructor struct{}

func (wrappingLegacyConstructor) UnmarshalYAML(unmarshal func(any) error) error {
	var target struct {
		Value string `yaml:"value"`
	}
	if err := unmarshal(&target); err != nil {
		return fmt.Errorf("wrapper failed: %w", err)
	}
	return nil
}

func loadErrorChainFinite(err error, limit int) bool {
	count := 0
	var walk func(error) bool
	walk = func(e error) bool {
		for e != nil {
			count++
			if count > limit {
				return false
			}
			if le, ok := e.(*LoadErrors); ok {
				for _, child := range le.Errors {
					if !walk(child) {
						return false
					}
				}
				return true
			}
			switch x := e.(type) {
			case interface{ Unwrap() []error }:
				for _, child := range x.Unwrap() {
					if !walk(child) {
						return false
					}
				}
				return true
			case interface{ Unwrap() error }:
				e = x.Unwrap()
			default:
				return true
			}
		}
		return true
	}
	return walk(err)
}

func TestCallLegacyConstructorWrappedErrorNoCycle(t *testing.T) {
	c := NewConstructor(&Options{})
	n := &Node{Kind: ScalarNode, Tag: strTag, Value: "not-an-object", Line: 1, Column: 1}

	good := c.callLegacyConstructor(n, wrappingLegacyConstructor{})
	assert.False(t, good)
	assert.NotNil(t, &c.TypeErrors)
	if len(c.TypeErrors) == 0 {
		t.Fatal("expected callLegacyConstructor to record a type error")
	}

	for _, e := range c.TypeErrors {
		assert.Truef(t, loadErrorChainFinite(e, 1000),
			"error chain is cyclic (issue #345 regression): %v", e)
	}
}
