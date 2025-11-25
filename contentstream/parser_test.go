package contentstream

import (
	"testing"

	"github.com/tsawler/tabula/core"
)

// TestParseSimpleOperator tests parsing a simple operator with no operands
func TestParseSimpleOperator(t *testing.T) {
	input := []byte("q")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}

	if ops[0].Operator != "q" {
		t.Errorf("expected operator 'q', got %q", ops[0].Operator)
	}

	if len(ops[0].Operands) != 0 {
		t.Errorf("expected 0 operands, got %d", len(ops[0].Operands))
	}
}

// TestParseOperatorWithInteger tests an operator with integer operand
func TestParseOperatorWithInteger(t *testing.T) {
	input := []byte("100 Tz")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}

	if ops[0].Operator != "Tz" {
		t.Errorf("expected operator 'Tz', got %q", ops[0].Operator)
	}

	if len(ops[0].Operands) != 1 {
		t.Fatalf("expected 1 operand, got %d", len(ops[0].Operands))
	}

	val, ok := ops[0].Operands[0].(core.Int)
	if !ok {
		t.Fatalf("expected Int operand, got %T", ops[0].Operands[0])
	}

	if val != 100 {
		t.Errorf("expected value 100, got %d", val)
	}
}

// TestParseOperatorWithReal tests an operator with real number operand
func TestParseOperatorWithReal(t *testing.T) {
	input := []byte("1.5 w")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}

	if ops[0].Operator != "w" {
		t.Errorf("expected operator 'w', got %q", ops[0].Operator)
	}

	val, ok := ops[0].Operands[0].(core.Real)
	if !ok {
		t.Fatalf("expected Real operand, got %T", ops[0].Operands[0])
	}

	if val != 1.5 {
		t.Errorf("expected value 1.5, got %f", val)
	}
}

// TestParseOperatorWithString tests an operator with string operand
func TestParseOperatorWithString(t *testing.T) {
	input := []byte("(Hello World) Tj")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}

	if ops[0].Operator != "Tj" {
		t.Errorf("expected operator 'Tj', got %q", ops[0].Operator)
	}

	val, ok := ops[0].Operands[0].(core.String)
	if !ok {
		t.Fatalf("expected String operand, got %T", ops[0].Operands[0])
	}

	if string(val) != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", val)
	}
}

// TestParseOperatorWithName tests an operator with name operand
func TestParseOperatorWithName(t *testing.T) {
	input := []byte("/F1 12 Tf")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}

	if ops[0].Operator != "Tf" {
		t.Errorf("expected operator 'Tf', got %q", ops[0].Operator)
	}

	if len(ops[0].Operands) != 2 {
		t.Fatalf("expected 2 operands, got %d", len(ops[0].Operands))
	}

	name, ok := ops[0].Operands[0].(core.Name)
	if !ok {
		t.Fatalf("expected Name operand, got %T", ops[0].Operands[0])
	}

	if string(name) != "F1" {
		t.Errorf("expected 'F1', got %q", name)
	}

	size, ok := ops[0].Operands[1].(core.Int)
	if !ok {
		t.Fatalf("expected Int operand, got %T", ops[0].Operands[1])
	}

	if size != 12 {
		t.Errorf("expected 12, got %d", size)
	}
}

// TestParseTextMatrix tests text matrix operator
func TestParseTextMatrix(t *testing.T) {
	input := []byte("1 0 0 1 100 200 Tm")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}

	if ops[0].Operator != "Tm" {
		t.Errorf("expected operator 'Tm', got %q", ops[0].Operator)
	}

	if len(ops[0].Operands) != 6 {
		t.Fatalf("expected 6 operands, got %d", len(ops[0].Operands))
	}
}

// TestParseTextBlock tests a complete text block
func TestParseTextBlock(t *testing.T) {
	input := []byte(`BT
/F1 12 Tf
100 200 Td
(Hello) Tj
ET`)
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 5 {
		t.Fatalf("expected 5 operations, got %d", len(ops))
	}

	// Check operators
	expectedOps := []string{"BT", "Tf", "Td", "Tj", "ET"}
	for i, expected := range expectedOps {
		if ops[i].Operator != expected {
			t.Errorf("operation %d: expected %q, got %q", i, expected, ops[i].Operator)
		}
	}
}

// TestParseArray tests parsing an array operand
func TestParseArray(t *testing.T) {
	input := []byte("[(Hello) -250 (World)] TJ")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}

	if ops[0].Operator != "TJ" {
		t.Errorf("expected operator 'TJ', got %q", ops[0].Operator)
	}

	arr, ok := ops[0].Operands[0].(core.Array)
	if !ok {
		t.Fatalf("expected Array operand, got %T", ops[0].Operands[0])
	}

	if len(arr) != 3 {
		t.Fatalf("expected 3 array elements, got %d", len(arr))
	}

	// Check array elements
	str1, ok := arr[0].(core.String)
	if !ok || string(str1) != "Hello" {
		t.Errorf("expected 'Hello', got %v", arr[0])
	}

	num, ok := arr[1].(core.Int)
	if !ok || num != -250 {
		t.Errorf("expected -250, got %v", arr[1])
	}

	str2, ok := arr[2].(core.String)
	if !ok || string(str2) != "World" {
		t.Errorf("expected 'World', got %v", arr[2])
	}
}

