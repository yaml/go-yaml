// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	errfmtv3 "go.yaml.in/yaml/v4/plugin/errfmt/v3"
	errfmtv4 "go.yaml.in/yaml/v4/plugin/errfmt/v4"
	"go.yaml.in/yaml/v4/plugin/limit"
)

// generateAliases builds YAML with n aliases referencing a large anchor.
func generateAliases(n int) []byte {
	var sb strings.Builder
	sb.WriteString("anchor: &anchor [1, 2, 3]\nrefs:\n")
	for i := 0; i < n; i++ {
		sb.WriteString("- *anchor\n")
	}
	return []byte(sb.String())
}

// generateDeepNesting builds deeply nested flow YAML.
func generateDeepNesting(depth int) []byte {
	var sb strings.Builder
	for i := 0; i < depth; i++ {
		sb.WriteString("[")
	}
	sb.WriteString("x")
	for i := 0; i < depth; i++ {
		sb.WriteString("]")
	}
	return []byte(sb.String())
}

func TestWithPlugin_Limit_AliasFunc(t *testing.T) {
	called := false
	fn := func(aliasCount, constructCount int) error {
		called = true
		return nil
	}
	data := generateAliases(200)
	var result any
	err := yaml.Load(data, &result, yaml.WithPlugin(limit.New(limit.AliasFunc(fn))))
	if err != nil {
		t.Fatalf("Expected success with custom AliasFunc, got: %v", err)
	}
	if !called {
		t.Error("Expected custom AliasFunc to be called")
	}
}

func TestWithPlugin_Limit_DepthFunc(t *testing.T) {
	called := false
	fn := func(depth int, ctx *yaml.DepthContext) error {
		called = true
		return nil
	}
	data := generateDeepNesting(5)
	var result any
	err := yaml.Load(data, &result, yaml.WithPlugin(limit.New(limit.DepthFunc(fn))))
	if err != nil {
		t.Fatalf("Expected success with custom DepthFunc, got: %v", err)
	}
	if !called {
		t.Error("Expected custom DepthFunc to be called")
	}
}

func TestWithPlugin_UnsupportedType(t *testing.T) {
	data := []byte(`key: value`)
	var result map[string]any
	// Pass an unsupported type (integer) as a plugin
	err := yaml.Load(data, &result, yaml.WithPlugin(42))
	if err == nil {
		t.Fatal("Expected error for unsupported plugin type, got nil")
	}
	if err.Error() != "yaml: unsupported plugin type: int" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestDefaultBehavior_HasLimit(t *testing.T) {
	// Bare NewLoader should have default depth limits
	data := generateDeepNesting(10001)
	loader, err := yaml.NewLoader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}
	var result any
	err = loader.Load(&result)
	if err == nil {
		t.Fatal("Expected error from default depth limits, got nil")
	}
}

// errfmtBadYAML triggers a scanner error (block sequence not allowed here).
var errfmtBadYAML = []byte("value: -\n")

func TestWithPlugin_ErrfmtV4_Default(t *testing.T) {
	var result any
	err := yaml.Load(errfmtBadYAML, &result, yaml.WithPlugin(errfmtv4.Must()))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	got := err.Error()
	want := "go-yaml load error in "
	if !strings.HasPrefix(got, want) {
		t.Errorf("Default format: got %q, want prefix %q", got, want)
	}
}

func TestWithPlugin_ErrfmtV3(t *testing.T) {
	var result any
	err := yaml.Load(errfmtBadYAML, &result, yaml.WithPlugin(errfmtv3.New()))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	got := err.Error()
	want := "yaml: line "
	if !strings.HasPrefix(got, want) {
		t.Errorf("Legacy format: got %q, want prefix %q", got, want)
	}
}

func TestWithPlugin_ErrfmtV4_LongPosition(t *testing.T) {
	p, err := errfmtv4.New(errfmtv4.WithPositionStyle(errfmtv4.PositionLong))
	if err != nil {
		t.Fatalf("New errfmtv4 failed: %v", err)
	}
	var result any
	err = yaml.Load(errfmtBadYAML, &result, yaml.WithPlugin(p))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	got := err.Error()
	if !strings.Contains(got, "line 1, column 8") {
		t.Errorf("Long position format: got %q", got)
	}
}

func TestWithPlugin_ErrfmtV4_LinePosition(t *testing.T) {
	p, err := errfmtv4.New(errfmtv4.WithPositionStyle(errfmtv4.PositionLine))
	if err != nil {
		t.Fatalf("New errfmtv4 failed: %v", err)
	}
	var result any
	err = yaml.Load(errfmtBadYAML, &result, yaml.WithPlugin(p))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	got := err.Error()
	if !strings.Contains(got, " at line 1: ") {
		t.Errorf("Line position format: got %q", got)
	}
}

func TestWithPlugin_ErrfmtV4_Template(t *testing.T) {
	p, err := errfmtv4.New(errfmtv4.WithTemplate("{{.Stage}} {{pos .Mark}} {{.Message}}"))
	if err != nil {
		t.Fatalf("New errfmtv4 failed: %v", err)
	}
	var result any
	err = yaml.Load(errfmtBadYAML, &result, yaml.WithPlugin(p))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	got := err.Error()
	want := "scanner L1.C8 block sequence entries are not allowed in this context"
	if got != want {
		t.Errorf("Template format: got %q, want %q", got, want)
	}
}

func TestWithPlugin_ErrfmtV4_InvalidTemplate(t *testing.T) {
	_, err := errfmtv4.New(errfmtv4.WithTemplate("{{"))
	if err == nil {
		t.Fatal("Expected invalid template error, got nil")
	}
}

func TestWithDefaults_ErrorFormatting(t *testing.T) {
	tests := []struct {
		name string
		opts []yaml.Option
		want string
	}{
		{
			name: "bare defaults use v4",
			want: "go-yaml load error in ",
		},
		{
			name: "v2 defaults use v4",
			opts: []yaml.Option{yaml.WithV2Defaults()},
			want: "go-yaml load error in ",
		},
		{
			name: "v3 defaults use v3",
			opts: []yaml.Option{yaml.WithV3Defaults()},
			want: "yaml: line ",
		},
		{
			name: "v4 defaults use v4",
			opts: []yaml.Option{yaml.WithV4Defaults()},
			want: "go-yaml load error in ",
		},
		{
			name: "explicit plugin after defaults wins",
			opts: []yaml.Option{yaml.WithV3Defaults(), yaml.WithPlugin(errfmtv4.Must())},
			want: "go-yaml load error in ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result any
			err := yaml.Load(errfmtBadYAML, &result, tt.opts...)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !strings.HasPrefix(err.Error(), tt.want) {
				t.Errorf("got %q, want prefix %q", err.Error(), tt.want)
			}
		})
	}
}
