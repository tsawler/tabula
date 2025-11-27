package font

import (
	"testing"

	"github.com/tsawler/tabula/core"
)

func TestNewType0Font_BasicFont(t *testing.T) {
	// Create a minimal CIDFont dictionary
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("HeiseiMin-W3"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
	}

	// Create a Type0 font dictionary
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("Type0"),
		"BaseFont": core.Name("HeiseiMin-W3"),
		"Encoding": core.Name("Identity-H"),
		"DescendantFonts": core.Array{
			cidFontDict,
		},
	}

	font, err := NewType0Font(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewType0Font failed: %v", err)
	}

	if font.BaseFont != "HeiseiMin-W3" {
		t.Errorf("Expected BaseFont 'HeiseiMin-W3', got '%s'", font.BaseFont)
	}

	if font.Subtype != "Type0" {
		t.Errorf("Expected Subtype 'Type0', got '%s'", font.Subtype)
	}

	if font.Encoding != "Identity-H" {
		t.Errorf("Expected encoding 'Identity-H', got '%s'", font.Encoding)
	}

	if font.IsVertical {
		t.Error("Identity-H should not be vertical")
	}
}

func TestNewType0Font_VerticalWriting(t *testing.T) {
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("HeiseiMin-W3"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
	}

	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("Type0"),
		"BaseFont": core.Name("HeiseiMin-W3"),
		"Encoding": core.Name("Identity-V"), // Vertical writing
		"DescendantFonts": core.Array{
			cidFontDict,
		},
	}

	font, err := NewType0Font(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewType0Font failed: %v", err)
	}

	if font.Encoding != "Identity-V" {
		t.Errorf("Expected encoding 'Identity-V', got '%s'", font.Encoding)
	}

	if !font.IsVertical {
		t.Error("Identity-V should be vertical")
	}
}

func TestNewType0Font_NotType0(t *testing.T) {
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("Type1"), // Wrong subtype
		"BaseFont": core.Name("Helvetica"),
	}

	_, err := NewType0Font(fontDict, mockResolver)
	if err == nil {
		t.Error("Expected error for non-Type0 font, got nil")
	}
}

func TestNewCIDFont_Japanese(t *testing.T) {
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("HeiseiMin-W3"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
		"DW": core.Int(1000),
	}

	cidFont, err := NewCIDFont(cidFontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewCIDFont failed: %v", err)
	}

	if cidFont.BaseFont != "HeiseiMin-W3" {
		t.Errorf("Expected BaseFont 'HeiseiMin-W3', got '%s'", cidFont.BaseFont)
	}

	if cidFont.Subtype != "CIDFontType0" {
		t.Errorf("Expected Subtype 'CIDFontType0', got '%s'", cidFont.Subtype)
	}

	if cidFont.DW != 1000.0 {
		t.Errorf("Expected DW 1000.0, got %f", cidFont.DW)
	}

	if !cidFont.IsJapanese() {
		t.Error("Font should be identified as Japanese")
	}

	if cidFont.IsChinese() || cidFont.IsKorean() {
		t.Error("Font should not be identified as Chinese or Korean")
	}

	if !cidFont.IsCJK() {
		t.Error("Font should be identified as CJK")
	}
}

func TestNewCIDFont_Chinese(t *testing.T) {
	tests := []struct {
		name     string
		ordering string
	}{
		{"Simplified Chinese", "GB1"},
		{"Traditional Chinese", "CNS1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cidFontDict := core.Dict{
				"Type":     core.Name("Font"),
				"Subtype":  core.Name("CIDFontType0"),
				"BaseFont": core.Name("STSong-Light"),
				"CIDSystemInfo": core.Dict{
					"Registry":   core.String("Adobe"),
					"Ordering":   core.String(tt.ordering),
					"Supplement": core.Int(2),
				},
			}

			cidFont, err := NewCIDFont(cidFontDict, mockResolver)
			if err != nil {
				t.Fatalf("NewCIDFont failed: %v", err)
			}

			if !cidFont.IsChinese() {
				t.Error("Font should be identified as Chinese")
			}

			if cidFont.IsJapanese() || cidFont.IsKorean() {
				t.Error("Font should not be identified as Japanese or Korean")
			}

			if !cidFont.IsCJK() {
				t.Error("Font should be identified as CJK")
			}
		})
	}
}

