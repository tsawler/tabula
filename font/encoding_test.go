package font

import "testing"

func TestDecodeUTF16BE(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Simple ASCII text",
			input:    []byte{0x00, 0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F}, // "Hello"
			expected: "Hello",
		},
		{
			name:     "Emoji waving hand (surrogate pair)",
			input:    []byte{0xD8, 0x3D, 0xDC, 0x4B}, // U+1F44B 👋
			expected: "👋",
		},
		{
			name: "Hello with emoji",
			input: []byte{
				0x00, 0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, // "Hello"
				0x00, 0x20, // space
				0xD8, 0x3D, 0xDC, 0x4B, // 👋 U+1F44B
			},
			expected: "Hello 👋",
		},
		{
			name:     "Grinning face emoji",
			input:    []byte{0xD8, 0x3D, 0xDE, 0x00}, // U+1F600 😀
			expected: "😀",
		},
		{
			name:     "Empty input",
			input:    []byte{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeUTF16BE(tt.input)
			if result != tt.expected {
				t.Errorf("DecodeUTF16BE() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDecodeUTF16LE(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Simple ASCII text",
			input:    []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00}, // "Hello"
			expected: "Hello",
		},
		{
			name:     "Emoji waving hand (surrogate pair)",
			input:    []byte{0x3D, 0xD8, 0x4B, 0xDC}, // U+1F44B 👋
			expected: "👋",
		},
		{
			name: "Hello with emoji",
			input: []byte{
				0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00, // "Hello"
				0x20, 0x00, // space
				0x3D, 0xD8, 0x4B, 0xDC, // 👋 U+1F44B
			},
			expected: "Hello 👋",
		},
		{
			name:     "Empty input",
			input:    []byte{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeUTF16LE(tt.input)
			if result != tt.expected {
				t.Errorf("DecodeUTF16LE() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFontDecodeStringWithUTF16BOM(t *testing.T) {
	font := NewFont("TestFont", "Helvetica", "Type1")

	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name: "UTF-16BE with BOM - Hello emoji",
			input: []byte{
				0xFE, 0xFF, // BOM (UTF-16BE)
				0x00, 0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, // "Hello"
				0x00, 0x20, // space
				0xD8, 0x3D, 0xDC, 0x4B, // 👋 U+1F44B
			},
			expected: "Hello 👋",
		},
		{
			name: "UTF-16LE with BOM - Hello emoji",
			input: []byte{
				0xFF, 0xFE, // BOM (UTF-16LE)
				0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00, // "Hello"
				0x20, 0x00, // space
				0x3D, 0xD8, 0x4B, 0xDC, // 👋 U+1F44B
			},
			expected: "Hello 👋",
		},
		{
			name: "No BOM - falls back to encoding",
			input: []byte{
				0x48, 0x65, 0x6C, 0x6C, 0x6F, // "Hello" as single bytes
			},
			expected: "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := font.DecodeString(tt.input)
			if result != tt.expected {
				t.Errorf("Font.DecodeString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsEmojiSequence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Simple emoji",
			input:    "Hello 👋",
			expected: true,
		},
		{
			name:     "No emoji",
			input:    "Hello World",
			expected: false,
		},
		{
			name:     "ZWJ sequence (family)",
			input:    "👨‍👩‍👧‍👦",
			expected: true,
		},
		{
			name:     "Flag emoji",
			input:    "🇺🇸",
			expected: true,
		},
		{
			name:     "Star emoji",
			input:    "⭐",
			expected: true,
		},
		{
			name:     "Multiple emoji",
			input:    "😀😃😄",
			expected: true,
		},
		{
			name:     "Emoji with text",
			input:    "I love coding 💻",
			expected: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Only text with accents",
			input:    "café résumé",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmojiSequence(tt.input)
			if result != tt.expected {
				t.Errorf("IsEmojiSequence(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStandardEncodingDecode(t *testing.T) {
	enc := WinAnsiEncoding

	tests := []struct {
		name     string
		input    byte
		expected rune
	}{
		{"space", 0x20, ' '},
		{"A", 0x41, 'A'},
		{"z", 0x7A, 'z'},
		{"euro", 0x80, '€'},
		{"bullet", 0x95, '•'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enc.Decode(tt.input)
			if result != tt.expected {
				t.Errorf("Decode(0x%02X) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStandardEncodingName(t *testing.T) {
	tests := []struct {
		enc  Encoding
		name string
	}{
		{WinAnsiEncoding, "WinAnsiEncoding"},
		{MacRomanEncoding, "MacRomanEncoding"},
		{PDFDocEncoding, "PDFDocEncoding"},
		{StandardEncodingTable, "StandardEncoding"},
		{SymbolEncoding, "SymbolEncoding"},
		{ZapfDingbatsEncoding, "ZapfDingbatsEncoding"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.enc.Name(); got != tt.name {
				t.Errorf("Name() = %q, want %q", got, tt.name)
			}
		})
	}
}

func TestGetEncoding(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"WinAnsiEncoding", "WinAnsiEncoding"},
		{"MacRomanEncoding", "MacRomanEncoding"},
		{"PDFDocEncoding", "PDFDocEncoding"},
		{"StandardEncoding", "StandardEncoding"},
		{"SymbolEncoding", "SymbolEncoding"},
		{"ZapfDingbatsEncoding", "ZapfDingbatsEncoding"},
		{"UnknownEncoding", "WinAnsiEncoding"}, // Falls back to WinAnsi
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := GetEncoding(tt.name)
			if enc.Name() != tt.expected {
				t.Errorf("GetEncoding(%q).Name() = %q, want %q", tt.name, enc.Name(), tt.expected)
			}
		})
	}
}

func TestInferEncodingFromFontName(t *testing.T) {
	tests := []struct {
		fontName    string
		expectedEnc string
	}{
		// Symbol fonts
		{"Symbol", "SymbolEncoding"},
		{"ZapfDingbats", "ZapfDingbatsEncoding"},
		{"Wingdings", "SymbolEncoding"},

		// CJK fonts - return WinAnsi as fallback
		{"MSMincho", "WinAnsiEncoding"},
		{"SimSun", "WinAnsiEncoding"},
		{"Batang", "WinAnsiEncoding"},

		// Mac fonts
		{"Menlo", "MacRomanEncoding"},
		{"Monaco", "MacRomanEncoding"},

		// Windows/standard fonts
		{"Arial", "WinAnsiEncoding"},
		{"Verdana", "WinAnsiEncoding"},
		{"Calibri", "WinAnsiEncoding"},

		// PostScript fonts
		{"Times-Roman", "StandardEncoding"},
		{"Courier", "StandardEncoding"},
		{"Helvetica", "StandardEncoding"},

		// Unknown - falls back to WinAnsi
		{"RandomFontName", "WinAnsiEncoding"},
	}

	for _, tt := range tests {
		t.Run(tt.fontName, func(t *testing.T) {
			enc := InferEncodingFromFontName(tt.fontName)
			if enc.Name() != tt.expectedEnc {
				t.Errorf("InferEncodingFromFontName(%q).Name() = %q, want %q", tt.fontName, enc.Name(), tt.expectedEnc)
			}
		})
	}
}

func TestDecodeWithEncoding(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		encodingName string
		expected     string
	}{
		{
			name:         "WinAnsi Hello",
			data:         []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F},
			encodingName: "WinAnsiEncoding",
			expected:     "Hello",
		},
		{
			name:         "WinAnsi with Euro",
			data:         []byte{0x80, 0x31, 0x30, 0x30}, // €100
			encodingName: "WinAnsiEncoding",
			expected:     "€100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeWithEncoding(tt.data, tt.encodingName)
			if result != tt.expected {
				t.Errorf("DecodeWithEncoding() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCustomEncoding(t *testing.T) {
	// Create a custom encoding with some differences
	differences := map[byte]rune{
		0x41: '★', // Override 'A' with star
		0x42: '♠', // Override 'B' with spade
	}
	custom := NewCustomEncoding(WinAnsiEncoding, differences)

	t.Run("Name", func(t *testing.T) {
		if !contains(custom.Name(), "custom") {
			t.Errorf("Name() = %q, want to contain 'custom'", custom.Name())
		}
	})

	t.Run("Decode with difference", func(t *testing.T) {
		if r := custom.Decode(0x41); r != '★' {
			t.Errorf("Decode(0x41) = %q, want '★'", r)
		}
	})

	t.Run("Decode without difference", func(t *testing.T) {
		if r := custom.Decode(0x43); r != 'C' {
			t.Errorf("Decode(0x43) = %q, want 'C'", r)
		}
	})

	t.Run("DecodeString", func(t *testing.T) {
		result := custom.DecodeString([]byte{0x41, 0x42, 0x43}) // Should be ★♠C
		if result != "★♠C" {
			t.Errorf("DecodeString() = %q, want '★♠C'", result)
		}
	})
}

func TestCustomEncodingFromGlyphs(t *testing.T) {
	// Create a custom encoding using glyph names
	differences := map[byte]string{
		0x41: "bullet",    // Map 'A' position to bullet
		0x42: "Euro",      // Map 'B' position to Euro
		0x43: "copyright", // Map 'C' position to copyright
	}
	custom := NewCustomEncodingFromGlyphs(WinAnsiEncoding, differences)

	t.Run("Decode bullet", func(t *testing.T) {
		if r := custom.Decode(0x41); r != '•' {
			t.Errorf("Decode(0x41) = %q, want '•'", r)
		}
	})

	t.Run("Decode Euro", func(t *testing.T) {
		if r := custom.Decode(0x42); r != '€' {
			t.Errorf("Decode(0x42) = %q, want '€'", r)
		}
	})

	t.Run("Decode copyright", func(t *testing.T) {
		if r := custom.Decode(0x43); r != '©' {
			t.Errorf("Decode(0x43) = %q, want '©'", r)
		}
	})

	t.Run("Unknown glyph falls through", func(t *testing.T) {
		// Glyph name not in table - should not create a difference
		diffs := map[byte]string{
			0x44: "unknownGlyphName",
		}
		enc := NewCustomEncodingFromGlyphs(WinAnsiEncoding, diffs)
		// Should fall back to base encoding's 'D'
		if r := enc.Decode(0x44); r != 'D' {
			t.Errorf("Decode(0x44) = %q, want 'D'", r)
		}
	})
}

func TestIsValidUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid ASCII", "Hello World", true},
		{"valid UTF-8", "Héllo Wörld", true},
		{"valid emoji", "Hello 👋", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidUTF8(tt.input); got != tt.expected {
				t.Errorf("IsValidUTF8(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeEmojiSequence(t *testing.T) {
	// Currently a passthrough, but test to ensure it doesn't break
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello 👋", "Hello 👋"},
		{"👨‍👩‍👧‍👦", "👨‍👩‍👧‍👦"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeEmojiSequence(tt.input); got != tt.expected {
				t.Errorf("NormalizeEmojiSequence(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDecodeUTF16BEEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Odd length input gets padded",
			input:    []byte{0x00, 0x48, 0x00}, // "H" + trailing byte (padded to 0x0000)
			expected: "H\x00",                  // Null char is added due to padding
		},
		{
			name:     "Orphan high surrogate",
			input:    []byte{0xD8, 0x00}, // High surrogate alone
			expected: "",                 // Should be skipped
		},
		{
			name:     "Orphan low surrogate",
			input:    []byte{0xDC, 0x00}, // Low surrogate alone
			expected: "",                 // Should be skipped
		},
		{
			name:     "High surrogate at end",
			input:    []byte{0x00, 0x48, 0xD8, 0x00}, // "H" + high surrogate at end
			expected: "H",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeUTF16BE(tt.input)
			if result != tt.expected {
				t.Errorf("DecodeUTF16BE() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDecodeUTF16LEEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Odd length input",
			input:    []byte{0x48, 0x00, 0x65}, // "H" + trailing byte
			expected: "He",
		},
		{
			name:     "Orphan high surrogate",
			input:    []byte{0x00, 0xD8}, // High surrogate alone (LE order)
			expected: "",                 // Should be skipped
		},
		{
			name:     "Orphan low surrogate",
			input:    []byte{0x00, 0xDC}, // Low surrogate alone (LE order)
			expected: "",                 // Should be skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeUTF16LE(tt.input)
			if result != tt.expected {
				t.Errorf("DecodeUTF16LE() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
