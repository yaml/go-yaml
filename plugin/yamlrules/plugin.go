// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package yamlrules implements declarative, structural YAML tag resolution.
package yamlrules

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4/internal/libyaml"
)

// Names returns the stable built-in rule-set identifiers.
func Names() []string {
	return []string{"yamlstar", "go-yaml-v2", "go-yaml-v3", "go-yaml-v4", "yaml-1.1-core", "yaml-1.2-core", "yaml-1.2-json", "failsafe", "json"}
}

// Plugin is a compiled immutable rule definition. Calls share no mutable state.
type Plugin struct {
	base       string
	definition map[string]any
}

var (
	concrete    = map[string]string{"Str": "!!str", "Int": "!!int", "Float": "!!float", "Bool": "!!bool", "Null": "!!null", "Map": "!!map", "Seq": "!!seq"}
	expressions = regexp.MustCompile(`^\+([A-Za-z][A-Za-z0-9/-]*)(\[\])?(.*)$`)
	clause      = regexp.MustCompile(`^~"(.*)"$`)
)

// New compiles a rule definition over one named base. Definitions are copied.
func New(name string, definition map[string]any) (*Plugin, error) {
	if base, ok := definition[".base"].(string); ok {
		name = base
	}
	known := false
	for _, candidate := range Names() {
		if candidate == name {
			known = true
		}
	}
	if !known {
		return nil, fmt.Errorf("yaml-rules: unknown rule set %q", name)
	}
	for key := range definition {
		switch key {
		case ".base", ".rules", ".root", ".schema", ".format":
		default:
			return nil, fmt.Errorf("yaml-rules: unknown definition directive %q", key)
		}
	}
	copied := copyData(definition).(map[string]any)
	p := &Plugin{base: name, definition: copied}
	if err := validateRules(copied[".rules"]); err != nil {
		return nil, err
	}
	if err := validateShape(copied[".root"]); err != nil {
		return nil, err
	}
	return p, nil
}

// Base identifies the selected immutable base rules.
func (p *Plugin) Base() string { return p.base }

func copyData(value any) any {
	switch v := value.(type) {
	case map[string]any:
		r := map[string]any{}
		for k, x := range v {
			r[k] = copyData(x)
		}
		return r
	case []any:
		r := make([]any, len(v))
		for i, x := range v {
			r[i] = copyData(x)
		}
		return r
	default:
		return value
	}
}

func regex(text string) (*regexp.Regexp, error) { return regexp.Compile(`\A(?:` + text + `)\z`) }
func matchText(pattern, text string) bool {
	r, err := regex(pattern)
	return err == nil && r.MatchString(text)
}

