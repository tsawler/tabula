package layout

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// makeHeadingFragment creates a text fragment for heading tests
func makeHeadingFragment(t string, x, y, w, h, fs float64, fontName string) text.TextFragment {
	return text.TextFragment{
		Text:     t,
		X:        x,
		Y:        y,
		Width:    w,
		Height:   h,
		FontSize: fs,
		FontName: fontName,
	}
}

func TestHeadingLevelString(t *testing.T) {
	tests := []struct {
		level    HeadingLevel
		expected string
	}{
		{HeadingLevelUnknown, "unknown"},
		{HeadingLevel1, "h1"},
		{HeadingLevel2, "h2"},
		{HeadingLevel3, "h3"},
		{HeadingLevel4, "h4"},
		{HeadingLevel5, "h5"},
		{HeadingLevel6, "h6"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("HeadingLevel(%d).String() = %q, want %q", tt.level, got, tt.expected)
		}
	}
}

func TestHeadingLevelHTMLTag(t *testing.T) {
	tests := []struct {
		level    HeadingLevel
		expected string
	}{
		{HeadingLevelUnknown, "p"},
		{HeadingLevel1, "h1"},
		{HeadingLevel2, "h2"},
		{HeadingLevel3, "h3"},
		{HeadingLevel4, "h4"},
		{HeadingLevel5, "h5"},
		{HeadingLevel6, "h6"},
	}

	for _, tt := range tests {
		if got := tt.level.HTMLTag(); got != tt.expected {
			t.Errorf("HeadingLevel(%d).HTMLTag() = %q, want %q", tt.level, got, tt.expected)
		}
	}
}

func TestNewHeadingDetector(t *testing.T) {
	detector := NewHeadingDetector()
	if detector == nil {
		t.Fatal("NewHeadingDetector returned nil")
	}
}

func TestNewHeadingDetectorWithConfig(t *testing.T) {
	config := HeadingConfig{
		MaxHeadingLines: 5,
		MinConfidence:   0.7,
	}
	detector := NewHeadingDetectorWithConfig(config)
	if detector == nil {
		t.Fatal("NewHeadingDetectorWithConfig returned nil")
	}
	if detector.config.MaxHeadingLines != 5 {
		t.Errorf("Expected MaxHeadingLines=5, got %d", detector.config.MaxHeadingLines)
	}
}

func TestDefaultHeadingConfig(t *testing.T) {
	config := DefaultHeadingConfig()

	if config.MaxHeadingLines != 3 {
		t.Errorf("Expected MaxHeadingLines=3, got %d", config.MaxHeadingLines)
	}
	if config.MinConfidence != 0.5 {
		t.Errorf("Expected MinConfidence=0.5, got %f", config.MinConfidence)
	}
	if len(config.FontSizeRatios) == 0 {
		t.Error("Expected FontSizeRatios to be populated")
	}
	if len(config.NumberedPatterns) == 0 {
		t.Error("Expected NumberedPatterns to be populated")
	}
}

func TestDetectFromParagraphs_Empty(t *testing.T) {
	detector := NewHeadingDetector()
	result := detector.DetectFromParagraphs([]Paragraph{}, 612, 792)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.HeadingCount() != 0 {
		t.Errorf("Expected 0 headings, got %d", result.HeadingCount())
	}
}

