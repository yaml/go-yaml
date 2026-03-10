// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml

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
