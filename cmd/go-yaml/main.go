// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// This binary provides a YAML node inspection tool that reads YAML from stdin
// and outputs a detailed analysis of its node structure, including comments
// and content organization.

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"go.yaml.in/yaml/v4"
)

const version = "4.0.0.1"

// hasUsefulContent checks if a StreamNode has meaningful content to display.
// For non-StreamNodes, always returns true.
func hasUsefulContent(n *yaml.Node) bool {
	if n.Kind != yaml.StreamNode {
		return true
	}
	// For now, only directives count as useful (comments not yet on StreamNodes)
	return n.Version != nil || len(n.TagDirectives) > 0
}

// main reads YAML from stdin, parses it, and outputs the node structure
func main() {
	// Parse command line flags
	showHelp := flag.Bool("h", false, "Show this help information")
	showVersion := flag.Bool("version", false, "Show version information")

	// YAML modes
	yamlMode := flag.Bool("y", false, "YAML encoding output")
	yamlPreserveMode := flag.Bool("Y", false, "YAML style and comments preserved")

	// JSON modes
	jsonMode := flag.Bool("j", false, "JSON compact output")
	jsonPrettyMode := flag.Bool("J", false, "JSON pretty output")

	// Token modes
	tokenMode := flag.Bool("t", false, "Token output")
	tokenProfuseMode := flag.Bool("T", false, "Token with line info")

	// Event modes
	eventMode := flag.Bool("e", false, "Event output")
	eventProfuseMode := flag.Bool("E", false, "Event with line info")

	// Node modes
	nodeMode := flag.Bool("n", false, "Node representation output")
	nodeProfuseMode := flag.Bool("N", false, "Node with tag and style for all scalars")
	streamMode := flag.Bool("S", false, "Show all stream nodes (use with -n)")

	// Shared flags
	longMode := flag.Bool("l", false, "Long (block) formatted output")
	unmarshalMode := flag.Bool("u", false, "Use Unmarshal instead of Decode for YAML input")
	marshalMode := flag.Bool("m", false, "Use Marshal instead of Encode for YAML output")

	// Long flag aliases
	flag.BoolVar(showHelp, "help", false, "Show this help information")
	flag.BoolVar(yamlMode, "yaml", false, "YAML encoding output")
	flag.BoolVar(yamlPreserveMode, "YAML", false, "YAML style and comments preserved")
	flag.BoolVar(jsonMode, "json", false, "JSON compact output")
	flag.BoolVar(jsonPrettyMode, "JSON", false, "JSON pretty output")
	flag.BoolVar(tokenMode, "token", false, "Token output")
	flag.BoolVar(tokenProfuseMode, "TOKEN", false, "Token with line info")
	flag.BoolVar(eventMode, "event", false, "Event output")
	flag.BoolVar(eventProfuseMode, "EVENT", false, "Event with line info")
	flag.BoolVar(nodeMode, "node", false, "Node representation output")
	flag.BoolVar(nodeProfuseMode, "NODE", false, "Node with tag and style for all scalars")
	flag.BoolVar(streamMode, "stream", false, "Show all stream nodes (use with -n)")
	flag.BoolVar(longMode, "long", false, "Long (block) formatted output")
	flag.BoolVar(unmarshalMode, "unmarshal", false, "Use Unmarshal instead of Decode for YAML input")
	flag.BoolVar(marshalMode, "marshal", false, "Use Marshal instead of Encode for YAML output")

	flag.Parse()

	// Validate flag combinations
	if *marshalMode && !*yamlMode && !*yamlPreserveMode {
		fmt.Fprintf(os.Stderr, "Error: -m/--marshal flag only makes sense with YAML output modes (-y/--yaml or -Y/--YAML)\n")
		os.Exit(1)
	}

	if *unmarshalMode && *yamlPreserveMode {
		fmt.Fprintf(os.Stderr, "Error: -u/--unmarshal flag doesn't make sense with preserving mode (-Y/--YAML) since unmarshal mode strips comments and styles\n")
		os.Exit(1)
	}

	// Show version and exit
	if *showVersion {
		fmt.Printf("go-yaml version %s\n", version)
		return
	}

	// Show help and exit
	if *showHelp {
		printHelp()
		return
	}

	// Check if stdin has data
	stat, err := os.Stdin.Stat()
	if err != nil {
		log.Fatal("Failed to stat stdin:", err)
	}

	// If no stdin and no flags, show help
	if (stat.Mode()&os.ModeCharDevice) != 0 && !*nodeMode && !*nodeProfuseMode && !*eventMode && !*eventProfuseMode && !*tokenMode && !*tokenProfuseMode && !*jsonMode && !*jsonPrettyMode && !*yamlMode && !*yamlPreserveMode && !*longMode {
		printHelp()
		return
	}

	// Error if stdin has data but no mode flags are provided
	if (stat.Mode()&os.ModeCharDevice) == 0 && !*nodeMode && !*nodeProfuseMode && !*eventMode && !*eventProfuseMode && !*tokenMode && !*tokenProfuseMode && !*jsonMode && !*jsonPrettyMode && !*yamlMode && !*yamlPreserveMode && !*longMode {
		fmt.Fprintf(os.Stderr, "Error: stdin has data but no mode specified. Use -n/--node, -N/--NODE, -e/--event, -E/--EVENT, -t/--token, -T/--TOKEN, -j/--json, -J/--JSON, -y/--yaml, -Y/--YAML flag.\n")
		os.Exit(1)
	}

	// Process YAML input
	if *eventMode {
		// Use event formatting mode (compact by default)
		compact := !*longMode // compact is default, long mode negates it
		if err := ProcessEvents(false, compact, *unmarshalMode); err != nil {
			log.Fatal("Failed to process events:", err)
		}
	} else if *eventProfuseMode {
		// Use event formatting mode with profuse output
		compact := !*longMode // compact is default, long mode negates it
		if err := ProcessEvents(true, compact, *unmarshalMode); err != nil {
			log.Fatal("Failed to process events:", err)
		}
	} else if *tokenMode {
		// Use token formatting mode (compact by default)
		compact := !*longMode // compact is default, long mode negates it
		if err := ProcessTokens(false, compact, *unmarshalMode); err != nil {
			log.Fatal("Failed to process tokens:", err)
		}
	} else if *tokenProfuseMode {
		// Use token formatting mode with profuse output
		compact := !*longMode // compact is default, long mode negates it
		if err := ProcessTokens(true, compact, *unmarshalMode); err != nil {
			log.Fatal("Failed to process tokens:", err)
		}
	} else if *jsonMode {
		// Use JSON formatting mode (compact by default)
		if err := ProcessJSON(false, *unmarshalMode); err != nil {
			log.Fatal("Failed to process JSON:", err)
		}
	} else if *jsonPrettyMode {
		// Use pretty JSON formatting mode
		if err := ProcessJSON(true, *unmarshalMode); err != nil {
			log.Fatal("Failed to process JSON:", err)
		}
	} else if *yamlMode {
		// Use YAML formatting mode (clean by default)
		if err := ProcessYAML(false, *unmarshalMode, *marshalMode); err != nil {
			log.Fatal("Failed to process YAML:", err)
		}
	} else if *yamlPreserveMode {
		// Use YAML formatting mode with preserve
		if err := ProcessYAML(true, *unmarshalMode, *marshalMode); err != nil {
			log.Fatal("Failed to process YAML:", err)
		}
	} else {
		// Use node formatting mode (default)
		profuse := *nodeProfuseMode
		if *unmarshalMode {
			// Use Unmarshal mode
			if err := ProcessNodeUnmarshal(profuse); err != nil {
				log.Fatal("Failed to process YAML node:", err)
			}
		} else {
			// Use Loader mode - always get stream nodes internally
			reader := io.Reader(os.Stdin)
			loader, err := yaml.NewLoader(reader, yaml.WithStreamNodes())
			if err != nil {
				log.Fatal("Failed to create loader:", err)
			}

			// Collect all documents
			var docs []any

			for {
				var node yaml.Node
				err := loader.Load(&node)
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					log.Fatal("Failed to load YAML node:", err)
				}

				// For -n flag: skip stream nodes without useful content
				// For -S flag: show all nodes including empty stream nodes
				if !*streamMode && !hasUsefulContent(&node) {
					continue
				}

				var info any
				if profuse {
					info = FormatNode(node, profuse)
				} else {
					info = FormatNodeCompact(node)
				}
				docs = append(docs, info)
			}

			// Output as sequence if multiple documents, otherwise output single document
			var output any
			if len(docs) == 1 {
				output = docs[0]
			} else {
				output = docs
			}

			// Use dumper for output
			var buf bytes.Buffer
			enc, err := yaml.NewDumper(&buf)
			if err != nil {
				log.Fatal("Failed to create dumper:", err)
			}
			if err := enc.Dump(output); err != nil {
				log.Fatal("Failed to dump node info:", err)
			}
			enc.Close()
			fmt.Print(buf.String())
		}
	}
}

