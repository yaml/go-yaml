# Go File Refactoring Guide

## Overview

This document describes the standardized formatting conventions for Go source
files in the go-yaml project.
The reference implementation is `yaml.go`.

## File Structure Order

All non-test Go files should follow this section order:

1. **Copyright header** (required)
2. **Package documentation** (with TOC for files with 3+ sections)
3. **Imports**
4. **Types** (public first, then private)
5. **Constants** (public first, then private)
6. **Variables** (public first, then private)
7. **Functions** (public first, then private)

### Dividers

Use a single divider comment only once per file, placed before the private
functions section at the bottom:

```go
// ----------------------------------------------------------------------------
// Private functions
// ----------------------------------------------------------------------------
```

### Table of Contents

For files with 3+ major sections, include a TOC in the package documentation:

```go
// Package yaml implements YAML support for Go.
//
// # Contents
//
//   - Types: Node, Kind, Style, Marshaler, Unmarshaler
//   - Functions: Marshal, Unmarshal, NewEncoder, NewDecoder
```

## Commenting Requirements

### All Blocks Need Comments

Every exported and unexported type, constant, variable, and function block
needs a leading comment:

```go
// NodeKind represents the kind of a YAML node.
type NodeKind uint8

// Node kinds.
const (
    DocumentNode NodeKind = iota
    SequenceNode
    MappingNode
    ScalarNode
    AliasNode
)

// defaultMapType is the default type for unmarshaling maps.
var defaultMapType = reflect.TypeOf(map[string]any{})

// Marshal serializes the value provided into a YAML document.
func Marshal(v any) ([]byte, error) {
```

### Comment Style

- Start comments with the name of the thing being documented
- Use complete sentences
- Wrap at 80 columns
- For grouped declarations (type blocks, const blocks), one comment for the
  block is sufficient

### Grouped Type Declarations

Related types can be grouped in a single `type ()` block with one leading
comment:

```go
// Marshaler and Unmarshaler interfaces for custom YAML handling.
type (
    Marshaler interface {
        MarshalYAML() (any, error)
    }
    Unmarshaler interface {
        UnmarshalYAML(node *Node) error
    }
)
```

## Test File Conventions

Test files (`*_test.go`) follow different rules:

### Structure

- Keep the standard Go test file interleaved structure
- Do NOT reorder into Types/Constants/Variables/Functions sections
- Tests naturally interleave helper types/functions with their tests

### Commenting in Test Files

- **Helper functions**: Add comments describing their purpose
- **Test types**: Add comments for standalone type definitions
- **Test data vars**: Can remain without comments (self-documenting table names)

Example:

```go
// errReader is a test io.Reader that always returns an error.
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
    return 0, errors.New("some read error")
}

// runDecodeTest runs a single decode test case from the data-driven test
// suite.
func runDecodeTest(t *testing.T, tc map[string]any) {
    t.Helper()
    // ...
}
```

## Verification

After each file modification:

```bash
make check        # Ensure all tests pass
```

## Refactoring Process

For each source file:

1. **Read the entire file** to understand its structure
2. **Identify sections**: types, constants, variables, functions
3. **Check for missing comments** on any declarations
4. **Reorder if needed**: Types → Constants → Variables → Functions
5. **Add divider** before private functions (if file has private functions)
6. **Add TOC** to package doc (if file has 3+ sections)
7. **Run verification** commands

For each test file:

1. **Read the entire file**
2. **Identify types and helper functions without comments**
3. **Add comments** to standalone types and helper functions
4. **Do NOT reorder** - keep interleaved structure
5. **Run verification** commands
