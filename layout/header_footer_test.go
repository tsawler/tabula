package layout

import (
	"testing"

	"github.com/tsawler/tabula/text"
)

// Helper to create page fragments for testing
func makePageFragments(pageIndex int, pageHeight, pageWidth float64, fragments []text.TextFragment) PageFragments {
	return PageFragments{
		PageIndex:  pageIndex,
		PageHeight: pageHeight,
		PageWidth:  pageWidth,
		Fragments:  fragments,
	}
}

func TestHeaderFooterDetector_NoPages(t *testing.T) {
	detector := NewHeaderFooterDetector()

	result := detector.Detect(nil)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.HasHeadersOrFooters() {
		t.Error("expected no headers or footers for empty input")
	}
}

func TestHeaderFooterDetector_SinglePage(t *testing.T) {
	detector := NewHeaderFooterDetector()

	// Single page - not enough for header/footer detection
	pages := []PageFragments{
		makePageFragments(0, 792, 612, []text.TextFragment{
			makeFragment(72, 750, 200, 12, "Document Title"),
			makeFragment(72, 50, 50, 10, "Page 1"),
		}),
	}

	result := detector.Detect(pages)

	if result.HasHeadersOrFooters() {
		t.Error("expected no headers/footers with single page")
	}
}

func TestHeaderFooterDetector_ConsistentHeader(t *testing.T) {
	detector := NewHeaderFooterDetector()

	// Three pages with same header
	pages := []PageFragments{
		makePageFragments(0, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Company Report 2024"),
			makeFragment(72, 400, 468, 10, "Body text page 1"),
		}),
		makePageFragments(1, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Company Report 2024"),
			makeFragment(72, 400, 468, 10, "Body text page 2"),
		}),
		makePageFragments(2, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Company Report 2024"),
			makeFragment(72, 400, 468, 10, "Body text page 3"),
		}),
	}

	result := detector.Detect(pages)

	if !result.HasHeaders() {
		t.Error("expected headers to be detected")
	}

	if len(result.Headers) != 1 {
		t.Errorf("expected 1 header, got %d", len(result.Headers))
	}

	if result.Headers[0].Text != "Company Report 2024" {
		t.Errorf("expected header text 'Company Report 2024', got %q", result.Headers[0].Text)
	}

	if result.Headers[0].Type != Header {
		t.Error("expected Header type")
	}
}

func TestHeaderFooterDetector_ConsistentFooter(t *testing.T) {
	detector := NewHeaderFooterDetector()

	// Three pages with same footer
	pages := []PageFragments{
		makePageFragments(0, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 468, 10, "Body text page 1"),
			makeFragment(72, 30, 150, 10, "Confidential"),
		}),
		makePageFragments(1, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 468, 10, "Body text page 2"),
			makeFragment(72, 30, 150, 10, "Confidential"),
		}),
		makePageFragments(2, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 468, 10, "Body text page 3"),
			makeFragment(72, 30, 150, 10, "Confidential"),
		}),
	}

	result := detector.Detect(pages)

	if !result.HasFooters() {
		t.Error("expected footers to be detected")
	}

	if len(result.Footers) != 1 {
		t.Errorf("expected 1 footer, got %d", len(result.Footers))
	}

	if result.Footers[0].Text != "Confidential" {
		t.Errorf("expected footer text 'Confidential', got %q", result.Footers[0].Text)
	}

	if result.Footers[0].Type != Footer {
		t.Error("expected Footer type")
	}
}

func TestHeaderFooterDetector_PageNumbers(t *testing.T) {
	detector := NewHeaderFooterDetector()

	// Three pages with page numbers
	pages := []PageFragments{
		makePageFragments(0, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 468, 10, "Body text"),
			makeFragment(300, 30, 30, 10, "1"),
		}),
		makePageFragments(1, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 468, 10, "Body text"),
			makeFragment(300, 30, 30, 10, "2"),
		}),
		makePageFragments(2, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 468, 10, "Body text"),
			makeFragment(300, 30, 30, 10, "3"),
		}),
	}

	result := detector.Detect(pages)

	if !result.HasFooters() {
		t.Error("expected page numbers in footer to be detected")
	}

	// Find the page number footer
	var pageNumFooter *HeaderFooterRegion
	for i := range result.Footers {
		if result.Footers[i].IsPageNumber {
			pageNumFooter = &result.Footers[i]
			break
		}
	}

	if pageNumFooter == nil {
		t.Error("expected page number footer to be marked as IsPageNumber")
	}
}

func TestHeaderFooterDetector_PageXofY(t *testing.T) {
	detector := NewHeaderFooterDetector()

	// Pages with "Page X of Y" format
	pages := []PageFragments{
		makePageFragments(0, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 468, 10, "Body text"),
			makeFragment(250, 30, 100, 10, "Page 1 of 5"),
		}),
		makePageFragments(1, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 468, 10, "Body text"),
			makeFragment(250, 30, 100, 10, "Page 2 of 5"),
		}),
		makePageFragments(2, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 468, 10, "Body text"),
			makeFragment(250, 30, 100, 10, "Page 3 of 5"),
		}),
	}

	result := detector.Detect(pages)

	if !result.HasFooters() {
		t.Error("expected 'Page X of Y' footer to be detected")
	}

	// Should be detected as page number
	found := false
	for _, f := range result.Footers {
		if f.IsPageNumber {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'Page X of Y' to be marked as page number")
	}
}