// Infer resolves unquoted scalar text, independently of native Go conversion.
func Infer(base, value string) (string, error) {
	goSet := strings.HasPrefix(base, "go-yaml-")
	if goSet {
		if base == "go-yaml-v2" && matchText(`yes|Yes|YES|no|No|NO|on|On|ON|off|Off|OFF|y|Y|n|N`, value) {
			return "!!bool", nil
		}
		return libyaml.InferGoScalarTag(value), nil
	}
	old := base == "yaml-1.1-core"
	json := base == "json" || base == "yaml-1.2-json"
	if base == "failsafe" {
		return "!!str", nil
	}
	nulls := "null|Null|NULL|~|"
	if json {
		nulls = "null"
	}
	if matchText(nulls, value) {
		return "!!null", nil
	}
	bools := "true|True|TRUE|false|False|FALSE"
	if old || base == "go-yaml-v2" {
		bools += "|yes|Yes|YES|no|No|NO|on|On|ON|off|Off|OFF|y|Y|n|N"
	}
	if json {
		bools = "true|false"
	}
	if matchText(bools, value) {
		return "!!bool", nil
	}
	ints := `[-+]?[0-9]+`
	switch {
	case json:
		ints = `-?(0|[1-9][0-9]*)`
	case old:
		ints = `[-+]?(0|[1-9][0-9_]*|0[0-7_]+|0x[0-9a-fA-F_]+|0b[01_]+|[1-9][0-9_]*(:[0-5]?[0-9])+)`
	case goSet:
		ints = `[-+]?(0b[01_]+|0o[0-7_]+|0x[0-9a-fA-F_]+|0[0-7_]+|[1-9][0-9_]*|0)`
	case base == "yaml-1.2-core":
		ints = `[-+]?[0-9]+|0o[0-7]+|0x[0-9a-fA-F]+`
	}
	if matchText(ints, value) {
		return "!!int", nil
	}
	floats := `[-+]?(\.[0-9]+|[0-9]+(\.[0-9]*)?)([eE][-+]?[0-9]+)?`
	if json {
		floats = `-?(0|[1-9][0-9]*)(\.[0-9]*)?([eE][-+]?[0-9]+)?`
	}
	if base == "json" {
		floats = `-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][-+]?[0-9]+)?`
	}
	text := value
	if old || goSet {
		text = strings.ReplaceAll(text, "_", "")
	}
	if (old && matchText(`[-+]?[1-9][0-9_]*(:[0-5]?[0-9])+\.[0-9_]*`, value)) || matchText(floats, text) || (!json && matchText(`[-+]?(\.inf|\.Inf|\.INF)|\.nan|\.NaN|\.NAN`, value)) {
		return "!!float", nil
	}
	if (old || goSet) && matchText(`[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}([Tt ].*)?`, value) {
		return "!!timestamp", nil
	}
	if (old || goSet || base == "yamlstar") && value == "<<" {
		return "!!merge", nil
	}
	if json {
		return "", fmt.Errorf("yaml-rules: scalar %q does not match %s", value, base)
	}
	return "!!str", nil
}

// BaseTag implements yaml.YAMLRulesPlugin.
func (p *Plugin) BaseTag(n *libyaml.Node) (string, error) {
	switch n.Kind {
	case libyaml.MappingNode:
		return "!!map", nil
	case libyaml.SequenceNode:
		return "!!seq", nil
	case libyaml.ScalarNode:
		if n.Tag == "!" || n.Style&(libyaml.SingleQuotedStyle|libyaml.DoubleQuotedStyle|libyaml.LiteralStyle|libyaml.FoldedStyle) != 0 {
			return "!!str", nil
		}
		if strings.HasPrefix(short(n.Tag), "!!") {
			return short(n.Tag), nil
		}
		return Infer(p.base, n.Value)
	}
	return "", nil
}

func short(tag string) string { return strings.Replace(tag, "tag:yaml.org,2002:", "!!", 1) }
func compact(text string) (map[string]any, error) {
	parts := expressions.FindStringSubmatch(text)
	if parts == nil {
		return nil, fmt.Errorf("yaml-rules: invalid compact expression %q", text)
	}
	typ := parts[1]
	if typ != "Any" && typ != "Num" && concrete[typ] == "" {
		return nil, fmt.Errorf("yaml-rules: unknown match type +%s", typ)
	}
	result := map[string]any{".type": "+" + typ}
	if parts[2] != "" {
		result = map[string]any{".list": result}
	}
	tail := strings.TrimSpace(parts[3])
	if tail != "" {
		pattern := clause.FindStringSubmatch(tail)
		if pattern == nil {
			return nil, fmt.Errorf("yaml-rules: unsupported compact clause %q", tail)
		}
		if parts[2] != "" {
			result[".list"].(map[string]any)[".match"] = pattern[1]
		} else {
			result[".match"] = pattern[1]
		}
	}
	return result, nil
}

func matchSpec(value any) (map[string]any, error) {
	if text, ok := value.(string); ok {
		return compact(text)
	}
	m, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("yaml-rules: match must be a mapping or compact expression")
	}
	return m, nil
}

func rulesList(value any) []any {
	if value == nil {
		return nil
	}
	if list, ok := value.([]any); ok {
		return list
	}
	return []any{value}
}

