// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Example: Basic Loader demonstrates simple YAML loading into structs.

package main

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Name    string   `yaml:"name"`
	Version string   `yaml:"version"`
	Tags    []string `yaml:"tags,omitempty"`
}

func main() {
	fmt.Println("Example 1: Basic Loader - Single Document")

	yamlData := `name: myapp
version: 1.0.0
tags:
  - web
  - api
`

	var cfg Config
	loader, err := yaml.NewLoader(strings.NewReader(yamlData))
	if err != nil {
		panic(err)
	}

	if err := loader.Load(&cfg); err != nil {
		panic(err)
	}

	fmt.Printf("Loaded: %+v\n", cfg)
}
