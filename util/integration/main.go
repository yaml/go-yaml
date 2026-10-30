// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// The integration utility reads the project catalog. Project-specific behavior
// belongs in the adjacent Bash scripts, not in the catalog or this runner.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v4"
)

type project struct {
	Name       string `yaml:"name"`
	Repository string `yaml:"repo"`
	Ref        string `yaml:"vref"`
	Stars      int    `yaml:"star"`
	Notes      string `yaml:"note,omitempty"`
	Time       string `yaml:"time,omitempty"`
	Skip       bool   `yaml:"skip,omitempty"`
}

type catalog map[string]project

type checkResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Detail     string `json:"detail,omitempty"`
	DurationMS int64  `json:"duration_ms"`
	ExitCode   *int   `json:"exit_code"`
}

type projectResult struct {
	Project    string        `json:"project"`
	Repository string        `json:"repository"`
	Ref        string        `json:"ref"`
	Commit     string        `json:"commit,omitempty"`
	Status     string        `json:"status"`
	Detail     string        `json:"detail,omitempty"`
	Baseline   []checkResult `json:"baseline,omitempty"`
	Candidate  []checkResult `json:"candidate,omitempty"`
	Logs       string        `json:"logs,omitempty"`
	DurationMS int64         `json:"duration_ms"`
	ExitCode   int           `json:"exit_code"`
}

type runner struct {
	root     string
	cache    string
	logs     string
	timeout  time.Duration
	verbose  bool
	out      io.Writer
	progress *progress
}

