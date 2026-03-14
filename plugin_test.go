// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"errors"
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
	p, err := errfmtv4.New(errfmtv4.WithLoadTemplate("{{.Stage}} {{pos .Mark}} {{.Message}}"))
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
	_, err := errfmtv4.New(errfmtv4.WithLoadTemplate("{{"))
	if err == nil {
		t.Fatal("Expected invalid template error, got nil")
	}
}

func TestWithPlugin_ErrfmtV4_DumpDefault(t *testing.T) {
	_, err := yaml.Dump(make(chan int), yaml.WithPlugin(errfmtv4.Must()))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	want := "go-yaml dump error in representer: "
	if !strings.HasPrefix(err.Error(), want) {
		t.Errorf("Default dump format: got %q, want prefix %q", err, want)
	}
}

func TestWithPlugin_ErrfmtV3_Dump(t *testing.T) {
	_, err := yaml.Dump(make(chan int), yaml.WithPlugin(errfmtv3.New()))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	want := "yaml: cannot represent type: chan int"
	if err.Error() != want {
		t.Errorf("Legacy dump format: got %q, want %q", err, want)
	}
}

func TestWithPlugin_ErrfmtV4_DumpTemplate(t *testing.T) {
	p, err := errfmtv4.New(errfmtv4.WithDumpTemplate("{{.Stage}}/{{.Message}}"))
	if err != nil {
		t.Fatalf("New errfmtv4 failed: %v", err)
	}
	_, err = yaml.Dump(make(chan int), yaml.WithPlugin(p))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	want := "representer/cannot represent type: chan int"
	if err.Error() != want {
		t.Errorf("Dump template format: got %q, want %q", err, want)
	}
}

func TestWithPlugin_ErrfmtV4_DumpStages(t *testing.T) {
	t.Run("serializer", func(t *testing.T) {
		n := yaml.Node{Kind: 99}
		_, err := yaml.Dump(&n, yaml.WithPlugin(errfmtv4.Must()))
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		want := "go-yaml dump error in serializer: "
		if !strings.HasPrefix(err.Error(), want) {
			t.Errorf("Serializer dump format: got %q, want prefix %q", err, want)
		}
	})

	t.Run("writer", func(t *testing.T) {
		dumper, err := yaml.NewDumper(errorWriter{}, yaml.WithPlugin(errfmtv4.Must()))
		if err != nil {
			t.Fatalf("NewDumper failed: %v", err)
		}
		err = dumper.Dump(map[string]string{"a": "b"})
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		want := "go-yaml dump error in writer: some write error"
		if err.Error() != want {
			t.Errorf("Writer dump format: got %q, want %q", err, want)
		}
	})
}

type customDumpErrorMarshaler struct{}

func (customDumpErrorMarshaler) MarshalYAML() (any, error) {
	return nil, yaml.NewDumpError(
		yaml.RepresenterStage,
		"custom dump error",
		errors.New("custom cause"),
	)
}

func TestWithPlugin_ErrfmtV3_CustomDumpError(t *testing.T) {
	_, err := yaml.Dump(customDumpErrorMarshaler{}, yaml.WithPlugin(errfmtv3.New()))
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if err.Error() != "yaml: custom dump error" {
		t.Errorf("Custom dump error format: got %q", err)
	}
	var dumpErr *yaml.DumpError
	if !errors.As(err, &dumpErr) {
		t.Fatal("Expected errors.As to find *yaml.DumpError")
	}
	if dumpErr.Stage != yaml.RepresenterStage {
		t.Errorf("DumpError stage: got %q, want %q", dumpErr.Stage, yaml.RepresenterStage)
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

func TestWithDefaults_DumpErrorFormatting(t *testing.T) {
	tests := []struct {
		name string
		opts []yaml.Option
		want string
	}{
		{
			name: "bare defaults use v4",
			want: "go-yaml dump error in representer: ",
		},
		{
			name: "v2 defaults use v4",
			opts: []yaml.Option{yaml.WithV2Defaults()},
			want: "go-yaml dump error in representer: ",
		},
		{
			name: "v3 defaults use v3",
			opts: []yaml.Option{yaml.WithV3Defaults()},
			want: "yaml: cannot represent type: chan int",
		},
		{
			name: "v4 defaults use v4",
			opts: []yaml.Option{yaml.WithV4Defaults()},
			want: "go-yaml dump error in representer: ",
		},
		{
			name: "explicit plugin after defaults wins",
			opts: []yaml.Option{yaml.WithV3Defaults(), yaml.WithPlugin(errfmtv4.Must())},
			want: "go-yaml dump error in representer: ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := yaml.Dump(make(chan int), tt.opts...)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !strings.HasPrefix(err.Error(), tt.want) {
				t.Errorf("got %q, want prefix %q", err.Error(), tt.want)
			}
		})
	}
}
