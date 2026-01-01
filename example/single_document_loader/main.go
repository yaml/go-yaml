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
	fmt.Println("Example 3: Loader with WithSingleDocument Option")

	multiDoc := `---
name: app1
version: 1.0.0
---
name: app2
version: 2.0.0
tags:
  - experimental
---
name: app3
version: 3.0.0
`

	// Create loader that stops after first document
	loader, err := yaml.NewLoader(strings.NewReader(multiDoc), yaml.WithSingleDocument())
	if err != nil {
		panic(err)
	}

	var firstDoc Config
	if err := loader.Load(&firstDoc); err != nil {
		panic(err)
	}
	fmt.Printf("First document: %+v\n", firstDoc)

	// Try to load second document
	var secondDoc Config
	err = loader.Load(&secondDoc)
	if err == io.EOF {
		fmt.Println("Second Load() returned io.EOF (as expected with WithSingleDocument)")
	} else {
		fmt.Printf("Unexpected: got %v\n", err)
	}
}
