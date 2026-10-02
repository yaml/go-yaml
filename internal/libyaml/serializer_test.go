// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for the Serializer stage

package libyaml

import (
	"bytes"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

// buildNodeFromSpec recursively builds a Node from a NodeSpec
func buildNodeFromSpec(spec map[string]any) *Node {
	node := &Node{}

	// Set kind
	if kindStr, ok := spec["kind"].(string); ok {
		switch kindStr {
		case "Document":
			node.Kind = DocumentNode
		case "Scalar":
			node.Kind = ScalarNode
		case "Sequence":
			node.Kind = SequenceNode
		case "Mapping":
			node.Kind = MappingNode
		}
	}

	// Set value
	if value, ok := spec["value"].(string); ok {
		node.Value = value
	}

	// Set style
	if styleStr, ok := spec["style"].(string); ok {
		switch styleStr {
		case "Single":
			node.Style = SingleQuotedStyle
		case "Double":
			node.Style = DoubleQuotedStyle
		case "Flow":
			node.Style = FlowStyle
		}
	}

	// Set content (recursive)
	if contentData, ok := spec["content"].([]any); ok {
		for _, item := range contentData {
			if itemMap, ok := item.(map[string]any); ok {
				child := buildNodeFromSpec(itemMap)
				node.Content = append(node.Content, child)
			}
		}
	}

	return node
}

func TestSerializer(t *testing.T) {
	RunTestCases(t, "serializer.yaml", map[string]TestHandler{
		"serialize-scalar": func(t *testing.T, tc TestCase) {
			t.Helper()

			// Build node from nested spec
			nodeData, ok := tc.Node.Content.([]any)
			if !ok || len(nodeData) == 0 {
				t.Fatal("expected content in node spec")
			}

			// Create document with scalar
			doc := &Node{Kind: DocumentNode}
			for _, item := range nodeData {
				if itemMap, ok := item.(map[string]any); ok {
					doc.Content = append(doc.Content, buildNodeFromSpec(itemMap))
				}
			}

			var buf bytes.Buffer
			s := NewSerializer(&buf, DefaultOptions)
			s.Serialize(doc)
			s.Finish()

			// Check output
			assert.Equal(t, tc.Want.(string), buf.String())
		},

		"serialize-collection": func(t *testing.T, tc TestCase) {
			t.Helper()

			// Build node from nested spec
			nodeData, ok := tc.Node.Content.([]any)
			if !ok || len(nodeData) == 0 {
				t.Fatal("expected content in node spec")
			}

			// Create document
			doc := &Node{Kind: DocumentNode}
			for _, item := range nodeData {
				if itemMap, ok := item.(map[string]any); ok {
					doc.Content = append(doc.Content, buildNodeFromSpec(itemMap))
				}
			}

			// Serialize with appropriate indent
			opts := DefaultOptions
			if tc.Indent > 0 {
				opts.Indent = tc.Indent
			}

			var buf bytes.Buffer
			s := NewSerializer(&buf, opts)
			s.Serialize(doc)
			s.Finish()

			// Check output
			assert.Equal(t, tc.Want.(string), buf.String())
		},

		"serialize-style": func(t *testing.T, tc TestCase) {
			t.Helper()

			// Build node from nested spec
			nodeData, ok := tc.Node.Content.([]any)
			if !ok || len(nodeData) == 0 {
				t.Fatal("expected content in node spec")
			}

			// Create document
			doc := &Node{Kind: DocumentNode}
			for _, item := range nodeData {
				if itemMap, ok := item.(map[string]any); ok {
					doc.Content = append(doc.Content, buildNodeFromSpec(itemMap))
				}
			}

			var buf bytes.Buffer
			s := NewSerializer(&buf, DefaultOptions)
			s.Serialize(doc)
			s.Finish()

			// Check output
			assert.Equal(t, tc.Want.(string), buf.String())
		},
	})
}
