package layout

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// makeParagraphLine creates a test line for paragraph tests
func makeParagraphLine(txt string, x, y, width, height, fontSize float64, alignment LineAlignment) Line {
	return Line{
		Text:            txt,
		BBox:            model.BBox{X: x, Y: y, Width: width, Height: height},
		AverageFontSize: fontSize,
		Alignment:       alignment,
		Height:          height,
	}
}

func TestParagraphDetector_EmptyLines(t *testing.T) {
	detector := NewParagraphDetector()
	layout := detector.Detect(nil, 612, 792)

	if layout == nil {
		t.Fatal("Expected non-nil layout")
	}

	if layout.ParagraphCount() != 0 {
		t.Errorf("Expected 0 paragraphs, got %d", layout.ParagraphCount())
	}
}

func TestParagraphDetector_SingleLine(t *testing.T) {
	detector := NewParagraphDetector()
	lines := []Line{
		makeParagraphLine("Hello World", 72, 700, 100, 12, 12, AlignLeft),
	}

	layout := detector.Detect(lines, 612, 792)

	if layout.ParagraphCount() != 1 {
		t.Errorf("Expected 1 paragraph, got %d", layout.ParagraphCount())
	}

	para := layout.GetParagraph(0)
	if para.Text != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", para.Text)
	}
}

func TestParagraphDetector_SingleParagraph_MultipleLines(t *testing.T) {
	detector := NewParagraphDetector()
	// Lines close together (normal line spacing)
	lines := []Line{
		makeParagraphLine("First line of paragraph", 72, 700, 200, 12, 12, AlignLeft),
		makeParagraphLine("Second line continues", 72, 686, 180, 12, 12, AlignLeft),
		makeParagraphLine("Third line ends here.", 72, 672, 160, 12, 12, AlignLeft),
	}
	// Set spacing
	lines[1].SpacingBefore = 2 // Normal spacing
	lines[2].SpacingBefore = 2

	layout := detector.Detect(lines, 612, 792)

	if layout.ParagraphCount() != 1 {
		t.Errorf("Expected 1 paragraph, got %d", layout.ParagraphCount())
	}

	para := layout.GetParagraph(0)
	if para.LineCount() != 3 {
		t.Errorf("Expected 3 lines, got %d", para.LineCount())
	}
}

func TestParagraphDetector_TwoParagraphs_SpacingBreak(t *testing.T) {
	detector := NewParagraphDetector()
	// Two paragraphs separated by large spacing
	lines := []Line{
		makeParagraphLine("First paragraph line one", 72, 700, 200, 12, 12, AlignLeft),
		makeParagraphLine("First paragraph line two", 72, 686, 200, 12, 12, AlignLeft),
		makeParagraphLine("Second paragraph line one", 72, 650, 210, 12, 12, AlignLeft),
		makeParagraphLine("Second paragraph line two", 72, 636, 200, 12, 12, AlignLeft),
	}
	lines[1].SpacingBefore = 2  // Normal
	lines[2].SpacingBefore = 24 // Large gap - paragraph break
	lines[3].SpacingBefore = 2  // Normal

	layout := detector.Detect(lines, 612, 792)

	if layout.ParagraphCount() != 2 {
		t.Errorf("Expected 2 paragraphs, got %d", layout.ParagraphCount())
	}

	para1 := layout.GetParagraph(0)
	para2 := layout.GetParagraph(1)

	if para1.LineCount() != 2 {
		t.Errorf("First paragraph should have 2 lines, got %d", para1.LineCount())
	}

	if para2.LineCount() != 2 {
		t.Errorf("Second paragraph should have 2 lines, got %d", para2.LineCount())
	}
}

func TestParagraphDetector_FirstLineIndent(t *testing.T) {
	detector := NewParagraphDetector()
	// Paragraph with first-line indent
	lines := []Line{
		makeParagraphLine("  Indented first line", 100, 700, 200, 12, 12, AlignLeft),
		makeParagraphLine("Normal second line", 72, 686, 200, 12, 12, AlignLeft),
		makeParagraphLine("Normal third line", 72, 672, 180, 12, 12, AlignLeft),
	}
	lines[1].SpacingBefore = 2
	lines[2].SpacingBefore = 2

	layout := detector.Detect(lines, 612, 792)

	para := layout.GetParagraph(0)
	if para.FirstLineIndent <= 0 {
		t.Errorf("Expected positive first-line indent, got %.1f", para.FirstLineIndent)
	}
}

