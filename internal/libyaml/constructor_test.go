// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestConstructor(t *testing.T) {
	RunTestCases(t, "constructor.yaml", map[string]TestHandler{
		"scalar-resolution": func(t *testing.T, tc TestCase) {
			t.Helper()

			// Load the YAML
			result, err := LoadYAML([]byte(tc.Yaml))
			assert.NoErrorf(t, err, "LoadYAML() error: %v", err)

			// Compare the result with expected value
			if !reflect.DeepEqual(result, tc.Want) {
				t.Errorf("LoadYAML() = %v (type: %T), want %v (type: %T)",
					result, result, tc.Want, tc.Want)
			}
		},
	})
}
