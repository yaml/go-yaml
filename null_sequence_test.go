// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestNullSequenceElements(t *testing.T) {
	type entry struct {
		Name string `yaml:"name"`
	}
	tests := []struct {
		name   string
		input  string
		target func() any
		want   any
	}{
		{
			name:   "string slice",
			input:  "[before, null, after, ~]",
			target: func() any { return new([]string) },
			want:   &[]string{"before", "", "after", ""},
		},
		{
			name:   "integer slice",
			input:  "[1, null, 2, ~]",
			target: func() any { return new([]int) },
			want:   &[]int{1, 0, 2, 0},
		},
		{
			name:   "boolean slice",
			input:  "[null, true, ~, false]",
			target: func() any { return new([]bool) },
			want:   &[]bool{false, true, false, false},
		},
		{
			name:   "omitted entries",
			input:  "- # first\n- # second\n- ret_jsr\n",
			target: func() any { return new([]string) },
			want:   &[]string{"", "", "ret_jsr"},
		},
		{
			name:   "struct slice",
			input:  "-\n- name: second\n-\n",
			target: func() any { return new([]entry) },
			want:   &[]entry{{}, {Name: "second"}, {}},
		},
		{
			name:   "array",
			input:  "[before, null, after, ~]",
			target: func() any { return new([4]string) },
			want:   &[4]string{"before", "", "after", ""},
		},
		{
			name:   "null alias",
			input:  "[&empty null, before, *empty, after]",
			target: func() any { return new([]string) },
			want:   &[]string{"", "before", "", "after"},
		},
		{
			name:   "interface slice",
			input:  "[null, value, ~]",
			target: func() any { return new([]any) },
			want:   &[]any{nil, "value", nil},
		},
		{
			name:   "pointer slice",
			input:  "[null, ~]",
			target: func() any { return new([]*string) },
			want:   &[]*string{nil, nil},
		},
	}
	loaders := []struct {
		name string
		load func([]byte, any) error
	}{
		{"Unmarshal", yaml.Unmarshal},
		{"Load", func(data []byte, value any) error {
			return yaml.Load(data, value)
		}},
	}
	for _, loader := range loaders {
		t.Run(loader.name, func(t *testing.T) {
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					got := test.target()
					if err := loader.load([]byte(test.input), got); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got, test.want) {
						t.Errorf("got %#v, want %#v",
							reflect.ValueOf(got).Elem().Interface(),
							reflect.ValueOf(test.want).Elem().Interface())
					}
				})
			}
		})
	}
}
