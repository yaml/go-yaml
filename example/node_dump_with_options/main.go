// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Node Dump With Options demonstrates node styles and output formatting.
package main

import (
	"fmt"
	"log"

	"go.yaml.in/yaml/v4"
)

func main() {
	config := map[string]any{
		"service": map[string]any{
			"host": "localhost",
			"port": 8080,
		},
		"enabled": "true",
	}

	// Quote styles are stored on the nodes built by Node.Dump.
	var node yaml.Node
	if err := node.Dump(config,
		yaml.WithQuotePreference(yaml.QuoteDouble),
	); err != nil {
		log.Fatal(err)
	}

	fmt.Println("1. Stored double-quoted style, default output indentation:")
	printNode(&node)

	// Indentation and document markers must be set at serialization time.
	fmt.Println("2. Same node, 4-space indentation and explicit start:")
	printNode(&node, yaml.WithIndent(4), yaml.WithExplicitStart())

	// Collection styles can also be stored on nodes.
	var flowNode yaml.Node
	if err := flowNode.Dump(config,
		yaml.WithFlowSimpleCollections(),
	); err != nil {
		log.Fatal(err)
	}
	fmt.Println("3. Flow style for the simple service mapping:")
	printNode(&flowNode)
}

func printNode(node *yaml.Node, opts ...yaml.Option) {
	data, err := yaml.Dump(node, opts...)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
}
