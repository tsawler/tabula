package layout

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// makeROFragment creates a text fragment for reading order tests
func makeROFragment(t string, x, y, w, h, fs float64) text.TextFragment {
	return text.TextFragment{
		Text:     t,
		X:        x,
		Y:        y,
		Width:    w,
		Height:   h,
		FontSize: fs,
	}
}

func TestNewReadingOrderDetector(t *testing.T) {
	detector := NewReadingOrderDetector()
	if detector == nil {
		t.Fatal("NewReadingOrderDetector returned nil")
	}
}

func TestNewReadingOrderDetectorWithConfig(t *testing.T) {
	config := ReadingOrderConfig{
		Direction:         RightToLeft,
		PreferColumnOrder: false,
		SpanningThreshold: 0.8,
	}
	detector := NewReadingOrderDetectorWithConfig(config)
	if detector == nil {
		t.Fatal("NewReadingOrderDetectorWithConfig returned nil")
	}
	if detector.config.Direction != RightToLeft {
		t.Errorf("Expected RTL direction, got %v", detector.config.Direction)
	}
}

func TestDefaultReadingOrderConfig(t *testing.T) {
	config := DefaultReadingOrderConfig()
	if config.Direction != LeftToRight {
		t.Errorf("Expected LTR direction, got %v", config.Direction)
	}
	if config.SpanningThreshold != 0.7 {
		t.Errorf("Expected spanning threshold 0.7, got %v", config.SpanningThreshold)
	}
	if !config.PreferColumnOrder {
		t.Error("Expected PreferColumnOrder to be true")
	}
}

func TestReadingDirectionString(t *testing.T) {
	tests := []struct {
		dir      ReadingDirection
		expected string
	}{
		{LeftToRight, "ltr"},
		{RightToLeft, "rtl"},
		{TopToBottom, "ttb"},
	}

	for _, tt := range tests {
		if got := tt.dir.String(); got != tt.expected {
			t.Errorf("ReadingDirection(%d).String() = %q, want %q", tt.dir, got, tt.expected)
		}
	}
}

func TestSectionTypeString(t *testing.T) {
	tests := []struct {
		st       SectionType
		expected string
	}{
		{SectionSpanning, "spanning"},
		{SectionColumn, "column"},
	}

	for _, tt := range tests {
		if got := tt.st.String(); got != tt.expected {
			t.Errorf("SectionType(%d).String() = %q, want %q", tt.st, got, tt.expected)
		}
	}
}

func TestDetectEmptyFragments(t *testing.T) {
	detector := NewReadingOrderDetector()
	result := detector.Detect([]text.TextFragment{}, 612, 792)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(result.Fragments) != 0 {
		t.Errorf("Expected 0 fragments, got %d", len(result.Fragments))
	}
	if len(result.Lines) != 0 {
		t.Errorf("Expected 0 lines, got %d", len(result.Lines))
	}
	if result.PageWidth != 612 {
		t.Errorf("Expected page width 612, got %f", result.PageWidth)
	}
	if result.PageHeight != 792 {
		t.Errorf("Expected page height 792, got %f", result.PageHeight)
	}
}

func TestDetectSingleColumn(t *testing.T) {
	// Single column - all fragments in one area
	fragments := []text.TextFragment{
		makeROFragment("Line 1", 72, 700, 200, 12, 12),
		makeROFragment("Line 2", 72, 680, 200, 12, 12),
		makeROFragment("Line 3", 72, 660, 200, 12, 12),
	}

	detector := NewReadingOrderDetector()
	result := detector.Detect(fragments, 612, 792)

	if result.ColumnCount != 1 {
		t.Errorf("Expected 1 column, got %d", result.ColumnCount)
	}
	if result.IsMultiColumn() {
		t.Error("Expected IsMultiColumn() to return false")
	}
}

