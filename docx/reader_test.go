package docx

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsawler/tabula/rag"
)

// createTestDOCX creates a minimal DOCX file for testing.
func createTestDOCX(t *testing.T, content string) string {
	t.Helper()

	// Create temp file
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")

	f, err := os.Create(docxPath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	zw := zip.NewWriter(f)

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

	zw.Close()
	f.Close()

	return docxPath
}

// createTestDOCXWithStyles creates a DOCX with styles.xml for heading detection.
func createTestDOCXWithStyles(t *testing.T, content, styles string) string {
	t.Helper()

	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")

	f, err := os.Create(docxPath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	zw := zip.NewWriter(f)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
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

	// word/styles.xml
	if styles != "" {
		stylesXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` + styles + `</w:styles>`
		w, _ = zw.Create("word/styles.xml")
		w.Write([]byte(stylesXML))
	}

	zw.Close()
	f.Close()

	return docxPath
}

func TestOpen(t *testing.T) {
	content := `<w:p><w:r><w:t>Hello World</w:t></w:r></w:p>`
	docxPath := createTestDOCX(t, content)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	// Verify reader is valid
	if r.document == nil {
		t.Error("document should not be nil")
	}
}

func TestOpen_NotFound(t *testing.T) {
	_, err := Open("/nonexistent/file.docx")
	if err == nil {
		t.Error("Open() should return error for nonexistent file")
	}
}

func TestOpen_InvalidZip(t *testing.T) {
	// Create a file that's not a valid ZIP
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.docx")
	os.WriteFile(invalidPath, []byte("not a zip file"), 0644)

	_, err := Open(invalidPath)
	if err == nil {
		t.Error("Open() should return error for invalid ZIP")
	}
}

func TestOpen_MissingDocumentXML(t *testing.T) {
	// Create a ZIP without word/document.xml
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "missing.docx")

	f, _ := os.Create(docxPath)
	zw := zip.NewWriter(f)

	// Only add content types
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
</Types>`
	w, _ := zw.Create("[Content_Types].xml")
	w.Write([]byte(contentTypes))

	zw.Close()
	f.Close()

	_, err := Open(docxPath)
	if err == nil {
		t.Error("Open() should return error when document.xml is missing")
	}
}

func TestReader_Text(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple paragraph",
			content:  `<w:p><w:r><w:t>Hello World</w:t></w:r></w:p>`,
			expected: "Hello World",
		},
		{
			name: "multiple paragraphs",
			content: `<w:p><w:r><w:t>First paragraph</w:t></w:r></w:p>
<w:p><w:r><w:t>Second paragraph</w:t></w:r></w:p>`,
			expected: "First paragraph\nSecond paragraph",
		},
		{
			name: "multiple runs",
			content: `<w:p>
  <w:r><w:t>Hello </w:t></w:r>
  <w:r><w:t>World</w:t></w:r>
</w:p>`,
			expected: "Hello World",
		},
		{
			name:     "empty document",
			content:  ``,
			expected: "",
		},
		{
			name:     "paragraph with no text",
			content:  `<w:p><w:r></w:r></w:p>`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docxPath := createTestDOCX(t, tt.content)

			r, err := Open(docxPath)
			if err != nil {
				t.Fatalf("Open() error = %v", err)
			}
			defer r.Close()

			text, err := r.Text()
			if err != nil {
				t.Fatalf("Text() error = %v", err)
			}

			if text != tt.expected {
				t.Errorf("Text() = %q, want %q", text, tt.expected)
			}
		})
	}
}

func TestReader_HeadingDetection(t *testing.T) {
	content := `<w:p>
  <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
  <w:r><w:t>Main Title</w:t></w:r>
</w:p>
<w:p><w:r><w:t>Regular paragraph</w:t></w:r></w:p>
<w:p>
  <w:pPr><w:pStyle w:val="Heading2"/></w:pPr>
  <w:r><w:t>Subtitle</w:t></w:r>
</w:p>`

	docxPath := createTestDOCX(t, content)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	// Check that headings are detected
	if len(r.paragraphs) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d", len(r.paragraphs))
	}

	if !r.paragraphs[0].IsHeading {
		t.Error("first paragraph should be detected as heading")
	}
	if r.paragraphs[0].Level != 1 {
		t.Errorf("first paragraph level = %d, want 1", r.paragraphs[0].Level)
	}

	if r.paragraphs[1].IsHeading {
		t.Error("second paragraph should not be a heading")
	}

	if !r.paragraphs[2].IsHeading {
		t.Error("third paragraph should be detected as heading")
	}
	if r.paragraphs[2].Level != 2 {
		t.Errorf("third paragraph level = %d, want 2", r.paragraphs[2].Level)
	}
}

