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

// processYAMLDecode uses Decoder.Decode for YAML processing
func processYAMLDecode(preserve, marshal bool) error {
	if preserve {
		// Preserve comments and styles by using yaml.Node
		decoder := yaml.NewDecoder(os.Stdin)
		firstDoc := true

		for {
			var node yaml.Node
			err := decoder.Decode(&node)
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
				// Use Marshal for output
				output, err := yaml.Marshal(outNode)
				if err != nil {
					return fmt.Errorf("failed to marshal YAML: %w", err)
				}
				fmt.Print(string(output))
			} else {
				// Use Encoder for output
				encoder := yaml.NewEncoder(os.Stdout)
				encoder.SetIndent(2)
				if err := encoder.Encode(outNode); err != nil {
					encoder.Close()
					return fmt.Errorf("failed to encode YAML: %w", err)
				}
				encoder.Close()
			}
		}
	} else {
		// Don't preserve comments and styles - use interface{} for clean output
		decoder := yaml.NewDecoder(os.Stdin)
		firstDoc := true

		for {
			var data interface{}
			err := decoder.Decode(&data)
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
				// Use Marshal for output
				output, err := yaml.Marshal(data)
				if err != nil {
					return fmt.Errorf("failed to marshal YAML: %w", err)
				}
				fmt.Print(string(output))
			} else {
				// Use Encoder for output
				encoder := yaml.NewEncoder(os.Stdout)
				encoder.SetIndent(2)
				if err := encoder.Encode(data); err != nil {
					encoder.Close()
					return fmt.Errorf("failed to encode YAML: %w", err)
				}
				encoder.Close()
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
			if err := yaml.Unmarshal(doc, &node); err != nil {
				return fmt.Errorf("failed to unmarshal YAML: %w", err)
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
				// Use Marshal for output
				output, err := yaml.Marshal(outNode)
				if err != nil {
					return fmt.Errorf("failed to marshal YAML: %w", err)
				}
				fmt.Print(string(output))
			} else {
				// Use Encoder for output
				encoder := yaml.NewEncoder(os.Stdout)
				encoder.SetIndent(2)
				if err := encoder.Encode(outNode); err != nil {
					encoder.Close()
					return fmt.Errorf("failed to encode YAML: %w", err)
				}
				encoder.Close()
			}
		} else {
			// For unmarshal mode with -y (not -Y), always use interface{} to avoid preserving comments
			var data interface{}
			if err := yaml.Unmarshal(doc, &data); err != nil {
				return fmt.Errorf("failed to unmarshal YAML: %w", err)
			}

			if marshal {
				// Use Marshal for output
				output, err := yaml.Marshal(data)
				if err != nil {
					return fmt.Errorf("failed to marshal YAML: %w", err)
				}
				fmt.Print(string(output))
			} else {
				// Use Encoder for output
				encoder := yaml.NewEncoder(os.Stdout)
				encoder.SetIndent(2)
				if err := encoder.Encode(data); err != nil {
					encoder.Close()
					return fmt.Errorf("failed to encode YAML: %w", err)
				}
				encoder.Close()
			}
		}
	}

	return nil
}
