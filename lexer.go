package gojsonlex

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"unicode"
)

const (
	defaultBufSize = 4096
)

type lexerState byte

const (
	stateLexerIdle lexerState = iota
	stateLexerSkipping
	stateLexerString
	stateLexerPendingEscapedSymbol
	stateLexerUnicodeRune
	stateLexerNumber
	stateLexerBool
	stateLexerNull
)

// JSONLexer is a JSON lexical analyzer with streaming API support, where stream is a sequence of
// JSON tokens. JSONLexer does its own IO buffering so prefer low-level readers if you want
// to miminize memory footprint.
//
// JSONLexer uses a ring buffer for parsing tokens, every token must fit in its size, otherwise
// buffer will be automatically grown. Initial size of buffer is 4096 bytes, however you can tweak
// it with SetBufSize() in case you know that most tokens are going to be long.
//
// JSONLexer uses unsafe pointers into the underlying buf to minimize allocations, see Token()
// for the provided guarantees.
type JSONLexer struct {
	r               io.Reader
	readingFinished bool // reports whether r has more data to read

	state lexerState

	buf     []byte
	currPos int // current positin in buffer

	unicodeRuneBytesCounter byte // a counter used to validate a unicode rune

	currTokenStart int // positin in the buf of current token start (if any)
	currTokenEnd   int // positin in the buf of current token start (if any)
	currTokenType  TokenType
	newTokenFound  bool // true if during the last feed() a new token was finished being parsed

	skipDelims bool

	debug bool
}

// NewJSONLexer creates a new JSONLexer with the given reader.
func NewJSONLexer(r io.Reader) (*JSONLexer, error) {
	l := &JSONLexer{
		r:   r,
		buf: make([]byte, defaultBufSize),
	}

	return l, nil
}

// SetBufSize creates a new buffer of the given size. MUST be called before parsing started.
func (l *JSONLexer) SetBufSize(bufSize int) {
	l.buf = make([]byte, bufSize)
}

// SetSkipDelims tells JSONLexer to skip delimiters and return only keys and values. This can
// be useful in case you want to simply match the input to some specific grammar and have no
// intention of doing full syntax analysis.
func (l *JSONLexer) SetSkipDelims(mustSkip bool) {
	l.skipDelims = true
}

// SetDebug enables debug logging
func (l *JSONLexer) SetDebug(debug bool) {
	l.debug = true
}

func (l *JSONLexer) processStateSkipping(c byte) error {
	switch {
	case c == '"':
		l.state = stateLexerString
		l.currTokenType = LexerTokenTypeString
		l.currTokenStart = l.currPos
	case CanAppearInNumber(rune(c)):
		l.state = stateLexerNumber
		l.currTokenType = LexerTokenTypeNumber
		l.currTokenStart = l.currPos
	case c == 't' || c == 'T':
		fallthrough
	case c == 'f' || c == 'F':
		l.state = stateLexerBool
		l.currTokenType = LexerTokenTypeBool
		l.currTokenStart = l.currPos
	case c == 'n' || c == 'N':
		l.state = stateLexerNull
		l.currTokenType = LexerTokenTypeNull
		l.currTokenStart = l.currPos
	default:
		// skipping
	}

	return nil
}

func (l *JSONLexer) processStateString(c byte) error {
	switch c {
	case '"':
		l.state = stateLexerSkipping
		l.currTokenEnd = l.currPos
		l.newTokenFound = true
	case '\\':
		l.state = stateLexerPendingEscapedSymbol
	default:
		// accumulating string
	}

	return nil
}

func (l *JSONLexer) processStatePendingEscapedSymbol(c byte) error {
	if !IsValidEscapedSymbol(rune(c)) {
		return fmt.Errorf("invalid escape sequence '\\%c'", c)
	}

	if c == 'u' || c == 'U' {
		l.state = stateLexerUnicodeRune
		l.unicodeRuneBytesCounter = 0
		return nil
	}

	l.state = stateLexerString

	return nil
}