func TestParagraphDetector_HeadingDetection(t *testing.T) {
	detector := NewParagraphDetector()
	// Heading followed by body text
	lines := []Line{
		makeParagraphLine("Chapter Title", 72, 700, 150, 24, 24, AlignLeft), // Large font
		makeParagraphLine("Body text starts here", 72, 660, 200, 12, 12, AlignLeft),
		makeParagraphLine("More body text", 72, 646, 180, 12, 12, AlignLeft),
	}
	lines[1].SpacingBefore = 16 // Space after heading
	lines[2].SpacingBefore = 2

	layout := detector.Detect(lines, 612, 792)

	if layout.ParagraphCount() != 2 {
		t.Errorf("Expected 2 paragraphs, got %d", layout.ParagraphCount())
	}

	heading := layout.GetParagraph(0)
	if heading.Style != StyleHeading {
		t.Errorf("First paragraph should be heading, got %s", heading.Style)
	}
}

func TestParagraphDetector_ListItems(t *testing.T) {
	detector := NewParagraphDetector()
	lines := []Line{
		makeParagraphLine("Introduction text", 72, 700, 200, 12, 12, AlignLeft),
		makeParagraphLine("• First bullet point", 72, 670, 180, 12, 12, AlignLeft),
		makeParagraphLine("• Second bullet point", 72, 656, 190, 12, 12, AlignLeft),
		makeParagraphLine("• Third bullet point", 72, 642, 170, 12, 12, AlignLeft),
	}
	lines[1].SpacingBefore = 14 // Gap before list
	lines[2].SpacingBefore = 2
	lines[3].SpacingBefore = 2

	layout := detector.Detect(lines, 612, 792)

	// Should detect list items
	listItems := layout.GetListItems()
	if len(listItems) < 1 {
		t.Errorf("Expected at least 1 list item, got %d", len(listItems))
	}
}

func TestParagraphDetector_NumberedList(t *testing.T) {
	detector := NewParagraphDetector()
	lines := []Line{
		makeParagraphLine("1. First item", 72, 700, 100, 12, 12, AlignLeft),
		makeParagraphLine("2. Second item", 72, 686, 110, 12, 12, AlignLeft),
		makeParagraphLine("3. Third item", 72, 672, 105, 12, 12, AlignLeft),
	}
	lines[1].SpacingBefore = 2
	lines[2].SpacingBefore = 2

	layout := detector.Detect(lines, 612, 792)

	// All should be detected as list items
	for i := 0; i < layout.ParagraphCount(); i++ {
		para := layout.GetParagraph(i)
		if para.Style != StyleListItem {
			t.Errorf("Paragraph %d should be list item, got %s", i, para.Style)
		}
	}
}

func TestParagraphDetector_BlockQuote(t *testing.T) {
	// Use custom config with lower block quote indent threshold
	config := DefaultParagraphConfig()
	config.BlockQuoteIndent = 20.0 // Lower threshold
	detector := NewParagraphDetectorWithConfig(config)

	leftMargin := 72.0
	quoteIndent := 120.0 // Significantly indented (48 points from margin)

	lines := []Line{
		makeParagraphLine("Normal paragraph text here", leftMargin, 700, 400, 12, 12, AlignLeft),
		makeParagraphLine("Normal continues", leftMargin, 686, 380, 12, 12, AlignLeft),
		makeParagraphLine("This is a block quote that", quoteIndent, 650, 300, 12, 12, AlignLeft),
		makeParagraphLine("spans multiple lines here", quoteIndent, 636, 280, 12, 12, AlignLeft),
		makeParagraphLine("Back to normal text", leftMargin, 600, 400, 12, 12, AlignLeft),
	}
	lines[1].SpacingBefore = 2
	lines[2].SpacingBefore = 20 // Paragraph break before quote
	lines[3].SpacingBefore = 2
	lines[4].SpacingBefore = 20 // Paragraph break after quote

	layout := detector.Detect(lines, 612, 792)

	// Find block quote - it should be the middle paragraph
	found := false
	for i := 0; i < layout.ParagraphCount(); i++ {
		para := layout.GetParagraph(i)
		if para.Style == StyleBlockQuote {
			found = true
			break
		}
	}

	if !found {
		// Print debug info
		for i := 0; i < layout.ParagraphCount(); i++ {
			para := layout.GetParagraph(i)
			textPreview := para.Text
			if len(textPreview) > 30 {
				textPreview = textPreview[:30]
			}
			t.Logf("Paragraph %d: style=%s, leftMargin=%.1f, text=%q", i, para.Style, para.LeftMargin, textPreview)
		}
		t.Error("Expected to find a block quote paragraph")
	}
}

