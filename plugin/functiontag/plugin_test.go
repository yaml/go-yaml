// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package functiontag

import "testing"

func TestNoReservedAliasLibrary(t *testing.T) {
	registry.RLock()
	_, exists := registry.libraries["alias-data"]
	registry.RUnlock()
	if exists {
		t.Fatal("functiontag must not register an alias-data library")
	}
	if err := RegisterLibrary("alias-data", Functions{
		"custom": func(value string) string { return value },
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		registry.Lock()
		delete(registry.libraries, "alias-data")
		registry.Unlock()
	})
	p, err := New(nil, map[string]any{"user": "alias-data"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.lookup("user/custom"); err != nil {
		t.Fatal(err)
	}
}

func TestNewRejectsMultipleLibraryMaps(t *testing.T) {
	if _, err := New(nil, nil, nil); err == nil {
		t.Fatal("expected an error for multiple library maps")
	}
}
