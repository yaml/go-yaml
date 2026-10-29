// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package v4migration

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	legacy "go.yaml.in/yaml/v3"
	"go.yaml.in/yaml/v4"
	gopkg "gopkg.in/yaml.v3"
)

type captureV3 func(*legacy.Node) error

func (f captureV3) UnmarshalYAML(n *legacy.Node) error { return f(n) }

type captureGopkg func(*gopkg.Node) error

func (f captureGopkg) UnmarshalYAML(n *gopkg.Node) error { return f(n) }

type adapterCase struct {
	name    string
	plugin  yaml.V4MigrationPlugin
	capture func(func(any) error) any
}

func adapterCases() []adapterCase {
	return []adapterCase{
		{"from-v3", FromV3(), func(f func(any) error) any {
			hook := captureV3(func(n *legacy.Node) error { return f(n) })
			return &hook
		}},
		{"from-gopkg-v3", FromGopkgV3(), func(f func(any) error) any {
			hook := captureGopkg(func(n *gopkg.Node) error { return f(n) })
			return &hook
		}},
	}
}

// compareGraph checks the public v3 fields and graph identity independently
// of the two typed conversion implementations.
func compareGraph(t *testing.T, src *yaml.Node, dst reflect.Value,
	seen map[*yaml.Node]any,
) {
	t.Helper()
	if src == nil {
		if !dst.IsNil() {
			t.Fatal("nil node acquired a destination")
		}
		return
	}
	if previous, ok := seen[src]; ok {
		if previous != dst.Interface() {
			t.Fatal("shared node identity was lost")
		}
		return
	}
	if dst.IsNil() {
		t.Fatal("non-nil node lost its destination")
	}
	if reflect.ValueOf(src).Pointer() == dst.Pointer() {
		t.Fatal("legacy node shares source storage")
	}
	seen[src] = dst.Interface()
	fields := dst.Elem()
	if fields.FieldByName("Kind").Uint() != uint64(src.Kind) ||
		fields.FieldByName("Style").Uint() != uint64(src.Style) {
		t.Fatal("node kind or style changed")
	}
	original := reflect.ValueOf(src).Elem()
	for _, name := range []string{
		"Tag", "Value", "Anchor", "HeadComment", "LineComment", "FootComment",
		"Line", "Column",
	} {
		if !reflect.DeepEqual(fields.FieldByName(name).Interface(),
			original.FieldByName(name).Interface()) {
			t.Errorf("%s changed", name)
		}
	}
	compareGraph(t, src.Alias, fields.FieldByName("Alias"), seen)
	content := fields.FieldByName("Content")
	if len(src.Content) != content.Len() ||
		(src.Content == nil) != content.IsNil() {
		t.Fatal("content shape changed")
	}
	for i, child := range src.Content {
		compareGraph(t, child, content.Index(i), seen)
	}
}

func TestSafeNodeGraphs(t *testing.T) {
	for _, tc := range adapterCases() {
		t.Run(tc.name, func(t *testing.T) {
			scalar := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Style: yaml.TaggedStyle | yaml.DoubleQuotedStyle,
				Tag:   "!!str", Value: "original", Anchor: "scalar",
				HeadComment: "head", LineComment: "line", FootComment: "foot",
				Line: 12, Column: 7,
				Stream: &yaml.Stream{},
			}
			mapping := &yaml.Node{Kind: yaml.MappingNode, Anchor: "mapping"}
			alias := &yaml.Node{Kind: yaml.AliasNode, Value: "mapping", Alias: mapping}
			mapping.Content = []*yaml.Node{scalar, alias, scalar, nil}
			sequence := &yaml.Node{
				Kind: yaml.SequenceNode, Style: yaml.FlowStyle,
				Content: []*yaml.Node{mapping, alias, scalar},
			}
			root := &yaml.Node{
				Kind:    yaml.DocumentNode,
				Content: []*yaml.Node{sequence},
			}
			target := tc.capture(func(n any) error {
				seen := map[*yaml.Node]any{}
				compareGraph(t, root, reflect.ValueOf(n), seen)
				reflect.ValueOf(seen[scalar]).Elem().FieldByName("Value").
					SetString("legacy mutation")
				return nil
			})
			handled, err := tc.plugin.TryUnmarshalYAML(target, root)
			if !handled || err != nil {
				t.Fatalf("handled=%v, error=%v", handled, err)
			}
			if scalar.Value != "original" {
				t.Fatal("legacy mutation changed a v4 node")
			}
		})
	}
}