func (l *JSONLexer) processStateUnicodeRune(c byte) error {
	if !IsHexDigit(rune(c)) {
		return fmt.Errorf("invalid hex digit '%c' inside escaped unicode rune", c)
	}

	l.unicodeRuneBytesCounter++
	if l.unicodeRuneBytesCounter == 4 {
		l.state = stateLexerString
	}

	return nil
}

func (l *JSONLexer) processStateNumber(c byte) error {
	switch {
	case unicode.IsDigit(rune(c)):
		fallthrough
	case c == '.':
		// accumulating number
	case IsDelim(rune(c)):
		fallthrough
	case unicode.IsSpace(rune(c)):
		l.state = stateLexerSkipping
		l.currTokenEnd = l.currPos
		l.newTokenFound = true
	}

	return nil
}

func (l *JSONLexer) processStateNull(c byte) error {
	currPositionInToken := l.currPos - l.currTokenStart

	if currPositionInToken == len("null") {
		l.state = stateLexerSkipping
		l.currTokenEnd = l.currPos
		l.newTokenFound = true
		return nil
	}

	expectedLiteral := rune("null"[currPositionInToken])

	if unicode.ToLower(rune(c)) != expectedLiteral {
		return fmt.Errorf("invalid literal '%c' while parsing 'Null' value", c)
	}

	return nil
}

func (l *JSONLexer) processStateBool(c byte) error {
	firstCharOfToken := unicode.ToLower(rune(l.buf[l.currTokenStart]))
	currPositionInToken := l.currPos - l.currTokenStart

	var expectedToken string

	switch firstCharOfToken {
	case 't':
		expectedToken = "true"
	case 'f':
		expectedToken = "false"
	}

	if currPositionInToken == len(expectedToken) {
		l.state = stateLexerSkipping
		l.currTokenEnd = l.currPos
		l.newTokenFound = true
		return nil
	}

	expectedLiteral := rune(expectedToken[currPositionInToken])

	if unicode.ToLower(rune(c)) != expectedLiteral {
		return fmt.Errorf("invalid literal '%c' while parsing bool value", c)
	}

	return nil
}

func (l *JSONLexer) feed(c byte) error {
	switch l.state {
	case stateLexerSkipping:
		return l.processStateSkipping(c)
	case stateLexerString:
		return l.processStateString(c)
	case stateLexerPendingEscapedSymbol:
		return l.processStatePendingEscapedSymbol(c)
	case stateLexerUnicodeRune:
		return l.processStateUnicodeRune(c)
	case stateLexerNumber:
		return l.processStateNumber(c)
	case stateLexerBool:
		return l.processStateBool(c)
	case stateLexerNull:
		return l.processStateNull(c)
	}

	return nil
}

func (l *JSONLexer) currTokenAsUnsafeString() (string, error) {
	// skipping "
	var subStr = l.buf[l.currTokenStart+1 : l.currTokenEnd]
	subStr, err := UnescapeBytesInplace(subStr)
	if err != nil {
		return "", err
	}

	return unsafeStringFromBytes(subStr), nil
}

func (l *JSONLexer) currTokenAsNumber() (float64, error) {
	str := unsafeStringFromBytes(l.buf[l.currTokenStart:l.currTokenEnd])

	n, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("could not convert '%s' to float64: %w", StringDeepCopy(str), err)
	}

	return n, nil
}

func (l *JSONLexer) currTokenAsBool() (bool, error) {
	if unicode.ToLower(rune(l.buf[l.currTokenStart])) == 't' {
		return true, nil
	}
	if unicode.ToLower(rune(l.buf[l.currTokenStart])) == 'f' {
		return false, nil
	}

	tokenAsStr := unsafeStringFromBytes(l.buf[l.currTokenStart:l.currTokenEnd])
	return false, fmt.Errorf("could not convert '%s' to bool", StringDeepCopy(tokenAsStr))
}

