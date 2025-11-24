package reader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tsawler/tabula/core"
)

// minimalPDF is a minimal valid PDF for testing
const minimalPDF = `%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [] /Count 0 >>
endobj
xref
0 3
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
trailer
<< /Size 3 /Root 1 0 R >>
startxref
110
%%EOF`

// pdfWithInfo is a PDF with an Info dictionary
const pdfWithInfo = `%PDF-1.7
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [] /Count 0 >>
endobj
3 0 obj
<< /Title (Test Document) /Author (Test Author) >>
endobj
xref
0 4
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000110 00000 n
trailer
<< /Size 4 /Root 1 0 R /Info 3 0 R >>
startxref
176
%%EOF`

// createTempPDF creates a temporary PDF file with the given content
func createTempPDF(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.pdf")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create temp PDF: %v", err)
	}

	return tmpFile
}

// TestOpen tests opening a PDF file
func TestOpen(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	if reader.file == nil {
		t.Error("expected file to be set")
	}
	if reader.xrefTable == nil {
		t.Error("expected xrefTable to be set")
	}
	if reader.trailer == nil {
		t.Error("expected trailer to be set")
	}
}

// TestOpenNonExistent tests opening a non-existent file
func TestOpenNonExistent(t *testing.T) {
	_, err := Open("/nonexistent/file.pdf")
	if err == nil {
		t.Error("expected error when opening non-existent file")
	}
}

// TestParseHeader tests PDF header parsing
func TestParseHeader(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantMajor int
		wantMinor int
		wantErr   bool
	}{
		{
			"PDF 1.4",
			"%PDF-1.4\n" + minimalPDF[9:],
			1, 4, false,
		},
		{
			"PDF 1.7",
			"%PDF-1.7\n" + minimalPDF[9:],
			1, 7, false,
		},
		{
			"PDF 2.0",
			"%PDF-2.0\n" + minimalPDF[9:],
			2, 0, false,
		},
		{
			"invalid header",
			"NOT-PDF-1.4\n" + minimalPDF[9:],
			0, 0, true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempPDF(t, tt.content)

			reader, err := Open(tmpFile)
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got error: %v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}
			defer reader.Close()

			version := reader.Version()
			if version.Major != tt.wantMajor {
				t.Errorf("expected major version %d, got %d", tt.wantMajor, version.Major)
			}
			if version.Minor != tt.wantMinor {
				t.Errorf("expected minor version %d, got %d", tt.wantMinor, version.Minor)
			}
		})
	}
}

// TestVersion tests version retrieval
func TestVersion(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	version := reader.Version()
	if version.Major != 1 {
		t.Errorf("expected major version 1, got %d", version.Major)
	}
	if version.Minor != 4 {
		t.Errorf("expected minor version 4, got %d", version.Minor)
	}

	versionStr := version.String()
	if versionStr != "1.4" {
		t.Errorf("expected version string '1.4', got '%s'", versionStr)
	}
}

// TestTrailer tests trailer retrieval
func TestTrailer(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	trailer := reader.Trailer()
	if trailer == nil {
		t.Fatal("expected trailer to be set")
	}

	// Check Size
	sizeObj := trailer.Get("Size")
	if sizeObj == nil {
		t.Fatal("expected Size in trailer")
	}
	if size, ok := sizeObj.(core.Int); !ok || int(size) != 3 {
		t.Errorf("expected Size=3, got %v", sizeObj)
	}

	// Check Root
	rootObj := trailer.Get("Root")
	if rootObj == nil {
		t.Fatal("expected Root in trailer")
	}
	if root, ok := rootObj.(core.IndirectRef); !ok || root.Number != 1 {
		t.Errorf("expected Root=1 0 R, got %v", rootObj)
	}
}

// TestGetObject tests object retrieval
func TestGetObject(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	// Get object 1 (Catalog)
	obj1, err := reader.GetObject(1)
	if err != nil {
		t.Fatalf("failed to get object 1: %v", err)
	}

	dict, ok := obj1.(core.Dict)
	if !ok {
		t.Fatalf("expected Dict, got %T", obj1)
	}

	// Check Type
	typeObj := dict.Get("Type")
	if typeObj == nil {
		t.Fatal("expected Type in catalog")
	}
	if typeName, ok := typeObj.(core.Name); !ok || string(typeName) != "Catalog" {
		t.Errorf("expected Type=/Catalog, got %v", typeObj)
	}
}

// TestGetObjectCaching tests that objects are cached
func TestGetObjectCaching(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	// Get object twice
	obj1a, err := reader.GetObject(1)
	if err != nil {
		t.Fatalf("failed to get object 1: %v", err)
	}

	if reader.CacheSize() != 1 {
		t.Errorf("expected cache size 1, got %d", reader.CacheSize())
	}

	obj1b, err := reader.GetObject(1)
	if err != nil {
		t.Fatalf("failed to get object 1 second time: %v", err)
	}

	// Should be the same object (from cache)
	if reader.CacheSize() != 1 {
		t.Errorf("expected cache size still 1, got %d", reader.CacheSize())
	}

	// Objects should be equal (same content)
	dict1a := obj1a.(core.Dict)
	dict1b := obj1b.(core.Dict)
	if len(dict1a) != len(dict1b) {
		t.Error("cached object differs from original")
	}
}

// TestGetObjectNotFound tests error when object not found
func TestGetObjectNotFound(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	// Try to get non-existent object
	_, err = reader.GetObject(999)
	if err == nil {
		t.Error("expected error when getting non-existent object")
	}
}

