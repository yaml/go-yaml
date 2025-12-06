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

package yaml_test

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

var marshalIntTest = 123

var marshalTests = []struct {
	value any
	data  string
}{
	// Basic types.
	{
		nil,
		"null\n",
	},
	{
		(*marshalerType)(nil),
		"null\n",
	},
	{
		&struct{}{},
		"{}\n",
	},
	{
		map[string]string{"v": "hi"},
		"v: hi\n",
	},
	{
		map[string]any{"v": "hi"},
		"v: hi\n",
	},
	{
		map[string]string{"v": "true"},
		"v: \"true\"\n",
	},
	{
		map[string]string{"v": "false"},
		"v: \"false\"\n",
	},
	{
		map[string]any{"v": true},
		"v: true\n",
	},
	{
		map[string]any{"v": false},
		"v: false\n",
	},
	{
		map[string]any{"v": 10},
		"v: 10\n",
	},
	{
		map[string]any{"v": -10},
		"v: -10\n",
	},
	{
		map[string]uint{"v": 42},
		"v: 42\n",
	},
	{
		map[string]any{"v": int64(4294967296)},
		"v: 4294967296\n",
	},
	{
		map[string]int64{"v": int64(4294967296)},
		"v: 4294967296\n",
	},
	{
		map[string]uint64{"v": 4294967296},
		"v: 4294967296\n",
	},
	{
		map[string]any{"v": "10"},
		"v: \"10\"\n",
	},
	{
		map[string]any{"v": 0.1},
		"v: 0.1\n",
	},
	{
		map[string]any{"v": float64(0.1)},
		"v: 0.1\n",
	},
	{
		map[string]any{"v": float32(0.99)},
		"v: 0.99\n",
	},
	{
		map[string]any{"v": -0.1},
		"v: -0.1\n",
	},
	{
		map[string]any{"v": math.Inf(+1)},
		"v: .inf\n",
	},
	{
		map[string]any{"v": math.Inf(-1)},
		"v: -.inf\n",
	},
	{
		map[string]any{"v": math.NaN()},
		"v: .nan\n",
	},
	{
		map[string]any{"v": nil},
		"v: null\n",
	},
	{
		map[string]any{"v": ""},
		"v: \"\"\n",
	},

	// Issue 65: Validate handling newlines at the start.
	{
		map[string]any{"v": "\nhi"},
		"v: |-\n\n    hi\n",
	},
	{
		map[string][]any{"v": {map[string]any{"v1": "\nhi"}}},
		"v:\n    - v1: |-\n\n        hi\n",
	},

	// Validate multiline strings.
	{
		map[string][]string{"v": {"A", "B"}},
		"v:\n    - A\n    - B\n",
	},
	{
		map[string][]string{"v": {"A", "B\nC"}},
		"v:\n    - A\n    - |-\n      B\n      C\n",
	},
	{
		map[string][]any{"v": {"A", 1, map[string][]int{"B": {2, 3}}}},
		"v:\n    - A\n    - 1\n    - B:\n        - 2\n        - 3\n",
	},
	{
		map[string]any{"a": map[any]any{"b": "c"}},
		"a:\n    b: c\n",
	},
	{
		map[string]any{"a": "-"},
		"a: '-'\n",
	},
	{
		map[string]any{"v": negativeZero},
		"v: -0\n",
	},

	// Simple values.
	{
		&marshalIntTest,
		"123\n",
	},
	{
		negativeZero,
		"-0\n",
	},
	{
		"\t\n",
		"\"\\t\\n\"\n",
	},

	// Structures
	{
		&struct{ Hello string }{"world"},
		"hello: world\n",
	},
	{
		&struct {
			A struct {
				B string
			}
		}{struct{ B string }{"c"}},
		"a:\n    b: c\n",
	},
	{
		&struct {
			A *struct {
				B string
			}
		}{&struct{ B string }{"c"}},
		"a:\n    b: c\n",
	},
	{
		&struct {
			A *struct {
				B string
			}
		}{},
		"a: null\n",
	},
	{
		&struct{ A int }{1},
		"a: 1\n",
	},
	{
		&struct{ A []int }{[]int{1, 2}},
		"a:\n    - 1\n    - 2\n",
	},
	{
		&struct{ A [2]int }{[2]int{1, 2}},
		"a:\n    - 1\n    - 2\n",
	},
	{
		&struct {
			B int `yaml:"a"`
		}{1},
		"a: 1\n",
	},
	{
		&struct{ A bool }{true},
		"a: true\n",
	},
	{
		&struct{ A string }{"true"},
		"a: \"true\"\n",
	},
	{
		&struct{ A string }{"off"},
		"a: \"off\"\n",
	},

	// Conditional flag
	{
		&struct {
			A int `yaml:"a,omitempty"`
			B int `yaml:"b,omitempty"`
		}{1, 0},
		"a: 1\n",
	},
	{
		&struct {
			A int `yaml:"a,omitempty"`
			B int `yaml:"b,omitempty"`
		}{0, 0},
		"{}\n",
	},
	{
		&struct {
			A *struct{ X, y int } `yaml:"a,omitempty,flow"`
		}{&struct{ X, y int }{1, 2}},
		"a: {x: 1}\n",
	},
	{
		&struct {
			A *struct{ X, y int } `yaml:"a,omitempty,flow"`
		}{nil},
		"{}\n",
	},
	{
		&struct {
			A *struct{ X, y int } `yaml:"a,omitempty,flow"`
		}{&struct{ X, y int }{}},
		"a: {x: 0}\n",
	},
	{
		&struct {
			A struct{ X, y int } `yaml:"a,omitempty,flow"`
		}{struct{ X, y int }{1, 2}},
		"a: {x: 1}\n",
	},
	{
		&struct {
			A struct{ X, y int } `yaml:"a,omitempty,flow"`
		}{struct{ X, y int }{0, 1}},
		"{}\n",
	},
	{
		&struct {
			A float64 `yaml:"a,omitempty"`
			B float64 `yaml:"b,omitempty"`
		}{1, 0},
		"a: 1\n",
	},
	{
		&struct {
			T1 time.Time  `yaml:"t1,omitempty"`
			T2 time.Time  `yaml:"t2,omitempty"`
			T3 *time.Time `yaml:"t3,omitempty"`
			T4 *time.Time `yaml:"t4,omitempty"`
		}{
			T2: time.Date(2018, 1, 9, 10, 40, 47, 0, time.UTC),
			T4: newTime(time.Date(2098, 1, 9, 10, 40, 47, 0, time.UTC)),
		},
		"t2: 2018-01-09T10:40:47Z\nt4: 2098-01-09T10:40:47Z\n",
	},
	// Nil interface that implements Marshaler.
	{
		map[string]yaml.Marshaler{
			"a": nil,
		},
		"a: null\n",
	},

	// Flow flag
	{
		&struct {
			A []int `yaml:"a,flow"`
		}{[]int{1, 2}},
		"a: [1, 2]\n",
	},
	{
		&struct {
			A map[string]string `yaml:"a,flow"`
		}{map[string]string{"b": "c", "d": "e"}},
		"a: {b: c, d: e}\n",
	},
	{
		&struct {
			A struct {
				B, D string
			} `yaml:"a,flow"`
		}{struct{ B, D string }{"c", "e"}},
		"a: {b: c, d: e}\n",
	},
	{
		&struct {
			A string `yaml:"a,flow"`
		}{"b\nc"},
		"a: \"b\\nc\"\n",
	},

	// Unexported field
	{
		&struct {
			u int
			A int
		}{0, 1},
		"a: 1\n",
	},

	// Ignored field
	{
		&struct {
			A int
			B int `yaml:"-"`
		}{1, 2},
		"a: 1\n",
	},

	// Struct inlining
	{
		&struct {
			A int
			C inlineB `yaml:",inline"`
		}{1, inlineB{2, inlineC{3}}},
		"a: 1\nb: 2\nc: 3\n",
	},
	// Struct inlining as a pointer
	{
		&struct {
			A int
			C *inlineB `yaml:",inline"`
		}{1, &inlineB{2, inlineC{3}}},
		"a: 1\nb: 2\nc: 3\n",
	},
	{
		&struct {
			A int
			C *inlineB `yaml:",inline"`
		}{1, nil},
		"a: 1\n",
	},
	{
		&struct {
			A int
			D *inlineD `yaml:",inline"`
		}{1, &inlineD{&inlineC{3}, 4}},
		"a: 1\nc: 3\nd: 4\n",
	},

	// Map inlining
	{
		&struct {
			A int
			C map[string]int `yaml:",inline"`
		}{1, map[string]int{"b": 2, "c": 3}},
		"a: 1\nb: 2\nc: 3\n",
	},

	// Duration
	{
		map[string]time.Duration{"a": 3 * time.Second},
		"a: 3s\n",
	},

	// Issue #24: bug in map merging logic.
	{
		map[string]string{"a": "<foo>"},
		"a: <foo>\n",
	},

	// Issue #34: marshal unsupported base 60 floats quoted for compatibility
	// with old YAML 1.1 parsers.
	{
		map[string]string{"a": "1:1"},
		"a: \"1:1\"\n",
	},

	// Binary data.
	{
		map[string]string{"a": "\x00"},
		"a: \"\\0\"\n",
	},
	{
		map[string]string{"a": "\x80\x81\x82"},
		"a: !!binary gIGC\n",
	},
	{
		map[string]string{"a": strings.Repeat("\x90", 54)},
		"a: !!binary |\n    " + strings.Repeat("kJCQ", 17) + "kJ\n    CQ\n",
	},

	// Encode unicode as utf-8 rather than in escaped form.
	{
		map[string]string{"a": "你好"},
		"a: 你好\n",
	},

	// Support encoding.TextMarshaler.
	{
		map[string]net.IP{"a": net.IPv4(1, 2, 3, 4)},
		"a: 1.2.3.4\n",
	},
	// time.Time gets a timestamp tag.
	{
		map[string]time.Time{"a": time.Date(2015, 2, 24, 18, 19, 39, 0, time.UTC)},
		"a: 2015-02-24T18:19:39Z\n",
	},
	{
		map[string]*time.Time{"a": newTime(time.Date(2015, 2, 24, 18, 19, 39, 0, time.UTC))},
		"a: 2015-02-24T18:19:39Z\n",
	},
	{
		// This is confirmed to be properly decoded in Python (libyaml) without a timestamp tag.
		map[string]time.Time{"a": time.Date(2015, 2, 24, 18, 19, 39, 123456789, time.FixedZone("FOO", -3*60*60))},
		"a: 2015-02-24T18:19:39.123456789-03:00\n",
	},
	// Ensure timestamp-like strings are quoted.
	{
		map[string]string{"a": "2015-02-24T18:19:39Z"},
		"a: \"2015-02-24T18:19:39Z\"\n",
	},

	// Ensure strings containing ": " are quoted (reported as PR #43, but not reproducible).
	{
		map[string]string{"a": "b: c"},
		"a: 'b: c'\n",
	},

	// Containing hash mark ('#') in string should be quoted
	{
		map[string]string{"a": "Hello #comment"},
		"a: 'Hello #comment'\n",
	},
	{
		map[string]string{"a": "你好 #comment"},
		"a: '你好 #comment'\n",
	},

	// Ensure MarshalYAML also gets called on the result of MarshalYAML itself.
	{
		&marshalerType{marshalerType{true}},
		"true\n",
	},
	{
		&marshalerType{&marshalerType{true}},
		"true\n",
	},

	// Check indentation of maps inside sequences inside maps.
	{
		map[string]any{"a": map[string]any{"b": []map[string]int{{"c": 1, "d": 2}}}},
		"a:\n    b:\n        - c: 1\n          d: 2\n",
	},

	// Strings with tabs were disallowed as literals (issue #471).
	{
		map[string]string{"a": "\tB\n\tC\n"},
		"a: |\n    \tB\n    \tC\n",
	},
	{
		map[string]string{"a": "\t\n\t\n"},
		"a: \"\\t\\n\\t\\n\"\n",
	},
	{
		map[string]any{"<<": []string{}},
		"\"<<\": []\n",
	},
	{
		map[string]any{"foo": "<<"},
		"foo: \"<<\"\n",
	},

	// Ensure that strings do not wrap
	{
		map[string]string{"a": "abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 1234567890 abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 1234567890 "},
		"a: 'abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 1234567890 abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 1234567890 '\n",
	},

	// yaml.Node
	{
		&struct {
			Value yaml.Node
		}{
			yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "foo",
				Style: yaml.SingleQuotedStyle,
			},
		},
		"value: 'foo'\n",
	},
	{
		yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "foo",
			Style: yaml.SingleQuotedStyle,
		},
		"'foo'\n",
	},

	// Enforced tagging with shorthand notation (issue #616).
	{
		&struct {
			Value yaml.Node
		}{
			yaml.Node{
				Kind:  yaml.ScalarNode,
				Style: yaml.TaggedStyle,
				Value: "foo",
				Tag:   "!!str",
			},
		},
		"value: !!str foo\n",
	},
	{
		&struct {
			Value yaml.Node
		}{
			yaml.Node{
				Kind:  yaml.MappingNode,
				Style: yaml.TaggedStyle,
				Tag:   "!!map",
			},
		},
		"value: !!map {}\n",
	},
	{
		&struct {
			Value yaml.Node
		}{
			yaml.Node{
				Kind:  yaml.SequenceNode,
				Style: yaml.TaggedStyle,
				Tag:   "!!seq",
			},
		},
		"value: !!seq []\n",
	},
	// bug: question mark in value
	{
		map[string]any{
			"foo": map[string]any{"bar": "a?bc"},
		},
		"foo:\n    bar: a?bc\n",
	},

	// issue https://github.com/yaml/go-yaml/issues/157
	{
		struct {
			F string `yaml:"foo"` // the correct tag, because it has `yaml` prefix
			B string `bar`        //nolint:govet // the incorrect tag, but supported
		}{
			F: "abc",
			B: "def", // value should be set using whole tag as a name, see issue: <https://github.com/yaml/go-yaml/issues/157>
		},
		"foo: abc\nbar: def\n",
	},
}