func TestDetectFromParagraphs_SingleHeading(t *testing.T) {
	// Create a heading paragraph with large font
	headingPara := Paragraph{
		Text:            "Main Title",
		Style:           StyleHeading,
		AverageFontSize: 24,
		Lines: []Line{
			{
				Text:            "Main Title",
				AverageFontSize: 24,
				BBox:            model.BBox{X: 100, Y: 700, Width: 200, Height: 24},
			},
		},
		BBox:      model.BBox{X: 100, Y: 700, Width: 200, Height: 24},
		Alignment: AlignCenter,
	}

	// Create multiple body paragraphs to ensure body font is correctly detected
	bodyPara1 := Paragraph{
		Text:            "This is body text that appears after the heading.",
		Style:           StyleNormal,
		AverageFontSize: 12,
		Lines: []Line{
			{Text: "This is body text that appears after the heading.", AverageFontSize: 12, BBox: model.BBox{X: 72, Y: 650, Width: 450, Height: 14}},
			{Text: "Second line of body text here.", AverageFontSize: 12, BBox: model.BBox{X: 72, Y: 634, Width: 300, Height: 14}},
		},
		BBox: model.BBox{X: 72, Y: 634, Width: 450, Height: 30},
	}

	bodyPara2 := Paragraph{
		Text:            "Another paragraph of body text.",
		Style:           StyleNormal,
		AverageFontSize: 12,
		Lines: []Line{
			{Text: "Another paragraph of body text.", AverageFontSize: 12, BBox: model.BBox{X: 72, Y: 600, Width: 350, Height: 14}},
		},
		BBox: model.BBox{X: 72, Y: 600, Width: 350, Height: 14},
	}

	detector := NewHeadingDetector()
	result := detector.DetectFromParagraphs([]Paragraph{headingPara, bodyPara1, bodyPara2}, 612, 792)

	if result.HeadingCount() == 0 {
		t.Fatal("Expected at least 1 heading")
	}

	heading := result.GetHeading(0)
	if heading == nil {
		t.Fatal("Expected non-nil heading")
	}
	if heading.Text != "Main Title" {
		t.Errorf("Expected heading text 'Main Title', got %q", heading.Text)
	}
}

func TestDetectFromParagraphs_MultipleHeadings(t *testing.T) {
	paragraphs := []Paragraph{
		{
			Text:            "Chapter One",
			Style:           StyleHeading,
			AverageFontSize: 24,
			Lines: []Line{
				{Text: "Chapter One", AverageFontSize: 24, BBox: model.BBox{X: 100, Y: 700, Width: 200, Height: 24}},
			},
			BBox: model.BBox{X: 100, Y: 700, Width: 200, Height: 24},
		},
		{
			Text:            "Introduction text here.",
			Style:           StyleNormal,
			AverageFontSize: 12,
			Lines: []Line{
				{Text: "Introduction text here.", AverageFontSize: 12, BBox: model.BBox{X: 72, Y: 650, Width: 400, Height: 14}},
			},
			BBox: model.BBox{X: 72, Y: 650, Width: 400, Height: 14},
		},
		{
			Text:            "Section 1.1",
			Style:           StyleHeading,
			AverageFontSize: 18,
			Lines: []Line{
				{Text: "Section 1.1", AverageFontSize: 18, BBox: model.BBox{X: 72, Y: 600, Width: 150, Height: 18}},
			},
			BBox: model.BBox{X: 72, Y: 600, Width: 150, Height: 18},
		},
		{
			Text:            "More body text.",
			Style:           StyleNormal,
			AverageFontSize: 12,
			Lines: []Line{
				{Text: "More body text.", AverageFontSize: 12, BBox: model.BBox{X: 72, Y: 550, Width: 400, Height: 14}},
			},
			BBox: model.BBox{X: 72, Y: 550, Width: 400, Height: 14},
		},
	}

	detector := NewHeadingDetector()
	result := detector.DetectFromParagraphs(paragraphs, 612, 792)

	if result.HeadingCount() < 2 {
		t.Errorf("Expected at least 2 headings, got %d", result.HeadingCount())
	}
}