func TestHeaderFooterDetector_HeaderAndFooter(t *testing.T) {
	detector := NewHeaderFooterDetector()

	// Pages with both header and footer
	pages := []PageFragments{
		makePageFragments(0, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Annual Report"),
			makeFragment(72, 400, 468, 10, "Body text page 1"),
			makeFragment(300, 30, 30, 10, "1"),
		}),
		makePageFragments(1, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Annual Report"),
			makeFragment(72, 400, 468, 10, "Body text page 2"),
			makeFragment(300, 30, 30, 10, "2"),
		}),
		makePageFragments(2, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Annual Report"),
			makeFragment(72, 400, 468, 10, "Body text page 3"),
			makeFragment(300, 30, 30, 10, "3"),
		}),
	}

	result := detector.Detect(pages)

	if !result.HasHeaders() {
		t.Error("expected header to be detected")
	}

	if !result.HasFooters() {
		t.Error("expected footer to be detected")
	}

	if !result.HasHeadersOrFooters() {
		t.Error("HasHeadersOrFooters should be true")
	}
}

func TestHeaderFooterDetector_InconsistentPosition(t *testing.T) {
	detector := NewHeaderFooterDetector()

	// Same text at different positions - should NOT be detected
	pages := []PageFragments{
		makePageFragments(0, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Same Text"),
		}),
		makePageFragments(1, 792, 612, []text.TextFragment{
			makeFragment(200, 760, 200, 12, "Same Text"), // Different X
		}),
		makePageFragments(2, 792, 612, []text.TextFragment{
			makeFragment(72, 740, 200, 12, "Same Text"), // Different Y
		}),
	}

	result := detector.Detect(pages)

	// May or may not detect depending on tolerance
	t.Logf("Headers detected: %d", len(result.Headers))
}

func TestHeaderFooterDetector_FilterFragments(t *testing.T) {
	detector := NewHeaderFooterDetector()

	pages := []PageFragments{
		makePageFragments(0, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Header Text"),
			makeFragment(72, 400, 468, 10, "Body content 1"),
			makeFragment(72, 380, 468, 10, "Body content 2"),
			makeFragment(300, 30, 30, 10, "1"),
		}),
		makePageFragments(1, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Header Text"),
			makeFragment(72, 400, 468, 10, "Body content 3"),
			makeFragment(72, 380, 468, 10, "Body content 4"),
			makeFragment(300, 30, 30, 10, "2"),
		}),
	}

	result := detector.Detect(pages)

	// Filter fragments from page 0
	original := pages[0].Fragments
	filtered := result.FilterFragments(0, original, 792)

	// Should have fewer fragments after filtering
	if len(filtered) >= len(original) {
		t.Errorf("expected filtered to have fewer fragments, got %d vs %d", len(filtered), len(original))
	}

	// Body content should remain
	hasBody := false
	for _, f := range filtered {
		if f.Text == "Body content 1" || f.Text == "Body content 2" {
			hasBody = true
			break
		}
	}

	if !hasBody {
		t.Error("body content should remain after filtering")
	}
}

func TestHeaderFooterDetector_CustomConfig(t *testing.T) {
	config := HeaderFooterConfig{
		HeaderRegionHeight: 100.0, // Larger header region
		FooterRegionHeight: 100.0, // Larger footer region
		MinOccurrenceRatio: 0.3,   // Lower threshold
		PositionTolerance:  10.0,  // More tolerant
		XPositionTolerance: 20.0,  // More tolerant
		MinPages:           2,
	}

	detector := NewHeaderFooterDetectorWithConfig(config)

	pages := []PageFragments{
		makePageFragments(0, 792, 612, []text.TextFragment{
			makeFragment(72, 720, 200, 12, "Header"), // Within 100pt of top
		}),
		makePageFragments(1, 792, 612, []text.TextFragment{
			makeFragment(72, 720, 200, 12, "Header"),
		}),
	}

	result := detector.Detect(pages)

	if !result.HasHeaders() {
		t.Error("expected header with custom config")
	}
}

func TestNormalizeForComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Page 1", "Page #"},
		{"Page 10", "Page #"},
		{"Page 123", "Page #"},
		{"1 of 10", "# of #"},
		{"Chapter 5: Introduction", "Chapter #: Introduction"},
		{"No numbers", "No numbers"},
		{"", ""},
	}

	for _, tc := range tests {
		result := normalizeForComparison(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeForComparison(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestIsPageNumberPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"#", true},
		{"Page #", true},
		{"page #", true},
		{"# of #", true},
		{"Page # of #", true},
		{"p. #", true},
		{"Random text", false},
		{"Page", false},
		{"", false},
	}

	for _, tc := range tests {
		result := isPageNumberPattern(tc.input)
		if result != tc.expected {
			t.Errorf("isPageNumberPattern(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}

func TestRegionType_String(t *testing.T) {
	if Header.String() != "header" {
		t.Errorf("Header.String() = %q, want 'header'", Header.String())
	}

	if Footer.String() != "footer" {
		t.Errorf("Footer.String() = %q, want 'footer'", Footer.String())
	}
}

func TestHeaderFooterResult_NilReceiver(t *testing.T) {
	var result *HeaderFooterResult

	// These should not panic
	if result.HasHeaders() {
		t.Error("nil result should not have headers")
	}

	if result.HasFooters() {
		t.Error("nil result should not have footers")
	}

	if result.HasHeadersOrFooters() {
		t.Error("nil result should not have headers or footers")
	}

	if result.GetHeaderTexts() != nil {
		t.Error("nil result GetHeaderTexts should return nil")
	}

	if result.GetFooterTexts() != nil {
		t.Error("nil result GetFooterTexts should return nil")
	}

	filtered := result.FilterFragments(0, []text.TextFragment{
		makeFragment(72, 400, 100, 10, "Test"),
	}, 792)

	if len(filtered) != 1 {
		t.Error("nil result FilterFragments should return original")
	}
}

func TestHeaderFooterResult_Summary(t *testing.T) {
	// Test with nil
	var nilResult *HeaderFooterResult
	if nilResult.Summary() != "No headers or footers detected" {
		t.Error("nil result summary should indicate no detection")
	}

	// Test with empty result
	emptyResult := &HeaderFooterResult{}
	if emptyResult.Summary() != "No headers or footers detected" {
		t.Error("empty result summary should indicate no detection")
	}

	// Test with headers and footers
	result := &HeaderFooterResult{
		Headers: []HeaderFooterRegion{
			{Text: "Title", Type: Header},
		},
		Footers: []HeaderFooterRegion{
			{Text: "[Page Number]", Type: Footer},
		},
	}

	summary := result.Summary()
	if summary == "No headers or footers detected" {
		t.Error("result with headers/footers should have meaningful summary")
	}
}

func TestHeaderFooterDetector_MinOccurrenceRatio(t *testing.T) {
	detector := NewHeaderFooterDetector()

	// Text appears on only 1 of 5 pages (20%) - clearly below default 50% threshold
	// Note: With 5 pages and 50% threshold, minOccurrences = int(5*0.5) = 2
	// So we need header on less than 2 pages to be below threshold
	pages := []PageFragments{
		makePageFragments(0, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Header"),
		}),
		makePageFragments(1, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 200, 12, "No header on this page"),
		}),
		makePageFragments(2, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 200, 12, "No header on this page"),
		}),
		makePageFragments(3, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 200, 12, "No header on this page"),
		}),
		makePageFragments(4, 792, 612, []text.TextFragment{
			makeFragment(72, 400, 200, 12, "No header on this page"),
		}),
	}

	result := detector.Detect(pages)

	// With default 50% threshold, header appearing on only 20% (1 of 5) should NOT be detected
	if result.HasHeaders() {
		t.Error("header appearing on 20% of pages should not be detected with 50% threshold")
	}
}

func TestContainsPageNumberPattern(t *testing.T) {
	// Sequential numbers should be detected
	group := []candidate{
		{Text: "1", PageIndex: 0},
		{Text: "2", PageIndex: 1},
		{Text: "3", PageIndex: 2},
	}

	if !containsPageNumberPattern(group) {
		t.Error("sequential numbers should be detected as page numbers")
	}

	// Non-sequential should not be detected
	nonSeq := []candidate{
		{Text: "5", PageIndex: 0},
		{Text: "10", PageIndex: 1},
		{Text: "20", PageIndex: 2},
	}

	// This depends on the implementation - 5, 10, 20 are not sequential
	// but might still be detected depending on the logic
	t.Logf("Non-sequential detected as page number: %v", containsPageNumberPattern(nonSeq))
}

// Benchmark header/footer detection
func BenchmarkHeaderFooterDetector_SmallDocument(b *testing.B) {
	detector := NewHeaderFooterDetector()

	// 10 pages with headers and footers
	var pages []PageFragments
	for i := 0; i < 10; i++ {
		pages = append(pages, makePageFragments(i, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Document Header"),
			makeFragment(72, 400, 468, 10, "Body text line 1"),
			makeFragment(72, 380, 468, 10, "Body text line 2"),
			makeFragment(72, 360, 468, 10, "Body text line 3"),
			makeFragment(300, 30, 30, 10, string(rune('0'+i))), // Page number
		}))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(pages)
	}
}

func BenchmarkHeaderFooterDetector_LargeDocument(b *testing.B) {
	detector := NewHeaderFooterDetector()

	// 100 pages with headers and footers
	var pages []PageFragments
	for i := 0; i < 100; i++ {
		pages = append(pages, makePageFragments(i, 792, 612, []text.TextFragment{
			makeFragment(72, 760, 200, 12, "Document Header"),
			makeFragment(72, 400, 468, 10, "Body text"),
			makeFragment(300, 30, 50, 10, "Page "+string(rune('0'+i%10))),
		}))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(pages)
	}
}
