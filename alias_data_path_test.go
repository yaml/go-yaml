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

func TestAliasDataPathScope(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options []aliasdata.Option
		stream  bool
	}{
		{"automatic", []aliasdata.Option{aliasdata.WithPath(true)}, true},
		{"explicit off last", []aliasdata.Option{
			aliasdata.WithPath(true), aliasdata.WithStream(false),
		}, false},
		{"explicit off first", []aliasdata.Option{
			aliasdata.WithStream(false), aliasdata.WithPath(true),
		}, false},
		{"explicit on", []aliasdata.Option{
			aliasdata.WithStream(true), aliasdata.WithPath(true),
		}, true},
		{"path false", []aliasdata.Option{aliasdata.WithPath(false)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newAliasPlugin(t, tc.options...)
			var values []any
			// Plain aliases also inherit the automatically enabled scope.
			err := yaml.Load([]byte("--- &saved {answer: 42}\n--- *saved\n"),
				&values, yaml.WithPlugin(p), yaml.WithAllDocuments())
			if tc.stream {
				if err != nil || len(values) != 2 || !reflect.DeepEqual(values[0], values[1]) {
					t.Fatalf("stream scope: %#v, %v", values, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "unknown anchor") {
				t.Fatalf("document scope was not retained: %v", err)
			}
		})
	}
}

func TestAliasDataPathSources(t *testing.T) {
	file := filepath.Join(t.TempDir(), "aliases.yaml")
	writeAliasFile(t, file, "from_file: {nested: [file]}\n")
	p := newAliasPlugin(t, aliasdata.WithPath(true), aliasdata.WithFile(file),
		aliasdata.WithData(map[string]any{"x": map[string]any{
			"nested": []any{"inline"},
		}}))
	var result any
	err := yaml.Load([]byte("[*x.nested.0, *from_file.nested.0]"),
		&result, yaml.WithPlugin(p))
	if err != nil || !reflect.DeepEqual(result, []any{"inline", "file"}) {
		t.Fatalf("path sources: %#v, %v", result, err)
	}
}

func TestAliasDataPathTypedConstruction(t *testing.T) {
	p := newAliasPlugin(t, aliasdata.WithPath(true),
		aliasdata.WithData(map[string]any{
			"configs": map[string]any{
				"server": map[string]any{"port": 8080, "host": "localhost"},
			},
		}))
	var config struct {
		Port aliasPort
		Host string
	}
	err := yaml.Load([]byte("*configs.server"), &config, yaml.WithPlugin(p))
	if err != nil || config.Port != 8081 || config.Host != "localhost" {
		t.Fatalf("typed path result: %+v, %v", config, err)
	}
	var node yaml.Node
	err = yaml.Load([]byte("*configs.server.port"), &node, yaml.WithPlugin(p))
	if err != nil || node.Content[0].Alias.Tag != "!!int" {
		t.Fatalf("path node target: %#v, %v", node, err)
	}
}

func TestAliasDataPathGraphTraversal(t *testing.T) {
	aliasCycle := &yaml.Node{Kind: yaml.AliasNode}
	aliasCycle.Alias = aliasCycle
	mergeCycle := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	mergeCycle.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!merge", Value: "<<"},
		{Kind: yaml.AliasNode, Alias: mergeCycle},
	}
	shared := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	dag := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!merge", Value: "<<"},
		{Kind: yaml.SequenceNode, Content: []*yaml.Node{
			{Kind: yaml.AliasNode, Alias: shared},
			{Kind: yaml.AliasNode, Alias: shared},
		}},
	}}
	for _, tc := range []struct {
		name, alias, failure string
		value                any
	}{
		{"alias cycle", "x.foo", "cyclic alias", aliasCycle},
		{"merge cycle", "x.foo", "cyclic merge", mergeCycle},
		{"shared merge is not a cycle", "x.foo", "missing mapping key", dag},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newAliasPlugin(t, aliasdata.WithPath(true),
				aliasdata.WithData(map[string]any{"x": tc.value}))
			var value any
			err := yaml.Load([]byte("*"+tc.alias), &value, yaml.WithPlugin(p))
			if err == nil || !strings.Contains(err.Error(), tc.failure) {
				t.Fatalf("graph traversal: %v, want %q", err, tc.failure)
			}
		})
	}
}

func TestAliasDataPathGrammarWithParserPlugin(t *testing.T) {
	for _, name := range []string{"x.é", "é.foo", "x.'word'", "missing.01"} {
		t.Run(name, func(t *testing.T) {
			parser := registeredSourceFunc(func([]byte) ([]yaml.PluginEvent, error) {
				return []yaml.PluginEvent{
					{Type: "stream_start"},
					{Type: "document_start"},
					{Type: "alias", Anchor: name},
					{Type: "document_end"},
					{Type: "stream_end"},
				}, nil
			})
			p := newAliasPlugin(t, aliasdata.WithPath(true))
			var result any
			err := yaml.Load([]byte("ignored"), &result, yaml.WithPlugin(parser, p))
			want := "invalid path segment"
			if name == "missing.01" {
				want = "sequence index must not have leading zeros"
			}
			if err == nil || !strings.Contains(err.Error(), want) ||
				!strings.Contains(err.Error(), name) {
				t.Fatalf("path grammar: %v; want %s for %s", err, want, name)
			}
		})
	}
}
