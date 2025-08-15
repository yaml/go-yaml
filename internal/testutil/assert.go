package testutil

import (
	"fmt"
	"reflect"
	"regexp"
)

type miniTB interface {
	Helper()
	Fatalf(string, ...any)
}

// formatSuffix builds an optional suffix from msgAndArgs
func formatSuffix(msgAndArgs ...any) string {
	switch len(msgAndArgs) {
	case 0:
		return ""
	case 1:
		return " - " + fmt.Sprintf("%+v", msgAndArgs[0])
	default:
		return " - " + fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
}

// Comparable types (numbers, strings, pointers to the same object, etc.).
func AssertEqual(tb miniTB, got, want any, msgAndArgs ...any) {
	tb.Helper()
	if got != want {
		suffix := formatSuffix(msgAndArgs...)
		tb.Fatalf("got %v; want %v%s", got, want, suffix)
	}
}

// Anything else (slices, maps, structs with slices...).
func AssertDeepEqual(tb miniTB, got, want any, msgAndArgs ...any) {
	tb.Helper()
	if !reflect.DeepEqual(got, want) {
		suffix := formatSuffix(msgAndArgs...)
		tb.Fatalf("got %+v; want %+v%s", got, want, suffix)
	}
}

func AssertErrorMatches(tb miniTB, err error, pattern string, msgAndArgs ...any) {
	tb.Helper()
	if err == nil {
		suffix := formatSuffix(msgAndArgs...)
		tb.Fatalf("got nil; want error matching %q%s", pattern, suffix)
		return
	}
	re, reErr := regexp.Compile(pattern)
	if reErr != nil {
		suffix := formatSuffix(msgAndArgs...)
		tb.Fatalf("invalid regexp %q: %v%s", pattern, reErr, suffix)
		return
	}
	if !re.MatchString(err.Error()) {
		suffix := formatSuffix(msgAndArgs...)
		tb.Fatalf("error %q does not match %q%s", err.Error(), pattern, suffix)
	}
}

func AssertNoError(tb miniTB, err error, msgAndArgs ...any) {
	tb.Helper()
	if err != nil {
		suffix := formatSuffix(msgAndArgs...)
		tb.Fatalf("unexpected error: %v%s", err, suffix)
	}
}

func AssertIsNil(tb miniTB, v any, msgAndArgs ...any) {
	tb.Helper()
	if !isNil(v) {
		suffix := formatSuffix(msgAndArgs...)
		tb.Fatalf("got non-nil (type %T): %#v%s", v, v, suffix)
	}
}

func AssertNotNil(tb miniTB, v any, msgAndArgs ...any) {
	tb.Helper()
	if isNil(v) {
		suffix := formatSuffix(msgAndArgs...)
		tb.Fatalf("got nil; want non-nil%s", suffix)
	}
}

func AssertPanicMatches(tb miniTB, f func(), pattern string, msgAndArgs ...any) {
	tb.Helper()
	var pan any
	func() {
		defer func() { pan = recover() }()
		f()
	}()
	if pan == nil {
		suffix := formatSuffix(msgAndArgs...)
		tb.Fatalf("function did not panic; want panic matching %q%s", pattern, suffix)
		return
	}
	var msg string
	switch x := pan.(type) {
	case error:
		msg = x.Error()
	case string:
		msg = x
	default:
		msg = fmt.Sprint(x)
	}
	re, reErr := regexp.Compile(pattern)
	if reErr != nil {
		suffix := formatSuffix(msgAndArgs...)
		tb.Fatalf("invalid regexp %q: %v%s", pattern, reErr, suffix)
		return
	}
	if !re.MatchString(msg) {
		suffix := formatSuffix(msgAndArgs...)
		tb.Fatalf("panic %q does not match %q%s", msg, pattern, suffix)
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
