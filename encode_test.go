// Copyright 2011-2019 Canonical Ltd
// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

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
	"go.yaml.in/yaml/v4/internal/libyaml"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
	"go.yaml.in/yaml/v4/internal/testutil/datatest"
)

var marshalIntTest = 123

var (
	encodeTypeRegistry  = datatest.NewTypeRegistry()
	encodeValueRegistry = datatest.NewValueRegistry()
)

func init() {
	// Register basic types
	encodeTypeRegistry.Register("string", "")

	// Register map types
	encodeTypeRegistry.Register("map[string]string", map[string]string{})
	encodeTypeRegistry.Register("map[string]any", map[string]any{})
	encodeTypeRegistry.Register("map[string]uint", map[string]uint{})
	encodeTypeRegistry.Register("map[string]int64", map[string]int64{})
	encodeTypeRegistry.Register("map[string]uint64", map[string]uint64{})
	encodeTypeRegistry.Register("map[string][]string", map[string][]string{})
	encodeTypeRegistry.Register("map[string][]any", map[string][]any{})
	encodeTypeRegistry.Register("map[string][]map[string]any", map[string][]map[string]any{})

	// Register struct types
	encodeTypeRegistry.Register("testStructHello", testStructHello{})
	encodeTypeRegistry.Register("testStructA_Int", testStructA_Int{})
	encodeTypeRegistry.Register("testStructA_Float64", testStructA_Float64{})
	encodeTypeRegistry.Register("testStructA_Bool", testStructA_Bool{})
	encodeTypeRegistry.Register("testStructA_String", testStructA_String{})
	encodeTypeRegistry.Register("testStructA_IntSlice", testStructA_IntSlice{})
	encodeTypeRegistry.Register("testStructA_IntArray2", testStructA_IntArray2{})
	encodeTypeRegistry.Register("testStructA_NestedB", testStructA_NestedB{})
	encodeTypeRegistry.Register("testStructA_NestedBPtr", testStructA_NestedBPtr{})
	encodeTypeRegistry.Register("testStructEmpty", testStructEmpty{})

	// Register struct types with yaml tags
	encodeTypeRegistry.Register("testStructB_Int_TagA", testStructB_Int_TagA{})

	// Register value constants
	encodeValueRegistry.Register("+Inf", math.Inf(+1))
	encodeValueRegistry.Register("-Inf", math.Inf(-1))
	encodeValueRegistry.Register("NaN", math.NaN())
	encodeValueRegistry.Register("-0", negativeZero)
}

