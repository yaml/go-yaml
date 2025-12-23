package yaml_test

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
	"go.yaml.in/yaml/v4/internal/testutil/datatest"
)

func TestLimits(t *testing.T) {
	datatest.RunTestCases(t, func() ([]map[string]any, error) {
		return datatest.LoadTestCasesFromFile("testdata/limit.yaml", libyaml.LoadYAML)
	}, map[string]datatest.TestHandler{
		"limit":       runLimitTest,
		"limit-error": runLimitTest,
		"limit-pass":  runLimitTest,
	})
}

func runLimitTest(t *testing.T, tc map[string]any) {
	t.Helper()

	// Generate data from spec
	dataSpec := tc["data"]
	data, err := datatest.GenerateData(dataSpec)
	if err != nil {
		t.Fatalf("Failed to generate data: %v", err)
	}

	// Get expected error if any (for limit-error tests)
	// For limit-pass tests, want might be a map describing expected structure
	expectedError := ""
	if wantVal, hasWant := tc["want"]; hasWant {
		switch v := wantVal.(type) {
		case string:
			expectedError = v
		case map[string]any:
			// Future: could validate structure here
			// For now, just ignore (treated as success case)
		default:
			t.Fatalf("want field must be a string or map, got %T", wantVal)
		}
	}

	// Run unmarshal
	var v any
	err = yaml.Unmarshal(data, &v)
	if expectedError != "" {
		if err == nil {
			t.Fatalf("expected error %q, got nil", expectedError)
		}
		assert.Equal(t, expectedError, err.Error())
		return
	}
	assert.NoError(t, err)
}

// Keep benchmark using hardcoded data for performance consistency
var limitTests = []struct {
	name  string
	data  []byte
	error string
}{
	{
		name:  "1000kb of maps with 100 aliases",
		data:  []byte(`{a: &a [{a}` + strings.Repeat(`,{a}`, 1000*1024/4-100) + `], b: &b [*a` + strings.Repeat(`,*a`, 99) + `]}`),
		error: "yaml: document contains excessive aliasing",
	},
	{
		name:  "1000kb of deeply nested slices",
		data:  []byte(strings.Repeat(`[`, 1000*1024)),
		error: "yaml: while increasing flow level at line 1, column 10001: exceeded max depth of 10000",
	},
	{
		name:  "1000kb of deeply nested maps",
		data:  []byte("x: " + strings.Repeat(`{`, 1000*1024)),
		error: "yaml: while increasing flow level at line 1, column 10004: exceeded max depth of 10000",
	},
	{
		name:  "1000kb of deeply nested indents",
		data:  []byte(strings.Repeat(`- `, 1000*1024)),
		error: "yaml: while increasing indent level at line 1: line 1, column 20001: exceeded max depth of 10000",
	},
	{
		name: "1000kb of 1000-indent lines",
		data: []byte(strings.Repeat(strings.Repeat(`- `, 1000)+"\n", 1024/2)),
	},
	{name: "1kb of maps", data: []byte(`a: &a [{a}` + strings.Repeat(`,{a}`, 1*1024/4-1) + `]`)},
	{name: "10kb of maps", data: []byte(`a: &a [{a}` + strings.Repeat(`,{a}`, 10*1024/4-1) + `]`)},
	{name: "100kb of maps", data: []byte(`a: &a [{a}` + strings.Repeat(`,{a}`, 100*1024/4-1) + `]`)},
	{name: "1000kb of maps", data: []byte(`a: &a [{a}` + strings.Repeat(`,{a}`, 1000*1024/4-1) + `]`)},
	{name: "1000kb slice nested at max-depth", data: []byte(strings.Repeat(`[`, 10000) + `1` + strings.Repeat(`,1`, 1000*1024/2-20000-1) + strings.Repeat(`]`, 10000))},
	{name: "1000kb slice nested in maps at max-depth", data: []byte("{a,b:\n" + strings.Repeat(" {a,b:", 10000-2) + ` [1` + strings.Repeat(",1", 1000*1024/2-6*10000-1) + `]` + strings.Repeat(`}`, 10000-1))},
	{name: "1000kb of 10000-nested lines", data: []byte(strings.Repeat(`- `+strings.Repeat(`[`, 10000)+strings.Repeat(`]`, 10000)+"\n", 1000*1024/20000))},
}

func BenchmarkLimits(b *testing.B) {
	for _, tc := range limitTests {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				var v any
				err := yaml.Unmarshal(tc.data, &v)
				if tc.error != "" {
					assert.ErrorMatches(b, tc.error, err)
					continue
				}
				assert.NoError(b, err)
			}
		})
	}
}
