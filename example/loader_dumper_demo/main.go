// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
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
	fmt.Println("=== YAML Loader/Dumper Demo ===")

	// Example 1: Basic Loader usage with single document
	fmt.Println("1. Basic Loader - Single Document:")
	singleDoc := `name: myapp
version: 1.0.0
tags:
  - web
  - api
`
	var cfg Config
	loader, err := yaml.NewLoader(strings.NewReader(singleDoc))
	if err != nil {
		panic(err)
	}
	if err := loader.Load(&cfg); err != nil {
		panic(err)
	}
	fmt.Printf("   Loaded: %+v\n\n", cfg)

	// Example 2: Multi-document stream
	fmt.Println("2. Multi-Document Loader:")
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
	loader2, err := yaml.NewLoader(strings.NewReader(multiDoc))
	if err != nil {
		panic(err)
	}

	docNum := 1
	for {
		var doc Config
		err := loader2.Load(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Printf("   Document %d: %+v\n", docNum, doc)
		docNum++
	}
	fmt.Println()

	// Example 3: WithSingleDocument option
	fmt.Println("3. Loader with WithSingleDocument (stops after first doc):")
	loader3, err := yaml.NewLoader(strings.NewReader(multiDoc), yaml.WithSingleDocument())
	if err != nil {
		panic(err)
	}

	var firstDoc Config
	if err := loader3.Load(&firstDoc); err != nil {
		panic(err)
	}
	fmt.Printf("   First document: %+v\n", firstDoc)

	var secondDoc Config
	err = loader3.Load(&secondDoc)
	if err == io.EOF {
		fmt.Println("   Second Load() returned io.EOF (as expected)")
	}

	// Example 4: Dumper - single document
	fmt.Println("4. Basic Dumper - Single Document:")
	var buf bytes.Buffer
	dumper, err := yaml.NewDumper(&buf)
	if err != nil {
		panic(err)
	}

	cfg1 := Config{Name: "service1", Version: "1.0.0", Tags: []string{"prod"}}
	if err := dumper.Dump(&cfg1); err != nil {
		panic(err)
	}
	if err := dumper.Close(); err != nil {
		panic(err)
	}
	fmt.Printf("   Output:\n%s\n", buf.String())

	// Example 5: Dumper - multiple documents
	fmt.Println("5. Dumper - Multiple Documents:")
	buf.Reset()
	dumper2, err := yaml.NewDumper(&buf)
	if err != nil {
		panic(err)
	}

	cfg2 := Config{Name: "service1", Version: "1.0.0"}
	cfg3 := Config{Name: "service2", Version: "2.0.0", Tags: []string{"dev"}}
	cfg4 := Config{Name: "service3", Version: "3.0.0"}

	if err := dumper2.Dump(&cfg2); err != nil {
		panic(err)
	}
	if err := dumper2.Dump(&cfg3); err != nil {
		panic(err)
	}
	if err := dumper2.Dump(&cfg4); err != nil {
		panic(err)
	}
	if err := dumper2.Close(); err != nil {
		panic(err)
	}
	fmt.Printf("   Output:\n%s\n", buf.String())

	// Example 6: Dumper with options
	fmt.Println("6. Dumper with WithIndent(4):")
	buf.Reset()
	dumper3, err := yaml.NewDumper(&buf, yaml.WithIndent(4))
	if err != nil {
		panic(err)
	}

	cfg5 := Config{Name: "service", Version: "1.0.0", Tags: []string{"a", "b", "c"}}
	if err := dumper3.Dump(&cfg5); err != nil {
		panic(err)
	}
	if err := dumper3.Close(); err != nil {
		panic(err)
	}
	fmt.Printf("   Output (4-space indent):\n%s\n", buf.String())

	fmt.Println("=== Demo Complete ===")
}
