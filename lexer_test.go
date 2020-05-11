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
					lexerTokenTypeString,
				},
				{
					"world",
					lexerTokenTypeString,
				},
			},
		},
		{
			input: `{"hello":{"0": 10}}`,
			output: []jsonLexerOutputToken{
				{
					"hello",
					lexerTokenTypeString,
				},
				{
					"0",
					lexerTokenTypeString,
				},
				{
					float64(10),
					lexerTokenTypeNumber,
				},
			},
		},
		{
			input: `{"liveness_info" : { "tstamp" : "2020-05-06T12:57:14.193447Z" }}`,
			output: []jsonLexerOutputToken{
				{
					"liveness_info",
					lexerTokenTypeString,
				},
				{
					"tstamp",
					lexerTokenTypeString,
				},
				{
					"2020-05-06T12:57:14.193447Z",
					lexerTokenTypeString,
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

		l.SetBufSize(64)

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
			case lexerTokenTypeString:
				if token != expectedOutput.token.(string) {
					t.Errorf("testcase '%s': expected token '%v', got '%s'",
						testcase.input, testcase.output[tokensFound].token, token)
					break
				}
			case lexerTokenTypeNumber:
				if token != expectedOutput.token.(float64) {
					t.Errorf("testcase '%s': expected token '%v', got '%f'",
						testcase.input, testcase.output[tokensFound].token, token)
					break
				}
			case lexerTokenTypeBool:
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
		{ "name" : "args", "path" : [ "ua" ], "value" : "Go-http-client/1.1" }
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

func BenchmarkJSONLexer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		input := bytes.NewBuffer(nil)
		generateBenchmarkInput(input, 100)
		p, err := NewJSONLexer(input)
		if err != nil {
			b.Errorf("could not create JSONLexer: %v", err)
		}
		b.StartTimer()

		for {
			t, err := p.Token()
			if err != nil {
				b.Errorf("could not get next token: %v", err)
			}

			if t == nil {
				break
			}
		}
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
