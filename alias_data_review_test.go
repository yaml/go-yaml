// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/aliasdata"
)

func TestAliasDataPrecomposedNodePolicy(t *testing.T) {
	var node yaml.Node
	if err := yaml.Load([]byte("[&value 42, *value]"), &node); err != nil {
		t.Fatal(err)
	}
	for _, options := range [][]aliasdata.Option{
		{aliasdata.WithDisable(true)},
		{aliasdata.WithStream(true)},
		{aliasdata.WithFile("/not/a/real/file")},
	} {
		p := newAliasPlugin(t, options...)
		var value any
		err := node.Load(&value, yaml.WithPlugin(p))
		if err == nil || !strings.Contains(err.Error(), "precomposed node input") {
			t.Fatalf("explicit alias policy bypassed: %#v, %v", value, err)
		}
	}
	var value any
	if err := node.Load(&value); err != nil ||
		!reflect.DeepEqual(value, []any{42, 42}) {
		t.Fatalf("default node loading changed: %#v, %v", value, err)
	}
}

type aliasDataObservedNode struct{ node yaml.Node }

func (v *aliasDataObservedNode) UnmarshalYAML(node *yaml.Node) error {
	v.node = *node
	return nil
}

func TestAliasDataFilePreservesNodes(t *testing.T) {
	for _, tc := range []struct {
		name, scalar, tag, value string
		kind                     yaml.Kind
		style                    yaml.Style
	}{
		{
			"binary", "!!binary SGVsbG8=", "!!binary", "SGVsbG8=",
			yaml.ScalarNode, yaml.TaggedStyle,
		},
		{
			"custom scalar", "!application 0x2a", "!application", "0x2a",
			yaml.ScalarNode, yaml.TaggedStyle,
		},
		{
			"integer spelling", "0x2a", "!!int", "0x2a",
			yaml.ScalarNode, 0,
		},
		{
			"quoted string", "'001'", "!!str", "001",
			yaml.ScalarNode, yaml.SingleQuotedStyle,
		},
		{
			"custom mapping", "!application {answer: 42}", "!application", "",
			yaml.MappingNode, yaml.TaggedStyle | yaml.FlowStyle,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "aliases.yaml")
			writeAliasFile(t, file, "value: "+tc.scalar+"\n")
			p := newAliasPlugin(t, aliasdata.WithFile(file))
			var value aliasDataObservedNode
			if err := yaml.Load([]byte("*value"), &value, yaml.WithPlugin(p)); err != nil {
				t.Fatal(err)
			}
			node := value.node
			if node.Kind != tc.kind || node.Tag != tc.tag ||
				node.Value != tc.value || node.Style != tc.style {
				t.Fatalf("file node changed: %+v", node)
			}
		})
	}
}

func TestAliasDataFileBinaryAndRootMerge(t *testing.T) {
	for _, text := range []string{
		"value: !!binary SGVsbG8=\n",
		"base: &base {value: !!binary SGVsbG8=}\n<<: *base\n",
		"name: &name value\n*name: !!binary SGVsbG8=\n",
	} {
		file := filepath.Join(t.TempDir(), "aliases.yaml")
		writeAliasFile(t, file, text)
		p := newAliasPlugin(t, aliasdata.WithFile(file))
		var value []byte
		if err := yaml.Load([]byte("*value"), &value, yaml.WithPlugin(p)); err != nil || string(value) != "Hello" {
			t.Fatalf("binary file source: %q, %v", value, err)
		}
	}
}

func TestAliasDataFileUsesSelectedParser(t *testing.T) {
	file := filepath.Join(t.TempDir(), "aliases.yaml")
	writeAliasFile(t, file, "custom file input")
	parser := registeredSourceFunc(func(input []byte) ([]yaml.PluginEvent, error) {
		events := []yaml.PluginEvent{
			{Type: "stream_start"}, {Type: "document_start"},
		}
		if string(input) == "main input" {
			events = append(events, yaml.PluginEvent{Type: "alias", Anchor: "value"})
		} else {
			events = append(events,
				yaml.PluginEvent{Type: "mapping_start"},
				yaml.PluginEvent{Type: "scalar", Value: "value"},
				yaml.PluginEvent{Type: "scalar", Tag: "!application", Value: "001"},
				yaml.PluginEvent{Type: "mapping_end"})
		}
		return append(events,
			yaml.PluginEvent{Type: "document_end"},
			yaml.PluginEvent{Type: "stream_end"}), nil
	})
	p := newAliasPlugin(t, aliasdata.WithFile(file))
	var value aliasDataObservedNode
	err := yaml.Load([]byte("main input"), &value, yaml.WithPlugin(parser, p))
	if err != nil || value.node.Tag != "!application" || value.node.Value != "001" {
		t.Fatalf("selected parser file node: %+v, %v", value.node, err)
	}
}

func TestAliasDataOverlappingSlices(t *testing.T) {
	source := make([]any, 2)
	source[0] = "leaf"
	source[1] = source[:1]
	p := newAliasPlugin(t, aliasdata.WithData(map[string]any{"value": source}))
	var value any
	err := yaml.Load([]byte("*value"), &value, yaml.WithPlugin(p))
	want := []any{"leaf", []any{"leaf"}}
	if err != nil || !reflect.DeepEqual(value, want) {
		t.Fatalf("acyclic overlapping slices: %#v, %v", value, err)
	}

	cycle := make([]any, 1)
	cycle[0] = cycle
	p = newAliasPlugin(t, aliasdata.WithData(map[string]any{"value": cycle}))
	err = yaml.Load([]byte("*value"), &value, yaml.WithPlugin(p))
	if err == nil || !strings.Contains(err.Error(), "cyclic Go value") {
		t.Fatalf("slice cycle was accepted: %v", err)
	}
}
