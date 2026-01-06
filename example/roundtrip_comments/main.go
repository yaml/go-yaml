package main

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/comment/v3"
)

func main() {
	fmt.Println("Example 8: Round-trip with Comments (Load + Dump)")

	yamlWithComments := `# This is the app configuration
name: myapp  # Application name
version: 1.0.0

# List of tags
tags:
  - web
  - api
`

	commentPlugin := v3.New()

	// Load with comments
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

	// Dump with comments preserved
	var buf bytes.Buffer
	dumper, err := yaml.NewDumper(&buf, yaml.WithPlugin(commentPlugin))
	if err != nil {
		panic(err)
	}

	if err := dumper.Dump(&node); err != nil {
		panic(err)
	}

	if err := dumper.Close(); err != nil {
		panic(err)
	}

	fmt.Printf("Comments preserved in output:\n%s", buf.String())
}
