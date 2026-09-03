// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Node merger.
// Merges mapping and sequence YAML nodes.

package libyaml

import (
	"errors"
)

var (
	ErrYamlUnknown          = errors.New("yaml: unknown error")
	ErrYamlUnmergeable      = errors.New("yaml: unmergeable error")
	ErrYamlInvalidNodeKinds = errors.New("yaml: invalid node kinds")
	ErrYamlManyDocs         = errors.New("yaml: too many documents")
)

// MergeOptions configures Node.Merge().
type MergeOptions struct {
	// When MergeMap is true mapping nodes are merged instead of replaced. Default is true.
	MergeMap bool
	// When AppendSeq is true sequence nodes are appended instead of replaced. Default is false.
	AppendSeq bool
}

type MergeOption func(*MergeOptions)

// MergeMap configures Node.Merge() to merge mapping nodes.
func MergeMap(opts *MergeOptions) {
	opts.MergeMap = true
}

// ReplaceMap configures Node.Merge() to replace mapping nodes.
func ReplaceMap(opts *MergeOptions) {
	opts.MergeMap = false
}

// AppendSeq configures Node.Merge() to append sequence nodes.
func AppendSeq(opts *MergeOptions) {
	opts.AppendSeq = true
}

// ReplaceSeq configures Node.Merge() to replace sequence nodes.
func ReplaceSeq(opts *MergeOptions) {
	opts.AppendSeq = false
}

// WithAppendSeq configures Node.Merge() to append sequence nodes.
func WithAppendSeq(appendSeq bool) MergeOption {
	return func(opts *MergeOptions) {
		opts.AppendSeq = appendSeq
	}
}

// WithMergeMap configures Node.Merge() to merge mapping nodes.
func WithMergeMap(mergeMap bool) MergeOption {
	return func(opts *MergeOptions) {
		opts.MergeMap = mergeMap
	}
}

// DefaultMergeOptions returns the default MergeOptions. By default mapping nodes are merged and sequence nodes are replaced.
func DefaultMergeOptions() MergeOptions {
	return MergeOptions{
		MergeMap:  true,
		AppendSeq: false,
	}
}

// Merge merges src into dst consuming src.
// It can be used to merge multiple YAML documents into one.
func (n *Node) Merge(src *Node, opts ...MergeOption) error {
	cfg := DefaultMergeOptions()

	for _, opt := range opts {
		opt(&cfg)
	}

	return n.MergeWithOptions(src, cfg)
}

// MergeWithOptions merges src into dst consuming src.
// It can be used to merge multiple YAML documents into one.
func (n *Node) MergeWithOptions(src *Node, opts MergeOptions) error {
	dst := n

	// The case of '' -> *yaml.Node.
	if dst.Kind == 0 {
		*dst = *src
		return nil
	}
	if src.Kind == 0 {
		return nil
	}

	// implicit (foo:) or explicit (foo: null) null scalars.
	if dst.Kind == ScalarNode && dst.ShortTag() == "!!null" {
		*dst = *src
		return nil
	} else if src.Kind == ScalarNode && src.ShortTag() == "!!null" {
		*dst = *src
		return nil
	}

	if dst.Kind != src.Kind {
		return ErrYamlInvalidNodeKinds
	}

	if src.HeadComment != "" {
		dst.HeadComment = src.HeadComment
	}
	if src.LineComment != "" {
		dst.LineComment = src.LineComment
	}
	if src.FootComment != "" {
		dst.FootComment = src.FootComment
	}

	switch dst.Kind {
	case DocumentNode:
		if len(dst.Content) == 0 {
			*dst = *src
			return nil
		}

		if len(src.Content) == 0 {
			return nil
		}

		if len(dst.Content) != 1 || len(src.Content) != 1 {
			return ErrYamlManyDocs
		}

		return dst.Content[0].MergeWithOptions(src.Content[0], opts)
	case MappingNode:
		switch opts.MergeMap {
		case true:
			// Do not allow to change types in dst for mapping node.
			if dst.ShortTag() != src.ShortTag() {
				if src.Tag != "" {
					if dst.Tag != "" {
						return ErrYamlUnmergeable
					}
					dst.Tag = src.Tag
				}
			}

			// Do not allow to break aliases in dst.
			if dst.Anchor != src.Anchor {
				if src.Anchor != "" {
					if dst.Anchor != "" {
						return ErrYamlUnmergeable
					}
					dst.Anchor = src.Anchor
				}
			}

			return dst.mergeMappingNodes(src, opts)
		case false:
			*dst = *src
		}
	case AliasNode:
		// Alias contains a pointer to an ANCHOR node in the hierarchy.
		// TODO: if it is a different anchor, we must remap the pointer to the node in dst.
		//       otherwise it may point to the nodes that were left and shadowed in src.
		*dst = *src
	case SequenceNode:
		switch opts.AppendSeq {
		case true:
			dst.Content = append(dst.Content, src.Content...)
		case false:
			*dst = *src
		}
	case ScalarNode:
		*dst = *src
	default:
		return ErrYamlUnknown
	}

	return nil
}

// mergeMappingNodes merges two mapping nodes.
func (n *Node) mergeMappingNodes(src *Node, opts MergeOptions) error {
	dst := n
	dstMap := mapNodeToMap(dst)

	for i := 0; i+1 < len(src.Content); i += 2 {
		key := src.Content[i]
		val := src.Content[i+1]

		if dstPair, exists := dstMap[key.Value]; exists {
			// key docs
			if err := dstPair.key.MergeWithOptions(key, opts); err != nil {
				return err
			}
			if err := dstPair.val.MergeWithOptions(val, opts); err != nil {
				return err
			}
		} else {
			dst.Content = append(dst.Content, key, val)
		}
	}

	return nil
}

type yamlNodeKVPair struct {
	key *Node
	val *Node
}

// mapNodeToMap converts a *Node of kind MappingNode to a Go map
func mapNodeToMap(node *Node) map[string]yamlNodeKVPair {
	result := make(map[string]yamlNodeKVPair)

	// Keys are at even indices, values at odd indices
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		val := node.Content[i+1]
		result[key.Value] = yamlNodeKVPair{key, val}
	}

	return result
}
