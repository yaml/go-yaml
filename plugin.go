//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

package yaml

// Plugin is the interface that all YAML plugins must implement.
type Plugin interface {
	// Name returns a unique identifier for the plugin.
	Name() string
}

// LoadPlugin processes nodes after parsing, before unmarshalling to Go values.
type LoadPlugin interface {
	Plugin
	// ProcessLoadNode is called for each node during loading.
	// Returns the (potentially modified) node and any error.
	ProcessLoadNode(node *Node) (*Node, error)
}

// DumpPlugin processes nodes before encoding to YAML output.
type DumpPlugin interface {
	Plugin
	// ProcessDumpNode is called for each node during dumping.
	// Returns the (potentially modified) node and any error.
	ProcessDumpNode(node *Node) (*Node, error)
}
