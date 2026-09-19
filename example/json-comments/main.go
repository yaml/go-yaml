package main

import (
	"encoding/json"
	"log"
	"os"
	"runtime/debug"

	"go.yaml.in/yaml/v4"
	jsoncomments "go.yaml.in/yaml/v4/plugin/json-comments"
)

const (
	pluginModule  = "github.com/yamlstar/yamlstar-plugin-json-comments"
	pluginVersion = "v0.1.8"
)

func main() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		log.Fatal("cannot inspect linked modules")
	}
	found := false
	for _, dependency := range info.Deps {
		if dependency.Path == pluginModule {
			found = true
			if dependency.Version != pluginVersion {
				log.Fatalf("expected %s, got %s",
					pluginVersion, dependency.Version)
			}
			break
		}
	}
	if !found {
		log.Fatalf("%s is not linked", pluginModule)
	}

	input, err := os.ReadFile("data.yaml")
	if err != nil {
		log.Fatal(err)
	}
	var value map[string]any
	if err = yaml.Load(input, &value,
		yaml.WithPlugin(jsoncomments.New())); err != nil {
		log.Fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(value); err != nil {
		log.Fatal(err)
	}
}
