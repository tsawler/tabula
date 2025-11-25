package contentstream

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/tsawler/tabula/core"
)

// Operation represents a single content stream operation
// consisting of an operator and its operands
type Operation struct {
	Operator string         // The operator (e.g., "Tj", "Tm", "q")
	Operands []core.Object  // The operands
}

// Parser parses PDF content streams
type Parser struct {
	data   []byte
	pos    int
	ops    []Operation
}

// NewParser creates a new content stream parser
func NewParser(data []byte) *Parser {
	return &Parser{
		data: data,
		pos:  0,
		ops:  make([]Operation, 0),
	}
}

// Parse parses the content stream and returns a list of operations
func (p *Parser) Parse() ([]Operation, error) {
	for p.pos < len(p.data) {
		// Skip whitespace
		p.skipWhitespace()

		if p.pos >= len(p.data) {
			break
		}

		// Try to parse an object or operator
		if err := p.parseNext(); err != nil {
			return nil, err
		}
	}

	return p.ops, nil
}

// operandStack temporarily holds operands until we hit an operator
var operandStack []core.Object

// parseNext parses the next token (operand or operator)
func (p *Parser) parseNext() error {
	start := p.pos

	// Skip whitespace
	p.skipWhitespace()
	if p.pos >= len(p.data) {
		return nil
	}

	c := p.data[p.pos]

	// Check for potential operator (starts with letter)
	if isLetter(c) {
		return p.parseOperator()
	}

	// Otherwise, parse as operand
	operand, err := p.parseOperand()
	if err != nil {
		return fmt.Errorf("at position %d: %w", start, err)
	}

	operandStack = append(operandStack, operand)
	return nil
}

// parseOperator parses an operator and creates an operation
func (p *Parser) parseOperator() error {
	start := p.pos

	// Read operator name (letters and possibly quotes for special operators)
	var op bytes.Buffer
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if isLetter(c) || c == '\'' || c == '"' || c == '*' {
			op.WriteByte(c)
			p.pos++
		} else {
			break
		}
	}

	operator := op.String()
	if operator == "" {
		return fmt.Errorf("empty operator at position %d", start)
	}

	// Create operation with current operand stack
	operation := Operation{
		Operator: operator,
		Operands: make([]core.Object, len(operandStack)),
	}
	copy(operation.Operands, operandStack)

	p.ops = append(p.ops, operation)

	// Clear operand stack
	operandStack = nil

	return nil
}

// parseOperand parses a single operand (number, string, name, array, dict, etc.)
func (p *Parser) parseOperand() (core.Object, error) {
	p.skipWhitespace()

	if p.pos >= len(p.data) {
		return nil, fmt.Errorf("unexpected end of stream")
	}

	c := p.data[p.pos]

	// Number (int or real)
	if c == '-' || c == '+' || c == '.' || (c >= '0' && c <= '9') {
		return p.parseNumber()
	}

	// String (literal)
	if c == '(' {
		return p.parseString()
	}

	// Hex string
	if c == '<' && p.pos+1 < len(p.data) && p.data[p.pos+1] != '<' {
		return p.parseHexString()
	}

	// Name
	if c == '/' {
		return p.parseName()
	}

	// Array
	if c == '[' {
		return p.parseArray()
	}

	// Dictionary (rare in content streams, but possible)
	if c == '<' && p.pos+1 < len(p.data) && p.data[p.pos+1] == '<' {
		return p.parseDict()
	}

	// Boolean or null
	if c == 't' || c == 'f' || c == 'n' {
		// Check if it's actually an operator
		// Peek ahead to see if followed by whitespace
		end := p.pos
		for end < len(p.data) && !isWhitespace(p.data[end]) {
			end++
		}
		token := string(p.data[p.pos:end])

		switch token {
		case "true":
			p.pos = end
			return core.Bool(true), nil
		case "false":
			p.pos = end
			return core.Bool(false), nil
		case "null":
			p.pos = end
			return core.Null{}, nil
		}
	}

	return nil, fmt.Errorf("unexpected character at position %d: %c", p.pos, c)
}

// parseNumber parses an integer or real number
func (p *Parser) parseNumber() (core.Object, error) {
	start := p.pos
	hasDecimal := false

	// Handle sign
	if p.data[p.pos] == '+' || p.data[p.pos] == '-' {
		p.pos++
	}

	// Read digits and decimal point
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if c >= '0' && c <= '9' {
			p.pos++
		} else if c == '.' && !hasDecimal {
			hasDecimal = true
			p.pos++
		} else {
			break
		}
	}

	numStr := string(p.data[start:p.pos])

	if hasDecimal {
		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid real number %q: %w", numStr, err)
		}
		return core.Real(val), nil
	}

	val, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid integer %q: %w", numStr, err)
	}
	return core.Int(val), nil
}

// parseString parses a literal string (...)
func (p *Parser) parseString() (core.Object, error) {
	if p.data[p.pos] != '(' {
		return nil, fmt.Errorf("string must start with '('")
	}
	p.pos++ // skip '('

	var result bytes.Buffer
	depth := 1 // Track parenthesis nesting

	for p.pos < len(p.data) && depth > 0 {
		c := p.data[p.pos]

		if c == '\\' && p.pos+1 < len(p.data) {
			// Escape sequence
			p.pos++
			next := p.data[p.pos]
			switch next {
			case 'n':
				result.WriteByte('\n')
			case 'r':
				result.WriteByte('\r')
			case 't':
				result.WriteByte('\t')
			case 'b':
				result.WriteByte('\b')
			case 'f':
				result.WriteByte('\f')
			case '(':
				result.WriteByte('(')
			case ')':
				result.WriteByte(')')
			case '\\':
				result.WriteByte('\\')
			default:
				// Unknown escape - keep as-is
				result.WriteByte(next)
			}
			p.pos++
		} else if c == '(' {
			depth++
			result.WriteByte(c)
			p.pos++
		} else if c == ')' {
			depth--
			if depth > 0 {
				result.WriteByte(c)
			}
			p.pos++
		} else {
			result.WriteByte(c)
			p.pos++
		}
	}

	if depth != 0 {
		return nil, fmt.Errorf("unclosed string")
	}

	return core.String(result.String()), nil
}

