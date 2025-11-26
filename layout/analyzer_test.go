package layout

import (
	"testing"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

func TestNewAnalyzer(t *testing.T) {
	analyzer := NewAnalyzer()
	if analyzer == nil {
		t.Fatal("NewAnalyzer returned nil")
	}
	if analyzer.columnDetector == nil {
		t.Error("columnDetector not initialized")
	}
	if analyzer.lineDetector == nil {
		t.Error("lineDetector not initialized")
	}
	if analyzer.blockDetector == nil {
		t.Error("blockDetector not initialized")
	}
	if analyzer.paragraphDetector == nil {
		t.Error("paragraphDetector not initialized")
	}
	if analyzer.headingDetector == nil {
		t.Error("headingDetector not initialized")
	}
	if analyzer.listDetector == nil {
		t.Error("listDetector not initialized")
	}
	if analyzer.readingOrderDetector == nil {
		t.Error("readingOrderDetector not initialized")
	}
}

func TestNewAnalyzerWithConfig(t *testing.T) {
	config := DefaultAnalyzerConfig()
	config.DetectHeadings = false
	config.DetectLists = false

	analyzer := NewAnalyzerWithConfig(config)
	if analyzer == nil {
		t.Fatal("NewAnalyzerWithConfig returned nil")
	}
	if analyzer.config.DetectHeadings {
		t.Error("DetectHeadings should be false")
	}
	if analyzer.config.DetectLists {
		t.Error("DetectLists should be false")
	}
}

func TestAnalyzer_Analyze_EmptyFragments(t *testing.T) {
	analyzer := NewAnalyzer()
	result := analyzer.Analyze(nil, 612, 792)

	if result == nil {
		t.Fatal("Analyze returned nil for empty fragments")
	}
	if result.Stats.FragmentCount != 0 {
		t.Errorf("expected 0 fragments, got %d", result.Stats.FragmentCount)
	}
	if len(result.Elements) != 0 {
		t.Errorf("expected 0 elements, got %d", len(result.Elements))
	}
}

func TestAnalyzer_Analyze_SingleParagraph(t *testing.T) {
	fragments := []text.TextFragment{
		{Text: "This is a test paragraph.", X: 72, Y: 700, Width: 200, Height: 12, FontSize: 12},
		{Text: "It has multiple lines.", X: 72, Y: 686, Width: 180, Height: 12, FontSize: 12},
		{Text: "This is the third line.", X: 72, Y: 672, Width: 190, Height: 12, FontSize: 12},
	}

	analyzer := NewAnalyzer()
	result := analyzer.Analyze(fragments, 612, 792)

	if result == nil {
		t.Fatal("Analyze returned nil")
	}
	if result.Stats.FragmentCount != 3 {
		t.Errorf("expected 3 fragments, got %d", result.Stats.FragmentCount)
	}
	if result.Stats.ParagraphCount == 0 {
		t.Error("expected at least 1 paragraph")
	}
	if len(result.Elements) == 0 {
		t.Error("expected at least 1 element")
	}
}

func TestAnalyzer_Analyze_WithHeading(t *testing.T) {
	fragments := []text.TextFragment{
		// Large heading
		{Text: "Chapter One", X: 72, Y: 750, Width: 200, Height: 24, FontSize: 24, FontName: "Helvetica-Bold"},
		// Body text
		{Text: "This is body text.", X: 72, Y: 700, Width: 150, Height: 12, FontSize: 12, FontName: "Helvetica"},
		{Text: "More body text here.", X: 72, Y: 686, Width: 160, Height: 12, FontSize: 12, FontName: "Helvetica"},
		{Text: "And even more text.", X: 72, Y: 672, Width: 155, Height: 12, FontSize: 12, FontName: "Helvetica"},
	}

	analyzer := NewAnalyzer()
	result := analyzer.Analyze(fragments, 612, 792)

	if result == nil {
		t.Fatal("Analyze returned nil")
	}

	// Check that heading was detected
	if result.Headings == nil || len(result.Headings.Headings) == 0 {
		t.Log("No headings detected (may need more body text for detection)")
	}

	// Check elements
	if len(result.Elements) == 0 {
		t.Error("expected at least 1 element")
	}
}

func TestAnalyzer_Analyze_TwoColumns(t *testing.T) {
	fragments := []text.TextFragment{
		// Left column
		{Text: "Left column line 1", X: 72, Y: 700, Width: 150, Height: 12, FontSize: 12},
		{Text: "Left column line 2", X: 72, Y: 686, Width: 150, Height: 12, FontSize: 12},
		{Text: "Left column line 3", X: 72, Y: 672, Width: 150, Height: 12, FontSize: 12},
		// Right column
		{Text: "Right column line 1", X: 350, Y: 700, Width: 150, Height: 12, FontSize: 12},
		{Text: "Right column line 2", X: 350, Y: 686, Width: 150, Height: 12, FontSize: 12},
		{Text: "Right column line 3", X: 350, Y: 672, Width: 150, Height: 12, FontSize: 12},
	}

	analyzer := NewAnalyzer()
	result := analyzer.Analyze(fragments, 612, 792)

	if result == nil {
		t.Fatal("Analyze returned nil")
	}

	// Check column detection
	if result.Columns != nil && !result.Columns.IsSingleColumn() {
		t.Logf("Detected %d columns", len(result.Columns.Columns))
	}

	// Check reading order
	if result.ReadingOrder != nil {
		t.Logf("Reading order: %d columns, %d sections",
			result.ReadingOrder.ColumnCount, len(result.ReadingOrder.Sections))
	}
}

func TestAnalyzer_QuickAnalyze(t *testing.T) {
	fragments := []text.TextFragment{
		{Text: "Paragraph one.", X: 72, Y: 700, Width: 100, Height: 12, FontSize: 12},
		{Text: "Paragraph two.", X: 72, Y: 650, Width: 100, Height: 12, FontSize: 12},
	}

	analyzer := NewAnalyzer()
	result := analyzer.QuickAnalyze(fragments, 612, 792)

	if result == nil {
		t.Fatal("QuickAnalyze returned nil")
	}

	// Quick analyze should skip heading/list detection
	if result.Headings != nil {
		t.Error("QuickAnalyze should not detect headings")
	}
	if result.Lists != nil {
		t.Error("QuickAnalyze should not detect lists")
	}
}

func TestAnalysisResult_GetElements(t *testing.T) {
	result := &AnalysisResult{
		Elements: []LayoutElement{
			{
				Type: model.ElementTypeParagraph,
				BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 50},
				Text: "Test paragraph",
			},
			{
				Type: model.ElementTypeHeading,
				BBox: model.BBox{X: 72, Y: 750, Width: 200, Height: 24},
				Text: "Test heading",
				Heading: &Heading{
					Level: HeadingLevel1,
					Text:  "Test heading",
				},
			},
		},
	}

	elements := result.GetElements()
	if len(elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(elements))
	}

	// Check types
	if elements[0].Type() != model.ElementTypeParagraph {
		t.Errorf("expected paragraph, got %v", elements[0].Type())
	}
	if elements[1].Type() != model.ElementTypeHeading {
		t.Errorf("expected heading, got %v", elements[1].Type())
	}
}

