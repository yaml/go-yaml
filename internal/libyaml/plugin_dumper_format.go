// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package libyaml

// DumperFormatPlugin formats a complete serialized YAML stream.
//
// Format is called once when a Dumper is closed, after all documents have
// been serialized. Returning an error prevents formatted output from being
// written.
type DumperFormatPlugin interface {
	Format([]byte) ([]byte, error)
}
