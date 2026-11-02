// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/loaderlimits"
)

// Keep expansion bounded so a missing alias check fails without hanging.
func nodeAliasLimitYAML(levels int) []byte {
	var data strings.Builder
	data.WriteString("a: &a {k: v}\n")
	for level := 0; level < levels; level++ {
		name := byte('b' + level)
		fmt.Fprintf(&data, "%c: &%c [*%c, *%c, *%c, *%c]\n",
			name, name, name-1, name-1, name-1, name-1)
	}
	return []byte(data.String())
}

func TestNodeDefaultAliasLimits(t *testing.T) {
	for _, levels := range []int{2, 7} {
		var node yaml.Node
		if err := yaml.Unmarshal(nodeAliasLimitYAML(levels), &node); err != nil {
			t.Fatal(err)
		}
		for _, method := range []struct {
			name string
			load func(any) error
		}{
			{"Decode", node.Decode},
			{"Load", func(out any) error { return node.Load(out) }},
		} {
			t.Run(fmt.Sprintf("%s/levels=%d", method.name, levels), func(t *testing.T) {
				var out any
				err := method.load(&out)
				if levels == 7 {
					if err == nil || !strings.Contains(err.Error(), "excessive aliasing") {
						t.Fatalf("got %v, want excessive aliasing", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				mapping, ok := out.(map[string]any)
				if !ok || len(mapping) != levels+1 {
					t.Fatalf("unexpected decoded value: %#v", out)
				}
			})
		}
	}
}

type nodeDecodeLimitUnmarshaler struct{ value any }

func (u *nodeDecodeLimitUnmarshaler) UnmarshalYAML(node *yaml.Node) error {
	return node.Decode(&u.value)
}

func TestNodeDecodeAliasLimitsInUnmarshaler(t *testing.T) {
	for _, method := range []struct {
		name string
		load func([]byte, any) error
	}{
		{"Unmarshal", yaml.Unmarshal},
		{"Decoder", func(data []byte, out any) error {
			return yaml.NewDecoder(bytes.NewReader(data)).Decode(out)
		}},
		{"Load", func(data []byte, out any) error { return yaml.Load(data, out) }},
	} {
		for _, levels := range []int{2, 7} {
			t.Run(fmt.Sprintf("%s/levels=%d", method.name, levels), func(t *testing.T) {
				var out nodeDecodeLimitUnmarshaler
				err := method.load(nodeAliasLimitYAML(levels), &out)
				if levels == 7 {
					if err == nil || !strings.Contains(err.Error(), "excessive aliasing") {
						t.Fatalf("got %v, want excessive aliasing", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				mapping, ok := out.value.(map[string]any)
				if !ok || len(mapping) != levels+1 {
					t.Fatalf("unexpected decoded value: %#v", out.value)
				}
			})
		}
	}
}

func TestNodeLoadAliasLimitOverrides(t *testing.T) {
	var node yaml.Node
	if err := yaml.Unmarshal(nodeAliasLimitYAML(7), &node); err != nil {
		t.Fatal(err)
	}
	var out any
	if err := node.Load(&out,
		yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasNone()))); err != nil {
		t.Fatalf("explicitly disabled alias limit: %v", err)
	}
	err := node.Load(&out,
		yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasValue(0))))
	if err == nil || !strings.Contains(err.Error(), "exceeded max alias count of 0") {
		t.Fatalf("custom alias limit: got %v", err)
	}
}
