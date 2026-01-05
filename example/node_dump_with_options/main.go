// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Node Dump With Options demonstrates dumping Nodes with custom options.

package main

import (
	"fmt"
	"log"

	"go.yaml.in/yaml/v4"
)

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	TLS  bool   `yaml:"tls"`
}

func main() {
	config := ServerConfig{
		Host: "localhost",
		Port: 8080,
		TLS:  true,
	}

	fmt.Println("=== Node.Dump() with Different Options ===")

	// Example 1: Default (v4 style - 2-space indent, compact)
	var node1 yaml.Node
	if err := node1.Dump(&config); err != nil {
		log.Fatal(err)
	}
	fmt.Println("1. Default (v4):")
	printNode(&node1)

	// Example 2: With 4-space indent
	var node2 yaml.Node
	if err := node2.Dump(&config, yaml.WithIndent(4)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("2. With 4-space indent:")
	printNode(&node2)

	// Example 3: With v3 preset
	var node3 yaml.Node
	if err := node3.Dump(&config, yaml.V3); err != nil {
		log.Fatal(err)
	}
	fmt.Println("3. With V3 preset:")
	printNode(&node3)

	// Example 4: Multiple options combined
	var node4 yaml.Node
	if err := node4.Dump(&config,
		yaml.WithIndent(2),
		yaml.WithExplicitStart(),
	); err != nil {
		log.Fatal(err)
	}
	fmt.Println("4. With explicit start marker:")
	printNode(&node4)
}

func printNode(node *yaml.Node) {
	data, err := yaml.Dump(node)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
}
