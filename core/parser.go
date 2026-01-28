package core

import (
	"fmt"
	"io"
	"strconv"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ReferenceResolver is an interface for resolving indirect references.
// This allows the parser to resolve indirect stream lengths when needed.
type ReferenceResolver interface {
	ResolveReference(ref IndirectRef) (Object, error)
}

// Parser parses PDF objects from an io.Reader using a Lexer for tokenization.
// It supports parsing all PDF object types including indirect objects and streams.
type Parser struct {
	lexer        *Lexer
	currentToken *Token // Current token being processed
	peekToken    *Token // Next token (lookahead)
	resolver     ReferenceResolver
}

// SetReferenceResolver sets the reference resolver for the parser.
// This is needed to resolve indirect stream lengths.
func (p *Parser) SetReferenceResolver(resolver ReferenceResolver) {
	p.resolver = resolver
}

// NewParser creates a new PDF parser for the given reader.
// It initializes the lexer and loads the first two tokens for lookahead.
func NewParser(r io.Reader) *Parser {
	p := &Parser{
		lexer: NewLexer(r),
	}
	// Load first two tokens
	p.nextToken()
	p.nextToken()
	return p
}

// nextToken advances the parser to the next token by shifting the lookahead.
func (p *Parser) nextToken() error {
	p.currentToken = p.peekToken

	// If we just moved "stream" into currentToken, don't try to read the next token
	// because it's binary data that can't be tokenized normally.
	// The parseStream function will handle reading the binary data directly.
	if p.currentToken != nil &&
		p.currentToken.Type == TokenKeyword &&
		string(p.currentToken.Value) == "stream" {
		p.peekToken = nil
		return nil
	}

	token, err := p.lexer.NextToken()
	if err != nil {
		return err
	}
	p.peekToken = token
	return nil
}

// skipComments skips over any consecutive comment tokens.
func (p *Parser) skipComments() error {
	for p.currentToken != nil && p.currentToken.Type == TokenComment {
		if err := p.nextToken(); err != nil {
			return err
		}
	}
	return nil
}

// ParseObject parses and returns the next PDF object from the input.
// It handles all PDF object types: null, boolean, integer, real, string,
// name, array, dictionary, and indirect references.
func (p *Parser) ParseObject() (Object, error) {
	// Skip any comments
	if err := p.skipComments(); err != nil {
		return nil, err
	}

	if p.currentToken == nil {
		return nil, fmt.Errorf("unexpected end of input")
	}

	switch p.currentToken.Type {
	case TokenEOF:
		return nil, io.EOF

	case TokenKeyword:
		keyword := string(p.currentToken.Value)
		switch keyword {
		case "null":
			p.nextToken()
			return Null{}, nil
		case "true":
			p.nextToken()
			return Bool(true), nil
		case "false":
			p.nextToken()
			return Bool(false), nil
		default:
			return nil, fmt.Errorf("unexpected keyword: %s", keyword)
		}

	case TokenInteger:
		// Could be integer, real, or start of indirect reference
		return p.parseNumber()

	case TokenReal:
		val, err := strconv.ParseFloat(string(p.currentToken.Value), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid real number: %w", err)
		}
		p.nextToken()
		return Real(val), nil

	case TokenString:
		val := string(p.currentToken.Value)
		p.nextToken()
		return String(val), nil

	case TokenHexString:
		// Convert hex string to bytes
		hexStr := string(p.currentToken.Value)
		if len(hexStr)%2 != 0 {
			hexStr += "0" // Pad if odd length
		}
		result := make([]byte, len(hexStr)/2)
		for i := 0; i < len(hexStr); i += 2 {
			b, err := strconv.ParseUint(hexStr[i:i+2], 16, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid hex string: %w", err)
			}
			result[i/2] = byte(b)
		}
		p.nextToken()
		return String(result), nil

	case TokenName:
		val := string(p.currentToken.Value)
		p.nextToken()
		return Name(val), nil

	case TokenArrayStart:
		return p.parseArray()

	case TokenDictStart:
		return p.parseDict()

	default:
		return nil, fmt.Errorf("unexpected token type: %v at position %d", p.currentToken.Type, p.currentToken.Pos)
	}
}

// parseNumber parses an integer, real number, or indirect reference.
// Indirect references are detected by lookahead: "num gen R" pattern.
func (p *Parser) parseNumber() (Object, error) {
	firstToken := string(p.currentToken.Value)

	// Try to parse as integer first
	firstInt, err := strconv.ParseInt(firstToken, 10, 64)
	if err != nil {
		// If it's not a valid integer, try as float
		f, err := strconv.ParseFloat(firstToken, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number: %s", firstToken)
		}
		p.nextToken()
		return Real(f), nil
	}

	// Use lookahead to check if this is an indirect reference (num gen R)
	// Don't consume tokens yet - just peek
	if p.peekToken != nil && p.peekToken.Type == TokenInteger {
		secondToken := string(p.peekToken.Value)
		secondInt, err := strconv.ParseInt(secondToken, 10, 64)
		if err == nil {
			// Peek ahead two tokens to see if there's an R
			// We need to temporarily consume to peek further
			p.nextToken() // Move to second integer
			if p.peekToken != nil && p.peekToken.Type == TokenIndirectRef {
				// It's an indirect reference - consume both tokens
				p.nextToken() // Move to R
				p.nextToken() // Move past R
				return IndirectRef{
					Number:     int(firstInt),
					Generation: int(secondInt),
				}, nil
			}
			// Not an indirect ref - we're now at the second integer
			// Return the first integer as Int
			return Int(firstInt), nil
		}
	}

	// Just a single integer
	p.nextToken()
	return Int(firstInt), nil
}

// parseArray parses a PDF array "[obj1 obj2 ...]".
func (p *Parser) parseArray() (Object, error) {
	if p.currentToken.Type != TokenArrayStart {
		return nil, fmt.Errorf("expected '[', got %v", p.currentToken.Type)
	}
	p.nextToken()

	var arr Array
	for {
		// Skip comments
		if err := p.skipComments(); err != nil {
			return nil, err
		}

		// Check for end of array
		if p.currentToken == nil {
			return nil, fmt.Errorf("unexpected end of input in array")
		}
		if p.currentToken.Type == TokenArrayEnd {
			p.nextToken()
			break
		}
		if p.currentToken.Type == TokenEOF {
			return nil, fmt.Errorf("unexpected EOF in array")
		}

		// Parse element
		obj, err := p.ParseObject()
		if err != nil {
			return nil, fmt.Errorf("error parsing array element: %w", err)
		}
		arr = append(arr, obj)
	}

	return arr, nil
}

// parseDict parses a PDF dictionary "<< /Key value ... >>".
func (p *Parser) parseDict() (Object, error) {
	if p.currentToken.Type != TokenDictStart {
		return nil, fmt.Errorf("expected '<<', got %v", p.currentToken.Type)
	}
	p.nextToken()

	dict := make(Dict)
	for {
		// Skip comments
		if err := p.skipComments(); err != nil {
			return nil, err
		}

		// Check for end of dict
		if p.currentToken == nil {
			return nil, fmt.Errorf("unexpected end of input in dictionary")
		}
		if p.currentToken.Type == TokenDictEnd {
			p.nextToken()
			break
		}
		if p.currentToken.Type == TokenEOF {
			return nil, fmt.Errorf("unexpected EOF in dictionary")
		}

		// Parse key (must be a name)
		if p.currentToken.Type != TokenName {
			return nil, fmt.Errorf("expected name for dictionary key, got %v", p.currentToken.Type)
		}
		key := string(p.currentToken.Value)
		p.nextToken()

		// Parse value
		value, err := p.ParseObject()
		if err != nil {
			return nil, fmt.Errorf("error parsing dictionary value for key '%s': %w", key, err)
		}

		dict[key] = value
	}

	return dict, nil
}

// ParseIndirectObject parses an indirect object definition.
// Format: "num gen obj <object> endobj" or "num gen obj <dict> stream ... endstream endobj"
func (p *Parser) ParseIndirectObject() (*IndirectObject, error) {
	// Skip comments
	if err := p.skipComments(); err != nil {
		return nil, err
	}

	// Parse object number
	if p.currentToken.Type != TokenInteger {
		return nil, fmt.Errorf("expected object number, got %v", p.currentToken.Type)
	}
	numStr := string(p.currentToken.Value)
	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid object number: %w", err)
	}
	p.nextToken()

	// Parse generation number
	if p.currentToken.Type != TokenInteger {
		return nil, fmt.Errorf("expected generation number, got %v", p.currentToken.Type)
	}
	genStr := string(p.currentToken.Value)
	gen, err := strconv.ParseInt(genStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid generation number: %w", err)
	}
	p.nextToken()

	// Parse 'obj' keyword
	if p.currentToken.Type != TokenKeyword || string(p.currentToken.Value) != "obj" {
		return nil, fmt.Errorf("expected 'obj' keyword, got %v", p.currentToken)
	}
	p.nextToken()

	// Parse the object value
	obj, err := p.ParseObject()
	if err != nil {
		return nil, fmt.Errorf("error parsing indirect object value: %w", err)
	}

	// Check for stream
	if p.currentToken.Type == TokenKeyword && string(p.currentToken.Value) == "stream" {
		// This is a stream object
		if dict, ok := obj.(Dict); ok {
			stream, err := p.parseStream(dict)
			if err != nil {
				return nil, fmt.Errorf("error parsing stream: %w", err)
			}
			obj = stream
		} else {
			return nil, fmt.Errorf("stream must follow a dictionary")
		}
	}

	// Parse 'endobj' keyword
	if p.currentToken.Type != TokenKeyword || string(p.currentToken.Value) != "endobj" {
		return nil, fmt.Errorf("expected 'endobj' keyword, got %v", p.currentToken)
	}
	p.nextToken()

	return &IndirectObject{
		Ref: IndirectRef{
			Number:     int(num),
			Generation: int(gen),
		},
		Object: obj,
	}, nil
}

