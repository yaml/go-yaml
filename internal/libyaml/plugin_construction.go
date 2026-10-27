// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
)

// ConstructionControl is an opaque instruction consumed during construction.
// Ordinary nil values remain YAML null.
type ConstructionControl struct {
	kind  string
	value any
}

var (
	// Omit suppresses one entry, sequence item, or document.
	Omit = ConstructionControl{kind: "omit"}
	// SpliceValue asks a key constructor to insert its associated mapping value.
	SpliceValue = ConstructionControl{kind: "splice-value"}
)

type noDocumentError int

func (noDocumentError) Error() string { return "yaml: no document produced" }

// NoDocument reports a successfully omitted document, separately from [io.EOF].
const NoDocument noDocumentError = 0

// Splice inserts a sequence's items or a mapping's entries at this position.
func Splice(value any) ConstructionControl { return ConstructionControl{kind: "splice", value: value} }

// Entry is an ordered mapping entry, useful when constructing a mapping splice.
type Entry struct{ Key, Value any }

// FunctionTagInput describes a tag call. Node contains its base-typed payload.
// Decode uses this operation's options and already constructed child results.
// Role is root, item, key, or value. Node and Path must be treated as read-only.
type FunctionTagInput struct {
	Tag  string
	Node *Node
	Role string
	Path []any
	// PairKey and PairValue expose the original resolved mapping entry for
	// key and value calls. Inspecting them does not construct either node.
	PairKey, PairValue *Node
	Decode             func(*Node, any) error
	// ConstructValue lazily constructs the associated mapping value once.
	ConstructValue func() (any, error)
	// DefineAlias adds data to this operation's active Alias-Data policy.
	DefineAlias func(string, any) error
	// LoadFile reads one isolated document without function constructors.
	LoadFile func(string) (any, error)
}

// FunctionTagContext is an optional leading argument to a typed constructor.
type FunctionTagContext = FunctionTagInput

// FunctionTagPlugin constructs tagged nodes. Return Omit, Splice, or SpliceValue
// for construction controls, or an ordinary native Go value.
type FunctionTagPlugin interface {
	ConstructFunctionTag(FunctionTagInput) (any, error)
}

type preparedResult struct {
	node    *Node
	control *ConstructionControl
}
type constructionSession struct {
	c                  *Constructor
	plugin             FunctionTagPlugin
	rules              YAMLRulesPlugin
	memo               map[*Node]preparedResult
	active             map[*Node]bool
	pairKey, pairValue *Node
	aliasData          AliasDataContext
	options            *Options
}

var functionAliasName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func (s *constructionSession) defineAlias(name string, value any) error {
	if s.aliasData == nil {
		return errors.New("function-tag: defining aliases requires Alias-Data")
	}
	if !functionAliasName.MatchString(name) {
		return fmt.Errorf("alias-data: invalid alias name %q", name)
	}
	if err := checkConstructionData(value, map[uintptr]bool{}); err != nil {
		return err
	}
	node, err := aliasDataServices(s.options).NewNode(value)
	if err != nil {
		return err
	}
	return s.aliasData.DefineAnchor(name, node)
}

func (s *constructionSession) loadFile(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value any
	err = Load(data, &value, func(o *Options) error {
		*o = *s.options
		o.FunctionTag, o.AliasData = nil, nil
		o.AllDocuments, o.SingleDocument, o.StreamNodes = false, false, false
		o.allowLegacyTrailingContent = false
		return nil
	})
	return value, err
}

func customTag(tag string) bool {
	return tag != "" && tag != "!" && !strings.HasPrefix(shortTag(tag), "!!")
}

func (s *constructionSession) decode(n *Node, target any) (err error) {
	defer handleErr(&err)
	out := reflect.ValueOf(target)
	if out.Kind() != reflect.Pointer || out.IsNil() {
		return errors.New("yaml: argument destination must be a non-nil pointer")
	}
	before := len(s.c.TypeErrors)
	if !s.c.Construct(n, out.Elem()) || len(s.c.TypeErrors) > before {
		result := append([]*LoadError(nil), s.c.TypeErrors[before:]...)
		s.c.TypeErrors = s.c.TypeErrors[:before]
		if len(result) > 0 {
			return &LoadErrors{Errors: result}
		}
		return errors.New("yaml: cannot decode constructor argument")
	}
	return nil
}

