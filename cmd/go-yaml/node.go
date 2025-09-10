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
func FormatNode(n yaml.Node) *NodeInfo {
	info := &NodeInfo{
		Kind: formatKind(n.Kind),
	}

	if style := formatStyle(n.Style); style != "" {
		info.Style = style
	}
	if n.Anchor != "" {
		info.Anchor = n.Anchor
	}
	if tag := formatTag(n.Tag, n.Style); tag != "" {
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
			info.Content[i] = FormatNode(*node)
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
func formatStyle(s yaml.Style) string {
	switch s {
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
	}
	return ""
}

// formatTag converts a YAML tag string to its string representation.
func formatTag(tag string, style yaml.Style) string {
	// Check if the tag was explicit in the input
	tagWasExplicit := style&yaml.TaggedStyle != 0

	// Show !!str only if it was explicit in the input
	switch tag {
	case "!!str", "!!map", "!!seq":
		if tagWasExplicit {
			return tag
		}
		return ""
	}

	// Show all other tags
	return tag
}
