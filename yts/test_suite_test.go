package yts

import (
	"bufio"
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

const (
	testDir        = "testdata/data-2022-01-17"
	knownTestsFile = "known-failing-tests"
)

// run `make test-data` to download the test data
//
//go:embed known-failing-tests
//go:embed testdata/data-2022-01-17
var testData embed.FS

var (
	knownFailingTests          map[string]int
	knownFailingButPassedTests = make(map[string]struct{})
)

func loadKnownFailingTests(tb testing.TB) map[string]int {
	tb.Helper()

	file, err := testData.Open(knownTestsFile)
	if err != nil {
		tb.Fatalf("failed to read known-failing-tests: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	knownTests := make(map[string]int)
	for scanner.Scan() {
		trimmedLine := strings.TrimSpace(scanner.Text())
		if trimmedLine != "" {
			knownTests[trimmedLine] = 0
		}
	}
	return knownTests
}

func shouldSkipTest(tb testing.TB) {
	tb.Helper()

	_, isKnownFailing := knownFailingTests[tb.Name()]
	if isKnownFailing {
		knownFailingTests[tb.Name()]++
	}
	tb.Cleanup(func() {
		if isKnownFailing && !tb.Failed() && !tb.Skipped() {
			knownFailingButPassedTests[tb.Name()] = struct{}{}
		}
	})
	if os.Getenv("RUNALL") == "1" {
		return
	}
	runFailing := os.Getenv("RUNFAILING") == "1"
	tb.Logf("runFailing: %v; isKnownFailing: %v", runFailing, isKnownFailing)

	switch {
	case runFailing && !isKnownFailing:
		tb.Skip("Skipping non-failing test")
	case !runFailing && isKnownFailing:
		tb.Skip("Skipping known failing test")
	}
}

func TestYAMLSuite(t *testing.T) {
	knownFailingTests = loadKnownFailingTests(t)

	runTestsInDir(t, testDir)

	t.Run("CheckKnownFailingTests", func(t *testing.T) {
		for testName, cnt := range knownFailingTests {
			if cnt == 0 {
				t.Errorf("Known failing test %q did not run", testName)
			}
		}
	})
	t.Run("CheckKnownFailingButPassedTests", func(t *testing.T) {
		for testName := range knownFailingButPassedTests {
			t.Errorf("Known failing test %q did not fail as expected", testName)
		}
	})
}

func runTestsInDir(t *testing.T, dirPath string) {
	t.Helper()

	entries, err := testData.ReadDir(dirPath)
	if err != nil {
		t.Fatalf("Failed to read directory %q: %v", dirPath, err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())
		if entry.IsDir() {
			t.Run(entry.Name(), func(t *testing.T) {
				// Check if it's a test case directory (contains in.yaml)
				if file, err := testData.Open(filepath.Join(entryPath, "in.yaml")); err == nil {
					file.Close()
					runTest(t, entryPath)
				} else {
					// Otherwise, recurse into the subdirectory
					runTestsInDir(t, entryPath)
				}
			})
		}
	}
}

func normalizeLineEndings(s string) string {
	return strings.NewReplacer(
		"\r", "",
		"+DOC ---", "+DOC",
	).Replace(s)
}

func getEvents(in []byte) (string, error) {
	return yaml.ParserGetEvents(in)
}

func runTest(t *testing.T, testPath string) {
	t.Helper()

	// Read test description
	descPath := filepath.Join(testPath, "===")
	desc, err := os.ReadFile(descPath)
	var testDescription string
	if err == nil {
		testDescription = string(desc)
	} else {
		testDescription = "No description available."
	}

	t.Logf("Running test: %s\nDescription: %s", testPath, testDescription)

	inYAMLPath := filepath.Join(testPath, "in.yaml")
	inYAML, err := os.ReadFile(inYAMLPath)
	if err != nil {
		t.Fatalf("Failed to read in.yaml: %v", err)
	}

	errorPath := filepath.Join(testPath, "error")
	_, err = os.Stat(errorPath)
	expectError := err == nil

	var unmarshaledValue any
	var unmarshalErr error

	t.Run("UnmarshalTest", func(t *testing.T) {
		t.Logf("Running test: %s\nDescription: %s", testPath, testDescription)
		shouldSkipTest(t)
		unmarshalErr = yaml.Unmarshal(inYAML, &unmarshaledValue)

		if expectError {
			if unmarshalErr == nil {
				t.Errorf("Expected unmarshal error but got none: %+v", unmarshaledValue)
			}
			return
		}
		if unmarshalErr != nil {
			t.Errorf("Unexpected unmarshal error: %v\n%s", unmarshalErr, inYAML)
		}
	})

	t.Run("EventComparisonTest", func(t *testing.T) {
		t.Logf("Running test: %s\nDescription: %s", testPath, testDescription)
		shouldSkipTest(t)
		expectedEventsPath := filepath.Join(testPath, "test.event")
		if _, err := os.Stat(expectedEventsPath); err != nil {
			t.Logf("No test.event file found: %s", expectedEventsPath)
			return
		}
		expectedEventsBytes, err := os.ReadFile(expectedEventsPath)
		if err != nil {
			t.Errorf("Failed to read test.event: %v", err)
			return
		}
		expectedEvents := normalizeLineEndings(string(expectedEventsBytes))
		expectedEvents = strings.TrimSuffix(expectedEvents, "\n")
		actualEvents, eventErr := getEvents(inYAML)

		if expectError {
			if eventErr == nil {
				t.Error("Expected error on event parsing but got none")
			}
			return
		}
		if eventErr != nil {
			t.Errorf("Unexpected error on event parsing: %v\n%s", eventErr, inYAML)
			return
		}
		actualEventsStr := normalizeLineEndings(actualEvents)
		if actualEventsStr != expectedEvents {
			t.Errorf("Event mismatch\nExpected:\n%q\nGot:\n%q", expectedEvents, actualEventsStr)
		}
	})

	// Only proceed with marshal and JSON tests if unmarshal was successful and no expected error

	t.Run("MarshalTest", func(t *testing.T) {
		t.Logf("Running test: %s\nDescription: %s", testPath, testDescription)
		shouldSkipTest(t)
		var currentUnmarshaledValue any

		currentUnmarshalErr := yaml.Unmarshal(inYAML, &currentUnmarshaledValue)

		if !(currentUnmarshalErr == nil || expectError) {
			return
		}
		marshaledYAML, marshalErr := yaml.Marshal(currentUnmarshaledValue)
		if marshalErr != nil {
			t.Errorf("Failed to marshal value: %v", marshalErr)
			return
		}
		outYAMLPath := filepath.Join(testPath, "out.yaml")
		if _, err := os.Stat(outYAMLPath); err != nil {
			t.Logf("No out.yaml file found: %s", outYAMLPath)
			return
		}
		expectedOutYAML, err := os.ReadFile(outYAMLPath)
		if err != nil {
			t.Errorf("Failed to read out.yaml: %v", err)
			return
		}
		var expectedUnmarshaledValue any
		err = yaml.Unmarshal(expectedOutYAML, &expectedUnmarshaledValue)
		if err != nil {
			t.Errorf("Failed to unmarshal out.yaml: %v", err)
			return
		}
		var reUnmarshaledValue any
		err = yaml.Unmarshal(marshaledYAML, &reUnmarshaledValue)
		if err != nil {
			t.Errorf("Failed to re-unmarshal marshaled YAML: %v", err)
		} else if !reflect.DeepEqual(reUnmarshaledValue, expectedUnmarshaledValue) {
			t.Errorf("Marshal output mismatch\nExpected: %+v\nGot     : %+v", expectedUnmarshaledValue, reUnmarshaledValue)
		}
	})

	t.Run("JSONComparisonTest", func(t *testing.T) {
		t.Logf("Running test: %s\nDescription: %s", testPath, testDescription)
		shouldSkipTest(t)
		var currentUnmarshaledValue any

		currentUnmarshalErr := yaml.Unmarshal(inYAML, &currentUnmarshaledValue)

		if !(currentUnmarshalErr == nil || expectError) {
			return
		}
		inJSONPath := filepath.Join(testPath, "in.json")
		if _, err := os.Stat(inJSONPath); err != nil {
			t.Logf("No in.json file found: %s", inJSONPath)
			return
		}
		inJSON, err := os.ReadFile(inJSONPath)
		if err != nil {
			t.Errorf("Failed to read in.json: %v", err)
			return
		}
		var jsonValue any
		jsonErr := json.Unmarshal(inJSON, &jsonValue)
		if jsonErr != nil {
			t.Errorf("Failed to unmarshal in.json: %v", jsonErr)
			return
		}

		// Comparing the unmarshaled YAML value with the JSON value fails
		// due to differences in how YAML and JSON handle number types.
		// JSON unmarshals all numbers as float64, while YAML preserves
		// integer types when possible. To work around this, we marshal
		// both values to JSON and compare the results.

		yamlAsJSONText, err := json.Marshal(currentUnmarshaledValue)
		if err != nil {
			t.Errorf("Failed to marshal YAML value to JSON: %v", err)
			return
		}
		jsonAsText, err := json.Marshal(jsonValue)
		if err != nil {
			t.Errorf("Failed to marshal JSON value to JSON: %v", err)
			return
		}
		if string(yamlAsJSONText) != string(jsonAsText) {
			t.Errorf("YAML unmarshal vs JSON unmarshal mismatch\nExpected: %s\nGot     : %s", string(jsonAsText), string(yamlAsJSONText))
		}
	})
}