func TestReader_PageCount(t *testing.T) {
	content := `<w:p><w:r><w:t>Hello World</w:t></w:r></w:p>`
	docxPath := createTestDOCX(t, content)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	count, err := r.PageCount()
	if err != nil {
		t.Fatalf("PageCount() error = %v", err)
	}

	// DOCX documents are treated as single page
	if count != 1 {
		t.Errorf("PageCount() = %d, want 1", count)
	}
}

func TestReader_Document(t *testing.T) {
	content := `<w:p>
  <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
  <w:r><w:t>Document Title</w:t></w:r>
</w:p>
<w:p><w:r><w:t>This is the body text.</w:t></w:r></w:p>`

	docxPath := createTestDOCX(t, content)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	doc, err := r.Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	if doc.PageCount() != 1 {
		t.Errorf("PageCount() = %d, want 1", doc.PageCount())
	}

	page := doc.GetPage(1)
	if page == nil {
		t.Fatal("GetPage(1) returned nil")
	}

	// Should have 2 elements (heading + paragraph)
	if len(page.Elements) != 2 {
		t.Errorf("Elements count = %d, want 2", len(page.Elements))
	}
}

func TestReader_Close(t *testing.T) {
	content := `<w:p><w:r><w:t>Hello World</w:t></w:r></w:p>`
	docxPath := createTestDOCX(t, content)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	// Close should not error
	if err := r.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Multiple closes should be safe
	if err := r.Close(); err != nil {
		t.Errorf("Close() second call error = %v", err)
	}
}

func TestIsHeadingStyle(t *testing.T) {
	tests := []struct {
		styleID   string
		isHeading bool
		level     int
	}{
		{"Heading1", true, 1},
		{"Heading2", true, 2},
		{"heading1", true, 1}, // case insensitive
		{"HEADING3", true, 3},
		{"Title", true, 1},
		{"Normal", false, 0},
		{"", false, 0},
		{"BodyText", false, 0},
	}

	r := &Reader{}
	for _, tt := range tests {
		t.Run(tt.styleID, func(t *testing.T) {
			isHeading, level := r.isHeadingStyle(tt.styleID)
			if isHeading != tt.isHeading {
				t.Errorf("isHeadingStyle(%q) isHeading = %v, want %v", tt.styleID, isHeading, tt.isHeading)
			}
			if level != tt.level {
				t.Errorf("isHeadingStyle(%q) level = %d, want %d", tt.styleID, level, tt.level)
			}
		})
	}
}

func TestParseOutlineLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"0", 0},
		{"1", 1},
		{"8", 8},
		{"9", -1}, // out of range
		{"", 0},
		{"abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseOutlineLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseOutlineLevel(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReader_TextWithSpecialCharacters(t *testing.T) {
	content := `<w:p><w:r><w:t xml:space="preserve">Hello   World</w:t></w:r></w:p>`
	docxPath := createTestDOCX(t, content)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}

	// Multiple spaces should be preserved
	if !strings.Contains(text, "   ") {
		t.Errorf("Text() = %q, expected preserved spaces", text)
	}
}

