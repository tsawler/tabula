package tabula

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testPDFPath returns the path to a test PDF file
func testPDFPath(filename string) string {
	// Look for pdf-samples directory
	return filepath.Join("..", "pdf-samples", filename)
}

func TestOpen(t *testing.T) {
	// Test with non-existent file
	_, err := Open("nonexistent.pdf").Text()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestBasicTextExtraction(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	text, err := Open(pdfPath).Text()
	if err != nil {
		t.Fatalf("failed to extract text: %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}

	// Check for expected content
	if !strings.Contains(text, "Dinosaurs") {
		t.Error("expected text to contain 'Dinosaurs'")
	}
}

func TestPageSelection(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Extract only page 1
	text1, err := Open(pdfPath).Pages(1).Text()
	if err != nil {
		t.Fatalf("failed to extract page 1: %v", err)
	}

	// Extract all pages
	textAll, err := Open(pdfPath).Text()
	if err != nil {
		t.Fatalf("failed to extract all pages: %v", err)
	}

	// Page 1 should be shorter than all pages
	if len(text1) >= len(textAll) {
		t.Error("expected page 1 to be shorter than all pages")
	}
}

func TestPageRange(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Get page count
	ext := Open(pdfPath)
	count, err := ext.PageCount()
	if err != nil {
		t.Fatalf("failed to get page count: %v", err)
	}
	ext.Close()

	if count < 2 {
		t.Skip("need at least 2 pages for this test")
	}

	// Extract pages 1-2
	text, err := Open(pdfPath).PageRange(1, 2).Text()
	if err != nil {
		t.Fatalf("failed to extract page range: %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text from page range")
	}
}

func TestInvalidPage(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Try to extract page 1000 (should fail)
	_, err := Open(pdfPath).Pages(1000).Text()
	if err == nil {
		t.Error("expected error for invalid page number")
	}

	// Try page 0 (should fail - 1-indexed)
	_, err = Open(pdfPath).Pages(0).Text()
	if err == nil {
		t.Error("expected error for page 0 (1-indexed)")
	}
}

func TestPageCount(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	ext := Open(pdfPath)
	defer ext.Close()

	count, err := ext.PageCount()
	if err != nil {
		t.Fatalf("failed to get page count: %v", err)
	}

	if count <= 0 {
		t.Error("expected positive page count")
	}
}

func TestFragments(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	fragments, err := Open(pdfPath).Pages(1).Fragments()
	if err != nil {
		t.Fatalf("failed to extract fragments: %v", err)
	}

	if len(fragments) == 0 {
		t.Error("expected non-empty fragments")
	}

	// Check that fragments have positions
	for _, frag := range fragments {
		if frag.FontSize <= 0 {
			t.Error("expected positive font size")
		}
	}
}

func TestChainImmutability(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Create base extractor
	base := Open(pdfPath)

	// Create derived extractors
	withPage1 := base.Pages(1)
	withPage2 := base.Pages(2)

	// Verify they're independent
	if len(base.options.pages) != 0 {
		t.Error("base extractor should have no pages set")
	}
	if len(withPage1.options.pages) != 1 || withPage1.options.pages[0] != 1 {
		t.Error("withPage1 should have page 1")
	}
	if len(withPage2.options.pages) != 1 || withPage2.options.pages[0] != 2 {
		t.Error("withPage2 should have page 2")
	}
}

func TestMust(t *testing.T) {
	// Test Must with successful result
	result := Must("hello", nil)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}

	// Test Must with error (should panic)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected Must to panic on error")
		}
	}()
	Must("", os.ErrNotExist)
}

func TestByColumn(t *testing.T) {
	pdfPath := testPDFPath("3cols.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Extract with column detection
	text, err := Open(pdfPath).ByColumn().Text()
	if err != nil {
		t.Fatalf("failed to extract with column detection: %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}
}

func TestExcludeHeadersFooters(t *testing.T) {
	pdfPath := testPDFPath("header-footer-column.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Extract with header/footer exclusion
	textFiltered, err := Open(pdfPath).
		ExcludeHeaders().
		ExcludeFooters().
		Text()
	if err != nil {
		t.Fatalf("failed to extract with filtering: %v", err)
	}

	// Extract without filtering
	textUnfiltered, err := Open(pdfPath).Text()
	if err != nil {
		t.Fatalf("failed to extract without filtering: %v", err)
	}

	// Filtered text should be different (likely shorter)
	if textFiltered == textUnfiltered {
		t.Log("Warning: filtered and unfiltered text are the same - header/footer detection may not have found anything")
	}
}

func TestCloseIdempotent(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	ext := Open(pdfPath)

	// Multiple closes should be safe
	err := ext.Close()
	if err != nil {
		t.Errorf("first close failed: %v", err)
	}

	err = ext.Close()
	if err != nil {
		t.Errorf("second close failed: %v", err)
	}
}