// parseHexString parses a hex string <...>
func (p *Parser) parseHexString() (core.Object, error) {
	if p.data[p.pos] != '<' {
		return nil, fmt.Errorf("hex string must start with '<'")
	}
	p.pos++ // skip '<'

	var result bytes.Buffer

	for p.pos < len(p.data) {
		c := p.data[p.pos]

		if c == '>' {
			p.pos++
			break
		}

		if isWhitespace(c) {
			p.pos++
			continue
		}

		// Read hex digit
		if !isHexDigit(c) {
			return nil, fmt.Errorf("invalid hex digit: %c", c)
		}

		p.pos++
		// Read second hex digit (if available)
		if p.pos >= len(p.data) || p.data[p.pos] == '>' {
			// Odd number of digits - assume trailing 0
			result.WriteByte(hexValue(c) << 4)
			break
		}

		c2 := p.data[p.pos]
		if isWhitespace(c2) {
			// Skip whitespace between hex digits
			p.skipWhitespace()
			if p.pos >= len(p.data) || p.data[p.pos] == '>' {
				result.WriteByte(hexValue(c) << 4)
				break
			}
			c2 = p.data[p.pos]
		}

		if !isHexDigit(c2) {
			return nil, fmt.Errorf("invalid hex digit: %c", c2)
		}

		result.WriteByte((hexValue(c) << 4) | hexValue(c2))
		p.pos++
	}

	return core.String(result.String()), nil
}

// parseName parses a name /Name
func (p *Parser) parseName() (core.Object, error) {
	if p.data[p.pos] != '/' {
		return nil, fmt.Errorf("name must start with '/'")
	}
	p.pos++ // skip '/'

	var result bytes.Buffer

	for p.pos < len(p.data) {
		c := p.data[p.pos]

		// Name ends at whitespace or delimiter
		if isWhitespace(c) || isDelimiter(c) {
			break
		}

		// Handle # escape
		if c == '#' && p.pos+2 < len(p.data) {
			p.pos++
			hex1 := p.data[p.pos]
			hex2 := p.data[p.pos+1]
			if isHexDigit(hex1) && isHexDigit(hex2) {
				result.WriteByte((hexValue(hex1) << 4) | hexValue(hex2))
				p.pos += 2
				continue
			}
			// Invalid escape - keep #
			result.WriteByte('#')
			continue
		}

		result.WriteByte(c)
		p.pos++
	}

	return core.Name(result.String()), nil
}

// parseArray parses an array [...]
func (p *Parser) parseArray() (core.Object, error) {
	if p.data[p.pos] != '[' {
		return nil, fmt.Errorf("array must start with '['")
	}
	p.pos++ // skip '['

	var arr core.Array

	for p.pos < len(p.data) {
		p.skipWhitespace()

		if p.pos >= len(p.data) {
			return nil, fmt.Errorf("unclosed array")
		}

		if p.data[p.pos] == ']' {
			p.pos++
			break
		}

		obj, err := p.parseOperand()
		if err != nil {
			return nil, err
		}

		arr = append(arr, obj)
	}

	return arr, nil
}

// parseDict parses a dictionary <<...>>
func (p *Parser) parseDict() (core.Object, error) {
	if p.pos+1 >= len(p.data) || p.data[p.pos] != '<' || p.data[p.pos+1] != '<' {
		return nil, fmt.Errorf("dictionary must start with '<<'")
	}
	p.pos += 2 // skip '<<'

	dict := make(core.Dict)

	for p.pos < len(p.data) {
		p.skipWhitespace()

		if p.pos+1 < len(p.data) && p.data[p.pos] == '>' && p.data[p.pos+1] == '>' {
			p.pos += 2
			break
		}

		// Parse key (must be a name)
		if p.data[p.pos] != '/' {
			return nil, fmt.Errorf("dictionary key must be a name")
		}

		key, err := p.parseName()
		if err != nil {
			return nil, err
		}

		name, ok := key.(core.Name)
		if !ok {
			return nil, fmt.Errorf("expected name for dictionary key")
		}

		// Parse value
		value, err := p.parseOperand()
		if err != nil {
			return nil, err
		}

		dict[string(name)] = value
	}

	return dict, nil
}

// skipWhitespace skips whitespace characters
func (p *Parser) skipWhitespace() {
	for p.pos < len(p.data) && isWhitespace(p.data[p.pos]) {
		p.pos++
	}
}

// Helper functions

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == 0
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDelimiter(c byte) bool {
	return c == '(' || c == ')' || c == '<' || c == '>' ||
		c == '[' || c == ']' || c == '{' || c == '}' ||
		c == '/' || c == '%'
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func hexValue(c byte) byte {
	if c >= '0' && c <= '9' {
		return c - '0'
	}
	if c >= 'a' && c <= 'f' {
		return c - 'a' + 10
	}
	if c >= 'A' && c <= 'F' {
		return c - 'A' + 10
	}
	return 0
}
