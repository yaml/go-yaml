// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package v4migration provides opt-in adapters for migrating to go-yaml v4.
// The adapters construct genuine legacy nodes without pointer reinterpretation.
// Legacy Node.Decode uses that package's own decoder, without v4 loader options.
// Importing or registering the package does not enable migration.
package v4migration

import (
	"errors"

	"go.yaml.in/yaml/v4"
)

// Register makes both adapters available to yaml.OptsYAML.
// Select from-v3 or from-gopkg-v3 explicitly; neither is a default.
func Register() error {
	for _, implementation := range []struct {
		name      string
		newPlugin func() yaml.V4MigrationPlugin
	}{
		{"from-v3", FromV3},
		{"from-gopkg-v3", FromGopkgV3},
	} {
		newPlugin := implementation.newPlugin
		if err := yaml.RegisterPlugin(yaml.PluginRegistration{
			API: "v4-migration", Name: implementation.name,
			Factory: func(cfg map[string]any) (any, error) {
				if len(cfg) != 0 {
					return nil, errors.New(
						"v4-migration: configuration must be empty")
				}
				return newPlugin(), nil
			},
		}); err != nil {
			return err
		}
	}
	return nil
}
