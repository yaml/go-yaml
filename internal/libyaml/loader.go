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
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"time"
)

// --------------------------------------------------------------------------
// Interfaces and types needed by decoder

// Unmarshaler interface may be implemented by types to customize their
// behavior when being unmarshaled from a YAML document.
type Unmarshaler interface {
	UnmarshalYAML(value *Node) error
}

type obsoleteUnmarshaler interface {
	UnmarshalYAML(unmarshal func(any) error) error
}

// Marshaler interface may be implemented by types to customize their
// behavior when being marshaled into a YAML document.
type Marshaler interface {
	MarshalYAML() (any, error)
}

// IsZeroer is used to check whether an object is zero to determine whether
// it should be omitted when marshaling with the ,omitempty flag. One notable
// implementation is time.Time.
type IsZeroer interface {
	IsZero() bool
}

// UnmarshalError represents a single, non-fatal error that occurred during
// the unmarshaling of a YAML document into a Go value.
type UnmarshalError struct {
	Err    error
	Line   int
	Column int
}

func (e *UnmarshalError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Err.Error())
}

func (e *UnmarshalError) Unwrap() error {
	return e.Err
}

// TypeError is returned when one or more fields cannot be properly decoded.
type TypeError struct {
	Errors []*UnmarshalError
}

func (e *TypeError) Error() string {
	var b strings.Builder
	b.WriteString("yaml: unmarshal errors:")
	for _, err := range e.Errors {
		b.WriteString("\n  ")
		b.WriteString(err.Error())
	}
	return b.String()
}

// Unwrap returns all errors for compatibility with errors.As/Is.
// This allows callers to unwrap TypeError and examine individual UnmarshalErrors.
// Implements the Go 1.20+ multiple error unwrapping interface.
func (e *TypeError) Unwrap() []error {
	errs := make([]error, len(e.Errors))
	for i, err := range e.Errors {
		errs[i] = err
	}
	return errs
}

// handleErr recovers from panics caused by yaml errors
func handleErr(err *error) {
	if v := recover(); v != nil {
		if e, ok := v.(*YAMLError); ok {
			*err = e.Err
		} else {
			panic(v)
		}
	}
}

// --------------------------------------------------------------------------
// Struct field information

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

var (
	structMap       = make(map[reflect.Type]*structInfo)
	fieldMapMutex   sync.RWMutex
	unmarshalerType reflect.Type
)

func init() {
	var v Unmarshaler
	unmarshalerType = reflect.ValueOf(&v).Elem().Type()
}

