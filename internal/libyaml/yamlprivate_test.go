// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestIsAlpha(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte("abc"), 0, true},
		{[]byte("ABC"), 0, true},
		{[]byte("123"), 0, true},
		{[]byte("_-"), 0, true},
		{[]byte("_-"), 1, true},
		{[]byte(" "), 0, false},
		{[]byte("!"), 0, false},
		{[]byte("@"), 0, false},
	}

	for _, tt := range tests {
		got := isAlpha(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isAlpha(%q, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestIsFlowIndicator(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte("["), 0, true},
		{[]byte("]"), 0, true},
		{[]byte("{"), 0, true},
		{[]byte("}"), 0, true},
		{[]byte(","), 0, true},
		{[]byte("a"), 0, false},
		{[]byte(":"), 0, false},
	}

	for _, tt := range tests {
		got := isFlowIndicator(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isFlowIndicator(%q, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestIsAnchorChar(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte("abc"), 0, true},
		{[]byte("123"), 0, true},
		{[]byte("_-"), 0, true},
		{[]byte(":"), 0, false},
		{[]byte("["), 0, false},
		{[]byte(" "), 0, false},
		{[]byte("\n"), 0, false},
		{[]byte{0xEF, 0xBB, 0xBF}, 0, false},
	}

	for _, tt := range tests {
		got := isAnchorChar(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isAnchorChar(%q, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestIsColon(t *testing.T) {
	assert.Truef(t, isColon([]byte(":"), 0), "isColon(\":\", 0) should be true")
	assert.Falsef(t, isColon([]byte("a"), 0), "isColon(\"a\", 0) should be false")
}

func TestIsDigit(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte("0"), 0, true},
		{[]byte("5"), 0, true},
		{[]byte("9"), 0, true},
		{[]byte("a"), 0, false},
		{[]byte(" "), 0, false},
	}

	for _, tt := range tests {
		got := isDigit(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isDigit(%q, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestAsDigit(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected int
	}{
		{[]byte("0"), 0, 0},
		{[]byte("5"), 0, 5},
		{[]byte("9"), 0, 9},
	}

	for _, tt := range tests {
		got := asDigit(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "asDigit(%q, %d) = %d, want %d", tt.b, tt.i, got, tt.expected)
	}
}

func TestIsHex(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte("0"), 0, true},
		{[]byte("9"), 0, true},
		{[]byte("A"), 0, true},
		{[]byte("F"), 0, true},
		{[]byte("a"), 0, true},
		{[]byte("f"), 0, true},
		{[]byte("G"), 0, false},
		{[]byte("g"), 0, false},
	}

	for _, tt := range tests {
		got := isHex(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isHex(%q, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestAsHex(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected int
	}{
		{[]byte("0"), 0, 0},
		{[]byte("9"), 0, 9},
		{[]byte("A"), 0, 10},
		{[]byte("F"), 0, 15},
		{[]byte("a"), 0, 10},
		{[]byte("f"), 0, 15},
	}

	for _, tt := range tests {
		got := asHex(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "asHex(%q, %d) = %d, want %d", tt.b, tt.i, got, tt.expected)
	}
}

func TestIsASCII(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte("a"), 0, true},
		{[]byte{0x7F}, 0, true},
		{[]byte{0x80}, 0, false},
		{[]byte{0xFF}, 0, false},
	}

	for _, tt := range tests {
		got := isASCII(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isASCII(%v, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestIsPrintable(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte{0x0A}, 0, true},
		{[]byte{0x20}, 0, true},
		{[]byte{0x7E}, 0, true},
		{[]byte{0xC2, 0xA0}, 0, true},
		{[]byte{0x00}, 0, false},
		{[]byte{0x19}, 0, false},
	}

	for _, tt := range tests {
		got := isPrintable(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isPrintable(%v, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestIsZeroChar(t *testing.T) {
	assert.Truef(t, isZeroChar([]byte{0x00}, 0), "isZeroChar should return true for 0x00")
	assert.Falsef(t, isZeroChar([]byte("a"), 0), "isZeroChar should return false for 'a'")
}

func TestIsBOM(t *testing.T) {
	assert.Truef(t, isBOM([]byte{0xEF, 0xBB, 0xBF}, 0), "isBOM should return true for UTF-8 BOM")
	assert.Falsef(t, isBOM([]byte("abc"), 0), "isBOM should return false for regular text")
}

func TestIsSpace(t *testing.T) {
	assert.Truef(t, isSpace([]byte(" "), 0), "isSpace should return true for space")
	assert.Falsef(t, isSpace([]byte("a"), 0), "isSpace should return false for 'a'")
}

func TestIsTab(t *testing.T) {
	assert.Truef(t, isTab([]byte("\t"), 0), "isTab should return true for tab")
	assert.Falsef(t, isTab([]byte(" "), 0), "isTab should return false for space")
}

func TestIsBlank(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte(" "), 0, true},
		{[]byte("\t"), 0, true},
		{[]byte("a"), 0, false},
		{[]byte("\n"), 0, false},
	}

	for _, tt := range tests {
		got := isBlank(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isBlank(%q, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestIsLineBreak(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte("\r"), 0, true},
		{[]byte("\n"), 0, true},
		{[]byte{0xC2, 0x85}, 0, true},
		{[]byte{0xE2, 0x80, 0xA8}, 0, true},
		{[]byte{0xE2, 0x80, 0xA9}, 0, true},
		{[]byte("a"), 0, false},
		{[]byte(" "), 0, false},
	}

	for _, tt := range tests {
		got := isLineBreak(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isLineBreak(%q, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestIsCRLF(t *testing.T) {
	assert.Truef(t, isCRLF([]byte("\r\n"), 0), "isCRLF should return true for CR LF")
	assert.Falsef(t, isCRLF([]byte("\n\x00"), 0), "isCRLF should return false for LF only")
	assert.Falsef(t, isCRLF([]byte("\r\x00"), 0), "isCRLF should return false for CR only")
}

func TestIsBreakOrZero(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte("\r"), 0, true},
		{[]byte("\n"), 0, true},
		{[]byte{0x00}, 0, true},
		{[]byte{0xC2, 0x85}, 0, true},
		{[]byte("a"), 0, false},
	}

	for _, tt := range tests {
		got := isBreakOrZero(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isBreakOrZero(%v, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestIsSpaceOrZero(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte(" "), 0, true},
		{[]byte("\r"), 0, true},
		{[]byte("\n"), 0, true},
		{[]byte{0x00}, 0, true},
		{[]byte("a"), 0, false},
	}

	for _, tt := range tests {
		got := isSpaceOrZero(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isSpaceOrZero(%v, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestIsBlankOrZero(t *testing.T) {
	tests := []struct {
		b        []byte
		i        int
		expected bool
	}{
		{[]byte(" "), 0, true},
		{[]byte("\t"), 0, true},
		{[]byte("\r"), 0, true},
		{[]byte("\n"), 0, true},
		{[]byte{0x00}, 0, true},
		{[]byte("a"), 0, false},
	}

	for _, tt := range tests {
		got := isBlankOrZero(tt.b, tt.i)
		assert.Equalf(t, tt.expected, got, "isBlankOrZero(%v, %d) = %v, want %v", tt.b, tt.i, got, tt.expected)
	}
}

func TestWidth(t *testing.T) {
	tests := []struct {
		b        byte
		expected int
	}{
		{0x00, 1},
		{0x7F, 1},
		{0xC0, 2},
		{0xDF, 2},
		{0xE0, 3},
		{0xEF, 3},
		{0xF0, 4},
		{0xF7, 4},
		{0xF8, 0},
	}

	for _, tt := range tests {
		got := width(tt.b)
		assert.Equalf(t, tt.expected, got, "width(%#x) = %d, want %d", tt.b, got, tt.expected)
	}
}
