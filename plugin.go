// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml

// ErrorPlugin customizes how YAML load and dump errors are rendered.
//
// When registered, FormatLoadError and FormatDumpError are called to produce
// the error message string for each [LoadError] and [DumpError], allowing the
// format to be customized.
//
// Example usage:
//
//	import errfmtv3 "go.yaml.in/yaml/v4/plugin/errfmt/v3"
//	loader := yaml.NewLoader(data, yaml.WithPlugin(errfmtv3.New()))
type ErrorPlugin interface {
	// FormatLoadError returns the string representation of a LoadError.
	// The returned string is used as the error message.
	FormatLoadError(err *LoadError) string

	// FormatDumpError returns the string representation of a DumpError.
	// The returned string is used as the error message.
	FormatDumpError(err *DumpError) string
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
