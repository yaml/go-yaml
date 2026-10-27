// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	yaml "go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/aliasdata"
	"go.yaml.in/yaml/v4/plugin/functiontag"
	"go.yaml.in/yaml/v4/plugin/yamlrules"
)

func loadLibraryOptions(t *testing.T, stream bool, functions functiontag.Functions) []yaml.Option {
	t.Helper()
	aliases, err := aliasdata.New(aliasdata.WithPath(true), aliasdata.WithStream(stream))
	if err != nil {
		t.Fatal(err)
	}
	tags, err := functiontag.New(functions, map[string]any{"aliases": "alias-data"})
	if err != nil {
		t.Fatal(err)
	}
	return []yaml.Option{yaml.WithPlugin(aliases), yaml.WithPlugin(tags)}
}

func importFile(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vars.yaml")
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%q", path)
}

func TestFunctionLoadAliases(t *testing.T) {
	path := importFile(t, "bar: 42\nhash2: {inherited: true, key: base}\n")
	input := "!load vars: " + path + "\nfoo: *vars.bar\nmap1:\n  <<: *vars.hash2\n  key: value\n"
	options := loadLibraryOptions(t, false, nil)
	var got map[string]any
	if err := yaml.Load([]byte(input), &got, options...); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"foo": 42, "map1": map[string]any{"inherited": true, "key": "value"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	var typed struct {
		Foo  int
		Map1 map[string]any
	}
	if err := yaml.Load([]byte(input), &typed, options...); err != nil {
		t.Fatal(err)
	}
	if typed.Foo != 42 || typed.Map1["inherited"] != true {
		t.Fatalf("%#v", typed)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var result map[string]any
			if err := yaml.Load([]byte(input), &result, options...); err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("%#v", result)
			}
		}()
	}
	wg.Wait()
}

func TestFunctionContextAndLazyValue(t *testing.T) {
	calls := 0
	options := loadLibraryOptions(t, false, functiontag.Functions{
		"filename": func(value string) string { calls++; return value },
		"define": func(c yaml.FunctionTagContext, name string) (any, error) {
			if c.Role != "key" || c.PairKey == nil || c.PairValue == nil {
				t.Fatal("missing pair context")
			}
			value, err := c.ConstructValue()
			if err != nil {
				return nil, err
			}
			again, err := c.ConstructValue()
			if err != nil || value != again {
				t.Fatal("value was not cached")
			}
			if err := c.DefineAlias(name, map[string]any{"value": value}); err != nil {
				return nil, err
			}
			return yaml.Omit, nil
		},
		"sum": func(c yaml.FunctionTagContext, values ...int) int {
			if c.Role != "value" {
				t.Fatal(c.Role)
			}
			total := 0
			for _, value := range values {
				total += value
			}
			return total
		},
	})
	var got map[string]any
	input := "!define vars: !filename data\nfoo: *vars.value\nzero: !sum* []\nsum: !sum* [1, 2]\n"
	if err := yaml.Load([]byte(input), &got, options...); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || got["foo"] != "data" || got["zero"] != 0 || got["sum"] != 3 {
		t.Fatalf("%#v, calls %d", got, calls)
	}
}

func TestFunctionLoadRootsAndRules(t *testing.T) {
	for _, data := range []string{"null\n", "false\n", "42\n", "text\n", "[a, b]\n", "{a: 1}\n"} {
		t.Run(strings.TrimSpace(data), func(t *testing.T) {
			options := loadLibraryOptions(t, false, nil)
			var got map[string]any
			input := "!load vars: " + importFile(t, data) + "\nvalue: *vars\n"
			if err := yaml.Load([]byte(input), &got, options...); err != nil {
				t.Fatal(err)
			}
			var want any
			if err := yaml.Load([]byte(data), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got["value"], want) {
				t.Fatalf("%#v != %#v", got, want)
			}
		})
	}
	rules, err := yamlrules.New("failsafe", nil)
	if err != nil {
		t.Fatal(err)
	}
	options := append(loadLibraryOptions(t, false, nil), yaml.WithPlugin(rules))
	var got map[string]any
	input := "!load vars: " + importFile(t, "bar: false\n") + "\nfoo: *vars.bar\n"
	if err := yaml.Load([]byte(input), &got, options...); err != nil {
		t.Fatal(err)
	}
	if got["foo"] != "false" {
		t.Fatal(got)
	}
}

func TestFunctionLoadScopeAndErrors(t *testing.T) {
	path := importFile(t, "bar: 42\n")
	for _, stream := range []bool{false, true} {
		var got []any
		err := yaml.Load([]byte("!load vars: "+path+"\n---\nfoo: *vars.bar\n"), &got,
			append(loadLibraryOptions(t, stream, nil), yaml.WithAllDocuments())...)
		if stream && (err != nil || len(got) != 2) {
			t.Fatalf("%#v: %v", got, err)
		}
		if !stream && err == nil {
			t.Fatal("cross-document alias accepted")
		}
	}
	for _, input := range []string{
		"foo: *vars.bar\n!load vars: " + path,
		"!load bad.name: " + path,
		"!load false: " + path,
		"!load vars.yaml",
		"x: !load vars.yaml",
		"!load vars: false",
		"!load vars: ''",
		"!load vars: /no/such/function-tag-file.yaml",
		"!load vars: " + importFile(t, ""),
		"!load vars: " + importFile(t, "---\na\n---\nb\n"),
		"!load vars: " + importFile(t, "[bad"),
		"!load vars: " + path + "\nfoo: *vars.missing",
		"!load vars: " + path + "\nmap: {<<: *vars.bar}",
	} {
		var got any
		if err := yaml.Load([]byte(input), &got, loadLibraryOptions(t, false, nil)...); err == nil {
			t.Errorf("accepted %q: %#v", input, got)
		}
	}
	var got map[string]any
	if err := yaml.Load([]byte("!load vars: "+path), &got, loadLibraryOptions(t, false, nil)...); err != nil || len(got) != 0 {
		t.Fatalf("%#v: %v", got, err)
	}
}

