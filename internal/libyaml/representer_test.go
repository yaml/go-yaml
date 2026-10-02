// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for the Representer stage

package libyaml

import (
	"fmt"
	"reflect"
	"testing"
)

// checkNode recursively validates a node against expected structure
func checkNode(t *testing.T, node *Node, wantMap map[string]any, path string) {
	t.Helper()

	// Check kind
	if kindStr, ok := wantMap["kind"].(string); ok {
		var expectedKind Kind
		switch kindStr {
		case "Scalar":
			expectedKind = ScalarNode
		case "Sequence":
			expectedKind = SequenceNode
		case "Mapping":
			expectedKind = MappingNode
		}
		if node.Kind != expectedKind {
			t.Fatalf("%s: got kind %v; want %v", path, node.Kind, expectedKind)
		}
	}

	// Check tag
	if wantTag, ok := wantMap["tag"].(string); ok {
		if node.Tag != wantTag {
			t.Fatalf("%s: got tag %q; want %q", path, node.Tag, wantTag)
		}
	}

	// Check value (for scalars)
	if wantValue, ok := wantMap["value"].(string); ok {
		if node.Value != wantValue {
			t.Fatalf("%s: got value %q; want %q", path, node.Value, wantValue)
		}
	}

	// Check content (for collections)
	if wantContent, ok := wantMap["content"].([]any); ok {
		if len(node.Content) != len(wantContent) {
			t.Fatalf("%s: got %d children; want %d", path, len(node.Content), len(wantContent))
		}
		for i, wantChild := range wantContent {
			if wantChildMap, ok := wantChild.(map[string]any); ok {
				childPath := fmt.Sprintf("%s[%d]", path, i)
				checkNode(t, node.Content[i], wantChildMap, childPath)
			}
		}
	}
}

func TestRepresenter(t *testing.T) {
	RunTestCases(t, "representer.yaml", map[string]TestHandler{
		"represent-scalar": func(t *testing.T, tc TestCase) {
			t.Helper()

			r := NewRepresenter(DefaultOptions)
			doc := r.Represent("", reflect.ValueOf(tc.From))

			if doc == nil || doc.Kind != DocumentNode {
				t.Fatal("expected DocumentNode")
			}
			if len(doc.Content) == 0 {
				t.Fatal("expected content in document")
			}
			node := doc.Content[0]

			// Check node against want spec
			wantMap := tc.Want.(map[string]any)
			checkNode(t, node, wantMap, "root")
		},

		"represent-collection": func(t *testing.T, tc TestCase) {
			t.Helper()

			r := NewRepresenter(DefaultOptions)
			doc := r.Represent("", reflect.ValueOf(tc.From))

			if doc == nil || doc.Kind != DocumentNode {
				t.Fatal("expected DocumentNode")
			}
			if len(doc.Content) == 0 {
				t.Fatal("expected content in document")
			}
			node := doc.Content[0]

			// Check node against want spec
			wantMap := tc.Want.(map[string]any)
			checkNode(t, node, wantMap, "root")
		},
	})
}
