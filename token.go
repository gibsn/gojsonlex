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

type bytesUnescaper struct {
	writeIter int
	readIter  int
	input     []byte

	// state modificators
	pendingEscapedSymbol bool
	pendingUnicodeBytes  byte

	// since in UTF-16 rune may be encoded by either 1 or 2 words we
	// may have to remember the previous word
	pendingSecondUTF16SeqPoint bool
	firstUTF16SeqPoint         rune
}

// UnescapeBytesInplace iterates over the given slice of byte unescaping all
// escaped symbols inplace. Since the unescaped symbols take less space the shrinked
// slice of bytes is returned
func UnescapeBytesInplace(input []byte) ([]byte, error) {
	u := bytesUnescaper{
		input: input,
	}

	return u.doUnescaping()
}

func (u *bytesUnescaper) processUnicodeByte(c byte) error {
	u.pendingUnicodeBytes--
	if u.pendingUnicodeBytes != 0 {
		return nil
	}

	// processing the last byte of unicode sequence
	utf16Sequence := u.input[u.readIter+1-utf16SequenceLength : u.readIter+1]

	runeAsUint, err := HexBytesToUint(utf16Sequence)
	if err != nil {
		return fmt.Errorf("invalid unicode sequence \\u%s", utf16Sequence)
	}

	outRune := rune(runeAsUint)

	if utf16.IsSurrogate(outRune) && !u.pendingSecondUTF16SeqPoint {
		u.pendingSecondUTF16SeqPoint = true
		u.firstUTF16SeqPoint = outRune
		return nil
	}

	if u.pendingSecondUTF16SeqPoint { // then we got a second elem and can decode now
		outRune = utf16.DecodeRune(u.firstUTF16SeqPoint, outRune)
		if outRune == unicode.ReplacementChar {
			return fmt.Errorf("invalid surrogate pair %x%x", u.firstUTF16SeqPoint, outRune)
		}

		u.pendingSecondUTF16SeqPoint = false
	}

	n := utf8.EncodeRune(u.input[u.writeIter:], outRune)
	u.writeIter += n

	return nil
}

func (u *bytesUnescaper) processSpecialByte(c byte) error {
	u.pendingEscapedSymbol = false

	if c == 'u' || c == 'U' {
		u.pendingUnicodeBytes = utf16SequenceLength
		return nil
	}

	if u.pendingSecondUTF16SeqPoint {
		return fmt.Errorf("missing second sequence point for %x", u.firstUTF16SeqPoint)
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
		return fmt.Errorf("invalid escape sequence \\%c", c)
	}

	u.input[u.writeIter] = outRune
	u.writeIter++

	return nil
}

func (u *bytesUnescaper) processBackSlashByte(c byte) {
	u.pendingEscapedSymbol = true
}

func (u *bytesUnescaper) processRegularByte(c byte) {
	u.input[u.writeIter] = c
	u.writeIter++
}

func (u *bytesUnescaper) terminate() error {
	if u.pendingSecondUTF16SeqPoint {
		return fmt.Errorf("missing second sequence point for %x", u.firstUTF16SeqPoint)
	}

	if u.pendingEscapedSymbol || u.pendingUnicodeBytes > 0 {
		return fmt.Errorf("incomplete escape sequence %s", string(u.input[u.writeIter:]))
	}

	return nil
}

func (u *bytesUnescaper) doUnescaping() (_ []byte, err error) {
	for u.readIter = 0; u.readIter < len(u.input); u.readIter++ {
		currByte := u.input[u.readIter]

		switch {
		case u.pendingUnicodeBytes > 0:
			err = u.processUnicodeByte(currByte)
		case u.pendingEscapedSymbol:
			err = u.processSpecialByte(currByte)
		case currByte == '\\':
			u.processBackSlashByte(currByte)
		default:
			u.processRegularByte(currByte)
		}

		if err != nil {
			return nil, err
		}
	}

	if err = u.terminate(); err != nil {
		return nil, err
	}

	return u.input[:u.writeIter], nil
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