func TestHeadingDetector_DetectFromLines(t *testing.T) {
	lines := []Line{
		{
			Text:            "Document Title",
			AverageFontSize: 24,
			BBox:            model.BBox{X: 150, Y: 700, Width: 300, Height: 24},
			Fragments: []text.TextFragment{
				makeHeadingFragment("Document Title", 150, 700, 300, 24, 24, "Arial-Bold"),
			},
		},
		{
			Text:            "Body text line one.",
			AverageFontSize: 12,
			BBox:            model.BBox{X: 72, Y: 650, Width: 400, Height: 14},
			Fragments: []text.TextFragment{
				makeHeadingFragment("Body text line one.", 72, 650, 400, 14, 12, "Arial"),
			},
		},
	}

	detector := NewHeadingDetector()
	result := detector.DetectFromLines(lines, 612, 792)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestHeadingDetector_DetectFromFragments(t *testing.T) {
	fragments := []text.TextFragment{
		makeHeadingFragment("Big Title", 150, 700, 300, 24, 24, "Arial-Bold"),
		makeHeadingFragment("Normal text content.", 72, 650, 400, 12, 12, "Arial"),
	}

	detector := NewHeadingDetector()
	result := detector.DetectFromFragments(fragments, 612, 792)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestDetectBold(t *testing.T) {
	detector := NewHeadingDetector()

	tests := []struct {
		name     string
		fontName string
		expected bool
	}{
		{"bold font", "Arial-Bold", true},
		{"bold in name", "Helvetica-BoldOblique", true},
		{"black font", "SourceSansPro-Black", true},
		{"heavy font", "OpenSans-Heavy", true},
		{"semibold font", "Roboto-SemiBold", true},
		{"demibold font", "Georgia-DemiBold", true},
		{"regular font", "Arial", false},
		{"italic font", "Arial-Italic", false},
		{"light font", "Helvetica-Light", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			para := Paragraph{
				Lines: []Line{
					{
						Fragments: []text.TextFragment{
							{FontName: tt.fontName},
						},
					},
				},
			}
			if got := detector.detectBold(para); got != tt.expected {
				t.Errorf("detectBold() with font %q = %v, want %v", tt.fontName, got, tt.expected)
			}
		})
	}
}

func TestDetectItalic(t *testing.T) {
	detector := NewHeadingDetector()

	tests := []struct {
		name     string
		fontName string
		expected bool
	}{
		{"italic font", "Arial-Italic", true},
		{"oblique font", "Helvetica-Oblique", true},
		{"bold italic", "Times-BoldItalic", true},
		{"regular font", "Arial", false},
		{"bold font", "Arial-Bold", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			para := Paragraph{
				Lines: []Line{
					{
						Fragments: []text.TextFragment{
							{FontName: tt.fontName},
						},
					},
				},
			}
			if got := detector.detectItalic(para); got != tt.expected {
				t.Errorf("detectItalic() with font %q = %v, want %v", tt.fontName, got, tt.expected)
			}
		})
	}
}

func TestDetectAllCaps(t *testing.T) {
	detector := NewHeadingDetector()

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"all caps", "INTRODUCTION", true},
		{"all caps with spaces", "CHAPTER ONE", true},
		{"mixed case", "Introduction", false},
		{"lowercase", "introduction", false},
		{"mostly caps", "INTRODUCTION text", false},
		{"caps with numbers", "CHAPTER 1", true},
		{"too short", "AB", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detector.detectAllCaps(tt.text); got != tt.expected {
				t.Errorf("detectAllCaps(%q) = %v, want %v", tt.text, got, tt.expected)
			}
		})
	}
}

func TestDetectNumbered(t *testing.T) {
	detector := NewHeadingDetector()

	tests := []struct {
		name           string
		text           string
		expectedNum    bool
		expectedPrefix string
	}{
		{"chapter", "Chapter 1 Introduction", true, "Chapter 1"},
		{"section", "Section 2 Methods", true, "Section 2"},
		{"simple number", "1. Introduction", true, "1."},
		{"two level", "1.2 Background", true, "1.2"},
		{"three level", "1.2.3 Details", true, "1.2.3"},
		{"roman numeral", "IV. Results", true, "IV."},
		{"letter prefix", "A. First Item", true, "A."},
		{"no number", "Introduction", false, ""},
		{"number in text", "There are 5 items", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isNum, prefix := detector.detectNumbered(tt.text)
			if isNum != tt.expectedNum {
				t.Errorf("detectNumbered(%q) isNumbered = %v, want %v", tt.text, isNum, tt.expectedNum)
			}
			if tt.expectedNum && prefix != tt.expectedPrefix {
				t.Errorf("detectNumbered(%q) prefix = %q, want %q", tt.text, prefix, tt.expectedPrefix)
			}
		})
	}
}

