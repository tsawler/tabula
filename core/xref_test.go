package core

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// TestXRefEntryType tests XRefEntryType enum
func TestXRefEntryType(t *testing.T) {
	tests := []struct {
		entryType XRefEntryType
		wantStr   string
	}{
		{XRefEntryFree, "free"},
		{XRefEntryUncompressed, "uncompressed"},
		{XRefEntryCompressed, "compressed"},
		{XRefEntryType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.entryType.String(); got != tt.wantStr {
			t.Errorf("XRefEntryType(%d).String() = %q, want %q", tt.entryType, got, tt.wantStr)
		}
	}
}

// TestXRefEntry tests XRef entry creation
func TestXRefEntry(t *testing.T) {
	// Test uncompressed entry
	entry := &XRefEntry{
		Type:       XRefEntryUncompressed,
		Offset:     1234,
		Generation: 0,
		InUse:      true,
	}

	if entry.Type != XRefEntryUncompressed {
		t.Errorf("expected Type XRefEntryUncompressed, got %v", entry.Type)
	}
	if entry.Offset != 1234 {
		t.Errorf("expected offset 1234, got %d", entry.Offset)
	}
	if entry.Generation != 0 {
		t.Errorf("expected generation 0, got %d", entry.Generation)
	}
	if !entry.InUse {
		t.Error("expected InUse to be true")
	}

	// Test compressed entry (object in object stream)
	compressedEntry := &XRefEntry{
		Type:       XRefEntryCompressed,
		Offset:     10, // Object stream number
		Generation: 5,  // Index within object stream
		InUse:      true,
	}

	if compressedEntry.Type != XRefEntryCompressed {
		t.Errorf("expected Type XRefEntryCompressed, got %v", compressedEntry.Type)
	}
	if compressedEntry.Offset != 10 {
		t.Errorf("expected offset (objstm number) 10, got %d", compressedEntry.Offset)
	}
	if compressedEntry.Generation != 5 {
		t.Errorf("expected generation (objstm index) 5, got %d", compressedEntry.Generation)
	}

	// Test free entry
	freeEntry := &XRefEntry{
		Type:       XRefEntryFree,
		Offset:     0,
		Generation: 65535,
		InUse:      false,
	}

	if freeEntry.Type != XRefEntryFree {
		t.Errorf("expected Type XRefEntryFree, got %v", freeEntry.Type)
	}
	if freeEntry.InUse {
		t.Error("expected InUse to be false for free entry")
	}
}

// TestXRefTable tests XRef table operations
func TestXRefTable(t *testing.T) {
	table := NewXRefTable()

	// Test Set and Get
	entry := &XRefEntry{
		Type:       XRefEntryUncompressed,
		Offset:     1000,
		Generation: 0,
		InUse:      true,
	}
	table.Set(5, entry)

	retrieved, ok := table.Get(5)
	if !ok {
		t.Fatal("expected to retrieve entry")
	}
	if retrieved.Offset != 1000 {
		t.Errorf("expected offset 1000, got %d", retrieved.Offset)
	}

	// Test Size
	if table.Size() != 1 {
		t.Errorf("expected size 1, got %d", table.Size())
	}

	// Test Get non-existent
	_, ok = table.Get(999)
	if ok {
		t.Error("expected Get to return false for non-existent entry")
	}
}

// TestParseEntry tests parsing individual XRef entries
func TestParseEntry(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantType   XRefEntryType
		wantOffset int64
		wantGen    int
		wantInUse  bool
		wantErr    bool
	}{
		{
			"in-use entry",
			"0000000017 00000 n ",
			XRefEntryUncompressed,
			17,
			0,
			true,
			false,
		},
		{
			"free entry",
			"0000000000 65535 f ",
			XRefEntryFree,
			0,
			65535,
			false,
			false,
		},
		{
			"large offset",
			"0001234567 00003 n ",
			XRefEntryUncompressed,
			1234567,
			3,
			true,
			false,
		},
		{
			"with trailing newline",
			"0000000100 00000 n \n",
			XRefEntryUncompressed,
			100,
			0,
			true,
			false,
		},
		{
			"too short",
			"short",
			0,
			0,
			0,
			false,
			true,
		},
	}

	parser := &XRefParser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := parser.parseEntry(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got error: %v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}

			if entry.Type != tt.wantType {
				t.Errorf("expected Type %v, got %v", tt.wantType, entry.Type)
			}
			if entry.Offset != tt.wantOffset {
				t.Errorf("expected offset %d, got %d", tt.wantOffset, entry.Offset)
			}
			if entry.Generation != tt.wantGen {
				t.Errorf("expected generation %d, got %d", tt.wantGen, entry.Generation)
			}
			if entry.InUse != tt.wantInUse {
				t.Errorf("expected InUse=%v, got %v", tt.wantInUse, entry.InUse)
			}
		})
	}
}