// TestParseGraphicsState tests graphics state operators
func TestParseGraphicsState(t *testing.T) {
	input := []byte(`q
1 0 0 1 50 50 cm
Q`)
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(ops))
	}

	if ops[0].Operator != "q" || len(ops[0].Operands) != 0 {
		t.Errorf("expected 'q' with 0 operands")
	}

	if ops[1].Operator != "cm" || len(ops[1].Operands) != 6 {
		t.Errorf("expected 'cm' with 6 operands")
	}

	if ops[2].Operator != "Q" || len(ops[2].Operands) != 0 {
		t.Errorf("expected 'Q' with 0 operands")
	}
}

// TestParsePathConstruction tests path construction operators
func TestParsePathConstruction(t *testing.T) {
	input := []byte(`10 20 m
100 20 l
100 100 l
10 100 l
s`)
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 5 {
		t.Fatalf("expected 5 operations, got %d", len(ops))
	}

	// Check move to
	if ops[0].Operator != "m" {
		t.Errorf("expected 'm', got %q", ops[0].Operator)
	}
	if len(ops[0].Operands) != 2 {
		t.Errorf("expected 2 operands for 'm', got %d", len(ops[0].Operands))
	}

	// Check line to
	if ops[1].Operator != "l" {
		t.Errorf("expected 'l', got %q", ops[1].Operator)
	}

	// Check stroke
	if ops[4].Operator != "s" {
		t.Errorf("expected 's', got %q", ops[4].Operator)
	}
}

// TestParseHexString tests hex string parsing
func TestParseHexString(t *testing.T) {
	input := []byte("<48656C6C6F> Tj")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	str, ok := ops[0].Operands[0].(core.String)
	if !ok {
		t.Fatalf("expected String, got %T", ops[0].Operands[0])
	}

	if string(str) != "Hello" {
		t.Errorf("expected 'Hello', got %q", str)
	}
}

// TestParseEscapedString tests string with escape sequences
func TestParseEscapedString(t *testing.T) {
	input := []byte(`(Hello\nWorld\t!) Tj`)
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	str, ok := ops[0].Operands[0].(core.String)
	if !ok {
		t.Fatalf("expected String, got %T", ops[0].Operands[0])
	}

	expected := "Hello\nWorld\t!"
	if string(str) != expected {
		t.Errorf("expected %q, got %q", expected, str)
	}
}

// TestParseNestedParentheses tests string with nested parentheses
func TestParseNestedParentheses(t *testing.T) {
	input := []byte(`(Text (with (nested) parens)) Tj`)
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	str, ok := ops[0].Operands[0].(core.String)
	if !ok {
		t.Fatalf("expected String, got %T", ops[0].Operands[0])
	}

	expected := "Text (with (nested) parens)"
	if string(str) != expected {
		t.Errorf("expected %q, got %q", expected, str)
	}
}

// TestParseNegativeNumbers tests negative numbers
func TestParseNegativeNumbers(t *testing.T) {
	input := []byte("-10 -20.5 Td")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops[0].Operands) != 2 {
		t.Fatalf("expected 2 operands, got %d", len(ops[0].Operands))
	}

	val1, ok := ops[0].Operands[0].(core.Int)
	if !ok || val1 != -10 {
		t.Errorf("expected -10, got %v", ops[0].Operands[0])
	}

	val2, ok := ops[0].Operands[1].(core.Real)
	if !ok || val2 != -20.5 {
		t.Errorf("expected -20.5, got %v", ops[0].Operands[1])
	}
}

// TestParseMultipleOperations tests multiple operations in sequence
func TestParseMultipleOperations(t *testing.T) {
	input := []byte(`q
1 w
1 0 0 RG
10 10 m
100 100 l
S
Q`)
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 7 {
		t.Fatalf("expected 7 operations, got %d", len(ops))
	}
}

// TestParseEmptyInput tests empty input
func TestParseEmptyInput(t *testing.T) {
	input := []byte("")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 0 {
		t.Errorf("expected 0 operations for empty input, got %d", len(ops))
	}
}

// TestParseWhitespaceOnly tests input with only whitespace
func TestParseWhitespaceOnly(t *testing.T) {
	input := []byte("   \n\t\r  ")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 0 {
		t.Errorf("expected 0 operations for whitespace-only input, got %d", len(ops))
	}
}

// TestParseNameWithSpecialChars tests names with # escapes
func TestParseNameWithSpecialChars(t *testing.T) {
	input := []byte("/Name#20With#20Spaces Tj")
	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	name, ok := ops[0].Operands[0].(core.Name)
	if !ok {
		t.Fatalf("expected Name, got %T", ops[0].Operands[0])
	}

	expected := "Name With Spaces"
	if string(name) != expected {
		t.Errorf("expected %q, got %q", expected, name)
	}
}

// TestParseRealWorld tests a more realistic content stream
func TestParseRealWorld(t *testing.T) {
	input := []byte(`BT
/F1 12 Tf
1 0 0 1 72 720 Tm
0 Tc
0 Tw
(The quick brown fox) Tj
0 -14 Td
(jumps over the lazy dog.) Tj
ET`)

	parser := NewParser(input)

	ops, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ops) != 9 {
		t.Fatalf("expected 9 operations, got %d", len(ops))
	}

	// Verify operation sequence
	expectedOps := []string{"BT", "Tf", "Tm", "Tc", "Tw", "Tj", "Td", "Tj", "ET"}
	for i, expected := range expectedOps {
		if ops[i].Operator != expected {
			t.Errorf("operation %d: expected %q, got %q", i, expected, ops[i].Operator)
		}
	}
}
