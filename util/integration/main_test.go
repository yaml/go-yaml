// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogSelectionAndSkippedReporting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.yaml")
	writeFile(t, path, `active:
  name: Active
  repo: example/active
  vref: main
  star: 1234
omitted:
  name: Omitted
  repo: example/omitted
  vref: master
  star: 2345
  skip: true
`)
	c, err := readCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := selectProjects(c, "omitted,active,active")
	if err != nil || !reflect.DeepEqual(ids, []string{"active", "omitted"}) {
		t.Fatalf("selection: %v, %v", ids, err)
	}
	if _, err := selectProjects(c, "unknown"); err == nil {
		t.Fatal("unknown project accepted")
	}
	report := renderReport(c)
	if !strings.Contains(report, "Omitted") || !strings.Contains(report, "Skipped") {
		t.Fatal("report omitted skipped project")
	}
	// A skipped project must not look for a script, fetch, or provision tools.
	r := runner{root: "/nonexistent", out: io.Discard}
	result := r.runProject(context.Background(), "omitted", c["omitted"])
	if result.Status != "skipped" || result.Logs != "" {
		t.Fatalf("skipped project performed setup: %+v", result)
	}
	if !strings.Contains(renderSummary([]projectResult{result}, 0, 1), "ZERO COVERAGE") {
		t.Fatal("zero coverage hidden")
	}
	writeFile(t, path, "typo:\n  repo: example/typo\n  vref: main\n  skpi: true\n")
	if _, err := readCatalog(path); err == nil {
		t.Fatal("unknown catalog key accepted")
	}
}

