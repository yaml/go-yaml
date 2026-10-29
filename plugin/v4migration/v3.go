// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package v4migration

import (
	"fmt"

	legacy "go.yaml.in/yaml/v3"
	"go.yaml.in/yaml/v4"
)

type fromV3 struct{}

// FromV3 adapts hooks accepting *go.yaml.in/yaml/v3.Node.
// Node trees are copied for each invocation; mutations do not affect v4 nodes.
func FromV3() yaml.V4MigrationPlugin {
	return fromV3{}
}

func (fromV3) TryUnmarshalYAML(target any, node *yaml.Node) (bool, error) {
	hook, ok := target.(interface {
		UnmarshalYAML(*legacy.Node) error
	})
	if !ok {
		return false, nil
	}
	n, err := convertV3(node, make(map[*yaml.Node]*legacy.Node))
	if err != nil {
		return true, err
	}
	return true, hook.UnmarshalYAML(n)
}

func convertV3(src *yaml.Node, seen map[*yaml.Node]*legacy.Node) (*legacy.Node, error) {
	if src == nil {
		return nil, nil
	}
	if dst, ok := seen[src]; ok {
		return dst, nil
	}
	switch src.Kind {
	case 0, yaml.DocumentNode, yaml.MappingNode, yaml.SequenceNode,
		yaml.ScalarNode, yaml.AliasNode:
	default:
		return nil, fmt.Errorf(
			"v4-migration: cannot convert node kind %d to go.yaml.in/yaml/v3.Node",
			src.Kind)
	}
	dst := &legacy.Node{
		Kind: legacy.Kind(src.Kind), Style: legacy.Style(src.Style),
		Tag: src.Tag, Value: src.Value, Anchor: src.Anchor,
		HeadComment: src.HeadComment, LineComment: src.LineComment,
		FootComment: src.FootComment, Line: src.Line, Column: src.Column,
	}
	seen[src] = dst
	var err error
	if dst.Alias, err = convertV3(src.Alias, seen); err != nil {
		return nil, err
	}
	if src.Content != nil {
		dst.Content = make([]*legacy.Node, len(src.Content))
		for i, child := range src.Content {
			dst.Content[i], err = convertV3(child, seen)
			if err != nil {
				return nil, err
			}
		}
	}
	return dst, nil
}
