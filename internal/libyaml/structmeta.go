// Copyright 2011-2019 Canonical Ltd
// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Struct metadata extraction for YAML marshaling/unmarshaling.
//
// This file analyzes Go struct types to build mappings between YAML keys and
// struct fields. It parses struct tags like `yaml:"name,omitempty,flow,inline"`
// and caches the results for efficient repeated access.
//
// Used by:
//   - Constructor: maps YAML keys to struct fields when unmarshaling
//   - Representer: maps struct fields to YAML keys when marshaling
//
// Key types:
//   - structInfo: cached metadata about a struct type
//   - fieldInfo: metadata about a single struct field
//   - getStructInfo(): analyzes a struct type and returns cached metadata

package libyaml

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// structInfo holds cached information about a struct's YAML-relevant fields.
type structInfo struct {
	FieldsMap  map[string]fieldInfo
	FieldsList []fieldInfo

	// InlineMap is the number of the field in the struct that
	// contains an ,inline map, or -1 if there's none.
	InlineMap int

	// InlineConstructors holds indexes to inlined fields that
	// contain constructor values.
	InlineConstructors [][]int
}

// fieldInfo holds information about a single struct field.
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

// customConstructorCheckFn is the type which a check function for the custom
// constructor must have.
type customConstructorCheckFn func(p reflect.Type) bool

// structMapKey defines the key for caching struct reflection information.
// Namely, we must cache the type of check performed for inline-constructors,
// since unmarshaling and marshaling are assymmetric in what data they should
// return (this allows nested fields to work properly).
type structMapKey struct {
	st reflect.Type
}

// structMap caches struct reflection information.
// fieldMapMutex protects access to structMap.
// constructorType holds the [reflect.Type] for the constructor interface.
var (
	structMap       = make(map[structMapKey]*structInfo)
	fieldMapMutex   sync.RWMutex
	constructorType reflect.Type
)

// constructor interface is defined here to detect types that implement
// UnmarshalYAML during struct reflection.
type constructor interface {
	UnmarshalYAML(value *Node) error
}

// init initializes the constructorType variable with the [reflect.Type] of constructor interface.
func init() {
	var v constructor
	constructorType = reflect.ValueOf(&v).Elem().Type()
}

// hasConstructYAMLMethod checks if a type has an UnmarshalYAML method
// that takes a *Node from an allowlisted v3 yaml package. This detects
// v3 backward-compatible Unmarshaler implementations whose Node type
// can't be checked via interface assertion from this package.
func hasConstructYAMLMethod(t reflect.Type) bool {
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
	if paramType.Kind() != reflect.Pointer {
		return false
	}

	elemType := paramType.Elem()
	if elemType.Kind() != reflect.Struct || elemType.Name() != "Node" || !isYAMLNodePkg(elemType.PkgPath()) {
		return false
	}

	// Return type should be error
	retType := mtype.Out(0)
	if retType.Kind() != reflect.Interface || retType.Name() != "error" {
		return false
	}

	return true
}


func isYAMLNodePkg(pkg string) bool {
	switch pkg {
	case "gopkg.in/yaml.v3", "go.yaml.in/yaml/v3":
		return true
	}
	return false
}

func defaultCustomConstructorCheck(p reflect.Type) bool {
	return true
}

// getStructInfo returns cached information about a struct type's fields.
// It parses struct tags and builds a map of field names to field info.
// hasConstructorCheck is an optional function which should be supplied by
// the caller if custom constructors are being used. In this case, localFieldMap
// can also be supplied to cache these lookups - it must be ensured that the
// supplied map is used when the same lookup function is used.
func getStructInfo(st reflect.Type, hasConstructorCheck customConstructorCheckFn, localFieldMap map[structMapKey]*structInfo) (*structInfo, error) {
	// Only use the lookup if we're not also looking for a custom constructor
	var found bool
	var sinfo *structInfo
	if hasConstructorCheck == nil {
		fieldMapMutex.RLock()
		sinfo, found = structMap[structMapKey{st}]
		fieldMapMutex.RUnlock()
		if found {
			return sinfo, nil
		}
	} else if localFieldMap != nil {
		sinfo, found = localFieldMap[structMapKey{st}]
		if found {
			return sinfo, nil
		}
	}

	// make the interface for this function easy to use. When not supplied it
	// should always return false
	var callableHasConstructorCheck customConstructorCheckFn
	if hasConstructorCheck == nil {
		callableHasConstructorCheck = defaultCustomConstructorCheck
	} else {
		callableHasConstructorCheck = hasConstructorCheck
	}

	n := st.NumField()
	fieldsMap := make(map[string]fieldInfo)
	fieldsList := make([]fieldInfo, 0, n)
	inlineMap := -1
	inlineConstructors := [][]int(nil)
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
				// Check for both libyaml.constructor and yaml.Unmarshaler (by method name).
				// Also check for a customTypeUnmarshaler in the current context.
				ftypePtr := reflect.PointerTo(ftype)
				if ftypePtr.Implements(constructorType) || hasConstructYAMLMethod(ftypePtr) || callableHasConstructorCheck(ftype) {
					inlineConstructors = append(inlineConstructors, []int{i})
				} else {
					sinfo, err := getStructInfo(ftype, hasConstructorCheck, localFieldMap)
					if err != nil {
						return nil, err
					}
					for _, index := range sinfo.InlineConstructors {
						inlineConstructors = append(inlineConstructors, append([]int{i}, index...))
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
		InlineConstructors: inlineConstructors,
	}

	if hasConstructorCheck == nil {
		fieldMapMutex.Lock()
		structMap[structMapKey{st}] = sinfo
		fieldMapMutex.Unlock()
	} else if localFieldMap != nil {
		localFieldMap[structMapKey{st}] = sinfo
	}
	return sinfo, nil
}