func TestStylesAndEmptyNodes(t *testing.T) {
	for _, tc := range adapterCases() {
		for _, style := range []yaml.Style{
			0, yaml.TaggedStyle, yaml.DoubleQuotedStyle, yaml.SingleQuotedStyle,
			yaml.LiteralStyle, yaml.FoldedStyle, yaml.FlowStyle,
		} {
			n := &yaml.Node{
				Kind: yaml.ScalarNode, Style: style,
				Content: []*yaml.Node{},
			}
			_, err := tc.plugin.TryUnmarshalYAML(tc.capture(func(got any) error {
				compareGraph(t, n, reflect.ValueOf(got), map[*yaml.Node]any{})
				return nil
			}), n)
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, n := range []*yaml.Node{nil, {}} {
			_, err := tc.plugin.TryUnmarshalYAML(tc.capture(func(got any) error {
				compareGraph(t, n, reflect.ValueOf(got), map[*yaml.Node]any{})
				return nil
			}), n)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestUnsupportedNodeKindsDoNotCallHooks(t *testing.T) {
	for _, tc := range adapterCases() {
		for _, kind := range []yaml.Kind{yaml.StreamNode, 1024} {
			called := false
			target := tc.capture(func(any) error { called = true; return nil })
			n := &yaml.Node{
				Kind:    yaml.SequenceNode,
				Content: []*yaml.Node{{Kind: kind}},
			}
			handled, err := tc.plugin.TryUnmarshalYAML(target, n)
			if !handled || err == nil || called {
				t.Fatalf("handled=%v, err=%v, called=%v", handled, err, called)
			}
		}
	}
}

func TestAdaptersDeclineOtherPackage(t *testing.T) {
	cases := adapterCases()
	for i, tc := range cases {
		target := cases[1-i].capture(func(any) error {
			t.Fatal("wrong package hook called")
			return nil
		})
		handled, err := tc.plugin.TryUnmarshalYAML(target,
			&yaml.Node{Kind: yaml.StreamNode})
		if handled || err != nil {
			t.Fatalf("handled=%v, err=%v", handled, err)
		}
	}
}

type valueV3 struct {
	Value string
}

func (v *valueV3) UnmarshalYAML(n *legacy.Node) error {
	type plain valueV3
	return n.Decode((*plain)(v))
}

type valueGopkg struct {
	Value string
}

func (v *valueGopkg) UnmarshalYAML(n *gopkg.Node) error {
	type plain valueGopkg
	return n.Decode((*plain)(v))
}

type holder[T any] struct {
	Item T
}

type inlineHolder[T any] struct {
	Item T `yaml:",inline"`
}

func TestRealHooksThroughLoadingAPIs(t *testing.T) {
	for _, tc := range []struct {
		name      string
		plugin    yaml.V4MigrationPlugin
		makeValue func() any
	}{
		{"v3", FromV3(), func() any { return new(valueV3) }},
		{"gopkg", FromGopkgV3(), func() any { return new(valueGopkg) }},
	} {
		opt := yaml.WithPlugin(tc.plugin)
		for name, load := range map[string]func([]byte, any) error{
			"Load": func(in []byte, out any) error {
				return yaml.Load(in, out, opt)
			},
			"Loader": func(in []byte, out any) error {
				loader, err := yaml.NewLoader(strings.NewReader(string(in)), opt)
				if err != nil {
					return err
				}
				return loader.Load(out)
			},
			"Node.Load": func(in []byte, out any) error {
				var n yaml.Node
				if err := yaml.Load(in, &n); err != nil {
					return err
				}
				return n.Load(out, opt)
			},
			"Unmarshal": func(in []byte, out any) error {
				return Unmarshal(in, out, opt)
			},
			"Decoder": func(in []byte, out any) error {
				return NewDecoder(strings.NewReader(string(in)), opt).Decode(out)
			},
		} {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				out := tc.makeValue()
				if err := load([]byte("value: loaded"), out); err != nil {
					t.Fatal(err)
				}
				if reflect.ValueOf(out).Elem().FieldByName("Value").String() != "loaded" {
					t.Fatal(out)
				}
			})
		}
	}
}

func TestMixedLegacyDestinations(t *testing.T) {
	var out struct {
		V3     valueV3
		Gopkg  *valueGopkg
		Items  []valueV3
		Lookup map[string]valueGopkg
		Nested holder[*valueV3]
		Inline inlineHolder[valueGopkg] `yaml:",inline"`
	}
	in := []byte(`v3: {value: v3}
gopkg: {value: gopkg}
items: [{value: item}]
lookup: {key: {value: map}}
nested: {item: {value: pointer}}
value: inline
`)
	if err := yaml.Load(in, &out,
		yaml.WithPlugin(FromV3(), FromGopkgV3())); err != nil {
		t.Fatal(err)
	}
	got := []string{
		out.V3.Value, out.Gopkg.Value, out.Items[0].Value,
		out.Lookup["key"].Value, out.Nested.Item.Value, out.Inline.Item.Value,
	}
	want := []string{"v3", "gopkg", "item", "map", "pointer", "inline"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestLegacyDecodeKeepsLegacyOptions(t *testing.T) {
	var out struct {
		V3    valueV3
		Gopkg valueGopkg
	}
	opts := []yaml.Option{
		yaml.WithKnownFields(),
		yaml.WithPlugin(FromV3(), FromGopkgV3()),
	}
	if err := yaml.Load([]byte(`v3: {value: a, unknown: ignored}
gopkg: {value: b, unknown: ignored}
`), &out, opts...); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Load([]byte("unknown: rejected"), &out, opts...); err == nil {
		t.Fatal("outer v4 KnownFields was lost")
	}
}

func TestHookErrorsPreserveIdentity(t *testing.T) {
	sentinel := errors.New("hook failed")
	for _, tc := range adapterCases() {
		target := tc.capture(func(any) error { return sentinel })
		if err := yaml.Load([]byte("input"), target,
			yaml.WithPlugin(tc.plugin)); !errors.Is(err, sentinel) {
			t.Fatalf("error identity lost: %v", err)
		}
	}
	gopkgErr := &gopkg.TypeError{Errors: []string{"gopkg failure"}}
	gopkgTarget := captureGopkg(func(*gopkg.Node) error { return gopkgErr })
	gopkgLoadErr := yaml.Load([]byte("input"), &gopkgTarget,
		yaml.WithPlugin(FromGopkgV3()))
	var gopkgGot *gopkg.TypeError
	if !errors.As(gopkgLoadErr, &gopkgGot) || gopkgGot != gopkgErr {
		t.Fatalf("gopkg TypeError lost: %v", gopkgLoadErr)
	}
	v3Err := &legacy.TypeError{Errors: []string{"v3 failure"}}
	target := captureV3(func(*legacy.Node) error { return v3Err })
	err := yaml.Load([]byte("input"), &target, yaml.WithPlugin(FromV3()))
	var got *legacy.TypeError
	if !errors.As(err, &got) || got != v3Err {
		t.Fatalf("v3 TypeError lost: %v", err)
	}
}

var (
	registerOnce  sync.Once
	registerError error
)

func TestNamedConfiguration(t *testing.T) {
	registerOnce.Do(func() { registerError = Register() })
	if registerError != nil {
		t.Fatal(registerError)
	}
	for _, tc := range adapterCases() {
		opt, err := yaml.OptsYAML("plugin: {v4-migration: " + tc.name + "}")
		if err != nil {
			t.Fatal(err)
		}
		target := tc.capture(func(any) error { return nil })
		if err := yaml.Load([]byte("input"), target, opt); err != nil {
			t.Fatal(err)
		}
	}
	for _, config := range []string{
		"plugin: {v4-migration: true}",
		"plugin: {v4-migration: missing}",
		"plugin: {v4-migration: {name: from-v3, unknown: true}}",
	} {
		if _, err := yaml.OptsYAML(config); err == nil {
			t.Fatalf("accepted %s", config)
		}
	}
	opt, err := yaml.OptsYAML("plugin: {v4-migration: false}")
	if err != nil {
		t.Fatal(err)
	}
	var out valueV3
	if err := yaml.Load([]byte("value: input"), &out, opt); err == nil {
		t.Fatal("registration or disabled config enabled migration")
	}
}

func TestConcurrentReuse(t *testing.T) {
	opt := yaml.WithPlugin(FromV3(), FromGopkgV3())
	var workers sync.WaitGroup
	for i := 0; i < 20; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			var out struct {
				V3    valueV3
				Gopkg valueGopkg
			}
			if err := yaml.Load([]byte("v3: {value: a}\ngopkg: {value: b}"),
				&out, opt); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
}
