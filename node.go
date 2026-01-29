package yaml

import "go.yaml.in/yaml/v4/internal/libyaml"

//-----------------------------------------------------------------------------
// Node-related type aliases and constants
//-----------------------------------------------------------------------------

type (
	// Node represents a YAML node in the document tree.
	// See internal/libyaml.Node.
	Node = libyaml.Node
	// Kind identifies the type of a YAML node.
	// See internal/libyaml.Kind.
	Kind = libyaml.Kind
	// Style controls the presentation of a YAML node.
	// See internal/libyaml.Style.
	Style = libyaml.Style
	// Marshaler is implemented by types with custom YAML marshaling.
	// See internal/libyaml.Marshaler.
	Marshaler = libyaml.Marshaler
	// IsZeroer is implemented by types that can report if they're zero.
	// See internal/libyaml.IsZeroer.
	IsZeroer = libyaml.IsZeroer
)

// Unmarshaler is the interface implemented by types
// that can unmarshal a YAML description of themselves.
type Unmarshaler interface {
	UnmarshalYAML(node *Node) error
}

// Re-export Kind constants
const (
	DocumentNode = libyaml.DocumentNode
	SequenceNode = libyaml.SequenceNode
	MappingNode  = libyaml.MappingNode
	ScalarNode   = libyaml.ScalarNode
	AliasNode    = libyaml.AliasNode
	StreamNode   = libyaml.StreamNode
)

// Re-export Style constants
const (
	TaggedStyle       = libyaml.TaggedStyle
	DoubleQuotedStyle = libyaml.DoubleQuotedStyle
	SingleQuotedStyle = libyaml.SingleQuotedStyle
	LiteralStyle      = libyaml.LiteralStyle
	FoldedStyle       = libyaml.FoldedStyle
	FlowStyle         = libyaml.FlowStyle
)
