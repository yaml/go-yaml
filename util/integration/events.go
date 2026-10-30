// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func parseCheck(fields []string) (checkResult, error) {
	var c checkResult
	if (len(fields) != 3 && len(fields) != 5) || !projectID.MatchString(fields[0]) {
		return c, errors.New("invalid check record")
	}
	c.Name, c.Status, c.Detail = fields[0], fields[1], fields[2]
	switch c.Status {
	case "pass", "fail", "skip", "setup-error", "setup-needed":
	default:
		return c, fmt.Errorf("invalid status %q", c.Status)
	}
	if len(fields) == 5 {
		ms, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil || ms < 0 {
			return c, errors.New("invalid duration")
		}
		code, err := strconv.Atoi(fields[4])
		if err != nil || code < -1 || code > 255 {
			return c, errors.New("invalid exit code")
		}
		c.DurationMS = ms
		if code >= 0 {
			c.ExitCode = &code
		}
	}
	return c, nil
}

type eventSummary struct{ failed, setup int }

func readEvents(input io.Reader, p *progress, prefix string) eventSummary {
	var summary eventSummary
	s := bufio.NewScanner(input)
	for s.Scan() {
		fields := strings.Split(s.Text(), "\t")
		if len(fields) == 2 && fields[0] == "start" && projectID.MatchString(fields[1]) {
			p.Begin(prefix + "/" + fields[1])
		} else if len(fields) == 6 && fields[0] == "end" {
			c, err := parseCheck(fields[1:])
			if err != nil {
				continue
			}
			p.Complete(prefix+"/"+c.Name, c.Status,
				time.Duration(c.DurationMS)*time.Millisecond, c.ExitCode, c.Detail)
			if c.Status == "fail" {
				summary.failed++
			}
			if c.Status == "setup-error" || c.Status == "setup-needed" {
				summary.setup++
			}
		}
	}
	return summary
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return 124
	}
	if errors.Is(err, context.Canceled) {
		return 130
	}
	var exited *exec.ExitError
	if errors.As(err, &exited) && exited.ExitCode() >= 0 {
		return exited.ExitCode()
	}
	return 1
}

func commandLabel(log string, args []string) string {
	prefix := filepath.Base(filepath.Dir(log)) + "/" + strings.TrimSuffix(filepath.Base(log), ".log")
	action := args[0]
	switch args[0] {
	case "git":
		for i := 1; i < len(args); i++ {
			if args[i] == "-C" {
				i++
			} else if !strings.HasPrefix(args[i], "-") {
				action = args[i]
				break
			}
		}
	case "go":
		end := len(args)
		if end > 3 {
			end = 3
		}
		action = strings.Join(args[1:end], "-")
	case "bash":
		if len(args) > 2 && strings.HasSuffix(args[1], ".bash") {
			action = args[2]
			if action == "test" {
				action = "checks"
			}
		} else {
			action = "verify-module"
		}
	}
	return prefix + "/" + action
}
