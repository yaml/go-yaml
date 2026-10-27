// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"path/filepath"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
)

// fileFunctionTags resolves constructor imports beside the CLI's input file.
type fileFunctionTags struct {
	yaml.FunctionTagPlugin
	directory string
}

func (p fileFunctionTags) ConstructFunctionTag(input yaml.FunctionTagInput) (any, error) {
	loadFile := input.LoadFile
	input.LoadFile = func(path string) (any, error) {
		if !filepath.IsAbs(path) {
			path = filepath.Join(p.directory, path)
		}
		return loadFile(path)
	}
	return p.FunctionTagPlugin.ConstructFunctionTag(input)
}

func withInputFile(path string) yaml.Option {
	return func(options *libyaml.Options) error {
		if options.FunctionTag != nil {
			options.FunctionTag = fileFunctionTags{
				FunctionTagPlugin: options.FunctionTag,
				directory:         filepath.Dir(path),
			}
		}
		return nil
	}
}
