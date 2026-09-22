// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectConfig(t *testing.T) {
	for _, tc := range []struct {
		name, config  string
		json, invalid bool
		version       string
	}{
		{"defaults", "{}", false, false, ""},
		{
			"limit", "plugin: {limit: {depth: 50, alias: null}}",
			false, false, "",
		},
		{"JSON comments", "plugin: {json-comments: true}", true, false, ""},
		{
			"JSON comments disabled", "plugin: {json-comments: false}",
			false, false, "",
		},
		{"limit disabled", "plugin: {limit: false}", false, false, ""},
		{
			"limit disabled in mapping",
			"plugin: {limit: {depth: 3, disable: true}}", false, false, "",
		},
		{
			"JSON disabled in mapping",
			"plugin: {json-comments: {disable: true}}", false, false, "",
		},
		{
			"unknown disabled in mapping",
			"plugin: {absent: {disable: true}}", false, false, "",
		},
		{
			"JSON enabled in mapping",
			"plugin: {json-comments: {disable: false}}", true, false, "",
		},
		{"unknown disabled", "plugin: {absent: false}", false, false, ""},
		{"both", "plugin: {limit: {}, json-comments: {}}", true, false, ""},
		{
			"null JSON comments", "plugin: {json-comments: null}",
			false, true, "",
		},
		{"null limit", "plugin: {limit: null}", false, true, ""},
		{
			"invalid disable", "plugin: {json-comments: {disable: null}}",
			false, true, "",
		},
		{"null plugin field", "plugin: null", false, true, ""},
		{"unknown plugin", "plugin: {absent: {}}", false, true, ""},
		{"unknown option", "indnet: 4", false, true, ""},
		{"bad indent", "indent: 100", false, true, ""},
		{"bad limit", "plugin: {limit: {depth: nope}}", false, true, ""},
		{"bad plugins", "plugin: [limit]", false, true, ""},
		{"syntax", "[", false, true, ""},
		{"multiple documents", "--- {}\n--- {}", false, true, ""},
		{
			name: "version", config: "plugin: {json-comments: {version: 0.1.9}}",
			json: true, version: "v0.1.9",
		},
		{
			name:   "prefixed version",
			config: "plugin: {json-comments: {version: v0.1.9}}",
			json:   true, version: "v0.1.9",
		},
		{
			name:   "implementation name",
			config: "plugin: {json-comments: {name: sanitizer}}",
			json:   true,
		},
		{
			name:   "unknown implementation",
			config: "plugin: {json-comments: {name: missing}}", invalid: true,
		},
		{
			name:   "disabled version",
			config: "plugin: {json-comments: {version: 0.1.9, disable: true}}",
		},
		{
			name:   "disabled invalid host fields",
			config: "plugin: {absent: {name: 1, version: bad, disable: true}}",
		},
		{
			name:   "null version",
			config: "plugin: {json-comments: {version: null}}", invalid: true,
		},
		{
			name:   "invalid version",
			config: "plugin: {json-comments: {version: latest}}", invalid: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := inspectConfig([]byte(tc.config))
			if tc.invalid {
				if err == nil {
					t.Fatal("invalid configuration accepted")
				}
				return
			}
			if err != nil || got.jsonComments != tc.json ||
				got.jsonVersion != tc.version {
				t.Fatalf("got %+v, %v", got, err)
			}
			if tc.version != "" &&
				!strings.Contains(string(got.embedded), "version:") {
				t.Fatalf("build version missing from defaults: %s", got.embedded)
			}
		})
	}
}

func TestConfiguredCLI(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	config := filepath.Join(directory, "options with spaces.yaml")
	binary := filepath.Join(directory, "go-yaml")
	write := func(text string) {
		t.Helper()
		if err := os.WriteFile(config, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(want string, flags ...string) {
		t.Helper()
		cmd := exec.Command(binary, flags...)
		cmd.Stdin = strings.NewReader("a: {b: 1}\n")
		out, err := cmd.CombinedOutput()
		if err != nil || string(out) != want {
			t.Fatalf("%v: got %q, %v; want %q", flags, out, err, want)
		}
	}
	write("indent: 4\n")
	if err := buildCLI(root, config, binary, goTool, ""); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(config); err != nil {
		t.Fatal(err)
	}
	run("a:\n    b: 1\n", "-y")
	run("a:\n  b: 1\n", "-y", "-o", "indent=2")
	write("indent: 6\n")
	run("a:\n      b: 1\n", "-y", "-C", config)

	// Rebuilding with the same file name must embed its new contents.
	if err := buildCLI(root, config, binary, goTool, ""); err != nil {
		t.Fatal(err)
	}
	run("a:\n      b: 1\n", "-y")
	before, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	write("indent: 100\n")
	if err := buildCLI(root, config, binary, goTool, ""); err == nil {
		t.Fatal("invalid config built")
	}
	after, err := os.ReadFile(binary)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed build replaced the binary")
	}
	if err := buildCLI(root, config+".missing", binary, goTool, ""); err == nil {
		t.Fatal("missing config built")
	}
	modules, err := exec.Command(goTool, "version", "-m", binary).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(modules, []byte("glojure")) {
		t.Fatal("ordinary configured build linked Glojure")
	}
	write("plugin: {json-comments: {disable: true, version: 0.1.9}}\n")
	if err := buildCLI(root, config, binary, goTool, ""); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "-j")
	cmd.Stdin = strings.NewReader("a: true // comment\n")
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "{\"a\":\"true // comment\"}\n" {
		t.Fatalf("disabled plugin: got %q, %v", out, err)
	}
	modules, err = exec.Command(goTool, "version", "-m", binary).CombinedOutput()
	if err != nil || bytes.Contains(modules, []byte("glojure")) {
		t.Fatalf("disabled plugin linked Glojure: %v", err)
	}
}