func TestNewCIDFont_Korean(t *testing.T) {
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("HYSMyeongJo-Medium"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Korea1"),
			"Supplement": core.Int(1),
		},
	}

	cidFont, err := NewCIDFont(cidFontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewCIDFont failed: %v", err)
	}

	if !cidFont.IsKorean() {
		t.Error("Font should be identified as Korean")
	}

	if cidFont.IsJapanese() || cidFont.IsChinese() {
		t.Error("Font should not be identified as Japanese or Chinese")
	}

	if !cidFont.IsCJK() {
		t.Error("Font should be identified as CJK")
	}
}

func TestCIDFont_CIDSystemInfo(t *testing.T) {
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("TestFont"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(6),
		},
	}

	cidFont, err := NewCIDFont(cidFontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewCIDFont failed: %v", err)
	}

	if cidFont.CIDSystemInfo == nil {
		t.Fatal("CIDSystemInfo should not be nil")
	}

	if cidFont.CIDSystemInfo.Registry != "Adobe" {
		t.Errorf("Expected Registry 'Adobe', got '%s'", cidFont.CIDSystemInfo.Registry)
	}

	if cidFont.CIDSystemInfo.Ordering != "Japan1" {
		t.Errorf("Expected Ordering 'Japan1', got '%s'", cidFont.CIDSystemInfo.Ordering)
	}

	if cidFont.CIDSystemInfo.Supplement != 6 {
		t.Errorf("Expected Supplement 6, got %d", cidFont.CIDSystemInfo.Supplement)
	}

	collection := cidFont.GetCharacterCollection()
	expected := "Adobe-Japan1-6"
	if collection != expected {
		t.Errorf("Expected character collection '%s', got '%s'", expected, collection)
	}
}

func TestCIDFont_WidthArray_RangeFormat(t *testing.T) {
	// W array format: cfirst clast w
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("TestFont"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
		"DW": core.Int(1000),
		"W": core.Array{
			core.Int(1),   // Start CID
			core.Int(10),  // End CID
			core.Int(500), // Width for all CIDs 1-10
		},
	}

	cidFont, err := NewCIDFont(cidFontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewCIDFont failed: %v", err)
	}

	// Test widths in range
	for cid := 1; cid <= 10; cid++ {
		width := cidFont.GetWidthForCID(cid)
		if width != 500.0 {
			t.Errorf("Expected width 500.0 for CID %d, got %f", cid, width)
		}
	}

	// Test width outside range (should use DW)
	width := cidFont.GetWidthForCID(20)
	if width != 1000.0 {
		t.Errorf("Expected default width 1000.0 for CID 20, got %f", width)
	}
}

func TestCIDFont_WidthArray_ArrayFormat(t *testing.T) {
	// W array format: c [w1 w2 ... wn]
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("TestFont"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
		"DW": core.Int(1000),
		"W": core.Array{
			core.Int(100), // Start CID
			core.Array{
				core.Int(250), // Width for CID 100
				core.Int(300), // Width for CID 101
				core.Int(350), // Width for CID 102
			},
		},
	}

	cidFont, err := NewCIDFont(cidFontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewCIDFont failed: %v", err)
	}

	// Test individual widths
	tests := []struct {
		cid      int
		expected float64
	}{
		{100, 250.0},
		{101, 300.0},
		{102, 350.0},
		{103, 1000.0}, // Outside range, use DW
	}

	for _, tt := range tests {
		width := cidFont.GetWidthForCID(tt.cid)
		if width != tt.expected {
			t.Errorf("Expected width %f for CID %d, got %f", tt.expected, tt.cid, width)
		}
	}
}

func TestCIDFont_WidthArray_Mixed(t *testing.T) {
	// W array with both formats
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("TestFont"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
		"DW": core.Int(1000),
		"W": core.Array{
			// Range format
			core.Int(1),
			core.Int(5),
			core.Int(500),
			// Array format
			core.Int(100),
			core.Array{
				core.Int(250),
				core.Int(300),
			},
			// Another range
			core.Int(200),
			core.Int(210),
			core.Int(600),
		},
	}

	cidFont, err := NewCIDFont(cidFontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewCIDFont failed: %v", err)
	}

	tests := []struct {
		cid      int
		expected float64
	}{
		{1, 500.0},    // First range
		{3, 500.0},    // First range
		{5, 500.0},    // First range
		{100, 250.0},  // Array format
		{101, 300.0},  // Array format
		{200, 600.0},  // Second range
		{205, 600.0},  // Second range
		{210, 600.0},  // Second range
		{50, 1000.0},  // Default
		{999, 1000.0}, // Default
	}

	for _, tt := range tests {
		width := cidFont.GetWidthForCID(tt.cid)
		if width != tt.expected {
			t.Errorf("Expected width %f for CID %d, got %f", tt.expected, tt.cid, width)
		}
	}
}

