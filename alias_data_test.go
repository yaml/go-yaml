// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/aliasdata"
	"go.yaml.in/yaml/v4/plugin/loaderlimits"
)

func TestAliasDataCases(t *testing.T) {
	data, err := os.ReadFile("testdata/alias-data.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, Config, YAML string
		All                bool
		Want               any
		Error              string
		ConfigError        string `yaml:"config-error"`
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
			option, err := yaml.OptsYAML(config)
			if tc.ConfigError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.ConfigError) {
					t.Fatalf("config error: %v, want %q", err, tc.ConfigError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got any
			if tc.All {
				var documents []any
				err = yaml.Load([]byte(tc.YAML), &documents,
					option, yaml.WithAllDocuments())
				got = documents
			} else {
				err = yaml.Load([]byte(tc.YAML), &got, option)
			}
			if tc.Error != "" {
				if err == nil || !strings.Contains(err.Error(), tc.Error) {
					t.Fatalf("load error: %v, want %q", err, tc.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.Want) {
				t.Fatalf("got %#v, want %#v", got, tc.Want)
			}
		})
	}
}

func newAliasPlugin(t *testing.T, options ...aliasdata.Option) *aliasdata.Plugin {
	t.Helper()
	p, err := aliasdata.New(options...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func writeAliasFile(t *testing.T, file, value string) {
	t.Helper()
	if err := os.WriteFile(file, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAliasDataSourcesAndSnapshots(t *testing.T) {
	t.Setenv("ALIAS_DATA_ENV", "001")
	t.Setenv("ALIAS_DATA_SHARED", "environment")
	t.Setenv("ALIAS_DATA_FILE_ENV", "environment")
	file := filepath.Join(t.TempDir(), "aliases.yaml")
	writeAliasFile(t, file, "file_only: {port: 8080}\n"+
		"ALIAS_DATA_SHARED: file\nALIAS_DATA_FILE_ENV: file\n")
	inline := map[string]any{"ALIAS_DATA_SHARED": "inline"}
	p := newAliasPlugin(t, aliasdata.WithData(inline),
		aliasdata.WithFile(file), aliasdata.WithEnv("ALIAS_DATA_*"))
	input := "--- [*ALIAS_DATA_SHARED, *file_only, *ALIAS_DATA_ENV, *ALIAS_DATA_FILE_ENV]\n" +
		"--- [*ALIAS_DATA_SHARED, *file_only, *ALIAS_DATA_ENV, *ALIAS_DATA_FILE_ENV]\n"
	loader, err := yaml.NewLoader(strings.NewReader(input), yaml.WithPlugin(p))
	if err != nil {
		t.Fatal(err)
	}
	want := []any{"inline", map[string]any{"port": 8080}, "001", "file"}
	var first, second any
	if err := loader.Load(&first); err != nil {
		t.Fatal(err)
	}
	writeAliasFile(t, file, "file_only: {port: 9090}\n"+
		"ALIAS_DATA_SHARED: changed\nALIAS_DATA_FILE_ENV: changed\n")
	t.Setenv("ALIAS_DATA_ENV", "002")
	inline["ALIAS_DATA_SHARED"] = "changed"
	if err := loader.Load(&second); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, want) || !reflect.DeepEqual(second, want) {
		t.Fatalf("snapshots: %#v, %#v, want %#v", first, second, want)
	}
	if err := loader.Load(&second); !errors.Is(err, io.EOF) {
		t.Fatalf("EOF: %v", err)
	}
	var refreshed any
	if err := yaml.Load([]byte("[*ALIAS_DATA_SHARED, *file_only, *ALIAS_DATA_ENV, *ALIAS_DATA_FILE_ENV]"),
		&refreshed, yaml.WithPlugin(p)); err != nil {
		t.Fatal(err)
	}
	want = []any{"changed", map[string]any{"port": 9090}, "002", "changed"}
	if !reflect.DeepEqual(refreshed, want) {
		t.Fatalf("fresh load: %#v, want %#v", refreshed, want)
	}
}

func TestAliasDataSourceErrors(t *testing.T) {
	for _, content := range []string{
		"", "[]\n", "--- {}\n--- {}\n", "bad.name: value\n",
		"42: value\n", "same: 1\nsame: 2\n",
	} {
		t.Run(content, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "aliases.yaml")
			writeAliasFile(t, file, content)
			p := newAliasPlugin(t, aliasdata.WithFile(file))
			var value any
			if err := yaml.Load([]byte("value: 1"), &value, yaml.WithPlugin(p)); err == nil {
				t.Fatal("invalid alias source accepted")
			}
		})
	}
	for _, selector := range []any{42, []any{42}, []string{"bad.name"}} {
		if _, err := aliasdata.New(aliasdata.WithEnv(selector)); err == nil {
			t.Fatalf("invalid env selector accepted: %#v", selector)
		}
	}
	t.Setenv("bad.name", "bad")
	p := newAliasPlugin(t, aliasdata.WithEnv(true))
	var value any
	if err := yaml.Load([]byte("value: 1"), &value, yaml.WithPlugin(p)); err == nil {
		t.Fatal("selected invalid environment name accepted")
	}
	// Brackets and question marks have no wildcard meaning.
	for _, selector := range []string{"ALIAS_DATA_[A-Z]*", "ALIAS_DATA_?*"} {
		p := newAliasPlugin(t, aliasdata.WithEnv(selector))
		if err := yaml.Load([]byte("value: 1"), &value, yaml.WithPlugin(p)); err != nil {
			t.Fatal(err)
		}
	}
}

type aliasPort int

type aliasTextValue struct {
	Self *aliasTextValue
}

func (*aliasTextValue) MarshalText() ([]byte, error) {
	return []byte("001"), nil
}

type aliasIgnoredCycle struct {
	Value string
	Self  *aliasIgnoredCycle `yaml:"-"`
}

func TestAliasDataSourceMarshaling(t *testing.T) {
	text := &aliasTextValue{}
	text.Self = text
	ignored := &aliasIgnoredCycle{Value: "safe"}
	ignored.Self = ignored
	inline := struct {
		Fields struct{ First, Second string } `yaml:",inline"`
	}{}
	inline.Fields.First, inline.Fields.Second = "one", "two"
	p := newAliasPlugin(t, aliasdata.WithData(map[string]any{
		"text": text, "ignored": ignored, "inline": inline,
	}))
	var value any
	if err := yaml.Load([]byte("[*text, *ignored, *inline]"), &value,
		yaml.WithPlugin(p)); err != nil {
		t.Fatal(err)
	}
	want := []any{
		"001",
		map[string]any{"value": "safe"},
		map[string]any{"first": "one", "second": "two"},
	}
	if !reflect.DeepEqual(value, want) {
		t.Fatalf("source marshaling: %#v", value)
	}
}

func (p *aliasPort) UnmarshalYAML(node *yaml.Node) error {
	var value int
	if err := node.Decode(&value); err != nil {
		return err
	}
	*p = aliasPort(value + 1)
	return nil
}

func TestAliasDataTypedConstruction(t *testing.T) {
	p := newAliasPlugin(t, aliasdata.WithData(map[string]any{
		"defaults": map[string]any{"port": 8080, "host": "localhost"},
		"port":     8080,
	}))
	var config struct {
		Port aliasPort
		Host string
	}
	if err := yaml.Load([]byte("<<: *defaults\nport: *port\n"),
		&config, yaml.WithPlugin(p)); err != nil {
		t.Fatal(err)
	}
	if config.Port != 8081 || config.Host != "localhost" {
		t.Fatalf("typed construction: %+v", config)
	}
	var node yaml.Node
	if err := yaml.Load([]byte("copy: *defaults"), &node, yaml.WithPlugin(p)); err != nil {
		t.Fatal(err)
	}
	if node.Content[0].Content[1].Alias.Kind != yaml.MappingNode {
		t.Fatal("external alias target absent in node output")
	}
}

func TestAliasDataLimits(t *testing.T) {
	p := newAliasPlugin(t, aliasdata.WithData(map[string]any{
		"value": map[string]any{"items": []any{1, 2, 3}},
	}))
	var value any
	err := yaml.Load([]byte("[*value, *value]"), &value, yaml.WithPlugin(p),
		yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasValue(0))))
	if err == nil || !strings.Contains(err.Error(), "alias count") {
		t.Fatalf("alias limit: %v", err)
	}
	err = yaml.Load([]byte("*value"), &value, yaml.WithPlugin(p),
		yaml.WithPlugin(loaderlimits.New(loaderlimits.DepthValue(1))))
	if err == nil || !strings.Contains(err.Error(), "depth") {
		t.Fatalf("source depth limit: %v", err)
	}
}

