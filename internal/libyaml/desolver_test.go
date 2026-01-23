// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Tests for the Desolver stage

package libyaml

import (
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestDesolver(t *testing.T) {
	RunTestCases(t, "desolver.yaml", map[string]TestHandler{
		"desolve-inferable": func(t *testing.T, tc TestCase) {
			t.Helper()

			node := &Node{
				Kind:  ScalarNode,
				Tag:   tc.Node.Tag,
				Value: tc.Node.Value,
			}

			d := NewDesolver(nil)
			d.Desolve(node)

			// Extract want fields from tc.Want (type any)
			wantMap := tc.Want.(map[string]any)
			wantTag := wantMap["tag"].(string)

			// Check tag
			assert.Equal(t, wantTag, node.Tag)

			// Check style
			if wantStyle, ok := wantMap["style"].(string); ok {
				hasQuote := node.Style&(SingleQuotedStyle|DoubleQuotedStyle) != 0
				switch wantStyle {
				case "Plain":
					assert.False(t, hasQuote)
				case "Single":
					assert.True(t, hasQuote)
				}
			}
		},

		"desolve-preserve": func(t *testing.T, tc TestCase) {
			t.Helper()

			node := &Node{
				Kind: ScalarNode,
				Tag:  tc.Node.Tag,
			}

			// Handle kind for collection tests
			if tc.Node.Kind != "" {
				switch tc.Node.Kind {
				case "Mapping":
					node.Kind = MappingNode
				case "Sequence":
					node.Kind = SequenceNode
				}
			} else {
				// Scalar node needs value
				node.Value = tc.Node.Value
			}

			// Handle style for explicitly tagged tests
			if tc.Node.Style == "Tagged" {
				node.Style = TaggedStyle
			}

			d := NewDesolver(nil)
			d.Desolve(node)

			// Extract want fields
			wantMap := tc.Want.(map[string]any)
			wantTag := wantMap["tag"].(string)

			// Check tag is preserved
			assert.Equal(t, wantTag, node.Tag)

			// Check style if present
			if wantStyle, ok := wantMap["style"].(string); ok {
				hasQuote := node.Style&(SingleQuotedStyle|DoubleQuotedStyle) != 0
				switch wantStyle {
				case "Plain":
					assert.False(t, hasQuote)
				case "Single":
					assert.True(t, hasQuote)
				}
			}
		},

		"desolve-string-quoting": func(t *testing.T, tc TestCase) {
			t.Helper()

			node := &Node{
				Kind:  ScalarNode,
				Tag:   tc.Node.Tag,
				Value: tc.Node.Value,
			}

			d := NewDesolver(nil)
			d.Desolve(node)

			// Extract want fields
			wantMap := tc.Want.(map[string]any)
			wantTag := wantMap["tag"].(string)

			// Check tag removed
			assert.Equal(t, wantTag, node.Tag)

			// Check style
			if wantStyle, ok := wantMap["style"].(string); ok {
				hasQuote := node.Style&(SingleQuotedStyle|DoubleQuotedStyle) != 0
				switch wantStyle {
				case "Plain":
					assert.False(t, hasQuote)
				case "Single":
					assert.True(t, hasQuote)
				}
			}
		},
	})
}
