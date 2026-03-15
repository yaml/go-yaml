// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for isYAMLNodePkg and the Node type allowlist.

package libyaml

import (
	"reflect"
	"testing"
)

func TestIsYAMLNodePkg(t *testing.T) {
	tests := []struct {
		pkg  string
		want bool
	}{
		{"gopkg.in/yaml.v3", true},
		{"go.yaml.in/yaml/v3", true},
		{"example.com/mypkg", false},
		{"gopkg.in/yaml.v2", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isYAMLNodePkg(tt.pkg); got != tt.want {
			t.Errorf("isYAMLNodePkg(%q) = %v, want %v",
				tt.pkg, got, tt.want)
		}
	}
}

// foreignNode is a user-defined type that happens to be named "Node"
// but is NOT from a known yaml package. The allowlist must reject it.
type foreignNode struct {
	Bogus int
}

// foreignNodeReceiver has an UnmarshalYAML method that takes a
// *foreignNode — this must NOT be treated as a yaml.Unmarshaler.
type foreignNodeReceiver struct {
	Value string
}

func (f *foreignNodeReceiver) UnmarshalYAML(n *foreignNode) error {
	f.Value = "should not be called"
	return nil
}

// yamlNodeReceiver has an UnmarshalYAML method that takes the real
// *Node from this package — this MUST be accepted.
type yamlNodeReceiver struct {
	Value string
}

func (y *yamlNodeReceiver) UnmarshalYAML(n *Node) error {
	if n.Value != "" {
		y.Value = n.Value
	}
	return nil
}

func TestForeignNodeRejected(t *testing.T) {
	// hasConstructYAMLMethod must return false for foreignNodeReceiver.
	ft := reflect.TypeOf(foreignNodeReceiver{})
	if hasConstructYAMLMethod(reflect.PointerTo(ft)) {
		t.Error("hasConstructYAMLMethod accepted foreignNodeReceiver")
	}
}

func TestRealNodeNotInAllowlist(t *testing.T) {
	// v4 types (this package) are no longer in the allowlist — they use
	// the native constructor interface instead of the unsafe cast path.
	yt := reflect.TypeOf(yamlNodeReceiver{})
	if hasConstructYAMLMethod(reflect.PointerTo(yt)) {
		t.Error("hasConstructYAMLMethod should not match v4 types")
	}
}

func TestForeignNodeUnmarshal(t *testing.T) {
	// Unmarshaling into foreignNodeReceiver must silently skip the
	// UnmarshalYAML method (not panic or corrupt memory).
	var f foreignNodeReceiver
	err := Load([]byte("hello"), &f)
	if err == nil {
		t.Fatal("expected error unmarshaling scalar into struct")
	}
	if f.Value == "should not be called" {
		t.Error("foreign UnmarshalYAML was incorrectly called")
	}
}

func TestRealNodeUnmarshal(t *testing.T) {
	// Unmarshaling into yamlNodeReceiver must call UnmarshalYAML.
	var y yamlNodeReceiver
	err := Load([]byte("hello"), &y)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if y.Value != "hello" {
		t.Errorf("Value = %q, want %q", y.Value, "hello")
	}
}