func (l *JSONLexer) currToken() (TokenGeneric, error) {
	switch l.currTokenType {
	case LexerTokenTypeDelim:
		return newTokenGenericFromDelim(l.buf[l.currTokenStart]), nil
	case LexerTokenTypeString:
		s, err := l.currTokenAsUnsafeString()
		return newTokenGenericFromString(s), err
	case LexerTokenTypeNumber:
		n, err := l.currTokenAsNumber()
		return newTokenGenericFromNumber(n), err
	case LexerTokenTypeBool:
		b, err := l.currTokenAsBool()
		return newTokenGenericFromBool(b), err
	case LexerTokenTypeNull:
		return newTokenGenericFromNull(), nil
	}

	panic("unexpected token type")
}

func (l *JSONLexer) fetchNewData() error {
	// if now some token is in the middle of parsing we gotta copy the part of it
	// that has already been parsed, otherwise we won't be able to construct it
	if l.state != stateLexerSkipping && l.state != stateLexerIdle {
		dstBuf := l.buf

		// checking if buf must be extended
		currTokenBytesParsed := l.currPos - l.currTokenStart
		if currTokenBytesParsed >= l.currTokenStart {
			newSize := 2 * len(l.buf)
			dstBuf = make([]byte, newSize)

			if l.debug {
				log.Printf("debug: gojsonlex: growing buffer %d -> %d", len(l.buf), newSize)
			}
		}

		// copying the part that has already been parsed
		copy(dstBuf, l.buf[l.currTokenStart:])
		l.currTokenStart = 0
		l.currPos = currTokenBytesParsed
		l.buf = dstBuf
	} else {
		l.currPos = 0
	}

	// reading new data into buf
	n, err := io.ReadFull(l.r, l.buf[l.currPos:])
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		l.readingFinished = true
		l.buf = l.buf[:l.currPos+n]
	} else if err != nil {
		return fmt.Errorf("could not fetch new data: %w", err)
	}

	return nil
}

func (l *JSONLexer) shutdown() error {
	if l.state != stateLexerSkipping {
		return fmt.Errorf("unexpected EOF")
	}

	return io.EOF
}

// Token returns the next JSON token, all delimiters are skipped. Token will return io.EOF when
// all input has been exhausted.  All strings returned by Token are guaranteed to be valid
// until the next Token call, otherwise you MUST make a deep copy.
func (l *JSONLexer) Token() (json.Token, error) {
	t, err := l.TokenFast()
	if err != nil {
		return nil, err
	}

	switch t.t {
	case LexerTokenTypeNull:
		return nil, nil
	case LexerTokenTypeDelim:
		return t.delim, nil
	case LexerTokenTypeNumber:
		return t.number, nil
	case LexerTokenTypeString:
		return t.str, nil
	case LexerTokenTypeBool:
		return t.boolean, nil
	}

	panic("unknown token type")
}

// TokenFast is a more efficient version of Token(). All strings returned by Token
// are guaranteed to be valid until the next Token call, otherwise you MUST make a deep copy.
func (l *JSONLexer) TokenFast() (TokenGeneric, error) {
	if l.state == stateLexerIdle {
		if err := l.fetchNewData(); err != nil {
			return TokenGeneric{}, err
		}

		l.state = stateLexerSkipping
	}

	for {
		if l.currPos >= len(l.buf) {
			if l.readingFinished {
				return TokenGeneric{}, l.shutdown()
			}

			if err := l.fetchNewData(); err != nil {
				return TokenGeneric{}, err
			}

			continue // last fetching could probably return 0 new bytes
		}

		if err := l.feed(l.buf[l.currPos]); err != nil {
			return TokenGeneric{}, err
		}

		l.currPos++

		if l.newTokenFound {
			l.newTokenFound = false
			break
		}
	}

	return l.currToken()
}
