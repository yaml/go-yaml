// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package v4

import "go.yaml.in/yaml/v4/internal/libyaml"

// CommentMeta holds side-channel metadata for a node's comments.
// This avoids adding fields to the Node struct while preserving
// formatting information needed for round-trip fidelity.
type CommentMeta struct {
	BlankLinesBefore  int // blank lines before this node
	BlankLinesAfter   int // blank lines after this node
	LineCommentColumn int // original column of '#' in line comment
}

// Meta returns the comment metadata for a node, or nil if none exists.
func (p *Plugin) Meta(node *libyaml.Node) *CommentMeta {
	if p.meta == nil {
		return nil
	}
	return p.meta[node]
}

// getOrCreateMeta returns the comment metadata for a node, creating it
// if it doesn't exist.
func (p *Plugin) getOrCreateMeta(node *libyaml.Node) *CommentMeta {
	if p.meta == nil {
		p.meta = make(map[*libyaml.Node]*CommentMeta)
	}
	m, ok := p.meta[node]
	if !ok {
		m = &CommentMeta{}
		p.meta[node] = m
	}
	return m
}
