package gojsonlex

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

type jsonLexerOutputToken struct {
	token     interface{}
	tokenType TokenType
}

type jsonLexerTestCase struct {
	input      string
	output     []jsonLexerOutputToken
	skipDelims bool
}

func TestJSONLexer(t *testing.T) {
	testcases := []jsonLexerTestCase{
		// tests for strings
		{
			input: `{"hello":"world"}`,
			output: []jsonLexerOutputToken{
				{
					"hello",
					LexerTokenTypeString,
				},
				{
					"world",
					LexerTokenTypeString,
				},
			},
		},
		{
			input: `{"liveness_info" : { "tstamp" : "2020-05-06T12:57:14.193447Z" }}`,
			output: []jsonLexerOutputToken{
				{
					"liveness_info",
					LexerTokenTypeString,
				},
				{
					"tstamp",
					LexerTokenTypeString,
				},
				{
					"2020-05-06T12:57:14.193447Z",
					LexerTokenTypeString,
				},
			},
		},
		{
			input: `{"ua": "\"SomeUA\""}`,
			output: []jsonLexerOutputToken{
				{
					"ua",
					LexerTokenTypeString,
				},
				{
					"\"SomeUA\"",
					LexerTokenTypeString,
				},
			},
		},
		// tests for numbers
		{
			input: `{"hello":{"0": 10, "1": 11.0}}`,
			output: []jsonLexerOutputToken{
				{
					"hello",
					LexerTokenTypeString,
				},
				{
					"0",
					LexerTokenTypeString,
				},
				{
					float64(10),
					LexerTokenTypeNumber,
				},
				{
					"1",
					LexerTokenTypeString,
				},
				{
					float64(11),
					LexerTokenTypeNumber,
				},
			},
		},
		// {
		// 	input: `{"hello":{"0": -10, "1": -11.0}}`,
		// 	output: []jsonLexerOutputToken{
		// 		{
		// 			"hello",
		// 			LexerTokenTypeString,
		// 		},
		// 		{
		// 			"0",
		// 			LexerTokenTypeString,
		// 		},
		// 		{
		// 			float64(-10),
		// 			LexerTokenTypeNumber,
		// 		},
		// 		{
		// 			"1",
		// 			LexerTokenTypeString,
		// 		},
		// 		{
		// 			float64(-11),
		// 			LexerTokenTypeNumber,
		// 		},
		// 	},
		// },
		// tests for special symbols
		{
			input: `{"ua": "\"\"Some\nWeird\tUA\"\""}`,
			output: []jsonLexerOutputToken{
				{
					"ua",
					LexerTokenTypeString,
				},
				{
					"\"\"Some\nWeird\tUA\"\"",
					LexerTokenTypeString,
				},
			},
		},
		// tests for Unicode
		{
			input: `{"desc": "\u041f\u0440\u043e\u0432\u0435\u0440\u043a\u0430 \u043f\u043e\u0447\u0442\u044b"}`,
			output: []jsonLexerOutputToken{
				{
					"desc",
					LexerTokenTypeString,
				},
				{
					"Проверка почты",
					LexerTokenTypeString,
				},
			},
		},
		// tests for Null
		{
			input: `{"ua": Null}`,
			output: []jsonLexerOutputToken{
				{
					"ua",
					LexerTokenTypeString,
				},
				{
					nil,
					LexerTokenTypeNull,
				},
			},
		},
		{
			input: `{"ua": null}`,
			output: []jsonLexerOutputToken{
				{
					"ua",
					LexerTokenTypeString,
				},
				{
					nil,
					LexerTokenTypeNull,
				},
			},
		},
		// tests for Bool
		{
			input: `{"isValid": true}`,
			output: []jsonLexerOutputToken{
				{
					"isValid",
					LexerTokenTypeString,
				},
				{
					true,
					LexerTokenTypeBool,
				},
			},
		},
		{
			input: `{"isValid": True}`,
			output: []jsonLexerOutputToken{
				{
					"isValid",
					LexerTokenTypeString,
				},
				{
					true,
					LexerTokenTypeBool,
				},
			},
		},
		{
			input: `{"isValid": false}`,
			output: []jsonLexerOutputToken{
				{
					"isValid",
					LexerTokenTypeString,
				},
				{
					false,
					LexerTokenTypeBool,
				},
			},
		},
		{
			input: `{"isValid": False}`,
			output: []jsonLexerOutputToken{
				{
					"isValid",
					LexerTokenTypeString,
				},
				{
					false,
					LexerTokenTypeBool,
				},
			},
		},
		{
			input: `{"isValid": False}`,
			output: []jsonLexerOutputToken{
				{
					"isValid",
					LexerTokenTypeString,
				},
				{
					false,
					LexerTokenTypeBool,
				},
			},
		},
		{
			input: `{"delta": 3.14, "temperature": -52, "distance": 1.57e+10, "size": 1.2E-10}`,
			output: []jsonLexerOutputToken{
				{"delta", LexerTokenTypeString},
				{float64(3.14), LexerTokenTypeNumber},
				{"temperature", LexerTokenTypeString},
				{float64(-52), LexerTokenTypeNumber},
				{"distance", LexerTokenTypeString},
				{float64(1.57e10), LexerTokenTypeNumber},
				{"size", LexerTokenTypeString},
				{float64(1.2e-10), LexerTokenTypeNumber},
			},
		},
		{
			// should not be supported according to json.org
			input: `{"delta1": .314, "delta2": 314.}`,
			output: []jsonLexerOutputToken{
				{"delta1", LexerTokenTypeString},
				{float64(0.314), LexerTokenTypeNumber},
				{"delta2", LexerTokenTypeString},
				{float64(314.), LexerTokenTypeNumber},
			},
		},
	}

	for _, testcase := range testcases {
		l, err := NewJSONLexer(strings.NewReader(testcase.input))
		if err != nil {
			t.Errorf("testcase '%s': could not create lexer: %v", testcase.input, err)
			continue
		}

		l.SetBufSize(4)

		tokensFound := 0

		for {
			token, err := l.Token()
			if err != nil {
				if err == io.EOF {
					break
				}

				t.Errorf("testcase '%s': %v", testcase.input, err)
				break
			}

			expectedOutput := testcase.output[tokensFound]
			switch expectedOutput.tokenType {
			case LexerTokenTypeString:
				if token != expectedOutput.token.(string) {
					t.Errorf("testcase '%s': expected token '%v', got '%s'",
						testcase.input, testcase.output[tokensFound].token, token)
					break
				}
			case LexerTokenTypeNumber:
				if token != expectedOutput.token.(float64) {
					t.Errorf("testcase '%s': expected token '%v', got '%f'",
						testcase.input, testcase.output[tokensFound].token, token)
					break
				}
			case LexerTokenTypeBool:
				if token != expectedOutput.token.(bool) {
					t.Errorf("testcase '%s': expected token '%v', got '%t'",
						testcase.input, testcase.output[tokensFound].token, token)
					break

				}
			case LexerTokenTypeNull:
				if token != nil {
					t.Errorf("testcase '%s': expected token '%v', got '%t'",
						testcase.input, testcase.output[tokensFound].token, token)
					break

				}
			}

			tokensFound++
		}

		if tokensFound != len(testcase.output) {
			t.Errorf("testcase '%s': expected %d tokens, got %d",
				testcase.input, len(testcase.output), tokensFound)
			continue
		}
	}
}

