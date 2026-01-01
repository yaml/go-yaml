// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log"

	"go.yaml.in/yaml/v4"
)

// Config with strict field checking in custom unmarshaler
type Config struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

// UnmarshalYAML implements custom unmarshaling with strict field checking
func (c *Config) UnmarshalYAML(node *yaml.Node) error {
	type plain Config
	// Use node.Load() to preserve WithKnownFields option
	// This solves Issue #460 - node.Decode() would lose this option
	return node.Load((*plain)(c), yaml.WithKnownFields())
}

func main() {
	fmt.Println("=== Node.Load() with WithKnownFields() ===")

	// Valid YAML - should succeed
	validYAML := `
name: myapp
port: 8080
`

	var config1 Config
	err := yaml.Load([]byte(validYAML), &config1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Valid config loaded: %+v\n\n", config1)

	// Invalid YAML with unknown field - should fail
	// Note: "prto" is intentionally misspelled (should be "port") to demonstrate error detection
	invalidYAML := `
name: myapp
prto: 8080
unknown: field
`

	var config2 Config
	err = yaml.Load([]byte(invalidYAML), &config2)
	if err != nil {
		fmt.Printf("✓ Expected error caught: %v\n", err)
	} else {
		fmt.Println("✗ ERROR: Should have failed on unknown fields!")
	}
}
