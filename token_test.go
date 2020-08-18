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
		{
			input:  []byte(""),
			output: []byte(""),
		},
		{
			input:  []byte("a"),
			output: []byte("a"),
		},
		{
			input:  []byte("hello\\nworld"),
			output: []byte("hello\nworld"),
		},
		{
			input:  []byte("hello\\rworld"),
			output: []byte("hello\rworld"),
		},
		{
			input:  []byte("hello\\tworld"),
			output: []byte("hello\tworld"),
		},
		{
			input:  []byte("hello\\bworld"),
			output: []byte("hello\bworld"),
		},
		{
			input:  []byte("hello\\fworld"),
			output: []byte("hello\fworld"),
		},
		{
			input:  []byte("hello\\\\world"),
			output: []byte("hello\\world"),
		},
		{
			input:  []byte("hello\\/world"),
			output: []byte("hello/world"),
		},
		{
			input:  []byte("hello\\\"world"),
			output: []byte("hello\"world"),
		},
		{
			input:  []byte("\\\"hello world\\\""),
			output: []byte("\"hello world\""),
		},
		{
			input:  []byte("hello\\u123aworld"),
			output: []byte("hello\\u123aworld"),
		},
	}
	for _, testcase := range testcases {
		currIn := string(testcase.input) // making a copy
		currOut := unescapeBytesInplace(testcase.input)

		if string(testcase.output) != string(currOut) {
			t.Errorf("testcase '%s': got '%s', expected '%s'",
				currIn, string(currOut), string(testcase.output))
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
