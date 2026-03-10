// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package comment provides comment processing plugins for YAML.
//
// Comment plugins control how comments from the YAML source are attached to
// nodes during parsing.
//
// # Available Plugins
//
//   - v3: V3-compatible comment handling (plugin/comment/v3)
//   - v4: V4 round-trip comment handling (plugin/comment/v4)
//
// # Usage
//
// Import a comment plugin and register it with WithPlugin:
//
//	import "go.yaml.in/yaml/v4"
//	import "go.yaml.in/yaml/v4/plugin/comment/v3"
//
//	loader := yaml.NewLoader(data, yaml.WithPlugin(v3.New()))
//
// # Default Behavior
//
// By default (without a comment plugin), comments are skipped during parsing
// for better performance.
// Use WithV3LegacyComments() for a simpler alternative that doesn't require
// plugin setup.
package comment
