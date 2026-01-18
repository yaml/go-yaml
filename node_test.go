// Copyright 2011-2019 Canonical Ltd
// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for Node operations, including encoding/decoding with Node,
// node manipulation, and style handling.

package yaml_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
	"go.yaml.in/yaml/v4/internal/testutil/datatest"
)

func assertNodeEqual(t *testing.T, want *yaml.Node, got *yaml.Node) {
	t.Helper()

	if reflect.DeepEqual(got, want) {
		// fast path
		return
	}

	if got.Tag != want.Tag {
		t.Errorf("Tag mismatch: want: %q got: %q", want.Tag, got.Tag)
	}

	if got.Kind != want.Kind {
		t.Errorf("Kind mismatch: want: %q got: %q", want.Kind, got.Kind)
	}

	if got.Style != want.Style {
		t.Errorf("Style mismatch: want: %q got: %q", want.Style, got.Style)
	}

	if got.HeadComment != want.HeadComment {
		t.Errorf("HeadComment mismatch: want: %#v got: %#v", want.HeadComment, got.HeadComment)
	}

	if got.LineComment != want.LineComment {
		t.Errorf("LineComment mismatch: want: %#v got: %#v", want.LineComment, got.LineComment)
	}

	if got.FootComment != want.FootComment {
		t.Errorf("FootComment mismatch: want: %#v got: %#v", want.FootComment, got.FootComment)
	}

	if got.Value != want.Value {
		t.Errorf("Value mismatch: want: %q got: %q", want.Value, got.Value)
	}

	if got.Anchor != want.Anchor {
		t.Errorf("Anchor mismatch: want: %q got: %q", want.Anchor, got.Anchor)
	}

	if got.Line != want.Line {
		t.Errorf("Line mismatch: want: %d got: %d", want.Line, got.Line)
	}

	if got.Column != want.Column {
		t.Errorf("Column mismatch: want: %d got: %d", want.Column, got.Column)
	}

	if !reflect.DeepEqual(got.Content, want.Content) {
		// Content differs

		if len(got.Content) != len(want.Content) {
			t.Errorf("Content length mismatch:\nwant: %d\ngot: %d", len(want.Content), len(got.Content))
		}

		for i := 0; i < len(want.Content) && i < len(got.Content); i++ {
			assertNodeEqual(t, want.Content[i], got.Content[i])
		}
	}

	if t.Failed() {
		// we already reported an error, there is no need to report it again.
		return
	}

	// this error message is harder to read, and is only shown if no other errors were reported.
	t.Errorf("nodes differ:\nwant:\n%#v\ngot:\n%#v", want, got)
}

var setStringTests = []struct {
	str  string
	yaml string
	node yaml.Node
}{
	{
		"something simple",
		"something simple\n",
		yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "something simple",
			Tag:   "!!str",
		},
	}, {
		`"quoted value"`,
		"'\"quoted value\"'\n",
		yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: `"quoted value"`,
			Tag:   "!!str",
		},
	}, {
		"multi\nline",
		"|-\n  multi\n  line\n",
		yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "multi\nline",
			Tag:   "!!str",
			Style: yaml.LiteralStyle,
		},
	}, {
		"123",
		"\"123\"\n",
		yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "123",
			Tag:   "!!str",
		},
	}, {
		"multi\nline\n",
		"|\n  multi\n  line\n",
		yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "multi\nline\n",
			Tag:   "!!str",
			Style: yaml.LiteralStyle,
		},
	}, {
		"\x80\x81\x82",
		"!!binary gIGC\n",
		yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "gIGC",
			Tag:   "!!binary",
		},
	},
}

func TestSetString(t *testing.T) {
	t.Setenv("TZ", "UTC")
	for _, item := range setStringTests {
		item := item
		t.Run("", func(t *testing.T) {
			t.Logf("str: %q", item.str)

			var node yaml.Node
			node.SetString(item.str)

			assertNodeEqual(t, &item.node, &node)

			buf := bytes.Buffer{}
			enc := yaml.NewEncoder(&buf)
			enc.SetIndent(2)
			err := enc.Encode(&item.node)
			assert.NoError(t, err)
			err = enc.Close()
			assert.NoError(t, err)
			assert.Equal(t, item.yaml, buf.String())

			var doc yaml.Node
			err = yaml.Unmarshal([]byte(item.yaml), &doc)
			assert.NoError(t, err)

			var str string
			err = node.Load(&str)
			assert.NoError(t, err)
			assert.Equal(t, item.str, str)
		})
	}
}

