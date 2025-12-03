// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"errors"
	"io"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

func TestParserSetReaderError(t *testing.T) {
	parser := NewParser()

	result := parser.setReaderError("test problem", 10, 0x1234)

	assert.Falsef(t, result, "setReaderError() should return false")
	assert.Equalf(t, READER_ERROR, parser.ErrorType, "setReaderError() ErrorType = %v, want READER_ERROR", parser.ErrorType)
	assert.Equalf(t, "test problem", parser.Problem, "setReaderError() Problem = %q, want \"test problem\"", parser.Problem)
	assert.Equalf(t, 10, parser.ProblemOffset, "setReaderError() ProblemOffset = %d, want 10", parser.ProblemOffset)
	assert.Equalf(t, 0x1234, parser.ProblemValue, "setReaderError() ProblemValue = %#x, want 0x1234", parser.ProblemValue)
}

func TestParserDetermineEncodingUTF8(t *testing.T) {
	parser := NewParser()
	input := []byte("\xEF\xBB\xBFtest")
	parser.SetInputString(input)

	assert.Truef(t, parser.determineEncoding(), "determineEncoding() failed")
	assert.Equalf(t, UTF8_ENCODING, parser.encoding, "determineEncoding() encoding = %v, want UTF8_ENCODING", parser.encoding)
	assert.Equalf(t, 3, parser.raw_buffer_pos, "determineEncoding() raw_buffer_pos = %d, want 3 (BOM skipped)", parser.raw_buffer_pos)
}

func TestParserDetermineEncodingUTF16LE(t *testing.T) {
	parser := NewParser()
	input := []byte("\xFF\xFEtest")
	parser.SetInputString(input)

	assert.Truef(t, parser.determineEncoding(), "determineEncoding() failed")
	assert.Equalf(t, UTF16LE_ENCODING, parser.encoding, "determineEncoding() encoding = %v, want UTF16LE_ENCODING", parser.encoding)
}

func TestParserDetermineEncodingUTF16BE(t *testing.T) {
	parser := NewParser()
	input := []byte("\xFE\xFFtest")
	parser.SetInputString(input)

	assert.Truef(t, parser.determineEncoding(), "determineEncoding() failed")
	assert.Equalf(t, UTF16BE_ENCODING, parser.encoding, "determineEncoding() encoding = %v, want UTF16BE_ENCODING", parser.encoding)
}

func TestParserDetermineEncodingDefault(t *testing.T) {
	parser := NewParser()
	input := []byte("test: value")
	parser.SetInputString(input)

	assert.Truef(t, parser.determineEncoding(), "determineEncoding() failed")
	assert.Equalf(t, UTF8_ENCODING, parser.encoding, "determineEncoding() encoding = %v, want UTF8_ENCODING (default)", parser.encoding)
}

func TestParserUpdateRawBuffer(t *testing.T) {
	parser := NewParser()
	input := []byte("test data")
	parser.SetInputString(input)

	assert.Truef(t, parser.updateRawBuffer(), "updateRawBuffer() failed")
	assert.Truef(t, len(parser.raw_buffer) > 0, "updateRawBuffer() should fill raw_buffer")
}

func TestParserUpdateRawBufferEOF(t *testing.T) {
	parser := NewParser()
	parser.SetInputString([]byte(""))

	assert.Truef(t, parser.updateRawBuffer(), "updateRawBuffer() should succeed at EOF")

	parser.eof = true
	assert.Truef(t, parser.updateRawBuffer(), "updateRawBuffer() should return true when already at EOF")
}

type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func TestParserUpdateRawBufferReadError(t *testing.T) {
	parser := NewParser()
	parser.SetInputReader(&errorReader{})

	assert.Falsef(t, parser.updateRawBuffer(), "updateRawBuffer() should fail on read error")
	assert.Equalf(t, READER_ERROR, parser.ErrorType, "updateRawBuffer() ErrorType = %v, want READER_ERROR", parser.ErrorType)
}

func TestParserUpdateBufferUTF8SingleByte(t *testing.T) {
	parser := NewParser()
	input := []byte("abc")
	parser.SetInputString(input)

	assert.Truef(t, parser.updateBuffer(3), "updateBuffer() failed")
	assert.Truef(t, parser.unread >= 3, "updateBuffer() unread = %d, want at least 3", parser.unread)
	assert.Equalf(t, byte('a'), parser.buffer[0], "updateBuffer() buffer[0] = %c, want 'a'", parser.buffer[0])
	assert.Equalf(t, byte('b'), parser.buffer[1], "updateBuffer() buffer[1] = %c, want 'b'", parser.buffer[1])
	assert.Equalf(t, byte('c'), parser.buffer[2], "updateBuffer() buffer[2] = %c, want 'c'", parser.buffer[2])
}

func TestParserUpdateBufferUTF8MultiByte(t *testing.T) {
	parser := NewParser()
	input := []byte("a\xC2\xA9b")
	parser.SetInputString(input)

	assert.Truef(t, parser.updateBuffer(3), "updateBuffer() failed")
	assert.Truef(t, parser.unread >= 3, "updateBuffer() unread = %d, want at least 3", parser.unread)
}