func TestDetectTwoColumns(t *testing.T) {
	// Two column layout - left column and right column
	fragments := []text.TextFragment{
		// Left column
		makeROFragment("Left 1", 50, 700, 200, 12, 12),
		makeROFragment("Left 2", 50, 680, 200, 12, 12),
		makeROFragment("Left 3", 50, 660, 200, 12, 12),
		// Right column
		makeROFragment("Right 1", 350, 700, 200, 12, 12),
		makeROFragment("Right 2", 350, 680, 200, 12, 12),
		makeROFragment("Right 3", 350, 660, 200, 12, 12),
	}

	detector := NewReadingOrderDetector()
	result := detector.Detect(fragments, 612, 792)

	if result.ColumnCount < 2 {
		t.Errorf("Expected at least 2 columns, got %d", result.ColumnCount)
	}
	if !result.IsMultiColumn() {
		t.Error("Expected IsMultiColumn() to return true")
	}

	// In LTR reading order, left column should come first
	// Check that "Left" fragments appear before "Right" fragments
	leftFound := false
	rightFound := false
	leftBeforeRight := true

	for _, frag := range result.Fragments {
		if strings.HasPrefix(frag.Text, "Left") {
			leftFound = true
			if rightFound {
				leftBeforeRight = false
			}
		}
		if strings.HasPrefix(frag.Text, "Right") {
			rightFound = true
		}
	}

	if !leftFound || !rightFound {
		t.Error("Expected to find both Left and Right fragments")
	}
	if !leftBeforeRight {
		t.Error("Expected Left column fragments before Right column in LTR order")
	}
}

func TestDetectTwoColumnsRTL(t *testing.T) {
	// Two column layout with RTL reading direction
	fragments := []text.TextFragment{
		// Left column
		makeROFragment("Left 1", 50, 700, 200, 12, 12),
		makeROFragment("Left 2", 50, 680, 200, 12, 12),
		// Right column
		makeROFragment("Right 1", 350, 700, 200, 12, 12),
		makeROFragment("Right 2", 350, 680, 200, 12, 12),
	}

	// Set RTL reading direction
	config := DefaultReadingOrderConfig()
	config.Direction = RightToLeft
	detector := NewReadingOrderDetectorWithConfig(config)
	result := detector.Detect(fragments, 612, 792)

	// In RTL reading order, right column should come first
	rightFound := false
	leftFound := false
	rightBeforeLeft := true

	for _, frag := range result.Fragments {
		if strings.HasPrefix(frag.Text, "Right") {
			rightFound = true
			if leftFound {
				rightBeforeLeft = false
			}
		}
		if strings.HasPrefix(frag.Text, "Left") {
			leftFound = true
		}
	}

	if !rightFound || !leftFound {
		t.Error("Expected to find both Right and Left fragments")
	}
	if !rightBeforeLeft {
		t.Error("Expected Right column fragments before Left column in RTL order")
	}
}

func TestDetectSpanningContent(t *testing.T) {
	// Layout with spanning title and two columns below
	fragments := []text.TextFragment{
		// Spanning title at top
		makeROFragment("Main Title of the Document", 100, 750, 400, 18, 18),
		// Left column
		makeROFragment("Left content 1", 50, 650, 200, 12, 12),
		makeROFragment("Left content 2", 50, 630, 200, 12, 12),
		// Right column
		makeROFragment("Right content 1", 350, 650, 200, 12, 12),
		makeROFragment("Right content 2", 350, 630, 200, 12, 12),
	}

	detector := NewReadingOrderDetector()
	result := detector.Detect(fragments, 612, 792)

	// Title should come first in reading order
	if len(result.Fragments) < 5 {
		t.Fatalf("Expected at least 5 fragments, got %d", len(result.Fragments))
	}

	// Check sections
	if result.GetSectionCount() == 0 {
		t.Error("Expected at least one section")
	}
}

func TestDetectReadingDirectionAuto(t *testing.T) {
	// Test auto-detection of RTL content
	rtlFragments := []text.TextFragment{
		{Text: "שלום", X: 100, Y: 700, Width: 50, Height: 12, FontSize: 12, Direction: text.RTL},
		{Text: "עולם", X: 100, Y: 680, Width: 50, Height: 12, FontSize: 12, Direction: text.RTL},
		{Text: "test", X: 100, Y: 660, Width: 50, Height: 12, FontSize: 12, Direction: text.LTR},
	}

	detector := NewReadingOrderDetector()
	result := detector.Detect(rtlFragments, 612, 792)

	// Majority is RTL, so direction should be detected as RTL
	if result.Direction != RightToLeft {
		t.Errorf("Expected auto-detected RTL direction, got %v", result.Direction)
	}
}

