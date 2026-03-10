// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for error types.
// Verifies error formatting, unwrapping, and error matching.

package libyaml

import (
	"errors"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestErrors(t *testing.T) {
	RunTestCases(t, "errors.yaml", map[string]TestHandler{
		"load-error":     runLoadErrorTest,
		"emitter-error":  runEmitterYAMLErrorTest,
		"writer-error":   runWriterYAMLErrorTest,
		"load-errors":    runLoadErrorsTest,
		"load-errors-as": runLoadErrorsAsTest,
		"load-errors-is": runLoadErrorsIsTest,
		"type-error":     runTypeYAMLErrorTest,
	})
}

func runLoadErrorTest(t *testing.T, tc TestCase) {
	t.Helper()

	// Extract error spec from 'from' field
	errorSpec, ok := tc.From.(map[string]any)
	assert.Truef(t, ok, "from should be map[string]any, got %T", tc.From)

	err := buildLoadError(t, errorSpec)
	got := err.Error()
	want, ok := tc.Want.(string)
	assert.Truef(t, ok, "want should be string, got %T", tc.Want)

	assert.Equalf(t, want, got, "error message mismatch")

	// Verify Stage field if specified
	if stageStr, ok := errorSpec["stage"].(string); ok {
		assert.Equalf(t, Stage(stageStr), err.Stage, "Stage mismatch")
	}

	// Test Unwrap if specified
	if tc.Also == "unwrap" {
		unwrapped := err.Unwrap()
		if err.err != nil {
			assert.NotNilf(t, unwrapped, "Unwrap() should return non-nil when Err is set")
			assert.Equalf(t, err.err.Error(), unwrapped.Error(), "Unwrap() error message mismatch")
		} else {
			if unwrapped != nil {
				t.Fatalf("Unwrap() should return nil when Err is not set, got %v", unwrapped)
			}
		}
	}
}

func runEmitterYAMLErrorTest(t *testing.T, tc TestCase) {
	t.Helper()

	errorSpec, ok := tc.From.(map[string]any)
	assert.Truef(t, ok, "from should be map[string]any, got %T", tc.From)

	message := getString(t, errorSpec, "message")
	err := EmitterError{Message: message}

	got := err.Error()
	want, ok := tc.Want.(string)
	assert.Truef(t, ok, "want should be string, got %T", tc.Want)
	assert.Equalf(t, want, got, "error message mismatch")
}

func runWriterYAMLErrorTest(t *testing.T, tc TestCase) {
	t.Helper()

	errorSpec, ok := tc.From.(map[string]any)
	assert.Truef(t, ok, "from should be map[string]any, got %T", tc.From)

	message := getString(t, errorSpec, "message")
	err := WriterError{Err: errors.New(message)}

	got := err.Error()
	want, ok := tc.Want.(string)
	assert.Truef(t, ok, "want should be string, got %T", tc.Want)
	assert.Equalf(t, want, got, "error message mismatch")

	// Test Unwrap if specified
	if tc.Also == "unwrap" {
		unwrapped := err.Unwrap()
		assert.NotNilf(t, unwrapped, "Unwrap() should return non-nil")
		assert.Equalf(t, message, unwrapped.Error(), "Unwrap() error message mismatch")
	}
}

func runLoadErrorsTest(t *testing.T, tc TestCase) {
	t.Helper()

	errorSpec, ok := tc.From.(map[string]any)
	assert.Truef(t, ok, "from should be map[string]any, got %T", tc.From)

	errList := buildLoadErrorList(t, errorSpec)
	err := &LoadErrors{Errors: errList}

	got := err.Error()
	want, ok := tc.Want.(string)
	assert.Truef(t, ok, "want should be string, got %T", tc.Want)

	// Normalize line endings for comparison
	gotNorm := strings.TrimSpace(got)
	wantNorm := strings.TrimSpace(want)

	assert.Equalf(t, wantNorm, gotNorm, "error message mismatch")
}

func runLoadErrorsAsTest(t *testing.T, tc TestCase) {
	t.Helper()

	errorSpec, ok := tc.From.(map[string]any)
	assert.Truef(t, ok, "from should be map[string]any, got %T", tc.From)

	errList := buildLoadErrorList(t, errorSpec)
	err := &LoadErrors{Errors: errList}

	switch tc.As {
	case "TypeError":
		var target *TypeError
		gotAs := errors.As(err, &target)
		assert.Equalf(t, tc.WantAs, gotAs, "errors.As result mismatch")

		if tc.WantAs && target != nil {
			assert.Equalf(t, len(tc.WantMessages), len(target.Errors), "TypeError.Errors length mismatch")
			for i, wantMsg := range tc.WantMessages {
				wantStr, ok := wantMsg.(string)
				assert.Truef(t, ok, "want_messages[%d] should be string, got %T", i, wantMsg)
				assert.Equalf(t, wantStr, target.Errors[i], "TypeError.Errors[%d] mismatch", i)
			}
		}

	default:
		t.Fatalf("unknown as type: %s", tc.As)
	}
}

func runLoadErrorsIsTest(t *testing.T, tc TestCase) {
	t.Helper()

	errorSpec, ok := tc.From.(map[string]any)
	assert.Truef(t, ok, "from should be map[string]any, got %T", tc.From)

	errList := buildLoadErrorList(t, errorSpec)
	err := &LoadErrors{Errors: errList}

	// Check if any of the wrapped errors contains the target message
	gotIs := false
	for _, cerr := range err.Errors {
		if cerr.err != nil && cerr.err.Error() == tc.Is {
			gotIs = true
			break
		}
	}

	assert.Equalf(t, tc.WantIs, gotIs, "errors.Is result mismatch")
}

func runTypeYAMLErrorTest(t *testing.T, tc TestCase) {
	t.Helper()

	errorSpec, ok := tc.From.(map[string]any)
	assert.Truef(t, ok, "from should be map[string]any, got %T", tc.From)

	errorMsgs := getStringSlice(t, errorSpec, "errors")
	err := &TypeError{Errors: errorMsgs}

	got := err.Error()
	want, ok := tc.Want.(string)
	assert.Truef(t, ok, "want should be string, got %T", tc.Want)

	// Normalize line endings for comparison
	gotNorm := strings.TrimSpace(got)
	wantNorm := strings.TrimSpace(want)

	assert.Equalf(t, wantNorm, gotNorm, "error message mismatch")
}

// Helper functions

func buildLoadError(t *testing.T, spec map[string]any) *LoadError {
	t.Helper()

	err := &LoadError{
		Stage:   Stage(getString(t, spec, "stage")),
		Mark:    buildMark(t, spec, "mark"),
		Message: getString(t, spec, "message"),
	}

	// Add context if specified
	if contextMsg, ok := spec["context_message"].(string); ok {
		err.ContextMsg = contextMsg
		err.ContextMark = buildMark(t, spec, "context_mark")
	}

	// Add underlying error if specified
	if errMsg, ok := spec["err"].(string); ok {
		err.err = errors.New(errMsg)
	}

	return err
}

func buildMark(t *testing.T, spec map[string]any, key string) Mark {
	t.Helper()

	markSpec, ok := spec[key].(map[string]any)
	if !ok {
		return Mark{}
	}

	return Mark{
		Line:   getInt(t, markSpec, "line"),
		Column: getInt(t, markSpec, "column"),
		Index:  getInt(t, markSpec, "index"),
	}
}

func buildLoadErrorList(t *testing.T, spec map[string]any) []*LoadError {
	t.Helper()

	errorsSpec, ok := spec["errors"].([]any)
	if !ok {
		return nil
	}

	var result []*LoadError
	for _, errSpec := range errorsSpec {
		errMap, ok := errSpec.(map[string]any)
		assert.Truef(t, ok, "error spec should be map[string]any")

		line := getInt(t, errMap, "line")
		message := getString(t, errMap, "message")

		result = append(result, &LoadError{
			Stage:   ConstructorStage,
			Mark:    Mark{Line: line},
			Message: message,
			err:     errors.New(message),
		})
	}

	return result
}

func getString(t *testing.T, spec map[string]any, key string) string {
	t.Helper()
	v, ok := spec[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	assert.Truef(t, ok, "%s should be string, got %T", key, v)
	return s
}

func getInt(t *testing.T, spec map[string]any, key string) int {
	t.Helper()
	v, ok := spec[key]
	if !ok {
		return 0
	}
	i, ok := v.(int)
	assert.Truef(t, ok, "%s should be int, got %T", key, v)
	return i
}

func getBool(t *testing.T, spec map[string]any, key string) bool {
	t.Helper()
	v, ok := spec[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	assert.Truef(t, ok, "%s should be bool, got %T", key, v)
	return b
}

func getStringSlice(t *testing.T, spec map[string]any, key string) []string {
	t.Helper()
	v, ok := spec[key]
	if !ok {
		return nil
	}
	slice, ok := v.([]any)
	assert.Truef(t, ok, "%s should be []any, got %T", key, v)

	var result []string
	for i, item := range slice {
		s, ok := item.(string)
		assert.Truef(t, ok, "%s[%d] should be string, got %T", key, i, item)
		result = append(result, s)
	}
	return result
}
