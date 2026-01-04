// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// YAML node formatting utilities for the go-yaml tool.

package main

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

// TagDirectiveInfo represents a YAML %TAG directive
type TagDirectiveInfo struct {
	Handle string `yaml:"handle"`
	Prefix string `yaml:"prefix"`
}

// MapItem is a single item in a MapSlice.
type MapItem struct {
	Key   string
	Value any
}

// MapSlice is a slice of MapItems that preserves order when marshaled to YAML.
type MapSlice []MapItem

// MarshalYAML implements yaml.Marshaler for MapSlice to preserve key order.
func (ms MapSlice) MarshalYAML() (any, error) {
	// Convert MapSlice to yaml.Node with MappingNode kind to preserve order
	node := &yaml.Node{
		Kind: yaml.MappingNode,
	}
	for _, item := range ms {
		// Add key node
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: item.Key,
		}
		// Add value node - let yaml dumper handle the value
		// Dump item.Value directly into a yaml.Node
		valueNode := &yaml.Node{}
		if err := valueNode.Dump(item.Value); err != nil {
			return nil, err
		}
		node.Content = append(node.Content, keyNode, valueNode)
	}
	return node, nil
}

// NodeInfo represents the information about a YAML node
type NodeInfo struct {
	Kind          string             `yaml:"kind"`
	Style         string             `yaml:"style,omitempty"`
	Tag           string             `yaml:"tag,omitempty"`
	Anchor        string             `yaml:"anchor,omitempty"`
	Head          string             `yaml:"head,omitempty"`
	Line          string             `yaml:"line,omitempty"`
	Foot          string             `yaml:"foot,omitempty"`
	Text          string             `yaml:"text,omitempty"`
	Content       []*NodeInfo        `yaml:"content,omitempty"`
	Encoding      string             `yaml:"encoding,omitempty"`
	Version       string             `yaml:"version,omitempty"`
	TagDirectives []TagDirectiveInfo `yaml:"tag-directives,omitempty"`
}

// FormatNode converts a YAML node into a NodeInfo structure
func FormatNode(n yaml.Node, profuse bool) *NodeInfo {
	info := &NodeInfo{
		Kind: formatKind(n.Kind),
	}

	// Don't set style for Document nodes
	if n.Kind != yaml.DocumentNode {
		if style := formatStyle(n.Style, profuse); style != "" {
			info.Style = style
		}
	}
	if n.Anchor != "" {
		info.Anchor = n.Anchor
	}
	if tag := formatTag(n.Tag, n.Style, profuse); tag != "" {
		info.Tag = tag
	}
	if n.HeadComment != "" {
		info.Head = n.HeadComment
	}
	if n.LineComment != "" {
		info.Line = n.LineComment
	}
	if n.FootComment != "" {
		info.Foot = n.FootComment
	}

	if info.Kind == "Scalar" {
		info.Text = n.Value
	} else if n.Content != nil {
		info.Content = make([]*NodeInfo, len(n.Content))
		for i, node := range n.Content {
			info.Content[i] = FormatNode(*node, profuse)
		}
	}

	// Handle StreamNode-specific fields
	if info.Kind == "Stream" {
		if n.Encoding != 0 {
			info.Encoding = formatEncoding(n.Encoding)
		}
		if n.Version != nil {
			info.Version = formatVersion(n.Version)
		}
		if len(n.TagDirectives) > 0 {
			info.TagDirectives = make([]TagDirectiveInfo, len(n.TagDirectives))
			for i, td := range n.TagDirectives {
				info.TagDirectives[i] = TagDirectiveInfo{
					Handle: td.Handle,
					Prefix: td.Prefix,
				}
			}
		}
	}

	return info
}

// formatEncoding converts an encoding constant to its string representation.
func formatEncoding(e yaml.Encoding) string {
	switch e {
	case yaml.EncodingAny:
		return "Any"
	case yaml.EncodingUTF8:
		return "UTF-8"
	case yaml.EncodingUTF16LE:
		return "UTF-16LE"
	case yaml.EncodingUTF16BE:
		return "UTF-16BE"
	default:
		return "Unknown"
	}
}

// formatVersion converts a VersionDirective to its string representation.
func formatVersion(v *yaml.VersionDirective) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// formatKind converts a YAML node kind into its string representation.
func formatKind(k yaml.Kind) string {
	switch k {
	case yaml.DocumentNode:
		return "Document"
	case yaml.SequenceNode:
		return "Sequence"
	case yaml.MappingNode:
		return "Mapping"
	case yaml.ScalarNode:
		return "Scalar"
	case yaml.AliasNode:
		return "Alias"
	case yaml.StreamNode:
		return "Stream"
	default:
		return "Unknown"
	}
}

// formatStyle converts a YAML node style into its string representation.
func formatStyle(s yaml.Style, profuse bool) string {
	// Remove tagged style bit for checking base style
	baseStyle := s &^ yaml.TaggedStyle

	switch baseStyle {
	case yaml.DoubleQuotedStyle:
		return "Double"
	case yaml.SingleQuotedStyle:
		return "Single"
	case yaml.LiteralStyle:
		return "Literal"
	case yaml.FoldedStyle:
		return "Folded"
	case yaml.FlowStyle:
		return "Flow"
	case 0:
		// Plain style - only show if profuse
		if profuse {
			return "Plain"
		}
	}
	return ""
}

// formatStyleName converts a YAML node style into a lowercase style name.
// Always returns a style name (defaults to "plain" for style 0).
func formatStyleName(s yaml.Style) string {
	// Remove tagged style bit for checking base style
	baseStyle := s &^ yaml.TaggedStyle

	switch baseStyle {
	case yaml.DoubleQuotedStyle:
		return "double"
	case yaml.SingleQuotedStyle:
		return "single"
	case yaml.LiteralStyle:
		return "literal"
	case yaml.FoldedStyle:
		return "folded"
	case yaml.FlowStyle:
		return "flow"
	case 0:
		return "plain"
	default:
		return "unknown"
	}
}

