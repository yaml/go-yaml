// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestReader(t *testing.T) {
	RunTestCases(t, "reader.yaml", map[string]TestHandler{
		"reader-set-error":          runReaderSetErrorTest,
		"reader-determine-encoding": runReaderDetermineEncodingTest,
		"reader-update-raw-buffer":  runReaderUpdateRawBufferTest,
		"reader-update-buffer":      runReaderUpdateBufferTest,
		"reader-panic":              runReaderPanicTest,
	})
}

func runReaderSetErrorTest(t *testing.T, tc TestCase) {
	t.Helper()

	parser := NewParser()

	// args should be [problem, offset, value]
	assert.Truef(t, len(tc.Args) >= 3, "Args should have at least 3 elements, got %d", len(tc.Args))
	problem, ok := tc.Args[0].(string)
	assert.Truef(t, ok, "Args[0] should be string, got %T", tc.Args[0])
	offset, ok := tc.Args[1].(int)
	assert.Truef(t, ok, "Args[1] should be int, got %T", tc.Args[1])
	value, ok := tc.Args[2].(int)
	assert.Truef(t, ok, "Args[2] should be int, got %T", tc.Args[2])

	result := parser.setReaderError(problem, offset, value)

	// Check return value
	want, ok := tc.Want.(bool)
	assert.Truef(t, ok, "Want should be bool, got %T", tc.Want)
	assert.Equalf(t, want, result, "setReaderError() = %v, want %v", result, want)

	// Run field checks
	runFieldChecks(t, &parser, tc.Checks)
}

func runReaderDetermineEncodingTest(t *testing.T, tc TestCase) {
	t.Helper()

	parser := NewParser()
	parser.SetInputString(tc.Input)

	result := parser.determineEncoding()

	// Check return value (defaults to true)
	want := WantBool(t, tc.Want, true)
	assert.Equalf(t, want, result, "determineEncoding() = %v, want %v", result, want)

	// Run field checks
	runFieldChecks(t, &parser, tc.Checks)
}

func runReaderUpdateRawBufferTest(t *testing.T, tc TestCase) {
	t.Helper()

	parser := NewParser()
	parser.SetInputString(tc.Input)

	// Apply any setup
	if tc.Setup != nil {
		applySetup(t, &parser, tc.Setup)
	}

	result := parser.updateRawBuffer()

	// Check return value (defaults to true)
	want := WantBool(t, tc.Want, true)
	assert.Equalf(t, want, result, "updateRawBuffer() = %v, want %v", result, want)

	// Run field checks
	runFieldChecks(t, &parser, tc.Checks)
}

func runReaderUpdateBufferTest(t *testing.T, tc TestCase) {
	t.Helper()

	parser := NewParser()
	parser.SetInputString(tc.Input)

	// Apply any setup
	if tc.Setup != nil {
		applySetup(t, &parser, tc.Setup)
	}

	// Get the length argument
	assert.Truef(t, len(tc.Args) >= 1, "Args should have at least 1 element, got %d", len(tc.Args))
	length, ok := tc.Args[0].(int)
	assert.Truef(t, ok, "Args[0] should be int, got %T", tc.Args[0])

	result := parser.updateBuffer(length)

	// Check return value (defaults to true)
	want := WantBool(t, tc.Want, true)
	assert.Equalf(t, want, result, "updateBuffer(%d) = %v, want %v", length, result, want)

	// Run field checks
	runFieldChecks(t, &parser, tc.Checks)
}

func runReaderPanicTest(t *testing.T, tc TestCase) {
	t.Helper()

	parser := NewParser()

	wantMsg, ok := tc.Want.(string)
	assert.Truef(t, ok, "Want should be string, got %T", tc.Want)

	assert.PanicMatchesf(t, wantMsg, func() {
		switch tc.Function {
		case "updateBuffer":
			assert.Truef(t, len(tc.Args) >= 1, "Args should have at least 1 element, got %d", len(tc.Args))
			length, ok := tc.Args[0].(int)
			assert.Truef(t, ok, "Args[0] should be int, got %T", tc.Args[0])
			parser.updateBuffer(length)
		default:
			t.Fatalf("unknown function: %s", tc.Function)
		}
	}, "Expected panic: %s", wantMsg)
}

func applySetup(t *testing.T, parser *Parser, setup interface{}) {
	t.Helper()

	if setup == nil {
		return
	}

	setupMap, ok := setup.(map[string]interface{})
	if !ok {
		t.Fatalf("setup must be a map, got %T", setup)
	}

	for key, value := range setupMap {
		switch key {
		case "eof":
			boolVal, ok := value.(bool)
			assert.Truef(t, ok, "setup.eof should be bool, got %T", value)
			parser.eof = boolVal
		default:
			t.Fatalf("unknown setup key: %s", key)
		}
	}
}
