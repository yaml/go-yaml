// SPDX-License-Identifier: Apache-2.0

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
	})
}

func runScanTokensTest(t *testing.T, tc TestCase) {
	types, ok := scanTokens(tc.Yaml)
	assert.Truef(t, ok, "scanTokens() failed")

	// Convert Want from interface{} to []string
	wantSlice, ok := tc.Want.([]interface{})
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

func runScanErrorTest(t *testing.T, tc TestCase) {
	parser := NewParser()
	parser.SetInputString([]byte(tc.Yaml))

	var token Token
	for parser.Scan(&token) && token.Type != STREAM_END_TOKEN {
	}

	// Convert Want from interface{} to check for error
	// Want can be either a boolean or a single-element sequence
	// Defaults to true if not specified
	wantError := true
	if tc.Want != nil {
		switch v := tc.Want.(type) {
		case bool:
			wantError = v
		case []interface{}:
			if len(v) > 0 {
				if boolVal, ok := v[0].(bool); ok {
					wantError = boolVal
				}
			}
		}
	}
	if wantError {
		assert.Truef(t, parser.ErrorType != NO_ERROR, "Expected scanner error, but got none")
	} else {
		assert.Truef(t, parser.ErrorType == NO_ERROR, "Expected no scanner error, but got %v", parser.ErrorType)
	}
}
