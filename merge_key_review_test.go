// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"context"
	"errors"
	"math"
	"os"
	"os/exec"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/loaderlimits"
	"go.yaml.in/yaml/v4/plugin/mergekey"
)

func TestMergeKeyCopiedAliasCycle(t *testing.T) {
	if os.Getenv("GO_YAML_TEST_MERGE_CYCLE") == "1" {
		// A broken cycle guard must fail this child, not exhaust the parent.
		debug.SetMaxStack(256 << 10)
		for _, input := range []string{
			"&root {<<: [&base {self: *root}, *base]}",
			"&root {<<: [&base {self: *root}, &middle {<<: *base}, *middle]}",
		} {
			var out any
			err := yaml.Load([]byte(input), &out, yaml.WithPlugin(
				mergekey.New(mergekey.WithConflicts("last")),
				loaderlimits.New(loaderlimits.AliasNone())))
			if err == nil || (!strings.Contains(err.Error(), "contains itself") &&
				!strings.Contains(err.Error(), "cyclic merge alias")) {
				t.Fatalf("expected a cycle error with limits disabled, got %v", err)
			}
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestMergeKeyCopiedAliasCycle$")
	cmd.Env = append(os.Environ(), "GO_YAML_TEST_MERGE_CYCLE=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 1000 {
			output = output[:1000]
		}
		t.Fatalf("cycle check failed: %v: %s", err, output)
	}
}

func TestMergeKeyNaNSourceKeys(t *testing.T) {
	for _, policy := range []string{"first", "last", "error"} {
		t.Run(policy, func(t *testing.T) {
			var got map[float64]int
			err := yaml.Load([]byte(".nan: 0\n<<: [{.nan: 1}, {.nan: 2}]"),
				&got, yaml.WithPlugin(mergekey.New(mergekey.WithConflicts(policy))))
			if err != nil {
				t.Fatal(err)
			}
			values := map[int]bool{}
			for key, value := range got {
				if !math.IsNaN(key) {
					t.Fatalf("unexpected key %v", key)
				}
				values[value] = true
			}
			if len(got) != 3 || !reflect.DeepEqual(values,
				map[int]bool{0: true, 1: true, 2: true}) {
				t.Fatalf("NaN entries lost: %#v", got)
			}
		})
	}
}

type mergeInlineCapture struct {
	Raw map[string]any
}

func (v *mergeInlineCapture) UnmarshalYAML(n *yaml.Node) error {
	return n.Load(&v.Raw, yaml.WithPlugin(
		mergekey.New(mergekey.WithDisable(true))))
}

func TestMergeKeyInlineOriginalNode(t *testing.T) {
	var got struct {
		X, Y   int
		Inline mergeInlineCapture `yaml:",inline"`
	}
	if err := yaml.Load([]byte("<<: {x: 1}\ny: 2"), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"<<": map[string]any{"x": 1}, "y": 2}
	if got.X != 1 || got.Y != 2 || !reflect.DeepEqual(got.Inline.Raw, want) {
		t.Fatalf("inline hook lost the original mapping: %#v", got)
	}
}

func TestMergeKeyPartialTypedKeys(t *testing.T) {
	var got map[int]string
	err := yaml.Load([]byte("1: one\nbad: invalid\n<<:\n"+
		"  2: two\n  nope: invalid\n  3: three\n"), &got)
	want := map[int]string{1: "one", 2: "two", 3: "three"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("partial result: got %#v, want %#v", got, want)
	}
	var failures *yaml.LoadErrors
	if !errors.As(err, &failures) || len(failures.Errors) != 2 {
		t.Fatalf("expected two recoverable key errors, got %v", err)
	}
	if failures.Errors[0].Mark.Line != 2 || failures.Errors[1].Mark.Line != 5 {
		t.Fatalf("key errors lost their locations: %v", failures)
	}
}

func TestMergeKeyConflictAfterInvalidKey(t *testing.T) {
	var got map[int]string
	err := yaml.Load([]byte("1: target\n<<:\n  invalid: skip\n  1: source\n"),
		&got, yaml.WithPlugin(mergekey.New(mergekey.WithConflicts("error"))))
	var failure *yaml.LoadError
	if !errors.As(err, &failure) || failure.Mark.Line != 4 ||
		!strings.Contains(failure.Message, "conflicting key") {
		t.Fatalf("conflict lost its location after a skipped key: %v", err)
	}
}

func TestMergeKeySkipsUnconstructibleNullKeys(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  map[int]string
	}{
		{"null: skip\n1: one\n", map[int]string{1: "one"}},
		{
			"null: skip\n1: one\n<<: {null: skip, 2: two}\n",
			map[int]string{1: "one", 2: "two"},
		},
	} {
		var got map[int]string
		if err := yaml.Load([]byte(tc.input), &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("got %#v, want %#v", got, tc.want)
		}
	}
}
