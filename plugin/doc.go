//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

// Package plugin provides official YAML plugins for go-yaml.
//
// Plugins extend the core YAML library with optional processing capabilities.
// This package contains official plugin implementations maintained by the
// go-yaml project.
//
// # Available Plugins
//
// Comment plugins (plugin/comment):
//   - v3legacy: V3-compatible comment handling
//
// # Usage
//
// Import the plugin you need and register it with WithPlugin:
//
//	import "go.yaml.in/yaml/v4"
//	import "go.yaml.in/yaml/v4/plugin/comment/v3legacy"
//
//	loader := yaml.NewLoader(data, yaml.WithPlugin(v3legacy.New()))
//
// # Third-Party Plugins
//
// Plugin interfaces use public types and can be implemented by external
// packages.
// Implement the relevant plugin interface (e.g., CommentPlugin) and register
// with WithPlugin.
package plugin
