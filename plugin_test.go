//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

package yaml_test

import (
	"bytes"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/comment/v3legacy"
)

func TestWithPlugin_Comment(t *testing.T) {
	data := []byte(`
# Head comment
key: value # Line comment
# Foot comment
`)

	// Test with comment plugin - use Node to verify comments are attached
	loader, err := yaml.NewLoader(bytes.NewReader(data), yaml.WithPlugin(v3legacy.New()))
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	err = loader.Load(&node)
	if err != nil {
		t.Fatalf("Load with plugin failed: %v", err)
	}

	// Verify comments are actually attached
	if node.Kind != yaml.DocumentNode {
		t.Errorf("Expected DocumentNode, got %v", node.Kind)
	}

	if len(node.Content) > 0 {
		content := node.Content[0]
		if content.Kind == yaml.MappingNode && len(content.Content) > 0 {
			// First key should have head comment
			key := content.Content[0]
			if key.HeadComment == "" {
				t.Error("Expected plugin to attach head comment, got none")
			}
		}
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

func TestWithV3LegacyComments(t *testing.T) {
	data := []byte(`
# Head comment
key: value # Line comment
`)

	// Test with V3LegacyComments option
	loader, err := yaml.NewLoader(bytes.NewReader(data), yaml.WithV3LegacyComments())
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	err = loader.Load(&node)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should have document node
	if node.Kind != yaml.DocumentNode {
		t.Errorf("Expected DocumentNode, got %v", node.Kind)
	}

	// Check that comments were attached
	if len(node.Content) > 0 {
		content := node.Content[0]
		if content.Kind == yaml.MappingNode && len(content.Content) > 0 {
			// First key should have head comment
			key := content.Content[0]
			if key.HeadComment == "" {
				t.Error("Expected head comment on key, got none")
			}
		}
	}
}

func TestWithoutPlugin_Comment(t *testing.T) {
	data := []byte(`
# Head comment
key: value # Line comment
`)

	// Test without any comment handling
	loader, err := yaml.NewLoader(bytes.NewReader(data), yaml.WithoutPlugin("comment"))
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	err = loader.Load(&node)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Comments should be skipped
	if len(node.Content) > 0 {
		content := node.Content[0]
		if content.Kind == yaml.MappingNode && len(content.Content) > 0 {
			key := content.Content[0]
			if key.HeadComment != "" {
				t.Error("Expected no comments with WithoutPlugin, got comments")
			}
		}
	}
}

func TestDefaultBehavior_NoComments(t *testing.T) {
	data := []byte(`
# Head comment
key: value # Line comment
`)

	// Default behavior should skip comments
	loader, err := yaml.NewLoader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	err = loader.Load(&node)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Comments should be skipped by default
	if len(node.Content) > 0 {
		content := node.Content[0]
		if content.Kind == yaml.MappingNode && len(content.Content) > 0 {
			key := content.Content[0]
			if key.HeadComment != "" {
				t.Error("Expected no comments by default, got comments")
			}
		}
	}
}

func TestUnmarshal_PreservesV3Behavior(t *testing.T) {
	data := []byte(`
# Head comment
key: value # Line comment
`)

	var node yaml.Node
	err := yaml.Unmarshal(data, &node)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Unmarshal should preserve V3 behavior (comments enabled)
	if node.Kind != yaml.DocumentNode {
		t.Errorf("Expected DocumentNode, got %v", node.Kind)
	}

	// Check that comments were attached
	if len(node.Content) > 0 {
		content := node.Content[0]
		if content.Kind == yaml.MappingNode && len(content.Content) > 0 {
			key := content.Content[0]
			if key.HeadComment == "" {
				t.Error("Expected Unmarshal to preserve V3 comment behavior")
			}
		}
	}
}
