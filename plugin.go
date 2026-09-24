// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml

import (
	"errors"

	"go.yaml.in/yaml/v4/internal/libyaml"
	pluginreg "go.yaml.in/yaml/v4/internal/plugin"
	loaderlimits "go.yaml.in/yaml/v4/plugin/loader-limits"
)

// LoaderLimitsPlugin configures safety limits for YAML loading.
//
// When registered, CheckDepth is called on each nesting depth increase,
// and CheckAlias is called on each alias expansion to detect excessive
// aliasing.
//
// Example usage:
//
//	import loaderlimits "go.yaml.in/yaml/v4/plugin/loader-limits"
//	loader := yaml.NewLoader(data,
//	    yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasNone())))
type LoaderLimitsPlugin interface {
	// CheckDepth is called when the parser increases nesting depth.
	// depth is the current nesting level; ctx.Kind is "flow" or "block".
	// Return an error to abort parsing.
	CheckDepth(depth int, ctx *DepthContext) error

	// CheckAlias is called during alias expansion.
	// Return an error to abort construction.
	CheckAlias(aliasCount, constructCount int) error
}

// LimitPlugin is the former name of [LoaderLimitsPlugin].
//
// Deprecated: use LoaderLimitsPlugin.
// This alias will be removed before v4.0.0.
type LimitPlugin = LoaderLimitsPlugin

// YAMLParserPlugin supplies a complete event stream in place of native parsing.
type YAMLParserPlugin = libyaml.YAMLParserPlugin

// JSONCommentsPlugin sanitizes JSON-style comments before parsing.
type JSONCommentsPlugin = libyaml.JSONCommentsPlugin

// PluginEvent carries source-independent YAML syntax information.
type PluginEvent = libyaml.PluginEvent

// PluginFactory constructs a plugin from its plugin-specific configuration.
type PluginFactory = pluginreg.Factory

// PluginRegistration describes one named implementation of a plugin API.
// Version is empty for an unversioned implementation.
// Default selects the implementation used by boolean configuration.
type PluginRegistration = pluginreg.Registration

type nativeYAMLParserPlugin struct{}

var pluginRegistry = pluginreg.NewRegistry(
	PluginRegistration{
		API:     pluginreg.LoaderLimitsAPI,
		Name:    pluginreg.LoaderLimitsAPI,
		Default: true,
		Factory: func(cfg map[string]any) (any, error) {
			return loaderlimits.NewFromYAML(cfg)
		},
	},
	PluginRegistration{
		API: pluginreg.YAMLParserAPI, Name: "go-yaml", Default: true,
		Factory: func(cfg map[string]any) (any, error) {
			if len(cfg) != 0 {
				return nil, errors.New(
					"yaml: go-yaml parser configuration must be empty")
			}
			return nativeYAMLParserPlugin{}, nil
		},
	},
)

// RegisterPlugin registers a named implementation for [OptsYAML] and
// [WithNamedPlugin]. Register during application startup, before parsing
// configuration. Duplicate API and implementation name pairs are errors.
func RegisterPlugin(registration PluginRegistration) error {
	return pluginRegistry.Register(registration)
}

func namedPlugin(api string, cfg map[string]any) (any, error) {
	return pluginRegistry.Create(api, cfg)
}

// WithNamedPlugin selects the implementation whose name matches the API.
func WithNamedPlugin(api string) Option {
	return func(o *libyaml.Options) error {
		p, err := namedPlugin(api, map[string]any{})
		if err != nil {
			return err
		}
		return WithPlugin(p)(o)
	}
}
