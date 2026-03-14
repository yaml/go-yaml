// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/errfmt"
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

func TestWithPlugin_Errfmt_Default(t *testing.T) {
	var result any
	err := yaml.Load(errfmtBadYAML, &result, yaml.WithPlugin(errfmt.New()))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	got := err.Error()
	want := "go-yaml load error in "
	if !strings.HasPrefix(got, want) {
		t.Errorf("Default format: got %q, want prefix %q", got, want)
	}
}

func TestWithPlugin_Errfmt_Legacy(t *testing.T) {
	var result any
	err := yaml.Load(errfmtBadYAML, &result, yaml.WithPlugin(errfmt.New(errfmt.FormatLegacy)))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	got := err.Error()
	want := "yaml: line "
	if !strings.HasPrefix(got, want) {
		t.Errorf("Legacy format: got %q, want prefix %q", got, want)
	}
}

func TestWithPlugin_Errfmt_Compact(t *testing.T) {
	var result any
	err := yaml.Load(errfmtBadYAML, &result, yaml.WithPlugin(errfmt.New(errfmt.FormatCompact)))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	got := err.Error()
	// Compact format: "stage:line:col: msg"
	if strings.HasPrefix(got, "go-yaml") || strings.HasPrefix(got, "yaml:") {
		t.Errorf("Compact format should not start with verbose prefix, got %q", got)
	}
	if !strings.Contains(got, ":") {
		t.Errorf("Compact format should contain ':', got %q", got)
	}
}
