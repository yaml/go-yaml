// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"errors"
	"fmt"
	"reflect"

	"go.yaml.in/yaml/v4/internal/plugin/mergekey"
)

// MergeKeyConfig configures merge-key recognition and conflict handling.
type MergeKeyConfig = mergekey.Config

// MergeKeyInput contains read-only effective mappings in precedence order.
// Target has no merge directives. Sources have already had aliases and nested
// merges expanded. Key constructs a comparable destination-aware key.
// Unconstructible keys are reported as *LoadErrors; conversion diagnostics,
// if any, are retained by the constructor. Algorithms can skip those entries.
// The input and callback are valid only for the duration of MergeMappings.
type MergeKeyInput struct {
	Target  *Node
	Sources []*Node
	Key     func(*Node) (any, error)
}

// MergeKeyPlugin chooses the effective mapping before value construction.
// Implementations must support independent concurrent calls, leave input
// graphs unchanged, and return a MappingNode with paired key/value children.
type MergeKeyPlugin interface {
	MergeKeyConfig() MergeKeyConfig
	MergeMappings(MergeKeyInput) (*Node, error)
}

// DefaultMergeKey implements the standard shallow merge algorithm.
// The algorithm itself lives in the internal plugin package.
type DefaultMergeKey struct {
	Config MergeKeyConfig
}

// MergeKeyConfig returns the configuration by value.
func (p DefaultMergeKey) MergeKeyConfig() MergeKeyConfig { return p.Config }

// MergeMappings adapts YAML nodes to the default plugin algorithm.
func (p DefaultMergeKey) MergeMappings(in MergeKeyInput) (*Node, error) {
	mappings := append([]*Node{in.Target}, in.Sources...)
	groups := make([][]mergekey.Entry[[]*Node], len(mappings))
	for group, mapping := range mappings {
		for i := 0; i < len(mapping.Content); i += 2 {
			key, err := in.Key(mapping.Content[i])
			if err != nil {
				var failures *LoadErrors
				if errors.As(err, &failures) {
					continue
				}
				return nil, err
			}
			groups[group] = append(groups[group], mergekey.Entry[[]*Node]{
				Key: key, Value: mapping.Content[i : i+2],
			})
		}
	}
	entries, err := mergekey.Merge(groups, p.Config.Conflicts)
	if err != nil {
		var conflict *mergekey.ConflictError
		if errors.As(err, &conflict) {
			n := groups[conflict.Group][conflict.Index].Value[0]
			return nil, NewLoadError(ConstructorStage, err.Error(),
				Mark{Line: n.Line, Column: n.Column}, err)
		}
		return nil, err
	}
	result := *in.Target
	result.Content = make([]*Node, 0, len(entries)*2)
	for _, entry := range entries {
		result.Content = append(result.Content, entry...)
	}
	return &result, nil
}

func (c *Constructor) mergeFailure(err error, n *Node) {
	var load *LoadError
	if errors.As(err, &load) {
		Fail(formatLoadError(load, c.formatError))
	}
	Fail(c.formatConstructorError(err, Mark{Line: n.Line, Column: n.Column}))
}

// Copied nodes retain provenance for accounting and their original identity
// for cycle detection, even when nested merges copy an existing copy.
func (c *Constructor) originalMergeNode(n *Node) *Node {
	if original := c.mergeAliases[n]; original != nil {
		return original
	}
	return n
}

