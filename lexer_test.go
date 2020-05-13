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

// TODO tests for Null
// TODO tests for Bool
func TestJSONLexer(t *testing.T) {
	testcases := []jsonLexerTestCase{
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
			input: `{"hello":{"0": 10}}`,
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
		{
			input: `{"ua": "SomeInternationalUA\U123A"}`,
			output: []jsonLexerOutputToken{
				{
					"ua",
					LexerTokenTypeString,
				},
				{
					"SomeInternationalUA\\U123A",
					LexerTokenTypeString,
				},
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
		{
			input: `{"hello":"\u123r"}`,
		},
		{
			input: `{"hello":"\a"}`,
		},
		{
			input: `{"hello`,
		},
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
		{ "name" : "args", "deletion_info" : { "marked_deleted" : "2020-05-06T12:57:14.193446Z", "local_delete_time" : "2020-05-06T12:57:14Z" } },
		{ "name" : "args", "path" : [ "f" ], "value" : "fdevmail.openstacklocal" },
		{ "name" : "args", "path" : [ "h" ], "value" : "internal-api.devmail.ru" },
		{ "name" : "args", "path" : [ "ip" ], "value" : "127.0.0.1" },
		{ "name" : "args", "path" : [ "rid" ], "value" : "8c28ca1055" },
		{ "name" : "args", "path" : [ "ua" ], "value" : "\"Go-http-client/1.1\"" }
	  ]
	}`
)

func generateBenchmarkInput(w io.Writer, numObjects int) {
	for i := 0; i < numObjects; i++ {
		if i > 0 {
			w.Write([]byte(","))
		}

		w.Write([]byte(jsonSample))
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
		input := bytes.NewBuffer(nil)
		generateBenchmarkInput(input, 100)
		l, err := NewJSONLexer(input)
		if err != nil {
			b.Errorf("could not create JSONLexer: %v", err)
		}

		b.StartTimer()

		for {
			_, err := l.TokenGeneric()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Errorf("could not get next token: %v", err)
			}
		}
	}
}
