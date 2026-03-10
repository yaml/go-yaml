// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package v4 provides round-trip comment handling for YAML.
//
// Unlike the v3 comment plugin which strips whitespace and uses lossy
// heuristics, the v4 plugin preserves original comment formatting
// including whitespace before '#' and blank lines between nodes.
//
// # Release Candidate
//
// This plugin is under active development and requires an RC gate.
// You must specify which release candidate you are targeting:
//
//	plugin, err := v4.New(v4.RC("rc1"))
//
// When the plugin makes breaking changes, the RC version is bumped.
// If your specified RC doesn't match the current one, New() returns
// an error with instructions to update.
//
// # Usage
//
//	import "go.yaml.in/yaml/v4"
//	import "go.yaml.in/yaml/v4/plugin/comment/v4"
//
//	plugin, err := v4.New(v4.RC("rc1"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	loader, err := yaml.NewLoader(data, yaml.WithPlugin(plugin))
package v4
