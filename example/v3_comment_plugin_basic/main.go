package main

import (
	"fmt"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/comment/v3"
)

func main() {
	fmt.Println("=== V3 Comment Plugin: Basic Usage ===")

	yamlWithComments := `# Application configuration
name: myapp  # Application name
version: 1.0.0

# Server settings
server:
    host: localhost  # Local development
    port: 8080

# Database configuration
database:
    # Connection details
    host: db.example.com
    port: 5432
`

	fmt.Println("Original YAML with comments:")
	fmt.Println(yamlWithComments)
	fmt.Println("---")

	// Load with V3 preset + v3 comment plugin
	var data map[string]interface{}
	err := yaml.Load([]byte(yamlWithComments), &data, yaml.V3, yaml.WithPlugin(v3.New()))
	if err != nil {
		panic(err)
	}

	fmt.Println("\nLoaded data structure:")
	fmt.Printf("%+v\n", data)

	// Now load into a Node to preserve comments
	var node yaml.Node
	err = yaml.Load([]byte(yamlWithComments), &node, yaml.V3, yaml.WithPlugin(v3.New()))
	if err != nil {
		panic(err)
	}

	fmt.Println("\n--- Node with Comments ---")
	fmt.Printf("Root node has %d document(s)\n", len(node.Content))
	if len(node.Content) > 0 {
		doc := node.Content[0]
		fmt.Printf("Document node has %d children\n", len(doc.Content))

		// Show comments on first few nodes
		for i := 0; i < min(4, len(doc.Content)); i++ {
			child := doc.Content[i]
			if child.HeadComment != "" {
				fmt.Printf("\nNode %d HeadComment: %q\n", i, child.HeadComment)
			}
			if child.LineComment != "" {
				fmt.Printf("Node %d LineComment: %q\n", i, child.LineComment)
			}
			if child.Value != "" {
				fmt.Printf("Node %d Value: %q\n", i, child.Value)
			}
		}
	}

	// Dump back with comments preserved
	output, err := yaml.Dump(&node, yaml.V3, yaml.WithPlugin(v3.New()))
	if err != nil {
		panic(err)
	}

	fmt.Println("\n--- Round-trip Output (Comments Preserved) ---")
	fmt.Println(string(output))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