var nodeEncodeDecodeTests = []struct {
	value any
	yaml  string
	node  yaml.Node
}{{
	"something simple",
	"something simple\n",
	yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "something simple",
		Tag:   "!!str",
	},
}, {
	`"quoted value"`,
	"'\"quoted value\"'\n",
	yaml.Node{
		Kind:  yaml.ScalarNode,
		Style: yaml.SingleQuotedStyle,
		Value: `"quoted value"`,
		Tag:   "!!str",
	},
}, {
	123,
	"123",
	yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: `123`,
		Tag:   "!!int",
	},
}, {
	[]any{1, 2},
	"[1, 2]",
	yaml.Node{
		Kind: yaml.SequenceNode,
		Tag:  "!!seq",
		Content: []*yaml.Node{{
			Kind:  yaml.ScalarNode,
			Value: "1",
			Tag:   "!!int",
		}, {
			Kind:  yaml.ScalarNode,
			Value: "2",
			Tag:   "!!int",
		}},
	},
}, {
	map[string]any{"a": "b"},
	"a: b",
	yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{{
			Kind:  yaml.ScalarNode,
			Value: "a",
			Tag:   "!!str",
		}, {
			Kind:  yaml.ScalarNode,
			Value: "b",
			Tag:   "!!str",
		}},
	},
}}

func TestNodeEncodeDecode(t *testing.T) {
	for _, item := range nodeEncodeDecodeTests {
		item := item
		t.Run("", func(t *testing.T) {
			t.Logf("Encode/Decode test value: %#v", item.value)

			var v any
			err := item.node.Load(&v)
			assert.NoError(t, err)
			assert.DeepEqual(t, item.value, v)

			var n yaml.Node
			err = n.Dump(item.value)
			assert.NoError(t, err)
			assert.DeepEqual(t, item.node, n)
		})
	}
}

func TestNodeZeroEncodeDecode(t *testing.T) {
	// Zero node value behaves as nil when encoding...
	var n yaml.Node
	data, err := yaml.Marshal(&n)
	assert.NoError(t, err)
	assert.Equal(t, "null\n", string(data))

	// ... and decoding.
	v := &struct{}{}
	err = n.Load(&v)
	assert.NoError(t, err)
	assert.IsNil(t, v)

	// ... and even when looking for its tag.
	assert.Equal(t, "!!null", n.ShortTag())

	// Kind zero is still unknown, though.
	n.Line = 1
	_, err = yaml.Marshal(&n)
	assert.ErrorMatches(t, "yaml: cannot represent node with unknown kind 0", err)
	err = n.Load(&v)
	assert.ErrorMatches(t, "yaml: cannot construct node with unknown kind 0", err)
}

func TestNodeOmitEmpty(t *testing.T) {
	var v struct {
		A int
		B yaml.Node `yaml:",omitempty"`
	}
	v.A = 1
	data, err := yaml.Marshal(&v)
	assert.NoError(t, err)
	assert.Equal(t, "a: 1\n", string(data))

	v.B.Line = 1
	_, err = yaml.Marshal(&v)
	assert.ErrorMatches(t, "yaml: cannot represent node with unknown kind 0", err)
}

// NodeInfo represents the information about a YAML node in a test-friendly format
type NodeInfo struct {
	Kind    string      `yaml:"kind"`
	Style   string      `yaml:"style,omitempty"`
	Anchor  string      `yaml:"anchor,omitempty"`
	Tag     string      `yaml:"tag,omitempty"`
	Head    string      `yaml:"head,omitempty"`
	Line    string      `yaml:"line,omitempty"`
	Foot    string      `yaml:"foot,omitempty"`
	Text    string      `yaml:"text,omitempty"`
	Content []*NodeInfo `yaml:"content,omitempty"`
}

// isStandardTag checks if a tag is a standard YAML tag
func isStandardTag(tag string) bool {
	switch tag {
	case "!!null", "!!bool", "!!int", "!!float", "!!str", "!!seq", "!!map":
		return true
	}
	return false
}

