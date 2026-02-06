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

// TestNoPlugin_ZeroComments verifies that loading without a plugin results in
// zero comments on nodes.
func TestNoPlugin_ZeroComments(t *testing.T) {
	data := []byte(`
# Head comment
key: value # Line comment
# Foot comment
another: test
`)

	loader, err := yaml.NewLoader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	err = loader.Load(&node)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify no comments are attached
	if node.Kind != yaml.DocumentNode {
		t.Errorf("Expected DocumentNode, got %v", node.Kind)
	}

	if len(node.Content) > 0 {
		content := node.Content[0]
		if content.Kind == yaml.MappingNode {
			for _, n := range content.Content {
				if n.HeadComment != "" || n.LineComment != "" || n.FootComment != "" {
					t.Errorf("Expected zero comments without plugin, got HeadComment=%q, LineComment=%q, FootComment=%q",
						n.HeadComment, n.LineComment, n.FootComment)
				}
			}
		}
	}
}

// TestV3LegacyPlugin_MatchesCurrentBehavior verifies that the v3legacy plugin
// produces the same results as the current V3 behavior.
func TestV3LegacyPlugin_MatchesCurrentBehavior(t *testing.T) {
	data := []byte(`
# Head comment
key: value # Line comment
# Foot comment
`)

	// Load with v3legacy plugin
	loader1, err := yaml.NewLoader(bytes.NewReader(data), yaml.WithPlugin(v3legacy.New()))
	if err != nil {
		t.Fatalf("NewLoader with plugin failed: %v", err)
	}

	var node1 yaml.Node
	err = loader1.Load(&node1)
	if err != nil {
		t.Fatalf("Load with plugin failed: %v", err)
	}

	// Load with WithV3LegacyComments
	loader2, err := yaml.NewLoader(bytes.NewReader(data), yaml.WithV3LegacyComments())
	if err != nil {
		t.Fatalf("NewLoader with WithV3LegacyComments failed: %v", err)
	}

	var node2 yaml.Node
	err = loader2.Load(&node2)
	if err != nil {
		t.Fatalf("Load with WithV3LegacyComments failed: %v", err)
	}

	// Compare the results - should be identical
	if !compareNodes(&node1, &node2) {
		t.Error("Plugin and WithV3LegacyComments produced different results")
	}
}

// TestWithV3LegacyComments_SameAsPlugin verifies that WithV3LegacyComments()
// produces identical results to using the v3legacy plugin directly.
func TestWithV3LegacyComments_SameAsPlugin(t *testing.T) {
	data := []byte(`
# Document head
mapping:
  key1: value1 # line comment
  # key2 head
  key2: value2
  key3: value3
# Document foot
`)

	// Load with plugin
	loader1, err := yaml.NewLoader(bytes.NewReader(data), yaml.WithPlugin(v3legacy.New()))
	if err != nil {
		t.Fatalf("NewLoader with plugin failed: %v", err)
	}

	var node1 yaml.Node
	err = loader1.Load(&node1)
	if err != nil {
		t.Fatalf("Load with plugin failed: %v", err)
	}

	// Load with WithV3LegacyComments
	loader2, err := yaml.NewLoader(bytes.NewReader(data), yaml.WithV3LegacyComments())
	if err != nil {
		t.Fatalf("NewLoader with WithV3LegacyComments failed: %v", err)
	}

	var node2 yaml.Node
	err = loader2.Load(&node2)
	if err != nil {
		t.Fatalf("Load with WithV3LegacyComments failed: %v", err)
	}

	// Compare the results - should be identical
	if !compareNodes(&node1, &node2) {
		t.Error("Plugin and WithV3LegacyComments produced different comment results")
		t.Logf("Node1: %+v", node1)
		t.Logf("Node2: %+v", node2)
	}
}

// compareNodes recursively compares two nodes for equality, including comments.
func compareNodes(n1, n2 *yaml.Node) bool {
	if n1 == nil && n2 == nil {
		return true
	}
	if n1 == nil || n2 == nil {
		return false
	}

	if n1.Kind != n2.Kind {
		return false
	}
	if n1.Tag != n2.Tag {
		return false
	}
	if n1.Value != n2.Value {
		return false
	}
	if n1.HeadComment != n2.HeadComment {
		return false
	}
	if n1.LineComment != n2.LineComment {
		return false
	}
	if n1.FootComment != n2.FootComment {
		return false
	}

	if len(n1.Content) != len(n2.Content) {
		return false
	}

	for i := range n1.Content {
		if !compareNodes(n1.Content[i], n2.Content[i]) {
			return false
		}
	}

	return true
}
