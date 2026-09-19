// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Build a CLI with the plugins and defaults named by CONFIG.
package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
)

func main() {
	root, err := os.Getwd()
	if err == nil {
		err = buildCLI(root, os.Getenv("GO_YAML_BUILD_CONFIG"),
			filepath.Join(root, "go-yaml"), os.Getenv("GO_YAML_BUILD_GO"),
			os.Getenv("GO_YAML_BUILD_PERL"))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "build CLI:", err)
		os.Exit(1)
	}
}

type buildConfig struct {
	jsonComments bool
	version      string
	embedded     []byte
}

var pluginVersion = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+$`)

// inspectConfig validates core options and selects known compiled plugins.
// The completed binary validates embedded defaults against the real factories.
func inspectConfig(data []byte) (buildConfig, error) {
	selection := buildConfig{embedded: data}
	var config map[string]any
	if err := yaml.Load(data, &config); err != nil {
		return selection, err
	}
	if value, present := config["plugin"]; present {
		plugins, ok := value.(map[string]any)
		if !ok {
			return selection, fmt.Errorf("plugin configuration must be a mapping")
		}
		for name, value := range plugins {
			disabled := false
			switch setting := value.(type) {
			case bool:
				if !setting {
					disabled = true
				}
			case map[string]any:
				if raw, found := setting["disable"]; found {
					disabled, ok = raw.(bool)
					if !ok {
						return selection, fmt.Errorf(
							"plugin %q disable must be a boolean", name)
					}
				}
			default:
				return selection, fmt.Errorf(
					"plugin %q value must be a mapping or boolean", name)
			}
			if disabled {
				continue
			}
			switch name {
			case "limit":
			case "json-comments":
				if setting, ok := value.(map[string]any); ok {
					implementation := name
					if raw, found := setting["name"]; found {
						implementation, ok = raw.(string)
						if !ok || implementation == "" {
							return selection, fmt.Errorf(
								"plugin %q name must be a non-empty string", name)
						}
					}
					if implementation != "json-comments" {
						return selection, fmt.Errorf(
							"no CLI build provider for plugin %q implementation %q",
							name, implementation)
					}
					if raw, found := setting["version"]; found {
						version, ok := raw.(string)
						if !ok || !pluginVersion.MatchString(version) {
							return selection, fmt.Errorf(
								"plugin %q version must be a release version",
								name)
						}
						selection.version = "v" + strings.TrimPrefix(version, "v")
					}
				}
				selection.jsonComments = true
			default:
				return selection, fmt.Errorf(
					"no CLI build provider for plugin %q", name)
			}
		}
		// Validation uses the actual factory after compilation.
		delete(plugins, "json-comments")
	}
	plain, err := yaml.Dump(config)
	if err != nil {
		return selection, err
	}
	options, err := yaml.OptsYAML(string(plain))
	if err != nil {
		return selection, err
	}
	_, err = libyaml.ApplyOptions(options)
	return selection, err
}

func buildCLI(root, configFile, output, goTool, perlTool string) error {
	if configFile == "" {
		return fmt.Errorf("CONFIG must name a YAML options file")
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read CONFIG: %w", err)
	}
	selection, err := inspectConfig(data)
	if err != nil {
		return fmt.Errorf("invalid CONFIG: %w", err)
	}
	if goTool == "" {
		goTool = "go"
	}
	if perlTool == "" {
		perlTool = "perl"
	}
	var stage, workspace string
	if selection.jsonComments {
		cmd := exec.Command(perlTool, "util/prepare-json-comments")
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GO_YAML_BUILD_GO="+goTool,
			"GO_YAML_JSON_COMMENTS_VERSION="+selection.version)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("prepare JSON-comments: %w\n%s", err, out)
		}
		stage = filepath.Join(root, ".cache", "cli-json-comments")
		workspace = filepath.Join(root, ".cache", "json-comments.work")
	} else {
		stage = filepath.Join(root, ".cache", "cli-config")
		workspace = "off"
		if err := stageNative(root, stage); err != nil {
			return err
		}
	}
	source := "package main\n\nfunc init() {\n\tdefaultConfig = " +
		strconv.Quote(string(selection.embedded)) + "\n}\n"
	if err := os.WriteFile(filepath.Join(stage, "config_defaults.go"), []byte(source), 0o644); err != nil {
		return err
	}

	// Build beside the destination and replace it only after validation.
	temporary, err := os.CreateTemp(filepath.Dir(output), ".go-yaml-build-*")
	if err != nil {
		return err
	}
	binary := temporary.Name()
	if err := temporary.Close(); err != nil {
		return err
	}
	defer os.Remove(binary)
	cmd := exec.Command(goTool, "build", "-o", binary, ".")
	cmd.Dir = stage
	cmd.Env = buildEnvironment(workspace)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compile configured CLI: %w", err)
	}
	// Help validates options before inspecting stdin or loading input.
	check := exec.Command(binary, "--help")
	var stderr bytes.Buffer
	check.Stderr = &stderr
	if err := check.Run(); err != nil {
		return fmt.Errorf("embedded configuration rejected: %w\n%s", err, stderr.String())
	}
	if err := os.Rename(binary, output); err != nil {
		return err
	}
	fmt.Printf("Built %s with defaults from %s\n", output, configFile)
	return nil
}

func buildEnvironment(workspace string) []string {
	var env []string
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GOWORK=") && !strings.HasPrefix(entry, "CGO_ENABLED=") {
			env = append(env, entry)
		}
	}
	return append(env, "GOWORK="+workspace, "CGO_ENABLED=0")
}

func stageNative(root, stage string) error {
	if err := os.RemoveAll(stage); err != nil {
		return err
	}
	source := filepath.Join(root, "cmd", "go-yaml")
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(stage, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected CLI source file %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return err
	}
	mod := "module go.yaml.in/yaml/v4/config-cli\n\ngo 1.18\n\n" +
		"require go.yaml.in/yaml/v4 v4.0.0-rc.6\n\n" +
		"replace go.yaml.in/yaml/v4 => " + strconv.Quote(root) + "\n"
	return os.WriteFile(filepath.Join(stage, "go.mod"), []byte(mod), 0o644)
}
