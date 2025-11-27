package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// TokenType represents the type of token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenWhitespace
	TokenComment
	TokenKeyword     // true, false, null, obj, endobj, stream, endstream, etc.
	TokenInteger     // 123
	TokenReal        // 3.14
	TokenString      // (hello)
	TokenHexString   // <48656C6C6F>
	TokenName        // /Type
	TokenArrayStart  // [
	TokenArrayEnd    // ]
	TokenDictStart   // <<
	TokenDictEnd     // >>
	TokenIndirectRef // R (after two numbers)
)

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value []byte
	Pos   int64 // Position in stream
}

// Lexer performs lexical analysis of PDF content
type Lexer struct {
	reader *bufio.Reader
	pos    int64
	line   int
	col    int
}

// NewLexer creates a new lexer
func NewLexer(r io.Reader) *Lexer {
	return &Lexer{
		reader: bufio.NewReader(r),
		pos:    0,
		line:   1,
		col:    0,
	}
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() (*Token, error) {
	// Skip whitespace but don't return it as a token
	l.skipWhitespace()

	// Check for EOF
	b, err := l.peek()
	if err == io.EOF {
		return &Token{Type: TokenEOF, Pos: l.pos}, nil
	}
	if err != nil {
		return nil, err
	}

	// Handle comments
	if b == '%' {
		return l.readComment()
	}

	// Handle delimiters
	switch b {
	case '[':
		l.readByte()
		return &Token{Type: TokenArrayStart, Value: []byte{'['}, Pos: l.pos - 1}, nil
	case ']':
		l.readByte()
		return &Token{Type: TokenArrayEnd, Value: []byte{']'}, Pos: l.pos - 1}, nil
	case '(':
		return l.readString()
	case '<':
		// Could be << (dict start) or <hex string>
		next, err := l.peekN(2)
		if err == nil && len(next) == 2 && next[1] == '<' {
			l.readByte()
			l.readByte()
			return &Token{Type: TokenDictStart, Value: []byte{'<', '<'}, Pos: l.pos - 2}, nil
		}
		return l.readHexString()
	case '>':
		// Must be >> (dict end)
		next, err := l.peekN(2)
		if err == nil && len(next) == 2 && next[1] == '>' {
			l.readByte()
			l.readByte()
			return &Token{Type: TokenDictEnd, Value: []byte{'>', '>'}, Pos: l.pos - 2}, nil
		}
		return nil, fmt.Errorf("unexpected '>' at position %d", l.pos)
	case '/':
		return l.readName()
	}

	// Handle numbers and keywords
	if isDigit(b) || b == '-' || b == '+' || b == '.' {
		return l.readNumber()
	}

	// Handle keywords (true, false, null, R, obj, endobj, stream, endstream, etc.)
	if isAlpha(b) {
		return l.readKeyword()
	}

	return nil, fmt.Errorf("unexpected character '%c' at position %d", b, l.pos)
}

// readByte reads a single byte and advances position
func (l *Lexer) readByte() (byte, error) {
	b, err := l.reader.ReadByte()
	if err != nil {
		return 0, err
	}
	l.pos++
	l.col++
	if b == '\n' {
		l.line++
		l.col = 0
	}
	return b, nil
}

// peek looks at the next byte without consuming it
func (l *Lexer) peek() (byte, error) {
	bytes, err := l.reader.Peek(1)
	if err != nil {
		return 0, err
	}
	return bytes[0], nil
}

// peekN looks at the next n bytes without consuming them
func (l *Lexer) peekN(n int) ([]byte, error) {
	return l.reader.Peek(n)
}

// unreadByte unreads the last byte
func (l *Lexer) unreadByte() error {
	err := l.reader.UnreadByte()
	if err != nil {
		return err
	}
	l.pos--
	l.col--
	return nil
}

// skipWhitespace skips all whitespace characters
// PDF whitespace: space (0x20), tab (0x09), LF (0x0A), CR (0x0D), FF (0x0C), null (0x00)
func (l *Lexer) skipWhitespace() error {
	for {
		b, err := l.peek()
		if err != nil {
			return err
		}
		if !isWhitespace(b) {
			return nil
		}
		l.readByte()
	}
}

// readComment reads a comment (% to end of line)
func (l *Lexer) readComment() (*Token, error) {
	startPos := l.pos
	var buf bytes.Buffer

	// Read the %
	b, err := l.readByte()
	if err != nil {
		return nil, err
	}
	buf.WriteByte(b)

	// Read until end of line
	for {
		b, err := l.peek()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Comments end at CR or LF
		if b == '\r' || b == '\n' {
			// Consume the newline
			l.readByte()
			// Handle CR LF sequence
			if b == '\r' {
				next, err := l.peek()
				if err == nil && next == '\n' {
					l.readByte()
				}
			}
			break
		}

		b, err = l.readByte()
		if err != nil {
			return nil, err
		}
		buf.WriteByte(b)
	}

	return &Token{Type: TokenComment, Value: buf.Bytes(), Pos: startPos}, nil
}

// readString reads a literal string (hello)
func (l *Lexer) readString() (*Token, error) {
	startPos := l.pos
	var buf bytes.Buffer

	// Read opening (
	b, err := l.readByte()
	if err != nil {
		return nil, err
	}
	if b != '(' {
		return nil, fmt.Errorf("expected '(' at position %d", l.pos-1)
	}

	depth := 1
	for depth > 0 {
		b, err := l.readByte()
		if err != nil {
			return nil, err
		}

		switch b {
		case '(':
			depth++
			buf.WriteByte(b)
		case ')':
			depth--
			if depth > 0 {
				buf.WriteByte(b)
			}
		case '\\':
			// Handle escape sequences
			next, err := l.readByte()
			if err != nil {
				return nil, err
			}
			switch next {
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			case 'b':
				buf.WriteByte('\b')
			case 'f':
				buf.WriteByte('\f')
			case '(', ')', '\\':
				buf.WriteByte(next)
			case '\r', '\n':
				// Line continuation - ignore the backslash and newline
				if next == '\r' {
					peek, err := l.peek()
					if err == nil && peek == '\n' {
						l.readByte()
					}
				}
			case '0', '1', '2', '3', '4', '5', '6', '7':
				// Octal escape \ddd
				octal := []byte{next}
				for i := 0; i < 2; i++ {
					peek, err := l.peek()
					if err != nil || !isOctalDigit(peek) {
						break
					}
					b, _ := l.readByte()
					octal = append(octal, b)
				}
				// Convert octal to byte
				var val byte
				for _, digit := range octal {
					val = val*8 + (digit - '0')
				}
				buf.WriteByte(val)
			default:
				// Unknown escape - keep the character
				buf.WriteByte(next)
			}
		default:
			buf.WriteByte(b)
		}
	}

	return &Token{Type: TokenString, Value: buf.Bytes(), Pos: startPos}, nil
}

// readHexString reads a hexadecimal string <48656C6C6F>
func (l *Lexer) readHexString() (*Token, error) {
	startPos := l.pos
	var buf bytes.Buffer

	// Read opening <
	b, err := l.readByte()
	if err != nil {
		return nil, err
	}
	if b != '<' {
		return nil, fmt.Errorf("expected '<' at position %d", l.pos-1)
	}

	for {
		b, err := l.peek()
		if err != nil {
			return nil, err
		}

		if b == '>' {
			l.readByte()
			break
		}

		b, err = l.readByte()
		if err != nil {
			return nil, err
		}

		// Skip whitespace in hex strings
		if isWhitespace(b) {
			continue
		}

		if !isHexDigit(b) {
			return nil, fmt.Errorf("invalid hex digit '%c' at position %d", b, l.pos-1)
		}

		buf.WriteByte(b)
	}

	return &Token{Type: TokenHexString, Value: buf.Bytes(), Pos: startPos}, nil
}

// readName reads a name object /Type
func (l *Lexer) readName() (*Token, error) {
	startPos := l.pos
	var buf bytes.Buffer

	// Read the /
	b, err := l.readByte()
	if err != nil {
		return nil, err
	}
	if b != '/' {
		return nil, fmt.Errorf("expected '/' at position %d", l.pos-1)
	}

	for {
		b, err := l.peek()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Names end at whitespace or delimiters
		if isWhitespace(b) || isDelimiter(b) {
			break
		}

		b, err = l.readByte()
		if err != nil {
			return nil, err
		}

		// Handle # escape sequences in names
		if b == '#' {
			hex1, err := l.readByte()
			if err != nil {
				return nil, err
			}
			hex2, err := l.readByte()
			if err != nil {
				return nil, err
			}
			if !isHexDigit(hex1) || !isHexDigit(hex2) {
				return nil, fmt.Errorf("invalid hex escape in name at position %d", l.pos-2)
			}
			// Convert hex to byte
			val := hexValue(hex1)*16 + hexValue(hex2)
			buf.WriteByte(val)
		} else {
			buf.WriteByte(b)
		}
	}

	return &Token{Type: TokenName, Value: buf.Bytes(), Pos: startPos}, nil
}

// readNumber reads an integer or real number
func (l *Lexer) readNumber() (*Token, error) {
	startPos := l.pos
	var buf bytes.Buffer
	hasDecimal := false

	for {
		b, err := l.peek()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if b == '.' {
			if hasDecimal {
				break // Second decimal point - not part of this number
			}
			hasDecimal = true
			b, _ = l.readByte()
			buf.WriteByte(b)
		} else if isDigit(b) || (buf.Len() == 0 && (b == '-' || b == '+')) {
			b, _ = l.readByte()
			buf.WriteByte(b)
		} else {
			break
		}
	}

	tokenType := TokenInteger
	if hasDecimal {
		tokenType = TokenReal
	}

	return &Token{Type: tokenType, Value: buf.Bytes(), Pos: startPos}, nil
}

// readKeyword reads a keyword (true, false, null, R, obj, endobj, etc.)
func (l *Lexer) readKeyword() (*Token, error) {
	startPos := l.pos
	var buf bytes.Buffer

	for {
		b, err := l.peek()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if !isAlpha(b) && !isDigit(b) {
			break
		}

		b, _ = l.readByte()
		buf.WriteByte(b)
	}

	value := buf.Bytes()

	// Check if it's R (indirect reference)
	if len(value) == 1 && value[0] == 'R' {
		return &Token{Type: TokenIndirectRef, Value: value, Pos: startPos}, nil
	}

	return &Token{Type: TokenKeyword, Value: value, Pos: startPos}, nil
}

// Helper functions

func isWhitespace(b byte) bool {
	// PDF whitespace: space, tab, LF, CR, FF, null
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == 0
}

func isDelimiter(b byte) bool {
	return b == '(' || b == ')' || b == '<' || b == '>' || b == '[' || b == ']' ||
		b == '{' || b == '}' || b == '/' || b == '%'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isOctalDigit(b byte) bool {
	return b >= '0' && b <= '7'
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func hexValue(b byte) byte {
	if b >= '0' && b <= '9' {
		return b - '0'
	}
	if b >= 'a' && b <= 'f' {
		return b - 'a' + 10
	}
	if b >= 'A' && b <= 'F' {
		return b - 'A' + 10
	}
	return 0
}

// ReadBytes reads exactly n bytes from the underlying reader
// This is used for reading binary stream data
func (l *Lexer) ReadBytes(n int) ([]byte, error) {
	data := make([]byte, n)
	totalRead := 0

	for totalRead < n {
		bytesRead, err := l.reader.Read(data[totalRead:])
		totalRead += bytesRead
		l.pos += int64(bytesRead)

		if err == io.EOF && totalRead < n {
			return data[:totalRead], fmt.Errorf("unexpected EOF: expected %d bytes, got %d", n, totalRead)
		}
		if err != nil && err != io.EOF {
			return data[:totalRead], err
		}
		if err == io.EOF {
			break
		}
	}

	return data, nil
}

// SkipBytes skips exactly n bytes from the underlying reader
func (l *Lexer) SkipBytes(n int) error {
	for i := 0; i < n; i++ {
		_, err := l.readByte()
		if err != nil {
			return err
		}
	}
	return nil
}

// Peek returns the next byte without consuming it (public wrapper for peek)
func (l *Lexer) Peek() (byte, error) {
	return l.peek()
}

// ReadByte reads and returns a single byte (public wrapper for readByte)
func (l *Lexer) ReadByte() (byte, error) {
	return l.readByte()
}
