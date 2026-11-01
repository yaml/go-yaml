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

func TestCoreConsumersWithoutTextTemplate(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, imports, body string
	}{
		{
			name: "Unmarshal",
			body: `var out any
				if err := yaml.Unmarshal([]byte("a: 1\n"), &out); err != nil {
					panic(err)
				}`,
		},
		{
			name: "Load",
			body: `var out any
				if err := yaml.Load([]byte("a: 1\n"), &out); err != nil {
					panic(err)
				}`,
		},
		{
			name:    "NewEncoder",
			imports: `import "io"`,
			body: `enc := yaml.NewEncoder(io.Discard)
				if err := enc.Encode(map[string]int{"a": 1}); err != nil {
					panic(err)
				}
				if err := enc.Close(); err != nil {
					panic(err)
				}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			module := fmt.Sprintf("module consumer\n\ngo 1.18\n\n"+
				"require go.yaml.in/yaml/v4 v4.0.0\n"+
				"replace go.yaml.in/yaml/v4 => %q\n", root)
			source := "package main\nimport \"go.yaml.in/yaml/v4\"\n" +
				tc.imports + "\nfunc main() {\n" + tc.body + "\n}\n"
			for name, data := range map[string]string{
				"go.mod": module, "main.go": source,
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			runGo := func(args ...string) []byte {
				t.Helper()
				cmd := exec.Command(goTool, args...)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "GOWORK=off")
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("go %v: %v\n%s", args, err, out)
				}
				return out
			}
			for _, dependency := range strings.Fields(string(runGo("list", "-deps", "."))) {
				if dependency == "text/template" {
					t.Fatal("core consumer imports text/template, preventing dead-code elimination")
				}
			}
			binary := filepath.Join(dir, "consumer")
			runGo("build", "-o", binary, ".")
			if out, err := exec.Command(binary).CombinedOutput(); err != nil {
				t.Fatalf("consumer: %v\n%s", err, out)
			}
		})
	}
}
