// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestYAMLPrivate(t *testing.T) {
	RunTestCases(t, "yamlprivate.yaml", map[string]TestHandler{
		"char-predicate": runCharPredicateTest,
		"char-convert":   runCharConvertTest,
	})
}

func runCharPredicateTest(t *testing.T, tc TestCase) {
	t.Helper()

	input := tc.Input
	index := tc.Index

	// Default want to true if not specified
	want := WantBool(t, tc.Want, true)

	var got bool
	switch tc.Function {
	case "isAlpha":
		got = isAlpha(input, index)
	case "isFlowIndicator":
		got = isFlowIndicator(input, index)
	case "isAnchorChar":
		got = isAnchorChar(input, index)
	case "isColon":
		got = isColon(input, index)
	case "isDigit":
		got = isDigit(input, index)
	case "isHex":
		got = isHex(input, index)
	case "isASCII":
		got = isASCII(input, index)
	case "isPrintable":
		got = isPrintable(input, index)
	case "isZeroChar":
		got = isZeroChar(input, index)
	case "isBOM":
		got = isBOM(input, index)
	case "isSpace":
		got = isSpace(input, index)
	case "isTab":
		got = isTab(input, index)
	case "isBlank":
		got = isBlank(input, index)
	case "isLineBreak":
		got = isLineBreak(input, index)
	case "isCRLF":
		got = isCRLF(input, index)
	case "isBreakOrZero":
		got = isBreakOrZero(input, index)
	case "isSpaceOrZero":
		got = isSpaceOrZero(input, index)
	case "isBlankOrZero":
		got = isBlankOrZero(input, index)
	default:
		t.Fatalf("unknown function: %s", tc.Function)
	}

	assert.Equalf(t, want, got, "%s(%q, %d) = %v, want %v", tc.Function, input, index, got, want)
}

func runCharConvertTest(t *testing.T, tc TestCase) {
	t.Helper()

	input := tc.Input
	index := tc.Index
	want, ok := tc.Want.(int)
	assert.Truef(t, ok, "Want should be int, got %T", tc.Want)

	var got int
	switch tc.Function {
	case "asDigit":
		got = asDigit(input, index)
	case "asHex":
		got = asHex(input, index)
	case "width":
		// width takes a single byte, not a byte array and index
		got = width(input[index])
	default:
		t.Fatalf("unknown function: %s", tc.Function)
	}

	assert.Equalf(t, want, got, "%s(%q, %d) = %d, want %d", tc.Function, input, index, got, want)
}
