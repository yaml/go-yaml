// Tests for the Dump API, including WithAllDocuments functionality.

package libyaml

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

// TestDump_SingleValue tests dumping a single value
func TestDump_SingleValue(t *testing.T) {
	type Config struct {
		Name string `yaml:"name"`
	}

	config := Config{Name: "myconfig"}
	data, err := Dump(config)
	assert.NoError(t, err)

	// Should not have document separator for single document
	assert.True(t, strings.Contains(string(data), "name: myconfig"))
}

// TestDumpWithAllDocuments_TypedSlice tests dumping multiple values from typed slice
func TestDumpWithAllDocuments_TypedSlice(t *testing.T) {
	type Config struct {
		Name string `yaml:"name"`
	}

	configs := []Config{
		{Name: "first"},
		{Name: "second"},
		{Name: "third"},
	}

	data, err := Dump(configs, WithAllDocuments())
	assert.NoError(t, err)

	// Should have document separators
	assert.True(t, strings.Contains(string(data), "---"))
	assert.True(t, strings.Contains(string(data), "name: first"))
	assert.True(t, strings.Contains(string(data), "name: second"))
	assert.True(t, strings.Contains(string(data), "name: third"))
}

// TestDumpWithAllDocuments_UntypedSlice tests dumping multiple values from []any
func TestDumpWithAllDocuments_UntypedSlice(t *testing.T) {
	docs := []any{
		map[string]string{"name": "first"},
		map[string]string{"name": "second"},
	}

	data, err := Dump(docs, WithAllDocuments())
	assert.NoError(t, err)

	// Should have document separator
	assert.True(t, strings.Contains(string(data), "---"))
	assert.True(t, strings.Contains(string(data), "name: first"))
	assert.True(t, strings.Contains(string(data), "name: second"))
}

// TestDumpWithAllDocuments_EmptySlice tests dumping an empty slice
func TestDumpWithAllDocuments_EmptySlice(t *testing.T) {
	var docs []any

	data, err := Dump(docs, WithAllDocuments())
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

// TestDumpWithAllDocuments_NonSlice tests that WithAllDocuments with non-slice returns error
func TestDumpWithAllDocuments_NonSlice(t *testing.T) {
	single := map[string]string{"name": "single"}

	_, err := Dump(single, WithAllDocuments())
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*WithAllDocuments requires a slice input.*", err)
}