func TestAliasDataIsolationAndNodes(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "001", Tag: "!!str"}
	p := newAliasPlugin(t, aliasdata.WithData(map[string]any{"value": node}))
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var result yaml.Node
			if err := yaml.Load([]byte("*value"), &result, yaml.WithPlugin(p)); err != nil {
				t.Error(err)
				return
			}
			target := result.Content[0].Alias
			if target == node || target.Value != "001" || target.Tag != "!!str" {
				t.Error("source node was not copied correctly")
			}
			target.Value = "private mutation"
		}()
	}
	wg.Wait()
	if node.Value != "001" {
		t.Fatal("supplied node mutated")
	}
	var result any
	if err := yaml.Load([]byte("&local value"), &result, yaml.WithPlugin(p)); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Load([]byte("*local"), &result, yaml.WithPlugin(p)); err == nil {
		t.Fatal("anchor leaked between streams")
	}
	if err := yaml.Load([]byte("value"), &result,
		yaml.WithPlugin(p, p)); err == nil {
		t.Fatal("multiple explicit policies accepted")
	}
}

func TestAliasDataInvalidGraphs(t *testing.T) {
	cyclicMap := map[string]any{}
	cyclicMap["self"] = cyclicMap
	cyclicNode := &yaml.Node{Kind: yaml.SequenceNode}
	cyclicNode.Content = []*yaml.Node{cyclicNode}
	for _, value := range []any{
		cyclicMap, cyclicNode,
		&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key"},
		}},
		&yaml.Node{Kind: yaml.AliasNode, Value: "missing"},
	} {
		p := newAliasPlugin(t, aliasdata.WithData(map[string]any{"value": value}))
		var result any
		if err := yaml.Load([]byte("*value"), &result, yaml.WithPlugin(p)); err == nil {
			t.Fatalf("invalid source graph accepted: %T", value)
		}
	}
}