func (s *constructionSession) prepare(n *Node, role string, path []any) (preparedResult, error) {
	if n.Kind == AliasNode {
		s.c.constructCount++
		s.c.aliasCount++
		if s.c.aliasCheck != nil {
			if err := s.c.aliasCheck(s.c.aliasCount, s.c.constructCount); err != nil {
				return preparedResult{}, err
			}
		}
		target := n.Alias
		if s.aliasData != nil {
			var found bool
			var err error
			target, found, err = s.aliasData.ResolveAlias(n.Value)
			if err != nil {
				return preparedResult{}, err
			}
			if !found {
				return preparedResult{}, fmt.Errorf("yaml: unknown anchor %q", n.Value)
			}
		}
		if target == nil {
			return preparedResult{}, errors.New("yaml: alias has no target")
		}
		return s.prepare(target, role, path)
	}
	if n.hasNativeValue {
		return preparedResult{node: n}, nil
	}
	if r, ok := s.memo[n]; ok {
		return r, nil
	}
	if s.active[n] {
		return preparedResult{}, errors.New("yaml: cyclic constructor dependency")
	}
	s.active[n] = true
	defer delete(s.active, n)
	copyNode := *n
	copyNode.Content = nil
	s.active[&copyNode] = true
	defer delete(s.active, &copyNode)
	if s.aliasData != nil && n.Anchor != "" {
		if err := s.aliasData.DefineAnchor(n.Anchor, &copyNode); err != nil {
			return preparedResult{}, err
		}
	}
	switch n.Kind {
	case DocumentNode:
		if len(n.Content) == 0 {
			return preparedResult{node: &copyNode}, nil
		}
		r, err := s.prepare(n.Content[0], "root", nil)
		if err != nil {
			return r, err
		}
		if r.control != nil {
			return r, nil
		}
		copyNode.Content = []*Node{r.node}
	case SequenceNode:
		for i, child := range n.Content {
			p := append(append([]any(nil), path...), i)
			r, err := s.prepare(child, "item", p)
			if err != nil {
				return r, err
			}
			if r.control == nil {
				copyNode.Content = append(copyNode.Content, r.node)
				continue
			}
			switch r.control.kind {
			case "omit":
			case "splice":
				values := reflect.ValueOf(r.control.value)
				if !values.IsValid() || (values.Kind() != reflect.Slice && values.Kind() != reflect.Array) {
					return r, errors.New("yaml: sequence splice requires a sequence")
				}
				for j := 0; j < values.Len(); j++ {
					if err := checkConstructionData(values.Index(j).Interface(), map[uintptr]bool{}); err != nil {
						return r, err
					}
					copyNode.Content = append(copyNode.Content, nativeNode(values.Index(j).Interface()))
				}
			default:
				return r, errors.New("yaml: SpliceValue is only valid in a mapping key")
			}
		}
	case MappingNode:
		for i := 0; i < len(n.Content); i += 2 {
			key, value := n.Content[i], n.Content[i+1]
			p := append(append([]any(nil), path...), key.Value)
			k, err := s.prepareEntry(key, "key", p, key, value)
			if err != nil {
				return k, err
			}
			if k.control != nil {
				switch k.control.kind {
				case "omit":
					continue
				case "splice":
					entries, err := s.entries(k.control.value)
					if err != nil {
						return k, err
					}
					copyNode.Content = append(copyNode.Content, entries...)
					continue
				case "splice-value":
					v, err := s.prepareEntry(value, "value", p, key, value)
					if err != nil {
						return v, err
					}
					if v.control != nil {
						return v, errors.New("yaml: SpliceValue requires an ordinary mapping value")
					}
					var data any
					if err := s.decode(v.node, &data); err != nil {
						return v, err
					}
					entries, err := s.entries(data)
					if err != nil {
						return v, err
					}
					copyNode.Content = append(copyNode.Content, entries...)
					continue
				default:
					return k, errors.New("yaml: invalid key construction control")
				}
			}
			v, err := s.prepareEntry(value, "value", p, key, value)
			if err != nil {
				return v, err
			}
			if v.control != nil {
				switch v.control.kind {
				case "omit":
					continue
				case "splice":
					entries, err := s.entries(v.control.value)
					if err != nil {
						return v, err
					}
					copyNode.Content = append(copyNode.Content, entries...)
					continue
				default:
					return v, errors.New("yaml: SpliceValue is only valid in a mapping key")
				}
			}
			copyNode.Content = append(copyNode.Content, k.node, v.node)
		}
	}
	result := preparedResult{node: &copyNode}
	if customTag(n.Tag) {
		payload := copyNode
		tag, err := payloadBaseTag(n, s.rules)
		if err != nil {
			return result, err
		}
		payload.Tag = tag
		payload.Style &^= TaggedStyle
		input := FunctionTagInput{Tag: n.Tag, Node: &payload, Role: role, Path: append([]any(nil), path...), Decode: s.decode}
		input.DefineAlias, input.LoadFile = s.defineAlias, s.loadFile
		if role == "key" || role == "value" {
			input.PairKey = s.pairKey
			input.PairValue = s.pairValue
			pairKey, pairValue := s.pairKey, s.pairValue
			var called bool
			var pairedData any
			var pairedError error
			input.ConstructValue = func() (any, error) {
				if called {
					return pairedData, pairedError
				}
				called = true
				pairedData, pairedError = func() (any, error) {
					var prepared preparedResult
					var err error
					if n == pairValue {
						prepared.node = &payload
					} else {
						prepared, err = s.prepareEntry(pairValue, "value", path, pairKey, pairValue)
					}
					if err != nil {
						return nil, err
					}
					if prepared.control != nil {
						return nil, errors.New("function-tag: paired value produced a construction control")
					}
					var value any
					err = s.decode(prepared.node, &value)
					return value, err
				}()
				return pairedData, pairedError
			}
		}
		value, err := s.plugin.ConstructFunctionTag(input)
		if err != nil {
			return result, fmt.Errorf("function tag %s (%s at %v): %w", n.Tag, role, path, err)
		}
		if control, ok := value.(ConstructionControl); ok {
			result = preparedResult{control: &control}
		} else {
			if err := checkConstructionData(value, map[uintptr]bool{}); err != nil {
				return result, err
			}
			result.node = nativeNode(value)
		}
	}
	s.memo[n] = result
	s.memo[&copyNode] = result
	if result.node != nil {
		s.memo[result.node] = result
	}
	if s.aliasData != nil && n.Anchor != "" && result.node != nil {
		target := result.node
		if target.hasNativeValue {
			target = nativeAliasNode(target.nativeValue, map[aliasDataVisit]*Node{})
		}
		// Policies retain the live pointer registered before child traversal.
		// Complete that node in place instead of defining the anchor twice.
		copyNode = *target
	}
	return result, nil
}

