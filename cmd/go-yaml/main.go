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
	"strings"

	"go.yaml.in/yaml/v4"
)

// version is the current version of the go-yaml CLI tool.
const version = "4.0.0.1"

// stringSlice is a custom flag type for collecting multiple -o flags
type stringSlice []string

// String returns the string representation of the slice for [flag.Value] interface.
func (s *stringSlice) String() string {
	if s == nil {
		return ""
	}
	return fmt.Sprint(*s)
}

// Set appends a value to the slice for [flag.Value] interface.
func (s *stringSlice) Set(value string) error {
	// Special case: empty value or explicit help request
	if value == "" || value == "help" || value == "?" {
		printAvailableOptions()
		os.Exit(0)
	}
	*s = append(*s, value)
	return nil
}

// optionSpec defines metadata for an option
type optionSpec struct {
	typ     string // "bool", "int", "string", "multi"
	handler func(value string) ([]yaml.Option, error)
}

// optionRegistry maps option names (including short aliases) to their specs
var optionRegistry map[string]optionSpec

// initOptionRegistry initializes the option registry
func initOptionRegistry() {
	optionRegistry = map[string]optionSpec{
		// Version presets
		"v2": {typ: "preset", handler: func(string) ([]yaml.Option, error) {
			return []yaml.Option{yaml.WithV2Defaults()}, nil
		}},
		"v3": {typ: "preset", handler: func(string) ([]yaml.Option, error) {
			return []yaml.Option{yaml.WithV3Defaults()}, nil
		}},
		"v4": {typ: "preset", handler: func(string) ([]yaml.Option, error) {
			return []yaml.Option{yaml.WithV4Defaults()}, nil
		}},

		// Formatting options
		"indent": {typ: "int", handler: func(value string) ([]yaml.Option, error) {
			var val int
			if _, err := fmt.Sscanf(value, "%d", &val); err != nil {
				return nil, fmt.Errorf("indent requires integer value (2-9)")
			}
			if val < 2 || val > 9 {
				return nil, fmt.Errorf("indent must be between 2 and 9")
			}
			return []yaml.Option{yaml.WithIndent(val)}, nil
		}},
		"compact-seq-indent": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithCompactSeqIndent(val)}, nil
		}},
		"compact": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithCompactSeqIndent(val)}, nil
		}},
		"line-width": {typ: "int", handler: func(value string) ([]yaml.Option, error) {
			var val int
			if _, err := fmt.Sscanf(value, "%d", &val); err != nil {
				return nil, fmt.Errorf("line-width requires integer value")
			}
			return []yaml.Option{yaml.WithLineWidth(val)}, nil
		}},
		"width": {typ: "int", handler: func(value string) ([]yaml.Option, error) {
			var val int
			if _, err := fmt.Sscanf(value, "%d", &val); err != nil {
				return nil, fmt.Errorf("width requires integer value")
			}
			return []yaml.Option{yaml.WithLineWidth(val)}, nil
		}},
		"unicode": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithUnicode(val)}, nil
		}},
		"unique-keys": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithUniqueKeys(val)}, nil
		}},
		"unique": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithUniqueKeys(val)}, nil
		}},
		"canonical": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithCanonical(val)}, nil
		}},
		"line-break": {typ: "string", handler: func(value string) ([]yaml.Option, error) {
			var lb yaml.LineBreak
			switch value {
			case "ln":
				lb = yaml.LineBreakLN
			case "cr":
				lb = yaml.LineBreakCR
			case "crln":
				lb = yaml.LineBreakCRLN
			default:
				return nil, fmt.Errorf("line-break must be ln, cr, or crln")
			}
			return []yaml.Option{yaml.WithLineBreak(lb)}, nil
		}},
		"explicit-start": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithExplicitStart(val)}, nil
		}},
		"explicit-end": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithExplicitEnd(val)}, nil
		}},
		"explicit": {typ: "multi", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithExplicitStart(val), yaml.WithExplicitEnd(val)}, nil
		}},
		"flow-simple-coll": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithFlowSimpleCollections(val)}, nil
		}},
		"known-fields": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithKnownFields(val)}, nil
		}},
		"single-document": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithSingleDocument(val)}, nil
		}},
		"single": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithSingleDocument(val)}, nil
		}},
		"all-documents": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithAllDocuments(val)}, nil
		}},
		"all": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithAllDocuments(val)}, nil
		}},
		"stream-nodes": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithStreamNodes(val)}, nil
		}},
		"stream": {typ: "bool", handler: func(value string) ([]yaml.Option, error) {
			val := value == "true"
			return []yaml.Option{yaml.WithStreamNodes(val)}, nil
		}},
	}
}

// parseOneOption parses a single option (name=value, name, no-name, or v2/v3/v4)
func parseOneOption(s string) ([]yaml.Option, error) {
	// Special case: help
	if s == "help" || s == "?" {
		printAvailableOptions()
		os.Exit(0)
	}

	// Check for version presets first
	if s == "v2" || s == "v3" || s == "v4" {
		spec, ok := optionRegistry[s]
		if !ok {
			return nil, fmt.Errorf("unknown option: %s", s)
		}
		return spec.handler("")
	}

	// Check for "no-" prefix for boolean false
	if len(s) > 3 && s[:3] == "no-" {
		name := s[3:]
		spec, ok := optionRegistry[name]
		if !ok {
			return nil, fmt.Errorf("unknown option: %s", name)
		}
		if spec.typ != "bool" && spec.typ != "multi" {
			return nil, fmt.Errorf("option %s is not boolean, cannot use no- prefix", name)
		}
		return spec.handler("false")
	}

	// Check for "name=value" format
	if name, value, found := strings.Cut(s, "="); found {
		spec, ok := optionRegistry[name]
		if !ok {
			return nil, fmt.Errorf("unknown option: %s", name)
		}
		if spec.typ == "bool" || spec.typ == "multi" {
			// For boolean options with explicit value
			if value != "true" && value != "false" {
				return nil, fmt.Errorf("option %s requires true or false value", name)
			}
		}
		return spec.handler(value)
	}

	// Must be "name" alone (boolean true)
	spec, ok := optionRegistry[s]
	if !ok {
		return nil, fmt.Errorf("unknown option: %s", s)
	}
	if spec.typ != "bool" && spec.typ != "multi" && spec.typ != "preset" {
		return nil, fmt.Errorf("option %s requires a value (use %s=value)", s, s)
	}
	return spec.handler("true")
}

