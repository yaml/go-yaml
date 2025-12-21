//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

// Package none provides a comment plugin that strips all comments.
package none

import "go.yaml.in/yaml/v4"

// Plugin strips all comments for better performance.
//
// This plugin removes all comments from nodes during both loading and dumping,
// which can improve performance when comments are not needed.
type Plugin struct{}

// New returns a new no-comments plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "comment-none"
}

// ProcessLoadNode processes a node during loading, stripping all comments.
func (p *Plugin) ProcessLoadNode(node *yaml.Node) (*yaml.Node, error) {
	stripComments(node)
	return node, nil
}

// ProcessDumpNode processes a node during dumping, stripping all comments.
func (p *Plugin) ProcessDumpNode(node *yaml.Node) (*yaml.Node, error) {
	stripComments(node)
	return node, nil
}

// stripComments recursively removes all comments from a node and its children.
func stripComments(node *yaml.Node) {
	if node == nil {
		return
	}

	node.HeadComment = ""
	node.LineComment = ""
	node.FootComment = ""

	for _, child := range node.Content {
		stripComments(child)
	}
}
