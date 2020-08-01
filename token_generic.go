package gojsonlex

// TokenGeneric is a generic struct used to represent any possible JSON token
type TokenGeneric struct {
	t TokenType

	boolean bool
	str     string
	number  float64
	delim   byte
}

func newTokenGenericFromString(s string) TokenGeneric {
	return TokenGeneric{
		t:   LexerTokenTypeString,
		str: s,
	}
}

func newTokenGenericFromNumber(f float64) TokenGeneric {
	return TokenGeneric{
		t:      LexerTokenTypeNumber,
		number: f,
	}
}

func newTokenGenericFromBool(b bool) TokenGeneric {
	return TokenGeneric{
		t:       LexerTokenTypeBool,
		boolean: b,
	}
}

func newTokenGenericFromNull() TokenGeneric {
	return TokenGeneric{
		t: LexerTokenTypeNull,
	}
}

func newTokenGenericFromDelim(d byte) TokenGeneric {
	return TokenGeneric{
		t:     LexerTokenTypeDelim,
		delim: d,
	}
}

// Type returns type of the token
func (t *TokenGeneric) Type() TokenType {
	return t.t
}

// String returns string that points into internal lexer buffer and is guaranteed
// to be valid until the next Token call, otherwise you MUST make a deep copy
func (t *TokenGeneric) String() string {
	return t.str
}

// StringCopy return a deep copy of string
func (t *TokenGeneric) StringCopy() string {
	return StringDeepCopy(t.str)
}

func (t *TokenGeneric) Bool() bool {
	return t.boolean
}

func (t *TokenGeneric) Delim() byte {
	return t.delim
}

func (t *TokenGeneric) Number() float64 {
	return t.number
}

func (t *TokenGeneric) IsNull() bool {
	return t.t == LexerTokenTypeNull
}
