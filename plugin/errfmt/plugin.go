// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package errfmt provides a configurable error formatting plugin for go-yaml.
//
// By default, go-yaml produces verbose structured error messages:
//
//	go-yaml load error in scanner (while scanning a plain scalar) at L2.C1-L2.C6: msg
//
// The errfmt plugin lets you choose between predefined formats:
//
//   - [FormatDefault] — the new verbose go-yaml format (the default)
//   - [FormatLegacy]  — the classic yaml-v2/v3 format: "yaml: line 2: msg"
//   - [FormatCompact] — a terse stage:line:col format: "scanner:2:1: msg"
//
// # Usage
//
//	import (
//	    "go.yaml.in/yaml/v4"
//	    "go.yaml.in/yaml/v4/plugin/errfmt"
//	)
//
//	// Use legacy error format
//	loader := yaml.NewLoader(data, yaml.WithPlugin(errfmt.New(errfmt.FormatLegacy)))
//
//	// Use compact format
//	loader := yaml.NewLoader(data, yaml.WithPlugin(errfmt.New(errfmt.FormatCompact)))
//
// # Third-Party Plugins
//
// You can implement [yaml.ErrorPlugin] directly instead of using this package:
//
//	type MyFmt struct{}
//	func (m *MyFmt) FormatLoadError(err *yaml.LoadError) string { return err.Message }
//	yaml.NewLoader(data, yaml.WithPlugin(&MyFmt{}))
package errfmt

import (
	"fmt"

	"go.yaml.in/yaml/v4/internal/libyaml"
)

// Format selects the error message format.
type Format int

const (
	// FormatDefault produces verbose structured messages.
	// Example: "go-yaml load error in scanner at L2.C6: msg"
	// Example: "go-yaml load error in scanner (while scanning a plain scalar) at L2.C1-L2.C6: msg"
	FormatDefault Format = iota

	// FormatLegacy reproduces the classic yaml-v2/v3 error format.
	// Example: "yaml: line 2: msg"
	FormatLegacy

	// FormatCompact produces a terse stage:line:col prefix.
	// Example: "scanner:2:6: msg"
	FormatCompact
)

// Plugin implements error formatting for YAML load errors.
type Plugin struct {
	format Format
}

// New creates an errfmt plugin using the specified format.
// With no argument, [FormatDefault] is used.
func New(format ...Format) *Plugin {
	f := FormatDefault
	if len(format) > 0 {
		f = format[0]
	}
	return &Plugin{format: f}
}

// FormatLoadError implements [yaml.ErrorPlugin].
func (p *Plugin) FormatLoadError(err *libyaml.LoadError) string {
	switch p.format {
	case FormatLegacy:
		return formatLegacy(err)
	case FormatCompact:
		return formatCompact(err)
	default:
		return formatDefault(err)
	}
}

// formatDefault reproduces the built-in go-yaml verbose format.
func formatDefault(e *libyaml.LoadError) string {
	if len(e.ContextMsg) > 0 {
		return fmt.Sprintf("go-yaml load error in %s (%s) at %s: %s",
			e.Stage, e.ContextMsg, e.ContextMark.RangeString(e.Mark), e.Message)
	}
	return fmt.Sprintf("go-yaml load error in %s at %s: %s",
		e.Stage, e.Mark.ShortString(), e.Message)
}

// formatLegacy reproduces the classic yaml-v2/v3 error format:
// "yaml: line N: msg"
func formatLegacy(e *libyaml.LoadError) string {
	line := e.Mark.Line
	if line == 0 {
		return fmt.Sprintf("yaml: %s", e.Message)
	}
	return fmt.Sprintf("yaml: line %d: %s", line, e.Message)
}

// formatCompact produces a terse stage:line:col: msg format.
func formatCompact(e *libyaml.LoadError) string {
	m := e.Mark
	if m.Line == 0 && m.Column == 0 {
		return fmt.Sprintf("%s: %s", e.Stage, e.Message)
	}
	return fmt.Sprintf("%s:%d:%d: %s", e.Stage, m.Line, m.Column, e.Message)
}

// NewFromYAML creates an errfmt plugin from a YAML config map.
// Key: "format" — one of "default", "legacy", "compact".
// Omitting the key uses [FormatDefault].
func NewFromYAML(cfg map[string]any) (*Plugin, error) {
	f := FormatDefault
	if val, ok := cfg["format"]; ok {
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("errfmt: format must be a string, got %T", val)
		}
		switch s {
		case "default":
			f = FormatDefault
		case "legacy":
			f = FormatLegacy
		case "compact":
			f = FormatCompact
		default:
			return nil, fmt.Errorf("errfmt: unknown format %q (want default, legacy, or compact)", s)
		}
	}
	for key := range cfg {
		if key != "format" {
			return nil, fmt.Errorf("errfmt: unknown key %q", key)
		}
	}
	return New(f), nil
}
