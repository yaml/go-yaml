// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestMergeKeyAliasCopyLimit(t *testing.T) {
	limit := errors.New("test alias limit")
	opts, err := ApplyOptions()
	if err != nil {
		t.Fatal(err)
	}
	opts.AliasCheck = func(aliases, constructs int) error {
		if aliases > 4 {
			return limit
		}
		return nil
	}
	c := NewConstructor(opts)
	source := &Node{Kind: MappingNode, Tag: mapTag}
	for i := 0; i < 32; i++ {
		source.Content = append(source.Content,
			&Node{Kind: ScalarNode, Tag: strTag, Value: strconv.Itoa(i)},
			&Node{Kind: ScalarNode, Tag: strTag, Value: "value"})
	}
	alias := &Node{Kind: AliasNode, Value: "source", Alias: source}
	load := func() (err error) {
		defer handleErr(&err)
		c.effectiveMapping(alias, reflect.TypeOf(""))
		return nil
	}
	if err := load(); !errors.Is(err, limit) {
		t.Errorf("copies were not limited before allocation: %v", err)
	}
	if copies := len(c.mergeAliases); copies > 4 {
		t.Errorf("allocated %d copied nodes before checking limit 4", copies)
	}
}
