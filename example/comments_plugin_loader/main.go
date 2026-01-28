package main

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/comment/v3"
)

func main() {
	fmt.Println("Example 7: Loader with Comments Plugin")

	yamlWithComments := `# This is the app configuration
name: myapp  # Application name
version: 1.0.0

# List of tags
tags:
  - web
  - api
`

	commentPlugin := v3.New()
	loader, err := yaml.NewLoader(
		strings.NewReader(yamlWithComments),
		yaml.WithPlugin(commentPlugin),
	)
	if err != nil {
		panic(err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		panic(err)
	}

	fmt.Println("Node with comments loaded successfully")
	fmt.Printf("Document has %d top-level items\n", len(node.Content[0].Content))

	if len(node.Content[0].Content) > 0 && node.Content[0].Content[0].HeadComment != "" {
		fmt.Printf("First field head comment: %q\n", node.Content[0].Content[0].HeadComment)
	}
}
