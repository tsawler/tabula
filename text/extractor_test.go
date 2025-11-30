package text

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/contentstream"
	"github.com/tsawler/tabula/core"
)

// TestNewExtractor tests extractor creation
func TestNewExtractor(t *testing.T) {
	ex := NewExtractor()

	if ex == nil {
		t.Fatal("expected non-nil extractor")
	}

	if ex.gs == nil {
		t.Error("expected graphics state to be initialized")
	}

	if ex.fonts == nil {
		t.Error("expected fonts map to be initialized")
	}
}

// TestRegisterFont tests font registration
func TestRegisterFont(t *testing.T) {
	ex := NewExtractor()

	ex.RegisterFont("F1", "Helvetica", "Type1")

	if _, ok := ex.fonts["F1"]; !ok {
		t.Error("font F1 not registered")
	}
}

// TestSimpleTextExtraction tests basic text extraction
func TestSimpleTextExtraction(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Hello")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(fragments))
	}

	if fragments[0].Text != "Hello" {
		t.Errorf("expected text 'Hello', got %q", fragments[0].Text)
	}

	if fragments[0].FontSize != 12 {
		t.Errorf("expected font size 12, got %f", fragments[0].FontSize)
	}
}

// TestTextWithPositioning tests text with positioning
func TestTextWithPositioning(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Td", Operands: []core.Object{core.Int(100), core.Int(200)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Text")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(fragments))
	}

	// Should be positioned at (100, 200)
	if fragments[0].X != 100 || fragments[0].Y != 200 {
		t.Errorf("expected position (100, 200), got (%f, %f)",
			fragments[0].X, fragments[0].Y)
	}
}

// TestMultipleTextFragments tests multiple text operations
func TestMultipleTextFragments(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Hello")}},
		{Operator: "Td", Operands: []core.Object{core.Int(0), core.Int(-14)}},
		{Operator: "Tj", Operands: []core.Object{core.String("World")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("expected 2 fragments, got %d", len(fragments))
	}

	if fragments[0].Text != "Hello" {
		t.Errorf("expected first fragment 'Hello', got %q", fragments[0].Text)
	}

	if fragments[1].Text != "World" {
		t.Errorf("expected second fragment 'World', got %q", fragments[1].Text)
	}
}

// TestTextArray tests TJ operator with array
func TestTextArray(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "TJ", Operands: []core.Object{
			core.Array{
				core.String("Hello"),
				core.Int(-200),
				core.String("World"),
			},
		}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("expected 2 fragments, got %d", len(fragments))
	}

	if fragments[0].Text != "Hello" {
		t.Errorf("expected first fragment 'Hello', got %q", fragments[0].Text)
	}

	if fragments[1].Text != "World" {
		t.Errorf("expected second fragment 'World', got %q", fragments[1].Text)
	}
}

// TestGraphicsStateSaveRestore tests q/Q operators
func TestGraphicsStateSaveRestore(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "q", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(18)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Large")}},
		{Operator: "Q", Operands: []core.Object{}},
		{Operator: "Tj", Operands: []core.Object{core.String("Normal")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("expected 2 fragments, got %d", len(fragments))
	}

	if fragments[0].FontSize != 18 {
		t.Errorf("expected first fragment font size 18, got %f", fragments[0].FontSize)
	}

	if fragments[1].FontSize != 12 {
		t.Errorf("expected second fragment font size 12, got %f", fragments[1].FontSize)
	}
}

// TestGetText tests full text concatenation
func TestGetText(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Tj", Operands: []core.Object{core.String("The")}},
		{Operator: "Tj", Operands: []core.Object{core.String("quick")}},
		{Operator: "Tj", Operands: []core.Object{core.String("brown")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	_, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	text := ex.GetText()

	if !strings.Contains(text, "The") ||
		!strings.Contains(text, "quick") ||
		!strings.Contains(text, "brown") {
		t.Errorf("expected text to contain all words, got %q", text)
	}
}

// TestExtractFromBytes tests parsing and extraction in one step
func TestExtractFromBytes(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	// Simple content stream
	data := []byte(`BT
/F1 12 Tf
(Hello World) Tj
ET`)

	fragments, err := ex.ExtractFromBytes(data)
	if err != nil {
		t.Fatalf("ExtractFromBytes failed: %v", err)
	}

	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(fragments))
	}

	if fragments[0].Text != "Hello World" {
		t.Errorf("expected text 'Hello World', got %q", fragments[0].Text)
	}
}

// TestTextWithTransform tests text with CTM transformation
func TestTextWithTransform(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "q", Operands: []core.Object{}},
		{Operator: "cm", Operands: []core.Object{
			core.Int(1), core.Int(0),
			core.Int(0), core.Int(1),
			core.Int(50), core.Int(50),
		}},
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Td", Operands: []core.Object{core.Int(10), core.Int(10)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Transformed")}},
		{Operator: "ET", Operands: []core.Object{}},
		{Operator: "Q", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(fragments))
	}

	// Position should include CTM translation
	// (10, 10) in text space + (50, 50) from CTM = (60, 60)
	if fragments[0].X != 60 || fragments[0].Y != 60 {
		t.Errorf("expected position (60, 60), got (%f, %f)",
			fragments[0].X, fragments[0].Y)
	}
}

