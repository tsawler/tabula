package tabula

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
