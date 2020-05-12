package gojsonlex

import (
	"encoding/json"
	"fmt"
	"io"
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
// JSONLexer uses a ring buffer of fixed size (4096 bytes by default) and currently will fail
// if some token length  exceeds the size of buffer, however you can tweak buffer size on
// JSONLexer creation.
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

func (l *JSONLexer) processStateSkipping(c byte) error {
	switch {
	case c == '"':
		l.state = stateLexerString
		l.currTokenType = lexerTokenTypeString
		l.currTokenStart = l.currPos
	case unicode.IsDigit(rune(c)):
		l.state = stateLexerNumber
		l.currTokenType = lexerTokenTypeNumber
		l.currTokenStart = l.currPos
	case c == 't' || c == 'T':
		fallthrough
	case c == 'f' || c == 'F':
		l.state = stateLexerBool
		l.currTokenType = lexerTokenTypeBool
		l.currTokenStart = l.currPos
	case c == 'n' || c == 'N':
		l.state = stateLexerNull
		l.currTokenType = lexerTokenTypeNull
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
		panic("parsing bool is not implemented")
	case stateLexerNull:
		panic("parsing null is not implemented")
	}

	return nil
}

func (l *JSONLexer) currTokenAsUnsafeString() (string, error) {
	// skipping "
	var subStr = l.buf[l.currTokenStart+1 : l.currTokenEnd]
	subStr = unescapeBytesInplace(subStr)

	return unsafeStringFromBytes(subStr), nil
}

func (l *JSONLexer) currTokenAsNumber() (float64, error) {
	str := unsafeStringFromBytes(l.buf[l.currTokenStart:l.currTokenEnd])

	n, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("could not convert '%s' to float64: %w", str, err)
	}

	return n, nil
}

func (l *JSONLexer) currTokenAsBool() (bool, error) {
	if unicode.ToLower(rune(l.buf[l.currTokenStart])) == 't' {
		return true, nil
	}
	if unicode.ToLower(rune(l.buf[l.currTokenStart])) == 'n' {
		return false, nil
	}

	tokenAsStr := unsafeStringFromBytes(l.buf[l.currTokenStart:l.currTokenEnd])
	return false, fmt.Errorf("could not convert '%s' to bool", tokenAsStr)
}

func (l *JSONLexer) currToken() (json.Token, error) {
	switch l.currTokenType {
	case lexerTokenTypeDelim:
		return l.buf[l.currTokenStart], nil
	case lexerTokenTypeString:
		return l.currTokenAsUnsafeString()
	case lexerTokenTypeNumber:
		return l.currTokenAsNumber()
	case lexerTokenTypeBool:
	}

	panic("unexpected token type")
}

func (l *JSONLexer) fetchNewData() error {
	// if now some token is in the middle of parsing we gotta copy the part of it
	// that has already been parsed, otherwise we won't be able to construct it
	if l.state != stateLexerSkipping && l.state != stateLexerIdle {
		// checking for overlapping
		currTokenRunesParsed := len(l.buf) - l.currTokenStart
		if currTokenRunesParsed >= l.currTokenStart {
			return fmt.Errorf("failed to fetchNewData due to buf overlapping")
		}

		// copying the part that has already been parsed
		copy(l.buf[0:], l.buf[l.currTokenStart:])
		l.currTokenStart = 0
		l.currPos = currTokenRunesParsed
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

func (l *JSONLexer) shutdown() (json.Token, error) {
	if l.state != stateLexerSkipping {
		return nil, fmt.Errorf("unexpected EOF")
	}

	return nil, io.EOF
}

// Token returns the next JSON token. All strings returned by Token are guaranteed to be valid
// until the next Token call, otherwise you MUST make a deep copy.
func (l *JSONLexer) Token() (json.Token, error) {
	if l.state == stateLexerIdle {
		if err := l.fetchNewData(); err != nil {
			return nil, err
		}

		l.state = stateLexerSkipping
	}

	for {
		if l.currPos >= len(l.buf) {
			if l.readingFinished {
				return l.shutdown()
			}

			if err := l.fetchNewData(); err != nil {
				return nil, err
			}

			continue // last fetching could probably return 0 new bytes
		}

		if err := l.feed(l.buf[l.currPos]); err != nil {
			return nil, err
		}

		l.currPos++

		if l.newTokenFound {
			l.newTokenFound = false
			break
		}
	}

	return l.currToken()
}
