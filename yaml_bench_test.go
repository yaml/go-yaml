// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v4"
)

// benchPlain is a decode target with no custom UnmarshalYAML.
type benchPlain struct {
	Fields map[string]string `yaml:",inline"`
}

// benchCustom is a decode target whose UnmarshalYAML calls node.Decode —
// the path that will inherit loader options after the tree-stamp fix.
type benchCustom struct {
	Fields map[string]string `yaml:",inline"`
}

func (b *benchCustom) UnmarshalYAML(node *yaml.Node) error {
	type plain benchCustom
	return node.Decode((*plain)(b))
}

func makeKVDoc(n int) []byte {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, "key%d: value%d\n", i, i)
	}
	return []byte(sb.String())
}

var (
	benchSmallDoc  = makeKVDoc(10)
	benchMediumDoc = makeKVDoc(100)
	benchLargeDoc  = makeKVDoc(1000)
)

func BenchmarkDecode(b *testing.B) {
	targets := []struct {
		name   string
		decode func(data []byte, known bool) error
	}{
		{
			name: "plain",
			decode: func(data []byte, known bool) error {
				var v benchPlain
				dec := yaml.NewDecoder(bytes.NewReader(data))
				dec.KnownFields(known)
				return dec.Decode(&v)
			},
		},
		{
			name: "custom",
			decode: func(data []byte, known bool) error {
				var v benchCustom
				dec := yaml.NewDecoder(bytes.NewReader(data))
				dec.KnownFields(known)
				return dec.Decode(&v)
			},
		},
	}

	options := []struct {
		name        string
		knownFields bool
	}{
		{"default", false},
		{"known-fields", true},
	}

	sizes := []struct {
		name string
		data []byte
	}{
		{"small", benchSmallDoc},
		{"medium", benchMediumDoc},
		{"large", benchLargeDoc},
	}

	for _, size := range sizes {
		for _, target := range targets {
			for _, opt := range options {
				size, target, opt := size, target, opt
				name := fmt.Sprintf("target=%s/option=%s/size=%s", target.name, opt.name, size.name)
				b.Run(name, func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						if err := target.decode(size.data, opt.knownFields); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		}
	}
}
