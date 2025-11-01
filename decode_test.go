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
	"encoding"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

// negativeZero represents -0.0 for YAML test cases
// this is needed because Go constants cannot express -0.0
// https://staticcheck.dev/docs/checks/#SA4026
var negativeZero = math.Copysign(0.0, -1.0)

var unmarshalIntTest = 123

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
	{
		"v: hi",
		map[string]string{"v": "hi"},
	},
	{
		"v: hi", map[string]any{"v": "hi"},
	},
	{
		"v: true",
		map[string]string{"v": "true"},
	},
	{
		"v: true",
		map[string]any{"v": true},
	},
	{
		"v: 10",
		map[string]any{"v": 10},
	},
	{
		"v: 0b10",
		map[string]any{"v": 2},
	},
	{
		"v: 0xA",
		map[string]any{"v": 10},
	},
	{
		"v: 4294967296",
		map[string]int64{"v": 4294967296},
	},
	{
		"v: 0.1",
		map[string]any{"v": 0.1},
	},
	{
		"v: .1",
		map[string]any{"v": 0.1},
	},
	{
		"v: .Inf",
		map[string]any{"v": math.Inf(+1)},
	},
	{
		"v: -.Inf",
		map[string]any{"v": math.Inf(-1)},
	},
	{
		"v: -10",
		map[string]any{"v": -10},
	},
	{
		"v: -.1",
		map[string]any{"v": -0.1},
	},
	{
		"v: -0\n",
		map[string]any{"v": negativeZero},
	},
	{
		"a: \"\\t\\n\\t\\n\"\n",
		map[string]string{"a": "\t\n\t\n"},
	},
	{
		"\"<<\": []\n",
		map[string]any{"<<": []any{}},
	},
	{
		"foo: \"<<\"\n",
		map[string]any{"foo": "<<"},
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

	// Floats from spec
	{
		"canonical: 6.8523e+5",
		map[string]any{"canonical": 6.8523e+5},
	},
	{
		"expo: 685.230_15e+03",
		map[string]any{"expo": 685.23015e+03},
	},
	{
		"fixed: 685_230.15",
		map[string]any{"fixed": 685230.15},
	},
	{
		"neginf: -.inf",
		map[string]any{"neginf": math.Inf(-1)},
	},
	{
		"fixed: 685_230.15",
		map[string]float64{"fixed": 685230.15},
	},
	//{"sexa: 190:20:30.15", map[string]any{"sexa": 0}}, // Unsupported
	//{"notanum: .NaN", map[string]any{"notanum": math.NaN()}}, // Equality of NaN fails.

	// Bools are per 1.2 spec.
	{
		"canonical: true",
		map[string]any{"canonical": true},
	},
	{
		"canonical: false",
		map[string]any{"canonical": false},
	},
	{
		"bool: True",
		map[string]any{"bool": true},
	},
	{
		"bool: False",
		map[string]any{"bool": false},
	},
	{
		"bool: TRUE",
		map[string]any{"bool": true},
	},
	{
		"bool: FALSE",
		map[string]any{"bool": false},
	},
	// For backwards compatibility with 1.1, decoding old strings into typed values still works.
	{
		"option: on",
		map[string]bool{"option": true},
	},
	{
		"option: y",
		map[string]bool{"option": true},
	},
	{
		"option: Off",
		map[string]bool{"option": false},
	},
	{
		"option: No",
		map[string]bool{"option": false},
	},
	{
		"option: other",
		map[string]bool{},
	},
	// Ints from spec
	{
		"canonical: 685230",
		map[string]any{"canonical": 685230},
	},
	{
		"decimal: +685_230",
		map[string]any{"decimal": 685230},
	},
	{
		"octal: 02472256",
		map[string]any{"octal": 685230},
	},
	{
		"octal: -02472256",
		map[string]any{"octal": -685230},
	},
	{
		"octal: 0o2472256",
		map[string]any{"octal": 685230},
	},
	{
		"octal: -0o2472256",
		map[string]any{"octal": -685230},
	},
	{
		"hexa: 0x_0A_74_AE",
		map[string]any{"hexa": 685230},
	},
	{
		"bin: 0b1010_0111_0100_1010_1110",
		map[string]any{"bin": 685230},
	},
	{
		"bin: -0b101010",
		map[string]any{"bin": -42},
	},
	{
		"bin: -0b1000000000000000000000000000000000000000000000000000000000000000",
		map[string]any{"bin": -9223372036854775808},
	},
	{
		"decimal: +685_230",
		map[string]int{"decimal": 685230},
	},

	//{"sexa: 190:20:30", map[string]any{"sexa": 0}}, // Unsupported

	// Nulls from spec
	{
		"empty:",
		map[string]any{"empty": nil},
	},
	{
		"canonical: ~",
		map[string]any{"canonical": nil},
	},
	{
		"english: null",
		map[string]any{"english": nil},
	},
	{
		"~: null key",
		map[any]string{nil: "null key"},
	},
	{
		"empty:",
		map[string]*bool{"empty": nil},
	},

	// Flow sequence
	{
		"seq: [A,B]",
		map[string]any{"seq": []any{"A", "B"}},
	},
	{
		"seq: [A,B,C,]",
		map[string][]string{"seq": {"A", "B", "C"}},
	},
	{
		"seq: [A,1,C]",
		map[string][]string{"seq": {"A", "1", "C"}},
	},
	{
		"seq: [A,1,C]",
		map[string][]int{"seq": {1}},
	},
	{
		"seq: [A,1,C]",
		map[string]any{"seq": []any{"A", 1, "C"}},
	},
	// Block sequence
	{
		"seq:\n - A\n - B",
		map[string]any{"seq": []any{"A", "B"}},
	},
	{
		"seq:\n - A\n - B\n - C",
		map[string][]string{"seq": {"A", "B", "C"}},
	},
	{
		"seq:\n - A\n - 1\n - C",
		map[string][]string{"seq": {"A", "1", "C"}},
	},
	{
		"seq:\n - A\n - 1\n - C",
		map[string][]int{"seq": {1}},
	},
	{
		"seq:\n - A\n - 1\n - C",
		map[string]any{"seq": []any{"A", 1, "C"}},
	},

	// Literal block scalar
	{
		"scalar: | # Comment\n\n literal\n\n \ttext\n\n",
		map[string]string{"scalar": "\nliteral\n\n\ttext\n"},
	},

	// Folded block scalar
	{
		"scalar: > # Comment\n\n folded\n line\n \n next\n line\n  * one\n  * two\n\n last\n line\n\n",
		map[string]string{"scalar": "\nfolded line\nnext line\n * one\n * two\n\nlast line\n"},
	},

	// Map inside interface with no type hints.
	{
		"a: {b: c}",
		map[any]any{"a": map[string]any{"b": "c"}},
	},
	// Non-string map inside interface with no type hints.
	{
		"a: {b: c, 1: d}",
		map[any]any{"a": map[any]any{"b": "c", 1: "d"}},
	},

	// Structs and type conversions.
	{
		"hello: world",
		&struct{ Hello string }{"world"},
	},
	{
		"a: {b: c}",
		&struct{ A struct{ B string } }{struct{ B string }{"c"}},
	},
	{
		"a: {b: c}",
		&struct{ A *struct{ B string } }{&struct{ B string }{"c"}},
	},
	{
		"a: 'null'",
		&struct{ A *unmarshalerType }{&unmarshalerType{"null"}},
	},
	{
		"a: {b: c}",
		&struct{ A map[string]string }{map[string]string{"b": "c"}},
	},
	{
		"a: {b: c}",
		&struct{ A *map[string]string }{&map[string]string{"b": "c"}},
	},
	{
		"a:",
		&struct{ A map[string]string }{},
	},
	{
		"a: 1",
		&struct{ A int }{1},
	},
	{
		"a: 1",
		&struct{ A float64 }{1},
	},
	{
		"a: 1.0",
		&struct{ A int }{1},
	},
	{
		"a: 1.0",
		&struct{ A uint }{1},
	},
	{
		"a: [1, 2]",
		&struct{ A []int }{[]int{1, 2}},
	},
	{
		"a: [1, 2]",
		&struct{ A [2]int }{[2]int{1, 2}},
	},
	{
		"a: 1",
		&struct{ B int }{0},
	},
	{
		"a: 1",
		&struct {
			B int `yaml:"a"`
		}{1},
	},
	{
		// Some limited backwards compatibility with the 1.1 spec.
		"a: YES",
		&struct{ A bool }{true},
	},

	// Some cross type conversions
	{
		"v: 42",
		map[string]uint{"v": 42},
	},
	{
		"v: -42",
		map[string]uint{},
	},
	{
		"v: 4294967296",
		map[string]uint64{"v": 4294967296},
	},
	{
		"v: -4294967296",
		map[string]uint64{},
	},

	// int
	{
		"int_max: 2147483647",
		map[string]int{"int_max": math.MaxInt32},
	},
	{
		"int_min: -2147483648",
		map[string]int{"int_min": math.MinInt32},
	},
	{
		"int_overflow: 9223372036854775808", // math.MaxInt64 + 1
		map[string]int{},
	},

	// int64
	{
		"int64_max: 9223372036854775807",
		map[string]int64{"int64_max": math.MaxInt64},
	},
	{
		"int64_max_base2: 0b111111111111111111111111111111111111111111111111111111111111111",
		map[string]int64{"int64_max_base2": math.MaxInt64},
	},
	{
		"int64_min: -9223372036854775808",
		map[string]int64{"int64_min": math.MinInt64},
	},
	{
		"int64_neg_base2: -0b111111111111111111111111111111111111111111111111111111111111111",
		map[string]int64{"int64_neg_base2": -math.MaxInt64},
	},
	{
		"int64_overflow: 9223372036854775808", // math.MaxInt64 + 1
		map[string]int64{},
	},

	// uint
	{
		"uint_min: 0",
		map[string]uint{"uint_min": 0},
	},
	{
		"uint_max: 4294967295",
		map[string]uint{"uint_max": math.MaxUint32},
	},
	{
		"uint_underflow: -1",
		map[string]uint{},
	},

	// uint64
	{
		"uint64_min: 0",
		map[string]uint{"uint64_min": 0},
	},
	{
		"uint64_max: 18446744073709551615",
		map[string]uint64{"uint64_max": math.MaxUint64},
	},
	{
		"uint64_max_base2: 0b1111111111111111111111111111111111111111111111111111111111111111",
		map[string]uint64{"uint64_max_base2": math.MaxUint64},
	},
	{
		"uint64_maxint64: 9223372036854775807",
		map[string]uint64{"uint64_maxint64": math.MaxInt64},
	},
	{
		"uint64_underflow: -1",
		map[string]uint64{},
	},

	// float32
	{
		"float32_max: 3.40282346638528859811704183484516925440e+38",
		map[string]float32{"float32_max": math.MaxFloat32},
	},
	{
		"float32_nonzero: 1.401298464324817070923729583289916131280e-45",
		map[string]float32{"float32_nonzero": math.SmallestNonzeroFloat32},
	},
	{
		"float32_maxuint64: 18446744073709551615",
		map[string]float32{"float32_maxuint64": float32(math.MaxUint64)},
	},
	{
		"float32_maxuint64+1: 18446744073709551616",
		map[string]float32{"float32_maxuint64+1": float32(math.MaxUint64 + 1)},
	},

	// float64
	{
		"float64_max: 1.797693134862315708145274237317043567981e+308",
		map[string]float64{"float64_max": math.MaxFloat64},
	},
	{
		"float64_nonzero: 4.940656458412465441765687928682213723651e-324",
		map[string]float64{"float64_nonzero": math.SmallestNonzeroFloat64},
	},
	{
		"float64_maxuint64: 18446744073709551615",
		map[string]float64{"float64_maxuint64": float64(math.MaxUint64)},
	},
	{
		"float64_maxuint64+1: 18446744073709551616",
		map[string]float64{"float64_maxuint64+1": float64(math.MaxUint64 + 1)},
	},

	// Overflow cases.
	{
		"v: 4294967297",
		map[string]int32{},
	},
	{
		"v: 128",
		map[string]int8{},
	},

	// Quoted values.
	{
		"'1': '\"2\"'",
		map[any]any{"1": "\"2\""},
	},
	{
		"v:\n- A\n- 'B\n\n  C'\n",
		map[string][]string{"v": {"A", "B\nC"}},
	},

	// Explicit tags.
	{
		"v: !!float '1.1'",
		map[string]any{"v": 1.1},
	},
	{
		"v: !!float 0",
		map[string]any{"v": float64(0)},
	},
	{
		"v: !!float -1",
		map[string]any{"v": float64(-1)},
	},
	{
		"v: !!null ''",
		map[string]any{"v": nil},
	},
	{
		"%TAG !y! tag:yaml.org,2002:\n---\nv: !y!int '1'",
		map[string]any{"v": 1},
	},

	// Non-specific tag (Issue #75)
	{
		"v: ! test",
		map[string]any{"v": "test"},
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
	{
		// alias must be followed by a space in mapping node
		"foo: &bar bar\n*bar : quz\n",
		map[string]any{"foo": "bar", "bar": "quz"},
	},

	{
		// alias can contain various characters specified by the YAML specification
		"foo: &b./ar bar\n*b./ar : quz\n",
		map[string]any{"foo": "bar", "bar": "quz"},
	},

	// Bug #1133337
	{
		"foo: ''",
		map[string]*string{"foo": new(string)},
	},
	{
		"foo: null",
		map[string]*string{"foo": nil},
	},
	{
		"foo: null",
		map[string]string{"foo": ""},
	},
	{
		"foo: null",
		map[string]any{"foo": nil},
	},

	// Support for ~
	{
		"foo: ~",
		map[string]*string{"foo": nil},
	},
	{
		"foo: ~",
		map[string]string{"foo": ""},
	},
	{
		"foo: ~",
		map[string]any{"foo": nil},
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

	// Struct inlining
	{
		"a: 1\nb: 2\nc: 3\n",
		&struct {
			A int
			C inlineB `yaml:",inline"`
		}{1, inlineB{2, inlineC{3}}},
	},

	// Struct inlining as a pointer.
	{
		"a: 1\nb: 2\nc: 3\n",
		&struct {
			A int
			C *inlineB `yaml:",inline"`
		}{1, &inlineB{2, inlineC{3}}},
	},
	{
		"a: 1\n",
		&struct {
			A int
			C *inlineB `yaml:",inline"`
		}{1, nil},
	},
	{
		"a: 1\nc: 3\nd: 4\n",
		&struct {
			A int
			C *inlineD `yaml:",inline"`
		}{1, &inlineD{&inlineC{3}, 4}},
	},

	// Map inlining
	{
		"a: 1\nb: 2\nc: 3\n",
		&struct {
			A int
			C map[string]int `yaml:",inline"`
		}{1, map[string]int{"b": 2, "c": 3}},
	},

	// bug 1243827
	{
		"a: -b_c",
		map[string]any{"a": "-b_c"},
	},
	{
		"a: +b_c",
		map[string]any{"a": "+b_c"},
	},
	{
		"a: 50cent_of_dollar",
		map[string]any{"a": "50cent_of_dollar"},
	},

	// issue #295 (allow scalars with colons in flow mappings and sequences)
	{
		"a: {b: https://github.com/go-yaml/yaml}",
		map[string]any{"a": map[string]any{
			"b": "https://github.com/go-yaml/yaml",
		}},
	},
	{
		"a: [https://github.com/go-yaml/yaml]",
		map[string]any{"a": []any{"https://github.com/go-yaml/yaml"}},
	},

	// Duration
	{
		"a: 3s",
		map[string]time.Duration{"a": 3 * time.Second},
	},

	// Issue #24.
	{
		"a: <foo>",
		map[string]string{"a": "<foo>"},
	},

	// Base 60 floats are obsolete and unsupported.
	{
		"a: 1:1\n",
		map[string]string{"a": "1:1"},
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
		// quoted string that's a valid timestamp
		"a: \"2015-01-01\"",
		map[string]any{"a": "2015-01-01"},
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

	// Encode empty lists as zero-length slices.
	{
		"a: []",
		&struct{ A []int }{[]int{}},
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

	// This *is* in fact a float number, per the spec. #171 was a mistake.
	{
		"a: 123456e1\n",
		M{"a": 123456e1},
	},
	{
		"a: 123456E1\n",
		M{"a": 123456e1},
	},
	// yaml-test-suite 3GZX: Spec Example 7.1. Alias Nodes
	{
		"First occurrence: &anchor Foo\nSecond occurrence: *anchor\nOverride anchor: &anchor Bar\nReuse anchor: *anchor\n",
		map[string]any{
			"First occurrence":  "Foo",
			"Second occurrence": "Foo",
			"Override anchor":   "Bar",
			"Reuse anchor":      "Bar",
		},
	},
	// Single document with garbage following it.
	{
		"---\nhello\n...\n}not yaml",
		"hello",
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

	// CRLF
	{
		"a: b\r\nc:\r\n- d\r\n- e\r\n",
		map[string]any{
			"a": "b",
			"c": []any{"d", "e"},
		},
	},
	// bug: question mark in value
	{
		"foo: {ba?r: a?bc}",
		map[string]any{
			"foo": map[string]any{"ba?r": "a?bc"},
		},
	},
	{
		"foo: {?bar: ?abc}",
		map[string]any{
			"foo": map[string]any{"?bar": "?abc"},
		},
	},
	{
		"foo: {bar?: abc?}",
		map[string]any{
			"foo": map[string]any{"bar?": "abc?"},
		},
	},
	{
		"foo: {? key: value}",
		map[string]any{
			"foo": map[string]any{"key": "value"},
		},
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
	assert.ErrorMatches(t, `yaml: input error: some read error`, err)
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

var unmarshalErrorTests = []struct {
	data, error string
}{
	{"v: !!float 'error'", "yaml: cannot decode !!str `error` as a !!float"},
	{"v: [A,", "yaml: line 1: did not find expected node content"},
	{"v:\n- [A,", "yaml: line 2: did not find expected node content"},
	{"a:\n- b: *,", "yaml: line 2: did not find expected alphabetic or numeric character"},
	{"a: *b\n", "yaml: unknown anchor 'b' referenced"},
	{"a: &a\n  b: *a\n", "yaml: anchor 'a' value contains itself"},
	{"value: -", "yaml: block sequence entries are not allowed in this context"},
	{"a: !!binary ==", "yaml: !!binary value contains invalid base64 data"},
	{"{[.]}", `yaml: cannot use '\[\]interface \{\}\{"\."\}' as a map key; try decoding into yaml.Node`},
	{"{{.}}", `yaml: cannot use 'map\[string]interface \{\}\{".":interface \{\}\(nil\)\}' as a map key; try decoding into yaml.Node`},
	{"b: *a\na: &a {c: 1}", `yaml: unknown anchor 'a' referenced`},
	{"%TAG !%79! tag:yaml.org,2002:\n---\nv: !%79!int '1'", "yaml: did not find expected whitespace"},
	{"a:\n  1:\nb\n  2:", ".*could not find expected ':'"},
	{"a: 1\nb: 2\nc 2\nd: 3\n", "^yaml: line 3: could not find expected ':'$"},
	{"#\n-\n{", "yaml: line 3: could not find expected ':'"},   // Issue #665
	{"0: [:!00 \xef", "yaml: incomplete UTF-8 octet sequence"}, // Issue #666
	// anchor cannot contain a colon
	// https://github.com/yaml/go-yaml/issues/109
	{"foo: &bar: bar\n*bar: : quz\n", "^yaml: mapping values are not allowed in this context$"},
	{
		"a: &a [00,00,00,00,00,00,00,00,00]\n" +
			"b: &b [*a,*a,*a,*a,*a,*a,*a,*a,*a]\n" +
			"c: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b]\n" +
			"d: &d [*c,*c,*c,*c,*c,*c,*c,*c,*c]\n" +
			"e: &e [*d,*d,*d,*d,*d,*d,*d,*d,*d]\n" +
			"f: &f [*e,*e,*e,*e,*e,*e,*e,*e,*e]\n" +
			"g: &g [*f,*f,*f,*f,*f,*f,*f,*f,*f]\n" +
			"h: &h [*g,*g,*g,*g,*g,*g,*g,*g,*g]\n" +
			"i: &i [*h,*h,*h,*h,*h,*h,*h,*h,*h]\n",
		"yaml: document contains excessive aliasing",
	},
}

func TestUnmarshalErrors(t *testing.T) {
	for i, item := range unmarshalErrorTests {
		t.Run(fmt.Sprintf("test %d: %q", i, item.data), func(t *testing.T) {
			var value any
			err := yaml.Unmarshal([]byte(item.data), &value)
			assert.ErrorMatchesf(t, item.error, err, "Partial unmarshal: %#v", value)
		})
	}
}

func TestDecoderErrors(t *testing.T) {
	for i, item := range unmarshalErrorTests {
		t.Run(fmt.Sprintf("test %d: %q", i, item.data), func(t *testing.T) {
			var value any
			err := yaml.NewDecoder(strings.NewReader(item.data)).Decode(&value)
			assert.ErrorMatchesf(t, item.error, err, "Partial unmarshal: %#v", value)
		})
	}
}

func TestParserErrorUnmarshal(t *testing.T) {
	var v struct {
		A, B int
	}
	data := "a: 1\n=\nb: 2"
	err := yaml.Unmarshal([]byte(data), &v)
	asErr := new(yaml.ParserError)
	assert.ErrorAs(t, err, &asErr)
	expectedErr := &yaml.ParserError{
		Message: "could not find expected ':'",
		Line:    2,
		Column:  0,
	}
	assert.DeepEqual(t, expectedErr, asErr)
}

func TestParserErrorDecoder(t *testing.T) {
	var v any
	data := "value: -"
	err := yaml.NewDecoder(strings.NewReader(data)).Decode(&v)
	asErr := new(yaml.ParserError)
	assert.ErrorAs(t, err, &asErr)
	expectedErr := &yaml.ParserError{
		Message: "block sequence entries are not allowed in this context",
		Line:    0,
		Column:  7,
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
	unmarshalerResult[2] = &yaml.TypeError{[]*yaml.UnmarshalError{{Err: errors.New("foo"), Line: 1, Column: 1}}}
	unmarshalerResult[4] = &yaml.TypeError{[]*yaml.UnmarshalError{{Err: errors.New("bar"), Line: 1, Column: 1}}}
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
	unmarshalerResult[2] = &yaml.TypeError{[]*yaml.UnmarshalError{{Err: errors.New("foo"), Line: 1, Column: 1}}}
	unmarshalerResult[4] = &yaml.TypeError{[]*yaml.UnmarshalError{{Err: errors.New("bar"), Line: 1, Column: 1}}}
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

func TestFuzzCrashers(t *testing.T) {
	cases := []string{
		// runtime error: index out of range
		"\"\\0\\\r\n",

		// should not happen
		"  0: [\n] 0",
		"? ? \"\n\" 0",
		"    - {\n000}0",
		"0:\n  0: [0\n] 0",
		"    - \"\n000\"0",
		"    - \"\n000\"\"",
		"0:\n    - {\n000}0",
		"0:\n    - \"\n000\"0",
		"0:\n    - \"\n000\"\"",

		// runtime error: index out of range
		" \ufeff\n",
		"? \ufeff\n",
		"? \ufeff:\n",
		"0: \ufeff\n",
		"? \ufeff: \ufeff\n",
	}
	for _, data := range cases {
		var v any
		_ = yaml.Unmarshal([]byte(data), &v)
	}
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

//var data []byte
//func init() {
//	var err error
//	data, err = ioutil.ReadFile("/tmp/file.yaml")
//	if err != nil {
//		panic(err)
//	}
//}
//
//func (s *S) BenchmarkUnmarshal(c *C) {
//	var err error
//	for i := 0; i < c.N; i++ {
//		var v map[string]any
//		err = yaml.Unmarshal(data, &v)
//	}
//	if err != nil {
//		panic(err)
//	}
//}
//
//func (s *S) BenchmarkMarshal(c *C) {
//	var v map[string]any
//	yaml.Unmarshal(data, &v)
//	c.ResetTimer()
//	for i := 0; i < c.N; i++ {
//		yaml.Marshal(&v)
//	}
//}
