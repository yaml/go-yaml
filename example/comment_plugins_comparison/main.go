package main

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/comment/none"
	"go.yaml.in/yaml/v4/plugin/comment/v3"
	"go.yaml.in/yaml/v4/plugin/comment/v4"
)

func main() {
	fmt.Println("=== Comment Plugin Comparison ===")

	yamlWithComments := `# Application configuration
name: myapp  # Application name
version: 1.0.0

# Server settings
server:
  host: localhost
  port: 8080
`

	// Test 1: v3 comment plugin
	fmt.Println("1. v3 Comment Plugin:")
	testPlugin(yamlWithComments, v3.New(), "v3")

	// Test 2: v4 comment plugin
	fmt.Println("\n2. v4 Comment Plugin:")
	testPlugin(yamlWithComments, v4.New(), "v4")

	// Test 3: none plugin (strips comments)
	fmt.Println("\n3. None Plugin (strips comments):")
	testPlugin(yamlWithComments, none.New(), "none")

	// Test 4: No plugin (default behavior)
	fmt.Println("\n4. No Plugin (default behavior):")
	testNoPlugin(yamlWithComments)
}

func testPlugin(yamlData string, plugin yaml.Plugin, name string) {
	// Load with plugin
	loader, err := yaml.NewLoader(
		strings.NewReader(yamlData),
		yaml.WithPlugins(plugin),
	)
	if err != nil {
		panic(err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		panic(err)
	}

	// Check if comments were loaded
	hasComments := false
	if len(node.Content) > 0 && len(node.Content[0].Content) > 0 {
		if node.Content[0].Content[0].HeadComment != "" ||
			node.Content[0].Content[0].LineComment != "" {
			hasComments = true
		}
	}
	fmt.Printf("   Plugin: %s\n", name)
	fmt.Printf("   Comments loaded: %v\n", hasComments)

	// Dump with plugin
	var buf bytes.Buffer
	dumper, err := yaml.NewDumper(&buf, yaml.WithPlugins(plugin))
	if err != nil {
		panic(err)
	}

	if err := dumper.Dump(&node); err != nil {
		panic(err)
	}

	if err := dumper.Close(); err != nil {
		panic(err)
	}

	fmt.Printf("   Output:\n")
	output := buf.String()
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if i < 8 { // Show first 8 lines
			fmt.Printf("   %s\n", line)
		}
	}
}

func testNoPlugin(yamlData string) {
	// Load without plugin
	loader, err := yaml.NewLoader(strings.NewReader(yamlData))
	if err != nil {
		panic(err)
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		panic(err)
	}

	// Check if comments were loaded
	hasComments := false
	if len(node.Content) > 0 && len(node.Content[0].Content) > 0 {
		if node.Content[0].Content[0].HeadComment != "" ||
			node.Content[0].Content[0].LineComment != "" {
			hasComments = true
		}
	}
	fmt.Printf("   Comments loaded: %v\n", hasComments)

	// Dump without plugin
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

	fmt.Printf("   Output:\n")
	output := buf.String()
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if i < 8 { // Show first 8 lines
			fmt.Printf("   %s\n", line)
		}
	}
}
