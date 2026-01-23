// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// YAML Test Suite compliance tests.
// Runs official YAML test suite cases and tracks known failures.

package yts

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
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
	t.Helper()

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
or just run the tests with 'make test'.`, testDir)
	}
	runTestsInDir(t, testDir, "")
}

func runTestsInDir(t *testing.T, rootDir string, relPath string) {
	t.Helper()

	fullPath := filepath.Join(rootDir, relPath)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		t.Fatalf("Failed to read directory %s: %v", fullPath, err)
	}

	for _, entry := range entries {
		entryRelPath := filepath.Join(relPath, entry.Name())
		entryFullPath := filepath.Join(rootDir, entryRelPath)

		if entry.IsDir() {
			// Check if it's a test case directory (contains in.yaml)
			if _, err := os.Stat(filepath.Join(entryFullPath, "in.yaml")); err == nil {
				t.Run(filepath.ToSlash(entryRelPath), func(t *testing.T) {
					runTest(t, entryFullPath)
				})
			} else {
				// Otherwise, recurse into the subdirectory
				runTestsInDir(t, rootDir, entryRelPath)
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
	return libyaml.ParserGetEvents(in)
}

// loadFirstDocument loads only the first document from YAML input.
// Returns nil value and nil error for empty input (0 documents).
// Ignores additional documents if present.
func loadFirstDocument(in []byte, out any) error {
	loader, err := yaml.NewLoader(bytes.NewReader(in))
	if err != nil {
		return err
	}
	err = loader.Load(out)
	if err == io.EOF {
		return nil // Empty input is OK for YTS
	}
	return err
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
		t.Fatalf("Test: %s\nDescription: %s\nError: Failed to read in.yaml: %v", testPath, testDescription, err)
	}

	errorPath := filepath.Join(testPath, "error")
	_, err = os.Stat(errorPath)
	expectError := err == nil

	var unmarshaledValue any
	var unmarshalErr error

	t.Run("LoadTest", func(t *testing.T) {
		shouldSkipTest(t)
		unmarshalErr = loadFirstDocument(inYAML, &unmarshaledValue)

		if expectError {
			if unmarshalErr == nil {
				t.Errorf("Test: %s\nDescription: %s\nError: Expected unmarshal error but got none", testPath, testDescription)
			}
			return
		}
		if unmarshalErr != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Unexpected unmarshal error: %v", testPath, testDescription, unmarshalErr)
		}
	})

	t.Run("EventComparisonTest", func(t *testing.T) {
		shouldSkipTest(t)
		expectedEventsPath := filepath.Join(testPath, "test.event")
		if _, err := os.Stat(expectedEventsPath); err != nil {
			return
		}
		expectedEventsBytes, err := os.ReadFile(expectedEventsPath)
		if err != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Failed to read test.event: %v", testPath, testDescription, err)
			return
		}
		expectedEvents := normalizeLineEndings(string(expectedEventsBytes))
		expectedEvents = strings.TrimSuffix(expectedEvents, "\n")
		actualEvents, eventErr := getEvents(inYAML)

		if expectError {
			if eventErr == nil {
				t.Errorf("Test: %s\nDescription: %s\nError: Expected error on event parsing but got none", testPath, testDescription)
			}
			return
		}
		if eventErr != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Unexpected error on event parsing: %v", testPath, testDescription, eventErr)
			return
		}
		actualEventsStr := normalizeLineEndings(actualEvents)
		if actualEventsStr != expectedEvents {
			t.Errorf("Test: %s\nDescription: %s\nError: Event mismatch\nExpected:\n%q\nGot:\n%q", testPath, testDescription, expectedEvents, actualEventsStr)
		}
	})

	// Only proceed with marshal and JSON tests if unmarshal was successful and no expected error

	t.Run("MarshalTest", func(t *testing.T) {
		shouldSkipTest(t)
		var currentUnmarshaledValue any

		currentUnmarshalErr := loadFirstDocument(inYAML, &currentUnmarshaledValue)

		if !(currentUnmarshalErr == nil || expectError) {
			return
		}
		marshaledYAML, marshalErr := yaml.Dump(currentUnmarshaledValue)
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
		err = loadFirstDocument(expectedOutYAML, &expectedUnmarshaledValue)
		if err != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Failed to unmarshal out.yaml: %v", testPath, testDescription, err)
			return
		}
		var reUnmarshaledValue any
		err = loadFirstDocument(marshaledYAML, &reUnmarshaledValue)
		if err != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Failed to re-unmarshal marshaled YAML: %v", testPath, testDescription, err)
		} else if !reflect.DeepEqual(reUnmarshaledValue, expectedUnmarshaledValue) {
			t.Errorf("Test: %s\nDescription: %s\nError: Marshal output mismatch\nExpected: %+v\nGot     : %+v", testPath, testDescription, expectedUnmarshaledValue, reUnmarshaledValue)
		}
	})

	t.Run("JSONComparisonTest", func(t *testing.T) {
		shouldSkipTest(t)
		var currentUnmarshaledValue any

		currentUnmarshalErr := loadFirstDocument(inYAML, &currentUnmarshaledValue)

		if !(currentUnmarshalErr == nil || expectError) {
			return
		}
		inJSONPath := filepath.Join(testPath, "in.json")
		if _, err := os.Stat(inJSONPath); err != nil {
			return
		}
		inJSON, err := os.ReadFile(inJSONPath)
		if err != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Failed to read in.json: %v", testPath, testDescription, err)
			return
		}
		var jsonValue any
		jsonErr := json.Unmarshal(inJSON, &jsonValue)
		if jsonErr != nil {
			t.Errorf("Test: %s\nDescription: %s\nError: Failed to unmarshal in.json: %v", testPath, testDescription, jsonErr)
		} else if !reflect.DeepEqual(currentUnmarshaledValue, jsonValue) {
			t.Errorf("Test: %s\nDescription: %s\nError: YAML unmarshal vs JSON unmarshal mismatch\nExpected: %+v\nGot     : %+v", testPath, testDescription, jsonValue, currentUnmarshaledValue)
		}
	})
}
