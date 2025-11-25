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