// effectiveMapping resolves directives without constructing their values.
// Alias traversal uses the same accounting and cycle guard as construction.
func (c *Constructor) effectiveMapping(n *Node, keyType reflect.Type) *Node {
	if n.Kind == AliasNode {
		if n.Alias == nil || n.Alias.Kind != MappingNode {
			c.failWantMap(n)
		}
		identity := c.originalMergeNode(n)
		if c.aliases[identity] {
			c.mergeFailure(errors.New("merge-key: cyclic merge alias"), n)
		}
		c.aliases[identity] = true
		c.aliasDepth++
		defer func() {
			c.aliasDepth--
			delete(c.aliases, identity)
		}()
		c.checkConstruction(n)
		resolved := c.effectiveMapping(n.Alias, keyType)
		if resolved == nil {
			return nil
		}
		// Charge expansion before allocating or retaining copied entries.
		for _, child := range resolved.Content {
			c.checkConstruction(child)
		}
		// Keep provenance on private node copies so selected values retain
		// alias accounting without changing the algorithm's node interface.
		result := *resolved
		result.Content = make([]*Node, len(resolved.Content))
		if c.mergeAliases == nil {
			c.mergeAliases = make(map[*Node]*Node)
		}
		for i, child := range resolved.Content {
			copyChild := *child
			result.Content[i] = &copyChild
			c.mergeAliases[&copyChild] = c.originalMergeNode(child)
		}
		return &result
	}
	if n.Kind != MappingNode {
		c.failWantMap(n)
	}
	if !c.checkMappingKeys(n) {
		return nil
	}
	target := *n
	target.Content = make([]*Node, 0, len(n.Content))
	var directives []*Node
	for i := 0; i < len(n.Content); i += 2 {
		key := n.Content[i]
		if c.mergeConfig.Disable && isMerge(key) {
			if key.Style&TaggedStyle != 0 {
				c.mergeFailure(errors.New("merge-key: merge tag is disabled"), key)
			}
			copyKey := *key
			copyKey.Tag = strTag
			key = &copyKey
		} else if isMerge(key) {
			directives = append(directives, n.Content[i+1])
			continue
		}
		target.Content = append(target.Content, key, n.Content[i+1])
	}
	if len(directives) == 0 {
		return &target
	}
	if c.mergeConfig.Multiple != "combine" {
		directives = directives[len(directives)-1:]
	}
	var sources []*Node
	for _, directive := range directives {
		candidates := []*Node{directive}
		if directive.Kind == SequenceNode {
			candidates = directive.Content
		}
		for _, candidate := range candidates {
			if source := c.effectiveMapping(candidate, keyType); source != nil {
				sources = append(sources, source)
			}
		}
	}
	key := func(n *Node) (result any, err error) {
		defer handleErr(&err)
		// Preserve diagnostics for collection keys even in string-keyed maps.
		if n.Kind != ScalarNode && keyType.Kind() == reflect.String {
			probe := reflect.New(ifaceType).Elem()
			if c.Construct(n, probe) {
				value := probe.Interface()
				if value != nil && !reflect.TypeOf(value).Comparable() {
					c.failUnhashableKey(value, n)
				}
			}
		}
		v := reflect.New(keyType).Elem()
		nerrs := len(c.TypeErrors)
		if !c.Construct(n, v) {
			// Null keys can be skipped without a conversion diagnostic.
			return nil, &LoadErrors{Errors: c.TypeErrors[nerrs:]}
		}
		value := v.Interface()
		if value != nil && !reflect.TypeOf(value).Comparable() {
			c.failUnhashableKey(value, n)
		}
		c.checkMergeKey(value, n)
		return value, nil
	}
	result, err := c.mergePlugin.MergeMappings(MergeKeyInput{
		Target: &target, Sources: sources, Key: key,
	})
	if err != nil {
		c.mergeFailure(err, n)
	}
	if result == nil || result.Kind != MappingNode || len(result.Content)%2 != 0 {
		c.mergeFailure(errors.New("merge-key: algorithm must return a mapping"), n)
	}
	for _, child := range result.Content {
		if child == nil {
			c.mergeFailure(errors.New("merge-key: algorithm returned a nil child"), n)
		}
	}
	return result
}

// Hashing can also fail for comparable structs containing interface fields.
func (c *Constructor) checkMergeKey(value any, n *Node) {
	defer func() {
		if recover() != nil {
			c.failUnhashableKey(value, n)
		}
	}()
	_ = map[any]bool{}[value]
}

// checkMappingKeys reports keys that repeat an earlier key of the same
// mapping. It runs in linear time: keys are looked up in a hash set instead of
// being compared with every earlier key, and a key that repeats gets one error
// that names the first earlier occurrence, so a mapping with n identical keys
// produces n-1 errors rather than n*(n-1)/2.
func (c *Constructor) checkMappingKeys(n *Node) bool {
	if !c.UniqueKeys && c.mergeConfig.Multiple != "error" {
		return true
	}
	type keyID struct {
		kind  Kind
		value string
	}
	// For one key, the first earlier occurrence that is a merge key and the
	// first that is not. Two merge keys are checked under the merge rules,
	// any other pair under UniqueKeys.
	type seenKeys struct{ merge, plain *Node }
	seen := make(map[keyID]*seenKeys, len(n.Content)/2)
	nerrs := len(c.TypeErrors)
	for i := 0; i < len(n.Content); i += 2 {
		nk := n.Content[i]
		id := keyID{nk.Kind, nk.Value}
		e := seen[id]
		if e == nil {
			e = &seenKeys{}
			seen[id] = e
		}
		isMergeKey := !c.mergeConfig.Disable && isMerge(nk)
		var first *Node
		if isMergeKey && e.merge != nil &&
			c.mergeConfig.Multiple != "combine" &&
			(c.UniqueKeys || c.mergeConfig.Multiple == "error") {
			first = e.merge
		}
		if first == nil && c.UniqueKeys {
			first = e.plain
			if !isMergeKey && e.merge != nil && (first == nil || e.merge.Line < first.Line) {
				first = e.merge
			}
		}
		if first != nil {
			c.TypeErrors = append(c.TypeErrors, c.formatConstructorError(
				fmt.Errorf("mapping key %#v already defined at line %d", nk.Value, first.Line),
				Mark{Line: nk.Line, Column: nk.Column},
			))
		}
		if isMergeKey {
			if e.merge == nil {
				e.merge = nk
			}
		} else if e.plain == nil {
			e.plain = nk
		}
	}
	return len(c.TypeErrors) == nerrs
}
