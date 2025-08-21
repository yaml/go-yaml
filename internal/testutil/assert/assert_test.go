package assert

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"
)

func TestAssertEqual_Success(t *testing.T) {
	Equal(t, 2, 2)
	Equal(t, "ok", "ok")
}

func TestAssertDeepEqual_Success(t *testing.T) {
	DeepEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
	DeepEqual(t, map[string]int{"a": 1}, map[string]int{"a": 1})
}

func TestErrorMatches_Success(t *testing.T) {
	err := fmt.Errorf("http 404: not found")
	ErrorMatches(t, `http \d+: not found`, err)
}

func TestAssertNoError_Success(t *testing.T) {
	var err error
	NoError(t, err)
}

func TestIsNil_NotNil_Success(t *testing.T) {
	var p *int
	IsNil(t, p)

	var s []int
	IsNil(t, s)

	var w io.Writer
	IsNil(t, w)

	s2 := make([]int, 0)
	NotNil(t, s2)

	x := 0
	NotNil(t, &x)
}

func TestPanicMatches_Success(t *testing.T) {
	PanicMatches(t, `boom \d+`, func() { panic("boom 123") })
	PanicMatches(t, `fail xyz`, func() { panic(fmt.Errorf("fail xyz")) })
}

func TestAssertTrueAndFalse_Success(t *testing.T) {
	True(t, true)
	False(t, false)
}

func Test_isNil_Basics(t *testing.T) {
	if !isNil(nil) {
		t.Fatalf("nil should be nil")
	}
	var p *int
	if !isNil(p) {
		t.Fatalf("nil pointer should be nil")
	}
	if isNil(0) {
		t.Fatalf("non-nil value reported as nil")
	}
	s := make([]int, 0)
	if isNil(s) {
		t.Fatalf("non-nil slice reported as nil")
	}
}

/************** failure-path checks **************/

type fakeTB struct {
	failed bool
	msg    string
}

func (f *fakeTB) Helper() {}

func (f *fakeTB) Fatalf(format string, args ...any) {
	f.failed = true
	f.msg = fmt.Sprintf(format, args...)
}

// assertFailureMessageMatches checks if fakeTB recorded a failure and that
// its message matches the regexp.
func assertFailureMessageMatches(t *testing.T, f *fakeTB, pattern string) {
	t.Helper()
	if !f.failed {
		t.Fatalf("expected failure")
	}
	re := regexp.MustCompile(pattern)
	if !re.MatchString(f.msg) {
		t.Fatalf("message does not match:\ngot: `%s`\nregexp: `%s`", f.msg, pattern)
	}
}

func assertFailureMessageContains(t *testing.T, f *fakeTB, substr string) {
	t.Helper()
	if !f.failed {
		t.Fatalf("expected failure")
	}
	if !strings.Contains(f.msg, substr) {
		t.Fatalf("message doesn't contain:\ngot:  `%s`\nwant: `%s`", f.msg, substr)
	}
}

func TestAssertEqual_Fails(t *testing.T) {
	mock := &fakeTB{}
	Equal(mock, 2, 1)
	assertFailureMessageMatches(t, mock, `^got 1; want 2$`)
}

func TestAssertDeepEqual_Fails(t *testing.T) {
	// slice mismatch
	mock := &fakeTB{}
	DeepEqual(mock, []int{2}, []int{1})
	assertFailureMessageMatches(t, mock, `^got \[1\]; want \[2\]$`)

	// map mismatch
	mock2 := &fakeTB{}
	DeepEqual(mock2, map[string]int{"a": 2}, map[string]int{"a": 1})
	assertFailureMessageMatches(t, mock2, `^got map\[a:1\]; want map\[a:2\]$`)
}

func TestErrorMatches_Fails(t *testing.T) {
	// nil error
	mockTB1 := &fakeTB{}
	ErrorMatches(mockTB1, `x`, nil)
	assertFailureMessageMatches(t, mockTB1, `^got nil; want error matching "x"$`)

	// invalid regexp (message may include parser details; check prefix)
	mockTB2 := &fakeTB{}
	ErrorMatches(mockTB2, `(`, fmt.Errorf("x"))
	assertFailureMessageMatches(t, mockTB2, `^invalid regexp "`)

	// no match
	mockTB3 := &fakeTB{}
	ErrorMatches(mockTB3, `def`, fmt.Errorf("abc"))
	assertFailureMessageMatches(t, mockTB3, `^error "abc" does not match "def"$`)
}

func TestAssertNoError_Fails(t *testing.T) {
	m := &fakeTB{}
	NoError(m, fmt.Errorf("problem"))
	assertFailureMessageMatches(t, m, `^unexpected error: problem$`)
}

func TestErrorIs_Success(t *testing.T) {
	base := fmt.Errorf("base error")
	wrapped := fmt.Errorf("wrap: %w", base)
	// direct match
	ErrorIs(t, base, base)
	ErrorIs(t, wrapped, wrapped)
	ErrorIs(t, wrapped, base)
	// both nil
	ErrorIs(t, nil, nil)
}

