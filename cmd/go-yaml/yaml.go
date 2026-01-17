// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package main provides YAML formatting utilities for the go-yaml tool.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v4"
)

// ProcessYAML reads YAML from reader and outputs formatted YAML
func ProcessYAML(reader io.Reader, preserve, unmarshalMode, decodeMode, marshalMode, encodeMode bool, opts []yaml.Option) error {
	if unmarshalMode {
		return processYAMLUnmarshal(reader, preserve, marshalMode)
	}
	if decodeMode {
		return processYAMLDecode(reader, preserve, encodeMode, nil) // Decode API doesn't support options
	}
	// Default: use Load API with options
	return processYAMLLoad(reader, preserve, marshalMode, encodeMode, opts)
}

// processYAMLLoad uses Loader.Load for YAML processing with options
func processYAMLLoad(reader io.Reader, preserve, marshal, encode bool, opts []yaml.Option) error {
	if preserve {
		// Preserve comments and styles by using yaml.Node
		loader, err := yaml.NewLoader(reader, opts...)
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
				// Use Marshal for output (no options)
				output, err := yaml.Marshal(outNode) //nolint:staticcheck // Intentionally using deprecated API for --marshal flag
				if err != nil {
					return fmt.Errorf("failed to marshal YAML: %w", err)
				}
				fmt.Print(string(output))
			} else if encode {
				// Use Encoder for output (no options)
				enc := yaml.NewEncoder(os.Stdout)           //nolint:staticcheck // Intentionally using deprecated API for --encode flag
				if err := enc.Encode(outNode); err != nil { //nolint:staticcheck // Deprecated API for --encode
					enc.Close() //nolint:staticcheck // Deprecated API for --encode
					return fmt.Errorf("failed to encode YAML: %w", err)
				}
				enc.Close() //nolint:staticcheck // Deprecated API for --encode
			} else {
				// Use Dumper for output with options
				dumper, err := yaml.NewDumper(os.Stdout, opts...)
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
		loader, err := yaml.NewLoader(reader, opts...)
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
				// Use Marshal for output (no options)
				output, err := yaml.Marshal(data) //nolint:staticcheck // Intentionally using deprecated API for --marshal flag
				if err != nil {
					return fmt.Errorf("failed to marshal YAML: %w", err)
				}
				fmt.Print(string(output))
			} else if encode {
				// Use Encoder for output (no options)
				enc := yaml.NewEncoder(os.Stdout)        //nolint:staticcheck // Intentionally using deprecated API for --encode flag
				if err := enc.Encode(data); err != nil { //nolint:staticcheck // Deprecated API for --encode
					enc.Close() //nolint:staticcheck // Deprecated API for --encode
					return fmt.Errorf("failed to encode YAML: %w", err)
				}
				enc.Close() //nolint:staticcheck // Deprecated API for --encode
			} else {
				// Use Dumper for output with options
				dumper, err := yaml.NewDumper(os.Stdout, opts...)
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

// processYAMLDecode uses deprecated Decoder.Decode for YAML processing (no options support)
func processYAMLDecode(reader io.Reader, preserve, encode bool, opts []yaml.Option) error {
	decoder := yaml.NewDecoder(reader) //nolint:staticcheck // Intentionally using deprecated API for --decode flag
	firstDoc := true

	for {
		if preserve {
			var node yaml.Node
			err := decoder.Decode(&node) //nolint:staticcheck // Deprecated API for --decode
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

			if encode {
				// Use Encoder for output
				enc := yaml.NewEncoder(os.Stdout)           //nolint:staticcheck // Intentionally using deprecated API for --encode flag
				if err := enc.Encode(outNode); err != nil { //nolint:staticcheck // Deprecated API for --encode
					enc.Close() //nolint:staticcheck // Deprecated API for --encode
					return fmt.Errorf("failed to encode YAML: %w", err)
				}
				enc.Close() //nolint:staticcheck // Deprecated API for --encode
			} else {
				// Default output (no options for deprecated Decode API)
				output, err := yaml.Marshal(outNode) //nolint:staticcheck // Deprecated API for --decode
				if err != nil {
					return fmt.Errorf("failed to marshal YAML: %w", err)
				}
				fmt.Print(string(output))
			}
		} else {
			var data any
			err := decoder.Decode(&data) //nolint:staticcheck // Deprecated API for --decode
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

			if encode {
				// Use Encoder for output
				enc := yaml.NewEncoder(os.Stdout)        //nolint:staticcheck // Intentionally using deprecated API for --encode flag
				if err := enc.Encode(data); err != nil { //nolint:staticcheck // Deprecated API for --encode
					enc.Close() //nolint:staticcheck // Deprecated API for --encode
					return fmt.Errorf("failed to encode YAML: %w", err)
				}
				enc.Close() //nolint:staticcheck // Deprecated API for --encode
			} else {
				// Default output (no options for deprecated Decode API)
				output, err := yaml.Marshal(data) //nolint:staticcheck // Deprecated API for --decode
				if err != nil {
					return fmt.Errorf("failed to marshal YAML: %w", err)
				}
				fmt.Print(string(output))
			}
		}
	}

	return nil
}

// processYAMLUnmarshal uses yaml.Unmarshal for YAML processing
func processYAMLUnmarshal(reader io.Reader, preserve, marshal bool) error {
	// Read all input from reader
	input, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
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
