// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestCanonicalDumpRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		input any
		want  any
	}{
		{"integer", 30, 30},
		{"true", true, true},
		{"false", false, false},
		{"float", 1.25, 1.25},
		{"integral float", float64(1), float64(1)},
		{"null", nil, nil},
		{"string", "hello", "hello"},
		{"numeric string", "30", "30"},
		{"boolean string", "true", "true"},
		{"null string", "null", "null"},
		{"empty string", "", ""},
		{"multiline string", "hello\nworld\n", "hello\nworld\n"},
		{"binary string", "\x80\x81", "\x80\x81"},
		{"timestamp", time.Unix(0, 0).UTC(), time.Unix(0, 0).UTC()},
		{
			"mapping",
			map[string]any{"age": 30, "enabled": true},
			map[string]any{"age": 30, "enabled": true},
		},
		{
			"typed keys",
			map[any]any{30: true, false: float64(1)},
			map[any]any{30: true, false: float64(1)},
		},
		{
			"sequence",
			[]any{30, true, float64(1), nil, "30"},
			[]any{30, true, float64(1), nil, "30"},
		},
		{"zero node", &yaml.Node{}, nil},
		{
			"untagged plain node",
			&yaml.Node{Kind: yaml.ScalarNode, Value: "30"}, 30,
		},
		{
			"untagged quoted node",
			&yaml.Node{
				Kind: yaml.ScalarNode, Style: yaml.SingleQuotedStyle,
				Value: "30",
			}, "30",
		},
		{
			"untagged binary node",
			&yaml.Node{Kind: yaml.ScalarNode, Value: "\x80\x81"},
			"\x80\x81",
		},
		{
			"untagged collection nodes",
			&yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "values"},
					{Kind: yaml.SequenceNode, Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "true"},
						{Kind: yaml.ScalarNode, Value: "null"},
					}},
				},
			},
			map[string]any{"values": []any{true, nil}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := yaml.Dump(tc.input, yaml.WithCanonical())
			assert.NoError(t, err)
			var got any
			assert.NoError(t, yaml.Load(data, &got))
			assert.DeepEqualf(t, tc.want, got, "got %T; want %T", got, tc.want)
		})
	}
}

func TestCanonicalDumpExplicitTags(t *testing.T) {
	data, err := yaml.Dump(map[string]int{"age": 30}, yaml.WithCanonical())
	assert.NoError(t, err)
	assert.Equal(t, "---\n!!map {\n  ? !!str \"age\"\n  : !!int \"30\",\n}\n", string(data))
}

func TestCanonicalOutputPaths(t *testing.T) {
	want := map[string]any{
		"age": 30, "enabled": true, "ratio": float64(1),
		"values": []any{false, nil, "30"},
	}
	t.Run("loaded nodes", func(t *testing.T) {
		var node yaml.Node
		input := "age: 30\nenabled: true\nratio: 1.0\nvalues: [false, null, '30']\n"
		assert.NoError(t, yaml.Load([]byte(input), &node))
		data, err := yaml.Dump(&node, yaml.WithCanonical())
		assert.NoError(t, err)
		var got any
		assert.NoError(t, yaml.Load(data, &got))
		assert.DeepEqual(t, want, got)
	})

	t.Run("node dump", func(t *testing.T) {
		var node yaml.Node
		assert.NoError(t, node.Dump(want, yaml.WithCanonical()))
		var got any
		assert.NoError(t, node.Load(&got))
		assert.DeepEqual(t, want, got)
	})

	t.Run("streaming", func(t *testing.T) {
		var buf bytes.Buffer
		dumper, err := yaml.NewDumper(&buf, yaml.WithCanonical())
		assert.NoError(t, err)
		assert.NoError(t, dumper.Dump(want))
		assert.NoError(t, dumper.Dump(want))
		assert.NoError(t, dumper.Close())
		loader, err := yaml.NewLoader(&buf)
		assert.NoError(t, err)
		for i := 0; i < 2; i++ {
			var got any
			assert.NoError(t, loader.Load(&got))
			assert.DeepEqual(t, want, got)
		}
		var extra any
		assert.Equal(t, io.EOF, loader.Load(&extra))
	})

	t.Run("YAML options", func(t *testing.T) {
		option, err := yaml.OptsYAML("canonical: true")
		assert.NoError(t, err)
		data, err := yaml.Dump(want, option)
		assert.NoError(t, err)
		var got any
		assert.NoError(t, yaml.Load(data, &got))
		assert.DeepEqual(t, want, got)
	})

	t.Run("anchors and aliases", func(t *testing.T) {
		var node yaml.Node
		assert.NoError(t, yaml.Load([]byte("a: &n 30\nb: *n\n"), &node))
		data, err := yaml.Dump(&node, yaml.WithCanonical())
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(data), "&n"))
		assert.True(t, strings.Contains(string(data), "*n"))
		var got any
		assert.NoError(t, yaml.Load(data, &got))
		assert.DeepEqual(t, map[string]any{"a": 30, "b": 30}, got)
	})
}

func TestCanonicalDisabled(t *testing.T) {
	data, err := yaml.Dump(map[string]int{"age": 30},
		yaml.WithCanonical(), yaml.WithCanonical(false))
	assert.NoError(t, err)
	assert.Equal(t, "age: 30\n", string(data))
}
