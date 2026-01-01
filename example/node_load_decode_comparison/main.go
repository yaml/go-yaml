// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

// DecodeUnmarshaler uses node.Decode() (loses options)
type DecodeUnmarshaler struct {
	Config
}

func (d *DecodeUnmarshaler) UnmarshalYAML(node *yaml.Node) error {
	type plain DecodeUnmarshaler
	return node.Load((*plain)(d)) // Options lost!
}

// LoadUnmarshaler uses node.Load() (preserves options)
type LoadUnmarshaler struct {
	Config
}

func (l *LoadUnmarshaler) UnmarshalYAML(node *yaml.Node) error {
	type plain LoadUnmarshaler
	return node.Load((*plain)(l), yaml.WithKnownFields()) // Options preserved!
}

func main() {
	// YAML with an unknown field
	yamlData := `
name: myapp
port: 8080
unknown: field
`

	fmt.Println("=== Comparing node.Decode() vs node.Load() ===")

	// Test 1: Using node.Decode() - unknown field ignored
	fmt.Println("1. Using node.Decode() (old way):")
	var config1 DecodeUnmarshaler
	err := yaml.Load([]byte(yamlData), &config1)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   ✗ No error - unknown field was ignored\n")
		fmt.Printf("   Loaded: %+v\n", config1)
	}

	fmt.Println()

	// Test 2: Using node.Load() with WithKnownFields() - unknown field rejected
	fmt.Println("2. Using node.Load() with WithKnownFields() (new way):")
	var config2 LoadUnmarshaler
	err = yaml.Load([]byte(yamlData), &config2)
	if err != nil {
		fmt.Printf("   ✓ Error caught: %v\n", err)
	} else {
		fmt.Printf("   ✗ No error - this shouldn't happen!\n")
	}
}
