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

// buildOptions creates the yaml.Option slice based on flags
func buildOptions(v2Mode, v3Mode bool, configFile string, indentLevel, lineWidth int, compactSeq bool) ([]yaml.Option, error) {
	var opts []yaml.Option

	// Apply version preset (if not using explicit API flags)
	if v2Mode {
		opts = append(opts, yaml.V2)
	} else if v3Mode {
		opts = append(opts, yaml.V3)
	} else {
		// V4 is the default, no need to explicitly add it
		opts = append(opts, yaml.V4)
	}

	// Load config file if specified
	if configFile != "" {
		configData, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		configOpts, err := yaml.OptsYAML(string(configData))
		if err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
		opts = append(opts, configOpts)
	}

	// Apply individual formatting options (these override version presets and config)
	if indentLevel > 0 {
		if indentLevel < 2 || indentLevel > 9 {
			return nil, fmt.Errorf("indent level must be between 2 and 9")
		}
		opts = append(opts, yaml.WithIndent(indentLevel))
	}
	if lineWidth > 0 {
		opts = append(opts, yaml.WithLineWidth(lineWidth))
	}
	if compactSeq {
		opts = append(opts, yaml.WithCompactSeqIndent(true))
	}

	return opts, nil
}

// main reads YAML from stdin, parses it, and outputs the node structure
func main() {
	// Parse command line flags
	showHelp := flag.Bool("h", false, "Show this help information")

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

	// Version flags (long form only)
	var v2Mode bool
	var v3Mode bool
	flag.BoolVar(&v2Mode, "v2", false, "Use V2 option preset")
	flag.BoolVar(&v3Mode, "v3", false, "Use V3 option preset")

	// Config file flag
	configFile := flag.String("C", "", "Load options from YAML config file")

	// Formatting option flags (long form only)
	var indentLevel int
	var lineWidth int
	var compactSeq bool
	flag.IntVar(&indentLevel, "indent", 0, "Set indentation level (2-9, 0=use default)")
	flag.IntVar(&lineWidth, "width", 0, "Set line width (0=use default)")
	flag.BoolVar(&compactSeq, "compact", false, "Enable compact sequence indentation")

	// API selection flags (long form only)
	var unmarshalMode bool
	var decodeMode bool
	var marshalMode bool
	var encodeMode bool

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
	flag.StringVar(configFile, "config", "", "Load options from YAML config file")

	// API selection flags (long form only)
	flag.BoolVar(&unmarshalMode, "unmarshal", false, "Use Unmarshal API for input")
	flag.BoolVar(&decodeMode, "decode", false, "Use Decode API for input")
	flag.BoolVar(&marshalMode, "marshal", false, "Use Marshal API for output")
	flag.BoolVar(&encodeMode, "encode", false, "Use Encode API for output")

	flag.Parse()

	// Validate flag combinations

	// Check that --v2 and --v3 are mutually exclusive
	if v2Mode && v3Mode {
		fmt.Fprintf(os.Stderr, "Error: --v2 and --v3 flags are mutually exclusive\n")
		os.Exit(1)
	}

	// Warn if version flags are used with explicit API flags
	if (v2Mode || v3Mode) && (unmarshalMode || decodeMode || marshalMode || encodeMode) {
		fmt.Fprintf(os.Stderr, "Warning: version flags (--v2/--v3) are ignored when using explicit API flags (--unmarshal/--decode/--marshal/--encode)\n")
	}

	// Check that marshal/encode flags only work with YAML output modes
	if (marshalMode || encodeMode) && !*yamlMode && !*yamlPreserveMode {
		fmt.Fprintf(os.Stderr, "Error: --marshal/--encode flags only make sense with YAML output modes (-y/--yaml or -Y/--YAML)\n")
		os.Exit(1)
	}

	// Check that unmarshal doesn't work with preserve mode
	if unmarshalMode && *yamlPreserveMode {
		fmt.Fprintf(os.Stderr, "Error: --unmarshal flag doesn't make sense with preserving mode (-Y/--YAML) since unmarshal mode strips comments and styles\n")
		os.Exit(1)
	}

	// Build options for Load API (only when not using explicit old APIs)
	var opts []yaml.Option
	if !unmarshalMode && !decodeMode {
		var err error
		opts, err = buildOptions(v2Mode, v3Mode, *configFile, indentLevel, lineWidth, compactSeq)
		if err != nil {
			log.Fatal("Failed to build options:", err)
		}
	}

	// Show help and exit
	if *showHelp {
		printHelp()
		return
	}

	// Get file argument (if any)
	args := flag.Args()
	var input io.Reader
	var inputFile *os.File

	if len(args) == 0 || (len(args) == 1 && args[0] == "-") {
		// No file argument or explicit stdin ("-")
		input = os.Stdin

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
	} else if len(args) == 1 {
		// File argument provided
		var err error
		inputFile, err = os.Open(args[0])
		if err != nil {
			log.Fatal("Failed to open file:", err)
		}
		defer inputFile.Close()
		input = inputFile
	} else {
		// Multiple files not supported
		fmt.Fprintf(os.Stderr, "Error: only one file argument supported\n")
		os.Exit(1)
	}

	// Process YAML input
	if *eventMode {
		// Use event formatting mode (compact by default)
		compact := !*longMode // compact is default, long mode negates it
		if err := ProcessEvents(input, false, compact, unmarshalMode); err != nil {
			log.Fatal("Failed to process events:", err)
		}
	} else if *eventProfuseMode {
		// Use event formatting mode with profuse output
		compact := !*longMode // compact is default, long mode negates it
		if err := ProcessEvents(input, true, compact, unmarshalMode); err != nil {
			log.Fatal("Failed to process events:", err)
		}
	} else if *tokenMode {
		// Use token formatting mode (compact by default)
		compact := !*longMode // compact is default, long mode negates it
		if err := ProcessTokens(input, false, compact, unmarshalMode); err != nil {
			log.Fatal("Failed to process tokens:", err)
		}
	} else if *tokenProfuseMode {
		// Use token formatting mode with profuse output
		compact := !*longMode // compact is default, long mode negates it
		if err := ProcessTokens(input, true, compact, unmarshalMode); err != nil {
			log.Fatal("Failed to process tokens:", err)
		}
	} else if *jsonMode {
		// Use JSON formatting mode (compact by default)
		if err := ProcessJSON(input, false, unmarshalMode, decodeMode, opts); err != nil {
			log.Fatal("Failed to process JSON:", err)
		}
	} else if *jsonPrettyMode {
		// Use pretty JSON formatting mode
		if err := ProcessJSON(input, true, unmarshalMode, decodeMode, opts); err != nil {
			log.Fatal("Failed to process JSON:", err)
		}
	} else if *yamlMode {
		// Use YAML formatting mode (clean by default)
		if err := ProcessYAML(input, false, unmarshalMode, decodeMode, marshalMode, encodeMode, opts); err != nil {
			log.Fatal("Failed to process YAML:", err)
		}
	} else if *yamlPreserveMode {
		// Use YAML formatting mode with preserve
		if err := ProcessYAML(input, true, unmarshalMode, decodeMode, marshalMode, encodeMode, opts); err != nil {
			log.Fatal("Failed to process YAML:", err)
		}
	} else {
		// Use node formatting mode (default)
		profuse := *nodeProfuseMode
		if unmarshalMode {
			// Use Unmarshal mode
			if err := ProcessNodeUnmarshal(input, profuse); err != nil {
				log.Fatal("Failed to process YAML node:", err)
			}
		} else {
			// Use Loader mode with options - always get stream nodes internally
			optsWithStream := append(opts, yaml.WithStreamNodes())
			loader, err := yaml.NewLoader(input, optsWithStream...)
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

// ProcessNodeUnmarshal reads YAML from reader using Unmarshal and outputs node structure
func ProcessNodeUnmarshal(reader io.Reader, profuse bool) error {
	// Read all input from reader
	input, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
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

It reads YAML input text from stdin or a file and writes results to stdout.

The go-yaml API has three sets of functions for reading/writing YAML:
  - Load/Dump (default, new API with options support - v4 defaults)
  - Decode/Encode (deprecated, no options)
  - Unmarshal/Marshal (deprecated, no options)


Usage:
  go-yaml [options] [file]

Output Mode Options:
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

Formatting Options (override version presets and config):
  --indent=NUM     Set indentation level (2-9)
  --width=NUM      Set line width
  --compact        Enable compact sequence indentation

Version Preset Options (only apply to default Load/Dump API):
  --v2             Use V2 option preset (indent:2, no compact-seq-indent)
  --v3             Use V3 option preset (indent:4, no compact-seq-indent)
                   (V4 is default: indent:2, compact-seq-indent enabled)

API Selection Options:
  --unmarshal      Use Unmarshal API for input (deprecated, v3 defaults)
  --decode         Use Decode API for input (deprecated)
  --marshal        Use Marshal API for output (deprecated)
  --encode         Use Encode API for output (deprecated)

Configuration:
  -C, --config     Load options from YAML config file

Other Options:
  -h, --help       Show this help information

`, version)
}