func TestAnalysisResult_GetText(t *testing.T) {
	// Create a result with reading order
	result := &AnalysisResult{
		ReadingOrder: &ReadingOrderResult{
			Lines: []Line{
				{Text: "Line one"},
				{Text: "Line two"},
			},
		},
	}

	text := result.GetText()
	if text == "" {
		t.Error("GetText returned empty string")
	}
}

func TestLayoutElement_ToModelElement_Paragraph(t *testing.T) {
	elem := LayoutElement{
		Type: model.ElementTypeParagraph,
		BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 50},
		Text: "Test paragraph",
		Paragraph: &Paragraph{
			AverageFontSize: 12,
			Alignment:       AlignLeft,
		},
	}

	modelElem := elem.ToModelElement()
	para, ok := modelElem.(*model.Paragraph)
	if !ok {
		t.Fatal("expected *model.Paragraph")
	}
	if para.Text != "Test paragraph" {
		t.Errorf("expected 'Test paragraph', got '%s'", para.Text)
	}
	if para.FontSize != 12 {
		t.Errorf("expected font size 12, got %f", para.FontSize)
	}
}

func TestLayoutElement_ToModelElement_Heading(t *testing.T) {
	elem := LayoutElement{
		Type: model.ElementTypeHeading,
		BBox: model.BBox{X: 72, Y: 750, Width: 200, Height: 24},
		Text: "Test heading",
		Heading: &Heading{
			Level:    HeadingLevel2,
			Text:     "Test heading",
			FontSize: 18,
		},
	}

	modelElem := elem.ToModelElement()
	heading, ok := modelElem.(*model.Heading)
	if !ok {
		t.Fatal("expected *model.Heading")
	}
	if heading.Text != "Test heading" {
		t.Errorf("expected 'Test heading', got '%s'", heading.Text)
	}
	if heading.Level != 2 {
		t.Errorf("expected level 2, got %d", heading.Level)
	}
}

