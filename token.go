package gojsonlex

import (
	"reflect"
	"unsafe"
)

type TokenType byte

const (
	lexerTokenTypeDelim TokenType = iota
	lexerTokenTypeString
	lexerTokenTypeNumber
	lexerTokenTypeBool
	lexerTokenTypeNull
)

func unsafeStringFromBytes(arr []byte) string {
	slice := (*reflect.SliceHeader)(unsafe.Pointer(&arr))
	str := (*reflect.StringHeader)(unsafe.Pointer(slice))
	str.Data = slice.Data
	str.Len = slice.Len

	return *(*string)(unsafe.Pointer(str))
}

// StringDeepCopy creates a copy of the given string with it's own underlying bytearray.
// Use this function to make a copy of string returned by Token()
func StringDeepCopy(s string) string {
	return unsafeStringFromBytes([]byte(s))
}

// IsDelim reports whether the given rune is a JSON delimiter
func IsDelim(c rune) bool {
	switch c {
	case '{':
		fallthrough
	case '}':
		fallthrough
	case '[':
		fallthrough
	case ']':
		fallthrough
	case ':':
		fallthrough
	case ',':
		return true
	}

	return false
}
