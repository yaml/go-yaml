package yaml_test

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestOptsYAML(t *testing.T) {
	tests := []struct {
		name      string
		yamlStr   string
		expectErr bool
		errMatch  string
	}{
		{
			name:      "valid options",
			yamlStr:   "indent: 4\nknown-fields: true",
			expectErr: false,
		},
		{
			name:      "typo in field name",
			yamlStr:   "knnown-fields: true",
			expectErr: true,
			errMatch:  "knnown-fields not found",
		},
		{
			name:      "another typo",
			yamlStr:   "indnt: 2",
			expectErr: true,
			errMatch:  "indnt not found",
		},
		{
			name:      "multiple options with one typo",
			yamlStr:   "indent: 2\nunicoode: true",
			expectErr: true,
			errMatch:  "unicoode not found",
		},
		{
			name: "all valid options",
			yamlStr: `
indent: 2
compact-seq-indent: true
line-width: 80
unicode: true
canonical: false
line-break: ln
explicit-start: true
explicit-end: false
flow-simple-coll: true
known-fields: true
single-document: true
unique-keys: true
`,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt, err := yaml.OptsYAML(tt.yamlStr)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				if tt.errMatch != "" && !strings.Contains(err.Error(), tt.errMatch) {
					t.Errorf("expected error to contain %q, got: %v", tt.errMatch, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if opt == nil {
					t.Fatal("expected non-nil option")
				}
			}
		})
	}
}