// ProcessNodeUnmarshal reads YAML from stdin using Unmarshal and outputs node structure
func ProcessNodeUnmarshal(profuse bool) error {
	// Read all input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Split input into documents
	documents := bytes.Split(input, []byte("---"))

	// Collect all documents
	var docs []any

	for _, doc := range documents {
		// Skip empty documents
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		// For unmarshal mode, use any first to avoid preserving comments
		var data any
		if err := yaml.Load(doc, &data); err != nil {
			return fmt.Errorf("failed to load YAML: %w", err)
		}

		// Convert to yaml.Node for node processing
		var node yaml.Node
		if err := yaml.Load(doc, &node); err != nil {
			return fmt.Errorf("failed to load YAML to node: %w", err)
		}

		var info any
		if profuse {
			info = FormatNode(node, profuse)
		} else {
			info = FormatNodeCompact(node)
		}
		docs = append(docs, info)
	}

	// Output as sequence if multiple documents, otherwise output single document
	var output any
	if len(docs) == 1 {
		output = docs[0]
	} else {
		output = docs
	}

	// Use dumper for output
	var buf bytes.Buffer
	enc, err := yaml.NewDumper(&buf)
	if err != nil {
		return fmt.Errorf("failed to create dumper: %w", err)
	}
	if err := enc.Dump(output); err != nil {
		enc.Close()
		return fmt.Errorf("failed to dump node info: %w", err)
	}
	enc.Close()
	fmt.Print(buf.String())

	return nil
}

// printHelp displays the help information for the program
func printHelp() {
	fmt.Printf(`go-yaml version %s

The 'go-yaml' tool shows how the go.yaml.in/yaml/v4 library handles YAML both
internally and externally. It is a tool for testing and debugging the library.

It reads YAML input text from stdin and writes results to stdout.

The go-yaml API has two different pairs of functions for reading and writing
YAML: Decode/Encode and Unmarshal/Marshal.

Decode and Encode are used by default. Use -u and -m for Unmarshal and Marshal.


Usage:
  go-yaml [options]

Options:
  -y, --yaml       YAML encoding output
  -Y, --YAML       YAML w/ style and comments preserved

  -j, --json       JSON compact output
  -J, --JSON       JSON pretty output

  -t, --token      Token output
  -T, --TOKEN      Token with line info

  -e, --event      Event output
  -E, --EVENT      Event with line info

  -n, --node       Node representation output
  -N, --NODE       Node with tag and style for all scalars

  -l, --long       Long (block) formatted output

  -u, --unmarshal  Use Unmarshal instead of Decode for YAML input
  -m, --marshal    Use Marshal instead of Encode for YAML output

  -h, --help       Show this help information
  --version        Show version information

`, version)
}
