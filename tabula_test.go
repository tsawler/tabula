package tabula

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsawler/tabula/layout"
	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/rag"
	"github.com/tsawler/tabula/text"
)

// newZipWriter creates a zip.Writer wrapper for test DOCX creation.
func newZipWriter(w io.Writer) *zip.Writer {
	return zip.NewWriter(w)
}

// testPDFPath returns the path to a test PDF file
func testPDFPath(filename string) string {
	// Look for pdf-samples directory
	return filepath.Join("..", "pdf-samples", filename)
}

func TestOpen(t *testing.T) {
	// Test with non-existent file
	_, _, err := Open("nonexistent.pdf").Text()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestBasicTextExtraction(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	text, _, err := Open(pdfPath).Text()
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
	text1, _, err := Open(pdfPath).Pages(1).Text()
	if err != nil {
		t.Fatalf("failed to extract page 1: %v", err)
	}

	// Extract all pages
	textAll, _, err := Open(pdfPath).Text()
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
	text, _, err := Open(pdfPath).PageRange(1, 2).Text()
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
	_, _, err := Open(pdfPath).Pages(1000).Text()
	if err == nil {
		t.Error("expected error for invalid page number")
	}

	// Try page 0 (should fail - 1-indexed)
	_, _, err = Open(pdfPath).Pages(0).Text()
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

	fragments, _, err := Open(pdfPath).Pages(1).Fragments()
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
	text, _, err := Open(pdfPath).ByColumn().Text()
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
	textFiltered, _, err := Open(pdfPath).
		ExcludeHeaders().
		ExcludeFooters().
		Text()
	if err != nil {
		t.Fatalf("failed to extract with filtering: %v", err)
	}

	// Extract without filtering
	textUnfiltered, _, err := Open(pdfPath).Text()
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

func TestDocument(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	doc, warnings, err := Open(pdfPath).Document()
	if err != nil {
		t.Fatalf("failed to extract document: %v", err)
	}

	// Document should not be nil
	if doc == nil {
		t.Fatal("expected non-nil document")
	}

	// Document should have at least one page
	if doc.PageCount() == 0 {
		t.Error("expected at least one page")
	}

	// First page should have layout info
	page := doc.GetPage(1)
	if page == nil {
		t.Fatal("expected to get page 1")
	}

	// Log warnings for debugging
	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}
}

func TestDocumentWithOptions(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Extract only page 1
	doc, _, err := Open(pdfPath).Pages(1).Document()
	if err != nil {
		t.Fatalf("failed to extract document: %v", err)
	}

	// Document should have exactly one page
	if doc.PageCount() != 1 {
		t.Errorf("expected 1 page, got %d", doc.PageCount())
	}
}

func TestDocumentWithHeaderFooterExclusion(t *testing.T) {
	pdfPath := testPDFPath("header-footer-column.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Extract with header/footer exclusion
	doc, _, err := Open(pdfPath).ExcludeHeadersAndFooters().Document()
	if err != nil {
		t.Fatalf("failed to extract document: %v", err)
	}

	if doc == nil {
		t.Fatal("expected non-nil document")
	}

	if doc.PageCount() == 0 {
		t.Error("expected at least one page")
	}
}

func TestChunks(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	chunks, warnings, err := Open(pdfPath).Chunks()
	if err != nil {
		t.Fatalf("failed to extract chunks: %v", err)
	}

	// Chunks should not be nil
	if chunks == nil {
		t.Fatal("expected non-nil chunk collection")
	}

	// Should have at least one chunk
	if len(chunks.Chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Each chunk should have non-empty text
	for i, chunk := range chunks.Chunks {
		if chunk.Text == "" {
			t.Errorf("chunk %d has empty text", i)
		}
		if chunk.ID == "" {
			t.Errorf("chunk %d has empty ID", i)
		}
	}

	// Log warnings for debugging
	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}
}

func TestChunksWithOptions(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Extract chunks with header/footer exclusion
	chunks, _, err := Open(pdfPath).
		ExcludeHeadersAndFooters().
		Chunks()
	if err != nil {
		t.Fatalf("failed to extract chunks: %v", err)
	}

	if chunks == nil {
		t.Fatal("expected non-nil chunk collection")
	}

	if len(chunks.Chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestChunksWithConfig(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Create custom config with smaller chunk size
	config := rag.ChunkerConfig{
		TargetChunkSize: 200,
		MaxChunkSize:    500,
		MinChunkSize:    50,
		OverlapSize:     20,
	}
	sizeConfig := rag.DefaultSizeConfig()

	chunks, _, err := Open(pdfPath).
		Pages(1).
		ChunksWithConfig(config, sizeConfig)
	if err != nil {
		t.Fatalf("failed to extract chunks with config: %v", err)
	}

	if chunks == nil {
		t.Fatal("expected non-nil chunk collection")
	}

	// With smaller max chunk size, we might get more chunks
	// but at minimum we should have at least one
	if len(chunks.Chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestDocumentErrorHandling(t *testing.T) {
	// Test with non-existent file
	_, _, err := Open("nonexistent.pdf").Document()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestChunksErrorHandling(t *testing.T) {
	// Test with non-existent file
	_, _, err := Open("nonexistent.pdf").Chunks()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestToMarkdown(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	md, warnings, err := Open(pdfPath).ToMarkdown()
	if err != nil {
		t.Fatalf("failed to extract markdown: %v", err)
	}

	// Markdown should not be empty
	if md == "" {
		t.Error("expected non-empty markdown")
	}

	// Log warnings for debugging
	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}
}

func TestToMarkdownWithOptions(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	opts := rag.MarkdownOptions{
		IncludePageNumbers: true,
		IncludeMetadata:    true,
	}

	md, _, err := Open(pdfPath).
		Pages(1).
		ToMarkdownWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to extract markdown: %v", err)
	}

	if md == "" {
		t.Error("expected non-empty markdown")
	}

	// Should have YAML front matter when metadata is enabled
	if !strings.Contains(md, "---") {
		t.Error("expected YAML front matter with IncludeMetadata option")
	}
}

func TestToMarkdownWithHeaderFooterExclusion(t *testing.T) {
	pdfPath := testPDFPath("header-footer-column.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	md, _, err := Open(pdfPath).
		ExcludeHeadersAndFooters().
		ToMarkdown()
	if err != nil {
		t.Fatalf("failed to extract markdown: %v", err)
	}

	if md == "" {
		t.Error("expected non-empty markdown")
	}
}

func TestToMarkdownErrorHandling(t *testing.T) {
	// Test with non-existent file
	_, _, err := Open("nonexistent.pdf").ToMarkdown()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// ============================================================================
// DOCX Tests
// ============================================================================

func TestDOCXTextExtraction(t *testing.T) {
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")

	if err := createMinimalDOCX(docxPath, "Hello from DOCX"); err != nil {
		t.Fatalf("failed to create test DOCX: %v", err)
	}

	text, warnings, err := Open(docxPath).Text()
	if err != nil {
		t.Fatalf("Open(docx).Text() error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if !strings.Contains(text, "Hello from DOCX") {
		t.Errorf("Text() = %q, expected to contain 'Hello from DOCX'", text)
	}
}

func TestDOCXPageCount(t *testing.T) {
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")

	if err := createMinimalDOCX(docxPath, "Test content"); err != nil {
		t.Fatalf("failed to create test DOCX: %v", err)
	}

	ext := Open(docxPath)
	defer ext.Close()

	count, err := ext.PageCount()
	if err != nil {
		t.Fatalf("PageCount() error = %v", err)
	}

	// DOCX is treated as single page
	if count != 1 {
		t.Errorf("PageCount() = %d, want 1", count)
	}
}

func TestDOCXDocument(t *testing.T) {
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")

	// Create DOCX with heading and paragraph
	content := `<w:p>
  <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
  <w:r><w:t>Test Heading</w:t></w:r>
</w:p>
<w:p><w:r><w:t>Test paragraph content.</w:t></w:r></w:p>`

	if err := createMinimalDOCXWithContent(docxPath, content); err != nil {
		t.Fatalf("failed to create test DOCX: %v", err)
	}

	doc, warnings, err := Open(docxPath).Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if doc.PageCount() != 1 {
		t.Errorf("PageCount() = %d, want 1", doc.PageCount())
	}

	page := doc.GetPage(1)
	if page == nil {
		t.Fatal("GetPage(1) returned nil")
	}

	// Should have heading and paragraph elements
	if len(page.Elements) < 1 {
		t.Errorf("Elements count = %d, want >= 1", len(page.Elements))
	}
}

func TestDOCXUnsupportedFormat(t *testing.T) {
	// Test with unsupported extension
	_, _, err := Open("test.xyz").Text()
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

// createMinimalDOCX creates a minimal valid DOCX file with simple text content.
func createMinimalDOCX(path, text string) error {
	content := `<w:p><w:r><w:t>` + text + `</w:t></w:r></w:p>`
	return createMinimalDOCXWithContent(path, content)
}

// createMinimalDOCXWithContent creates a minimal valid DOCX file with custom XML content.
func createMinimalDOCXWithContent(path, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := newZipWriter(f)
	defer zw.Close()

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`
	w, _ := zw.Create("[Content_Types].xml")
	w.Write([]byte(contentTypes))

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
	w, _ = zw.Create("_rels/.rels")
	w.Write([]byte(rels))

	// word/document.xml
	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>` + content + `</w:body>
</w:document>`
	w, _ = zw.Create("word/document.xml")
	w.Write([]byte(document))

	return nil
}

func TestPreserveLayout(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Extract text with PreserveLayout
	text, _, err := Open(pdfPath).Pages(1).PreserveLayout().Text()
	if err != nil {
		t.Fatalf("failed to extract text with PreserveLayout: %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}

	// PreserveLayout should contain spaces for positioning
	// The output should still contain the word "Dinosaurs"
	if !strings.Contains(text, "Dinosaurs") {
		t.Error("expected text to contain 'Dinosaurs'")
	}
}

func TestPreserveLayoutWithForm(t *testing.T) {
	// Test with a form-like PDF if available
	pdfPath := testPDFPath("form.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("form.pdf not found:", pdfPath)
	}

	text, _, err := Open(pdfPath).PreserveLayout().Text()
	if err != nil {
		t.Fatalf("failed to extract text: %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}
}

func TestPreserveLayoutVsNormal(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Get normal text
	normalText, _, err := Open(pdfPath).Pages(1).Text()
	if err != nil {
		t.Fatalf("failed to extract normal text: %v", err)
	}

	// Get preserve layout text
	layoutText, _, err := Open(pdfPath).Pages(1).PreserveLayout().Text()
	if err != nil {
		t.Fatalf("failed to extract layout text: %v", err)
	}

	// Both should be non-empty
	if len(normalText) == 0 || len(layoutText) == 0 {
		t.Error("expected non-empty text from both methods")
	}

	// Layout preserved text typically has more spaces
	normalSpaces := strings.Count(normalText, " ")
	layoutSpaces := strings.Count(layoutText, " ")

	// PreserveLayout should generally have more spaces due to positioning
	// (though this isn't always true for all documents)
	t.Logf("Normal text has %d spaces, PreserveLayout has %d spaces", normalSpaces, layoutSpaces)
}

func TestExtractPreserveLayoutUnit(t *testing.T) {
	// Test with synthetic fragments simulating a form-like layout:
	// Name: ____________    Date: ____________
	// Address: _________________________________

	ext := &Extractor{}

	// Page width of 612 points (US Letter)
	pageWidth := 612.0

	// Create fragments that simulate a form layout
	// Font size 12, so charWidth ~7.2 points
	fragments := []text.TextFragment{
		// First line: "Name:" at left, "Date:" at right
		{Text: "Name:", X: 72, Y: 720, Width: 35, Height: 12, FontSize: 12},
		{Text: "Date:", X: 400, Y: 720, Width: 30, Height: 12, FontSize: 12},
		// Second line: "Address:" at left
		{Text: "Address:", X: 72, Y: 700, Width: 50, Height: 12, FontSize: 12},
	}

	result := ext.extractPreserveLayout(fragments, pageWidth)

	// Check that output is non-empty
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}

	// Check that we have both "Name:" and "Date:" on the same line
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Errorf("expected at least 2 lines, got %d", len(lines))
	}

	// First line should have Name: and Date: with spacing between them
	if len(lines) > 0 {
		firstLine := lines[0]
		if !strings.Contains(firstLine, "Name:") {
			t.Error("expected first line to contain 'Name:'")
		}
		if !strings.Contains(firstLine, "Date:") {
			t.Error("expected first line to contain 'Date:'")
		}

		// There should be significant space between Name: and Date:
		nameIdx := strings.Index(firstLine, "Name:")
		dateIdx := strings.Index(firstLine, "Date:")
		if dateIdx <= nameIdx+10 {
			t.Error("expected significant spacing between 'Name:' and 'Date:'")
		}
	}

	// Second line (or later) should have Address:
	found := false
	for i := 1; i < len(lines); i++ {
		if strings.Contains(lines[i], "Address:") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Address:' on a line after the first")
	}
}

func TestExtractPreserveLayoutEmpty(t *testing.T) {
	ext := &Extractor{}
	result := ext.extractPreserveLayout(nil, 612)
	if result != "" {
		t.Errorf("expected empty string for nil fragments, got %q", result)
	}

	result = ext.extractPreserveLayout([]text.TextFragment{}, 612)
	if result != "" {
		t.Errorf("expected empty string for empty fragments, got %q", result)
	}
}

// ============================================================================
// Lines, Paragraphs, ReadingOrder, Analyze tests
// ============================================================================

func TestLines(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	lines, err := Open(pdfPath).Pages(1).Lines()
	if err != nil {
		t.Fatalf("Lines() error = %v", err)
	}

	if len(lines) == 0 {
		t.Error("expected non-empty lines")
	}

	// Check that lines have non-empty text
	for i, line := range lines {
		if len(line.Text) == 0 {
			t.Errorf("line %d has empty text", i)
		}
	}
}

func TestLinesWithHeaderFooterExclusion(t *testing.T) {
	pdfPath := testPDFPath("header-footer-column.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Extract with header/footer exclusion
	linesFiltered, err := Open(pdfPath).
		ExcludeHeaders().
		ExcludeFooters().
		Lines()
	if err != nil {
		t.Fatalf("Lines() with filtering error = %v", err)
	}

	// Extract without filtering
	linesUnfiltered, err := Open(pdfPath).Lines()
	if err != nil {
		t.Fatalf("Lines() without filtering error = %v", err)
	}

	// Lines should be non-empty in both cases
	if len(linesFiltered) == 0 {
		t.Error("expected non-empty filtered lines")
	}
	if len(linesUnfiltered) == 0 {
		t.Error("expected non-empty unfiltered lines")
	}

	t.Logf("Filtered lines: %d, Unfiltered lines: %d", len(linesFiltered), len(linesUnfiltered))
}

func TestLinesError(t *testing.T) {
	_, err := Open("nonexistent.pdf").Lines()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestParagraphs(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	paragraphs, err := Open(pdfPath).Pages(1).Paragraphs()
	if err != nil {
		t.Fatalf("Paragraphs() error = %v", err)
	}

	if len(paragraphs) == 0 {
		t.Error("expected non-empty paragraphs")
	}

	// Check that paragraphs have non-empty text
	for i, para := range paragraphs {
		if len(para.Text) == 0 {
			t.Errorf("paragraph %d has empty text", i)
		}
	}
}

func TestParagraphsWithHeaderFooterExclusion(t *testing.T) {
	pdfPath := testPDFPath("header-footer-column.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	paragraphs, err := Open(pdfPath).
		ExcludeHeadersAndFooters().
		Paragraphs()
	if err != nil {
		t.Fatalf("Paragraphs() error = %v", err)
	}

	if len(paragraphs) == 0 {
		t.Error("expected non-empty paragraphs")
	}
}

func TestParagraphsError(t *testing.T) {
	_, err := Open("nonexistent.pdf").Paragraphs()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestReadingOrder(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	ro, err := Open(pdfPath).Pages(1).ReadingOrder()
	if err != nil {
		t.Fatalf("ReadingOrder() error = %v", err)
	}

	if ro == nil {
		t.Fatal("expected non-nil reading order result")
	}

	if ro.ColumnCount < 1 {
		t.Errorf("expected at least 1 column, got %d", ro.ColumnCount)
	}

	if ro.PageWidth == 0 || ro.PageHeight == 0 {
		t.Error("expected non-zero page dimensions")
	}
}

func TestReadingOrderMultiColumn(t *testing.T) {
	pdfPath := testPDFPath("3cols.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	ro, err := Open(pdfPath).Pages(1).ReadingOrder()
	if err != nil {
		t.Fatalf("ReadingOrder() error = %v", err)
	}

	if ro == nil {
		t.Fatal("expected non-nil reading order result")
	}

	// Multi-column PDF should have more than 1 column
	t.Logf("Detected %d columns", ro.ColumnCount)
}

func TestReadingOrderWithHeaderFooterExclusion(t *testing.T) {
	pdfPath := testPDFPath("header-footer-column.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	ro, err := Open(pdfPath).
		ExcludeHeadersAndFooters().
		ReadingOrder()
	if err != nil {
		t.Fatalf("ReadingOrder() error = %v", err)
	}

	if ro == nil {
		t.Fatal("expected non-nil reading order result")
	}
}

func TestReadingOrderError(t *testing.T) {
	_, err := Open("nonexistent.pdf").ReadingOrder()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestAnalyze(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	result, err := Open(pdfPath).Pages(1).Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil analysis result")
	}

	if len(result.Elements) == 0 {
		t.Error("expected non-empty elements")
	}

	if result.Stats.ElementCount == 0 {
		t.Error("expected non-zero element count in stats")
	}

	if result.PageWidth == 0 || result.PageHeight == 0 {
		t.Error("expected non-zero page dimensions")
	}
}

func TestAnalyzeWithHeaderFooterExclusion(t *testing.T) {
	pdfPath := testPDFPath("header-footer-column.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	result, err := Open(pdfPath).
		ExcludeHeadersAndFooters().
		Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil analysis result")
	}
}

func TestAnalyzeError(t *testing.T) {
	_, err := Open("nonexistent.pdf").Analyze()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestHeadings(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	headings, err := Open(pdfPath).Headings()
	if err != nil {
		t.Fatalf("Headings() error = %v", err)
	}

	// Note: May or may not have headings depending on PDF content
	t.Logf("Found %d headings", len(headings))

	// Verify heading structure if any found
	for i, h := range headings {
		if h.Level < 1 || h.Level > 6 {
			t.Errorf("heading %d has invalid level %d", i, h.Level)
		}
	}
}

func TestHeadingsWithExclusion(t *testing.T) {
	pdfPath := testPDFPath("header-footer-column.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	headings, err := Open(pdfPath).
		ExcludeHeadersAndFooters().
		Headings()
	if err != nil {
		t.Fatalf("Headings() error = %v", err)
	}

	t.Logf("Found %d headings after exclusion", len(headings))
}

func TestHeadingsError(t *testing.T) {
	_, err := Open("nonexistent.pdf").Headings()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLists(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	lists, err := Open(pdfPath).Lists()
	if err != nil {
		t.Fatalf("Lists() error = %v", err)
	}

	// Note: May or may not have lists depending on PDF content
	t.Logf("Found %d lists", len(lists))

	// Verify list structure if any found
	for i, l := range lists {
		if len(l.Items) == 0 {
			t.Errorf("list %d has no items", i)
		}
	}
}

func TestListsWithExclusion(t *testing.T) {
	pdfPath := testPDFPath("header-footer-column.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	lists, err := Open(pdfPath).
		ExcludeHeadersAndFooters().
		Lists()
	if err != nil {
		t.Fatalf("Lists() error = %v", err)
	}

	t.Logf("Found %d lists after exclusion", len(lists))
}

func TestListsError(t *testing.T) {
	_, err := Open("nonexistent.pdf").Lists()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestBlocks(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	blocks, err := Open(pdfPath).Pages(1).Blocks()
	if err != nil {
		t.Fatalf("Blocks() error = %v", err)
	}

	if len(blocks) == 0 {
		t.Error("expected non-empty blocks")
	}

	// Check that blocks have bounding boxes
	for i, b := range blocks {
		if b.BBox.Width == 0 || b.BBox.Height == 0 {
			t.Errorf("block %d has zero-size bounding box", i)
		}
	}
}

func TestBlocksWithExclusion(t *testing.T) {
	pdfPath := testPDFPath("header-footer-column.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	blocks, err := Open(pdfPath).
		ExcludeHeadersAndFooters().
		Blocks()
	if err != nil {
		t.Fatalf("Blocks() error = %v", err)
	}

	if len(blocks) == 0 {
		t.Error("expected non-empty blocks")
	}
}

func TestBlocksError(t *testing.T) {
	_, err := Open("nonexistent.pdf").Blocks()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestElements(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	elements, err := Open(pdfPath).Pages(1).Elements()
	if err != nil {
		t.Fatalf("Elements() error = %v", err)
	}

	if len(elements) == 0 {
		t.Error("expected non-empty elements")
	}

	// Check element types
	for i, e := range elements {
		if e.Type == model.ElementType(0) {
			t.Errorf("element %d has empty type", i)
		}
	}
}

func TestElementsError(t *testing.T) {
	_, err := Open("nonexistent.pdf").Elements()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// ============================================================================
// JoinParagraphs tests
// ============================================================================

func TestJoinParagraphs(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Extract with JoinParagraphs
	textJoined, _, err := Open(pdfPath).Pages(1).JoinParagraphs().Text()
	if err != nil {
		t.Fatalf("Text() with JoinParagraphs error = %v", err)
	}

	if len(textJoined) == 0 {
		t.Error("expected non-empty text")
	}

	// Extract without JoinParagraphs
	textNormal, _, err := Open(pdfPath).Pages(1).Text()
	if err != nil {
		t.Fatalf("Text() without JoinParagraphs error = %v", err)
	}

	// Both should have content
	if len(textNormal) == 0 {
		t.Error("expected non-empty normal text")
	}

	// JoinParagraphs typically reduces newlines
	normalNewlines := strings.Count(textNormal, "\n")
	joinedNewlines := strings.Count(textJoined, "\n")
	t.Logf("Normal text has %d newlines, JoinParagraphs has %d", normalNewlines, joinedNewlines)
}

// ============================================================================
// IsCharacterLevel and IsMultiColumn tests
// ============================================================================

func TestIsCharacterLevel(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	ext := Open(pdfPath)
	defer ext.Close()

	isCharLevel, err := ext.IsCharacterLevel()
	if err != nil {
		t.Fatalf("IsCharacterLevel() error = %v", err)
	}

	// Log the result - could be either true or false
	t.Logf("IsCharacterLevel: %v", isCharLevel)
}

func TestIsCharacterLevelError(t *testing.T) {
	ext := Open("nonexistent.pdf")
	_, err := ext.IsCharacterLevel()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestIsMultiColumn(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	ext := Open(pdfPath)
	defer ext.Close()

	isMultiCol, err := ext.IsMultiColumn()
	if err != nil {
		t.Fatalf("IsMultiColumn() error = %v", err)
	}

	t.Logf("IsMultiColumn: %v", isMultiCol)
}

func TestIsMultiColumnWith3Cols(t *testing.T) {
	pdfPath := testPDFPath("3cols.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	ext := Open(pdfPath)
	defer ext.Close()

	isMultiCol, err := ext.IsMultiColumn()
	if err != nil {
		t.Fatalf("IsMultiColumn() error = %v", err)
	}

	// 3cols.pdf should be detected as multi-column
	t.Logf("3cols.pdf IsMultiColumn: %v", isMultiCol)
}

func TestIsMultiColumnError(t *testing.T) {
	ext := Open("nonexistent.pdf")
	_, err := ext.IsMultiColumn()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// ============================================================================
// assembleText tests
// ============================================================================

func TestAssembleText(t *testing.T) {
	ext := &Extractor{}

	// Test empty fragments
	result := ext.assembleText(nil)
	if result != "" {
		t.Errorf("expected empty string for nil fragments, got %q", result)
	}

	result = ext.assembleText([]text.TextFragment{})
	if result != "" {
		t.Errorf("expected empty string for empty fragments, got %q", result)
	}

	// Test single fragment
	fragments := []text.TextFragment{
		{Text: "Hello", X: 100, Y: 700, Width: 30, Height: 12, FontSize: 12},
	}
	result = ext.assembleText(fragments)
	if result != "Hello" {
		t.Errorf("expected 'Hello', got %q", result)
	}

	// Test multiple fragments on same line
	fragments = []text.TextFragment{
		{Text: "Hello", X: 100, Y: 700, Width: 30, Height: 12, FontSize: 12},
		{Text: "World", X: 145, Y: 700, Width: 30, Height: 12, FontSize: 12},
	}
	result = ext.assembleText(fragments)
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "World") {
		t.Errorf("expected 'Hello' and 'World' in result, got %q", result)
	}

	// Test fragments on different lines
	fragments = []text.TextFragment{
		{Text: "Line1", X: 100, Y: 700, Width: 30, Height: 12, FontSize: 12},
		{Text: "Line2", X: 100, Y: 680, Width: 30, Height: 12, FontSize: 12},
	}
	result = ext.assembleText(fragments)
	if !strings.Contains(result, "Line1") || !strings.Contains(result, "Line2") {
		t.Errorf("expected 'Line1' and 'Line2' in result, got %q", result)
	}
	if !strings.Contains(result, "\n") {
		t.Error("expected newline between lines")
	}
}

func TestAssembleTextParagraphBreaks(t *testing.T) {
	ext := &Extractor{}

	// Test fragments with large vertical gap (paragraph break)
	fragments := []text.TextFragment{
		{Text: "Para1", X: 100, Y: 700, Width: 30, Height: 12, FontSize: 12},
		{Text: "Para2", X: 100, Y: 650, Width: 30, Height: 12, FontSize: 12}, // Large gap
	}
	result := ext.assembleText(fragments)
	if !strings.Contains(result, "\n\n") {
		t.Error("expected double newline for paragraph break")
	}
}

// ============================================================================
// extractWithParagraphs tests
// ============================================================================

func TestExtractWithParagraphs(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Use JoinParagraphs which internally uses extractWithParagraphs
	text, _, err := Open(pdfPath).Pages(1).JoinParagraphs().Text()
	if err != nil {
		t.Fatalf("JoinParagraphs().Text() error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}

	// Content should be preserved
	if !strings.Contains(text, "Dinosaurs") {
		t.Error("expected text to contain 'Dinosaurs'")
	}
}

// ============================================================================
// isCharacterLevel and detectMultiColumn helper function tests
// ============================================================================

func TestIsCharacterLevelHelper(t *testing.T) {
	tests := []struct {
		name      string
		fragments []text.TextFragment
		want      bool
	}{
		{
			name:      "empty fragments",
			fragments: []text.TextFragment{},
			want:      false,
		},
		{
			name:      "too few fragments",
			fragments: make([]text.TextFragment, 5),
			want:      false,
		},
		{
			name: "word-level fragments",
			fragments: func() []text.TextFragment {
				frags := make([]text.TextFragment, 20)
				for i := range frags {
					frags[i] = text.TextFragment{Text: "word"}
				}
				return frags
			}(),
			want: false,
		},
		{
			name: "character-level fragments",
			fragments: func() []text.TextFragment {
				frags := make([]text.TextFragment, 20)
				for i := range frags {
					frags[i] = text.TextFragment{Text: "a"}
				}
				return frags
			}(),
			want: true,
		},
		{
			name: "mixed fragments below threshold",
			fragments: func() []text.TextFragment {
				frags := make([]text.TextFragment, 20)
				for i := range frags {
					if i < 10 {
						frags[i] = text.TextFragment{Text: "a"}
					} else {
						frags[i] = text.TextFragment{Text: "word"}
					}
				}
				return frags
			}(),
			want: false, // 50% single chars, threshold is 60%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCharacterLevel(tt.fragments)
			if got != tt.want {
				t.Errorf("isCharacterLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// convertListType tests
// ============================================================================

func TestConvertListType(t *testing.T) {
	tests := []struct {
		input layout.ListType
		want  model.ListType
	}{
		{layout.ListTypeBullet, model.ListTypeBullet},
		{layout.ListTypeNumbered, model.ListTypeNumbered},
		{layout.ListTypeLettered, model.ListTypeLettered},
		{layout.ListTypeRoman, model.ListTypeRoman},
		{layout.ListTypeCheckbox, model.ListTypeCheckbox},
		{layout.ListType(99), model.ListTypeUnknown},
	}

	for _, tt := range tests {
		got := convertListType(tt.input)
		if got != tt.want {
			t.Errorf("convertListType(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ============================================================================
// detectMultiColumn tests
// ============================================================================

func TestDetectMultiColumn(t *testing.T) {
	// Test with too few fragments
	result := detectMultiColumn(nil, 612, 792)
	if result {
		t.Error("expected false for nil fragments")
	}

	// Test with zero page width
	frags := make([]text.TextFragment, 25)
	result = detectMultiColumn(frags, 0, 792)
	if result {
		t.Error("expected false for zero page width")
	}

	// Test with small number of fragments
	smallFrags := make([]text.TextFragment, 10)
	result = detectMultiColumn(smallFrags, 612, 792)
	if result {
		t.Error("expected false for too few fragments")
	}
}

// ============================================================================
// ToMarkdownWithOptions tests
// ============================================================================

func TestToMarkdownWithOptions_PDF(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	opts := rag.MarkdownOptions{
		IncludeMetadata:        true,
		IncludeTableOfContents: false,
	}

	md, warnings, err := Open(pdfPath).Pages(1).ToMarkdownWithOptions(opts)
	if err != nil {
		t.Fatalf("ToMarkdownWithOptions() error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
}

func TestToMarkdownWithOptions_DOCX(t *testing.T) {
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")

	content := `<w:p>
  <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
  <w:r><w:t>Main Title</w:t></w:r>
</w:p>
<w:p><w:r><w:t>Document content here.</w:t></w:r></w:p>`

	if err := createMinimalDOCXWithContent(docxPath, content); err != nil {
		t.Fatalf("failed to create test DOCX: %v", err)
	}

	opts := rag.MarkdownOptions{
		IncludeMetadata:        true,
		IncludeTableOfContents: true,
	}

	md, warnings, err := Open(docxPath).ToMarkdownWithOptions(opts)
	if err != nil {
		t.Fatalf("ToMarkdownWithOptions() error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
}

func TestToMarkdownWithOptions_Error(t *testing.T) {
	_, _, err := Open("nonexistent.pdf").ToMarkdownWithOptions(rag.MarkdownOptions{})
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestToMarkdown_Basic(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	md, warnings, err := Open(pdfPath).Pages(1).ToMarkdown()
	if err != nil {
		t.Fatalf("ToMarkdown() error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
}

// ============================================================================
// Close tests
// ============================================================================

func TestClose_MultipleCalls(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	ext := Open(pdfPath)

	// First close should work
	err := ext.Close()
	if err != nil {
		t.Errorf("first Close() error = %v", err)
	}

	// Second close should also work (be idempotent)
	err = ext.Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestClose_AfterText(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	ext := Open(pdfPath)
	_, _, _ = ext.Text()

	// Close after terminal operation
	err := ext.Close()
	if err != nil {
		t.Errorf("Close() after Text() error = %v", err)
	}
}

func TestClose_WithoutOpen(t *testing.T) {
	ext := Open("nonexistent.pdf")
	// Close should not panic even if file was never opened
	err := ext.Close()
	// May or may not have error depending on implementation
	t.Logf("Close() on non-existent file: %v", err)
}

// ============================================================================
// PageCount tests
// ============================================================================

func TestPageCount_PDF(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	ext := Open(pdfPath)
	defer ext.Close()

	count, err := ext.PageCount()
	if err != nil {
		t.Fatalf("PageCount() error = %v", err)
	}

	if count <= 0 {
		t.Errorf("PageCount() = %d, want > 0", count)
	}
	t.Logf("PDF has %d pages", count)
}

func TestPageCount_Error(t *testing.T) {
	ext := Open("nonexistent.pdf")
	_, err := ext.PageCount()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// ============================================================================
// Document tests
// ============================================================================

func TestDocument_PDF(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	doc, warnings, err := Open(pdfPath).Pages(1).Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if doc == nil {
		t.Fatal("Document() returned nil")
	}

	if doc.PageCount() == 0 {
		t.Error("expected at least one page")
	}
}

func TestDocument_Error(t *testing.T) {
	_, _, err := Open("nonexistent.pdf").Document()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// ============================================================================
// Fragments tests
// ============================================================================

func TestFragments_Detailed(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	frags, warnings, err := Open(pdfPath).Pages(1).Fragments()
	if err != nil {
		t.Fatalf("Fragments() error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if len(frags) == 0 {
		t.Error("expected non-empty fragments")
	}

	// Verify fragment structure
	for i, frag := range frags[:minInt(5, len(frags))] {
		t.Logf("Fragment %d: text=%q, x=%.1f, y=%.1f", i, frag.Text, frag.X, frag.Y)
	}
}

func TestFragments_Error(t *testing.T) {
	_, _, err := Open("nonexistent.pdf").Fragments()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// ============================================================================
// Chunks additional tests
// ============================================================================

func TestChunks_Detailed(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	result, warnings, err := Open(pdfPath).Pages(1).Chunks()
	if err != nil {
		t.Fatalf("Chunks() error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if result == nil {
		t.Fatal("Chunks() returned nil")
	}

	if len(result.Chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	t.Logf("Got %d chunks", len(result.Chunks))
}

func TestChunks_Error(t *testing.T) {
	_, _, err := Open("nonexistent.pdf").Chunks()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestChunksWithConfig_Detailed(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	chunkerCfg := rag.DefaultChunkerConfig()
	chunkerCfg.MaxChunkSize = 500
	sizeCfg := rag.DefaultSizeConfig()

	result, warnings, err := Open(pdfPath).Pages(1).ChunksWithConfig(chunkerCfg, sizeCfg)
	if err != nil {
		t.Fatalf("ChunksWithConfig() error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if result == nil {
		t.Fatal("ChunksWithConfig() returned nil")
	}

	t.Logf("Got %d chunks with max size %d", len(result.Chunks), chunkerCfg.MaxChunkSize)
}

// ============================================================================
// Text extraction with different options
// ============================================================================

func TestText_WithByColumn(t *testing.T) {
	pdfPath := testPDFPath("3cols.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	text, warnings, err := Open(pdfPath).Pages(1).ByColumn().Text()
	if err != nil {
		t.Fatalf("Text() with ByColumn error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}
}

func TestText_WithPreserveLayout(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	text, warnings, err := Open(pdfPath).Pages(1).PreserveLayout().Text()
	if err != nil {
		t.Fatalf("Text() with PreserveLayout error = %v", err)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}
}

// ============================================================================
// Extractor method chaining tests
// ============================================================================

func TestMethodChaining(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Test chaining multiple methods
	text, _, err := Open(pdfPath).
		Pages(1).
		ExcludeHeaders().
		ExcludeFooters().
		JoinParagraphs().
		Text()

	if err != nil {
		t.Fatalf("Method chaining error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}
}

func TestMethodChaining_PageRange(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	text, _, err := Open(pdfPath).
		PageRange(1, 2).
		Text()

	if err != nil {
		t.Fatalf("PageRange chaining error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}
}

// ============================================================================
// min helper for tests
// ============================================================================

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================================
// XLSX Format Tests
// ============================================================================

func testXLSXPath(filename string) string {
	return filepath.Join("xlsx", "testdata", filename)
}

func TestXLSXTextExtraction(t *testing.T) {
	xlsxPath := testXLSXPath("simple.xlsx")
	if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
		t.Skip("test XLSX not found:", xlsxPath)
	}

	text, warnings, err := Open(xlsxPath).Text()
	if err != nil {
		t.Fatalf("XLSX text extraction error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text from XLSX")
	}

	// Warnings may or may not be present
	_ = warnings
}

func TestXLSXPageCount(t *testing.T) {
	xlsxPath := testXLSXPath("simple.xlsx")
	if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
		t.Skip("test XLSX not found:", xlsxPath)
	}

	count, err := Open(xlsxPath).PageCount()
	if err != nil {
		t.Fatalf("XLSX page count error = %v", err)
	}

	// XLSX should have at least 1 sheet
	if count < 1 {
		t.Errorf("expected at least 1 sheet, got %d", count)
	}
}

func TestXLSXDocument(t *testing.T) {
	xlsxPath := testXLSXPath("simple.xlsx")
	if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
		t.Skip("test XLSX not found:", xlsxPath)
	}

	doc, warnings, err := Open(xlsxPath).Document()
	if err != nil {
		t.Fatalf("XLSX document error = %v", err)
	}

	if doc == nil {
		t.Error("expected non-nil document")
	}

	if len(doc.Pages) == 0 {
		t.Error("expected at least one page (sheet)")
	}

	_ = warnings
}

func TestXLSXToMarkdown(t *testing.T) {
	xlsxPath := testXLSXPath("simple.xlsx")
	if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
		t.Skip("test XLSX not found:", xlsxPath)
	}

	md, warnings, err := Open(xlsxPath).ToMarkdown()
	if err != nil {
		t.Fatalf("XLSX to markdown error = %v", err)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown from XLSX")
	}

	_ = warnings
}

func TestXLSXChunks(t *testing.T) {
	xlsxPath := testXLSXPath("simple.xlsx")
	if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
		t.Skip("test XLSX not found:", xlsxPath)
	}

	chunks, warnings, err := Open(xlsxPath).Chunks()
	if err != nil {
		t.Fatalf("XLSX chunks error = %v", err)
	}

	if chunks == nil {
		t.Error("expected non-nil chunks")
	}

	_ = warnings
}

func TestXLSXClose(t *testing.T) {
	xlsxPath := testXLSXPath("simple.xlsx")
	if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
		t.Skip("test XLSX not found:", xlsxPath)
	}

	ext := Open(xlsxPath)
	_, _, _ = ext.Text() // Open and extract

	err := ext.Close()
	if err != nil {
		t.Fatalf("XLSX close error = %v", err)
	}

	// Second close should be safe
	err = ext.Close()
	if err != nil {
		t.Fatalf("XLSX second close error = %v", err)
	}
}

// ============================================================================
// PPTX Format Tests
// ============================================================================

func testPPTXPath(filename string) string {
	return filepath.Join("pptx", "testdata", filename)
}

func TestPPTXTextExtraction(t *testing.T) {
	pptxPath := testPPTXPath("test.pptx")
	if _, err := os.Stat(pptxPath); os.IsNotExist(err) {
		t.Skip("test PPTX not found:", pptxPath)
	}

	text, warnings, err := Open(pptxPath).Text()
	if err != nil {
		t.Fatalf("PPTX text extraction error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text from PPTX")
	}

	_ = warnings
}

func TestPPTXPageCount(t *testing.T) {
	pptxPath := testPPTXPath("test.pptx")
	if _, err := os.Stat(pptxPath); os.IsNotExist(err) {
		t.Skip("test PPTX not found:", pptxPath)
	}

	count, err := Open(pptxPath).PageCount()
	if err != nil {
		t.Fatalf("PPTX page count error = %v", err)
	}

	// PPTX should have at least 1 slide
	if count < 1 {
		t.Errorf("expected at least 1 slide, got %d", count)
	}
}

func TestPPTXDocument(t *testing.T) {
	pptxPath := testPPTXPath("test.pptx")
	if _, err := os.Stat(pptxPath); os.IsNotExist(err) {
		t.Skip("test PPTX not found:", pptxPath)
	}

	doc, warnings, err := Open(pptxPath).Document()
	if err != nil {
		t.Fatalf("PPTX document error = %v", err)
	}

	if doc == nil {
		t.Error("expected non-nil document")
	}

	if len(doc.Pages) == 0 {
		t.Error("expected at least one page (slide)")
	}

	_ = warnings
}

func TestPPTXToMarkdown(t *testing.T) {
	pptxPath := testPPTXPath("test.pptx")
	if _, err := os.Stat(pptxPath); os.IsNotExist(err) {
		t.Skip("test PPTX not found:", pptxPath)
	}

	md, warnings, err := Open(pptxPath).ToMarkdown()
	if err != nil {
		t.Fatalf("PPTX to markdown error = %v", err)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown from PPTX")
	}

	_ = warnings
}

func TestPPTXChunks(t *testing.T) {
	pptxPath := testPPTXPath("test.pptx")
	if _, err := os.Stat(pptxPath); os.IsNotExist(err) {
		t.Skip("test PPTX not found:", pptxPath)
	}

	chunks, warnings, err := Open(pptxPath).Chunks()
	if err != nil {
		t.Fatalf("PPTX chunks error = %v", err)
	}

	if chunks == nil {
		t.Error("expected non-nil chunks")
	}

	_ = warnings
}

func TestPPTXClose(t *testing.T) {
	pptxPath := testPPTXPath("test.pptx")
	if _, err := os.Stat(pptxPath); os.IsNotExist(err) {
		t.Skip("test PPTX not found:", pptxPath)
	}

	ext := Open(pptxPath)
	_, _, _ = ext.Text() // Open and extract

	err := ext.Close()
	if err != nil {
		t.Fatalf("PPTX close error = %v", err)
	}
}

func TestPPTXPageSelection(t *testing.T) {
	pptxPath := testPPTXPath("test.pptx")
	if _, err := os.Stat(pptxPath); os.IsNotExist(err) {
		t.Skip("test PPTX not found:", pptxPath)
	}

	// Extract only first slide
	text, _, err := Open(pptxPath).Pages(1).Text()
	if err != nil {
		t.Fatalf("PPTX page selection error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text from first slide")
	}
}

// ============================================================================
// ODT Format Tests (via main extractor API)
// ============================================================================

func testODTPath(filename string) string {
	return filepath.Join("odt", "testdata", filename)
}

func TestODTTextExtraction(t *testing.T) {
	odtPath := testODTPath("sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		t.Skip("test ODT not found:", odtPath)
	}

	text, warnings, err := Open(odtPath).Text()
	if err != nil {
		t.Fatalf("ODT text extraction error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text from ODT")
	}

	_ = warnings
}

func TestODTPageCount(t *testing.T) {
	odtPath := testODTPath("sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		t.Skip("test ODT not found:", odtPath)
	}

	count, err := Open(odtPath).PageCount()
	if err != nil {
		t.Fatalf("ODT page count error = %v", err)
	}

	// ODT should have at least 1 page
	if count < 1 {
		t.Errorf("expected at least 1 page, got %d", count)
	}
}

func TestODTDocument(t *testing.T) {
	odtPath := testODTPath("sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		t.Skip("test ODT not found:", odtPath)
	}

	doc, warnings, err := Open(odtPath).Document()
	if err != nil {
		t.Fatalf("ODT document error = %v", err)
	}

	if doc == nil {
		t.Error("expected non-nil document")
	}

	if len(doc.Pages) == 0 {
		t.Error("expected at least one page")
	}

	_ = warnings
}

func TestODTToMarkdown(t *testing.T) {
	odtPath := testODTPath("sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		t.Skip("test ODT not found:", odtPath)
	}

	md, warnings, err := Open(odtPath).ToMarkdown()
	if err != nil {
		t.Fatalf("ODT to markdown error = %v", err)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown from ODT")
	}

	_ = warnings
}

func TestODTChunks(t *testing.T) {
	odtPath := testODTPath("sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		t.Skip("test ODT not found:", odtPath)
	}

	chunks, warnings, err := Open(odtPath).Chunks()
	if err != nil {
		t.Fatalf("ODT chunks error = %v", err)
	}

	if chunks == nil {
		t.Error("expected non-nil chunks")
	}

	_ = warnings
}

func TestODTClose(t *testing.T) {
	odtPath := testODTPath("sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		t.Skip("test ODT not found:", odtPath)
	}

	ext := Open(odtPath)
	_, _, _ = ext.Text() // Open and extract

	err := ext.Close()
	if err != nil {
		t.Fatalf("ODT close error = %v", err)
	}
}

// ============================================================================
// Unsupported Format Tests
// ============================================================================

func TestUnsupportedFormat(t *testing.T) {
	// Create a temp file with unsupported extension
	tmpFile, err := os.CreateTemp("", "test*.xyz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	_, _, extractErr := Open(tmpFile.Name()).Text()
	if extractErr == nil {
		t.Error("expected error for unsupported format")
	}
	if !strings.Contains(extractErr.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error, got: %v", extractErr)
	}
}

func TestEmptyFilename(t *testing.T) {
	ext := &Extractor{}
	_, _, err := ext.Text()
	if err == nil {
		t.Error("expected error for empty filename")
	}
}

// ============================================================================
// Error Propagation Tests
// ============================================================================

func TestErrorPropagation_InChain(t *testing.T) {
	// Start with a file that doesn't exist
	ext := Open("nonexistent_file.pdf")

	// Chain multiple methods - error should propagate
	result := ext.Pages(1, 2).ExcludeHeaders().JoinParagraphs()

	// The error should be captured when we call a terminal operation
	_, _, err := result.Text()
	if err == nil {
		t.Error("expected error to propagate through chain")
	}
}

func TestResolvePages_SinglePage(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Single page request
	text, _, err := Open(pdfPath).Pages(1).Text()
	if err != nil {
		t.Fatalf("single page error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text from single page")
	}
}

func TestResolvePages_MultiplePages(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Multiple specific pages
	text, _, err := Open(pdfPath).Pages(1, 2).Text()
	if err != nil {
		t.Fatalf("multiple pages error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text from multiple pages")
	}
}

func TestResolvePages_PageRange(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Page range
	text, _, err := Open(pdfPath).PageRange(1, 3).Text()
	if err != nil {
		t.Fatalf("page range error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text from page range")
	}
}

// ============================================================================
// extractByColumn Tests
// ============================================================================

func TestExtractByColumn_PDF(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	text, _, err := Open(pdfPath).ByColumn().Text()
	if err != nil {
		t.Fatalf("ByColumn extraction error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text with ByColumn")
	}
}

// ============================================================================
// detectHeaderFooter Tests
// ============================================================================

func TestDetectHeaderFooter_WithExclusion(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Test with header exclusion
	text1, _, err := Open(pdfPath).ExcludeHeaders().Text()
	if err != nil {
		t.Fatalf("ExcludeHeaders error = %v", err)
	}

	// Test without exclusion
	text2, _, err := Open(pdfPath).Text()
	if err != nil {
		t.Fatalf("normal extraction error = %v", err)
	}

	// Both should have content
	if len(text1) == 0 || len(text2) == 0 {
		t.Error("expected non-empty text in both cases")
	}
}

// ============================================================================
// collectAllPages Tests
// ============================================================================

func TestCollectAllPages_PDF(t *testing.T) {
	pdfPath := testPDFPath("dinosaurs.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("test PDF not found:", pdfPath)
	}

	// Get fragments (which internally collects pages)
	frags, _, err := Open(pdfPath).Fragments()
	if err != nil {
		t.Fatalf("Fragments error = %v", err)
	}

	if len(frags) == 0 {
		t.Error("expected non-empty fragments")
	}
}

// ============================================================================
// XLSX with Options Tests
// ============================================================================

func TestXLSXToMarkdownWithOptions(t *testing.T) {
	xlsxPath := testXLSXPath("simple.xlsx")
	if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
		t.Skip("test XLSX not found:", xlsxPath)
	}

	opts := rag.MarkdownOptions{
		IncludeMetadata:        true,
		IncludeTableOfContents: false,
	}

	md, warnings, err := Open(xlsxPath).ToMarkdownWithOptions(opts)
	if err != nil {
		t.Fatalf("XLSX ToMarkdownWithOptions error = %v", err)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}

	_ = warnings
}

func TestXLSXChunksWithConfig(t *testing.T) {
	xlsxPath := testXLSXPath("simple.xlsx")
	if _, err := os.Stat(xlsxPath); os.IsNotExist(err) {
		t.Skip("test XLSX not found:", xlsxPath)
	}

	cfg := rag.DefaultChunkerConfig()
	sizeCfg := rag.DefaultSizeConfig()

	chunks, warnings, err := Open(xlsxPath).ChunksWithConfig(cfg, sizeCfg)
	if err != nil {
		t.Fatalf("XLSX ChunksWithConfig error = %v", err)
	}

	if chunks == nil {
		t.Error("expected non-nil chunks")
	}

	_ = warnings
}

// ============================================================================
// PPTX with Options Tests
// ============================================================================

func TestPPTXToMarkdownWithOptions(t *testing.T) {
	pptxPath := testPPTXPath("test.pptx")
	if _, err := os.Stat(pptxPath); os.IsNotExist(err) {
		t.Skip("test PPTX not found:", pptxPath)
	}

	opts := rag.MarkdownOptions{
		IncludeMetadata:        true,
		IncludeTableOfContents: false,
	}

	md, warnings, err := Open(pptxPath).ToMarkdownWithOptions(opts)
	if err != nil {
		t.Fatalf("PPTX ToMarkdownWithOptions error = %v", err)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}

	_ = warnings
}

func TestPPTXChunksWithConfig(t *testing.T) {
	pptxPath := testPPTXPath("test.pptx")
	if _, err := os.Stat(pptxPath); os.IsNotExist(err) {
		t.Skip("test PPTX not found:", pptxPath)
	}

	cfg := rag.DefaultChunkerConfig()
	sizeCfg := rag.DefaultSizeConfig()

	chunks, warnings, err := Open(pptxPath).ChunksWithConfig(cfg, sizeCfg)
	if err != nil {
		t.Fatalf("PPTX ChunksWithConfig error = %v", err)
	}

	if chunks == nil {
		t.Error("expected non-nil chunks")
	}

	_ = warnings
}

// ============================================================================
// ODT with Options Tests
// ============================================================================

func TestODTToMarkdownWithOptions(t *testing.T) {
	odtPath := testODTPath("sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		t.Skip("test ODT not found:", odtPath)
	}

	opts := rag.MarkdownOptions{
		IncludeMetadata:        true,
		IncludeTableOfContents: false,
	}

	md, warnings, err := Open(odtPath).ToMarkdownWithOptions(opts)
	if err != nil {
		t.Fatalf("ODT ToMarkdownWithOptions error = %v", err)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}

	_ = warnings
}

func TestODTChunksWithConfig(t *testing.T) {
	odtPath := testODTPath("sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		t.Skip("test ODT not found:", odtPath)
	}

	cfg := rag.DefaultChunkerConfig()
	sizeCfg := rag.DefaultSizeConfig()

	chunks, warnings, err := Open(odtPath).ChunksWithConfig(cfg, sizeCfg)
	if err != nil {
		t.Fatalf("ODT ChunksWithConfig error = %v", err)
	}

	if chunks == nil {
		t.Error("expected non-nil chunks")
	}

	_ = warnings
}
