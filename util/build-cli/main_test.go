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
		name, config                 string
		external, reference, invalid bool
		version                      string
	}{
		{
			name:   "error format",
			config: "plugin: {error-format: true}",
		},
		{
			name:   "error format implementation",
			config: "plugin: {error-format: error-format}",
		},
		{
			name:   "error format v3",
			config: "plugin: {error-format: {v3: {}}}",
		},
		{
			name:   "error format v3 shorthand",
			config: "plugin: {error-format: v3}",
		},
		{
			name:   "error format v4 shorthand",
			config: "plugin: {error-format: v4}",
		},
		{
			name: "error format v4",
			config: "plugin: {error-format: " +
				"{name: error-format, v4: {position: long}}}",
		},
		{
			name:    "invalid error format",
			config:  "plugin: {error-format: {v4: {position: bad}}}",
			invalid: true,
		},
		{
			name:    "versioned error format",
			config:  "plugin: {error-format: {version: 1.2.3}}",
			invalid: true,
		},
		{
			name:    "unknown error format implementation",
			config:  "plugin: {error-format: {name: missing}}",
			invalid: true,
		},
		{
			name:    "old error format API",
			config:  "plugin: {errfmt: true}",
			invalid: true,
		},
		{
			name:    "old error format implementation",
			config:  "plugin: {error-format: errfmt}",
			invalid: true,
		},
		{
			name:   "disabled old error format API",
			config: "plugin: {errfmt: false}",
		},
		{
			name:   "old limit API",
			config: "plugin: {limit: true}",
		},
		{
			name:    "limit API conflict",
			config:  "plugin: {limit: true, loader-limits: true}",
			invalid: true,
		},
		{
			name:   "YAML parser",
			config: "plugin: {yaml-parser: go-yaml}",
		},
		{
			name:    "old parser API",
			config:  "plugin: {parser: go-yaml}",
			invalid: true,
		},
		{
			name:   "disabled invalid host fields",
			config: "plugin: {absent: {name: 1, version: bad, disable: true}}",
		},
		{name: "defaults", config: "{}"},
		{name: "limit", config: "plugin: {loader-limits: {depth: 50, alias: null}}"},
		{name: "tab indent", config: "plugin: {tab-indent: true}"},
		{name: "merge key", config: "plugin: {merge-key: true}"},
		{name: "merge disabled", config: "plugin: {merge-key: disable}"},
		{name: "merge false", config: "plugin: {merge-key: false}"},
		{
			name:   "merge policy",
			config: "plugin: {merge-key: {multiple: combine, conflicts: last}}",
		},
		{
			name:   "invalid merge policy",
			config: "plugin: {merge-key: {multiple: last}}", invalid: true,
		},
		{
			name:   "tab details",
			config: "plugin: {tab-indent: {load: spaces, dump: spaces, auto: stream}}",
		},
		{name: "limit disabled", config: "plugin: {loader-limits: false}"},
		{
			name:   "disabled mapping",
			config: "plugin: {loader-limits: {depth: 3, disable: true}}",
		},
		{name: "unknown disabled", config: "plugin: {absent: false}"},
		{
			name:   "unknown disabled mapping",
			config: "plugin: {absent: {disable: true, name: 1}}",
		},
		{name: "null plugin", config: "plugin: null", invalid: true},
		{name: "null limit", config: "plugin: {loader-limits: null}", invalid: true},
		{
			name:    "invalid disable",
			config:  "plugin: {loader-limits: {disable: null}}",
			invalid: true,
		},
		{name: "unknown plugin", config: "plugin: {absent: {}}", invalid: true},
		{name: "unknown option", config: "indnet: 4", invalid: true},
		{name: "bad indent", config: "indent: 100", invalid: true},
		{
			name:    "bad limit",
			config:  "plugin: {loader-limits: {depth: nope}}",
			invalid: true,
		},
		{
			name:    "bad tab indent",
			config:  "plugin: {tab-indent: {mode: spaces}}",
			invalid: true,
		},
		{
			name:    "removed tab scope",
			config:  "plugin: {tab-indent: {scope: document}}",
			invalid: true,
		},
		{name: "bad plugins", config: "plugin: [loader-limits]", invalid: true},
		{name: "syntax", config: "[", invalid: true},
		{name: "documents", config: "--- {}\n--- {}", invalid: true},
		{
			name:      "reference",
			config:    "plugin: {yaml-parser: reference@v0.2.6}",
			reference: true,
			version:   "v0.2.6",
		},
		{
			name:      "reference mapping",
			config:    "plugin: {yaml-parser: {name: reference, version: 0.2.6}}",
			reference: true,
			version:   "v0.2.6",
		},
		{
			name:    "reference null version",
			config:  "plugin: {yaml-parser: {name: reference, version: null}}",
			invalid: true,
		},
		{
			name:    "reference latest version",
			config:  "plugin: {yaml-parser: reference@latest}",
			invalid: true,
		},
		{
			name:     "external parser",
			config:   "plugin: {yaml-parser: custom}",
			external: true,
		},
		{
			name:     "external limit",
			config:   "plugin: {loader-limits: custom}",
			external: true,
		},
		{
			name:     "external API",
			config:   "plugin: {custom: {setting: true}}",
			external: true,
		},
		{
			name:      "external with reference",
			config:    "plugin: {yaml-parser: reference@v0.2.6, custom: true}",
			external:  true,
			reference: true,
			version:   "v0.2.6",
		},
		{
			name:     "external invalid core option",
			config:   "indent: 100",
			external: true,
			invalid:  true,
		},
		{
			name:     "external invalid plugin value",
			config:   "plugin: {custom: null}",
			external: true,
			invalid:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := inspectConfig([]byte(tc.config), tc.external)
			if tc.invalid {
				if err == nil {
					t.Fatal("invalid configuration accepted")
				}
				return
			}
			if err != nil || got.referenceParser != tc.reference ||
				got.referenceVersion != tc.version {
				t.Fatalf("got %+v, %v", got, err)
			}
			if string(got.embedded) != tc.config {
				t.Fatal("embedded configuration changed")
			}
		})
	}
}

