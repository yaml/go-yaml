package main

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/comment/v3"
)

func main() {
	fmt.Println("=== Comment Plugin Comparison ===")

	yamlWithComments := `# Configuration file
# Version 1.0

# Application settings
app:
    name: demo  # The application name
    debug: true

# Server configuration
server:
    # Network settings
    port: 8080
    host: localhost
`

	fmt.Println("Original YAML:")
	fmt.Println(yamlWithComments)
	fmt.Println(strings.Repeat("=", 60))

	// Test 1: V3 preset + v3 comment plugin
	fmt.Println("\n1. V3 Preset + V3 Comment Plugin")
	testWithOptions(yamlWithComments, yaml.V3, yaml.WithPlugin(v3.New()))

	// Test 2: V4 preset (no comment plugin by default)
	fmt.Println("\n2. V4 Preset (no comment plugin)")
	testWithOptions(yamlWithComments, yaml.V4)

	// Test 3: Explicitly disable comment plugin with WithoutPlugin
	fmt.Println("\n3. Explicit WithoutPlugin(\"comment\")")
	testWithOptions(yamlWithComments, yaml.V3, yaml.WithoutPlugin("comment"))

	// Test 4: Explicit v3 plugin with V4 preset
	fmt.Println("\n4. V4 Preset + V3 Comment Plugin")
	testWithOptions(yamlWithComments, yaml.V4, yaml.WithPlugin(v3.New()))
}

func testWithOptions(yamlData string, opts ...yaml.Option) {
	var node yaml.Node
	err := yaml.Load([]byte(yamlData), &node, opts...)
	if err != nil {
		panic(err)
	}

	// Count comments
	commentCount := countComments(&node)
	fmt.Printf("   Comments loaded: %d\n", commentCount)

	// Dump back
	output, err := yaml.Dump(&node, opts...)
	if err != nil {
		panic(err)
	}

	// Show first 10 lines of output
	fmt.Println("   Output (first 10 lines):")
	lines := strings.Split(string(output), "\n")
	for i := 0; i < min(10, len(lines)); i++ {
		fmt.Printf("   %s\n", lines[i])
	}

	if commentCount > 0 {
		fmt.Println("   ✓ Comments preserved")
	} else {
		fmt.Println("   ✗ No comments (stripped)")
	}
}

func countComments(node *yaml.Node) int {
	if node == nil {
		return 0
	}

	count := 0
	if node.HeadComment != "" {
		count++
	}
	if node.LineComment != "" {
		count++
	}
	if node.FootComment != "" {
		count++
	}

	for _, child := range node.Content {
		count += countComments(child)
	}

	return count
}

//nolint:modernize // Keep custom min for Go 1.18 compatibility
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
