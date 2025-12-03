// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func scanTokens(input string) ([]TokenType, bool) {
	parser := NewParser()
	parser.SetInputString([]byte(input))

	var types []TokenType
	for {
		var token Token
		if !parser.Scan(&token) {
			if parser.ErrorType != NO_ERROR {
				return nil, false
			}
			return types, true
		}
		types = append(types, token.Type)
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}
	return types, true
}

func TestScannerSimpleScalar(t *testing.T) {
	input := "hello"
	types, ok := scanTokens(input)
	assert.Truef(t, ok, "scanTokens() failed")

	expected := []TokenType{
		STREAM_START_TOKEN,
		SCALAR_TOKEN,
		STREAM_END_TOKEN,
	}

	assert.Equalf(t, len(expected), len(types), "scanTokens() got %d tokens, want %d", len(types), len(expected))
	for i, tt := range expected {
		assert.Equalf(t, tt, types[i], "token[%d] = %v, want %v", i, types[i], tt)
	}
}

func TestScannerSimpleMapping(t *testing.T) {
	input := "key: value"
	types, ok := scanTokens(input)
	assert.Truef(t, ok, "scanTokens() failed")

	expected := []TokenType{
		STREAM_START_TOKEN,
		BLOCK_MAPPING_START_TOKEN,
		KEY_TOKEN,
		SCALAR_TOKEN,
		VALUE_TOKEN,
		SCALAR_TOKEN,
		BLOCK_END_TOKEN,
		STREAM_END_TOKEN,
	}

	assert.Equalf(t, len(expected), len(types), "scanTokens() got %d tokens, want %d", len(types), len(expected))
	for i, tt := range expected {
		assert.Equalf(t, tt, types[i], "token[%d] = %v, want %v", i, types[i], tt)
	}
}

func TestScannerBlockSequence(t *testing.T) {
	input := "- item1\n- item2"
	types, ok := scanTokens(input)
	assert.Truef(t, ok, "scanTokens() failed")

	expected := []TokenType{
		STREAM_START_TOKEN,
		BLOCK_SEQUENCE_START_TOKEN,
		BLOCK_ENTRY_TOKEN,
		SCALAR_TOKEN,
		BLOCK_ENTRY_TOKEN,
		SCALAR_TOKEN,
		BLOCK_END_TOKEN,
		STREAM_END_TOKEN,
	}

	assert.Equalf(t, len(expected), len(types), "scanTokens() got %d tokens, want %d", len(types), len(expected))
	for i, tt := range expected {
		assert.Equalf(t, tt, types[i], "token[%d] = %v, want %v", i, types[i], tt)
	}
}

func TestScannerFlowSequence(t *testing.T) {
	input := "[1, 2, 3]"
	types, ok := scanTokens(input)
	assert.Truef(t, ok, "scanTokens() failed")

	expected := []TokenType{
		STREAM_START_TOKEN,
		FLOW_SEQUENCE_START_TOKEN,
		SCALAR_TOKEN,
		FLOW_ENTRY_TOKEN,
		SCALAR_TOKEN,
		FLOW_ENTRY_TOKEN,
		SCALAR_TOKEN,
		FLOW_SEQUENCE_END_TOKEN,
		STREAM_END_TOKEN,
	}

	assert.Equalf(t, len(expected), len(types), "scanTokens() got %d tokens, want %d", len(types), len(expected))
	for i, tt := range expected {
		assert.Equalf(t, tt, types[i], "token[%d] = %v, want %v", i, types[i], tt)
	}
}

func TestScannerFlowMapping(t *testing.T) {
	input := "{a: 1, b: 2}"
	types, ok := scanTokens(input)
	assert.Truef(t, ok, "scanTokens() failed")

	expected := []TokenType{
		STREAM_START_TOKEN,
		FLOW_MAPPING_START_TOKEN,
		KEY_TOKEN,
		SCALAR_TOKEN,
		VALUE_TOKEN,
		SCALAR_TOKEN,
		FLOW_ENTRY_TOKEN,
		KEY_TOKEN,
		SCALAR_TOKEN,
		VALUE_TOKEN,
		SCALAR_TOKEN,
		FLOW_MAPPING_END_TOKEN,
		STREAM_END_TOKEN,
	}

	assert.Equalf(t, len(expected), len(types), "scanTokens() got %d tokens, want %d", len(types), len(expected))
	for i, tt := range expected {
		assert.Equalf(t, tt, types[i], "token[%d] = %v, want %v", i, types[i], tt)
	}
}