func TestMarshal(t *testing.T) {
	defer os.Setenv("TZ", os.Getenv("TZ"))
	os.Setenv("TZ", "UTC")
	for i, item := range marshalTests {
		t.Run(fmt.Sprintf("test %d: %q", i, item.data), func(t *testing.T) {
			data, err := yaml.Marshal(item.value)
			assert.NoError(t, err)
			assert.Equal(t, item.data, string(data))
		})
	}
}

func TestEncoderSingleDocument(t *testing.T) {
	for i, item := range marshalTests {
		t.Run(fmt.Sprintf("test %d. %q", i, item.data), func(t *testing.T) {
			var buf bytes.Buffer
			enc := yaml.NewEncoder(&buf)
			err := enc.Encode(item.value)
			assert.NoError(t, err)
			err = enc.Close()
			assert.NoError(t, err)
			assert.Equal(t, item.data, buf.String())
		})
	}
}

func TestEncoderMultipleDocuments(t *testing.T) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	err := enc.Encode(map[string]string{"a": "b"})
	assert.NoError(t, err)
	err = enc.Encode(map[string]string{"c": "d"})
	assert.NoError(t, err)
	err = enc.Close()
	assert.NoError(t, err)
	assert.Equal(t, "a: b\n---\nc: d\n", buf.String())
}

