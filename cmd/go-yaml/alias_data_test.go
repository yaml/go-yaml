// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAliasDataCLI(t *testing.T) {
	for _, tc := range []struct {
		name, config, selector, input, output, failure string
	}{
		{
			name:  "ordinary aliases",
			input: "[&a 1, *a]\n", output: "[1,1]\n",
		},
		{
			name: "default document scope", input: "--- &a 1\n--- *a\n",
			failure: "unknown anchor",
		},
		{
			name: "bare selector enables stream", selector: "alias-data",
			input: "--- &a 1\n--- *a\n", output: "1\n1\n",
		},
		{
			name: "document scope", config: "{stream: false}",
			input: "--- &a 1\n--- *a\n", failure: "unknown anchor",
		},
		{
			name: "inline data", config: "{data: {external: 42}}",
			input: "value: *external\n", output: "{\"value\":42}\n",
		},
		{
			name: "disable scalar", config: "disable", input: "[&a 1, *a]\n",
			failure: "aliases are disabled",
		},
		{
			name: "disable boolean", config: "false", input: "[&a 1, *a]\n",
			failure: "aliases are disabled",
		},
		{
			name: "disable mapping", config: "{disable: true}",
			input: "[&a 1, *a]\n", failure: "aliases are disabled",
		},
		{
			name: "disable selector", selector: "alias-data=disable",
			input: "[&a 1, *a]\n", failure: "aliases are disabled",
		},
		{
			name: "selector overrides config", config: "false", selector: "alias-data",
			input: "[&a 1, *a]\n", output: "[1,1]\n",
		},
		{
			name: "nested path", config: "{path: true, data: {x: {foo: {bar: 42}}}}",
			input: "copy: *x.foo.bar\n", output: "{\"copy\":42}\n",
		},
		{
			name: "automatic stream path", config: "{path: true}",
			input:  "--- &x {foo: 42}\n---\ncopy: *x.foo\n",
			output: "{\"foo\":42}\n{\"copy\":42}\n",
		},
		{
			name: "explicit scope for paths", config: "{path: true, stream: false}",
			input: "--- &x {foo: 42}\n---\ncopy: *x.foo\n", failure: "unknown anchor",
		},
		{
			name: "invalid path option", config: "{path: wrong}",
			input: "value: 1\n", failure: "path must be a boolean",
		},
		{
			name: "quoted numeric key", config: "{path: true}",
			input:  "base: &foo {'42': {bar: 42}}\ncopy: *foo.'42'.bar\n",
			output: "{\"base\":{\"42\":{\"bar\":42}},\"copy\":42}\n",
		},
		{
			name: "bare numeric mapping key", config: "{path: true}",
			input:   "base: &foo {'42': found}\ncopy: *foo.42\n",
			failure: "numeric mapping keys must be single-quoted",
		},
		{
			name: "leading zero index", config: "{path: true}",
			input:   "base: &foo [found]\ncopy: *foo.00\n",
			failure: "sequence index must not have leading zeros",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"-j"}
			if tc.config != "" {
				file := filepath.Join(t.TempDir(), "options.yaml")
				text := "plugin: {alias-data: " + tc.config + "}\n"
				if err := os.WriteFile(file, []byte(text), 0o600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "-C", file)
			}
			if tc.selector != "" {
				args = append(args, "--plugin="+tc.selector)
			}
			cmd := exec.Command(testBinary, args...)
			cmd.Stdin = strings.NewReader(tc.input)
			output, err := cmd.CombinedOutput()
			if tc.failure != "" {
				if err == nil || !strings.Contains(string(output), tc.failure) {
					t.Fatalf("got %q, %v, want %q", output, err, tc.failure)
				}
			} else if err != nil || string(output) != tc.output {
				t.Fatalf("got %q, %v, want %q", output, err, tc.output)
			}
		})
	}
}

func TestAliasDataCLIBypassModes(t *testing.T) {
	for _, flags := range [][]string{
		{"-t"},
		{"-T"},
		{"-e"},
		{"-E"},
		{"-Y"},
		{"-j", "--unmarshal"},
		{"-j", "--decode"},
	} {
		cmd := exec.Command(testBinary,
			append(flags, "--plugin=alias-data=disable")...)
		cmd.Stdin = strings.NewReader("value: 1\n")
		output, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(output), "alias-data") {
			t.Fatalf("unsupported mode %v accepted: %q, %v", flags, output, err)
		}
	}
}
