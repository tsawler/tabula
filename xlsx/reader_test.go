package xlsx

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/tsawler/tabula/rag"
)

// createTestXLSX creates a minimal valid XLSX file in memory for testing.
func createTestXLSX(t *testing.T, sheets map[string]string, sharedStrings []string) string {
	t.Helper()

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test-*.xlsx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	// Create ZIP writer
	f, err := os.Create(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>`
	writeZipFile(t, zw, "[Content_Types].xml", contentTypes)

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`
	writeZipFile(t, zw, "_rels/.rels", rels)

	// xl/_rels/workbook.xml.rels
	var sheetRels strings.Builder
	sheetRels.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>`)

	i := 2
	for name := range sheets {
		_ = name
		sheetRels.WriteString("\n  <Relationship Id=\"rId")
		sheetRels.WriteString(string(rune('0' + i)))
		sheetRels.WriteString("\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet\" Target=\"worksheets/sheet")
		sheetRels.WriteString(string(rune('0' + i - 1)))
		sheetRels.WriteString(".xml\"/>")
		i++
	}
	sheetRels.WriteString("\n</Relationships>")
	writeZipFile(t, zw, "xl/_rels/workbook.xml.rels", sheetRels.String())

	// xl/workbook.xml
	var workbook strings.Builder
	workbook.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets>`)

	i = 1
	for name := range sheets {
		workbook.WriteString("\n  <sheet name=\"")
		workbook.WriteString(name)
		workbook.WriteString("\" sheetId=\"")
		workbook.WriteString(string(rune('0' + i)))
		workbook.WriteString("\" r:id=\"rId")
		workbook.WriteString(string(rune('0' + i + 1)))
		workbook.WriteString("\"/>")
		i++
	}
	workbook.WriteString("\n</sheets>\n</workbook>")
	writeZipFile(t, zw, "xl/workbook.xml", workbook.String())

	// xl/sharedStrings.xml
	var ss strings.Builder
	ss.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="`)
	ss.WriteString(string(rune('0' + len(sharedStrings))))
	ss.WriteString(`" uniqueCount="`)
	ss.WriteString(string(rune('0' + len(sharedStrings))))
	ss.WriteString(`">`)
	for _, s := range sharedStrings {
		ss.WriteString("\n  <si><t>")
		ss.WriteString(s)
		ss.WriteString("</t></si>")
	}
	ss.WriteString("\n</sst>")
	writeZipFile(t, zw, "xl/sharedStrings.xml", ss.String())

	// xl/worksheets/sheet*.xml
	i = 1
	for _, content := range sheets {
		writeZipFile(t, zw, "xl/worksheets/sheet"+string(rune('0'+i))+".xml", content)
		i++
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	return tmpFile.Name()
}

func writeZipFile(t *testing.T, zw *zip.Writer, name, content string) {
	t.Helper()
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("Failed to create %s in zip: %v", name, err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write %s: %v", name, err)
	}
}

// createMinimalXLSX creates a minimal XLSX for basic testing.
func createMinimalXLSX(t *testing.T) string {
	t.Helper()

	sheetContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData>
  <row r="1">
    <c r="A1" t="s"><v>0</v></c>
    <c r="B1" t="s"><v>1</v></c>
    <c r="C1" t="s"><v>2</v></c>
  </row>
  <row r="2">
    <c r="A2"><v>1</v></c>
    <c r="B2"><v>2</v></c>
    <c r="C2"><v>3</v></c>
  </row>
  <row r="3">
    <c r="A3"><v>4</v></c>
    <c r="B3"><v>5</v></c>
    <c r="C3"><v>6</v></c>
  </row>
</sheetData>
</worksheet>`

	return createTestXLSX(t, map[string]string{"Sheet1": sheetContent}, []string{"Name", "Age", "Score"})
}

func TestOpen(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	if r.SheetCount() != 1 {
		t.Errorf("SheetCount() = %d, want 1", r.SheetCount())
	}
}

func TestOpen_NotFound(t *testing.T) {
	_, err := Open("/nonexistent/file.xlsx")
	if err == nil {
		t.Error("Open() expected error for nonexistent file")
	}
}

func TestOpen_InvalidZip(t *testing.T) {
	// Create a non-zip file
	tmpFile, err := os.CreateTemp("", "test-*.xlsx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.WriteString("not a zip file")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = Open(tmpFile.Name())
	if err == nil {
		t.Error("Open() expected error for invalid zip")
	}
}

func TestOpen_MissingWorkbook(t *testing.T) {
	// Create a zip without workbook.xml
	tmpFile, err := os.CreateTemp("", "test-*.xlsx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	f, _ := os.Create(tmpFile.Name())
	zw := zip.NewWriter(f)
	writeZipFile(t, zw, "[Content_Types].xml", "<Types/>")
	zw.Close()
	f.Close()
	defer os.Remove(tmpFile.Name())

	_, err = Open(tmpFile.Name())
	if err == nil {
		t.Error("Open() expected error for missing workbook.xml")
	}
}

func TestReader_Close(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// First close should succeed
	if err := r.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Second close should be safe (no-op)
	if err := r.Close(); err != nil {
		t.Errorf("Second Close() failed: %v", err)
	}
}

func TestReader_SheetCount(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	if got := r.SheetCount(); got != 1 {
		t.Errorf("SheetCount() = %d, want 1", got)
	}
}

func TestReader_SheetNames(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	names := r.SheetNames()
	if len(names) != 1 {
		t.Fatalf("SheetNames() returned %d names, want 1", len(names))
	}
	if names[0] != "Sheet1" {
		t.Errorf("SheetNames()[0] = %q, want 'Sheet1'", names[0])
	}
}

func TestReader_Sheet(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	// Valid index
	sheet, err := r.Sheet(0)
	if err != nil {
		t.Errorf("Sheet(0) failed: %v", err)
	}
	if sheet == nil {
		t.Error("Sheet(0) returned nil")
	}

	// Invalid index
	_, err = r.Sheet(-1)
	if err == nil {
		t.Error("Sheet(-1) expected error")
	}

	_, err = r.Sheet(100)
	if err == nil {
		t.Error("Sheet(100) expected error")
	}
}

func TestReader_SheetByName(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	// Valid name
	sheet, err := r.SheetByName("Sheet1")
	if err != nil {
		t.Errorf("SheetByName('Sheet1') failed: %v", err)
	}
	if sheet == nil {
		t.Error("SheetByName('Sheet1') returned nil")
	}

	// Invalid name
	_, err = r.SheetByName("NonExistent")
	if err == nil {
		t.Error("SheetByName('NonExistent') expected error")
	}
}

func TestReader_PageCount(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	count, err := r.PageCount()
	if err != nil {
		t.Errorf("PageCount() failed: %v", err)
	}
	if count != 1 {
		t.Errorf("PageCount() = %d, want 1", count)
	}
}

func TestReader_Text(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	text, err := r.Text()
	if err != nil {
		t.Errorf("Text() failed: %v", err)
	}

	// Check that shared strings are resolved
	if !strings.Contains(text, "Name") {
		t.Errorf("Text() missing 'Name', got: %s", text)
	}
	if !strings.Contains(text, "Age") {
		t.Errorf("Text() missing 'Age', got: %s", text)
	}
	if !strings.Contains(text, "Score") {
		t.Errorf("Text() missing 'Score', got: %s", text)
	}

	// Check numeric values
	if !strings.Contains(text, "1") {
		t.Errorf("Text() missing numeric values, got: %s", text)
	}
}

func TestReader_TextWithOptions(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	tests := []struct {
		name    string
		opts    ExtractOptions
		wantHas string
	}{
		{
			name:    "with headers",
			opts:    ExtractOptions{IncludeHeaders: true},
			wantHas: "=== Sheet1 ===",
		},
		{
			name:    "custom delimiter",
			opts:    ExtractOptions{Delimiter: ","},
			wantHas: ",",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := r.TextWithOptions(tt.opts)
			if err != nil {
				t.Errorf("TextWithOptions() failed: %v", err)
			}
			if !strings.Contains(text, tt.wantHas) {
				t.Errorf("TextWithOptions() missing %q, got: %s", tt.wantHas, text)
			}
		})
	}
}

func TestReader_Markdown(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	md, err := r.Markdown()
	if err != nil {
		t.Errorf("Markdown() failed: %v", err)
	}

	// Should have sheet heading
	if !strings.Contains(md, "## Sheet1") {
		t.Errorf("Markdown() missing sheet heading, got: %s", md)
	}

	// Should have table structure
	if !strings.Contains(md, "|") {
		t.Errorf("Markdown() missing table pipes, got: %s", md)
	}
	if !strings.Contains(md, "---|") {
		t.Errorf("Markdown() missing separator, got: %s", md)
	}
}

func TestReader_Metadata(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	meta := r.Metadata()
	// Basic metadata structure should exist (may be empty)
	_ = meta.Title
	_ = meta.Author
}

func TestReader_Document(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	doc, err := r.Document()
	if err != nil {
		t.Errorf("Document() failed: %v", err)
	}

	if doc == nil {
		t.Fatal("Document() returned nil")
	}

	if len(doc.Pages) != 1 {
		t.Errorf("Document has %d pages, want 1", len(doc.Pages))
	}
}

func TestReader_Tables(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	tables := r.Tables()
	if len(tables) != 1 {
		t.Fatalf("Tables() returned %d tables, want 1", len(tables))
	}

	table := tables[0]
	if table.Name != "Sheet1" {
		t.Errorf("Table.Name = %q, want 'Sheet1'", table.Name)
	}

	// Check headers (first row)
	if len(table.Headers) != 3 {
		t.Errorf("Table.Headers has %d columns, want 3", len(table.Headers))
	}

	// Check data rows
	if len(table.Rows) != 2 {
		t.Errorf("Table.Rows has %d rows, want 2", len(table.Rows))
	}
}

func TestSheet_Cell(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	sheet, _ := r.Sheet(0)

	tests := []struct {
		row, col int
		wantNil  bool
	}{
		{0, 0, false},
		{0, 1, false},
		{-1, 0, true},
		{0, -1, true},
		{100, 0, true},
		{0, 100, true},
	}

	for _, tt := range tests {
		cell := sheet.Cell(tt.row, tt.col)
		if tt.wantNil && cell != nil {
			t.Errorf("Cell(%d, %d) = %v, want nil", tt.row, tt.col, cell)
		}
		if !tt.wantNil && cell == nil {
			t.Errorf("Cell(%d, %d) = nil, want non-nil", tt.row, tt.col)
		}
	}
}

func TestSheet_CellByRef(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	sheet, _ := r.Sheet(0)

	tests := []struct {
		ref     string
		wantNil bool
	}{
		{"A1", false},
		{"B1", false},
		{"C1", false},
		{"Z99", true}, // Out of bounds
		{"", true},    // Invalid ref
		{"1A", true},  // Invalid ref format
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			cell := sheet.CellByRef(tt.ref)
			if tt.wantNil && cell != nil {
				t.Errorf("CellByRef(%q) = %v, want nil", tt.ref, cell)
			}
			if !tt.wantNil && cell == nil {
				t.Errorf("CellByRef(%q) = nil, want non-nil", tt.ref)
			}
		})
	}
}

func TestSheet_RowCount(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	sheet, _ := r.Sheet(0)
	if got := sheet.RowCount(); got != 3 {
		t.Errorf("RowCount() = %d, want 3", got)
	}
}

func TestSheet_ColCount(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	sheet, _ := r.Sheet(0)
	if got := sheet.ColCount(); got != 3 {
		t.Errorf("ColCount() = %d, want 3", got)
	}
}

func TestCell_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		cell Cell
		want bool
	}{
		{
			name: "empty type",
			cell: Cell{Type: CellTypeEmpty, Value: ""},
			want: true,
		},
		{
			name: "empty value",
			cell: Cell{Type: CellTypeString, Value: ""},
			want: true,
		},
		{
			name: "has value",
			cell: Cell{Type: CellTypeString, Value: "hello"},
			want: false,
		},
		{
			name: "number with value",
			cell: Cell{Type: CellTypeNumber, Value: "42"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cell.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsedTable_ToText(t *testing.T) {
	table := ParsedTable{
		Name:    "Test",
		Headers: []string{"A", "B", "C"},
		Rows: [][]string{
			{"1", "2", "3"},
			{"4", "5", "6"},
		},
	}

	text := table.ToText()

	if !strings.Contains(text, "A\tB\tC") {
		t.Errorf("ToText() missing headers, got: %s", text)
	}
	if !strings.Contains(text, "1\t2\t3") {
		t.Errorf("ToText() missing first row, got: %s", text)
	}
	if !strings.Contains(text, "4\t5\t6") {
		t.Errorf("ToText() missing second row, got: %s", text)
	}
}

func TestParsedTable_ToMarkdown(t *testing.T) {
	table := ParsedTable{
		Name:    "Test",
		Headers: []string{"A", "B", "C"},
		Rows: [][]string{
			{"1", "2", "3"},
		},
	}

	md := table.ToMarkdown()

	if !strings.Contains(md, "| A |") {
		t.Errorf("ToMarkdown() missing headers, got: %s", md)
	}
	if !strings.Contains(md, "|---|---|---|") {
		t.Errorf("ToMarkdown() missing separator, got: %s", md)
	}
	if !strings.Contains(md, "| 1 |") {
		t.Errorf("ToMarkdown() missing data, got: %s", md)
	}
}

func TestParsedTable_ToMarkdown_Empty(t *testing.T) {
	table := ParsedTable{Name: "Empty"}
	md := table.ToMarkdown()
	if md != "" {
		t.Errorf("ToMarkdown() for empty table = %q, want empty", md)
	}
}

func TestReader_MarkdownWithRAGOptions(t *testing.T) {
	path := createMinimalXLSX(t)
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	// Test with metadata
	md, err := r.MarkdownWithRAGOptions(
		ExtractOptions{},
		rag.MarkdownOptions{IncludeMetadata: true},
	)
	if err != nil {
		t.Errorf("MarkdownWithRAGOptions() failed: %v", err)
	}

	// Should have YAML front matter
	if !strings.Contains(md, "---") {
		t.Errorf("MarkdownWithRAGOptions() missing YAML front matter, got: %s", md)
	}
	if !strings.Contains(md, "sheets:") {
		t.Errorf("MarkdownWithRAGOptions() missing sheets metadata, got: %s", md)
	}

	// Test with table of contents (multiple sheets needed)
	// Create multi-sheet file for TOC test
	tmpFile, _ := os.CreateTemp("", "test-toc-*.xlsx")
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	f, _ := os.Create(tmpFile.Name())
	zw := zip.NewWriter(f)

	contentTypes := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`
	writeZipFile(t, zw, "[Content_Types].xml", contentTypes)
	writeZipFile(t, zw, "_rels/.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`)
	writeZipFile(t, zw, "xl/_rels/workbook.xml.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/></Relationships>`)
	writeZipFile(t, zw, "xl/workbook.xml", `<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="First Sheet" sheetId="1" r:id="rId2"/><sheet name="Second Sheet" sheetId="2" r:id="rId3"/></sheets></workbook>`)

	sheet := `<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1"><v>1</v></c></row></sheetData></worksheet>`
	writeZipFile(t, zw, "xl/worksheets/sheet1.xml", sheet)
	writeZipFile(t, zw, "xl/worksheets/sheet2.xml", sheet)
	zw.Close()
	f.Close()

	r2, err := Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("Open multi-sheet failed: %v", err)
	}
	defer r2.Close()

	md2, err := r2.MarkdownWithRAGOptions(
		ExtractOptions{},
		rag.MarkdownOptions{IncludeTableOfContents: true},
	)
	if err != nil {
		t.Errorf("MarkdownWithRAGOptions() with TOC failed: %v", err)
	}

	// Should have TOC with links
	if !strings.Contains(md2, "Table of Contents") {
		t.Errorf("MarkdownWithRAGOptions() missing TOC, got: %s", md2)
	}
	if !strings.Contains(md2, "[First Sheet]") {
		t.Errorf("MarkdownWithRAGOptions() missing sheet link in TOC, got: %s", md2)
	}
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"with|pipe", "with\\|pipe"},
		{"line\nbreak", "line break"},
		{"both|and\nnew", "both\\|and new"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := escapeMarkdown(tt.input); got != tt.want {
				t.Errorf("escapeMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Integration test with real XLSX file
func TestIntegration_RealXLSX(t *testing.T) {
	samplePath := filepath.Join("testdata", "simple.xlsx")

	r, err := Open(samplePath)
	if err != nil {
		t.Fatalf("Open(%s) failed: %v", samplePath, err)
	}
	defer r.Close()

	t.Run("SheetCount", func(t *testing.T) {
		count := r.SheetCount()
		if count < 1 {
			t.Errorf("SheetCount() = %d, want >= 1", count)
		}
		t.Logf("Sheet count: %d", count)
	})

	t.Run("SheetNames", func(t *testing.T) {
		names := r.SheetNames()
		if len(names) < 1 {
			t.Errorf("SheetNames() returned empty")
		}
		t.Logf("Sheet names: %v", names)
	})

	t.Run("Text", func(t *testing.T) {
		text, err := r.Text()
		if err != nil {
			t.Errorf("Text() failed: %v", err)
		}
		if text == "" {
			t.Error("Text() returned empty string")
		}
		t.Logf("Text length: %d chars", len(text))
		// Log first 500 chars for debugging
		if len(text) > 500 {
			t.Logf("Text preview: %s...", text[:500])
		} else {
			t.Logf("Text: %s", text)
		}
	})

	t.Run("Markdown", func(t *testing.T) {
		md, err := r.Markdown()
		if err != nil {
			t.Errorf("Markdown() failed: %v", err)
		}
		if md == "" {
			t.Error("Markdown() returned empty string")
		}
		if !strings.Contains(md, "|") {
			t.Error("Markdown() missing table pipes")
		}
		t.Logf("Markdown length: %d chars", len(md))
	})

	t.Run("Metadata", func(t *testing.T) {
		meta := r.Metadata()
		t.Logf("Metadata: Title=%q, Author=%q, Subject=%q",
			meta.Title, meta.Author, meta.Subject)
	})

	t.Run("Document", func(t *testing.T) {
		doc, err := r.Document()
		if err != nil {
			t.Errorf("Document() failed: %v", err)
		}
		if doc == nil {
			t.Fatal("Document() returned nil")
		}
		t.Logf("Document pages: %d", len(doc.Pages))
	})

	t.Run("Tables", func(t *testing.T) {
		tables := r.Tables()
		if len(tables) < 1 {
			t.Errorf("Tables() returned %d tables, want >= 1", len(tables))
		}
		for i, table := range tables {
			t.Logf("Table %d: %q, Headers: %v, Rows: %d",
				i, table.Name, table.Headers, len(table.Rows))
		}
	})

	t.Run("SheetAccess", func(t *testing.T) {
		sheet, err := r.Sheet(0)
		if err != nil {
			t.Fatalf("Sheet(0) failed: %v", err)
		}

		t.Logf("Sheet: %q, Rows: %d, Cols: %d",
			sheet.Name, sheet.RowCount(), sheet.ColCount())

		// Try to access some cells
		if sheet.RowCount() > 0 && sheet.ColCount() > 0 {
			cell := sheet.Cell(0, 0)
			if cell != nil {
				t.Logf("Cell A1: Type=%s, Value=%q", cell.Type.String(), cell.Value)
			}
		}
	})
}

// Test cell type handling
func TestCellTypeHandling(t *testing.T) {
	// Create XLSX with various cell types
	sheetContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData>
  <row r="1">
    <c r="A1" t="s"><v>0</v></c>
    <c r="B1"><v>42</v></c>
    <c r="C1" t="b"><v>1</v></c>
    <c r="D1" t="b"><v>0</v></c>
    <c r="E1" t="e"><v>#REF!</v></c>
    <c r="F1" t="str"><v>formula result</v></c>
  </row>
</sheetData>
</worksheet>`

	path := createTestXLSX(t, map[string]string{"Sheet1": sheetContent}, []string{"text"})
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	sheet, _ := r.Sheet(0)

	tests := []struct {
		ref      string
		wantType CellType
		wantVal  string
	}{
		{"A1", CellTypeString, "text"},
		{"B1", CellTypeNumber, "42"},
		{"C1", CellTypeBoolean, "TRUE"},
		{"D1", CellTypeBoolean, "FALSE"},
		{"E1", CellTypeError, "#REF!"},
		{"F1", CellTypeString, "formula result"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			cell := sheet.CellByRef(tt.ref)
			if cell == nil {
				t.Fatalf("Cell %s not found", tt.ref)
			}
			if cell.Type != tt.wantType {
				t.Errorf("Cell %s Type = %v, want %v", tt.ref, cell.Type, tt.wantType)
			}
			if cell.Value != tt.wantVal {
				t.Errorf("Cell %s Value = %q, want %q", tt.ref, cell.Value, tt.wantVal)
			}
		})
	}
}

// Test merged cells
func TestMergedCells(t *testing.T) {
	sheetContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData>
  <row r="1">
    <c r="A1" t="s"><v>0</v></c>
    <c r="B1"><v>1</v></c>
    <c r="C1"><v>2</v></c>
  </row>
  <row r="2">
    <c r="A2"><v>3</v></c>
    <c r="B2"><v>4</v></c>
    <c r="C2"><v>5</v></c>
  </row>
</sheetData>
<mergeCells count="1">
  <mergeCell ref="A1:B2"/>
</mergeCells>
</worksheet>`

	path := createTestXLSX(t, map[string]string{"Sheet1": sheetContent}, []string{"merged"})
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	sheet, _ := r.Sheet(0)

	// Check merged regions were parsed
	if len(sheet.MergedRegions) != 1 {
		t.Fatalf("MergedRegions = %d, want 1", len(sheet.MergedRegions))
	}

	mr := sheet.MergedRegions[0]
	if mr.StartCol != 0 || mr.StartRow != 0 || mr.EndCol != 1 || mr.EndRow != 1 {
		t.Errorf("MergedRegion = %+v, want A1:B2", mr)
	}

	// Check cell merge properties
	a1 := sheet.CellByRef("A1")
	if a1 == nil {
		t.Fatal("A1 not found")
	}
	if !a1.IsMerged || !a1.IsMergeRoot {
		t.Errorf("A1: IsMerged=%v, IsMergeRoot=%v, want both true", a1.IsMerged, a1.IsMergeRoot)
	}
	if a1.MergeRows != 2 || a1.MergeCols != 2 {
		t.Errorf("A1: MergeRows=%d, MergeCols=%d, want 2, 2", a1.MergeRows, a1.MergeCols)
	}

	// B1 should be merged but not root
	b1 := sheet.CellByRef("B1")
	if b1 == nil {
		t.Fatal("B1 not found")
	}
	if !b1.IsMerged || b1.IsMergeRoot {
		t.Errorf("B1: IsMerged=%v, IsMergeRoot=%v, want true, false", b1.IsMerged, b1.IsMergeRoot)
	}
}

// Test inline strings
func TestInlineStrings(t *testing.T) {
	sheetContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData>
  <row r="1">
    <c r="A1" t="inlineStr"><is><t>inline text</t></is></c>
  </row>
</sheetData>
</worksheet>`

	path := createTestXLSX(t, map[string]string{"Sheet1": sheetContent}, []string{})
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	sheet, _ := r.Sheet(0)
	cell := sheet.CellByRef("A1")
	if cell == nil {
		t.Fatal("A1 not found")
	}

	if cell.Type != CellTypeString {
		t.Errorf("Cell Type = %v, want CellTypeString", cell.Type)
	}
	if cell.Value != "inline text" {
		t.Errorf("Cell Value = %q, want 'inline text'", cell.Value)
	}
}

// Test formulas
func TestFormulas(t *testing.T) {
	sheetContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData>
  <row r="1">
    <c r="A1"><v>10</v></c>
    <c r="B1"><v>20</v></c>
    <c r="C1"><f>A1+B1</f><v>30</v></c>
  </row>
</sheetData>
</worksheet>`

	path := createTestXLSX(t, map[string]string{"Sheet1": sheetContent}, []string{})
	defer os.Remove(path)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	sheet, _ := r.Sheet(0)
	cell := sheet.CellByRef("C1")
	if cell == nil {
		t.Fatal("C1 not found")
	}

	if cell.Formula != "A1+B1" {
		t.Errorf("Cell Formula = %q, want 'A1+B1'", cell.Formula)
	}
	if cell.Value != "30" {
		t.Errorf("Cell Value = %q, want '30'", cell.Value)
	}
}

// Test sheet selection with options
func TestTextWithOptions_SheetSelection(t *testing.T) {
	// Create multi-sheet workbook
	sheet1 := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData>
  <row r="1"><c r="A1" t="s"><v>0</v></c></row>
</sheetData>
</worksheet>`

	sheet2 := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData>
  <row r="1"><c r="A1" t="s"><v>1</v></c></row>
</sheetData>
</worksheet>`

	// Create a custom multi-sheet xlsx
	tmpFile, err := os.CreateTemp("", "test-multisheet-*.xlsx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	f, _ := os.Create(tmpFile.Name())
	zw := zip.NewWriter(f)

	contentTypes := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>`
	writeZipFile(t, zw, "[Content_Types].xml", contentTypes)

	rels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`
	writeZipFile(t, zw, "_rels/.rels", rels)

	wbRels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/>
</Relationships>`
	writeZipFile(t, zw, "xl/_rels/workbook.xml.rels", wbRels)

	workbook := `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets>
  <sheet name="First" sheetId="1" r:id="rId2"/>
  <sheet name="Second" sheetId="2" r:id="rId3"/>
</sheets>
</workbook>`
	writeZipFile(t, zw, "xl/workbook.xml", workbook)

	sharedStrings := `<?xml version="1.0" encoding="UTF-8"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="2" uniqueCount="2">
  <si><t>FirstSheet</t></si>
  <si><t>SecondSheet</t></si>
</sst>`
	writeZipFile(t, zw, "xl/sharedStrings.xml", sharedStrings)
	writeZipFile(t, zw, "xl/worksheets/sheet1.xml", sheet1)
	writeZipFile(t, zw, "xl/worksheets/sheet2.xml", sheet2)

	zw.Close()
	f.Close()

	r, err := Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer r.Close()

	// Test selecting specific sheets
	text, err := r.TextWithOptions(ExtractOptions{Sheets: []int{0}})
	if err != nil {
		t.Fatalf("TextWithOptions() failed: %v", err)
	}

	if !strings.Contains(text, "FirstSheet") {
		t.Errorf("Text should contain FirstSheet, got: %s", text)
	}
	if strings.Contains(text, "SecondSheet") {
		t.Errorf("Text should NOT contain SecondSheet when only sheet 0 selected, got: %s", text)
	}
}

// Benchmark tests
func BenchmarkOpen(b *testing.B) {
	// Create a test file once
	sheetContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData>
  <row r="1">
    <c r="A1" t="s"><v>0</v></c>
    <c r="B1" t="s"><v>1</v></c>
  </row>
</sheetData>
</worksheet>`

	// Create temp file for benchmark
	tmpFile, _ := os.CreateTemp("", "bench-*.xlsx")
	tmpFile.Close()
	path := tmpFile.Name()
	defer os.Remove(path)

	// Create the xlsx content
	f, _ := os.Create(path)
	zw := zip.NewWriter(f)

	contentTypes := `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
<Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>`

	w, _ := zw.Create("[Content_Types].xml")
	w.Write([]byte(contentTypes))

	w, _ = zw.Create("_rels/.rels")
	w.Write([]byte(`<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`))

	w, _ = zw.Create("xl/_rels/workbook.xml.rels")
	w.Write([]byte(`<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`))

	w, _ = zw.Create("xl/workbook.xml")
	w.Write([]byte(`<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId2"/></sheets></workbook>`))

	w, _ = zw.Create("xl/sharedStrings.xml")
	w.Write([]byte(`<?xml version="1.0"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="2" uniqueCount="2"><si><t>A</t></si><si><t>B</t></si></sst>`))

	w, _ = zw.Create("xl/worksheets/sheet1.xml")
	w.Write([]byte(sheetContent))

	zw.Close()
	f.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := Open(path)
		if err != nil {
			b.Fatalf("Open failed: %v", err)
		}
		r.Close()
	}
}

func BenchmarkText(b *testing.B) {
	// Create a larger test file
	var rows bytes.Buffer
	rows.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<sheetData>`)

	for i := 1; i <= 100; i++ {
		rows.WriteString("<row r=\"")
		rows.WriteString(itoa(i))
		rows.WriteString("\">")
		for j := 0; j < 10; j++ {
			col := IndexToColumn(j)
			rows.WriteString("<c r=\"")
			rows.WriteString(col)
			rows.WriteString(itoa(i))
			rows.WriteString("\"><v>")
			rows.WriteString(itoa(i*10 + j))
			rows.WriteString("</v></c>")
		}
		rows.WriteString(`</row>`)
	}
	rows.WriteString(`</sheetData></worksheet>`)

	// Create temp file for benchmark
	tmpFile, _ := os.CreateTemp("", "bench-*.xlsx")
	tmpFile.Close()
	path := tmpFile.Name()
	defer os.Remove(path)

	f, _ := os.Create(path)
	zw := zip.NewWriter(f)

	w, _ := zw.Create("[Content_Types].xml")
	w.Write([]byte(`<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`))

	w, _ = zw.Create("_rels/.rels")
	w.Write([]byte(`<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`))

	w, _ = zw.Create("xl/_rels/workbook.xml.rels")
	w.Write([]byte(`<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`))

	w, _ = zw.Create("xl/workbook.xml")
	w.Write([]byte(`<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId2"/></sheets></workbook>`))

	w, _ = zw.Create("xl/worksheets/sheet1.xml")
	w.Write(rows.Bytes())

	zw.Close()
	f.Close()

	r, err := Open(path)
	if err != nil {
		b.Fatalf("Open failed: %v", err)
	}
	defer r.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := r.Text()
		if err != nil {
			b.Fatalf("Text failed: %v", err)
		}
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