func TestDetectReadingDirectionAutoLTR(t *testing.T) {
	// Test auto-detection with LTR majority
	ltrFragments := []text.TextFragment{
		{Text: "Hello", X: 100, Y: 700, Width: 50, Height: 12, FontSize: 12, Direction: text.LTR},
		{Text: "World", X: 100, Y: 680, Width: 50, Height: 12, FontSize: 12, Direction: text.LTR},
		{Text: "שלום", X: 100, Y: 660, Width: 50, Height: 12, FontSize: 12, Direction: text.RTL},
	}

	detector := NewReadingOrderDetector()
	result := detector.Detect(ltrFragments, 612, 792)

	// Majority is LTR
	if result.Direction != LeftToRight {
		t.Errorf("Expected auto-detected LTR direction, got %v", result.Direction)
	}
}

func TestReadingOrderResultGetText(t *testing.T) {
	fragments := []text.TextFragment{
		makeROFragment("First line", 72, 700, 200, 12, 12),
		makeROFragment("Second line", 72, 680, 200, 12, 12),
		makeROFragment("Third line", 72, 660, 200, 12, 12),
	}

	detector := NewReadingOrderDetector()
	result := detector.Detect(fragments, 612, 792)

	text := result.GetText()
	if !strings.Contains(text, "First") {
		t.Error("Expected text to contain 'First'")
	}
	if !strings.Contains(text, "Second") {
		t.Error("Expected text to contain 'Second'")
	}
}

func TestReadingOrderResultGetTextEmpty(t *testing.T) {
	result := &ReadingOrderResult{}
	text := result.GetText()
	if text != "" {
		t.Errorf("Expected empty text for empty result, got %q", text)
	}
}

func TestReadingOrderResultGetTextNil(t *testing.T) {
	var result *ReadingOrderResult
	text := result.GetText()
	if text != "" {
		t.Errorf("Expected empty text for nil result, got %q", text)
	}
}

func TestReadingOrderResultGetParagraphs(t *testing.T) {
	// Create fragments that form a paragraph
	fragments := []text.TextFragment{
		makeROFragment("Paragraph line one.", 72, 700, 200, 12, 12),
		makeROFragment("Paragraph line two.", 72, 685, 200, 12, 12),
		makeROFragment("Paragraph line three.", 72, 670, 200, 12, 12),
	}

	detector := NewReadingOrderDetector()
	result := detector.Detect(fragments, 612, 792)

	paras := result.GetParagraphs()
	if paras == nil {
		t.Fatal("Expected non-nil paragraph layout")
	}
}

func TestReadingOrderResultGetParagraphsEmpty(t *testing.T) {
	result := &ReadingOrderResult{
		PageWidth:  612,
		PageHeight: 792,
	}
	paras := result.GetParagraphs()
	if paras == nil {
		t.Fatal("Expected non-nil paragraph layout for empty result")
	}
}