type aliasPolicyFactory struct {
	create func(yaml.AliasDataServices) (yaml.AliasDataContext, error)
}

func (p aliasPolicyFactory) NewAliasDataContext(
	services yaml.AliasDataServices,
) (yaml.AliasDataContext, error) {
	return p.create(services)
}

type tracingAliasPolicy struct {
	actions *[]string
	anchors map[string]*yaml.Node
	fail    string
	cause   error
}

type fixedAliasPolicy struct {
	tracingAliasPolicy
	node *yaml.Node
}

func (p *fixedAliasPolicy) ResolveAlias(string) (*yaml.Node, bool, error) {
	return p.node, true, nil
}

func TestAliasDataPolicyNodeContract(t *testing.T) {
	for _, tc := range []struct {
		name string
		node *yaml.Node
	}{
		{"nil", nil},
		{"invalid kind", &yaml.Node{Kind: yaml.DocumentNode}},
		{"unresolved alias", &yaml.Node{Kind: yaml.AliasNode}},
		{"null", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var actions []string
			factory := aliasPolicyFactory{func(yaml.AliasDataServices) (yaml.AliasDataContext, error) {
				return &fixedAliasPolicy{
					tracingAliasPolicy: tracingAliasPolicy{actions: &actions}, node: tc.node,
				}, nil
			}}
			var value any
			err := yaml.Load([]byte("*value"), &value, yaml.WithPlugin(factory))
			if tc.name == "null" {
				if err != nil || value != nil {
					t.Fatalf("null alias: %#v, %v", value, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "invalid node") {
				t.Fatalf("invalid target was not rejected: %v", err)
			}
		})
	}
}

func (p *tracingAliasPolicy) action(action string) error {
	*p.actions = append(*p.actions, action)
	if action == p.fail {
		return p.cause
	}
	return nil
}

func (p *tracingAliasPolicy) BeginDocument() error {
	p.anchors = map[string]*yaml.Node{}
	return p.action("begin")
}

func (p *tracingAliasPolicy) DefineAnchor(name string, node *yaml.Node) error {
	p.anchors[name] = node
	return p.action("define " + name)
}

func (p *tracingAliasPolicy) ResolveAlias(name string) (*yaml.Node, bool, error) {
	node, found := p.anchors[name]
	return node, found, p.action("resolve " + name)
}

func (p *tracingAliasPolicy) EndDocument() error {
	return p.action("end")
}

func TestAliasDataCustomPolicy(t *testing.T) {
	for _, failure := range []string{
		"", "create", "begin", "define value",
		"resolve value", "end",
	} {
		t.Run(failure, func(t *testing.T) {
			var actions []string
			cause := errors.New("policy failed")
			factory := aliasPolicyFactory{func(yaml.AliasDataServices) (yaml.AliasDataContext, error) {
				actions = append(actions, "create")
				if failure == "create" {
					return nil, cause
				}
				return &tracingAliasPolicy{actions: &actions, fail: failure, cause: cause}, nil
			}}
			var result any
			err := yaml.Load([]byte("local: &value 1\ncopy: *value\n"), &result,
				yaml.WithPlugin(factory))
			if failure != "" {
				var loadError *yaml.LoadError
				if !errors.Is(err, cause) || !errors.As(err, &loadError) ||
					loadError.Stage != yaml.ComposerStage {
					t.Fatalf("callback error lost its cause/stage: %v", err)
				}
				if failure == "resolve value" && loadError.Mark.Line != 2 {
					t.Fatalf("alias error position: %+v", loadError.Mark)
				}
				return
			}
			want := []string{"create", "begin", "define value", "resolve value", "end"}
			if err != nil || !reflect.DeepEqual(actions, want) {
				t.Fatalf("policy lifecycle: %v, %v", actions, err)
			}
		})
	}
}

func TestAliasDataNamedPolicy(t *testing.T) {
	name := fmt.Sprintf("alias-policy-%d", atomic.AddUint64(&registryTestID, 1))
	var actions []string
	err := yaml.RegisterPlugin(yaml.PluginRegistration{
		API: "alias-data", Name: name,
		Factory: func(map[string]any) (any, error) {
			return aliasPolicyFactory{func(yaml.AliasDataServices) (yaml.AliasDataContext, error) {
				return &tracingAliasPolicy{actions: &actions}, nil
			}}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	option, err := yaml.OptsYAML("plugin: {alias-data: {name: " + name + "}}")
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := yaml.Load([]byte("[&a 1, *a]"), &value, option); err != nil {
		t.Fatal(err)
	}
	want := []string{"begin", "define a", "resolve a", "end"}
	if !reflect.DeepEqual(actions, want) {
		t.Fatalf("named policy was not selected: %v", actions)
	}
	var documents []int
	if err := yaml.Load([]byte("--- &a 1\n--- *a\n"), &documents,
		yaml.WithNamedPlugin("alias-data"), yaml.WithAllDocuments()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(documents, []int{1, 1}) {
		t.Fatalf("named default did not enable stream scope: %v", documents)
	}
}

func TestAliasDataProgrammaticScope(t *testing.T) {
	input := []byte("--- &a 1\n--- *a\n")
	data := aliasdata.WithData(map[string]any{"external": 42})
	for _, tc := range []struct {
		name    string
		options []aliasdata.Option
		stream  bool
	}{
		{"empty selection", nil, true},
		{"source configuration", []aliasdata.Option{data}, false},
		{"explicit stream", []aliasdata.Option{data, aliasdata.WithStream(true)}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newAliasPlugin(t, tc.options...)
			var documents []int
			err := yaml.Load(input, &documents,
				yaml.WithPlugin(p), yaml.WithAllDocuments())
			if tc.stream {
				if err != nil || !reflect.DeepEqual(documents, []int{1, 1}) {
					t.Fatalf("stream aliases: %v, %v", documents, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "unknown anchor") {
				t.Fatalf("source selection changed document scope: %v", err)
			}
		})
	}
}

func TestAliasDataWithParserPlugin(t *testing.T) {
	parser := registeredSourceFunc(func([]byte) ([]yaml.PluginEvent, error) {
		return []yaml.PluginEvent{
			{Type: "stream_start"},
			{Type: "document_start"},
			{Type: "alias", Anchor: "external"},
			{Type: "document_end"},
			{Type: "stream_end"},
		}, nil
	})
	p := newAliasPlugin(t, aliasdata.WithData(map[string]any{"external": 42}))
	var value int
	if err := yaml.Load([]byte("ignored"), &value, yaml.WithPlugin(parser, p)); err != nil {
		t.Fatal(err)
	}
	if value != 42 {
		t.Fatalf("parser alias: %d", value)
	}
}