// TestFindXRef tests finding the XRef offset from EOF
func TestFindXRef(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantOffset int64
		wantErr    bool
	}{
		{
			"simple case",
			"some pdf content\nstartxref\n1234\n%%EOF",
			1234,
			false,
		},
		{
			"with extra whitespace",
			"content\nstartxref\n  5678  \n%%EOF\n",
			5678,
			false,
		},
		{
			"CR-only line endings",
			"some pdf content\rstartxref\r1234\r%%EOF",
			1234,
			false,
		},
		{
			"CRLF line endings",
			"some pdf content\r\nstartxref\r\n1234\r\n%%EOF",
			1234,
			false,
		},
		{
			"no startxref",
			"content without startxref\n%%EOF",
			0,
			true,
		},
		{
			"invalid offset",
			"content\nstartxref\nabc\n%%EOF",
			0,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			parser := NewXRefParser(reader)

			offset, err := parser.FindXRef()
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got error: %v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}

			if offset != tt.wantOffset {
				t.Errorf("expected offset %d, got %d", tt.wantOffset, offset)
			}
		})
	}
}

// TestParseXRef tests parsing a complete XRef table
func TestParseXRef(t *testing.T) {
	input := `xref
0 6
0000000000 65535 f
0000000017 00000 n
0000000081 00000 n
0000000000 00007 f
0000000331 00000 n
0000000409 00000 n
trailer
<< /Size 6 /Root 1 0 R >>
startxref
0
%%EOF`

	reader := strings.NewReader(input)
	parser := NewXRefParser(reader)

	table, err := parser.ParseXRef(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check table size
	if table.Size() != 6 {
		t.Errorf("expected 6 entries, got %d", table.Size())
	}

	// Check specific entries
	tests := []struct {
		objNum     int
		wantOffset int64
		wantGen    int
		wantInUse  bool
	}{
		{0, 0, 65535, false}, // Free entry
		{1, 17, 0, true},     // In-use entry
		{2, 81, 0, true},     // In-use entry
		{3, 0, 7, false},     // Free entry
		{4, 331, 0, true},    // In-use entry
		{5, 409, 0, true},    // In-use entry
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.objNum)), func(t *testing.T) {
			entry, ok := table.Get(tt.objNum)
			if !ok {
				t.Fatalf("expected entry %d to exist", tt.objNum)
			}
			if entry.Offset != tt.wantOffset {
				t.Errorf("entry %d: expected offset %d, got %d", tt.objNum, tt.wantOffset, entry.Offset)
			}
			if entry.Generation != tt.wantGen {
				t.Errorf("entry %d: expected generation %d, got %d", tt.objNum, tt.wantGen, entry.Generation)
			}
			if entry.InUse != tt.wantInUse {
				t.Errorf("entry %d: expected InUse=%v, got %v", tt.objNum, tt.wantInUse, entry.InUse)
			}
		})
	}

	// Check trailer
	sizeObj := table.Trailer.Get("Size")
	if sizeObj == nil {
		t.Fatal("expected Size in trailer")
	}
	if size, ok := sizeObj.(Int); !ok || int(size) != 6 {
		t.Errorf("expected Size=6, got %v", sizeObj)
	}

	rootObj := table.Trailer.Get("Root")
	if rootObj == nil {
		t.Fatal("expected Root in trailer")
	}
	if root, ok := rootObj.(IndirectRef); !ok || root.Number != 1 {
		t.Errorf("expected Root=1 0 R, got %v", rootObj)
	}
}

