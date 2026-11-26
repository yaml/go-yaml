// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Command linkcheck is a minimal program that only calls yaml.Unmarshal.
// TestUnmarshalDoesNotLinkPluginFactories builds it and checks which plugin
// symbols the linker kept.
package main

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

func main() {
	var v map[string]any
	if err := yaml.Unmarshal([]byte("a: 1\n"), &v); err != nil {
		panic(err)
	}
	fmt.Println(v)
}
