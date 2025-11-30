package odt

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsawler/tabula/rag"
)

// createTestODT creates a minimal ODT file for testing.
func createTestODT(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	odtPath := filepath.Join(tmpDir, "test.odt")

	f, err := os.Create(odtPath)
	if err != nil {
		t.Fatalf("failed to create ODT file: %v", err)
	}

	zw := zip.NewWriter(f)

	// Add mimetype file (must be first, uncompressed)
	mw, err := zw.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // No compression
	})
	if err != nil {
		t.Fatalf("failed to create mimetype: %v", err)
	}
	mw.Write([]byte("application/vnd.oasis.opendocument.text"))

	// Add content.xml
	cw, err := zw.Create("content.xml")
	if err != nil {
		t.Fatalf("failed to create content.xml: %v", err)
	}
	cw.Write([]byte(content))

	// Add empty styles.xml
	sw, err := zw.Create("styles.xml")
	if err != nil {
		t.Fatalf("failed to create styles.xml: %v", err)
	}
	sw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-styles xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0">
</office:document-styles>`))

	// Add meta.xml
	metaw, err := zw.Create("meta.xml")
	if err != nil {
		t.Fatalf("failed to create meta.xml: %v", err)
	}
	metaw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                      xmlns:dc="http://purl.org/dc/elements/1.1/"
                      xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0">
  <office:meta>
    <dc:title>Test Document</dc:title>
    <dc:creator>Test Author</dc:creator>
    <meta:generator>Test Generator</meta:generator>
  </office:meta>
</office:document-meta>`))

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close file: %v", err)
	}

	return odtPath
}

func TestOpenAndClose(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:p>Hello, World!</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestText(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:p>First paragraph</text:p>
      <text:p>Second paragraph</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text failed: %v", err)
	}

	if !strings.Contains(text, "First paragraph") {
		t.Errorf("expected 'First paragraph' in text, got: %s", text)
	}
	if !strings.Contains(text, "Second paragraph") {
		t.Errorf("expected 'Second paragraph' in text, got: %s", text)
	}
}

func TestHeadings(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:h text:outline-level="1">Main Title</text:h>
      <text:p>Some content</text:p>
      <text:h text:outline-level="2">Section</text:h>
      <text:p>More content</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	md, err := r.Markdown()
	if err != nil {
		t.Fatalf("Markdown failed: %v", err)
	}

	if !strings.Contains(md, "# Main Title") {
		t.Errorf("expected '# Main Title' in markdown, got: %s", md)
	}
	if !strings.Contains(md, "## Section") {
		t.Errorf("expected '## Section' in markdown, got: %s", md)
	}
}

func TestTables(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
                         xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0">
  <office:body>
    <office:text>
      <table:table table:name="TestTable">
        <table:table-column/>
        <table:table-column/>
        <table:table-row>
          <table:table-cell><text:p>A1</text:p></table:table-cell>
          <table:table-cell><text:p>B1</text:p></table:table-cell>
        </table:table-row>
        <table:table-row>
          <table:table-cell><text:p>A2</text:p></table:table-cell>
          <table:table-cell><text:p>B2</text:p></table:table-cell>
        </table:table-row>
      </table:table>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	tables := r.Tables()
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	tbl := tables[0]
	if len(tbl.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(tbl.Rows))
	}
	if len(tbl.Rows[0].Cells) != 2 {
		t.Errorf("expected 2 cells in first row, got %d", len(tbl.Rows[0].Cells))
	}

	// Check cell content
	if tbl.Rows[0].Cells[0].Text != "A1" {
		t.Errorf("expected 'A1', got '%s'", tbl.Rows[0].Cells[0].Text)
	}
	if tbl.Rows[1].Cells[1].Text != "B2" {
		t.Errorf("expected 'B2', got '%s'", tbl.Rows[1].Cells[1].Text)
	}
}

