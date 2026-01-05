//
// Copyright (c) 2011-2019 Canonical Ltd
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

// Composer stage: Builds a node tree from a libyaml event stream.
// Handles document structure, anchors, and comment attachment.

package libyaml

import (
	"fmt"
	"io"
)

// Composer produces a node tree out of a libyaml event stream.
type Composer struct {
	Parser       Parser
	event        Event
	doc          *Node
	anchors      map[string]*Node
	doneInit     bool
	Textless     bool
	streamNodes  bool     // enable stream node emission
	returnStream bool     // flag to return stream node next
	atStreamEnd  bool     // at stream end
	encoding     Encoding // stream encoding from STREAM_START
}

// NewComposer creates a new composer from a byte slice.
func NewComposer(b []byte) *Composer {
	p := Composer{
		Parser: NewParser(),
	}
	if len(b) == 0 {
		b = []byte{'\n'}
	}
	p.Parser.SetInputString(b)
	return &p
}

// NewComposerFromReader creates a new composer from an io.Reader.
func NewComposerFromReader(r io.Reader) *Composer {
	p := Composer{
		Parser: NewParser(),
	}
	p.Parser.SetInputReader(r)
	return &p
}

func (p *Composer) init() {
	if p.doneInit {
		return
	}
	p.anchors = make(map[string]*Node)
	// Peek to get the encoding from STREAM_START_EVENT
	if p.peek() == STREAM_START_EVENT {
		p.encoding = p.event.GetEncoding()
	}
	p.expect(STREAM_START_EVENT)
	p.doneInit = true

	// If stream nodes are enabled, prepare to return the first stream node
	if p.streamNodes {
		p.returnStream = true
	}
}

func (p *Composer) Destroy() {
	if p.event.Type != NO_EVENT {
		p.event.Delete()
	}
	p.Parser.Delete()
}

// SetStreamNodes enables or disables stream node emission.
func (p *Composer) SetStreamNodes(enable bool) {
	p.streamNodes = enable
}

// expect consumes an event from the event stream and
// checks that it's of the expected type.
func (p *Composer) expect(e EventType) {
	if p.event.Type == NO_EVENT {
		if err := p.Parser.Parse(&p.event); err != nil {
			p.fail(err)
		}
	}
	if p.event.Type == STREAM_END_EVENT {
		failf("attempted to go past the end of stream; corrupted value?")
	}
	if p.event.Type != e {
		p.fail(fmt.Errorf("expected %s event but got %s", e, p.event.Type))
	}
	p.event.Delete()
	p.event.Type = NO_EVENT
}

// peek peeks at the next event in the event stream,
// puts the results into p.event and returns the event type.
func (p *Composer) peek() EventType {
	if p.event.Type != NO_EVENT {
		return p.event.Type
	}
	// It's curious choice from the underlying API to generally return a
	// positive result on success, but on this case return true in an error
	// scenario. This was the source of bugs in the past (issue #666).
	if err := p.Parser.Parse(&p.event); err != nil {
		p.fail(err)
	}
	return p.event.Type
}

func (p *Composer) fail(err error) {
	Fail(err)
}

func (p *Composer) anchor(n *Node, anchor []byte) {
	if anchor != nil {
		n.Anchor = string(anchor)
		p.anchors[n.Anchor] = n
	}
}

// Parse parses the next YAML node from the event stream.
func (p *Composer) Parse() *Node {
	p.init()

	// Handle stream nodes if enabled
	if p.streamNodes {
		// Check for stream end first
		if p.peek() == STREAM_END_EVENT {
			// If we haven't returned the final stream node yet, return it now
			if !p.atStreamEnd {
				p.atStreamEnd = true
				return p.createStreamNode()
			}
			// Already returned final stream node
			return nil
		}

		// Check if we should return a stream node before the next document
		if p.returnStream {
			p.returnStream = false
			n := p.createStreamNode()
			// Capture directives from upcoming document
			p.captureDirectives(n)
			return n
		}
	}

	switch p.peek() {
	case SCALAR_EVENT:
		return p.scalar()
	case ALIAS_EVENT:
		return p.alias()
	case MAPPING_START_EVENT:
		return p.mapping()
	case SEQUENCE_START_EVENT:
		return p.sequence()
	case DOCUMENT_START_EVENT:
		return p.document()
	case STREAM_END_EVENT:
		// Happens when attempting to decode an empty buffer (when not using stream nodes).
		return nil
	case TAIL_COMMENT_EVENT:
		panic("internal error: unexpected tail comment event (please report)")
	default:
		panic("internal error: attempted to parse unknown event (please report): " + p.event.Type.String())
	}
}

func (p *Composer) node(kind Kind, defaultTag, tag, value string) *Node {
	var style Style
	if tag != "" && tag != "!" {
		// Normalize tag to short form (e.g., tag:yaml.org,2002:str -> !!str)
		tag = shortTag(tag)
		style = TaggedStyle
	} else if defaultTag != "" {
		tag = defaultTag
	} else if kind == ScalarNode {
		// Delegate to resolver to determine tag from value
		tag, _ = resolve("", value)
	}
	n := &Node{
		Kind:  kind,
		Tag:   tag,
		Value: value,
		Style: style,
	}
	if !p.Textless {
		n.Line = p.event.StartMark.Line + 1
		n.Column = p.event.StartMark.Column + 1
		n.HeadComment = string(p.event.HeadComment)
		n.LineComment = string(p.event.LineComment)
		n.FootComment = string(p.event.FootComment)
	}
	return n
}

func (p *Composer) parseChild(parent *Node) *Node {
	child := p.Parse()
	parent.Content = append(parent.Content, child)
	return child
}

