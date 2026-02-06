// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Example: With V4 Option demonstrates using the V4 version preset.

package main

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Name     string            `yaml:"name"`
	Version  string            `yaml:"version"`
	Server   ServerConfig      `yaml:"server"`
	Tags     []string          `yaml:"tags,omitempty"`
	Metadata map[string]string `yaml:"metadata,omitempty"`
}

type ServerConfig struct {
	Host  string `yaml:"host"`
	Port  int    `yaml:"port"`
	Debug bool   `yaml:"debug"`
}

func main() {
	fmt.Println("Example: v4 Options (Now the Default)")
	fmt.Println("The new API (Dump, Load, NewDumper, NewLoader) defaults to v4:")
	fmt.Println("  - 2-space indentation")
	fmt.Println("  - Compact sequence indentation")

	yamlData := `# Application configuration
name: myapp
version: 1.0.0
# Server settings
server:
  host: localhost
  port: 8080
  debug: true
tags:
  - web
  - api
metadata:
  owner: platform-team
  env: prod
`

	// Load with default options (now v4)
	loader, err := yaml.NewLoader(strings.NewReader(yamlData))
	if err != nil {
		panic(err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		panic(err)
	}

	fmt.Println("--- Loaded with Default Options (v4) ---")
	fmt.Printf("Comments preserved: %v\n", node.Content[0].Content[0].HeadComment != "")
	fmt.Println()

	// Now dump it back with default options (v4: 2-space indent, compact)
	var buf bytes.Buffer
	dumper, err := yaml.NewDumper(&buf)
	if err != nil {
		panic(err)
	}

	if err := dumper.Dump(&node); err != nil {
		panic(err)
	}

	if err := dumper.Close(); err != nil {
		panic(err)
	}

	fmt.Println("--- Default Output (v4: 2-space indent, compact, comments preserved) ---")
	fmt.Print(buf.String())

	// Compare with v3 defaults for backward compatibility
	fmt.Println("\n--- For comparison: v3 defaults (4-space indent, non-compact) ---")
	buf.Reset()
	dumper2, err := yaml.NewDumper(&buf, yaml.WithV3Defaults())
	if err != nil {
		panic(err)
	}

	if err := dumper2.Dump(&node); err != nil {
		panic(err)
	}

	if err := dumper2.Close(); err != nil {
		panic(err)
	}

	fmt.Print(buf.String())
}
