// Package main provides node conversion utilities.

package main

import (
	"unsafe"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
)

// toLibNode converts yaml.Node to libyaml.Node for internal use.
// This is safe because yaml.Node and libyaml.Node have the same memory layout.
func toLibNode(n *yaml.Node) *libyaml.Node {
	return (*libyaml.Node)(unsafe.Pointer(n))
}

// fromLibNode converts libyaml.Node to yaml.Node for internal use.
// This is safe because yaml.Node and libyaml.Node have the same memory layout.
func fromLibNode(n *libyaml.Node) *yaml.Node {
	return (*yaml.Node)(unsafe.Pointer(n))
}
