// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/limits"
)

// generateAliases builds YAML with n aliases referencing a large anchor.
func generateAliases(n int) []byte {
	var sb strings.Builder
	sb.WriteString("anchor: &anchor [1, 2, 3]\nrefs:\n")
	for i := 0; i < n; i++ {
		sb.WriteString("- *anchor\n")
	}
	return []byte(sb.String())
}

// generateDeepNesting builds deeply nested flow YAML.
func generateDeepNesting(depth int) []byte {
	var sb strings.Builder
	for i := 0; i < depth; i++ {
		sb.WriteString("[")
	}
	sb.WriteString("x")
	for i := 0; i < depth; i++ {
		sb.WriteString("]")
	}
	return []byte(sb.String())
}

func TestWithPlugin_Limits_AliasNone(t *testing.T) {
	// Without AliasNone, 11000 aliases would trigger the default limit.
	data := generateAliases(11000)
	var result any
	err := yaml.Load(data, &result, yaml.WithPlugin(limits.New(limits.AliasNone())))
	if err != nil {
		t.Fatalf("Expected success with AliasNone, got: %v", err)
	}
}

func TestWithPlugin_Limits_AliasValue(t *testing.T) {
	data := generateAliases(200)
	var result any

	// Should succeed with high threshold
	err := yaml.Load(data, &result, yaml.WithPlugin(limits.New(limits.AliasValue(10000))))
	if err != nil {
		t.Fatalf("Expected success with high AliasValue, got: %v", err)
	}

	// Should fail with low threshold
	err = yaml.Load(data, &result, yaml.WithPlugin(limits.New(limits.AliasValue(5))))
	if err == nil {
		t.Fatal("Expected error with low AliasValue, got nil")
	}
}

func TestWithPlugin_Limits_AliasFunc(t *testing.T) {
	called := false
	fn := func(aliasCount, constructCount int) error {
		called = true
		return nil
	}
	data := generateAliases(200)
	var result any
	err := yaml.Load(data, &result, yaml.WithPlugin(limits.New(limits.AliasFunc(fn))))
	if err != nil {
		t.Fatalf("Expected success with custom AliasFunc, got: %v", err)
	}
	if !called {
		t.Error("Expected custom AliasFunc to be called")
	}
}

func TestWithPlugin_Limits_DepthValue(t *testing.T) {
	data := generateDeepNesting(20)
	var result any

	// Should succeed with depth limit > 20
	err := yaml.Load(data, &result, yaml.WithPlugin(limits.New(limits.DepthValue(50))))
	if err != nil {
		t.Fatalf("Expected success with DepthValue(50), got: %v", err)
	}

	// Should fail with depth limit < 20
	err = yaml.Load(data, &result, yaml.WithPlugin(limits.New(limits.DepthValue(5))))
	if err == nil {
		t.Fatal("Expected error with DepthValue(5) and depth 20, got nil")
	}
}

func TestWithPlugin_Limits_DepthNone(t *testing.T) {
	// 10001 levels would exceed the default limit of 10000
	data := generateDeepNesting(10001)
	var result any
	err := yaml.Load(data, &result, yaml.WithPlugin(limits.New(limits.DepthNone())))
	if err != nil {
		t.Fatalf("Expected success with DepthNone, got: %v", err)
	}
}

func TestWithPlugin_Limits_DepthFunc(t *testing.T) {
	called := false
	fn := func(depth int, ctx *yaml.DepthContext) error {
		called = true
		return nil
	}
	data := generateDeepNesting(5)
	var result any
	err := yaml.Load(data, &result, yaml.WithPlugin(limits.New(limits.DepthFunc(fn))))
	if err != nil {
		t.Fatalf("Expected success with custom DepthFunc, got: %v", err)
	}
	if !called {
		t.Error("Expected custom DepthFunc to be called")
	}
}

func TestWithPlugin_UnsupportedType(t *testing.T) {
	data := []byte(`key: value`)
	var result map[string]any
	// Pass an unsupported type (integer) as a plugin
	err := yaml.Load(data, &result, yaml.WithPlugin(42))
	if err == nil {
		t.Fatal("Expected error for unsupported plugin type, got nil")
	}
	if err.Error() != "yaml: unsupported plugin type" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestWithoutPlugin_Limits(t *testing.T) {
	// WithoutPlugin("limits") should reset to defaults (depth limit applies)
	data := generateDeepNesting(10001)
	var result any
	err := yaml.Load(data, &result, yaml.WithoutPlugin("limits"))
	if err == nil {
		t.Fatal("Expected error after WithoutPlugin reset to defaults, got nil")
	}
}

func TestDefaultBehavior_HasLimits(t *testing.T) {
	// Bare NewLoader should have default depth limits
	data := generateDeepNesting(10001)
	loader, err := yaml.NewLoader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}
	var result any
	err = loader.Load(&result)
	if err == nil {
		t.Fatal("Expected error from default depth limits, got nil")
	}
}
