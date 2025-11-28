package docx

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestDOCXWithTable creates a DOCX file with a simple table.
func createTestDOCXWithTable(t *testing.T, tableXML string) string {
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

	// word/document.xml with table
	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>` + tableXML + `</w:body>
</w:document>`
	w, _ = zw.Create("word/document.xml")
	w.Write([]byte(document))

	zw.Close()
	f.Close()

	return docxPath
}

func TestTableParsing_Simple(t *testing.T) {
	// Simple 2x2 table
	tableXML := `
<w:tbl>
  <w:tblPr>
    <w:tblBorders>
      <w:top w:val="single"/>
      <w:bottom w:val="single"/>
    </w:tblBorders>
  </w:tblPr>
  <w:tblGrid>
    <w:gridCol w:w="2880"/>
    <w:gridCol w:w="2880"/>
  </w:tblGrid>
  <w:tr>
    <w:tc>
      <w:p><w:r><w:t>Header 1</w:t></w:r></w:p>
    </w:tc>
    <w:tc>
      <w:p><w:r><w:t>Header 2</w:t></w:r></w:p>
    </w:tc>
  </w:tr>
  <w:tr>
    <w:tc>
      <w:p><w:r><w:t>Cell A</w:t></w:r></w:p>
    </w:tc>
    <w:tc>
      <w:p><w:r><w:t>Cell B</w:t></w:r></w:p>
    </w:tc>
  </w:tr>
</w:tbl>`

	docxPath := createTestDOCXWithTable(t, tableXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	tables := r.Tables()
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	table := tables[0]
	if len(table.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(table.Rows))
	}

	if len(table.Rows[0].Cells) != 2 {
		t.Errorf("expected 2 cells in row 0, got %d", len(table.Rows[0].Cells))
	}

	// Check cell content
	if table.Rows[0].Cells[0].Text != "Header 1" {
		t.Errorf("cell[0][0] = %q, want 'Header 1'", table.Rows[0].Cells[0].Text)
	}
	if table.Rows[1].Cells[1].Text != "Cell B" {
		t.Errorf("cell[1][1] = %q, want 'Cell B'", table.Rows[1].Cells[1].Text)
	}
}

func TestTableParsing_ColSpan(t *testing.T) {
	// Table with column span
	tableXML := `
<w:tbl>
  <w:tblGrid>
    <w:gridCol w:w="2000"/>
    <w:gridCol w:w="2000"/>
    <w:gridCol w:w="2000"/>
  </w:tblGrid>
  <w:tr>
    <w:tc>
      <w:tcPr>
        <w:gridSpan w:val="2"/>
      </w:tcPr>
      <w:p><w:r><w:t>Merged Header</w:t></w:r></w:p>
    </w:tc>
    <w:tc>
      <w:p><w:r><w:t>Single</w:t></w:r></w:p>
    </w:tc>
  </w:tr>
  <w:tr>
    <w:tc>
      <w:p><w:r><w:t>A</w:t></w:r></w:p>
    </w:tc>
    <w:tc>
      <w:p><w:r><w:t>B</w:t></w:r></w:p>
    </w:tc>
    <w:tc>
      <w:p><w:r><w:t>C</w:t></w:r></w:p>
    </w:tc>
  </w:tr>
</w:tbl>`

	docxPath := createTestDOCXWithTable(t, tableXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	tables := r.Tables()
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	table := tables[0]

	// First row should have 2 cells (one spanning 2 columns)
	if len(table.Rows[0].Cells) != 2 {
		t.Errorf("expected 2 cells in row 0, got %d", len(table.Rows[0].Cells))
	}

	// Check column span
	if table.Rows[0].Cells[0].ColSpan != 2 {
		t.Errorf("cell[0][0].ColSpan = %d, want 2", table.Rows[0].Cells[0].ColSpan)
	}
}

func TestTableParsing_ToModelTable(t *testing.T) {
	tableXML := `
<w:tbl>
  <w:tblGrid>
    <w:gridCol w:w="2880"/>
    <w:gridCol w:w="2880"/>
  </w:tblGrid>
  <w:tr>
    <w:trPr><w:tblHeader/></w:trPr>
    <w:tc>
      <w:p><w:r><w:t>Name</w:t></w:r></w:p>
    </w:tc>
    <w:tc>
      <w:p><w:r><w:t>Value</w:t></w:r></w:p>
    </w:tc>
  </w:tr>
  <w:tr>
    <w:tc>
      <w:p><w:r><w:t>Foo</w:t></w:r></w:p>
    </w:tc>
    <w:tc>
      <w:p><w:r><w:t>Bar</w:t></w:r></w:p>
    </w:tc>
  </w:tr>
</w:tbl>`

	docxPath := createTestDOCXWithTable(t, tableXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	modelTables := r.ModelTables()
	if len(modelTables) != 1 {
		t.Fatalf("expected 1 model table, got %d", len(modelTables))
	}

	table := modelTables[0]

	if table.RowCount() != 2 {
		t.Errorf("RowCount() = %d, want 2", table.RowCount())
	}

	if table.ColCount() != 2 {
		t.Errorf("ColCount() = %d, want 2", table.ColCount())
	}

	// Check header row
	cell := table.GetCell(0, 0)
	if cell == nil {
		t.Fatal("GetCell(0,0) returned nil")
	}
	if cell.Text != "Name" {
		t.Errorf("cell[0][0].Text = %q, want 'Name'", cell.Text)
	}
	if !cell.IsHeader {
		t.Error("cell[0][0].IsHeader should be true")
	}

	// Check data row
	cell = table.GetCell(1, 1)
	if cell == nil {
		t.Fatal("GetCell(1,1) returned nil")
	}
	if cell.Text != "Bar" {
		t.Errorf("cell[1][1].Text = %q, want 'Bar'", cell.Text)
	}
}

func TestTableParsing_CellAlignment(t *testing.T) {
	tableXML := `
<w:tbl>
  <w:tblGrid>
    <w:gridCol w:w="2880"/>
  </w:tblGrid>
  <w:tr>
    <w:tc>
      <w:tcPr>
        <w:vAlign w:val="center"/>
      </w:tcPr>
      <w:p><w:r><w:t>Centered</w:t></w:r></w:p>
    </w:tc>
  </w:tr>
</w:tbl>`

	docxPath := createTestDOCXWithTable(t, tableXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	tables := r.Tables()
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	cell := tables[0].Rows[0].Cells[0]
	if cell.VerticalAlign != "center" {
		t.Errorf("VerticalAlign = %q, want 'center'", cell.VerticalAlign)
	}
}

func TestTableParsing_EmptyTable(t *testing.T) {
	tableXML := `
<w:tbl>
  <w:tblGrid>
    <w:gridCol w:w="2880"/>
  </w:tblGrid>
</w:tbl>`

	docxPath := createTestDOCXWithTable(t, tableXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	tables := r.Tables()
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	if len(tables[0].Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(tables[0].Rows))
	}
}

func TestTableParser_ParseTwipsOrPercent(t *testing.T) {
	tests := []struct {
		value     string
		widthType string
		want      float64
	}{
		{"2880", "dxa", 144},      // 2880 twips = 144pt
		{"5000", "pct", 100},      // 5000 = 100% (stored as 50ths)
		{"0", "auto", 0},
		{"1440", "", 72},          // Default to twips
	}

	for _, tt := range tests {
		t.Run(tt.value+"_"+tt.widthType, func(t *testing.T) {
			got := parseTwipsOrPercent(tt.value, tt.widthType)
			if got != tt.want {
				t.Errorf("parseTwipsOrPercent(%q, %q) = %v, want %v", tt.value, tt.widthType, got, tt.want)
			}
		})
	}
}

func TestTableText(t *testing.T) {
	tableXML := `
<w:tbl>
  <w:tblGrid>
    <w:gridCol w:w="2880"/>
    <w:gridCol w:w="2880"/>
  </w:tblGrid>
  <w:tr>
    <w:tc><w:p><w:r><w:t>Name</w:t></w:r></w:p></w:tc>
    <w:tc><w:p><w:r><w:t>Value</w:t></w:r></w:p></w:tc>
  </w:tr>
  <w:tr>
    <w:tc><w:p><w:r><w:t>Foo</w:t></w:r></w:p></w:tc>
    <w:tc><w:p><w:r><w:t>Bar</w:t></w:r></w:p></w:tc>
  </w:tr>
</w:tbl>`

	docxPath := createTestDOCXWithTable(t, tableXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}

	// Should contain table content
	if !strings.Contains(text, "Name") {
		t.Error("Text() should contain 'Name'")
	}
	if !strings.Contains(text, "Foo") {
		t.Error("Text() should contain 'Foo'")
	}
	if !strings.Contains(text, "Bar") {
		t.Error("Text() should contain 'Bar'")
	}
}

func TestTableInDocument(t *testing.T) {
	tableXML := `
<w:p><w:r><w:t>Before table</w:t></w:r></w:p>
<w:tbl>
  <w:tblGrid>
    <w:gridCol w:w="2880"/>
    <w:gridCol w:w="2880"/>
  </w:tblGrid>
  <w:tr>
    <w:tc><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc>
    <w:tc><w:p><w:r><w:t>B</w:t></w:r></w:p></w:tc>
  </w:tr>
</w:tbl>
<w:p><w:r><w:t>After table</w:t></w:r></w:p>`

	docxPath := createTestDOCXWithTable(t, tableXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	doc, err := r.Document()
	if err != nil {
		t.Fatalf("Document() error = %v", err)
	}

	page := doc.GetPage(1)
	if page == nil {
		t.Fatal("GetPage(1) returned nil")
	}

	// Should have paragraphs and table
	if len(page.Elements) < 3 {
		t.Errorf("expected at least 3 elements, got %d", len(page.Elements))
	}
}

func TestTableOrderInText(t *testing.T) {
	// Test that table appears between paragraphs in text output
	tableXML := `
<w:p><w:r><w:t>Before table</w:t></w:r></w:p>
<w:tbl>
  <w:tblGrid>
    <w:gridCol w:w="2880"/>
    <w:gridCol w:w="2880"/>
  </w:tblGrid>
  <w:tr>
    <w:tc><w:p><w:r><w:t>TableCell</w:t></w:r></w:p></w:tc>
    <w:tc><w:p><w:r><w:t>Data</w:t></w:r></w:p></w:tc>
  </w:tr>
</w:tbl>
<w:p><w:r><w:t>After table</w:t></w:r></w:p>`

	docxPath := createTestDOCXWithTable(t, tableXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}

	// Find positions of each content
	beforeIdx := strings.Index(text, "Before table")
	tableIdx := strings.Index(text, "TableCell")
	afterIdx := strings.Index(text, "After table")

	if beforeIdx == -1 {
		t.Fatal("'Before table' not found in text")
	}
	if tableIdx == -1 {
		t.Fatal("'TableCell' not found in text")
	}
	if afterIdx == -1 {
		t.Fatal("'After table' not found in text")
	}

	// Verify order: Before < Table < After
	if beforeIdx >= tableIdx {
		t.Errorf("'Before table' (at %d) should appear before 'TableCell' (at %d)", beforeIdx, tableIdx)
	}
	if tableIdx >= afterIdx {
		t.Errorf("'TableCell' (at %d) should appear before 'After table' (at %d)", tableIdx, afterIdx)
	}
}
