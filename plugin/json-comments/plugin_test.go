package jsoncomments_test

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
	jsoncomments "go.yaml.in/yaml/v4/plugin/json-comments"
	"go.yaml.in/yaml/v4/plugin/loaderlimits"
)

func TestFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/comments.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ Name, Input, Want, Error string }
	if err := yaml.Load(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var got, want []any
			err := yaml.Load([]byte(tc.Input), &got, yaml.WithAllDocuments(), yaml.WithPlugin(jsoncomments.New()))
			if tc.Error != "" {
				if err == nil || !strings.Contains(err.Error(), tc.Error) {
					t.Fatalf("got %v, want %s", err, tc.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := yaml.Load([]byte(tc.Want), &want, yaml.WithAllDocuments()); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}
		})
	}
}

func TestLoaderAndNodes(t *testing.T) {
	input := "--- &a [!!str 1, 'two'] // comment\n"
	loader, err := yaml.NewLoader(strings.NewReader(input), yaml.WithPlugin(jsoncomments.New()))
	if err != nil {
		t.Fatal(err)
	}
	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		t.Fatal(err)
	}
	seq := node.Content[0]
	if seq.Style&yaml.FlowStyle == 0 || seq.Anchor != "a" || seq.Content[0].Tag != "!!str" || seq.Content[1].Style&yaml.SingleQuotedStyle == 0 {
		t.Fatalf("lost node syntax: %#v", seq)
	}
	// A buffered parser still exposes documents through successive Load calls.
	loader, err = yaml.NewLoader(strings.NewReader("--- true// first\n--- false\n"), yaml.WithPlugin(jsoncomments.New()))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []bool{true, false} {
		var got bool
		if err := loader.Load(&got); err != nil || got != want {
			t.Fatalf("got %v, %v", got, err)
		}
	}
	for range 2 {
		if err := loader.Load(&node); !errors.Is(err, io.EOF) {
			t.Fatalf("want EOF: %v", err)
		}
	}
}

var (
	registerOnce sync.Once
	registerErr  error
)

func TestRegistration(t *testing.T) {
	registerOnce.Do(func() { registerErr = jsoncomments.Register() })
	if registerErr != nil {
		t.Fatal(registerErr)
	}
	if err := jsoncomments.Register(); err == nil {
		t.Fatal("duplicate registration succeeded")
	}
	for _, config := range []string{
		"plugin:\n  json-comments: true\n",
		"plugin:\n  json-comments: {}\n",
		"plugin:\n  json-comments: {disable: false}\n",
		"plugin:\n  json-comments: sanitizer@v0.1.9\n",
		"plugin:\n  json-comments: {name: sanitizer, version: 0.1.9}\n",
		"plugin:\n  json-comments: {version: v0.1.9}\n",
	} {
		opt, err := yaml.OptsYAML(config)
		if err != nil {
			t.Fatal(err)
		}
		var got bool
		if err := yaml.Load([]byte("true// yes"), &got, opt); err != nil || !got {
			t.Fatalf("%v, %v", got, err)
		}
	}
	for _, config := range []string{
		"plugin:\n  json-comments: null\n",
		"plugin:\n  json-comments: {unknown: true}\n",
		"plugin:\n  json-comments: {name: missing}\n",
		"plugin:\n  json-comments: {version: 0.1.7}\n",
	} {
		if _, err := yaml.OptsYAML(config); err == nil {
			t.Fatalf("accepted %q", config)
		}
	}
	opt, err := yaml.OptsYAML("plugin: {json-comments: false}")
	if err != nil {
		t.Fatal(err)
	}
	var plain string
	if err := yaml.Load([]byte("true// yes"), &plain, opt); err != nil || plain != "true// yes" {
		t.Fatalf("disabled plugin: %q, %v", plain, err)
	}
	opt, err = yaml.OptsYAML("plugin: {json-comments: {disable: true, unknown: true}}")
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Load([]byte("true// yes"), &plain, opt); err != nil || plain != "true// yes" {
		t.Fatalf("disabled mapping: %q, %v", plain, err)
	}
	var got bool
	if err := yaml.Load([]byte("true// yes"), &got, yaml.WithNamedPlugin("json-comments")); err != nil || !got {
		t.Fatalf("%v, %v", got, err)
	}
}

func TestLimitsAndErrors(t *testing.T) {
	plugin := yaml.WithPlugin(jsoncomments.New())
	for _, tc := range []struct {
		input string
		limit yaml.Option
	}{
		{"[[[0]]] // depth", yaml.WithPlugin(
			loaderlimits.New(loaderlimits.DepthValue(2)))},
		{"[&a 1, *a] // aliases", yaml.WithPlugin(
			loaderlimits.New(loaderlimits.AliasValue(0)))},
	} {
		var got any
		if err := yaml.Load([]byte(tc.input), &got, plugin, tc.limit); err == nil {
			t.Fatalf("limit ignored for %s", tc.input)
		}
	}
	var got any
	if err := yaml.Load([]byte{0xff}, &got, plugin); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("got %v", err)
	}
	if err := yaml.Load([]byte("[a, a]"), &got, plugin, plugin); err == nil {
		t.Fatal("multiple json-comments plugins accepted")
	}
	loader, err := yaml.NewLoader(bytes.NewBufferString("--- true\n--- false"), plugin, yaml.WithSingleDocument())
	if err != nil {
		t.Fatal(err)
	}
	if err := loader.Load(&got); err != nil {
		t.Fatal(err)
	}
	if err := loader.Load(&got); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
}

func TestConcurrentLoads(t *testing.T) {
	plugin := jsoncomments.New()
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				var got map[string]bool
				err := yaml.Load([]byte("{a: true /* yes */}"), &got, yaml.WithPlugin(plugin))
				if err != nil || !got["a"] {
					t.Errorf("got %v, %v", got, err)
				}
			}
		}()
	}
	wg.Wait()
}
