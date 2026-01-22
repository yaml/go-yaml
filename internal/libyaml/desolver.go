// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Desolver removes unnecessary tags from YAML nodes.
// This is the inverse of tag resolution - tags that match implicit
// resolution can be omitted from the output.

package libyaml

// Desolver handles tag desolution for YAML nodes.
type Desolver struct {
	opts *Options
}

// NewDesolver creates a new Desolver with the given options.
func NewDesolver(opts *Options) *Desolver {
	return &Desolver{opts: opts}
}

// Desolve walks the node tree and removes unnecessary tags.
// Tags that match the implicit type resolution can be omitted.
// This is the inverse of Resolver.Resolve().
//
// This is currently a no-op placeholder. The actual tag-omission logic
// is still in serializer.go and will be moved here when represent()
// is refactored to build Node trees.
func (d *Desolver) Desolve(n *Node) {
	// TODO: Move tag-omission logic from serializer.go here
	// For now, this is a no-op. The serializer still handles tag omission
	// during the serialize pass.
}
