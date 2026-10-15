package tabindent_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/tabindent"
)

func TestLoadTabIndentation(t *testing.T) {
	input := "root:\n\tchild:\n\t\tvalue: true\n"
	var got map[string]map[string]map[string]bool
	err := yaml.Load(
		[]byte(input), &got, yaml.WithPlugin(tabindent.New()))
	if err != nil {
		t.Fatal(err)
	}
	if !got["root"]["child"]["value"] {
		t.Fatalf("unexpected value: %#v", got)
	}
}

func TestAutoMode(t *testing.T) {
	for _, input := range []string{
		"root:\n  value: true\n",
		"root:\n\tvalue: true\n",
		"root:\n\t# ignored\n\tvalue: true\n",
	} {
		var got any
		if err := yaml.Load(
			[]byte(input), &got,
			yaml.WithPlugin(tabindent.New())); err != nil {
			t.Fatalf("%q: %v", input, err)
		}
	}
	for _, input := range []string{
		"root:\n\tvalue: true\n  other: false\n",
		"root:\n \tvalue: true\n",
	} {
		var got any
		if err := yaml.Load(
			[]byte(input), &got,
			yaml.WithPlugin(tabindent.New())); err == nil {
			t.Fatalf("accepted mixed indentation %q", input)
		}
	}
	var got any
	err := yaml.Load(
		[]byte("root:\n\t value: true\n"), &got,
		yaml.WithPlugin(tabindent.New()))
	if err == nil || !strings.Contains(err.Error(), "L2.C2") {
		t.Fatalf("tab did not count as one source column: %v", err)
	}
}

func TestBlankAndCommentIndentationIgnored(t *testing.T) {
	input := "root:\n \t# ignored\n\t \n\tvalue: true\n"
	for _, plugin := range []*tabindent.Plugin{
		tabindent.New(),
		tabindent.New(tabindent.WithMode(tabindent.ModeTabs)),
	} {
		var got map[string]map[string]bool
		if err := yaml.Load(
			[]byte(input), &got, yaml.WithPlugin(plugin)); err != nil {
			t.Fatal(err)
		}
		if !got["root"]["value"] {
			t.Fatalf("unexpected value: %#v", got)
		}
	}

	var got map[string]string
	if err := yaml.Load(
		[]byte("text: line one\n \t\n\tline two\n"), &got,
		yaml.WithPlugin(tabindent.New())); err != nil {
		t.Fatal(err)
	}
}

