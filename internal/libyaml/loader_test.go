// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestLoader(t *testing.T) {
	RunTestCases(t, "loader.yaml", map[string]TestHandler{
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
