package font

import (
	"testing"

	"github.com/tsawler/tabula/core"
)

// mockResolver is a simple resolver for testing
func mockResolver(ref core.IndirectRef) (core.Object, error) {
	// Return nil for now - tests will provide direct objects
	return nil, nil
}

func TestNewType1Font_BasicFont(t *testing.T) {
	// Create a minimal Type1 font dictionary
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("Type1"),
		"BaseFont": core.Name("Helvetica"),
	}

	font, err := NewType1Font(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewType1Font failed: %v", err)
	}

	if font.BaseFont != "Helvetica" {
		t.Errorf("Expected BaseFont 'Helvetica', got '%s'", font.BaseFont)
	}

	if font.Subtype != "Type1" {
		t.Errorf("Expected Subtype 'Type1', got '%s'", font.Subtype)
	}

	if font.Encoding != "StandardEncoding" {
		t.Errorf("Expected default encoding 'StandardEncoding', got '%s'", font.Encoding)
	}
}

func TestNewType1Font_WithWidths(t *testing.T) {
	// Create a font dictionary with width information
	fontDict := core.Dict{
		"Type":      core.Name("Font"),
		"Subtype":   core.Name("Type1"),
		"BaseFont":  core.Name("CustomFont"),
		"FirstChar": core.Int(32),  // Space
		"LastChar":  core.Int(126), // Tilde
		"Widths": core.Array{
			core.Real(250.0), // Space width
			core.Real(333.0), // ! width
			core.Real(408.0), // " width
			// ... (in real test, would have all 95 characters)
		},
	}

	font, err := NewType1Font(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewType1Font failed: %v", err)
	}

	if font.FirstChar != 32 {
		t.Errorf("Expected FirstChar 32, got %d", font.FirstChar)
	}

	if font.LastChar != 126 {
		t.Errorf("Expected LastChar 126, got %d", font.LastChar)
	}

	if len(font.Widths) != 3 {
		t.Errorf("Expected 3 widths, got %d", len(font.Widths))
	}

	// Check that widths were parsed correctly
	if font.Widths[0] != 250.0 {
		t.Errorf("Expected first width 250.0, got %f", font.Widths[0])
	}

	// Check that width was added to the width map
	spaceWidth := font.GetWidth(' ')
	if spaceWidth != 250.0 {
		t.Errorf("Expected space width 250.0, got %f", spaceWidth)
	}
}

func TestNewType1Font_WithNamedEncoding(t *testing.T) {
	tests := []struct {
		name     string
		encoding core.Name
		expected string
	}{
		{"WinAnsi", "WinAnsiEncoding", "WinAnsiEncoding"},
		{"MacRoman", "MacRomanEncoding", "MacRomanEncoding"},
		{"MacExpert", "MacExpertEncoding", "MacExpertEncoding"},
		{"Standard", "StandardEncoding", "StandardEncoding"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fontDict := core.Dict{
				"Type":     core.Name("Font"),
				"Subtype":  core.Name("Type1"),
				"BaseFont": core.Name("TestFont"),
				"Encoding": tt.encoding,
			}

			font, err := NewType1Font(fontDict, mockResolver)
			if err != nil {
				t.Fatalf("NewType1Font failed: %v", err)
			}

			if font.Encoding != tt.expected {
				t.Errorf("Expected encoding '%s', got '%s'", tt.expected, font.Encoding)
			}
		})
	}
}

func TestNewType1Font_WithCustomEncoding(t *testing.T) {
	// Create a custom encoding with Differences
	encodingDict := core.Dict{
		"Type":         core.Name("Encoding"),
		"BaseEncoding": core.Name("WinAnsiEncoding"),
		"Differences": core.Array{
			core.Int(39),        // Starting at character code 39
			core.Name("quotesingle"), // Replace with quotesingle glyph
			core.Int(96),        // Starting at character code 96
			core.Name("grave"),  // Replace with grave glyph
		},
	}

	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("Type1"),
		"BaseFont": core.Name("TestFont"),
		"Encoding": encodingDict,
	}

	font, err := NewType1Font(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewType1Font failed: %v", err)
	}

	// Should use base encoding
	if font.Encoding != "WinAnsiEncoding" {
		t.Errorf("Expected base encoding 'WinAnsiEncoding', got '%s'", font.Encoding)
	}
}

func TestNewType1Font_NotType1(t *testing.T) {
	// Try to create a Type1 font from a non-Type1 dictionary
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("TrueType"), // Wrong subtype
		"BaseFont": core.Name("Arial"),
	}

	_, err := NewType1Font(fontDict, mockResolver)
	if err == nil {
		t.Error("Expected error for non-Type1 font, got nil")
	}
}

