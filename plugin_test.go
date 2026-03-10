// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/comment/v3"
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

// --- Comment plugin tests ---

var commentTestData = []byte(`
# Head comment
key: value # Line comment
# Foot comment
`)

func TestWithPlugin_Comment(t *testing.T) {
	loader, err := yaml.NewLoader(
		bytes.NewReader(commentTestData),
		yaml.WithPlugin(v3.New()),
	)
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		t.Fatal("Expected non-empty DocumentNode")
	}
	mapping := node.Content[0]
	if mapping.Kind != yaml.MappingNode || len(mapping.Content) < 2 {
		t.Fatal("Expected MappingNode with at least one pair")
	}
	key := mapping.Content[0]
	if key.HeadComment == "" {
		t.Error("Expected plugin to attach head comment, got none")
	}
}

func TestWithV3LegacyComments(t *testing.T) {
	loader, err := yaml.NewLoader(
		bytes.NewReader(commentTestData),
		yaml.WithV3LegacyComments(),
	)
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		t.Fatal("Expected non-empty DocumentNode")
	}
	key := node.Content[0].Content[0]
	if key.HeadComment == "" {
		t.Error("Expected head comment on key, got none")
	}
}

func TestWithPlugin_BothPluginTypes(t *testing.T) {
	// A single WithPlugin call with both limits and comment plugins
	data := []byte("# comment\nkey: value\n")
	loader, err := yaml.NewLoader(
		bytes.NewReader(data),
		yaml.WithPlugin(limits.New(limits.DepthValue(50))),
		yaml.WithPlugin(v3.New()),
	)
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	key := node.Content[0].Content[0]
	if key.HeadComment == "" {
		t.Error("Expected head comment with both plugins registered")
	}
}

func TestWithoutPlugin_Comment(t *testing.T) {
	loader, err := yaml.NewLoader(
		bytes.NewReader(commentTestData),
		yaml.WithV3LegacyComments(),
		yaml.WithoutPlugin("comment"),
	)
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	key := node.Content[0].Content[0]
	if key.HeadComment != "" {
		t.Error("Expected no comments after WithoutPlugin, got comments")
	}
}

func TestDefaultBehavior_NoComments(t *testing.T) {
	// Default behavior (no version preset) should skip comments
	loader, err := yaml.NewLoader(bytes.NewReader(commentTestData))
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	key := node.Content[0].Content[0]
	if key.HeadComment != "" {
		t.Error("Expected no comments by default, got comments")
	}
}

func TestUnmarshal_PreservesV3Behavior(t *testing.T) {
	var node yaml.Node
	err := yaml.Unmarshal(commentTestData, &node)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Unmarshal uses WithV3Defaults which includes WithV3LegacyComments.
	// UnmarshalYAML unwraps DocumentNode, so we get MappingNode directly.
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		// If we got a DocumentNode, look inside
		mapping := node.Content[0]
		if mapping.Kind != yaml.MappingNode || len(mapping.Content) < 2 {
			t.Fatal("Expected MappingNode inside DocumentNode")
		}
		if mapping.Content[0].HeadComment == "" {
			t.Error("Expected Unmarshal to preserve V3 comment behavior")
		}
	} else if node.Kind == yaml.MappingNode {
		if len(node.Content) < 2 {
			t.Fatal("Expected MappingNode with content")
		}
		if node.Content[0].HeadComment == "" {
			t.Error("Expected Unmarshal to preserve V3 comment behavior")
		}
	} else {
		t.Fatalf("Expected DocumentNode or MappingNode, got %v", node.Kind)
	}
}
