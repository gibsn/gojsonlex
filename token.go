package gojsonlex

import (
	"fmt"
	"reflect"
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
	utf16SequenceLength  = 4 // in digits
	utf16MaxWordsForRune = 2
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
	// state modificators
	var (
		pendingEscapedSymbol bool
		pendingUnicodeBytes  byte
	)

	// since in UTF-16 rune may be encoded by either 1 or 2 words we
	// may have to remember the previous word
	var (
		pendingSecondUTF16SeqPoint bool
		firstUTF16SeqPoint         rune
	)

	var (
		writeIter int
		readIter  int
	)

	for ; readIter < len(input); readIter++ {
		c := input[readIter]

		switch {
		case pendingUnicodeBytes > 0:
			pendingUnicodeBytes--
			if pendingUnicodeBytes != 0 {
				continue
			}

			// processing the last byte of unicode sequence
			utf16Sequence := input[readIter+1-utf16SequenceLength : readIter+1]

			runeAsUint, err := HexBytesToUint(utf16Sequence)
			if err != nil {
				return nil, fmt.Errorf("invalid unicode sequence \\u%s", utf16Sequence)
			}

			outRune := rune(runeAsUint)

			if utf16.IsSurrogate(outRune) && !pendingSecondUTF16SeqPoint {
				pendingSecondUTF16SeqPoint = true
				firstUTF16SeqPoint = outRune
				continue
			}

			if pendingSecondUTF16SeqPoint { // then we got a second elem and can decode now
				outRune = utf16.DecodeRune(firstUTF16SeqPoint, outRune)
				if outRune == unicode.ReplacementChar {
					return nil, fmt.Errorf("invalid surrogate pair %x%x", firstUTF16SeqPoint, outRune)
				}

				pendingSecondUTF16SeqPoint = false
			}

			n := utf8.EncodeRune(input[writeIter:], outRune)
			writeIter += n
		case pendingEscapedSymbol:
			pendingEscapedSymbol = false

			if c == 'u' || c == 'U' {
				pendingUnicodeBytes = utf16SequenceLength
				continue
			}

			if pendingSecondUTF16SeqPoint {
				return nil, fmt.Errorf("missing second sequence point for %x", firstUTF16SeqPoint)
			}

			var outRune byte

			switch c {
			case 'n':
				outRune = '\n'
			case 'r':
				outRune = '\r'
			case 't':
				outRune = '\t'
			case 'b':
				outRune = '\b'
			case 'f':
				outRune = '\f'
			case '\\':
				outRune = '\\'
			case '/':
				outRune = '/'
			case '"':
				outRune = '"'
			default:
				return nil, fmt.Errorf("invalid escape sequence \\%c", c)
			}

			input[writeIter] = outRune
			writeIter++
		case c == '\\':
			pendingEscapedSymbol = true
			continue
		default:
			input[writeIter] = c
			writeIter++
		}
	}

	if pendingSecondUTF16SeqPoint {
		return nil, fmt.Errorf("missing second sequence point for %x", firstUTF16SeqPoint)
	}

	if pendingEscapedSymbol || pendingUnicodeBytes > 0 {
		return nil, fmt.Errorf(
			"incomplete escape sequence %s",
			string(input[writeIter:readIter]),
		)
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

func HexBytesToUint(in []byte) (result uint64, err error) {
	for _, c := range in {
		result *= 0x10

		var v byte

		switch {
		case '0' <= c && c <= '9':
			v = c - '0'
		case 'a' <= c && c <= 'f':
			v = c - 'a' + 10
		case 'A' <= c && c <= 'F':
			v = c - 'A' + 10
		default:
			return 0, fmt.Errorf("'%s' is not a hex number", string(in))
		}

		result += uint64(v)
	}

	return result, nil
}
