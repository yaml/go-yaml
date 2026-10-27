// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenerateExports(t *testing.T) {
	path := "go.yaml.in/yaml/v4/cmd/functiontag-gen/testdata/functions"
	first, err := generate(path, "example", "Library", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := generate(path, "example", "Library", "")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("nondeterministic generation")
	}
	text := string(first)
	if !strings.Contains(text, `"Add":`) || !strings.Contains(text, `"Effect":`) || strings.Contains(text, `library.Generic`) || strings.Contains(text, `library.Multiple`) {
		t.Fatal(text)
	}
	for _, symbol := range []string{"Generic", "Multiple", "Missing"} {
		if _, err := generate(path, "example", "Library", symbol); err == nil {
			t.Fatalf("%s should fail", symbol)
		}
	}
}
