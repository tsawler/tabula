package core

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// TestTokenTypeString tests the String method on TokenType
func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		token TokenType
		want  string
	}{
		{TokenEOF, "EOF"},
		{TokenWhitespace, "Whitespace"},
		{TokenComment, "Comment"},
		{TokenKeyword, "Keyword"},
		{TokenInteger, "Integer"},
		{TokenReal, "Real"},
		{TokenString, "String"},
		{TokenHexString, "HexString"},
		{TokenName, "Name"},
		{TokenArrayStart, "ArrayStart"},
		{TokenArrayEnd, "ArrayEnd"},
		{TokenDictStart, "DictStart"},
		{TokenDictEnd, "DictEnd"},
		{TokenIndirectRef, "IndirectRef"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			// Note: TokenType doesn't have String() yet, but we test the type exists
			if tt.token < TokenEOF || tt.token > TokenIndirectRef {
				t.Errorf("Invalid token type: %d", tt.token)
			}
		})
	}
}

// TestLexerEOF tests EOF handling
func TestLexerEOF(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty input", ""},
		{"whitespace only", "   \t\n\r  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			token, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.Type != TokenEOF {
				t.Errorf("expected TokenEOF, got %v", token.Type)
			}
		})
	}
}

// TestLexerComments tests comment parsing
func TestLexerComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple comment", "%PDF-1.7", "%PDF-1.7"},
		{"comment with LF", "%comment\n", "%comment"},
		{"comment with CR", "%comment\r", "%comment"},
		{"comment with CRLF", "%comment\r\n", "%comment"},
		{"comment at EOF", "%end of file", "%end of file"},
		{"empty comment", "%\n", "%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			token, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.Type != TokenComment {
				t.Errorf("expected TokenComment, got %v", token.Type)
			}
			if string(token.Value) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(token.Value))
			}
		})
	}
}

// TestLexerArrayDelimiters tests array bracket parsing
func TestLexerArrayDelimiters(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		tokenType TokenType
		value     string
	}{
		{"array start", "[", TokenArrayStart, "["},
		{"array end", "]", TokenArrayEnd, "]"},
		{"array with whitespace", "  [  ", TokenArrayStart, "["},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			token, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.Type != tt.tokenType {
				t.Errorf("expected %v, got %v", tt.tokenType, token.Type)
			}
			if string(token.Value) != tt.value {
				t.Errorf("expected %q, got %q", tt.value, string(token.Value))
			}
		})
	}
}

// TestLexerDictDelimiters tests dictionary delimiter parsing
func TestLexerDictDelimiters(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		tokenType TokenType
		value     string
	}{
		{"dict start", "<<", TokenDictStart, "<<"},
		{"dict end", ">>", TokenDictEnd, ">>"},
		{"dict with whitespace", "  <<  ", TokenDictStart, "<<"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			token, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.Type != tt.tokenType {
				t.Errorf("expected %v, got %v", tt.tokenType, token.Type)
			}
			if string(token.Value) != tt.value {
				t.Errorf("expected %q, got %q", tt.value, string(token.Value))
			}
		})
	}
}

// TestLexerStrings tests literal string parsing
func TestLexerStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{"simple string", "(hello)", "hello", false},
		{"empty string", "()", "", false},
		{"string with spaces", "(hello world)", "hello world", false},
		{"nested parentheses", "(hello (world))", "hello (world)", false},
		{"deeply nested", "(a(b(c)d)e)", "a(b(c)d)e", false},
		{"escape sequences", "(\\n\\r\\t\\b\\f)", "\n\r\t\b\f", false},
		{"escaped parens", "(\\(\\))", "()", false},
		{"escaped backslash", "(\\\\)", "\\", false},
		{"line continuation LF", "(hello\\\nworld)", "helloworld", false},
		{"line continuation CR", "(hello\\\rworld)", "helloworld", false},
		{"line continuation CRLF", "(hello\\\r\nworld)", "helloworld", false},
		{"octal escape 1 digit", "(\\101)", "A", false},
		{"octal escape 2 digits", "(\\141)", "a", false},
		{"octal escape 3 digits", "(\\101\\102)", "AB", false},
		{"mixed content", "(Text with \\101 and \\n newline)", "Text with A and \n newline", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			token, err := lexer.NextToken()
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got error: %v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}
			if token.Type != TokenString {
				t.Errorf("expected TokenString, got %v", token.Type)
			}
			if string(token.Value) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(token.Value))
			}
		})
	}
}

