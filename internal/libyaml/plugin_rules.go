// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// RuleInput is the read-only snapshot used by tag rules. PairKey and PairValue
// expose both halves of a mapping entry. Path contains keys or sequence indices.
type RuleInput struct {
	Node               *Node
	Role               string
	Path               []any
	PairKey, PairValue *Node
}

// RuleDecision assigns either one tag or both tags of a mapping entry.
// Empty tags leave a node unchanged.
type RuleDecision struct{ Tag, KeyTag, ValueTag string }

// YAMLRulesPlugin resolves tags and supplies the base policy for constructor
// payloads. Methods must not mutate input nodes or execute constructors.
type YAMLRulesPlugin interface {
	BaseTag(*Node) (string, error)
	ResolveYAMLRule(RuleInput) (RuleDecision, error)
}

func payloadBaseTag(n *Node, rules YAMLRulesPlugin) (string, error) {
	if n.baseRuleTag != "" {
		return n.baseRuleTag, nil
	}
	if rules != nil {
		return rules.BaseTag(n)
	}
	switch n.Kind {
	case MappingNode:
		return mapTag, nil
	case SequenceNode:
		return seqTag, nil
	case ScalarNode:
		if n.Style&(SingleQuotedStyle|DoubleQuotedStyle|LiteralStyle|FoldedStyle) != 0 {
			return strTag, nil
		}
		tag, _ := resolve("", n.Value, nil)
		return tag, nil
	}
	return "", errors.New("yaml: function tag requires a scalar or collection")
}

