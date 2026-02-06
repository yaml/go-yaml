// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Example: Version Options demonstrates using YAML version presets.

package main

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Name  string   `yaml:"name"`
	Items []string `yaml:"items"`
}

func main() {
	cfg := Config{
		Name:  "test",
		Items: []string{"apple", "banana", "cherry"},
	}

	fmt.Println("Example: Comparing v2, v3, and v4 option presets")

	// v2 options - 2-space indent, non-compact sequences
	fmt.Println("=== yaml.WithV2Defaults() - 2-space indent, non-compact sequences ===")
	out, _ := yaml.Dump(&cfg, yaml.WithV2Defaults())
	fmt.Print(string(out))

	// v3 options - 4-space indent (default), non-compact sequences
	fmt.Println("=== yaml.WithV3Defaults() - 4-space indent, non-compact sequences ===")
	out, _ = yaml.Dump(&cfg, yaml.WithV3Defaults())
	fmt.Print(string(out))

	// v4 options - 2-space indent, compact sequences
	fmt.Println("=== yaml.WithV4Defaults() - 2-space indent, compact sequences ===")
	out, _ = yaml.Dump(&cfg, yaml.WithV4Defaults())
	fmt.Print(string(out))

	// Override v4 options
	fmt.Println("=== yaml.WithV4Defaults() with WithIndent(3) override ===")
	out, _ = yaml.Dump(&cfg, yaml.WithV4Defaults(), yaml.WithIndent(3))
	fmt.Print(string(out))

	fmt.Println("\nNotice how:")
	fmt.Println("- v2 and v4 both use 2-space indentation")
	fmt.Println("- v3 uses 4-space indentation (classic go-yaml v3 style)")
	fmt.Println("- v4 uses compact sequences (items: flows from dash)")
	fmt.Println("- Options can be combined and later ones win")
}