// parseOptionFlags parses comma-separated options string into individual options
func parseOptionFlags(s string) ([]yaml.Option, error) {
	parts := strings.Split(s, ",")
	var opts []yaml.Option
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		opt, err := parseOneOption(trimmed)
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt...)
	}
	return opts, nil
}

// printAvailableOptions prints the list of available options for -o flag
func printAvailableOptions() {
	fmt.Print(`Available options for -o/--option:

Version presets:
  v2                    V2 defaults (indent:2, no compact-seq-indent)
  v3                    V3 defaults (indent:4, no compact-seq-indent)
  v4                    V4 defaults (indent:2, compact-seq-indent) [default]

Formatting options:
  indent=NUM            Indentation spaces (2-9)
  compact-seq-indent    '- ' counts as indentation (short: compact)
  line-width=NUM        Preferred line width, -1=unlimited (short: width)
  unicode               Allow non-ASCII in output
  canonical             Canonical YAML output format
  line-break=TYPE       Line ending: ln, cr, or crln
  explicit-start        Always emit '---' marker
  explicit-end          Always emit '...' marker
  explicit              Both explicit-start and explicit-end
  flow-simple-coll      Flow style for simple collections

Loading options:
  unique-keys           Duplicate key detection (short: unique)
  known-fields          Strict field checking
  single-document       Only process first document (short: single)
  all-documents         Multi-document mode (short: all)
  stream-nodes          Enable stream boundary nodes (short: stream)

Boolean options: use 'name' for true, 'no-name' for false
Multiple options: comma-separated or repeat -o flag

Examples:
  go-yaml -y -o indent=4,canonical
  go-yaml -y -o v3,width=120,explicit
  go-yaml -y -o no-unicode,no-compact
`)
}

// buildOptions creates the yaml.Option slice based on config file and -o flags
func buildOptions(configFile string, optionFlags []string) ([]yaml.Option, error) {
	var opts []yaml.Option

	// Default to V4 preset
	opts = append(opts, yaml.WithV4Defaults())

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

	// Process -o flags (can override default preset and config)
	for _, optStr := range optionFlags {
		parsedOpts, err := parseOptionFlags(optStr)
		if err != nil {
			return nil, err
		}
		opts = append(opts, parsedOpts...)
	}

	return opts, nil
}

// main reads YAML from stdin, parses it, and outputs the node structure
func main() {
	// Initialize option registry
	initOptionRegistry()

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

	// Shared flags
	longMode := flag.Bool("l", false, "Long (block) formatted output")

	// Config file flag
	configFile := flag.String("C", "", "Load options from YAML config file")

	// Option flags (-o/--option)
	var optionFlags stringSlice
	flag.Var(&optionFlags, "o", "Set option (name=value, name, no-name)")
	flag.Var(&optionFlags, "option", "Set option (name=value, name, no-name)")

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
	flag.BoolVar(longMode, "long", false, "Long (block) formatted output")
	flag.StringVar(configFile, "config", "", "Load options from YAML config file")

	// API selection flags (long form only)
	flag.BoolVar(&unmarshalMode, "unmarshal", false, "Use Unmarshal API for input")
	flag.BoolVar(&decodeMode, "decode", false, "Use Decode API for input")
	flag.BoolVar(&marshalMode, "marshal", false, "Use Marshal API for output")
	flag.BoolVar(&encodeMode, "encode", false, "Use Encode API for output")

	// Custom usage function to provide helpful message when -o is used without value
	flag.Usage = func() {
		// Check if the error is about -o needing an argument
		if len(os.Args) > 1 {
			for i, arg := range os.Args[1:] {
				if (arg == "-o" || arg == "--option") && (i+2 >= len(os.Args) || os.Args[i+2][0] == '-') {
					fmt.Fprintf(os.Stderr, "The -o flag requires a value.\n")
					fmt.Fprintf(os.Stderr, "Use '-o help' or '-o ?' to see available options.\n\n")
					printAvailableOptions()
					return
				}
			}
		}
		// Default usage
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	// Validate flag combinations

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
		opts, err = buildOptions(*configFile, optionFlags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
			printAvailableOptions()
			os.Exit(1)
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
		if err := ProcessJSON(input, false, unmarshalMode, decodeMode, opts...); err != nil {
			log.Fatal("Failed to process JSON:", err)
		}
	} else if *jsonPrettyMode {
		// Use pretty JSON formatting mode
		if err := ProcessJSON(input, true, unmarshalMode, decodeMode, opts...); err != nil {
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
			// Use Loader mode with options
			loader, err := yaml.NewLoader(input, opts...)
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
			if err := enc.Close(); err != nil {
				log.Fatal("Failed to close dumper:", err)
			}
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
	if err := enc.Close(); err != nil {
		return fmt.Errorf("failed to close dumper: %w", err)
	}
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

Formatting Options:
  -o, --option OPT Set option (use without value to see all options)
                   Multiple: -o opt1,opt2 or -o opt1 -o opt2
                   Booleans: name (true) or no-name (false)
                   Presets: v2, v3, v4 (default: v4)

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