func TestConfiguredJSONCLI(t *testing.T) {
	if os.Getenv("GO_YAML_TEST_JSON_COMMENTS") == "" {
		t.Skip("run make test-json-comments for the optional plugin")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	config := filepath.Join(directory, "options.yaml")
	binary := filepath.Join(directory, "go-yaml")
	write := func(text string) {
		t.Helper()
		if err := os.WriteFile(config, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(want string, flags ...string) {
		t.Helper()
		cmd := exec.Command(binary, flags...)
		cmd.Stdin = strings.NewReader("a: true // comment\n")
		out, err := cmd.CombinedOutput()
		if err != nil || string(out) != want {
			t.Fatalf("%v: got %q, %v", flags, out, err)
		}
	}
	write("plugin:\n  limit: {depth: 50, alias: 100}\n  json-comments: true\n")
	if err := buildCLI(root, config, binary, goTool, os.Getenv("GO_YAML_BUILD_PERL")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(config); err != nil {
		t.Fatal(err)
	}
	run("{\"a\":true}\n", "-j")
	run("{\"a\":true}\n", "-j", "--plugin=json-comments")
	run("{\"a\":true}\n", "-j",
		"--plugin=json-comments=sanitizer")
	write("plugin: {json-comments: {name: sanitizer, version: 0.1.9}}\n")
	if err := buildCLI(root, config, binary, goTool,
		os.Getenv("GO_YAML_BUILD_PERL")); err != nil {
		t.Fatal(err)
	}
	run("{\"a\":true}\n", "-j")
	run("{\"a\":true}\n", "-j", "-C", config)
	modules, err := exec.Command(goTool, "version", "-m", binary).CombinedOutput()
	if err != nil || !bytes.Contains(modules, []byte(
		"github.com/yamlstar/yamlstar-plugin-json-comments\tv0.1.9")) {
		t.Fatalf("linked plugin version: %v\n%s", err, modules)
	}
	write("plugin: {json-comments: {version: 0.1.7}}\n")
	versionBefore, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	err = buildCLI(root, config, binary, goTool,
		os.Getenv("GO_YAML_BUILD_PERL"))
	if err == nil {
		t.Fatal("unavailable plugin version built")
	}
	versionAfter, readErr := os.ReadFile(binary)
	if readErr != nil || !bytes.Equal(versionBefore, versionAfter) {
		t.Fatal("failed version selection replaced the binary")
	}
	cmd := exec.Command(binary, "-j", "-C", config)
	cmd.Stdin = strings.NewReader("a: true // comment\n")
	out, err := cmd.CombinedOutput()
	if err == nil || !bytes.Contains(out, []byte("version mismatch")) {
		t.Fatalf("runtime version mismatch: %v, %s", err, out)
	}
	write("{}\n")
	run("{\"a\":\"true // comment\"}\n", "-j", "-C", config)
	run("{\"a\":true}\n", "-j", "-C", config, "--plugin=json-comments")
	write("plugin: {json-comments: false}\n")
	run("{\"a\":\"true // comment\"}\n", "-j", "-C", config)
	run("{\"a\":true}\n", "-j", "-C", config, "--plugin=json-comments")
	write("plugin: {json-comments: {disable: true}}\n")
	run("{\"a\":\"true // comment\"}\n", "-j", "-C", config)
	run("{\"a\":true}\n", "-j", "-C", config, "--plugin=json-comments")

	// Validate optional configuration against the linked implementation.
	before, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	write("plugin: {json-comments: {unknown: true}}\n")
	err = buildCLI(root, config, binary, goTool, os.Getenv("GO_YAML_BUILD_PERL"))
	if err == nil || !strings.Contains(err.Error(), "configuration must be empty") {
		t.Fatalf("got %v", err)
	}
	after, err := os.ReadFile(binary)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed plugin validation replaced the binary")
	}
}

func TestConfiguredReferenceCLI(t *testing.T) {
	if os.Getenv("GO_YAML_TEST_REFERENCE_PARSER") == "" {
		t.Skip("run make test-reference-parser for the optional plugin")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	config := filepath.Join(directory, "options.yaml")
	binary := filepath.Join(directory, "go-yaml")
	write := func(text string) {
		t.Helper()
		if err := os.WriteFile(config, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(flags ...string) {
		t.Helper()
		cmd := exec.Command(binary, flags...)
		cmd.Stdin = strings.NewReader("a: 1\n")
		out, err := cmd.CombinedOutput()
		if err != nil || string(out) != "{\"a\":1}\n" {
			t.Fatalf("%v: got %q, %v", flags, out, err)
		}
	}
	write("plugin: {parser: reference@v0.2.5}\n")
	if err := buildCLI(root, config, binary, goTool,
		os.Getenv("GO_YAML_BUILD_PERL")); err != nil {
		t.Fatal(err)
	}
	run("-j")
	run("-j", "--plugin=parser=reference@0.2.5")
	run("-j", "--plugin=parser=go-yaml")
	modules, err := exec.Command(goTool, "version", "-m", binary).CombinedOutput()
	if err != nil || !bytes.Contains(modules, []byte(
		"github.com/yamlstar/yamlstar-plugin-parser-reference\tv0.2.5")) {
		t.Fatalf("linked plugin version: %v\n%s", err, modules)
	}
	write("plugin: {parser: reference@v0.2.4}\n")
	if err := buildCLI(root, config, binary, goTool,
		os.Getenv("GO_YAML_BUILD_PERL")); err == nil {
		t.Fatal("unavailable parser version built")
	}
}
