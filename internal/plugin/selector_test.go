// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"reflect"
	"testing"
)

func TestParseSelectors(t *testing.T) {
	got, err := ParseSelectors(
		"yaml-parser=reference@v0.2.5,json-comments")
	if err != nil {
		t.Fatal(err)
	}
	want := []Selection{
		{API: "yaml-parser", Name: "reference", Version: "v0.2.5"},
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
	got, err = ParseSelectors("yaml-parser=reference@0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	want = []Selection{
		{API: "yaml-parser", Name: "reference", Version: "0.2.5"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for _, text := range []string{
		"", ",", "yaml-parser=", "@v0.2.5",
		"yaml-parser=@v0.2.5", "yaml-parser=reference@",
		"yaml-parser=reference@latest", "yaml-parser==reference",
		"yaml-parser=reference@@v0.2.5",
	} {
		if _, err := ParseSelectors(text); err == nil {
			t.Fatalf("accepted %q", text)
		}
	}
}

func TestConfigValue(t *testing.T) {
	got, err := ConfigValue("yaml-parser", "reference@v0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"name": "reference", "version": "v0.2.5"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for _, tc := range []struct {
		api, value string
	}{
		{"", "reference"},
		{"yaml-parser", ""},
		{"yaml-parser", "reference@"},
		{"yaml-parser", "reference@latest"},
		{"yaml-parser=other", "reference"},
	} {
		if _, err := ConfigValue(tc.api, tc.value); err == nil {
			t.Fatalf("accepted API %q with value %q", tc.api, tc.value)
		}
	}
}

func TestLoaderLimitsAliases(t *testing.T) {
	got, err := ParseSelectors("limit=limit")
	if err != nil {
		t.Fatal(err)
	}
	want := []Selection{{API: "loader-limits", Name: "loader-limits"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}

	plugins := map[string]any{"limit": true}
	if err := NormalizeConfig(plugins); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plugins, map[string]any{"loader-limits": true}) {
		t.Fatalf("got %#v", plugins)
	}
	if err := NormalizeConfig(map[string]any{
		"limit": true, "loader-limits": true,
	}); err == nil {
		t.Fatal("accepted both legacy and canonical loader-limits APIs")
	}
}