func TestMetadata(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:p>Hello</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	meta := r.Metadata()
	if meta.Title != "Test Document" {
		t.Errorf("expected title 'Test Document', got '%s'", meta.Title)
	}
	if meta.Author != "Test Author" {
		t.Errorf("expected author 'Test Author', got '%s'", meta.Author)
	}
}

func TestDocument(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:h text:outline-level="1">Title</text:h>
      <text:p>Content paragraph</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	doc, err := r.Document()
	if err != nil {
		t.Fatalf("Document failed: %v", err)
	}

	if len(doc.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(doc.Pages))
	}

	page := doc.Pages[0]
	if len(page.Elements) < 2 {
		t.Errorf("expected at least 2 elements, got %d", len(page.Elements))
	}
}

func TestPageCount(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:p>Hello</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	count, err := r.PageCount()
	if err != nil {
		t.Fatalf("PageCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected page count 1, got %d", count)
	}
}

// createTestODTWithHeadersFooters creates an ODT file with headers and footers in styles.xml.
func createTestODTWithHeadersFooters(t *testing.T, bodyContent, headerContent, footerContent string) string {
	t.Helper()

	tmpDir := t.TempDir()
	odtPath := filepath.Join(tmpDir, "test_with_hf.odt")

	f, err := os.Create(odtPath)
	if err != nil {
		t.Fatalf("failed to create ODT file: %v", err)
	}

	zw := zip.NewWriter(f)

	// Add mimetype file (must be first, uncompressed)
	mw, err := zw.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // No compression
	})
	if err != nil {
		t.Fatalf("failed to create mimetype: %v", err)
	}
	mw.Write([]byte("application/vnd.oasis.opendocument.text"))

	// Add content.xml
	contentXML := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>` + bodyContent + `</office:text>
  </office:body>
</office:document-content>`
	cw, err := zw.Create("content.xml")
	if err != nil {
		t.Fatalf("failed to create content.xml: %v", err)
	}
	cw.Write([]byte(contentXML))

	// Add styles.xml with headers and footers in master pages
	stylesXML := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-styles xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                        xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0"
                        xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:automatic-styles>
    <style:page-layout style:name="pm1">
      <style:page-layout-properties/>
      <style:header-style>
        <style:header-footer-properties fo:min-height="0.5in"/>
      </style:header-style>
      <style:footer-style>
        <style:header-footer-properties fo:min-height="0.5in"/>
      </style:footer-style>
    </style:page-layout>
  </office:automatic-styles>
  <office:master-styles>
    <style:master-page style:name="Standard" style:page-layout-name="pm1">
      <style:header>
        <text:p>` + headerContent + `</text:p>
      </style:header>
      <style:footer>
        <text:p>` + footerContent + `</text:p>
      </style:footer>
    </style:master-page>
  </office:master-styles>
</office:document-styles>`
	sw, err := zw.Create("styles.xml")
	if err != nil {
		t.Fatalf("failed to create styles.xml: %v", err)
	}
	sw.Write([]byte(stylesXML))

	// Add meta.xml
	metaw, err := zw.Create("meta.xml")
	if err != nil {
		t.Fatalf("failed to create meta.xml: %v", err)
	}
	metaw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                      xmlns:dc="http://purl.org/dc/elements/1.1/"
                      xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0">
  <office:meta>
    <dc:title>Test Document</dc:title>
  </office:meta>
