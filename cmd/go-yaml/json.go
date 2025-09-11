// Package main provides YAML to JSON conversion utilities for the go-yaml tool.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v4"
)

// ProcessJSON reads YAML from stdin and outputs JSON encoding
func ProcessJSON(pretty, unmarshal bool) error {
	if unmarshal {
		return processJSONUnmarshal(pretty)
	}
	return processJSONDecode(pretty)
}

// processJSONDecode uses Decoder.Decode for YAML processing
func processJSONDecode(pretty bool) error {
	decoder := yaml.NewDecoder(os.Stdin)

	for {
		// Read each document
		var data interface{}
		err := decoder.Decode(&data)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to decode YAML: %w", err)
		}

		// Encode as JSON
		encoder := json.NewEncoder(os.Stdout)
		if pretty {
			encoder.SetIndent("", "  ")
		}
		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	}

	return nil
}

// processJSONUnmarshal uses yaml.Unmarshal for YAML processing
func processJSONUnmarshal(pretty bool) error {
	// Read all input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Split input into documents
	documents := bytes.Split(input, []byte("---"))

	for _, doc := range documents {
		// Skip empty documents
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		// For unmarshal mode, always use interface{} to avoid preserving comments
		var data interface{}
		err := yaml.Unmarshal(doc, &data)
		if err != nil {
			return fmt.Errorf("failed to unmarshal YAML: %w", err)
		}

		// Encode as JSON
		encoder := json.NewEncoder(os.Stdout)
		if pretty {
			encoder.SetIndent("", "  ")
		}
		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	}

	return nil
}
