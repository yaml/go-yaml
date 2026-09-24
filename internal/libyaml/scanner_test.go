// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for the scanner stage.
// Verifies input stream to token stream transformation, indentation handling,
// and simple keys.

package libyaml

import (
	"bytes"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestScanner(t *testing.T) {
	RunTestCases(t, "scanner.yaml", map[string]TestHandler{
		"scan-tokens":          runScanTokensTest,
		"scan-tokens-detailed": runScanTokensDetailedTest,
		"scan-error":           runScanErrorTest,
		"char-predicate":       runCharPredicateTest,
		"char-convert":         runCharConvertTest,
	})
}

// TestTrailingUTF8LeadByte ensures a truncated multi-byte UTF-8 sequence at EOF
// reports a reader error and does not panic.
func TestTrailingUTF8LeadByte(t *testing.T) {
	parser := NewParser()
	parser.SetInputString([]byte{0xEF, 0xBB}) // Incomplete 3-byte UTF-8 sequence (missing third byte)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Scan panicked: %v", r)
		}
	}()

	var token Token
	err := parser.Scan(&token)
	assert.ErrorMatchesf(t, "incomplete UTF-8 octet sequence", err, "trailing UTF-8 lead byte must fail cleanly")
}

// TestPredicateMissingLookahead verifies that predicates handle truncated UTF-8
// safely when called without scanner buffer lookahead guarantees.
func TestPredicateMissingLookahead(t *testing.T) {
	// Helper to check that a function doesn't panic
	notPanic := func(name string, f func()) {
		t.Helper()
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("%s panicked: %v", name, r)
			}
		}()
		f()
	}

	// isPrintable should not panic on truncated UTF-8 sequences
	notPanic("isPrintable with 0xF0", func() {
		_ = isPrintable([]byte{0xF0}, 0)
	})

	// Test other predicates with truncated sequences
	notPanic("isLineBreak with 0xC2", func() {
		_ = isLineBreak([]byte{0xC2}, 0)
	})

	notPanic("isBOM with truncated", func() {
		_ = isBOM([]byte{0xEF, 0xBB}, 0)
	})

	notPanic("isEndOfScalarInFlowContentChar", func() {
		_ = isEndOfScalarInFlowContentChar([]byte{':'}, 0)
	})
}

// runScanTokensTest tests the scanTokens function.
//
//nolint:thelper // because this function is the real test
func runScanTokensTest(t *testing.T, tc TestCase) {
	types, ok := scanTokens(tc.Yaml)
	assert.Truef(t, ok, "scanTokens() failed")

	// Convert Want from interface{} to []string
	wantSlice, ok := tc.Want.([]any)
	assert.Truef(t, ok, "Want should be []interface{}")

	var expected []TokenType
	for _, item := range wantSlice {
		tokenStr, ok := item.(string)
		assert.Truef(t, ok, "Want item should be string")
		expected = append(expected, ParseTokenType(t, tokenStr))
	}

	assert.Equalf(t, len(expected), len(types), "scanTokens() got %d tokens, want %d", len(types), len(expected))
	for i, tt := range expected {
		assert.Equalf(t, tt, types[i], "token[%d] = %v, want %v", i, types[i], tt)
	}
}

// runScanTokensDetailedTest tests the scanTokensDetailed function.
//
//nolint:thelper // because this function is the real test
func runScanTokensDetailedTest(t *testing.T, tc TestCase) {
	tokens, ok := scanTokensDetailed(tc.Yaml)
	assert.Truef(t, ok, "scanTokensDetailed() failed")

	assert.Equalf(t, len(tc.WantTokens), len(tokens), "scanTokensDetailed() got %d tokens, want %d", len(tokens), len(tc.WantTokens))

	for i, wantSpec := range tc.WantTokens {
		token := tokens[i]
		wantType := ParseTokenType(t, wantSpec.Type)

		assert.Equalf(t, wantType, token.Type, "token[%d].Type = %v, want %v", i, token.Type, wantType)

		// Check specific token properties if specified
		if wantSpec.Value != "" {
			assert.Truef(t, bytes.Equal(token.Value, []byte(wantSpec.Value)),
				"token[%d].Value = %q, want %q", i, token.Value, wantSpec.Value)
		}

		if wantSpec.Style != "" {
			wantStyle := ParseScalarStyle(t, wantSpec.Style)
			assert.Equalf(t, wantStyle, token.Style, "token[%d].Style = %v, want %v", i, token.Style, wantStyle)
		}
	}
}

// runScanErrorTest tests the scanner error handling.
//
//nolint:thelper // because this function is the real test
func runScanErrorTest(t *testing.T, tc TestCase) {
	parser := NewParser()
	parser.SetInputString([]byte(tc.Yaml))

	var token Token
	var scanErr error
	for scanErr == nil && token.Type != STREAM_END_TOKEN {
		scanErr = parser.Scan(&token)
	}

	// Convert Want from interface{} to check for error
	// Want can be either a boolean or a single-element sequence
	// Defaults to true if not specified
	wantError := true
	if tc.Want != nil {
		switch v := tc.Want.(type) {
		case bool:
			wantError = v
		case []any:
			if len(v) > 0 {
				if boolVal, ok := v[0].(bool); ok {
					wantError = boolVal
				}
			}
		}
	}
	if wantError {
		assert.Truef(t, scanErr != nil, "Expected scanner error, but got none")
		// Check error message against regex pattern if provided
		if tc.Like != "" {
			assert.ErrorMatchesf(t, tc.Like, scanErr, "")
		}
	} else {
		assert.Truef(t, scanErr == nil, "Expected no scanner error, but got %v", scanErr)
	}
}

// Character classification and conversion tests
// These tests are now part of scanner.yaml and run via TestScanner

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
	case "isTagURIChar":
		// Default verbatim to false if not specified
		verbatim := false
		if len(tc.Args) >= 1 {
			v, ok := tc.Args[0].(bool)
			assert.Truef(t, ok, "Args[0] should be bool, got %T", tc.Args[0])
			verbatim = v
		}
		got = isTagURIChar(input, index, verbatim)
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