// parseNodeInfo converts a NodeInfo structure into a yaml.Node
func parseNodeInfo(info *NodeInfo) (*yaml.Node, error) {
	if info == nil {
		return nil, fmt.Errorf("nil NodeInfo")
	}

	node := &yaml.Node{}

	// Parse Kind
	switch info.Kind {
	case "Document":
		node.Kind = yaml.DocumentNode
	case "Sequence":
		node.Kind = yaml.SequenceNode
	case "Mapping":
		node.Kind = yaml.MappingNode
	case "Scalar":
		node.Kind = yaml.ScalarNode
	case "Alias":
		node.Kind = yaml.AliasNode
	default:
		return nil, fmt.Errorf("unknown node kind: %s", info.Kind)
	}

	// Parse Style
	if info.Style != "" {
		switch info.Style {
		case "Double":
			node.Style = yaml.DoubleQuotedStyle
		case "Single":
			node.Style = yaml.SingleQuotedStyle
		case "Literal":
			node.Style = yaml.LiteralStyle
		case "Folded":
			node.Style = yaml.FoldedStyle
		case "Flow":
			node.Style = yaml.FlowStyle
		case "Tagged":
			node.Style = yaml.TaggedStyle
		default:
			return nil, fmt.Errorf("unknown style: %s", info.Style)
		}
	}

	// Set other fields
	node.Anchor = info.Anchor
	node.Tag = info.Tag
	node.HeadComment = info.Head
	node.LineComment = info.Line
	node.FootComment = info.Foot

	// Add TaggedStyle bit for custom tags (not standard YAML tags)
	if info.Tag != "" && !isStandardTag(info.Tag) && node.Style != 0 {
		node.Style |= yaml.TaggedStyle
	}

	// Set value for scalar nodes
	if node.Kind == yaml.ScalarNode {
		node.Value = info.Text
	}

	// Parse content for non-scalar nodes
	if info.Content != nil {
		node.Content = make([]*yaml.Node, len(info.Content))
		for i, childInfo := range info.Content {
			childNode, err := parseNodeInfo(childInfo)
			if err != nil {
				return nil, fmt.Errorf("content[%d]: %w", i, err)
			}
			node.Content[i] = childNode
		}
	}

	return node, nil
}

// formatNodeInfo converts a yaml.Node into a NodeInfo structure for comparison
func formatNodeInfo(n yaml.Node) *NodeInfo {
	info := &NodeInfo{
		Kind: formatKindForTest(n.Kind),
	}

	if style := formatStyleForTest(n.Style); style != "" {
		info.Style = style
	}
	if n.Anchor != "" {
		info.Anchor = n.Anchor
	}
	if tag := formatTagForTest(n.Tag, n.Style); tag != "" {
		info.Tag = tag
	}
	if n.HeadComment != "" {
		info.Head = n.HeadComment
	}
	if n.LineComment != "" {
		info.Line = n.LineComment
	}
	if n.FootComment != "" {
		info.Foot = n.FootComment
	}

	if info.Kind == "Scalar" {
		info.Text = n.Value
	} else if n.Content != nil {
		info.Content = make([]*NodeInfo, len(n.Content))
		for i, node := range n.Content {
			info.Content[i] = formatNodeInfo(*node)
		}
	}

	return info
}

// formatKindForTest converts a YAML node kind into its string representation.
func formatKindForTest(k yaml.Kind) string {
	switch k {
	case yaml.DocumentNode:
		return "Document"
	case yaml.SequenceNode:
		return "Sequence"
	case yaml.MappingNode:
		return "Mapping"
	case yaml.ScalarNode:
		return "Scalar"
	case yaml.AliasNode:
		return "Alias"
	default:
		return "Unknown"
	}
}

// formatStyleForTest converts a YAML node style into its string representation.
func formatStyleForTest(s yaml.Style) string {
	// Strip out TaggedStyle bit - it's implicit when we have a custom tag
	baseStyle := s &^ yaml.TaggedStyle

	switch baseStyle {
	case yaml.DoubleQuotedStyle:
		return "Double"
	case yaml.SingleQuotedStyle:
		return "Single"
	case yaml.LiteralStyle:
		return "Literal"
	case yaml.FoldedStyle:
		return "Folded"
	case yaml.FlowStyle:
		return "Flow"
	}
	return ""
}

// formatTagForTest converts a YAML tag string to its string representation.
func formatTagForTest(tag string, style yaml.Style) string {
	// Check if the tag was explicit in the input
	tagWasExplicit := style&yaml.TaggedStyle != 0

	// Show !!str only if it was explicit in the input
	switch tag {
	case "!!str", "!!map", "!!seq":
		if tagWasExplicit {
			return tag
		}
		return ""
	}

	// Show all other tags
	return tag
}

// nodeInfoHasComments recursively checks if a NodeInfo expects any comments
func nodeInfoHasComments(info *NodeInfo) bool {
	if info.Head != "" || info.Line != "" || info.Foot != "" {
		return true
	}
	for _, child := range info.Content {
		if nodeInfoHasComments(child) {
			return true
		}
	}
	return false
}