func TestEncoderWriteError(t *testing.T) {
	enc := yaml.NewEncoder(errorWriter{})
	err := enc.Encode(map[string]string{"a": "b"})
	assert.ErrorMatches(t, `yaml: write error: some write error`, err) // Data not flushed yet
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("some write error")
}

var marshalErrorTests = []struct {
	value any
	error string
	panic string
}{{
	value: &struct {
		B       int
		inlineB `yaml:",inline"`
	}{1, inlineB{2, inlineC{3}}},
	//nolint:dupword // struct is duplicated here as the first one is the struct and the second is the name of the inline struct
	panic: `duplicated key 'b' in struct struct \{ B int; .*`,
}, {
	value: &struct {
		A int
		B map[string]int `yaml:",inline"`
	}{1, map[string]int{"a": 2}},
	panic: `cannot have key "a" in inlined map: conflicts with struct field`,
}}

func TestMarshalErrors(t *testing.T) {
	for _, item := range marshalErrorTests {
		t.Run(item.panic, func(t *testing.T) {
			if item.panic != "" {
				assert.PanicMatches(t, item.panic, func() { yaml.Marshal(item.value) })
			} else {
				_, err := yaml.Marshal(item.value)
				assert.ErrorMatches(t, item.error, err)
			}
		})
	}
}

