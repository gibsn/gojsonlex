package gojsonlex

import (
	"testing"
)

type stringDeepCopyTestCase struct {
	input string
}

func TestStringDeepCopy(t *testing.T) {
	testcases := []stringDeepCopyTestCase{
		{"hello, world!"}, {""},
	}

	for _, testcase := range testcases {
		currIn := testcase.input
		currOut := StringDeepCopy(testcase.input)

		if currIn != currOut {
			t.Errorf("testcase '%s': got '%s'", currIn, currOut)
		}
	}
}

type unescapeBytesInplaceTestCase struct {
	input  []byte
	output []byte
}

func TestUnescapeBytesInplace(t *testing.T) {
	testcases := []unescapeBytesInplaceTestCase{
		{[]byte(""), []byte("")},
		{[]byte("a"), []byte("a")},
		{[]byte("hello\\nworld"), []byte("hello\nworld")},
		{[]byte("hello\\rworld"), []byte("hello\rworld")},
		{[]byte("hello\\tworld"), []byte("hello\tworld")},
		{[]byte("hello\\bworld"), []byte("hello\bworld")},
		{[]byte("hello\\fworld"), []byte("hello\fworld")},
		{[]byte("hello\\\\world"), []byte("hello\\world")},
		{[]byte("hello\\/world"), []byte("hello/world")},
		{[]byte("hello\\\"world"), []byte("hello\"world")},
		{[]byte("\\\"hello world\\\""), []byte("\"hello world\"")},
		{
			[]byte("hello \\u043f\\u0440\\u0438\\u0432\\u0435\\u0442\\u0020\\u043c\\u0438\\u0440 world"),
			[]byte("hello привет мир world"),
		},
		{[]byte("hello \\UD83D\\UDCA9 world"), []byte("hello 💩 world")},
	}
	for _, testcase := range testcases {
		currIn := string(testcase.input) // making a copy
		currOut, err := UnescapeBytesInplace(testcase.input)
		if err != nil {
			t.Errorf("testcase '%s': %v", currIn, err)
			continue
		}

		if string(testcase.output) != string(currOut) {
			t.Errorf("testcase '%s': got '%s', expected '%s'",
				currIn, string(currOut), string(testcase.output))
		}
	}
}

func TestUnescapeBytesInplaceFails(t *testing.T) {
	testcases := []unescapeBytesInplaceTestCase{
		{input: []byte("\\")},
		// unknown escape sequnce
		{input: []byte("\\a")},
		// not enough symbols
		{input: []byte("\\u043")},
		// wrong utf16 surrogate pair
		{input: []byte("hello \\ud83d\\ufca9 world")},
		// missing second elem in a utf16 surrogate pair
		{input: []byte("hello \\ud83d world")},
	}
	for _, testcase := range testcases {
		currIn := string(testcase.input) // making a copy

		_, err := UnescapeBytesInplace(testcase.input)
		if err == nil {
			t.Errorf("testcase '%s': must have failed", currIn)
		}
	}
}

type hexBytesToUintTestcase struct {
	input  []byte
	output uint64
}

func TestHexBytesToUint(t *testing.T) {
	testcases := []hexBytesToUintTestcase{
		{
			input:  []byte("000f"),
			output: 15,
		},
		{
			input:  []byte("003F"),
			output: 63,
		},
		{
			input:  []byte("043f"),
			output: 1087,
		},
		{
			input:  []byte("543f"),
			output: 21567,
		},
	}
	for _, testcase := range testcases {
		out, err := HexBytesToUint(testcase.input)
		if err != nil {
			t.Errorf("testcase '%s': %v", string(testcase.input), err)
			continue
		}

		if testcase.output != out {
			t.Errorf("testcase '%s': got '%d', expected '%d'",
				testcase.input, out, testcase.output)
		}
	}
}

func TestHexBytesToUintFails(t *testing.T) {
	testcases := []unescapeBytesInplaceTestCase{
		{
			input: []byte("043z"),
		},
	}
	for _, testcase := range testcases {
		_, err := HexBytesToUint(testcase.input)
		if err == nil {
			t.Errorf("testcase '%s': must have failed", testcase.input)
		}
	}
}

type canAppearInNumberTestCase struct {
	input  rune
	output bool
}

func TestCanAppearInNumber(t *testing.T) {
	testcases := []canAppearInNumberTestCase{
		{'0', true},
		{'9', true},
		{'-', true},
		{'.', true},
		{'+', true},
		{'e', true},
		{'E', true},
		{'е', false}, // russian 'е'
		{'*', false},
	}

	for _, testcase := range testcases {
		currOut := CanAppearInNumber(testcase.input)

		if testcase.output != currOut {
			t.Errorf("testcase '%c': got '%t', expected '%t'",
				testcase.input, currOut, testcase.output)
		}
	}
}

// TODO tests for IsDelim
