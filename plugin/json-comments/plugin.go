// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package jsoncomments removes JSON-style comments before YAML parsing.
// Only UTF-8 input is supported.
package jsoncomments

import (
	"fmt"

	"github.com/yamlstar/yamlstar-plugin-json-comments/sanitizer"
	"go.yaml.in/yaml/v4"
)

// Plugin sanitizes JSON-style comments before parsing.
type Plugin struct{}

var _ yaml.JSONCommentsPlugin = (*Plugin)(nil)

// Version is the linked sanitizer release version.
const Version = sanitizer.Version

// New creates a JSON-comments sanitizer plugin.
// No global registration is needed for WithPlugin.
func New() *Plugin { return &Plugin{} }

// Register enables json-comments in yaml.OptsYAML and yaml.WithNamedPlugin.
// Call once at application startup. Duplicate registrations return an error.
func Register() error {
	return yaml.RegisterPlugin(yaml.PluginRegistration{
		API: "json-comments", Name: "sanitizer", Version: Version,
		Default: true,
		Factory: func(cfg map[string]any) (any, error) {
			if len(cfg) != 0 {
				return nil, fmt.Errorf(
					"json-comments: configuration must be empty")
			}
			return New(), nil
		},
	})
}

// Sanitize implements yaml.JSONCommentsPlugin.
func (p *Plugin) Sanitize(input []byte) ([]byte, error) {
	return sanitizer.Sanitize(input)
}
