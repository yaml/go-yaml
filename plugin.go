//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

package yaml

// Plugin is the interface that all YAML plugins must implement.
type Plugin interface {
	// Kind returns the plugin type (e.g., "comment").
	// Used as the key in the plugins map.
	Kind() string
}

// CommentPlugin handles comment attachment during YAML processing.
//
// This plugin type provides hooks at multiple stages of the YAML pipeline
// to control how comments are attached to events and nodes.
type CommentPlugin interface {
	Plugin

	// ProcessEventComments is called during parsing to attach comments to events.
	//
	// The plugin receives a CommentContext with raw comment data from the scanner
	// and can modify it to control which comments attach to the current event.
	//
	// This hook is called after token scanning but before event emission.
	ProcessEventComments(ctx *CommentContext) error

	// ProcessNodeComments is called when creating nodes to transfer comments
	// from events to nodes.
	//
	// The plugin receives the node being created and a CommentContext with
	// comment data from the current event. It should populate the node's
	// HeadComment, LineComment, and FootComment fields as appropriate.
	//
	// This hook is called in the decode.go parser when building the node tree.
	ProcessNodeComments(node *Node, ctx *CommentContext) error
}

// CommentContext provides comment data to CommentPlugin implementations.
//
// This type abstracts the internal libyaml types to provide a stable API
// for external comment plugins.
type CommentContext struct {
	// HeadComment holds comments that appear before the current element.
	HeadComment []byte

	// LineComment holds comments on the same line as the current element.
	LineComment []byte

	// FootComment holds comments that appear after the current element.
	FootComment []byte

	// TailComment holds foot comments at the end of a mapping value.
	TailComment []byte

	// StemComment holds comments preceding a nested structure.
	StemComment []byte
}

// ResolvePlugin handles tag resolution during YAML processing.
//
// This plugin type is called during the Resolve stage to assign tags
// to nodes in the representation tree. The plugin is called for every
// node in the tree (depth-first) and must set a valid tag for each node.
type ResolvePlugin interface {
	Plugin

	// ResolveNode is called for each node in the tree during resolution.
	//
	// The plugin should set node.Tag to an appropriate tag based on the
	// node's Kind, Value, and context. The plugin may also normalize
	// node.Value (string to string transformation).
	//
	// The plugin must always set a valid tag; it cannot leave the tag empty.
	ResolveNode(node *Node, ctx *ResolveContext) error
}

// ResolveContext provides context to ResolvePlugin implementations.
//
// This type provides information about the node's position in the tree
// to enable context-aware tag resolution.
type ResolveContext struct {
	// Path is the path from the root to the current node.
	// For mappings, keys are included as path elements.
	// For sequences, indices are included as string representations.
	// Example: ["root", "key1", "0", "nested"]
	Path []string

	// Parent is the parent node of the current node, or nil for root.
	Parent *Node

	// Root is the root node of the document.
	Root *Node
}