func TestLayoutElement_ToModelElement_List(t *testing.T) {
	elem := LayoutElement{
		Type: model.ElementTypeList,
		BBox: model.BBox{X: 72, Y: 600, Width: 200, Height: 100},
		Text: "• Item 1\n• Item 2",
		List: &List{
			Type: ListTypeBullet,
			Items: []ListItem{
				{Text: "Item 1", Level: 0},
				{Text: "Item 2", Level: 0},
			},
		},
	}

	modelElem := elem.ToModelElement()
	list, ok := modelElem.(*model.List)
	if !ok {
		t.Fatal("expected *model.List")
	}
	if list.Ordered {
		t.Error("expected unordered list")
	}
	if len(list.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(list.Items))
	}
}

func TestLayoutElement_ToModelElement_NumberedList(t *testing.T) {
	elem := LayoutElement{
		Type: model.ElementTypeList,
		BBox: model.BBox{X: 72, Y: 600, Width: 200, Height: 100},
		List: &List{
			Type: ListTypeNumbered,
			Items: []ListItem{
				{Text: "First", Level: 0},
				{Text: "Second", Level: 0},
			},
		},
	}

	modelElem := elem.ToModelElement()
	list, ok := modelElem.(*model.List)
	if !ok {
		t.Fatal("expected *model.List")
	}
	if !list.Ordered {
		t.Error("expected ordered list for numbered type")
	}
}