func TestMarshalTypeCache(t *testing.T) {
	var data []byte
	var err error
	func() {
		type T struct{ A int }
		data, err = yaml.Marshal(&T{})
		assert.NoError(t, err)
	}()
	func() {
		type T struct{ B int }
		data, err = yaml.Marshal(&T{})
		assert.NoError(t, err)
	}()
	assert.Equal(t, "b: 0\n", string(data))
}

var marshalerTests = []struct {
	data  string
	value any
}{
	{"_:\n    hi: there\n", map[any]any{"hi": "there"}},
	{"_:\n    - 1\n    - A\n", []any{1, "A"}},
	{"_: 10\n", 10},
	{"_: null\n", nil},
	{"_: BAR!\n", "BAR!"},
}

type marshalerType struct {
	value any
}

func (o marshalerType) MarshalText() ([]byte, error) {
	panic("MarshalText called on type with MarshalYAML")
}

func (o marshalerType) MarshalYAML() (any, error) {
	return o.value, nil
}

type marshalerValue struct {
	Field marshalerType `yaml:"_"`
}

func TestMarshaler(t *testing.T) {
	for _, item := range marshalerTests {
		t.Run(string(item.data), func(t *testing.T) {
			obj := &marshalerValue{}
			obj.Field.value = item.value
			data, err := yaml.Marshal(obj)
			assert.NoError(t, err)
			assert.Equal(t, string(item.data), string(data))
		})
	}
}

