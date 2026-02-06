//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

// Package v3legacy provides V3-compatible comment handling.
//
// This plugin replicates the comment attachment behavior from go-yaml v3,
// providing backward compatibility for code that relies on comment processing.
//
// # Usage
//
//	import "go.yaml.in/yaml/v4"
//	import "go.yaml.in/yaml/v4/plugin/comment/v3legacy"
//
//	loader := yaml.NewLoader(data, yaml.WithPlugin(v3legacy.New()))
//	var result interface{}
//	loader.Load(&result)
//
// # Alternative
//
// For simpler use cases, consider WithV3LegacyComments() instead:
//
//	loader := yaml.NewLoader(data, yaml.WithV3LegacyComments())
package v3legacy

import "go.yaml.in/yaml/v4/internal/libyaml"

// CommentPlugin implements V3-style comment processing.
type CommentPlugin struct{}

// Options configures the V3 comment plugin.
type Options struct{}

// New creates a new V3 comment plugin.
func New(opts ...Options) *CommentPlugin {
	return &CommentPlugin{}
}

// ProcessComment attaches comments to the node in V3 style.
// This directly populates the node's comment fields from the context.
func (p *CommentPlugin) ProcessComment(node *libyaml.Node, ctx *libyaml.CommentContext) error {
	node.HeadComment = string(ctx.HeadComment)
	node.LineComment = string(ctx.LineComment)
	node.FootComment = string(ctx.FootComment)
	// TailComment and StemComment are handled separately in the composer
	return nil
}