func TestFunctionLoadPlainFiles(t *testing.T) {
	calls := 0
	options := loadLibraryOptions(t, false, functiontag.Functions{"effect": func(s string) string { calls++; return s }})
	var got any
	input := "!load vars: " + importFile(t, "!effect value\n") + "\nvalue: *vars\n"
	_ = yaml.Load([]byte(input), &got, options...)
	if calls != 0 {
		t.Fatal("import executed constructor")
	}
	var node yaml.Node
	input = "!load vars: " + importFile(t, "a: 1") + "\nfoo: *vars.a\n"
	if err := yaml.Load([]byte(input), &node, options...); err == nil {
		t.Fatal("inspection resolved dynamic alias")
	}
}

func TestFunctionLoadDefinitionOrder(t *testing.T) {
	first := importFile(t, "bar: first\n")
	second := importFile(t, "bar: second\n")
	options := loadLibraryOptions(t, false, nil)
	input := "!load vars: " + first + "\nbefore: *vars.bar\n!load vars: " + second + "\nafter: *vars.bar\nlocal: &vars {bar: local}\nlast: *vars.bar\n"
	var got map[string]any
	if err := yaml.Load([]byte(input), &got, options...); err != nil {
		t.Fatal(err)
	}
	if got["before"] != "first" || got["after"] != "second" || got["last"] != "local" {
		t.Fatal(got)
	}
}

func TestFunctionContextStreamResults(t *testing.T) {
	calls := 0
	options := loadLibraryOptions(t, true, functiontag.Functions{
		"result":  func(s string) map[string]any { calls++; return map[string]any{"bar": s} },
		"discard": func(s string) any { calls++; return yaml.Omit },
	})
	var got []any
	input := "a: &vars !result first\nb: &gone !discard ignored\n---\nc: *vars.bar\nd: *gone\n"
	if err := yaml.Load([]byte(input), &got, append(options, yaml.WithAllDocuments())...); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(got) != 2 {
		t.Fatalf("calls %d: %#v", calls, got)
	}
	if !reflect.DeepEqual(got[1], map[string]any{"c": "first"}) {
		t.Fatal(got)
	}
}

type loadOnlyResult struct{ marshals *int }

func (v *loadOnlyResult) MarshalYAML() (any, error) {
	*v.marshals++
	return "dumped", nil
}

func TestFunctionContextPreservesNativeResults(t *testing.T) {
	marshals := 0
	object := &loadOnlyResult{marshals: &marshals}
	channel := make(chan int)
	options := loadLibraryOptions(t, false, functiontag.Functions{
		"object":  func(string) any { return object },
		"channel": func(string) any { return channel },
		"map":     func(string) any { return map[string]any{"nested": map[string]any{"object": object, "value": 42}} },
	})
	var got map[string]any
	input := "a: &a !object ignored\nb: *a\nc: &c !channel ignored\nd: *c\ne: &e !map ignored\nf: *e.nested.object\ng: *e.nested.value\n"
	if err := yaml.Load([]byte(input), &got, options...); err != nil {
		t.Fatal(err)
	}
	if marshals != 0 || got["a"] != object || got["b"] != object || got["d"] != channel || got["f"] != object || got["g"] != 42 {
		t.Fatalf("marshals %d: %#v", marshals, got)
	}
}

func TestFunctionLoadSpread(t *testing.T) {
	input := "!load* [vars]: " + importFile(t, "bar: 42\n") + "\nfoo: *vars.bar\n"
	var got map[string]any
	if err := yaml.Load([]byte(input), &got, loadLibraryOptions(t, false, nil)...); err != nil {
		t.Fatal(err)
	}
	if got["foo"] != 42 {
		t.Fatal(got)
	}
}

func TestFunctionLoadExplicitBindings(t *testing.T) {
	path := importFile(t, "bar: 42\n")
	aliases, err := aliasdata.New(aliasdata.WithPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, namespace := range []bool{false, true} {
		t.Run(fmt.Sprintf("namespace=%t", namespace), func(t *testing.T) {
			bindings := functiontag.Functions{"import": aliasdata.Load}
			var libraries []map[string]any
			input := "!import vars: " + path + "\nfoo: *vars.bar\n"
			want := map[string]any{"foo": 42}
			if namespace {
				libraries = append(libraries, map[string]any{
					"math": functiontag.Functions{
						"add": func(a, b int) int { return a + b },
					},
				})
				input += "total: !math/add* [1, 2]\n"
				want["total"] = 3
			}
			tags, err := functiontag.New(bindings, libraries...)
			if err != nil {
				t.Fatal(err)
			}
			options := []yaml.Option{yaml.WithPlugin(aliases), yaml.WithPlugin(tags)}
			var got map[string]any
			if err := yaml.Load([]byte(input), &got, options...); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}
			if err := yaml.Load([]byte("!load vars: "+path), &got, options...); err == nil {
				t.Fatal("unbound load tag unexpectedly succeeded")
			}
		})
	}
}