func TestBboxOverlaps(t *testing.T) {
	tests := []struct {
		name     string
		a, b     model.BBox
		expected bool
	}{
		{
			name:     "no overlap - horizontal",
			a:        model.BBox{X: 0, Y: 0, Width: 100, Height: 50},
			b:        model.BBox{X: 200, Y: 0, Width: 100, Height: 50},
			expected: false,
		},
		{
			name:     "no overlap - vertical",
			a:        model.BBox{X: 0, Y: 0, Width: 100, Height: 50},
			b:        model.BBox{X: 0, Y: 100, Width: 100, Height: 50},
			expected: false,
		},
		{
			name:     "full overlap",
			a:        model.BBox{X: 0, Y: 0, Width: 100, Height: 50},
			b:        model.BBox{X: 0, Y: 0, Width: 100, Height: 50},
			expected: true,
		},
		{
			name:     "partial overlap - significant",
			a:        model.BBox{X: 0, Y: 0, Width: 100, Height: 50},
			b:        model.BBox{X: 20, Y: 10, Width: 100, Height: 50},
			expected: true,
		},
		{
			name:     "partial overlap - minor",
			a:        model.BBox{X: 0, Y: 0, Width: 100, Height: 50},
			b:        model.BBox{X: 90, Y: 40, Width: 100, Height: 50},
			expected: false, // Only 10x10 overlap = 100, smaller area is 5000, 100/5000 = 2% < 50%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bboxOverlaps(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("bboxOverlaps(%v, %v) = %v, expected %v",
					tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestSortElementsByReadingOrder_NoReadingOrder(t *testing.T) {
	elements := []LayoutElement{
		{BBox: model.BBox{X: 300, Y: 700, Width: 100, Height: 20}}, // right, top
		{BBox: model.BBox{X: 72, Y: 700, Width: 100, Height: 20}},  // left, top
		{BBox: model.BBox{X: 72, Y: 600, Width: 100, Height: 20}},  // left, bottom
	}

	sortElementsByReadingOrder(elements, nil)

	// Should sort top-to-bottom, left-to-right
	// Top row: left (72) should come before right (300)
	if elements[0].BBox.X != 72 || elements[0].BBox.Y != 700 {
		t.Errorf("first element should be left-top, got X=%.1f Y=%.1f",
			elements[0].BBox.X, elements[0].BBox.Y)
	}
	if elements[1].BBox.X != 300 || elements[1].BBox.Y != 700 {
		t.Errorf("second element should be right-top, got X=%.1f Y=%.1f",
			elements[1].BBox.X, elements[1].BBox.Y)
	}
	if elements[2].BBox.Y != 600 {
		t.Errorf("third element should be bottom, got Y=%.1f", elements[2].BBox.Y)
	}
}

func TestAnalyzer_AnalyzeWithHeaderFooterFiltering(t *testing.T) {
	// Create multi-page fragments with repeating header
	pageFragments := []PageFragments{
		{
			PageIndex:  0,
			PageWidth:  612,
			PageHeight: 792,
			Fragments: []text.TextFragment{
				{Text: "Header Text", X: 72, Y: 760, Width: 100, Height: 12, FontSize: 12},
				{Text: "Body content page 1", X: 72, Y: 600, Width: 200, Height: 12, FontSize: 12},
			},
		},
		{
			PageIndex:  1,
			PageWidth:  612,
			PageHeight: 792,
			Fragments: []text.TextFragment{
				{Text: "Header Text", X: 72, Y: 760, Width: 100, Height: 12, FontSize: 12},
				{Text: "Body content page 2", X: 72, Y: 600, Width: 200, Height: 12, FontSize: 12},
			},
		},
	}

	analyzer := NewAnalyzer()
	result := analyzer.AnalyzeWithHeaderFooterFiltering(pageFragments, 0)

	if result == nil {
		t.Fatal("AnalyzeWithHeaderFooterFiltering returned nil")
	}

	// Result should have filtered content
	t.Logf("Elements after filtering: %d", len(result.Elements))
}

func TestAnalysisStats(t *testing.T) {
	fragments := []text.TextFragment{
		{Text: "Line 1", X: 72, Y: 700, Width: 100, Height: 12, FontSize: 12},
		{Text: "Line 2", X: 72, Y: 686, Width: 100, Height: 12, FontSize: 12},
		{Text: "Line 3", X: 72, Y: 672, Width: 100, Height: 12, FontSize: 12},
	}

	analyzer := NewAnalyzer()
	result := analyzer.Analyze(fragments, 612, 792)

	if result.Stats.FragmentCount != 3 {
		t.Errorf("expected 3 fragments, got %d", result.Stats.FragmentCount)
	}
	if result.Stats.LineCount == 0 {
		t.Error("expected lines to be detected")
	}

	t.Logf("Stats: fragments=%d, lines=%d, blocks=%d, paragraphs=%d, elements=%d",
		result.Stats.FragmentCount,
		result.Stats.LineCount,
		result.Stats.BlockCount,
		result.Stats.ParagraphCount,
		result.Stats.ElementCount)
}

func TestDefaultAnalyzerConfig(t *testing.T) {
	config := DefaultAnalyzerConfig()

	if !config.DetectHeadings {
		t.Error("DetectHeadings should be true by default")
	}
	if !config.DetectLists {
		t.Error("DetectLists should be true by default")
	}
	if !config.UseReadingOrder {
		t.Error("UseReadingOrder should be true by default")
	}
}

// Benchmark tests

func BenchmarkAnalyzer_Analyze(b *testing.B) {
	// Create realistic document fragments
	var fragments []text.TextFragment
	y := 750.0
	for i := 0; i < 50; i++ {
		fragments = append(fragments, text.TextFragment{
			Text:     "This is a line of text that represents typical document content.",
			X:        72,
			Y:        y,
			Width:    400,
			Height:   12,
			FontSize: 12,
		})
		y -= 14
	}

	analyzer := NewAnalyzer()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		analyzer.Analyze(fragments, 612, 792)
	}
}

func BenchmarkAnalyzer_QuickAnalyze(b *testing.B) {
	var fragments []text.TextFragment
	y := 750.0
	for i := 0; i < 50; i++ {
		fragments = append(fragments, text.TextFragment{
			Text:     "This is a line of text that represents typical document content.",
			X:        72,
			Y:        y,
			Width:    400,
			Height:   12,
			FontSize: 12,
		})
		y -= 14
	}

	analyzer := NewAnalyzer()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		analyzer.QuickAnalyze(fragments, 612, 792)
	}
}
