package layout

import (
	"testing"

	"github.com/tsawler/tabula/text"
)

// makeBlockFragment creates a test text fragment for block tests
func makeBlockFragment(t string, x, y, width, height, fontSize float64) text.TextFragment {
	return text.TextFragment{
		Text:     t,
		X:        x,
		Y:        y,
		Width:    width,
		Height:   height,
		FontSize: fontSize,
	}
}

func TestBlockDetector_EmptyFragments(t *testing.T) {
	detector := NewBlockDetector()
	layout := detector.Detect(nil, 612, 792)

	if layout == nil {
		t.Fatal("Expected non-nil layout")
	}

	if layout.BlockCount() != 0 {
		t.Errorf("Expected 0 blocks, got %d", layout.BlockCount())
	}

	if layout.PageWidth != 612 || layout.PageHeight != 792 {
		t.Errorf("Page dimensions not set correctly")
	}
}

func TestBlockDetector_SingleFragment(t *testing.T) {
	detector := NewBlockDetector()
	fragments := []text.TextFragment{
		makeBlockFragment("Hello", 100, 700, 50, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.BlockCount() != 1 {
		t.Errorf("Expected 1 block, got %d", layout.BlockCount())
	}

	block := layout.GetBlock(0)
	if block.FragmentCount() != 1 {
		t.Errorf("Expected 1 fragment in block, got %d", block.FragmentCount())
	}

	if block.GetText() != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", block.GetText())
	}
}

func TestBlockDetector_SingleLine(t *testing.T) {
	detector := NewBlockDetector()
	fragments := []text.TextFragment{
		makeBlockFragment("Hello", 100, 700, 40, 12, 12),
		makeBlockFragment("World", 145, 700, 45, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.BlockCount() != 1 {
		t.Errorf("Expected 1 block, got %d", layout.BlockCount())
	}

	block := layout.GetBlock(0)
	if block.LineCount() != 1 {
		t.Errorf("Expected 1 line, got %d", block.LineCount())
	}

	text := block.GetText()
	if text != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", text)
	}
}

func TestBlockDetector_MultipleLines_SameBlock(t *testing.T) {
	detector := NewBlockDetector()
	// Two lines close together (should be same block)
	fragments := []text.TextFragment{
		makeBlockFragment("Line one", 100, 700, 60, 12, 12),
		makeBlockFragment("Line two", 100, 685, 60, 12, 12), // 15 points below (< 1.5 * 12)
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.BlockCount() != 1 {
		t.Errorf("Expected 1 block, got %d", layout.BlockCount())
	}

	block := layout.GetBlock(0)
	if block.LineCount() != 2 {
		t.Errorf("Expected 2 lines, got %d", block.LineCount())
	}
}

func TestBlockDetector_MultipleBlocks_VerticalGap(t *testing.T) {
	detector := NewBlockDetector()
	// Two blocks separated by large vertical gap
	fragments := []text.TextFragment{
		makeBlockFragment("First block", 100, 700, 80, 12, 12),
		makeBlockFragment("Second block", 100, 600, 90, 12, 12), // 100 points below (> 1.5 * 12)
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.BlockCount() != 2 {
		t.Errorf("Expected 2 blocks, got %d", layout.BlockCount())
	}

	// Blocks should be in reading order (top to bottom)
	block1 := layout.GetBlock(0)
	block2 := layout.GetBlock(1)

	if block1.GetText() != "First block" {
		t.Errorf("First block should be 'First block', got '%s'", block1.GetText())
	}

	if block2.GetText() != "Second block" {
		t.Errorf("Second block should be 'Second block', got '%s'", block2.GetText())
	}
}

func TestBlockDetector_HorizontalSeparation(t *testing.T) {
	detector := NewBlockDetector()
	// Two fragments far apart horizontally on different lines
	// Note: Block detection groups by lines first - fragments on the same Y
	// are grouped into the same line, then lines into blocks.
	// For true horizontal separation (columns), use ColumnDetector.
	fragments := []text.TextFragment{
		makeBlockFragment("Left block", 50, 700, 70, 12, 12),
		makeBlockFragment("continues", 50, 685, 60, 12, 12),
		makeBlockFragment("Right block", 400, 700, 80, 12, 12),
		makeBlockFragment("continues", 400, 685, 60, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	// Block detection sees these as one block since they share Y positions
	// (they're on the same horizontal lines)
	if layout.BlockCount() != 1 {
		t.Errorf("Expected 1 block (same Y lines), got %d", layout.BlockCount())
	}

	// But they should form 2 lines
	block := layout.GetBlock(0)
	if block.LineCount() != 2 {
		t.Errorf("Expected 2 lines, got %d", block.LineCount())
	}
}

func TestBlockDetector_Paragraph(t *testing.T) {
	detector := NewBlockDetector()
	// Simulate a paragraph with multiple lines
	fragments := []text.TextFragment{
		makeBlockFragment("This is the first line of a paragraph", 100, 700, 250, 12, 12),
		makeBlockFragment("and this is the second line continuing", 100, 686, 260, 12, 12),
		makeBlockFragment("and this is the third and final line.", 100, 672, 240, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.BlockCount() != 1 {
		t.Errorf("Expected 1 block for paragraph, got %d", layout.BlockCount())
	}

	block := layout.GetBlock(0)
	if block.LineCount() != 3 {
		t.Errorf("Expected 3 lines in paragraph block, got %d", block.LineCount())
	}
}

func TestBlockDetector_TwoParagraphs(t *testing.T) {
	detector := NewBlockDetector()
	// Two paragraphs separated by blank line
	fragments := []text.TextFragment{
		// First paragraph
		makeBlockFragment("First paragraph line one", 100, 700, 180, 12, 12),
		makeBlockFragment("First paragraph line two", 100, 686, 180, 12, 12),
		// Second paragraph (with large gap)
		makeBlockFragment("Second paragraph line one", 100, 640, 190, 12, 12), // 46 points gap
		makeBlockFragment("Second paragraph line two", 100, 626, 190, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.BlockCount() != 2 {
		t.Errorf("Expected 2 blocks for two paragraphs, got %d", layout.BlockCount())
	}

	block1 := layout.GetBlock(0)
	block2 := layout.GetBlock(1)

	if block1.LineCount() != 2 {
		t.Errorf("First paragraph should have 2 lines, got %d", block1.LineCount())
	}

	if block2.LineCount() != 2 {
		t.Errorf("Second paragraph should have 2 lines, got %d", block2.LineCount())
	}
}

func TestBlockDetector_WithConfig(t *testing.T) {
	config := BlockConfig{
		LineHeightTolerance:    0.3,
		HorizontalGapThreshold: 2.0,
		VerticalGapThreshold:   2.0, // Higher threshold (2.0 * 12 = 24 points)
		MinBlockWidth:          5.0,
		MinBlockHeight:         3.0,
		MergeOverlappingBlocks: false,
	}

	detector := NewBlockDetectorWithConfig(config)
	// Gap calculation: bottom of line 1 (Y=700) to top of line 2 (Y+Height=650+12=662)
	// Gap = 700 - 662 = 38, threshold = 2.0 * 12 = 24
	// 38 > 24 so they should be separate
	fragments := []text.TextFragment{
		makeBlockFragment("Block 1", 100, 700, 60, 12, 12),
		makeBlockFragment("Block 2", 100, 650, 60, 12, 12), // 50 points Y diff, 38 points actual gap
	}

	layout := detector.Detect(fragments, 612, 792)

	// With threshold 24 and gap 38, these should be separate blocks
	if layout.BlockCount() != 2 {
		t.Errorf("Expected 2 blocks with custom config, got %d", layout.BlockCount())
	}
}

func TestBlockDetector_ReadingOrder(t *testing.T) {
	detector := NewBlockDetector()
	// Fragments out of order
	fragments := []text.TextFragment{
		makeBlockFragment("Bottom", 100, 400, 60, 12, 12),
		makeBlockFragment("Top", 100, 700, 40, 12, 12),
		makeBlockFragment("Middle", 100, 550, 50, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.BlockCount() != 3 {
		t.Errorf("Expected 3 blocks, got %d", layout.BlockCount())
	}

	// Should be sorted top to bottom
	texts := []string{"Top", "Middle", "Bottom"}
	for i, expected := range texts {
		block := layout.GetBlock(i)
		if block.GetText() != expected {
			t.Errorf("Block %d should be '%s', got '%s'", i, expected, block.GetText())
		}
	}
}

func TestBlockDetector_GetAllFragments(t *testing.T) {
	detector := NewBlockDetector()
	fragments := []text.TextFragment{
		makeBlockFragment("One", 100, 700, 30, 12, 12),
		makeBlockFragment("Two", 100, 600, 30, 12, 12),
		makeBlockFragment("Three", 100, 500, 40, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)
	allFragments := layout.GetAllFragments()

	if len(allFragments) != 3 {
		t.Errorf("Expected 3 fragments, got %d", len(allFragments))
	}
}

func TestBlockDetector_GetText(t *testing.T) {
	detector := NewBlockDetector()
	fragments := []text.TextFragment{
		makeBlockFragment("First", 100, 700, 40, 12, 12),
		makeBlockFragment("Second", 100, 600, 50, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)
	text := layout.GetText()

	expected := "First\n\nSecond"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

func TestBlock_AverageFontSize(t *testing.T) {
	detector := NewBlockDetector()
	fragments := []text.TextFragment{
		makeBlockFragment("Large", 100, 700, 60, 24, 24),
		makeBlockFragment("Small", 100, 680, 40, 10, 10),
	}

	layout := detector.Detect(fragments, 612, 792)
	block := layout.GetBlock(0)

	avgSize := block.AverageFontSize()
	expected := 17.0 // (24 + 10) / 2

	if absFloat64(avgSize-expected) > 0.1 {
		t.Errorf("Expected average font size %.1f, got %.1f", expected, avgSize)
	}
}

func TestBlock_ContainsPoint(t *testing.T) {
	detector := NewBlockDetector()
	fragments := []text.TextFragment{
		makeBlockFragment("Test", 100, 700, 50, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)
	block := layout.GetBlock(0)

	// Point inside block
	if !block.ContainsPoint(125, 706) {
		t.Error("Expected point (125, 706) to be inside block")
	}

	// Point outside block
	if block.ContainsPoint(50, 706) {
		t.Error("Expected point (50, 706) to be outside block")
	}
}

func TestBlockDetector_CharacterLevelFragments(t *testing.T) {
	detector := NewBlockDetector()
	// Simulate character-level fragments (like Google Docs PDFs)
	fragments := []text.TextFragment{
		makeBlockFragment("H", 100, 700, 8, 12, 12),
		makeBlockFragment("e", 108, 700, 6, 12, 12),
		makeBlockFragment("l", 114, 700, 4, 12, 12),
		makeBlockFragment("l", 118, 700, 4, 12, 12),
		makeBlockFragment("o", 122, 700, 6, 12, 12),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.BlockCount() != 1 {
		t.Errorf("Expected 1 block for character-level fragments, got %d", layout.BlockCount())
	}

	block := layout.GetBlock(0)
	if block.LineCount() != 1 {
		t.Errorf("Expected 1 line, got %d", block.LineCount())
	}
}

func TestBlockDetector_MixedFontSizes(t *testing.T) {
	detector := NewBlockDetector()
	// Title followed by body text
	fragments := []text.TextFragment{
		makeBlockFragment("Title", 100, 700, 100, 24, 24),          // Large title
		makeBlockFragment("Body text here", 100, 660, 120, 12, 12), // Normal body (40 point gap)
	}

	layout := detector.Detect(fragments, 612, 792)

	// With default config, these might be separate blocks due to size difference
	// The gap (40) is > 1.5 * avg(24, 12) = 27, so should be separate
	if layout.BlockCount() != 2 {
		t.Errorf("Expected 2 blocks for title + body, got %d", layout.BlockCount())
	}
}

func TestBlockLayout_NilSafety(t *testing.T) {
	var layout *BlockLayout

	if layout.BlockCount() != 0 {
		t.Error("nil layout should return 0 blocks")
	}

	if layout.GetBlock(0) != nil {
		t.Error("nil layout should return nil block")
	}

	if layout.GetText() != "" {
		t.Error("nil layout should return empty string")
	}

	if layout.GetAllFragments() != nil {
		t.Error("nil layout should return nil fragments")
	}
}

func TestBlock_NilSafety(t *testing.T) {
	var block *Block

	if block.GetText() != "" {
		t.Error("nil block should return empty string")
	}

	if block.LineCount() != 0 {
		t.Error("nil block should return 0 lines")
	}

	if block.FragmentCount() != 0 {
		t.Error("nil block should return 0 fragments")
	}

	if block.AverageFontSize() != 0 {
		t.Error("nil block should return 0 average font size")
	}

	if block.ContainsPoint(0, 0) {
		t.Error("nil block should return false for ContainsPoint")
	}
}

func TestBlockDetector_MinimumBlockSize(t *testing.T) {
	config := DefaultBlockConfig()
	config.MinBlockWidth = 50
	config.MinBlockHeight = 10

	detector := NewBlockDetectorWithConfig(config)
	fragments := []text.TextFragment{
		makeBlockFragment("A", 100, 700, 5, 8, 8),                    // Too small
		makeBlockFragment("Valid block text", 200, 700, 100, 12, 12), // Valid
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.BlockCount() != 1 {
		t.Errorf("Expected 1 valid block (tiny block filtered), got %d", layout.BlockCount())
	}
}

func BenchmarkBlockDetector_SmallDocument(b *testing.B) {
	detector := NewBlockDetector()

	// Simulate a page with 5 paragraphs, ~10 lines each
	var fragments []text.TextFragment
	y := 750.0
	for para := 0; para < 5; para++ {
		for line := 0; line < 10; line++ {
			fragments = append(fragments, makeBlockFragment("Sample text content here", 72, y, 400, 12, 12))
			y -= 14
		}
		y -= 20 // Paragraph gap
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(fragments, 612, 792)
	}
}

func BenchmarkBlockDetector_LargeDocument(b *testing.B) {
	detector := NewBlockDetector()

	// Simulate a dense page with many fragments
	var fragments []text.TextFragment
	y := 750.0
	for i := 0; i < 500; i++ {
		fragments = append(fragments, makeBlockFragment("Word", 72+float64(i%10)*50, y, 40, 12, 12))
		if i%10 == 9 {
			y -= 14
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(fragments, 612, 792)
	}
}