// runNodeTestCase executes a single node test case
func runNodeTestCase(t *testing.T, tc map[string]any) {
	t.Helper()

	name := tc["name"].(string)
	yamlInput := tc["yaml"].(string)

	// Get the expected node structure
	nodeInfoData, ok := tc["node"]
	if !ok {
		t.Fatal("test case missing 'node' field")
	}

	// Convert the node data to NodeInfo
	var expectedInfo NodeInfo
	nodeBytes, err := yaml.Marshal(nodeInfoData)
	assert.NoError(t, err)
	err = yaml.Unmarshal(nodeBytes, &expectedInfo)
	assert.NoError(t, err)

	// Parse expected NodeInfo into yaml.Node
	expectedNode, err := parseNodeInfo(&expectedInfo)
	assert.NoError(t, err)

	// Check if decode/encode should be skipped
	decodeTest := true
	encodeTest := true
	if skipDecode, ok := tc["decode"].(bool); ok && !skipDecode {
		decodeTest = false
	}
	if skipEncode, ok := tc["encode"].(bool); ok && !skipEncode {
		encodeTest = false
	}

	if decodeTest {
		var actualNode yaml.Node
		// If the test expects comments, use Loader with WithLegacyComments
		if nodeInfoHasComments(&expectedInfo) {
			loader, err := yaml.NewLoader(bytes.NewReader([]byte(yamlInput)), yaml.WithLegacyComments())
			assert.NoError(t, err)
			err = loader.Load(&actualNode)
			assert.NoError(t, err)
		} else {
			err := yaml.Unmarshal([]byte(yamlInput), &actualNode)
			assert.NoError(t, err)
		}

		// Compare using NodeInfo for better error messages
		actualInfo := formatNodeInfo(actualNode)
		assertNodeInfoEqual(t, &expectedInfo, actualInfo, name)
	}

	if encodeTest {
		// Encode the expected node with 2-space indent
		buf := bytes.Buffer{}
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		err := enc.Encode(expectedNode)
		assert.NoError(t, err)
		err = enc.Close()
		assert.NoError(t, err)

		assert.Equal(t, yamlInput, buf.String())
	}
}

// assertNodeInfoEqual compares two NodeInfo structures and reports differences
func assertNodeInfoEqual(t *testing.T, expected, actual *NodeInfo, context string) {
	t.Helper()

	if expected == nil && actual == nil {
		return
	}
	if expected == nil {
		t.Fatalf("%s: expected nil, got %+v", context, actual)
		return
	}
	if actual == nil {
		t.Fatalf("%s: expected %+v, got nil", context, expected)
		return
	}

	assert.Equalf(t, expected.Kind, actual.Kind, "%s: Kind mismatch", context)
	assert.Equalf(t, expected.Style, actual.Style, "%s: Style mismatch", context)
	assert.Equalf(t, expected.Anchor, actual.Anchor, "%s: Anchor mismatch", context)
	assert.Equalf(t, expected.Tag, actual.Tag, "%s: Tag mismatch", context)
	assert.Equalf(t, expected.Head, actual.Head, "%s: Head comment mismatch", context)
	assert.Equalf(t, expected.Line, actual.Line, "%s: Line comment mismatch", context)
	assert.Equalf(t, expected.Foot, actual.Foot, "%s: Foot comment mismatch", context)
	assert.Equalf(t, expected.Text, actual.Text, "%s: Text mismatch", context)

	if len(expected.Content) != len(actual.Content) {
		t.Fatalf("%s: Content length mismatch: expected %d, got %d",
			context, len(expected.Content), len(actual.Content))
	}

	for i := range expected.Content {
		assertNodeInfoEqual(t, expected.Content[i], actual.Content[i],
			fmt.Sprintf("%s.content[%d]", context, i))
	}
}

func TestNodeFromYAML(t *testing.T) {
	t.Setenv("TZ", "UTC")
	datatest.RunTestCases(t, func() ([]map[string]any, error) {
		return datatest.LoadTestCasesFromFile("testdata/node.yaml", libyaml.LoadYAML)
	}, map[string]datatest.TestHandler{
		"node-test": runNodeTestCase,
	})
}

func TestNodeLoad(t *testing.T) {
	// Test basic Load functionality
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "name", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "test", Tag: "!!str"},
		},
	}

	var result map[string]string
	err := node.Load(&result)
	assert.NoError(t, err)
	assert.Equal(t, "test", result["name"])
}

