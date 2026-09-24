// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package limit provides the deprecated loader-limits plugin API.
//
// Deprecated: use go.yaml.in/yaml/v4/plugin/loader-limits.
// This package will be removed before v4.0.0.
package limit

import (
	loaderlimits "go.yaml.in/yaml/v4/plugin/loader-limits"
)

// DepthContext is an alias for the type used in depth check callbacks.
//
// Deprecated: use loaderlimits.DepthContext.
type DepthContext = loaderlimits.DepthContext

// Plugin implements configurable loader limits.
//
// Deprecated: use loaderlimits.Plugin.
type Plugin = loaderlimits.Plugin

// Option configures a [Plugin].
//
// Deprecated: use loaderlimits.Option.
type Option = loaderlimits.Option

// New creates a loader-limits plugin.
//
// Deprecated: use loaderlimits.New.
func New(opts ...Option) *Plugin { return loaderlimits.New(opts...) }

// DepthValue sets a maximum nesting depth.
//
// Deprecated: use loaderlimits.DepthValue.
func DepthValue(n int) Option { return loaderlimits.DepthValue(n) }

// DepthNone disables depth checking.
//
// Deprecated: use loaderlimits.DepthNone.
func DepthNone() Option { return loaderlimits.DepthNone() }

// DepthFunc sets a custom depth check function.
//
// Deprecated: use loaderlimits.DepthFunc.
func DepthFunc(fn func(int, *DepthContext) error) Option {
	return loaderlimits.DepthFunc(fn)
}

// AliasValue sets a simple alias expansion count threshold.
//
// Deprecated: use loaderlimits.AliasValue.
func AliasValue(n int) Option { return loaderlimits.AliasValue(n) }

// AliasNone disables alias checking.
//
// Deprecated: use loaderlimits.AliasNone.
func AliasNone() Option { return loaderlimits.AliasNone() }

// AliasFunc sets a custom alias check function.
//
// Deprecated: use loaderlimits.AliasFunc.
func AliasFunc(fn func(int, int) error) Option {
	return loaderlimits.AliasFunc(fn)
}

// NewFromYAML creates a loader-limits plugin from YAML configuration.
//
// Deprecated: use loaderlimits.NewFromYAML.
func NewFromYAML(cfg map[string]any) (*Plugin, error) {
	return loaderlimits.NewFromYAML(cfg)
}
