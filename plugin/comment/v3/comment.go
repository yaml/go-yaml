//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

// Package v3 provides a comment plugin with v3-specific behavior.
package v3

import "go.yaml.in/yaml/v4"

// Plugin handles YAML comment preservation with v3-specific behavior.
//
// Comments are automatically preserved during parsing and encoding through
// the Node.HeadComment, Node.LineComment, and Node.FootComment fields.
type Plugin struct{}

// New returns a new v3 comments plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "comment-v3"
}

// ProcessLoadNode processes a node during loading.
//
// Comments are already populated in Node.HeadComment, Node.LineComment,
// and Node.FootComment fields by the parser.
func (p *Plugin) ProcessLoadNode(node *yaml.Node) (*yaml.Node, error) {
	// Comments are already populated in the node by the parser
	return node, nil
}

// ProcessDumpNode processes a node during dumping.
//
// Comments from Node.HeadComment, Node.LineComment, and Node.FootComment
// fields are automatically written during encoding.
func (p *Plugin) ProcessDumpNode(node *yaml.Node) (*yaml.Node, error) {
	// Comments from the node fields are already used during encoding
	return node, nil
}