func TestScannerDocumentMarkers(t *testing.T) {
	input := "---\nkey: value\n..."
	types, ok := scanTokens(input)
	assert.Truef(t, ok, "scanTokens() failed")

	assert.Equalf(t, STREAM_START_TOKEN, types[0], "token[0] = %v, want STREAM_START_TOKEN", types[0])
	assert.Equalf(t, DOCUMENT_START_TOKEN, types[1], "token[1] = %v, want DOCUMENT_START_TOKEN", types[1])

	hasDocumentEnd := false
	for _, tt := range types {
		if tt == DOCUMENT_END_TOKEN {
			hasDocumentEnd = true
			break
		}
	}
	assert.Truef(t, hasDocumentEnd, "Expected DOCUMENT_END_TOKEN not found")
}

func TestScannerAnchorAndAlias(t *testing.T) {
	input := "- &anchor value\n- *anchor"
	types, ok := scanTokens(input)
	assert.Truef(t, ok, "scanTokens() failed")

	hasAnchor := false
	hasAlias := false
	for _, tt := range types {
		if tt == ANCHOR_TOKEN {
			hasAnchor = true
		}
		if tt == ALIAS_TOKEN {
			hasAlias = true
		}
	}

	assert.Truef(t, hasAnchor, "Expected ANCHOR_TOKEN not found")
	assert.Truef(t, hasAlias, "Expected ALIAS_TOKEN not found")
}

func TestScannerTag(t *testing.T) {
	input := "!!str value"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	hasTag := false
	for {
		var token Token
		if !parser.Scan(&token) {
			break
		}
		if token.Type == TAG_TOKEN {
			hasTag = true
			assert.DeepEqualf(t, []byte("!!"), token.Value, "TAG_TOKEN Value = %q, want \"!!\"", token.Value)
			assert.DeepEqualf(t, []byte("str"), token.suffix, "TAG_TOKEN suffix = %q, want \"str\"", token.suffix)
		}
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}

	assert.Truef(t, hasTag, "Expected TAG_TOKEN not found")
}

func TestScannerSingleQuotedScalar(t *testing.T) {
	input := "'single quoted'"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	foundScalar := false
	for {
		var token Token
		if !parser.Scan(&token) {
			break
		}
		if token.Type == SCALAR_TOKEN {
			foundScalar = true
			assert.Equalf(t, SINGLE_QUOTED_SCALAR_STYLE, token.Style, "SCALAR_TOKEN Style = %v, want SINGLE_QUOTED_SCALAR_STYLE", token.Style)
			assert.DeepEqualf(t, []byte("single quoted"), token.Value, "SCALAR_TOKEN Value = %q, want \"single quoted\"", token.Value)
		}
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}

	assert.Truef(t, foundScalar, "Expected SCALAR_TOKEN not found")
}

func TestScannerDoubleQuotedScalar(t *testing.T) {
	input := "\"double quoted\""
	parser := NewParser()
	parser.SetInputString([]byte(input))

	foundScalar := false
	for {
		var token Token
		if !parser.Scan(&token) {
			break
		}
		if token.Type == SCALAR_TOKEN {
			foundScalar = true
			assert.Equalf(t, DOUBLE_QUOTED_SCALAR_STYLE, token.Style, "SCALAR_TOKEN Style = %v, want DOUBLE_QUOTED_SCALAR_STYLE", token.Style)
			assert.DeepEqualf(t, []byte("double quoted"), token.Value, "SCALAR_TOKEN Value = %q, want \"double quoted\"", token.Value)
		}
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}

	assert.Truef(t, foundScalar, "Expected SCALAR_TOKEN not found")
}

func TestScannerLiteralScalar(t *testing.T) {
	input := "key: |\n  literal\n  scalar"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	foundLiteral := false
	for {
		var token Token
		if !parser.Scan(&token) {
			break
		}
		if token.Type == SCALAR_TOKEN && token.Style == LITERAL_SCALAR_STYLE {
			foundLiteral = true
		}
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}

	assert.Truef(t, foundLiteral, "Expected LITERAL_SCALAR_STYLE not found")
}

func TestScannerFoldedScalar(t *testing.T) {
	input := "key: >\n  folded\n  scalar"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	foundFolded := false
	for {
		var token Token
		if !parser.Scan(&token) {
			break
		}
		if token.Type == SCALAR_TOKEN && token.Style == FOLDED_SCALAR_STYLE {
			foundFolded = true
		}
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}

	assert.Truef(t, foundFolded, "Expected FOLDED_SCALAR_STYLE not found")
}