func validateMatch(value any) error {
	spec, err := matchSpec(value)
	if err != nil {
		return err
	}
	for key, v := range spec {
		switch key {
		case ".type":
			text, ok := v.(string)
			if !ok {
				return errors.New("yaml-rules: .type must be a compact type name")
			}
			if _, err := compact(text); err != nil {
				return err
			}
		case ".match":
			text, ok := v.(string)
			if !ok {
				return errors.New("yaml-rules: .match must be text")
			}
			if _, err := regex(text); err != nil {
				return err
			}
		case ".const":
		case ".open":
			if _, ok := v.(bool); !ok {
				return errors.New("yaml-rules: .open must be boolean")
			}
		case ".size":
			bounds, ok := v.([]any)
			if !ok {
				bounds = []any{v}
			}
			if len(bounds) < 1 || len(bounds) > 2 {
				return errors.New("yaml-rules: invalid .size bounds")
			}
			for _, bound := range bounds {
				if bound == nil {
					continue
				}
				parsed, err := strconv.Atoi(fmt.Sprint(bound))
				if err != nil || parsed < 0 {
					return errors.New("yaml-rules: .size requires nonnegative integer bounds")
				}
			}

		case ".list", ".not":
			if err := validateMatch(v); err != nil {
				return err
			}
		case ".any", ".all", ".one":
			list, ok := v.([]any)
			if !ok {
				return errors.New("yaml-rules: alternatives must be a sequence")
			}
			for _, x := range list {
				if err := validateMatch(x); err != nil {
					return err
				}
			}
		default:
			if strings.HasPrefix(key, ".") {
				return fmt.Errorf("yaml-rules: unsupported match directive %s", key)
			}
			if err := validateMatch(v); err != nil {
				return err
			}
		}
	}
	return nil
}

func action(value any) (string, error) {
	if text, ok := value.(string); ok {
		if strings.HasPrefix(text, "+") {
			tag := concrete[text[1:]]
			if tag == "" {
				return "", fmt.Errorf("yaml-rules: action %s requires a concrete tag", text)
			}
			return tag, nil
		}
		if !strings.HasPrefix(text, "!") && !strings.Contains(text, ":") {
			return "", errors.New("yaml-rules: tag must be a YAML tag")
		}
		return text, nil
	}
	if m, ok := value.(map[string]any); ok {
		if v, ok := m[".tag"]; ok {
			return action(v)
		}
	}
	return "", nil
}

