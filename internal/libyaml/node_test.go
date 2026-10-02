// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for node.go functions and methods.

package libyaml

import (
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
	"go.yaml.in/yaml/v4/internal/testutil/datatest"
)

func TestNode(t *testing.T) {
	handlers := map[string]TestHandler{
		"isZero":            runIsZeroTest,
		"set-string":        runSetStringTest,
		"set-string-binary": runSetStringBinaryTest,
		"short-tag":         runShortTagTest,
		"long-tag":          runLongTagTest,
		"node-is-zero":      runNodeIsZeroTest,
		"should-literal":    runShouldLiteralTest,
	}

	RunTestCases(t, "node.yaml", handlers)
}

// runIsZeroTest tests the isZero function
func runIsZeroTest(t *testing.T, tc TestCase) {
	t.Helper()

	var v reflect.Value

	// Handle special modifiers in 'also' field
	switch tc.Also {
	case "slice":
		// Nil slice case
		if tc.From == nil {
			v = reflect.ValueOf(([]int)(nil))
		} else {
			// Convert from to slice
			if slice, ok := tc.From.([]any); ok {
				v = reflect.ValueOf(slice)
			} else {
				t.Fatalf("expected slice, got %T", tc.From)
			}
		}
	case "map":
		// Nil map case
		if tc.From == nil {
			v = reflect.ValueOf((map[string]any)(nil))
		} else {
			// Convert from to map
			if m, ok := tc.From.(map[string]any); ok {
				v = reflect.ValueOf(m)
			} else {
				t.Fatalf("expected map, got %T", tc.From)
			}
		}
	default:
		// Regular value
		v = reflect.ValueOf(tc.From)
	}

	got := isZero(v)
	want := datatest.WantBool(t, tc.Want, false)

	assert.Equalf(t, want, got, "isZero() = %v, want %v", got, want)
}

// runSetStringTest tests the SetString method
func runSetStringTest(t *testing.T, tc TestCase) {
	t.Helper()

	str, ok := tc.From.(string)
	if !ok {
		t.Fatalf("from should be string, got %T", tc.From)
	}

	node := &Node{}
	node.SetString(str)

	wantMap, ok := tc.Want.(map[string]any)
	if !ok {
		t.Fatalf("want should be a map, got %T", tc.Want)
	}

	// Check Kind
	if wantKind, ok := wantMap["kind"].(string); ok {
		gotKind := kindToString(node.Kind)
		assert.Equalf(t, wantKind, gotKind, "Kind = %v, want %v", gotKind, wantKind)
	}

	// Check Tag
	if wantTag, ok := wantMap["tag"].(string); ok {
		assert.Equalf(t, wantTag, node.Tag, "Tag = %v, want %v", node.Tag, wantTag)
	}

	// Check Value
	if wantValue, ok := wantMap["value"].(string); ok {
		assert.Equalf(t, wantValue, node.Value, "Value = %v, want %v", node.Value, wantValue)
	}

	// Check Style
	if wantStyle, ok := wantMap["style"].(string); ok {
		gotStyle := styleToString(node.Style)
		assert.Equalf(t, wantStyle, gotStyle, "Style = %v, want %v", gotStyle, wantStyle)
	}
}

// runSetStringBinaryTest tests SetString with invalid UTF-8
func runSetStringBinaryTest(t *testing.T, tc TestCase) {
	t.Helper()

	// Get binary input from hex
	input := HexToBytes(t, tc.InputHex)
	str := string(input)

	node := &Node{}
	node.SetString(str)

	wantMap, ok := tc.Want.(map[string]any)
	if !ok {
		t.Fatalf("want should be a map, got %T", tc.Want)
	}

	// Check Kind
	if wantKind, ok := wantMap["kind"].(string); ok {
		gotKind := kindToString(node.Kind)
		assert.Equalf(t, wantKind, gotKind, "Kind = %v, want %v", gotKind, wantKind)
	}

	// Check Tag
	if wantTag, ok := wantMap["tag"].(string); ok {
		assert.Equalf(t, wantTag, node.Tag, "Tag = %v, want %v", node.Tag, wantTag)
	}

	// For binary data, we just verify it's base64 encoded (not checking exact value)
	if node.Tag == binaryTag {
		assert.Truef(t, len(node.Value) > 0, "binary value should not be empty")
	}

	// Check Style
	if wantStyle, ok := wantMap["style"].(string); ok {
		gotStyle := styleToString(node.Style)
		assert.Equalf(t, wantStyle, gotStyle, "Style = %v, want %v", gotStyle, wantStyle)
	}
}

