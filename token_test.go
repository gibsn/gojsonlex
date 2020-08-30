package gojsonlex

import (
	"testing"
	// "unicode/utf8"
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
			input:  []byte("hello \\u043f\\u0440\\u0438\\u0432\\u0435\\u0442\\u0020\\u043c\\u0438\\u0440 world"),
			output: []byte("hello привет мир world"),
		},
		{
			input:  []byte("hello \\UD83D\\UDCA9 world"),
			output: []byte("hello 💩 world"),
		},
	}
	for _, testcase := range testcases {
		currIn := string(testcase.input) // making a copy
		currOut, err := unescapeBytesInplace(testcase.input)
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
		{
			input: []byte("\\"),
		},
		// unknown escape sequnce
		{
			input: []byte("\\a"),
		},
		// not enough symbols
		{
			input: []byte("\\u043"),
		},
		// wrong utf16 surrogate pair
		{
			input: []byte("hello \\ud83d\\ufca9 world"),
		},
		// missing second elem in a utf16 surrogate pair
		{
			input: []byte("hello \\ud83d world"),
		},
	}
	for _, testcase := range testcases {
		currIn := string(testcase.input) // making a copy

		_, err := unescapeBytesInplace(testcase.input)
		if err == nil {
			t.Errorf("testcase '%s': must have failed", currIn)
		}
	}
}

// type utf16ToUTF8BytesTestcase struct {
// 	input  []byte
// 	output string
// }
//
// func TestUTF16ToUTF8Bytes(t *testing.T) {
// 	testcases := []utf16ToUTF8BytesTestcase{
// 		{
// 			input:  []byte("041f"),
// 			output: "П",
// 		},
// 	}
// 	for _, testcase := range testcases {
// 		output := make([]byte, 0, utf8.UTFMax)
//
// 		output, err := UTF16ToUTF8Bytes(testcase.input, output[:])
// 		if err != nil {
// 			t.Errorf("testcase '%s': %v", testcase.input, err)
// 			continue
// 		}
//
// 		if string(output) != testcase.output {
// 			t.Errorf("testcase '%s': expected '%s', got '%s'",
// 				testcase.input, testcase.output, output)
// 			continue
// 		}
// 	}
// }
//
// // TODO tests for wrong surrogate pair
// func TestUTF16ToUTF8BytesFails(t *testing.T) {
// 	testcases := []utf16ToUTF8BytesTestcase{
// 		{
// 			input:  []byte(""),
// 			output: "",
// 		},
// 		// not enough symbols
// 		{
// 			input:  []byte("\\u043"),
// 			output: "",
// 		},
// 	}
// 	for _, testcase := range testcases {
// 		var output [utf8.UTFMax]byte
//
// 		_, err := UTF16ToUTF8Bytes(testcase.input, output[:])
// 		if err == nil {
// 			t.Errorf("testcase '%s': must have failed", testcase.input)
// 			continue
// 		}
// 	}
// }

// TODO tests for IsDelim