func TestMarshalerWholeDocument(t *testing.T) {
	obj := &marshalerType{}
	obj.value = map[string]string{"hello": "world!"}
	data, err := yaml.Marshal(obj)
	assert.NoError(t, err)
	assert.Equal(t, "hello: world!\n", string(data))
}

type failingMarshaler struct{}

func (ft *failingMarshaler) MarshalYAML() (any, error) {
	return nil, errFailing
}

func TestMarshalerError(t *testing.T) {
	_, err := yaml.Marshal(&failingMarshaler{})
	assert.ErrorIs(t, errFailing, err)
}

func TestSetIndent(t *testing.T) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(8)
	err := enc.Encode(map[string]any{"a": map[string]any{"b": map[string]string{"c": "d"}}})
	assert.NoError(t, err)
	err = enc.Close()
	assert.NoError(t, err)
	assert.Equal(t, "a:\n        b:\n                c: d\n", buf.String())
}

func TestSortedOutput(t *testing.T) {
	order := []any{
		false,
		true,
		1,
		uint(1),
		1.0,
		1.1,
		1.2,
		2,
		uint(2),
		2.0,
		2.1,
		"",
		".1",
		".2",
		".a",
		"1",
		"2",
		"a!10",
		"a/0001",
		"a/002",
		"a/3",
		"a/10",
		"a/11",
		"a/0012",
		"a/100",
		"a~10",
		"ab/1",
		"b/1",
		"b/01",
		"b/2",
		"b/02",
		"b/3",
		"b/03",
		"b1",
		"b01",
		"b3",
		"c2.10",
		"c10.2",
		"d1",
		"d7",
		"d7abc",
		"d12",
		"d12a",
		"e2b",
		"e4b",
		"e21a",
	}
	m := make(map[any]int)
	for _, k := range order {
		m[k] = 1
	}
	data, err := yaml.Marshal(m)
	assert.NoError(t, err)
	out := "\n" + string(data)
	last := 0
	for i, k := range order {
		repr := fmt.Sprint(k)
		if s, ok := k.(string); ok {
			if _, err = strconv.ParseFloat(repr, 32); s == "" || err == nil {
				repr = `"` + repr + `"`
			}
		}
		index := strings.Index(out, "\n"+repr+":")
		if index == -1 {
			t.Fatalf("%#v is not in the output: %#v", k, out)
		}
		if index < last {
			t.Fatalf("%#v was generated before %#v: %q", k, order[i-1], out)
		}
		last = index
	}
}

func newTime(t time.Time) *time.Time {
	return &t
}

func TestCompactSeqIndentDefault(t *testing.T) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.CompactSeqIndent()
	err := enc.Encode(map[string]any{"a": []string{"b", "c"}})
	assert.NoError(t, err)
	err = enc.Close()
	assert.NoError(t, err)
	// The default indent is 4, so these sequence elements get 2 indents as before
	assert.Equal(t, `a:
  - b
  - c
`, buf.String())
}

func TestCompactSequenceWithSetIndent(t *testing.T) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.CompactSeqIndent()
	enc.SetIndent(2)
	err := enc.Encode(map[string]any{"a": []string{"b", "c"}})
	assert.NoError(t, err)
	err = enc.Close()
	assert.NoError(t, err)
	// The sequence indent is 2, so these sequence elements don't get indented at all
	assert.Equal(t, `a:
- b
- c
`, buf.String())
}

type (
	normal  string
	compact string
)

// newlinePlusNormalToNewlinePlusCompact maps the normal encoding (prefixed with a newline)
// to the compact encoding (prefixed with a newline), for test cases in marshalTests
var newlinePlusNormalToNewlinePlusCompact = map[normal]compact{
	normal(`
v:
    - A
    - B
`): compact(`
v:
  - A
  - B
`),

	normal(`
v:
    - A
    - |-
      B
      C
`): compact(`
v:
  - A
  - |-
    B
    C
`),

	normal(`
v:
    - A
    - 1
    - B:
        - 2
        - 3
`): compact(`
v:
  - A
  - 1
  - B:
      - 2
      - 3
`),

	normal(`
a:
    - 1
    - 2
`): compact(`
a:
  - 1
  - 2
`),

	normal(`
a:
    b:
        - c: 1
          d: 2
`): compact(`
a:
    b:
      - c: 1
        d: 2
`),

	normal(`
v:
    - v1: |-

        hi
`): compact(`
v:
  - v1: |-

        hi
`),
}

