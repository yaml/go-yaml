// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestUnmarshalOmitZeroFlag(t *testing.T) {
	type T struct {
		A string `yaml:"a,omitzero"`
		B int    `yaml:"b,omitzero"`
	}
	var v T
	if err := yaml.Unmarshal([]byte("a: hi\nb: 2\n"), &v); err != nil {
		t.Fatal(err)
	}
	if v.A != "hi" || v.B != 2 {
		t.Fatalf("got %+v", v)
	}
}

func TestMarshalOmitZeroFlag(t *testing.T) {
	type T struct {
		A string `yaml:"a,omitzero"`
		B int    `yaml:"b,omitzero"`
	}
	out, err := yaml.Marshal(T{})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "{}\n" {
		t.Fatalf("got %q", out)
	}
	out, err = yaml.Marshal(T{A: "hi", B: 2})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "a: hi\nb: 2\n" {
		t.Fatalf("got %q", out)
	}
}
