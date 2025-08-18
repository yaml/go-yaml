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

// Comparable types (numbers, strings, pointers to the same object, etc.).
func Equal(tb miniTB, want, got any) {
	tb.Helper()
	Equalf(tb, want, got, "")
}

func Equalf(tb miniTB, want, got any, msgFormat string, args ...any) {
	tb.Helper()
	if got != want {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got %v; want %v%s", got, want, suffix)
	}
}

// Anything else (slices, maps, structs with slices...).
func DeepEqual(tb miniTB, want, got any) {
	tb.Helper()
	DeepEqualf(tb, want, got, "")
}

func DeepEqualf(tb miniTB, want, got any, msgFormat string, args ...any) {
	tb.Helper()
	if !reflect.DeepEqual(got, want) {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got %+v; want %+v%s", got, want, suffix)
	}
}

func ErrorMatches(tb miniTB, pattern string, err error) {
	tb.Helper()
	ErrorMatchesf(tb, pattern, err, "")
}

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

func ErrorIs(tb miniTB, want, got error) {
	tb.Helper()
	if !errors.Is(got, want) {
		tb.Fatalf("got %#v; want %#v", got, want)
	}
}
func NoError(tb miniTB, err error) {
	tb.Helper()
	NoErrorf(tb, err, "")
}

func NoErrorf(tb miniTB, err error, msgFormat string, args ...any) {
	tb.Helper()
	if err != nil {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("unexpected error: %v%s", err, suffix)
	}
}

func IsNil(tb miniTB, v any) {
	tb.Helper()
	IsNilf(tb, v, "")
}

func IsNilf(tb miniTB, v any, msgFormat string, args ...any) {
	tb.Helper()
	if !isNil(v) {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got non-nil (type %T): %#v%s", v, v, suffix)
	}
}

func NotNil(tb miniTB, v any) {
	tb.Helper()
	NotNilf(tb, v, "")
}

func NotNilf(tb miniTB, v any, msgFormat string, args ...any) {
	tb.Helper()
	if isNil(v) {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got nil; want non-nil%s", suffix)
	}
}

func True(tb miniTB, got bool) {
	tb.Helper()
	Truef(tb, got, "")
}

func Truef(tb miniTB, got bool, msgFormat string, args ...any) {
	tb.Helper()
	if !got {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got false; want true%s", suffix)
	}
}

func False(tb miniTB, got bool) {
	tb.Helper()
	Falsef(tb, got, "")
}

func Falsef(tb miniTB, got bool, msgFormat string, args ...any) {
	tb.Helper()
	if got {
		suffix := formatSuffix(msgFormat, args...)
		tb.Fatalf("got true; want false%s", suffix)
	}
}

func PanicMatches(tb miniTB, pattern string, f func()) {
	tb.Helper()
	PanicMatchesf(tb, pattern, f, "")
}

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
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Interface, reflect.UnsafePointer:
		return rv.IsNil()
	default:
		return false
	}
}
