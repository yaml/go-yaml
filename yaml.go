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

// Package yaml implements YAML support for the Go language.
//
// Source code and other details for the project are available at GitHub:
//
//	https://github.com/yaml/go-yaml
package yaml

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"go.yaml.in/yaml/v4/internal/libyaml"
)

var noWriter io.Writer

// Re-export types from internal/libyaml
type (
	Node      = libyaml.Node
	Kind      = libyaml.Kind
	Style     = libyaml.Style
	Marshaler = libyaml.Marshaler
	IsZeroer  = libyaml.IsZeroer
)

// Unmarshaler is the interface implemented by types
// that can unmarshal a YAML description of themselves.
type Unmarshaler interface {
	UnmarshalYAML(node *Node) error
}

// Re-export error types
type (
	UnmarshalError = libyaml.UnmarshalError
	TypeError      = libyaml.TypeError
)

// Re-export Kind constants
const (
	DocumentNode = libyaml.DocumentNode
	SequenceNode = libyaml.SequenceNode
	MappingNode  = libyaml.MappingNode
	ScalarNode   = libyaml.ScalarNode
	AliasNode    = libyaml.AliasNode
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

// LineBreak represents the line ending style for YAML output.
type LineBreak = libyaml.LineBreak

// Line break constants for different platforms.
const (
	LineBreakLN   = libyaml.LN_BREAK   // Unix-style \n (default)
	LineBreakCR   = libyaml.CR_BREAK   // Old Mac-style \r
	LineBreakCRLN = libyaml.CRLN_BREAK // Windows-style \r\n
)

//-----------------------------------------------------------------------------
// Load / Dump API
//-----------------------------------------------------------------------------

// Load decodes the first YAML document with the given options.
//
// Maps and pointers (to a struct, string, int, etc) are accepted as out
// values. If an internal pointer within a struct is not initialized,
// the yaml package will initialize it if necessary. The out parameter
// must not be nil.
//
// The type of the decoded values should be compatible with the respective
// values in out. If one or more values cannot be decoded due to type
// mismatches, decoding continues partially until the end of the YAML
// content, and a *yaml.TypeError is returned with details for all
// missed values.
//
// Struct fields are only loaded if they are exported (have an upper case
// first letter), and are loaded using the field name lowercased as the
// default key. Custom keys may be defined via the "yaml" name in the field
// tag: the content preceding the first comma is used as the key, and the
// following comma-separated options control the loading and dumping behavior.
//
// For example:
//
//	type T struct {
//	    F int `yaml:"a,omitempty"`
//	    B int
//	}
//	var t T
//	yaml.Load([]byte("a: 1\nb: 2"), &t)
//
// See the documentation of Dump for the format of tags and a list of
// supported tag options.
func Load(in []byte, out any, opts ...Option) error {
	return unmarshal(in, out, opts...)
}

// LoadAll decodes all YAML documents from the input.
//
// Returns a slice containing all decoded documents. Each document is
// decoded into an any value (typically map[string]any or []any).
//
// See [Unmarshal] for details about the conversion of YAML into Go values.
func LoadAll(in []byte, opts ...Option) ([]any, error) {
	l, err := NewLoader(bytes.NewReader(in), opts...)
	if err != nil {
		return nil, err
	}
	var docs []any
	for {
		var doc any
		err := l.Load(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return docs, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// A Loader reads and decodes YAML values from an input stream with configurable
// options.
type Loader struct {
	composer *libyaml.Composer
	decoder  *libyaml.Decoder
	opts     *libyaml.Options
	docCount int
}

// NewLoader returns a new Loader that reads from r with the given options.
//
// The Loader introduces its own buffering and may read data from r beyond the
// YAML values requested.
func NewLoader(r io.Reader, opts ...Option) (*Loader, error) {
	o, err := libyaml.ApplyOptions(opts...)
	if err != nil {
		return nil, err
	}
	return &Loader{
		composer: libyaml.NewComposerFromReader(r),
		decoder:  libyaml.NewDecoder(o),
		opts:     o,
	}, nil
}

// Load reads the next YAML-encoded document from its input and stores it
// in the value pointed to by v.
//
// Returns io.EOF when there are no more documents to read.
// If WithSingleDocument option was set and a document was already read,
// subsequent calls return io.EOF.
//
// Maps and pointers (to a struct, string, int, etc) are accepted as v
// values. If an internal pointer within a struct is not initialized,
// the yaml package will initialize it if necessary. The v parameter
// must not be nil.
//
// Struct fields are only loaded if they are exported (have an upper case
// first letter), and are loaded using the field name lowercased as the
// default key. Custom keys may be defined via the "yaml" name in the field
// tag: the content preceding the first comma is used as the key, and the
// following comma-separated options control the loading and dumping behavior.
//
// See the documentation of the package-level Load function for more details
// about YAML to Go conversion and tag options.
func (l *Loader) Load(v any) (err error) {
	defer handleErr(&err)
	if l.opts.SingleDocument && l.docCount > 0 {
		return io.EOF
	}
	node := l.composer.Parse() // *libyaml.Node
	if node == nil {
		return io.EOF
	}
	l.docCount++

	out := reflect.ValueOf(v)
	if out.Kind() == reflect.Pointer && !out.IsNil() {
		out = out.Elem()
	}
	l.decoder.Unmarshal(node, out) // Pass libyaml.Node directly
	if len(l.decoder.Terrors) > 0 {
		terrors := l.decoder.Terrors
		l.decoder.Terrors = nil
		return &TypeError{Errors: terrors}
	}
	return nil
}

// Dump encodes a value to YAML with the given options.
//
// See [Marshal] for details about the conversion of Go values to YAML.
func Dump(in any, opts ...Option) (out []byte, err error) {
	defer handleErr(&err)
	var buf bytes.Buffer
	d, err := NewDumper(&buf, opts...)
	if err != nil {
		return nil, err
	}
	if err := d.Dump(in); err != nil {
		return nil, err
	}
	if err := d.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DumpAll encodes multiple values as a multi-document YAML stream.
//
// Each value becomes a separate YAML document, separated by "---".
// See [Marshal] for details about the conversion of Go values to YAML.
func DumpAll(in []any, opts ...Option) (out []byte, err error) {
	defer handleErr(&err)
	var buf bytes.Buffer
	d, err := NewDumper(&buf, opts...)
	if err != nil {
		return nil, err
	}
	for _, v := range in {
		if err := d.Dump(v); err != nil {
			return nil, err
		}
	}
	if err := d.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// A Dumper writes YAML values to an output stream with configurable options.
type Dumper struct {
	encoder *libyaml.Encoder
	opts    *libyaml.Options
}

// NewDumper returns a new Dumper that writes to w with the given options.
//
// The Dumper should be closed after use to flush all data to w.
func NewDumper(w io.Writer, opts ...Option) (*Dumper, error) {
	o, err := libyaml.ApplyOptions(opts...)
	if err != nil {
		return nil, err
	}
	return &Dumper{
		encoder: libyaml.NewEncoder(w, o),
		opts:    o,
	}, nil
}

// Dump writes the YAML encoding of v to the stream.
//
// If multiple values are dumped to the stream, the second and subsequent
// documents will be preceded with a "---" document separator.
//
// See the documentation for [Marshal] for details about the conversion of Go
// values to YAML.
func (d *Dumper) Dump(v any) (err error) {
	defer handleErr(&err)
	d.encoder.MarshalDoc("", reflect.ValueOf(v))
	return nil
}

// Close closes the Dumper by writing any remaining data.
// It does not write a stream terminating string "...".
func (d *Dumper) Close() (err error) {
	defer handleErr(&err)
	d.encoder.Finish()
	return nil
}

//-----------------------------------------------------------------------------
// Decode / Encode API
//-----------------------------------------------------------------------------

// A Decoder reads and decodes YAML values from an input stream.
//
// Deprecated: Use Loader instead. Will be removed in v5.
type Decoder struct {
	composer    *libyaml.Composer
	knownFields bool
}

// NewDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may read
// data from r beyond the YAML values requested.
//
// Deprecated: Use NewLoader instead. Will be removed in v5.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		composer: libyaml.NewComposerFromReader(r),
	}
}

// KnownFields ensures that the keys in decoded mappings to
// exist as fields in the struct being decoded into.
//
// Deprecated: Use NewLoader with WithKnownFields option instead.
// Will be removed in v5.
func (dec *Decoder) KnownFields(enable bool) {
	dec.knownFields = enable
}

// Decode reads the next YAML-encoded value from its input
// and stores it in the value pointed to by v.
//
// See the documentation for Unmarshal for details about the
// conversion of YAML into a Go value.
//
// Deprecated: Use Loader.Load instead. Will be removed in v5.
func (dec *Decoder) Decode(v any) (err error) {
	d := libyaml.NewDecoder(libyaml.LegacyOptions)
	d.KnownFields = dec.knownFields
	defer handleErr(&err)
	node := dec.composer.Parse()
	if node == nil {
		return io.EOF
	}
	out := reflect.ValueOf(v)
	if out.Kind() == reflect.Pointer && !out.IsNil() {
		out = out.Elem()
	}
	d.Unmarshal(node, out)
	if len(d.Terrors) > 0 {
		return &TypeError{Errors: d.Terrors}
	}
	return nil
}

// An Encoder writes YAML values to an output stream.
//
// Deprecated: Use Dumper instead. Will be removed in v5.
type Encoder struct {
	encoder *libyaml.Encoder
}

// NewEncoder returns a new encoder that writes to w.
// The Encoder should be closed after use to flush all data
// to w.
//
// Deprecated: Use NewDumper instead. Will be removed in v5.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		encoder: libyaml.NewEncoder(w, libyaml.LegacyOptions),
	}
}

// Encode writes the YAML encoding of v to the stream.
// If multiple items are encoded to the stream, the
// second and subsequent document will be preceded
// with a "---" document separator, but the first will not.
//
// See the documentation for Marshal for details about the conversion of Go
// values to YAML.
//
// Deprecated: Use Dumper.Dump instead. Will be removed in v5.
func (e *Encoder) Encode(v any) (err error) {
	defer handleErr(&err)
	e.encoder.MarshalDoc("", reflect.ValueOf(v))
	return nil
}

// SetIndent changes the used indentation used when encoding.
//
// Deprecated: Use NewDumper with WithIndent option instead. Will be removed in v5.
func (e *Encoder) SetIndent(spaces int) {
	if spaces < 0 {
		panic("yaml: cannot indent to a negative number of spaces")
	}
	e.encoder.Indent = spaces
}

// CompactSeqIndent makes it so that '- ' is considered part of the indentation.
//
// Deprecated: Use NewDumper with WithCompactSeqIndent option instead. Will be removed in v5.
func (e *Encoder) CompactSeqIndent() {
	e.encoder.Emitter.CompactSequenceIndent = true
}

// DefaultSeqIndent makes it so that '- ' is not considered part of the indentation.
//
// Deprecated: This is the default behavior for Dumper. Will be removed in v5.
func (e *Encoder) DefaultSeqIndent() {
	e.encoder.Emitter.CompactSequenceIndent = false
}

// Close closes the encoder by writing any remaining data.
// It does not write a stream terminating string "...".
//
// Deprecated: Use Dumper.Close instead. Will be removed in v5.
func (e *Encoder) Close() (err error) {
	defer handleErr(&err)
	e.encoder.Finish()
	return nil
}

//-----------------------------------------------------------------------------
// Unmarshal / Marshal API
//-----------------------------------------------------------------------------

// Unmarshal decodes the first document found within the in byte slice
// and assigns decoded values into the out value.
//
// Maps and pointers (to a struct, string, int, etc) are accepted as out
// values. If an internal pointer within a struct is not initialized,
// the yaml package will initialize it if necessary for unmarshalling
// the provided data. The out parameter must not be nil.
//
// The type of the decoded values should be compatible with the respective
// values in out. If one or more values cannot be decoded due to a type
// mismatches, decoding continues partially until the end of the YAML
// content, and a *yaml.TypeError is returned with details for all
// missed values.
//
// Struct fields are only unmarshalled if they are exported (have an
// upper case first letter), and are unmarshalled using the field name
// lowercased as the default key. Custom keys may be defined via the
// "yaml" name in the field tag: the content preceding the first comma
// is used as the key, and the following comma-separated options are
// used to tweak the marshaling process (see Marshal).
// Conflicting names result in a runtime error.
//
// For example:
//
//	type T struct {
//	    F int `yaml:"a,omitempty"`
//	    B int
//	}
//	var t T
//	yaml.Unmarshal([]byte("a: 1\nb: 2"), &t)
//
// See the documentation of Marshal for the format of tags and a list of
// supported tag options.
//
// Deprecated: Use Load instead. Will be removed in v5.
func Unmarshal(in []byte, out any) (err error) {
	return unmarshal(in, out, V3)
}

func unmarshal(in []byte, out any, opts ...Option) (err error) {
	defer handleErr(&err)
	o, err := libyaml.ApplyOptions(opts...)
	if err != nil {
		return err
	}

	// Check if out implements yaml.Unmarshaler
	if u, ok := out.(Unmarshaler); ok {
		p := libyaml.NewComposer(in)
		defer p.Destroy()
		node := p.Parse()
		if node != nil {
			return u.UnmarshalYAML(node)
		}
		return nil
	}

	d := libyaml.NewDecoder(o)
	p := libyaml.NewComposer(in)
	defer p.Destroy()
	node := p.Parse()
	if node != nil {
		v := reflect.ValueOf(out)
		if v.Kind() == reflect.Pointer && !v.IsNil() {
			v = v.Elem()
		}
		d.Unmarshal(node, v)
	}
	if len(d.Terrors) > 0 {
		return &TypeError{Errors: d.Terrors}
	}
	return nil
}

// Marshal serializes the value provided into a YAML document. The structure
// of the generated document will reflect the structure of the value itself.
// Maps and pointers (to struct, string, int, etc) are accepted as the in value.
//
// Struct fields are only marshaled if they are exported (have an upper case
// first letter), and are marshaled using the field name lowercased as the
// default key. Custom keys may be defined via the "yaml" name in the field
// tag: the content preceding the first comma is used as the key, and the
// following comma-separated options are used to tweak the marshaling process.
// Conflicting names result in a runtime error.
//
// The field tag format accepted is:
//
//	`(...) yaml:"[<key>][,<flag1>[,<flag2>]]" (...)`
//
// The following flags are currently supported:
//
//	omitempty    Only include the field if it's not set to the zero
//	             value for the type or to empty slices or maps.
//	             Zero valued structs will be omitted if all their public
//	             fields are zero, unless they implement an IsZero
//	             method (see the IsZeroer interface type), in which
//	             case the field will be excluded if IsZero returns true.
//
//	flow         Marshal using a flow style (useful for structs,
//	             sequences and maps).
//
//	inline       Inline the field, which must be a struct or a map,
//	             causing all of its fields or keys to be processed as if
//	             they were part of the outer struct. For maps, keys must
//	             not conflict with the yaml keys of other struct fields.
//	             See doc/inline-tags.md for detailed examples and use cases.
//
// In addition, if the key is "-", the field is ignored.
//
// For example:
//
//	type T struct {
//	    F int `yaml:"a,omitempty"`
//	    B int
//	}
//	yaml.Marshal(&T{B: 2}) // Returns "b: 2\n"
//	yaml.Marshal(&T{F: 1}} // Returns "a: 1\nb: 0\n"
//
// Deprecated: Use Dump instead. Will be removed in v5.
func Marshal(in any) (out []byte, err error) {
	defer handleErr(&err)
	e := libyaml.NewEncoder(noWriter, libyaml.LegacyOptions)
	defer e.Destroy()
	e.MarshalDoc("", reflect.ValueOf(in))
	e.Finish()
	out = e.Out
	return out, err
}

//-----------------------------------------------------------------------------
// Helper functions
//-----------------------------------------------------------------------------

// The code in this section was copied from mgo/bson.

var (
	structMap       = make(map[reflect.Type]*structInfo)
	fieldMapMutex   sync.RWMutex
	unmarshalerType reflect.Type
)

// structInfo holds details for the serialization of fields of
// a given struct.
type structInfo struct {
	FieldsMap  map[string]fieldInfo
	FieldsList []fieldInfo

	// InlineMap is the number of the field in the struct that
	// contains an ,inline map, or -1 if there's none.
	InlineMap int

	// InlineUnmarshalers holds indexes to inlined fields that
	// contain unmarshaler values.
	InlineUnmarshalers [][]int
}

type fieldInfo struct {
	Key       string
	Num       int
	OmitEmpty bool
	Flow      bool
	// Id holds the unique field identifier, so we can cheaply
	// check for field duplicates without maintaining an extra map.
	Id int

	// Inline holds the field index if the field is part of an inlined struct.
	Inline []int
}

func getStructInfo(st reflect.Type) (*structInfo, error) {
	fieldMapMutex.RLock()
	sinfo, found := structMap[st]
	fieldMapMutex.RUnlock()
	if found {
		return sinfo, nil
	}

	n := st.NumField()
	fieldsMap := make(map[string]fieldInfo)
	fieldsList := make([]fieldInfo, 0, n)
	inlineMap := -1
	inlineUnmarshalers := [][]int(nil)
	for i := 0; i != n; i++ {
		field := st.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue // Private field
		}

		info := fieldInfo{Num: i}

		tag := field.Tag.Get("yaml")
		if tag == "" && !strings.Contains(string(field.Tag), ":") {
			tag = string(field.Tag)
		}
		if tag == "-" {
			continue
		}

		inline := false
		fields := strings.Split(tag, ",")
		if len(fields) > 1 {
			for _, flag := range fields[1:] {
				switch flag {
				case "omitempty":
					info.OmitEmpty = true
				case "flow":
					info.Flow = true
				case "inline":
					inline = true
				default:
					return nil, fmt.Errorf("unsupported flag %q in tag %q of type %s", flag, tag, st)
				}
			}
			tag = fields[0]
		}

		if inline {
			switch field.Type.Kind() {
			case reflect.Map:
				if inlineMap >= 0 {
					return nil, errors.New("multiple ,inline maps in struct " + st.String())
				}
				if field.Type.Key() != reflect.TypeOf("") {
					return nil, errors.New("option ,inline needs a map with string keys in struct " + st.String())
				}
				inlineMap = info.Num
			case reflect.Struct, reflect.Pointer:
				ftype := field.Type
				for ftype.Kind() == reflect.Pointer {
					ftype = ftype.Elem()
				}
				if ftype.Kind() != reflect.Struct {
					return nil, errors.New("option ,inline may only be used on a struct or map field")
				}
				if reflect.PointerTo(ftype).Implements(unmarshalerType) {
					inlineUnmarshalers = append(inlineUnmarshalers, []int{i})
				} else {
					sinfo, err := getStructInfo(ftype)
					if err != nil {
						return nil, err
					}
					for _, index := range sinfo.InlineUnmarshalers {
						inlineUnmarshalers = append(inlineUnmarshalers, append([]int{i}, index...))
					}
					for _, finfo := range sinfo.FieldsList {
						if _, found := fieldsMap[finfo.Key]; found {
							msg := "duplicated key '" + finfo.Key + "' in struct " + st.String()
							return nil, errors.New(msg)
						}
						if finfo.Inline == nil {
							finfo.Inline = []int{i, finfo.Num}
						} else {
							finfo.Inline = append([]int{i}, finfo.Inline...)
						}
						finfo.Id = len(fieldsList)
						fieldsMap[finfo.Key] = finfo
						fieldsList = append(fieldsList, finfo)
					}
				}
			default:
				return nil, errors.New("option ,inline may only be used on a struct or map field")
			}
			continue
		}

		if tag != "" {
			info.Key = tag
		} else {
			info.Key = strings.ToLower(field.Name)
		}

		if _, found = fieldsMap[info.Key]; found {
			msg := "duplicated key '" + info.Key + "' in struct " + st.String()
			return nil, errors.New(msg)
		}

		info.Id = len(fieldsList)
		fieldsList = append(fieldsList, info)
		fieldsMap[info.Key] = info
	}

	sinfo = &structInfo{
		FieldsMap:          fieldsMap,
		FieldsList:         fieldsList,
		InlineMap:          inlineMap,
		InlineUnmarshalers: inlineUnmarshalers,
	}

	fieldMapMutex.Lock()
	structMap[st] = sinfo
	fieldMapMutex.Unlock()
	return sinfo, nil
}

//-----------------------------------------------------------------------------
// Error function
//-----------------------------------------------------------------------------

func handleErr(err *error) {
	if v := recover(); v != nil {
		if e, ok := v.(*libyaml.YAMLError); ok {
			*err = e.Err
		} else {
			panic(v)
		}
	}
}
