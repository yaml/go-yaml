// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"go.yaml.in/yaml/v4"
)

type dumperFormatFunc func([]byte) ([]byte, error)

func (f dumperFormatFunc) Format(input []byte) ([]byte, error) {
	return f(input)
}

func TestDumperFormatWholeStream(t *testing.T) {
	var calls int
	var gotInput string
	formatter := dumperFormatFunc(func(input []byte) ([]byte, error) {
		calls++
		gotInput = string(input)
		return append([]byte("formatted:\n"), input...), nil
	})

	got, err := yaml.Dump(
		[]any{map[string]string{"a": "b"}, map[string]string{"c": "d"}},
		yaml.WithAllDocuments(),
		yaml.WithPlugin(formatter),
	)
	if err != nil {
		t.Fatalf("Dump failed: %v", err)
	}
	if calls != 1 {
		t.Fatalf("Format called %d times, want 1", calls)
	}
	wantInput := "a: b\n---\nc: d\n"
	if gotInput != wantInput {
		t.Fatalf("Format input = %q, want %q", gotInput, wantInput)
	}
	want := "formatted:\n" + wantInput
	if string(got) != want {
		t.Fatalf("Dump output = %q, want %q", got, want)
	}
}

func TestDumperFormatRunsAtClose(t *testing.T) {
	var output bytes.Buffer
	var calls int
	formatter := dumperFormatFunc(func(input []byte) ([]byte, error) {
		calls++
		return bytes.ToUpper(input), nil
	})
	dumper, err := yaml.NewDumper(&output, yaml.WithPlugin(formatter))
	if err != nil {
		t.Fatalf("NewDumper failed: %v", err)
	}
	if err := dumper.Dump(map[string]string{"a": "b"}); err != nil {
		t.Fatalf("Dumper.Dump failed: %v", err)
	}
	if output.Len() != 0 || calls != 0 {
		t.Fatalf("output or formatting occurred before Close")
	}
	if err := dumper.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if output.String() != "A: B\n" || calls != 1 {
		t.Fatalf("output = %q and calls = %d", output.String(), calls)
	}
	if err := dumper.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}
	if output.String() != "A: B\n" || calls != 1 {
		t.Fatalf("second Close repeated output or formatting")
	}
}

func TestDumperFormatError(t *testing.T) {
	cause := errors.New("format failed")
	formatter := dumperFormatFunc(func([]byte) ([]byte, error) {
		return nil, cause
	})
	var output bytes.Buffer
	dumper, err := yaml.NewDumper(&output, yaml.WithPlugin(formatter))
	if err != nil {
		t.Fatalf("NewDumper failed: %v", err)
	}
	if err := dumper.Dump("value"); err != nil {
		t.Fatalf("Dumper.Dump failed: %v", err)
	}
	err = dumper.Close()
	if !errors.Is(err, cause) {
		t.Fatalf("Close error = %v, want wrapped cause", err)
	}
	var dumpErr *yaml.DumpError
	if !errors.As(err, &dumpErr) {
		t.Fatalf("Close error type = %T, want *yaml.DumpError", err)
	}
	if dumpErr.Stage != yaml.DumperFormatStage {
		t.Fatalf("Close error stage = %q", dumpErr.Stage)
	}
	if output.Len() != 0 {
		t.Fatalf("writer received %q after formatting failed", output.String())
	}
	if err2 := dumper.Close(); !errors.Is(err2, cause) {
		t.Fatalf("second Close error = %v, want cached cause", err2)
	}
}

func TestDumperFormatWriterError(t *testing.T) {
	cause := errors.New("write failed")
	formatter := dumperFormatFunc(func(input []byte) ([]byte, error) {
		return input, nil
	})
	dumper, err := yaml.NewDumper(errorDumperWriter{cause},
		yaml.WithPlugin(formatter))
	if err != nil {
		t.Fatalf("NewDumper failed: %v", err)
	}
	if err := dumper.Dump("value"); err != nil {
		t.Fatalf("Dumper.Dump failed: %v", err)
	}
	err = dumper.Close()
	var dumpErr *yaml.DumpError
	if !errors.Is(err, cause) || !errors.As(err, &dumpErr) ||
		dumpErr.Stage != yaml.WriterStage {
		t.Fatalf("Close error = %v, want writer-stage cause", err)
	}
}

func TestDumperFormatMultiplePlugins(t *testing.T) {
	formatter := dumperFormatFunc(func(input []byte) ([]byte, error) {
		return input, nil
	})
	_, err := yaml.Dump("value", yaml.WithPlugin(formatter, formatter))
	want := "yaml: multiple dumper-format plugins"
	if err == nil || err.Error() != want {
		t.Fatalf("Dump error = %v, want %q", err, want)
	}
}

type errorDumperWriter struct {
	err error
}

func (w errorDumperWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("writer: %w", w.err)
}
