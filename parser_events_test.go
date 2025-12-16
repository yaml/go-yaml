package yaml_test

import (
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
	"go.yaml.in/yaml/v4/internal/testutil/datatest"
)

func TestParserGetEvents(t *testing.T) {
	datatest.RunTestCases(t, func() ([]map[string]interface{}, error) {
		return datatest.LoadTestCasesFromFile("testdata/parser_events.yaml", libyaml.LoadYAML)
	}, map[string]datatest.TestHandler{
		"parser-events": runParserEventsTest,
	})
}

func runParserEventsTest(t *testing.T, tc map[string]interface{}) {
	t.Helper()

	// Extract test data
	yamlInput := datatest.RequireString(t, tc, "yaml")
	want := datatest.RequireString(t, tc, "want")

	// Run test
	events, err := yaml.ParserGetEvents([]byte(yamlInput))
	if err != nil {
		t.Fatalf("ParserGetEvents error: %v", err)
	}

	// Trim trailing newline from want (YAML literal blocks add one)
	want = datatest.TrimTrailingNewline(want)

	assert.Equal(t, want, events)
}
