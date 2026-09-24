// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package plugin provides official YAML plugins for go-yaml.
//
// Plugins extend the core YAML library with optional processing capabilities.
// This package contains official plugin implementations maintained by the
// go-yaml project.
//
// # Available Plugins
//
// Loader-limits plugin (plugin/loaderlimits), API loader-limits:
//   - Configurable depth and alias expansion limits
//
// JSON-comments plugin (plugin/jsoncomments), API json-comments:
//   - Optional JSON-style comment sanitizing
//
// Reference YAML-parser plugin (plugin/parser/reference), API yaml-parser:
//   - Optional YAMLStar reference parser implementation
//
// The JSON-comments and reference YAML-parser packages are separate Go modules
// so their dependencies remain optional.
//
// # Usage
//
// Import the plugin you need and register it with WithPlugin:
//
//	import "go.yaml.in/yaml/v4"
//	import "go.yaml.in/yaml/v4/plugin/loaderlimits"
//
//	// Disable alias checking for documents with many aliases
//	loader := yaml.NewLoader(data,
//	    yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasNone())))
//
// # Third-Party Plugins
//
// Plugin interfaces use public types and can be implemented by external
// packages.
// Implement the relevant plugin interface (e.g., LoaderLimitsPlugin) and
// register with WithPlugin.
package plugin