// createTestDOCXWithHeadersFooters creates a DOCX file with headers and footers for testing.
func createTestDOCXWithHeadersFooters(t *testing.T, bodyContent, headerContent, footerContent string) string {
	t.Helper()

	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test_with_hf.docx")

	f, err := os.Create(docxPath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	zw := zip.NewWriter(f)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>
  <Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>
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

	// word/_rels/document.xml.rels - includes relationships to header and footer
	docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>
</Relationships>`
	w, _ = zw.Create("word/_rels/document.xml.rels")
	w.Write([]byte(docRels))

	// word/document.xml
	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>` + bodyContent + `</w:body>
</w:document>`
	w, _ = zw.Create("word/document.xml")
	w.Write([]byte(document))

	// word/header1.xml
	header := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:p><w:r><w:t>` + headerContent + `</w:t></w:r></w:p>
</w:hdr>`
	w, _ = zw.Create("word/header1.xml")
	w.Write([]byte(header))

	// word/footer1.xml
	footer := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:p><w:r><w:t>` + footerContent + `</w:t></w:r></w:p>
</w:ftr>`
	w, _ = zw.Create("word/footer1.xml")
	w.Write([]byte(footer))

	zw.Close()
	f.Close()

	return docxPath
}

func TestReader_HeaderFooterParsing(t *testing.T) {
	bodyContent := `<w:p><w:r><w:t>Main document content</w:t></w:r></w:p>`
	headerContent := "Company Header"
	footerContent := "Page 1 of 10"

	docxPath := createTestDOCXWithHeadersFooters(t, bodyContent, headerContent, footerContent)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
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
	bodyContent := `<w:p><w:r><w:t>Company Header</w:t></w:r></w:p>
<w:p><w:r><w:t>Main document content</w:t></w:r></w:p>`

	docxPath := createTestDOCXWithHeadersFooters(t, bodyContent, headerText, "Footer")

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
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
	bodyContent := `<w:p><w:r><w:t>Main document content</w:t></w:r></w:p>
<w:p><w:r><w:t>Page 1 of 10</w:t></w:r></w:p>`

	docxPath := createTestDOCXWithHeadersFooters(t, bodyContent, "Header", footerText)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
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
	bodyContent := `<w:p><w:r><w:t>Document Title</w:t></w:r></w:p>
<w:p><w:r><w:t>Important content here</w:t></w:r></w:p>
<w:p><w:r><w:t>Confidential</w:t></w:r></w:p>`

	docxPath := createTestDOCXWithHeadersFooters(t, bodyContent, headerText, footerText)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
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
	bodyContent := `<w:p><w:r><w:t>Report Header</w:t></w:r></w:p>
<w:p><w:r><w:t>The main report content</w:t></w:r></w:p>
<w:p><w:r><w:t>Report Footer</w:t></w:r></w:p>`

	docxPath := createTestDOCXWithHeadersFooters(t, bodyContent, headerText, footerText)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
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
	// Test a document without headers/footers
	content := `<w:p><w:r><w:t>Simple document</w:t></w:r></w:p>`
	docxPath := createTestDOCX(t, content)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
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
// Integration Tests with Real DOCX Files
// ============================================================================

func testDOCXPath(filename string) string {
	return filepath.Join("testdata", filename)
}

func TestIntegration_RealDOCX_Text(t *testing.T) {
	docxPath := testDOCXPath("hills.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		t.Skip("test DOCX not found:", docxPath)
	}

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}

	if len(text) == 0 {
		t.Error("expected non-empty text")
	}
}

func TestIntegration_RealDOCX_Markdown(t *testing.T) {
	docxPath := testDOCXPath("hills.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		t.Skip("test DOCX not found:", docxPath)
	}

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	md, err := r.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
}

func TestIntegration_RealDOCX_Document(t *testing.T) {
	docxPath := testDOCXPath("hills.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		t.Skip("test DOCX not found:", docxPath)
	}

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	doc, err := r.Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	if doc == nil {
		t.Fatal("expected non-nil document")
	}

	if len(doc.Pages) == 0 {
		t.Error("expected at least one page")
	}
}

func TestIntegration_RealDOCX_WithTables(t *testing.T) {
	docxPath := testDOCXPath("table.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		t.Skip("test DOCX not found:", docxPath)
	}

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	// Test Tables method
	tables := r.Tables()
	if len(tables) == 0 {
		t.Error("expected at least one table")
	}

	// Test ModelTables method
	modelTables := r.ModelTables()
	if len(modelTables) == 0 {
		t.Error("expected at least one model table")
	}

	// Test text extraction includes table content
	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}
	if len(text) == 0 {
		t.Error("expected non-empty text")
	}
}

func TestIntegration_RealDOCX_Lists(t *testing.T) {
	docxPath := testDOCXPath("guests-of-the-nation.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		t.Skip("test DOCX not found:", docxPath)
	}

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	// Test Lists method
	lists := r.Lists()
	// Lists may or may not exist in this document
	_ = lists

	// Test text extraction
	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}
	if len(text) == 0 {
		t.Error("expected non-empty text")
	}
}

func TestIntegration_RealDOCX_Metadata(t *testing.T) {
	docxPath := testDOCXPath("hills.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		t.Skip("test DOCX not found:", docxPath)
	}

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	meta := r.Metadata()
	// Metadata may have values or be empty depending on the document
	_ = meta
}

func TestIntegration_RealDOCX_MarkdownWithRAGOptions(t *testing.T) {
	docxPath := testDOCXPath("hills.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		t.Skip("test DOCX not found:", docxPath)
	}

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	extractOpts := ExtractOptions{}
	mdOpts := rag.MarkdownOptions{
		IncludeMetadata:        true,
		IncludeTableOfContents: false,
	}

	md, err := r.MarkdownWithRAGOptions(extractOpts, mdOpts)
	if err != nil {
		t.Fatalf("MarkdownWithRAGOptions() error = %v", err)
	}

	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
}

