// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package reference provides the YAMLStar reference parser for go-yaml.
package reference

import (
	"errors"
	"fmt"

	referenceparser "github.com/yamlstar/yamlstar-plugin-parser-reference/parser"
	"go.yaml.in/yaml/v4"
)

// Plugin parses YAML with the YAMLStar reference parser.
type Plugin struct{}

var _ yaml.ParserPlugin = (*Plugin)(nil)

// Version is the linked reference parser release version.
const Version = referenceparser.Version

// New creates a reference parser plugin.
func New() *Plugin { return &Plugin{} }

// Register enables parser=reference in yaml.OptsYAML.
func Register() error {
	return yaml.RegisterPlugin(yaml.PluginRegistration{
		API: "parser", Name: "reference", Version: Version,
		Factory: func(cfg map[string]any) (any, error) {
			if len(cfg) != 0 {
				return nil, errors.New(
					"reference parser configuration must be empty")
			}
			return New(), nil
		},
	})
}

// Parse implements yaml.ParserPlugin.
func (p *Plugin) Parse(input []byte) ([]yaml.PluginEvent, error) {
	source, err := referenceparser.Parse(input)
	if err != nil {
		return nil, err
	}
	events := make([]yaml.PluginEvent, len(source))
	for i, event := range source {
		events[i] = yaml.PluginEvent{
			Type: event.Type, Value: event.Value, Anchor: event.Anchor,
			Tag: event.Tag, Style: event.Style, Flow: event.Flow,
			Explicit: event.Explicit,
		}
		if event.Type == "alias" {
			events[i].Anchor = event.Name
		}
		if event.Version != "" {
			switch event.Version {
			case "1.1":
				events[i].Version = &yaml.VersionDirective{
					Major: 1, Minor: 1,
				}
			case "1.2":
				events[i].Version = &yaml.VersionDirective{
					Major: 1, Minor: 2,
				}
			default:
				return nil, fmt.Errorf(
					"reference parser: unsupported YAML version %q",
					event.Version)
			}
		}
	}
	return events, nil
}
