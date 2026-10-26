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
//   - The former plugin/limit module is deprecated
//
// Merge-Key plugin (plugin/mergekey), API merge-key:
//   - Disable merging, combine repeated directives, and select conflict policy
//   - Replace the mapping merge algorithm with a third-party implementation
//
// Tab indentation plugin (plugin/tabindent), API tab-indent:
//   - Tab-aware structural indentation for loading and dumping
//
// Reference YAML-parser plugin (plugin/yamlparser/reference), API yaml-parser:
//   - Optional YAMLStar reference parser implementation
//
// The reference YAML-parser package is a separate Go module so its
// dependencies remain optional.
//
// ErrorFormat plugins (plugin/errorformat/v3 and plugin/errorformat/v4):
//   - Versioned YAML load and dump error message formatting
//
// # Usage
//
// Import the plugin you need and register it with WithPlugin:
//
//	import "go.yaml.in/yaml/v4"
//	import "go.yaml.in/yaml/v4/plugin/loaderlimits"
//	import errorformatv3 "go.yaml.in/yaml/v4/plugin/errorformat/v3"
//
//	// Disable alias checking for documents with many aliases
//	loader, err := yaml.NewLoader(data,
//	    yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasNone())))
//	if err != nil {
//	    return err
//	}
//
//	// Use legacy error format (yaml: line N: msg)
//	loader, err = yaml.NewLoader(data, yaml.WithPlugin(errorformatv3.New()))
//	if err != nil {
//	    return err
//	}
//
// # Third-Party Plugins
//
// Plugin interfaces use public types and can be implemented by external
// packages.
// Implement the relevant plugin interface (e.g., LoaderLimitsPlugin) and
// register with WithPlugin.
package plugin
