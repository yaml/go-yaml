// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/zip"
	"bytes"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParsePackages(t *testing.T) {
	got, err := parsePackages(
		" example.org/parser/go@v1.2.3,example.org/limit@v0.2.0-rc.1 ")
	want := []string{
		"example.org/parser/go@v1.2.3", "example.org/limit@v0.2.0-rc.1",
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, %v", got, err)
	}
	for _, spec := range []string{
		"example.org/parser", "example.org/parser@latest",
		"example.org/parser@main", "example.org/parser@1.2.3",
		"./repos/parser@v1.2.3", "example.org/../parser@v1.2.3",
		"example.org/parser@v1.2.3,", "example.org/parser@v1.2.3,,",
		"example.org/parser@v1.2.3,example.org/parser@v2.0.0",
		"example.org/parser@v1.2.3;echo injected",
	} {
		if _, err := parsePackages(spec); err == nil {
			t.Fatalf("accepted %q", spec)
		}
	}
}

// pluginProxy serves a tagged external module without network dependencies.
func pluginProxy(t *testing.T) string {
	t.Helper()
	const module = "example.org/cli-plugins"
	const version = "v1.2.3"
	const parser = `package parser

import (
	"fmt"
	"strings"
	"go.yaml.in/yaml/v4"
)

type Plugin struct { prefix string }

func (p *Plugin) Parse(input []byte) ([]yaml.PluginEvent, error) {
	return []yaml.PluginEvent{
		{Type: "stream_start"}, {Type: "document_start"},
		{Type: "scalar", Style: "single",
			Value: p.prefix + strings.TrimSpace(string(input))},
		{Type: "document_end"}, {Type: "stream_end"},
	}, nil
}

func Register() error {
	return yaml.RegisterPlugin(yaml.PluginRegistration{
		API: "yaml-parser", Name: "fixture", Version: "1.2.3",
		Factory: func(config map[string]any) (any, error) {
			p := &Plugin{prefix: "fixture: "}
			for key, value := range config {
				if key != "prefix" { return nil, fmt.Errorf("unknown setting %s", key) }
				var ok bool
				p.prefix, ok = value.(string)
				if !ok { return nil, fmt.Errorf("prefix must be a string") }
			}
			return p, nil
		},
	})
}
`
	files := map[string]string{
		"go.mod": "module " + module + "\n\ngo 1.18\n\n" +
			"require go.yaml.in/yaml/v4 v4.0.0-rc.6\n",
		"parser/plugin.go": parser,
		"extra/plugin.go": `package extra
import "go.yaml.in/yaml/v4"
func Register() error {
	_, err := yaml.OptsYAML("plugin: {yaml-parser: fixture}")
	return err
}
`,
		"duplicate/plugin.go": strings.Replace(parser,
			"package parser", "package duplicate", 1),
		"missing/plugin.go": "package missing\nconst Version = 1\n",
		"wrong/plugin.go":   "package wrong\nfunc Register() {}\n",
		"failure/plugin.go": "package failure\nimport \"errors\"\n" +
			"func Register() error { return errors.New(\"fixture failed\") }\n",
	}
	proxy := t.TempDir()
	directory := filepath.Join(proxy, module, "@v")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for name, contents := range files {
		entry, err := writer.Create(module + "@" + version + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for suffix, contents := range map[string][]byte{
		".mod": []byte(files["go.mod"]),
		".info": []byte(`{"Version":"v1.2.3",` +
			`"Time":"2026-01-01T00:00:00Z"}`),
		".zip": archive.Bytes(),
	} {
		if err := os.WriteFile(filepath.Join(directory, version+suffix),
			contents, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GOPROXY", (&url.URL{Scheme: "file", Path: proxy}).String())
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GONOPROXY", "none")
	t.Setenv("GOMODCACHE", t.TempDir())
	t.Setenv("GOFLAGS", strings.TrimSpace(os.Getenv("GOFLAGS")+" -modcacherw"))
	return module
}

func TestExternalPluginCLI(t *testing.T) {
	module := pluginProxy(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	binary := filepath.Join(directory, "go-yaml")
	config := filepath.Join(directory, "options.yaml")
	packages := module + "/parser@v1.2.3," + module + "/extra@v1.2.3"
	build := func(config, selection, packages string) error {
		t.Helper()
		return buildCLIWithPlugins(root, config, selection, binary,
			goTool, "", packages)
	}
	run := func(want string, flags ...string) {
		t.Helper()
		cmd := exec.Command(binary, flags...)
		cmd.Stdin = strings.NewReader("hello\n")
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), want) {
			t.Fatalf("%v: got %q, %v; want %q", flags, out, err, want)
		}
	}
	// Linking packages alone keeps native defaults and permits runtime selection.
	if err := build("", "", packages); err != nil {
		t.Fatal(err)
	}
	run("\"hello\"\n", "-j")
	run("\"fixture: hello\"\n", "-j", "--plugin=yaml-parser=fixture@1.2.3,loader-limits")
	if err := build("", "yaml-parser=fixture,loader-limits", packages); err != nil {
		t.Fatal(err)
	}
	for flag, want := range map[string]string{
		"-j": "\"fixture: hello\"\n", "-y": "fixture: hello",
		"-n": "fixture: hello", "-e": "SCALAR",
	} {
		run(want, flag)
	}
	for _, flags := range [][]string{
		{"-t"}, {"-T"}, {"-j", "--unmarshal"}, {"-j", "--decode"},
	} {
		cmd := exec.Command(binary, flags...)
		out, err := cmd.CombinedOutput()
		if err == nil ||
			!bytes.Contains(out, []byte("parser plugins are not supported")) {
			t.Fatalf("%v: got %q, %v", flags, out, err)
		}
	}
	writeConfig := func(contents string) {
		t.Helper()
		if err := os.WriteFile(config, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeConfig("plugin: {yaml-parser: {name: fixture, version: 1.2.3, prefix: 'custom: '}}\n")
	if err := build(config, "", packages); err != nil {
		t.Fatal(err)
	}
	run("\"custom: hello\"", "-j")
	run("\"fixture: hello\"", "-j", "--plugin=yaml-parser=fixture")
	writeConfig("{}\n")
	run("\"hello\"", "-j", "-C", config)
	run("\"fixture: hello\"", "-j", "-C", config, "--plugin=yaml-parser=fixture")

	before, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, setting, packages, want string
	}{
		{"unknown implementation", "yaml-parser: absent", packages, "absent"},
		{"version mismatch", "yaml-parser: fixture@v1.2.2", packages, "version mismatch"},
		{
			"bad option", "yaml-parser: {name: fixture, unknown: true}",
			packages, "unknown setting",
		},
		{
			"duplicate", "yaml-parser: fixture",
			packages + "," + module + "/duplicate@v1.2.3", "already registered",
		},
		{"missing Register", "", module + "/missing@v1.2.3", "Register"},
		{"wrong Register", "", module + "/wrong@v1.2.3", "Register"},
		{"registration error", "", module + "/failure@v1.2.3", "fixture failed"},
		{"unavailable version", "", module + "/parser@v1.2.4", "v1.2.4"},
		{"malformed package", "", module + "/parser", "invalid PLUGIN-PACKAGE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contents := "{}\n"
			if tc.setting != "" {
				contents = "plugin: {" + tc.setting + "}\n"
			}
			writeConfig(contents)
			err := build(config, "", tc.packages)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v; want %s", err, tc.want)
			}
			after, err := os.ReadFile(binary)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("failed build replaced the binary")
			}
		})
	}
	modules, err := exec.Command(goTool, "version", "-m", binary).CombinedOutput()
	if err != nil || !bytes.Contains(modules, []byte(module+"\tv1.2.3")) {
		t.Fatalf("linked package version: %v\n%s", err, modules)
	}
}
