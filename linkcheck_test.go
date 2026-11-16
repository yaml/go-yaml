// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestUnmarshalDoesNotLinkPluginFactories guards against the plugin registry
// being built by a package initializer again. A program that only calls
// Unmarshal must not carry the YAML-configuration factories of the built-in
// plugins; see https://github.com/yaml/go-yaml/issues/513.
func TestUnmarshalDoesNotLinkPluginFactories(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a program with the go command")
	}
	goCmd := filepath.Join(runtime.GOROOT(), "bin", "go")
	bin := filepath.Join(t.TempDir(), "linkcheck")
	build := exec.Command(goCmd, "build", "-o", bin, "./testdata/linkcheck")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	out, err := exec.Command(goCmd, "tool", "nm", bin).CombinedOutput()
	if err != nil {
		t.Fatalf("go tool nm: %v\n%s", err, out)
	}
	symbols := string(out)

	// Witness: the program really links the package, so an empty or
	// unrelated symbol table cannot pass the test below.
	if !strings.Contains(symbols, "go.yaml.in/yaml/v4.Unmarshal") {
		t.Fatalf("go.yaml.in/yaml/v4.Unmarshal is not in the symbol table; the check below would prove nothing")
	}

	for _, unwanted := range []string{
		"go.yaml.in/yaml/v4.newPluginRegistry",
		"go.yaml.in/yaml/v4/plugin/aliasdata.NewFromYAML",
		"go.yaml.in/yaml/v4/plugin/mergekey.NewFromYAML",
		"go.yaml.in/yaml/v4/plugin/loaderlimits.NewFromYAML",
		"go.yaml.in/yaml/v4/plugin/errorformat/v4.NewFromYAML",
	} {
		if strings.Contains(symbols, unwanted) {
			t.Errorf("%s is linked into a program that only calls Unmarshal", unwanted)
		}
	}
}
