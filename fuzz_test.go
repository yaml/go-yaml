//go:build go1.18
// +build go1.18

package yaml_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.yaml.in/yaml/v3"
)

// FuzzEncodeFromJSON checks that any JSON encoded value can also be encoded as YAML... and decoded.
func FuzzEncodeFromJSON(f *testing.F) {
	f.Add(`null`)
	f.Add(`""`)
	f.Add(`0`)
	f.Add(`true`)
	f.Add(`false`)
	f.Add(`{}`)
	f.Add(`[]`)
	f.Add(`[[]]`)
	f.Add(`{"a":[]}`)
	f.Add(`-0`)
	f.Add(`-0.000`)
	f.Add(`"\n"`)
	f.Add(`"\t"`)

	f.Fuzz(func(t *testing.T, s string) {

		var v interface{}
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			t.Skip("not valid JSON")
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
		var v2 interface{}
		if err := yaml.Unmarshal(b, &v2); err != nil {
			t.Error(err)
		}

		t.Logf("Go   %q <%[1]x>", v2)

		/*
			// Handling of number is different, so we can't have universal exact matching
			if !reflect.DeepEqual(v2, v) {
				t.Errorf("mismatch:\n-      got: %#v\n- expected: %#v", v2, v)
			}
		*/

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