func TestNodeLoadWithKnownFields(t *testing.T) {
	// Test that KnownFields option is respected
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "known", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "value", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "unknown", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "other", Tag: "!!str"},
		},
	}

	type Target struct {
		Known string `yaml:"known"`
	}

	// Without KnownFields - should succeed
	var result1 Target
	err := node.Load(&result1)
	assert.NoError(t, err)
	assert.Equal(t, "value", result1.Known)

	// With KnownFields - should fail
	var result2 Target
	err = node.Load(&result2, yaml.WithKnownFields())
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*unknown not found.*", err)
}

func TestNodeLoadPreservesKnownFieldsInUnmarshaler(t *testing.T) {
	// This test validates the fix for Issue #460
	type strictConfig struct {
		Name string `yaml:"name"`
		Port int    `yaml:"port"`
	}

	// Custom unmarshaler using Load with KnownFields
	type Config struct {
		strictConfig
	}

	var unmarshalCalled bool
	unmarshaler := struct {
		Config
	}{}

	// Override UnmarshalYAML to use node.Load
	oldUnmarshal := func(node *yaml.Node) error {
		unmarshalCalled = true
		type plain strictConfig
		return node.Load((*plain)(&unmarshaler.strictConfig), yaml.WithKnownFields())
	}

	// Valid YAML - should succeed
	validYAML := []byte(`
name: test
port: 8080
`)

	var validNode yaml.Node
	err := yaml.Unmarshal(validYAML, &validNode)
	assert.NoError(t, err)

	err = oldUnmarshal(validNode.Content[0])
	assert.NoError(t, err)
	assert.True(t, unmarshalCalled)
	assert.Equal(t, "test", unmarshaler.Name)
	assert.Equal(t, 8080, unmarshaler.Port)

	// Invalid YAML with unknown field - should fail
	invalidYAML := []byte(`
name: test
port: 8080
unknown: field
`)

	var invalidNode yaml.Node
	err = yaml.Unmarshal(invalidYAML, &invalidNode)
	assert.NoError(t, err)

	unmarshalCalled = false
	err = oldUnmarshal(invalidNode.Content[0])
	assert.NotNil(t, err)
	assert.True(t, unmarshalCalled)
	assert.ErrorMatches(t, ".*unknown not found.*", err)
}

func TestNodeDump(t *testing.T) {
	// Test basic Dump functionality
	value := map[string]string{"name": "test"}

	var node yaml.Node
	err := node.Dump(value)
	assert.NoError(t, err)
	assert.Equal(t, yaml.MappingNode, node.Kind)
	assert.Equal(t, "!!map", node.Tag)
	assert.Equal(t, 2, len(node.Content))
}

func TestNodeDumpWithOptions(t *testing.T) {
	// Test Dump with encoder options
	type Config struct {
		Name string `yaml:"name"`
		Port int    `yaml:"port"`
	}
	value := Config{Name: "myapp", Port: 8080}

	// Dump with V4 (default)
	var node1 yaml.Node
	err := node1.Dump(value, yaml.V4)
	assert.NoError(t, err)
	assert.Equal(t, yaml.MappingNode, node1.Kind)

	// Dump with V3
	var node2 yaml.Node
	err = node2.Dump(value, yaml.V3)
	assert.NoError(t, err)
	assert.Equal(t, yaml.MappingNode, node2.Kind)

	// Both should produce valid nodes with same content structure
	assert.Equal(t, len(node1.Content), len(node2.Content))
}

func TestNodeLoadWithUniqueKeys(t *testing.T) {
	// Test that UniqueKeys option is respected
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "value1", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "key", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "value2", Tag: "!!str"},
		},
	}

	// With UniqueKeys (default) - should fail on duplicate
	var result1 map[string]string
	err := node.Load(&result1, yaml.WithUniqueKeys())
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*already defined.*", err)

	// Without UniqueKeys - should succeed (last value wins)
	var result2 map[string]string
	err = node.Load(&result2, yaml.WithUniqueKeys(false))
	assert.NoError(t, err)
	assert.Equal(t, "value2", result2["key"])
}

func TestNodeLoadInvalidOptions(t *testing.T) {
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "test",
		Tag:   "!!str",
	}

	// Test with invalid indent option (should fail during applyOptions)
	var result string
	err := node.Load(&result, yaml.WithIndent(100))
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*indent must be.*", err)
}

func TestNodeDumpInvalidOptions(t *testing.T) {
	value := "test"

	// Test with invalid indent option
	var node yaml.Node
	err := node.Dump(value, yaml.WithIndent(100))
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*indent must be.*", err)
}
