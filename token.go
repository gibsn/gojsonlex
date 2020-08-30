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

	unescapedBuf := make([]byte, 0, utf8.UTFMax)

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

			runeAsUint, err := strconv.ParseUint(unsafeStringFromBytes(utf16Sequence), 16, 16)
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

			n := utf8.EncodeRune(unescapedBuf[:cap(unescapedBuf)], outRune)
			unescapedBuf = unescapedBuf[:n]
		case pendingEscapedSymbol:
			pendingEscapedSymbol = false

			switch c {
			case 'u', 'U':
				pendingUnicodeBytes = utf16SequenceLength
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
				return nil, fmt.Errorf("invalid escape sequence \\%c", c)
			}
		case c == '\\':
			pendingEscapedSymbol = true
			continue
		default:
			unescapedBuf = append(unescapedBuf, c)
		}

		if pendingSecondUTF16SeqPoint {
			return nil, fmt.Errorf(
				"missing second sequence point for %s",
				string(input[writeIter:readIter]),
			)
		}

		if pendingEscapedSymbol || pendingUnicodeBytes > 0 {
			return nil, fmt.Errorf(
				"incomplete escape sequence %s",
				string(input[writeIter:readIter]),
			)
		}

		for i := 0; i < len(unescapedBuf); i++ {
			input[writeIter+i] = unescapedBuf[i]
		}

		// copy(input[writeIter:], unescapedBuf)

		writeIter += len(unescapedBuf)
		unescapedBuf = unescapedBuf[:0]
	}

	if pendingSecondUTF16SeqPoint {
		return nil, fmt.Errorf(
			"missing second sequence point for %s",
			string(input[writeIter:readIter]),
		)
	}

	if pendingEscapedSymbol || pendingUnicodeBytes > 0 {
		return nil, fmt.Errorf(
			"incomplete escape sequence %s",
			string(input[writeIter:readIter]),
		)
	}

	return input[:writeIter], nil
}

// unescapeUnicode expects the input format to be as (hex_digit){4}(\u(hex_digit){4})?.
// 'in' then is unescaped to 'out' by being converting to UTF-8 bytes.
// func unescapeUnicode(in []byte, out []byte) (int, int, error) {
// 	writeIter := 0
// 	readIter := 0
//
// 	pendingDigits := utf16SequenceLength
// 	isASurrogatePair := false
//
// 	for c, i := range in {
// 		readIter++
//
// 		switch {
// 		case pendingDigits > 0:
// 			pendingDigits--
// 			if pendingDigits != 0 {
// 				continue
// 			}
//
// 			seqPoint, err := strconv.ParseUint(unsafeStringFromBytes(in[:4]), 16, 16)
// 			if err != nil {
// 				return 0, 0, fmt.Errorf("invalid unicode sequence \\u%s", in[:readIter])
// 			}
//
// 			if utf16.IsSurrogate(rune(seqPoint)) {
// 				isASurrogatePair = true
// 			}
// 		}
// 		// case c == '\\':
// 		// 	pendingUSymbol
// 	}
// 	for i := 0; i < utf16SequenceLength; i++ {
// 		readIter++
// 	}
//
// }
//
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
// func UTF16ToUTF8Bytes(in []byte, out []byte) ([]byte, error) {
// 	if len(in) != unicodeSequenceLength {
// 		return nil, fmt.Errorf("unicode sequence must consist of exactly %d symbols",
// 			unicodeSequenceLength,
// 		)
// 	}
//
// 	inAsUint, err := strconv.ParseUint(unsafeStringFromBytes(in[:4]), 16, 16)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid unicode sequence %c%c%c%c", in[0], in[1], in[2], in[3])
// 	}
//
// 	// TODO wtf is surrogate pair
// 	outRune := rune(inAsUint)
// 	// outRune := utf16.Decode([]uint16{uint16(inAsUint)})[0]
// 	// if outRune == unicode.ReplacementChar {
// 	// 	return nil, fmt.Errorf("invalid unicode sequence %c%c%c%c", in[0], in[1], in[2], in[3])
// 	// }
//
// 	n := utf8.EncodeRune(out[:cap(out)], outRune)
//
// 	return out[:n], nil
// }
