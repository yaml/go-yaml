//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

package yaml

// CommentPlugin processes comments during YAML parsing.
//
// When registered, the ProcessComment method is called for each node during
// parsing, allowing the plugin to attach or transform comment data.
//
// Example usage:
//
//	loader := yaml.NewLoader(data, yaml.WithPlugin(commentPlugin))
type CommentPlugin interface {
	// ProcessComment is called for each node during parsing.
	// The node parameter is the node being processed.
	// The ctx parameter contains the raw comment data from the parser.
	// Plugins can modify the node's comment fields based on ctx.
	ProcessComment(node *Node, ctx *CommentContext) error
}