// formatTag converts a YAML tag string to its string representation.
func formatTag(tag string, style yaml.Style, profuse bool) string {
	// Check if the tag was explicit in the input
	tagWasExplicit := style&yaml.TaggedStyle != 0

	// In profuse mode, always show tag
	if profuse {
		return tag
	}

	// Default YAML tags - only show if they were explicit in the input
	switch tag {
	case "!!str", "!!map", "!!seq", "!!int", "!!float", "!!bool", "!!null":
		if tagWasExplicit {
			return tag
		}
		return ""
	}

	// Show all other tags (custom tags)
	return tag
}

// FormatNodeCompact converts a YAML node into a compact representation.
// Document nodes return their content directly.
// Mapping/Sequence nodes use lowercase keys: "mapping:", "sequence:".
// Scalar nodes use style as key: "plain:", "double:", etc.
func FormatNodeCompact(n yaml.Node) any {
	switch n.Kind {
	case yaml.DocumentNode:
		// Check if document has properties that need to be preserved
		hasProperties := n.Anchor != "" || n.HeadComment != "" || n.LineComment != "" || n.FootComment != ""
		if tag := formatTag(n.Tag, n.Style, false); tag != "" && tag != "!!str" {
			hasProperties = true
		}

		// If document has no properties, return content directly (unwrap)
		if !hasProperties {
			if len(n.Content) > 0 {
				return FormatNodeCompact(*n.Content[0])
			}
			return nil
		}

		// Document has properties - create a result map with ordered keys
		result := MapSlice{}

		// Add optional fields in order: tag, anchor, comments
		if tag := formatTag(n.Tag, n.Style, false); tag != "" && tag != "!!str" {
			result = append(result, MapItem{Key: "tag", Value: tag})
		}
		if n.Anchor != "" {
			result = append(result, MapItem{Key: "anchor", Value: n.Anchor})
		}
		if n.HeadComment != "" {
			result = append(result, MapItem{Key: "head", Value: n.HeadComment})
		}
		if n.LineComment != "" {
			result = append(result, MapItem{Key: "line", Value: n.LineComment})
		}
		if n.FootComment != "" {
			result = append(result, MapItem{Key: "foot", Value: n.FootComment})
		}

		// Add content if present
		if len(n.Content) > 0 {
			content := FormatNodeCompact(*n.Content[0])
			// Merge the content into result at the top level
			if contentMap, ok := content.(MapSlice); ok {
				result = append(result, contentMap...)
			}
		}

		return result

	case yaml.MappingNode:
		result := MapSlice{}

		// Add optional fields in order: tag, anchor, comments
		if tag := formatTag(n.Tag, n.Style, false); tag != "" && tag != "!!str" {
			result = append(result, MapItem{Key: "tag", Value: tag})
		}
		if n.Anchor != "" {
			result = append(result, MapItem{Key: "anchor", Value: n.Anchor})
		}
		if n.HeadComment != "" {
			result = append(result, MapItem{Key: "head", Value: n.HeadComment})
		}
		if n.LineComment != "" {
			result = append(result, MapItem{Key: "line", Value: n.LineComment})
		}
		if n.FootComment != "" {
			result = append(result, MapItem{Key: "foot", Value: n.FootComment})
		}

		// Convert content (added last)
		var content []any
		for _, node := range n.Content {
			content = append(content, FormatNodeCompact(*node))
		}
		result = append(result, MapItem{Key: "mapping", Value: content})
		return result

	case yaml.SequenceNode:
		result := MapSlice{}

		// Add optional fields in order: tag, anchor, comments
		if tag := formatTag(n.Tag, n.Style, false); tag != "" && tag != "!!str" {
			result = append(result, MapItem{Key: "tag", Value: tag})
		}
		if n.Anchor != "" {
			result = append(result, MapItem{Key: "anchor", Value: n.Anchor})
		}
		if n.HeadComment != "" {
			result = append(result, MapItem{Key: "head", Value: n.HeadComment})
		}
		if n.LineComment != "" {
			result = append(result, MapItem{Key: "line", Value: n.LineComment})
		}
		if n.FootComment != "" {
			result = append(result, MapItem{Key: "foot", Value: n.FootComment})
		}

		// Convert content (added last)
		var content []any
		for _, node := range n.Content {
			content = append(content, FormatNodeCompact(*node))
		}
		result = append(result, MapItem{Key: "sequence", Value: content})
		return result

	case yaml.ScalarNode:
		result := MapSlice{}

		// Add optional fields in order: tag, anchor, comments
		if tag := formatTag(n.Tag, n.Style, false); tag != "" && tag != "!!str" {
			result = append(result, MapItem{Key: "tag", Value: tag})
		}
		if n.Anchor != "" {
			result = append(result, MapItem{Key: "anchor", Value: n.Anchor})
		}
		if n.HeadComment != "" {
			result = append(result, MapItem{Key: "head", Value: n.HeadComment})
		}
		if n.LineComment != "" {
			result = append(result, MapItem{Key: "line", Value: n.LineComment})
		}
		if n.FootComment != "" {
			result = append(result, MapItem{Key: "foot", Value: n.FootComment})
		}

		// Use style name as the key (added last)
		styleName := formatStyleName(n.Style)
		result = append(result, MapItem{Key: styleName, Value: n.Value})
		return result

	case yaml.AliasNode:
		result := MapSlice{}
		result = append(result, MapItem{Key: "alias", Value: n.Value})
		return result

	default:
		return nil
	}
}
