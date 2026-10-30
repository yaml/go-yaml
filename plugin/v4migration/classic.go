// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package v4migration

import (
	"io"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
)

func classicOptions(opts []yaml.Option) []yaml.Option {
	defaults := []yaml.Option{
		yaml.WithV3Defaults(),
		func(o *libyaml.Options) error {
			libyaml.SetLegacyAllowTrailingContent(o)
			return nil
		},
	}
	return append(defaults, opts...)
}

// Unmarshal is the classic YAML decoder with optional v4 options.
// Migration is enabled only when explicitly requested in opts.
func Unmarshal(in []byte, out any, opts ...yaml.Option) error {
	return yaml.Load(in, out, classicOptions(opts)...)
}

// Decoder is an option-aware classic YAML stream decoder.
// It is a separate type from yaml.Decoder.
type Decoder struct {
	loader  *libyaml.Loader
	initErr error
}

// NewDecoder creates a classic stream decoder with optional v4 options.
// Invalid options are reported by Decode before any input is read.
func NewDecoder(r io.Reader, opts ...yaml.Option) *Decoder {
	loader, err := libyaml.NewLoader(r, classicOptions(opts)...)
	return &Decoder{loader: loader, initErr: err}
}

// Decode reads the next document or returns the initialization error.
func (d *Decoder) Decode(out any) error {
	if d.initErr != nil {
		return d.initErr
	}
	return d.loader.Load(out)
}

// KnownFields sets strict struct field checking for subsequent Decode calls.
// It is safe to call when initialization failed; Decode still returns that error.
func (d *Decoder) KnownFields(enable bool) {
	if d.loader != nil {
		d.loader.SetLegacyLoaderKnownFields(enable)
	}
}