func TestCheckClassification(t *testing.T) {
	check := func(status string) []checkResult {
		return []checkResult{{Name: "unit", Status: status}}
	}
	for _, tt := range []struct{ baseline, candidate, want string }{
		{"pass", "pass", "pass"},
		{"pass", "fail", "upgrade-blocked"},
		{"fail", "fail", "baseline-failed"},
		{"fail", "pass", "baseline-failed"},
		{"setup-needed", "pass", "partial"},
		{"setup-needed", "setup-needed", "no-coverage"},
		{"setup-needed", "fail", "candidate-failed"},
		{"setup-error", "pass", "setup-error"},
		{"pass", "setup-error", "setup-error"},
		{"pass", "skip", "partial"},
		{"skip", "skip", "no-coverage"},
	} {
		t.Run(tt.baseline+"/"+tt.candidate, func(t *testing.T) {
			if got := classify(check(tt.baseline), check(tt.candidate)); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
	// Unrelated baseline setup gaps must not hide a proven upgrade regression.
	baseline := append(check("pass"), checkResult{Name: "cloud", Status: "setup-needed"})
	if got := classify(baseline, check("fail")); got != "upgrade-blocked" {
		t.Fatalf("regression hidden: %s", got)
	}
	if got := classify(check("pass"), nil); got != "setup-error" {
		t.Fatalf("missing candidate check accepted: %s", got)
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

// This fixture uses real Git clones and Go module resolution without network
// access. The published module passes, while replacing it exposes a regression.
func fixture(t *testing.T) (runner, string, string, string) {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module go.yaml.in/yaml/v4\n\ngo 1.18\n")
	writeFile(t, filepath.Join(root, "yaml.go"), "package yaml\nfunc Value() string { return \"candidate\" }\n")
	baseline := filepath.Join(t.TempDir(), "published")
	writeFile(t, filepath.Join(baseline, "go.mod"), "module go.yaml.in/yaml/v4\n\ngo 1.18\n")
	writeFile(t, filepath.Join(baseline, "yaml.go"), "package yaml\nfunc Value() string { return \"published\" }\n")
	source := t.TempDir()
	writeFile(t, filepath.Join(source, "go.mod"), "module example.test/downstream\n\ngo 1.18\n\n"+
		"require go.yaml.in/yaml/v4 v4.0.0\nreplace go.yaml.in/yaml/v4 => "+filepath.ToSlash(baseline)+"\n")
	writeFile(t, filepath.Join(source, "go.work"), "intentionally invalid upstream workspace\n")
	writeFile(t, filepath.Join(source, "downstream_test.go"), `package downstream
import (
  "testing"
  "go.yaml.in/yaml/v4"
)
func TestCompatibility(t *testing.T) {
  if yaml.Value() != "published" { t.Fatal("upgrade changed behavior") }
}
`)
	git(t, source, "init", "-q")
	git(t, source, "add", ".")
	git(t, source, "-c", "user.name=Integration Fixture", "-c", "user.email=fixture@example.test", "commit", "-qm", "Fixture")
	commit := git(t, source, "rev-parse", "HEAD")
	repository := filepath.Join(t.TempDir(), "downstream.git")
	git(t, source, "clone", "--bare", "-q", source, repository)
	common, err := os.ReadFile("common.bash")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "util/integration/common.bash"), string(common))
	script := filepath.Join(root, "util/integration/fixture.bash")
	writeFile(t, script, `#!/usr/bin/env bash
source "$INTEGRATION_ROOT/util/integration/common.bash"
case $1 in
  prepare) echo prepared > prepared ;;
  test)
    check preparation 'test "$(cat prepared)" = prepared'
    check compatibility 'go test -count=1 ./...'
    check later 'touch generated'
    finish
    ;;
esac
`)
	logs := t.TempDir()
	r := runner{root: root, cache: t.TempDir(), logs: logs, timeout: time.Minute, out: io.Discard}
	return r, script, repository, commit
}

func TestRealBaselineAndCandidate(t *testing.T) {
	r, script, repository, commit := fixture(t)
	work := t.TempDir()
	ctx := context.Background()
	baseline := r.runPhase(ctx, script, repository, filepath.Join(work, "baseline"), commit, "baseline", "fixture", r.logs)
	candidate := r.runPhase(ctx, script, repository, filepath.Join(work, "candidate"), commit, "candidate", "fixture", r.logs)
	if got := classify(baseline, candidate); got != "upgrade-blocked" {
		data, _ := os.ReadFile(filepath.Join(r.logs, "candidate.log"))
		t.Fatalf("got %s\nbaseline: %+v\ncandidate: %+v\n%s", got, baseline, candidate, data)
	}
	if len(candidate) != 3 || candidate[2].Name != "later" || candidate[2].Status != "pass" {
		t.Fatalf("later check did not execute: %+v", candidate)
	}
	for _, phase := range []string{"baseline", "candidate"} {
		p := filepath.Join(work, phase)
		if _, err := os.Stat(filepath.Join(p, "generated")); err != nil {
			t.Fatal(err)
		}
		got := git(t, p, "diff", "--", "go.mod", "go.work")
		if got != "" {
			t.Fatalf("%s module files edited: %s", phase, got)
		}
	}
}

func TestTruncatedScriptIsAnError(t *testing.T) {
	r, script, repository, commit := fixture(t)
	writeFile(t, script, `#!/usr/bin/env bash
source "$INTEGRATION_ROOT/util/integration/common.bash"
if [[ $1 == test ]]; then
  check failed 'false'
  exit 1
fi
`)
	checks := r.runPhase(context.Background(), script, repository, filepath.Join(t.TempDir(), "source"), commit, "baseline", "fixture", r.logs)
	if len(checks) != 2 || checks[1].Status != "setup-error" {
		t.Fatalf("truncated script accepted: %+v", checks)
	}
}

func TestPendingCoverageDoesNotFailProject(t *testing.T) {
	r, script, repository, commit := fixture(t)
	writeFile(t, script, `#!/usr/bin/env bash
source "$INTEGRATION_ROOT/util/integration/common.bash"
if [[ $1 == test ]]; then
  check unit 'true'
  pending ci-security 'Security workflow needs translation'
  finish
fi
`)
	if err := os.Rename(repository, filepath.Join(r.cache, "fixture.git")); err != nil {
		t.Fatal(err)
	}
	result := r.runProject(context.Background(), "fixture", project{
		Repository: "example/unused", Ref: commit,
	})
	if result.Status != "partial" || result.ExitCode != 0 {
		t.Fatalf("pending coverage failed the project: %+v", result)
	}
	for _, checks := range [][]checkResult{result.Baseline, result.Candidate} {
		if len(checks) != 2 || checks[0].Status != "pass" ||
			checks[1].Status != "setup-needed" || checks[1].ExitCode != nil {
			t.Fatalf("pending coverage lost or treated as executed: %+v", checks)
		}
	}
}

func TestPinnedCacheWithoutBranchRefs(t *testing.T) {
	r, _, repository, commit := fixture(t)
	cache := filepath.Join(r.cache, "fixture.git")
	if err := os.Rename(repository, cache); err != nil {
		t.Fatal(err)
	}
	refs := git(t, r.root, "--git-dir="+cache, "for-each-ref", "--format=%(refname)")
	for _, ref := range strings.Fields(refs) {
		git(t, r.root, "--git-dir="+cache, "update-ref", "-d", ref)
	}
	result := r.runProject(context.Background(), "fixture", project{
		Repository: "example/unused", Ref: commit,
	})
	if result.Status != "upgrade-blocked" || result.Commit != commit {
		t.Fatalf("cached commit without branch refs: %+v", result)
	}
	entries, err := os.ReadDir(r.cache)
	if err != nil || len(entries) != 1 || entries[0].Name() != "fixture.git" {
		t.Fatalf("disposable source not removed: %v, %v", entries, err)
	}
}

func TestTimeoutStopsDescendants(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := runCommand(ctx, exec.Command("bash", "-c", "sleep 60 & wait"))
	if err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("timeout failed to stop descendants: %v", err)
	}
}