// parseStream parses a stream object after the "stream" keyword.
// It reads the binary data according to the /Length entry in the dictionary.
func (p *Parser) parseStream(dict Dict) (*Stream, error) {
	// We're at the 'stream' keyword
	if p.currentToken.Type != TokenKeyword || string(p.currentToken.Value) != "stream" {
		return nil, fmt.Errorf("expected 'stream' keyword")
	}

	// Get the length from the dictionary
	lengthObj := dict.Get("Length")
	if lengthObj == nil {
		return nil, fmt.Errorf("stream dictionary missing 'Length' entry")
	}

	var length int
	switch v := lengthObj.(type) {
	case Int:
		length = int(v)
	case IndirectRef:
		// Length is an indirect reference - resolve it using the resolver
		if p.resolver == nil {
			return nil, fmt.Errorf("indirect reference for stream length requires a reference resolver")
		}
		resolved, err := p.resolver.ResolveReference(v)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve stream length reference: %w", err)
		}
		resolvedInt, ok := resolved.(Int)
		if !ok {
			return nil, fmt.Errorf("stream length reference resolved to %T, expected Int", resolved)
		}
		length = int(resolvedInt)
	default:
		return nil, fmt.Errorf("invalid type for stream length: %T", lengthObj)
	}

	if length < 0 {
		return nil, fmt.Errorf("invalid stream length: %d", length)
	}

	// Per PDF spec, the 'stream' keyword is followed by either:
	// - A single LF (0x0A), or
	// - A CR+LF sequence (0x0D 0x0A)
	// Then exactly 'length' bytes of stream data follow.
	//
	// Since we stopped loading peekToken when we saw 'stream', the lexer
	// is positioned right after the 'stream' keyword. We need to:
	// 1. Skip the mandatory EOL
	// 2. Read exactly 'length' bytes

	// Skip the EOL after 'stream'
	if err := p.lexer.SkipStreamEOL(); err != nil {
		return nil, fmt.Errorf("failed to skip EOL after stream keyword: %w", err)
	}

	// Read exactly 'length' bytes of stream data
	data, err := p.lexer.ReadBytes(length)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream data: %w", err)
	}

	// After the stream data, there should be an 'endstream' keyword
	// The lexer is now positioned right after the binary data
	// We need to get the next token which should be 'endstream'

	// Reset token state and get next token
	token, err := p.lexer.NextToken()
	if err != nil {
		return nil, fmt.Errorf("failed to read token after stream data: %w", err)
	}

	if token.Type != TokenKeyword || string(token.Value) != "endstream" {
		return nil, fmt.Errorf("expected 'endstream' keyword, got %v (%s)", token.Type, string(token.Value))
	}

	// Now reload the parser's current and peek tokens
	// This ensures ParseIndirectObject can continue normally
	p.currentToken = nil
	p.peekToken = nil
	p.nextToken()
	p.nextToken()

	return &Stream{
		Dict: dict,
		Data: data,
	}, nil
}
