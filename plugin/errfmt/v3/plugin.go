// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package errfmtv3 provides the go-yaml v3-compatible load error formatter.
//
// The v3 formatter renders errors as "yaml: line N: message", matching the
// legacy go-yaml error style.
package errfmtv3

import (
	"fmt"

	"go.yaml.in/yaml/v4/internal/libyaml"
)

// Plugin implements the v3-compatible YAML load error formatter.
type Plugin struct{}

// New creates a v3-compatible error formatting plugin.
func New() *Plugin {
	return &Plugin{}
}

// FormatLoadError implements [yaml.ErrorPlugin].
func (p *Plugin) FormatLoadError(err *libyaml.LoadError) string {
	line := err.Mark.Line
	if line == 0 {
		return fmt.Sprintf("yaml: %s", err.Message)
	}
	return fmt.Sprintf("yaml: line %d: %s", line, err.Message)
}

// NewFromYAML creates a v3 error formatting plugin from YAML config.
func NewFromYAML(cfg map[string]any) (*Plugin, error) {
	for key := range cfg {
		return nil, fmt.Errorf("errfmt v3: unknown key %q", key)
	}
	return New(), nil
}
