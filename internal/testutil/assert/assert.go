// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package assert provides assertion functions for tests.
// This is an internal package that was created to provide a simple way to
// write tests.
//
// The idea was to do not add a dependency on a testing framework.
package assert

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
)

type miniTB interface {
	Helper()
	Fatalf(string, ...any)
}

// formatSuffix builds an optional suffix from a printf-style format and args.
// If msgFormat is empty, an empty string is returned.
func formatSuffix(msgFormat string, args ...any) string {
	if msgFormat == "" {
		return ""
	}
	return " - " + fmt.Sprintf(msgFormat, args...)
}

// Equal asserts that two values are equal.
//
// It can be used with comparable types: numbers, strings, pointers to the same object, etc.
//
// For any other types, use [DeepEqual].
func Equal(tb miniTB, want, got any) {
	tb.Helper()
	Equalf(tb, want, got, "")
}

// Equalf asserts that two values are equal, and reports a message if they are not.
func Equalf(tb miniTB, want, got any, msgFormat string, args ...any) {
	tb.Helper()
	if got != want {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got %v; want %v%s", got, want, suffix)
	}
}

// DeepEqual asserts that two values are deeply equal.
//
// It can be used with slices, maps, structs with slices...
//
// Please consider [Equal] for other types.
func DeepEqual(tb miniTB, want, got any) {
	tb.Helper()
	DeepEqualf(tb, want, got, "")
}

// DeepEqualf asserts that two values are deeply equal, and reports a message if they are not.
func DeepEqualf(tb miniTB, want, got any, msgFormat string, args ...any) {
	tb.Helper()
	if !reflect.DeepEqual(got, want) {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got %+v; want %+v%s", got, want, suffix)
	}
}

// ErrorMatches asserts that an error matches a regular expression.
func ErrorMatches(tb miniTB, pattern string, err error) {
	tb.Helper()
	ErrorMatchesf(tb, pattern, err, "")
}

// ErrorMatchesf asserts that an error matches a regular expression, and reports a message if it does not.
func ErrorMatchesf(tb miniTB, pattern string, err error, msgFormat string, args ...any) {
	tb.Helper()
	if err == nil {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got nil; want error matching %q%s", pattern, suffix)
		return
	}
	re, reErr := regexp.Compile(pattern)
	if reErr != nil {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("invalid regexp %q: %v%s", pattern, reErr, suffix)
		return
	}
	if !re.MatchString(err.Error()) {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("error %q does not match %q%s", err.Error(), pattern, suffix)
	}
}

// ErrorIs asserts that two errors are equal by using [errors.Is].
func ErrorIs(tb miniTB, got, want error) {
	tb.Helper()
	if !errors.Is(got, want) {
		tb.Fatalf("got %#v; want %#v", got, want)
	}
}

// errorAsNoPanic calls [errors.As], but catch possible panic and returns it as an error
func errorAsNoPanic(tb miniTB, err error, target any) (ok bool, panic error) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
			panic = fmt.Errorf("panic: %v", r)
			return
		}
	}()

	return errors.As(err, target), nil
}

// ErrorAs asserts that an error can be assigned to a target variable by using [errors.As].
func ErrorAs(tb miniTB, err error, target any) {
	tb.Helper()

	ok, panicErr := errorAsNoPanic(tb, err, target)
	if panicErr != nil {
		tb.Fatalf("%s", panicErr)
		return
	}
	if ok {
		return
	}

	reflectedType := reflect.TypeOf(target)
	if reflectedType.Kind() != reflect.Pointer {
		// this is not supposed to happen with the current implementation of [errors.As]
		tb.Fatalf("a pointer was expected: got: %s; want: ptr", reflectedType.Kind())
		return
	}

	tb.Fatalf("got %#v; want %s", err, reflectedType.Elem())
}

// NoError asserts that an error is nil.
func NoError(tb miniTB, err error) {
	tb.Helper()
	NoErrorf(tb, err, "")
}

// NoErrorf asserts that an error is nil, and reports a message if it is not.
func NoErrorf(tb miniTB, err error, msgFormat string, args ...any) {
	tb.Helper()
	if err != nil {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("unexpected error: %v%s", err, suffix)
	}
}

// IsNil asserts that a value is nil.
func IsNil(tb miniTB, v any) {
	tb.Helper()
	IsNilf(tb, v, "")
}

// IsNilf asserts that a value is nil, and reports a message if it is not.
func IsNilf(tb miniTB, v any, msgFormat string, args ...any) {
	tb.Helper()
	if !isNil(v) {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got non-nil (type %T): %#v%s", v, v, suffix)
	}
}

// NotNil asserts that a value is not nil.
func NotNil(tb miniTB, v any) {
	tb.Helper()
	NotNilf(tb, v, "")
}

// NotNilf asserts that a value is not nil, and reports a message if it is.
func NotNilf(tb miniTB, v any, msgFormat string, args ...any) {
	tb.Helper()
	if isNil(v) {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got nil; want non-nil%s", suffix)
	}
}

// True asserts that a value is true.
func True(tb miniTB, got bool) {
	tb.Helper()
	Truef(tb, got, "")
}

// Truef asserts that a value is true, and reports a message if it is not.
func Truef(tb miniTB, got bool, msgFormat string, args ...any) {
	tb.Helper()
	if !got {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got false; want true%s", suffix)
	}
}

// False asserts that a value is false.
func False(tb miniTB, got bool) {
	tb.Helper()
	Falsef(tb, got, "")
}

// Falsef asserts that a value is false, and reports a message if it is not.
func Falsef(tb miniTB, got bool, msgFormat string, args ...any) {
	tb.Helper()
	if got {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got true; want false%s", suffix)
	}
}

// PanicMatches asserts that a function panics with a message matching the given pattern.
func PanicMatches(tb miniTB, pattern string, f func()) {
	tb.Helper()
	PanicMatchesf(tb, pattern, f, "")
}

// PanicMatchesf asserts that a function panics with a message matching the given pattern,
// and reports a message if it does not.
func PanicMatchesf(tb miniTB, pattern string, f func(), msgFormat string, args ...any) {
	tb.Helper()
	var pan any
	func() {
		defer func() { pan = recover() }()
		f()
	}()
	if pan == nil {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("function did not panic; want panic matching %q%s", pattern, suffix)
		return
	}
	var pmsg string
	switch x := pan.(type) {
	case error:
		pmsg = x.Error()
	case string:
		pmsg = x
	default:
		pmsg = fmt.Sprint(x)
	}
	re, reErr := regexp.Compile(pattern)
	if reErr != nil {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("invalid regexp %q: %v%s", pattern, reErr, suffix)
		return
	}
	if !re.MatchString(pmsg) {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("panic %q does not match %q%s", pmsg, pattern, suffix)
	}
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Interface, reflect.UnsafePointer:
		return rv.IsNil()
	default:
		return false
	}
}
