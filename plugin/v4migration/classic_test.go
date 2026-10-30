// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package v4migration

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

// Keep the core APIs' original function types intact.
var (
	_ func([]byte, any) error       = yaml.Unmarshal
	_ func(io.Reader) *yaml.Decoder = yaml.NewDecoder
)

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestClassicUnmarshalParity(t *testing.T) {
	for _, input := range []string{
		"", "# comment\n", "null", "value: input", "---\nvalue: input\n---\nnext: doc",
		"value: input\n---\n[", "value: one\nvalue: two", "[",
	} {
		t.Run(input, func(t *testing.T) {
			var want, got any
			wantErr := yaml.Unmarshal([]byte(input), &want)
			gotErr := Unmarshal([]byte(input), &got)
			if !reflect.DeepEqual(got, want) ||
				errorText(gotErr) != errorText(wantErr) {
				t.Fatalf("got (%v, %v), want (%v, %v)", got, gotErr, want, wantErr)
			}
		})
	}
}

func TestClassicDecoderParity(t *testing.T) {
	for _, input := range []string{
		"", "# comment\n", "null", "value: input\n---\nnext: doc", "value: input\n---\n[",
	} {
		t.Run(input, func(t *testing.T) {
			core := yaml.NewDecoder(strings.NewReader(input))
			wrapper := NewDecoder(strings.NewReader(input))
			for i := 0; i < 3; i++ {
				var want, got any
				wantErr := core.Decode(&want)
				gotErr := wrapper.Decode(&got)
				if !reflect.DeepEqual(got, want) ||
					errorText(gotErr) != errorText(wantErr) {
					t.Fatalf("got (%v, %v), want (%v, %v)", got, gotErr, want, wantErr)
				}
				if wantErr != nil && !errors.Is(wantErr, io.EOF) {
					break
				}
			}
		})
	}
}

type countingReader struct {
	reads int
}

func (r *countingReader) Read([]byte) (int, error) {
	r.reads++
	return 0, io.EOF
}

func TestDecoderInitializationError(t *testing.T) {
	r := new(countingReader)
	decoder := NewDecoder(r, yaml.WithIndent(1))
	decoder.KnownFields(true)
	var out any
	first := decoder.Decode(&out)
	if first == nil {
		t.Fatal("invalid option accepted")
	}
	decoder.KnownFields(false)
	if second := decoder.Decode(&out); second != first {
		t.Fatalf("initialization error changed: %v", second)
	}
	if r.reads != 0 {
		t.Fatalf("read input %d times after invalid options", r.reads)
	}
	if err := Unmarshal([]byte("input"), &out, yaml.WithIndent(1)); err == nil {
		t.Fatal("Unmarshal accepted invalid options")
	}
}

func TestWrapperOptionsOverrideDefaults(t *testing.T) {
	var out struct{ Value string }
	if err := Unmarshal([]byte("unknown: input"), &out,
		yaml.WithKnownFields()); err == nil {
		t.Fatal("caller KnownFields ignored")
	}
	err := Unmarshal([]byte("["), &out, yaml.WithV4Defaults())
	if err == nil || !strings.HasPrefix(err.Error(), "go-yaml load error") {
		t.Fatalf("caller formatter ignored: %v", err)
	}
	var docs []map[string]string
	if err := Unmarshal([]byte("value: first\n---\nvalue: second"), &docs,
		yaml.WithAllDocuments()); err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs[1]["value"] != "second" {
		t.Fatal(docs)
	}
	decoder := NewDecoder(strings.NewReader("unknown: input\n---\nunknown: next"),
		yaml.WithKnownFields())
	if err := decoder.Decode(&out); err == nil {
		t.Fatal("decoder ignored caller KnownFields")
	}
	decoder.KnownFields(false)
	if err := decoder.Decode(&out); err != nil {
		t.Fatal(err)
	}
}

func TestWrappersRequireExplicitMigration(t *testing.T) {
	var out valueV3
	for name, load := range map[string]func() error{
		"Unmarshal": func() error { return Unmarshal([]byte("value: input"), &out) },
		"Decoder": func() error {
			return NewDecoder(strings.NewReader("value: input")).Decode(&out)
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := load(); err == nil ||
				!strings.Contains(err.Error(), "v4 cannot invoke v3 Node hooks") {
				t.Fatalf("expected migration error: %v", err)
			}
		})
	}
}