func TestNewType1Font_StandardFont(t *testing.T) {
	// Test that Standard 14 fonts work without font descriptors
	standardFonts := []string{
		"Helvetica",
		"Helvetica-Bold",
		"Helvetica-Oblique",
		"Helvetica-BoldOblique",
		"Times-Roman",
		"Times-Bold",
		"Times-Italic",
		"Times-BoldItalic",
		"Courier",
		"Courier-Bold",
		"Courier-Oblique",
		"Courier-BoldOblique",
		"Symbol",
		"ZapfDingbats",
	}

	for _, fontName := range standardFonts {
		t.Run(fontName, func(t *testing.T) {
			fontDict := core.Dict{
				"Type":     core.Name("Font"),
				"Subtype":  core.Name("Type1"),
				"BaseFont": core.Name(fontName),
			}

			font, err := NewType1Font(fontDict, mockResolver)
			if err != nil {
				t.Fatalf("NewType1Font failed for %s: %v", fontName, err)
			}

			if !font.IsStandardFont() {
				t.Errorf("Font %s should be recognized as standard font", fontName)
			}

			// Standard fonts should have widths loaded
			width := font.GetWidth('A')
			if width == 0 {
				t.Errorf("Font %s should have width for 'A'", fontName)
			}
		})
	}
}

func TestParseFontDescriptor(t *testing.T) {
	// Create a font descriptor dictionary
	descriptorDict := core.Dict{
		"Type":         core.Name("FontDescriptor"),
		"FontName":     core.Name("TestFont-Regular"),
		"Flags":        core.Int(32),
		"FontBBox":     core.Array{core.Real(-100), core.Real(-200), core.Real(1000), core.Real(800)},
		"ItalicAngle":  core.Real(0),
		"Ascent":       core.Real(750),
		"Descent":      core.Real(-250),
		"CapHeight":    core.Real(700),
		"StemV":        core.Real(80),
		"MissingWidth": core.Real(500),
	}

	fontDict := core.Dict{
		"Type":           core.Name("Font"),
		"Subtype":        core.Name("Type1"),
		"BaseFont":       core.Name("TestFont-Regular"),
		"FontDescriptor": descriptorDict,
	}

	font, err := NewType1Font(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewType1Font failed: %v", err)
	}

	if font.FontDescriptor == nil {
		t.Fatal("Font descriptor should be parsed")
	}

	fd := font.FontDescriptor

	if fd.FontName != "TestFont-Regular" {
		t.Errorf("Expected FontName 'TestFont-Regular', got '%s'", fd.FontName)
	}

	if fd.Flags != 32 {
		t.Errorf("Expected Flags 32, got %d", fd.Flags)
	}

	if fd.Ascent != 750 {
		t.Errorf("Expected Ascent 750, got %f", fd.Ascent)
	}

	if fd.Descent != -250 {
		t.Errorf("Expected Descent -250, got %f", fd.Descent)
	}

	if fd.CapHeight != 700 {
		t.Errorf("Expected CapHeight 700, got %f", fd.CapHeight)
	}

	if fd.FontBBox[0] != -100 || fd.FontBBox[1] != -200 || fd.FontBBox[2] != 1000 || fd.FontBBox[3] != 800 {
		t.Errorf("FontBBox not parsed correctly: %v", fd.FontBBox)
	}
}

