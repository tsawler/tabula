package font

import (
	"testing"
)

// TestWinAnsiEncoding tests Windows CP1252 encoding
func TestWinAnsiEncoding(t *testing.T) {
	enc := WinAnsiEncoding

	tests := []struct {
		name     string
		input    byte
		expected rune
	}{
		{"space", 0x20, ' '},
		{"uppercase A", 0x41, 'A'},
		{"lowercase a", 0x61, 'a'},
		{"euro sign", 0x80, '€'},
		{"smart quote left", 0x91, '\u2018'},  // '
		{"smart quote right", 0x92, '\u2019'}, // '
		{"lowercase e-acute", 0xE9, 'é'},
		{"lowercase c-cedilla", 0xE7, 'ç'},
		{"uppercase A-grave", 0xC0, 'À'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enc.Decode(tt.input)
			if got != tt.expected {
				t.Errorf("WinAnsiEncoding.Decode(0x%02X) = U+%04X, want U+%04X", tt.input, got, tt.expected)
			}
		})
	}
}

// TestMacRomanEncoding tests Mac Roman encoding
func TestMacRomanEncoding(t *testing.T) {
	enc := MacRomanEncoding

	tests := []struct {
		name     string
		input    byte
		expected rune
	}{
		{"space", 0x20, ' '},
		{"uppercase A", 0x41, 'A'},
		{"lowercase a", 0x61, 'a'},
		{"A-umlaut", 0x80, 'Ä'},
		{"e-acute", 0x8E, 'é'},
		{"e-grave", 0x8F, 'è'},
		{"degrees", 0xA1, '°'},
		{"copyright", 0xA9, '©'},
		{"trademark", 0xAA, '™'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enc.Decode(tt.input)
			if got != tt.expected {
				t.Errorf("MacRomanEncoding.Decode(0x%02X) = U+%04X, want U+%04X", tt.input, got, tt.expected)
			}
		})
	}
}

// TestPDFDocEncoding tests PDF's default encoding
func TestPDFDocEncoding(t *testing.T) {
	enc := PDFDocEncoding

	tests := []struct {
		name     string
		input    byte
		expected rune
	}{
		{"space", 0x20, ' '},
		{"uppercase A", 0x41, 'A'},
		{"bullet", 0x80, '•'},
		{"dagger", 0x81, '†'},
		{"double dagger", 0x82, '‡'},
		{"ellipsis", 0x83, '…'},
		{"em dash", 0x84, '—'},
		{"en dash", 0x85, '–'},
		{"euro", 0xA0, '€'},
		{"lowercase e-acute", 0xE9, 'é'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enc.Decode(tt.input)
			if got != tt.expected {
				t.Errorf("PDFDocEncoding.Decode(0x%02X) = U+%04X, want U+%04X", tt.input, got, tt.expected)
			}
		})
	}
}

// TestStandardEncoding tests Adobe StandardEncoding
func TestStandardEncoding(t *testing.T) {
	enc := StandardEncodingTable

	tests := []struct {
		name     string
		input    byte
		expected rune
	}{
		{"space", 0x20, ' '},
		{"uppercase A", 0x41, 'A'},
		{"lowercase a", 0x61, 'a'},
		{"exclamation inverted", 0xA1, '¡'},
		{"cent", 0xA2, '¢'},
		{"pound", 0xA3, '£'},
		{"fraction slash", 0xA4, '⁄'},
		{"yen", 0xA5, '¥'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enc.Decode(tt.input)
			if got != tt.expected {
				t.Errorf("StandardEncoding.Decode(0x%02X) = U+%04X, want U+%04X", tt.input, got, tt.expected)
			}
		})
	}
}

