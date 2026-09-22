// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"
	"strings"
)

// Selection identifies one plugin API, implementation, and optional version.
type Selection struct {
	API, Name, Version string
}

// ParseSelectors parses comma-separated plugin selector syntax.
func ParseSelectors(text string) ([]Selection, error) {
	var selections []Selection
	for _, raw := range strings.Split(text, ",") {
		spec := strings.TrimSpace(raw)
		if spec == "" {
			return nil, fmt.Errorf("plugin selector must not be empty")
		}
		if strings.Count(spec, "=") > 1 || strings.Count(spec, "@") > 1 {
			return nil, fmt.Errorf("invalid plugin selector %q", spec)
		}
		api, implementation, found := strings.Cut(spec, "=")
		if !found {
			implementation = ""
		}
		name, version, versioned := strings.Cut(implementation, "@")
		if !found {
			api, version, versioned = strings.Cut(api, "@")
			name = ""
		}
		if api == "" || (found && name == "") ||
			(versioned && version == "") {
			return nil, fmt.Errorf("invalid plugin selector %q", spec)
		}
		if versioned {
			if _, ok := normalizeVersion(version); !ok {
				return nil, fmt.Errorf(
					"plugin selector %q version must be a release version",
					spec)
			}
		}
		selections = append(selections, Selection{
			API: api, Name: name, Version: version,
		})
	}
	return selections, nil
}

// ConfigValue converts an implementation short form into registry config.
func ConfigValue(api, value string) (map[string]any, error) {
	selections, err := ParseSelectors(api + "=" + value)
	if err != nil {
		return nil, err
	}
	selection := selections[0]
	config := map[string]any{"name": selection.Name}
	if selection.Version != "" {
		config["version"] = selection.Version
	}
	return config, nil
}
