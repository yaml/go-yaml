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
// # Available Plugin
//
// **v3** - Version 3 comment handling
//
//	import "go.yaml.in/yaml/v4/plugin/comment/v3"
//	yaml.NewLoader(r, yaml.WithPlugin(v3.New()))
//
// # When to Use
//
// **Use v3** when:
//   - You need to preserve comments through load/dump cycles
//   - You're building tools that manipulate YAML while keeping comments
//   - Comment preservation is important for your use case
//
// **Use default (no plugin)** when:
//   - You don't need comment preservation
//   - You want the default V4 behavior
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
// # Example: Preserving Comments
//
//	import (
//	    "go.yaml.in/yaml/v4"
//	    "go.yaml.in/yaml/v4/plugin/comment/v3"
//	)
//
//	// Load YAML and preserve comments
//	loader, _ := yaml.NewLoader(reader, yaml.WithPlugin(v3.New()))
//	var node yaml.Node
//	loader.Load(&node)
//
//	// Dump with comments preserved
//	dumper, _ := yaml.NewDumper(writer, yaml.WithPlugin(v3.New()))
//	dumper.Dump(&node)
//	dumper.Close()
package comment