// TestDecodeString tests decoding byte sequences to strings
func TestDecodeString(t *testing.T) {
	tests := []struct {
		name     string
		encoding Encoding
		input    []byte
		expected string
	}{
		{
			name:     "WinAnsi: Hello",
			encoding: WinAnsiEncoding,
			input:    []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F}, // "Hello"
			expected: "Hello",
		},
		{
			name:     "WinAnsi: café",
			encoding: WinAnsiEncoding,
			input:    []byte{0x63, 0x61, 0x66, 0xE9}, // "café"
			expected: "café",
		},
		{
			name:     "PDFDoc: bullet point",
			encoding: PDFDocEncoding,
			input:    []byte{0x80, 0x20, 0x54, 0x65, 0x78, 0x74}, // "• Text"
			expected: "• Text",
		},
		{
			name:     "MacRoman: naïve",
			encoding: MacRomanEncoding,
			input:    []byte{0x6E, 0x61, 0x95, 0x76, 0x65}, // "naïve" (0x95 = ï)
			expected: "naïve",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.encoding.DecodeString(tt.input)
			if got != tt.expected {
				t.Errorf("DecodeString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestGetEncoding tests the encoding lookup function
func TestGetEncoding(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		expected string
	}{
		{"WinAnsiEncoding", "WinAnsiEncoding", "WinAnsiEncoding"},
		{"MacRomanEncoding", "MacRomanEncoding", "MacRomanEncoding"},
		{"PDFDocEncoding", "PDFDocEncoding", "PDFDocEncoding"},
		{"StandardEncoding", "StandardEncoding", "StandardEncoding"},
		{"Unknown defaults to WinAnsi", "UnknownEncoding", "WinAnsiEncoding"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := GetEncoding(tt.encoding)
			if enc.Name() != tt.expected {
				t.Errorf("GetEncoding(%q).Name() = %q, want %q", tt.encoding, enc.Name(), tt.expected)
			}
		})
	}
}

// TestNormalizeUnicode tests Unicode normalization to NFC
func TestNormalizeUnicode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized",
			input:    "café",
			expected: "café",
		},
		{
			name:     "decomposed to composed",
			input:    "café", // e + combining acute
			expected: "café", // é as single character
		},
		{
			name:     "ASCII unchanged",
			input:    "Hello World",
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeUnicode(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeUnicode(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestDecodeWithEncoding tests the convenience function
func TestDecodeWithEncoding(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		encodingName string
		expected     string
	}{
		{
			name:         "WinAnsi: simple text",
			data:         []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F},
			encodingName: "WinAnsiEncoding",
			expected:     "Hello",
		},
		{
			name:         "WinAnsi: accented characters",
			data:         []byte{0xE9, 0xE8, 0xEA, 0xEB}, // éèêë
			encodingName: "WinAnsiEncoding",
			expected:     "éèêë",
		},
		{
			name:         "PDFDoc: special characters",
			data:         []byte{0x80, 0x81, 0x82}, // •††
			encodingName: "PDFDocEncoding",
			expected:     "•†‡",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeWithEncoding(tt.data, tt.encodingName)
			if got != tt.expected {
				t.Errorf("DecodeWithEncoding() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestFontDecodeStringWithEncoding tests Font.DecodeString using encodings
func TestFontDecodeStringWithEncoding(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		input    []byte
		expected string
	}{
		{
			name:     "WinAnsi encoding",
			encoding: "WinAnsiEncoding",
			input:    []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F}, // "Hello"
			expected: "Hello",
		},
		{
			name:     "WinAnsi with accents",
			encoding: "WinAnsiEncoding",
			input:    []byte{0x63, 0x61, 0x66, 0xE9}, // "café"
			expected: "café",
		},
		{
			name:     "PDFDoc encoding",
			encoding: "PDFDocEncoding",
			input:    []byte{0x80, 0x20, 0x54, 0x65, 0x73, 0x74}, // "• Test"
			expected: "• Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			font := &Font{
				Name:     "TestFont",
				BaseFont: "Helvetica",
				Subtype:  "Type1",
				Encoding: tt.encoding,
				widths:   make(map[rune]float64),
			}

			got := font.DecodeString(tt.input)
			if got != tt.expected {
				t.Errorf("Font.DecodeString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestFontDecodeStringPriority tests the priority order in Font.DecodeString
func TestFontDecodeStringPriority(t *testing.T) {
	// Create a simple CMap
	cmap := &CMap{
		charMappings:  make(map[uint32]string),
		rangeMappings: []CMapRange{},
	}
	cmap.charMappings[0x41] = "X" // Map 'A' to 'X'

	font := &Font{
		Name:          "TestFont",
		BaseFont:      "Helvetica",
		Subtype:       "Type1",
		Encoding:      "WinAnsiEncoding",
		widths:        make(map[rune]float64),
		ToUnicodeCMap: cmap,
	}

	// Test that ToUnicode CMap takes priority over Encoding
	input := []byte{0x41} // 'A'
	got := font.DecodeString(input)

	// Should use CMap (maps to "X"), not WinAnsi (would be "A")
	if got != "X" {
		t.Errorf("Font.DecodeString() = %q, want %q (CMap should take priority)", got, "X")
	}

	// Test without CMap - should use encoding
	font.ToUnicodeCMap = nil
	got = font.DecodeString(input)
	if got != "A" {
		t.Errorf("Font.DecodeString() = %q, want %q (should use encoding)", got, "A")
	}
}

// TestIsValidUTF8 tests UTF-8 validation
func TestIsValidUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid ASCII", "Hello", true},
		{"valid UTF-8 with accents", "café", true},
		{"valid UTF-8 with emoji", "Hello 👋", true},
		{"invalid UTF-8", string([]byte{0xFF, 0xFE}), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidUTF8(tt.input)
			if got != tt.expected {
				t.Errorf("IsValidUTF8(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// TestWinAnsiExtendedCharacters tests characters specific to Windows CP1252
func TestWinAnsiExtendedCharacters(t *testing.T) {
	enc := WinAnsiEncoding

	tests := []struct {
		byte byte
		want rune
		name string
	}{
		{0x80, 0x20AC, "Euro"},       // €
		{0x82, 0x201A, "Single Low-9 Quotation"}, // ‚
		{0x83, 0x0192, "Latin Small Letter F with Hook"}, // ƒ
		{0x84, 0x201E, "Double Low-9 Quotation"}, // „
		{0x85, 0x2026, "Horizontal Ellipsis"}, // …
		{0x86, 0x2020, "Dagger"}, // †
		{0x87, 0x2021, "Double Dagger"}, // ‡
		{0x91, 0x2018, "Left Single Quotation"}, // '
		{0x92, 0x2019, "Right Single Quotation"}, // '
		{0x93, 0x201C, "Left Double Quotation"}, // "
		{0x94, 0x201D, "Right Double Quotation"}, // "
		{0x95, 0x2022, "Bullet"}, // •
		{0x96, 0x2013, "En Dash"}, // –
		{0x97, 0x2014, "Em Dash"}, // —
		{0x99, 0x2122, "Trademark"}, // ™
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enc.Decode(tt.byte)
			if got != tt.want {
				t.Errorf("WinAnsi[0x%02X] = U+%04X, want U+%04X (%s)", tt.byte, got, tt.want, tt.name)
			}
		})
	}
}

// TestCustomEncodingWithRunes tests custom encoding using direct rune mappings
func TestCustomEncodingWithRunes(t *testing.T) {
	// Create custom encoding: WinAnsi with byte 0x80 mapped to 'X' instead of Euro
	differences := map[byte]rune{
		0x80: 'X',
		0x81: 'Y',
		0x82: 'Z',
	}

	customEnc := NewCustomEncoding(WinAnsiEncoding, differences)

	tests := []struct {
		name     string
		input    byte
		expected rune
	}{
		{"custom mapping 0x80", 0x80, 'X'},
		{"custom mapping 0x81", 0x81, 'Y'},
		{"custom mapping 0x82", 0x82, 'Z'},
		{"base encoding 0x41", 0x41, 'A'},  // Should fall through to base encoding
		{"base encoding 0xE9", 0xE9, 'é'},  // Should fall through to base encoding
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := customEnc.Decode(tt.input)
			if got != tt.expected {
				t.Errorf("CustomEncoding.Decode(0x%02X) = %c (U+%04X), want %c (U+%04X)",
					tt.input, got, got, tt.expected, tt.expected)
			}
		})
	}
}

// TestCustomEncodingFromGlyphs tests custom encoding using glyph names
func TestCustomEncodingFromGlyphs(t *testing.T) {
	// Example: PDF Differences array that remaps some characters
	// /Differences [39 /quoteright 96 /quoteleft 128 /Euro]
	differences := map[byte]string{
		39:  "quoteright", // ' instead of '
		96:  "quoteleft",  // ' instead of `
		128: "Euro",       // € (ensure it's mapped)
	}

	customEnc := NewCustomEncodingFromGlyphs(StandardEncodingTable, differences)

	tests := []struct {
		name     string
		input    byte
		expected rune
	}{
		{"apostrophe → right quote", 39, '\u2019'},  // '
		{"grave → left quote", 96, '\u2018'},        // '
		{"byte 128 → Euro", 128, '€'},
		{"base encoding: A", 0x41, 'A'},             // Should use base
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := customEnc.Decode(tt.input)
			if got != tt.expected {
				t.Errorf("CustomEncoding.Decode(0x%02X) = U+%04X, want U+%04X",
					tt.input, got, tt.expected)
			}
		})
	}
}

// TestCustomEncodingDecodeString tests string decoding with custom encoding
func TestCustomEncodingDecodeString(t *testing.T) {
	// Create custom encoding that swaps some letters
	differences := map[byte]rune{
		'A': 'Z',
		'B': 'Y',
		'C': 'X',
	}

	customEnc := NewCustomEncoding(WinAnsiEncoding, differences)

	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"custom letters", []byte{'A', 'B', 'C'}, "ZYX"},
		{"mixed custom and base", []byte{'A', 'D', 'B'}, "ZDY"},
		{"only base encoding", []byte{'D', 'E', 'F'}, "DEF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := customEnc.DecodeString(tt.input)
			if got != tt.expected {
				t.Errorf("CustomEncoding.DecodeString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestGlyphNameToUnicode tests the glyph name mapping table
func TestGlyphNameToUnicode(t *testing.T) {
	tests := []struct {
		glyphName string
		expected  rune
	}{
		{"space", ' '},
		{"A", 'A'},
		{"a", 'a'},
		{"Euro", '€'},
		{"bullet", '•'},
		{"eacute", 'é'},
		{"Ntilde", 'Ñ'},
		{"quoteright", '\u2019'},
		{"quoteleft", '\u2018'},
		{"emdash", '—'},
		{"endash", '–'},
		{"trademark", '™'},
		{"copyright", '©'},
		{"registered", '®'},
	}

	for _, tt := range tests {
		t.Run(tt.glyphName, func(t *testing.T) {
			got, ok := glyphNameToUnicode[tt.glyphName]
			if !ok {
				t.Errorf("glyphNameToUnicode[%q] not found", tt.glyphName)
				return
			}
			if got != tt.expected {
				t.Errorf("glyphNameToUnicode[%q] = U+%04X, want U+%04X", tt.glyphName, got, tt.expected)
			}
		})
	}
}

// TestCustomEncodingRealWorldExample tests a realistic PDF Differences scenario
func TestCustomEncodingRealWorldExample(t *testing.T) {
	// Real PDF example: StandardEncoding with custom smart quotes
	// Many PDFs do this to ensure proper quote rendering
	differences := map[byte]string{
		0x27: "quoteright",    // ASCII apostrophe → smart right quote
		0x60: "quoteleft",     // ASCII grave → smart left quote
		0x91: "quoteleft",     // WinAnsi left quote position
		0x92: "quoteright",    // WinAnsi right quote position
		0x93: "quotedblleft",  // WinAnsi left double quote
		0x94: "quotedblright", // WinAnsi right double quote
	}

	customEnc := NewCustomEncodingFromGlyphs(StandardEncodingTable, differences)

	// Test string: "Don't use 'dumb' quotes"
	// With bytes: 0x22 (") + Don + 0x27 (') + t use + 0x60 (`) + dumb + 0x27 (') + quotes + 0x22 (")
	input := []byte{0x22, 'D', 'o', 'n', 0x27, 't', ' ', 'u', 's', 'e', ' ', 0x60, 'd', 'u', 'm', 'b', 0x27, ' ', 'q', 'u', 'o', 't', 'e', 's', 0x22}

	result := customEnc.DecodeString(input)

	// Should have smart quotes: "Don't use 'dumb' quotes"
	// 0x27 → ' (U+2019)
	// 0x60 → ' (U+2018)
	if !contains(result, '\u2019') {
		t.Errorf("Expected smart right quote (U+2019) in result, got: %q", result)
	}
	if !contains(result, '\u2018') {
		t.Errorf("Expected smart left quote (U+2018) in result, got: %q", result)
	}
}

// TestCustomEncodingName tests that custom encoding has correct name
func TestCustomEncodingName(t *testing.T) {
	differences := map[byte]rune{
		0x80: 'X',
	}

	customEnc := NewCustomEncoding(WinAnsiEncoding, differences)

	name := customEnc.Name()
	expected := "WinAnsiEncoding+custom"

	if name != expected {
		t.Errorf("CustomEncoding.Name() = %q, want %q", name, expected)
	}
}

// Helper function to check if string contains a rune
func contains(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
