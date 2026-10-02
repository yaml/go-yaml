// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package tabindent configures structural YAML indentation.
//
// ModeAuto defaults to automatic loading and tab dumping.
// ModeTabs defaults to tab loading and tab dumping.
// Explicit load and dump options override those presets.
package tabindent

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

// Mode supplies loading and dumping defaults.
type Mode = yaml.IndentMode

const (
	ModeAuto = yaml.IndentModeAuto
	ModeTabs = yaml.IndentModeTabs
)

// Style identifies the characters used for structural indentation.
type Style = yaml.IndentStyle

const (
	StyleAuto   = yaml.IndentStyleAuto
	StyleSpaces = yaml.IndentStyleSpaces
	StyleTabs   = yaml.IndentStyleTabs
)

// Scope controls how long an auto-detected style remains active.
type Scope = yaml.IndentScope

const (
	ScopeDocument = yaml.IndentScopeDocument
	ScopeStream   = yaml.IndentScopeStream
)

// Plugin configures tab-aware YAML loading and dumping.
type Plugin struct {
	config yaml.IndentConfig
}

var _ yaml.TabIndentPlugin = (*Plugin)(nil)

// Option configures a [Plugin].
type Option func(*Plugin)

// New creates a tab-indentation plugin.
func New(opts ...Option) *Plugin {
	p := &Plugin{config: yaml.IndentConfig{
		Mode:  ModeAuto,
		Scope: ScopeDocument,
	}}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithMode sets the indentation mode.
func WithMode(mode Mode) Option {
	return func(p *Plugin) { p.config.Mode = mode }
}

// WithLoad sets accepted structural indentation.
func WithLoad(style Style) Option {
	return func(p *Plugin) { p.config.LoadStyle = style }
}

// WithDump sets emitted structural indentation.
func WithDump(style Style) Option {
	return func(p *Plugin) { p.config.DumpStyle = style }
}

// WithAuto sets the auto-detection scope.
func WithAuto(scope Scope) Option {
	return func(p *Plugin) { p.config.Scope = scope }
}

// TabIndentConfig implements [yaml.TabIndentPlugin].
func (p *Plugin) TabIndentConfig() yaml.IndentConfig {
	return p.config
}

// NewFromYAML creates a plugin from a named-plugin configuration.
func NewFromYAML(cfg map[string]any) (*Plugin, error) {
	p := New()
	for key, value := range cfg {
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf(
				"tab-indent: %s must be a string, got %T", key, value)
		}
		switch key {
		case "mode":
			p.config.Mode = Mode(text)
		case "load":
			p.config.LoadStyle = Style(text)
		case "dump":
			p.config.DumpStyle = Style(text)
		case "auto":
			p.config.Scope = Scope(text)
		default:
			return nil, fmt.Errorf("tab-indent: unknown key %q", key)
		}
	}
	config, err := p.config.Normalize()
	if err != nil {
		return nil, err
	}
	p.config = config
	return p, nil
}

// Register enables tab-indent in yaml.OptsYAML and yaml.WithNamedPlugin.
func Register() error {
	return yaml.RegisterPlugin(yaml.PluginRegistration{
		API:     "tab-indent",
		Name:    "tab-indent",
		Default: true,
		Factory: func(cfg map[string]any) (any, error) {
			return NewFromYAML(cfg)
		},
	})
}