func TestEncoderCompactIndents(t *testing.T) {
	for i, item := range marshalTests {
		t.Run(fmt.Sprintf("test %d. %q", i, item.data), func(t *testing.T) {
			var buf bytes.Buffer
			enc := yaml.NewEncoder(&buf)
			enc.CompactSeqIndent()
			err := enc.Encode(item.value)
			assert.NoError(t, err)
			err = enc.Close()
			assert.NoError(t, err)

			// Default to expecting the item data
			expected := item.data
			// If there's a different compact representation, use that
			if c, ok := newlinePlusNormalToNewlinePlusCompact[normal("\n"+item.data)]; ok {
				expected = string(c[1:])
			}

			assert.Equal(t, expected, buf.String())
		})
	}
}

func TestNewLinePreserved(t *testing.T) {
	obj := &marshalerValue{}
	obj.Field.value = "a:\n        b:\n                c: d\n"
	data, err := yaml.Marshal(obj)
	assert.NoError(t, err)
	assert.Equal(t, "_: |\n    a:\n            b:\n                    c: d\n", string(data))

	obj.Field.value = "\na:\n        b:\n                c: d\n"
	data, err = yaml.Marshal(obj)
	assert.NoError(t, err)
	// the newline at the start of the file should be preserved
	assert.Equal(t, "_: |\n\n    a:\n            b:\n                    c: d\n", string(data))
}

func TestScalarStyleRules(t *testing.T) {
	// Test cases for the new scalar style rules
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			"\t\n",
			"\"\\t\\n\"\n",
			"Tab + newline - should be double quoted (short, starts with whitespace)",
		},
		{
			"\n",
			"\"\\n\"\n",
			"Just newline - should be double quoted (no non-ws chars)",
		},
		{
			"\t",
			"\"\\t\"\n",
			"Just tab - should be double quoted (control char)",
		},
		{
			"hello\nworld",
			"|-\n    hello\n    world\n",
			"Text with newline - should be literal (>= 2 chars, doesn't start with whitespace)",
		},
		{
			"hello\tworld",
			"\"hello\\tworld\"\n",
			"Text with tab - should be double quoted (control char)",
		},
		{
			"hello",
			"hello\n",
			"Simple text - should be plain",
		},
		{
			"123",
			"\"123\"\n",
			"Number-like - should be quoted (looks like number)",
		},
		{
			"true",
			"\"true\"\n",
			"Boolean-like - should be quoted (looks like boolean)",
		},
		{
			"This is a longer string\nwith multiple lines\nthat should use literal style",
			"|-\n    This is a longer string\n    with multiple lines\n    that should use literal style\n",
			"Long multi-line - should be literal",
		},
		{
			" This starts with space\nand is long enough\nfor literal style",
			"|4-\n     This starts with space\n    and is long enough\n    for literal style\n",
			"Long multi-line starting with space - should be literal (>= 6 chars)",
		},
		{
			"\tThis starts with tab\nand is long enough\nfor literal style",
			"|-\n    \tThis starts with tab\n    and is long enough\n    for literal style\n",
			"Long multi-line starting with tab - should be literal (>= 6 chars)",
		},
		{
			"\tB\n\tC\n",
			"|\n    \tB\n    \tC\n",
			"Tab + B + newline + tab + C + newline - should be literal (6 chars)",
		},
		{
			"a\n",
			"|\n    a\n",
			"Single char + newline - should be literal (2 chars, has content)",
		},
		{
			"a\nb",
			"|-\n    a\n    b\n",
			"Two chars with newline - should be literal (3 chars, has content)",
		},
		{
			" a\n",
			"|4\n     a\n",
			"Space + char + newline - should be literal (3 chars, has content)",
		},
		{
			"\ta\n",
			"|\n    \ta\n",
			"Tab + char + newline - should be literal (3 chars, has content)",
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test_%d_%s", i, testCase.desc), func(t *testing.T) {
			data, err := yaml.Marshal(testCase.input)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, string(data))
		})
	}
}

