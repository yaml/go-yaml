// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/functiontag"
	"go.yaml.in/yaml/v4/plugin/yamlrules"
)

func functions(t *testing.T, exports functiontag.Functions) yaml.Option {
	t.Helper()
	p, err := functiontag.New(exports, nil)
	if err != nil {
		t.Fatal(err)
	}
	return yaml.WithPlugin(p)
}

func TestFunctionTagConstruction(t *testing.T) {
	type Point struct{ X, Y int }
	calls := 0
	option := functions(t, functiontag.Functions{
		"point": func(x, y int) *Point { calls++; return &Point{x, y} },
		"sum": func(xs []int) int {
			total := 0
			for _, x := range xs {
				total += x
			}
			return total
		},
		"drop":    func(any) yaml.ConstructionControl { return yaml.Omit },
		"items":   func(any) yaml.ConstructionControl { return yaml.Splice([]int{3, 4}) },
		"yes":     func(any) yaml.ConstructionControl { return yaml.SpliceValue },
		"entries": func(any) yaml.ConstructionControl { return yaml.Splice([]yaml.Entry{{Key: "z", Value: 9}}) },
	})
	var data map[string]any
	err := yaml.Load([]byte("a: &p !point* [1, 2]\nb: *p\nsum: !sum [1, 2, 3]\nitems: [1, !drop ignored, !items ignored, 5]\n!drop hidden: !missing ignored\n!yes condition: {x: 7, y: 8}\n!entries ignored: !missing ignored\n"), &data, option)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || data["a"] != data["b"] {
		t.Fatalf("alias repeated constructor or lost identity: %d %#v", calls, data)
	}
	if !reflect.DeepEqual(data["items"], []any{1, 3, 4, 5}) || data["sum"] != 6 || data["x"] != 7 || data["z"] != 9 {
		t.Fatalf("unexpected constructed data: %#v", data)
	}
	var point *Point
	if err := yaml.Load([]byte("!point* [4, 5]"), &point, option); err != nil {
		t.Fatal(err)
	}
	if *point != (Point{4, 5}) {
		t.Fatal(point)
	}
	var config struct{ Keep int }
	if err := yaml.Load([]byte("!drop unknown: !missing ignored\nkeep: 2"), &config, option, yaml.WithKnownFields()); err != nil {
		t.Fatal(err)
	}
	if config.Keep != 2 {
		t.Fatal(config)
	}
}

