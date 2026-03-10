// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"testing"

	"go.yaml.in/yaml/v4"
	v4comment "go.yaml.in/yaml/v4/plugin/comment/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
	"go.yaml.in/yaml/v4/internal/testutil/datatest"
)

func TestCommentV4(t *testing.T) {
	datatest.RunTestCases(t, func() ([]map[string]any, error) {
		return datatest.LoadTestCasesFromFile("testdata/comment-v4.yaml", libyaml.LoadAny)
	}, map[string]datatest.TestHandler{
		"comment-v4-roundtrip": runCommentV4Roundtrip,
		"comment-v4-node":      runCommentV4Node,
		"comment-v4-error":     runCommentV4Error,
	})
}

func runCommentV4Roundtrip(t *testing.T, tc map[string]any) {
	t.Helper()

	from, ok := tc["from"].(string)
	if !ok {
		t.Fatal("missing 'from' field")
	}

	want := from
	if w, ok := tc["want"].(string); ok {
		want = w
	}

	plugin, err := v4comment.New(v4comment.RC("rc1"))
	if err != nil {
		t.Fatalf("Failed to create v4 plugin: %v", err)
	}

	// Load into node tree
	loader, err := yaml.NewLoader(
		bytes.NewReader([]byte(from)),
		yaml.WithPlugin(plugin),
	)
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Dump back to YAML
	var buf bytes.Buffer
	dumper, err := yaml.NewDumper(&buf)
	if err != nil {
		t.Fatalf("NewDumper failed: %v", err)
	}
	if err := dumper.Dump(&node); err != nil {
		t.Fatalf("Dump failed: %v", err)
	}
	if err := dumper.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	assert.Equal(t, want, buf.String())
}

func runCommentV4Node(t *testing.T, tc map[string]any) {
	t.Helper()

	from, ok := tc["from"].(string)
	if !ok {
		t.Fatal("missing 'from' field")
	}

	check, ok := tc["check"].(map[string]any)
	if !ok {
		t.Fatal("missing 'check' field")
	}

	plugin, err := v4comment.New(v4comment.RC("rc1"))
	if err != nil {
		t.Fatalf("Failed to create v4 plugin: %v", err)
	}

	loader, err := yaml.NewLoader(
		bytes.NewReader([]byte(from)),
		yaml.WithPlugin(plugin),
	)
	if err != nil {
		t.Fatalf("NewLoader failed: %v", err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Navigate to the target node via path
	target := &node
	if path, ok := check["path"].([]any); ok {
		for _, idx := range path {
			i, ok := idx.(int)
			if !ok {
				t.Fatalf("path index must be int, got %T", idx)
			}
			if target.Kind == yaml.DocumentNode || target.Kind == yaml.MappingNode || target.Kind == yaml.SequenceNode {
				if i >= len(target.Content) {
					t.Fatalf("path index %d out of range (len=%d)", i, len(target.Content))
				}
				target = target.Content[i]
			} else {
				t.Fatalf("cannot index into node kind %d", target.Kind)
			}
		}
	}

	if expected, ok := check["head"].(string); ok {
		assert.Equal(t, expected, target.HeadComment)
	}
	if expected, ok := check["line"].(string); ok {
		assert.Equal(t, expected, target.LineComment)
	}
	if expected, ok := check["foot"].(string); ok {
		assert.Equal(t, expected, target.FootComment)
	}
}

func runCommentV4Error(t *testing.T, tc map[string]any) {
	t.Helper()

	wantErr, ok := tc["want"].(string)
	if !ok {
		t.Fatal("missing 'want' error string")
	}

	// Build plugin config via OptsYAML
	optsCfg := tc["opts"]
	if optsCfg == nil {
		t.Fatal("missing 'opts' field")
	}

	optsYAML, err := yaml.Dump(optsCfg)
	if err != nil {
		t.Fatalf("Failed to marshal opts: %v", err)
	}

	_, err = yaml.OptsYAML(string(optsYAML))
	if err == nil {
		t.Fatalf("Expected error %q, got nil", wantErr)
	}
	assert.Equal(t, wantErr, err.Error())
}