// nativeAliasNode exposes collection paths without calling MarshalYAML on
// constructor results. Every node retains its original native value.
func nativeAliasNode(value any, seen map[aliasDataVisit]*Node) *Node {
	node := nativeNode(value)
	if value == nil {
		return node
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map, reflect.Slice:
		visit := aliasDataVisit{typeOf: v.Type(), ptr: uintptr(v.UnsafePointer()), length: v.Len()}
		if prior, ok := seen[visit]; ok {
			return prior
		}
		seen[visit] = node
	case reflect.Array:
	default:
		return node
	}
	if v.Kind() == reflect.Map {
		node.Kind, node.Tag = MappingNode, mapTag
		for _, key := range v.MapKeys() {
			node.Content = append(node.Content,
				nativeAliasNode(key.Interface(), seen), nativeAliasNode(v.MapIndex(key).Interface(), seen))
		}
	} else {
		node.Kind, node.Tag = SequenceNode, seqTag
		for i := 0; i < v.Len(); i++ {
			node.Content = append(node.Content, nativeAliasNode(v.Index(i).Interface(), seen))
		}
	}
	return node
}

func (s *constructionSession) prepareEntry(node *Node, role string, path []any, key, value *Node) (preparedResult, error) {
	savedKey, savedValue := s.pairKey, s.pairValue
	s.pairKey, s.pairValue = key, value
	defer func() { s.pairKey, s.pairValue = savedKey, savedValue }()
	return s.prepare(node, role, path)
}

func (s *constructionSession) entries(value any) ([]*Node, error) {
	var entries []Entry
	switch v := value.(type) {
	case []Entry:
		entries = v
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Map {
			return nil, errors.New("yaml: mapping splice requires a map or []yaml.Entry")
		}
		for _, key := range rv.MapKeys() {
			entries = append(entries, Entry{key.Interface(), rv.MapIndex(key).Interface()})
		}
	}
	nodes := make([]*Node, 0, len(entries)*2)
	for _, entry := range entries {
		if err := checkConstructionData(entry.Key, map[uintptr]bool{}); err != nil {
			return nil, err
		}
		if err := checkConstructionData(entry.Value, map[uintptr]bool{}); err != nil {
			return nil, err
		}
		nodes = append(nodes, nativeNode(entry.Key), nativeNode(entry.Value))
	}
	return nodes, nil
}