func TestFunctionTagNoDocumentAndNull(t *testing.T) {
	count := 0
	option := functions(t, functiontag.Functions{"effect": func(any) { count++ }, "null": func(any) any { return nil }})
	value := any("unchanged")
	if err := yaml.Load([]byte("!effect ignored"), &value, option); !errors.Is(err, yaml.NoDocument) {
		t.Fatal(err)
	}
	if value != "unchanged" {
		t.Fatalf("omission changed destination: %#v", value)
	}
	var docs []any
	if err := yaml.Load([]byte("--- !effect ignored\n--- null\n--- !null ignored\n"), &docs, option, yaml.WithAllDocuments()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(docs, []any{nil, nil}) || count != 2 {
		t.Fatalf("docs=%#v count=%d", docs, count)
	}
	loader, err := yaml.NewLoader(strings.NewReader("--- !effect ignored\n--- 2"), option)
	if err != nil {
		t.Fatal(err)
	}
	if err := loader.Load(&value); !errors.Is(err, yaml.NoDocument) {
		t.Fatal(err)
	}
	if err := loader.Load(&value); err != nil || value != 2 {
		t.Fatalf("stream did not resume: %v %#v", err, value)
	}
}

func TestRuleSetNames(t *testing.T) {
	cases := []struct {
		set, input string
		want       any
	}{
		{"yamlstar", "012", 12},
		{"yamlstar", "0x10", "0x10"},
		{"go-yaml-v2", "yes", true},
		{"go-yaml-v3", "yes", "yes"},
		{"go-yaml-v4", "yes", "yes"},
		{"yaml-1.1-core", "yes", true},
		{"yaml-1.2-core", "0o12", 10},
		{"failsafe", "true", "true"},
		{"yaml-1.2-json", "0.", float64(0)},
		{"json", "0.5", float64(.5)},
	}
	for _, tc := range cases {
		t.Run(tc.set+"/"+tc.input, func(t *testing.T) {
			p, err := yamlrules.New(tc.set, nil)
			if err != nil {
				t.Fatal(err)
			}
			var got any
			if err := yaml.Load([]byte(tc.input), &got, yaml.WithPlugin(p)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v (%T), want %#v (%T)", got, got, tc.want, tc.want)
			}
		})
	}
	p, _ := yamlrules.New("json", nil)
	var value any
	if err := yaml.Load([]byte("0."), &value, yaml.WithPlugin(p)); err == nil {
		t.Fatal("JSON accepted 0.")
	}
}

func TestRulesStructurePairsAndDesolve(t *testing.T) {
	ruleText := `plugin:
  yaml-rules:
    name: yaml-1.2-core
    .rules:
    - .when: {x: +Num, y: +Num}
      .tag: '!point'
    .root:
      id: {.rules: +Str}
      .pairs:
      - .when:
          .key: +Str ~"if .*"
          .value: +Map
        .tags: {key: '!yes'}
`
	options, err := yaml.OptsYAML(ruleText)
	if err != nil {
		t.Fatal(err)
	}
	option := functions(t, functiontag.Functions{"point": func(p map[string]int) map[string]int { return p }, "yes": func(any) yaml.ConstructionControl { return yaml.SpliceValue }})
	var got map[string]any
	if err := yaml.Load([]byte("id: 123\nif condition: {promoted: true}\np: {x: 1, y: 2}"), &got, options, option); err != nil {
		t.Fatal(err)
	}
	if got["id"] != "123" || got["promoted"] != true {
		t.Fatal(got)
	}
	for _, name := range yamlrules.Names() {
		p, _ := yamlrules.New(name, nil)
		original := map[string]any{"true": "true", "number": "42", "bool": true, "int": 12}
		bytes, err := yaml.Dump(original, yaml.WithPlugin(p))
		if err != nil {
			t.Fatal(name, err)
		}
		var roundtrip map[string]any
		if err := yaml.Load(bytes, &roundtrip, yaml.WithPlugin(p)); err != nil {
			t.Fatalf("%s: %s: %v", name, bytes, err)
		}
		if !reflect.DeepEqual(roundtrip, original) {
			t.Fatalf("%s: %s: %#v", name, bytes, roundtrip)
		}
	}
}

func TestFunctionTagNodeAndFailures(t *testing.T) {
	failure := errors.New("expected constructor failure")
	option := functions(t, functiontag.Functions{
		"add":   func(x, y int) (int, error) { return x + y, nil },
		"fail":  func(any) (any, error) { return nil, failure },
		"items": func(any) yaml.ConstructionControl { return yaml.Splice([]int{1, 2}) },
	})
	var node yaml.Node
	if err := yaml.Load([]byte("!add* [1, 2]"), &node); err != nil {
		t.Fatal(err)
	}
	var value int
	if err := node.Load(&value, option); err != nil || value != 3 {
		t.Fatalf("Node.Load: %d %v", value, err)
	}
	var target any
	if err := yaml.Load([]byte("!fail ignored"), &target, option); !errors.Is(err, failure) {
		t.Fatalf("lost error cause: %v", err)
	}
	for _, input := range []string{"!missing value", "!add* 1", "!add* [1]", "!items ignored"} {
		if err := yaml.Load([]byte(input), &target, option); err == nil {
			t.Fatalf("%s should fail", input)
		}
	}
	var array [2]int
	if err := yaml.Load([]byte("[!items ignored]"), &array, option); err != nil || array != [2]int{1, 2} {
		t.Fatalf("array splice: %v %v", array, err)
	}
}

func TestRulesPreserveStructuralMeaningOnDump(t *testing.T) {
	p, err := yamlrules.New("yaml-1.2-core", map[string]any{
		".rules": []any{map[string]any{".when": map[string]any{"x": "+Num", "y": "+Num"}, ".tag": "!point"}},
		".root":  map[string]any{"id": map[string]any{".rules": "+Int"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	option := functions(t, functiontag.Functions{"point": func(value any) any { count++; return value }})
	value := map[string]any{"x": 1, "y": 2, "id": "123"}
	data, err := yaml.Dump(value, yaml.WithPlugin(p), option)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := yaml.Load(data, &got, yaml.WithPlugin(p), option); err != nil {
		t.Fatal(string(data), err)
	}
	if count != 0 || !reflect.DeepEqual(value, got) {
		t.Fatalf("dump altered meaning: %s %#v calls=%d", data, got, count)
	}
}

func TestRulesRequireCollectionConstructors(t *testing.T) {
	p, err := yamlrules.New("yaml-1.2-core", map[string]any{".rules": []any{map[string]any{".when": "+Map", ".tag": "!missing"}}})
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := yaml.Load([]byte("{x: 1}"), &value, yaml.WithPlugin(p)); err == nil {
		t.Fatal("custom collection tag bypassed construction")
	}
	if _, err := functiontag.New(nil, map[string]any{"missing": "unregistered-library"}); err == nil {
		t.Fatal("unknown library accepted")
	}
}

type entryInspector struct{ seen *yaml.FunctionTagInput }

func (p entryInspector) ConstructFunctionTag(input yaml.FunctionTagInput) (any, error) {
	*p.seen = input
	return yaml.Omit, nil
}

func TestFunctionTagEntryContext(t *testing.T) {
	var seen yaml.FunctionTagInput
	var value map[string]any
	input := "!inspect 'vars =': !unregistered 'import(vars.yaml)'\nkeep: 1\n"
	if err := yaml.Load([]byte(input), &value, yaml.WithPlugin(entryInspector{seen: &seen})); err != nil {
		t.Fatal(err)
	}
	if seen.Role != "key" || seen.PairKey == nil || seen.PairValue == nil || seen.PairKey.Value != "vars =" || seen.PairValue.Value != "import(vars.yaml)" || seen.PairValue.Tag != "!unregistered" {
		t.Fatalf("missing read-only pair context: %#v", seen)
	}
	if !reflect.DeepEqual(value, map[string]any{"keep": 1}) {
		t.Fatal(value)
	}
}