// ============================================================================
// Additional Edge Case Tests
// ============================================================================

func TestReader_ModelLists(t *testing.T) {
	// Create a document with a list structure
	content := `<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl w:val="0"/>
      <w:numId w:val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>First item</w:t></w:r>
</w:p>
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl w:val="0"/>
      <w:numId w:val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>Second item</w:t></w:r>
</w:p>`

	docxPath := createTestDOCX(t, content)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	modelLists := r.ModelLists()
	// May or may not detect lists depending on numbering.xml presence
	_ = modelLists
}

func TestFormatNumber_NilResolver(t *testing.T) {
	// formatNumber with nil resolver should return decimal format
	result := formatNumber(1, "1", 0, nil)
	if result != "1" {
		t.Errorf("formatNumber(1, '1', 0, nil) = %q, want %q", result, "1")
	}

	result = formatNumber(10, "1", 0, nil)
	if result != "10" {
		t.Errorf("formatNumber(10, '1', 0, nil) = %q, want %q", result, "10")
	}
}

func TestToLowerLetter(t *testing.T) {
	tests := []struct {
		num      int
		expected string
	}{
		{1, "a"},
		{2, "b"},
		{26, "z"},
		{27, "aa"},
		{28, "ab"},
		{52, "az"},
		{53, "ba"},
	}

	for _, tt := range tests {
		result := toLowerLetter(tt.num)
		if result != tt.expected {
			t.Errorf("toLowerLetter(%d) = %q, want %q", tt.num, result, tt.expected)
		}
	}
}

func TestToUpperRoman(t *testing.T) {
	tests := []struct {
		num      int
		expected string
	}{
		{1, "I"},
		{2, "II"},
		{3, "III"},
		{4, "IV"},
		{5, "V"},
		{9, "IX"},
		{10, "X"},
		{40, "XL"},
		{50, "L"},
		{90, "XC"},
		{100, "C"},
		{400, "CD"},
		{500, "D"},
		{900, "CM"},
		{1000, "M"},
		{1994, "MCMXCIV"},
	}

	for _, tt := range tests {
		result := toUpperRoman(tt.num)
		if result != tt.expected {
			t.Errorf("toUpperRoman(%d) = %q, want %q", tt.num, result, tt.expected)
		}
	}
}

func TestToLowerRoman(t *testing.T) {
	tests := []struct {
		num      int
		expected string
	}{
		{1, "i"},
		{4, "iv"},
		{9, "ix"},
		{10, "x"},
	}

	for _, tt := range tests {
		result := toLowerRoman(tt.num)
		if result != tt.expected {
			t.Errorf("toLowerRoman(%d) = %q, want %q", tt.num, result, tt.expected)
		}
	}
}

func TestToUpperLetter(t *testing.T) {
	tests := []struct {
		num      int
		expected string
	}{
		{1, "A"},
		{26, "Z"},
		{27, "AA"},
	}

	for _, tt := range tests {
		result := toUpperLetter(tt.num)
		if result != tt.expected {
			t.Errorf("toUpperLetter(%d) = %q, want %q", tt.num, result, tt.expected)
		}
	}
}

func TestReader_GetListFormat(t *testing.T) {
	// Test via the reader's getListFormat method - when no numbering resolver
	// is present, it should return default unordered list format
	docxPath := testDOCXPath("hills.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		t.Skip("test DOCX not found:", docxPath)
	}

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	// getListFormat is exercised through Markdown() for documents with lists
	md, err := r.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}

	// Just ensure we got valid output
	if len(md) == 0 {
		t.Error("expected non-empty markdown")
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkOpen(b *testing.B) {
	docxPath := testDOCXPath("hills.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		b.Skip("test DOCX not found")
	}

	for i := 0; i < b.N; i++ {
		r, err := Open(docxPath)
		if err != nil {
			b.Fatal(err)
		}
		r.Close()
	}
}

func BenchmarkText(b *testing.B) {
	docxPath := testDOCXPath("hills.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		b.Skip("test DOCX not found")
	}

	r, err := Open(docxPath)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := r.Text()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarkdown(b *testing.B) {
	docxPath := testDOCXPath("hills.docx")
	if _, err := os.Stat(docxPath); os.IsNotExist(err) {
		b.Skip("test DOCX not found")
	}

	r, err := Open(docxPath)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := r.Markdown()
		if err != nil {
			b.Fatal(err)
		}
	}
}
