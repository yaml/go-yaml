package yaml_test

import (
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestParserGetEvents(t *testing.T) {
	for _, tc := range []struct {
		in  string
		exp string
	}{
		// ImplicitDocumentStart
		{
			in: `a: b`,
			exp: `+STR
+DOC
+MAP
=VAL :a
=VAL :b
-MAP
-DOC
-STR`,
		},
		// ExplicitDocumentStart
		{
			in: `---
a: b`,
			exp: `+STR
+DOC ---
+MAP
=VAL :a
=VAL :b
-MAP
-DOC
-STR`,
		},
	} {
		t.Run(tc.in, func(t *testing.T) {
			events, err := yaml.ParserGetEvents([]byte(tc.in))
			if err != nil {
				t.Fatalf("ParserGetEvents error: %v", err)
			}
			assert.Equal(t, tc.exp, events)
		})
	}
}