func TestParagraphDetector_GetText(t *testing.T) {
	detector := NewParagraphDetector()
	// Multiple lines with clear paragraph breaks
	lines := []Line{
		makeParagraphLine("First paragraph line one.", 72, 700, 200, 12, 12, AlignLeft),
		makeParagraphLine("First paragraph line two.", 72, 686, 200, 12, 12, AlignLeft),
		makeParagraphLine("Second paragraph line one.", 72, 640, 210, 12, 12, AlignLeft),
		makeParagraphLine("Second paragraph line two.", 72, 626, 200, 12, 12, AlignLeft),
	}
	lines[1].SpacingBefore = 2  // Normal spacing within paragraph
	lines[2].SpacingBefore = 34 // Large gap - paragraph break (> default avg * 1.5)
	lines[3].SpacingBefore = 2  // Normal spacing within paragraph

	layout := detector.Detect(lines, 612, 792)

	if layout.ParagraphCount() != 2 {
		t.Errorf("Expected 2 paragraphs, got %d", layout.ParagraphCount())
		for i := 0; i < layout.ParagraphCount(); i++ {
			t.Logf("Para %d: %q", i, layout.GetParagraph(i).Text)
		}
		return
	}

	text := layout.GetText()

	if !strings.Contains(text, "First paragraph") || !strings.Contains(text, "Second paragraph") {
		t.Errorf("Text should contain both paragraphs, got: %s", text)
	}

	// Should have paragraph break (two paragraphs = one double newline separator)
	if !strings.Contains(text, "\n\n") {
		t.Errorf("Expected paragraph break (double newline), got: %q", text)
	}
}

func TestParagraphDetector_GetHeadings(t *testing.T) {
	detector := NewParagraphDetector()
	lines := []Line{
		makeParagraphLine("Main Title", 72, 700, 150, 24, 24, AlignLeft),
		makeParagraphLine("Body text here", 72, 660, 200, 12, 12, AlignLeft),
		makeParagraphLine("Subtitle", 72, 620, 100, 18, 18, AlignLeft),
		makeParagraphLine("More body text", 72, 590, 200, 12, 12, AlignLeft),
	}
	lines[1].SpacingBefore = 16
	lines[2].SpacingBefore = 16
	lines[3].SpacingBefore = 16

	layout := detector.Detect(lines, 612, 792)
	headings := layout.GetHeadings()

	if len(headings) < 1 {
		t.Errorf("Expected at least 1 heading, got %d", len(headings))
	}
}

func TestParagraphDetector_FindParagraphsInRegion(t *testing.T) {
	detector := NewParagraphDetector()
	lines := []Line{
		makeParagraphLine("Top paragraph", 72, 750, 200, 12, 12, AlignLeft),
		makeParagraphLine("Middle paragraph", 72, 500, 200, 12, 12, AlignLeft),
		makeParagraphLine("Bottom paragraph", 72, 250, 200, 12, 12, AlignLeft),
	}
	lines[1].SpacingBefore = 200
	lines[2].SpacingBefore = 200

	layout := detector.Detect(lines, 612, 792)

	// Find paragraphs in middle region
	region := model.BBox{X: 0, Y: 400, Width: 612, Height: 200}
	found := layout.FindParagraphsInRegion(region)

	if len(found) != 1 {
		t.Errorf("Expected 1 paragraph in region, got %d", len(found))
	}
}

func TestParagraphDetector_DetectFromFragments(t *testing.T) {
	detector := NewParagraphDetector()
	fragments := []text.TextFragment{
		{Text: "First line", X: 72, Y: 700, Width: 80, Height: 12, FontSize: 12},
		{Text: "Second line", X: 72, Y: 686, Width: 90, Height: 12, FontSize: 12},
	}

	layout := detector.DetectFromFragments(fragments, 612, 792)

	if layout.ParagraphCount() < 1 {
		t.Error("Expected at least 1 paragraph")
	}
}

