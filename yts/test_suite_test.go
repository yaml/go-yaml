package yts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

var knownFailingTests = loadKnownFailingTests()

func loadKnownFailingTests() map[string]bool {
	fileContent, err := os.ReadFile("known-failing-tests")
	if err != nil {
		return make(map[string]bool)
	}

	lines := strings.Split(string(fileContent), "\n")
	knownTests := make(map[string]bool)
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			knownTests[trimmedLine] = true
		}
	}
	return knownTests
}

func shouldSkipTest(t *testing.T) {
	if os.Getenv("RUNALL") == "1" {
		return
	}
	name := t.Name()
	runFailing := os.Getenv("RUNFAILING") == "1"
	isKnownFailing := knownFailingTests[name]
	t.Logf("NAME::: %v, %v, %v", name, runFailing, isKnownFailing)

	switch {
	case runFailing && !isKnownFailing:
		t.Skipf("Skipping non-failing test: %s", name)
	case !runFailing && isKnownFailing:
		t.Skipf("Skipping known failing test: %s", name)
	}
}

func TestYAMLSuite(t *testing.T) {
	testDir := "./testdata/data-2022-01-17"
	if _, err := os.Stat(testDir + "/229Q"); os.IsNotExist(err) {
		t.Fatalf(`YTS tests require data files to be present at '%s'.
Run 'make test-data' to download them first,
or just run the tests with 'make test-all'.`, testDir)
	}
	runTestsInDir(t, testDir)
}

func runTestsInDir(t *testing.T, dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		t.Fatalf("Failed to read directory %s: %v", dirPath, err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())
		if entry.IsDir() {
			// Check if it's a test case directory (contains in.yaml)
			if _, err := os.Stat(filepath.Join(entryPath, "in.yaml")); err == nil {
				t.Run(entry.Name(), func(t *testing.T) {
					runTest(t, entryPath)
				})
			} else {
				// Otherwise, recurse into the subdirectory
				runTestsInDir(t, entryPath)
			}
		}
	}
}

func normalizeLineEndings(s string) string {
	return strings.NewReplacer(
		"\r", "",
	).Replace(s)
}

func getEvents(in []byte) (string, error) {
	return yaml.ParserGetEvents(in)
}

func mustRead(t *testing.T, path, name string) []byte {
	data, err := os.ReadFile(filepath.Join(path, name))
	if err != nil {
		t.Fatalf("Failed to read %s (%s): %v", name, path, err)
	}
	return data
}

func fileExists(path, name string) bool {
	_, err := os.Stat(filepath.Join(path, name))
	return err == nil
}

func runTest(t *testing.T, testPath string) {
	t.Helper()

	// Read test description
	testDescription := mustRead(t, testPath, "===")

	t.Logf("Running test: %s\nDescription: %s", testPath, testDescription)

	inYAML := mustRead(t, testPath, "in.yaml")
	expectError := fileExists(testPath, "error")
	expectedEvents := normalizeLineEndings(string(mustRead(t, testPath, "test.event")))

	var unmarshaledValue any
	var unmarshalErr error

	t.Run("UnmarshalTest", func(t *testing.T) {
		shouldSkipTest(t)
		unmarshalErr = yaml.Unmarshal(inYAML, &unmarshaledValue)

		if expectError {
			if unmarshalErr == nil {
				t.Errorf(
					"Test: %s\nDescription: %s\nError: Expected unmarshal error but got none",
					testPath, testDescription)
			}
			return
		}
		if unmarshalErr != nil {
			t.Errorf(
				"Test: %s\nDescription: %s\nError: Unexpected unmarshal error: %v",
				testPath, testDescription, unmarshalErr)
		}
	})

	t.Run("EventComparisonTest", func(t *testing.T) {
		shouldSkipTest(t)
		expectedEvents = strings.TrimSuffix(expectedEvents, "\n")
		actualEvents, eventErr := getEvents(inYAML)

		if expectError {
			if eventErr == nil {
				t.Errorf(
					"Test: %s\nDescription: %s\nError: Expected error on event parsing but got none",
					testPath, testDescription)
			}
			return
		}
		if eventErr != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Unexpected error on event parsing: %v",
				testPath, testDescription, eventErr)
			return
		}
		actualEventsStr := normalizeLineEndings(actualEvents)
		if actualEventsStr != expectedEvents {
			t.Errorf(
				"Test: %s\nDescription: %s\nError: Event mismatch\nExpected:\n%q\nGot:\n%q",
				testPath, testDescription, expectedEvents, actualEventsStr)
		}
	})

	// Only proceed with marshal and JSON tests if unmarshal was successful and no expected error

	t.Run("MarshalTest", func(t *testing.T) {
		shouldSkipTest(t)
		var currentUnmarshaledValue any

		currentUnmarshalErr := yaml.Unmarshal(inYAML, &currentUnmarshaledValue)

		if !(currentUnmarshalErr == nil || expectError) {
			return
		}
		marshaledYAML, marshalErr := yaml.Marshal(currentUnmarshaledValue)
		if marshalErr != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Failed to marshal value: %v", testPath, testDescription, marshalErr)
			return
		}
		outYAMLPath := filepath.Join(testPath, "out.yaml")
		if _, err := os.Stat(outYAMLPath); err != nil {
			return
		}
		expectedOutYAML, err := os.ReadFile(outYAMLPath)
		if err != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Failed to read out.yaml: %v", testPath, testDescription, err)
			return
		}
		var expectedUnmarshaledValue any
		err = yaml.Unmarshal(expectedOutYAML, &expectedUnmarshaledValue)
		if err != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Failed to unmarshal out.yaml: %v", testPath, testDescription, err)
			return
		}
		var reUnmarshaledValue any
		err = yaml.Unmarshal(marshaledYAML, &reUnmarshaledValue)
		if err != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Failed to re-unmarshal marshaled YAML: %v", testPath, testDescription, err)
		} else if !reflect.DeepEqual(reUnmarshaledValue, expectedUnmarshaledValue) {
			t.Errorf("Test: %s\nDescription: %s\nError: Marshal output mismatch\nExpected: %+v\nGot     : %+v", testPath, testDescription, expectedUnmarshaledValue, reUnmarshaledValue)
		}
	})

	t.Run("JSONComparisonTest", func(t *testing.T) {
		shouldSkipTest(t)

		// Nothing to test if unmarshall is expected to error
		if expectError || !fileExists(testPath, "in.json") {
			return
		}
		inJSON := mustRead(t, testPath, "in.json")

		var unmarshaledValue interface{}
		if err := yaml.Unmarshal(inYAML, &unmarshaledValue); err != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Failed to unmarshal in.yaml: %v", testPath, testDescription, err)
		}

		var jsonValue interface{}

		if err := json.Unmarshal(inJSON, &jsonValue); err != nil {
			t.Errorf(
				"Test: %s\nDescription: %s\nError: Failed to unmarshal in.json: %v",
				testPath, testDescription, err)
		} else if !reflect.DeepEqual(unmarshaledValue, jsonValue) {
			t.Errorf(
				"Test: %s\nDescription: %s\nError: YAML unmarshal vs JSON unmarshal mismatch\n"+
					"Expected: %+v\nGot     : %+v",
				testPath, testDescription, jsonValue, unmarshaledValue)
		}
	})
}
