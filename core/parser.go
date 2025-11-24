package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// Parser parses PDF syntax
type Parser struct {
	reader *bufio.Reader
	pos    int64
}

// NewParser creates a new PDF parser
func NewParser(r io.Reader) *Parser {
	return &Parser{
		reader: bufio.NewReader(r),
	}
}

// ParseObject parses a PDF object
func (p *Parser) ParseObject() (Object, error) {
	p.skipWhitespace()

	// Peek at next character to determine object type
	b, err := p.reader.ReadByte()
	if err != nil {
		return nil, err
	}
	p.reader.UnreadByte()

	switch {
	case b == 'n':
		return p.parseNull()
	case b == 't' || b == 'f':
		return p.parseBool()
	case b == '(':
		return p.parseString()
	case b == '/':
		return p.parseName()
	case b == '[':
		return p.parseArray()
	case b == '<':
		// Could be dict or hex string
		next, err := p.peekByte(1)
		if err != nil {
			return nil, err
		}
		if next == '<' {
			return p.parseDict()
		}
		return p.parseHexString()
	case isDigit(b) || b == '-' || b == '+' || b == '.':
		return p.parseNumber()
	default:
		return nil, fmt.Errorf("unexpected character: %c", b)
	}
}

// parseNull parses a null object
func (p *Parser) parseNull() (Object, error) {
	token, err := p.readToken()
	if err != nil {
		return nil, err
	}
	if token != "null" {
		return nil, fmt.Errorf("expected 'null', got '%s'", token)
	}
	return Null{}, nil
}

// parseBool parses a boolean object
func (p *Parser) parseBool() (Object, error) {
	token, err := p.readToken()
	if err != nil {
		return nil, err
	}
	switch token {
	case "true":
		return Bool(true), nil
	case "false":
		return Bool(false), nil
	default:
		return nil, fmt.Errorf("expected 'true' or 'false', got '%s'", token)
	}
}

// parseNumber parses an integer or real number
func (p *Parser) parseNumber() (Object, error) {
	token, err := p.readToken()
	if err != nil {
		return nil, err
	}

	// Check if it's an indirect reference (num gen R)
	if p.peekToken() != "" {
		gen, genErr := strconv.ParseInt(p.peekToken(), 10, 64)
		if genErr == nil {
			p.readToken() // consume generation
			if p.peekToken() == "R" {
				p.readToken() // consume R
				num, _ := strconv.ParseInt(token, 10, 64)
				return IndirectRef{Number: int(num), Generation: int(gen)}, nil
			}
		}
	}

	// Try parsing as integer
	if i, err := strconv.ParseInt(token, 10, 64); err == nil {
		return Int(i), nil
	}

	// Try parsing as real
	if f, err := strconv.ParseFloat(token, 64); err == nil {
		return Real(f), nil
	}

	return nil, fmt.Errorf("invalid number: %s", token)
}

// parseString parses a literal string
func (p *Parser) parseString() (Object, error) {
	// Read opening (
	if b, err := p.reader.ReadByte(); err != nil || b != '(' {
		return nil, fmt.Errorf("expected '('")
	}

	var buf bytes.Buffer
	depth := 1

	for depth > 0 {
		b, err := p.reader.ReadByte()
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
			// Escape sequence
			next, err := p.reader.ReadByte()
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
			case '\\', '(', ')':
				buf.WriteByte(next)
			default:
				buf.WriteByte(next)
			}
		default:
			buf.WriteByte(b)
		}
	}

	return String(buf.String()), nil
}

// parseHexString parses a hexadecimal string
func (p *Parser) parseHexString() (Object, error) {
	// Read opening <
	if b, err := p.reader.ReadByte(); err != nil || b != '<' {
		return nil, fmt.Errorf("expected '<'")
	}

	var buf bytes.Buffer
	for {
		b, err := p.reader.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == '>' {
			break
		}
		if isHexDigit(b) {
			buf.WriteByte(b)
		}
	}

	// Convert hex to bytes
	hexStr := buf.String()
	if len(hexStr)%2 != 0 {
		hexStr += "0"
	}

	result := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		b, _ := strconv.ParseUint(hexStr[i:i+2], 16, 8)
		result[i/2] = byte(b)
	}

	return String(result), nil
}

// parseName parses a name object
func (p *Parser) parseName() (Object, error) {
	// Read opening /
	if b, err := p.reader.ReadByte(); err != nil || b != '/' {
		return nil, fmt.Errorf("expected '/'")
	}

	token, err := p.readToken()
	if err != nil {
		return nil, err
	}

	return Name(token), nil
}

// parseArray parses an array object
func (p *Parser) parseArray() (Object, error) {
	// Read opening [
	if b, err := p.reader.ReadByte(); err != nil || b != '[' {
		return nil, fmt.Errorf("expected '['")
	}

	var arr Array
	for {
		p.skipWhitespace()

		// Check for closing ]
		b, err := p.reader.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == ']' {
			break
		}
		p.reader.UnreadByte()

		// Parse next object
		obj, err := p.ParseObject()
		if err != nil {
			return nil, err
		}
		arr = append(arr, obj)
	}

	return arr, nil
}

// parseDict parses a dictionary object
func (p *Parser) parseDict() (Object, error) {
	// Read opening <<
	if b1, err := p.reader.ReadByte(); err != nil || b1 != '<' {
		return nil, fmt.Errorf("expected '<<'")
	}
	if b2, err := p.reader.ReadByte(); err != nil || b2 != '<' {
		return nil, fmt.Errorf("expected '<<'")
	}

	dict := make(Dict)
	for {
		p.skipWhitespace()

		// Check for closing >>
		b, err := p.reader.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == '>' {
			if next, err := p.reader.ReadByte(); err == nil && next == '>' {
				break
			}
		}
		p.reader.UnreadByte()

		// Parse key (must be name)
		keyObj, err := p.parseName()
		if err != nil {
			return nil, err
		}
		key := string(keyObj.(Name))

		// Parse value
		val, err := p.ParseObject()
		if err != nil {
			return nil, err
		}

		dict[key] = val
	}

	return dict, nil
}

// Helper functions

func (p *Parser) skipWhitespace() {
	for {
		b, err := p.reader.ReadByte()
		if err != nil {
			return
		}
		if !isWhitespace(b) {
			p.reader.UnreadByte()
			return
		}
	}
}

func (p *Parser) readToken() (string, error) {
	var buf bytes.Buffer
	for {
		b, err := p.reader.ReadByte()
		if err != nil {
			if err == io.EOF && buf.Len() > 0 {
				return buf.String(), nil
			}
			return "", err
		}
		if isWhitespace(b) || isDelimiter(b) {
			p.reader.UnreadByte()
			break
		}
		buf.WriteByte(b)
	}
	return buf.String(), nil
}

func (p *Parser) peekToken() string {
	// pos := p.pos
	token, _ := p.readToken()
	// Reset position (simplified - in real implementation need to track position)
	return token
}

func (p *Parser) peekByte(offset int) (byte, error) {
	bytes, err := p.reader.Peek(offset + 1)
	if err != nil {
		return 0, err
	}
	return bytes[offset], nil
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == 0
}

func isDelimiter(b byte) bool {
	return b == '(' || b == ')' || b == '<' || b == '>' || b == '[' || b == ']' ||
		b == '{' || b == '}' || b == '/' || b == '%'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isHexDigit(b byte) bool {
	return isDigit(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}