func nativeNode(value any) *Node {
	tag := strTag
	if value != nil {
		switch reflect.TypeOf(value).Kind() {
		case reflect.Bool:
			tag = boolTag
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			tag = intTag
		case reflect.Float32, reflect.Float64:
			tag = floatTag
		default:
			if _, ok := value.(string); !ok {
				tag = "!native"
			}
		}
	} else {
		tag = nullTag
	}
	return &Node{Kind: ScalarNode, Tag: tag, Value: fmt.Sprint(value), nativeValue: value, hasNativeValue: true}
}

func assignNative(value any, out reflect.Value) bool {
	if value == nil {
		switch out.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
			out.Set(reflect.Zero(out.Type()))
			return true
		}
		return false
	}
	v := reflect.ValueOf(value)
	if v.Type().AssignableTo(out.Type()) {
		out.Set(v)
		return true
	}
	if out.Kind() == reflect.Pointer {
		temp := reflect.New(out.Type().Elem())
		if assignNative(value, temp.Elem()) {
			out.Set(temp)
			return true
		}
	}
	if v.Type().ConvertibleTo(out.Type()) {
		switch out.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if v.Kind() >= reflect.Int && v.Kind() <= reflect.Int64 {
				if out.OverflowInt(v.Int()) {
					return false
				}
				out.SetInt(v.Int())
				return true
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if v.Kind() >= reflect.Uint && v.Kind() <= reflect.Uint64 {
				if out.OverflowUint(v.Uint()) {
					return false
				}
				out.SetUint(v.Uint())
				return true
			}
		case reflect.Float32, reflect.Float64:
			if v.Kind() == reflect.Float32 || v.Kind() == reflect.Float64 {
				if out.OverflowFloat(v.Float()) {
					return false
				}
				out.SetFloat(v.Float())
				return true
			}
		}
	}
	return false
}

func (c *Constructor) constructNative(n *Node, out reflect.Value) bool {
	if assignNative(n.nativeValue, out) {
		return true
	}
	// Mapping and sequence return values may fill typed destinations without a
	// serialization round trip. Each leaf retains its original native value.
	v := reflect.ValueOf(n.nativeValue)
	if v.IsValid() {
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return c.constructInt(n, v.Int(), out)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return c.constructInt(n, v.Uint(), out)
		case reflect.Float32, reflect.Float64:
			return c.constructFloat(n, v.Float(), out)
		case reflect.String:
			return c.constructStr(n, v.String(), out)
		case reflect.Bool:
			return c.constructBool(n, v.Bool(), out)
		}
	}
	if v.IsValid() && (v.Kind() == reflect.Map || v.Kind() == reflect.Slice || v.Kind() == reflect.Array) {
		copyNode := *n
		copyNode.hasNativeValue = false
		copyNode.Content = nil
		if v.Kind() == reflect.Map {
			copyNode.Kind = MappingNode
			copyNode.Tag = mapTag
			for _, key := range v.MapKeys() {
				copyNode.Content = append(copyNode.Content, nativeNode(key.Interface()), nativeNode(v.MapIndex(key).Interface()))
			}
		} else {
			copyNode.Kind = SequenceNode
			copyNode.Tag = seqTag
			for i := 0; i < v.Len(); i++ {
				copyNode.Content = append(copyNode.Content, nativeNode(v.Index(i).Interface()))
			}
		}
		return c.Construct(&copyNode, out)
	}
	c.tagError(n, n.Tag, out)
	return false
}

func checkConstructionData(value any, seen map[uintptr]bool) error {
	if _, ok := value.(ConstructionControl); ok {
		return errors.New("yaml: construction control nested inside ordinary data")
	}
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface {
		return checkConstructionData(v.Elem().Interface(), seen)
	}
	if v.Kind() == reflect.Map || v.Kind() == reflect.Slice || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		pointer := uintptr(0)
		if v.Kind() == reflect.Map {
			pointer = uintptr(v.UnsafePointer())
		} else {
			pointer = v.Pointer()
		}
		if seen[pointer] {
			return nil
		}
		seen[pointer] = true
	}
	switch v.Kind() {
	case reflect.Pointer:
		return checkConstructionData(v.Elem().Interface(), seen)
	case reflect.Map:
		for _, key := range v.MapKeys() {
			if err := checkConstructionData(key.Interface(), seen); err != nil {
				return err
			}
			if err := checkConstructionData(v.MapIndex(key).Interface(), seen); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := checkConstructionData(v.Index(i).Interface(), seen); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).CanInterface() {
				if err := checkConstructionData(v.Field(i).Interface(), seen); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
