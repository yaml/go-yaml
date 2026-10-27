// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFunctionLoadCLI(t *testing.T) {
	directory := t.TempDir()
	config := `plugin:
  alias-data:
    path: true
  function-tag:
    libraries:
      aliases: alias-data
    functions:
      load: aliases/load
`
	for name, content := range map[string]string{
		"options.yaml": config,
		"vars.yaml":    "bar: 42\nhash2: {inherited: true, key: base}\n",
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	binary, err := filepath.Abs(testBinary)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "-C", "options.yaml", "-j")
	command.Dir = directory
	command.Stdin = strings.NewReader("!load vars: vars.yaml\nfoo: *vars.bar\nmap1:\n  <<: *vars.hash2\n  key: value\n")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, output)
	}
	want := "{\"foo\":42,\"map1\":{\"inherited\":true,\"key\":\"value\"}}\n"
	if string(output) != want {
		t.Fatalf("got %s, want %s", output, want)
	}
}

func TestFunctionLoadCLIInputRelativePaths(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source files")
	workingDir := filepath.Join(root, "working")
	configDir := filepath.Join(root, "config")
	missingDir := filepath.Join(root, "missing")
	for _, directory := range []string{sourceDir, workingDir, configDir, missingDir} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	input := "!load vars: vars.yaml\nfoo: *vars.bar\nmap1:\n  <<: *vars.hash2\n  key: value\n"
	config := "plugin: {alias-data: {path: true}, function-tag: {libraries: {aliases: alias-data}}}\n"
	for path, content := range map[string]string{
		filepath.Join(sourceDir, "input.yaml"):    input,
		filepath.Join(sourceDir, "vars.yaml"):     "bar: 42\nhash2: {inherited: true, key: base}\n",
		filepath.Join(workingDir, "vars.yaml"):    "bar: 99\nhash2: {inherited: false, key: other}\n",
		filepath.Join(configDir, "options.yaml"):  config,
		filepath.Join(missingDir, "input.yaml"):   input,
		filepath.Join(sourceDir, "absolute.yaml"): "!load vars: " + fmt.Sprintf("%q", filepath.Join(sourceDir, "vars.yaml")) + "\nfoo: *vars.bar\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	binary, err := filepath.Abs(testBinary)
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(workingDir, filepath.Join(sourceDir, "input.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, mode, file, stdin, want string }{
		{
			name: "relative input", mode: "-j", file: relative,
			want: "{\"foo\":42,\"map1\":{\"inherited\":true,\"key\":\"value\"}}\n",
		},
		{
			name: "absolute input YAML", mode: "-y", file: filepath.Join(sourceDir, "input.yaml"),
			want: "foo: 42\nmap1:\n  inherited: true\n  key: value\n",
		},
		{
			name: "absolute import", mode: "-j", file: filepath.Join(sourceDir, "absolute.yaml"),
			want: "{\"foo\":42}\n",
		},
		{
			name: "stdin", mode: "-j", stdin: input,
			want: "{\"foo\":99,\"map1\":{\"inherited\":false,\"key\":\"value\"}}\n",
		},
		{
			name: "explicit stdin", mode: "-j", file: "-", stdin: input,
			want: "{\"foo\":99,\"map1\":{\"inherited\":false,\"key\":\"value\"}}\n",
		},
		{name: "missing source import", mode: "-j", file: filepath.Join(missingDir, "input.yaml")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"-C", filepath.Join(configDir, "options.yaml"), tc.mode}
			if tc.file != "" {
				args = append(args, tc.file)
			}
			command := exec.Command(binary, args...)
			command.Dir = workingDir
			command.Stdin = strings.NewReader(tc.stdin)
			output, err := command.CombinedOutput()
			if tc.want == "" {
				if err == nil || !strings.Contains(string(output), filepath.Join(missingDir, "vars.yaml")) {
					t.Fatalf("expected missing source-relative import: %v, %s", err, output)
				}
				return
			}
			if err != nil || string(output) != tc.want {
				t.Fatalf("got %q, %v; want %q", output, err, tc.want)
			}
		})
	}
}
