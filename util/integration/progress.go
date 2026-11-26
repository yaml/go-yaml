// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

// progress keeps completed results in scrollback and redraws only the current
// command's output. Logs always receive the complete, unmodified byte stream.
type progress struct {
	mu       sync.Mutex
	out      io.Writer
	live     bool
	color    bool
	verbose  bool
	width    int
	height   int
	rows     int
	label    string
	started  time.Time
	lines    []string
	fragment string
	active   bool
	stop     chan struct{}
	done     chan struct{}
}

func newProgress(out io.Writer, verbose bool) *progress {
	if out == nil {
		out = io.Discard
	}
	tty, width, height := false, 80, 24
	if f, ok := out.(*os.File); ok && os.Getenv("TERM") != "dumb" {
		tty, width, height = terminalSize(f)
	}
	_, noColor := os.LookupEnv("NO_COLOR")
	p := &progress{
		out: out, live: tty && !verbose, color: tty && !noColor,
		verbose: verbose, width: width, height: height,
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	go func() {
		defer close(p.done)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-p.stop:
				return
			case <-ticker.C:
				p.mu.Lock()
				if p.live && p.active {
					if f, ok := p.out.(*os.File); ok {
						_, p.width, p.height = terminalSize(f)
					}
					p.draw()
				}
				p.mu.Unlock()
			}
		}
	}()
	return p
}

func (p *progress) Close() {
	close(p.stop)
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clear()
}

func (p *progress) Begin(label string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clear()
	p.label, p.started, p.active = label, time.Now(), true
	p.lines, p.fragment = nil, ""
	if p.live {
		p.draw()
	} else {
		fmt.Fprintf(p.out, "%s %s\n", p.paint("RUN", "36"), label)
	}
}

func (p *progress) Complete(label, status string, duration time.Duration, code *int, detail string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completeLine("", label, status, duration, code, detail)
}

var projectRule = strings.Repeat("-", 80)

func (p *progress) StartProject(label string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clear()
	p.active = false
	fmt.Fprintf(p.out, "\n%s\n%s - Starting integration testing\n%s\n",
		projectRule, label, projectRule)
}

func (p *progress) EndProject(result projectResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clear()
	p.active = false
	fmt.Fprintf(p.out, "\n> %s\n", projectRule)
	if result.Status == "skipped" {
		fmt.Fprintf(p.out, "> %s - SKIPPED\n", result.Project)
	} else {
		fmt.Fprintf(p.out, "> %s - Finished integration testing\n", result.Project)
		p.completeLine("> ", result.Project, result.Status,
			time.Duration(result.DurationMS)*time.Millisecond, &result.ExitCode, result.Detail)
	}
	fmt.Fprintf(p.out, "> %s\n> \n", projectRule)
}

// The prefix precedes any color controls so anchored grep works on TTY output.
func (p *progress) completeLine(prefix, label, status string, duration time.Duration, code *int, detail string) {
	p.clear()
	p.active = false
	exit := "n/a"
	if code != nil {
		exit = strconv.Itoa(*code)
	}
	color := "33"
	switch status {
	case "pass":
		color = "32"
	case "fail", "upgrade-blocked", "baseline-failed", "candidate-failed":
		color = "31"
	}
	fmt.Fprintf(p.out, "%s%s %s  %s  exit=%s", prefix,
		p.paint(strings.ToUpper(status), color), label, durationText(duration), exit)
	if detail != "" {
		fmt.Fprintf(p.out, "  %s", cleanLine(detail))
	}
	fmt.Fprintln(p.out)
}

func durationText(duration time.Duration) string {
	if duration < time.Second {
		return duration.Round(time.Millisecond).String()
	}
	return duration.Round(100 * time.Millisecond).String()
}

func (p *progress) paint(text, color string) string {
	if p.color {
		return "\x1b[" + color + "m" + text + "\x1b[0m"
	}
	return text
}

func (p *progress) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.verbose {
		return p.out.Write(data)
	}
	if !p.live || !p.active {
		return len(data), nil
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	parts := strings.Split(p.fragment+text, "\n")
	p.fragment = parts[len(parts)-1]
	if len(p.fragment) > 8192 {
		p.fragment = p.fragment[len(p.fragment)-8192:]
	}
	p.lines = append(p.lines, parts[:len(parts)-1]...)
	if len(p.lines) > 10 {
		p.lines = p.lines[len(p.lines)-10:]
	}
	return len(data), nil
}

func (p *progress) clear() {
	if p.rows > 0 {
		fmt.Fprintf(p.out, "\x1b[%dA\r\x1b[J", p.rows)
		p.rows = 0
	}
}

func (p *progress) draw() {
	p.clear()
	text := fmt.Sprintf("RUN %s  %s", p.label, durationText(time.Since(p.started)))
	fmt.Fprintln(p.out, p.paint(clipLine(text, p.width-1), "36"))
	lines := append([]string(nil), p.lines...)
	if p.fragment != "" {
		lines = append(lines, p.fragment)
	}
	count := 10
	if p.height-2 < count {
		count = p.height - 2
	}
	if count < 0 {
		count = 0
	}
	if len(lines) > count {
		lines = lines[len(lines)-count:]
	}
	for _, line := range lines {
		fmt.Fprintf(p.out, "  %s\n", clipLine(cleanLine(line), p.width-3))
	}
	p.rows = 1 + len(lines)
}

// Discard terminal controls from the live preview, including OSC hyperlinks and
// titles. The raw logs and verbose output retain the upstream output.
var terminalControls = regexp.MustCompile("\x1b\\][^\a]*(?:\a|\x1b\\\\)|\x1b\\[[0-?]*[ -/]*[@-~]")

func cleanLine(text string) string {
	text = terminalControls.ReplaceAllString(text, "")
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
}

func clipLine(text string, columns int) string {
	if columns <= 0 {
		return ""
	}
	used := 0
	for index, r := range text {
		width := 1
		switch {
		case unicode.Is(unicode.Mn, r), r == '\u200d':
			width = 0
		case r >= 0x1100 && (r <= 0x115f || r >= 0x2329 && r <= 0x232a ||
			r >= 0x2e80 && r <= 0xa4cf || r >= 0xac00 && r <= 0xd7a3 ||
			r >= 0xf900 && r <= 0xfaff || r >= 0xfe10 && r <= 0xfe6f ||
			r >= 0xff00 && r <= 0xff60 || r >= 0xffe0 && r <= 0xffe6 ||
			r >= 0x1f000 && r <= 0x1ffff || r >= 0x20000):
			width = 2
		}
		if used+width > columns {
			return text[:index]
		}
		used += width
	}
	return text
}
