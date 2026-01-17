// Copyright 2011-2019 Canonical Ltd
// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for YAML marshal/unmarshal functionality, including struct tags,
// type conversions, anchors/aliases, and edge cases.

package yaml_test

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
	"go.yaml.in/yaml/v4/internal/testutil/datatest"
)

// negativeZero represents -0.0 for YAML test cases
// this is needed because Go constants cannot express -0.0
// https://staticcheck.dev/docs/checks/#SA4026
var negativeZero = math.Copysign(0.0, -1.0)

var unmarshalIntTest = 123

// archSafeInt returns v as int if it fits in the architecture's int type,
// otherwise returns int64.
func archSafeInt(v int64) any {
	if strconv.IntSize == 64 || math.MinInt32 <= v && v <= math.MaxInt32 {
		return int(v) // int is safe
	}

	// on 32-bit systems, and v overflows int, we need to return an int64
	return int64(v)
}

// Named struct types for data-driven tests
type (
	testStructHello                struct{ Hello string }
	testStructA_Int                struct{ A int }
	testStructA_Float64            struct{ A float64 }
	testStructA_Uint               struct{ A uint }
	testStructA_Bool               struct{ A bool }
	testStructA_IntSlice           struct{ A []int }
	testStructA_IntArray2          struct{ A [2]int }
	testStructA_MapStringString    struct{ A map[string]string }
	testStructA_MapStringStringPtr struct{ A *map[string]string }
	testStructB_Int                struct{ B int }
	nestedStructB                  struct{ B string }
	testStructA_NestedB            struct{ A nestedStructB }
	testStructA_NestedBPtr         struct{ A *nestedStructB }
	testStructABCD_Int             struct{ A, B, C, D int }
	testStructB_IntSlice           struct{ B []int }
	testStructA_IntSliceEmpty      struct{ A []int }
	testStructA_String             struct{ A string }
	testStructEmpty                struct{}
)

// Types with yaml struct tags
type (
	testStructB_Int_TagA struct {
		B int `yaml:"a"`
	}
	testStructAB_Int_BIgnored struct {
		A int
		B int `yaml:"-"`
	}
	testStructA_Int_InlineB struct {
		A int
		C inlineB `yaml:",inline"`
	}
	testStructA_Int_InlineBPtr struct {
		A int
		C *inlineB `yaml:",inline"`
	}
	testStructA_Int_InlineDPtr struct {
		A int
		C *inlineD `yaml:",inline"`
	}
	testStructA_Int_InlineMapStringInt struct {
		A int
		C map[string]int `yaml:",inline"`
	}
)

// simpleTextUnmarshaler is a simple type implementing encoding.TextUnmarshaler
// for testing TextUnmarshaler validation.
type simpleTextUnmarshaler struct {
	Value string
}

func (s *simpleTextUnmarshaler) UnmarshalText(text []byte) error {
	s.Value = string(text)
	return nil
}

// Test types for TextUnmarshaler validation
type (
	testStructA_TextUnmarshaler struct {
		A simpleTextUnmarshaler
	}
	testStructA_TextUnmarshalerPtr struct {
		A *simpleTextUnmarshaler
	}
	testStructA_TextUnmarshalerPtrPtr struct {
		A **simpleTextUnmarshaler
	}
)

// Type and value registries for data-driven tests
var (
	decodeTypes  = datatest.NewTypeRegistry()
	decodeValues = datatest.NewValueRegistry()
)

func init() {
	// Register basic map types
	decodeTypes.RegisterFactory("map[string]string", func() any {
		return make(map[string]string)
	})
	decodeTypes.RegisterFactory("map[string]any", func() any {
		return make(map[string]any)
	})
	decodeTypes.RegisterFactory("map[string]int64", func() any {
		return make(map[string]int64)
	})
	decodeTypes.RegisterFactory("map[string]float64", func() any {
		return make(map[string]float64)
	})
	decodeTypes.RegisterFactory("map[string][]byte", func() any {
		return make(map[string][]byte)
	})
	decodeTypes.RegisterFactory("map[any]any", func() any {
		return make(map[any]any)
	})

	// Register slice types
	decodeTypes.RegisterFactory("[]string", func() any {
		return []string{}
	})
	decodeTypes.RegisterFactory("[]int", func() any {
		return []int{}
	})
	decodeTypes.RegisterFactory("[]any", func() any {
		return []any{}
	})

	// Register primitive types
	decodeTypes.RegisterFactory("string", func() any {
		return ""
	})

	// Register map types with slice values
	decodeTypes.RegisterFactory("map[string][]string", func() any {
		return make(map[string][]string)
	})
	decodeTypes.RegisterFactory("map[string][]int", func() any {
		return make(map[string][]int)
	})

	// Register additional map types
	decodeTypes.RegisterFactory("map[string]bool", func() any {
		return make(map[string]bool)
	})
	decodeTypes.RegisterFactory("map[string]int", func() any {
		return make(map[string]int)
	})
	decodeTypes.RegisterFactory("map[any]string", func() any {
		return make(map[any]string)
	})
	decodeTypes.RegisterFactory("map[string]uint", func() any {
		return make(map[string]uint)
	})
	decodeTypes.RegisterFactory("map[string]uint64", func() any {
		return make(map[string]uint64)
	})
	decodeTypes.RegisterFactory("map[string]int32", func() any {
		return make(map[string]int32)
	})
	decodeTypes.RegisterFactory("map[string]int8", func() any {
		return make(map[string]int8)
	})
	decodeTypes.RegisterFactory("map[string]float32", func() any {
		return make(map[string]float32)
	})

	// Register struct types
	decodeTypes.Register("testStructHello", testStructHello{})
	decodeTypes.Register("testStructA_Int", testStructA_Int{})
	decodeTypes.Register("testStructA_Float64", testStructA_Float64{})
	decodeTypes.Register("testStructA_Uint", testStructA_Uint{})
	decodeTypes.Register("testStructA_Bool", testStructA_Bool{})
	decodeTypes.Register("testStructA_IntSlice", testStructA_IntSlice{})
	decodeTypes.Register("testStructA_IntArray2", testStructA_IntArray2{})
	decodeTypes.Register("testStructA_MapStringString", testStructA_MapStringString{})
	decodeTypes.Register("testStructA_MapStringStringPtr", testStructA_MapStringStringPtr{})
	decodeTypes.Register("testStructB_Int", testStructB_Int{})
	decodeTypes.Register("testStructA_NestedB", testStructA_NestedB{})
	decodeTypes.Register("testStructA_NestedBPtr", testStructA_NestedBPtr{})
	decodeTypes.Register("testStructABCD_Int", testStructABCD_Int{})
	decodeTypes.Register("testStructB_IntSlice", testStructB_IntSlice{})
	decodeTypes.Register("testStructA_IntSliceEmpty", testStructA_IntSliceEmpty{})
	decodeTypes.Register("testStructA_String", testStructA_String{})
	decodeTypes.Register("testStructEmpty", testStructEmpty{})

	// Register struct types with yaml tags
	decodeTypes.Register("testStructB_Int_TagA", testStructB_Int_TagA{})
	decodeTypes.Register("testStructAB_Int_BIgnored", testStructAB_Int_BIgnored{})
	decodeTypes.Register("testStructA_Int_InlineB", testStructA_Int_InlineB{})
	decodeTypes.Register("testStructA_Int_InlineBPtr", testStructA_Int_InlineBPtr{})
	decodeTypes.Register("testStructA_Int_InlineDPtr", testStructA_Int_InlineDPtr{})
	decodeTypes.Register("testStructA_Int_InlineMapStringInt", testStructA_Int_InlineMapStringInt{})

	// Register TextUnmarshaler test types
	decodeTypes.Register("testStructA_TextUnmarshaler", testStructA_TextUnmarshaler{})
	decodeTypes.Register("testStructA_TextUnmarshalerPtr", testStructA_TextUnmarshalerPtr{})
	decodeTypes.Register("testStructA_TextUnmarshalerPtrPtr", testStructA_TextUnmarshalerPtrPtr{})

	// Register math constants
	decodeValues.Register("+Inf", math.Inf(+1))
	decodeValues.Register("-Inf", math.Inf(-1))
	decodeValues.Register("NaN", math.NaN())
	decodeValues.Register("-0", negativeZero)

	// Register math limit constants
	decodeValues.Register("MaxInt32", int(math.MaxInt32))
	decodeValues.Register("MinInt32", int(math.MinInt32))
	decodeValues.Register("MaxInt64", int64(math.MaxInt64))
	decodeValues.Register("MinInt64", int64(math.MinInt64))
	decodeValues.Register("MaxUint32", uint(math.MaxUint32))
	decodeValues.Register("MaxUint64", uint64(math.MaxUint64))
	decodeValues.Register("MaxFloat32", math.MaxFloat32)
	decodeValues.Register("MaxFloat64", math.MaxFloat64)
	decodeValues.Register("SmallestNonzeroFloat32", math.SmallestNonzeroFloat32)
	decodeValues.Register("SmallestNonzeroFloat64", math.SmallestNonzeroFloat64)
}