func validateRules(value any) error {
	for _, r := range rulesList(value) {
		if _, ok := r.(string); ok {
			if _, err := action(r); err != nil {
				return err
			}
			continue
		}
		rule, ok := r.(map[string]any)
		if !ok {
			return errors.New("yaml-rules: rule must be a mapping or tag expression")
		}
		for key, v := range rule {
			switch key {
			case ".role":
				role, ok := v.(string)
				if !ok || (role != "root" && role != "key" && role != "value" && role != "item" && role != "pair") {
					return errors.New("yaml-rules: invalid node role")
				}
			case ".when":
				if rule[".role"] == "pair" {
					pair, ok := v.(map[string]any)
					if !ok {
						return errors.New("yaml-rules: pair match requires .key/.value")
					}
					for k, pattern := range pair {
						if k != ".key" && k != ".value" {
							return errors.New("yaml-rules: pair match requires .key/.value")
						}
						if err := validateMatch(pattern); err != nil {
							return err
						}
					}
				} else if err := validateMatch(v); err != nil {
					return err
				}
			case ".tag":
				if _, err := action(v); err != nil {
					return err
				}
			case ".tags":
				tags, ok := v.(map[string]any)
				if !ok {
					return errors.New("yaml-rules: .tags must be a mapping")
				}
				for k, tag := range tags {
					if k != "key" && k != "value" {
						return errors.New("yaml-rules: .tags accepts key/value only")
					}
					if _, err := action(tag); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("yaml-rules: unknown rule directive %s", key)
			}
		}
	}
	return nil
}

func validateShape(value any) error {
	if value == nil {
		return nil
	}
	if _, ok := value.(string); ok {
		_, err := action(value)
		return err
	}
	shape, ok := value.(map[string]any)
	if !ok {
		return errors.New("yaml-rules: structural rule must be a mapping")
	}
	for key, v := range shape {
		switch key {
		case ".rules":
			if err := validateRules(v); err != nil {
				return err
			}
		case ".pairs":
			for _, entry := range rulesList(v) {
				rule, ok := entry.(map[string]any)
				if !ok {
					return errors.New("yaml-rules: pair rule must be a mapping")
				}
				copyRule := copyData(rule).(map[string]any)
				copyRule[".role"] = "pair"
				if err := validateRules(copyRule); err != nil {
					return err
				}
			}
		case ".schema", ".format": // Reserved action sections, not executed.
		case ".list":
			if err := validateShape(v); err != nil {
				return err
			}
		default:
			if strings.HasPrefix(key, ".") {
				return fmt.Errorf("yaml-rules: unknown structural directive %s", key)
			}
			if err := validateShape(v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Matches evaluates the supported YAMLSchema-style predicate subset.
func Matches(node *libyaml.Node, value any) bool {
	return matches(node, value, map[*libyaml.Node]bool{})
}

func matches(n *libyaml.Node, value any, active map[*libyaml.Node]bool) bool {
	if n == nil {
		return false
	}
	if n.Kind == libyaml.AliasNode {
		if active[n] {
			return false
		}
		active[n] = true
		defer delete(active, n)
		return matches(n.Alias, value, active)
	}
	spec, err := matchSpec(value)
	if err != nil {
		return false
	}
	if typ, ok := spec[".type"].(string); ok {
		tag := short(n.Tag)
		typ = strings.TrimPrefix(typ, "+")
		if typ != "Any" && !(typ == "Num" && (tag == "!!int" || tag == "!!float")) && concrete[typ] != tag {
			return false
		}
	}
	if pattern, ok := spec[".match"].(string); ok {
		if n.Kind != libyaml.ScalarNode || !matchText(pattern, n.Value) {
			return false
		}
	}
	if constant, exists := spec[".const"]; exists {
		if n.Kind != libyaml.ScalarNode || (constant == nil && short(n.Tag) != "!!null") || (constant != nil && fmt.Sprint(constant) != n.Value) {
			return false
		}
	}
	if size, exists := spec[".size"]; exists {
		count := len(n.Content)
		if n.Kind == libyaml.MappingNode {
			count /= 2
		}
		if n.Kind == libyaml.ScalarNode {
			count = len([]rune(n.Value))
		}
		low, high := -1, -1
		if bounds, ok := size.([]any); ok {
			if len(bounds) > 0 && bounds[0] != nil {
				low, _ = strconv.Atoi(fmt.Sprint(bounds[0]))
			}
			if len(bounds) > 1 && bounds[1] != nil {
				high, _ = strconv.Atoi(fmt.Sprint(bounds[1]))
			}
		} else {
			low, _ = strconv.Atoi(fmt.Sprint(size))
			high = low
		}
		if (low >= 0 && count < low) || (high >= 0 && count > high) {
			return false
		}
	}
	if item, exists := spec[".list"]; exists {
		if n.Kind != libyaml.SequenceNode {
			return false
		}
		for _, child := range n.Content {
			if !matches(child, item, active) {
				return false
			}
		}
	}
	propertyKeys := map[string]bool{}
	for key := range spec {
		if !strings.HasPrefix(key, ".") {
			propertyKeys[strings.TrimSuffix(key, "?")] = true
		}
	}
	for key, pattern := range spec {
		if strings.HasPrefix(key, ".") {
			continue
		}
		if n.Kind != libyaml.MappingNode {
			return false
		}
		optional := strings.HasSuffix(key, "?")
		key = strings.TrimSuffix(key, "?")
		propertyKeys[key] = true
		found := false
		for i := 0; i < len(n.Content); i += 2 {
			if (key == "+Str" && !propertyKeys[n.Content[i].Value]) || n.Content[i].Value == key {
				found = true
				if !matches(n.Content[i+1], pattern, active) {
					return false
				}
			}
		}
		if !found && !optional && key != "+Str" {
			return false
		}
	}
	if spec[".open"] == false && n.Kind == libyaml.MappingNode && !propertyKeys["+Str"] {
		for i := 0; i < len(n.Content); i += 2 {
			if !propertyKeys[n.Content[i].Value] {
				return false
			}
		}
	}
	for _, key := range []string{".all", ".any", ".one"} {
		if alternatives, ok := spec[key].([]any); ok {
			count := 0
			for _, pattern := range alternatives {
				if matches(n, pattern, active) {
					count++
				}
			}
			if (key == ".all" && count != len(alternatives)) || (key == ".any" && count == 0) || (key == ".one" && count != 1) {
				return false
			}
		}
	}
	if pattern, ok := spec[".not"]; ok && matches(n, pattern, active) {
		return false
	}
	return true
}

func shapeMap(value any) map[string]any {
	if text, ok := value.(string); ok {
		return map[string]any{".rules": text}
	}
	m, _ := value.(map[string]any)
	return m
}

func (p *Plugin) shape(input libyaml.RuleInput) map[string]any {
	shape := shapeMap(p.definition[".root"])
	limit := len(input.Path)
	if input.Role == "pair" {
		limit--
	}
	for _, part := range input.Path[:maxInt(0, limit)] {
		switch v := part.(type) {
		case int:
			shape = shapeMap(shape[".list"])
		case string:
			next, exists := shape[v]
			if !exists {
				next = shape["+Str"]
			}
			shape = shapeMap(next)
		}
	}
	if input.Role == "key" {
		return nil
	}
	return shape
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func choose(list any, input libyaml.RuleInput) (any, bool) {
	for _, rule := range rulesList(list) {
		if _, ok := rule.(string); ok {
			if input.Role != "pair" {
				return rule, true
			}
			continue
		}
		r := rule.(map[string]any)
		if role, ok := r[".role"].(string); ok {
			if role != input.Role {
				continue
			}
		} else if input.Role == "pair" {
			continue
		}
		if predicate, ok := r[".when"]; ok {
			if input.Role == "pair" {
				when := predicate.(map[string]any)
				if k, ok := when[".key"]; ok && !Matches(input.PairKey, k) {
					continue
				}
				if v, ok := when[".value"]; ok && !Matches(input.PairValue, v) {
					continue
				}
			} else if !Matches(input.Node, predicate) {
				continue
			}
		}
		return rule, true
	}
	return nil, false
}

// ResolveYAMLRule implements yaml.YAMLRulesPlugin.
func (p *Plugin) ResolveYAMLRule(input libyaml.RuleInput) (libyaml.RuleDecision, error) {
	shape := p.shape(input)
	if input.Role == "pair" {
		list := []any{}
		for _, rule := range rulesList(shape[".pairs"]) {
			copyRule := copyData(rule).(map[string]any)
			copyRule[".role"] = "pair"
			list = append(list, copyRule)
		}
		list = append(list, rulesList(p.definition[".rules"])...)
		if rule, ok := choose(list, input); ok {
			tags, _ := rule.(map[string]any)[".tags"].(map[string]any)
			k, _ := action(tags["key"])
			v, _ := action(tags["value"])
			return libyaml.RuleDecision{KeyTag: k, ValueTag: v}, nil
		}
		return libyaml.RuleDecision{}, nil
	}
	if rule, ok := choose(shape[".rules"], input); ok {
		tag, err := action(rule)
		return libyaml.RuleDecision{Tag: tag}, err
	}
	if rule, ok := choose(p.definition[".rules"], input); ok {
		tag, err := action(rule)
		return libyaml.RuleDecision{Tag: tag}, err
	}
	return libyaml.RuleDecision{}, nil
}
