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

package libyaml

import (
	"encoding"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

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


type keyList []reflect.Value

func (l keyList) Len() int      { return len(l) }
func (l keyList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l keyList) Less(i, j int) bool {
	a := l[i]
	b := l[j]
	ak := a.Kind()
	bk := b.Kind()
	for (ak == reflect.Interface || ak == reflect.Pointer) && !a.IsNil() {
		a = a.Elem()
		ak = a.Kind()
	}
	for (bk == reflect.Interface || bk == reflect.Pointer) && !b.IsNil() {
		b = b.Elem()
		bk = b.Kind()
	}
	af, aok := keyFloat(a)
	bf, bok := keyFloat(b)
	if aok && bok {
		if af != bf {
			return af < bf
		}
		if ak != bk {
			return ak < bk
		}
		return numLess(a, b)
	}
	if ak != reflect.String || bk != reflect.String {
		return ak < bk
	}
	ar, br := []rune(a.String()), []rune(b.String())
	digits := false
	for i := 0; i < len(ar) && i < len(br); i++ {
		if ar[i] == br[i] {
			digits = unicode.IsDigit(ar[i])
			continue
		}
		al := unicode.IsLetter(ar[i])
		bl := unicode.IsLetter(br[i])
		if al && bl {
			return ar[i] < br[i]
		}
		if al || bl {
			if digits {
				return al
			} else {
				return bl
			}
		}
		var ai, bi int
		var an, bn int64
		if ar[i] == '0' || br[i] == '0' {
			for j := i - 1; j >= 0 && unicode.IsDigit(ar[j]); j-- {
				if ar[j] != '0' {
					an = 1
					bn = 1
					break
				}
			}
		}
		for ai = i; ai < len(ar) && unicode.IsDigit(ar[ai]); ai++ {
			an = an*10 + int64(ar[ai]-'0')
		}
		for bi = i; bi < len(br) && unicode.IsDigit(br[bi]); bi++ {
			bn = bn*10 + int64(br[bi]-'0')
		}
		if an != bn {
			return an < bn
		}
		if ai != bi {
			return ai < bi
		}
		return ar[i] < br[i]
	}
	return len(ar) < len(br)
}

// keyFloat returns a float value for v if it is a number/bool
// and whether it is a number/bool or not.
func keyFloat(v reflect.Value) (f float64, ok bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(v.Uint()), true
	case reflect.Bool:
		if v.Bool() {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// numLess returns whether a < b.
// a and b must necessarily have the same kind.
func numLess(a, b reflect.Value) bool {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() < b.Int()
	case reflect.Float32, reflect.Float64:
		return a.Float() < b.Float()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return a.Uint() < b.Uint()
	case reflect.Bool:
		return !a.Bool() && b.Bool()
	}
	panic("not a number")
}
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


// Sentinel values for newEncoder parameters.
// These provide clarity at call sites, similar to http.NoBody.
var (
	noWriter           io.Writer                 = nil
	noVersionDirective *VersionDirective = nil
	noTagDirective     []TagDirective    = nil
)

type Encoder struct {
	Emitter               Emitter
	Out                   []byte
	flow                  bool
	Indent                int
	lineWidth             int
	doneInit              bool
	explicitStart         bool
	explicitEnd           bool
	flowSimpleCollections bool
}

// newEncoder creates a new YAML encoder with the given options.
//
// The writer parameter specifies the output destination for the encoder.
// If writer is nil, the encoder will write to an internal buffer.
func NewEncoder(writer io.Writer, opts *Options) *Encoder {
	emitter := NewEmitter()
	emitter.CompactSequenceIndent = opts.CompactSeqIndent
	emitter.SetWidth(opts.LineWidth)
	emitter.SetUnicode(opts.Unicode)
	emitter.SetCanonical(opts.Canonical)
	emitter.SetLineBreak(opts.LineBreak)

	e := &Encoder{
		Emitter:               emitter,
		Indent:                opts.Indent,
		lineWidth:             opts.LineWidth,
		explicitStart:         opts.ExplicitStart,
		explicitEnd:           opts.ExplicitEnd,
		flowSimpleCollections: opts.FlowSimpleCollections,
	}

	if writer != nil {
		e.Emitter.SetOutputWriter(writer)
	} else {
		e.Emitter.SetOutputString(&e.Out)
	}

	return e
}

func (e *Encoder) init() {
	if e.doneInit {
		return
	}
	if e.Indent == 0 {
		e.Indent = 4
	}
	e.Emitter.BestIndent = e.Indent
	e.emit(NewStreamStartEvent(UTF8_ENCODING))
	e.doneInit = true
}

func (e *Encoder) Finish() {
	e.Emitter.OpenEnded = false
	e.emit(NewStreamEndEvent())
}

func (e *Encoder) Destroy() {
	e.Emitter.Delete()
}

func (e *Encoder) emit(event Event) {
	// This will internally delete the event value.
	e.must(e.Emitter.Emit(&event))
}

func (e *Encoder) must(err error) {
	if err != nil {
		msg := err.Error()
		if msg == "" {
			msg = "unknown problem generating YAML content"
		}
		failf("%s", msg)
	}
}

func (e *Encoder) MarshalDoc(tag string, in reflect.Value) {
	e.init()
	var node *Node
	if in.IsValid() {
		node, _ = in.Interface().(*Node)
	}
	if node != nil && node.Kind == DocumentNode {
		e.nodev(in)
	} else {
		// Use !explicitStart for implicit flag (true = implicit/no marker)
		e.emit(NewDocumentStartEvent(noVersionDirective, noTagDirective, !e.explicitStart))
		e.marshal(tag, in)
		// Use !explicitEnd for implicit flag
		e.emit(NewDocumentEndEvent(!e.explicitEnd))
	}
}

// isSimpleCollection checks if a node contains only scalar values and would
// fit within the line width when rendered in flow style.
func (e *Encoder) isSimpleCollection(node *Node) bool {
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
func (e *Encoder) estimateFlowLength(node *Node) int {
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

func (e *Encoder) marshal(tag string, in reflect.Value) {
	tag = shortTag(tag)
	if !in.IsValid() || in.Kind() == reflect.Pointer && in.IsNil() {
		e.nilv()
		return
	}
	iface := in.Interface()
	switch value := iface.(type) {
	case *Node:
		e.nodev(in)
		return
	case Node:
		if !in.CanAddr() {
			n := reflect.New(in.Type()).Elem()
			n.Set(in)
			in = n
		}
		e.nodev(in.Addr())
		return
	case time.Time:
		e.timev(tag, in)
		return
	case *time.Time:
		e.timev(tag, in.Elem())
		return
	case time.Duration:
		e.stringv(tag, reflect.ValueOf(value.String()))
		return
	case Marshaler:
		v, err := value.MarshalYAML()
		if err != nil {
			fail(err)
		}
		if v == nil {
			e.nilv()
			return
		}
		e.marshal(tag, reflect.ValueOf(v))
		return
	case encoding.TextMarshaler:
		text, err := value.MarshalText()
		if err != nil {
			fail(err)
		}
		in = reflect.ValueOf(string(text))
	case nil:
		e.nilv()
		return
	}
	switch in.Kind() {
	case reflect.Interface:
		e.marshal(tag, in.Elem())
	case reflect.Map:
		e.mapv(tag, in)
	case reflect.Pointer:
		e.marshal(tag, in.Elem())
	case reflect.Struct:
		e.structv(tag, in)
	case reflect.Slice, reflect.Array:
		e.slicev(tag, in)
	case reflect.String:
		e.stringv(tag, in)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.intv(tag, in)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		e.uintv(tag, in)
	case reflect.Float32, reflect.Float64:
		e.floatv(tag, in)
	case reflect.Bool:
		e.boolv(tag, in)
	default:
		panic("cannot marshal type: " + in.Type().String())
	}
}

func (e *Encoder) mapv(tag string, in reflect.Value) {
	e.mappingv(tag, func() {
		keys := keyList(in.MapKeys())
		sort.Sort(keys)
		for _, k := range keys {
			e.marshal("", k)
			e.marshal("", in.MapIndex(k))
		}
	})
}

func (e *Encoder) fieldByIndex(v reflect.Value, index []int) (field reflect.Value) {
	for _, num := range index {
		for {
			if v.Kind() == reflect.Pointer {
				if v.IsNil() {
					return reflect.Value{}
				}
				v = v.Elem()
				continue
			}
			break
		}
		v = v.Field(num)
	}
	return v
}

func (e *Encoder) structv(tag string, in reflect.Value) {
	sinfo, err := getStructInfo(in.Type())
	if err != nil {
		panic(err)
	}
	e.mappingv(tag, func() {
		for _, info := range sinfo.FieldsList {
			var value reflect.Value
			if info.Inline == nil {
				value = in.Field(info.Num)
			} else {
				value = e.fieldByIndex(in, info.Inline)
				if !value.IsValid() {
					continue
				}
			}
			if info.OmitEmpty && isZero(value) {
				continue
			}
			e.marshal("", reflect.ValueOf(info.Key))
			e.flow = info.Flow
			e.marshal("", value)
		}
		if sinfo.InlineMap >= 0 {
			m := in.Field(sinfo.InlineMap)
			if m.Len() > 0 {
				e.flow = false
				keys := keyList(m.MapKeys())
				sort.Sort(keys)
				for _, k := range keys {
					if _, found := sinfo.FieldsMap[k.String()]; found {
						panic(fmt.Sprintf("cannot have key %q in inlined map: conflicts with struct field", k.String()))
					}
					e.marshal("", k)
					e.flow = false
					e.marshal("", m.MapIndex(k))
				}
			}
		}
	})
}

func (e *Encoder) mappingv(tag string, f func()) {
	implicit := tag == ""
	style := BLOCK_MAPPING_STYLE
	if e.flow {
		e.flow = false
		style = FLOW_MAPPING_STYLE
	}
	e.emit(NewMappingStartEvent(nil, []byte(tag), implicit, style))
	f()
	e.emit(NewMappingEndEvent())
}

func (e *Encoder) slicev(tag string, in reflect.Value) {
	implicit := tag == ""
	style := BLOCK_SEQUENCE_STYLE
	if e.flow {
		e.flow = false
		style = FLOW_SEQUENCE_STYLE
	}
	e.emit(NewSequenceStartEvent(nil, []byte(tag), implicit, style))
	n := in.Len()
	for i := 0; i < n; i++ {
		e.marshal("", in.Index(i))
	}
	e.emit(NewSequenceEndEvent())
}

// isBase60 returns whether s is in base 60 notation as defined in YAML 1.1.
//
// The base 60 float notation in YAML 1.1 is a terrible idea and is unsupported
// in YAML 1.2 and by this package, but these should be marshaled quoted for
// the time being for compatibility with other parsers.
func isBase60Float(s string) (result bool) {
	// Fast path.
	if s == "" {
		return false
	}
	c := s[0]
	if !(c == '+' || c == '-' || c >= '0' && c <= '9') || strings.IndexByte(s, ':') < 0 {
		return false
	}
	// Do the full match.
	return base60float.MatchString(s)
}

// From http://yaml.org/type/float.html, except the regular expression there
// is bogus. In practice parsers do not enforce the "\.[0-9_]*" suffix.
var base60float = regexp.MustCompile(`^[-+]?[0-9][0-9_]*(?::[0-5]?[0-9])+(?:\.[0-9_]*)?$`)

// isOldBool returns whether s is bool notation as defined in YAML 1.1.
//
// We continue to force strings that YAML 1.1 would interpret as booleans to be
// rendered as quotes strings so that the marshaled output valid for YAML 1.1
// parsing.
func isOldBool(s string) (result bool) {
	switch s {
	case "y", "Y", "yes", "Yes", "YES", "on", "On", "ON",
		"n", "N", "no", "No", "NO", "off", "Off", "OFF":
		return true
	default:
		return false
	}
}

// looksLikeMerge returns true if the given string is the merge indicator "<<".
//
// When encoding a scalar with this exact value, it must be quoted to prevent it
// from being interpreted as a merge indicator during decoding.
func looksLikeMerge(s string) (result bool) {
	return s == "<<"
}

func (e *Encoder) stringv(tag string, in reflect.Value) {
	var style ScalarStyle
	s := in.String()
	canUsePlain := true
	switch {
	case !utf8.ValidString(s):
		if tag == binaryTag {
			failf("explicitly tagged !!binary data must be base64-encoded")
		}
		if tag != "" {
			failf("cannot marshal invalid UTF-8 data as %s", shortTag(tag))
		}
		// It can't be encoded directly as YAML so use a binary tag
		// and encode it as base64.
		tag = binaryTag
		s = encodeBase64(s)
	case tag == "":
		// Check to see if it would resolve to a specific
		// tag when encoded unquoted. If it doesn't,
		// there's no need to quote it.
		rtag, _ := resolve("", s)
		canUsePlain = rtag == strTag &&
			!(isBase60Float(s) ||
				isOldBool(s) ||
				looksLikeMerge(s))
	}
	// Note: it's possible for user code to emit invalid YAML
	// if they explicitly specify a tag and a string containing
	// text that's incompatible with that tag.
	switch {
	case strings.Contains(s, "\n"):
		if e.flow || !shouldUseLiteralStyle(s) {
			style = DOUBLE_QUOTED_SCALAR_STYLE
		} else {
			style = LITERAL_SCALAR_STYLE
		}
	case canUsePlain:
		style = PLAIN_SCALAR_STYLE
	default:
		style = DOUBLE_QUOTED_SCALAR_STYLE
	}
	e.emitScalar(s, "", tag, style, nil, nil, nil, nil)
}

func (e *Encoder) boolv(tag string, in reflect.Value) {
	var s string
	if in.Bool() {
		s = "true"
	} else {
		s = "false"
	}
	e.emitScalar(s, "", tag, PLAIN_SCALAR_STYLE, nil, nil, nil, nil)
}

func (e *Encoder) intv(tag string, in reflect.Value) {
	s := strconv.FormatInt(in.Int(), 10)
	e.emitScalar(s, "", tag, PLAIN_SCALAR_STYLE, nil, nil, nil, nil)
}

func (e *Encoder) uintv(tag string, in reflect.Value) {
	s := strconv.FormatUint(in.Uint(), 10)
	e.emitScalar(s, "", tag, PLAIN_SCALAR_STYLE, nil, nil, nil, nil)
}

func (e *Encoder) timev(tag string, in reflect.Value) {
	t := in.Interface().(time.Time)
	s := t.Format(time.RFC3339Nano)
	e.emitScalar(s, "", tag, PLAIN_SCALAR_STYLE, nil, nil, nil, nil)
}

func (e *Encoder) floatv(tag string, in reflect.Value) {
	// Issue #352: When formatting, use the precision of the underlying value
	precision := 64
	if in.Kind() == reflect.Float32 {
		precision = 32
	}

	s := strconv.FormatFloat(in.Float(), 'g', -1, precision)
	switch s {
	case "+Inf":
		s = ".inf"
	case "-Inf":
		s = "-.inf"
	case "NaN":
		s = ".nan"
	}
	e.emitScalar(s, "", tag, PLAIN_SCALAR_STYLE, nil, nil, nil, nil)
}

func (e *Encoder) nilv() {
	e.emitScalar("null", "", "", PLAIN_SCALAR_STYLE, nil, nil, nil, nil)
}

func (e *Encoder) emitScalar(
	value, anchor, tag string, style ScalarStyle, head, line, foot, tail []byte,
) {
	// TODO Kill this function. Replace all initialize calls by their underlining Go literals.
	implicit := tag == ""
	if !implicit {
		tag = longTag(tag)
	}
	event := NewScalarEvent([]byte(anchor), []byte(tag), []byte(value), implicit, implicit, style)
	event.HeadComment = head
	event.LineComment = line
	event.FootComment = foot
	event.TailComment = tail
	e.emit(event)
}

func (e *Encoder) nodev(in reflect.Value) {
	e.node(in.Interface().(*Node), "")
}

func (e *Encoder) node(node *Node, tail string) {
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
			// It can't be encoded directly as YAML so use a binary tag
			// and encode it as base64.
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
		failf("cannot encode node with unknown kind %d", node.Kind)
	}
}