var unmarshalTests = []struct {
	data  string
	value any
}{
	{
		"",
		(*struct{})(nil),
	},
	{
		"{}", &struct{}{},
	},

	// Simple values.
	{
		"123",
		&unmarshalIntTest,
	},
	{
		"-0",
		negativeZero,
	},
	{
		"\"\\t\\n\"\n",
		"\t\n",
	},

	// Cross-architecture numeric tests
	{
		"bin: -0b1000000000000000000000000000000000000000000000000000000000000000",
		map[string]any{"bin": archSafeInt(math.MinInt64)},
	},
	{
		// When unmarshaling into map[string]int64, values that overflow int64
		// cannot be decoded and result in an empty map.
		"int_overflow: 9223372036854775808", // math.MaxInt64 + 1
		map[string]int64{},
	},

	// Structs and type conversions.
	{
		"a: 'null'",
		&struct{ A *unmarshalerType }{&unmarshalerType{"null"}},
	},

	// Anchors and aliases.
	{
		"a: &x 1\nb: &y 2\nc: *x\nd: *y\n",
		&struct{ A, B, C, D int }{1, 2, 1, 2},
	},
	{
		"a: &a {c: 1}\nb: *a",
		&struct {
			A, B struct {
				C int
			}
		}{struct{ C int }{1}, struct{ C int }{1}},
	},
	{
		"a: &a [1, 2]\nb: *a",
		&struct{ B []int }{[]int{1, 2}},
	},
	{
		"a: &a.b1.c [1, 2]\nb: *a.b1.c",
		&struct{ B []int }{[]int{1, 2}},
	},

	// Bug https://github.com/yaml/go-yaml/issues/109
	// Bug #1133337
	{
		"foo: ''",
		map[string]*string{"foo": new(string)},
	},
	{
		"foo: null",
		map[string]*string{"foo": nil},
	},

	// Support for ~
	{
		"foo: ~",
		map[string]*string{"foo": nil},
	},

	// Ignored field
	{
		"a: 1\nb: 2\n",
		&struct {
			A int
			B int `yaml:"-"`
		}{1, 0},
	},

	// Bug #1191981
	{
		"" +
			"%YAML 1.1\n" +
			"--- !!str\n" +
			`"Generic line break (no glyph)\n\` + "\n" +
			` Generic line break (glyphed)\n\` + "\n" +
			` Line separator\u2028\` + "\n" +
			` Paragraph separator\u2029"` + "\n",
		"" +
			"Generic line break (no glyph)\n" +
			"Generic line break (glyphed)\n" +
			"Line separator\u2028Paragraph separator\u2029",
	},

	// Duration
	{
		"a: 3s",
		map[string]time.Duration{"a": 3 * time.Second},
	},
	// Zero duration as a string.
	{
		"a: '0'",
		map[string]time.Duration{"a": 0},
	},
	// Zero duration as an int.
	{
		"a: 0",
		map[string]time.Duration{"a": 0},
	},

	// Binary data.
	{
		"a: !!binary gIGC\n",
		map[string]string{"a": "\x80\x81\x82"},
	},
	{
		"a: !!binary |\n  " + strings.Repeat("kJCQ", 17) + "kJ\n  CQ\n",
		map[string]string{"a": strings.Repeat("\x90", 54)},
	},
	{
		"a: !!binary |\n  " + strings.Repeat("A", 70) + "\n  ==\n",
		map[string]string{"a": strings.Repeat("\x00", 52)},
	},

	// Issue #39.
	{
		"a:\n b:\n  c: d\n",
		map[string]struct{ B any }{"a": {map[string]any{"c": "d"}}},
	},

	// Custom map type.
	{
		"a: {b: c}",
		M{"a": M{"b": "c"}},
	},

	// Support encoding.TextUnmarshaler.
	{
		"a: 1.2.3.4\n",
		map[string]textUnmarshaler{"a": {S: "1.2.3.4"}},
	},
	{
		"a: 2015-02-24T18:19:39Z\n",
		map[string]textUnmarshaler{"a": {"2015-02-24T18:19:39Z"}},
	},

	// Timestamps
	{
		// Date only.
		"a: 2015-01-01\n",
		map[string]time.Time{"a": time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
	},
	{
		// RFC3339
		"a: 2015-02-24T18:19:39.12Z\n",
		map[string]time.Time{"a": time.Date(2015, 2, 24, 18, 19, 39, .12e9, time.UTC)},
	},
	{
		// RFC3339 with short dates.
		"a: 2015-2-3T3:4:5Z",
		map[string]time.Time{"a": time.Date(2015, 2, 3, 3, 4, 5, 0, time.UTC)},
	},
	{
		// ISO8601 lower case t
		"a: 2015-02-24t18:19:39Z\n",
		map[string]time.Time{"a": time.Date(2015, 2, 24, 18, 19, 39, 0, time.UTC)},
	},
	{
		// space separate, no time zone
		"a: 2015-02-24 18:19:39\n",
		map[string]time.Time{"a": time.Date(2015, 2, 24, 18, 19, 39, 0, time.UTC)},
	},
	// Some cases not currently handled. Uncomment these when
	// the code is fixed.
	//	{
	//		// space separated with time zone
	//		"a: 2001-12-14 21:59:43.10 -5",
	//		map[string]any{"a": time.Date(2001, 12, 14, 21, 59, 43, .1e9, time.UTC)},
	//	},
	//	{
	//		// arbitrary whitespace between fields
	//		"a: 2001-12-14 \t\t \t21:59:43.10 \t Z",
	//		map[string]any{"a": time.Date(2001, 12, 14, 21, 59, 43, .1e9, time.UTC)},
	//	},
	{
		// explicit string tag
		"a: !!str 2015-01-01",
		map[string]any{"a": "2015-01-01"},
	},
	{
		// explicit timestamp tag on quoted string
		"a: !!timestamp \"2015-01-01\"",
		map[string]time.Time{"a": time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
	},
	{
		// explicit timestamp tag on unquoted string
		"a: !!timestamp 2015-01-01",
		map[string]time.Time{"a": time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
	},
	{
		// explicit timestamp tag into interface.
		"a: !!timestamp \"2015-01-01\"",
		map[string]any{"a": time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
	},
	{
		// implicit timestamp tag into interface.
		"a: 2015-01-01",
		map[string]any{"a": time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
	},

	// UTF-16-LE
	{
		"\xff\xfe\xf1\x00o\x00\xf1\x00o\x00:\x00 \x00v\x00e\x00r\x00y\x00 \x00y\x00e\x00s\x00\n\x00",
		M{"ñoño": "very yes"},
	},
	// UTF-16-LE with surrogate.
	{
		"\xff\xfe\xf1\x00o\x00\xf1\x00o\x00:\x00 \x00v\x00e\x00r\x00y\x00 \x00y\x00e\x00s\x00 \x00=\xd8\xd4\xdf\n\x00",
		M{"ñoño": "very yes 🟔"},
	},

	// UTF-16-BE
	{
		"\xfe\xff\x00\xf1\x00o\x00\xf1\x00o\x00:\x00 \x00v\x00e\x00r\x00y\x00 \x00y\x00e\x00s\x00\n",
		M{"ñoño": "very yes"},
	},
	// UTF-16-BE with surrogate.
	{
		"\xfe\xff\x00\xf1\x00o\x00\xf1\x00o\x00:\x00 \x00v\x00e\x00r\x00y\x00 \x00y\x00e\x00s\x00 \xd8=\xdf\xd4\x00\n",
		M{"ñoño": "very yes 🟔"},
	},

	// Comment scan exhausting the input buffer (issue #469).
	{
		"true\n#" + strings.Repeat(" ", 512*3),
		"true",
	},
	{
		"true #" + strings.Repeat(" ", 512*3),
		"true",
	},
	{
		`---
foo:
  ? complex key
  : complex value
ba?r: a?bc
`,
		map[string]any{
			"foo":  map[string]any{"complex key": "complex value"},
			"ba?r": "a?bc",
		},
	},

	// issue https://github.com/yaml/go-yaml/issues/157
	{
		`foo: abc
bar: def`,
		struct {
			F string `yaml:"foo"` // the correct tag, because it has `yaml` prefix
			B string `bar`        //nolint:govet // the incorrect tag, but supported
		}{
			F: "abc",
			B: "def", // value should be set using whole tag as a name, see issue: <https://github.com/yaml/go-yaml/issues/157>
		},
	},
}

type M map[string]any

type inlineB struct {
	B       int
	inlineC `yaml:",inline"`
}

type inlineC struct {
	C int
}

type inlineD struct {
	C *inlineC `yaml:",inline"`
	D int
}

func TestUnmarshal(t *testing.T) {
	for i, item := range unmarshalTests {
		t.Run(fmt.Sprintf("test %d: %q", i, item.data), func(t *testing.T) {
			typ := reflect.ValueOf(item.value).Type()
			value := reflect.New(typ)
			err := yaml.Unmarshal([]byte(item.data), value.Interface())
			if _, ok := err.(*yaml.TypeError); !ok {
				assert.NoError(t, err)
			}
			assert.DeepEqualf(t, item.value, value.Elem().Interface(), "error: %v", err)
		})
	}
}

func TestDecodeFromYAML(t *testing.T) {
	datatest.RunTestCases(t, func() ([]map[string]any, error) {
		return datatest.LoadTestCasesFromFile("testdata/decode.yaml", libyaml.LoadYAML)
	}, map[string]datatest.TestHandler{
		"decode": runDecodeTest,
	})
}

func runDecodeTest(t *testing.T, tc map[string]any) {
	t.Helper()

	// Get test inputs
	yamlData, ok := tc["yaml"].(string)
	if !ok {
		t.Fatal("yaml field must be string")
	}

	// Note: "type" field in YAML is renamed to "output_type" during normalization
	// to avoid conflict with the test type ("decode")
	typeName, ok := tc["output_type"].(string)
	if !ok {
		t.Fatal("output_type field must be string")
	}

	want := tc["want"]

	// Create target from type registry
	target, err := decodeTypes.NewPointerInstance(typeName)
	if err != nil {
		t.Fatalf("failed to create instance of type %s: %v", typeName, err)
	}

	// Unmarshal
	err = yaml.Unmarshal([]byte(yamlData), target)

	// Get the actual value (dereference pointer)
	actual := reflect.ValueOf(target).Elem().Interface()

	// Resolve constants in want
	wantResolved := decodeValues.Resolve(want)

	// Handle construct errors - if error occurred, check if want is empty/nil
	if err != nil {
		// For TypeError, values that can't be converted are skipped
		if _, ok := err.(*yaml.TypeError); !ok {
			t.Fatalf("unmarshal failed with unexpected error: %v", err)
		}
		// TypeError is expected for invalid conversions - compare with want
	}

	// Compare by marshaling both to YAML and comparing the output
	// This handles type differences (e.g., int vs float with same value)
	actualYAML, err := yaml.Marshal(actual)
	if err != nil {
		t.Fatalf("failed to marshal actual: %v", err)
	}

	wantYAML, err := yaml.Marshal(wantResolved)
	if err != nil {
		t.Fatalf("failed to marshal want: %v", err)
	}

	// Compare the YAML representations
	if string(actualYAML) != string(wantYAML) {
		t.Fatalf("YAML mismatch:\nGot:\n%s\nWant:\n%s", actualYAML, wantYAML)
	}
}

func TestUnmarshalFullTimestamp(t *testing.T) {
	// Full timestamp in same format as encoded. This is confirmed to be
	// properly decoded by Python as a timestamp as well.
	str := "2015-02-24T18:19:39.123456789-03:00"
	var tm any
	err := yaml.Unmarshal([]byte(str), &tm)
	assert.NoError(t, err)
	expectedTime := time.Date(2015, 2, 24, 18, 19, 39, 123456789, tm.(time.Time).Location())
	assert.DeepEqual(t, expectedTime, tm)
	assert.DeepEqual(t, time.Date(2015, 2, 24, 21, 19, 39, 123456789, time.UTC), tm.(time.Time).In(time.UTC))
}

func TestDecoderSingleDocument(t *testing.T) {
	// Test that Decoder.Decode works as expected on
	// all the unmarshal tests.
	for i, item := range unmarshalTests {
		t.Run(fmt.Sprintf("test %d: %q", i, item.data), func(t *testing.T) {
			if item.data == "" {
				// Behavior differs when there's no YAML.
				return
			}
			typ := reflect.ValueOf(item.value).Type()
			value := reflect.New(typ)
			err := yaml.NewDecoder(strings.NewReader(item.data)).Decode(value.Interface())
			if _, ok := err.(*yaml.TypeError); !ok {
				assert.NoError(t, err)
			}
			assert.DeepEqual(t, item.value, value.Elem().Interface())
		})
	}
}

var decoderTests = []struct {
	data   string
	values []any
}{{
	"",
	nil,
}, {
	"a: b",
	[]any{
		map[string]any{"a": "b"},
	},
}, {
	"---\na: b\n...\n",
	[]any{
		map[string]any{"a": "b"},
	},
}, {
	"---\n'hello'\n...\n---\ngoodbye\n...\n",
	[]any{
		"hello",
		"goodbye",
	},
}}

func TestDecoder(t *testing.T) {
	for i, item := range decoderTests {
		t.Run(fmt.Sprintf("test %d: %q", i, item.data), func(t *testing.T) {
			var values []any
			dec := yaml.NewDecoder(strings.NewReader(item.data))
			for {
				var value any
				err := dec.Decode(&value)
				if err == io.EOF {
					break
				}
				assert.NoError(t, err)
				values = append(values, value)
			}
			assert.DeepEqual(t, item.values, values)
		})
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("some read error")
}

func TestDecoderReadError(t *testing.T) {
	err := yaml.NewDecoder(errReader{}).Decode(&struct{}{})
	assert.ErrorMatches(t, `yaml: offset 0: input error: some read error`, err)
}

func TestUnmarshalNaN(t *testing.T) {
	value := map[string]any{}
	err := yaml.Unmarshal([]byte("notanum: .NaN"), &value)
	assert.NoError(t, err)
	assert.True(t, math.IsNaN(value["notanum"].(float64)))
}

func TestUnmarshalDurationInt(t *testing.T) {
	// Don't accept plain ints as durations as it's unclear (issue #200).
	var d time.Duration
	err := yaml.Unmarshal([]byte("123"), &d)
	assert.ErrorMatches(t, "line 1: cannot construct !!int `123` into time.Duration", err)
}

func TestUnmarshalErrorsFromYAML(t *testing.T) {
	datatest.RunTestCases(t, func() ([]map[string]any, error) {
		return datatest.LoadTestCasesFromFile("testdata/unmarshal_errors.yaml", libyaml.LoadYAML)
	}, map[string]datatest.TestHandler{
		"unmarshal-error": runUnmarshalErrorTest,
	})
}

func runUnmarshalErrorTest(t *testing.T, tc map[string]any) {
	t.Helper()

	yamlInput := datatest.RequireString(t, tc, "yaml")
	expectedError := datatest.RequireString(t, tc, "want")

	var target any
	// If a type is specified, use it; otherwise default to any
	if typeName, ok := tc["output_type"].(string); ok {
		var err error
		target, err = decodeTypes.NewPointerInstance(typeName)
		if err != nil {
			t.Fatalf("failed to create instance of type %s: %v", typeName, err)
		}
	} else {
		var value any
		target = &value
	}

	err := yaml.Unmarshal([]byte(yamlInput), target)
	if err == nil {
		t.Fatalf("got nil; want error %q - Partial unmarshal: %#v", expectedError, target)
	}
	assert.Equalf(t, expectedError, err.Error(), "Partial unmarshal: %#v", target)
}

func TestDecoderErrors(t *testing.T) {
	datatest.RunTestCases(t, func() ([]map[string]any, error) {
		return datatest.LoadTestCasesFromFile("testdata/unmarshal_errors.yaml", libyaml.LoadYAML)
	}, map[string]datatest.TestHandler{
		"unmarshal-error": func(t *testing.T, tc map[string]any) {
			t.Helper()
			yamlInput := datatest.RequireString(t, tc, "yaml")
			expectedError := datatest.RequireString(t, tc, "want")

			var target any
			// If a type is specified, use it; otherwise default to any
			if typeName, ok := tc["output_type"].(string); ok {
				var err error
				target, err = decodeTypes.NewPointerInstance(typeName)
				if err != nil {
					t.Fatalf("failed to create instance of type %s: %v", typeName, err)
				}
			} else {
				var value any
				target = &value
			}

			err := yaml.NewDecoder(strings.NewReader(yamlInput)).Decode(target)
			if err == nil {
				t.Fatalf("got nil; want error %q - Partial decode: %#v", expectedError, target)
			}
			assert.Equalf(t, expectedError, err.Error(), "Partial decode: %#v", target)
		},
	})
}

func TestParserErrorUnmarshal(t *testing.T) {
	var v struct {
		A, B int
	}
	data := "a: 1\n=\nb: 2"
	err := yaml.Unmarshal([]byte(data), &v)
	var asErr libyaml.ScannerError
	assert.ErrorAs(t, err, &asErr)
	expectedErr := libyaml.ScannerError{
		ContextMark: libyaml.Mark{
			Index:  5,
			Line:   2,
			Column: 0,
		},
		ContextMessage: "while scanning a simple key",

		Mark: libyaml.Mark{
			Index:  7,
			Line:   3,
			Column: 0,
		},
		Message: "could not find expected ':'",
	}
	assert.DeepEqual(t, expectedErr, asErr)
}

func TestParserErrorDecoder(t *testing.T) {
	var v any
	data := "value: -"
	err := yaml.NewDecoder(strings.NewReader(data)).Decode(&v)
	var asErr libyaml.ScannerError
	assert.ErrorAs(t, err, &asErr)
	expectedErr := libyaml.ScannerError{
		Mark: libyaml.Mark{
			Index:  7,
			Line:   1,
			Column: 7,
		},
		Message: "block sequence entries are not allowed in this context",
	}
	assert.DeepEqual(t, expectedErr, asErr)
}

var unmarshalerTests = []struct {
	data, tag string
	value     any
}{
	{"_: {hi: there}", "!!map", map[string]any{"hi": "there"}},
	{"_: [1,A]", "!!seq", []any{1, "A"}},
	{"_: 10", "!!int", 10},
	{"_: null", "!!null", nil},
	{`_: BAR!`, "!!str", "BAR!"},
	{`_: "BAR!"`, "!!str", "BAR!"},
	{"_: !!foo 'BAR!'", "!!foo", "BAR!"},
	{`_: ""`, "!!str", ""},
}

var unmarshalerResult = map[int]error{}

type unmarshalerType struct {
	value any
}

func (o *unmarshalerType) UnmarshalYAML(value *yaml.Node) error {
	if err := value.Load(&o.value); err != nil {
		return err
	}
	if i, ok := o.value.(int); ok {
		if result, ok := unmarshalerResult[i]; ok {
			return result
		}
	}
	return nil
}

type unmarshalerPointer struct {
	Field *unmarshalerType `yaml:"_"`
}

type unmarshalerInlined struct {
	Field   *unmarshalerType `yaml:"_"`
	Inlined unmarshalerType  `yaml:",inline"`
}

type unmarshalerInlinedTwice struct {
	InlinedTwice unmarshalerInlined `yaml:",inline"`
}

type obsoleteUnmarshalerType struct {
	value any
}

func (o *obsoleteUnmarshalerType) UnmarshalYAML(unmarshal func(v any) error) error {
	if err := unmarshal(&o.value); err != nil {
		return err
	}
	if i, ok := o.value.(int); ok {
		if result, ok := unmarshalerResult[i]; ok {
			return result
		}
	}
	return nil
}

type obsoleteUnmarshalerPointer struct {
	Field *obsoleteUnmarshalerType `yaml:"_"`
}

type obsoleteUnmarshalerValue struct {
	Field obsoleteUnmarshalerType `yaml:"_"`
}

func TestUnmarshalerPointerField(t *testing.T) {
	for _, item := range unmarshalerTests {
		obj := &unmarshalerPointer{}
		err := yaml.Unmarshal([]byte(item.data), obj)
		assert.NoError(t, err)
		if item.value == nil {
			assert.IsNil(t, obj.Field)
		} else {
			assert.NotNilf(t, obj.Field, "Pointer not initialized (%#v)", item.value)
			assert.DeepEqual(t, item.value, obj.Field.value)
		}
	}
	for _, item := range unmarshalerTests {
		obj := &obsoleteUnmarshalerPointer{}
		err := yaml.Unmarshal([]byte(item.data), obj)
		assert.NoError(t, err)
		if item.value == nil {
			assert.IsNil(t, obj.Field)
		} else {
			assert.NotNilf(t, obj.Field, "Pointer not initialized (%#v)", item.value)
			assert.DeepEqual(t, item.value, obj.Field.value)
		}
	}
}

func TestUnmarshalerValueField(t *testing.T) {
	for _, item := range unmarshalerTests {
		obj := &obsoleteUnmarshalerValue{}
		err := yaml.Unmarshal([]byte(item.data), obj)
		assert.NoError(t, err)
		assert.NotNilf(t, obj.Field, "Pointer not initialized (%#v)", item.value)
		assert.DeepEqual(t, item.value, obj.Field.value)
	}
}

func TestUnmarshalerInlinedField(t *testing.T) {
	obj := &unmarshalerInlined{}
	err := yaml.Unmarshal([]byte("_: a\ninlined: b\n"), obj)
	assert.NoError(t, err)
	assert.DeepEqual(t, &unmarshalerType{"a"}, obj.Field)
	assert.DeepEqual(t, unmarshalerType{map[string]any{"_": "a", "inlined": "b"}}, obj.Inlined)

	twc := &unmarshalerInlinedTwice{}
	err = yaml.Unmarshal([]byte("_: a\ninlined: b\n"), twc)
	assert.NoError(t, err)
	assert.DeepEqual(t, &unmarshalerType{"a"}, twc.InlinedTwice.Field)
	assert.DeepEqual(t, unmarshalerType{map[string]any{"_": "a", "inlined": "b"}}, twc.InlinedTwice.Inlined)
}

func TestUnmarshalerWholeDocument(t *testing.T) {
	obj := &obsoleteUnmarshalerType{}
	err := yaml.Unmarshal([]byte(unmarshalerTests[0].data), obj)
	assert.NoError(t, err)
	value, ok := obj.value.(map[string]any)
	assert.Truef(t, ok, "value: %#v", obj.value)
	assert.DeepEqual(t, unmarshalerTests[0].value, value["_"])
}

func TestUnmarshalerTypeError(t *testing.T) {
	unmarshalerResult[2] = &yaml.TypeError{Errors: []*yaml.LoadError{{Err: errors.New("foo"), Line: 1, Column: 1}}}
	unmarshalerResult[4] = &yaml.TypeError{Errors: []*yaml.LoadError{{Err: errors.New("bar"), Line: 1, Column: 1}}}
	defer func() {
		delete(unmarshalerResult, 2)
		delete(unmarshalerResult, 4)
	}()

	type T struct {
		Before int
		After  int
		M      map[string]*unmarshalerType
	}
	var v T
	data := `{before: A, m: {abc: 1, def: 2, ghi: 3, jkl: 4}, after: B}`
	err := yaml.Unmarshal([]byte(data), &v)
	expectedError := "" +
		"yaml: construct errors:\n" +
		"  line 1: cannot construct !!str `A` into int\n" +
		"  line 1: foo\n" +
		"  line 1: bar\n" +
		"  line 1: cannot construct !!str `B` into int"
	assert.ErrorMatches(t, expectedError, err)
	assert.NotNil(t, v.M["abc"])
	assert.IsNil(t, v.M["def"])
	assert.NotNil(t, v.M["ghi"])
	assert.IsNil(t, v.M["jkl"])

	assert.Equal(t, 1, v.M["abc"].value)
	assert.Equal(t, 3, v.M["ghi"].value)
}

func TestObsoleteUnmarshalerTypeError(t *testing.T) {
	unmarshalerResult[2] = &yaml.TypeError{Errors: []*yaml.LoadError{{Err: errors.New("foo"), Line: 1, Column: 1}}}
	unmarshalerResult[4] = &yaml.TypeError{Errors: []*yaml.LoadError{{Err: errors.New("bar"), Line: 1, Column: 1}}}
	defer func() {
		delete(unmarshalerResult, 2)
		delete(unmarshalerResult, 4)
	}()

	type T struct {
		Before int
		After  int
		M      map[string]*obsoleteUnmarshalerType
	}
	var v T
	data := `{before: A, m: {abc: 1, def: 2, ghi: 3, jkl: 4}, after: B}`
	err := yaml.Unmarshal([]byte(data), &v)
	expectedError := "" +
		"yaml: construct errors:\n" +
		"  line 1: cannot construct !!str `A` into int\n" +
		"  line 1: foo\n" +
		"  line 1: bar\n" +
		"  line 1: cannot construct !!str `B` into int"
	assert.ErrorMatches(t, expectedError, err)

	assert.NotNil(t, v.M["abc"])
	assert.IsNil(t, v.M["def"])
	assert.NotNil(t, v.M["ghi"])
	assert.IsNil(t, v.M["jkl"])

	assert.Equal(t, 1, v.M["abc"].value)
	assert.Equal(t, 3, v.M["ghi"].value)
}

func TestTypeError_Unwrapping(t *testing.T) {
	errSentinel := errors.New("foo")
	errSentinel2 := errors.New("bar")

	errUnmarshal := &yaml.LoadError{
		Line:   1,
		Column: 2,
		Err:    errSentinel,
	}

	errUnmarshal2 := &yaml.LoadError{
		Line:   2,
		Column: 2,
		Err:    errSentinel2,
	}

	// Simulate a TypeError
	err := &yaml.TypeError{
		Errors: []*yaml.LoadError{
			errUnmarshal,
			errUnmarshal2,
		},
	}

	var errTarget *yaml.LoadError
	// check we can unwrap an error
	assert.ErrorAs(t, err, &errTarget)

	// check we got the first error
	assert.ErrorIs(t, errTarget, errUnmarshal)

	// check we can unwrap any sentinel error wrapped in any UnmarshalError
	assert.ErrorIs(t, err, errSentinel)
	assert.ErrorIs(t, err, errSentinel2)
}

func TestTypeError_Unwrapping_Failures(t *testing.T) {
	errSentinel := errors.New("foo")

	errUnmarshal := &yaml.LoadError{
		Line:   1,
		Column: 2,
		Err:    errSentinel,
	}

	errUnmarshal2 := &yaml.LoadError{
		Line:   2,
		Column: 2,
		Err:    errors.New("bar"),
	}

	// Simulate a TypeError
	err := &yaml.TypeError{
		Errors: []*yaml.LoadError{
			errUnmarshal,
			errUnmarshal2,
		},
	}

	var errTarget *yaml.LoadError
	// check we can unwrap an error
	assert.ErrorAs(t, err, &errTarget)

	// check we got the first error
	assert.ErrorIs(t, errTarget, errUnmarshal)

	// check we can still unwrap the error wrapped in UnmarshalError
	assert.ErrorIs(t, errTarget, errSentinel)
}

type proxyTypeError struct{}

func (v *proxyTypeError) UnmarshalYAML(node *yaml.Node) error {
	var s string
	var a int32
	var b int64
	if err := node.Load(&s); err != nil {
		panic(err)
	}
	if s == "a" {
		if err := node.Load(&b); err == nil {
			panic("should have failed")
		}
		return node.Load(&a)
	}
	if err := node.Load(&a); err == nil {
		panic("should have failed")
	}
	return node.Load(&b)
}

func TestUnmarshalerTypeErrorProxying(t *testing.T) {
	type T struct {
		Before int
		After  int
		M      map[string]*proxyTypeError
	}
	var v T
	data := `{before: A, m: {abc: a, def: b}, after: B}`
	err := yaml.Unmarshal([]byte(data), &v)
	expectedError := "" +
		"yaml: construct errors:\n" +
		"  line 1: cannot construct !!str `A` into int\n" +
		"  line 1: cannot construct !!str `a` into int32\n" +
		"  line 1: cannot construct !!str `b` into int64\n" +
		"  line 1: cannot construct !!str `B` into int"
	assert.ErrorMatches(t, expectedError, err)
}

type obsoleteProxyTypeError struct{}

func (v *obsoleteProxyTypeError) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	var a int32
	var b int64
	if err := unmarshal(&s); err != nil {
		panic(err)
	}
	if s == "a" {
		if err := unmarshal(&b); err == nil {
			panic("should have failed")
		}
		return unmarshal(&a)
	}
	if err := unmarshal(&a); err == nil {
		panic("should have failed")
	}
	return unmarshal(&b)
}

func TestObsoleteUnmarshalerTypeErrorProxying(t *testing.T) {
	type T struct {
		Before int
		After  int
		M      map[string]*obsoleteProxyTypeError
	}
	var v T
	data := `{before: A, m: {abc: a, def: b}, after: B}`
	err := yaml.Unmarshal([]byte(data), &v)
	expectedError := "" +
		"yaml: construct errors:\n" +
		"  line 1: cannot construct !!str `A` into int\n" +
		"  line 1: cannot construct !!str `a` into int32\n" +
		"  line 1: cannot construct !!str `b` into int64\n" +
		"  line 1: cannot construct !!str `B` into int"
	assert.ErrorMatches(t, expectedError, err)
}

var errFailing = errors.New("failingErr")

type failingUnmarshaler struct{}

func (ft *failingUnmarshaler) UnmarshalYAML(node *yaml.Node) error {
	return errFailing
}

func TestUnmarshalerError(t *testing.T) {
	data := `{foo: 123, bar: {}, spam: "test"}`
	dst := struct {
		Foo  int
		Bar  *failingUnmarshaler
		Spam string
	}{}
	err := yaml.Unmarshal([]byte(data), &dst)
	expectedErr := &yaml.TypeError{
		Errors: []*yaml.LoadError{
			{Line: 1, Column: 17, Err: errFailing},
		},
	}
	assert.DeepEqual(t, expectedErr, err)
	// whatever could be unmarshaled must be unmarshaled
	assert.Equal(t, 123, dst.Foo)
	assert.DeepEqual(t, &failingUnmarshaler{}, dst.Bar)
	assert.Equal(t, "test", dst.Spam)
}

type obsoleteFailingUnmarshaler struct{}

func (ft *obsoleteFailingUnmarshaler) UnmarshalYAML(unmarshal func(any) error) error {
	return errFailing
}

func TestObsoleteUnmarshalerError(t *testing.T) {
	data := `{foo: 123, bar: {}, spam: "test"}`
	dst := struct {
		Foo  int
		Bar  *obsoleteFailingUnmarshaler
		Spam string
	}{}
	err := yaml.Unmarshal([]byte(data), &dst)
	expectedErr := &yaml.TypeError{
		Errors: []*yaml.LoadError{
			{Line: 1, Column: 17, Err: errFailing},
		},
	}
	assert.DeepEqual(t, expectedErr, err)
	// whatever could be unmarshaled must be unmarshaled
	assert.Equal(t, 123, dst.Foo)
	assert.DeepEqual(t, &obsoleteFailingUnmarshaler{}, dst.Bar)
	assert.Equal(t, "test", dst.Spam)
}

type failingTextUnmarshaler struct{}

var _ encoding.TextUnmarshaler = &failingTextUnmarshaler{}

func (ft *failingTextUnmarshaler) UnmarshalText(b []byte) error {
	return errFailing
}

func TestTextUnmarshalerError(t *testing.T) {
	data := `{foo: 123, bar: "456", spam: "test"}`
	dst := struct {
		Foo  int
		Bar  *failingTextUnmarshaler
		Spam string
	}{}
	err := yaml.Unmarshal([]byte(data), &dst)
	expectedErr := &yaml.TypeError{
		Errors: []*yaml.LoadError{
			{Line: 1, Column: 17, Err: errFailing},
		},
	}
	assert.DeepEqual(t, expectedErr, err)
	// whatever could be unmarshaled must be unmarshaled
	assert.Equal(t, 123, dst.Foo)
	assert.DeepEqual(t, &failingTextUnmarshaler{}, dst.Bar)
	assert.Equal(t, "test", dst.Spam)
}

func TestUnmarshalError_Unwrapping(t *testing.T) {
	errSentinel := errors.New("foo")

	errUnmarshal := &yaml.LoadError{
		Line:   1,
		Column: 2,
		Err:    errSentinel,
	}

	assert.ErrorIs(t, errUnmarshal, errSentinel)
}

func TestTextUnmarshalerNonScalar(t *testing.T) {
	dst := struct {
		A textUnmarshaler
	}{}
	inputs := []string{
		`a: {}`,
		`a: []`,
	}

	for _, input := range inputs {
		err := yaml.Unmarshal([]byte(input), &dst)
		t.Logf("%s -> err=%v", input, err)
		var target *yaml.TypeError
		if !errors.As(err, &target) {
			t.Errorf("expected yaml.TypeError, got %v", err)
		}
	}
}

type sliceUnmarshaler []int

func (su *sliceUnmarshaler) UnmarshalYAML(node *yaml.Node) error {
	var slice []int
	err := node.Load(&slice)
	if err == nil {
		*su = slice
		return nil
	}

	var intVal int
	err = node.Load(&intVal)
	if err == nil {
		*su = []int{intVal}
		return nil
	}

	return err
}

func TestUnmarshalerRetry(t *testing.T) {
	var su sliceUnmarshaler
	err := yaml.Unmarshal([]byte("[1, 2, 3]"), &su)
	assert.NoError(t, err)
	assert.DeepEqual(t, sliceUnmarshaler([]int{1, 2, 3}), su)

	err = yaml.Unmarshal([]byte("1"), &su)
	assert.NoError(t, err)
	assert.DeepEqual(t, sliceUnmarshaler([]int{1}), su)
}

type obsoleteSliceUnmarshaler []int

func (su *obsoleteSliceUnmarshaler) UnmarshalYAML(unmarshal func(any) error) error {
	var slice []int
	err := unmarshal(&slice)
	if err == nil {
		*su = slice
		return nil
	}

	var intVal int
	err = unmarshal(&intVal)
	if err == nil {
		*su = []int{intVal}
		return nil
	}

	return err
}

func TestObsoleteUnmarshalerRetry(t *testing.T) {
	var su obsoleteSliceUnmarshaler
	err := yaml.Unmarshal([]byte("[1, 2, 3]"), &su)
	assert.NoError(t, err)
	assert.DeepEqual(t, obsoleteSliceUnmarshaler([]int{1, 2, 3}), su)

	err = yaml.Unmarshal([]byte("1"), &su)
	assert.NoError(t, err)
	assert.DeepEqual(t, obsoleteSliceUnmarshaler([]int{1}), su)
}

// From http://yaml.org/type/merge.html
var mergeTests = `
anchors:
  list:
    - &CENTER { "x": 1, "y": 2 }
    - &LEFT   { "x": 0, "y": 2 }
    - &BIG    { "r": 10 }
    - &SMALL  { "r": 1 }

# All the following maps are equal:

plain:
  # Explicit keys
  "x": 1
  "y": 2
  "r": 10
  label: center/big

mergeOne:
  # Merge one map
  << : *CENTER
  "r": 10
  label: center/big

mergeMultiple:
  # Merge multiple maps
  << : [ *CENTER, *BIG ]
  label: center/big

override:
  # Override
  << : [ *BIG, *LEFT, *SMALL ]
  "x": 1
  label: center/big

shortTag:
  # Explicit short merge tag
  !!merge "<<" : [ *CENTER, *BIG ]
  label: center/big

longTag:
  # Explicit merge long tag
  !<tag:yaml.org,2002:merge> "<<" : [ *CENTER, *BIG ]
  label: center/big

inlineMap:
  # Inlined map
  << : {"x": 1, "y": 2, "r": 10}
  label: center/big

inlineSequenceMap:
  # Inlined map in sequence
  << : [ *CENTER, {"r": 10} ]
  label: center/big
`

func TestMerge(t *testing.T) {
	want := map[string]any{
		"x":     1,
		"y":     2,
		"r":     10,
		"label": "center/big",
	}

	wantStringMap := make(map[string]any)
	for k, v := range want {
		wantStringMap[fmt.Sprintf("%v", k)] = v
	}

	var m map[any]any
	err := yaml.Unmarshal([]byte(mergeTests), &m)
	assert.NoError(t, err)
	for name, test := range m {
		if name == "anchors" {
			continue
		}
		if name == "plain" {
			assert.DeepEqualf(t, wantStringMap, test, "test %q failed", name)
			continue
		}
		assert.DeepEqualf(t, want, test, "test %q failed", name)
	}
}

func TestMergeStruct(t *testing.T) {
	type Data struct {
		X, Y, R int
		Label   string
	}
	want := Data{1, 2, 10, "center/big"}

	var m map[string]Data
	err := yaml.Unmarshal([]byte(mergeTests), &m)
	assert.NoError(t, err)
	for name, test := range m {
		if name == "anchors" {
			continue
		}
		assert.DeepEqualf(t, want, test, "test %q failed", name)
	}
}

var mergeTestsNested = `
mergeouter1: &mergeouter1
    d: 40
    e: 50

mergeouter2: &mergeouter2
    e: 5
    f: 6
    g: 70

mergeinner1: &mergeinner1
    <<: *mergeouter1
    inner:
        a: 1
        b: 2

mergeinner2: &mergeinner2
    <<: *mergeouter2
    inner:
        a: -1
        b: -2

outer:
    <<: [*mergeinner1, *mergeinner2]
    f: 60
    inner:
        a: 10
`

func TestMergeNestedStruct(t *testing.T) {
	// Issue #818: Merging used to just unmarshal twice on the target
	// value, which worked for maps as these were replaced by the new map,
	// but not on struct values as these are preserved. This resulted in
	// the nested data from the merged map to be mixed up with the data
	// from the map being merged into.
	//
	// This test also prevents two potential bugs from showing up:
	//
	// 1) A simple implementation might just zero out the nested value
	//    before unmarshaling the second time, but this would clobber previous
	//    data that is usually respected ({C: 30} below).
	//
	// 2) A simple implementation might attempt to handle the key skipping
	//    directly by iterating over the merging map without recursion, but
	//    there are more complex cases that require recursion.
	//
	// Quick summary of the fields:
	//
	// - A must come from outer and not overridden
	// - B must not be set as its in the ignored merge
	// - C should still be set as it's preset in the value
	// - D should be set from the recursive merge
	// - E should be set from the first recursive merge, ignored on the second
	// - F should be set in the inlined map from outer, ignored later
	// - G should be set in the inlined map from the second recursive merge
	//

	type Inner struct {
		A, B, C int
	}
	type Outer struct {
		D, E   int
		Inner  Inner
		Inline map[string]int `yaml:",inline"`
	}
	type Data struct {
		Outer Outer
	}

	test := Data{Outer{0, 0, Inner{C: 30}, nil}}
	want := Data{Outer{40, 50, Inner{A: 10, C: 30}, map[string]int{"f": 60, "g": 70}}}

	err := yaml.Unmarshal([]byte(mergeTestsNested), &test)
	assert.NoError(t, err)
	assert.DeepEqual(t, want, test)

	// Repeat test with a map.

	var testm map[string]any
	wantm := map[string]any{
		"f": 60,
		"inner": map[string]any{
			"a": 10,
		},
		"d": 40,
		"e": 50,
		"g": 70,
	}
	err = yaml.Unmarshal([]byte(mergeTestsNested), &testm)
	assert.NoError(t, err)
	assert.DeepEqual(t, wantm, testm["outer"])
}

var unmarshalNullTests = []struct {
	input              string
	pristine, expected func() any
}{{
	"null",
	func() any { var v any = "v"; return &v },
	func() any { var v any = nil; return &v },
}, {
	"null",
	func() any { s := "s"; return &s },
	func() any { s := "s"; return &s },
}, {
	"null",
	func() any { s := "s"; sptr := &s; return &sptr },
	func() any { var sptr *string; return &sptr },
}, {
	"null",
	func() any { i := 1; return &i },
	func() any { i := 1; return &i },
}, {
	"null",
	func() any { i := 1; iptr := &i; return &iptr },
	func() any { var iptr *int; return &iptr },
}, {
	"null",
	func() any { m := map[string]int{"s": 1}; return &m },
	func() any { var m map[string]int; return &m },
}, {
	"null",
	func() any { m := map[string]int{"s": 1}; return m },
	func() any { m := map[string]int{"s": 1}; return m },
}, {
	"s2: null\ns3: null",
	func() any { m := map[string]int{"s1": 1, "s2": 2}; return m },
	func() any { m := map[string]int{"s1": 1, "s2": 2, "s3": 0}; return m },
}, {
	"s2: null\ns3: null",
	func() any { m := map[string]any{"s1": 1, "s2": 2}; return m },
	func() any { m := map[string]any{"s1": 1, "s2": nil, "s3": nil}; return m },
}}

func TestUnmarshalNull(t *testing.T) {
	for _, test := range unmarshalNullTests {
		pristine := test.pristine()
		expected := test.expected()
		err := yaml.Unmarshal([]byte(test.input), pristine)
		assert.NoError(t, err)
		assert.DeepEqual(t, expected, pristine)
	}
}

func TestUnmarshalPreservesData(t *testing.T) {
	var v struct {
		A, B int
		C    int `yaml:"-"`
	}
	v.A = 42
	v.C = 88
	err := yaml.Unmarshal([]byte("---"), &v)
	assert.NoError(t, err)
	assert.Equal(t, 42, v.A)
	assert.Equal(t, 0, v.B)
	assert.Equal(t, 88, v.C)

	err = yaml.Unmarshal([]byte("b: 21\nc: 99"), &v)
	assert.NoError(t, err)
	assert.Equal(t, 42, v.A)
	assert.Equal(t, 21, v.B)
	assert.Equal(t, 88, v.C)
}

func TestUnmarshalSliceOnPreset(t *testing.T) {
	// Issue #48.
	v := struct{ A []int }{[]int{1}}
	err := yaml.Unmarshal([]byte("a: [2]"), &v)
	assert.NoError(t, err)
	assert.DeepEqual(t, []int{2}, v.A)
}

var unmarshalStrictTests = []struct {
	known  bool
	unique bool
	data   string
	value  any
	error  string
}{{
	known: true,
	data:  "a: 1\nc: 2\n",
	value: struct{ A, B int }{A: 1},
	error: `yaml: construct errors:\n  line 2: field c not found in type struct { A int; B int }`,
}, {
	unique: true,
	data:   "a: 1\nb: 2\na: 3\n",
	value:  struct{ A, B int }{A: 3, B: 2},
	error:  `yaml: construct errors:\n  line 3: mapping key "a" already defined at line 1`,
}, {
	unique: true,
	data:   "c: 3\na: 1\nb: 2\nc: 4\n",
	value: struct {
		A       int
		inlineB `yaml:",inline"`
	}{
		A: 1,
		inlineB: inlineB{
			B: 2,
			inlineC: inlineC{
				C: 4,
			},
		},
	},
	error: `yaml: construct errors:\n  line 4: mapping key "c" already defined at line 1`,
}, {
	unique: true,
	data:   "c: 0\na: 1\nb: 2\nc: 1\n",
	value: struct {
		A       int
		inlineB `yaml:",inline"`
	}{
		A: 1,
		inlineB: inlineB{
			B: 2,
			inlineC: inlineC{
				C: 1,
			},
		},
	},
	error: `yaml: construct errors:\n  line 4: mapping key "c" already defined at line 1`,
}, {
	unique: true,
	data:   "c: 1\na: 1\nb: 2\nc: 3\n",
	value: struct {
		A int
		M map[string]any `yaml:",inline"`
	}{
		A: 1,
		M: map[string]any{
			"b": 2,
			"c": 3,
		},
	},
	error: `yaml: construct errors:\n  line 4: mapping key "c" already defined at line 1`,
}, {
	unique: true,
	data:   "a: 1\n9: 2\nnull: 3\n9: 4",
	value: map[any]any{
		"a": 1,
		nil: 3,
		9:   4,
	},
	error: `yaml: construct errors:\n  line 4: mapping key "9" already defined at line 2`,
}}

func TestUnmarshalKnownFields(t *testing.T) {
	for i, item := range unmarshalStrictTests {
		t.Logf("test %d: %q", i, item.data)
		// First test that normal Unmarshal unmarshals to the expected value.
		if !item.unique {
			typ := reflect.ValueOf(item.value).Type()
			value := reflect.New(typ)
			err := yaml.Unmarshal([]byte(item.data), value.Interface())
			assert.NoError(t, err)
			assert.DeepEqual(t, item.value, value.Elem().Interface())
		}

		// Then test that it fails on the same thing with KnownFields on.
		typ := reflect.ValueOf(item.value).Type()
		value := reflect.New(typ)
		dec := yaml.NewDecoder(bytes.NewBuffer([]byte(item.data)))
		dec.KnownFields(item.known)
		err := dec.Decode(value.Interface())
		assert.ErrorMatches(t, item.error, err)
	}
}

type textUnmarshaler struct {
	S string
}

func (t *textUnmarshaler) UnmarshalText(s []byte) error {
	t.S = string(s)
	return nil
}

func TestFuzzCrashersFromYAML(t *testing.T) {
	datatest.RunTestCases(t, func() ([]map[string]any, error) {
		return datatest.LoadTestCasesFromFile("testdata/fuzz_crashers.yaml", libyaml.LoadYAML)
	}, map[string]datatest.TestHandler{
		"fuzz-crasher": runFuzzCrasherTest,
	})
}

func runFuzzCrasherTest(t *testing.T, tc map[string]any) {
	t.Helper()

	yamlInput := datatest.RequireString(t, tc, "yaml")

	// Just unmarshal and ensure it doesn't crash
	var v any
	_ = yaml.Unmarshal([]byte(yamlInput), &v)
}

func TestIssue117(t *testing.T) {
	data := []byte(`
a:
<<:
-
?
-
`)

	x := map[string]any{}
	err := yaml.Unmarshal([]byte(data), &x)
	if err == nil {
		t.Errorf("expected error, got none")
	}
}

func TestParserErrorUnknownAnchorPosition(t *testing.T) {
	tests := []struct {
		data   string
		line   int
		column int
	}{
		{"*x", 1, 1},
		{"a: *x", 1, 4},
		{"a:\n  b: *x", 2, 6},
	}

	for _, test := range tests {
		var n yaml.Node
		err := yaml.Unmarshal([]byte(test.data), &n)
		asErr := new(libyaml.ParserError)
		assert.ErrorAs(t, err, &asErr)
		expected := &libyaml.ParserError{
			Message: "unknown anchor 'x' referenced",
			Mark: libyaml.Mark{
				Line:   test.line,
				Column: test.column,
			},
		}
		assert.DeepEqual(t, expected, asErr)
	}
}

func TestTypeError_Strings(t *testing.T) {
	// Create a TypeError with multiple errors
	typeErr := &yaml.TypeError{
		Errors: []*yaml.LoadError{
			{Err: errors.New("cannot unmarshal string into int"), Line: 5, Column: 3},
			{Err: errors.New("cannot unmarshal bool into string"), Line: 10, Column: 7},
		},
	}

	strings := typeErr.Strings()

	assert.Equal(t, 2, len(strings))
	assert.Equal(t, "line 5: cannot unmarshal string into int", strings[0])
	assert.Equal(t, "line 10: cannot unmarshal bool into string", strings[1])
}

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
		"encode":      runEncodeTest,
		"encode-opts": runEncodeOptsTest,
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

func runEncodeOptsTest(t *testing.T, tc map[string]any) {
	t.Helper()

	// Get type and create instance
	// Note: "type" field in YAML is renamed to "output_type" during normalization
	// to avoid conflict with the test type ("encode-opts")
	typeName := tc["output_type"].(string)
	data := tc["data"]

	// Parse options
	var opts []yaml.Option
	if optsMap, ok := tc["opts"].(map[string]any); ok {
		if rq, ok := optsMap["required-quotes"].(string); ok {
			switch rq {
			case "single":
				opts = append(opts, yaml.WithRequiredQuotes(yaml.QuoteSingle))
			case "double":
				opts = append(opts, yaml.WithRequiredQuotes(yaml.QuoteDouble))
			case "legacy":
				opts = append(opts, yaml.WithRequiredQuotes(yaml.QuoteLegacy))
			default:
				t.Fatalf("Unknown required-quotes value: %s", rq)
			}
		}
	}

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

	// Dump the target to YAML with options
	output, err := yaml.Dump(targetPtr, opts...)
	if err != nil {
		t.Fatalf("Dump failed: %v", err)
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

func TestOptsYAML(t *testing.T) {
	tests := []struct {
		name      string
		yamlStr   string
		expectErr bool
		errMatch  string
	}{
		{
			name:      "valid options",
			yamlStr:   "indent: 4\nknown-fields: true",
			expectErr: false,
		},
		{
			name:      "typo in field name",
			yamlStr:   "knnown-fields: true",
			expectErr: true,
			errMatch:  "knnown-fields not found",
		},
		{
			name:      "another typo",
			yamlStr:   "indnt: 2",
			expectErr: true,
			errMatch:  "indnt not found",
		},
		{
			name:      "multiple options with one typo",
			yamlStr:   "indent: 2\nunicoode: true",
			expectErr: true,
			errMatch:  "unicoode not found",
		},
		{
			name: "all valid options",
			yamlStr: `
indent: 2
compact-seq-indent: true
line-width: 80
unicode: true
canonical: false
line-break: ln
explicit-start: true
explicit-end: false
flow-simple-coll: true
known-fields: true
single-document: true
unique-keys: true
`,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt, err := yaml.OptsYAML(tt.yamlStr)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				if tt.errMatch != "" && !strings.Contains(err.Error(), tt.errMatch) {
					t.Errorf("expected error to contain %q, got: %v", tt.errMatch, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if opt == nil {
					t.Fatal("expected non-nil option")
				}
			}
		})
	}
}

// FuzzEncodeFromJSON checks that any JSON encoded value can also be encoded as YAML... and decoded.
func FuzzEncodeFromJSON(f *testing.F) {
	// Load seed corpus from testdata YAML file
	cases, err := datatest.LoadTestCasesFromFile("testdata/fuzz_json_roundtrip.yaml", libyaml.LoadYAML)
	if err != nil {
		f.Fatalf("Failed to load seed corpus: %v", err)
	}

	// Add each seed to the fuzz corpus
	for _, tc := range cases {
		if jsonInput, ok := datatest.GetString(tc, "json"); ok {
			f.Add(jsonInput)
		}
	}

	f.Fuzz(func(t *testing.T, s string) {
		var v any
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			t.Skipf("not valid JSON %q", s)
		}

		t.Logf("JSON %q", s)
		t.Logf("Go   %q <%[1]x>", v)

		// Encode as YAML
		b, err := yaml.Marshal(v)
		if err != nil {
			t.Error(err)
		}
		t.Logf("YAML %q <%[1]x>", b)

		// Decode as YAML
		var v2 any
		if err := yaml.Unmarshal(b, &v2); err != nil {
			t.Error(err)
		}

		t.Logf("Go   %q <%[1]x>", v2)

		b2, err := yaml.Marshal(v2)
		if err != nil {
			t.Error(err)
		}
		t.Logf("YAML %q <%[1]x>", b2)

		if !bytes.Equal(b, b2) {
			t.Errorf("Marshal->Unmarshal->Marshal mismatch:\n- expected: %q\n- got:      %q", b, b2)
		}
	})
}

func TestLimits(t *testing.T) {
	datatest.RunTestCases(t, func() ([]map[string]any, error) {
		return datatest.LoadTestCasesFromFile("testdata/limit.yaml", libyaml.LoadYAML)
	}, map[string]datatest.TestHandler{
		"limit":       runLimitTest,
		"limit-error": runLimitTest,
		"limit-pass":  runLimitTest,
	})
}

func runLimitTest(t *testing.T, tc map[string]any) {
	t.Helper()

	// Generate data from spec
	dataSpec := tc["data"]
	data, err := datatest.GenerateData(dataSpec)
	if err != nil {
		t.Fatalf("Failed to generate data: %v", err)
	}

	// Get expected error if any (for limit-error tests)
	// For limit-pass tests, want might be a map describing expected structure
	expectedError := ""
	if wantVal, hasWant := tc["want"]; hasWant {
		switch v := wantVal.(type) {
		case string:
			expectedError = v
		case map[string]any:
			// Future: could validate structure here
			// For now, just ignore (treated as success case)
		default:
			t.Fatalf("want field must be a string or map, got %T", wantVal)
		}
	}

	// Run unmarshal
	var v any
	err = yaml.Unmarshal(data, &v)
	if expectedError != "" {
		if err == nil {
			t.Fatalf("expected error %q, got nil", expectedError)
		}
		assert.Equal(t, expectedError, err.Error())
		return
	}
	assert.NoError(t, err)
}

// Keep benchmark using hardcoded data for performance consistency
var limitTests = []struct {
	name  string
	data  []byte
	error string
}{
	{
		name:  "1000kb of maps with 100 aliases",
		data:  []byte(`{a: &a [{a}` + strings.Repeat(`,{a}`, 1000*1024/4-100) + `], b: &b [*a` + strings.Repeat(`,*a`, 99) + `]}`),
		error: "yaml: document contains excessive aliasing",
	},
	{
		name:  "1000kb of deeply nested slices",
		data:  []byte(strings.Repeat(`[`, 1000*1024)),
		error: "yaml: while increasing flow level at line 1, column 10001: exceeded max depth of 10000",
	},
	{
		name:  "1000kb of deeply nested maps",
		data:  []byte("x: " + strings.Repeat(`{`, 1000*1024)),
		error: "yaml: while increasing flow level at line 1, column 10004: exceeded max depth of 10000",
	},
	{
		name:  "1000kb of deeply nested indents",
		data:  []byte(strings.Repeat(`- `, 1000*1024)),
		error: "yaml: while increasing indent level at line 1: line 1, column 20001: exceeded max depth of 10000",
	},
	{
		name: "1000kb of 1000-indent lines",
		data: []byte(strings.Repeat(strings.Repeat(`- `, 1000)+"\n", 1024/2)),
	},
	{name: "1kb of maps", data: []byte(`a: &a [{a}` + strings.Repeat(`,{a}`, 1*1024/4-1) + `]`)},
	{name: "10kb of maps", data: []byte(`a: &a [{a}` + strings.Repeat(`,{a}`, 10*1024/4-1) + `]`)},
	{name: "100kb of maps", data: []byte(`a: &a [{a}` + strings.Repeat(`,{a}`, 100*1024/4-1) + `]`)},
	{name: "1000kb of maps", data: []byte(`a: &a [{a}` + strings.Repeat(`,{a}`, 1000*1024/4-1) + `]`)},
	{name: "1000kb slice nested at max-depth", data: []byte(strings.Repeat(`[`, 10000) + `1` + strings.Repeat(`,1`, 1000*1024/2-20000-1) + strings.Repeat(`]`, 10000))},
	{name: "1000kb slice nested in maps at max-depth", data: []byte("{a,b:\n" + strings.Repeat(" {a,b:", 10000-2) + ` [1` + strings.Repeat(",1", 1000*1024/2-6*10000-1) + `]` + strings.Repeat(`}`, 10000-1))},
	{name: "1000kb of 10000-nested lines", data: []byte(strings.Repeat(`- `+strings.Repeat(`[`, 10000)+strings.Repeat(`]`, 10000)+"\n", 1000*1024/20000))},
}

func BenchmarkLimits(b *testing.B) {
	for _, tc := range limitTests {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				var v any
				err := yaml.Unmarshal(tc.data, &v)
				if tc.error != "" {
					assert.ErrorMatches(b, tc.error, err)
					continue
				}
				assert.NoError(b, err)
			}
		})
	}
}

func TestParserGetEvents(t *testing.T) {
	datatest.RunTestCases(t, func() ([]map[string]any, error) {
		return datatest.LoadTestCasesFromFile("testdata/parser_events.yaml", libyaml.LoadYAML)
	}, map[string]datatest.TestHandler{
		"parser-events": runParserEventsTest,
	})
}

func runParserEventsTest(t *testing.T, tc map[string]any) {
	t.Helper()

	// Extract test data
	yamlInput := datatest.RequireString(t, tc, "yaml")
	want := datatest.RequireString(t, tc, "want")

	// Run test
	events, err := libyaml.ParserGetEvents([]byte(yamlInput))
	if err != nil {
		t.Fatalf("ParserGetEvents error: %v", err)
	}

	// Trim trailing newline from want (YAML literal blocks add one)
	want = datatest.TrimTrailingNewline(want)

	assert.Equal(t, want, events)
}
