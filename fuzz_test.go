// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
	"go.yaml.in/yaml/v4/internal/testutil/datatest"
)

// FuzzEncodeFromJSON checks that any JSON encoded value can also be encoded as YAML... and decoded.
func FuzzEncodeFromJSON(f *testing.F) {
	// Load seed corpus from testdata YAML file
	cases, err := datatest.LoadTestCasesFromFile("testdata/fuzz_json_roundtrip.yaml", libyaml.LoadYAML)
	if err != nil {
		f.Fatalf("Failed to load seed corpus: %v", err)
	}

	// Add each seed to the fuzz corpus
	for _, tc := range cases {
		if jsonInput, ok := datatest.GetString(tc, "json"); ok {
			f.Add(jsonInput)
		}
	}

	f.Fuzz(func(t *testing.T, s string) {
		var v any
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			t.Skipf("not valid JSON %q", s)
		}

		t.Logf("JSON %q", s)
		t.Logf("Go   %q <%[1]x>", v)

		// Encode as YAML
		b, err := yaml.Marshal(v)
		if err != nil {
			t.Error(err)
		}
		t.Logf("YAML %q <%[1]x>", b)

		// Decode as YAML
		var v2 any
		if err := yaml.Unmarshal(b, &v2); err != nil {
			t.Error(err)
		}

		t.Logf("Go   %q <%[1]x>", v2)

		b2, err := yaml.Marshal(v2)
		if err != nil {
			t.Error(err)
		}
		t.Logf("YAML %q <%[1]x>", b2)

		if !bytes.Equal(b, b2) {
			t.Errorf("Marshal->Unmarshal->Marshal mismatch:\n- expected: %q\n- got:      %q", b, b2)
		}
	})
}
