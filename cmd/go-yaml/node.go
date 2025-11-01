// YAML node formatting utilities for the go-yaml tool.

package main

import (
	"go.yaml.in/yaml/v4"
)

// NodeInfo represents the information about a YAML node
type NodeInfo struct {
	Kind    string      `yaml:"kind"`
	Style   string      `yaml:"style,omitempty"`
	Anchor  string      `yaml:"anchor,omitempty"`
	Tag     string      `yaml:"tag,omitempty"`
	Head    string      `yaml:"head,omitempty"`
	Line    string      `yaml:"line,omitempty"`
	Foot    string      `yaml:"foot,omitempty"`
	Text    string      `yaml:"text,omitempty"`
	Content []*NodeInfo `yaml:"content,omitempty"`
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

	return info
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
