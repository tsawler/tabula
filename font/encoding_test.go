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
