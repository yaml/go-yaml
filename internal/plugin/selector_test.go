// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"reflect"
	"testing"
)

func TestParseSelectors(t *testing.T) {
	got, err := ParseSelectors(
		"yaml-parser=reference@v0.2.5,loader-limits")
	if err != nil {
		t.Fatal(err)
	}
	want := []Selection{
		{API: "yaml-parser", Name: "reference", Version: "v0.2.5"},
		{API: "loader-limits"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	got, err = ParseSelectors("loader-limits@1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	want = []Selection{{API: "loader-limits", Version: "1.2.3"}}
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

func TestSelectorsPreservePluginNames(t *testing.T) {
	got, err := ParseSelectors("limit=limit,loader-limits=limit")
	if err != nil {
		t.Fatal(err)
	}
	want := []Selection{
		{API: "limit", Name: "limit"},
		{API: "loader-limits", Name: "limit"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	config, err := ConfigValue("loader-limits", "limit")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(config, map[string]any{"name": "limit"}) {
		t.Fatalf("got %#v", config)
	}
}

func TestRegistryPreservesPluginNames(t *testing.T) {
	registry := NewRegistry()
	for _, tc := range []struct {
		api, name, value string
	}{
		{LoaderLimitsAPI, LoaderLimitsAPI, "default"},
		{"limit", "limit", "custom API"},
		{LoaderLimitsAPI, "limit", "custom implementation"},
	} {
		value := tc.value
		if err := registry.Register(Registration{
			API: tc.api, Name: tc.name,
			Factory: func(map[string]any) (any, error) { return value, nil },
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		api, name, want string
	}{
		{LoaderLimitsAPI, LoaderLimitsAPI, "default"},
		{"limit", "limit", "custom API"},
		{LoaderLimitsAPI, "limit", "custom implementation"},
	} {
		got, err := registry.Create(tc.api, map[string]any{"name": tc.name})
		if err != nil || got != tc.want {
			t.Fatalf("%s=%s: got %v, %v; want %q", tc.api, tc.name, got, err, tc.want)
		}
	}
}
