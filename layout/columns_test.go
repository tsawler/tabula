package layout

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/text"
)

// Helper to create a text fragment
func makeFragment(x, y, width, height float64, txt string) text.TextFragment {
	return text.TextFragment{
		X:        x,
		Y:        y,
		Width:    width,
		Height:   height,
		Text:     txt,
		FontSize: height,
	}
}

func TestColumnDetector_EmptyInput(t *testing.T) {
	detector := NewColumnDetector()

	layout := detector.Detect(nil, 612, 792) // US Letter size

	if layout == nil {
		t.Fatal("expected non-nil layout")
	}

	if layout.ColumnCount() != 0 {
		t.Errorf("expected 0 columns for empty input, got %d", layout.ColumnCount())
	}

	if !layout.IsSingleColumn() {
		t.Error("empty layout should be treated as single column")
	}
}

func TestColumnDetector_SingleColumn(t *testing.T) {
	detector := NewColumnDetector()

	// Single column of text spanning the full page width
	fragments := []text.TextFragment{
		makeFragment(72, 700, 468, 12, "This is the title of the document"),
		makeFragment(72, 680, 468, 10, "First paragraph of text that spans the full width of the page."),
		makeFragment(72, 660, 468, 10, "Second line of the first paragraph continues here."),
		makeFragment(72, 630, 468, 10, "Another paragraph starts here with more content."),
		makeFragment(72, 610, 468, 10, "This is the final line of content."),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.ColumnCount() != 1 {
		t.Errorf("expected 1 column, got %d", layout.ColumnCount())
	}

	if !layout.IsSingleColumn() {
		t.Error("expected single column layout")
	}

	if layout.IsMultiColumn() {
		t.Error("should not be multi-column")
	}

	// Verify reading order
	orderedFrags := layout.GetFragmentsInReadingOrder()
	if len(orderedFrags) != len(fragments) {
		t.Errorf("expected %d fragments, got %d", len(fragments), len(orderedFrags))
	}

	// First fragment should be the title (highest Y)
	if orderedFrags[0].Text != "This is the title of the document" {
		t.Errorf("expected title first, got %q", orderedFrags[0].Text)
	}
}

func TestColumnDetector_TwoColumns(t *testing.T) {
	detector := NewColumnDetector()

	// Two-column academic paper style layout
	// Left column: x=72 to x=290
	// Gap: x=290 to x=322 (32 points)
	// Right column: x=322 to x=540
	fragments := []text.TextFragment{
		// Left column (top to bottom)
		makeFragment(72, 700, 218, 12, "Left Column Title"),
		makeFragment(72, 680, 218, 10, "Left paragraph one."),
		makeFragment(72, 660, 218, 10, "Left paragraph two."),
		makeFragment(72, 640, 218, 10, "Left paragraph three."),
		// Right column (top to bottom)
		makeFragment(322, 700, 218, 12, "Right Column Title"),
		makeFragment(322, 680, 218, 10, "Right paragraph one."),
		makeFragment(322, 660, 218, 10, "Right paragraph two."),
		makeFragment(322, 640, 218, 10, "Right paragraph three."),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.ColumnCount() != 2 {
		t.Errorf("expected 2 columns, got %d", layout.ColumnCount())
	}

	if layout.IsSingleColumn() {
		t.Error("should not be single column")
	}

	if !layout.IsMultiColumn() {
		t.Error("expected multi-column layout")
	}

	// Verify columns
	leftCol := layout.GetColumn(0)
	rightCol := layout.GetColumn(1)

	if leftCol == nil || rightCol == nil {
		t.Fatal("expected both columns to exist")
	}

	// Left column should have fragments with lower X values
	if len(leftCol.Fragments) != 4 {
		t.Errorf("left column should have 4 fragments, got %d", len(leftCol.Fragments))
	}

	if len(rightCol.Fragments) != 4 {
		t.Errorf("right column should have 4 fragments, got %d", len(rightCol.Fragments))
	}

	// Verify reading order within left column (top to bottom)
	if leftCol.Fragments[0].Text != "Left Column Title" {
		t.Errorf("left column first fragment should be title, got %q", leftCol.Fragments[0].Text)
	}

	// Verify reading order within right column (top to bottom)
	if rightCol.Fragments[0].Text != "Right Column Title" {
		t.Errorf("right column first fragment should be title, got %q", rightCol.Fragments[0].Text)
	}
}

func TestColumnDetector_ThreeColumns(t *testing.T) {
	detector := NewColumnDetector()

	// Three-column newsletter style layout
	// Column 1: x=50 to x=180
	// Gap: x=180 to x=210 (30 points)
	// Column 2: x=210 to x=340
	// Gap: x=340 to x=370 (30 points)
	// Column 3: x=370 to x=500
	fragments := []text.TextFragment{
		// Column 1
		makeFragment(50, 700, 130, 12, "Col 1 Title"),
		makeFragment(50, 680, 130, 10, "Col 1 text."),
		// Column 2
		makeFragment(210, 700, 130, 12, "Col 2 Title"),
		makeFragment(210, 680, 130, 10, "Col 2 text."),
		// Column 3
		makeFragment(370, 700, 130, 12, "Col 3 Title"),
		makeFragment(370, 680, 130, 10, "Col 3 text."),
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.ColumnCount() != 3 {
		t.Errorf("expected 3 columns, got %d", layout.ColumnCount())
	}

	// Verify each column has correct content
	for i := 0; i < 3; i++ {
		col := layout.GetColumn(i)
		if col == nil {
			t.Errorf("column %d should exist", i)
			continue
		}
		if len(col.Fragments) != 2 {
			t.Errorf("column %d should have 2 fragments, got %d", i, len(col.Fragments))
		}
	}
}

func TestColumnDetector_GetText(t *testing.T) {
	detector := NewColumnDetector()

	// Two-column layout
	fragments := []text.TextFragment{
		// Left column
		makeFragment(72, 700, 200, 12, "Left Title"),
		makeFragment(72, 680, 200, 10, "Left content."),
		// Right column
		makeFragment(322, 700, 200, 12, "Right Title"),
		makeFragment(322, 680, 200, 10, "Right content."),
	}

	layout := detector.Detect(fragments, 612, 792)
	text := layout.GetText()

	// Text should read left column first, then right column
	if !strings.Contains(text, "Left Title") {
		t.Error("text should contain Left Title")
	}
	if !strings.Contains(text, "Right Title") {
		t.Error("text should contain Right Title")
	}

	// Left content should appear before right content
	leftIdx := strings.Index(text, "Left Title")
	rightIdx := strings.Index(text, "Right Title")

	if leftIdx > rightIdx {
		t.Error("left column content should appear before right column content")
	}
}

func TestColumnDetector_NarrowGap(t *testing.T) {
	detector := NewColumnDetector()

	// Text with narrow gap (less than MinGapWidth) - should be single column
	fragments := []text.TextFragment{
		makeFragment(72, 700, 200, 12, "First part"),
		makeFragment(282, 700, 200, 12, "Second part"), // Only 10 point gap
	}

	layout := detector.Detect(fragments, 612, 792)

	// With default MinGapWidth of 20, this should be single column
	if layout.ColumnCount() != 1 {
		t.Errorf("expected 1 column with narrow gap, got %d", layout.ColumnCount())
	}
}

func TestColumnDetector_WideGap(t *testing.T) {
	detector := NewColumnDetector()

	// Text with wide gap (more than MinGapWidth) - should be two columns
	fragments := []text.TextFragment{
		makeFragment(72, 700, 200, 12, "First part"),
		makeFragment(322, 700, 200, 12, "Second part"), // 50 point gap
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.ColumnCount() != 2 {
		t.Errorf("expected 2 columns with wide gap, got %d", layout.ColumnCount())
	}
}

func TestColumnDetector_PartialVerticalGap(t *testing.T) {
	detector := NewColumnDetector()

	// Gap that only exists for part of the page (should not create columns)
	fragments := []text.TextFragment{
		// Full-width header
		makeFragment(72, 750, 468, 12, "Full Width Header"),
		// Two-column body
		makeFragment(72, 700, 200, 10, "Left body"),
		makeFragment(322, 700, 200, 10, "Right body"),
		// Full-width footer
		makeFragment(72, 600, 468, 10, "Full Width Footer"),
	}

	layout := detector.Detect(fragments, 612, 792)

	// The gap is blocked by header and footer, so gap extent is low
	// With MinGapHeightRatio of 0.5, this might still be single column
	// depending on exact calculations
	t.Logf("Detected %d columns", layout.ColumnCount())
}

func TestColumnDetector_CustomConfig(t *testing.T) {
	config := ColumnConfig{
		MinColumnWidth:    30.0,  // Allow narrower columns
		MinGapWidth:       10.0,  // Allow narrower gaps
		MinGapHeightRatio: 0.3,   // Lower vertical threshold
		MaxColumns:        4,     // Limit to 4 columns
		MergeThreshold:    5.0,   // Tighter merging
	}

	detector := NewColumnDetectorWithConfig(config)

	// With narrower gap threshold, this should be two columns
	fragments := []text.TextFragment{
		makeFragment(72, 700, 200, 12, "First part"),
		makeFragment(287, 700, 200, 12, "Second part"), // 15 point gap
	}

	layout := detector.Detect(fragments, 612, 792)

	if layout.ColumnCount() != 2 {
		t.Errorf("expected 2 columns with custom config, got %d", layout.ColumnCount())
	}
}

func TestGap_Methods(t *testing.T) {
	gap := Gap{
		Left:   100,
		Right:  150,
		Top:    500,
		Bottom: 100,
	}

	if gap.Width() != 50 {
		t.Errorf("expected width 50, got %f", gap.Width())
	}

	if gap.Height() != 400 {
		t.Errorf("expected height 400, got %f", gap.Height())
	}

	if gap.Center() != 125 {
		t.Errorf("expected center 125, got %f", gap.Center())
	}
}

func TestColumnLayout_GetColumn_OutOfBounds(t *testing.T) {
	detector := NewColumnDetector()

	fragments := []text.TextFragment{
		makeFragment(72, 700, 468, 12, "Single column"),
	}

	layout := detector.Detect(fragments, 612, 792)

	// Valid index
	col := layout.GetColumn(0)
	if col == nil {
		t.Error("expected column 0 to exist")
	}

	// Invalid indices
	if layout.GetColumn(-1) != nil {
		t.Error("expected nil for negative index")
	}

	if layout.GetColumn(10) != nil {
		t.Error("expected nil for out of bounds index")
	}
}

func TestColumnLayout_NilReceiver(t *testing.T) {
	var layout *ColumnLayout

	// These should not panic
	if layout.ColumnCount() != 0 {
		t.Error("nil layout should have 0 columns")
	}

	if layout.GetColumn(0) != nil {
		t.Error("nil layout GetColumn should return nil")
	}

	if layout.GetFragmentsInReadingOrder() != nil {
		t.Error("nil layout GetFragmentsInReadingOrder should return nil")
	}
}

func TestFragmentsBBox(t *testing.T) {
	fragments := []text.TextFragment{
		makeFragment(100, 200, 50, 10, "A"),
		makeFragment(200, 300, 60, 12, "B"),
		makeFragment(50, 150, 40, 8, "C"),
	}

	bbox := fragmentsBBox(fragments)

	// Min X should be 50 (fragment C)
	if bbox.X != 50 {
		t.Errorf("expected X=50, got %f", bbox.X)
	}

	// Min Y should be 150 (fragment C)
	if bbox.Y != 150 {
		t.Errorf("expected Y=150, got %f", bbox.Y)
	}

	// Max X should be 260 (fragment B: 200 + 60)
	expectedWidth := 260 - 50 // 210
	if bbox.Width != float64(expectedWidth) {
		t.Errorf("expected Width=%d, got %f", expectedWidth, bbox.Width)
	}

	// Max Y should be 312 (fragment B: 300 + 12)
	expectedHeight := 312 - 150 // 162
	if bbox.Height != float64(expectedHeight) {
		t.Errorf("expected Height=%d, got %f", expectedHeight, bbox.Height)
	}
}

func TestFragmentsBBox_Empty(t *testing.T) {
	bbox := fragmentsBBox(nil)

	if bbox.X != 0 || bbox.Y != 0 || bbox.Width != 0 || bbox.Height != 0 {
		t.Error("empty fragments should return zero bbox")
	}
}

func TestColumnDetector_AcademicPaperStyle(t *testing.T) {
	detector := NewColumnDetector()

	// Simulate a typical academic paper layout:
	// - Title (full width)
	// - Authors (full width)
	// - Abstract (full width)
	// - Two-column body
	fragments := []text.TextFragment{
		// Full-width title
		makeFragment(100, 750, 400, 16, "Research Paper Title"),
		// Full-width authors
		makeFragment(150, 720, 300, 10, "Author One, Author Two"),
		// Full-width abstract
		makeFragment(100, 680, 400, 10, "Abstract: This is the abstract text."),
		// Two-column body
		// Left column
		makeFragment(100, 600, 180, 10, "1. Introduction"),
		makeFragment(100, 580, 180, 10, "Body text left col."),
		makeFragment(100, 560, 180, 10, "More left text."),
		// Right column
		makeFragment(320, 600, 180, 10, "2. Methods"),
		makeFragment(320, 580, 180, 10, "Body text right col."),
		makeFragment(320, 560, 180, 10, "More right text."),
	}

	layout := detector.Detect(fragments, 612, 792)

	// This is complex - the header section might block the gap
	// The result depends on MinGapHeightRatio
	t.Logf("Academic paper style: detected %d columns", layout.ColumnCount())
	t.Logf("Layout text:\n%s", layout.GetText())
}

func TestColumnDetector_ReadingOrderPreservation(t *testing.T) {
	detector := NewColumnDetector()

	// Two columns with numbered paragraphs
	fragments := []text.TextFragment{
		// Left column (should be read first)
		makeFragment(72, 700, 200, 10, "1. First point"),
		makeFragment(72, 680, 200, 10, "2. Second point"),
		makeFragment(72, 660, 200, 10, "3. Third point"),
		// Right column (should be read second)
		makeFragment(322, 700, 200, 10, "4. Fourth point"),
		makeFragment(322, 680, 200, 10, "5. Fifth point"),
		makeFragment(322, 660, 200, 10, "6. Sixth point"),
	}

	layout := detector.Detect(fragments, 612, 792)
	orderedFrags := layout.GetFragmentsInReadingOrder()

	if len(orderedFrags) != 6 {
		t.Fatalf("expected 6 fragments, got %d", len(orderedFrags))
	}

	// Verify order: 1, 2, 3, 4, 5, 6
	expectedOrder := []string{
		"1. First point",
		"2. Second point",
		"3. Third point",
		"4. Fourth point",
		"5. Fifth point",
		"6. Sixth point",
	}

	for i, expected := range expectedOrder {
		if orderedFrags[i].Text != expected {
			t.Errorf("position %d: expected %q, got %q", i, expected, orderedFrags[i].Text)
		}
	}
}

// Benchmark column detection
func BenchmarkColumnDetector_TwoColumns(b *testing.B) {
	detector := NewColumnDetector()

	// Create 100 fragments in two columns
	var fragments []text.TextFragment
	for i := 0; i < 50; i++ {
		y := 700 - float64(i)*10
		fragments = append(fragments,
			makeFragment(72, y, 200, 10, "Left column text"),
			makeFragment(322, y, 200, 10, "Right column text"),
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(fragments, 612, 792)
	}
}

func BenchmarkColumnDetector_SingleColumn(b *testing.B) {
	detector := NewColumnDetector()

	// Create 100 fragments in single column
	var fragments []text.TextFragment
	for i := 0; i < 100; i++ {
		y := 700 - float64(i)*7
		fragments = append(fragments, makeFragment(72, y, 468, 10, "Full width text line"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(fragments, 612, 792)
	}
}