// TestAutoFontRegistration tests automatic font registration
func TestAutoFontRegistration(t *testing.T) {
	ex := NewExtractor()
	// Don't pre-register the font

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Auto")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(fragments))
	}

	// Font should have been auto-registered
	if _, ok := ex.fonts["/F1"]; !ok {
		t.Error("expected font to be auto-registered")
	}
}

// TestTextSpacing tests character and word spacing
func TestTextSpacing(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Tc", Operands: []core.Object{core.Real(0.5)}},
		{Operator: "Tw", Operands: []core.Object{core.Real(2.0)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Spaced")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(fragments))
	}

	if fragments[0].Text != "Spaced" {
		t.Errorf("expected text 'Spaced', got %q", fragments[0].Text)
	}
}

// TestEmptyContentStream tests handling of empty content stream
func TestEmptyContentStream(t *testing.T) {
	ex := NewExtractor()

	fragments, err := ex.ExtractFromBytes([]byte{})
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 0 {
		t.Errorf("expected 0 fragments for empty stream, got %d", len(fragments))
	}
}

// TestGetFragments tests the GetFragments method
func TestGetFragments(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Test")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	_, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	fragments := ex.GetFragments()
	if len(fragments) != 1 {
		t.Errorf("expected 1 fragment, got %d", len(fragments))
	}
}

// TestGetFonts tests the GetFonts method
func TestGetFonts(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")
	ex.RegisterFont("/F2", "Times", "Type1")

	fonts := ex.GetFonts()
	if len(fonts) != 2 {
		t.Errorf("expected 2 fonts, got %d", len(fonts))
	}

	if _, ok := fonts["/F1"]; !ok {
		t.Error("expected font /F1 to be registered")
	}
	if _, ok := fonts["/F2"]; !ok {
		t.Error("expected font /F2 to be registered")
	}
}

// TestToFloat tests the toFloat helper function
func TestToFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    core.Object
		expected float64
		ok       bool
	}{
		{"Int", core.Int(42), 42.0, true},
		{"Real", core.Real(3.14), 3.14, true},
		{"String", core.String("hello"), 0, false},
		{"Name", core.Name("test"), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat(tt.input)
			if ok != tt.ok {
				t.Errorf("toFloat() ok = %v, want %v", ok, tt.ok)
			}
			if ok && result != tt.expected {
				t.Errorf("toFloat() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestToInt tests the toInt helper function
func TestToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    core.Object
		expected int
		ok       bool
	}{
		{"Int", core.Int(42), 42, true},
		{"Real", core.Real(3.14), 0, false},
		{"String", core.String("hello"), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toInt(tt.input)
			if ok != tt.ok {
				t.Errorf("toInt() ok = %v, want %v", ok, tt.ok)
			}
			if ok && result != tt.expected {
				t.Errorf("toInt() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestOperandsToMatrix tests matrix construction from operands
func TestOperandsToMatrix(t *testing.T) {
	t.Run("valid matrix", func(t *testing.T) {
		operands := []core.Object{
			core.Real(1), core.Real(0),
			core.Real(0), core.Real(1),
			core.Real(100), core.Real(200),
		}
		m := operandsToMatrix(operands)
		if m[4] != 100 || m[5] != 200 {
			t.Errorf("expected translation (100, 200), got (%v, %v)", m[4], m[5])
		}
	})

	t.Run("invalid length returns identity", func(t *testing.T) {
		operands := []core.Object{core.Int(1), core.Int(2)}
		m := operandsToMatrix(operands)
		// Identity matrix should have 1s on diagonal
		if m[0] != 1 || m[3] != 1 {
			t.Error("expected identity matrix for invalid input")
		}
	})
}

// TestTextMatrix tests Tm operator
func TestTextMatrix(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Tm", Operands: []core.Object{
			core.Real(1), core.Real(0),
			core.Real(0), core.Real(1),
			core.Real(50), core.Real(100),
		}},
		{Operator: "Tj", Operands: []core.Object{core.String("Matrix")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(fragments))
	}

	// Position should be set by text matrix
	if fragments[0].X != 50 || fragments[0].Y != 100 {
		t.Errorf("expected position (50, 100), got (%f, %f)",
			fragments[0].X, fragments[0].Y)
	}
}

// TestTextLeading tests TL and T* operators
func TestTextLeading(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Td", Operands: []core.Object{core.Int(0), core.Int(700)}},
		{Operator: "TL", Operands: []core.Object{core.Int(14)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Line 1")}},
		{Operator: "T*", Operands: []core.Object{}},
		{Operator: "Tj", Operands: []core.Object{core.String("Line 2")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("expected 2 fragments, got %d", len(fragments))
	}

	// Second line should be below first by leading amount
	if fragments[1].Y >= fragments[0].Y {
		t.Error("expected second line to be below first")
	}
}

// TestTextRise tests Ts operator
func TestTextRise(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Td", Operands: []core.Object{core.Int(0), core.Int(100)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Normal")}},
		{Operator: "Ts", Operands: []core.Object{core.Int(5)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Raised")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("expected 2 fragments, got %d", len(fragments))
	}

	// Second fragment should be raised above first
	if fragments[1].Y <= fragments[0].Y {
		t.Error("expected raised text to be above normal text")
	}
}

// TestTextScaling tests Tz operator
func TestTextScaling(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Tz", Operands: []core.Object{core.Int(200)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Wide")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(fragments))
	}

	if fragments[0].Text != "Wide" {
		t.Errorf("expected text 'Wide', got %q", fragments[0].Text)
	}
}

// TestTextTDOperator tests TD operator (moves and sets leading)
func TestTextTDOperator(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "TD", Operands: []core.Object{core.Int(100), core.Int(200)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Moved")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(fragments))
	}

	if fragments[0].X != 100 || fragments[0].Y != 200 {
		t.Errorf("expected position (100, 200), got (%f, %f)",
			fragments[0].X, fragments[0].Y)
	}
}

// TestShowTextWithNewLine tests ' and " operators
func TestShowTextWithNewLine(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "TL", Operands: []core.Object{core.Int(14)}},
		{Operator: "Td", Operands: []core.Object{core.Int(0), core.Int(700)}},
		{Operator: "Tj", Operands: []core.Object{core.String("First")}},
		{Operator: "'", Operands: []core.Object{core.String("Second")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("expected 2 fragments, got %d", len(fragments))
	}
}

// TestDoubleQuoteOperator tests the " operator (set spacing, move, show text)
func TestDoubleQuoteOperator(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "TL", Operands: []core.Object{core.Int(14)}},
		{Operator: "Td", Operands: []core.Object{core.Int(0), core.Int(700)}},
		{Operator: "Tj", Operands: []core.Object{core.String("First")}},
		{Operator: "\"", Operands: []core.Object{core.Real(1), core.Real(2), core.String("Second")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("expected 2 fragments, got %d", len(fragments))
	}
}

// TestTJWithKerning tests TJ array with kerning adjustments
func TestTJWithKerning(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "TJ", Operands: []core.Object{
			core.Array{
				core.String("A"),
				core.Int(-80), // Negative adjustment (kerning)
				core.String("V"),
			},
		}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("expected 2 fragments, got %d", len(fragments))
	}
}

// TestTJWithRealNumber tests TJ array with real number adjustments
func TestTJWithRealNumber(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "TJ", Operands: []core.Object{
			core.Array{
				core.String("Hello"),
				core.Real(-100.5),
				core.String("World"),
			},
		}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("expected 2 fragments, got %d", len(fragments))
	}
}

// TestMultipleBTET tests multiple text blocks
func TestMultipleBTET(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Block1")}},
		{Operator: "ET", Operands: []core.Object{}},
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(14)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Block2")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("expected 2 fragments, got %d", len(fragments))
	}

	if fragments[0].FontSize != 12 || fragments[1].FontSize != 14 {
		t.Error("font sizes not preserved between text blocks")
	}
}

