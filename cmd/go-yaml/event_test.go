// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"strings"
	"testing"
)

// Regression test for issue #247: when Dump fails, the error returned
// by the Dumper's Close method must be reported alongside the Dump
// error instead of being discarded.
func TestDumpEventInfoReportsCloseErrorOnDumpFailure(t *testing.T) {
	// A func value cannot be represented as YAML, so Dump fails.
	// Closing the Dumper afterwards fails as well, because the stream
	// was never opened; that second error used to be silently dropped.
	_, err := dumpEventInfo("event info", func() {})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot represent type") {
		t.Errorf("error does not mention the dump failure: %v", err)
	}
	if !strings.Contains(err.Error(), "failed to close dumper") {
		t.Errorf("error does not mention the close failure; the Close error was discarded: %v", err)
	}
}

func TestDumpEventInfoSuccess(t *testing.T) {
	out, err := dumpEventInfo("event info", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("output %q does not contain the dumped value", out)
	}
}