// TestParseXRefMultipleSubsections tests parsing XRef with multiple subsections
func TestParseXRefMultipleSubsections(t *testing.T) {
	input := `xref
0 1
0000000000 65535 f
3 2
0000000331 00000 n
0000000409 00000 n
trailer
<< /Size 5 >>
startxref
0
%%EOF`

	reader := strings.NewReader(input)
	parser := NewXRefParser(reader)

	table, err := parser.ParseXRef(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have entries for objects 0, 3, 4
	if table.Size() != 3 {
		t.Errorf("expected 3 entries, got %d", table.Size())
	}

	// Check object 0
	entry0, ok := table.Get(0)
	if !ok {
		t.Error("expected entry 0 to exist")
	} else if entry0.InUse {
		t.Error("expected entry 0 to be free")
	}

	// Check object 3
	entry3, ok := table.Get(3)
	if !ok {
		t.Error("expected entry 3 to exist")
	} else if entry3.Offset != 331 {
		t.Errorf("expected entry 3 offset 331, got %d", entry3.Offset)
	}

	// Check object 4
	entry4, ok := table.Get(4)
	if !ok {
		t.Error("expected entry 4 to exist")
	} else if entry4.Offset != 409 {
		t.Errorf("expected entry 4 offset 409, got %d", entry4.Offset)
	}

	// Check objects 1, 2 don't exist
	if _, ok := table.Get(1); ok {
		t.Error("did not expect entry 1 to exist")
	}
	if _, ok := table.Get(2); ok {
		t.Error("did not expect entry 2 to exist")
	}
}

// TestParseXRefFromEOF tests finding and parsing XRef from EOF
func TestParseXRefFromEOF(t *testing.T) {
	input := `%PDF-1.4
some content
xref
0 2
0000000000 65535 f
0000000017 00000 n
trailer
<< /Size 2 /Root 1 0 R >>
startxref
22
%%EOF`

	reader := strings.NewReader(input)
	parser := NewXRefParser(reader)

	table, err := parser.ParseXRefFromEOF()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if table.Size() != 2 {
		t.Errorf("expected 2 entries, got %d", table.Size())
	}

	// Check entry 1
	entry, ok := table.Get(1)
	if !ok {
		t.Fatal("expected entry 1 to exist")
	}
	if entry.Offset != 17 {
		t.Errorf("expected offset 17, got %d", entry.Offset)
	}
}

// TestMergeXRefTables tests merging multiple XRef tables
func TestMergeXRefTables(t *testing.T) {
	// Create first table
	table1 := NewXRefTable()
	table1.Set(1, &XRefEntry{Offset: 100, Generation: 0, InUse: true})
	table1.Set(2, &XRefEntry{Offset: 200, Generation: 0, InUse: true})
	table1.Trailer = Dict{"Size": Int(3)}

	// Create second table (updates object 1, adds object 3)
	table2 := NewXRefTable()
	table2.Set(1, &XRefEntry{Offset: 150, Generation: 1, InUse: true}) // Updated
	table2.Set(3, &XRefEntry{Offset: 300, Generation: 0, InUse: true}) // New
	table2.Trailer = Dict{"Size": Int(4)}

	// Merge
	merged := MergeXRefTables(table1, table2)

	// Check merged size
	if merged.Size() != 3 {
		t.Errorf("expected 3 entries, got %d", merged.Size())
	}

	// Check object 1 was updated
	entry1, ok := merged.Get(1)
	if !ok {
		t.Fatal("expected entry 1")
	}
	if entry1.Offset != 150 {
		t.Errorf("expected updated offset 150, got %d", entry1.Offset)
	}
	if entry1.Generation != 1 {
		t.Errorf("expected updated generation 1, got %d", entry1.Generation)
	}

	// Check object 2 still exists
	entry2, ok := merged.Get(2)
	if !ok {
		t.Fatal("expected entry 2")
	}
	if entry2.Offset != 200 {
		t.Errorf("expected offset 200, got %d", entry2.Offset)
	}

	// Check object 3 was added
	entry3, ok := merged.Get(3)
	if !ok {
		t.Fatal("expected entry 3")
	}
	if entry3.Offset != 300 {
		t.Errorf("expected offset 300, got %d", entry3.Offset)
	}

	// Check trailer is from last table
	sizeObj := merged.Trailer.Get("Size")
	if size, ok := sizeObj.(Int); !ok || int(size) != 4 {
		t.Errorf("expected Size=4 from last trailer, got %v", sizeObj)
	}
}

// TestParseTrailer tests parsing trailer dictionaries
func TestParseTrailer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSize int
		wantRoot int
		wantPrev int
		hasPrev  bool
	}{
		{
			"basic trailer",
			"<< /Size 5 /Root 1 0 R >>",
			5,
			1,
			0,
			false,
		},
		{
			"trailer with Prev",
			"<< /Size 10 /Root 2 0 R /Prev 1234 >>",
			10,
			2,
			1234,
			true,
		},
		{
			"multiline trailer",
			`<<
/Size 3
/Root 1 0 R
/Info 2 0 R
>>`,
			3,
			1,
			0,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a scanner with the input
			reader := strings.NewReader(tt.input)
			scanner := bufio.NewScanner(reader)

			parser := &XRefParser{}
			dict, err := parser.parseTrailer(scanner)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check Size
			sizeObj := dict.Get("Size")
			if sizeObj == nil {
				t.Fatal("expected Size in trailer")
			}
			if size, ok := sizeObj.(Int); !ok || int(size) != tt.wantSize {
				t.Errorf("expected Size=%d, got %v", tt.wantSize, sizeObj)
			}

			// Check Root
			rootObj := dict.Get("Root")
			if rootObj == nil {
				t.Fatal("expected Root in trailer")
			}
			if root, ok := rootObj.(IndirectRef); !ok || root.Number != tt.wantRoot {
				t.Errorf("expected Root=%d 0 R, got %v", tt.wantRoot, rootObj)
			}

			// Check Prev if expected
			prevObj := dict.Get("Prev")
			if tt.hasPrev {
				if prevObj == nil {
					t.Fatal("expected Prev in trailer")
				}
				if prev, ok := prevObj.(Int); !ok || int(prev) != tt.wantPrev {
					t.Errorf("expected Prev=%d, got %v", tt.wantPrev, prevObj)
				}
			} else {
				if prevObj != nil {
					t.Errorf("did not expect Prev in trailer, got %v", prevObj)
				}
			}
		})
	}
}

