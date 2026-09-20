// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package jsoncomments loads YAML with JSON-style comments using the
// YAMLStar generated reference parser. Input is buffered, comments are
// discarded, and node positions are unknown. Only UTF-8 input is supported.
package jsoncomments

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/yamlstar/yamlstar-plugin-json-comments/parser"
	"go.yaml.in/yaml/v4"
)

// Plugin supplies JSON-comments events to go-yaml.
type Plugin struct{}

var _ yaml.EventSourcePlugin = (*Plugin)(nil)

// New creates an event-source plugin.
// No global registration is needed for WithPlugin.
func New() *Plugin { return &Plugin{} }

// Register enables json-comments in yaml.OptsYAML and yaml.WithNamedPlugin.
// Call once at application startup. Duplicate registrations return an error.
func Register() error {
	version, err := linkedVersion()
	if err != nil {
		return err
	}
	return yaml.RegisterPlugin(yaml.PluginRegistration{
		API: "json-comments", Name: "json-comments", Version: version,
		Factory: func(cfg map[string]any) (any, error) {
			if len(cfg) != 0 {
				return nil, fmt.Errorf(
					"json-comments: configuration must be empty")
			}
			return New(), nil
		},
	})
}

func linkedVersion() (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", fmt.Errorf("json-comments: cannot inspect linked modules")
	}
	const module = "github.com/yamlstar/yamlstar-plugin-json-comments"
	for _, dependency := range info.Deps {
		if dependency.Path == module {
			return strings.TrimPrefix(dependency.Version, "v"), nil
		}
	}
	return "", fmt.Errorf("json-comments: %s is not linked", module)
}

// Parse implements yaml.EventSourcePlugin with the generated reference parser.
func (p *Plugin) Parse(input []byte) ([]yaml.PluginEvent, error) {
	source, err := parser.Parse(input)
	if err != nil {
		return nil, err
	}
	events := make([]yaml.PluginEvent, len(source))
	for i, e := range source {
		event := yaml.PluginEvent{
			Type: e.Type, Value: e.Value, Anchor: e.Anchor, Tag: e.Tag,
			Style: e.Style, Flow: e.Flow, Explicit: e.Explicit,
		}
		if e.Type == "alias" {
			event.Anchor = e.Name
		}
		if e.Version != "" {
			switch e.Version {
			case "1.1":
				event.Version = &yaml.VersionDirective{Major: 1, Minor: 1}
			case "1.2":
				event.Version = &yaml.VersionDirective{Major: 1, Minor: 2}
			default:
				return nil, fmt.Errorf("json-comments: unsupported YAML version %q", e.Version)
			}
		}
		events[i] = event
	}
	return events, nil
}
