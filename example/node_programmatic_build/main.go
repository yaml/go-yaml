package main

import (
	"fmt"
	"log"

	"go.yaml.in/yaml/v4"
)

func main() {
	fmt.Println("=== Building YAML Nodes Programmatically ===")

	// Build a complex structure
	environments := map[string]interface{}{
		"development": map[string]interface{}{
			"database": "dev.db",
			"debug":    true,
		},
		"production": map[string]interface{}{
			"database": "prod.db",
			"debug":    false,
		},
	}

	// Convert to node with specific formatting
	var node yaml.Node
	err := node.Dump(environments,
		yaml.WithIndent(2),
		yaml.WithCompactSeqIndent(),
		yaml.WithExplicitStart(),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Now we can manipulate the node or output it
	fmt.Println("Generated YAML with custom formatting:")
	data, err := yaml.Dump(&node)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)

	// We can also load it back with options
	var result map[string]interface{}
	if err := node.Load(&result); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Loaded back: %+v\n", result)
}
