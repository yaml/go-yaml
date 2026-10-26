// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
)

func TestRegisterCompiledPlugins(t *testing.T) {
	saved := compiledPluginRegistrations
	t.Cleanup(func() { compiledPluginRegistrations = saved })
	want := errors.New("registration failed")
	called := false
	compiledPluginRegistrations = []func() error{
		func() error { return nil },
		func() error { return want },
		func() error {
			called = true
			return nil
		},
	}
	if err := registerCompiledPlugins(); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
	if called {
		t.Fatal("registration continued after an error")
	}
}

func TestEmbeddedOptions(t *testing.T) {
	saved := defaultConfig
	t.Cleanup(func() { defaultConfig = saved })
	defaultConfig = "indent: 4\nplugin:\n  loader-limits:\n    depth: 1\n"
	initOptionRegistry()
	opts, err := buildOptions("", nil)
	if err != nil {
		t.Fatal(err)
	}
	options, err := libyaml.ApplyOptions(opts...)
	if err != nil || options.Indent != 4 {
		t.Fatalf("embedded options: %v, %v", options, err)
	}
	var value any
	if err := yaml.Load([]byte("[[]]"), &value, opts...); err == nil {
		t.Fatal("embedded limit not applied")
	}

	opts, err = buildOptions("", []string{"indent=2"}, "loader-limits")
	if err != nil {
		t.Fatal(err)
	}
	options, err = libyaml.ApplyOptions(opts...)
	if err != nil || options.Indent != 2 {
		t.Fatalf("flag options: %v, %v", options, err)
	}
	if err := yaml.Load([]byte("[[]]"), &value, opts...); err != nil {
		t.Fatalf("explicit limit should use defaults: %v", err)
	}
	opts, err = buildOptions("", nil, "loader-limits=loader-limits")
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Load([]byte("[[]]"), &value, opts...); err != nil {
		t.Fatalf("explicit implementation should use defaults: %v", err)
	}
	if _, err := buildOptions("", nil, "limit=limit"); err != nil {
		t.Fatalf("legacy limit selector: %v", err)
	}
	if _, err := buildOptions("", nil, "parser=go-yaml"); err == nil {
		t.Fatal("old parser selector accepted")
	}
	for _, selector := range []string{"", "=loader-limits", "loader-limits=", "a=b=c"} {
		if _, err := buildOptions("", nil, selector); err == nil {
			t.Fatalf("invalid selector %q accepted", selector)
		}
	}
	defaultConfig = "plugin: {limit: true, loader-limits: true}\n"
	if _, err := buildOptions("", nil); err == nil {
		t.Fatal("legacy and canonical loader-limits APIs accepted together")
	}
	config := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(config, []byte("indent: 6\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts, err = buildOptions(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	options, err = libyaml.ApplyOptions(opts...)
	if err != nil || options.Indent != 6 {
		t.Fatalf("runtime file options: %v, %v", options, err)
	}
	if err := yaml.Load([]byte("[[]]"), &value, opts...); err != nil {
		t.Fatalf("runtime config did not replace embedded limits: %v", err)
	}
}

func TestErrorFormatOptions(t *testing.T) {
	saved := defaultConfig
	t.Cleanup(func() { defaultConfig = saved })
	defaultConfig = "plugin: {error-format: {v3: {}}}\n"
	for _, tc := range []struct {
		name, selector, want string
		invalid              bool
	}{
		{"embedded v3", "", "yaml: line 1:", false},
		{"v3 shorthand", "error-format=v3", "yaml: line 1:", false},
		{"v4 shorthand", "error-format=v4", "go-yaml load error in scanner", false},
		{"default", "error-format", "go-yaml load error in scanner", false},
		{
			"explicit implementation", "error-format=error-format",
			"go-yaml load error in scanner", false,
		},
		{"old API", "errfmt", "", true},
		{"old implementation", "error-format=errfmt", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var selectors []string
			if tc.selector != "" {
				selectors = append(selectors, tc.selector)
			}
			opts, err := buildOptions("", nil, selectors...)
			if tc.invalid {
				if err == nil {
					t.Fatal("old plugin name accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var value any
			err = yaml.Load([]byte("value: -\n"), &value, opts...)
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("got %v, want prefix %q", err, tc.want)
			}
		})
	}
}

func TestMergeKeyOptions(t *testing.T) {
	initOptionRegistry()
	for _, tc := range []struct {
		selector string
		input    string
		key      string
	}{
		{"merge-key", "<<: {x: 1}", "x"},
		{"merge-key=disable", "<<: {x: 1}", "<<"},
	} {
		opts, err := buildOptions("", nil, tc.selector)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := yaml.Load([]byte(tc.input), &got, opts...); err != nil {
			t.Fatal(err)
		}
		if _, found := got[tc.key]; !found {
			t.Fatalf("%s: %#v", tc.selector, got)
		}
	}
}
