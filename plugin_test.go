// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"go.yaml.in/yaml/v4"
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

type registeredSourceFunc func([]byte) ([]yaml.PluginEvent, error)

func (f registeredSourceFunc) Parse(input []byte) ([]yaml.PluginEvent, error) {
	return f(input)
}

func registeredScalarStream(value string) []yaml.PluginEvent {
	return []yaml.PluginEvent{
		{Type: "stream_start"},
		{Type: "document_start"},
		{Type: "scalar", Value: value},
		{Type: "document_end"},
		{Type: "stream_end"},
	}
}

var registryTestID uint64

func TestPluginRegistry(t *testing.T) {
	api := fmt.Sprintf("test-source-%d", atomic.AddUint64(&registryTestID, 1))
	factory := func(cfg map[string]any) (any, error) {
		for _, key := range []string{"disable", "name", "version"} {
			if _, found := cfg[key]; found {
				return nil, fmt.Errorf("%s leaked to plugin factory", key)
			}
		}
		value, _ := cfg["value"].(string)
		return registeredSourceFunc(func([]byte) ([]yaml.PluginEvent, error) {
			return registeredScalarStream(value), nil
		}), nil
	}
	registration := yaml.PluginRegistration{
		API: api, Name: api, Version: "v0.1.8", Factory: factory,
	}
	if err := yaml.RegisterPlugin(registration); err != nil {
		t.Fatal(err)
	}
	if err := yaml.RegisterPlugin(registration); err == nil {
		t.Fatal("duplicate accepted")
	}
	if err := yaml.RegisterPlugin(yaml.PluginRegistration{
		API: api, Name: "alternate", Version: "0.1.8", Factory: factory,
	}); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []yaml.PluginRegistration{
		{Name: "name", Factory: factory},
		{API: api + "-name", Factory: factory},
		{API: api + "-factory", Name: "name"},
		{API: api + "-version", Name: "name", Version: "latest", Factory: factory},
	} {
		if err := yaml.RegisterPlugin(invalid); err == nil {
			t.Fatalf("invalid registration accepted: %+v", invalid)
		}
	}
	opt, err := yaml.OptsYAML(fmt.Sprintf(
		"plugin:\n  %s:\n    name: alternate\n    version: 0.1.8\n"+
			"    value: configured\n    disable: false\n", api))
	if err != nil {
		t.Fatal(err)
	}
	var value string
	if err := yaml.Load(nil, &value, opt); err != nil || value != "configured" {
		t.Fatalf("%q, %v", value, err)
	}
	opt, err = yaml.OptsYAML(fmt.Sprintf(
		"plugin: {%s: {name: absent, version: bad, value: ignored, disable: true}}",
		api))
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Load([]byte("plain"), &value, opt); err != nil ||
		value != "plain" {
		t.Fatalf("disabled plugin: %q, %v", value, err)
	}
	if _, err := yaml.OptsYAML(fmt.Sprintf(
		"plugin: {%s: {version: 0.1.7}}", api)); err == nil ||
		!strings.Contains(err.Error(), "linked 0.1.8, requested 0.1.7") {
		t.Fatalf("version mismatch: %v", err)
	}
	if _, err := yaml.OptsYAML(fmt.Sprintf(
		"plugin: {%s: {name: missing}}", api)); err == nil ||
		!strings.Contains(err.Error(), "unknown implementation") {
		t.Fatalf("unknown implementation: %v", err)
	}
	for _, setting := range []string{
		"name: null", "name: 12", "version: null", "version: latest",
	} {
		if _, err := yaml.OptsYAML(fmt.Sprintf(
			"plugin: {%s: {%s}}", api, setting)); err == nil {
			t.Fatalf("invalid host setting accepted: %s", setting)
		}
	}
	if _, err := yaml.OptsYAML(
		"plugin: {limit: {version: 0.1.8}}"); err == nil ||
		!strings.Contains(err.Error(), "unversioned") {
		t.Fatalf("unversioned implementation: %v", err)
	}
	if err := yaml.Load(nil, &value,
		yaml.WithNamedPlugin("missing")); err == nil {
		t.Fatal("unknown name accepted")
	}
	if err := yaml.Load(nil, &value, yaml.WithNamedPlugin(api)); err != nil {
		t.Fatal(err)
	}
	p, _ := factory(nil)
	if err := yaml.Load(nil, &value, yaml.WithPlugin(p, p)); err == nil {
		t.Fatal("multiple event sources accepted")
	}
	if err := yaml.Load(nil, &value,
		yaml.WithPlugin(p, limit.New())); err != nil {
		t.Fatal(err)
	}
}
