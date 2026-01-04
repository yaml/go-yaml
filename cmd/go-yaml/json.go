// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

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

// ProcessJSON reads YAML from reader and outputs JSON encoding
func ProcessJSON(reader io.Reader, pretty, unmarshalMode, decodeMode bool, opts []yaml.Option) error {
	if unmarshalMode {
		return processJSONUnmarshal(reader, pretty)
	}
	if decodeMode {
		return processJSONDecode(reader, pretty, nil) // Decode API doesn't support options
	}
	// Default: use Load API with options
	return processJSONLoad(reader, pretty, opts)
}

// processJSONLoad uses Loader.Load for YAML processing with options
func processJSONLoad(reader io.Reader, pretty bool, opts []yaml.Option) error {
	loader, err := yaml.NewLoader(reader, opts...)
	if err != nil {
		return fmt.Errorf("failed to create loader: %w", err)
	}

	for {
		// Read each document
		var data any
		err := loader.Load(&data)
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

// processJSONDecode uses deprecated Decoder.Decode for YAML processing (no options support)
func processJSONDecode(reader io.Reader, pretty bool, opts []yaml.Option) error {
	decoder := yaml.NewDecoder(reader) //nolint:staticcheck // Intentionally using deprecated API for --decode flag

	for {
		// Read each document
		var data any
		err := decoder.Decode(&data) //nolint:staticcheck // Deprecated API for --decode
		if err != nil {
			if err == io.EOF || err.Error() == "EOF" {
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
func processJSONUnmarshal(reader io.Reader, pretty bool) error {
	// Read all input from reader
	input, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	// Split input into documents
	documents := bytes.Split(input, []byte("---"))

	for _, doc := range documents {
		// Skip empty documents
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		// For unmarshal mode, always use `any` to avoid preserving comments
		var data any
		err := yaml.Load(doc, &data)
		if err != nil {
			return fmt.Errorf("failed to load YAML: %w", err)
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
