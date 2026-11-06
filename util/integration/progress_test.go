// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlainProgress(t *testing.T) {
	var output bytes.Buffer
	p := newProgress(&output, false)
	p.Begin("compose/baseline/prepare")
	_, _ = p.Write([]byte("quiet output remains in the log\n"))
	readEvents(strings.NewReader("start\tunit\nend\tunit\tpass\t\t1250\t0\n"+
		"end\tcontainer\tsetup-needed\tneeds setup\t0\t-1\n"), p, "compose/baseline")
	p.Close()
	got := output.String()
	for _, want := range []string{
		"RUN compose/baseline/unit", "PASS compose/baseline/unit",
		"1.3s  exit=0", "SETUP-NEEDED compose/baseline/container", "exit=n/a",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "\x1b") || strings.Contains(got, "quiet output") {
		t.Fatalf("plain progress emitted terminal controls or quiet output: %q", got)
	}
}

func TestLiveProgressKeepsTenOutputLines(t *testing.T) {
	var output bytes.Buffer
	p := &progress{out: &output, live: true, color: true, width: 80, height: 24}
	p.Begin("yq/candidate/go-tests")
	for i := 1; i <= 15; i++ {
		_, _ = fmt.Fprintf(p, "line-%02d\n", i)
	}
	_, _ = p.Write([]byte("partial line"))
	p.mu.Lock()
	p.draw()
	rows := p.rows
	p.mu.Unlock()
	code := 0
	p.Complete("yq/candidate/go-tests", "pass", 1500*time.Millisecond, &code, "")
	got := output.String()
	if rows != 11 || strings.Contains(got, "line-05") || !strings.Contains(got, "line-15") ||
		!strings.Contains(got, "partial line") || !strings.Contains(got, "\x1b[11A\r\x1b[J") {
		t.Fatalf("live output was not bounded/cleared: rows=%d, %q", rows, got)
	}
	if !strings.Contains(got, "\x1b[32mPASS\x1b[0m yq/candidate/go-tests  1.5s  exit=0") {
		t.Fatalf("permanent colored result missing: %q", got)
	}
}

func TestPreviewSanitizesAndClips(t *testing.T) {
	if got := cleanLine("\x1b[31mred\x1b[0m\x1b]0;title\a\r"); got != "red " {
		t.Fatalf("terminal controls retained: %q", got)
	}
	if got := clipLine("漢a字", 3); got != "漢a" {
		t.Fatalf("wide output wraps: %q", got)
	}
	if got := clipLine("e\u0301x", 1); got != "e\u0301" {
		t.Fatalf("combining marks clipped incorrectly: %q", got)
	}
}

func TestVerboseProgressStreamsOutput(t *testing.T) {
	var output bytes.Buffer
	p := newProgress(&output, true)
	p.Begin("unit")
	_, _ = p.Write([]byte("complete output\n"))
	code := 7
	p.Complete("unit", "fail", time.Second, &code, "")
	p.Close()
	if got := output.String(); !strings.Contains(got, "complete output\nFAIL unit") ||
		!strings.Contains(got, "exit=7") || strings.Contains(got, "\x1b") {
		t.Fatalf("verbose output changed: %q", got)
	}
}

func TestShellCheckEventsAndMetrics(t *testing.T) {
	r, script, repository, commit := fixture(t)
	writeFile(t, script, `#!/usr/bin/env bash
source "$INTEGRATION_ROOT/util/integration/common.bash"
if [[ $1 == test ]]; then
  check failed 'sleep 1.05; exit 7'
  check later 'true'
  skip unavailable 'No Docker daemon'
  finish
fi
`)
	var output bytes.Buffer
	r.progress = newProgress(&output, false)
	checks := r.runPhase(context.Background(), script, repository,
		filepath.Join(t.TempDir(), "source"), commit, "baseline", "fixture", r.logs)
	r.progress.Close()
	if len(checks) != 3 || checks[0].ExitCode == nil || *checks[0].ExitCode != 7 ||
		checks[0].DurationMS == 0 || checks[1].Status != "pass" || checks[2].ExitCode != nil {
		t.Fatalf("shell metrics missing: %+v", checks)
	}
	for _, want := range []string{
		"RUN fixture/baseline/failed", "FAIL fixture/baseline/failed",
		"exit=7", "PASS fixture/baseline/later", "SKIP fixture/baseline/unavailable",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("missing live event %q in %q", want, output.String())
		}
	}
	read, err := parseCheck([]string{"old", "pass", "historical record"})
	if err != nil || read.Name != "old" {
		t.Fatalf("old result records no longer readable: %+v, %v", read, err)
	}
}
