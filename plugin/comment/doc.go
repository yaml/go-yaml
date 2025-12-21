//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

// Package comment provides plugins for controlling YAML comment handling.
//
// # Overview
//
// YAML comments are automatically parsed into [yaml.Node] fields
// (HeadComment, LineComment, FootComment) and written back during encoding.
// Comment plugins allow you to customize this behavior.
//
// # Available Plugins
//
// Three comment plugin variants are available:
//
// **v3** - Version 3 comment handling
//
//	import "go.yaml.in/yaml/v4/plugin/comment/v3"
//	yaml.NewLoader(r, yaml.WithPlugins(v3.New()))
//
// **v4** - Version 4 comment handling
//
//	import "go.yaml.in/yaml/v4/plugin/comment/v4"
//	yaml.NewLoader(r, yaml.WithPlugins(v4.New()))
//
// **none** - Strip all comments for performance
//
//	import "go.yaml.in/yaml/v4/plugin/comment/none"
//	yaml.NewLoader(r, yaml.WithPlugins(none.New()))
//
// # When to Use Each Variant
//
// **Use v3 or v4** when:
//   - You need to preserve comments through load/dump cycles
//   - You're building tools that manipulate YAML while keeping comments
//   - Comment preservation is important for your use case
//
// **Use none** when:
//   - You don't need comments in your application
//   - You want faster parsing and lower memory usage
//   - Comments should be stripped from output
//
// **Use default (no plugin)** when:
//   - Standard comment handling is sufficient
//   - You want the default behavior without explicit plugin configuration
//
// # Comment Fields in yaml.Node
//
// Comments are stored in [yaml.Node] as:
//
//   - HeadComment - Comments appearing before the node
//   - LineComment - Comments on the same line as the node
//   - FootComment - Comments appearing after the node
//
// These fields are strings where lines are separated by "\n".
//
// # Example: Stripping Comments
//
//	import (
//	    "go.yaml.in/yaml/v4"
//	    "go.yaml.in/yaml/v4/plugin/comment/none"
//	)
//
//	// Load YAML and strip all comments
//	loader, _ := yaml.NewLoader(reader, yaml.WithPlugins(none.New()))
//	var node yaml.Node
//	loader.Load(&node)
//
//	// Dump without comments
//	dumper, _ := yaml.NewDumper(writer, yaml.WithPlugins(none.New()))
//	dumper.Dump(&node)
//	dumper.Close()
//
// # Example: Using v4 preset with comment plugin
//
//	import "go.yaml.in/yaml/v4/plugin/comment/v4"
//
//	// v4 preset + explicit comment plugin
//	dumper, _ := yaml.NewDumper(w, yaml.V4, yaml.WithPlugins(v4.New()))
package comment
