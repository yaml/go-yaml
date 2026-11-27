// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
)

var tuiOutputNames = []string{"yaml", "yaml-preserved", "json", "json-pretty", "tokens", "tokens-detailed", "events", "events-detailed", "nodes", "nodes-detailed"}

type tuiSavedSession struct {
	Version int              `yaml:"version"`
	Layout  string           `yaml:"layout"`
	Output  string           `yaml:"output"`
	Live    bool             `yaml:"live"`
	Split   float64          `yaml:"split,omitempty"`
	Args    []string         `yaml:"args,omitempty"`
	Options map[string]*bool `yaml:"options"`
}

func tuiChoice(value string, choices []string) (int, error) {
	for i, c := range choices {
		if value == c {
			return i, nil
		}
	}
	return 0, fmt.Errorf("invalid choice %q; choose %s", value, strings.Join(choices, ", "))
}

func tuiLayoutNames() []string {
	var names []string
	for _, n := range tuiLayouts {
		names = append(names, strings.ToLower(n))
	}
	return names
}

func decodeTUISession(data []byte) (tuiSession, error) {
	var header struct {
		Version int `yaml:"version"`
	}
	if err := yaml.Load(data, &header); err != nil {
		return tuiSession{}, err
	}
	if header.Version == 1 {
		var s tuiSession
		err := yaml.Load(data, &s, yaml.WithKnownFields())
		return s, err
	}
	if header.Version != 0 && header.Version != 2 {
		return tuiSession{}, fmt.Errorf("unsupported session version %d", header.Version)
	}
	var saved tuiSavedSession
	if err := yaml.Load(data, &saved, yaml.WithKnownFields()); err != nil {
		return tuiSession{}, err
	}
	if saved.Layout == "" {
		saved.Layout = "columns"
	}
	if saved.Output == "" {
		saved.Output = "yaml"
	}
	layout, err := tuiChoice(saved.Layout, tuiLayoutNames())
	if err != nil {
		return tuiSession{}, err
	}
	output, err := tuiChoice(saved.Output, tuiOutputNames)
	if err != nil {
		return tuiSession{}, err
	}
	s := tuiSession{Version: 1, Layout: layout, Split: saved.Split, Live: saved.Live, Args: append([]string{tuiModes[output]}, saved.Args...)}
	for name := range saved.Options {
		if _, err := tuiChoice(name, tuiBooleans); err != nil {
			return s, err
		}
	}
	for _, name := range tuiBooleans {
		if value := saved.Options[name]; value != nil {
			arg := name
			if !*value {
				arg = "no-" + name
			}
			s.Args = append(s.Args, "-o", arg)
		}
	}
	return s, nil
}

func encodeTUISession(s tuiSession) ([]byte, error) {
	if s.Layout < 0 || s.Layout >= len(tuiLayouts) {
		return nil, fmt.Errorf("invalid layout %d", s.Layout)
	}
	saved := tuiSavedSession{Version: 2, Layout: tuiLayoutNames()[s.Layout], Output: "yaml", Live: s.Live, Split: s.Split, Options: map[string]*bool{}}
	for _, name := range tuiBooleans {
		saved.Options[name] = nil
	}
	for i := 0; i < len(s.Args); i++ {
		arg := s.Args[i]
		mode := -1
		for j, flag := range tuiModes {
			if arg == flag || arg == []string{"--yaml", "--YAML", "--json", "--JSON", "--token", "--TOKEN", "--event", "--EVENT", "--node", "--NODE"}[j] {
				mode = j
			}
		}
		if mode >= 0 {
			saved.Output = tuiOutputNames[mode]
			continue
		}
		if (arg == "-o" || arg == "--option") && i+1 < len(s.Args) {
			i++
			var rest []string
			for _, opt := range strings.Split(s.Args[i], ",") {
				name := strings.TrimSpace(opt)
				if name == "v2" || name == "v3" || name == "v4" {
					for key := range saved.Options {
						saved.Options[key] = nil
					}
				}
				value := true
				if strings.HasPrefix(name, "no-") {
					name = strings.TrimPrefix(name, "no-")
					value = false
				}
				if key, v, ok := strings.Cut(name, "="); ok && (v == "true" || v == "false") {
					name = key
					value = v == "true"
				}
				aliases := map[string]string{"compact": "compact-seq-indent", "unique": "unique-keys", "single": "single-document", "all": "all-documents", "stream": "stream-nodes"}
				if canonical, ok := aliases[name]; ok {
					name = canonical
				}
				if name == "explicit" {
					a, b := value, value
					saved.Options["explicit-start"] = &a
					saved.Options["explicit-end"] = &b
					continue
				}
				if _, ok := saved.Options[name]; ok {
					v := value
					saved.Options[name] = &v
				} else {
					rest = append(rest, opt)
				}
			}
			if len(rest) > 0 {
				saved.Args = append(saved.Args, "-o", strings.Join(rest, ","))
			}
			continue
		}
		saved.Args = append(saved.Args, arg)
	}
	// Render the editable settings in a stable, self-documenting order.
	options := saved.Options
	var out strings.Builder
	out.WriteString("### go-yaml session configuration\n#\n" +
		"# Boolean values:\n" +
		"#   true  → enabled   `+`\n" +
		"#   false → disabled  `-`\n" +
		"#   blank → default\n###\n\n")
	fmt.Fprintf(&out, "live: %t\n\nlayout: %s\n", saved.Live, saved.Layout)
	out.WriteString("# Layout choices:\n")
	for _, name := range tuiLayoutNames() {
		fmt.Fprintf(&out, "#   %s\n", name)
	}
	fmt.Fprintf(&out, "\noutput: %s\n# Output choices:\n", saved.Output)
	for i, name := range tuiOutputNames {
		fmt.Fprintf(&out, "#   %-18s (%s)\n", name, tuiModes[i])
	}
	if saved.Split != 0 || len(saved.Args) != 0 {
		extra := struct {
			Split float64  `yaml:"split,omitempty"`
			Args  []string `yaml:"args,omitempty"`
		}{saved.Split, saved.Args}
		data, err := yaml.Dump(extra)
		if err != nil {
			return nil, err
		}
		out.WriteByte('\n')
		out.Write(data)
	}
	out.WriteString("\noptions:\n")
	for _, name := range tuiBooleans {
		fmt.Fprintf(&out, "  %s:", name)
		if v := options[name]; v != nil {
			fmt.Fprintf(&out, " %t", *v)
		}
		out.WriteByte('\n')
	}
	return []byte(out.String()), nil
}
