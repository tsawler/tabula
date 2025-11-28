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
