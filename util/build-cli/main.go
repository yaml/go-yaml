// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Build a CLI with CONFIG, PLUGIN, and PLUGIN-PACKAGE settings.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
	pluginreg "go.yaml.in/yaml/v4/internal/plugin"
	"go.yaml.in/yaml/v4/plugin/tabindent"
)

func main() {
	root, err := os.Getwd()
	if err == nil {
		err = buildCLIWithPlugins(
			root,
			os.Getenv("GO_YAML_BUILD_CONFIG"),
			os.Getenv("GO_YAML_BUILD_PLUGIN"),
			filepath.Join(root, "go-yaml"),
			os.Getenv("GO_YAML_BUILD_GO"),
			os.Getenv("GO_YAML_BUILD_PERL"),
			os.Getenv("GO_YAML_BUILD_PLUGIN_PACKAGE"),
		)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "build CLI:", err)
		os.Exit(1)
	}
}

type buildConfig struct {
	referenceParser  bool
	referenceVersion string
	embedded         []byte
}

var pluginVersion = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+$`)

var (
	registerBuildPluginsOnce sync.Once
	registerBuildPluginsErr  error
)

func registerBuildPlugins() error {
	registerBuildPluginsOnce.Do(func() {
		registerBuildPluginsErr = tabindent.Register()
	})
	return registerBuildPluginsErr
}

func canonicalVersion(value any, api string) (string, error) {
	version, ok := value.(string)
	if !ok || !pluginVersion.MatchString(version) {
		return "", fmt.Errorf(
			"plugin %q version must be a release version", api)
	}
	return "v" + strings.TrimPrefix(version, "v"), nil
}

func pluginConfig(api string, value any) (map[string]any, bool, error) {
	switch setting := value.(type) {
	case bool:
		return map[string]any{}, !setting, nil
	case string:
		config, err := pluginreg.ConfigValue(api, setting)
		return config, false, err
	case map[string]any:
		if raw, found := setting["disable"]; found {
			disabled, ok := raw.(bool)
			if !ok {
				return nil, false, fmt.Errorf(
					"plugin %q disable must be a boolean", api)
			}
			if disabled {
				return nil, true, nil
			}
		}
		config := make(map[string]any, len(setting))
		for key, item := range setting {
			if key != "disable" {
				config[key] = item
			}
		}
		return config, false, nil
	default:
		return nil, false, fmt.Errorf(
			"plugin %q value must be a mapping, string, or boolean", api)
	}
}

// inspectConfig validates core options and selects known compiled plugins.
// The completed binary validates embedded defaults against the real factories.
func inspectConfig(data []byte, external bool) (buildConfig, error) {
	selection := buildConfig{embedded: data}
	if err := registerBuildPlugins(); err != nil {
		return selection, err
	}
	var config map[string]any
	if err := yaml.Load(data, &config); err != nil {
		return selection, err
	}
	if config == nil {
		config = map[string]any{}
	}
	if value, present := config["plugin"]; present {
		plugins, ok := value.(map[string]any)
		if !ok {
			return selection, errors.New("plugin configuration must be a mapping")
		}
		if err := pluginreg.NormalizeConfig(plugins); err != nil {
			return selection, err
		}
		for api, value := range plugins {
			setting, disabled, err := pluginConfig(api, value)
			if err != nil {
				return selection, err
			}
			if disabled {
				continue
			}
			// External factories are available only in the staged executable.
			// Keep automatic reference-parser preparation and validate all
			// other plugin settings after registering the external packages.
			if external {
				if api != pluginreg.YAMLParserAPI || setting["name"] != "reference" {
					delete(plugins, api)
					continue
				}
			}
			name := ""
			if raw, found := setting["name"]; found {
				name, ok = raw.(string)
				if !ok || name == "" {
					return selection, fmt.Errorf(
						"plugin %q name must be a non-empty string", api)
				}
				name = pluginreg.CanonicalName(api, name)
			}
			switch api {
			case pluginreg.ErrorFormatAPI, pluginreg.LoaderLimitsAPI,
				pluginreg.MergeKeyAPI:
				_, shortForm := value.(string)
				if name != "" && name != api && !shortForm {
					return selection, fmt.Errorf(
						"no CLI build provider for plugin %q implementation %q",
						api, name)
				}
			case pluginreg.YAMLParserAPI:
				if name == "" {
					name = "go-yaml"
				}
				switch name {
				case "go-yaml":
				case "reference":
					selection.referenceParser = true
					if raw, found := setting["version"]; found {
						selection.referenceVersion, err = canonicalVersion(raw, api)
						if err != nil {
							return selection, err
						}
					}
					delete(plugins, api)
				default:
					return selection, fmt.Errorf(
						"no CLI build provider for plugin %q implementation %q",
						api, name)
				}
			case "tab-indent":
				switch name {
				case "", "tab-indent":
				default:
					return selection, fmt.Errorf(
						"no CLI build provider for plugin %q implementation %q",
						api, name)
				}
			default:
				return selection, fmt.Errorf(
					"no CLI build provider for plugin %q", api)
			}
		}
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

func mergePluginDSL(data []byte, specs string) ([]byte, error) {
	if strings.TrimSpace(specs) == "" {
		return data, nil
	}
	var config map[string]any
	if len(data) != 0 {
		if err := yaml.Load(data, &config); err != nil {
			return nil, err
		}
	}
	if config == nil {
		config = map[string]any{}
	}
	plugins := map[string]any{}
	if value, found := config["plugin"]; found {
		var ok bool
		plugins, ok = value.(map[string]any)
		if !ok {
			return nil, errors.New("plugin configuration must be a mapping")
		}
		if err := pluginreg.NormalizeConfig(plugins); err != nil {
			return nil, err
		}
	}
	selections, err := pluginreg.ParseSelectors(specs)
	if err != nil {
		return nil, err
	}
	for _, selection := range selections {
		if selection.Name != "" && selection.Version == "" {
			plugins[selection.API] = selection.Name
			continue
		}
		if selection.Name == "" && selection.Version == "" {
			plugins[selection.API] = true
			continue
		}
		setting := map[string]any{}
		if selection.Name != "" {
			setting["name"] = selection.Name
		}
		if selection.Version != "" {
			setting["version"] = selection.Version
		}
		plugins[selection.API] = setting
	}
	config["plugin"] = plugins
	return yaml.Dump(config)
}

var packageSpec = regexp.MustCompile(
	`^[A-Za-z0-9][A-Za-z0-9._~/-]*@v[0-9]+\.[0-9]+\.[0-9]+` +
		`(?:-[0-9A-Za-z.-]+)?(?:\+incompatible)?$`)

func parsePackages(specs string) ([]string, error) {
	if strings.TrimSpace(specs) == "" {
		return nil, nil
	}
	var packages []string
	seen := map[string]bool{}
	for _, spec := range strings.Split(specs, ",") {
		spec = strings.TrimSpace(spec)
		path, _, _ := strings.Cut(spec, "@")
		if !packageSpec.MatchString(spec) ||
			!strings.Contains(strings.Split(path, "/")[0], ".") {
			return nil, fmt.Errorf(
				"expected package/path@vMAJOR.MINOR.PATCH, got %q", spec)
		}
		for _, part := range strings.Split(path, "/") {
			if part == "" || part == "." || part == ".." {
				return nil, fmt.Errorf("invalid package path %q", path)
			}
		}
		if seen[path] {
			return nil, fmt.Errorf("duplicate plugin package %q", path)
		}
		seen[path] = true
		packages = append(packages, spec)
	}
	return packages, nil
}

func stagePackages(stage, goTool string, packages []string) error {
	if len(packages) == 0 {
		return nil
	}
	var source strings.Builder
	source.WriteString("package main\n\nimport (\n")
	for i, spec := range packages {
		path, _, _ := strings.Cut(spec, "@")
		fmt.Fprintf(&source, "plugin%d %q\n", i, path)
	}
	source.WriteString(
		")\n\nfunc init() {\n" +
			"compiledPluginRegistrations = append(" +
			"compiledPluginRegistrations,\n")
	for i := range packages {
		fmt.Fprintf(&source, "plugin%d.Register,\n", i)
	}
	source.WriteString(")\n}\n")
	if err := os.WriteFile(filepath.Join(stage, "external_plugins.go"),
		[]byte(source.String()), 0o644); err != nil {
		return err
	}
	for _, args := range [][]string{
		append([]string{"get"}, packages...),
		{"mod", "tidy"},
	} {
		cmd := exec.Command(goTool, args...)
		cmd.Dir = stage
		cmd.Env = buildEnvironment("off")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("prepare external plugins: %w\n%s", err, out)
		}
	}
	return nil
}

func buildCLI(root, configFile, output, goTool, perlTool string) error {
	return buildCLIWithPlugins(
		root, configFile, "", output, goTool, perlTool)
}

func buildCLIWithPlugins(
	root, configFile, pluginSpecs, output, goTool, perlTool string,
	packageSpecs ...string,
) error {
	packages, err := parsePackages(strings.Join(packageSpecs, ","))
	if err != nil {
		return fmt.Errorf("invalid PLUGIN-PACKAGE: %w", err)
	}
	var data []byte
	if configFile != "" {
		data, err = os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("read CONFIG: %w", err)
		}
	} else if pluginSpecs == "" && len(packages) == 0 {
		return errors.New("CONFIG, PLUGIN, or PLUGIN-PACKAGE must be set")
	}
	data, err = mergePluginDSL(data, pluginSpecs)
	if err != nil {
		return fmt.Errorf("invalid PLUGIN: %w", err)
	}
	if len(data) == 0 {
		data = []byte("{}\n")
	}
	selection, err := inspectConfig(data, len(packages) != 0)
	if err != nil {
		return fmt.Errorf("invalid CONFIG: %w", err)
	}
	if goTool == "" {
		goTool = "go"
	}
	if perlTool == "" {
		perlTool = "perl"
	}
	stage := filepath.Join(root, ".cache", "cli-config")
	if selection.referenceParser {
		cmd := exec.Command(perlTool, "util/prepare-plugins")
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GO_YAML_BUILD_GO="+goTool,
			"GO_YAML_BUILD_REFERENCE_PARSER="+
				strconv.FormatBool(selection.referenceParser),
			"GO_YAML_REFERENCE_PARSER_VERSION="+selection.referenceVersion)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("prepare plugins: %w\n%s", err, out)
		}
		stage = filepath.Join(root, ".cache", "cli-plugins")
	} else if err := stageNative(root, stage, goTool); err != nil {
		return err
	}
	if err := stagePackages(stage, goTool, packages); err != nil {
		return err
	}
	source := "package main\n\nfunc init() {\n\tdefaultConfig = " +
		strconv.Quote(string(selection.embedded)) + "\n}\n"
	if err := os.WriteFile(
		filepath.Join(stage, "config_defaults.go"),
		[]byte(source), 0o644,
	); err != nil {
		return err
	}

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
	cmd.Env = buildEnvironment("off")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compile configured CLI: %w\n%s", err, out)
	}
	check := exec.Command(binary, "--help")
	var stderr bytes.Buffer
	check.Stderr = &stderr
	if err := check.Run(); err != nil {
		return fmt.Errorf(
			"embedded configuration rejected: %w\n%s",
			err, stderr.String())
	}
	if err := os.Rename(binary, output); err != nil {
		return err
	}
	fmt.Printf("Built %s with configured plugins\n", output)
	return nil
}

func buildEnvironment(workspace string) []string {
	var env []string
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GOWORK=") &&
			!strings.HasPrefix(entry, "CGO_ENABLED=") {
			env = append(env, entry)
		}
	}
	return append(env, "GOWORK="+workspace, "CGO_ENABLED=0")
}

func stageNative(root, stage, goTool string) error {
	if err := os.RemoveAll(stage); err != nil {
		return err
	}
	source := filepath.Join(root, "cmd", "go-yaml")
	err := filepath.WalkDir(
		source,
		func(path string, entry fs.DirEntry, err error) error {
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
		},
	)
	if err != nil {
		return err
	}
	for _, args := range [][]string{
		{"mod", "init", "go.yaml.in/yaml/v4/config-cli"},
		{"mod", "edit", "-go=1.18"},
		{"mod", "edit", "-replace=go.yaml.in/yaml/v4=" + root},
		{"get", "go.yaml.in/yaml/v4@v4.0.0-rc.6"},
		{"mod", "tidy"},
	} {
		cmd := exec.Command(goTool, args...)
		cmd.Dir = stage
		cmd.Env = buildEnvironment("off")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf(
				"go %s: %w\n%s", strings.Join(args, " "), err, output)
		}
	}
	return nil
}
