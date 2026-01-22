// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for the Resolver stage

package libyaml

import (
	"testing"
)

func TestResolver(t *testing.T) {
	RunTestCases(t, "resolver.yaml", map[string]TestHandler{
		"resolve-infer": func(t *testing.T, tc TestCase) {
			t.Helper()

			// Create scalar node from test case
			node := &Node{
				Kind:  ScalarNode,
				Tag:   tc.Node.Tag,
				Value: tc.Node.Value,
			}

			// Resolve the node
			r := NewResolver(nil)
			r.Resolve(node)

			// Extract want fields
			wantMap := tc.Want.(map[string]any)
			wantTag := wantMap["tag"].(string)

			// Check tag
			if node.Tag != wantTag {
				t.Fatalf("got tag %q; want %q", node.Tag, wantTag)
			}
		},

		"resolve-preserve": func(t *testing.T, tc TestCase) {
			t.Helper()

			// Create scalar node with pre-existing tag
			node := &Node{
				Kind:  ScalarNode,
				Tag:   tc.Node.Tag,
				Value: tc.Node.Value,
			}

			// Resolve the node
			r := NewResolver(nil)
			r.Resolve(node)

			// Extract want fields
			wantMap := tc.Want.(map[string]any)
			wantTag := wantMap["tag"].(string)

			// Check tag is preserved
			if node.Tag != wantTag {
				t.Fatalf("got tag %q; want %q (tag should be preserved)", node.Tag, wantTag)
			}
		},
	})
}
