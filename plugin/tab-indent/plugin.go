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
type Mode = yaml.TabIndentMode

const (
	ModeAuto = yaml.TabIndentModeAuto
	ModeTabs = yaml.TabIndentModeTabs
)

// Load controls accepted structural indentation.
type Load = yaml.TabIndentLoad

const (
	LoadTabs   = yaml.TabIndentLoadTabs
	LoadSpaces = yaml.TabIndentLoadSpaces
	LoadAuto   = yaml.TabIndentLoadAuto
)

// Dump controls emitted structural indentation.
type Dump = yaml.TabIndentDump

const (
	DumpTabs   = yaml.TabIndentDumpTabs
	DumpSpaces = yaml.TabIndentDumpSpaces
)

// Auto controls how long auto-detected indentation remains active.
type Auto = yaml.TabIndentAuto

const (
	AutoDocument = yaml.TabIndentAutoDocument
	AutoStream   = yaml.TabIndentAutoStream
)

// Plugin configures tab-aware YAML loading and dumping.
type Plugin struct {
	config yaml.TabIndentConfig
}

var _ yaml.TabIndentPlugin = (*Plugin)(nil)

// Option configures a [Plugin].
type Option func(*Plugin)

// New creates a tab-indentation plugin.
func New(opts ...Option) *Plugin {
	p := &Plugin{config: yaml.TabIndentConfig{
		Mode: ModeAuto,
		Auto: AutoDocument,
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
func WithLoad(load Load) Option {
	return func(p *Plugin) { p.config.Load = load }
}

// WithDump sets emitted structural indentation.
func WithDump(dump Dump) Option {
	return func(p *Plugin) { p.config.Dump = dump }
}

// WithAuto sets the auto-detection scope.
func WithAuto(auto Auto) Option {
	return func(p *Plugin) { p.config.Auto = auto }
}

// TabIndentConfig implements [yaml.TabIndentPlugin].
func (p *Plugin) TabIndentConfig() yaml.TabIndentConfig {
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
			p.config.Load = Load(text)
		case "dump":
			p.config.Dump = Dump(text)
		case "auto":
			p.config.Auto = Auto(text)
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
