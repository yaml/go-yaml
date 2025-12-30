package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

// TestCase represents a single test case from a test file
type TestCase struct {
	Name  string `yaml:"name"`
	Text  string `yaml:"text"`
	Token string `yaml:"token,omitempty"`
	TOKEN string `yaml:"TOKEN,omitempty"`
	Event string `yaml:"event,omitempty"`
	EVENT string `yaml:"EVENT,omitempty"`
	Node  string `yaml:"node,omitempty"`
	NODE  string `yaml:"NODE,omitempty"`
	Yaml  string `yaml:"yaml,omitempty"`
	YAML  string `yaml:"YAML,omitempty"`
	Json  string `yaml:"json,omitempty"`
	JSON  string `yaml:"JSON,omitempty"`
}

// TestSuite is a sequence of test cases
type TestSuite []TestCase

// flagMapping maps test file field names to CLI flags
var flagMapping = map[string]string{
	"token": "-t",
	"TOKEN": "-T",
	"event": "-e",
	"EVENT": "-E",
	"node":  "-n",
	"NODE":  "-N",
	"yaml":  "-y",
	"YAML":  "-Y",
	"json":  "-j",
	"JSON":  "-J",
}

func TestCLI(t *testing.T) {
	// Find all test files in testdata/
	testFiles, err := filepath.Glob("testdata/*.yaml")
	if err != nil {
		t.Fatalf("Failed to find test files: %v", err)
	}

	if len(testFiles) == 0 {
		t.Skip("No test files found in testdata/")
	}

	// Process each test file
	for _, testFile := range testFiles {
		testFileName := filepath.Base(testFile)
		t.Run(testFileName, func(t *testing.T) {
			runTestFile(t, testFile)
		})
	}
}

func runTestFile(t *testing.T, testFile string) {
	// Read and parse the test file
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file %s: %v", testFile, err)
	}

	var suite TestSuite
	if err := yaml.Load(data, &suite); err != nil {
		t.Fatalf("Failed to parse test file %s: %v", testFile, err)
	}

	// Run each test case
	for _, testCase := range suite {
		t.Run(testCase.Name, func(t *testing.T) {
			runTestCase(t, testCase)
		})
	}
}

func runTestCase(t *testing.T, tc TestCase) {
	// Test each output format that has an expected value
	tests := []struct {
		field    string
		flag     string
		expected string
	}{
		{"token", flagMapping["token"], tc.Token},
		{"TOKEN", flagMapping["TOKEN"], tc.TOKEN},
		{"event", flagMapping["event"], tc.Event},
		{"EVENT", flagMapping["EVENT"], tc.EVENT},
		{"node", flagMapping["node"], tc.Node},
		{"NODE", flagMapping["NODE"], tc.NODE},
		{"yaml", flagMapping["yaml"], tc.Yaml},
		{"YAML", flagMapping["YAML"], tc.YAML},
		{"json", flagMapping["json"], tc.Json},
		{"JSON", flagMapping["JSON"], tc.JSON},
	}

	for _, test := range tests {
		if test.expected == "" {
			continue // Skip if no expected output for this format
		}

		t.Run(test.field, func(t *testing.T) {
			// Run the CLI command
			cmd := exec.Command("go", "run", ".", test.flag)
			cmd.Stdin = strings.NewReader(tc.Text)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				t.Fatalf("Command failed: %v\nStderr: %s", err, stderr.String())
			}

			// Normalize output for comparison
			actual := normalizeOutput(stdout.String())
			expected := normalizeOutput(test.expected)

			if actual != expected {
				t.Errorf("Output mismatch for flag %s\nExpected:\n%s\n\nActual:\n%s\n\nDiff:\n%s",
					test.flag, expected, actual, diff(expected, actual))
			}
		})
	}
}

// normalizeOutput trims whitespace and ensures consistent line endings
func normalizeOutput(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s
}

// diff provides a simple diff output for debugging
func diff(expected, actual string) string {
	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")

	maxLines := len(expLines)
	if len(actLines) > maxLines {
		maxLines = len(actLines)
	}

	var result strings.Builder
	for i := 0; i < maxLines; i++ {
		expLine := ""
		actLine := ""

		if i < len(expLines) {
			expLine = expLines[i]
		}
		if i < len(actLines) {
			actLine = actLines[i]
		}

		if expLine != actLine {
			result.WriteString(fmt.Sprintf("Line %d:\n", i+1))
			if expLine != "" {
				result.WriteString("- ")
				result.WriteString(expLine)
				result.WriteString("\n")
			}
			if actLine != "" {
				result.WriteString("+ ")
				result.WriteString(actLine)
				result.WriteString("\n")
			}
		}
	}

	return result.String()
}
