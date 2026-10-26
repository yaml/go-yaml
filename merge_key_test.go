// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/loaderlimits"
	"go.yaml.in/yaml/v4/plugin/mergekey"
)

func TestMergeKeyData(t *testing.T) {
	data, err := os.ReadFile("testdata/merge-key.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, Config, YAML, Want, Error string
	}
	if err := yaml.Load(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			config := tc.Config
			if config == "" {
				config = "{}"
			}
			opts, err := yaml.OptsYAML(config)
			if err != nil {
				t.Fatal(err)
			}
			var got any
			err = yaml.Load([]byte(tc.YAML), &got, opts)
			if tc.Error != "" {
				if err == nil || !strings.Contains(err.Error(), tc.Error) {
					t.Fatalf("got %v, want error containing %q", err, tc.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var want any
			if err := yaml.Load([]byte(tc.Want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}
		})
	}
}

// computedMerge demonstrates a replacement beyond first/last selection.
type computedMerge struct{ *mergekey.Plugin }

func (p computedMerge) MergeMappings(in yaml.MergeKeyInput) (*yaml.Node, error) {
	return &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "answer"},
		{Kind: yaml.ScalarNode, Tag: "!!int", Value: "42"},
	}}, nil
}

var registerComputedMerge sync.Once

func TestMergeKeyReplacement(t *testing.T) {
	registerComputedMerge.Do(func() {
		if err := yaml.RegisterPlugin(yaml.PluginRegistration{
			API: "merge-key", Name: "computed", Version: "1.0.0",
			Factory: func(map[string]any) (any, error) {
				return computedMerge{mergekey.New()}, nil
			},
		}); err != nil {
			t.Fatal(err)
		}
	})
	opts, err := yaml.OptsYAML("plugin: {merge-key: computed@1.0.0}")
	if err != nil {
		t.Fatal(err)
	}
	var got struct{ Answer int }
	if err := yaml.Load([]byte("<<: {x: 1}"), &got, opts); err != nil {
		t.Fatal(err)
	}
	if got.Answer != 42 {
		t.Fatalf("algorithm not called: %#v", got)
	}
}

func TestMergeKeyNodeAndTypedDestinations(t *testing.T) {
	var n yaml.Node
	if err := yaml.Load([]byte("<<: {x: 1}\ny: 2\n"), &n); err != nil {
		t.Fatal(err)
	}
	before, err := yaml.Dump(&n)
	if err != nil {
		t.Fatal(err)
	}
	var plain map[string]any
	if err := n.Load(&plain, yaml.WithPlugin(
		mergekey.New(mergekey.WithDisable(true)))); err != nil {
		t.Fatal(err)
	}
	if _, found := plain["<<"]; !found {
		t.Fatal("disabled Node.Load did not preserve the key")
	}
	var typed struct{ X, Y, Z int }
	typed.Z = 3
	if err := n.Load(&typed); err != nil {
		t.Fatal(err)
	}
	if typed.X != 1 || typed.Y != 2 || typed.Z != 3 {
		t.Fatalf("typed construction: %#v", typed)
	}
	after, err := yaml.Dump(&n)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("node graph changed: %s, %v", after, err)
	}
	var ints map[int]string
	if err := yaml.Load([]byte("<<: {1: first}\n1: target"), &ints); err != nil {
		t.Fatal(err)
	}
	if ints[1] != "target" {
		t.Fatalf("typed keys: %#v", ints)
	}
}

func TestMergeKeyStreamConcurrentAndLimits(t *testing.T) {
	p := mergekey.New(mergekey.WithConflicts("last"))
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			loader, err := yaml.NewLoader(strings.NewReader(
				"<<: {x: 1}\nx: 2\n---\n<<: {x: 3}\n"), yaml.WithPlugin(p))
			if err != nil {
				t.Error(err)
				return
			}
			for _, want := range []int{1, 3} {
				var got map[string]int
				if err := loader.Load(&got); err != nil || got["x"] != want {
					t.Errorf("stream: %#v, %v", got, err)
				}
			}
			var end any
			if err := loader.Load(&end); !errors.Is(err, io.EOF) {
				t.Errorf("end of stream: %v", err)
			}
		}()
	}
	workers.Wait()
	var out any
	if err := yaml.Load([]byte("a: &a {x: 1}\nb: {<<: *a}"), &out,
		yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasValue(0)))); err == nil {
		t.Fatal("merge alias bypassed alias limits")
	}
}

func TestMergeKeyInvalidConfiguration(t *testing.T) {
	for _, config := range []string{
		"{multiple: last}", "{conflicts: combine}", "{disable: wrong}",
		"{implicit: false}", "{multiple: true}", "{conflicts: false}",
	} {
		if _, err := yaml.OptsYAML("plugin: {merge-key: " + config + "}"); err == nil {
			t.Errorf("accepted %s", config)
		}
	}
	var out any
	if err := yaml.Load([]byte("{}"), &out,
		yaml.WithPlugin(mergekey.New(), mergekey.New())); err == nil {
		t.Fatal("multiple implementations accepted")
	}
}

type resultMerge struct {
	*mergekey.Plugin
	result *yaml.Node
	err    error
}

func (p resultMerge) MergeMappings(yaml.MergeKeyInput) (*yaml.Node, error) {
	return p.result, p.err
}

func TestMergeKeyAlgorithmFailures(t *testing.T) {
	for _, result := range []*yaml.Node{
		nil,
		{Kind: yaml.ScalarNode, Value: "invalid"},
		{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode}}},
		{Kind: yaml.MappingNode, Content: []*yaml.Node{nil, nil}},
	} {
		var out any
		err := yaml.Load([]byte("<<: {}"), &out,
			yaml.WithPlugin(resultMerge{Plugin: mergekey.New(), result: result}))
		var load *yaml.LoadError
		if !errors.As(err, &load) || load.Stage != "constructor" {
			t.Fatalf("invalid result %#v: %v", result, err)
		}
	}
	want := errors.New("custom merge failure")
	var out any
	err := yaml.Load([]byte("<<: {}"), &out,
		yaml.WithPlugin(resultMerge{Plugin: mergekey.New(), err: want}))
	if !errors.Is(err, want) {
		t.Fatalf("lost plugin error: %v", err)
	}
}

func TestMergeKeyInheritedAliasAccounting(t *testing.T) {
	base := "base: &base {items: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]}\n"
	limits := yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasValue(8)))
	var out any
	if err := yaml.Load([]byte(base+"item: {<<: *base}\n"), &out, limits); err == nil {
		t.Fatal("inherited sequence bypassed alias accounting")
	}
	if err := yaml.Load([]byte(base+"item: {items: [], <<: *base}\n"), &out, limits); err != nil {
		t.Fatalf("overridden value should not be expanded: %v", err)
	}
}
