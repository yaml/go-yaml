package main

import (
	"bytes"
	"fmt"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Name    string   `yaml:"name"`
	Version string   `yaml:"version"`
	Tags    []string `yaml:"tags,omitempty"`
}

func main() {
	fmt.Println("Example 4: Basic Dumper - Single Document")

	var buf bytes.Buffer
	dumper, err := yaml.NewDumper(&buf)
	if err != nil {
		panic(err)
	}

	cfg := Config{
		Name:    "service1",
		Version: "1.0.0",
		Tags:    []string{"prod"},
	}

	if err := dumper.Dump(&cfg); err != nil {
		panic(err)
	}

	if err := dumper.Close(); err != nil {
		panic(err)
	}

	fmt.Printf("Output:\n%s", buf.String())
}