// TestResolveReference tests resolving indirect references
func TestResolveReference(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	// Get the catalog through its reference
	ref := core.IndirectRef{Number: 1, Generation: 0}
	obj, err := reader.ResolveReference(ref)
	if err != nil {
		t.Fatalf("failed to resolve reference: %v", err)
	}

	dict, ok := obj.(core.Dict)
	if !ok {
		t.Fatalf("expected Dict, got %T", obj)
	}

	typeObj := dict.Get("Type")
	if typeName, ok := typeObj.(core.Name); !ok || string(typeName) != "Catalog" {
		t.Errorf("expected Type=/Catalog, got %v", typeObj)
	}
}

// TestGetCatalog tests getting the document catalog
func TestGetCatalog(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	catalog, err := reader.GetCatalog()
	if err != nil {
		t.Fatalf("failed to get catalog: %v", err)
	}

	// Check Type
	typeObj := catalog.Get("Type")
	if typeObj == nil {
		t.Fatal("expected Type in catalog")
	}
	if typeName, ok := typeObj.(core.Name); !ok || string(typeName) != "Catalog" {
		t.Errorf("expected Type=/Catalog, got %v", typeObj)
	}

	// Check Pages
	pagesObj := catalog.Get("Pages")
	if pagesObj == nil {
		t.Fatal("expected Pages in catalog")
	}
	if pagesRef, ok := pagesObj.(core.IndirectRef); !ok || pagesRef.Number != 2 {
		t.Errorf("expected Pages=2 0 R, got %v", pagesObj)
	}
}

// TestGetInfo tests getting the document info dictionary
func TestGetInfo(t *testing.T) {
	tmpFile := createTempPDF(t, pdfWithInfo)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	info, err := reader.GetInfo()
	if err != nil {
		t.Fatalf("failed to get info: %v", err)
	}

	if info == nil {
		t.Fatal("expected info dictionary")
	}

	// Check Title
	titleObj := info.Get("Title")
	if titleObj == nil {
		t.Fatal("expected Title in info")
	}
	if title, ok := titleObj.(core.String); !ok || string(title) != "Test Document" {
		t.Errorf("expected Title='Test Document', got %v", titleObj)
	}

	// Check Author
	authorObj := info.Get("Author")
	if authorObj == nil {
		t.Fatal("expected Author in info")
	}
	if author, ok := authorObj.(core.String); !ok || string(author) != "Test Author" {
		t.Errorf("expected Author='Test Author', got %v", authorObj)
	}
}

// TestGetInfoMissing tests when Info dictionary is missing
func TestGetInfoMissing(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	info, err := reader.GetInfo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info != nil {
		t.Error("expected info to be nil when not present")
	}
}

// TestNumObjects tests getting the number of objects
func TestNumObjects(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	numObjects := reader.NumObjects()
	if numObjects != 3 {
		t.Errorf("expected 3 objects, got %d", numObjects)
	}
}

// TestFileSize tests getting the file size
func TestFileSize(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	fileSize := reader.FileSize()
	if fileSize <= 0 {
		t.Errorf("expected positive file size, got %d", fileSize)
	}

	// Should match the length of our test PDF
	expectedSize := int64(len(minimalPDF))
	if fileSize != expectedSize {
		t.Errorf("expected file size %d, got %d", expectedSize, fileSize)
	}
}

// TestXRefTable tests accessing the XRef table
func TestXRefTable(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	xrefTable := reader.XRefTable()
	if xrefTable == nil {
		t.Fatal("expected xref table to be set")
	}

	// Check it has entries
	if xrefTable.Size() != 3 {
		t.Errorf("expected 3 xref entries, got %d", xrefTable.Size())
	}

	// Check entry 1 exists
	entry, ok := xrefTable.Get(1)
	if !ok {
		t.Error("expected entry 1 to exist")
	} else if !entry.InUse {
		t.Error("expected entry 1 to be in use")
	}
}

// TestClearCache tests clearing the object cache
func TestClearCache(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	// Load some objects
	reader.GetObject(1)
	reader.GetObject(2)

	if reader.CacheSize() != 2 {
		t.Errorf("expected cache size 2, got %d", reader.CacheSize())
	}

	// Clear cache
	reader.ClearCache()

	if reader.CacheSize() != 0 {
		t.Errorf("expected cache size 0 after clear, got %d", reader.CacheSize())
	}

	// Should still be able to load objects after clearing cache
	_, err = reader.GetObject(1)
	if err != nil {
		t.Errorf("failed to get object after cache clear: %v", err)
	}
}

// TestClose tests closing the reader
func TestClose(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}

	err = reader.Close()
	if err != nil {
		t.Errorf("failed to close reader: %v", err)
	}

	// Should not be able to read after closing
	_, err = reader.GetObject(1)
	if err == nil {
		t.Error("expected error when getting object after close")
	}
}

// TestMultipleObjects tests loading multiple objects
func TestMultipleObjects(t *testing.T) {
	tmpFile := createTempPDF(t, minimalPDF)

	reader, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer reader.Close()

	// Load object 1 (Catalog)
	obj1, err := reader.GetObject(1)
	if err != nil {
		t.Fatalf("failed to get object 1: %v", err)
	}
	if _, ok := obj1.(core.Dict); !ok {
		t.Error("object 1 should be a Dict")
	}

	// Load object 2 (Pages)
	obj2, err := reader.GetObject(2)
	if err != nil {
		t.Fatalf("failed to get object 2: %v", err)
	}
	if _, ok := obj2.(core.Dict); !ok {
		t.Error("object 2 should be a Dict")
	}

	// Both should be cached
	if reader.CacheSize() != 2 {
		t.Errorf("expected cache size 2, got %d", reader.CacheSize())
	}
}