func TestExtractName(t *testing.T) {
	tests := []struct {
		name     string
		input    core.Object
		expected string
	}{
		{"Name", core.Name("TestName"), "TestName"},
		{"String", core.String("TestString"), "TestString"},
		{"Nil", nil, ""},
		{"Int", core.Int(123), ""}, // Should return empty for non-name types
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    core.Object
		expected float64
	}{
		{"Int", core.Int(42), 42.0},
		{"Real", core.Real(3.14), 3.14},
		{"Nil", nil, 0.0},
		{"Name", core.Name("test"), 0.0}, // Should return 0 for non-numeric types
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNumber(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestCharacterWidthCalculation(t *testing.T) {
	// Create a font with specific widths
	fontDict := core.Dict{
		"Type":      core.Name("Font"),
		"Subtype":   core.Name("Type1"),
		"BaseFont":  core.Name("TestFont"),
		"FirstChar": core.Int(65), // 'A'
		"LastChar":  core.Int(67), // 'C'
		"Widths": core.Array{
			core.Real(700.0), // A width
			core.Real(600.0), // B width
			core.Real(650.0), // C width
		},
	}

	font, err := NewType1Font(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewType1Font failed: %v", err)
	}

	// Test GetWidth for defined characters
	if w := font.GetWidth('A'); w != 700.0 {
		t.Errorf("Expected width 700.0 for 'A', got %f", w)
	}

	if w := font.GetWidth('B'); w != 600.0 {
		t.Errorf("Expected width 600.0 for 'B', got %f", w)
	}

	if w := font.GetWidth('C'); w != 650.0 {
		t.Errorf("Expected width 650.0 for 'C', got %f", w)
	}

	// Test GetWidth for undefined character (should return default)
	// Using a character outside the defined range (65-67 = A-C)
	// Since this is a non-standard font without defaults, it will get the fallback width
	if w := font.GetWidth('Ω'); w != 500.0 { // Greek Omega - not in standard ASCII
		t.Errorf("Expected default width 500.0 for 'Ω', got %f", w)
	}

	// Test GetStringWidth
	stringWidth := font.GetStringWidth("ABC")
	expectedWidth := 700.0 + 600.0 + 650.0
	if stringWidth != expectedWidth {
		t.Errorf("Expected string width %f, got %f", expectedWidth, stringWidth)
	}
}

func TestFontDescriptorFlags(t *testing.T) {
	// Test common font descriptor flags
	tests := []struct {
		name     string
		flags    int
		isFixedPitch bool
		isSerif      bool
		isSymbolic   bool
		isItalic     bool
		isBold       bool
	}{
		{"Proportional Sans", 0x20, false, false, false, false, false},  // Bit 6: Nonsymbolic
		{"Fixed Pitch", 0x21, true, false, false, false, false},         // Bits 1,6
		{"Serif", 0x22, false, true, false, false, false},               // Bits 2,6
		{"Symbolic", 0x04, false, false, true, false, false},            // Bit 3
		{"Italic", 0x40, false, false, false, true, false},              // Bit 7
		{"Bold", 0x40000, false, false, false, false, true},             // Bit 19
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			descriptorDict := core.Dict{
				"Type":     core.Name("FontDescriptor"),
				"FontName": core.Name("TestFont"),
				"Flags":    core.Int(tt.flags),
			}

			fontDict := core.Dict{
				"Type":           core.Name("Font"),
				"Subtype":        core.Name("Type1"),
				"BaseFont":       core.Name("TestFont"),
				"FontDescriptor": descriptorDict,
			}

			font, err := NewType1Font(fontDict, mockResolver)
			if err != nil {
				t.Fatalf("NewType1Font failed: %v", err)
			}

			if font.FontDescriptor.Flags != tt.flags {
				t.Errorf("Expected flags %d, got %d", tt.flags, font.FontDescriptor.Flags)
			}

			// Could test individual flag bits here
			// isFixedPitch := (flags & 0x01) != 0
			// isSerif := (flags & 0x02) != 0
			// etc.
		})
	}
}

func TestEncodingDifferences(t *testing.T) {
	// Test the differences array parsing
	diffs := core.Array{
		core.Int(39),              // Start at code 39
		core.Name("quotesingle"),  // Map to quotesingle
		core.Name("quoteright"),   // Map to quoteright (code 40)
		core.Int(96),              // Start at code 96
		core.Name("grave"),        // Map to grave
	}

	t1 := &Type1Font{
		Font: NewFont("Test", "Test", "Type1"),
	}

	err := t1.applyEncodingDifferences(diffs)
	if err != nil {
		t.Fatalf("applyEncodingDifferences failed: %v", err)
	}

	// The function should process the differences without error
	// Full implementation would update character code to glyph name mappings
}

func TestWidthsArrayEdgeCases(t *testing.T) {
	// Test empty widths array
	t.Run("EmptyWidths", func(t *testing.T) {
		fontDict := core.Dict{
			"Type":      core.Name("Font"),
			"Subtype":   core.Name("Type1"),
			"BaseFont":  core.Name("TestFont"),
			"FirstChar": core.Int(32),
			"LastChar":  core.Int(32),
			"Widths":    core.Array{},
		}

		font, err := NewType1Font(fontDict, mockResolver)
		if err != nil {
			t.Fatalf("NewType1Font failed: %v", err)
		}

		if len(font.Widths) != 0 {
			t.Errorf("Expected 0 widths, got %d", len(font.Widths))
		}
	})

	// Test missing widths array
	t.Run("MissingWidths", func(t *testing.T) {
		fontDict := core.Dict{
			"Type":      core.Name("Font"),
			"Subtype":   core.Name("Type1"),
			"BaseFont":  core.Name("Helvetica"), // Standard font, should still work
			"FirstChar": core.Int(32),
			"LastChar":  core.Int(126),
			// No Widths array
		}

		font, err := NewType1Font(fontDict, mockResolver)
		if err != nil {
			t.Fatalf("NewType1Font failed: %v", err)
		}

		// Should fall back to standard font widths
		if !font.IsStandardFont() {
			t.Error("Should recognize as standard font")
		}
	})

	// Test widths with mixed Int and Real
	t.Run("MixedWidths", func(t *testing.T) {
		fontDict := core.Dict{
			"Type":      core.Name("Font"),
			"Subtype":   core.Name("Type1"),
			"BaseFont":  core.Name("TestFont"),
			"FirstChar": core.Int(65),
			"LastChar":  core.Int(67),
			"Widths": core.Array{
				core.Int(700),    // Int
				core.Real(600.5), // Real
				core.Int(650),    // Int
			},
		}

		font, err := NewType1Font(fontDict, mockResolver)
		if err != nil {
			t.Fatalf("NewType1Font failed: %v", err)
		}

		if font.Widths[0] != 700.0 {
			t.Errorf("Expected width 700.0, got %f", font.Widths[0])
		}

		if font.Widths[1] != 600.5 {
			t.Errorf("Expected width 600.5, got %f", font.Widths[1])
		}
	})
}