// TestXRefErrors tests error handling in XRef parsing
func TestXRefErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing xref keyword", "0 2\n0000000000 65535 f\n"},
		{"invalid subsection header", "xref\nabc def\n"},
		{"truncated entries", "xref\n0 2\n0000000000 65535 f\n"},
		{"missing trailer", "xref\n0 1\n0000000000 65535 f\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			parser := NewXRefParser(reader)

			_, err := parser.ParseXRef(0)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// BenchmarkParseXRef benchmarks XRef table parsing
func BenchmarkParseXRef(b *testing.B) {
	input := `xref
0 100
0000000000 65535 f
`
	// Add 99 more entries
	for i := 1; i < 100; i++ {
		input += "0000001234 00000 n \n"
	}
	input += `trailer
<< /Size 100 /Root 1 0 R >>
startxref
0
%%EOF`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(input)
		parser := NewXRefParser(reader)
		parser.ParseXRef(0)
	}
}

// BenchmarkFindXRef benchmarks finding XRef from EOF
func BenchmarkFindXRef(b *testing.B) {
	// Create a buffer with content before startxref
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	for i := 0; i < 1000; i++ {
		buf.WriteString("some pdf content line\n")
	}
	buf.WriteString("startxref\n")
	buf.WriteString("12345\n")
	buf.WriteString("%%EOF\n")

	input := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(input)
		parser := NewXRefParser(reader)
		parser.FindXRef()
	}
}

// ============================================================================
// ParsePrevXRef tests
// ============================================================================

// TestParsePrevXRef_NoPrev tests parsing XRef when there's no Prev entry
func TestParsePrevXRef_NoPrev(t *testing.T) {
	table := NewXRefTable()
	table.Trailer = Dict{"Size": Int(3)}

	parser := NewXRefParser(strings.NewReader(""))
	prevTable, err := parser.ParsePrevXRef(table)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prevTable != nil {
		t.Error("expected nil for table without /Prev")
	}
}

// TestParsePrevXRef_InvalidPrevType tests error handling for invalid /Prev type
func TestParsePrevXRef_InvalidPrevType(t *testing.T) {
	table := NewXRefTable()
	table.Trailer = Dict{
		"Size": Int(3),
		"Prev": String("invalid"), // Should be Int, not String
	}

	parser := NewXRefParser(strings.NewReader(""))
	_, err := parser.ParsePrevXRef(table)
	if err == nil {
		t.Error("expected error for invalid /Prev type")
	}
}

