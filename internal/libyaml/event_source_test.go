// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package libyaml_test

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
)

type sourceFunc func([]byte) ([]yaml.PluginEvent, error)

func (f sourceFunc) Parse(b []byte) ([]yaml.PluginEvent, error) { return f(b) }

func sourceScalarStream(value string) []yaml.PluginEvent {
	return []yaml.PluginEvent{
		{Type: "stream_start"},
		{Type: "document_start"},
		{Type: "scalar", Value: value},
		{Type: "document_end"},
		{Type: "stream_end"},
	}
}

func TestEventSourcePluginValidation(t *testing.T) {
	data, err := os.ReadFile("testdata/event-source.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name  string
		Types []string
	}
	if err := yaml.Load(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			p := sourceFunc(func([]byte) ([]yaml.PluginEvent, error) {
				events := make([]yaml.PluginEvent, len(tc.Types))
				for i, typ := range tc.Types {
					events[i].Type = typ
				}
				return events, nil
			})
			var got any
			err := yaml.Load(nil, &got, yaml.WithPlugin(p))
			var loadErr *yaml.LoadError
			if !errors.As(err, &loadErr) || loadErr.Stage != yaml.ParserStage {
				t.Fatalf("want event source error, got %v", err)
			}
		})
	}
}

type brokenReader struct{ err error }

func (r brokenReader) Read([]byte) (int, error) { return 0, r.err }

func TestEventSourcePluginBufferingAndErrors(t *testing.T) {
	calls := 0
	cause := errors.New("reader failed")
	p := sourceFunc(func(input []byte) ([]yaml.PluginEvent, error) {
		calls++
		if string(input) != "input" {
			t.Fatalf("unexpected input: %q", input)
		}
		return sourceScalarStream("true"), nil
	})
	loader, err := yaml.NewLoader(strings.NewReader("input"), yaml.WithPlugin(p))
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("event source called before Load")
	}
	var value bool
	if err := loader.Load(&value); err != nil || !value {
		t.Fatalf("%v, %v", value, err)
	}
	if err := loader.Load(&value); !errors.Is(err, io.EOF) {
		t.Fatalf("want EOF, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("event source called %d times", calls)
	}
	loader, err = yaml.NewLoader(brokenReader{cause}, yaml.WithPlugin(p))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := loader.Load(&value); !errors.Is(err, cause) {
			t.Fatalf("lost reader cause: %v", err)
		}
	}
	if calls != 1 {
		t.Fatal("event source called on failed read")
	}
	p = func([]byte) ([]yaml.PluginEvent, error) { return nil, cause }
	if err := yaml.Load(nil, &value, yaml.WithPlugin(p)); !errors.Is(err, cause) {
		t.Fatalf("lost event source cause: %v", err)
	}
}

func TestEventSourcePluginMetadata(t *testing.T) {
	p := sourceFunc(func([]byte) ([]yaml.PluginEvent, error) {
		events := sourceScalarStream("true")
		events[1].Version = &yaml.VersionDirective{Major: 1, Minor: 2}
		events[1].Explicit = true
		events[2].Style = "double"
		events[2].StartMark = yaml.Mark{Line: 4, Column: 7}
		events[2].HeadComment = "# comment"
		return events, nil
	})
	var node yaml.Node
	if err := yaml.Load(nil, &node, yaml.WithPlugin(p)); err != nil {
		t.Fatal(err)
	}
	scalar := node.Content[0]
	if scalar.Tag != "!!str" || scalar.Style&yaml.DoubleQuotedStyle == 0 || scalar.Line != 4 || scalar.Column != 7 || scalar.HeadComment != "# comment" {
		t.Fatalf("lost metadata: %#v", scalar)
	}
}

func TestPluginEventScalarImplicitness(t *testing.T) {
	for _, tc := range []struct {
		name, tag, style string
		implicit, quoted bool
	}{
		{"plain untagged", "", "", true, false},
		{"quoted untagged", "", "double", false, true},
		{"plain non-specific", "!", "", true, false},
		{"quoted non-specific", "!", "double", true, false},
		{"plain explicit", "tag:yaml.org,2002:str", "", false, false},
		{"quoted explicit", "tag:yaml.org,2002:str", "double", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			events := sourceScalarStream("value")
			events[2].Tag = tc.tag
			events[2].Style = tc.style
			reader := libyaml.NewEventReader(strings.NewReader("input"),
				&libyaml.Options{EventSource: sourceFunc(
					func([]byte) ([]yaml.PluginEvent, error) {
						return events, nil
					})})
			defer reader.Delete()
			var event libyaml.Event
			for i := 0; i < 3; i++ {
				if err := reader.Parse(&event); err != nil {
					t.Fatal(err)
				}
			}
			if event.Implicit != tc.implicit ||
				event.GetQuotedImplicit() != tc.quoted {
				t.Fatalf("got implicit %t, quoted %t; want %t, %t",
					event.Implicit, event.GetQuotedImplicit(),
					tc.implicit, tc.quoted)
			}
		})
	}
}
