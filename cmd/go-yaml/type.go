// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"unicode/utf16"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
)

func newDestination(destinationType reflect.Type) any {
	if destinationType != nil {
		return reflect.New(destinationType).Interface()
	}
	return new(any)
}

// loadDestination handles the --unmarshal path.
// Typed invocations must exercise Unmarshal to reproduce classic API behavior.
// Untyped invocations retain the CLI's historical use of Load for compatibility.
func loadDestination(destinationType reflect.Type, data []byte, dst any) error {
	if destinationType != nil {
		return yaml.Unmarshal(data, dst)
	}
	return yaml.Load(data, dst)
}

func resolveType(expr string) (typ reflect.Type, err error) {
	defer func() {
		if r := recover(); r != nil {
			typ = nil
			err = fmt.Errorf("invalid --type %q: %v", expr, r)
		}
	}()
	node, err := parser.ParseExpr(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid --type: %w", err)
	}
	return resolveTypeNode(node)
}

func resolveTypeNode(node ast.Expr) (reflect.Type, error) {
	switch n := node.(type) {
	case *ast.Ident:
		if n.Name == "any" {
			return reflect.TypeOf((*any)(nil)).Elem(), nil
		}
		types := map[string]any{
			"string": "", "bool": false, "int": int(0), "int8": int8(0),
			"int16": int16(0), "int32": int32(0), "int64": int64(0),
			"uint": uint(0), "uint8": uint8(0), "uint16": uint16(0),
			"uint32": uint32(0), "uint64": uint64(0), "uintptr": uintptr(0),
			"byte": byte(0), "rune": rune(0),
			"float32": float32(0), "float64": float64(0),
		}
		if v, ok := types[n.Name]; ok {
			return reflect.TypeOf(v), nil
		}
	case *ast.ParenExpr:
		return resolveTypeNode(n.X)
	case *ast.StarExpr:
		t, err := resolveTypeNode(n.X)
		if err != nil {
			return nil, err
		}
		return reflect.PointerTo(t), nil
	case *ast.ArrayType:
		t, err := resolveTypeNode(n.Elt)
		if err != nil {
			return nil, err
		}
		if n.Len == nil {
			return reflect.SliceOf(t), nil
		}
		lit, ok := n.Len.(*ast.BasicLit)
		if !ok || lit.Kind != token.INT {
			return nil, errors.New("--type array length must be a nonnegative integer literal")
		}
		// Normalize Go literal syntax, then parse directly into the native int.
		// Atoi checks the platform range without a narrowing integer conversion.
		value := constant.MakeFromLiteral(lit.Value, token.INT, 0)
		size, err := strconv.Atoi(value.ExactString())
		if err != nil {
			return nil, errors.New("--type array length exceeds int range")
		}
		return reflect.ArrayOf(size, t), nil
	case *ast.MapType:
		key, err := resolveTypeNode(n.Key)
		if err != nil {
			return nil, err
		}
		if !key.Comparable() {
			return nil, errors.New("--type map key must be comparable")
		}
		value, err := resolveTypeNode(n.Value)
		if err != nil {
			return nil, err
		}
		return reflect.MapOf(key, value), nil
	}
	return nil, errors.New("unsupported --type expression; use primitives, any, slices, arrays, maps, or pointers")
}

// splitTypedDocuments uses parser boundaries so quoted separators and block
// scalars remain intact. Construction still happens in Unmarshal itself.
func splitTypedDocuments(input []byte) ([][]byte, error) {
	// Parse the original bytes so the native reader rejects malformed encodings,
	// including unpaired UTF-16 surrogates, before any document is returned.
	options, err := libyaml.ApplyOptions()
	if err != nil {
		return nil, err
	}
	parser := libyaml.NewEventReader(bytes.NewReader(input), options)
	defer parser.Delete()

	runes, err := documentRunes(input)
	if err != nil {
		return nil, err
	}
	start := 0
	var docs [][]byte
	for {
		var event libyaml.Event
		if err := parser.Parse(&event); err != nil {
			return nil, err
		}
		kind, end := event.Type, event.EndMark.Index
		event.Delete()
		if kind == libyaml.DOCUMENT_END_EVENT {
			if end < start || end > len(runes) {
				return nil, errors.New("invalid document boundary")
			}
			docs = append(docs, []byte(string(runes[start:end])))
			start = end
		}
		if kind == libyaml.STREAM_END_EVENT {
			return docs, nil
		}
	}
}

// documentRunes creates a slicing buffer indexed by Unicode code points,
// matching the parser's character marks, rather than bytes or graphemes.
// The parser must still validate the original input before using this buffer.
func documentRunes(input []byte) ([]rune, error) {
	// The reader excludes the initial BOM from its character indices.
	input = bytes.TrimPrefix(input, []byte{0xef, 0xbb, 0xbf})

	if len(input) >= 2 && (bytes.Equal(input[:2], []byte{0xff, 0xfe}) ||
		bytes.Equal(input[:2], []byte{0xfe, 0xff})) {
		var order binary.ByteOrder = binary.LittleEndian
		if input[0] == 0xfe {
			order = binary.BigEndian
		}
		if len(input)%2 != 0 {
			return nil, errors.New("invalid UTF-16 input length")
		}
		units := make([]uint16, (len(input)-2)/2)
		for i := range units {
			units[i] = order.Uint16(input[2+i*2:])
		}
		input = []byte(string(utf16.Decode(units)))
	}
	return []rune(string(input)), nil
}