</office:document-meta>`))

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close file: %v", err)
	}

	return odtPath
}

func TestReader_HeaderFooterParsing(t *testing.T) {
	bodyContent := `<text:p>Main document content</text:p>`
	headerContent := "Company Header"
	footerContent := "Page 1 of 10"

	odtPath := createTestODTWithHeadersFooters(t, bodyContent, headerContent, footerContent)

	r, err := Open(odtPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// Check that headers are detected
	if !r.HasHeaders() {
		t.Error("HasHeaders() should return true")
	}

	// Check that footers are detected
	if !r.HasFooters() {
		t.Error("HasFooters() should return true")
	}

	// Check header text
	headerTexts := r.HeaderTexts()
	if len(headerTexts) == 0 {
		t.Error("HeaderTexts() should not be empty")
	} else if !strings.Contains(headerTexts[0], headerContent) {
		t.Errorf("HeaderTexts() = %v, expected to contain %q", headerTexts, headerContent)
	}

	// Check footer text
	footerTexts := r.FooterTexts()
	if len(footerTexts) == 0 {
		t.Error("FooterTexts() should not be empty")
	} else if !strings.Contains(footerTexts[0], footerContent) {
		t.Errorf("FooterTexts() = %v, expected to contain %q", footerTexts, footerContent)
	}
}

func TestReader_TextWithOptions_ExcludeHeaders(t *testing.T) {
	// Create a document where the body contains the same text as the header
	headerText := "Company Header"
	bodyContent := `<text:p>Company Header</text:p>
<text:p>Main document content</text:p>`

	odtPath := createTestODTWithHeadersFooters(t, bodyContent, headerText, "Footer")

	r, err := Open(odtPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// Without exclusion, header text should appear
	textWithHeader, err := r.TextWithOptions(ExtractOptions{ExcludeHeaders: false})
	if err != nil {
		t.Fatalf("TextWithOptions() error = %v", err)
	}
	if !strings.Contains(textWithHeader, "Company Header") {
		t.Error("Text without exclusion should contain header text")
	}

	// With exclusion, header text should be removed
	textWithoutHeader, err := r.TextWithOptions(ExtractOptions{ExcludeHeaders: true})
	if err != nil {
		t.Fatalf("TextWithOptions() error = %v", err)
	}
	if strings.Contains(textWithoutHeader, "Company Header") {
		t.Error("Text with ExcludeHeaders=true should not contain header text")
	}
	if !strings.Contains(textWithoutHeader, "Main document content") {
		t.Error("Text should still contain main content")
	}
}

func TestReader_TextWithOptions_ExcludeFooters(t *testing.T) {
	// Create a document where the body contains the same text as the footer
	footerText := "Page 1 of 10"
	bodyContent := `<text:p>Main document content</text:p>
<text:p>Page 1 of 10</text:p>`

	odtPath := createTestODTWithHeadersFooters(t, bodyContent, "Header", footerText)

	r, err := Open(odtPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// Without exclusion, footer text should appear
	textWithFooter, err := r.TextWithOptions(ExtractOptions{ExcludeFooters: false})
	if err != nil {
		t.Fatalf("TextWithOptions() error = %v", err)
	}
	if !strings.Contains(textWithFooter, "Page 1 of 10") {
		t.Error("Text without exclusion should contain footer text")
	}

	// With exclusion, footer text should be removed
	textWithoutFooter, err := r.TextWithOptions(ExtractOptions{ExcludeFooters: true})
	if err != nil {
		t.Fatalf("TextWithOptions() error = %v", err)
	}
	if strings.Contains(textWithoutFooter, "Page 1 of 10") {
		t.Error("Text with ExcludeFooters=true should not contain footer text")
	}
	if !strings.Contains(textWithoutFooter, "Main document content") {
		t.Error("Text should still contain main content")
	}
}

func TestReader_TextWithOptions_ExcludeBoth(t *testing.T) {
	headerText := "Document Title"
	footerText := "Confidential"
	bodyContent := `<text:p>Document Title</text:p>
<text:p>Important content here</text:p>
<text:p>Confidential</text:p>`

	odtPath := createTestODTWithHeadersFooters(t, bodyContent, headerText, footerText)

	r, err := Open(odtPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// Exclude both headers and footers
	text, err := r.TextWithOptions(ExtractOptions{ExcludeHeaders: true, ExcludeFooters: true})
	if err != nil {
		t.Fatalf("TextWithOptions() error = %v", err)
	}

	if strings.Contains(text, "Document Title") {
		t.Error("Text should not contain header text")
	}
	if strings.Contains(text, "Confidential") {
		t.Error("Text should not contain footer text")
	}
	if !strings.Contains(text, "Important content here") {
		t.Error("Text should contain main content")
	}
}

func TestReader_MarkdownWithOptions_ExcludeHeadersFooters(t *testing.T) {
	headerText := "Report Header"
	footerText := "Report Footer"
	bodyContent := `<text:p>Report Header</text:p>