func TestCIDFont_DefaultWidth(t *testing.T) {
	// Font without W array should use DW
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("TestFont"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
		"DW": core.Int(850),
	}

	cidFont, err := NewCIDFont(cidFontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewCIDFont failed: %v", err)
	}

	// All CIDs should use default width
	for cid := 0; cid < 100; cid++ {
		width := cidFont.GetWidthForCID(cid)
		if width != 850.0 {
			t.Errorf("Expected default width 850.0 for CID %d, got %f", cid, width)
		}
	}
}

func TestCIDFont_NoDW(t *testing.T) {
	// Font without DW should use 1000 as default
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("TestFont"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
		// No DW specified
	}

	cidFont, err := NewCIDFont(cidFontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewCIDFont failed: %v", err)
	}

	if cidFont.DW != 1000.0 {
		t.Errorf("Expected default DW 1000.0, got %f", cidFont.DW)
	}
}

func TestCIDFont_CIDFontType2(t *testing.T) {
	// Test CIDFontType2 (TrueType-based CIDFont)
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType2"), // TrueType-based
		"BaseFont": core.Name("TestFont"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
	}

	cidFont, err := NewCIDFont(cidFontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewCIDFont failed: %v", err)
	}

	if cidFont.Subtype != "CIDFontType2" {
		t.Errorf("Expected Subtype 'CIDFontType2', got '%s'", cidFont.Subtype)
	}
}

func TestCIDFont_NotCIDFont(t *testing.T) {
	// Try to create CIDFont from wrong type
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("Type1"), // Wrong type
		"BaseFont": core.Name("Helvetica"),
	}

	_, err := NewCIDFont(fontDict, mockResolver)
	if err == nil {
		t.Error("Expected error for non-CIDFont, got nil")
	}
}

func TestType0Font_MissingDescendantFonts(t *testing.T) {
	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("Type0"),
		"BaseFont": core.Name("TestFont"),
		"Encoding": core.Name("Identity-H"),
		// Missing DescendantFonts
	}

	_, err := NewType0Font(fontDict, mockResolver)
	if err == nil {
		t.Error("Expected error for missing DescendantFonts, got nil")
	}
}

func TestType0Font_EmptyDescendantFonts(t *testing.T) {
	fontDict := core.Dict{
		"Type":            core.Name("Font"),
		"Subtype":         core.Name("Type0"),
		"BaseFont":        core.Name("TestFont"),
		"Encoding":        core.Name("Identity-H"),
		"DescendantFonts": core.Array{}, // Empty array
	}

	_, err := NewType0Font(fontDict, mockResolver)
	if err == nil {
		t.Error("Expected error for empty DescendantFonts, got nil")
	}
}

func TestCIDFont_MissingCIDSystemInfo(t *testing.T) {
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("TestFont"),
		// Missing CIDSystemInfo
	}

	_, err := NewCIDFont(cidFontDict, mockResolver)
	if err == nil {
		t.Error("Expected error for missing CIDSystemInfo, got nil")
	}
}

func TestType0Font_ToUnicode(t *testing.T) {
	toUnicodeStream := &core.Stream{
		Dict: core.Dict{"Length": core.Int(100)},
		Data: []byte("dummy cmap data"),
	}

	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("HeiseiMin-W3"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
	}

	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("Type0"),
		"BaseFont": core.Name("HeiseiMin-W3"),
		"Encoding": core.Name("Identity-H"),
		"DescendantFonts": core.Array{
			cidFontDict,
		},
		"ToUnicode": toUnicodeStream,
	}

	font, err := NewType0Font(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewType0Font failed: %v", err)
	}

	if font.ToUnicode == nil {
		t.Error("ToUnicode stream should be captured")
	}

	if font.ToUnicode != toUnicodeStream {
		t.Error("ToUnicode stream should match provided stream")
	}
}

func TestType0Font_GetWidth(t *testing.T) {
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("TestFont"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
		"DW": core.Int(1000),
		"W": core.Array{
			core.Int(100),
			core.Int(105),
			core.Int(500),
		},
	}

	fontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("Type0"),
		"BaseFont": core.Name("TestFont"),
		"Encoding": core.Name("Identity-H"),
		"DescendantFonts": core.Array{
			cidFontDict,
		},
	}

	font, err := NewType0Font(fontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewType0Font failed: %v", err)
	}

	// Test GetWidth (treats rune as CID)
	width := font.GetWidth(rune(102))
	if width != 500.0 {
		t.Errorf("Expected width 500.0 for CID 102, got %f", width)
	}

	width = font.GetWidth(rune(200))
	if width != 1000.0 {
		t.Errorf("Expected default width 1000.0 for CID 200, got %f", width)
	}
}