var marshalTests = []struct {
	value any
	data  string
}{
	{
		(*marshalerType)(nil),
		"null\n",
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

	// Binary data.
	{
		map[string]string{"a": "\x80\x81\x82"},
		"a: !!binary gIGC\n",
	},
	{
		map[string]string{"a": strings.Repeat("\x90", 54)},
		"a: !!binary |\n    " + strings.Repeat("kJCQ", 17) + "kJ\n    CQ\n",
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

	// Ensure MarshalYAML also gets called on the result of MarshalYAML itself.
	{
		&marshalerType{marshalerType{true}},
		"true\n",
	},
	{
		&marshalerType{&marshalerType{true}},
		"true\n",
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

func TestEncodeToYAML(t *testing.T) {
	datatest.RunTestCases(t, func() ([]map[string]any, error) {
		return datatest.LoadTestCasesFromFile("testdata/encode.yaml", libyaml.LoadYAML)
	}, map[string]datatest.TestHandler{
		"encode": runEncodeTest,
	})
}

func runEncodeTest(t *testing.T, tc map[string]any) {
	t.Helper()

	// Get type and create instance
	// Note: "type" field in YAML is renamed to "output_type" during normalization
	// to avoid conflict with the test type ("encode")
	typeName := tc["output_type"].(string)
	data := tc["data"]

	// Create pointer target of the specified type (for addressability)
	targetPtr, err := encodeTypeRegistry.NewPointerInstance(typeName)
	if err != nil {
		t.Fatalf("Failed to create instance of type %s: %v", typeName, err)
	}

	// Resolve value constants (like +Inf, -Inf, NaN, -0)
	resolvedData := encodeValueRegistry.Resolve(data)

	// Unmarshal the data into the target to populate it
	dataBytes, err := yaml.Marshal(resolvedData)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	err = yaml.Unmarshal(dataBytes, targetPtr)
	if err != nil {
		t.Fatalf("Failed to unmarshal data into target: %v", err)
	}

	// Marshal the target back to YAML
	output, err := yaml.Marshal(targetPtr)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Compare with expected output
	want := tc["want"].(string)
	assert.Equal(t, want, string(output))
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

// Scalar style tests for complex whitespace (tabs and Unicode)
// These tests are kept in Go because they involve whitespace characters
// that are difficult to represent accurately in YAML test data files.

func TestScalarStyleWithTabs(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			"\t\n",
			"\"\\t\\n\"\n",
			"Tab + newline",
		},
		{
			"\t",
			"\"\\t\"\n",
			"Just tab",
		},
		{
			"hello\tworld",
			"\"hello\\tworld\"\n",
			"Text with tab",
		},
		{
			"\tThis starts with tab\nand is long enough\nfor literal style",
			"|-\n    \tThis starts with tab\n    and is long enough\n    for literal style\n",
			"Multiline starting with tab",
		},
		{
			"\tB\n\tC\n",
			"|\n    \tB\n    \tC\n",
			"Tab B newline tab C newline",
		},
		{
			"\ta\n",
			"|\n    \ta\n",
			"Tab + char + newline",
		},
		{
			"\thello\n",
			"|\n    \thello\n",
			"Tab + text + newline",
		},
		{
			"\t\nhello",
			"|-\n    \t\n    hello\n",
			"Tab + newline + text",
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
			"Non-breaking space with content",
		},
		{
			"\u00A0\n", // non-breaking space
			"\"\u00A0\\n\"\n",
			"Non-breaking space only",
		},
		{
			"hello\u2000\n", // en quad
			"|\n    hello\u2000\n",
			"En quad with content",
		},
		{
			"\u2000\n", // en quad
			"\"\u2000\\n\"\n",
			"En quad only",
		},
		{
			"hello\u2001\n", // em quad
			"|\n    hello\u2001\n",
			"Em quad with content",
		},
		{
			"\u2001\n", // em quad
			"\"\u2001\\n\"\n",
			"Em quad only",
		},
		{
			"hello\u2002\n", // en space
			"|\n    hello\u2002\n",
			"En space with content",
		},
		{
			"\u2002\n", // en space
			"\"\u2002\\n\"\n",
			"En space only",
		},
		{
			"hello\u2003\n", // em space
			"|\n    hello\u2003\n",
			"Em space with content",
		},
		{
			"\u2003\n", // em space
			"\"\u2003\\n\"\n",
			"Em space only",
		},
		{
			"hello\u2004\n", // three-per-em space
			"|\n    hello\u2004\n",
			"Three-per-em space with content",
		},
		{
			"\u2004\n", // three-per-em space
			"\"\u2004\\n\"\n",
			"Three-per-em space only",
		},
		{
			"hello\u2005\n", // four-per-em space
			"|\n    hello\u2005\n",
			"Four-per-em space with content",
		},
		{
			"\u2005\n", // four-per-em space
			"\"\u2005\\n\"\n",
			"Four-per-em space only",
		},
		{
			"hello\u2006\n", // six-per-em space
			"|\n    hello\u2006\n",
			"Six-per-em space with content",
		},
		{
			"\u2006\n", // six-per-em space
			"\"\u2006\\n\"\n",
			"Six-per-em space only",
		},
		{
			"hello\u2007\n", // figure space
			"|\n    hello\u2007\n",
			"Figure space with content",
		},
		{
			"\u2007\n", // figure space
			"\"\u2007\\n\"\n",
			"Figure space only",
		},
		{
			"hello\u2008\n", // punctuation space
			"|\n    hello\u2008\n",
			"Punctuation space with content",
		},
		{
			"\u2008\n", // punctuation space
			"\"\u2008\\n\"\n",
			"Punctuation space only",
		},
		{
			"hello\u2009\n", // thin space
			"|\n    hello\u2009\n",
			"Thin space with content",
		},
		{
			"\u2009\n", // thin space
			"\"\u2009\\n\"\n",
			"Thin space only",
		},
		{
			"hello\u200A\n", // hair space
			"|\n    hello\u200A\n",
			"Hair space with content",
		},
		{
			"\u200A\n", // hair space
			"\"\u200A\\n\"\n",
			"Hair space only",
		},
		// Other Unicode whitespace
		{
			"hello\u2028\n", // line separator
			"|+\n    hello\u2028\n",
			"Line separator with content",
		},
		{
			"\u2028\n", // line separator
			"\"\\L\\n\"\n",
			"Line separator only",
		},
		{
			"hello\u2029\n", // paragraph separator
			"|+\n    hello\u2029\n",
			"Paragraph separator with content",
		},
		{
			"\u2029\n", // paragraph separator
			"\"\\P\\n\"\n",
			"Paragraph separator only",
		},
		{
			"hello\u205F\n", // medium mathematical space
			"|\n    hello\u205F\n",
			"Medium mathematical space with content",
		},
		{
			"\u205F\n", // medium mathematical space
			"\"\u205F\\n\"\n",
			"Medium mathematical space only",
		},
		{
			"hello\u3000\n", // ideographic space
			"|\n    hello\u3000\n",
			"Ideographic space with content",
		},
		{
			"\u3000\n", // ideographic space
			"\"\u3000\\n\"\n",
			"Ideographic space only",
		},
		// Mixed Unicode whitespace
		{
			"hello\u00A0\u2000\u2001\n", // mixed Unicode spaces
			"|\n    hello\u00A0\u2000\u2001\n",
			"Mixed Unicode spaces with content",
		},
		{
			"\u00A0\u2000\u2001\n", // mixed Unicode spaces
			"\"\u00A0\u2000\u2001\\n\"\n",
			"Mixed Unicode spaces only",
		},
		// Unicode whitespace with ASCII whitespace
		{
			"hello \u00A0\t\n", // ASCII + Unicode spaces
			"|\n    hello \u00A0\t\n",
			"ASCII + Unicode spaces with content",
		},
		{
			" \u00A0\t\n", // ASCII + Unicode spaces
			"\" \u00A0\\t\\n\"\n",
			"ASCII + Unicode spaces only",
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
