package docx

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
