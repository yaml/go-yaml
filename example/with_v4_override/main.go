// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Name   string            `yaml:"name"`
	Server map[string]string `yaml:"server"`
	Tags   []string          `yaml:"tags"`
}

func main() {
	fmt.Println("Example: Overriding yaml.V4 options")

	cfg := Config{
		Name: "myapp",
		Server: map[string]string{
			"host": "localhost",
			"port": "8080",
		},
		Tags: []string{"web", "api"},
	}

	// Test 1: v4 options - should use 2-space indent
	fmt.Println("1. yaml.V4 - 2-space indent:")
	var buf bytes.Buffer
	dumper, err := yaml.NewDumper(&buf, yaml.V4)
	if err != nil {
		panic(err)
	}
	if err := dumper.Dump(&cfg); err != nil {
		panic(err)
	}
	if err := dumper.Close(); err != nil {
		panic(err)
	}
	fmt.Print(buf.String())

	// Test 2: v4 options then WithIndent(3) - WithIndent(3) overrides
	fmt.Println("\n2. v4 options, then WithIndent(3) - should use 3-space indent:")
	buf.Reset()
	dumper2, err := yaml.NewDumper(&buf, yaml.V4, yaml.WithIndent(3))
	if err != nil {
		panic(err)
	}
	if err := dumper2.Dump(&cfg); err != nil {
		panic(err)
	}
	if err := dumper2.Close(); err != nil {
		panic(err)
	}
	fmt.Print(buf.String())

	// Test 3: WithIndent(5) then v4 options - v4 options override to 2
	fmt.Println("\n3. WithIndent(5), then v4 options - should use 2-space indent (v4 wins):")
	buf.Reset()
	dumper3, err := yaml.NewDumper(&buf, yaml.WithIndent(5), yaml.V4)
	if err != nil {
		panic(err)
	}
	if err := dumper3.Dump(&cfg); err != nil {
		panic(err)
	}
	if err := dumper3.Close(); err != nil {
		panic(err)
	}
	fmt.Print(buf.String())

	fmt.Println("\nConclusion: Options are applied left-to-right, later options override earlier ones.")
}