func TestWhitespaceOnlyStrings(t *testing.T) {
	// Test cases for whitespace-only strings that should not use literal style
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			"\n",
			"\"\\n\"\n",
			"Just newline - should be double quoted (no non-ws chars)",
		},
		{
			"\n\n",
			"\"\\n\\n\"\n",
			"Two newlines - should be double quoted (no non-ws chars)",
		},
		{
			" \n",
			"\" \\n\"\n",
			"Space + newline - should be double quoted (no non-ws chars)",
		},
		{
			"\t\n",
			"\"\\t\\n\"\n",
			"Tab + newline - should be double quoted (no non-ws chars)",
		},
		{
			" \n ",
			"\" \\n \"\n",
			"Space + newline + space - should be double quoted (no non-ws chars)",
		},
		{
			"\n \n",
			"\"\\n \\n\"\n",
			"Newline + space + newline - should be double quoted (no non-ws chars)",
		},
		{
			"\t \n\t",
			"\"\\t \\n\\t\"\n",
			"Tab + space + newline + tab - should be double quoted (no non-ws chars)",
		},
		{
			"   \n   ",
			"\"   \\n   \"\n",
			"Multiple spaces + newline + multiple spaces - should be double quoted (no non-ws chars)",
		},
		{
			"\n\n\n",
			"\"\\n\\n\\n\"\n",
			"Three newlines - should be double quoted (no non-ws chars)",
		},
		{
			" \t\n \t",
			"\" \\t\\n \\t\"\n",
			"Space + tab + newline + space + tab - should be double quoted (no non-ws chars)",
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test_%d_%s", i, testCase.desc), func(t *testing.T) {
			data, err := yaml.Marshal(testCase.input)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, string(data))
		})
	}
}

func TestWhitespaceWithContent(t *testing.T) {
	// Test cases for strings with whitespace AND content that should use literal style
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			"hello\n",
			"|\n    hello\n",
			"Text + newline - should be literal (has non-ws chars)",
		},
		{
			" hello\n",
			"|4\n     hello\n",
			"Space + text + newline - should be literal (has non-ws chars)",
		},
		{
			" \nhello",
			"\" \\nhello\"\n",
			"Space + newline + text - should be double quoted (short, starts with whitespace)",
		},
		{
			"\thello\n",
			"|\n    \thello\n",
			"Tab + text + newline - should be literal (has non-ws chars)",
		},
		{
			"\t\nhello",
			"|-\n    \t\n    hello\n",
			"Tab + newline + text - should be literal (has non-ws chars)",
		},
		{
			"  hello  \n",
			"\"  hello  \\n\"\n",
			"Multiple spaces + text + spaces + newline - should be double quoted (ends with spaces)",
		},
		{
			"hello  \n",
			"\"hello  \\n\"\n",
			"Text + spaces + newline - should be double quoted (ends with spaces)",
		},
		{
			"hello\n  ",
			"\"hello\\n  \"\n",
			"Text + newline + spaces - should be double quoted (ends with spaces)",
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test_%d_%s", i, testCase.desc), func(t *testing.T) {
			data, err := yaml.Marshal(testCase.input)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, string(data))
		})
	}
}