func (p *Composer) document() *Node {
	n := p.node(DocumentNode, "", "", "")
	p.doc = n
	p.expect(DOCUMENT_START_EVENT)
	p.parseChild(n)
	if p.peek() == DOCUMENT_END_EVENT {
		n.FootComment = string(p.event.FootComment)
	}
	p.expect(DOCUMENT_END_EVENT)

	// If stream nodes enabled, prepare to return a stream node next
	if p.streamNodes {
		p.returnStream = true
	}

	return n
}

func (p *Composer) createStreamNode() *Node {
	n := &Node{
		Kind:     StreamNode,
		Encoding: p.encoding,
	}
	if !p.Textless && p.event.Type != NO_EVENT {
		n.Line = p.event.StartMark.Line + 1
		n.Column = p.event.StartMark.Column + 1
	}
	return n
}

// captureDirectives captures version and tag directives from upcoming DOCUMENT_START.
func (p *Composer) captureDirectives(n *Node) {
	if p.peek() == DOCUMENT_START_EVENT {
		if vd := p.event.GetVersionDirective(); vd != nil {
			n.Version = &StreamVersionDirective{
				Major: vd.Major(),
				Minor: vd.Minor(),
			}
		}
		if tds := p.event.GetTagDirectives(); len(tds) > 0 {
			n.TagDirectives = make([]StreamTagDirective, len(tds))
			for i, td := range tds {
				n.TagDirectives[i] = StreamTagDirective{
					Handle: td.GetHandle(),
					Prefix: td.GetPrefix(),
				}
			}
		}
	}
}

func (p *Composer) alias() *Node {
	n := p.node(AliasNode, "", "", string(p.event.Anchor))
	n.Alias = p.anchors[n.Value]
	if n.Alias == nil {
		msg := fmt.Sprintf("unknown anchor '%s' referenced", n.Value)
		Fail(&ParserError{
			Message: msg,
			Mark: Mark{
				Line:   n.Line,
				Column: n.Column,
			},
		})
	}
	p.expect(ALIAS_EVENT)
	return n
}

func (p *Composer) scalar() *Node {
	parsedStyle := p.event.ScalarStyle()
	var nodeStyle Style
	switch {
	case parsedStyle&DOUBLE_QUOTED_SCALAR_STYLE != 0:
		nodeStyle = DoubleQuotedStyle
	case parsedStyle&SINGLE_QUOTED_SCALAR_STYLE != 0:
		nodeStyle = SingleQuotedStyle
	case parsedStyle&LITERAL_SCALAR_STYLE != 0:
		nodeStyle = LiteralStyle
	case parsedStyle&FOLDED_SCALAR_STYLE != 0:
		nodeStyle = FoldedStyle
	}
	nodeValue := string(p.event.Value)
	nodeTag := string(p.event.Tag)
	var defaultTag string
	if nodeStyle != 0 {
		defaultTag = strTag
	}
	n := p.node(ScalarNode, defaultTag, nodeTag, nodeValue)
	n.Style |= nodeStyle
	p.anchor(n, p.event.Anchor)
	p.expect(SCALAR_EVENT)
	return n
}

func (p *Composer) sequence() *Node {
	n := p.node(SequenceNode, seqTag, string(p.event.Tag), "")
	if p.event.SequenceStyle()&FLOW_SEQUENCE_STYLE != 0 {
		n.Style |= FlowStyle
	}
	p.anchor(n, p.event.Anchor)
	p.expect(SEQUENCE_START_EVENT)
	for p.peek() != SEQUENCE_END_EVENT {
		p.parseChild(n)
	}
	n.LineComment = string(p.event.LineComment)
	n.FootComment = string(p.event.FootComment)
	p.expect(SEQUENCE_END_EVENT)
	return n
}

func (p *Composer) mapping() *Node {
	n := p.node(MappingNode, mapTag, string(p.event.Tag), "")
	block := true
	if p.event.MappingStyle()&FLOW_MAPPING_STYLE != 0 {
		block = false
		n.Style |= FlowStyle
	}
	p.anchor(n, p.event.Anchor)
	p.expect(MAPPING_START_EVENT)
	for p.peek() != MAPPING_END_EVENT {
		k := p.parseChild(n)
		if block && k.FootComment != "" {
			// Must be a foot comment for the prior value when being dedented.
			if len(n.Content) > 2 {
				n.Content[len(n.Content)-3].FootComment = k.FootComment
				k.FootComment = ""
			}
		}
		v := p.parseChild(n)
		if k.FootComment == "" && v.FootComment != "" {
			k.FootComment = v.FootComment
			v.FootComment = ""
		}
		if p.peek() == TAIL_COMMENT_EVENT {
			if k.FootComment == "" {
				k.FootComment = string(p.event.FootComment)
			}
			p.expect(TAIL_COMMENT_EVENT)
		}
	}
	n.LineComment = string(p.event.LineComment)
	n.FootComment = string(p.event.FootComment)
	if n.Style&FlowStyle == 0 && n.FootComment != "" && len(n.Content) > 1 {
		n.Content[len(n.Content)-2].FootComment = n.FootComment
		n.FootComment = ""
	}
	p.expect(MAPPING_END_EVENT)
	return n
}

// YAMLError is an internal error wrapper type.
type YAMLError struct {
	Err error
}

func (e *YAMLError) Error() string {
	return e.Err.Error()
}

func Fail(err error) {
	panic(&YAMLError{err})
}

func failf(format string, args ...any) {
	panic(&YAMLError{fmt.Errorf("yaml: "+format, args...)})
}
