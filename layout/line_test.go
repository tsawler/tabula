package layout

import (
	"testing"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// makeLineFragment creates a test text fragment for line tests
func makeLineFragment(txt string, x, y, width, height, fontSize float64) text.TextFragment {
	return text.TextFragment{
		Text:      txt,
		X:         x,
		Y:         y,
		Width:     width,
		Height:    height,
		FontSize:  fontSize,
		Direction: text.LTR,
	}
}

func TestLineDetector_EmptyFragments(t *testing.T) {
	detector := NewLineDetector()
	layout := detector.Detect(nil, 612, 792)

	if layout == nil {
		t.Fatal("Expected non-nil layout")
	}

	if layout.LineCount() != 0 {
		t.Errorf("Expected 0 lines, got %d", layout.LineCount())
	}

	if layout.PageWidth != 612 || layout.PageHeight != 792 {
		t.Errorf("Page dimensions not set correctly")
	}
}

func TestLineDetector_SingleFragment(t *testing.T) {
	detector := NewLineDetector()
	fragments := []text.TextFragment{
		makeLineFragment("Hello", 100, 700, 50, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.LineCount() != 1 {
		t.Errorf("Expected 1 line, got %d", layout.LineCount())
	}

	line := layout.GetLine(0)
	if line.Text != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", line.Text)
	}

	if line.Index != 0 {
		t.Errorf("Expected index 0, got %d", line.Index)
	}
}

func TestLineDetector_SingleLine_MultipleFragments(t *testing.T) {
	detector := NewLineDetector()
	fragments := []text.TextFragment{
		makeLineFragment("Hello", 100, 700, 40, 12, 12),
		makeLineFragment("World", 145, 700, 45, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.LineCount() != 1 {
		t.Errorf("Expected 1 line, got %d", layout.LineCount())
	}

	line := layout.GetLine(0)
	if line.Text != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", line.Text)
	}

	if len(line.Fragments) != 2 {
		t.Errorf("Expected 2 fragments, got %d", len(line.Fragments))
	}
}

func TestLineDetector_MultipleLines(t *testing.T) {
	detector := NewLineDetector()
	fragments := []text.TextFragment{
		makeLineFragment("Line one", 100, 700, 60, 12, 12),
		makeLineFragment("Line two", 100, 685, 60, 12, 12),
		makeLineFragment("Line three", 100, 670, 70, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.LineCount() != 3 {
		t.Errorf("Expected 3 lines, got %d", layout.LineCount())
	}

	// Lines should be in reading order (top to bottom)
	expectedTexts := []string{"Line one", "Line two", "Line three"}
	for i, expected := range expectedTexts {
		line := layout.GetLine(i)
		if line.Text != expected {
			t.Errorf("Line %d: expected '%s', got '%s'", i, expected, line.Text)
		}
	}
}

func TestLineDetector_LineSpacing(t *testing.T) {
	detector := NewLineDetector()
	// Lines with consistent 15-point spacing
	fragments := []text.TextFragment{
		makeLineFragment("Line one", 100, 700, 60, 12, 12),
		makeLineFragment("Line two", 100, 685, 60, 12, 12), // 15 points below
		makeLineFragment("Line three", 100, 670, 70, 12, 12), // 15 points below
	}

	layout := detector.Detect(fragments, 612, 792)

	// Check spacing calculations
	line1 := layout.GetLine(0)
	line2 := layout.GetLine(1)

	if line1.SpacingAfter <= 0 {
		t.Errorf("Line 1 should have spacing after, got %.1f", line1.SpacingAfter)
	}

	if line2.SpacingBefore <= 0 {
		t.Errorf("Line 2 should have spacing before, got %.1f", line2.SpacingBefore)
	}

	// Spacing should match
	if absFloat64(line1.SpacingAfter-line2.SpacingBefore) > 0.1 {
		t.Errorf("Spacing mismatch: after=%.1f, before=%.1f", line1.SpacingAfter, line2.SpacingBefore)
	}
}

func TestLineDetector_AverageLineSpacing(t *testing.T) {
	detector := NewLineDetector()
	fragments := []text.TextFragment{
		makeLineFragment("Line one", 100, 700, 60, 12, 12),
		makeLineFragment("Line two", 100, 685, 60, 12, 12),
		makeLineFragment("Line three", 100, 670, 70, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.AverageLineSpacing <= 0 {
		t.Errorf("Expected positive average line spacing, got %.1f", layout.AverageLineSpacing)
	}

	if layout.AverageLineHeight <= 0 {
		t.Errorf("Expected positive average line height, got %.1f", layout.AverageLineHeight)
	}
}

func TestLineDetector_Alignment_Left(t *testing.T) {
	detector := NewLineDetector()
	// Left-aligned lines (all start at same X)
	fragments := []text.TextFragment{
		makeLineFragment("Short line", 72, 700, 80, 12, 12),
		makeLineFragment("Medium length line", 72, 685, 140, 12, 12),
		makeLineFragment("A much longer line of text", 72, 670, 200, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	for i := 0; i < layout.LineCount(); i++ {
		line := layout.GetLine(i)
		if line.Alignment != AlignLeft && line.Alignment != AlignJustified {
			t.Errorf("Line %d: expected left alignment, got %s", i, line.Alignment)
		}
	}
}

func TestLineDetector_Alignment_Center(t *testing.T) {
	detector := NewLineDetector()
	// Centered lines (different start X but same center)
	// Use widths that are clearly not justified (< 90% of max width)
	centerX := 306.0 // Center of 612-width page
	fragments := []text.TextFragment{
		makeLineFragment("Short", centerX-25, 700, 50, 12, 12),    // 50 wide, center at 306
		makeLineFragment("Medium", centerX-40, 685, 80, 12, 12),   // 80 wide, center at 306
		makeLineFragment("Longest line", centerX-100, 670, 200, 12, 12), // 200 wide, center at 306
	}

	layout := detector.Detect(fragments, 612, 792)

	// At least the shorter lines should be detected as centered
	centerCount := 0
	for i := 0; i < layout.LineCount(); i++ {
		line := layout.GetLine(i)
		if line.Alignment == AlignCenter {
			centerCount++
		}
	}

	if centerCount < 2 {
		t.Errorf("Expected at least 2 center-aligned lines, got %d", centerCount)
	}
}

func TestLineDetector_Alignment_Right(t *testing.T) {
	detector := NewLineDetector()
	// Right-aligned lines (all end at same X)
	// Use widths that are clearly not justified (< 90% of max width)
	rightEdge := 540.0
	fragments := []text.TextFragment{
		makeLineFragment("Short", rightEdge-50, 700, 50, 12, 12),    // 50 wide
		makeLineFragment("Medium", rightEdge-80, 685, 80, 12, 12),   // 80 wide
		makeLineFragment("Longest line", rightEdge-200, 670, 200, 12, 12), // 200 wide
	}

	layout := detector.Detect(fragments, 612, 792)

	// At least the shorter lines should be detected as right-aligned
	rightCount := 0
	for i := 0; i < layout.LineCount(); i++ {
		line := layout.GetLine(i)
		if line.Alignment == AlignRight {
			rightCount++
		}
	}

	if rightCount < 2 {
		t.Errorf("Expected at least 2 right-aligned lines, got %d", rightCount)
	}
}

func TestLineDetector_Alignment_Justified(t *testing.T) {
	detector := NewLineDetector()
	// Justified lines (all span full width)
	fragments := []text.TextFragment{
		makeLineFragment("Full width justified line one", 72, 700, 468, 12, 12),
		makeLineFragment("Full width justified line two", 72, 685, 468, 12, 12),
		makeLineFragment("Full width justified line three", 72, 670, 468, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	for i := 0; i < layout.LineCount(); i++ {
		line := layout.GetLine(i)
		if line.Alignment != AlignJustified {
			t.Errorf("Line %d: expected justified alignment, got %s", i, line.Alignment)
		}
	}
}

func TestLineDetector_Indentation(t *testing.T) {
	detector := NewLineDetector()
	fragments := []text.TextFragment{
		makeLineFragment("Normal line", 72, 700, 100, 12, 12),
		makeLineFragment("Indented line", 100, 685, 100, 12, 12), // Indented
	}

	layout := detector.Detect(fragments, 612, 792)

	line1 := layout.GetLine(0)
	line2 := layout.GetLine(1)

	if line1.Indentation != 72 {
		t.Errorf("Line 1 indentation: expected 72, got %.1f", line1.Indentation)
	}

	if line2.Indentation != 100 {
		t.Errorf("Line 2 indentation: expected 100, got %.1f", line2.Indentation)
	}

	// Test IsIndented helper
	if !line2.IsIndented(72, 10) {
		t.Error("Line 2 should be indented relative to margin 72")
	}
}

func TestLineDetector_GetText(t *testing.T) {
	detector := NewLineDetector()
	fragments := []text.TextFragment{
		makeLineFragment("First line", 100, 700, 80, 12, 12),
		makeLineFragment("Second line", 100, 685, 90, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)
	text := layout.GetText()

	expected := "First line\nSecond line"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

func TestLineDetector_GetAllFragments(t *testing.T) {
	detector := NewLineDetector()
	fragments := []text.TextFragment{
		makeLineFragment("One", 100, 700, 30, 12, 12),
		makeLineFragment("Two", 140, 700, 30, 12, 12),
		makeLineFragment("Three", 100, 685, 40, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)
	allFragments := layout.GetAllFragments()

	if len(allFragments) != 3 {
		t.Errorf("Expected 3 fragments, got %d", len(allFragments))
	}
}

func TestLineDetector_FindLinesInRegion(t *testing.T) {
	detector := NewLineDetector()
	fragments := []text.TextFragment{
		makeLineFragment("Top line", 100, 750, 80, 12, 12),
		makeLineFragment("Middle line", 100, 700, 90, 12, 12),
		makeLineFragment("Bottom line", 100, 650, 85, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	// Find lines in middle region
	region := model.BBox{X: 0, Y: 690, Width: 612, Height: 50}
	linesInRegion := layout.FindLinesInRegion(region)

	if len(linesInRegion) != 1 {
		t.Errorf("Expected 1 line in region, got %d", len(linesInRegion))
	}

	if len(linesInRegion) > 0 && linesInRegion[0].Text != "Middle line" {
		t.Errorf("Expected 'Middle line', got '%s'", linesInRegion[0].Text)
	}
}

func TestLineDetector_GetLinesByAlignment(t *testing.T) {
	detector := NewLineDetector()
	// Mix of alignments
	fragments := []text.TextFragment{
		makeLineFragment("Left aligned line", 72, 700, 120, 12, 12),
		makeLineFragment("Full width justified line here", 72, 685, 468, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	justifiedLines := layout.GetLinesByAlignment(AlignJustified)
	if len(justifiedLines) != 1 {
		t.Errorf("Expected 1 justified line, got %d", len(justifiedLines))
	}
}

func TestLineDetector_IsParagraphBreak(t *testing.T) {
	detector := NewLineDetector()
	// Lines with a paragraph break
	fragments := []text.TextFragment{
		makeLineFragment("Paragraph one line one", 100, 700, 160, 12, 12),
		makeLineFragment("Paragraph one line two", 100, 685, 160, 12, 12),
		// Large gap here (paragraph break)
		makeLineFragment("Paragraph two line one", 100, 640, 160, 12, 12), // 45 point gap
	}

	layout := detector.Detect(fragments, 612, 792)

	// Line 1 (second line of first paragraph) should have paragraph break after
	if !layout.IsParagraphBreak(1) {
		t.Error("Expected paragraph break after line 1")
	}

	// Line 0 should not have paragraph break after
	if layout.IsParagraphBreak(0) {
		t.Error("Did not expect paragraph break after line 0")
	}
}

func TestLine_WordCount(t *testing.T) {
	detector := NewLineDetector()
	fragments := []text.TextFragment{
		makeLineFragment("This is a test line with seven words", 100, 700, 250, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)
	line := layout.GetLine(0)

	if line.WordCount() != 8 {
		t.Errorf("Expected 8 words, got %d", line.WordCount())
	}
}

func TestLine_IsEmpty(t *testing.T) {
	line := &Line{Text: "   "}
	if !line.IsEmpty() {
		t.Error("Line with only spaces should be empty")
	}

	line2 := &Line{Text: "Hello"}
	if line2.IsEmpty() {
		t.Error("Line with text should not be empty")
	}
}

func TestLine_HasLargerFont(t *testing.T) {
	line := &Line{AverageFontSize: 24}

	if !line.HasLargerFont(12) {
		t.Error("24pt font should be larger than 12pt")
	}

	if line.HasLargerFont(30) {
		t.Error("24pt font should not be larger than 30pt")
	}
}

func TestLine_ContainsPoint(t *testing.T) {
	line := &Line{
		BBox: model.BBox{X: 100, Y: 700, Width: 200, Height: 12},
	}

	if !line.ContainsPoint(150, 706) {
		t.Error("Point (150, 706) should be inside line")
	}

	if line.ContainsPoint(50, 706) {
		t.Error("Point (50, 706) should be outside line")
	}
}

func TestLineAlignment_String(t *testing.T) {
	tests := []struct {
		alignment LineAlignment
		expected  string
	}{
		{AlignLeft, "left"},
		{AlignCenter, "center"},
		{AlignRight, "right"},
		{AlignJustified, "justified"},
		{AlignUnknown, "unknown"},
	}

	for _, tc := range tests {
		if tc.alignment.String() != tc.expected {
			t.Errorf("Expected '%s', got '%s'", tc.expected, tc.alignment.String())
		}
	}
}

func TestLineLayout_NilSafety(t *testing.T) {
	var layout *LineLayout

	if layout.LineCount() != 0 {
		t.Error("nil layout should return 0 lines")
	}

	if layout.GetLine(0) != nil {
		t.Error("nil layout should return nil line")
	}

	if layout.GetText() != "" {
		t.Error("nil layout should return empty string")
	}

	if layout.GetAllFragments() != nil {
		t.Error("nil layout should return nil fragments")
	}

	if layout.FindLinesInRegion(model.BBox{}) != nil {
		t.Error("nil layout should return nil for FindLinesInRegion")
	}

	if layout.GetLinesByAlignment(AlignLeft) != nil {
		t.Error("nil layout should return nil for GetLinesByAlignment")
	}

	if layout.IsParagraphBreak(0) {
		t.Error("nil layout should return false for IsParagraphBreak")
	}
}

func TestLine_NilSafety(t *testing.T) {
	var line *Line

	if line.IsIndented(0, 0) {
		t.Error("nil line should return false for IsIndented")
	}

	if line.ContainsPoint(0, 0) {
		t.Error("nil line should return false for ContainsPoint")
	}

	if line.WordCount() != 0 {
		t.Error("nil line should return 0 for WordCount")
	}

	if !line.IsEmpty() {
		t.Error("nil line should be empty")
	}

	if line.HasLargerFont(12) {
		t.Error("nil line should return false for HasLargerFont")
	}
}

func TestLineDetector_CharacterLevelFragments(t *testing.T) {
	detector := NewLineDetector()
	// Simulate character-level fragments (Google Docs style)
	fragments := []text.TextFragment{
		makeLineFragment("H", 100, 700, 8, 12, 12),
		makeLineFragment("e", 108, 700, 6, 12, 12),
		makeLineFragment("l", 114, 700, 4, 12, 12),
		makeLineFragment("l", 118, 700, 4, 12, 12),
		makeLineFragment("o", 122, 700, 6, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.LineCount() != 1 {
		t.Errorf("Expected 1 line for character-level fragments, got %d", layout.LineCount())
	}

	line := layout.GetLine(0)
	if line.Text != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", line.Text)
	}
}

func TestLineDetector_CustomConfig(t *testing.T) {
	config := LineConfig{
		LineHeightTolerance:    0.3,
		MinLineWidth:           20.0,
		AlignmentTolerance:     5.0,
		JustificationThreshold: 0.95,
	}

	detector := NewLineDetectorWithConfig(config)
	fragments := []text.TextFragment{
		makeLineFragment("Test", 100, 700, 40, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.Config.LineHeightTolerance != 0.3 {
		t.Errorf("Config not applied: expected tolerance 0.3, got %.1f", layout.Config.LineHeightTolerance)
	}
}

func TestLineDetector_OutOfOrderFragments(t *testing.T) {
	detector := NewLineDetector()
	// Fragments added out of visual order
	fragments := []text.TextFragment{
		makeLineFragment("World", 150, 700, 45, 12, 12),
		makeLineFragment("Line two", 100, 685, 60, 12, 12),
		makeLineFragment("Hello", 100, 700, 45, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.LineCount() != 2 {
		t.Errorf("Expected 2 lines, got %d", layout.LineCount())
	}

	// First line should have "Hello World" (sorted by X)
	line1 := layout.GetLine(0)
	if line1.Text != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", line1.Text)
	}
}

func BenchmarkLineDetector_SmallDocument(b *testing.B) {
	detector := NewLineDetector()

	// Simulate a page with ~50 lines
	var fragments []text.TextFragment
	y := 750.0
	for i := 0; i < 50; i++ {
		fragments = append(fragments, makeLineFragment("Sample text content here", 72, y, 400, 12, 12))
		y -= 14
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(fragments, 612, 792)
	}
}

func BenchmarkLineDetector_LargeDocument(b *testing.B) {
	detector := NewLineDetector()

	// Simulate a dense page with many fragments per line
	var fragments []text.TextFragment
	y := 750.0
	for line := 0; line < 50; line++ {
		for word := 0; word < 10; word++ {
			fragments = append(fragments, makeLineFragment("Word", 72+float64(word)*50, y, 40, 12, 12))
		}
		y -= 14
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(fragments, 612, 792)
	}
}
