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
// It embeds DefaultCommentBehavior and overrides specific methods.
type CommentPlugin struct {
	libyaml.DefaultCommentBehavior
}

// Options configures the V3 comment plugin.
type Options struct{}

// New creates a new V3 comment plugin.
func New(opts ...Options) *CommentPlugin {
	return &CommentPlugin{}
}

// ProcessComment attaches comments to the node in V3 style.
// This directly populates the node's comment fields from the context.
func (p *CommentPlugin) ProcessComment(node *libyaml.Node, ctx *libyaml.CommentContext) (bool, error) {
	node.HeadComment = string(ctx.HeadComment)
	node.LineComment = string(ctx.LineComment)
	node.FootComment = string(ctx.FootComment)
	return true, nil
}

// ProcessMappingPair handles foot comment migration for mapping key-value pairs.
// This implements the V3 behavior of moving comments between keys and values.
func (p *CommentPlugin) ProcessMappingPair(ctx *libyaml.MappingPairContext) (bool, error) {
	k, v, n := ctx.Key, ctx.Value, ctx.Mapping
	if ctx.Block && k.FootComment != "" {
		if len(n.Content) > 2 {
			n.Content[len(n.Content)-3].FootComment = k.FootComment
			k.FootComment = ""
		}
	}
	if k.FootComment == "" && v.FootComment != "" {
		k.FootComment = v.FootComment
		v.FootComment = ""
	}
	if ctx.TailComment != nil && k.FootComment == "" {
		k.FootComment = string(ctx.TailComment)
	}
	return true, nil
}

// ProcessEndComments handles end-event comments for collections and documents.
// This implements the V3 behavior of attaching LineComment and FootComment.
func (p *CommentPlugin) ProcessEndComments(node *libyaml.Node, ctx *libyaml.CommentContext) (bool, error) {
	node.LineComment = string(ctx.LineComment)
	node.FootComment = string(ctx.FootComment)
	if node.Kind == libyaml.MappingNode &&
		node.Style&libyaml.FlowStyle == 0 &&
		node.FootComment != "" &&
		len(node.Content) > 1 {
		node.Content[len(node.Content)-2].FootComment = node.FootComment
		node.FootComment = ""
	}
	return true, nil
}
