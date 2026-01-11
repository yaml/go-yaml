// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Error types for YAML parsing and emitting.
// Provides structured error reporting with line/column information.

package libyaml

import (
	"errors"
	"fmt"
	"strings"
)

type MarkedYAMLError struct {
	// optional context
	ContextMark    Mark
	ContextMessage string

	Mark    Mark
	Message string
}

func (e MarkedYAMLError) Error() string {
	var builder strings.Builder
	builder.WriteString("yaml: ")
	if len(e.ContextMessage) > 0 {
		fmt.Fprintf(&builder, "%s at %s: ", e.ContextMessage, e.ContextMark)
	}
	if len(e.ContextMessage) == 0 || e.ContextMark != e.Mark {
		fmt.Fprintf(&builder, "%s: ", e.Mark)
	}
	builder.WriteString(e.Message)
	return builder.String()
}

type ParserError MarkedYAMLError

func (e ParserError) Error() string {
	return MarkedYAMLError(e).Error()
}

type ScannerError MarkedYAMLError

func (e ScannerError) Error() string {
	return MarkedYAMLError(e).Error()
}

type ReaderError struct {
	Offset int
	Value  int
	Err    error
}

func (e ReaderError) Error() string {
	return fmt.Sprintf("yaml: offset %d: %s", e.Offset, e.Err)
}

func (e ReaderError) Unwrap() error {
	return e.Err
}

type EmitterError struct {
	Message string
}

func (e EmitterError) Error() string {
	return fmt.Sprintf("yaml: %s", e.Message)
}

type WriterError struct {
	Err error
}

func (e WriterError) Error() string {
	return fmt.Sprintf("yaml: %s", e.Err)
}

func (e WriterError) Unwrap() error {
	return e.Err
}

// ConstructError represents a single, non-fatal error that occurred during
// the constructing of a YAML document into a Go value.
type ConstructError struct {
	Err    error
	Line   int
	Column int
}

func (e *ConstructError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Err.Error())
}

func (e *ConstructError) Unwrap() error {
	return e.Err
}

// TypeError is returned when one or more fields cannot be properly decoded.
type TypeError struct {
	Errors []*ConstructError
}

func (e *TypeError) Error() string {
	var b strings.Builder
	b.WriteString("yaml: construct errors:")
	for _, err := range e.Errors {
		b.WriteString("\n  ")
		b.WriteString(err.Error())
	}
	return b.String()
}

// Unwrap returns all errors for compatibility with errors.As/Is.
// This allows callers to unwrap TypeError and examine individual ConstructErrors.
// Implements the Go 1.20+ multiple error unwrapping interface.
func (e *TypeError) Unwrap() []error {
	errs := make([]error, len(e.Errors))
	for i, err := range e.Errors {
		errs[i] = err
	}
	return errs
}

// As implements errors.As for Go versions prior to 1.20 that don't support
// the Unwrap() []error interface. It allows TypeError to match against
// *ConstructError targets by returning the first error in the list.
func (e *TypeError) As(target any) bool {
	if len(e.Errors) == 0 {
		return false
	}
	if t, ok := target.(**ConstructError); ok {
		*t = e.Errors[0]
		return true
	}
	return false
}

// Is implements errors.Is for Go versions prior to 1.20 that don't support
// the Unwrap() []error interface. It checks if any wrapped error matches
// the target error.
func (e *TypeError) Is(target error) bool {
	for _, err := range e.Errors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// Strings returns the error messages as a string slice.
//
// This method is provided for compatibility with code migrating from v3,
// where TypeError.Errors was []string. New code should access the Errors
// field directly to get structured error information including line and
// column numbers.
func (e *TypeError) Strings() []string {
	result := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		result[i] = err.Error()
	}
	return result
}

// YAMLError is an internal error wrapper type.
type YAMLError struct {
	Err error
}

func (e *YAMLError) Error() string {
	return e.Err.Error()
}
