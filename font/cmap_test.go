package font

import (
	"testing"

	"github.com/tsawler/tabula/core"
)

func TestCMapParseBfChar(t *testing.T) {
	// Sample ToUnicode CMap with beginbfchar/endbfchar
	cmapData := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
4 beginbfchar
<0003> <0020>
<0004> <0041>
<0005> <0042>
<0006> <0043>
endbfchar
endcmap
CMapName currentdict /CMap defineresource pop
end
end
`

	cmap, err := parseCMapData([]byte(cmapData))
	if err != nil {
		t.Fatalf("Failed to parse CMap: %v", err)
	}

	// Test lookups
	tests := []struct {
		code     uint32
		expected string
	}{
		{0x0003, " "}, // Space
		{0x0004, "A"}, // A
		{0x0005, "B"}, // B
		{0x0006, "C"}, // C
		{0x0007, ""},  // Not mapped, should return empty (caller handles fallback)
	}

	for _, tt := range tests {
		result := cmap.Lookup(tt.code)
		if result != tt.expected {
			t.Errorf("Lookup(%04x) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}

func TestCMapParseBfRange(t *testing.T) {
	// Sample ToUnicode CMap with beginbfrange/endbfrange
	cmapData := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
2 beginbfrange
<0020> <007E> <0020>
<00A0> <00A2> <00A0>
endbfrange
endcmap
CMapName currentdict /CMap defineresource pop
end
end
`

	cmap, err := parseCMapData([]byte(cmapData))
	if err != nil {
		t.Fatalf("Failed to parse CMap: %v", err)
	}

	// Test range lookups
	tests := []struct {
		code     uint32
		expected string
	}{
		{0x0020, " "},                  // Space (start of range)
		{0x0041, "A"},                  // A (middle of range)
		{0x007E, "~"},                  // ~ (end of range)
		{0x00A0, string(rune(0x00A0))}, // Non-breaking space
		{0x00A1, string(rune(0x00A1))}, // ¡
		{0x00A2, string(rune(0x00A2))}, // ¢
		{0x00A3, ""},                   // Not in range, returns empty
	}

	for _, tt := range tests {
		result := cmap.Lookup(tt.code)
		if result != tt.expected {
			t.Errorf("Lookup(%04x) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}

func TestCMapParseBfRangeArray(t *testing.T) {
	// Sample ToUnicode CMap with array format
	cmapData := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
1 beginbfrange
<0010> <0013> [<0041> <0042> <0043> <0044>]
endbfrange
endcmap
CMapName currentdict /CMap defineresource pop
end
end
`

	cmap, err := parseCMapData([]byte(cmapData))
	if err != nil {
		t.Fatalf("Failed to parse CMap: %v", err)
	}

	// Test array format lookups
	tests := []struct {
		code     uint32
		expected string
	}{
		{0x0010, "A"},
		{0x0011, "B"},
		{0x0012, "C"},
		{0x0013, "D"},
		{0x0014, ""}, // Not in range, returns empty
	}

	for _, tt := range tests {
		result := cmap.Lookup(tt.code)
		if result != tt.expected {
			t.Errorf("Lookup(%04x) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}

func TestCMapLookupString(t *testing.T) {
	// Create a simple CMap
	cmapData := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
5 beginbfchar
<0003> <0048>
<0004> <0065>
<0005> <006C>
<0006> <006F>
<0007> <0021>
endbfchar
endcmap
CMapName currentdict /CMap defineresource pop
end
end
`

	cmap, err := parseCMapData([]byte(cmapData))
	if err != nil {
		t.Fatalf("Failed to parse CMap: %v", err)
	}

	// Test string decoding
	// Input: character codes for "Hello!"
	input := []byte{0x00, 0x03, 0x00, 0x04, 0x00, 0x05, 0x00, 0x05, 0x00, 0x06, 0x00, 0x07}
	expected := "Hello!"

	result := cmap.LookupString(input)
	if result != expected {
		t.Errorf("LookupString() = %q, want %q", result, expected)
	}
}

func TestCMapMultiByte(t *testing.T) {
	// Test with 2-byte character codes (common for CJK)
	cmapData := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /Adobe-Japan1-UCS2 def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
3 beginbfchar
<0001> <3042>
<0002> <3044>
<0003> <3046>
endbfchar
endcmap
CMapName currentdict /CMap defineresource pop
end
end
`

	cmap, err := parseCMapData([]byte(cmapData))
	if err != nil {
		t.Fatalf("Failed to parse CMap: %v", err)
	}

	// Test hiragana characters
	tests := []struct {
		code     uint32
		expected string
	}{
		{0x0001, "あ"}, // Hiragana A
		{0x0002, "い"}, // Hiragana I
		{0x0003, "う"}, // Hiragana U
	}

	for _, tt := range tests {
		result := cmap.Lookup(tt.code)
		if result != tt.expected {
			t.Errorf("Lookup(%04x) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}

func TestParseToUnicodeCMap(t *testing.T) {
	cmapData := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
2 beginbfchar
<0003> <0041>
<0004> <0042>
endbfchar
endcmap
CMapName currentdict /CMap defineresource pop
end
end
`

	// Create a stream
	stream := &core.Stream{
		Dict: core.Dict{},
		Data: []byte(cmapData),
	}

	cmap, err := ParseToUnicodeCMap(stream)
	if err != nil {
		t.Fatalf("ParseToUnicodeCMap failed: %v", err)
	}

	// Test lookup
	result := cmap.Lookup(0x0003)
	if result != "A" {
		t.Errorf("Lookup(0x0003) = %q, want %q", result, "A")
	}
}

func TestCMapEmpty(t *testing.T) {
	// Test with empty CMap
	cmap := NewCMap()

	// Should return empty string for unmapped character
	result := cmap.Lookup(0x0041)
	expected := "" // Empty CMap returns empty (caller handles fallback)

	if result != expected {
		t.Errorf("Lookup(0x0041) = %q, want %q", result, expected)
	}

	// But LookupString should handle fallback
	input := []byte{0x41} // 'A'
	stringResult := cmap.LookupString(input)
	expectedString := "A" // Fallback to Unicode interpretation

	if stringResult != expectedString {
		t.Errorf("LookupString([0x41]) = %q, want %q", stringResult, expectedString)
	}
}

func TestCMapNil(t *testing.T) {
	// Test with nil CMap
	var cmap *CMap = nil

	// LookupString should handle nil gracefully
	input := []byte("Hello")
	result := cmap.LookupString(input)
	expected := "Hello"

	if result != expected {
		t.Errorf("LookupString with nil CMap = %q, want %q", result, expected)
	}
}

func TestExtractHexString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<ABCD>", "ABCD"},
		{"<1234>", "1234"},
		{"<>", ""},
		{"ABCD", ""},
		{"<ABCD", ""},
		{"ABCD>", ""},
	}

	for _, tt := range tests {
		result := extractHexString(tt.input)
		if result != tt.expected {
			t.Errorf("extractHexString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestParseHexToUint32(t *testing.T) {
	tests := []struct {
		input    string
		expected uint32
		wantErr  bool
	}{
		{"0041", 0x0041, false},
		{"FFFF", 0xFFFF, false},
		{"1234", 0x1234, false},
		{"41", 0x0041, false}, // Should pad odd length
		{"", 0, true},
		{"GGGG", 0, true},
	}

	for _, tt := range tests {
		result, err := parseHexToUint32(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseHexToUint32(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if result != tt.expected {
			t.Errorf("parseHexToUint32(%q) = %04x, want %04x", tt.input, result, tt.expected)
		}
	}
}

func TestHexToUnicode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"0041", "A", false},           // Single character UTF-16BE
		{"3042", "あ", false},           // Hiragana A
		{"004100420043", "ABC", false}, // Multiple characters
		{"FEFF0041", "A", false},       // With BOM
		{"D83DDE00", "😀", false},       // Emoji (surrogate pair)
		{"41", "A", false},             // Single byte (padded)
		{"", "", true},                 // Empty
	}

	for _, tt := range tests {
		result, err := hexToUnicode(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("hexToUnicode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && result != tt.expected {
			t.Errorf("hexToUnicode(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCMapComplexScenario(t *testing.T) {
	// Test with both bfchar and bfrange
	cmapData := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
3 beginbfchar
<0001> <0048>
<0002> <0065>
<0003> <006C>
endbfchar
2 beginbfrange
<0020> <007E> <0020>
<00A0> <00FF> <00A0>
endbfrange
endcmap
CMapName currentdict /CMap defineresource pop
end
end
`

	cmap, err := parseCMapData([]byte(cmapData))
	if err != nil {
		t.Fatalf("Failed to parse CMap: %v", err)
	}

	// Test that bfchar takes precedence
	if result := cmap.Lookup(0x0001); result != "H" {
		t.Errorf("Lookup(0x0001) = %q, want %q", result, "H")
	}

	// Test range
	if result := cmap.Lookup(0x0041); result != "A" {
		t.Errorf("Lookup(0x0041) = %q, want %q", result, "A")
	}

	// Test second range
	if result := cmap.Lookup(0x00A9); result != "©" {
		t.Errorf("Lookup(0x00A9) = %q, want %q", result, "©")
	}
}

func TestCMapWithEmojiSurrogatePair(t *testing.T) {
	cmapData := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo <<
  /Registry (Adobe)
  /Ordering (UCS)
  /Supplement 0
>> def
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<00><FF>
endcodespacerange
1 beginbfchar
<21><d83d dc4b>
endbfchar
endcmap
CMapName currentdict /CMap defineresource pop
end
end
`

	cmap, err := parseCMapData([]byte(cmapData))
	if err != nil {
		t.Fatalf("parseCMapData failed: %v", err)
	}

	// Test character code 0x21 should map to emoji U+1F44B (👋)
	result := cmap.Lookup(0x21)
	runes := []rune(result)

	if len(runes) == 0 {
		t.Fatalf("Expected emoji, got empty string")
	}

	if runes[0] != 0x1F44B {
		t.Errorf("Lookup(0x21) = U+%04X %q, want U+1F44B 👋", runes[0], result)
	} else {
		t.Logf("✅ Correctly parsed emoji: U+%04X %q", runes[0], result)
	}
}

func TestCMapTightPacking(t *testing.T) {
	// Test with tight packing in bfrange (no spaces between tokens)
	// This mimics the behavior seen in rnb.pdf
	cmapData := "/CIDInit /ProcSet findresource begin\n" +
		"12 dict begin\n" +
		"begincmap\n" +
		"/CMapName /Adobe-Identity-UCS def\n" +
		"/CMapType 2 def\n" +
		"1 begincodespacerange\n" +
		"<00><FF>\n" +
		"endcodespacerange\n" +
		"2 beginbfrange\n" +
		"<21><21><0052>\n" +
		"<22><22><0065>\n" +
		"endbfrange\n" +
		"endcmap\n" +
		"CMapName currentdict /CMap defineresource pop\n" +
		"end\n" +
		"end\n"

	cmap, err := parseCMapData([]byte(cmapData))
	if err != nil {
		t.Fatalf("Failed to parse CMap: %v", err)
	}

	// Test lookups
	tests := []struct {
		code     uint32
		expected string
	}{
		{0x21, "R"},
		{0x22, "e"},
	}

	for _, tt := range tests {
		result := cmap.Lookup(tt.code)
		if result != tt.expected {
			t.Errorf("Lookup(%02x) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}
