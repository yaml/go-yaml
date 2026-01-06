//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

// Package v3 provides a comment plugin with v3-specific behavior.
//
// This plugin implements the reference comment handling strategy used in
// go-yaml v3, preserving comments during load/dump cycles.
package v3

import "go.yaml.in/yaml/v4"

// Plugin handles YAML comment preservation with v3-specific behavior.
//
// This is the reference implementation of the CommentPlugin interface,
// demonstrating how comments are attached to events and nodes during
// YAML processing.
type Plugin struct {
	// Future: add configuration options
	// preserveEmpty bool
	// strictFootComments bool
}

// New returns a new v3 comments plugin.
//
// The v3 plugin preserves comments using the strategy from go-yaml v3:
// - Head comments attach to the following element
// - Line comments attach to the current element
// - Foot comments attach to the preceding element
func New() *Plugin {
	return &Plugin{}
}

// Kind returns the plugin type.
func (p *Plugin) Kind() string {
	return "comment"
}

// ProcessEventComments processes comments at the parser/event level.
//
// This is called during parsing after tokens are scanned but before
// events are emitted. The plugin can modify the comment context to
// control which comments attach to the current event.
//
// For v3, this is currently a pass-through - the internal parser has
// already classified comments as head/line/foot based on v3 rules.
// Future versions may add v3-specific transformations here.
func (p *Plugin) ProcessEventComments(ctx *yaml.CommentContext) error {
	// V3 strategy: accept comments as-is from the parser
	// The internal parser has already applied v3 classification rules
	return nil
}

// ProcessNodeComments transfers comments from events to nodes.
//
// This is called when creating nodes from events in the decode.go parser.
// The plugin receives comment data from the current event (via ctx) and
// should populate the node's comment fields appropriately.
//
// This implements the logic from decode.go node() method (lines 163-165).
func (p *Plugin) ProcessNodeComments(node *yaml.Node, ctx *yaml.CommentContext) error {
	// Transfer comments from event context to node
	// This is the core v3 comment attachment logic
	node.HeadComment = string(ctx.HeadComment)
	node.LineComment = string(ctx.LineComment)
	node.FootComment = string(ctx.FootComment)
	return nil
}
