// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestMarshalTruncatedUTF8DoesNotPanic(t *testing.T) {
	inputs := []string{
		string([]byte{0xC2}),
		string([]byte{'x', 0xC2}),
		string([]byte{0xEF, 0xBB}),
		string([]byte{0xE2, 0x80}),
		string([]byte{0xED}),
	}
	for i, in := range inputs {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			defer func() {
				if rec := recover(); rec != nil {
					t.Fatalf("panic: %v", rec)
				}
			}()
			_, err := yaml.Marshal(in)
			if err != nil {
				t.Logf("marshal error (ok): %v", err)
			}
		})
	}
}
