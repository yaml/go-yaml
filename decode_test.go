// Copyright 2011-2019 Canonical Ltd
// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"io"
	"math"
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

	// Handle unmarshal errors - if error occurred, check if want is empty/nil
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
	assert.ErrorMatches(t, "line 1: cannot unmarshal !!int `123` into time.Duration", err)
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
	if err := value.Decode(&o.value); err != nil {
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
	unmarshalerResult[2] = &yaml.TypeError{Errors: []*yaml.UnmarshalError{{Err: errors.New("foo"), Line: 1, Column: 1}}}
	unmarshalerResult[4] = &yaml.TypeError{Errors: []*yaml.UnmarshalError{{Err: errors.New("bar"), Line: 1, Column: 1}}}
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
		"yaml: unmarshal errors:\n" +
		"  line 1: cannot unmarshal !!str `A` into int\n" +
		"  line 1: foo\n" +
		"  line 1: bar\n" +
		"  line 1: cannot unmarshal !!str `B` into int"
	assert.ErrorMatches(t, expectedError, err)
	assert.NotNil(t, v.M["abc"])
	assert.IsNil(t, v.M["def"])
	assert.NotNil(t, v.M["ghi"])
	assert.IsNil(t, v.M["jkl"])

	assert.Equal(t, 1, v.M["abc"].value)
	assert.Equal(t, 3, v.M["ghi"].value)
}

func TestObsoleteUnmarshalerTypeError(t *testing.T) {
	unmarshalerResult[2] = &yaml.TypeError{Errors: []*yaml.UnmarshalError{{Err: errors.New("foo"), Line: 1, Column: 1}}}
	unmarshalerResult[4] = &yaml.TypeError{Errors: []*yaml.UnmarshalError{{Err: errors.New("bar"), Line: 1, Column: 1}}}
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
		"yaml: unmarshal errors:\n" +
		"  line 1: cannot unmarshal !!str `A` into int\n" +
		"  line 1: foo\n" +
		"  line 1: bar\n" +
		"  line 1: cannot unmarshal !!str `B` into int"
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

	errUnmarshal := &yaml.UnmarshalError{
		Line:   1,
		Column: 2,
		Err:    errSentinel,
	}

	errUnmarshal2 := &yaml.UnmarshalError{
		Line:   2,
		Column: 2,
		Err:    errSentinel2,
	}

	// Simulate a TypeError
	err := &yaml.TypeError{
		Errors: []*yaml.UnmarshalError{
			errUnmarshal,
			errUnmarshal2,
		},
	}

	var errTarget *yaml.UnmarshalError
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

	errUnmarshal := &yaml.UnmarshalError{
		Line:   1,
		Column: 2,
		Err:    errSentinel,
	}

	errUnmarshal2 := &yaml.UnmarshalError{
		Line:   2,
		Column: 2,
		Err:    errors.New("bar"),
	}

	// Simulate a TypeError
	err := &yaml.TypeError{
		Errors: []*yaml.UnmarshalError{
			errUnmarshal,
			errUnmarshal2,
		},
	}

	var errTarget *yaml.UnmarshalError
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
	if err := node.Decode(&s); err != nil {
		panic(err)
	}
	if s == "a" {
		if err := node.Decode(&b); err == nil {
			panic("should have failed")
		}
		return node.Decode(&a)
	}
	if err := node.Decode(&a); err == nil {
		panic("should have failed")
	}
	return node.Decode(&b)
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
		"yaml: unmarshal errors:\n" +
		"  line 1: cannot unmarshal !!str `A` into int\n" +
		"  line 1: cannot unmarshal !!str `a` into int32\n" +
		"  line 1: cannot unmarshal !!str `b` into int64\n" +
		"  line 1: cannot unmarshal !!str `B` into int"
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
		"yaml: unmarshal errors:\n" +
		"  line 1: cannot unmarshal !!str `A` into int\n" +
		"  line 1: cannot unmarshal !!str `a` into int32\n" +
		"  line 1: cannot unmarshal !!str `b` into int64\n" +
		"  line 1: cannot unmarshal !!str `B` into int"
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
		Errors: []*yaml.UnmarshalError{
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
		Errors: []*yaml.UnmarshalError{
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
		Errors: []*yaml.UnmarshalError{
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

	errUnmarshal := &yaml.UnmarshalError{
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
	err := node.Decode(&slice)
	if err == nil {
		*su = slice
		return nil
	}

	var intVal int
	err = node.Decode(&intVal)
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
	error: `yaml: unmarshal errors:\n  line 2: field c not found in type struct { A int; B int }`,
}, {
	unique: true,
	data:   "a: 1\nb: 2\na: 3\n",
	value:  struct{ A, B int }{A: 3, B: 2},
	error:  `yaml: unmarshal errors:\n  line 3: mapping key "a" already defined at line 1`,
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
	error: `yaml: unmarshal errors:\n  line 4: mapping key "c" already defined at line 1`,
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
	error: `yaml: unmarshal errors:\n  line 4: mapping key "c" already defined at line 1`,
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
	error: `yaml: unmarshal errors:\n  line 4: mapping key "c" already defined at line 1`,
}, {
	unique: true,
	data:   "a: 1\n9: 2\nnull: 3\n9: 4",
	value: map[any]any{
		"a": 1,
		nil: 3,
		9:   4,
	},
	error: `yaml: unmarshal errors:\n  line 4: mapping key "9" already defined at line 2`,
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
		Errors: []*yaml.UnmarshalError{
			{Err: errors.New("cannot unmarshal string into int"), Line: 5, Column: 3},
			{Err: errors.New("cannot unmarshal bool into string"), Line: 10, Column: 7},
		},
	}

	strings := typeErr.Strings()

	assert.Equal(t, 2, len(strings))
	assert.Equal(t, "line 5: cannot unmarshal string into int", strings[0])
	assert.Equal(t, "line 10: cannot unmarshal bool into string", strings[1])
}
