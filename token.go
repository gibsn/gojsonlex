package gojsonlex

import (
	"reflect"
	"unicode"
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

func unsafeStringFromBytes(arr []byte) string {
	slice := (*reflect.SliceHeader)(unsafe.Pointer(&arr))
	str := (*reflect.StringHeader)(unsafe.Pointer(slice))
	str.Data = slice.Data
	str.Len = slice.Len

	return *(*string)(unsafe.Pointer(str))
}

// TODO support UTF16 unescaping
func unescapeBytesInplace(data []byte) []byte {
	offset := 0

	pendingEscapedSymbol := false
	// pendingUnicodeRune := false
	//
	// unicodeRuneBytesCounter := 0
	for i, r := range data {
		if pendingEscapedSymbol {
			pendingEscapedSymbol = false

			switch r {
			case 'u', 'U':
				offset-- // to save original sequence
				// unicodeRuneBytesCounter = 0
			case 'n':
				r = '\n'
			case 'r':
				r = '\r'
			case 't':
				r = '\t'
			case 'b':
				r = '\b'
			case 'f':
				r = '\f'
			case '\\':
				r = '\\'
			case '/':
				r = '/'
			case '"':
				r = '"'
			}
		} else if r == '\\' {
			pendingEscapedSymbol = true
			offset += 1
			continue
		}
		// } else if pendingUnicodeRune {
		// 	unicodeRuneBytesCounter++
		// 	offset++
		// 	continue
		// }

		data[i-offset] = r
	}

	return data[:len(data)-offset]
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
