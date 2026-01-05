// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Example: Load Into Node demonstrates loading YAML into Node structures.

package main

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
)

func main() {
	fmt.Println("Example: Load into yaml.Node")

	yamlData := `# Application configuration
name: myapp  # The app name
version: 1.0.0

# Server settings
server:
  host: localhost
  port: 8080
  # Enable debug mode
  debug: true

# List of enabled features
features:
  - auth
  - logging
  - metrics
`

	loader, err := yaml.NewLoader(strings.NewReader(yamlData))
	if err != nil {
		panic(err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		panic(err)
	}

	fmt.Println("Successfully loaded YAML into Node")
	fmt.Printf("Root node kind: %v\n", node.Kind)
	fmt.Printf("Root has %d child nodes (the document)\n\n", len(node.Content))

	// The first content is the document node
	doc := node.Content[0]
	fmt.Printf("Document node kind: %v\n", doc.Kind)
	fmt.Printf("Document has %d children (key-value pairs)\n\n", len(doc.Content))

	// Walk through the top-level keys
	fmt.Println("Top-level keys with their values:")
	for i := 0; i < len(doc.Content); i += 2 {
		key := doc.Content[i]
		value := doc.Content[i+1]

		fmt.Printf("\nKey: %q\n", key.Value)
		if key.HeadComment != "" {
			fmt.Printf("  Head comment: %q\n", key.HeadComment)
		}
		if key.LineComment != "" {
			fmt.Printf("  Line comment: %q\n", key.LineComment)
		}

		switch value.Kind {
		case yaml.ScalarNode:
			fmt.Printf("  Value (scalar): %q\n", value.Value)
			if value.LineComment != "" {
				fmt.Printf("  Value line comment: %q\n", value.LineComment)
			}
		case yaml.MappingNode:
			fmt.Printf("  Value (mapping): %d key-value pairs\n", len(value.Content)/2)
			// Show nested keys
			for j := 0; j < len(value.Content); j += 2 {
				nestedKey := value.Content[j]
				nestedValue := value.Content[j+1]
				fmt.Printf("    %s: %s", nestedKey.Value, nestedValue.Value)
				if nestedValue.LineComment != "" {
					fmt.Printf("  %s", nestedValue.LineComment)
				}
				fmt.Println()
			}
		case yaml.SequenceNode:
			fmt.Printf("  Value (sequence): %d items\n", len(value.Content))
			for j, item := range value.Content {
				fmt.Printf("    [%d]: %s\n", j, item.Value)
			}
		}
	}

	// Demonstrate modifying the node
	fmt.Println("\n--- Modifying Node ---")
	// Add a new field programmatically
	newKey := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "environment",
	}
	newValue := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "production",
	}
	doc.Content = append(doc.Content, newKey, newValue)

	// Re-dump to see the modified YAML
	out, err := yaml.Dump(&node)
	if err != nil {
		panic(err)
	}

	fmt.Println("\nModified YAML (added 'environment' field):")
	fmt.Print(string(out))
}
