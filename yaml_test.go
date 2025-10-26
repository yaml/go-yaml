package yaml

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func Test_getStructInfo(t *testing.T) {
	type Inline struct {
		Inline string
	}
	tests := []struct {
		name           string
		st             reflect.Type
		fallbackToJSON bool
		want           *structInfo
		wantErr        string
	}{
		{
			name: "tag names",
			st: reflect.TypeOf(struct {
				A string `yaml:"yaml_a"`
				B string `just_b`
				C string `json:"json_c" yaml:"yaml_c"`
				D string `json:"json_d"`
				E string
			}{}),
			fallbackToJSON: false,
			want: &structInfo{
				FieldsMap: map[string]fieldInfo{
					"yaml_a": {
						Key:       "yaml_a",
						Num:       0,
						OmitEmpty: false,
						Flow:      false,
						Id:        0,
						Inline:    nil,
					},
					"just_b": {
						Key:       "just_b",
						Num:       1,
						OmitEmpty: false,
						Flow:      false,
						Id:        1,
						Inline:    nil,
					},
					"yaml_c": {
						Key:       "yaml_c",
						Num:       2,
						OmitEmpty: false,
						Flow:      false,
						Id:        2,
						Inline:    nil,
					},
					"d": {
						Key:       "d",
						Num:       3,
						OmitEmpty: false,
						Flow:      false,
						Id:        3,
						Inline:    nil,
					},
					"e": {
						Key:       "e",
						Num:       4,
						OmitEmpty: false,
						Flow:      false,
						Id:        4,
						Inline:    nil,
					},
				},
				FieldsList: []fieldInfo{
					{
						Key:       "yaml_a",
						Num:       0,
						OmitEmpty: false,
						Flow:      false,
						Id:        0,
						Inline:    nil,
					},
					{
						Key:       "just_b",
						Num:       1,
						OmitEmpty: false,
						Flow:      false,
						Id:        1,
						Inline:    nil,
					},
					{
						Key:       "yaml_c",
						Num:       2,
						OmitEmpty: false,
						Flow:      false,
						Id:        2,
						Inline:    nil,
					},
					{
						Key:       "d",
						Num:       3,
						OmitEmpty: false,
						Flow:      false,
						Id:        3,
						Inline:    nil,
					},
					{
						Key:       "e",
						Num:       4,
						OmitEmpty: false,
						Flow:      false,
						Id:        4,
						Inline:    nil,
					},
				},
				InlineMap:          -1,
				InlineUnmarshalers: nil,
			},
		},
		{
			name: "tag names with fallback to json",
			st: reflect.TypeOf(struct {
				A string `yaml:"yaml_a"`
				B string `just_b`
				C string `json:"json_c" yaml:"yaml_c"`
				D string `json:"json_d"`
				E string
			}{}),
			fallbackToJSON: true,
			want: &structInfo{
				FieldsMap: map[string]fieldInfo{
					"yaml_a": {
						Key:       "yaml_a",
						Num:       0,
						OmitEmpty: false,
						Flow:      false,
						Id:        0,
						Inline:    nil,
					},
					"just_b": {
						Key:       "just_b",
						Num:       1,
						OmitEmpty: false,
						Flow:      false,
						Id:        1,
						Inline:    nil,
					},
					"yaml_c": {
						Key:       "yaml_c",
						Num:       2,
						OmitEmpty: false,
						Flow:      false,
						Id:        2,
						Inline:    nil,
					},
					"json_d": {
						Key:       "json_d",
						Num:       3,
						OmitEmpty: false,
						Flow:      false,
						Id:        3,
						Inline:    nil,
					},
					"e": {
						Key:       "e",
						Num:       4,
						OmitEmpty: false,
						Flow:      false,
						Id:        4,
						Inline:    nil,
					},
				},
				FieldsList: []fieldInfo{
					{
						Key:       "yaml_a",
						Num:       0,
						OmitEmpty: false,
						Flow:      false,
						Id:        0,
						Inline:    nil,
					},
					{
						Key:       "just_b",
						Num:       1,
						OmitEmpty: false,
						Flow:      false,
						Id:        1,
						Inline:    nil,
					},
					{
						Key:       "yaml_c",
						Num:       2,
						OmitEmpty: false,
						Flow:      false,
						Id:        2,
						Inline:    nil,
					},
					{
						Key:       "json_d",
						Num:       3,
						OmitEmpty: false,
						Flow:      false,
						Id:        3,
						Inline:    nil,
					},
					{
						Key:       "e",
						Num:       4,
						OmitEmpty: false,
						Flow:      false,
						Id:        4,
						Inline:    nil,
					},
				},
				InlineMap:          -1,
				InlineUnmarshalers: nil,
			},
		},
		{
			name: "inline",
			st: reflect.TypeOf(struct {
				Inline `yaml:",inline"`
				B      string
			}{}),
			fallbackToJSON: false,
			want: &structInfo{
				FieldsMap: map[string]fieldInfo{
					"inline": {
						Key:       "inline",
						Num:       0,
						OmitEmpty: false,
						Flow:      false,
						Id:        0,
						Inline:    []int{0, 0},
					},
					"b": {
						Key:       "b",
						Num:       1,
						OmitEmpty: false,
						Flow:      false,
						Id:        1,
						Inline:    nil,
					},
				},
				FieldsList: []fieldInfo{
					{
						Key:       "inline",
						Num:       0,
						OmitEmpty: false,
						Flow:      false,
						Id:        0,
						Inline:    []int{0, 0},
					},
					{
						Key:       "b",
						Num:       1,
						OmitEmpty: false,
						Flow:      false,
						Id:        1,
						Inline:    nil,
					},
				},
				InlineMap:          -1,
				InlineUnmarshalers: nil,
			},
		},
		{
			name: "inline with fallbackToJSON",
			st: reflect.TypeOf(struct {
				Inline
				B string
			}{}),
			fallbackToJSON: true,
			want: &structInfo{
				FieldsMap: map[string]fieldInfo{
					"inline": {
						Key:       "inline",
						Num:       0,
						OmitEmpty: false,
						Flow:      false,
						Id:        0,
						Inline:    []int{0, 0},
					},
					"b": {
						Key:       "b",
						Num:       1,
						OmitEmpty: false,
						Flow:      false,
						Id:        1,
						Inline:    nil,
					},
				},
				FieldsList: []fieldInfo{
					{
						Key:       "inline",
						Num:       0,
						OmitEmpty: false,
						Flow:      false,
						Id:        0,
						Inline:    []int{0, 0},
					},
					{
						Key:       "b",
						Num:       1,
						OmitEmpty: false,
						Flow:      false,
						Id:        1,
						Inline:    nil,
					},
				},
				InlineMap:          -1,
				InlineUnmarshalers: nil,
			},
		},
		{
			name: "inline pointer with fallbackToJSON",
			st: reflect.TypeOf(struct {
				*Inline
				B string
			}{}),
			fallbackToJSON: true,
			want: &structInfo{
				FieldsMap: map[string]fieldInfo{
					"inline": {
						Key:       "inline",
						Num:       0,
						OmitEmpty: false,
						Flow:      false,
						Id:        0,
						Inline:    []int{0, 0},
					},
					"b": {
						Key:       "b",
						Num:       1,
						OmitEmpty: false,
						Flow:      false,
						Id:        1,
						Inline:    nil,
					},
				},
				FieldsList: []fieldInfo{
					{
						Key:       "inline",
						Num:       0,
						OmitEmpty: false,
						Flow:      false,
						Id:        0,
						Inline:    []int{0, 0},
					},
					{
						Key:       "b",
						Num:       1,
						OmitEmpty: false,
						Flow:      false,
						Id:        1,
						Inline:    nil,
					},
				},
				InlineMap:          -1,
				InlineUnmarshalers: nil,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			oldStructMap := structMap
			t.Cleanup(func() {
				structMap = oldStructMap
			})
			structMap = make(map[reflect.Type]*structInfo) // reset cache
			got, err := getStructInfo(tt.st, tt.fallbackToJSON)
			if tt.wantErr != "" {
				assert.ErrorMatchesf(t, tt.wantErr, err, "getStructInfo() error")
			} else {
				assert.NoError(t, err)
			}

			assert.DeepEqualf(t, tt.want, got, "getStructInfo() failed")
		})
	}
}

func ExampleDecoder_FallbackToJSON() {
	type Inline struct {
		A int `json:"json_a"`
	}
	type Test struct {
		Inline
		B string `json:"json_b"`
	}

	text := []byte(`---
json_a: 42
json_b: "foo"
`)

	var v Test
	d := NewDecoder(bytes.NewReader(text))
	d.FallbackToJSON(true)
	if err := d.Decode(&v); err != nil {
		panic(err)
	}
	fmt.Println(v.A, v.B)
	// Output:
	// 42 foo
}

func ExampleEncoder_FallbackToJSON() {
	type Inline struct {
		A int `json:"json_a"`
	}
	type Test struct {
		Inline
		B string `json:"json_b"`
	}
	v := Test{
		Inline: Inline{
			A: 42,
		},
		B: "foo",
	}
	var buf bytes.Buffer
	e := NewEncoder(&buf)
	e.FallbackToJSON(true)
	if err := e.Encode(v); err != nil {
		panic(err)
	}
	fmt.Println(buf.String())
	// Output:
	// json_a: 42
	// json_b: foo
}