func TestJSONLexerFails(t *testing.T) {
	testcases := []jsonLexerTestCase{
		{`{"hello":"\u123r"}`, nil, false},
		{`{"hello":"\a"}`, nil, false},
		{`{"hello`, nil, false},
		{`{"hello": Nuii}`, nil, false},
		{`{"isValid": tru}`, nil, false},
		{`{"isValid": folse}`, nil, false},
		{`{"delta": 3.1.4}`, nil, false},
		{`{"temperature": 5-2}`, nil, false},
		{`{"distance": 1.57+10}`, nil, false},
		{`{"size": 1.2e*10}`, nil, false},
		{`{"distance": 1.57+e10}`, nil, false},
		{`{"size": 1.210-e}`, nil, false},
	}

	for _, testcase := range testcases {
		l, err := NewJSONLexer(strings.NewReader(testcase.input))
		if err != nil {
			t.Errorf("testcase '%s': could not create lexer: %v", testcase.input, err)
			continue
		}

		l.SetBufSize(64)
		errFound := false

		for {
			_, err := l.Token()
			if err != nil {
				if err == io.EOF {
					break
				}

				errFound = true
				break
			}

		}
		if !errFound {
			t.Errorf("testcase '%s': must have failed", testcase.input)
		}
	}
}

const (
	jsonSample = ` {
	  "type" : "row",
	  "position" : 471,
	  "clustering" : [ "1b5bf100-8f99-11ea-8e8d-fa163e4302ba" ],
	  "liveness_info" : { "tstamp" : "2020-05-06T12:57:14.193447Z" },
	  "cells" : [
		{ "name" : "event_id", "value" : 253 },
		{ "name" : "ip", "value" : "5.61.233.11" },
		{ "name" : "is_valid", "value" : true },
		{ "name" : "session_id", "value" : null },
		{ "name" : "delta", "value" : 3.14 },
		{ "name" : "temperature", "value" : -52 },
		{ "name" : "distance", "value" : 1.57e10 },
		{ "name" : "args", "deletion_info" : { "marked_deleted" : "2020-05-06T12:57:14.193446Z", "local_delete_time" : "2020-05-06T12:57:14Z" } },
		{ "name" : "args", "path" : [ "f" ], "value" : "fdevmail.openstacklocal" },
		{ "name" : "args", "path" : [ "h" ], "value" : "internal-api.devmail.ru" },
		{ "name" : "args", "path" : [ "ip" ], "value" : "127.0.0.1" },
		{ "name" : "args", "path" : [ "rid" ], "value" : "8c28ca1055" },
		{ "name" : "args", "path" : [ "ua" ], "value" : "\"Go-http-client/1.1\"" },
		{ "desc": "\u041f\u0440\u043e\u0432\u0435\u0440\u043a\u0430 \\UD83D\\UDCA9 \u043f\u043e\u0447\u0442\u044b"}
	  ]
	}`
)

