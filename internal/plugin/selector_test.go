// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"reflect"
	"testing"
)

func TestParseSelectors(t *testing.T) {
	got, err := ParseSelectors(
		"parser=reference@v0.2.5,json-comments")
	if err != nil {
		t.Fatal(err)
	}
	want := []Selection{
		{API: "parser", Name: "reference", Version: "v0.2.5"},
		{API: "json-comments"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	got, err = ParseSelectors("json-comments@0.1.9")
	if err != nil {
		t.Fatal(err)
	}
	want = []Selection{{API: "json-comments", Version: "0.1.9"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for _, text := range []string{
		"", ",", "parser=", "@v0.2.5", "parser=@v0.2.5",
		"parser=reference@latest", "parser==reference",
	} {
		if _, err := ParseSelectors(text); err == nil {
			t.Fatalf("accepted %q", text)
		}
	}
}

func TestConfigValue(t *testing.T) {
	got, err := ConfigValue("parser", "reference@v0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"name": "reference", "version": "v0.2.5"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