var (
	projectID    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	repositoryID = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	commitID     = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

func readCatalog(path string) (catalog, error) {
	var c catalog
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	d := yaml.NewDecoder(bytes.NewReader(data))
	d.KnownFields(true)
	if err = d.Decode(&c); err != nil {
		return c, err
	}
	if len(c) == 0 {
		return c, errors.New("catalog has no projects")
	}
	for id, p := range c {
		if !projectID.MatchString(id) || !repositoryID.MatchString(p.Repository) {
			return c, fmt.Errorf("invalid project or repository: %s", id)
		}
		if p.Ref == "" || strings.HasPrefix(p.Ref, "-") {
			return c, fmt.Errorf("invalid ref for %s", id)
		}
	}
	return c, nil
}

func selectProjects(c catalog, selection string) ([]string, error) {
	var ids []string
	if selection == "" {
		for id := range c {
			ids = append(ids, id)
		}
	} else {
		seen := make(map[string]bool)
		for _, id := range strings.Split(selection, ",") {
			id = strings.TrimSpace(id)
			if _, ok := c[id]; !ok {
				return nil, fmt.Errorf("unknown integration project %q", id)
			}
			if !seen[id] {
				ids = append(ids, id)
				seen[id] = true
			}
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func main() {
	if err := runCLI(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCLI() error {
	config := flag.String("config", "testdata/integration/projects.yaml", "project catalog")
	mode := flag.String("mode", "run", "run, list, matrix, or report")
	selection := flag.String("project", "", "comma-separated project keys (default: all)")
	ref := flag.String("ref", "", "revision override for one selected project")
	report := flag.String("report", "go-yaml-v4-projects.md", "generated Markdown report")
	timeout := flag.Duration("timeout", 30*time.Minute, "timeout per project phase")
	verbose := flag.Bool("v", false, "stream command output and request verbose Go tests")
	flag.Parse()
	if flag.NArg() != 0 || *timeout <= 0 {
		return errors.New("unexpected arguments or invalid timeout")
	}
	c, err := readCatalog(*config)
	if err != nil {
		return err
	}
	ids, err := selectProjects(c, *selection)
	if err != nil {
		return err
	}
	if *ref != "" {
		if len(ids) != 1 || strings.HasPrefix(*ref, "-") {
			return errors.New("-ref requires exactly one selected project and a valid revision")
		}
		p := c[ids[0]]
		p.Ref = *ref
		c[ids[0]] = p
	}
	switch *mode {
	case "list":
		for _, id := range ids {
			p := c[id]
			state := "enabled"
			if p.Skip {
				state = "skipped"
			}
			fmt.Printf("%s\t%s\t%s\t%s\n", id, p.Repository, p.Ref, state)
		}
		return nil
	case "matrix":
		enabled := make([]string, 0, len(ids))
		for _, id := range ids {
			if !c[id].Skip {
				enabled = append(enabled, id)
			}
		}
		return json.NewEncoder(os.Stdout).Encode(enabled)
	case "report":
		return os.WriteFile(*report, []byte(renderReport(c)), 0o644)
	case "run":
	default:
		return fmt.Errorf("unknown mode %q", *mode)
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	r := runner{
		root: root, cache: filepath.Join(root, ".cache/integration"),
		timeout: *timeout, verbose: *verbose, out: os.Stdout,
	}
	r.progress = newProgress(r.out, r.verbose)
	defer r.progress.Close()
	if err := os.MkdirAll(filepath.Join(r.cache, "runs"), 0o755); err != nil {
		return err
	}
	r.logs, err = os.MkdirTemp(filepath.Join(r.cache, "runs"), "run-")
	if err != nil {
		return err
	}
	fmt.Fprintf(r.out, "Results: %s\n", r.logs)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var results []projectResult
	failed, executed, skipped := false, 0, 0
	for _, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		p := c[id]
		if *selection != "" {
			p.Skip = false
		}
		if !p.Skip {
			r.progress.StartProject(id)
		}
		result := r.runProject(ctx, id, p)
		results = append(results, result)
		r.progress.EndProject(result)
		if result.Status == "skipped" {
			skipped++
		}
		for _, checks := range [][]checkResult{result.Baseline, result.Candidate} {
			for _, check := range checks {
				if check.Status == "pass" || check.Status == "fail" {
					executed++
				}
			}
		}
		if result.Status != "pass" && result.Status != "partial" &&
			result.Status != "skipped" && result.Status != "no-coverage" {
			failed = true
		}
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(r.logs, "results.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	summary := renderSummary(results, executed, skipped)
	if err = os.WriteFile(filepath.Join(r.logs, "summary.md"), []byte(summary), 0o644); err != nil {
		return err
	}
	if path := os.Getenv("GITHUB_STEP_SUMMARY"); path != "" {
		if f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644); err == nil {
			_, _ = io.WriteString(f, summary)
			_ = f.Close()
		}
	}
	fmt.Fprintf(r.out, "%d checks executed; %d projects skipped. Results: %s\n", executed, skipped, r.logs)
	if executed == 0 {
		fmt.Fprintln(r.out, "ZERO COVERAGE: no downstream checks executed")
	}
	if failed {
		return errors.New("downstream integration failures or setup errors; see results.json and logs")
	}
	return nil
}

// command saves complete output and reports command and named-check progress.
func (r runner) command(ctx context.Context, dir, log string, env []string, args ...string) error {
	f, err := os.OpenFile(log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "+ %q\n", args)
	p := r.progress
	if p == nil {
		p = newProgress(r.out, r.verbose)
		defer p.Close()
	}
	label := commandLabel(log, args)
	started := time.Now()
	p.Begin(label)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = io.MultiWriter(f, p)
	cmd.Stderr = cmd.Stdout
	var reader, writer *os.File
	var events chan eventSummary
	if runtime.GOOS != "windows" && args[0] == "bash" && len(args) > 1 && strings.HasSuffix(args[1], ".bash") {
		reader, writer, err = os.Pipe()
		if err != nil {
			p.Complete(label, "setup-error", time.Since(started), nil, err.Error())
			return err
		}
		cmd.ExtraFiles = []*os.File{writer}
		cmd.Env = append(cmd.Env, "INTEGRATION_EVENTS_FD=3")
		events = make(chan eventSummary, 1)
		prefix := filepath.Base(filepath.Dir(log)) + "/" + strings.TrimSuffix(filepath.Base(log), ".log")
		for _, entry := range env {
			if strings.HasPrefix(entry, "INTEGRATION_PROJECT=") {
				prefix = strings.TrimPrefix(entry, "INTEGRATION_PROJECT=") + "/" + strings.TrimSuffix(filepath.Base(log), ".log")
			}
		}
		go func() { events <- readEvents(reader, p, prefix) }()
	}
	err = runCommand(ctx, cmd)
	var summary eventSummary
	if events != nil {
		_ = writer.Close()
		select {
		case summary = <-events:
		case <-time.After(250 * time.Millisecond):
			// A detached child may retain the event descriptor after its parent.
			_ = reader.Close()
			summary = <-events
		}
		_ = reader.Close()
	}
	status, detail := "pass", ""
	code := exitCode(err)
	if err == nil && summary.setup > 0 {
		status = "partial"
		detail = fmt.Sprintf("%d checks need setup", summary.setup)
	}
	if err != nil {
		status, detail = "fail", err.Error()
		if summary.setup > 0 && summary.failed == 0 {
			status = "setup-needed"
			detail = fmt.Sprintf("%d checks need setup", summary.setup)
		}
	}
	p.Complete(label, status, time.Since(started), &code, detail)
	if err != nil {
		fmt.Fprintf(f, "Command error: %v\n", err)
		return fmt.Errorf("%s: %w (log: %s)", args[0], err, log)
	}
	return nil
}

func (r runner) runProject(ctx context.Context, id string, p project) (result projectResult) {
	started := time.Now()
	result = projectResult{Project: id, Repository: p.Repository, Ref: p.Ref}
	defer func() {
		result.DurationMS = time.Since(started).Milliseconds()
		if result.Status != "pass" && result.Status != "partial" &&
			result.Status != "no-coverage" && result.Status != "skipped" {
			result.ExitCode = 1
		}
	}()
	if p.Skip {
		result.Status, result.Detail = "skipped", "skip: true"
		return result
	}
	result.Logs = filepath.Join(r.logs, id)
	if err := os.MkdirAll(result.Logs, 0o755); err != nil {
		result.Status, result.Detail = "setup-error", err.Error()
		return result
	}
	setupLog := filepath.Join(result.Logs, "source.log")
	ctx, cancel := context.WithTimeout(ctx, 3*r.timeout)
	defer cancel()
	script := filepath.Join(r.root, "util/integration", id+".bash")
	if _, err := os.Stat(script); err != nil {
		result.Status, result.Detail = "setup-error", err.Error()
		return result
	}
	repository := filepath.Join(r.cache, id+".git")
	if err := r.fetch(ctx, repository, p.Repository, p.Ref, setupLog); err != nil {
		result.Status, result.Detail = "setup-error", err.Error()
		return result
	}
	output, err := exec.Command("git", "--git-dir="+repository, "rev-parse", "--verify", p.Ref+"^{commit}").Output()
	if !commitID.MatchString(p.Ref) {
		output, err = exec.Command("git", "--git-dir="+repository, "rev-parse", "--verify", "FETCH_HEAD^{commit}").Output()
	}
	if err != nil {
		result.Status, result.Detail = "setup-error", "cannot resolve fetched commit: "+err.Error()
		return result
	}
	result.Commit = strings.TrimSpace(string(output))
	// Older caches contain only FETCH_HEAD. Give clone a reachable ref even
	// when reusing a pinned commit without fetching it again.
	if err := r.command(ctx, r.root, setupLog, nil, "git", "--git-dir="+repository,
		"update-ref", "refs/heads/integration", result.Commit); err != nil {
		result.Status, result.Detail = "setup-error", err.Error()
		return result
	}
	fmt.Fprintf(r.out, "%s: %s resolved %s\n", id, p.Ref, result.Commit)
	work, err := os.MkdirTemp(r.cache, id+".")
	if err != nil {
		result.Status, result.Detail = "setup-error", err.Error()
		return result
	}
	defer os.RemoveAll(work)
	for _, phase := range []string{"baseline", "candidate"} {
		source := filepath.Join(work, phase)
		checks := r.runPhase(ctx, script, repository, source, result.Commit, phase, id, result.Logs)
		if phase == "baseline" {
			result.Baseline = checks
		} else {
			result.Candidate = checks
		}
	}
	result.Status = classify(result.Baseline, result.Candidate)
	var passed, failed, setup, skipped int
	for _, checks := range [][]checkResult{result.Baseline, result.Candidate} {
		for _, c := range checks {
			switch c.Status {
			case "pass":
				passed++
			case "fail":
				failed++
			case "setup-error", "setup-needed":
				setup++
			case "skip":
				skipped++
			}
		}
	}
	result.Detail = fmt.Sprintf("%d passed, %d failed, %d need setup, %d skipped", passed, failed, setup, skipped)
	return result
}

func (r runner) fetch(ctx context.Context, path, repository, ref, log string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := r.command(ctx, r.root, log, nil, "git", "init", "--bare", "-q", path); err != nil {
			return err
		}
	}
	if commitID.MatchString(ref) {
		if exec.Command("git", "--git-dir="+path, "cat-file", "-e", ref+"^{commit}").Run() == nil {
			return nil
		}
	}
	return r.command(ctx, r.root, log, nil, "git", "--git-dir="+path,
		"fetch", "--depth=1", "--no-tags", "--force", "https://github.com/"+repository, ref)
}

func (r runner) runPhase(ctx context.Context, script, repository, source, commit, phase, id, logs string) []checkResult {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	log := filepath.Join(logs, phase+".log")
	resultPath := filepath.Join(logs, phase+".tsv")
	setupError := func(err error) []checkResult {
		return []checkResult{{Name: "setup", Status: "setup-error", Detail: err.Error()}}
	}
	for _, args := range [][]string{
		{"git", "clone", "--shared", "--no-checkout", "-q", repository, source},
		{"git", "-C", source, "checkout", "-q", "--detach", commit},
	} {
		if err := r.command(ctx, r.root, log, nil, args...); err != nil {
			return setupError(err)
		}
	}
	env := []string{
		"GOWORK=off", "INTEGRATION_ROOT=" + r.root,
		"INTEGRATION_PROJECT=" + id, "INTEGRATION_PHASE=" + phase,
		"INTEGRATION_RESULTS=" + resultPath, "INTEGRATION_LOG_DIR=" + logs,
		"INTEGRATION_COMPLETION=" + resultPath + ".complete",
		"INTEGRATION_GO_FLAGS=",
	}
	if r.verbose {
		env[len(env)-1] += "-v"
	}
	if phase == "candidate" {
		workspace := filepath.Join(filepath.Dir(source), "go.work")
		if err := r.command(ctx, filepath.Dir(source), log, env, "go", "work", "init", source); err != nil {
			return setupError(err)
		}
		env = append(env, "GOWORK="+workspace)
		if err := r.command(ctx, source, log, env, "go", "work", "edit",
			"-replace=go.yaml.in/yaml/v4="+r.root); err != nil {
			return setupError(err)
		}
	}
	if err := r.command(ctx, source, log, env, "bash", script, "prepare", source); err != nil {
		return setupError(err)
	}
	if phase == "candidate" {
		verify := `test "$(go list -m -f '{{.Dir}}' go.yaml.in/yaml/v4)" = "$INTEGRATION_ROOT"`
		if err := r.command(ctx, source, log, env, "bash", "-euo", "pipefail", "-c", verify); err != nil {
			return setupError(err)
		}
	}
	err := r.command(ctx, source, log, env, "bash", script, "test", source)
	checks, readErr := readResults(resultPath)
	if readErr != nil {
		return setupError(fmt.Errorf("missing or invalid check results: %w", readErr))
	}
	if err != nil && !hasErrors(checks) {
		checks = append(checks, checkResult{Name: "script", Status: "setup-error", Detail: err.Error()})
	}
	if _, completeErr := os.Stat(resultPath + ".complete"); completeErr != nil &&
		(err == nil || hasErrors(checks)) {
		detail := "script stopped before finish"
		if err != nil {
			detail += ": " + err.Error()
		}
		checks = append(checks, checkResult{Name: "completion", Status: "setup-error", Detail: detail})
	}
	if len(checks) == 0 {
		checks = append(checks, checkResult{Name: "script", Status: "setup-error", Detail: "script reported no checks"})
	}
	return checks
}

func readResults(path string) ([]checkResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var checks []checkResult
	seen := make(map[string]bool)
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Split(s.Text(), "\t")
		check, err := parseCheck(fields)
		if err != nil || seen[check.Name] {
			return nil, fmt.Errorf("invalid or duplicate result: %q", s.Text())
		}
		seen[check.Name] = true
		checks = append(checks, check)
	}
	return checks, s.Err()
}

func hasErrors(checks []checkResult) bool {
	for _, c := range checks {
		if c.Status == "fail" || c.Status == "setup-error" {
			return true
		}
	}
	return false
}

func classify(baseline, candidate []checkResult) string {
	base := make(map[string]string)
	for _, c := range baseline {
		base[c.Name] = c.Status
	}
	for _, c := range candidate {
		if c.Status == "fail" && base[c.Name] == "pass" {
			return "upgrade-blocked"
		}
	}
	for _, c := range baseline {
		if c.Status == "fail" {
			return "baseline-failed"
		}
	}
	for _, checks := range [][]checkResult{baseline, candidate} {
		for _, c := range checks {
			if c.Status == "setup-error" {
				return "setup-error"
			}
		}
	}
	if hasErrors(candidate) {
		return "candidate-failed"
	}
	if len(baseline) != len(candidate) {
		return "setup-error"
	}
	for _, c := range candidate {
		if _, found := base[c.Name]; !found {
			return "setup-error"
		}
	}
	executed, uncovered := 0, 0
	for _, checks := range [][]checkResult{baseline, candidate} {
		for _, c := range checks {
			switch c.Status {
			case "pass":
				executed++
			case "skip", "setup-needed":
				uncovered++
			}
		}
	}
	if executed == 0 {
		return "no-coverage"
	}
	if uncovered > 0 {
		return "partial"
	}
	return "pass"
}

func renderSummary(results []projectResult, executed, skipped int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Downstream integration\n\n%d checks executed; %d projects skipped.\n\n", executed, skipped)
	if executed == 0 {
		b.WriteString("**ZERO COVERAGE: no downstream checks executed.**\n\n")
	}
	for _, p := range results {
		fmt.Fprintf(&b, "## %s: %s\n\n", p.Project, p.Status)
		if p.Commit != "" {
			fmt.Fprintf(&b, "Revision: `%s` (`%s`).\n\n", p.Ref, p.Commit)
		}
		if p.Detail != "" {
			fmt.Fprintf(&b, "%s\n\n", p.Detail)
		}
		if len(p.Baseline)+len(p.Candidate) == 0 {
			continue
		}
		fmt.Fprintf(&b, "Runtime: %s; exit code: %d.\n\n", durationText(time.Duration(p.DurationMS)*time.Millisecond), p.ExitCode)
		b.WriteString("| Phase | Check | Result | Runtime | Exit | Detail |\n|---|---|---|---|---|---|\n")
		for i, checks := range [][]checkResult{p.Baseline, p.Candidate} {
			phase := []string{"baseline", "candidate"}[i]
			for _, c := range checks {
				detail := strings.NewReplacer("|", "\\|", "\n", " ").Replace(c.Detail)
				exit := "n/a"
				if c.ExitCode != nil {
					exit = strconv.Itoa(*c.ExitCode)
				}
				fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", phase, c.Name, c.Status,
					durationText(time.Duration(c.DurationMS)*time.Millisecond), exit, detail)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func renderReport(c catalog) string {
	var b strings.Builder
	b.WriteString("# Projects Using go-yaml v4 Release Candidates\n\n")
	b.WriteString("Generated from `testdata/integration/projects.yaml`.\n")
	b.WriteString("Regenerate with `make integration-report`; Makes supplies Go.\n\n")
	b.WriteString("The list is not exhaustive.\n")
	b.WriteString("Star counts are informational metadata, not refreshed by test runs.\n\n")
	b.WriteString("| Project | Stars | Integration |\n|---|---:|---|\n")
	ids, _ := selectProjects(c, "")
	sort.SliceStable(ids, func(i, j int) bool { return c[ids[i]].Stars > c[ids[j]].Stars })
	for _, id := range ids {
		p := c[id]
		state := "Enabled"
		if p.Skip {
			state = "Skipped"
		}
		fmt.Fprintf(&b, "| [%s](https://github.com/%s/blob/%s/go.mod) | %d | %s |\n",
			p.Name, p.Repository, p.Ref, p.Stars, state)
	}
	b.WriteString("\nEnabled means selected for execution, not confirmed compatible.\n")
	b.WriteString("Run results report failures, capability skips, and unfinished setup.\n")
	for _, id := range ids {
		p := c[id]
		if p.Notes != "" {
			fmt.Fprintf(&b, "\n%s: %s\n", p.Name, p.Notes)
		}
	}
	return b.String()
}
