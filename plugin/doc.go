//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

// Package plugin provides interfaces and implementations for extending
// YAML processing with custom node transformations.
//
// # Overview
//
// Plugins allow you to process or transform [yaml.Node] values during
// YAML loading (after parsing) or dumping (before encoding). This enables
// use cases like:
//
//   - Stripping or preserving comments
//   - Node validation or transformation
//   - Custom formatting or normalization
//   - Logging or debugging YAML structures
//
// # Plugin Interfaces
//
// Plugins implement one or more of these interfaces:
//
//	type Plugin interface {
//	    Name() string
//	}
//
//	type LoadPlugin interface {
//	    Plugin
//	    ProcessLoadNode(node *yaml.Node) (*yaml.Node, error)
//	}
//
//	type DumpPlugin interface {
//	    Plugin
//	    ProcessDumpNode(node *yaml.Node) (*yaml.Node, error)
//	}
//
// A plugin that implements both LoadPlugin and DumpPlugin can process
// nodes during both loading and dumping operations.
//
// # Using Plugins
//
// Add plugins to a [yaml.Loader] or [yaml.Dumper] with [yaml.WithPlugin]:
//
//	import "go.yaml.in/yaml/v4/plugin/comment/v3"
//
//	// Preserve comments during loading
//	loader, _ := yaml.NewLoader(r, yaml.WithPlugin(v3.New()))
//
//	// Apply multiple plugins (they execute in order)
//	dumper, _ := yaml.NewDumper(w, yaml.WithPlugin(plugin1), yaml.WithPlugin(plugin2))
//
// # Plugin Execution
//
// Plugins execute at specific points in the YAML processing pipeline:
//
//   - LoadPlugin.ProcessLoadNode runs after parsing completes, before
//     unmarshaling to Go values
//   - DumpPlugin.ProcessDumpNode runs after marshaling from Go values,
//     before encoding to YAML
//
// Multiple plugins execute in the order they are added. Each plugin
// receives the node returned by the previous plugin.
//
// # Creating Custom Plugins
//
// To create a custom plugin, implement the Plugin interface and at
// least one of LoadPlugin or DumpPlugin:
//
//	type UppercasePlugin struct{}
//
//	func (p *UppercasePlugin) Kind() string {
//	    return "uppercase"
//	}
//
//	func (p *UppercasePlugin) ProcessDumpNode(node *yaml.Node) (*yaml.Node, error) {
//	    if node.Kind == yaml.ScalarNode {
//	        node.Value = strings.ToUpper(node.Value)
//	    }
//	    for _, child := range node.Content {
//	        p.ProcessDumpNode(child)
//	    }
//	    return node, nil
//	}
//
//	// Usage
//	dumper, _ := yaml.NewDumper(w, yaml.WithPlugin(&UppercasePlugin{}))
//
// # Built-in Plugins
//
// The plugin/comment subpackage provides comment handling:
//
//   - comment/v3 - v3-style comment preservation
//
// See the comment subpackage documentation for details.
package plugin