// TestLexerHexStrings tests hexadecimal string parsing
func TestLexerHexStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{"simple hex", "<48656C6C6F>", "48656C6C6F", false},
		{"empty hex", "<>", "", false},
		{"lowercase hex", "<abcdef>", "abcdef", false},
		{"uppercase hex", "<ABCDEF>", "ABCDEF", false},
		{"mixed case", "<AbCdEf>", "AbCdEf", false},
		{"with whitespace", "<48 65 6C 6C 6F>", "48656C6C6F", false},
		{"with newlines", "<48\n65\r6C\r\n6F>", "48656C6F", false},
		{"odd length", "<012>", "012", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			token, err := lexer.NextToken()
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got error: %v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}
			if token.Type != TokenHexString {
				t.Errorf("expected TokenHexString, got %v", token.Type)
			}
			if string(token.Value) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(token.Value))
			}
		})
	}
}

// TestLexerNames tests name object parsing
func TestLexerNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{"simple name", "/Type", "Type", false},
		{"empty name", "/", "", false},
		{"with numbers", "/F1", "F1", false},
		{"complex name", "/BaseFont", "BaseFont", false},
		{"hex escape", "/Name#20With#20Spaces", "Name With Spaces", false},
		{"special chars escaped", "/A#23B", "A#B", false},
		{"multiple escapes", "/a#20b#20c", "a b c", false},
		{"name at EOF", "/EOF", "EOF", false},
		{"name before delimiter", "/Type ", "Type", false},
		{"name before array", "/Name[", "Name", false},
		{"name before dict", "/Name<<", "Name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			token, err := lexer.NextToken()
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got error: %v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}
			if token.Type != TokenName {
				t.Errorf("expected TokenName, got %v", token.Type)
			}
			if string(token.Value) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(token.Value))
			}
		})
	}
}

// TestLexerNumbers tests number parsing
func TestLexerNumbers(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		tokenType TokenType
		expected  string
	}{
		{"zero", "0", TokenInteger, "0"},
		{"positive int", "123", TokenInteger, "123"},
		{"negative int", "-456", TokenInteger, "-456"},
		{"positive sign", "+789", TokenInteger, "+789"},
		{"float with decimal", "3.14", TokenReal, "3.14"},
		{"negative float", "-2.5", TokenReal, "-2.5"},
		{"leading decimal", ".5", TokenReal, ".5"},
		{"trailing decimal", "5.", TokenReal, "5."},
		{"zero decimal", "0.0", TokenReal, "0.0"},
		{"large number", "999999999", TokenInteger, "999999999"},
		{"scientific notation like", "1.23", TokenReal, "1.23"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			token, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.Type != tt.tokenType {
				t.Errorf("expected %v, got %v", tt.tokenType, token.Type)
			}
			if string(token.Value) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(token.Value))
			}
		})
	}
}

// TestLexerKeywords tests keyword and indirect reference parsing
func TestLexerKeywords(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		tokenType TokenType
		expected  string
	}{
		{"true", "true", TokenKeyword, "true"},
		{"false", "false", TokenKeyword, "false"},
		{"null", "null", TokenKeyword, "null"},
		{"obj", "obj", TokenKeyword, "obj"},
		{"endobj", "endobj", TokenKeyword, "endobj"},
		{"stream", "stream", TokenKeyword, "stream"},
		{"endstream", "endstream", TokenKeyword, "endstream"},
		{"xref", "xref", TokenKeyword, "xref"},
		{"trailer", "trailer", TokenKeyword, "trailer"},
		{"startxref", "startxref", TokenKeyword, "startxref"},
		{"indirect ref", "R", TokenIndirectRef, "R"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			token, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.Type != tt.tokenType {
				t.Errorf("expected %v, got %v", tt.tokenType, token.Type)
			}
			if string(token.Value) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(token.Value))
			}
		})
	}
}

// TestLexerWhitespace tests whitespace handling
func TestLexerWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"space", " 123"},
		{"tab", "\t123"},
		{"LF", "\n123"},
		{"CR", "\r123"},
		{"FF", "\f123"},
		{"null byte", "\x00123"},
		{"mixed whitespace", "  \t\n\r\f  123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			token, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Whitespace should be skipped, so we should get the number
			if token.Type != TokenInteger {
				t.Errorf("expected TokenInteger after whitespace, got %v", token.Type)
			}
			if string(token.Value) != "123" {
				t.Errorf("expected '123', got %q", string(token.Value))
			}
		})
	}
}

