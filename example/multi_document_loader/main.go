// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Example: Multi-Document Loader demonstrates loading multiple YAML documents.

package main

import (
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Name    string   `yaml:"name"`
	Version string   `yaml:"version"`
	Tags    []string `yaml:"tags,omitempty"`
}

func main() {
	fmt.Println("Example 2: Multi-Document Loader")

	multiDoc := `---
name: app1
version: 1.0.0
---
name: app2
version: 2.0.0
tags:
  - experimental
---
name: app3
version: 3.0.0
`

	loader, err := yaml.NewLoader(strings.NewReader(multiDoc))
	if err != nil {
		panic(err)
	}

	docNum := 1
	for {
		var doc Config
		err := loader.Load(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Printf("Document %d: %+v\n", docNum, doc)
		docNum++
	}
}
