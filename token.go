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
func unescapeBytesInplace(data []byte) ([]byte, error) {
	// because presentation of escaped symbols in the
	// input and output arrays may differ in size
	offset := 0

	pendingEscapedSymbol := false
	pendingUnicodeBytes := -1

	unescaped := make([]byte, 0, utf8.UTFMax)

	for i, r := range data {
		switch {
		case pendingUnicodeBytes == 0: // processing the last byte of unicode sequence
			runeLen, err := UTF16ToUTF8Bytes(data[i-unicodeSequenceLength:i], unescaped[:])
			if err != nil {
				return nil, fmt.Errorf("could not unescape string '%s': %w", string(data), err)
			}

			offset += unicodeSequenceLength - runeLen
			pendingUnicodeBytes = -1
		case pendingUnicodeBytes > 0:
			pendingUnicodeBytes--
			continue
		case pendingEscapedSymbol:
			pendingEscapedSymbol = false

			switch r {
			case 'u', 'U':
				offset++
				pendingUnicodeBytes = unicodeSequenceLength
				continue
			case 'n':
				unescaped = append(unescaped, '\n')
			case 'r':
				unescaped = append(unescaped, '\r')
			case 't':
				unescaped = append(unescaped, '\t')
			case 'b':
				unescaped = append(unescaped, '\b')
			case 'f':
				unescaped = append(unescaped, '\f')
			case '\\':
				unescaped = append(unescaped, '\\')
			case '/':
				unescaped = append(unescaped, '/')
			case '"':
				unescaped = append(unescaped, '"')
			default:
				// return 0
			}
		case r == '\\':
			pendingEscapedSymbol = true
			offset++
			continue
		default:
			unescaped = append(unescaped, r)
		}

		copy(data[i-offset:], unescaped)
		unescaped = unescaped[:0]
	}

	return data[:len(data)-offset], nil
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

// UTF16ToUTF8Bytes returns the length of the output rune in bytes and error if any
func UTF16ToUTF8Bytes(in []byte, out []byte) (int, error) {
	if len(in) != unicodeSequenceLength {
		return 0, fmt.Errorf("unicode sequence must consist of exactly %d symbols",
			unicodeSequenceLength,
		)
	}

	in1, err := strconv.ParseUint(unsafeStringFromBytes(in[:2]), 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid unicode sequence %c%c%c%c", in[0], in[1], in[2], in[3])
	}
	in2, err := strconv.ParseUint(unsafeStringFromBytes(in[2:]), 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid unicode sequence %c%c%c%c", in[0], in[1], in[2], in[3])
	}

	outRune := utf16.DecodeRune(rune(in1), rune(in2))
	if outRune == unicode.ReplacementChar {
		return 0, fmt.Errorf("invalid utf16 surrogate pair %x:%x", in1, in2)
	}

	n := utf8.EncodeRune(out, outRune)

	return n, nil
}