func TestParagraph_WordCount(t *testing.T) {
	para := &Paragraph{
		Text: "This is a test paragraph with eight words.",
	}

	if para.WordCount() != 8 {
		t.Errorf("Expected 8 words, got %d", para.WordCount())
	}
}

func TestParagraph_IsHeading(t *testing.T) {
	para := &Paragraph{Style: StyleHeading}
	if !para.IsHeading() {
		t.Error("Expected IsHeading to return true")
	}

	para2 := &Paragraph{Style: StyleNormal}
	if para2.IsHeading() {
		t.Error("Expected IsHeading to return false for normal paragraph")
	}
}

func TestParagraph_IsListItem(t *testing.T) {
	para := &Paragraph{Style: StyleListItem}
	if !para.IsListItem() {
		t.Error("Expected IsListItem to return true")
	}
}

func TestParagraph_IsBlockQuote(t *testing.T) {
	para := &Paragraph{Style: StyleBlockQuote}
	if !para.IsBlockQuote() {
		t.Error("Expected IsBlockQuote to return true")
	}
}

func TestParagraph_HasFirstLineIndent(t *testing.T) {
	para := &Paragraph{FirstLineIndent: 20}
	if !para.HasFirstLineIndent() {
		t.Error("Expected HasFirstLineIndent to return true")
	}

	para2 := &Paragraph{FirstLineIndent: 2}
	if para2.HasFirstLineIndent() {
		t.Error("Expected HasFirstLineIndent to return false for small indent")
	}
}

func TestParagraph_ContainsPoint(t *testing.T) {
	para := &Paragraph{
		BBox: model.BBox{X: 72, Y: 700, Width: 400, Height: 50},
	}

	if !para.ContainsPoint(200, 720) {
		t.Error("Point should be inside paragraph")
	}

	if para.ContainsPoint(50, 720) {
		t.Error("Point should be outside paragraph")
	}
}

func TestParagraph_GetFirstLastLine(t *testing.T) {
	para := &Paragraph{
		Lines: []Line{
			{Text: "First"},
			{Text: "Middle"},
			{Text: "Last"},
		},
	}

	first := para.GetFirstLine()
	if first == nil || first.Text != "First" {
		t.Error("GetFirstLine failed")
	}

	last := para.GetLastLine()
	if last == nil || last.Text != "Last" {
		t.Error("GetLastLine failed")
	}
}

func TestParagraphStyle_String(t *testing.T) {
	tests := []struct {
		style    ParagraphStyle
		expected string
	}{
		{StyleNormal, "normal"},
		{StyleHeading, "heading"},
		{StyleBlockQuote, "blockquote"},
		{StyleListItem, "list-item"},
		{StyleCode, "code"},
		{StyleCaption, "caption"},
	}

	for _, tc := range tests {
		if tc.style.String() != tc.expected {
			t.Errorf("Expected '%s', got '%s'", tc.expected, tc.style.String())
		}
	}
}

func TestParagraphLayout_NilSafety(t *testing.T) {
	var layout *ParagraphLayout

	if layout.ParagraphCount() != 0 {
		t.Error("nil layout should return 0 paragraphs")
	}

	if layout.GetParagraph(0) != nil {
		t.Error("nil layout should return nil paragraph")
	}

	if layout.GetText() != "" {
		t.Error("nil layout should return empty string")
	}

	if layout.GetParagraphsByStyle(StyleNormal) != nil {
		t.Error("nil layout should return nil for GetParagraphsByStyle")
	}

	if layout.GetHeadings() != nil {
		t.Error("nil layout should return nil for GetHeadings")
	}

	if layout.GetListItems() != nil {
		t.Error("nil layout should return nil for GetListItems")
	}

	if layout.FindParagraphsInRegion(model.BBox{}) != nil {
		t.Error("nil layout should return nil for FindParagraphsInRegion")
	}
}

