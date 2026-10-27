package main

import (
	"log"
	"os"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/aliasdata"
	"go.yaml.in/yaml/v4/plugin/functiontag"
)

func main() {
	aliases, err := aliasdata.New(aliasdata.WithPath())
	if err != nil {
		log.Fatal(err)
	}
	functions, err := functiontag.New(functiontag.Functions{
		"load": aliasdata.Load,
	})
	if err != nil {
		log.Fatal(err)
	}
	options := []yaml.Option{
		yaml.WithPlugin(aliases),
		yaml.WithPlugin(functions),
	}
	input, err := os.ReadFile("input.yaml")
	if err != nil {
		log.Fatal(err)
	}
	var value any
	if err := yaml.Load(input, &value, options...); err != nil {
		log.Fatal(err)
	}
	output, err := yaml.Dump(value, options...)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(output); err != nil {
		log.Fatal(err)
	}
}
