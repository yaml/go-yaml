// Package main provides YAML formatting utilities for the go-yaml tool.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v4"
)

// ProcessYAML reads YAML from stdin and outputs formatted YAML
func ProcessYAML(preserve, unmarshal, marshal bool) error {
	if unmarshal {
		return processYAMLUnmarshal(preserve, marshal)
	}
	return processYAMLDecode(preserve, marshal)
}

// processYAMLDecode uses Loader.Load for YAML processing
func processYAMLDecode(preserve, marshal bool) error {
	if preserve {
		// Preserve comments and styles by using yaml.Node
		loader, err := yaml.NewLoader(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to create loader: %w", err)
		}
		firstDoc := true

		for {
			var node yaml.Node
			err := loader.Load(&node)
			if err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("failed to decode YAML: %w", err)
			}

			// Add document separator for all documents except the first
			if !firstDoc {
				fmt.Println("---")
			}
			firstDoc = false

			// If the node is not a DocumentNode, wrap it in one
			var outNode *yaml.Node
			if node.Kind == yaml.DocumentNode {
				outNode = &node
			} else {
				outNode = &yaml.Node{
					Kind:    yaml.DocumentNode,
					Content: []*yaml.Node{&node},
				}
			}

			if marshal {
				// Use Dump for output
				output, err := yaml.Dump(outNode)
				if err != nil {
					return fmt.Errorf("failed to dump YAML: %w", err)
				}
				fmt.Print(string(output))
			} else {
				// Use Dumper for output
				dumper, err := yaml.NewDumper(os.Stdout)
				if err != nil {
					return fmt.Errorf("failed to create dumper: %w", err)
				}
				if err := dumper.Dump(outNode); err != nil {
					dumper.Close()
					return fmt.Errorf("failed to dump YAML: %w", err)
				}
				dumper.Close()
			}
		}
	} else {
		// Don't preserve comments and styles - use `any` for clean output
		loader, err := yaml.NewLoader(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to create loader: %w", err)
		}
		firstDoc := true

		for {
			var data any
			err := loader.Load(&data)
			if err != nil {
				if err == io.EOF || err.Error() == "EOF" {
					break
				}
				return fmt.Errorf("failed to decode YAML: %w", err)
			}

			// Add document separator for all documents except the first
			if !firstDoc {
				fmt.Println("---")
			}
			firstDoc = false

			if marshal {
				// Use Dump for output
				output, err := yaml.Dump(data)
				if err != nil {
					return fmt.Errorf("failed to dump YAML: %w", err)
				}
				fmt.Print(string(output))
			} else {
				// Use Dumper for output
				dumper, err := yaml.NewDumper(os.Stdout)
				if err != nil {
					return fmt.Errorf("failed to create dumper: %w", err)
				}
				if err := dumper.Dump(data); err != nil {
					dumper.Close()
					return fmt.Errorf("failed to dump YAML: %w", err)
				}
				dumper.Close()
			}
		}
	}

	return nil
}

// processYAMLUnmarshal uses yaml.Unmarshal for YAML processing
func processYAMLUnmarshal(preserve, marshal bool) error {
	// Read all input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Split input into documents
	documents := bytes.Split(input, []byte("---"))
	firstDoc := true

	for _, doc := range documents {
		// Skip empty documents
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		// Add document separator for all documents except the first
		if !firstDoc {
			fmt.Println("---")
		}
		firstDoc = false

		if preserve {
			// Preserve comments and styles by using yaml.Node
			var node yaml.Node
			if err := yaml.Load(doc, &node); err != nil {
				return fmt.Errorf("failed to load YAML: %w", err)
			}

			// If the node is not a DocumentNode, wrap it in one
			var outNode *yaml.Node
			if node.Kind == yaml.DocumentNode {
				outNode = &node
			} else {
				outNode = &yaml.Node{
					Kind:    yaml.DocumentNode,
					Content: []*yaml.Node{&node},
				}
			}

			if marshal {
				// Use Dump for output
				output, err := yaml.Dump(outNode)
				if err != nil {
					return fmt.Errorf("failed to dump YAML: %w", err)
				}
				fmt.Print(string(output))
			} else {
				// Use Dumper for output
				dumper, err := yaml.NewDumper(os.Stdout)
				if err != nil {
					return fmt.Errorf("failed to create dumper: %w", err)
				}
				if err := dumper.Dump(outNode); err != nil {
					dumper.Close()
					return fmt.Errorf("failed to dump YAML: %w", err)
				}
				dumper.Close()
			}
		} else {
			// For unmarshal mode with -y (not -Y), always use `any` to avoid preserving comments
			var data any
			if err := yaml.Load(doc, &data); err != nil {
				return fmt.Errorf("failed to load YAML: %w", err)
			}

			if marshal {
				// Use Dump for output
				output, err := yaml.Dump(data)
				if err != nil {
					return fmt.Errorf("failed to dump YAML: %w", err)
				}
				fmt.Print(string(output))
			} else {
				// Use Dumper for output
				dumper, err := yaml.NewDumper(os.Stdout)
				if err != nil {
					return fmt.Errorf("failed to create dumper: %w", err)
				}
				if err := dumper.Dump(data); err != nil {
					dumper.Close()
					return fmt.Errorf("failed to dump YAML: %w", err)
				}
				dumper.Close()
			}
		}
	}

	return nil
}
