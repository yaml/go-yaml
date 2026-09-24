package main

import (
	"encoding/json"
	"log"
	"os"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/jsoncomments"
)

const pluginOptions = `
plugin:
  json-comments:
    version: v0.1.9
`

func main() {
	if err := jsoncomments.Register(); err != nil {
		log.Fatal(err)
	}
	options, err := yaml.OptsYAML(pluginOptions)
	if err != nil {
		log.Fatal(err)
	}

	input, err := os.ReadFile("data.yaml")
	if err != nil {
		log.Fatal(err)
	}
	var value map[string]any
	if err = yaml.Load(input, &value, options); err != nil {
		log.Fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(value); err != nil {
		log.Fatal(err)
	}
}
