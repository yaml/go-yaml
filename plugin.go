// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml

// ErrorPlugin customizes how YAML load errors are rendered.
//
// When registered, FormatLoadError is called to produce the error message
// string for each [LoadError], allowing the format to be customized.
//
// Example usage:
//
//	import "go.yaml.in/yaml/v4/plugin/errfmt"
//	loader := yaml.NewLoader(data, yaml.WithPlugin(errfmt.New(errfmt.FormatLegacy)))
type ErrorPlugin interface {
	// FormatLoadError returns the string representation of a LoadError.
	// The returned string is used as the error message.
	FormatLoadError(err *LoadError) string
}

// LimitPlugin configures safety limits for YAML parsing.
//
// When registered, CheckDepth is called on each nesting depth increase,
// and CheckAlias is called on each alias expansion to detect excessive
// aliasing.
//
// Example usage:
//
//	import "go.yaml.in/yaml/v4/plugin/limit"
//	loader := yaml.NewLoader(data, yaml.WithPlugin(limit.New(limit.AliasNone())))
type LimitPlugin interface {
	// CheckDepth is called when the parser increases nesting depth.
	// depth is the current nesting level; ctx.Kind is "flow" or "block".
	// Return an error to abort parsing.
	CheckDepth(depth int, ctx *DepthContext) error

	// CheckAlias is called during alias expansion.
	// Return an error to abort construction.
	CheckAlias(aliasCount, constructCount int) error
}
