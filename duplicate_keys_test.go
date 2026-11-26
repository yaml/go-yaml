// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"runtime"
	"strconv"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v4"
)

// duplicateKeysDoc returns a mapping that repeats the key "a" n times.
func duplicateKeysDoc(n int) []byte {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString("a: " + strconv.Itoa(i) + "\n")
	}
	return []byte(b.String())
}

func TestUniqueKeysDuplicatesAreLinear(t *testing.T) {
	const n = 20000
	var m map[string]int

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	err := yaml.Load(duplicateKeysDoc(n), &m, yaml.WithUniqueKeys(true))

	runtime.ReadMemStats(&after)
	if err == nil {
		t.Fatal("expected an error for duplicate keys")
	}
	// One error per repeated key, not one per pair of equal keys.
	if got, want := strings.Count(err.Error(), "already defined"), n-1; got != want {
		t.Fatalf("got %d errors, want %d", got, want)
	}
	// The quadratic version allocated hundreds of MB for this input.
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > 64<<20 {
		t.Fatalf("allocated %d MB while reporting duplicate keys", alloc>>20)
	}
}

func TestUniqueKeysReportsFirstOccurrence(t *testing.T) {
	var m map[string]int
	err := yaml.Load([]byte("a: 1\nb: 2\na: 3\na: 4\n"), &m, yaml.WithUniqueKeys(true))
	if err == nil {
		t.Fatal("expected an error for duplicate keys")
	}
	want := []string{
		`line 3: mapping key "a" already defined at line 1`,
		`line 4: mapping key "a" already defined at line 1`,
	}
	for _, w := range want {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error %q does not contain %q", err, w)
		}
	}
}