// TestIgnoredOperators tests that non-text operators don't cause errors
func TestIgnoredOperators(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	// Include various graphics operators that should be ignored
	operations := []contentstream.Operation{
		{Operator: "q", Operands: []core.Object{}},
		{Operator: "w", Operands: []core.Object{core.Int(1)}},               // line width
		{Operator: "J", Operands: []core.Object{core.Int(0)}},               // line cap
		{Operator: "j", Operands: []core.Object{core.Int(0)}},               // line join
		{Operator: "M", Operands: []core.Object{core.Int(10)}},              // miter limit
		{Operator: "d", Operands: []core.Object{core.Array{}, core.Int(0)}}, // dash pattern
		{Operator: "ri", Operands: []core.Object{core.Name("RelativeColorimetric")}},
		{Operator: "i", Operands: []core.Object{core.Int(1)}},       // flatness
		{Operator: "gs", Operands: []core.Object{core.Name("GS1")}}, // graphics state
		{Operator: "m", Operands: []core.Object{core.Int(0), core.Int(0)}},
		{Operator: "l", Operands: []core.Object{core.Int(100), core.Int(100)}},
		{Operator: "c", Operands: []core.Object{core.Int(10), core.Int(10), core.Int(20), core.Int(20), core.Int(30), core.Int(30)}},
		{Operator: "v", Operands: []core.Object{core.Int(10), core.Int(10), core.Int(20), core.Int(20)}},
		{Operator: "y", Operands: []core.Object{core.Int(10), core.Int(10), core.Int(20), core.Int(20)}},
		{Operator: "h", Operands: []core.Object{}},
		{Operator: "re", Operands: []core.Object{core.Int(0), core.Int(0), core.Int(100), core.Int(100)}},
		{Operator: "S", Operands: []core.Object{}},
		{Operator: "s", Operands: []core.Object{}},
		{Operator: "f", Operands: []core.Object{}},
		{Operator: "F", Operands: []core.Object{}},
		{Operator: "f*", Operands: []core.Object{}},
		{Operator: "B", Operands: []core.Object{}},
		{Operator: "B*", Operands: []core.Object{}},
		{Operator: "b", Operands: []core.Object{}},
		{Operator: "b*", Operands: []core.Object{}},
		{Operator: "n", Operands: []core.Object{}},
		{Operator: "W", Operands: []core.Object{}},
		{Operator: "W*", Operands: []core.Object{}},
		{Operator: "Q", Operands: []core.Object{}},
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Test")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	fragments, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(fragments) != 1 || fragments[0].Text != "Test" {
		t.Error("expected single fragment with text 'Test'")
	}
}