func TestErrorIs_Fails(t *testing.T) {
	// mismatch
	mock1 := &fakeTB{}
	base := fmt.Errorf("base")
	other := fmt.Errorf("other")
	ErrorIs(mock1, base, other)
	assertFailureMessageMatches(t, mock1, `got &errors.errorString{s:"base"}; want &errors.errorString{s:"other"}`)

	// expected non-nil, actual nil
	mock2 := &fakeTB{}
	ErrorIs(mock2, nil, base)
	assertFailureMessageMatches(t, mock2, `got <nil>; want &errors.errorString{s:"base"}`)

	// expected nil, actual non-nil
	mock3 := &fakeTB{}
	ErrorIs(mock3, other, nil)
	assertFailureMessageMatches(t, mock3, `got &errors.errorString{s:"other"}; want <nil>`)
}

// customErr is a custom error type for testing.
type customErr struct {
	msg string
}

func (e *customErr) Error() string {
	return e.msg
}

func TestErrorAs_Success(t *testing.T) {
	// this simulates an error returned from a function as error
	var err error = &customErr{"foo"}

	var target *customErr
	ErrorAs(t, err, &target)
	Equal(t, "foo", target.Error())
}

func TestErrorAs_Fails(t *testing.T) {
	tb := &fakeTB{}

	err := errors.New("foo")

	var target *customErr
	ErrorAs(tb, err, &target)
	assertFailureMessageContains(t, tb, `got &errors.errorString{s:"foo"}; want *assert.customErr`)

	tb = &fakeTB{}
	ErrorAs(tb, nil, &target)
	assertFailureMessageContains(t, tb, `got <nil>; want *assert.customErr`)

	tb = &fakeTB{}
	ErrorAs(tb, err, nil) // this is invalid, a panic is expected
	assertFailureMessageContains(t, tb, `panic`)

	tb = &fakeTB{}
	ErrorAs(tb, err, 42) // this is invalid, a panic is expected
	assertFailureMessageContains(t, tb, `panic`)

	var a int
	tb = &fakeTB{}
	ErrorAs(tb, err, &a) // this is invalid, a panic is expected
	assertFailureMessageContains(t, tb, `panic`)
}

func TestIsNil_Fails(t *testing.T) {
	// non-nil slice
	mockTB1 := &fakeTB{}
	s := make([]int, 0)
	IsNil(mockTB1, s)
	assertFailureMessageMatches(t, mockTB1, `^got non-nil \(type `)

	// non-nil pointer
	mockTB2 := &fakeTB{}
	x := 1
	IsNil(mockTB2, &x)
	assertFailureMessageMatches(t, mockTB2, `^got non-nil \(type `)
}

func TestNotNil_Fails(t *testing.T) {
	// nil interface
	mockTB1 := &fakeTB{}
	var w io.Writer
	NotNil(mockTB1, w)
	assertFailureMessageMatches(t, mockTB1, `^got nil; want non-nil$`)

	// nil pointer
	mockTB2 := &fakeTB{}
	var p *int
	NotNil(mockTB2, p)
	assertFailureMessageMatches(t, mockTB2, `^got nil; want non-nil$`)
}

func TestPanicMatches_Fails(t *testing.T) {
	// no panic
	mockTB1 := &fakeTB{}
	PanicMatches(mockTB1, `x`, func() {})
	assertFailureMessageMatches(t, mockTB1, `^function did not panic; want panic matching "x"$`)

	// invalid regexp
	mockTB2 := &fakeTB{}
	PanicMatches(mockTB2, `(`, func() { panic("oops") })
	assertFailureMessageMatches(t, mockTB2, `^invalid regexp "`)

	// pattern mismatch
	mockTB3 := &fakeTB{}
	PanicMatches(mockTB3, `bar`, func() { panic("foo") })
	assertFailureMessageMatches(t, mockTB3, `^panic "foo" does not match "bar"$`)
}

func TestAssertTrueAndFalse_Fails(t *testing.T) {
	mock1 := &fakeTB{}
	True(mock1, false)
	assertFailureMessageMatches(t, mock1, `^got false; want true$`)

	mock2 := &fakeTB{}
	False(mock2, true)
	assertFailureMessageMatches(t, mock2, `^got true; want false$`)
}

func TestFormatSuffix_NoArgs(t *testing.T) {
	if got := formatSuffix(""); got != "" {
		t.Fatalf("expected empty suffix; got %q", got)
	}
}

func TestFormatSuffix_FormatString(t *testing.T) {
	got := formatSuffix("failed %s", "case")
	want := " - failed case"
	if got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}

func TestFormatSuffix_JustString(t *testing.T) {
	got := formatSuffix("hello")
	want := " - hello"
	if got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}

func TestFormatSuffix_AsUsedByAssertions(t *testing.T) {
	mockTB1 := &fakeTB{}
	var w io.Writer // nil interface

	// with format string
	NotNilf(mockTB1, w, "extra %s options %d foo %+v", "str", 42, map[int]bool{3: true})
	assertFailureMessageMatches(t, mockTB1, `^got nil; want non-nil - extra str options 42 foo map\[3:true\]$`)

	// with just a string arg
	NotNilf(mockTB1, w, "ba-dum-tss")
	assertFailureMessageMatches(t, mockTB1, `^got nil; want non-nil - ba-dum-tss$`)

	// with no message args
	NotNil(mockTB1, w)
	assertFailureMessageMatches(t, mockTB1, `^got nil; want non-nil$`)
}
