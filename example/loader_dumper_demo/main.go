package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/comment/v3"
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

	// Example 7: With Comments Plugin
	fmt.Println("7. Loader with Comments Plugin:")
	yamlWithComments := `# This is the app configuration
name: myapp  # Application name
version: 1.0.0

# List of tags
tags:
  - web
  - api
`
	commentsPlugin := v3.New()
	loader4, err := yaml.NewLoader(
		strings.NewReader(yamlWithComments),
		yaml.WithPlugins(commentsPlugin),
	)
	if err != nil {
		panic(err)
	}

	var node yaml.Node
	if err := loader4.Load(&node); err != nil {
		panic(err)
	}

	fmt.Println("   Node with comments loaded successfully")
	fmt.Printf("   Document has %d top-level items\n", len(node.Content[0].Content))
	if len(node.Content[0].Content) > 0 && node.Content[0].Content[0].HeadComment != "" {
		fmt.Printf("   First field head comment: %q\n", node.Content[0].Content[0].HeadComment)
	}
	fmt.Println()

	// Example 8: Round-trip with comments
	fmt.Println("8. Round-trip with Comments (Load + Dump):")
	buf.Reset()
	dumper4, err := yaml.NewDumper(&buf, yaml.WithPlugins(commentsPlugin))
	if err != nil {
		panic(err)
	}

	if err := dumper4.Dump(&node); err != nil {
		panic(err)
	}
	if err := dumper4.Close(); err != nil {
		panic(err)
	}
	fmt.Printf("   Comments preserved in output:\n%s", buf.String())

	// Example 9: Combining multiple options
	fmt.Println("9. Loader with Multiple Options:")
	fmt.Println("   (WithSingleDocument + WithKnownFields + WithPlugin)")

	multiDocWithUnknown := `---
name: app1
version: 1.0.0
unknownField: this-will-be-rejected
---
name: app2
version: 2.0.0
`
	loader5, err := yaml.NewLoader(
		strings.NewReader(multiDocWithUnknown),
		yaml.WithSingleDocument(),
		yaml.WithKnownFields(),
		yaml.WithPlugins(commentsPlugin),
	)
	if err != nil {
		panic(err)
	}

	var strictCfg Config
	err = loader5.Load(&strictCfg)
	if err != nil {
		fmt.Printf("   ✓ Got expected error (unknown field): %v\n", err)
	} else {
		fmt.Printf("   Unexpected: no error for unknown field\n")
	}

	// Try with known fields only
	validDoc := `---
name: validapp
version: 1.0.0
tags:
  - valid
---
name: app2
version: 2.0.0
`
	loader6, err := yaml.NewLoader(
		strings.NewReader(validDoc),
		yaml.WithSingleDocument(),
		yaml.WithKnownFields(),
	)
	if err != nil {
		panic(err)
	}

	var validCfg Config
	if err := loader6.Load(&validCfg); err != nil {
		panic(err)
	}
	fmt.Printf("   ✓ First doc loaded: %+v\n", validCfg)

	// Second call should return EOF due to WithSingleDocument
	var secondCfg Config
	err = loader6.Load(&secondCfg)
	if err == io.EOF {
		fmt.Println("   ✓ Second Load() returned io.EOF (WithSingleDocument enforced)")
	}
	fmt.Println()

	fmt.Println("=== Demo Complete ===")
}