func TestTabsMode(t *testing.T) {
	var got any
	plugin := yaml.WithPlugin(tabindent.New(
		tabindent.WithMode(tabindent.ModeTabs)))
	if err := yaml.Load(
		[]byte("root:\n  value: true\n"), &got, plugin); err == nil {
		t.Fatal("accepted structural spaces")
	}
	if err := yaml.Load(
		[]byte("root:\n\tvalue: true\n"), &got, plugin); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSpaces(t *testing.T) {
	var got any
	plugin := yaml.WithPlugin(tabindent.New(
		tabindent.WithMode(tabindent.ModeTabs),
		tabindent.WithLoad(tabindent.StyleSpaces)))
	if err := yaml.Load(
		[]byte("root:\n  value: true\n"), &got, plugin); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Load(
		[]byte("root:\n\tvalue: true\n"), &got, plugin); err == nil {
		t.Fatal("accepted structural tabs")
	}
}

func TestFlowWhitespaceDoesNotSetIndentStyle(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		plugin *tabindent.Plugin
	}{
		{
			name: "spaces with tab mode",
			input: "items: [\n  one,\n  two\n]\n" +
				"root:\n\tvalue: true\n",
			plugin: tabindent.New(
				tabindent.WithMode(tabindent.ModeTabs)),
		},
		{
			name: "tabs with space loading",
			input: "items: [\n\tone,\n\ttwo\n]\n" +
				"root:\n  value: true\n",
			plugin: tabindent.New(
				tabindent.WithLoad(tabindent.StyleSpaces)),
		},
		{
			name: "spaces before automatic tabs",
			input: "items: [\n  one,\n  two\n]\n" +
				"root:\n\tvalue: true\n",
			plugin: tabindent.New(),
		},
		{
			name: "tabs before automatic spaces",
			input: "items: [\n\tone,\n\ttwo\n]\n" +
				"root:\n  value: true\n",
			plugin: tabindent.New(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got any
			if err := yaml.Load(
				[]byte(test.input), &got,
				yaml.WithPlugin(test.plugin)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBlockScalar(t *testing.T) {
	input := "text: |1\n\tline one\n\t\tline two\n"
	var got map[string]string
	if err := yaml.Load(
		[]byte(input), &got,
		yaml.WithPlugin(tabindent.New())); err != nil {
		t.Fatal(err)
	}
	if got["text"] != "line one\n\tline two\n" {
		t.Fatalf("unexpected scalar: %q", got["text"])
	}
}

func TestPlainScalarContinuation(t *testing.T) {
	var got map[string]string
	if err := yaml.Load(
		[]byte("text: line one\n\tline two\n"), &got,
		yaml.WithPlugin(tabindent.New())); err != nil {
		t.Fatal(err)
	}
	if got["text"] != "line one line two" {
		t.Fatalf("unexpected scalar: %q", got["text"])
	}
}

func TestDumpTabIndentation(t *testing.T) {
	value := map[string]any{
		"root": map[string]any{
			"child": []any{
				map[string]any{"value": true},
			},
		},
	}
	data, err := yaml.Dump(
		value, yaml.WithPlugin(tabindent.New()))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "\n ") {
		t.Fatalf("found space indentation:\n%s", data)
	}
	if !strings.Contains(string(data), "\n\tchild:") ||
		!strings.Contains(string(data), "\n\t-") ||
		!strings.Contains(string(data), "\n\t\tvalue:") {
		t.Fatalf("unexpected tab indentation:\n%s", data)
	}
	var roundTrip any
	if err := yaml.Load(
		data, &roundTrip,
		yaml.WithPlugin(tabindent.New())); err != nil {
		t.Fatal(err)
	}
}

func TestDumpCompactSequenceUnderMapping(t *testing.T) {
	data, err := yaml.Dump(
		map[string]any{"foo": []string{"bar"}},
		yaml.WithPlugin(tabindent.New()))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "foo:\n- bar\n" {
		t.Fatalf("unexpected compact sequence indentation:\n%s", data)
	}
}

func TestDumpNestedCollections(t *testing.T) {
	values := []any{
		[]any{1, []any{2, []any{3, 4}, 5}, 6},
		map[string]any{
			"steps": []any{
				map[string]any{
					"name": "checkout",
					"with": map[string]any{"ref": "main"},
				},
			},
		},
	}
	options := []yaml.Option{
		yaml.WithIndent(4), yaml.WithPlugin(tabindent.New()),
	}
	for _, value := range values {
		data, err := yaml.Dump(value, options...)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "\n ") {
			t.Fatalf("found space indentation:\n%s", data)
		}
		var got any
		if err := yaml.Load(data, &got, options...); err != nil {
			t.Fatalf("failed to reload:\n%s\n%v", data, err)
		}
	}
}

func TestDumpSpaceIndentation(t *testing.T) {
	plugin := tabindent.New(
		tabindent.WithMode(tabindent.ModeTabs),
		tabindent.WithLoad(tabindent.StyleSpaces),
		tabindent.WithDump(tabindent.StyleSpaces),
		tabindent.WithAuto(tabindent.ScopeStream))
	value := map[string]any{"root": map[string]any{"value": true}}
	data, err := yaml.Dump(
		value, yaml.WithIndent(4), yaml.WithPlugin(plugin))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "root:\n    value: true\n" {
		t.Fatalf("unexpected space indentation:\n%s", data)
	}
	var got any
	if err := yaml.Load(data, &got, yaml.WithPlugin(plugin)); err != nil {
		t.Fatal(err)
	}
}

var (
	registerOnce sync.Once
	registerErr  error
)

func TestRegistration(t *testing.T) {
	registerOnce.Do(func() { registerErr = tabindent.Register() })
	if registerErr != nil {
		t.Fatal(registerErr)
	}
	option, err := yaml.OptsYAML(
		"plugin:\n  tab-indent:\n    mode: auto\n" +
			"    load: auto\n    dump: tabs\n" +
			"    auto: document\n")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]map[string]bool
	if err := yaml.Load(
		[]byte("root:\n\tvalue: true\n"), &got, option); err != nil {
		t.Fatal(err)
	}
	if !got["root"]["value"] {
		t.Fatalf("unexpected value: %#v", got)
	}
	if _, err := yaml.OptsYAML(
		"plugin: {tab-indent: {scope: document}}"); err == nil {
		t.Fatal("accepted removed scope option")
	}
}

func TestLoaderDumperAndNodeAPIs(t *testing.T) {
	plugin := yaml.WithPlugin(tabindent.New())
	loader, err := yaml.NewLoader(
		strings.NewReader("root:\n\tvalue: true\n"), plugin)
	if err != nil {
		t.Fatal(err)
	}
	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		t.Fatal(err)
	}
	var value map[string]map[string]bool
	if err := node.Load(&value, plugin); err != nil {
		t.Fatal(err)
	}
	if !value["root"]["value"] {
		t.Fatalf("unexpected value: %#v", value)
	}
	var output bytes.Buffer
	dumper, err := yaml.NewDumper(&output, plugin)
	if err != nil {
		t.Fatal(err)
	}
	if err := dumper.Dump(value); err != nil {
		t.Fatal(err)
	}
	if err := dumper.Close(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "\n\tvalue:") {
		t.Fatalf("unexpected dumper output:\n%s", output.String())
	}
	var dumped yaml.Node
	if err := dumped.Dump(value, plugin); err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Dump(&dumped, plugin)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\n\tvalue:") {
		t.Fatalf("unexpected node output:\n%s", data)
	}
}

func TestAutoDetectionScope(t *testing.T) {
	input := "---\nroot:\n\tvalue: true\n---\nroot:\n  value: true\n"
	var values []any
	err := yaml.Load(
		[]byte(input), &values, yaml.WithAllDocuments(),
		yaml.WithPlugin(tabindent.New()))
	if err != nil {
		t.Fatal(err)
	}
	values = nil
	err = yaml.Load(
		[]byte(input), &values, yaml.WithAllDocuments(),
		yaml.WithPlugin(tabindent.New(
			tabindent.WithAuto(tabindent.ScopeStream))))
	if err == nil {
		t.Fatal("stream scope accepted a later indentation style")
	}
}
