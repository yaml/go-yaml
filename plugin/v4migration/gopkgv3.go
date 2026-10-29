// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package v4migration

import (
	"fmt"

	"go.yaml.in/yaml/v4"
	legacy "gopkg.in/yaml.v3"
)

type fromGopkgV3 struct{}

// FromGopkgV3 adapts hooks accepting *gopkg.in/yaml.v3.Node.
// Node trees are copied for each invocation; mutations do not affect v4 nodes.
func FromGopkgV3() yaml.V4MigrationPlugin {
	return fromGopkgV3{}
}

func (fromGopkgV3) TryUnmarshalYAML(target any, node *yaml.Node) (bool, error) {
	hook, ok := target.(interface {
		UnmarshalYAML(*legacy.Node) error
	})
	if !ok {
		return false, nil
	}
	n, err := convertGopkgV3(node, make(map[*yaml.Node]*legacy.Node))
	if err != nil {
		return true, err
	}
	return true, hook.UnmarshalYAML(n)
}

func convertGopkgV3(src *yaml.Node, seen map[*yaml.Node]*legacy.Node) (*legacy.Node, error) {
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
			"v4-migration: cannot convert node kind %d to gopkg.in/yaml.v3.Node",
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
	if dst.Alias, err = convertGopkgV3(src.Alias, seen); err != nil {
		return nil, err
	}
	if src.Content != nil {
		dst.Content = make([]*legacy.Node, len(src.Content))
		for i, child := range src.Content {
			dst.Content[i], err = convertGopkgV3(child, seen)
			if err != nil {
				return nil, err
			}
		}
	}
	return dst, nil
}