func TestDetermineLevel(t *testing.T) {
	detector := NewHeadingDetector()
	bodyFontSize := 12.0

	tests := []struct {
		name     string
		fontSize float64
		expected HeadingLevel
	}{
		{"very large (H1)", 24.0, HeadingLevel1}, // 2.0x body
		{"large (H1)", 22.0, HeadingLevel1},      // 1.83x body
		{"medium-large (H2)", 18.0, HeadingLevel2},
		{"medium (H3)", 16.0, HeadingLevel3},
		{"small-medium (H4)", 14.0, HeadingLevel4},
		{"small (H5)", 13.5, HeadingLevel5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			heading := Heading{FontSize: tt.fontSize}
			level := detector.determineLevel(tt.fontSize, bodyFontSize, heading)
			if level != tt.expected {
				t.Errorf("determineLevel(%.1f, %.1f) = %v, want %v", tt.fontSize, bodyFontSize, level, tt.expected)
			}
		})
	}
}

func TestHeadingLayout_GetHeadingsAtLevel(t *testing.T) {
	layout := &HeadingLayout{
		Headings: []Heading{
			{Text: "Title", Level: HeadingLevel1},
			{Text: "Section 1", Level: HeadingLevel2},
			{Text: "Section 2", Level: HeadingLevel2},
			{Text: "Subsection", Level: HeadingLevel3},
		},
	}

	h2s := layout.GetHeadingsAtLevel(HeadingLevel2)
	if len(h2s) != 2 {
		t.Errorf("Expected 2 H2 headings, got %d", len(h2s))
	}

	h1s := layout.GetH1()
	if len(h1s) != 1 {
		t.Errorf("Expected 1 H1 heading, got %d", len(h1s))
	}
}

func TestHeadingLayout_GetHeadingsInRange(t *testing.T) {
	layout := &HeadingLayout{
		Headings: []Heading{
			{Text: "Title", Level: HeadingLevel1},
			{Text: "Section 1", Level: HeadingLevel2},
			{Text: "Subsection", Level: HeadingLevel3},
			{Text: "Minor", Level: HeadingLevel4},
		},
	}

	h2h3 := layout.GetHeadingsInRange(HeadingLevel2, HeadingLevel3)
	if len(h2h3) != 2 {
		t.Errorf("Expected 2 headings in H2-H3 range, got %d", len(h2h3))
	}
}

func TestHeadingLayout_GetOutline(t *testing.T) {
	layout := &HeadingLayout{
		Headings: []Heading{
			{Text: "Chapter 1", Level: HeadingLevel1},
			{Text: "Section 1.1", Level: HeadingLevel2},
			{Text: "Section 1.2", Level: HeadingLevel2},
			{Text: "Subsection 1.2.1", Level: HeadingLevel3},
			{Text: "Chapter 2", Level: HeadingLevel1},
		},
	}

	outline := layout.GetOutline()
	if len(outline) != 2 {
		t.Errorf("Expected 2 top-level entries, got %d", len(outline))
	}

	if outline[0].Heading.Text != "Chapter 1" {
		t.Errorf("Expected first entry to be 'Chapter 1'")
	}

	if len(outline[0].Children) != 2 {
		t.Errorf("Expected 2 children for Chapter 1, got %d", len(outline[0].Children))
	}
}

func TestHeadingLayout_GetTableOfContents(t *testing.T) {
	layout := &HeadingLayout{
		Headings: []Heading{
			{Text: "Introduction", Level: HeadingLevel1},
			{Text: "Background", Level: HeadingLevel2},
			{Text: "Methods", Level: HeadingLevel2},
		},
	}

	toc := layout.GetTableOfContents()
	if !strings.Contains(toc, "Introduction") {
		t.Error("TOC should contain 'Introduction'")
	}
	if !strings.Contains(toc, "  Background") {
		t.Error("TOC should contain indented 'Background'")
	}
}

