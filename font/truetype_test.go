package font

import (
	"testing"

	"github.com/tsawler/tabula/core"
)

func TestNewTrueTypeFont_BasicFont(t *testing.T) {
	// Create a minimal TrueType font dictionary
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("TrueType"),
		"BaseFont": core.Name("Arial"),
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	if font.BaseFont != "Arial" {
		t.Errorf("Expected BaseFont 'Arial', got '%s'", font.BaseFont)
	}

	if font.Subtype != "TrueType" {
		t.Errorf("Expected Subtype 'TrueType', got '%s'", font.Subtype)
	}

	if font.Encoding != "WinAnsiEncoding" {
		t.Errorf("Expected default encoding 'WinAnsiEncoding', got '%s'", font.Encoding)
	}
}

func TestNewTrueTypeFont_WithWidths(t *testing.T) {
	// Create a font dictionary with width information
	fontDict := core.Dict{
		"Type":      core.Name("Font"),
		"Subtype":   core.Name("TrueType"),
		"BaseFont":  core.Name("Arial"),
		"FirstChar": core.Int(32),  // Space
		"LastChar":  core.Int(126), // Tilde
		"Widths": core.Array{
			core.Int(278), // Space width
			core.Int(278), // ! width
			core.Int(355), // " width
		},
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
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
	if font.Widths[0] != 278.0 {
		t.Errorf("Expected first width 278.0, got %f", font.Widths[0])
	}

	// Check that width was added to the width map
	spaceWidth := font.GetWidth(' ')
	if spaceWidth != 278.0 {
		t.Errorf("Expected space width 278.0, got %f", spaceWidth)
	}
}

func TestNewTrueTypeFont_NotTrueType(t *testing.T) {
	// Try to create a TrueType font from a non-TrueType dictionary
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("Type1"), // Wrong subtype
		"BaseFont": core.Name("Helvetica"),
	}

	_, err := NewTrueTypeFont(fontDict, mockResolver)
	if err == nil {
		t.Error("Expected error for non-TrueType font, got nil")
	}
}

func TestIsSubsetFont(t *testing.T) {
	tests := []struct {
		name     string
		fontName string
		expected bool
	}{
		{"Subset font", "ABCDEF+Arial", true},
		{"Subset font 2", "XYZABC+TimesRoman", true},
		{"Regular font", "Arial", false},
		{"Short name", "ABC+X", false}, // Too short prefix
		{"No plus", "ABCDEF-Arial", false},
		{"Lowercase prefix", "abcdef+Arial", false},
		{"Mixed case prefix", "AbCdEf+Arial", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSubsetFont(tt.fontName)
			if result != tt.expected {
				t.Errorf("isSubsetFont(%s) = %v, want %v", tt.fontName, result, tt.expected)
			}
		})
	}
}

func TestTrueTypeFont_SubsetDetection(t *testing.T) {
	// Test subset font
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("TrueType"),
		"BaseFont": core.Name("ABCDEF+Arial"),
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	if !font.isSubset {
		t.Error("Font should be detected as subset")
	}

	// Test regular font
	fontDict2 := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("TrueType"),
		"BaseFont": core.Name("Arial"),
	}

	font2, err := NewTrueTypeFont(fontDict2, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	if font2.isSubset {
		t.Error("Font should not be detected as subset")
	}
}

func TestTrueTypeFont_WithEncoding(t *testing.T) {
	tests := []struct {
		name     string
		encoding core.Object
		expected string
	}{
		{"WinAnsi", core.Name("WinAnsiEncoding"), "WinAnsiEncoding"},
		{"MacRoman", core.Name("MacRomanEncoding"), "MacRomanEncoding"},
		{"Identity-H", core.Name("Identity-H"), "Identity-H"},
		{"Default", nil, "WinAnsiEncoding"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fontDict := core.Dict{
				"Type":     core.Name("Font"),
				"Subtype":  core.Name("TrueType"),
				"BaseFont": core.Name("Arial"),
			}

			if tt.encoding != nil {
				fontDict["Encoding"] = tt.encoding
			}

			font, err := NewTrueTypeFont(fontDict, mockResolver)
			if err != nil {
				t.Fatalf("NewTrueTypeFont failed: %v", err)
			}

			if font.Encoding != tt.expected {
				t.Errorf("Expected encoding '%s', got '%s'", tt.expected, font.Encoding)
			}
		})
	}
}

