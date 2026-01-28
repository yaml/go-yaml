// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for the Composer stage

package libyaml

import (
	"fmt"
	"testing"
)

// checkComposedNode recursively validates a composed node against expected structure
func checkComposedNode(t *testing.T, node *Node, wantMap map[string]any, path string) {
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

	// Check style
	if wantStyle, ok := wantMap["style"].(string); ok {
		var expectedStyle Style
		switch wantStyle {
		case "Single":
			expectedStyle = SingleQuotedStyle
		case "Double":
			expectedStyle = DoubleQuotedStyle
		case "Literal":
			expectedStyle = LiteralStyle
		case "Folded":
			expectedStyle = FoldedStyle
		case "Flow":
			expectedStyle = FlowStyle
		}
		if expectedStyle != 0 && node.Style&expectedStyle == 0 {
			t.Fatalf("%s: expected style %v but got %v", path, expectedStyle, node.Style)
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
				checkComposedNode(t, node.Content[i], wantChildMap, childPath)
			}
		}
	}
}

func TestComposer(t *testing.T) {
	RunTestCases(t, "composer.yaml", map[string]TestHandler{
		"compose-scalar": func(t *testing.T, tc TestCase) {
			t.Helper()

			// Parse YAML from tc.From (YAML input string)
			yaml := tc.From.(string)
			c := NewComposer([]byte(yaml), nil)
			defer c.Destroy()

			// Get document node
			doc := c.Compose()
			if doc == nil || doc.Kind != DocumentNode {
				t.Fatal("expected DocumentNode")
			}
			if len(doc.Content) == 0 {
				t.Fatal("expected content in document")
			}
			node := doc.Content[0]

			// Check node against want spec
			wantMap := tc.Want.(map[string]any)
			checkComposedNode(t, node, wantMap, "root")
		},

		"compose-collection": func(t *testing.T, tc TestCase) {
			t.Helper()

			// Parse YAML from tc.From (YAML input string)
			yaml := tc.From.(string)
			c := NewComposer([]byte(yaml), nil)
			defer c.Destroy()

			// Get document node
			doc := c.Compose()
			if doc == nil || doc.Kind != DocumentNode {
				t.Fatal("expected DocumentNode")
			}
			if len(doc.Content) == 0 {
				t.Fatal("expected content in document")
			}
			node := doc.Content[0]

			// Check node against want spec
			wantMap := tc.Want.(map[string]any)
			checkComposedNode(t, node, wantMap, "root")
		},

		"compose-style": func(t *testing.T, tc TestCase) {
			t.Helper()

			// Parse YAML from tc.From (YAML input string)
			yaml := tc.From.(string)
			c := NewComposer([]byte(yaml), nil)
			defer c.Destroy()

			// Get document node
			doc := c.Compose()
			if doc == nil || doc.Kind != DocumentNode {
				t.Fatal("expected DocumentNode")
			}
			if len(doc.Content) == 0 {
				t.Fatal("expected content in document")
			}
			node := doc.Content[0]

			// Check node against want spec
			wantMap := tc.Want.(map[string]any)
			checkComposedNode(t, node, wantMap, "root")
		},
	})
}