func TestHeadingLayout_GetMarkdownTOC(t *testing.T) {
	layout := &HeadingLayout{
		Headings: []Heading{
			{Text: "Introduction", Level: HeadingLevel1},
			{Text: "Background", Level: HeadingLevel2},
		},
	}

	toc := layout.GetMarkdownTOC()
	if !strings.Contains(toc, "- Introduction") {
		t.Error("Markdown TOC should contain '- Introduction'")
	}
	if !strings.Contains(toc, "  - Background") {
		t.Error("Markdown TOC should contain indented '  - Background'")
	}
}

func TestHeadingLayout_NilSafety(t *testing.T) {
	var layout *HeadingLayout

	if layout.HeadingCount() != 0 {
		t.Error("HeadingCount on nil should return 0")
	}
	if layout.GetHeading(0) != nil {
		t.Error("GetHeading on nil should return nil")
	}
	if layout.GetHeadingsAtLevel(HeadingLevel1) != nil {
		t.Error("GetHeadingsAtLevel on nil should return nil")
	}
	if layout.GetOutline() != nil {
		t.Error("GetOutline on nil should return nil")
	}
	if layout.GetTableOfContents() != "" {
		t.Error("GetTableOfContents on nil should return empty string")
	}
	if layout.FindHeadingBefore(100) != nil {
		t.Error("FindHeadingBefore on nil should return nil")
	}
}

func TestHeading_IsTopLevel(t *testing.T) {
	tests := []struct {
		level    HeadingLevel
		expected bool
	}{
		{HeadingLevel1, true},
		{HeadingLevel2, false},
		{HeadingLevel3, false},
		{HeadingLevelUnknown, false},
	}

	for _, tt := range tests {
		heading := &Heading{Level: tt.level}
		if got := heading.IsTopLevel(); got != tt.expected {
			t.Errorf("Heading{Level: %v}.IsTopLevel() = %v, want %v", tt.level, got, tt.expected)
		}
	}
}

