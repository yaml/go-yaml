package yts

import (
	"bufio"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

const (
	testDir        = "testdata/data-2022-01-17"
	knownTestsFile = "known-failing-tests"
)

var (
	knownFailingTests          map[string]bool
	knownFailingButPassedTests = make(map[string]struct{})
)

func loadKnownFailingTests(tb testing.TB) {
	tb.Helper()

	file, err := os.Open(knownTestsFile)
	assert.NoErrorf(tb, err, "failed to open known-failing-tests file")
	defer func() {
		err := file.Close()
		assert.NoErrorf(tb, err, "failed to close known-failing-tests file")
	}()

	scanner := bufio.NewScanner(file)
	knownTests := make(map[string]bool)
	for scanner.Scan() {
		trimmedLine := strings.TrimSpace(scanner.Text())
		if trimmedLine != "" {
			knownTests[trimmedLine] = false
		}
	}
	knownFailingTests = knownTests
}

func shouldSkipTest(tb testing.TB) {
	tb.Helper()

	_, isKnownFailing := knownFailingTests[tb.Name()]
	if isKnownFailing {
		knownFailingTests[tb.Name()] = true
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
	loadKnownFailingTests(t)

	if !runTestsInDir(t, testDir) {
		t.Errorf(
			`YTS tests require data files to be present at %q. Run 'make test-data' to download them first,
or just run the tests with 'make test-all'.`,
			testDir,
		)
	}

	t.Run("CheckKnownFailingTests", func(t *testing.T) {
		var list []string
		for testName, ran := range knownFailingTests {
			if !ran {
				list = append(list, testName)
			}
		}
		if len(list) > 0 {
			sort.Strings(list)
			t.Fatalf(
				"The following known failing tests did not run; please remove them from %q:\n%s",
				knownTestsFile,
				strings.Join(list, "\n"),
			)
		}
	})

	t.Run("CheckKnownFailingButPassedTests", func(t *testing.T) {
		var list []string
		for testName := range knownFailingButPassedTests {
			list = append(list, testName)
		}
		if len(list) > 0 {
			sort.Strings(list)
			t.Fatalf(
				"The following known failing tests passed; please remove them from %q:\n%s",
				knownTestsFile,
				strings.Join(list, "\n"),
			)
		}
	})
}

func runTestsInDir(t *testing.T, dirPath string) bool {
	t.Helper()

	// Track if any tests were found in this directory or its subdirectories
	hasTests := false

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		t.Fatalf("Failed to read directory %q: %v", dirPath, err)
	}

	for _, entry := range entries {
		switch {
		// skip hidden files/directories, like .git
		case strings.HasPrefix(entry.Name(), "."):
			fallthrough
		// skip special folders
		case entry.Name() == "name", entry.Name() == "tags":
			continue
		}
		entryPath := filepath.Join(dirPath, entry.Name())
		if entry.IsDir() {
			t.Run(entry.Name(), func(t *testing.T) {
				// Check if it's a test case directory (contains in.yaml)
				if fileExists(entryPath, "in.yaml") {
					runTest(t, entryPath)
					hasTests = true
				} else {
					// Otherwise, recurse into the subdirectory
					hasTests = runTestsInDir(t, entryPath) || hasTests
				}
			})
		}
	}

	return hasTests
}

func fileExists(elem ...string) bool {
	_, err := os.Stat(path.Join(elem...))
	return err == nil
}

func readFile(tb testing.TB, must bool, elem ...string) []byte {
	tb.Helper()
	filename := path.Join(elem...)
	data, err := os.ReadFile(filename)
	if err != nil && must {
		tb.Fatalf("Failed to read file %q: %v", filename, err)
	}
	return data
}

func normalizeLineEndings(s string) string {
	return strings.NewReplacer(
		"\r", "",
	).Replace(s)
}

func runTest(t *testing.T, testPath string) {
	t.Helper()
	
	// Read test description
	testDescription := string(readFile(t, false, testPath, "==="))
	if testDescription == "" {
		testDescription = "No description available."
	}

	t.Logf("Running test: %s\nDescription: %s", testPath, testDescription)

	inYAML := readFile(t, true, testPath, "in.yaml")
	expectError := fileExists(testPath, "error")

	var unmarshaledValue any
	var unmarshalErr error

	t.Run("UnmarshalTest", func(t *testing.T) {
		t.Logf("Running test: %s\nDescription: %s", testPath, testDescription)
		shouldSkipTest(t)
		unmarshalErr = yaml.Unmarshal(inYAML, &unmarshaledValue)

		if expectError {
			t.Logf("Got error: %v", unmarshalErr)
			assert.NotNilf(t, unmarshalErr, "Expected unmarshal error but got none:\n%+v", unmarshaledValue)
			return
		}
		assert.NoErrorf(t, unmarshalErr, "Failed to unmarshal YAML:\n%s", string(inYAML))
	})

	t.Run("EventComparisonTest", func(t *testing.T) {
		t.Logf("Running test: %s\nDescription: %s", testPath, testDescription)
		shouldSkipTest(t)

		expectedEvents := string(readFile(t, true, testPath, "test.event"))
		expectedEvents = normalizeLineEndings(expectedEvents)
		expectedEvents = strings.TrimSpace(expectedEvents)

		actualEvents, eventErr := yaml.ParserGetEvents(inYAML)
		if expectError {
			t.Logf("Got error: %v", eventErr)
			assert.NotNilf(t, eventErr, "Expected unmarshal error but got none")
			return
		}
		assert.NoErrorf(t, eventErr, "Failed to parse events for yaml:\n%s", string(inYAML))

		actualEvents = normalizeLineEndings(actualEvents)
		assert.Equalf(t, expectedEvents, actualEvents, "Event mismatch")
	})

	// Only proceed with marshal and JSON tests if unmarshal was successful and no expected error
	if unmarshalErr == nil && !expectError {
		return
	}

	if fileExists(testPath, "out.yaml") {
		t.Run("MarshalTest", func(t *testing.T) {
			t.Logf("Running test: %s\nDescription: %s", testPath, testDescription)
			shouldSkipTest(t)

			expectedOutYAML := readFile(t, true, testPath, "out.yaml")

			marshaledYAML, marshalErr := yaml.Marshal(unmarshaledValue)
			assert.NoErrorf(t, marshalErr, "Failed to marshal value:\n%+v", unmarshaledValue)

			var expectedUnmarshaledValue any
			err := yaml.Unmarshal(expectedOutYAML, &expectedUnmarshaledValue)
			assert.NoErrorf(t, err, "Failed to unmarshal YAML:\n%s", string(expectedOutYAML))

			var reUnmarshaledValue any
			err = yaml.Unmarshal(marshaledYAML, &reUnmarshaledValue)
			assert.NoErrorf(t, err, "Failed to re-unmarshal YAML:\n%s", string(marshaledYAML))

			assert.DeepEqual(t, expectedUnmarshaledValue, reUnmarshaledValue)
		})
	}

	if fileExists(testPath, "in.json") {
		t.Run("JSONComparisonTest", func(t *testing.T) {
			t.Logf("Running test: %s\nDescription: %s", testPath, testDescription)
			shouldSkipTest(t)

			inJSON := readFile(t, true, testPath, "in.json")

			var jsonValue any
			jsonErr := json.Unmarshal(inJSON, &jsonValue)
			assert.NoErrorf(t, jsonErr, "Failed to unmarshal in.json:\n%s", string(inJSON))

			// Comparing the unmarshaled YAML value with the JSON value fails
			// due to differences in how YAML and JSON handle number types.
			// JSON unmarshals all numbers as float64, while YAML preserves
			// integer types when possible. To work around this, we marshal
			// both values to JSON and compare the results.

			yamlAsJSONText, err := json.Marshal(unmarshaledValue)
			assert.NoErrorf(t, err, "Failed to marshal as JSON:\n%+v", unmarshaledValue)

			jsonAsText, err := json.Marshal(jsonValue)
			assert.NoErrorf(t, err, "Failed to marshal as JSON:\n%+v", jsonValue)

			assert.Equalf(t, string(jsonAsText), string(yamlAsJSONText), "YAML unmarshal vs JSON unmarshal mismatch")
		})
	}
}
