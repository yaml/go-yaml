// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import "strings"

const (
	tuiConfigKey  = "ctrl+g"
	tuiEditKey    = "ctrl+e"
	tuiRunKey     = "ctrl+r"
	tuiCommandKey = "ctrl+s"
	tuiLayoutKey  = "ctrl+l"
	tuiQuitKey    = "ctrl+q"
)

func tuiKeyLabel(key string) string {
	parts := strings.Split(key, "+")
	for i, part := range parts {
		switch part {
		case "ctrl":
			parts[i] = "Ctrl"
		case "alt":
			parts[i] = "Alt"
		default:
			parts[i] = strings.ToUpper(part)
		}
	}
	return strings.Join(parts, "-")
}

func tuiShortcutHelp(separator string) string {
	bindings := []struct{ key, action string }{
		{tuiConfigKey, "toggle config pane"},
		{tuiEditKey, "external edit"},
		{tuiRunKey, "run"},
		{tuiCommandKey, "show CLI command"},
		{tuiLayoutKey, "change layout"},
		{tuiQuitKey, "save and quit"},
	}
	var labels []string
	for _, binding := range bindings {
		labels = append(labels, tuiKeyLabel(binding.key)+" "+binding.action)
	}
	return strings.Join(labels, separator)
}