func TestHeading_GetCleanText(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		isNumbered   bool
		numberPrefix string
		expected     string
	}{
		{"no number", "Introduction", false, "", "Introduction"},
		{"with number", "1. Introduction", true, "1.", "Introduction"},
		{"chapter", "Chapter 1 Methods", true, "Chapter 1", "Methods"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Heading{
				Text:         tt.text,
				IsNumbered:   tt.isNumbered,
				NumberPrefix: tt.numberPrefix,
			}
			if got := h.GetCleanText(); got != tt.expected {
				t.Errorf("GetCleanText() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestHeading_GetAnchorID(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"simple", "Introduction", "introduction"},
		{"with spaces", "Getting Started", "getting-started"},
		{"with special chars", "What's New?", "whats-new"},
		{"multiple spaces", "Section  One", "section-one"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Heading{Text: tt.text}
			if got := h.GetAnchorID(); got != tt.expected {
				t.Errorf("GetAnchorID() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestHeading_ToMarkdown(t *testing.T) {
	tests := []struct {
		level    HeadingLevel
		text     string
		expected string
	}{
		{HeadingLevel1, "Title", "# Title"},
		{HeadingLevel2, "Section", "## Section"},
		{HeadingLevel3, "Subsection", "### Subsection"},
	}

	for _, tt := range tests {
		h := &Heading{Level: tt.level, Text: tt.text}
		if got := h.ToMarkdown(); got != tt.expected {
			t.Errorf("ToMarkdown() = %q, want %q", got, tt.expected)
		}
	}
}

func TestHeading_WordCount(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"Introduction", 1},
		{"Getting Started Guide", 3},
		{"", 0},
	}

	for _, tt := range tests {
		h := &Heading{Text: tt.text}
		if got := h.WordCount(); got != tt.expected {
			t.Errorf("WordCount(%q) = %d, want %d", tt.text, got, tt.expected)
		}
	}
}

func TestHeading_ContainsPoint(t *testing.T) {
	h := &Heading{
		BBox: model.BBox{X: 100, Y: 200, Width: 300, Height: 24},
	}

	tests := []struct {
		name     string
		x, y     float64
		expected bool
	}{
		{"inside", 200, 210, true},
		{"on edge", 100, 200, true},
		{"outside left", 50, 210, false},
		{"outside right", 450, 210, false},
		{"outside top", 200, 250, false},
		{"outside bottom", 200, 150, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := h.ContainsPoint(tt.x, tt.y); got != tt.expected {
				t.Errorf("ContainsPoint(%.1f, %.1f) = %v, want %v", tt.x, tt.y, got, tt.expected)
			}
		})
	}
}

func TestHeading_NilSafety(t *testing.T) {
	var h *Heading

	if h.IsTopLevel() {
		t.Error("IsTopLevel on nil should return false")
	}
	if h.WordCount() != 0 {
		t.Error("WordCount on nil should return 0")
	}
	if h.GetCleanText() != "" {
		t.Error("GetCleanText on nil should return empty string")
	}
	if h.GetAnchorID() != "" {
		t.Error("GetAnchorID on nil should return empty string")
	}
	if h.ToMarkdown() != "" {
		t.Error("ToMarkdown on nil should return empty string")
	}
	if h.ContainsPoint(0, 0) {
		t.Error("ContainsPoint on nil should return false")
	}
}

func TestCalculateConfidence(t *testing.T) {
	detector := NewHeadingDetector()
	bodyFontSize := 12.0

	// Large font, bold, centered, short = high confidence
	highConfPara := Paragraph{
		AverageFontSize: 24,
		Text:            "Title",
		Lines:           []Line{{Text: "Title", Fragments: []text.TextFragment{{FontName: "Arial-Bold"}}}},
		Alignment:       AlignCenter,
	}
	highConfHeading := Heading{IsBold: true, IsAllCaps: false, IsNumbered: false}
	highConf := detector.calculateConfidence(highConfPara, bodyFontSize, highConfHeading)
	if highConf < 0.7 {
		t.Errorf("Expected high confidence (>=0.7), got %.2f", highConf)
	}

	// Normal font, long text = low confidence
	lowConfPara := Paragraph{
		AverageFontSize: 12,
		Text:            "This is a long paragraph that contains many words and is definitely not a heading because it has way too much text.",
		Lines: []Line{
			{Text: "Line 1"},
			{Text: "Line 2"},
			{Text: "Line 3"},
			{Text: "Line 4"},
		},
	}
	lowConfHeading := Heading{IsBold: false, IsAllCaps: false, IsNumbered: false}
	lowConf := detector.calculateConfidence(lowConfPara, bodyFontSize, lowConfHeading)
	if lowConf >= 0.5 {
		t.Errorf("Expected low confidence (<0.5), got %.2f", lowConf)
	}
}

func TestRefineHeadingLevels(t *testing.T) {
	detector := NewHeadingDetector()

	headings := []Heading{
		{Text: "Main Title", FontSize: 24},
		{Text: "Section", FontSize: 18},
		{Text: "Subsection", FontSize: 14},
		{Text: "Another Section", FontSize: 18},
	}

	detector.refineHeadingLevels(headings, 12.0)

	// After refinement, headings with same font size should have same level
	if headings[1].Level != headings[3].Level {
		t.Error("Headings with same font size should have same level after refinement")
	}

	// Larger fonts should have lower level numbers (H1 < H2)
	if headings[0].Level >= headings[1].Level {
		t.Error("Larger font heading should have lower level number")
	}
}

func TestFindHeadingBefore(t *testing.T) {
	// Headings in document order (top to bottom in reading order)
	// In PDF coords: higher Y = higher on page
	layout := &HeadingLayout{
		Headings: []Heading{
			{Text: "Heading 1", BBox: model.BBox{Y: 700, Height: 20}},
			{Text: "Heading 2", BBox: model.BBox{Y: 500, Height: 20}},
			{Text: "Heading 3", BBox: model.BBox{Y: 300, Height: 20}},
		},
	}

	// Find heading before Y=450 - headings above 450 are at Y=700 and Y=500
	// Should return the closest (last) one above, which is Heading 2 (Y=500)
	h := layout.FindHeadingBefore(450)
	if h == nil {
		t.Fatal("Expected to find a heading before Y=450")
	}
	if h.Text != "Heading 2" {
		t.Errorf("Expected 'Heading 2', got %q", h.Text)
	}

	// Find heading before Y=800 - no headings above 800
	// Should return nil since all headings are below Y=800
	h = layout.FindHeadingBefore(800)
	if h != nil {
		t.Errorf("Expected nil before Y=800 (above all content), got %q", h.Text)
	}

	// Find heading before Y=200 - all headings are above 200
	// Should return Heading 3 (Y=300), the last/closest one
	h = layout.FindHeadingBefore(200)
	if h == nil {
		t.Fatal("Expected to find a heading before Y=200")
	}
	if h.Text != "Heading 3" {
		t.Errorf("Expected 'Heading 3', got %q", h.Text)
	}

	// Find heading before Y=600 - headings at Y=700 are above 600
	// Should return Heading 1 (only heading above 600)
	h = layout.FindHeadingBefore(600)
	if h == nil || h.Text != "Heading 1" {
		if h == nil {
			t.Errorf("Expected 'Heading 1' before Y=600, got nil")
		} else {
			t.Errorf("Expected 'Heading 1' before Y=600, got %q", h.Text)
		}
	}
}

func TestFindHeadingsInRegion(t *testing.T) {
	layout := &HeadingLayout{
		Headings: []Heading{
			{Text: "Top", BBox: model.BBox{X: 100, Y: 700, Width: 200, Height: 20}},
			{Text: "Middle", BBox: model.BBox{X: 100, Y: 500, Width: 200, Height: 20}},
			{Text: "Bottom", BBox: model.BBox{X: 100, Y: 300, Width: 200, Height: 20}},
		},
	}

	// Region that covers middle heading
	region := model.BBox{X: 50, Y: 450, Width: 300, Height: 100}
	found := layout.FindHeadingsInRegion(region)

	if len(found) != 1 {
		t.Errorf("Expected 1 heading in region, got %d", len(found))
	}
	if len(found) > 0 && found[0].Text != "Middle" {
		t.Errorf("Expected 'Middle' heading, got %q", found[0].Text)
	}
}

// Benchmark tests
func BenchmarkHeadingDetection(b *testing.B) {
	// Create a realistic document structure
	var paragraphs []Paragraph

	// Title
	paragraphs = append(paragraphs, Paragraph{
		Text:            "Document Title",
		Style:           StyleHeading,
		AverageFontSize: 24,
		Lines: []Line{
			{Text: "Document Title", AverageFontSize: 24, BBox: model.BBox{X: 150, Y: 750, Width: 300, Height: 24}},
		},
		BBox: model.BBox{X: 150, Y: 750, Width: 300, Height: 24},
	})

	// Add 20 sections with body text
	for i := 0; i < 20; i++ {
		// Section heading
		paragraphs = append(paragraphs, Paragraph{
			Text:            "Section Heading",
			Style:           StyleHeading,
			AverageFontSize: 16,
			Lines: []Line{
				{Text: "Section Heading", AverageFontSize: 16, BBox: model.BBox{X: 72, Y: float64(700 - i*30), Width: 200, Height: 16}},
			},
			BBox: model.BBox{X: 72, Y: float64(700 - i*30), Width: 200, Height: 16},
		})

		// Body paragraphs
		for j := 0; j < 3; j++ {
			paragraphs = append(paragraphs, Paragraph{
				Text:            "Body text paragraph content.",
				Style:           StyleNormal,
				AverageFontSize: 12,
				Lines: []Line{
					{Text: "Body text paragraph content.", AverageFontSize: 12},
				},
			})
		}
	}

	detector := NewHeadingDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectFromParagraphs(paragraphs, 612, 792)
	}
}

func BenchmarkGetOutline(b *testing.B) {
	layout := &HeadingLayout{
		Headings: make([]Heading, 50),
	}

	// Create hierarchical headings
	for i := 0; i < 50; i++ {
		level := HeadingLevel((i % 3) + 1)
		layout.Headings[i] = Heading{
			Text:  "Heading",
			Level: level,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		layout.GetOutline()
	}
}