// TestParsePrevXRef_WithPrev tests parsing XRef with a /Prev entry
func TestParsePrevXRef_WithPrev(t *testing.T) {
	// Create a PDF with two XRef tables (incremental update)
	// Previous XRef at offset 0
	prevXRef := `xref
0 2
0000000000 65535 f
0000000017 00000 n
trailer
<< /Size 2 /Root 1 0 R >>
`
	// Padding to position main xref at known offset
	mainOffset := len(prevXRef)

	// Main XRef at known offset with /Prev pointing to 0
	mainXRef := `xref
0 3
0000000000 65535 f
0000000017 00000 n
0000000100 00000 n
trailer
<< /Size 3 /Root 1 0 R /Prev 0 >>
startxref
` + string(rune(mainOffset+'0')) + `
%%EOF`

	fullPDF := prevXRef + mainXRef

	reader := strings.NewReader(fullPDF)
	parser := NewXRefParser(reader)

	// Parse main XRef first
	mainTable, err := parser.ParseXRef(int64(mainOffset))
	if err != nil {
		t.Fatalf("failed to parse main xref: %v", err)
	}

	// Verify main table has /Prev
	prevObj := mainTable.Trailer.Get("Prev")
	if prevObj == nil {
		t.Fatal("main table should have /Prev")
	}

	// Now parse previous XRef
	prevTable, err := parser.ParsePrevXRef(mainTable)
	if err != nil {
		t.Fatalf("ParsePrevXRef() error = %v", err)
	}

	if prevTable == nil {
		t.Fatal("expected non-nil prev table")
	}

	// Previous table should have 2 entries
	if prevTable.Size() != 2 {
		t.Errorf("prev table size = %d, want 2", prevTable.Size())
	}
}

// ============================================================================
// ParseAllXRefs tests
// ============================================================================

// TestParseAllXRefs_Single tests parsing a PDF with single XRef
func TestParseAllXRefs_Single(t *testing.T) {
	input := `%PDF-1.4
some content
xref
0 2
0000000000 65535 f
0000000017 00000 n
trailer
<< /Size 2 /Root 1 0 R >>
startxref
22
%%EOF`

	reader := strings.NewReader(input)
	parser := NewXRefParser(reader)

	tables, err := parser.ParseAllXRefs()
	if err != nil {
		t.Fatalf("ParseAllXRefs() error = %v", err)
	}

	if len(tables) != 1 {
		t.Errorf("expected 1 table, got %d", len(tables))
	}

	if tables[0].Size() != 2 {
		t.Errorf("table size = %d, want 2", tables[0].Size())
	}
}

// TestParseAllXRefs_Error tests error handling
func TestParseAllXRefs_Error(t *testing.T) {
	// Invalid PDF without startxref
	input := `%PDF-1.4
some content without xref
%%EOF`

	reader := strings.NewReader(input)
	parser := NewXRefParser(reader)

	_, err := parser.ParseAllXRefs()
	if err == nil {
		t.Error("expected error for PDF without startxref")
	}
}

// ============================================================================
// MergeXRefTables edge cases
// ============================================================================

// TestMergeXRefTables_Empty tests merging zero tables
func TestMergeXRefTables_Empty(t *testing.T) {
	merged := MergeXRefTables()
	if merged == nil {
		t.Fatal("expected non-nil result")
	}
	if merged.Size() != 0 {
		t.Errorf("expected empty table, got size %d", merged.Size())
	}
}

// TestMergeXRefTables_Single tests merging single table
func TestMergeXRefTables_Single(t *testing.T) {
	table := NewXRefTable()
	table.Set(1, &XRefEntry{Offset: 100, InUse: true})
	table.Trailer = Dict{"Size": Int(2)}

	merged := MergeXRefTables(table)
	if merged.Size() != 1 {
		t.Errorf("expected size 1, got %d", merged.Size())
	}

	entry, ok := merged.Get(1)
	if !ok || entry.Offset != 100 {
		t.Error("expected entry 1 with offset 100")
	}
}

// ============================================================================
// parseEntry edge cases
// ============================================================================

// TestParseEntry_InvalidFlag tests error handling for invalid in-use flag
func TestParseEntry_InvalidFlag(t *testing.T) {
	parser := &XRefParser{}
	_, err := parser.parseEntry("0000000017 00000 x ")
	if err == nil {
		t.Error("expected error for invalid flag 'x'")
	}
}

// TestParseEntry_InvalidGeneration tests error for non-numeric generation
func TestParseEntry_InvalidGeneration(t *testing.T) {
	parser := &XRefParser{}
	_, err := parser.parseEntry("0000000017 abcde n ")
	if err == nil {
		t.Error("expected error for invalid generation")
	}
}

