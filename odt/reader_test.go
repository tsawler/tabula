package odt

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
