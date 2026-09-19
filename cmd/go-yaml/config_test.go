// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
)

func TestEmbeddedOptions(t *testing.T) {
	saved := defaultConfig
	t.Cleanup(func() { defaultConfig = saved })
	defaultConfig = "indent: 4\nplugin:\n  limit:\n    depth: 1\n"
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

	opts, err = buildOptions("", []string{"indent=2"}, "limit")
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
	opts, err = buildOptions("", nil, "limit=limit")
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Load([]byte("[[]]"), &value, opts...); err != nil {
		t.Fatalf("explicit implementation should use defaults: %v", err)
	}
	for _, selector := range []string{"", "=limit", "limit=", "a=b=c"} {
		if _, err := buildOptions("", nil, selector); err == nil {
			t.Fatalf("invalid selector %q accepted", selector)
		}
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