// TestParseEntry_InvalidOffset tests error for non-numeric offset
func TestParseEntry_InvalidOffset(t *testing.T) {
	parser := &XRefParser{}
	_, err := parser.parseEntry("abcdefghij 00000 n ")
	if err == nil {
		t.Error("expected error for invalid offset")
	}
}

// crOnlyPDF builds a minimal valid PDF using \r (CR-only) line endings,
// matching files produced by some generators (e.g. Illustrator, InDesign).
func crOnlyPDF(t *testing.T, text string) []byte {
	t.Helper()

	var buf bytes.Buffer
	offsets := make([]int, 6)

	buf.WriteString("%PDF-1.4\r")

	offsets[1] = buf.Len()
	buf.WriteString("1 0 obj\r<</Type /Catalog /Pages 2 0 R>>\rendobj\r")

	offsets[2] = buf.Len()
	buf.WriteString("2 0 obj\r<</Type /Pages /Kids [3 0 R] /Count 1>>\rendobj\r")

	offsets[3] = buf.Len()
	buf.WriteString("3 0 obj\r<</Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources <</Font <</F1 5 0 R>>>>>>\rendobj\r")

	stream := fmt.Sprintf("BT /F1 12 Tf 100 700 Td (%s) Tj ET", text)
	offsets[4] = buf.Len()
	fmt.Fprintf(&buf, "4 0 obj\r<</Length %d>>\rstream\r%s\rendstream\rendobj\r", len(stream), stream)

	offsets[5] = buf.Len()
	buf.WriteString("5 0 obj\r<</Type /Font /Subtype /Type1 /BaseFont /Helvetica>>\rendobj\r")

	xrefOffset := buf.Len()
	buf.WriteString("xref\r0 6\r")
	fmt.Fprintf(&buf, "%010d 65535 f \r", 0)
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \r", offsets[i])
	}
	buf.WriteString("trailer\r<</Size 6 /Root 1 0 R>>\rstartxref\r")
	fmt.Fprintf(&buf, "%d\r", xrefOffset)
	buf.WriteString("%%EOF\r")

	return buf.Bytes()
}

// TestParseXRef_CROnlyLineEndings tests parsing a complete XRef table
// from a PDF that uses bare CR (\r) line endings.
func TestParseXRef_CROnlyLineEndings(t *testing.T) {
	pdf := crOnlyPDF(t, "hello")
	reader := bytes.NewReader(pdf)
	parser := NewXRefParser(reader)

	table, err := parser.ParseXRefFromEOF()
	if err != nil {
		t.Fatalf("ParseXRefFromEOF() error: %v", err)
	}

	if table.Size() != 6 {
		t.Errorf("expected 6 entries, got %d", table.Size())
	}

	// Verify free entry
	entry0, ok := table.Get(0)
	if !ok {
		t.Fatal("expected entry 0 to exist")
	}
	if entry0.InUse {
		t.Error("expected entry 0 to be free")
	}

	// Verify in-use entries exist
	for i := 1; i <= 5; i++ {
		entry, ok := table.Get(i)
		if !ok {
			t.Errorf("expected entry %d to exist", i)
			continue
		}
		if !entry.InUse {
			t.Errorf("expected entry %d to be in-use", i)
		}
	}

	// Verify trailer
	rootObj := table.Trailer.Get("Root")
	if rootObj == nil {
		t.Fatal("expected Root in trailer")
	}
}

// TestScanLines tests the custom line scanner
func TestScanLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"LF only", "a\nb\nc", []string{"a", "b", "c"}},
		{"CR only", "a\rb\rc", []string{"a", "b", "c"}},
		{"CRLF only", "a\r\nb\r\nc", []string{"a", "b", "c"}},
		{"mixed endings", "a\nb\rc\r\nd", []string{"a", "b", "c", "d"}},
		{"empty lines LF", "a\n\nb", []string{"a", "", "b"}},
		{"empty lines CR", "a\r\rb", []string{"a", "", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader(tt.input))
			scanner.Split(scanLines)

			var got []string
			for scanner.Scan() {
				got = append(got, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				t.Fatalf("scanner error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %d lines %v, want %d lines %v", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestIsXRefStream_CROnlyTraditional tests detection of traditional xref
// with CR-only line endings.
func TestIsXRefStream_CROnlyTraditional(t *testing.T) {
	reader := strings.NewReader("xref\r0 6\r")
	parser := NewXRefParser(reader)

	isStream, err := parser.isXRefStream()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isStream {
		t.Error("expected traditional xref, got stream")
	}
}
