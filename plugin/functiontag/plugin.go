// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package functiontag constructs tagged YAML values using registered functions.
package functiontag

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"go.yaml.in/yaml/v4/internal/libyaml"
)

// Functions maps local tag names to typed functions or uniform callbacks.
type Functions map[string]any

// Library is a snapshot of named function exports.
type Library = Functions

// Plugin is an immutable, operation-independent function lookup.
type Plugin struct {
	functions Functions
	libraries map[string]Library
}

var registry = struct {
	sync.RWMutex
	libraries map[string]Library
}{libraries: map[string]Library{}}

var (
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
	contextType = reflect.TypeOf(libyaml.FunctionTagContext{})
)

// Validate rejects unsupported constructor signatures before processing input.
func Validate(function any) error {
	value := reflect.ValueOf(function)
	if !value.IsValid() || value.Kind() != reflect.Func || value.IsNil() {
		return errors.New("function-tag: constructor must be a non-nil function")
	}
	t := value.Type()
	if t.NumOut() > 2 || (t.NumOut() == 2 && t.Out(1) != errorType) {
		return fmt.Errorf("function-tag: unsupported return signature %s", t)
	}
	return nil
}

// Exports snapshots an export map or the exported methods of an object.
func Exports(library any) (Library, error) {
	result := Library{}
	if functions, ok := library.(map[string]any); ok {
		for name, f := range functions {
			result[name] = f
		}
	} else if functions, ok := library.(Functions); ok {
		for name, f := range functions {
			result[name] = f
		}
	} else {
		value := reflect.ValueOf(library)
		if !value.IsValid() || (value.Kind() == reflect.Pointer && value.IsNil()) {
			return nil, errors.New("function-tag: library must not be nil")
		}
		if value.NumMethod() == 0 {
			return nil, errors.New("function-tag: library object has no exported methods")
		}
		for i := 0; i < value.NumMethod(); i++ {
			result[value.Type().Method(i).Name] = value.Method(i).Interface()
		}
	}
	for name, f := range result {
		if name == "" {
			return nil, errors.New("function-tag: empty function name")
		}
		if err := Validate(f); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
	}
	return result, nil
}

// RegisterLibrary registers exports once, normally during application startup.
func RegisterLibrary(name string, library any) error {
	exports, err := Exports(library)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("function-tag: empty library name")
	}
	registry.Lock()
	defer registry.Unlock()
	if _, exists := registry.libraries[name]; exists {
		return fmt.Errorf("function-tag: library %q already registered", name)
	}
	registry.libraries[name] = exports
	return nil
}

// New constructs an immutable plugin from explicit functions and an optional
// map of library aliases.
func New(functions Functions, libraries ...map[string]any) (*Plugin, error) {
	if len(libraries) > 1 {
		return nil, errors.New("function-tag: expected at most one library map")
	}
	var selected map[string]any
	if len(libraries) == 1 {
		selected = libraries[0]
	}
	exports, err := Exports(functions)
	if err != nil {
		return nil, err
	}
	p := &Plugin{functions: exports, libraries: map[string]Library{}}
	for name, library := range selected {
		if registered, ok := library.(string); ok {
			registry.RLock()
			exports, found := registry.libraries[registered]
			registry.RUnlock()
			if !found {
				return nil, fmt.Errorf("function-tag: unknown library %q", registered)
			}
			library = exports
		}
		values, err := Exports(library)
		if err != nil {
			return nil, err
		}
		p.libraries[name] = values
	}
	return p, nil
}

