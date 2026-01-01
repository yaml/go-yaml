// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Name    string   `yaml:"name"`
	Version string   `yaml:"version"`
	Tags    []string `yaml:"tags,omitempty"`
}

func main() {
	fmt.Println("Example 9: Loader with Multiple Options")
	fmt.Println("(WithSingleDocument + WithKnownFields)")

	// First test: unknown field should fail
	multiDocWithUnknown := `---
name: app1
version: 1.0.0
unknownField: this-will-be-rejected
---
name: app2
version: 2.0.0
`

	loader1, err := yaml.NewLoader(
		strings.NewReader(multiDocWithUnknown),
		yaml.WithSingleDocument(),
		yaml.WithKnownFields(),
	)
	if err != nil {
		panic(err)
	}

	var strictCfg Config
	err = loader1.Load(&strictCfg)
	if err != nil {
		fmt.Printf("✓ Got expected error (unknown field): %v\n\n", err)
	} else {
		fmt.Printf("Unexpected: no error for unknown field\n\n")
	}

	// Second test: valid document with all options
	validDoc := `---
name: validapp
version: 1.0.0
tags:
  - valid
---
name: app2
version: 2.0.0
`

	loader2, err := yaml.NewLoader(
		strings.NewReader(validDoc),
		yaml.WithSingleDocument(),
		yaml.WithKnownFields(),
	)
	if err != nil {
		panic(err)
	}

	var validCfg Config
	if err := loader2.Load(&validCfg); err != nil {
		panic(err)
	}
	fmt.Printf("✓ First doc loaded: %+v\n", validCfg)

	// Second call should return EOF due to WithSingleDocument
	var secondCfg Config
	err = loader2.Load(&secondCfg)
	if err == io.EOF {
		fmt.Println("✓ Second Load() returned io.EOF (WithSingleDocument enforced)")
	} else {
		fmt.Printf("Unexpected: got %v\n", err)
	}
}
