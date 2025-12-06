// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestReader(t *testing.T) {
	RunTestCases(t, "reader_test.yaml", map[string]TestHandler{
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
	problem := tc.Args[0].(string)
	offset := tc.Args[1].(int)
	value := tc.Args[2].(int)

	result := parser.setReaderError(problem, offset, value)

	// Check return value
	want := tc.Want.(bool)
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
	want := WantBool(tc.Want, true)
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
	want := WantBool(tc.Want, true)
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
	length := tc.Args[0].(int)

	result := parser.updateBuffer(length)

	// Check return value (defaults to true)
	want := WantBool(tc.Want, true)
	assert.Equalf(t, want, result, "updateBuffer(%d) = %v, want %v", length, result, want)

	// Run field checks
	runFieldChecks(t, &parser, tc.Checks)
}

func runReaderPanicTest(t *testing.T, tc TestCase) {
	t.Helper()

	parser := NewParser()

	wantMsg := tc.Want.(string)

	assert.PanicMatchesf(t, wantMsg, func() {
		switch tc.Function {
		case "updateBuffer":
			length := tc.Args[0].(int)
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
			parser.eof = value.(bool)
		default:
			t.Fatalf("unknown setup key: %s", key)
		}
	}
}
