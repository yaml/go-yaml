//
// Copyright (c) 2011-2019 Canonical Ltd
// Copyright 2025 The go-yaml Project Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package libyaml

import (
	"strings"
	"unicode/utf8"
)

// Serializer Stage: Converts representation tree (Nodes) to event stream
// The serializer walks the node tree and produces events via emitter.Emit()

// node serializes a Node tree into YAML events.
// This is the core of the serializer stage - it walks the tree and produces events.
func (e *Representer) node(node *Node, tail string) {
	// Zero nodes behave as nil.
	if node.Kind == 0 && node.IsZero() {
		e.nilv()
		return
	}

	// If the tag was not explicitly requested, and dropping it won't change the
	// implicit tag of the value, don't include it in the presentation.
	tag := node.Tag
	stag := shortTag(tag)
	var forceQuoting bool
	if tag != "" && node.Style&TaggedStyle == 0 {
		if node.Kind == ScalarNode {
			if stag == strTag && node.Style&(SingleQuotedStyle|DoubleQuotedStyle|LiteralStyle|FoldedStyle) != 0 {
				tag = ""
			} else {
				rtag, _ := resolve("", node.Value)
				if rtag == stag {
					tag = ""
				} else if stag == strTag {
					tag = ""
					forceQuoting = true
				}
			}
		} else {
			var rtag string
			switch node.Kind {
			case MappingNode:
				rtag = mapTag
			case SequenceNode:
				rtag = seqTag
			}
			if rtag == stag {
				tag = ""
			}
		}
	}

	switch node.Kind {
	case DocumentNode:
		event := NewDocumentStartEvent(noVersionDirective, noTagDirective, true)
		event.HeadComment = []byte(node.HeadComment)
		e.emit(event)
		for _, node := range node.Content {
			e.node(node, "")
		}
		event = NewDocumentEndEvent(true)
		event.FootComment = []byte(node.FootComment)
		e.emit(event)

	case SequenceNode:
		style := BLOCK_SEQUENCE_STYLE
		// Use flow style if explicitly requested or if it's a simple
		// collection (scalar-only contents that fit within line width,
		// enabled via WithFlowSimpleCollections)
		if node.Style&FlowStyle != 0 || e.isSimpleCollection(node) {
			style = FLOW_SEQUENCE_STYLE
		}
		event := NewSequenceStartEvent([]byte(node.Anchor), []byte(longTag(tag)), tag == "", style)
		event.HeadComment = []byte(node.HeadComment)
		e.emit(event)
		for _, node := range node.Content {
			e.node(node, "")
		}
		event = NewSequenceEndEvent()
		event.LineComment = []byte(node.LineComment)
		event.FootComment = []byte(node.FootComment)
		e.emit(event)

	case MappingNode:
		style := BLOCK_MAPPING_STYLE
		// Use flow style if explicitly requested or if it's a simple
		// collection (scalar-only contents that fit within line width,
		// enabled via WithFlowSimpleCollections)
		if node.Style&FlowStyle != 0 || e.isSimpleCollection(node) {
			style = FLOW_MAPPING_STYLE
		}
		event := NewMappingStartEvent([]byte(node.Anchor), []byte(longTag(tag)), tag == "", style)
		event.TailComment = []byte(tail)
		event.HeadComment = []byte(node.HeadComment)
		e.emit(event)

		// The tail logic below moves the foot comment of prior keys to the following key,
		// since the value for each key may be a nested structure and the foot needs to be
		// processed only the entirety of the value is streamed. The last tail is processed
		// with the mapping end event.
		var tail string
		for i := 0; i+1 < len(node.Content); i += 2 {
			k := node.Content[i]
			foot := k.FootComment
			if foot != "" {
				kopy := *k
				kopy.FootComment = ""
				k = &kopy
			}
			e.node(k, tail)
			tail = foot

			v := node.Content[i+1]
			e.node(v, "")
		}

		event = NewMappingEndEvent()
		event.TailComment = []byte(tail)
		event.LineComment = []byte(node.LineComment)
		event.FootComment = []byte(node.FootComment)
		e.emit(event)

	case AliasNode:
		event := NewAliasEvent([]byte(node.Value))
		event.HeadComment = []byte(node.HeadComment)
		event.LineComment = []byte(node.LineComment)
		event.FootComment = []byte(node.FootComment)
		e.emit(event)

	case ScalarNode:
		value := node.Value
		if !utf8.ValidString(value) {
			if stag == binaryTag {
				failf("explicitly tagged !!binary data must be base64-encoded")
			}
			if stag != "" {
				failf("cannot marshal invalid UTF-8 data as %s", stag)
			}
			// It can't be representd directly as YAML so use a binary tag
			// and represent it as base64.
			tag = binaryTag
			value = encodeBase64(value)
		}

		style := PLAIN_SCALAR_STYLE
		switch {
		case node.Style&DoubleQuotedStyle != 0:
			style = DOUBLE_QUOTED_SCALAR_STYLE
		case node.Style&SingleQuotedStyle != 0:
			style = SINGLE_QUOTED_SCALAR_STYLE
		case node.Style&LiteralStyle != 0:
			style = LITERAL_SCALAR_STYLE
		case node.Style&FoldedStyle != 0:
			style = FOLDED_SCALAR_STYLE
		case strings.Contains(value, "\n"):
			style = LITERAL_SCALAR_STYLE
		case forceQuoting:
			style = DOUBLE_QUOTED_SCALAR_STYLE
		}

		e.emitScalar(value, node.Anchor, tag, style, []byte(node.HeadComment), []byte(node.LineComment), []byte(node.FootComment), []byte(tail))
	default:
		failf("cannot represent node with unknown kind %d", node.Kind)
	}
}

// isSimpleCollection checks if a node contains only scalar values and would
// fit within the line width when rendered in flow style.
func (e *Representer) isSimpleCollection(node *Node) bool {
	if !e.flowSimpleCollections {
		return false
	}
	if node.Kind != SequenceNode && node.Kind != MappingNode {
		return false
	}
	// Check all children are scalars
	for _, child := range node.Content {
		if child.Kind != ScalarNode {
			return false
		}
	}
	// Estimate flow style length
	estimatedLen := e.estimateFlowLength(node)
	width := e.lineWidth
	if width <= 0 {
		width = 80 // Default width if not set
	}
	return estimatedLen > 0 && estimatedLen <= width
}

// estimateFlowLength estimates the character length of a node in flow style.
func (e *Representer) estimateFlowLength(node *Node) int {
	if node.Kind == SequenceNode {
		// [item1, item2, ...] = 2 + sum(len(items)) + 2*(len-1)
		length := 2 // []
		for i, child := range node.Content {
			if i > 0 {
				length += 2 // ", "
			}
			length += len(child.Value)
		}
		return length
	}
	if node.Kind == MappingNode {
		// {key1: val1, key2: val2} = 2 + sum(key: val) + 2*(pairs-1)
		length := 2 // {}
		for i := 0; i < len(node.Content); i += 2 {
			if i > 0 {
				length += 2 // ", "
			}
			length += len(node.Content[i].Value) + 2 + len(node.Content[i+1].Value) // "key: val"
		}
		return length
	}
	return 0
}
