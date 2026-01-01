// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Name    string   `yaml:"name"`
	Version string   `yaml:"version"`
	Tags    []string `yaml:"tags,omitempty"`
}

func main() {
	fmt.Println("Example 6: Dumper with WithIndent Option")

	var buf bytes.Buffer
	dumper, err := yaml.NewDumper(&buf, yaml.WithIndent(4))
	if err != nil {
		panic(err)
	}

	cfg := Config{
		Name:    "service",
		Version: "1.0.0",
		Tags:    []string{"a", "b", "c"},
	}

	if err := dumper.Dump(&cfg); err != nil {
		panic(err)
	}

	if err := dumper.Close(); err != nil {
		panic(err)
	}

	fmt.Printf("Output (4-space indent):\n%s", buf.String())
}