func TestParagraph_NilSafety(t *testing.T) {
	var para *Paragraph

	if para.LineCount() != 0 {
		t.Error("nil paragraph should return 0 lines")
	}

	if para.WordCount() != 0 {
		t.Error("nil paragraph should return 0 words")
	}

	if para.IsHeading() {
		t.Error("nil paragraph should not be heading")
	}

	if para.IsListItem() {
		t.Error("nil paragraph should not be list item")
	}

	if para.IsBlockQuote() {
		t.Error("nil paragraph should not be block quote")
	}

	if para.HasFirstLineIndent() {
		t.Error("nil paragraph should not have first line indent")
	}

	if para.ContainsPoint(0, 0) {
		t.Error("nil paragraph should return false for ContainsPoint")
	}

	if para.GetFirstLine() != nil {
		t.Error("nil paragraph should return nil for GetFirstLine")
	}

	if para.GetLastLine() != nil {
		t.Error("nil paragraph should return nil for GetLastLine")
	}
}

func TestParagraphDetector_CustomConfig(t *testing.T) {
	config := ParagraphConfig{
		SpacingThreshold:     2.0, // More aggressive paragraph splitting
		IndentThreshold:      10.0,
		HeadingFontSizeRatio: 1.3,
		MinParagraphLines:    1,
		BlockQuoteIndent:     40.0,
	}

	detector := NewParagraphDetectorWithConfig(config)
	lines := []Line{
		makeParagraphLine("Line one", 72, 700, 200, 12, 12, AlignLeft),
		makeParagraphLine("Line two", 72, 686, 200, 12, 12, AlignLeft),
	}
	lines[1].SpacingBefore = 8 // Small gap

	layout := detector.Detect(lines, 612, 792)

	if layout.Config.SpacingThreshold != 2.0 {
		t.Errorf("Config not applied: expected threshold 2.0, got %.1f", layout.Config.SpacingThreshold)
	}
}

func TestParagraphDetector_AlignmentChange(t *testing.T) {
	detector := NewParagraphDetector()
	lines := []Line{
		makeParagraphLine("Left aligned text", 72, 700, 200, 12, 12, AlignLeft),
		makeParagraphLine("Centered text", 256, 680, 100, 12, 12, AlignCenter),
		makeParagraphLine("More left aligned", 72, 660, 200, 12, 12, AlignLeft),
	}
	lines[1].SpacingBefore = 4
	lines[2].SpacingBefore = 4

	layout := detector.Detect(lines, 612, 792)

	// Alignment change should create new paragraph
	if layout.ParagraphCount() < 2 {
		t.Errorf("Expected at least 2 paragraphs due to alignment change, got %d", layout.ParagraphCount())
	}
}

func TestIsListItem(t *testing.T) {
	detector := NewParagraphDetector()

	tests := []struct {
		text     string
		expected bool
	}{
		{"• Bullet point", true},
		{"- Dash item", true},
		{"* Star item", true},
		{"1. Numbered item", true},
		{"2) Parenthesis number", true},
		{"a. Letter item", true},
		{"b) Letter parenthesis", true},
		{"Normal text", false},
		{"10. Double digit", true},
		{"", false},
	}

	for _, tc := range tests {
		result := detector.isListItem(tc.text)
		if result != tc.expected {
			t.Errorf("isListItem(%q) = %v, expected %v", tc.text, result, tc.expected)
		}
	}
}

func BenchmarkParagraphDetector_SmallDocument(b *testing.B) {
	detector := NewParagraphDetector()

	// Simulate 5 paragraphs with 10 lines each
	var lines []Line
	y := 750.0
	for para := 0; para < 5; para++ {
		for line := 0; line < 10; line++ {
			l := makeParagraphLine("Sample text content here for testing", 72, y, 400, 12, 12, AlignLeft)
			if line > 0 {
				l.SpacingBefore = 2
			} else if para > 0 {
				l.SpacingBefore = 20 // Paragraph break
			}
			lines = append(lines, l)
			y -= 14
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(lines, 612, 792)
	}
}

func BenchmarkParagraphDetector_LargeDocument(b *testing.B) {
	detector := NewParagraphDetector()

	// Simulate 50 paragraphs with 5 lines each
	var lines []Line
	y := 750.0
	for para := 0; para < 50; para++ {
		for line := 0; line < 5; line++ {
			l := makeParagraphLine("Sample text content here", 72, y, 300, 12, 12, AlignLeft)
			if line > 0 {
				l.SpacingBefore = 2
			} else if para > 0 {
				l.SpacingBefore = 20
			}
			lines = append(lines, l)
			y -= 14
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(lines, 612, 792)
	}
}
