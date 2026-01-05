// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// This file contains the Dumper API for writing YAML documents.
//
// Primary functions:
// - Dump: Encode a value to YAML
// - DumpAll: Encode multiple values as multi-document YAML
// - NewDumper: Create a streaming dumper to io.Writer

package yaml

import (
	"bytes"
	"io"
	"reflect"

	"go.yaml.in/yaml/v4/internal/libyaml"
)

// Dump encodes a value to YAML with the given options.
//
// See [Marshal] for details about the conversion of Go values to YAML.
func Dump(in any, opts ...Option) (out []byte, err error) {
	defer handleErr(&err)
	var buf bytes.Buffer
	d, err := NewDumper(&buf, opts...)
	if err != nil {
		return nil, err
	}
	if err := d.Dump(in); err != nil {
		return nil, err
	}
	if err := d.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DumpAll encodes multiple values as a multi-document YAML stream.
//
// Each value becomes a separate YAML document, separated by "---".
// See [Marshal] for details about the conversion of Go values to YAML.
func DumpAll(in []any, opts ...Option) (out []byte, err error) {
	defer handleErr(&err)
	var buf bytes.Buffer
	d, err := NewDumper(&buf, opts...)
	if err != nil {
		return nil, err
	}
	for _, v := range in {
		if err := d.Dump(v); err != nil {
			return nil, err
		}
	}
	if err := d.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// A Dumper writes YAML values to an output stream with configurable options.
type Dumper struct {
	encoder *libyaml.Representer
	opts    *libyaml.Options
}

// NewDumper returns a new Dumper that writes to w with the given options.
//
// The Dumper should be closed after use to flush all data to w.
func NewDumper(w io.Writer, opts ...Option) (*Dumper, error) {
	o, err := libyaml.ApplyOptions(opts...)
	if err != nil {
		return nil, err
	}
	return &Dumper{
		encoder: libyaml.NewRepresenter(w, o),
		opts:    o,
	}, nil
}

// Dump writes the YAML encoding of v to the stream.
//
// If multiple values are dumped to the stream, the second and subsequent
// documents will be preceded with a "---" document separator.
//
// See the documentation for [Marshal] for details about the conversion of Go
// values to YAML.
func (d *Dumper) Dump(v any) (err error) {
	defer handleErr(&err)
	d.encoder.MarshalDoc("", reflect.ValueOf(v))
	return nil
}

// Close closes the Dumper by writing any remaining data.
// It does not write a stream terminating string "...".
func (d *Dumper) Close() (err error) {
	defer handleErr(&err)
	d.encoder.Finish()
	return nil
}