// ResolveRuleTree applies decisions against one stable base-resolved snapshot.
func ResolveRuleTree(root *Node, plugin YAMLRulesPlugin) error {
	originals := map[*Node]*Node{}
	var snapshot func(*Node) (*Node, error)
	snapshot = func(n *Node) (*Node, error) {
		if n == nil {
			return nil, nil
		}
		if copyNode, ok := originals[n]; ok {
			return copyNode, nil
		}
		copyNode := *n
		copyNode.Content = nil
		originals[n] = &copyNode
		if n.Kind != DocumentNode && n.Kind != AliasNode {
			tag, err := plugin.BaseTag(n)
			if err != nil {
				return nil, err
			}
			copyNode.baseRuleTag = tag
			if n.Tag == "" || n.Tag == "!" {
				copyNode.Tag = tag
			}
		}
		for _, child := range n.Content {
			childCopy, err := snapshot(child)
			if err != nil {
				return nil, err
			}
			copyNode.Content = append(copyNode.Content, childCopy)
		}
		if n.Alias != nil {
			alias, err := snapshot(n.Alias)
			if err != nil {
				return nil, err
			}
			copyNode.Alias = alias
		}
		return &copyNode, nil
	}
	tree, err := snapshot(root)
	if err != nil {
		return err
	}
	visited := map[*Node]bool{}
	var visit func(*Node, *Node, string, []any, *Node, *Node, string) error
	visit = func(n, snap *Node, role string, path []any, key, value *Node, forced string) error {
		if visited[n] {
			return nil
		}
		visited[n] = true
		n.baseRuleTag = snap.baseRuleTag
		if named, ok := plugin.(interface{ Base() string }); ok {
			n.ruleSet = named.Base()
		}
		if n.Kind == AliasNode {
			return nil
		}
		if n.Kind == DocumentNode {
			for i, child := range n.Content {
				if err := visit(child, snap.Content[i], "root", nil, nil, nil, ""); err != nil {
					return err
				}
			}
			return nil
		}
		decision, err := plugin.ResolveYAMLRule(RuleInput{snap, role, path, key, value})
		if err != nil {
			return err
		}
		explicit := n.Tag != "" && n.Tag != "!" && n.Style&TaggedStyle != 0
		if !explicit {
			n.Tag = snap.Tag
			if decision.Tag != "" {
				n.Tag = decision.Tag
			}
			if forced != "" {
				n.Tag = forced
			}
		}
		switch n.Kind {
		case MappingNode:
			for i := 0; i < len(n.Content); i += 2 {
				k, v := snap.Content[i], snap.Content[i+1]
				p := append(append([]any(nil), path...), k.Value)
				pair, err := plugin.ResolveYAMLRule(RuleInput{snap, "pair", p, k, v})
				if err != nil {
					return err
				}
				if err := visit(n.Content[i], k, "key", p, k, v, pair.KeyTag); err != nil {
					return err
				}
				if err := visit(n.Content[i+1], v, "value", p, k, v, pair.ValueTag); err != nil {
					return err
				}
			}
		case SequenceNode:
			for i, child := range n.Content {
				p := append(append([]any(nil), path...), i)
				if err := visit(child, snap.Content[i], "item", p, nil, nil, ""); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visit(root, tree, "root", nil, nil, nil, "")
}

func (d *Desolver) desolveRules(root *Node) {
	// Predict tags with the same structural and pair context used on load.
	var candidate func(*Node) *Node
	candidate = func(n *Node) *Node {
		copyNode := *n
		copyNode.Tag = ""
		copyNode.Style &^= TaggedStyle
		copyNode.Content = nil
		if n.Kind == ScalarNode {
			copyNode.Style = 0
		}
		for _, child := range n.Content {
			copyNode.Content = append(copyNode.Content, candidate(child))
		}
		return &copyNode
	}
	inferred := candidate(root)
	err := ResolveRuleTree(inferred, d.opts.YAMLRules)
	var visit func(*Node, *Node)
	visit = func(n, predicted *Node) {
		if n.Kind == DocumentNode {
			n.Tag = ""
		} else if n.Style&TaggedStyle == 0 {
			same := err == nil && shortTag(n.Tag) == shortTag(predicted.Tag)
			if same {
				n.Tag = ""
			} else if shortTag(n.Tag) == strTag {
				n.Tag = strTag
				n.Style |= TaggedStyle
				if n.Style&(SingleQuotedStyle|DoubleQuotedStyle|LiteralStyle|FoldedStyle) == 0 {
					n.Style |= SingleQuotedStyle
				}
			} else {
				n.Style |= TaggedStyle
			}
		}
		for i, child := range n.Content {
			visit(child, predicted.Content[i])
		}
	}
	visit(root, inferred)
}

// RuleError adds source context to rule failures.
func RuleError(err error, n *Node, formatter func(*LoadError) string) *LoadError {
	return &LoadError{Stage: ResolverStage, Mark: Mark{Line: n.Line, Column: n.Column}, Message: fmt.Sprintf("YAML rules: %s", err), err: err, formatter: formatter}
}

func standardRuleTag(tag string) bool { return strings.HasPrefix(shortTag(tag), "!!") }

// resolveRuleScalar preserves a resolved tag while interpreting that rule
// set's scalar spelling. Native destination conversion remains in Constructor.
func resolveRuleScalar(n *Node, formatter func(*LoadError) string) (string, any) {
	tag := shortTag(n.Tag)
	text := strings.ReplaceAll(n.Value, "_", "")
	switch tag {
	case intTag:
		base := 10
		if strings.HasPrefix(n.ruleSet, "go-yaml-") || n.ruleSet == "yaml-1.1-core" {
			base = 0
		}
		sign := ""
		digits := text
		if strings.HasPrefix(digits, "-") || strings.HasPrefix(digits, "+") {
			sign = digits[:1]
			digits = digits[1:]
		}
		if n.ruleSet == "yaml-1.2-core" && len(digits) > 2 {
			switch digits[:2] {
			case "0x":
				base = 16
				digits = digits[2:]
			case "0o":
				base = 8
				digits = digits[2:]
			}
			text = sign + digits
		}
		if n.ruleSet == "yaml-1.1-core" && strings.Contains(digits, ":") {
			total := new(big.Int)
			for _, part := range strings.Split(digits, ":") {
				piece, ok := new(big.Int).SetString(part, 10)
				if !ok {
					Fail((&Constructor{formatError: formatter}).formatConstructorError(fmt.Errorf("invalid integer %q", n.Value), Mark{Line: n.Line, Column: n.Column}))
				}
				total.Mul(total, big.NewInt(60))
				total.Add(total, piece)
			}
			if sign == "-" {
				total.Neg(total)
			}
			text = total.String()
			base = 10
		}
		if value, err := strconv.ParseInt(text, base, 64); err == nil {
			if int64(int(value)) == value {
				return tag, int(value)
			}
			return tag, value
		}
		if value, err := strconv.ParseUint(text, base, 64); err == nil {
			return tag, value
		}
		Fail((&Constructor{formatError: formatter}).formatConstructorError(fmt.Errorf("invalid or overflowing integer %q", n.Value), Mark{Line: n.Line, Column: n.Column}))
	case floatTag:
		if n.ruleSet == "yaml-1.1-core" && strings.Contains(text, ":") {
			negative := strings.HasPrefix(text, "-")
			text = strings.TrimLeft(text, "+-")
			total := float64(0)
			for _, part := range strings.Split(text, ":") {
				piece, err := strconv.ParseFloat(part, 64)
				if err != nil {
					Fail((&Constructor{formatError: formatter}).formatConstructorError(err, Mark{Line: n.Line, Column: n.Column}))
				}
				total = total*60 + piece
			}
			if negative {
				total = -total
			}
			return tag, total
		}
	case boolTag:
		if n.ruleSet == "go-yaml-v2" || n.ruleSet == "yaml-1.1-core" {
			switch n.Value {
			case "yes", "Yes", "YES", "on", "On", "ON", "y", "Y":
				return tag, true
			case "no", "No", "NO", "off", "Off", "OFF", "n", "N":
				return tag, false
			}
		}
	}
	return resolve(n.Tag, n.Value, formatter)
}

// InferGoScalarTag reports the existing native resolver's implicit tag.
// It lets named compatibility rules reuse the exact current lexical policy.
func InferGoScalarTag(value string) string {
	tag, _ := resolve("", value, nil)
	return tag
}