func generateBenchmarkInput(b *bytes.Buffer, numObjects int) {
	for i := 0; i < numObjects; i++ {
		if i > 0 {
			b.WriteRune(',')
		}

		b.WriteString(jsonSample)
	}
}

func BenchmarkEncodingJSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		input := bytes.NewBuffer(nil)
		input.WriteRune('[')
		generateBenchmarkInput(input, 100)
		input.WriteRune(']')
		dec := json.NewDecoder(input)
		b.StartTimer()

		for {
			_, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Errorf("could not get next token: %v", err)
			}
		}
	}
}

func BenchmarkJSONLexer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		input := bytes.NewBuffer(nil)
		generateBenchmarkInput(input, 100)
		l, err := NewJSONLexer(input)
		if err != nil {
			b.Errorf("could not create JSONLexer: %v", err)
		}

		b.StartTimer()

		for {
			_, err := l.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Errorf("could not get next token: %v", err)
			}
		}
	}
}

func BenchmarkJSONLexerFast(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		input := bytes.Buffer{}
		generateBenchmarkInput(&input, 100)

		l, err := NewJSONLexer(&input)
		if err != nil {
			b.Errorf("could not create JSONLexer: %v", err)
		}

		l.SetBufSize(1024)

		b.StartTimer()

		for {
			_, err := l.TokenFast()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Errorf("could not get next token: %v", err)
			}
		}
	}
}