func TestTrueTypeFont_WithCustomEncoding(t *testing.T) {
	// Create a custom encoding dictionary
	encodingDict := core.Dict{
		"Type":         core.Name("Encoding"),
		"BaseEncoding": core.Name("WinAnsiEncoding"),
	}

	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("TrueType"),
		"BaseFont": core.Name("Arial"),
		"Encoding": encodingDict,
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	// Should use base encoding from custom encoding dict
	if font.Encoding != "WinAnsiEncoding" {
		t.Errorf("Expected base encoding 'WinAnsiEncoding', got '%s'", font.Encoding)
	}
}

func TestTrueTypeFont_ParseFontDescriptor(t *testing.T) {
	// Create a font descriptor dictionary
	descriptorDict := core.Dict{
		"Type":         core.Name("FontDescriptor"),
		"FontName":     core.Name("Arial"),
		"Flags":        core.Int(32),
		"FontBBox":     core.Array{core.Int(-665), core.Int(-210), core.Int(2000), core.Int(728)},
		"ItalicAngle":  core.Real(0),
		"Ascent":       core.Int(728),
		"Descent":      core.Int(-210),
		"CapHeight":    core.Int(716),
		"StemV":        core.Int(80),
		"MissingWidth": core.Int(750),
	}

	fontDict := core.Dict{
		"Type":           core.Name("Font"),
		"Subtype":        core.Name("TrueType"),
		"BaseFont":       core.Name("Arial"),
		"FontDescriptor": descriptorDict,
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	if font.FontDescriptor == nil {
		t.Fatal("Font descriptor should be parsed")
	}

	fd := font.FontDescriptor

	if fd.FontName != "Arial" {
		t.Errorf("Expected FontName 'Arial', got '%s'", fd.FontName)
	}

	if fd.Flags != 32 {
		t.Errorf("Expected Flags 32, got %d", fd.Flags)
	}

	if fd.Ascent != 728 {
		t.Errorf("Expected Ascent 728, got %f", fd.Ascent)
	}

	if fd.Descent != -210 {
		t.Errorf("Expected Descent -210, got %f", fd.Descent)
	}
}

func TestTrueTypeFont_CharacterWidthCalculation(t *testing.T) {
	// Create a font with specific widths
	fontDict := core.Dict{
		"Type":      core.Name("Font"),
		"Subtype":   core.Name("TrueType"),
		"BaseFont":  core.Name("Arial"),
		"FirstChar": core.Int(65), // 'A'
		"LastChar":  core.Int(67), // 'C'
		"Widths": core.Array{
			core.Int(722), // A width
			core.Int(667), // B width
			core.Int(722), // C width
		},
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	// Test GetWidth for defined characters
	if w := font.GetWidth('A'); w != 722.0 {
		t.Errorf("Expected width 722.0 for 'A', got %f", w)
	}

	if w := font.GetWidth('B'); w != 667.0 {
		t.Errorf("Expected width 667.0 for 'B', got %f", w)
	}

	if w := font.GetWidth('C'); w != 722.0 {
		t.Errorf("Expected width 722.0 for 'C', got %f", w)
	}

	// Test GetStringWidth
	stringWidth := font.GetStringWidth("ABC")
	expectedWidth := 722.0 + 667.0 + 722.0
	if stringWidth != expectedWidth {
		t.Errorf("Expected string width %f, got %f", expectedWidth, stringWidth)
	}
}

func TestTrueTypeFont_MixedWidthTypes(t *testing.T) {
	// Test widths with mixed Int and Real
	fontDict := core.Dict{
		"Type":      core.Name("Font"),
		"Subtype":   core.Name("TrueType"),
		"BaseFont":  core.Name("Arial"),
		"FirstChar": core.Int(65),
		"LastChar":  core.Int(67),
		"Widths": core.Array{
			core.Int(722),    // Int
			core.Real(667.5), // Real
			core.Int(722),    // Int
		},
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	if font.Widths[0] != 722.0 {
		t.Errorf("Expected width 722.0, got %f", font.Widths[0])
	}

	if font.Widths[1] != 667.5 {
		t.Errorf("Expected width 667.5, got %f", font.Widths[1])
	}
}

func TestTrueTypeFont_EmptyWidths(t *testing.T) {
	// Test empty widths array
	fontDict := core.Dict{
		"Type":      core.Name("Font"),
		"Subtype":   core.Name("TrueType"),
		"BaseFont":  core.Name("Arial"),
		"FirstChar": core.Int(32),
		"LastChar":  core.Int(32),
		"Widths":    core.Array{},
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	if len(font.Widths) != 0 {
		t.Errorf("Expected 0 widths, got %d", len(font.Widths))
	}
}

func TestTrueTypeFont_MissingWidths(t *testing.T) {
	// Test missing widths array (should not fail)
	fontDict := core.Dict{
		"Type":      core.Name("Font"),
		"Subtype":   core.Name("TrueType"),
		"BaseFont":  core.Name("Arial"),
		"FirstChar": core.Int(32),
		"LastChar":  core.Int(126),
		// No Widths array
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	// Should have empty widths array
	if font.Widths != nil && len(font.Widths) > 0 {
		t.Errorf("Expected nil or empty widths, got %d widths", len(font.Widths))
	}
}

func TestTrueTypeFont_CommonFonts(t *testing.T) {
	// Test common TrueType fonts
	commonFonts := []string{
		"Arial",
		"Arial-Bold",
		"Arial-Italic",
		"Arial-BoldItalic",
		"TimesNewRoman",
		"TimesNewRoman-Bold",
		"Verdana",
		"Georgia",
		"CourierNew",
	}

	for _, fontName := range commonFonts {
		t.Run(fontName, func(t *testing.T) {
			fontDict := core.Dict{
				"Type":     core.Name("Font"),
				"Subtype":  core.Name("TrueType"),
				"BaseFont": core.Name(fontName),
			}

			font, err := NewTrueTypeFont(fontDict, mockResolver)
			if err != nil {
				t.Fatalf("NewTrueTypeFont failed for %s: %v", fontName, err)
			}

			if font.BaseFont != fontName {
				t.Errorf("Expected BaseFont '%s', got '%s'", fontName, font.BaseFont)
			}
		})
	}
}

func TestTrueTypeFont_TablesInitialization(t *testing.T) {
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("TrueType"),
		"BaseFont": core.Name("Arial"),
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	// Tables map should be initialized
	if font.Tables == nil {
		t.Error("Tables map should be initialized")
	}

	// glyphWidths map should be initialized
	if font.glyphWidths == nil {
		t.Error("glyphWidths map should be initialized")
	}
}

func TestTrueTypeFont_GetGlyphID(t *testing.T) {
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("TrueType"),
		"BaseFont": core.Name("Arial"),
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	// Without a cmap table, should return 0 (.notdef)
	gid := font.GetGlyphID('A')
	if gid != 0 {
		t.Errorf("Expected glyph ID 0 (no cmap), got %d", gid)
	}
}

func TestTrueTypeFont_GetWidthFromGlyph(t *testing.T) {
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("TrueType"),
		"BaseFont": core.Name("Arial"),
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	// Without glyph widths, should return default
	width := font.GetWidthFromGlyph(0)
	if width != 500.0 {
		t.Errorf("Expected default width 500.0, got %f", width)
	}
}

func TestTrueTypeFont_ToUnicode(t *testing.T) {
	// Test that ToUnicode stream is captured if present
	toUnicodeStream := &core.Stream{
		Dict: core.Dict{"Length": core.Int(100)},
		Data: []byte("dummy cmap data"),
	}

	fontDict := core.Dict{
		"Type":      core.Name("Font"),
		"Subtype":   core.Name("TrueType"),
		"BaseFont":  core.Name("Arial"),
		"ToUnicode": toUnicodeStream,
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	if font.ToUnicode == nil {
		t.Error("ToUnicode stream should be captured")
	}

	if font.ToUnicode != toUnicodeStream {
		t.Error("ToUnicode stream should match provided stream")
	}
}

func TestTrueTypeFont_FontDescriptorOptional(t *testing.T) {
	// Font without descriptor should not fail
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("TrueType"),
		"BaseFont": core.Name("Arial"),
		// No FontDescriptor
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	if font.FontDescriptor != nil {
		t.Error("FontDescriptor should be nil when not provided")
	}
}

func TestTrueTypeFont_InvalidWidthsArray(t *testing.T) {
	// Test invalid width type in array
	fontDict := core.Dict{
		"Type":      core.Name("Font"),
		"Subtype":   core.Name("TrueType"),
		"BaseFont":  core.Name("Arial"),
		"FirstChar": core.Int(65),
		"LastChar":  core.Int(67),
		"Widths": core.Array{
			core.Int(722),
			core.Name("Invalid"), // Invalid type
			core.Int(722),
		},
	}

	_, err := NewTrueTypeFont(fontDict, mockResolver)
	if err == nil {
		t.Error("Expected error for invalid width type, got nil")
	}
}

func TestParseFontProgram_NoProgram(t *testing.T) {
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("TrueType"),
		"BaseFont": core.Name("Arial"),
	}

	font, err := NewTrueTypeFont(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewTrueTypeFont failed: %v", err)
	}

	// Should not have font program
	if font.FontProgram != nil {
		t.Error("FontProgram should be nil when not embedded")
	}

	if len(font.Tables) != 0 {
		t.Error("Tables should be empty when no font program")
	}
}