func TestUnicodeWhitespaceHandling(t *testing.T) {
	// Test cases for Unicode whitespace characters that should be properly handled
	// by the shouldUseLiteralStyle function using unicode.IsSpace()
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		// Unicode whitespace characters
		{
			"hello\u00A0\n", // non-breaking space
			"|\n    hello\u00A0\n",
			"Non-breaking space with content - should use literal style",
		},
		{
			"\u00A0\n", // non-breaking space
			"\"\u00A0\\n\"\n",
			"Non-breaking space only - should not use literal style",
		},
		{
			"hello\u2000\n", // en quad
			"|\n    hello\u2000\n",
			"En quad with content - should use literal style",
		},
		{
			"\u2000\n", // en quad
			"\"\u2000\\n\"\n",
			"En quad only - should not use literal style",
		},
		{
			"hello\u2001\n", // em quad
			"|\n    hello\u2001\n",
			"Em quad with content - should use literal style",
		},
		{
			"\u2001\n", // em quad
			"\"\u2001\\n\"\n",
			"Em quad only - should not use literal style",
		},
		{
			"hello\u2002\n", // en space
			"|\n    hello\u2002\n",
			"En space with content - should use literal style",
		},
		{
			"\u2002\n", // en space
			"\"\u2002\\n\"\n",
			"En space only - should not use literal style",
		},
		{
			"hello\u2003\n", // em space
			"|\n    hello\u2003\n",
			"Em space with content - should use literal style",
		},
		{
			"\u2003\n", // em space
			"\"\u2003\\n\"\n",
			"Em space only - should not use literal style",
		},
		{
			"hello\u2004\n", // three-per-em space
			"|\n    hello\u2004\n",
			"Three-per-em space with content - should use literal style",
		},
		{
			"\u2004\n", // three-per-em space
			"\"\u2004\\n\"\n",
			"Three-per-em space only - should not use literal style",
		},
		{
			"hello\u2005\n", // four-per-em space
			"|\n    hello\u2005\n",
			"Four-per-em space with content - should use literal style",
		},
		{
			"\u2005\n", // four-per-em space
			"\"\u2005\\n\"\n",
			"Four-per-em space only - should not use literal style",
		},
		{
			"hello\u2006\n", // six-per-em space
			"|\n    hello\u2006\n",
			"Six-per-em space with content - should use literal style",
		},
		{
			"\u2006\n", // six-per-em space
			"\"\u2006\\n\"\n",
			"Six-per-em space only - should not use literal style",
		},
		{
			"hello\u2007\n", // figure space
			"|\n    hello\u2007\n",
			"Figure space with content - should use literal style",
		},
		{
			"\u2007\n", // figure space
			"\"\u2007\\n\"\n",
			"Figure space only - should not use literal style",
		},
		{
			"hello\u2008\n", // punctuation space
			"|\n    hello\u2008\n",
			"Punctuation space with content - should use literal style",
		},
		{
			"\u2008\n", // punctuation space
			"\"\u2008\\n\"\n",
			"Punctuation space only - should not use literal style",
		},
		{
			"hello\u2009\n", // thin space
			"|\n    hello\u2009\n",
			"Thin space with content - should use literal style",
		},
		{
			"\u2009\n", // thin space
			"\"\u2009\\n\"\n",
			"Thin space only - should not use literal style",
		},
		{
			"hello\u200A\n", // hair space
			"|\n    hello\u200A\n",
			"Hair space with content - should use literal style",
		},
		{
			"\u200A\n", // hair space
			"\"\u200A\\n\"\n",
			"Hair space only - should not use literal style",
		},
		// Other Unicode whitespace
		{
			"hello\u2028\n", // line separator
			"|+\n    hello\u2028\n",
			"Line separator with content - should use literal style",
		},
		{
			"\u2028\n", // line separator
			"\"\\L\\n\"\n",
			"Line separator only - should not use literal style",
		},
		{
			"hello\u2029\n", // paragraph separator
			"|+\n    hello\u2029\n",
			"Paragraph separator with content - should use literal style",
		},
		{
			"\u2029\n", // paragraph separator
			"\"\\P\\n\"\n",
			"Paragraph separator only - should not use literal style",
		},
		{
			"hello\u205F\n", // medium mathematical space
			"|\n    hello\u205F\n",
			"Medium mathematical space with content - should use literal style",
		},
		{
			"\u205F\n", // medium mathematical space
			"\"\u205F\\n\"\n",
			"Medium mathematical space only - should not use literal style",
		},
		{
			"hello\u3000\n", // ideographic space
			"|\n    hello\u3000\n",
			"Ideographic space with content - should use literal style",
		},
		{
			"\u3000\n", // ideographic space
			"\"\u3000\\n\"\n",
			"Ideographic space only - should not use literal style",
		},
		// Mixed Unicode whitespace
		{
			"hello\u00A0\u2000\u2001\n", // mixed Unicode spaces
			"|\n    hello\u00A0\u2000\u2001\n",
			"Mixed Unicode spaces with content - should use literal style",
		},
		{
			"\u00A0\u2000\u2001\n", // mixed Unicode spaces
			"\"\u00A0\u2000\u2001\\n\"\n",
			"Mixed Unicode spaces only - should not use literal style",
		},
		// Unicode whitespace with ASCII whitespace
		{
			"hello \u00A0\t\n", // ASCII + Unicode spaces
			"|\n    hello \u00A0\t\n",
			"ASCII + Unicode spaces with content - should use literal style",
		},
		{
			" \u00A0\t\n", // ASCII + Unicode spaces
			"\" \u00A0\\t\\n\"\n",
			"ASCII + Unicode spaces only - should not use literal style",
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test_%d_%s", i, testCase.desc), func(t *testing.T) {
			data, err := yaml.Marshal(testCase.input)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, string(data))
		})
	}
}