<text:p>The main report content</text:p>
<text:p>Report Footer</text:p>`

	odtPath := createTestODTWithHeadersFooters(t, bodyContent, headerText, footerText)

	r, err := Open(odtPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// With exclusion
	md, err := r.MarkdownWithOptions(ExtractOptions{ExcludeHeaders: true, ExcludeFooters: true})
	if err != nil {
		t.Fatalf("MarkdownWithOptions() error = %v", err)
	}

	if strings.Contains(md, "Report Header") {
		t.Error("Markdown should not contain header text")
	}
	if strings.Contains(md, "Report Footer") {
		t.Error("Markdown should not contain footer text")
	}
	if !strings.Contains(md, "The main report content") {
		t.Error("Markdown should contain main content")
	}
}

func TestReader_NoHeadersFooters(t *testing.T) {
	// Test a document without headers/footers (using the basic createTestODT)
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:p>Simple document</text:p>
    </office:text>
  </office:body>
</office:document-content>`
	odtPath := createTestODT(t, content)

	r, err := Open(odtPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	if r.HasHeaders() {
		t.Error("HasHeaders() should return false for document without headers")
	}
	if r.HasFooters() {
		t.Error("HasFooters() should return false for document without footers")
	}

	// TextWithOptions should still work
	text, err := r.TextWithOptions(ExtractOptions{ExcludeHeaders: true, ExcludeFooters: true})
	if err != nil {
		t.Fatalf("TextWithOptions() error = %v", err)
	}
	if !strings.Contains(text, "Simple document") {
		t.Error("Text should contain document content")
	}
}

// ============================================================================
// Lists tests
// ============================================================================

func TestLists(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:list text:style-name="L1">
        <text:list-item>
          <text:p>First item</text:p>
        </text:list-item>
        <text:list-item>
          <text:p>Second item</text:p>
        </text:list-item>
        <text:list-item>
          <text:p>Third item</text:p>
        </text:list-item>
      </text:list>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	lists := r.Lists()
	if len(lists) != 1 {
		t.Fatalf("expected 1 list, got %d", len(lists))
	}

	list := lists[0]
	if len(list.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(list.Items))
	}

	// Check text content
	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}
	if !strings.Contains(text, "First item") {
		t.Error("expected text to contain 'First item'")
	}
	if !strings.Contains(text, "Second item") {
		t.Error("expected text to contain 'Second item'")
	}
}

func TestListsMarkdown(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:p>Introduction:</text:p>
      <text:list text:style-name="L1">
        <text:list-item>
          <text:p>Item one</text:p>
        </text:list-item>
        <text:list-item>
          <text:p>Item two</text:p>
        </text:list-item>
      </text:list>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	md, err := r.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}

	// Should contain list markers (bullets)
	if !strings.Contains(md, "Item one") {
		t.Error("markdown should contain 'Item one'")
	}
	if !strings.Contains(md, "Item two") {
		t.Error("markdown should contain 'Item two'")
	}
}

// ============================================================================
// ModelTables tests
// ============================================================================

func TestModelTables(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
                         xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0">
  <office:body>
    <office:text>
      <table:table table:name="TestTable">
        <table:table-column/>
        <table:table-column/>
        <table:table-row>
          <table:table-cell><text:p>Name</text:p></table:table-cell>
          <table:table-cell><text:p>Value</text:p></table:table-cell>
        </table:table-row>
        <table:table-row>
          <table:table-cell><text:p>Alpha</text:p></table:table-cell>
          <table:table-cell><text:p>100</text:p></table:table-cell>
        </table:table-row>
      </table:table>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	modelTables := r.ModelTables()
	if len(modelTables) != 1 {
		t.Fatalf("expected 1 model table, got %d", len(modelTables))
	}

	tbl := modelTables[0]
	if tbl.RowCount() != 2 {
		t.Errorf("expected 2 rows, got %d", tbl.RowCount())
	}
	if tbl.ColCount() != 2 {
		t.Errorf("expected 2 cols, got %d", tbl.ColCount())
	}
}