// runShortTagTest tests the shortTag function
func runShortTagTest(t *testing.T, tc TestCase) {
	t.Helper()

	node := nodeFromSpec(t, tc.Node)
	got := node.ShortTag()
	want, ok := tc.Want.(string)
	if !ok {
		t.Fatalf("want should be string, got %T", tc.Want)
	}

	assert.Equalf(t, want, got, "ShortTag() = %v, want %v", got, want)
}

// runLongTagTest tests the longTag function
func runLongTagTest(t *testing.T, tc TestCase) {
	t.Helper()

	node := nodeFromSpec(t, tc.Node)
	got := node.LongTag()
	want, ok := tc.Want.(string)
	if !ok {
		t.Fatalf("want should be string, got %T", tc.Want)
	}

	assert.Equalf(t, want, got, "LongTag() = %v, want %v", got, want)
}

// runNodeIsZeroTest tests the Node.IsZero method
func runNodeIsZeroTest(t *testing.T, tc TestCase) {
	t.Helper()

	node := nodeFromSpec(t, tc.Node)
	got := node.IsZero()
	want := datatest.WantBool(t, tc.Want, false)

	assert.Equalf(t, want, got, "Node.IsZero() = %v, want %v", got, want)
}

// runShouldLiteralTest tests the shouldUseLiteralStyle helper
func runShouldLiteralTest(t *testing.T, tc TestCase) {
	t.Helper()

	str, ok := tc.From.(string)
	if !ok {
		t.Fatalf("from should be string, got %T", tc.From)
	}

	got := shouldUseLiteralStyle(str)
	want := datatest.WantBool(t, tc.Want, false)

	assert.Equalf(t, want, got, "shouldUseLiteralStyle() = %v, want %v", got, want)
}

// nodeFromSpec creates a Node from a NodeSpec
func nodeFromSpec(t *testing.T, spec NodeSpec) *Node {
	t.Helper()

	node := &Node{}

	// Set Kind
	if spec.Kind != "" {
		node.Kind = parseKind(t, spec.Kind)
	}

	// Set Tag
	node.Tag = spec.Tag

	// Set Value
	node.Value = spec.Value

	// Set Style
	if spec.Style != "" {
		node.Style = parseStyle(t, spec.Style)
	}

	return node
}

// parseKind converts a string to Kind
func parseKind(t *testing.T, s string) Kind {
	t.Helper()
	switch s {
	case "Document":
		return DocumentNode
	case "Sequence":
		return SequenceNode
	case "Mapping":
		return MappingNode
	case "Scalar":
		return ScalarNode
	case "Alias":
		return AliasNode
	case "Stream":
		return StreamNode
	case "":
		return 0
	default:
		t.Fatalf("unknown kind: %s", s)
		return 0
	}
}

// parseStyle converts a string to Style
func parseStyle(t *testing.T, s string) Style {
	t.Helper()
	switch s {
	case "Tagged":
		return TaggedStyle
	case "DoubleQuoted":
		return DoubleQuotedStyle
	case "SingleQuoted":
		return SingleQuotedStyle
	case "Literal":
		return LiteralStyle
	case "Folded":
		return FoldedStyle
	case "Flow":
		return FlowStyle
	case "":
		return 0
	default:
		t.Fatalf("unknown style: %s", s)
		return 0
	}
}

// kindToString converts Kind to string for comparison
func kindToString(k Kind) string {
	switch k {
	case DocumentNode:
		return "DocumentNode"
	case SequenceNode:
		return "SequenceNode"
	case MappingNode:
		return "MappingNode"
	case ScalarNode:
		return "ScalarNode"
	case AliasNode:
		return "AliasNode"
	case StreamNode:
		return "StreamNode"
	case 0:
		return ""
	default:
		return "unknown"
	}
}

// styleToString converts Style to string for comparison
func styleToString(s Style) string {
	switch s {
	case TaggedStyle:
		return "tagged"
	case DoubleQuotedStyle:
		return "double-quoted"
	case SingleQuotedStyle:
		return "single-quoted"
	case LiteralStyle:
		return "literal"
	case FoldedStyle:
		return "folded"
	case FlowStyle:
		return "flow"
	case 0:
		return "plain"
	default:
		return "unknown"
	}
}