func TestReadingOrderResultIsMultiColumn(t *testing.T) {
	tests := []struct {
		name        string
		columnCount int
		expected    bool
	}{
		{"single column", 1, false},
		{"two columns", 2, true},
		{"three columns", 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ReadingOrderResult{ColumnCount: tt.columnCount}
			if got := result.IsMultiColumn(); got != tt.expected {
				t.Errorf("IsMultiColumn() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReadingOrderResultIsMultiColumnNil(t *testing.T) {
	var result *ReadingOrderResult
	if result.IsMultiColumn() {
		t.Error("Expected IsMultiColumn() to return false for nil result")
	}
}

func TestReadingOrderResultGetSectionCount(t *testing.T) {
	result := &ReadingOrderResult{
		Sections: []ReadingSection{
			{Type: SectionSpanning},
			{Type: SectionColumn},
			{Type: SectionColumn},
		},
	}
	if got := result.GetSectionCount(); got != 3 {
		t.Errorf("GetSectionCount() = %d, want 3", got)
	}
}

func TestReadingOrderResultGetSectionCountNil(t *testing.T) {
	var result *ReadingOrderResult
	if got := result.GetSectionCount(); got != 0 {
		t.Errorf("GetSectionCount() = %d, want 0 for nil result", got)
	}
}

func TestDetectFromLines(t *testing.T) {
	// Create lines directly
	lines := []Line{
		{
			Text: "First line",
			BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 12},
			Fragments: []text.TextFragment{
				makeROFragment("First line", 72, 700, 200, 12, 12),
			},
		},
		{
			Text: "Second line",
			BBox: model.BBox{X: 72, Y: 680, Width: 200, Height: 12},
			Fragments: []text.TextFragment{
				makeROFragment("Second line", 72, 680, 200, 12, 12),
			},
		},
	}

	detector := NewReadingOrderDetector()
	result := detector.DetectFromLines(lines, 612, 792)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(result.Fragments) == 0 {
		t.Error("Expected fragments in result")
	}
}

func TestReorderForReading(t *testing.T) {
	// Two column fragments in scrambled order
	fragments := []text.TextFragment{
		makeROFragment("Right 1", 350, 700, 200, 12, 12),
		makeROFragment("Left 1", 50, 700, 200, 12, 12),
		makeROFragment("Right 2", 350, 680, 200, 12, 12),
		makeROFragment("Left 2", 50, 680, 200, 12, 12),
	}

	reordered := ReorderForReading(fragments, 612, 792)

	if len(reordered) != 4 {
		t.Fatalf("Expected 4 fragments, got %d", len(reordered))
	}
}

func TestReorderLinesForReading(t *testing.T) {
	lines := []Line{
		{
			Text: "Right column",
			BBox: model.BBox{X: 350, Y: 700, Width: 200, Height: 12},
			Fragments: []text.TextFragment{
				makeROFragment("Right column", 350, 700, 200, 12, 12),
			},
		},
		{
			Text: "Left column",
			BBox: model.BBox{X: 50, Y: 700, Width: 200, Height: 12},
			Fragments: []text.TextFragment{
				makeROFragment("Left column", 50, 700, 200, 12, 12),
			},
		},
	}

	reordered := ReorderLinesForReading(lines, 612, 792)

	if len(reordered) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(reordered))
	}
}

func TestCalculateAverageSpacing(t *testing.T) {
	tests := []struct {
		name     string
		lines    []Line
		expected float64
	}{
		{
			name:     "empty lines",
			lines:    []Line{},
			expected: 0,
		},
		{
			name:     "single line",
			lines:    []Line{{SpacingBefore: 10}},
			expected: 0, // Need at least 2 lines
		},
		{
			name: "multiple lines with spacing",
			lines: []Line{
				{SpacingBefore: 0},
				{SpacingBefore: 14},
				{SpacingBefore: 14},
				{SpacingBefore: 14},
			},
			expected: 14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateAverageSpacing(tt.lines)
			if got != tt.expected {
				t.Errorf("calculateAverageSpacing() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOrderSectionsVertical(t *testing.T) {
	// Test that sections are ordered top to bottom
	detector := NewReadingOrderDetector()
	sections := []ReadingSection{
		{
			Type: SectionColumn,
			BBox: struct{ X, Y, Width, Height float64 }{50, 400, 200, 100}, // Lower on page
		},
		{
			Type: SectionColumn,
			BBox: struct{ X, Y, Width, Height float64 }{50, 700, 200, 100}, // Higher on page
		},
	}

	detector.orderSections(sections, LeftToRight, false)

	// Higher Y (top of page) should come first
	if sections[0].BBox.Y < sections[1].BBox.Y {
		t.Error("Expected sections to be ordered top to bottom (higher Y first)")
	}
}

func TestOrderSectionsHorizontalLTR(t *testing.T) {
	// Test that sections at same Y level are ordered left to right
	detector := NewReadingOrderDetector()
	sections := []ReadingSection{
		{
			Type: SectionColumn,
			BBox: struct{ X, Y, Width, Height float64 }{350, 700, 200, 100}, // Right column
		},
		{
			Type: SectionColumn,
			BBox: struct{ X, Y, Width, Height float64 }{50, 700, 200, 100}, // Left column
		},
	}

	detector.orderSections(sections, LeftToRight, false)

	// In LTR, smaller X should come first
	if sections[0].BBox.X > sections[1].BBox.X {
		t.Error("Expected sections to be ordered left to right in LTR mode")
	}
}

func TestOrderSectionsHorizontalRTL(t *testing.T) {
	// Test that sections at same Y level are ordered right to left
	detector := NewReadingOrderDetector()
	sections := []ReadingSection{
		{
			Type: SectionColumn,
			BBox: struct{ X, Y, Width, Height float64 }{50, 700, 200, 100}, // Left column
		},
		{
			Type: SectionColumn,
			BBox: struct{ X, Y, Width, Height float64 }{350, 700, 200, 100}, // Right column
		},
	}

	detector.orderSections(sections, RightToLeft, false)

	// In RTL, larger X should come first
	if sections[0].BBox.X < sections[1].BBox.X {
		t.Error("Expected sections to be ordered right to left in RTL mode")
	}
}

func TestOrderSectionsSpanningFirst(t *testing.T) {
	// Test that spanning sections come before column sections at same level
	// In PDF coordinates, higher Y means higher on the page
	// Spanning section is at Y=780 (top = 800), Column at Y=700 (top = 800)
	// They're at the same level, spanning should come first
	detector := NewReadingOrderDetector()
	sections := []ReadingSection{
		{
			Type: SectionColumn,
			BBox: struct{ X, Y, Width, Height float64 }{50, 700, 200, 100}, // top = 800
		},
		{
			Type: SectionSpanning,
			BBox: struct{ X, Y, Width, Height float64 }{50, 780, 500, 20}, // top = 800
		},
	}

	detector.orderSections(sections, LeftToRight, false)

	// Spanning section should come first
	if sections[0].Type != SectionSpanning {
		t.Error("Expected spanning section to come before column section")
	}
}

func TestBuildColumnSection(t *testing.T) {
	detector := NewReadingOrderDetector()
	col := Column{
		Fragments: []text.TextFragment{
			makeROFragment("Line 1", 50, 700, 200, 12, 12),
			makeROFragment("Line 2", 50, 680, 200, 12, 12),
		},
		BBox: model.BBox{X: 50, Y: 680, Width: 200, Height: 32},
	}

	section := detector.buildColumnSection(col, 0, 792, false)

	if section.Type != SectionColumn {
		t.Errorf("Expected SectionColumn, got %v", section.Type)
	}
	if section.ColumnIndex != 0 {
		t.Errorf("Expected column index 0, got %d", section.ColumnIndex)
	}
	if len(section.Fragments) != 2 {
		t.Errorf("Expected 2 fragments, got %d", len(section.Fragments))
	}
}

func TestBuildSpanningSection(t *testing.T) {
	detector := NewReadingOrderDetector()
	fragments := []text.TextFragment{
		makeROFragment("Title", 100, 750, 400, 18, 18),
	}

	section := detector.buildSpanningSection(fragments, 612, 792, false)

	if section.Type != SectionSpanning {
		t.Errorf("Expected SectionSpanning, got %v", section.Type)
	}
	if section.ColumnIndex != -1 {
		t.Errorf("Expected column index -1 for spanning, got %d", section.ColumnIndex)
	}
	if len(section.Fragments) != 1 {
		t.Errorf("Expected 1 fragment, got %d", len(section.Fragments))
	}
}

// Benchmark tests
func BenchmarkReadingOrderDetect(b *testing.B) {
	// Create a realistic multi-column document
	var fragments []text.TextFragment

	// Title
	fragments = append(fragments, makeROFragment("Document Title", 150, 750, 300, 24, 24))

	// Two columns of 50 lines each
	for i := 0; i < 50; i++ {
		// Left column
		fragments = append(fragments, makeROFragment(
			"Left column line content",
			50, float64(700-i*14), 200, 12, 12,
		))
		// Right column
		fragments = append(fragments, makeROFragment(
			"Right column line content",
			350, float64(700-i*14), 200, 12, 12,
		))
	}

	detector := NewReadingOrderDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(fragments, 612, 792)
	}
}

func BenchmarkReadingOrderDetectSingleColumn(b *testing.B) {
	// Single column document
	var fragments []text.TextFragment
	for i := 0; i < 100; i++ {
		fragments = append(fragments, makeROFragment(
			"Single column line content",
			72, float64(700-i*14), 450, 12, 12,
		))
	}

	detector := NewReadingOrderDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(fragments, 612, 792)
	}
}