func TestParserUpdateBufferControlCharacterError(t *testing.T) {
	parser := NewParser()
	input := []byte{0x01}
	parser.SetInputString(input)

	assert.Falsef(t, parser.updateBuffer(1), "updateBuffer() should fail on control character")
	assert.Equalf(t, READER_ERROR, parser.ErrorType, "updateBuffer() ErrorType = %v, want READER_ERROR", parser.ErrorType)
}

func TestParserUpdateBufferAllowedControlCharacters(t *testing.T) {
	parser := NewParser()
	input := []byte{0x09, 0x0A, 0x0D}
	parser.SetInputString(input)

	assert.Truef(t, parser.updateBuffer(3), "updateBuffer() should allow tab, LF, CR")
}

func TestParserUpdateBufferPanicWithoutReadHandler(t *testing.T) {
	parser := NewParser()

	assert.PanicMatchesf(t, "read handler must be set", func() {
		_ = parser.updateBuffer(1)
	}, "updateBuffer() without read handler should panic")
}

func TestParserUpdateBufferUTF16LE(t *testing.T) {
	parser := NewParser()
	input := []byte{0xFF, 0xFE, 0x61, 0x00}
	parser.SetInputString(input)

	assert.Truef(t, parser.updateBuffer(1), "updateBuffer() failed for UTF-16LE")
	assert.Equalf(t, UTF16LE_ENCODING, parser.encoding, "encoding = %v, want UTF16LE_ENCODING", parser.encoding)
}

func TestParserUpdateBufferUTF16BE(t *testing.T) {
	parser := NewParser()
	input := []byte{0xFE, 0xFF, 0x00, 0x61}
	parser.SetInputString(input)

	assert.Truef(t, parser.updateBuffer(1), "updateBuffer() failed for UTF-16BE")
	assert.Equalf(t, UTF16BE_ENCODING, parser.encoding, "encoding = %v, want UTF16BE_ENCODING", parser.encoding)
}

func TestYamlStringReadHandler(t *testing.T) {
	parser := NewParser()
	input := []byte("test data")
	parser.input = input
	parser.input_pos = 0

	buffer := make([]byte, 10)
	n, err := yamlStringReadHandler(&parser, buffer)

	assert.Truef(t, errors.Is(err, nil) || errors.Is(err, io.EOF), "yamlStringReadHandler() error = %v, want nil", err)
	assert.Equalf(t, len(input), n, "yamlStringReadHandler() n = %d, want %d", n, len(input))
	assert.DeepEqualf(t, input, buffer[:n], "yamlStringReadHandler() buffer = %q, want %q", buffer[:n], input)
}

func TestYamlStringReadHandlerEOF(t *testing.T) {
	parser := NewParser()
	input := []byte("test")
	parser.input = input
	parser.input_pos = len(input)

	buffer := make([]byte, 10)
	n, err := yamlStringReadHandler(&parser, buffer)

	assert.ErrorIs(t, err, io.EOF)
	assert.Equalf(t, 0, n, "yamlStringReadHandler() n = %d, want 0", n)
}

func TestYamlReaderReadHandler(t *testing.T) {
	parser := NewParser()
	reader := strings.NewReader("test data")
	parser.input_reader = reader

	buffer := make([]byte, 10)
	n, err := yamlReaderReadHandler(&parser, buffer)

	assert.Truef(t, errors.Is(err, nil) || errors.Is(err, io.EOF), "yamlReaderReadHandler() error = %v, want nil", err)
	assert.Truef(t, n > 0, "yamlReaderReadHandler() should read data")
}

func TestParserUpdateBufferSurrogatePairUTF16(t *testing.T) {
	parser := NewParser()
	input := []byte{
		0xFF, 0xFE,
		0x3D, 0xD8, 0x4A, 0xDC,
	}
	parser.SetInputString(input)

	assert.Truef(t, parser.updateBuffer(1), "updateBuffer() failed for UTF-16 surrogate pair")
	assert.Truef(t, parser.unread >= 1, "updateBuffer() should decode surrogate pair")
}

func TestParserUpdateBufferInvalidSurrogatePair(t *testing.T) {
	parser := NewParser()
	input := []byte{
		0xFF, 0xFE,
		0x3D, 0xD8, 0x00, 0x00,
	}
	parser.SetInputString(input)

	assert.Falsef(t, parser.updateBuffer(1), "updateBuffer() should fail on invalid surrogate pair")
	assert.Equalf(t, READER_ERROR, parser.ErrorType, "ErrorType = %v, want READER_ERROR", parser.ErrorType)
}

func TestParserUpdateBufferUnexpectedLowSurrogate(t *testing.T) {
	parser := NewParser()
	input := []byte{
		0xFF, 0xFE,
		0x00, 0xDC, 0x00, 0x00,
	}
	parser.SetInputString(input)

	assert.Falsef(t, parser.updateBuffer(1), "updateBuffer() should fail on unexpected low surrogate")
	assert.Equalf(t, READER_ERROR, parser.ErrorType, "ErrorType = %v, want READER_ERROR", parser.ErrorType)
}
