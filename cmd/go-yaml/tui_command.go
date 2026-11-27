// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"path/filepath"
	"strings"
)

func tuiCLICommand(binary, cwd string, args []string) (string, error) {
	relative, err := filepath.Rel(cwd, binary)
	if err != nil {
		return "", err
	}
	relative = filepath.ToSlash(relative)
	if !strings.HasPrefix(relative, "../") {
		relative = "./" + relative
	}
	parts := []string{shellQuote(relative)}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	parts = append(parts, ".go-yaml/input.yaml")
	return strings.Join(parts, " "), nil
}

func shellQuote(s string) string {
	if s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' || strings.ContainsRune("_@%+=:,./-", r))
	}) == -1 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
