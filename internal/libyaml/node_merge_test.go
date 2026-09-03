// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for node_merge.go functions and methods.
package libyaml

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

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
  !<tag:org,2002:merge> "<<" : [ *CENTER, *BIG ]
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

func TestMergeYAMLMapNodes(t *testing.T) {
	tests := []struct {
		name      string
		dst       string
		src       string
		expected  string
		seqAppend bool // false by default
	}{
		{
			name:     "scalar values",
			dst:      `"original"`,
			src:      `"updated"`,
			expected: `"updated"`,
		},
		{
			name:     "empty to empty",
			dst:      "",
			src:      "",
			expected: "",
		},
		{
			name:     "merge to empty",
			dst:      "",
			src:      "foo: bar",
			expected: "foo: bar",
		},
		{
			name:     "empty to something",
			dst:      "foo: bar",
			src:      "",
			expected: "foo: bar",
		},
		{
			name: "empty doc to something",
			dst:  "foo: bar",
			// doc with scalar
			src: "---",
			// TODO: should it be different (e.g., dst)?
			expected: "---",
		},
		{
			name:     "empty({}) to something",
			dst:      "foo: bar",
			src:      "{}",
			expected: "foo: bar",
		},
		{
			name: "mapping nodes - simple merge to empty",
			dst:  "",
			src: `
key2: updated_value2
key3: value3
`,
			expected: `
key2: updated_value2
key3: value3
`,
		},
		{
			name: "mapping nodes - simple merge",
			dst: `
key1: value1
key2: value2
`,
			src: `
key2: updated_value2
key3: value3
`,
			expected: `
key1: value1
key2: updated_value2
key3: value3
`,
		},
		{
			name: "mapping nodes - nested merge",
			dst: `
nested:
  key1: value1
  key2: value2
`,
			src: `
nested:
  key2: updated_value2
  key3: value3
`,
			expected: `
nested:
  key1: value1
  key2: updated_value2
  key3: value3
`,
		},
		{
			name: "sequence nodes - replaced",
			dst: `
- item1
- item2
`,
			src: `
- item3
- item4
`,
			expected: `
- item3
- item4
`,
		},
		{
			name:     "different node kinds - does nothing with error",
			dst:      `"scalar"`,
			src:      `[sequence]`,
			expected: `error: invalid node kinds`,
		},
		{
			name: "empty destination {} - if dst is empty replace with src",
			dst:  `{}`,
			src: `
key1: value1
key2: value2
`,
			expected: "{key1: value1, key2: value2}\n",
		},
		{
			name: "empty destination '' - if dst is empty replace with src",
			dst:  ``,
			src: `
key1: value1
key2: value2
`,
			expected: `
key1: value1
key2: value2
`,
		},
		{
			name: "empty source - if src is empty do nothing",
			dst: `
key1: value1
key2: value2
`,
			src: `{}`,
			expected: `
key1: value1
key2: value2
`,
		},
		{
			name: "single docs",
			dst: `
---
key1: value1
key2: value2
`,
			src: `
---
key2: updated_value2
key3: value3
`,
			expected: `
key1: value1
key2: updated_value2
key3: value3
`,
		},
		{
			name: "multiple docs",
			dst: `
---
key1: value1
key2: value2
---
key3: value3
key4: value4
`,
			src: `
---
key2: updated_value2
key3: value3
`,
			expected: `
key1: value1
key2: updated_value2
key3: value3
`,
		},
		{
			name: "merge test from yaml spec",
			dst:  mergeTests,
			src:  mergeTests,
			expected: `
anchors:
    list:
        - &CENTER {"x": 1, "y": 2}
        - &LEFT {"x": 0, "y": 2}
        - &BIG {"r": 10}
        - &SMALL {"r": 1}
# All the following maps are equal:
plain:
    # Explicit keys
    "x": 1
    "y": 2
    "r": 10
    label: center/big
mergeOne:
    # Merge one map
    !!merge <<: *CENTER
    "r": 10
    label: center/big
mergeMultiple:
    # Merge multiple maps
    !!merge <<: [*CENTER, *BIG]
    label: center/big
override:
    # Override
    !!merge <<: [*BIG, *LEFT, *SMALL]
    "x": 1
    label: center/big
shortTag:
    # Explicit short merge tag
    !!merge "<<": [*CENTER, *BIG]
    label: center/big
longTag:
    # Explicit merge long tag
    !<tag:org,2002:merge> "<<": [*CENTER, *BIG]
    label: center/big
inlineMap:
    # Inlined map
    !!merge <<: {"x": 1, "y": 2, "r": 10}
    label: center/big
inlineSequenceMap:
    # Inlined map in sequence
    !!merge <<: [*CENTER, {"r": 10}]
    label: center/big
`,
		},
		{
			name: "merge test 2 and lists got replaced",
			dst: `
anchors:
    list:
        - &BIG {"r": 10}
mergeOne:
    !!merge <<: *BIG
    "a": 1
    "b": 2
`,
			src: `
anchors:
    list:
        - &SMALL {"r": 1}
mergeOne:
    !!merge <<: *SMALL
    "b": 22
    "c": 33
`,
			expected: `
anchors:
    list:
        - &SMALL {"r": 1}
mergeOne:
    !!merge <<: *SMALL
    "a": 1
    "b": 22
    "c": 33
`,
		},
		{
			name: "merge test 3 - extend anchor",
			dst: `
# 1
anchors:
    # 2
    list:
        &BIG {"a": 10}
    # 3
# 4
mergeOne:
    !!merge <<: *BIG
    "a": 1
    "b": 2
# 5
`,
			src: `
# 1x
anchors:
    # 2x
    list:
        &BIG {"b": 10}
    # 3x
# 4x
mergeOne:
    !!merge <<: *BIG
    "b": 22
    "c": 33
# 5x
`,
			expected: `
# 1x
anchors:
    # 2x
    list:
        &BIG {"a": 10, "b": 10}
    # 3x
# 4x
mergeOne:
    !!merge <<: *BIG
    "a": 1
    "b": 22
    "c": 33
# 5x
`,
		},
		{
			name: "merge test 4 - anchor collision",
			dst: `
inner: &I1
    a: 1

outer:
    in: *I1
`,
			src: `
inner: &I2
    b: 2

outer:
    in: *I2
`,
			// I2 is not allowed to replace I1.
			expected: "error: unmergeable error",
		},
		{
			name: "empty keys",
			dst: `
foo:
    bar:
x:
`,
			src: `
foo:
    zoo:
`,
			expected: `
foo:
    bar:
    zoo:
x:
`,
		},
		{
			name: "map onto empty map that is null scalar",
			dst: `
foo:
`,
			src: `
foo:
    zoo:
`,
			expected: `
foo:
    zoo:
`,
		},
		{
			name: "map onto scalar",
			dst: `
foo: 1
`,
			src: `
foo:
    zoo:
`,
			expected: "error: invalid node kinds",
		},
		{
			name: "null scalar onto map",
			dst: `
foo:
    zoo:
`,
			src: `
foo:
`,
			expected: `
foo:
`,
		},
		{
			name: "nested null scalar onto null scalar",
			dst: `
foo:
    zoo:
`,
			src: `
foo:
    zoo:
`,
			expected: `
foo:
    zoo:
`,
		},
		{
			name: "list to map",
			dst: `
foo:
    zoo:
`,
			src: `
foo:
    - blah
`,
			expected: "error: invalid node kinds",
		},
		{
			name: "set list to null scalar",
			dst: `
foo:
`,
			src: `
foo:
    - x
    - y
    - z
`,
			expected: `
foo:
    - x
    - y
    - z
`,
		},
		{
			name: "erase list",
			dst: `
foo:
    - x
    - y
    - z
`,
			src: `
foo:
`,
			expected: `
foo:
`,
		},
		{
			name: "set scalar",
			dst: `
foo:
`,
			src: `
foo: 1
`,
			expected: `
foo: 1
`,
		},
		{
			name: "erase scalar",
			dst: `
foo: 1
`,
			src: `
foo:
`,
			expected: `
foo:
`,
		},
		{
			name: "null nodes",
			dst: `
a: 1
b:
  - a
foo:
  a: 1
c: null
d: null
e: null
f: null
g: null
`,
			src: `
a: null
b: null
foo: null
c: null
d: 1
e:
f:
  a: 1
g:
- a
`,
			expected: `
a: null
b: null
foo: null
c: null
d: 1
e:
f:
  a: 1
g:
- a
`,
		},
		// demonstrate deep maps merge and that dest ordering is preserved. seq is replaced as by defaults.
		{
			src: `
a:
  z: 1
  c: "c"
  map:
    seq:
      - 2
  map2:
    foo: bar
  map3:
    foo: bar
c: 1
b:
  c: 2
  d: 4
`,
			dst: `
a:
  b: 3.3
  c: null
  map:
    seq:
      - omg
      - 1
  map2:
    a: 1
b:
  a: 2
  c: 3
`,
			expected: `
a:
  b: 3.3
  c: "c"
  map:
    seq:
      - 2
  map2:
    a: 1
    foo: bar
  z: 1
  map3:
    foo: bar
b:
  a: 2
  c: 2
  d: 4
c: 1
`,
		},
		// demonstrate deep maps merge together with seq append.
		{
			seqAppend: true,
			src: `
a:
  z: 1
  c: "c"
  map:
    seq:
      - 2
      - map2:
          bar: foo
`,
			dst: `
a:
  b: 3.3
  c: null
  map:
    seq:
      - omg
      - 1
      - map2:
          foo: bar
`,
			expected: `
a:
  b: 3.3
  c: "c"
  map:
    seq:
      - omg
      - 1
      - map2:
          foo: bar
      - 2
      - map2:
          bar: foo
  z: 1
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dst, src, expected Node

			assert.NoError(t, Unmarshal([]byte(tt.dst), &dst))
			assert.NoError(t, Unmarshal([]byte(tt.src), &src))
			assert.NoError(t, Unmarshal([]byte(tt.expected), &expected))

			// fmt.Println(">>", (*DebugYamlNode)(&dst).String())
			// fmt.Println(">>", (*DebugYamlNode)(&src).String())

			err := dst.Merge(&src, WithAppendSeq(tt.seqAppend))

			// fmt.Println(">>", (*DebugYamlNode)(&dst).String())

			if strings.HasPrefix(tt.expected, "error: ") {
				assert.ErrorMatches(t, tt.expected[len("error: "):], err)
				return
			} else {
				assert.NoError(t, err)
			}

			dstBytes, err := Dump(&dst)
			assert.NoError(t, err)

			expectedBytes, err := Dump(&expected)
			assert.NoError(t, err)

			assert.Equal(t, string(expectedBytes), string(dstBytes))
		})
	}
}

// withFromLegacy is a private option that indicates this call is from
// a legacy API (Unmarshal/Decoder). It enables Unmarshaler interface
// checking and allows trailing content for backward compatibility.
func withFromLegacy() Option {
	return func(o *Options) error {
		o.FromLegacy = true
		return nil
	}
}

func WithV3Defaults() Option {
	return CombineOptions(
		WithIndent(4),
		WithCompactSeqIndent(false),
		WithLineWidth(80),
		WithUnicode(true),
		WithUniqueKeys(true),
		WithQuotePreference(QuoteLegacy),
		// WithPlugin(limit.New()),
	)
}

func Unmarshal(in []byte, out any) (err error) {
	if err := Load(in, out, WithV3Defaults(), withFromLegacy()); err != nil {
		// Simulate v3.0.4 implementation for empty input when: '' -> empty Node, kind == 0.
		if !strings.Contains(err.Error(), "no documents in stream") {
			return err
		}
	}
	return nil
}

type DebugYamlNode Node

func (p *DebugYamlNode) String() string {
	if p == nil {
		return ""
	}

	var buf bytes.Buffer

	_, _ = fmt.Fprintf(&buf, "%p %+v\n", p, *p)

	for _, n := range p.Content {
		_, _ = fmt.Fprint(&buf, (*DebugYamlNode)(n).String())
	}

	return buf.String()
}