// NewFromYAML resolves serializable bindings against registered libraries.
func NewFromYAML(config map[string]any) (*Plugin, error) {
	libraries := map[string]any{}
	if raw, ok := config["libraries"]; ok {
		var good bool
		libraries, good = raw.(map[string]any)
		if !good {
			return nil, errors.New("function-tag: libraries must be a mapping")
		}
	}
	p, err := New(Functions{}, libraries)
	if err != nil {
		return nil, err
	}
	for key := range config {
		if key != "libraries" && key != "functions" {
			return nil, fmt.Errorf("function-tag: unknown configuration key %q", key)
		}
	}
	if raw, ok := config["functions"]; ok {
		bindings, good := raw.(map[string]any)
		if !good {
			return nil, errors.New("function-tag: functions must be a mapping")
		}
		for name, binding := range bindings {
			ref, good := binding.(string)
			if !good {
				return nil, errors.New("function-tag: configured functions must be library references")
			}
			f, err := p.lookup(ref)
			if err != nil {
				return nil, err
			}
			p.functions[name] = f
		}
	}
	return p, nil
}

func (p *Plugin) lookup(name string) (any, error) {
	if f, found := p.functions[name]; found {
		return f, nil
	}
	if alias, member, qualified := strings.Cut(name, "/"); qualified {
		if f, found := p.libraries[alias][member]; found {
			return f, nil
		}
		return nil, fmt.Errorf("function-tag: unknown function %q", name)
	}
	var found any
	for _, library := range p.libraries {
		if f, ok := library[name]; ok {
			if found != nil {
				return nil, fmt.Errorf("function-tag: ambiguous function %q", name)
			}
			found = f
		}
	}
	if found == nil {
		return nil, fmt.Errorf("function-tag: unknown function %q", name)
	}
	return found, nil
}

// ConstructFunctionTag implements yaml.FunctionTagPlugin. Parameters are decoded
// using the active YAML constructor, preserving custom child values and options.
func (p *Plugin) ConstructFunctionTag(input libyaml.FunctionTagInput) (result any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if cause, ok := recovered.(error); ok {
				err = fmt.Errorf("function-tag call: %w", cause)
			} else {
				err = fmt.Errorf("function-tag call: %v", recovered)
			}
		}
	}()
	name := strings.TrimSuffix(input.Tag, "*")
	if strings.HasPrefix(name, "!") && !strings.HasPrefix(name, "!!") {
		name = name[1:]
	}
	f, err := p.lookup(name)
	if err != nil {
		return nil, err
	}
	nodes := []*libyaml.Node{input.Node}
	if strings.HasSuffix(input.Tag, "*") {
		if input.Node.Kind != libyaml.SequenceNode {
			return nil, errors.New("function-tag: spread requires a sequence")
		}
		nodes = input.Node.Content
	}
	fn := reflect.ValueOf(f)
	t := fn.Type()
	minimum := t.NumIn()
	offset := 0
	if minimum > 0 && t.In(0) == contextType {
		offset = 1
		minimum--
	}
	if t.IsVariadic() {
		minimum--
	}
	if len(nodes) < minimum || (!t.IsVariadic() && len(nodes) != minimum) {
		return nil, fmt.Errorf("function-tag: %s expects %d arguments, got %d", name, minimum, len(nodes))
	}
	args := make([]reflect.Value, len(nodes)+offset)
	if offset != 0 {
		args[0] = reflect.ValueOf(input)
	}
	for i, node := range nodes {
		var param reflect.Type
		if t.IsVariadic() && i >= minimum {
			param = t.In(t.NumIn() - 1).Elem()
		} else {
			param = t.In(i + offset)
		}
		target := reflect.New(param)
		if err := input.Decode(node, target.Interface()); err != nil {
			return nil, fmt.Errorf("argument %d: %w", i+1, err)
		}
		args[i+offset] = target.Elem()
	}
	values := fn.Call(args)
	if len(values) == 0 {
		return libyaml.Omit, nil
	}
	if t.Out(len(values)-1) == errorType {
		last := values[len(values)-1]
		if !last.IsNil() {
			return nil, last.Interface().(error)
		}
		if len(values) == 1 {
			return libyaml.Omit, nil
		}
	}
	return values[0].Interface(), nil
}