// hasUnmarshalYAMLMethod checks if a type has an UnmarshalYAML method
// that looks like it implements yaml.Unmarshaler (from root package).
// This is needed because we can't directly check for the interface type
// since it's in a different package that we can't import.
func hasUnmarshalYAMLMethod(t reflect.Type) bool {
	method, found := t.MethodByName("UnmarshalYAML")
	if !found {
		return false
	}

	// Check signature: func(*T) UnmarshalYAML(*Node) error
	mtype := method.Type
	if mtype.NumIn() != 2 || mtype.NumOut() != 1 {
		return false
	}

	// First param is receiver (already checked by MethodByName)
	// Second param should be a pointer to a Node-like struct
	paramType := mtype.In(1)
	if paramType.Kind() != reflect.Ptr {
		return false
	}

	elemType := paramType.Elem()
	if elemType.Kind() != reflect.Struct || elemType.Name() != "Node" {
		return false
	}

	// Return type should be error
	retType := mtype.Out(0)
	if retType.Kind() != reflect.Interface || retType.Name() != "error" {
		return false
	}

	return true
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
				// Check for both libyaml.Unmarshaler and yaml.Unmarshaler (by method name)
				if reflect.PointerTo(ftype).Implements(unmarshalerType) || hasUnmarshalYAMLMethod(reflect.PointerTo(ftype)) {
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

// IsZeroer is used to check whether an object is zero to
// determine whether it should be omitted when marshaling
// with the omitempty flag. One notable implementation
// is time.Time.
func isZero(v reflect.Value) bool {
	kind := v.Kind()
	if z, ok := v.Interface().(IsZeroer); ok {
		if (kind == reflect.Pointer || kind == reflect.Interface) && v.IsNil() {
			return true
		}
		return z.IsZero()
	}
	switch kind {
	case reflect.String:
		return len(v.String()) == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	case reflect.Slice:
		return v.Len() == 0
	case reflect.Map:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Struct:
		vt := v.Type()
		for i := v.NumField() - 1; i >= 0; i-- {
			if vt.Field(i).PkgPath != "" {
				continue // Private field
			}
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	}
	return false
}

type Decoder struct {
	doc     *Node
	aliases map[*Node]bool
	Terrors []*UnmarshalError

	stringMapType  reflect.Type
	generalMapType reflect.Type

	KnownFields bool
	UniqueKeys  bool
	decodeCount int
	aliasCount  int
	aliasDepth  int

	mergedFields map[any]bool
}

var (
	nodeType       = reflect.TypeOf(Node{})
	durationType   = reflect.TypeOf(time.Duration(0))
	stringMapType  = reflect.TypeOf(map[string]any{})
	generalMapType = reflect.TypeOf(map[any]any{})
	ifaceType      = generalMapType.Elem()
)

func NewDecoder(opts *Options) *Decoder {
	return &Decoder{
		stringMapType:  stringMapType,
		generalMapType: generalMapType,
		KnownFields:    opts.KnownFields,
		UniqueKeys:     opts.UniqueKeys,
		aliases:        make(map[*Node]bool),
	}
}

// Unmarshal decodes YAML input into the provided output value.
// The out parameter must be a pointer to the value to decode into.
// Returns a TypeError if type mismatches occur during decoding.
func Unmarshal(in []byte, out any, opts *Options) error {
	d := NewDecoder(opts)
	p := NewComposer(in)
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

func (d *Decoder) terror(n *Node, tag string, out reflect.Value) {
	if n.Tag != "" {
		tag = n.Tag
	}
	value := n.Value
	if tag != seqTag && tag != mapTag {
		if len(value) > 10 {
			value = " `" + value[:7] + "...`"
		} else {
			value = " `" + value + "`"
		}
	}
	d.Terrors = append(d.Terrors, &UnmarshalError{
		Err:    fmt.Errorf("cannot unmarshal %s%s into %s", shortTag(tag), value, out.Type()),
		Line:   n.Line,
		Column: n.Column,
	})
}

func (d *Decoder) callUnmarshaler(n *Node, u Unmarshaler) (good bool) {
	err := u.UnmarshalYAML(n)
	switch e := err.(type) {
	case nil:
		return true
	case *TypeError:
		d.Terrors = append(d.Terrors, e.Errors...)
		return false
	default:
		d.Terrors = append(d.Terrors, &UnmarshalError{
			Err:    err,
			Line:   n.Line,
			Column: n.Column,
		})
		return false
	}
}

func (d *Decoder) callObsoleteUnmarshaler(n *Node, u obsoleteUnmarshaler) (good bool) {
	terrlen := len(d.Terrors)
	err := u.UnmarshalYAML(func(v any) (err error) {
		defer handleErr(&err)
		d.Unmarshal(n, reflect.ValueOf(v))
		if len(d.Terrors) > terrlen {
			issues := d.Terrors[terrlen:]
			d.Terrors = d.Terrors[:terrlen]
			return &TypeError{issues}
		}
		return nil
	})
	switch e := err.(type) {
	case nil:
		return true
	case *TypeError:
		d.Terrors = append(d.Terrors, e.Errors...)
		return false
	default:
		d.Terrors = append(d.Terrors, &UnmarshalError{
			Err:    err,
			Line:   n.Line,
			Column: n.Column,
		})
		return false
	}
}

// d.prepare initializes and dereferences pointers and calls UnmarshalYAML
// if a value is found to implement it.
// It returns the initialized and dereferenced out value, whether
// unmarshalling was already done by UnmarshalYAML, and if so whether
// its types unmarshalled appropriately.
//
// If n holds a null value, prepare returns before doing anything.
func (d *Decoder) prepare(n *Node, out reflect.Value) (newout reflect.Value, unmarshaled, good bool) {
	if n.ShortTag() == nullTag {
		return out, false, false
	}
	again := true
	for again {
		again = false
		if out.Kind() == reflect.Pointer {
			if out.IsNil() {
				out.Set(reflect.New(out.Type().Elem()))
			}
			out = out.Elem()
			again = true
		}
		if out.CanAddr() {
			// Try yaml.Unmarshaler (from root package) first
			if called, good := d.tryCallYAMLUnmarshaler(n, out); called {
				return out, true, good
			}

			outi := out.Addr().Interface()
			// Check for libyaml.Unmarshaler
			if u, ok := outi.(Unmarshaler); ok {
				good = d.callUnmarshaler(n, u)
				return out, true, good
			}
			if u, ok := outi.(obsoleteUnmarshaler); ok {
				good = d.callObsoleteUnmarshaler(n, u)
				return out, true, good
			}
		}
	}
	return out, false, false
}

func (d *Decoder) fieldByIndex(n *Node, v reflect.Value, index []int) (field reflect.Value) {
	if n.ShortTag() == nullTag {
		return reflect.Value{}
	}
	for _, num := range index {
		for {
			if v.Kind() == reflect.Pointer {
				if v.IsNil() {
					v.Set(reflect.New(v.Type().Elem()))
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

const (
	// 400,000 decode operations is ~500kb of dense object declarations, or
	// ~5kb of dense object declarations with 10000% alias expansion
	alias_ratio_range_low = 400000

	// 4,000,000 decode operations is ~5MB of dense object declarations, or
	// ~4.5MB of dense object declarations with 10% alias expansion
	alias_ratio_range_high = 4000000

	// alias_ratio_range is the range over which we scale allowed alias ratios
	alias_ratio_range = float64(alias_ratio_range_high - alias_ratio_range_low)
)

func allowedAliasRatio(decodeCount int) float64 {
	switch {
	case decodeCount <= alias_ratio_range_low:
		// allow 99% to come from alias expansion for small-to-medium documents
		return 0.99
	case decodeCount >= alias_ratio_range_high:
		// allow 10% to come from alias expansion for very large documents
		return 0.10
	default:
		// scale smoothly from 99% down to 10% over the range.
		// this maps to 396,000 - 400,000 allowed alias-driven decodes over the range.
		// 400,000 decode operations is ~100MB of allocations in worst-case scenarios (single-item maps).
		return 0.99 - 0.89*(float64(decodeCount-alias_ratio_range_low)/alias_ratio_range)
	}
}

// unmarshalerAdapter is an interface that wraps the root package's Unmarshaler interface.
// This allows the decoder to call unmarshalers that expect *yaml.Node instead of *libyaml.Node.
type unmarshalerAdapter interface {
	CallRootUnmarshaler(n *Node) error
}

// tryCallYAMLUnmarshaler checks if the value has an UnmarshalYAML method that takes
// a *yaml.Node (from the root package) and calls it if found.
// This handles the case where user types implement yaml.Unmarshaler instead of libyaml.Unmarshaler.
func (d *Decoder) tryCallYAMLUnmarshaler(n *Node, out reflect.Value) (called bool, good bool) {
	if !out.CanAddr() {
		return false, false
	}

	addr := out.Addr()
	// Check for UnmarshalYAML method
	method := addr.MethodByName("UnmarshalYAML")
	if !method.IsValid() {
		return false, false
	}

	// Check method signature: func(*yaml.Node) error
	mtype := method.Type()
	if mtype.NumIn() != 1 || mtype.NumOut() != 1 {
		return false, false
	}

	// Check if parameter is a pointer to a Node-like struct
	paramType := mtype.In(0)
	if paramType.Kind() != reflect.Ptr {
		return false, false
	}

	elemType := paramType.Elem()
	if elemType.Kind() != reflect.Struct {
		return false, false
	}

	// Check if it's the same underlying type as our Node
	// Both yaml.Node and libyaml.Node have the same structure
	if elemType.Name() != "Node" {
		return false, false
	}

	// Call the method with a converted node
	// Since yaml.Node and libyaml.Node have the same structure,
	// we can convert using unsafe pointer cast
	nodeValue := reflect.NewAt(elemType, reflect.ValueOf(n).UnsafePointer())

	results := method.Call([]reflect.Value{nodeValue})
	err := results[0].Interface()

	if err == nil {
		return true, true
	}

	switch e := err.(type) {
	case *TypeError:
		d.Terrors = append(d.Terrors, e.Errors...)
		return true, false
	default:
		d.Terrors = append(d.Terrors, &UnmarshalError{
			Err:    e.(error),
			Line:   n.Line,
			Column: n.Column,
		})
		return true, false
	}
}

func (d *Decoder) Unmarshal(n *Node, out reflect.Value) (good bool) {
	d.decodeCount++
	if d.aliasDepth > 0 {
		d.aliasCount++
	}
	if d.aliasCount > 100 && d.decodeCount > 1000 && float64(d.aliasCount)/float64(d.decodeCount) > allowedAliasRatio(d.decodeCount) {
		failf("document contains excessive aliasing")
	}
	if out.Type() == nodeType {
		out.Set(reflect.ValueOf(n).Elem())
		return true
	}
	switch n.Kind {
	case DocumentNode:
		return d.document(n, out)
	case AliasNode:
		return d.alias(n, out)
	}
	out, unmarshaled, good := d.prepare(n, out)
	if unmarshaled {
		return good
	}
	switch n.Kind {
	case ScalarNode:
		good = d.scalar(n, out)
	case MappingNode:
		good = d.mapping(n, out)
	case SequenceNode:
		good = d.sequence(n, out)
	case 0:
		if n.IsZero() {
			return d.null(out)
		}
		fallthrough
	default:
		failf("cannot decode node with unknown kind %d", n.Kind)
	}
	return good
}

func (d *Decoder) document(n *Node, out reflect.Value) (good bool) {
	if len(n.Content) == 1 {
		d.doc = n
		d.Unmarshal(n.Content[0], out)
		return true
	}
	return false
}

func (d *Decoder) alias(n *Node, out reflect.Value) (good bool) {
	if d.aliases[n] {
		// TODO this could actually be allowed in some circumstances.
		failf("anchor '%s' value contains itself", n.Value)
	}
	d.aliases[n] = true
	d.aliasDepth++
	good = d.Unmarshal(n.Alias, out)
	d.aliasDepth--
	delete(d.aliases, n)
	return good
}

func (d *Decoder) null(out reflect.Value) bool {
	if out.CanAddr() {
		switch out.Kind() {
		case reflect.Interface, reflect.Pointer, reflect.Map, reflect.Slice:
			out.Set(reflect.Zero(out.Type()))
			return true
		}
	}
	return false
}

func (d *Decoder) scalar(n *Node, out reflect.Value) bool {
	var tag string
	var resolved any
	if n.indicatedString() {
		tag = strTag
		resolved = n.Value
	} else {
		tag, resolved = resolve(n.Tag, n.Value)
		if tag == binaryTag {
			data, err := base64.StdEncoding.DecodeString(resolved.(string))
			if err != nil {
				failf("!!binary value contains invalid base64 data")
			}
			resolved = string(data)
		}
	}
	if resolved == nil {
		return d.null(out)
	}
	if resolvedv := reflect.ValueOf(resolved); out.Type() == resolvedv.Type() {
		// We've resolved to exactly the type we want, so use that.
		out.Set(resolvedv)
		return true
	}
	// Perhaps we can use the value as a TextUnmarshaler to
	// set its value.
	if out.CanAddr() {
		u, ok := out.Addr().Interface().(encoding.TextUnmarshaler)
		if ok {
			var text []byte
			if tag == binaryTag {
				text = []byte(resolved.(string))
			} else {
				// We let any value be unmarshaled into TextUnmarshaler.
				// That might be more lax than we'd like, but the
				// TextUnmarshaler itself should bowl out any dubious values.
				text = []byte(n.Value)
			}
			err := u.UnmarshalText(text)
			if err != nil {
				d.Terrors = append(d.Terrors, &UnmarshalError{
					Err:    err,
					Line:   n.Line,
					Column: n.Column,
				})
				return false
			}
			return true
		}
	}
	switch out.Kind() {
	case reflect.String:
		if tag == binaryTag {
			out.SetString(resolved.(string))
			return true
		}
		out.SetString(n.Value)
		return true
	case reflect.Slice:
		// allow decoding !!binary-tagged value into []byte specifically
		if out.Type().Elem().Kind() == reflect.Uint8 {
			if tag == binaryTag {
				out.SetBytes([]byte(resolved.(string)))
				return true
			}
		}
	case reflect.Interface:
		out.Set(reflect.ValueOf(resolved))
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// This used to work in v2, but it's very unfriendly.
		isDuration := out.Type() == durationType

		switch resolved := resolved.(type) {
		case int:
			if !isDuration && !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				return true
			} else if isDuration && resolved == 0 {
				out.SetInt(0)
				return true
			}
		case int64:
			if !isDuration && !out.OverflowInt(resolved) {
				out.SetInt(resolved)
				return true
			}
		case uint64:
			if !isDuration && resolved <= math.MaxInt64 && !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				return true
			}
		case float64:
			if !isDuration && resolved <= math.MaxInt64 && !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				return true
			}
		case string:
			if out.Type() == durationType {
				d, err := time.ParseDuration(resolved)
				if err == nil {
					out.SetInt(int64(d))
					return true
				}
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		switch resolved := resolved.(type) {
		case int:
			if resolved >= 0 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				return true
			}
		case int64:
			if resolved >= 0 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				return true
			}
		case uint64:
			if !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				return true
			}
		case float64:
			if resolved <= math.MaxUint64 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				return true
			}
		}
	case reflect.Bool:
		switch resolved := resolved.(type) {
		case bool:
			out.SetBool(resolved)
			return true
		case string:
			// This offers some compatibility with the 1.1 spec (https://yaml.org/type/bool.html).
			// It only works if explicitly attempting to unmarshal into a typed bool value.
			switch resolved {
			case "y", "Y", "yes", "Yes", "YES", "on", "On", "ON":
				out.SetBool(true)
				return true
			case "n", "N", "no", "No", "NO", "off", "Off", "OFF":
				out.SetBool(false)
				return true
			}
		}
	case reflect.Float32, reflect.Float64:
		switch resolved := resolved.(type) {
		case int:
			out.SetFloat(float64(resolved))
			return true
		case int64:
			out.SetFloat(float64(resolved))
			return true
		case uint64:
			out.SetFloat(float64(resolved))
			return true
		case float64:
			out.SetFloat(resolved)
			return true
		}
	case reflect.Struct:
		if resolvedv := reflect.ValueOf(resolved); out.Type() == resolvedv.Type() {
			out.Set(resolvedv)
			return true
		}
	case reflect.Pointer:
		panic("yaml internal error: please report the issue")
	}
	d.terror(n, tag, out)
	return false
}

func settableValueOf(i any) reflect.Value {
	v := reflect.ValueOf(i)
	sv := reflect.New(v.Type()).Elem()
	sv.Set(v)
	return sv
}

func (d *Decoder) sequence(n *Node, out reflect.Value) (good bool) {
	l := len(n.Content)

	var iface reflect.Value
	switch out.Kind() {
	case reflect.Slice:
		out.Set(reflect.MakeSlice(out.Type(), l, l))
	case reflect.Array:
		if l != out.Len() {
			failf("invalid array: want %d elements but got %d", out.Len(), l)
		}
	case reflect.Interface:
		// No type hints. Will have to use a generic sequence.
		iface = out
		out = settableValueOf(make([]any, l))
	default:
		d.terror(n, seqTag, out)
		return false
	}
	et := out.Type().Elem()

	j := 0
	for i := 0; i < l; i++ {
		e := reflect.New(et).Elem()
		if ok := d.Unmarshal(n.Content[i], e); ok {
			out.Index(j).Set(e)
			j++
		}
	}
	if out.Kind() != reflect.Array {
		out.Set(out.Slice(0, j))
	}
	if iface.IsValid() {
		iface.Set(out)
	}
	return true
}

func (d *Decoder) mapping(n *Node, out reflect.Value) (good bool) {
	l := len(n.Content)
	if d.UniqueKeys {
		nerrs := len(d.Terrors)
		for i := 0; i < l; i += 2 {
			ni := n.Content[i]
			for j := i + 2; j < l; j += 2 {
				nj := n.Content[j]
				if ni.Kind == nj.Kind && ni.Value == nj.Value {
					d.Terrors = append(d.Terrors, &UnmarshalError{
						Err:    fmt.Errorf("mapping key %#v already defined at line %d", nj.Value, ni.Line),
						Line:   nj.Line,
						Column: nj.Column,
					})
				}
			}
		}
		if len(d.Terrors) > nerrs {
			return false
		}
	}
	switch out.Kind() {
	case reflect.Struct:
		return d.mappingStruct(n, out)
	case reflect.Map:
		// okay
	case reflect.Interface:
		iface := out
		if isStringMap(n) {
			out = reflect.MakeMap(d.stringMapType)
		} else {
			out = reflect.MakeMap(d.generalMapType)
		}
		iface.Set(out)
	default:
		d.terror(n, mapTag, out)
		return false
	}

	outt := out.Type()
	kt := outt.Key()
	et := outt.Elem()

	stringMapType := d.stringMapType
	generalMapType := d.generalMapType
	if outt.Elem() == ifaceType {
		if outt.Key().Kind() == reflect.String {
			d.stringMapType = outt
		} else if outt.Key() == ifaceType {
			d.generalMapType = outt
		}
	}

	mergedFields := d.mergedFields
	d.mergedFields = nil

	var mergeNode *Node

	mapIsNew := false
	if out.IsNil() {
		out.Set(reflect.MakeMap(outt))
		mapIsNew = true
	}
	for i := 0; i < l; i += 2 {
		if isMerge(n.Content[i]) {
			mergeNode = n.Content[i+1]
			continue
		}
		k := reflect.New(kt).Elem()
		if d.Unmarshal(n.Content[i], k) {
			if mergedFields != nil {
				ki := k.Interface()
				if d.getPossiblyUnhashableKey(mergedFields, ki) {
					continue
				}
				d.setPossiblyUnhashableKey(mergedFields, ki, true)
			}
			kkind := k.Kind()
			if kkind == reflect.Interface {
				kkind = k.Elem().Kind()
			}
			if kkind == reflect.Map || kkind == reflect.Slice {
				failf("cannot use '%#v' as a map key; try decoding into yaml.Node", k.Interface())
			}
			e := reflect.New(et).Elem()
			if d.Unmarshal(n.Content[i+1], e) || n.Content[i+1].ShortTag() == nullTag && (mapIsNew || !out.MapIndex(k).IsValid()) {
				out.SetMapIndex(k, e)
			}
		}
	}

	d.mergedFields = mergedFields
	if mergeNode != nil {
		d.merge(n, mergeNode, out)
	}

	d.stringMapType = stringMapType
	d.generalMapType = generalMapType
	return true
}

func isStringMap(n *Node) bool {
	if n.Kind != MappingNode {
		return false
	}
	l := len(n.Content)
	for i := 0; i < l; i += 2 {
		shortTag := n.Content[i].ShortTag()
		if shortTag != strTag && shortTag != mergeTag {
			return false
		}
	}
	return true
}

func (d *Decoder) mappingStruct(n *Node, out reflect.Value) (good bool) {
	sinfo, err := getStructInfo(out.Type())
	if err != nil {
		panic(err)
	}

	var inlineMap reflect.Value
	var elemType reflect.Type
	if sinfo.InlineMap != -1 {
		inlineMap = out.Field(sinfo.InlineMap)
		elemType = inlineMap.Type().Elem()
	}

	for _, index := range sinfo.InlineUnmarshalers {
		field := d.fieldByIndex(n, out, index)
		d.prepare(n, field)
	}

	mergedFields := d.mergedFields
	d.mergedFields = nil
	var mergeNode *Node
	var doneFields []bool
	if d.UniqueKeys {
		doneFields = make([]bool, len(sinfo.FieldsList))
	}
	name := settableValueOf("")
	l := len(n.Content)
	for i := 0; i < l; i += 2 {
		ni := n.Content[i]
		if isMerge(ni) {
			mergeNode = n.Content[i+1]
			continue
		}
		if !d.Unmarshal(ni, name) {
			continue
		}
		sname := name.String()
		if mergedFields != nil {
			if mergedFields[sname] {
				continue
			}
			mergedFields[sname] = true
		}
		if info, ok := sinfo.FieldsMap[sname]; ok {
			if d.UniqueKeys {
				if doneFields[info.Id] {
					d.Terrors = append(d.Terrors, &UnmarshalError{
						Err:    fmt.Errorf("field %s already set in type %s", name.String(), out.Type()),
						Line:   ni.Line,
						Column: ni.Column,
					})
					continue
				}
				doneFields[info.Id] = true
			}
			var field reflect.Value
			if info.Inline == nil {
				field = out.Field(info.Num)
			} else {
				field = d.fieldByIndex(n, out, info.Inline)
			}
			d.Unmarshal(n.Content[i+1], field)
		} else if sinfo.InlineMap != -1 {
			if inlineMap.IsNil() {
				inlineMap.Set(reflect.MakeMap(inlineMap.Type()))
			}
			value := reflect.New(elemType).Elem()
			d.Unmarshal(n.Content[i+1], value)
			inlineMap.SetMapIndex(name, value)
		} else if d.KnownFields {
			d.Terrors = append(d.Terrors, &UnmarshalError{
				Err:    fmt.Errorf("field %s not found in type %s", name.String(), out.Type()),
				Line:   ni.Line,
				Column: ni.Column,
			})
		}
	}

	d.mergedFields = mergedFields
	if mergeNode != nil {
		d.merge(n, mergeNode, out)
	}
	return true
}

func failWantMap() {
	failf("map merge requires map or sequence of maps as the value")
}

func (d *Decoder) setPossiblyUnhashableKey(m map[any]bool, key any, value bool) {
	defer func() {
		if err := recover(); err != nil {
			failf("%v", err)
		}
	}()
	m[key] = value
}

func (d *Decoder) getPossiblyUnhashableKey(m map[any]bool, key any) bool {
	defer func() {
		if err := recover(); err != nil {
			failf("%v", err)
		}
	}()
	return m[key]
}

func (d *Decoder) merge(parent *Node, merge *Node, out reflect.Value) {
	mergedFields := d.mergedFields
	if mergedFields == nil {
		d.mergedFields = make(map[any]bool)
		for i := 0; i < len(parent.Content); i += 2 {
			k := reflect.New(ifaceType).Elem()
			if d.Unmarshal(parent.Content[i], k) {
				d.setPossiblyUnhashableKey(d.mergedFields, k.Interface(), true)
			}
		}
	}

	switch merge.Kind {
	case MappingNode:
		d.Unmarshal(merge, out)
	case AliasNode:
		if merge.Alias != nil && merge.Alias.Kind != MappingNode {
			failWantMap()
		}
		d.Unmarshal(merge, out)
	case SequenceNode:
		for i := 0; i < len(merge.Content); i++ {
			ni := merge.Content[i]
			if ni.Kind == AliasNode {
				if ni.Alias != nil && ni.Alias.Kind != MappingNode {
					failWantMap()
				}
			} else if ni.Kind != MappingNode {
				failWantMap()
			}
			d.Unmarshal(ni, out)
		}
	default:
		failWantMap()
	}

	d.mergedFields = mergedFields
}

func isMerge(n *Node) bool {
	return n.Kind == ScalarNode && n.Value == "<<" && (n.Tag == "" || n.Tag == "!" || shortTag(n.Tag) == mergeTag)
}
