// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for options.go functions and methods.

package libyaml

import (
	"regexp"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestOptions(t *testing.T) {
	handlers := map[string]TestHandler{
		"with-indent":                   runWithIndentTest,
		"with-compact-seq-indent":       runWithCompactSeqIndentTest,
		"with-known-fields":             runWithKnownFieldsTest,
		"with-single-document":          runWithSingleDocumentTest,
		"with-stream-nodes":             runWithStreamNodesTest,
		"with-all-documents":            runWithAllDocumentsTest,
		"with-line-width":               runWithLineWidthTest,
		"with-unicode":                  runWithUnicodeTest,
		"with-unique-keys":              runWithUniqueKeysTest,
		"with-canonical":                runWithCanonicalTest,
		"with-line-break":               runWithLineBreakTest,
		"with-explicit-start":           runWithExplicitStartTest,
		"with-explicit-end":             runWithExplicitEndTest,
		"with-flow-simple-collections":  runWithFlowSimpleCollectionsTest,
		"with-quote-preference":         runWithQuotePreferenceTest,
		"with-no-aliasing-restrictions": runWithAliasingRestrictionFunctionTest,
		"apply-options":                 runApplyOptionsTest,
	}

	RunTestCases(t, "options.yaml", handlers)
}

// runWithIndentTest tests WithIndent
func runWithIndentTest(t *testing.T, tc TestCase) {
	t.Helper()

	indent, ok := tc.From.(int)
	if !ok {
		t.Fatalf("from should be int, got %T", tc.From)
	}

	opt := WithIndent(indent)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		// Expect error
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		// Expect success
		assert.NoErrorf(t, err, "WithIndent(%d) error: %v", indent, err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithCompactSeqIndentTest tests WithCompactSeqIndent
func runWithCompactSeqIndentTest(t *testing.T, tc TestCase) {
	t.Helper()

	args := parseBoolSlice(t, tc.From)
	opt := WithCompactSeqIndent(args...)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithCompactSeqIndent error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithKnownFieldsTest tests WithKnownFields
func runWithKnownFieldsTest(t *testing.T, tc TestCase) {
	t.Helper()

	args := parseBoolSlice(t, tc.From)
	opt := WithKnownFields(args...)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithKnownFields error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithSingleDocumentTest tests WithSingleDocument
func runWithSingleDocumentTest(t *testing.T, tc TestCase) {
	t.Helper()

	args := parseBoolSlice(t, tc.From)
	opt := WithSingleDocument(args...)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithSingleDocument error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithStreamNodesTest tests WithStreamNodes
func runWithStreamNodesTest(t *testing.T, tc TestCase) {
	t.Helper()

	args := parseBoolSlice(t, tc.From)
	opt := WithStreamNodes(args...)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithStreamNodes error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithAllDocumentsTest tests WithAllDocuments
func runWithAllDocumentsTest(t *testing.T, tc TestCase) {
	t.Helper()

	args := parseBoolSlice(t, tc.From)
	opt := WithAllDocuments(args...)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithAllDocuments error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithLineWidthTest tests WithLineWidth
func runWithLineWidthTest(t *testing.T, tc TestCase) {
	t.Helper()

	width, ok := tc.From.(int)
	if !ok {
		t.Fatalf("from should be int, got %T", tc.From)
	}

	opt := WithLineWidth(width)
	opts := &Options{}
	err := opt(opts)

	assert.NoErrorf(t, err, "WithLineWidth(%d) error: %v", width, err)
	checkWantFields(t, opts, tc.Want)
}

// runWithUnicodeTest tests WithUnicode
func runWithUnicodeTest(t *testing.T, tc TestCase) {
	t.Helper()

	args := parseBoolSlice(t, tc.From)
	opt := WithUnicode(args...)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithUnicode error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithUniqueKeysTest tests WithUniqueKeys
func runWithUniqueKeysTest(t *testing.T, tc TestCase) {
	t.Helper()

	args := parseBoolSlice(t, tc.From)
	opt := WithUniqueKeys(args...)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithUniqueKeys error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithCanonicalTest tests WithCanonical
func runWithCanonicalTest(t *testing.T, tc TestCase) {
	t.Helper()

	args := parseBoolSlice(t, tc.From)
	opt := WithCanonical(args...)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithCanonical error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithLineBreakTest tests WithLineBreak
func runWithLineBreakTest(t *testing.T, tc TestCase) {
	t.Helper()

	lineBreak := parseLineBreak(t, tc.From)
	opt := WithLineBreak(lineBreak)
	opts := &Options{}
	err := opt(opts)

	assert.NoErrorf(t, err, "WithLineBreak error: %v", err)
	checkWantFields(t, opts, tc.Want)
}

// runWithExplicitStartTest tests WithExplicitStart
func runWithExplicitStartTest(t *testing.T, tc TestCase) {
	t.Helper()

	args := parseBoolSlice(t, tc.From)
	opt := WithExplicitStart(args...)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithExplicitStart error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithExplicitEndTest tests WithExplicitEnd
func runWithExplicitEndTest(t *testing.T, tc TestCase) {
	t.Helper()

	args := parseBoolSlice(t, tc.From)
	opt := WithExplicitEnd(args...)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithExplicitEnd error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithFlowSimpleCollectionsTest tests WithFlowSimpleCollections
func runWithFlowSimpleCollectionsTest(t *testing.T, tc TestCase) {
	t.Helper()

	args := parseBoolSlice(t, tc.From)
	opt := WithFlowSimpleCollections(args...)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithFlowSimpleCollections error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithQuotePreferenceTest tests WithQuotePreference
func runWithQuotePreferenceTest(t *testing.T, tc TestCase) {
	t.Helper()

	style := parseQuoteStyle(t, tc.From)
	opt := WithQuotePreference(style)
	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithQuotePreference error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runWithAliasingRestrictionFunctionTest tests that WithAliasingRestrictionFunction
// correctly can set a restriction function.
func runWithAliasingRestrictionFunctionTest(t *testing.T, tc TestCase) {
	t.Helper()

	var altFn AliasingRestrictionFunction
	altFn = func(aliasCount int, constructCount int) bool {
		return true
	}

	opt := WithAliasingRestrictionFunction(altFn)

	opts := &Options{}
	err := opt(opts)

	if tc.Like != "" {
		assert.NotNilf(t, err, "expected error matching %q", tc.Like)
		if err != nil {
			matched, _ := regexp.MatchString(tc.Like, err.Error())
			assert.Truef(t, matched, "error %q should match %q", err.Error(), tc.Like)
		}
	} else {
		assert.NoErrorf(t, err, "WithAliasingRestrictionFunction error: %v", err)
		checkWantFields(t, opts, tc.Want)
	}
}

// runApplyOptionsTest tests ApplyOptions
func runApplyOptionsTest(t *testing.T, tc TestCase) {
	t.Helper()

	// Test with no options to verify v4 defaults
	opts, err := ApplyOptions()

	assert.NoErrorf(t, err, "ApplyOptions error: %v", err)
	if opts != nil {
		checkWantFields(t, opts, tc.Want)
	}
}

// Helper functions

// parseBoolSlice converts tc.From to []bool
func parseBoolSlice(t *testing.T, from any) []bool {
	t.Helper()

	slice, ok := from.([]any)
	if !ok {
		t.Fatalf("from should be []any, got %T", from)
	}

	result := make([]bool, len(slice))
	for i, v := range slice {
		b, ok := v.(bool)
		if !ok {
			t.Fatalf("from[%d] should be bool, got %T", i, v)
		}
		result[i] = b
	}
	return result
}

// parseLineBreak converts string or int to LineBreak
func parseLineBreak(t *testing.T, from any) LineBreak {
	t.Helper()

	switch v := from.(type) {
	case string:
		switch v {
		case "LN_BREAK":
			return LN_BREAK
		case "CR_BREAK":
			return CR_BREAK
		case "CRLN_BREAK":
			return CRLN_BREAK
		default:
			t.Fatalf("unknown LineBreak constant: %s", v)
		}
	case int:
		return LineBreak(v)
	default:
		t.Fatalf("from should be string or int, got %T", from)
	}
	return 0
}

// parseQuoteStyle converts string or int to QuoteStyle
func parseQuoteStyle(t *testing.T, from any) QuoteStyle {
	t.Helper()

	switch v := from.(type) {
	case string:
		switch v {
		case "QuoteSingle":
			return QuoteSingle
		case "QuoteDouble":
			return QuoteDouble
		case "QuoteLegacy":
			return QuoteLegacy
		default:
			t.Fatalf("unknown QuoteStyle constant: %s", v)
		}
	case int:
		return QuoteStyle(v)
	default:
		t.Fatalf("from should be string or int, got %T", from)
	}
	return 0
}

// checkWantFields verifies expected fields in Options
func checkWantFields(t *testing.T, opts *Options, want any) {
	t.Helper()

	if want == nil {
		return
	}

	wantMap, ok := want.(map[string]any)
	if !ok {
		t.Fatalf("want should be map, got %T", want)
	}

	for key, expectedValue := range wantMap {
		switch key {
		case "indent":
			expected, ok := expectedValue.(int)
			if !ok {
				t.Fatalf("want.indent should be int, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.Indent, "Indent = %d, want %d", opts.Indent, expected)

		case "compact_seq_indent":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.compact_seq_indent should be bool, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.CompactSeqIndent, "CompactSeqIndent = %v, want %v", opts.CompactSeqIndent, expected)

		case "known_fields":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.known_fields should be bool, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.KnownFields, "KnownFields = %v, want %v", opts.KnownFields, expected)

		case "single_document":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.single_document should be bool, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.SingleDocument, "SingleDocument = %v, want %v", opts.SingleDocument, expected)

		case "stream_nodes":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.stream_nodes should be bool, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.StreamNodes, "StreamNodes = %v, want %v", opts.StreamNodes, expected)

		case "all_documents":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.all_documents should be bool, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.AllDocuments, "AllDocuments = %v, want %v", opts.AllDocuments, expected)

		case "line_width":
			expected, ok := expectedValue.(int)
			if !ok {
				t.Fatalf("want.line_width should be int, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.LineWidth, "LineWidth = %d, want %d", opts.LineWidth, expected)

		case "unicode":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.unicode should be bool, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.Unicode, "Unicode = %v, want %v", opts.Unicode, expected)

		case "unique_keys":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.unique_keys should be bool, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.UniqueKeys, "UniqueKeys = %v, want %v", opts.UniqueKeys, expected)

		case "canonical":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.canonical should be bool, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.Canonical, "Canonical = %v, want %v", opts.Canonical, expected)

		case "line_break":
			expectedStr, ok := expectedValue.(string)
			if !ok {
				t.Fatalf("want.line_break should be string, got %T", expectedValue)
			}
			expected := parseLineBreak(t, expectedStr)
			assert.Equalf(t, expected, opts.LineBreak, "LineBreak = %v, want %v", opts.LineBreak, expected)

		case "explicit_start":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.explicit_start should be bool, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.ExplicitStart, "ExplicitStart = %v, want %v", opts.ExplicitStart, expected)

		case "explicit_end":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.explicit_end should be bool, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.ExplicitEnd, "ExplicitEnd = %v, want %v", opts.ExplicitEnd, expected)

		case "flow_simple_collections":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.flow_simple_collections should be bool, got %T", expectedValue)
			}
			assert.Equalf(t, expected, opts.FlowSimpleCollections, "FlowSimpleCollections = %v, want %v", opts.FlowSimpleCollections, expected)

		case "quote_preference":
			expectedStr, ok := expectedValue.(string)
			if !ok {
				t.Fatalf("want.quote_preference should be string, got %T", expectedValue)
			}
			expected := parseQuoteStyle(t, expectedStr)
			assert.Equalf(t, expected, opts.QuotePreference, "QuotePreference = %v, want %v", opts.QuotePreference, expected)

		case "aliasing_restriction_function":
			expected, ok := expectedValue.(bool)
			if !ok {
				t.Fatalf("want.quote_preference should be bool, got %T", expectedValue)
			}
			result := opts.AliasingRestrictionFunction(0, 0)
			assert.Equalf(t, expected, result, "AliasingRestrictionFunction call = %v, want %v", expected, result)

		default:
			t.Fatalf("unknown want field: %s", key)
		}
	}
}
