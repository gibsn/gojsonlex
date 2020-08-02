package gojsonlex

import (
	"fmt"
	"reflect"
	"strconv"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

type TokenType byte

const (
	LexerTokenTypeDelim TokenType = iota
	LexerTokenTypeString
	LexerTokenTypeNumber
	LexerTokenTypeBool
	LexerTokenTypeNull
)

const (
	unicodeSequenceLength = 4
)

func (t TokenType) String() string {
	switch t {
	case LexerTokenTypeDelim:
		return "delim"
	case LexerTokenTypeString:
		return "string"
	case LexerTokenTypeNumber:
		return "number"
	case LexerTokenTypeBool:
		return "bool"
	case LexerTokenTypeNull:
		return "null"
	}

	panic("unknown token type")
}

func unsafeStringFromBytes(arr []byte) string {
	slice := (*reflect.SliceHeader)(unsafe.Pointer(&arr))
	str := (*reflect.StringHeader)(unsafe.Pointer(slice))
	str.Data = slice.Data
	str.Len = slice.Len

	return *(*string)(unsafe.Pointer(str))
}

// TODO comment
func unescapeBytesInplace(input []byte) ([]byte, error) {
	// because presentation of escaped symbols in the
	// input and output arrays may differ in size
	writeIter := 0

	var (
		pendingEscapedSymbol bool
		pendingUnicodeBytes  byte
		err                  error
	)

	unescapedBuf := make([]byte, 0, utf8.UTFMax)

	for i, r := range input {
		switch {
		case pendingUnicodeBytes > 0:
			pendingUnicodeBytes--

			if pendingUnicodeBytes != 0 {
				continue
			}

			// processing the last byte of unicode sequence
			utf16Sequence := input[i-unicodeSequenceLength+1 : i+1]

			unescapedBuf, err = UTF16ToUTF8Bytes(utf16Sequence, unescapedBuf[:])
			if err != nil {
				return nil, fmt.Errorf("could not unescape string '%s': %w", string(input), err)
			}
		case pendingEscapedSymbol:
			pendingEscapedSymbol = false

			switch r {
			case 'u', 'U':
				pendingUnicodeBytes = unicodeSequenceLength
				continue
			case 'n':
				unescapedBuf = append(unescapedBuf, '\n')
			case 'r':
				unescapedBuf = append(unescapedBuf, '\r')
			case 't':
				unescapedBuf = append(unescapedBuf, '\t')
			case 'b':
				unescapedBuf = append(unescapedBuf, '\b')
			case 'f':
				unescapedBuf = append(unescapedBuf, '\f')
			case '\\':
				unescapedBuf = append(unescapedBuf, '\\')
			case '/':
				unescapedBuf = append(unescapedBuf, '/')
			case '"':
				unescapedBuf = append(unescapedBuf, '"')
			default:
				return nil, fmt.Errorf("invalid escape sequence \\%c", r)
			}
		case r == '\\':
			pendingEscapedSymbol = true
			continue
		default:
			unescapedBuf = append(unescapedBuf, r)
		}

		copy(input[writeIter:], unescapedBuf)

		writeIter += len(unescapedBuf)
		unescapedBuf = unescapedBuf[:0]
	}

	if pendingEscapedSymbol || pendingUnicodeBytes > 0 {
		return nil, fmt.Errorf("incomplete escape sequence %s", string(input[writeIter:]))
	}

	return input[:writeIter], nil
}

// StringDeepCopy creates a copy of the given string with it's own underlying bytearray.
// Use this function to make a copy of string returned by Token()
func StringDeepCopy(s string) string {
	return unsafeStringFromBytes([]byte(s))
}

// IsDelim reports whether the given rune is a JSON delimiter
func IsDelim(c rune) bool {
	switch c {
	case '{', '}', '[', ']', ':', ',':
		return true
	}

	return false
}

func IsValidEscapedSymbol(c rune) bool {
	switch c {
	case 'n', 'r', 't', 'b', 'f', '\\', '/', '"', 'u', 'U':
		return true
	}

	return false
}

func IsHexDigit(c rune) bool {
	switch {
	case unicode.IsDigit(c):
		fallthrough
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true

	}
	return false
}

// UTF16ToUTF8Bytes returns the shrinked output buffer and error if any
func UTF16ToUTF8Bytes(in []byte, out []byte) ([]byte, error) {
	if len(in) != unicodeSequenceLength {
		return nil, fmt.Errorf("unicode sequence must consist of exactly %d symbols",
			unicodeSequenceLength,
		)
	}

	inAsUint, err := strconv.ParseUint(unsafeStringFromBytes(in[:4]), 16, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid unicode sequence %c%c%c%c", in[0], in[1], in[2], in[3])
	}

	outRune := utf16.Decode([]uint16{uint16(inAsUint)})[0]
	if outRune == unicode.ReplacementChar {
		return nil, fmt.Errorf("invalid unicode sequence %c%c%c%c", in[0], in[1], in[2], in[3])
	}

	n := utf8.EncodeRune(out[:cap(out)], outRune)

	return out[:n], nil
}
