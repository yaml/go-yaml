// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

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
	cfg := Config{
		Name:    "myapp",
		Version: "1.0.0",
		Server: ServerConfig{
			Host:  "localhost",
			Port:  8080,
			Debug: true,
		},
		Tags: []string{"web", "api", "production"},
		Metadata: map[string]string{
			"owner": "platform-team",
			"env":   "prod",
		},
	}

	// Check if indent level was provided as command-line argument
	if len(os.Args) > 1 {
		indent, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid indent value %q (must be a number)\n", os.Args[1])
			os.Exit(1)
		}

		if indent < 2 || indent > 9 {
			fmt.Fprintf(os.Stderr, "Error: indent value must be between 2 and 9 (got %d)\n", indent)
			os.Exit(1)
		}

		fmt.Printf("Example: Dumper with %d-space indent\n\n", indent)

		var buf bytes.Buffer
		dumper, err := yaml.NewDumper(&buf, yaml.WithIndent(indent))
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
		return
	}

	// No argument provided - show comparison of different indent levels
	fmt.Println("Example: Dumper with Different Indent Levels")
	fmt.Println("(Run with argument to test specific indent: go run dumper_indent_comparison.go 3)")

	indentLevels := []int{2, 4, 8}

	for _, indent := range indentLevels {
		fmt.Printf("--- Indent: %d spaces ---\n", indent)

		var buf bytes.Buffer
		dumper, err := yaml.NewDumper(&buf, yaml.WithIndent(indent))
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
		fmt.Println()
	}

	// Also show default indent (no option)
	fmt.Println("--- Default indent (no option) ---")
	var buf bytes.Buffer
	dumper, err := yaml.NewDumper(&buf)
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
	fmt.Println("\nTip: Run with a specific indent value as an argument:")
	fmt.Println("  go run example/dumper_indent_comparison.go 3")
}