func TestConfiguredTabIndentCLI(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "go-yaml")
	if err := buildCLIWithPlugins(
		root, "", "tab-indent", binary, goTool, ""); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "-y")
	cmd.Stdin = strings.NewReader("root:\n\tvalue: true\n")
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "root:\n\tvalue: true\n" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestConfiguredErrorFormatCLI(t *testing.T) {
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
	data := "plugin:\n  error-format:\n    v4:\n" +
		"      load-template: 'custom {{.Stage}}: {{.Message}}'\n"
	if err := os.WriteFile(config, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := buildCLI(root, config, binary, goTool, ""); err != nil {
		t.Fatal(err)
	}
	run := func(want string, flags ...string) {
		t.Helper()
		cmd := exec.Command(binary, append([]string{"-j"}, flags...)...)
		cmd.Stdin = strings.NewReader("value: -\n")
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), want) {
			t.Fatalf("%v: got %q, %v; want %q", flags, out, err, want)
		}
	}
	run("custom scanner: block sequence entries are not allowed in this context")
	for _, selector := range []string{
		"error-format", "error-format=error-format",
	} {
		run("go-yaml load error in scanner at L1.C8:", "--plugin="+selector)
		if err := buildCLIWithPlugins(
			root, "", selector, binary, goTool, ""); err != nil {
			t.Fatal(err)
		}
		run("go-yaml load error in scanner at L1.C8:")
	}
	data = "plugin: {error-format: {v3: {}}}\n"
	if err := os.WriteFile(config, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	run("yaml: line 1: block sequence entries are not allowed in this context",
		"-C", config)
	data = "plugin: {error-format: v3}\n"
	if err := os.WriteFile(config, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	want := "yaml: line 1: block sequence entries are not allowed in this context"
	run(want, "-C", config)
	run(want, "--plugin=error-format=v3")
	if err := buildCLI(root, config, binary, goTool, ""); err != nil {
		t.Fatal(err)
	}
	run(want)
	if err := buildCLIWithPlugins(
		root, "", "error-format=v3", binary, goTool, ""); err != nil {
		t.Fatal(err)
	}
	run(want)
	run(`plugin "errfmt" has no default implementation`, "--plugin=errfmt")
	run(`unknown implementation "errfmt"`, "--plugin=error-format=errfmt")
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
	write("plugin: {absent: {disable: true, version: 1.2.3}}\n")
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
	write("plugin: {yaml-parser: reference@v0.2.6}\n")
	if err := buildCLI(root, config, binary, goTool,
		os.Getenv("GO_YAML_BUILD_PERL")); err != nil {
		t.Fatal(err)
	}
	run("-j")
	run("-j", "--plugin=yaml-parser=reference@0.2.6")
	run("-j", "--plugin=yaml-parser=go-yaml")
	modules, err := exec.Command(goTool, "version", "-m", binary).CombinedOutput()
	if err != nil || !bytes.Contains(modules, []byte(
		"github.com/yamlstar/yamlstar-plugin-parser-reference\tv0.2.6")) {
		t.Fatalf("linked plugin version: %v\n%s", err, modules)
	}
	write("plugin: {yaml-parser: reference@v0.2.4}\n")
	if err := buildCLI(root, config, binary, goTool,
		os.Getenv("GO_YAML_BUILD_PERL")); err == nil {
		t.Fatal("unavailable parser version built")
	}
}

func TestConfiguredMergeKeyCLI(t *testing.T) {
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
	if err := buildCLIWithPlugins(root, "", "merge-key=disable",
		binary, goTool, ""); err != nil {
		t.Fatal(err)
	}
	run := func(input, want string, flags ...string) {
		t.Helper()
		cmd := exec.Command(binary, append([]string{"-j"}, flags...)...)
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != want {
			t.Fatalf("got %q, %v; want %q", out, err, want)
		}
	}
	run("<<: {x: 1}", `{"\u003c\u003c":{"x":1}}`)
	run("<<: {x: 1}", `{"x":1}`, "--plugin=merge-key")
	config := filepath.Join(directory, "merge.yaml")
	if err := os.WriteFile(config, []byte(
		"plugin: {merge-key: {multiple: combine, conflicts: last}}\n"),
		0o600); err != nil {
		t.Fatal(err)
	}
	if err := buildCLI(root, config, binary, goTool, ""); err != nil {
		t.Fatal(err)
	}
	run("<<: {x: 1, a: 1}\n<<: {x: 2, b: 2}", `{"a":1,"b":2,"x":2}`)
}