func TestCIDFont_VerticalMetrics_DW2(t *testing.T) {
	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("TestFont"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
		"DW2": core.Array{
			core.Int(880),   // w1y
			core.Int(-1000), // w1
		},
	}

	cidFont, err := NewCIDFont(cidFontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewCIDFont failed: %v", err)
	}

	if cidFont.DW2[0] != 880.0 {
		t.Errorf("Expected DW2[0] 880.0, got %f", cidFont.DW2[0])
	}

	if cidFont.DW2[1] != -1000.0 {
		t.Errorf("Expected DW2[1] -1000.0, got %f", cidFont.DW2[1])
	}
}

func TestExtractString(t *testing.T) {
	tests := []struct {
		name     string
		input    core.Object
		expected string
	}{
		{"String", core.String("TestString"), "TestString"},
		{"Name", core.Name("TestName"), "TestName"},
		{"Nil", nil, ""},
		{"Int", core.Int(123), ""}, // Should return empty for non-string types
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCIDFont_FontDescriptor(t *testing.T) {
	descriptorDict := core.Dict{
		"Type":        core.Name("FontDescriptor"),
		"FontName":    core.Name("HeiseiMin-W3"),
		"Flags":       core.Int(4),
		"FontBBox":    core.Array{core.Int(-123), core.Int(-257), core.Int(1001), core.Int(910)},
		"ItalicAngle": core.Real(0),
		"Ascent":      core.Int(859),
		"Descent":     core.Int(-140),
		"CapHeight":   core.Int(709),
		"StemV":       core.Int(69),
	}

	cidFontDict := core.Dict{
		"Type":     core.Name("Font"),
		"Subtype":  core.Name("CIDFontType0"),
		"BaseFont": core.Name("HeiseiMin-W3"),
		"CIDSystemInfo": core.Dict{
			"Registry":   core.String("Adobe"),
			"Ordering":   core.String("Japan1"),
			"Supplement": core.Int(2),
		},
		"FontDescriptor": descriptorDict,
	}

	cidFont, err := NewCIDFont(cidFontDict, mockResolver)
	if err != nil {
		t.Fatalf("NewCIDFont failed: %v", err)
	}

	if cidFont.FontDescriptor == nil {
		t.Fatal("Font descriptor should be parsed")
	}

	fd := cidFont.FontDescriptor

	if fd.FontName != "HeiseiMin-W3" {
		t.Errorf("Expected FontName 'HeiseiMin-W3', got '%s'", fd.FontName)
	}

	if fd.Ascent != 859 {
		t.Errorf("Expected Ascent 859, got %f", fd.Ascent)
	}

	if fd.Descent != -140 {
		t.Errorf("Expected Descent -140, got %f", fd.Descent)
	}
}

func TestCIDFont_CommonCollections(t *testing.T) {
	tests := []struct {
		name       string
		ordering   string
		isJapanese bool
		isChinese  bool
		isKorean   bool
	}{
		{"Adobe-Japan1", "Japan1", true, false, false},
		{"Adobe-GB1 (Simplified Chinese)", "GB1", false, true, false},
		{"Adobe-CNS1 (Traditional Chinese)", "CNS1", false, true, false},
		{"Adobe-Korea1", "Korea1", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cidFontDict := core.Dict{
				"Type":     core.Name("Font"),
				"Subtype":  core.Name("CIDFontType0"),
				"BaseFont": core.Name("TestFont"),
				"CIDSystemInfo": core.Dict{
					"Registry":   core.String("Adobe"),
					"Ordering":   core.String(tt.ordering),
					"Supplement": core.Int(0),
				},
			}

			cidFont, err := NewCIDFont(cidFontDict, mockResolver)
			if err != nil {
				t.Fatalf("NewCIDFont failed: %v", err)
			}

			if cidFont.IsJapanese() != tt.isJapanese {
				t.Errorf("IsJapanese() = %v, want %v", cidFont.IsJapanese(), tt.isJapanese)
			}

			if cidFont.IsChinese() != tt.isChinese {
				t.Errorf("IsChinese() = %v, want %v", cidFont.IsChinese(), tt.isChinese)
			}

			if cidFont.IsKorean() != tt.isKorean {
				t.Errorf("IsKorean() = %v, want %v", cidFont.IsKorean(), tt.isKorean)
			}

			if !cidFont.IsCJK() {
				t.Error("Should be identified as CJK")
			}
		})
	}
}