// ============================================================================
// MarkdownWithRAGOptions tests
// ============================================================================

func TestMarkdownWithRAGOptions(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:h text:outline-level="1">Main Heading</text:h>
      <text:p>Some content here.</text:p>
      <text:h text:outline-level="2">Section One</text:h>
      <text:p>Section one content.</text:p>
      <text:h text:outline-level="2">Section Two</text:h>
      <text:p>Section two content.</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	t.Run("with metadata", func(t *testing.T) {
		md, err := r.MarkdownWithRAGOptions(
			ExtractOptions{},
			rag.MarkdownOptions{IncludeMetadata: true},
		)
		if err != nil {
			t.Fatalf("MarkdownWithRAGOptions() error = %v", err)
		}

		// Should have YAML front matter
		if !strings.Contains(md, "---") {
			t.Error("expected YAML front matter markers")
		}
		if !strings.Contains(md, "title:") {
			t.Error("expected title in metadata")
		}
	})

	t.Run("with table of contents", func(t *testing.T) {
		md, err := r.MarkdownWithRAGOptions(
			ExtractOptions{},
			rag.MarkdownOptions{IncludeTableOfContents: true},
		)
		if err != nil {
			t.Fatalf("MarkdownWithRAGOptions() error = %v", err)
		}

		if !strings.Contains(md, "Table of Contents") {
			t.Error("expected table of contents")
		}
		if !strings.Contains(md, "[Main Heading]") {
			t.Error("expected Main Heading in TOC")
		}
		if !strings.Contains(md, "[Section One]") {
			t.Error("expected Section One in TOC")
		}
	})

	t.Run("with heading level offset", func(t *testing.T) {
		md, err := r.MarkdownWithRAGOptions(
			ExtractOptions{},
			rag.MarkdownOptions{HeadingLevelOffset: 1},
		)
		if err != nil {
			t.Fatalf("MarkdownWithRAGOptions() error = %v", err)
		}

		// H1 should become H2
		if !strings.Contains(md, "## Main Heading") {
			t.Error("expected H1 to become H2 with offset")
		}
		// H2 should become H3
		if !strings.Contains(md, "### Section One") {
			t.Error("expected H2 to become H3 with offset")
		}
	})

	t.Run("with max heading level", func(t *testing.T) {
		md, err := r.MarkdownWithRAGOptions(
			ExtractOptions{},
			rag.MarkdownOptions{MaxHeadingLevel: 2},
		)
		if err != nil {
			t.Fatalf("MarkdownWithRAGOptions() error = %v", err)
		}

		// H1 should stay H1
		if !strings.Contains(md, "# Main Heading") {
			t.Error("expected H1 to remain H1")
		}
		// H2 should stay H2 (at max)
		if !strings.Contains(md, "## Section One") {
			t.Error("expected H2 to remain H2")
		}
	})

	t.Run("with exclude headers", func(t *testing.T) {
		// Create document with header text
		headerText := "Document Header"
		bodyContent := `<text:p>Document Header</text:p>
<text:p>Main content</text:p>`

		odtPath := createTestODTWithHeadersFooters(t, bodyContent, headerText, "Footer")

		r2, err := Open(odtPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer r2.Close()

		md, err := r2.MarkdownWithRAGOptions(
			ExtractOptions{ExcludeHeaders: true},
			rag.MarkdownOptions{},
		)
		if err != nil {
			t.Fatalf("MarkdownWithRAGOptions() error = %v", err)
		}

		if strings.Contains(md, "Document Header") {
			t.Error("expected header text to be excluded")
		}
		if !strings.Contains(md, "Main content") {
			t.Error("expected main content to be included")
		}
	})
}

// ============================================================================
// Error handling tests
// ============================================================================

func TestOpenError_NonExistent(t *testing.T) {
	_, err := Open("nonexistent.odt")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestOpenError_InvalidZip(t *testing.T) {
	// Create an invalid file (not a zip)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.odt")
	if err := os.WriteFile(path, []byte("not a zip file"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	_, err := Open(path)
	if err == nil {
		t.Error("expected error for invalid zip file")
	}
}

func TestOpenError_MissingContentXML(t *testing.T) {
	// Create a zip without content.xml
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "missing_content.odt")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	zw := zip.NewWriter(f)
	mw, _ := zw.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	mw.Write([]byte("application/vnd.oasis.opendocument.text"))
	zw.Close()
	f.Close()

	_, err = Open(path)
	if err == nil {
		t.Error("expected error for missing content.xml")
	}
}

func TestCloseMultipleTimes(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:p>Hello</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// First close should succeed
	if err := r.Close(); err != nil {
		t.Errorf("first Close failed: %v", err)
	}

	// Second close should not fail (already closed)
	if err := r.Close(); err != nil {
		t.Errorf("second Close failed: %v", err)
	}
}

// ============================================================================
// Integration tests with real ODT files
// ============================================================================

func TestIntegration_RealODT(t *testing.T) {
	odtPath := filepath.Join("testdata", "sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		t.Skip("test ODT not found:", odtPath)
	}

	r, err := Open(odtPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// Test PageCount
	count, err := r.PageCount()
	if err != nil {
		t.Fatalf("PageCount() error = %v", err)
	}
	if count != 1 {
		t.Errorf("expected page count 1, got %d", count)
	}

	// Test Text extraction
	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}
	if len(text) == 0 {
		t.Error("expected non-empty text")
	}
	t.Logf("Extracted %d characters of text", len(text))

	// Test Markdown
	md, err := r.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
	t.Logf("Generated %d characters of markdown", len(md))

	// Test Metadata
	meta := r.Metadata()
	t.Logf("Metadata - Title: %q, Author: %q, Creator: %q", meta.Title, meta.Author, meta.Creator)

	// Test Document
	doc, err := r.Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}
	if len(doc.Pages) == 0 {
		t.Error("expected at least one page")
	}
	if len(doc.Pages[0].Elements) == 0 {
		t.Error("expected elements on page")
	}
	t.Logf("Document has %d elements on first page", len(doc.Pages[0].Elements))

	// Test Tables
	tables := r.Tables()
	t.Logf("Found %d tables", len(tables))

	// Test ModelTables
	modelTables := r.ModelTables()
	t.Logf("Converted %d model tables", len(modelTables))

	// Test Lists
	lists := r.Lists()
	t.Logf("Found %d lists", len(lists))
}

func TestIntegration_RealODT_MarkdownWithRAGOptions(t *testing.T) {
	odtPath := filepath.Join("testdata", "sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		t.Skip("test ODT not found:", odtPath)
	}

	r, err := Open(odtPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	// Test with all options enabled
	md, err := r.MarkdownWithRAGOptions(
		ExtractOptions{},
		rag.MarkdownOptions{
			IncludeMetadata:        true,
			IncludeTableOfContents: true,
			HeadingLevelOffset:     0,
		},
	)
	if err != nil {
		t.Fatalf("MarkdownWithRAGOptions() error = %v", err)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
	t.Logf("Generated %d characters with RAG options", len(md))
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkOpen(b *testing.B) {
	odtPath := filepath.Join("testdata", "sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		b.Skip("test ODT not found:", odtPath)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := Open(odtPath)
		if err != nil {
			b.Fatalf("Open failed: %v", err)
		}
		r.Close()
	}
}

func BenchmarkText(b *testing.B) {
	odtPath := filepath.Join("testdata", "sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		b.Skip("test ODT not found:", odtPath)
	}

	r, err := Open(odtPath)
	if err != nil {
		b.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Text()
	}
}

func BenchmarkMarkdown(b *testing.B) {
	odtPath := filepath.Join("testdata", "sample1.odt")
	if _, err := os.Stat(odtPath); os.IsNotExist(err) {
		b.Skip("test ODT not found:", odtPath)
	}

	r, err := Open(odtPath)
	if err != nil {
		b.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Markdown()
	}
}

// ============================================================================
// Additional tests for better coverage
// ============================================================================

func TestTextWithOptions_EmptyDocument(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	text, err := r.TextWithOptions(ExtractOptions{})
	if err != nil {
		t.Fatalf("TextWithOptions() error = %v", err)
	}
	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
}

func TestMarkdownWithOptions_EmptyDocument(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	md, err := r.MarkdownWithOptions(ExtractOptions{})
	if err != nil {
		t.Fatalf("MarkdownWithOptions() error = %v", err)
	}
	if md != "" {
		t.Errorf("expected empty markdown, got %q", md)
	}
}

func TestMarkdownWithRAGOptions_EmptyDocument(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	md, err := r.MarkdownWithRAGOptions(ExtractOptions{}, rag.MarkdownOptions{})
	if err != nil {
		t.Fatalf("MarkdownWithRAGOptions() error = %v", err)
	}
	if md != "" {
		t.Errorf("expected empty markdown, got %q", md)
	}
}

func TestDocument_WithTable(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
                         xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0">
  <office:body>
    <office:text>
      <text:p>Before table</text:p>
      <table:table table:name="TestTable">
        <table:table-column/>
        <table:table-row>
          <table:table-cell><text:p>Cell</text:p></table:table-cell>
        </table:table-row>
      </table:table>
      <text:p>After table</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	doc, err := r.Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	if len(doc.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(doc.Pages))
	}

	// Should have elements for paragraphs and table
	if len(doc.Pages[0].Elements) < 3 {
		t.Errorf("expected at least 3 elements (2 paragraphs + 1 table), got %d", len(doc.Pages[0].Elements))
	}
}

func TestDocument_WithList(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:p>Before list</text:p>
      <text:list text:style-name="L1">
        <text:list-item>
          <text:p>Item 1</text:p>
        </text:list-item>
        <text:list-item>
          <text:p>Item 2</text:p>
        </text:list-item>
      </text:list>
      <text:p>After list</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	doc, err := r.Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	if len(doc.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(doc.Pages))
	}

	// Should have paragraph and list elements
	if len(doc.Pages[0].Elements) < 2 {
		t.Errorf("expected at least 2 elements, got %d", len(doc.Pages[0].Elements))
	}
}

func TestMarkdownWithOptions_TableInContent(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
                         xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0">
  <office:body>
    <office:text>
      <text:p>Introduction</text:p>
      <table:table table:name="DataTable">
        <table:table-column/>
        <table:table-column/>
        <table:table-row>
          <table:table-cell><text:p>A</text:p></table:table-cell>
          <table:table-cell><text:p>B</text:p></table:table-cell>
        </table:table-row>
      </table:table>
      <text:p>Conclusion</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	md, err := r.MarkdownWithOptions(ExtractOptions{})
	if err != nil {
		t.Fatalf("MarkdownWithOptions() error = %v", err)
	}

	if !strings.Contains(md, "Introduction") {
		t.Error("expected 'Introduction' in markdown")
	}
	if !strings.Contains(md, "Conclusion") {
		t.Error("expected 'Conclusion' in markdown")
	}
	// Table should be present in markdown format
	if !strings.Contains(md, "|") {
		t.Error("expected table markers in markdown")
	}
}

func TestMarkdownWithRAGOptions_WithKeywords(t *testing.T) {
	// Create ODT with keywords in metadata
	tmpDir := t.TempDir()
	odtPath := filepath.Join(tmpDir, "keywords.odt")

	f, err := os.Create(odtPath)
	if err != nil {
		t.Fatalf("failed to create ODT file: %v", err)
	}

	zw := zip.NewWriter(f)

	// Add mimetype
	mw, _ := zw.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	mw.Write([]byte("application/vnd.oasis.opendocument.text"))

	// Add content.xml
	cw, _ := zw.Create("content.xml")
	cw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:h text:outline-level="1">Test Document</text:h>
      <text:p>Some content</text:p>
    </office:text>
  </office:body>
</office:document-content>`))

	// Add styles.xml
	sw, _ := zw.Create("styles.xml")
	sw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-styles xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0">
</office:document-styles>`))

	// Add meta.xml with keywords
	metaw, _ := zw.Create("meta.xml")
	metaw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                      xmlns:dc="http://purl.org/dc/elements/1.1/"
                      xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0">
  <office:meta>
    <dc:title>Document With Keywords</dc:title>
    <dc:creator>Author Name</dc:creator>
    <dc:subject>Test Subject</dc:subject>
    <meta:keyword>keyword1</meta:keyword>
    <meta:keyword>keyword2</meta:keyword>
    <meta:generator>Test Generator</meta:generator>
  </office:meta>
</office:document-meta>`))

	zw.Close()
	f.Close()

	r, err := Open(odtPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	md, err := r.MarkdownWithRAGOptions(
		ExtractOptions{},
		rag.MarkdownOptions{IncludeMetadata: true},
	)
	if err != nil {
		t.Fatalf("MarkdownWithRAGOptions() error = %v", err)
	}

	if !strings.Contains(md, "---") {
		t.Error("expected YAML front matter")
	}
	if !strings.Contains(md, "author:") {
		t.Error("expected author in metadata")
	}
	if !strings.Contains(md, "subject:") {
		t.Error("expected subject in metadata")
	}
}

func TestMarkdownWithRAGOptions_NegativeHeadingOffset(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:h text:outline-level="2">Heading Level 2</text:h>
      <text:p>Content</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	md, err := r.MarkdownWithRAGOptions(
		ExtractOptions{},
		rag.MarkdownOptions{HeadingLevelOffset: -1},
	)
	if err != nil {
		t.Fatalf("MarkdownWithRAGOptions() error = %v", err)
	}

	// H2 with -1 offset should become H1
	if !strings.Contains(md, "# Heading Level 2") {
		t.Errorf("expected H2 to become H1, got: %s", md)
	}
}

func TestTextWithTables(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
                         xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0">
  <office:body>
    <office:text>
      <table:table table:name="DataTable">
        <table:table-column/>
        <table:table-row>
          <table:table-cell><text:p>Cell Content</text:p></table:table-cell>
        </table:table-row>
      </table:table>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}

	if !strings.Contains(text, "Cell Content") {
		t.Error("expected 'Cell Content' in text output")
	}
}

func TestHeadingLevelClamping(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:h text:outline-level="9">Level 9 Heading</text:h>
      <text:p>Content</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	md, err := r.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}

	// Level 9 should be clamped to 6
	if !strings.Contains(md, "###### Level 9 Heading") {
		t.Errorf("expected H9 to be clamped to H6, got: %s", md)
	}
}

func TestHeadingInvalidLevel(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
                         xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
  <office:body>
    <office:text>
      <text:h text:outline-level="0">Level 0 Heading</text:h>
      <text:h text:outline-level="invalid">Invalid Level</text:h>
      <text:p>Content</text:p>
    </office:text>
  </office:body>
</office:document-content>`

	path := createTestODT(t, content)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	md, err := r.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}

	// Invalid/zero levels should default to level 1
	if !strings.Contains(md, "# Level 0 Heading") {
		t.Errorf("expected level 0 to become H1")
	}
	if !strings.Contains(md, "# Invalid Level") {
		t.Errorf("expected invalid level to become H1")
	}
}