func TestScannerVersionDirective(t *testing.T) {
	input := "%YAML 1.2\n---"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	foundVersionDirective := false
	for {
		var token Token
		if !parser.Scan(&token) {
			break
		}
		if token.Type == VERSION_DIRECTIVE_TOKEN {
			foundVersionDirective = true
			assert.Equalf(t, 1, int(token.major), "VERSION_DIRECTIVE major = %d, want 1", token.major)
			assert.Equalf(t, 2, int(token.minor), "VERSION_DIRECTIVE minor = %d, want 2", token.minor)
		}
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}

	assert.Truef(t, foundVersionDirective, "Expected VERSION_DIRECTIVE_TOKEN not found")
}

func TestScannerTagDirective(t *testing.T) {
	input := "%TAG !yaml! tag:yaml.org,2002:\n---"
	parser := NewParser()
	parser.SetInputString([]byte(input))

	foundTagDirective := false
	for {
		var token Token
		if !parser.Scan(&token) {
			break
		}
		if token.Type == TAG_DIRECTIVE_TOKEN {
			foundTagDirective = true
			assert.DeepEqualf(t, []byte("!yaml!"), token.Value, "TAG_DIRECTIVE handle = %q, want \"!yaml!\"", token.Value)
		}
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}

	assert.Truef(t, foundTagDirective, "Expected TAG_DIRECTIVE_TOKEN not found")
}

func TestScannerEmptyInput(t *testing.T) {
	input := ""
	types, ok := scanTokens(input)
	assert.Truef(t, ok, "scanTokens() failed")

	assert.Equalf(t, 2, len(types), "scanTokens() got %d tokens, want 2", len(types))
	assert.Equalf(t, STREAM_START_TOKEN, types[0], "token[0] = %v, want STREAM_START_TOKEN", types[0])
	assert.Equalf(t, STREAM_END_TOKEN, types[1], "token[1] = %v, want STREAM_END_TOKEN", types[1])
}

func TestScannerInvalidCharacter(t *testing.T) {
	input := string([]byte{0x01})
	parser := NewParser()
	parser.SetInputString([]byte(input))

	gotError := false
	for {
		var token Token
		if !parser.Scan(&token) {
			if parser.ErrorType == SCANNER_ERROR || parser.ErrorType == READER_ERROR {
				gotError = true
			}
			break
		}
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}

	assert.Truef(t, gotError, "Expected scanner error for invalid character")
}

func TestScannerComplexNesting(t *testing.T) {
	input := `
parent:
  - item1
  - item2:
      nested: value
`
	types, ok := scanTokens(input)
	assert.Truef(t, ok, "scanTokens() failed")

	assert.Equalf(t, STREAM_START_TOKEN, types[0], "First token should be STREAM_START_TOKEN, got %v", types[0])

	hasBlockMapping := false
	hasBlockSequence := false
	hasBlockEnd := false

	for _, tt := range types {
		switch tt {
		case BLOCK_MAPPING_START_TOKEN:
			hasBlockMapping = true
		case BLOCK_SEQUENCE_START_TOKEN:
			hasBlockSequence = true
		case BLOCK_END_TOKEN:
			hasBlockEnd = true
		}
	}

	assert.Truef(t, hasBlockMapping, "Expected BLOCK_MAPPING_START_TOKEN not found")
	assert.Truef(t, hasBlockSequence, "Expected BLOCK_SEQUENCE_START_TOKEN not found")
	assert.Truef(t, hasBlockEnd, "Expected BLOCK_END_TOKEN not found")
}

func TestScannerEscapeSequences(t *testing.T) {
	input := "\"\\n\\t\\r\\\\\\\"\\x41\\u0042\""
	parser := NewParser()
	parser.SetInputString([]byte(input))

	found := false
	for {
		var token Token
		if !parser.Scan(&token) {
			break
		}
		if token.Type == SCALAR_TOKEN {
			expected := []byte("\n\t\r\\\"AB")
			assert.DeepEqualf(t, expected, token.Value, "Escape sequences: got %q, want %q", token.Value, expected)
			found = true
			return
		}
		if token.Type == STREAM_END_TOKEN {
			break
		}
	}

	assert.Truef(t, found, "Expected SCALAR_TOKEN with escape sequences not found")
}
