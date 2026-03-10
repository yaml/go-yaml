// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml

import "go.yaml.in/yaml/v4/internal/libyaml"

// LimitsPlugin configures safety limits for YAML parsing.
//
// When registered, CheckDepth is called on each nesting depth increase,
// and CheckAlias is called on each alias expansion to detect excessive
// aliasing.
//
// Example usage:
//
//	import "go.yaml.in/yaml/v4/plugin/limits"
//	loader := yaml.NewLoader(data, yaml.WithPlugin(limits.New(limits.AliasNone())))
type LimitsPlugin interface {
	// CheckDepth is called when the parser increases nesting depth.
	// depth is the current nesting level; ctx.Kind is "flow" or "block".
	// Return an error to abort parsing.
	CheckDepth(depth int, ctx *DepthContext) error

	// CheckAlias is called during alias expansion.
	// Return an error to abort construction.
	CheckAlias(aliasCount, constructCount int) error
}

// CommentPlugin processes comments during YAML parsing.
//
// Each method returns a bool indicating whether it handled the comment.
// If handled=false, the caller runs default behavior.
//
// Plugins should embed DefaultCommentBehavior and override only the methods
// they need.
//
// Example usage:
//
//	loader := yaml.NewLoader(data, yaml.WithPlugin(commentPlugin))
type CommentPlugin interface {
	// ProcessEventComments is called at event creation (8 sites in parser).
	// Plugin can modify the event's comment fields and/or the comment queue.
	// Return true to skip default processing.
	ProcessEventComments(ctx *EventCommentContext) bool

	// ProcessComment is called when each node is created in the composer.
	// Plugin attaches event comments to the node.
	// Return true to skip default processing.
	ProcessComment(node *Node, ctx *CommentContext) (bool, error)

	// ProcessMappingPair is called after each mapping key-value pair.
	// Plugin handles foot comment migration, tail comments.
	// Return true to skip default processing.
	ProcessMappingPair(ctx *MappingPairContext) (bool, error)

	// ProcessEndComments is called after composing a collection or document.
	// Plugin handles end-event comments (Line, Foot).
	// Return true to skip default processing.
	ProcessEndComments(node *Node, ctx *CommentContext) (bool, error)
}

// DumpCommentPlugin processes comments during YAML serialization and
// emission.
// Plugins that implement both CommentPlugin and DumpCommentPlugin can
// achieve full round-trip comment fidelity.
type DumpCommentPlugin = libyaml.DumpCommentPlugin

// CommentKind identifies the type of comment being emitted.
type CommentKind = libyaml.CommentKind

// Comment kind constants.
const (
	HeadCommentKind = libyaml.HeadCommentKind
	LineCommentKind = libyaml.LineCommentKind
	FootCommentKind = libyaml.FootCommentKind
	TailCommentKind = libyaml.TailCommentKind
)

// DefaultCommentBehavior returns handled=false for all hooks.
// Embed in plugin structs to only override methods you need.
type DefaultCommentBehavior = libyaml.DefaultCommentBehavior

// EventCommentContext holds comment data at parser level when creating events.
type EventCommentContext = libyaml.EventCommentContext

// MappingPairContext holds context for processing mapping key-value pairs.
type MappingPairContext = libyaml.MappingPairContext
