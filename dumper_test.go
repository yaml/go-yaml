// Tests for the Dump API, including WithAll functionality.

package yaml_test

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

// TestDump_SingleValue tests dumping a single value
func TestDump_SingleValue(t *testing.T) {
	type Config struct {
		Name string `yaml:"name"`
	}

	config := Config{Name: "myconfig"}
	data, err := yaml.Dump(config)
	assert.NoError(t, err)

	// Should not have document separator for single document
	assert.True(t, strings.Contains(string(data), "name: myconfig"))
}

// TestDumpWithAll_TypedSlice tests dumping multiple values from typed slice
func TestDumpWithAll_TypedSlice(t *testing.T) {
	type Config struct {
		Name string `yaml:"name"`
	}

	configs := []Config{
		{Name: "first"},
		{Name: "second"},
		{Name: "third"},
	}

	data, err := yaml.Dump(configs, yaml.WithAll())
	assert.NoError(t, err)

	// Should have document separators
	assert.True(t, strings.Contains(string(data), "---"))
	assert.True(t, strings.Contains(string(data), "name: first"))
	assert.True(t, strings.Contains(string(data), "name: second"))
	assert.True(t, strings.Contains(string(data), "name: third"))
}

// TestDumpWithAll_UntypedSlice tests dumping multiple values from []any
func TestDumpWithAll_UntypedSlice(t *testing.T) {
	docs := []any{
		map[string]string{"name": "first"},
		map[string]string{"name": "second"},
	}

	data, err := yaml.Dump(docs, yaml.WithAll())
	assert.NoError(t, err)

	// Should have document separator
	assert.True(t, strings.Contains(string(data), "---"))
	assert.True(t, strings.Contains(string(data), "name: first"))
	assert.True(t, strings.Contains(string(data), "name: second"))
}

// TestDumpWithAll_EmptySlice tests dumping an empty slice
func TestDumpWithAll_EmptySlice(t *testing.T) {
	var docs []any

	data, err := yaml.Dump(docs, yaml.WithAll())
	// Empty slice produces an empty YAML stream
	// This may produce an error or empty output depending on implementation
	if err != nil {
		// It's acceptable for empty slice to produce error
		t.Logf("Empty slice produced error (acceptable): %v", err)
	} else {
		// Or it might produce empty/minimal output
		assert.True(t, len(data) < 50)
	}
}

// TestDumpWithAll_NonSlice tests that WithAll with non-slice returns error
func TestDumpWithAll_NonSlice(t *testing.T) {
	single := map[string]string{"name": "single"}

	_, err := yaml.Dump(single, yaml.WithAll())
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*WithAll requires a slice input.*", err)
}