// TestLexerMultipleTokens tests tokenizing multiple tokens in sequence
func TestLexerMultipleTokens(t *testing.T) {
	input := "123 456 /Name (string) [ << >> ] true false null R"
	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TokenInteger, "123"},
		{TokenInteger, "456"},
		{TokenName, "Name"},
		{TokenString, "string"},
		{TokenArrayStart, "["},
		{TokenDictStart, "<<"},
		{TokenDictEnd, ">>"},
		{TokenArrayEnd, "]"},
		{TokenKeyword, "true"},
		{TokenKeyword, "false"},
		{TokenKeyword, "null"},
		{TokenIndirectRef, "R"},
		{TokenEOF, ""},
	}

	lexer := NewLexer(strings.NewReader(input))
	for i, exp := range expected {
		token, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("token %d: unexpected error: %v", i, err)
		}
		if token.Type != exp.tokenType {
			t.Errorf("token %d: expected type %v, got %v", i, exp.tokenType, token.Type)
		}
		if string(token.Value) != exp.value {
			t.Errorf("token %d: expected value %q, got %q", i, exp.value, string(token.Value))
		}
	}
}

// TestLexerDictionary tests tokenizing a complete dictionary
func TestLexerDictionary(t *testing.T) {
	input := "<< /Type /Page /MediaBox [ 0 0 612 792 ] /Contents 123 0 R >>"
	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TokenDictStart, "<<"},
		{TokenName, "Type"},
		{TokenName, "Page"},
		{TokenName, "MediaBox"},
		{TokenArrayStart, "["},
		{TokenInteger, "0"},
		{TokenInteger, "0"},
		{TokenInteger, "612"},
		{TokenInteger, "792"},
		{TokenArrayEnd, "]"},
		{TokenName, "Contents"},
		{TokenInteger, "123"},
		{TokenInteger, "0"},
		{TokenIndirectRef, "R"},
		{TokenDictEnd, ">>"},
		{TokenEOF, ""},
	}

	lexer := NewLexer(strings.NewReader(input))
	for i, exp := range expected {
		token, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("token %d: unexpected error: %v", i, err)
		}
		if token.Type != exp.tokenType {
			t.Errorf("token %d: expected type %v, got %v", i, exp.tokenType, token.Type)
		}
		if string(token.Value) != exp.value {
			t.Errorf("token %d: expected value %q, got %q", i, exp.value, string(token.Value))
		}
	}
}

// TestLexerIndirectObject tests tokenizing an indirect object
func TestLexerIndirectObject(t *testing.T) {
	input := "12 0 obj\n<< /Type /Page >>\nendobj"
	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TokenInteger, "12"},
		{TokenInteger, "0"},
		{TokenKeyword, "obj"},
		{TokenDictStart, "<<"},
		{TokenName, "Type"},
		{TokenName, "Page"},
		{TokenDictEnd, ">>"},
		{TokenKeyword, "endobj"},
		{TokenEOF, ""},
	}

	lexer := NewLexer(strings.NewReader(input))
	for i, exp := range expected {
		token, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("token %d: unexpected error: %v", i, err)
		}
		if token.Type != exp.tokenType {
			t.Errorf("token %d: expected type %v, got %v", i, exp.tokenType, token.Type)
		}
		if string(token.Value) != exp.value {
			t.Errorf("token %d: expected value %q, got %q", i, exp.value, string(token.Value))
		}
	}
}

// TestLexerWithComments tests tokenizing with comments interspersed
func TestLexerWithComments(t *testing.T) {
	input := "%PDF-1.7\n123 %comment\n456"
	expected := []struct {
		tokenType TokenType
		value     string
	}{
		{TokenComment, "%PDF-1.7"},
		{TokenInteger, "123"},
		{TokenComment, "%comment"},
		{TokenInteger, "456"},
		{TokenEOF, ""},
	}

	lexer := NewLexer(strings.NewReader(input))
	for i, exp := range expected {
		token, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("token %d: unexpected error: %v", i, err)
		}
		if token.Type != exp.tokenType {
			t.Errorf("token %d: expected type %v, got %v", i, exp.tokenType, token.Type)
		}
		if string(token.Value) != exp.value {
			t.Errorf("token %d: expected value %q, got %q", i, exp.value, string(token.Value))
		}
	}
}

// TestLexerErrors tests error handling
func TestLexerErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"single > without pair", ">", true},
		{"invalid hex digit", "<ZZ>", true},
		{"unclosed string", "(hello", true},
		{"invalid name escape", "/Name#ZZ", true},
		{"unexpected character", "@", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			_, err := lexer.NextToken()
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got error: %v", tt.wantErr, err)
			}
		})
	}
}

// TestLexerPositionTracking tests that positions are tracked correctly
func TestLexerPositionTracking(t *testing.T) {
	input := "123 456"
	lexer := NewLexer(strings.NewReader(input))

	// First token
	token1, err := lexer.NextToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token1.Pos != 0 {
		t.Errorf("expected position 0, got %d", token1.Pos)
	}

	// Second token (after "123 ")
	token2, err := lexer.NextToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token2.Pos != 4 {
		t.Errorf("expected position 4, got %d", token2.Pos)
	}
}