// TestGetTextWithMultipleLines tests text extraction with multiple lines
func TestGetTextWithMultipleLines(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Td", Operands: []core.Object{core.Int(100), core.Int(700)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Line 1")}},
		{Operator: "Td", Operands: []core.Object{core.Int(0), core.Int(-14)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Line 2")}},
		{Operator: "Td", Operands: []core.Object{core.Int(0), core.Int(-14)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Line 3")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	_, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	text := ex.GetText()

	if !strings.Contains(text, "Line 1") || !strings.Contains(text, "Line 2") || !strings.Contains(text, "Line 3") {
		t.Errorf("expected all lines in text output, got: %q", text)
	}
}

// TestShouldInsertSpaceSmart tests space insertion logic
func TestShouldInsertSpaceSmart(t *testing.T) {
	ex := NewExtractor()
	ex.RegisterFont("/F1", "Helvetica", "Type1")

	// Create a simple word-level extraction scenario
	operations := []contentstream.Operation{
		{Operator: "BT", Operands: []core.Object{}},
		{Operator: "Tf", Operands: []core.Object{core.Name("F1"), core.Int(12)}},
		{Operator: "Td", Operands: []core.Object{core.Int(0), core.Int(100)}},
		{Operator: "Tj", Operands: []core.Object{core.String("Hello")}},
		{Operator: "Td", Operands: []core.Object{core.Int(50), core.Int(0)}}, // Large horizontal gap
		{Operator: "Tj", Operands: []core.Object{core.String("World")}},
		{Operator: "ET", Operands: []core.Object{}},
	}

	_, err := ex.Extract(operations)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	text := ex.GetText()
	// Should have space inserted between "Hello" and "World"
	if !strings.Contains(text, "Hello") || !strings.Contains(text, "World") {
		t.Errorf("unexpected text output: %q", text)
	}
}

// TestAbsFunction tests the abs helper
func TestAbsFunction(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{5.0, 5.0},
		{-5.0, 5.0},
		{0.0, 0.0},
		{-0.0, 0.0},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.expected {
			t.Errorf("abs(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// TestIsWhitespace tests whitespace detection
func TestIsWhitespace(t *testing.T) {
	tests := []struct {
		input    byte
		expected bool
	}{
		{' ', true},
		{'\t', true},
		{'\n', true},
		{'\r', true},
		{'a', false},
		{'0', false},
	}

	for _, tt := range tests {
		result := isWhitespace(tt.input)
		if result != tt.expected {
			t.Errorf("isWhitespace(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