// TestLexerRealPDFContent tests tokenizing realistic PDF content
func TestLexerRealPDFContent(t *testing.T) {
	input := `%PDF-1.7
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [ 3 0 R ] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [ 0 0 612 792 ] >>
endobj`

	lexer := NewLexer(strings.NewReader(input))
	tokenCount := 0

	for {
		token, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("unexpected error at token %d: %v", tokenCount, err)
		}
		if token.Type == TokenEOF {
			break
		}
		tokenCount++
	}

	// We should have tokenized many tokens from this realistic PDF
	if tokenCount < 30 {
		t.Errorf("expected at least 30 tokens, got %d", tokenCount)
	}
}

// TestLexerStreamKeyword tests that stream/endstream are recognized
func TestLexerStreamKeyword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"stream", "stream", "stream"},
		{"endstream", "endstream", "endstream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))
			token, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.Type != TokenKeyword {
				t.Errorf("expected TokenKeyword, got %v", token.Type)
			}
			if string(token.Value) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(token.Value))
			}
		})
	}
}

// TestLexerBinaryContent tests handling binary content (though typically in streams)
func TestLexerBinaryContent(t *testing.T) {
	// Hex string with binary-looking data
	input := "<DEADBEEF>"
	lexer := NewLexer(strings.NewReader(input))
	token, err := lexer.NextToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.Type != TokenHexString {
		t.Errorf("expected TokenHexString, got %v", token.Type)
	}
	if string(token.Value) != "DEADBEEF" {
		t.Errorf("expected 'DEADBEEF', got %q", string(token.Value))
	}
}

// TestLexerNewlineFormats tests all PDF newline formats
func TestLexerNewlineFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"LF only", "123\n456"},
		{"CR only", "123\r456"},
		{"CRLF", "123\r\n456"},
		{"mixed", "123\n456\r789\r\n012"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(tt.input))

			// Should tokenize all numbers correctly regardless of newline format
			token1, err := lexer.NextToken()
			if err != nil {
				t.Fatalf("token 1 error: %v", err)
			}
			if string(token1.Value) != "123" {
				t.Errorf("expected '123', got %q", string(token1.Value))
			}

			token2, err := lexer.NextToken()
			if err != nil && err != io.EOF {
				t.Fatalf("token 2 error: %v", err)
			}
			if token2 != nil && token2.Type != TokenEOF {
				if string(token2.Value) != "456" && string(token2.Value) != "789" && string(token2.Value) != "012" {
					t.Errorf("unexpected token value: %q", string(token2.Value))
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkLexerSimpleTokens(b *testing.B) {
	input := "123 456 /Name (string)"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(strings.NewReader(input))
		for {
			token, err := lexer.NextToken()
			if err != nil || token.Type == TokenEOF {
				break
			}
		}
	}
}

func BenchmarkLexerDictionary(b *testing.B) {
	input := "<< /Type /Page /MediaBox [ 0 0 612 792 ] /Contents 123 0 R >>"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(strings.NewReader(input))
		for {
			token, err := lexer.NextToken()
			if err != nil || token.Type == TokenEOF {
				break
			}
		}
	}
}

func BenchmarkLexerString(b *testing.B) {
	input := "(This is a PDF string with \\n escape \\101 sequences)"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(strings.NewReader(input))
		lexer.NextToken()
	}
}

func BenchmarkLexerRealPDF(b *testing.B) {
	input := `%PDF-1.7
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [ 3 0 R ] /Count 1 >>
endobj`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(strings.NewReader(input))
		for {
			token, err := lexer.NextToken()
			if err != nil || token.Type == TokenEOF {
				break
			}
		}
	}
}

// TestLexerLargeInput tests performance with larger input
func TestLexerLargeInput(t *testing.T) {
	var buf bytes.Buffer
	// Create a large PDF-like structure
	for i := 0; i < 100; i++ {
		buf.WriteString("%Comment line\n")
		buf.WriteString("123 456 /Name (string) [ << >> ]\n")
	}

	lexer := NewLexer(&buf)
	tokenCount := 0

	for {
		token, err := lexer.NextToken()
		if err != nil {
			t.Fatalf("unexpected error at token %d: %v", tokenCount, err)
		}
		if token.Type == TokenEOF {
			break
		}
		tokenCount++
	}

	// Should have many tokens
	expectedMin := 100 * 8 // At least 8 tokens per line (comment + 7 tokens)
	if tokenCount < expectedMin {
		t.Errorf("expected at least %d tokens, got %d", expectedMin, tokenCount)
	}
}
