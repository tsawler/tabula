package core

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

// TestXRefEntry tests XRef entry creation
func TestXRefEntry(t *testing.T) {
	entry := &XRefEntry{
		Offset:     1234,
		Generation: 0,
		InUse:      true,
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
}

// TestXRefTable tests XRef table operations
func TestXRefTable(t *testing.T) {
	table := NewXRefTable()

	// Test Set and Get
	entry := &XRefEntry{
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
		wantOffset int64
		wantGen    int
		wantInUse  bool
		wantErr    bool
	}{
		{
			"in-use entry",
			"0000000017 00000 n ",
			17,
			0,
			true,
			false,
		},
		{
			"free entry",
			"0000000000 65535 f ",
			0,
			65535,
			false,
			false,
		},
		{
			"large offset",
			"0001234567 00003 n ",
			1234567,
			3,
			true,
			false,
		},
		{
			"with trailing newline",
			"0000000100 00000 n \n",
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
		{0, 0, 65535, false},  // Free entry
		{1, 17, 0, true},      // In-use entry
		{2, 81, 0, true},      // In-use entry
		{3, 0, 7, false},      // Free entry
		{4, 331, 0, true},     // In-use entry
		{5, 409, 0, true},     // In-use entry
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
		name      string
		input     string
		wantSize  int
		wantRoot  int
		wantPrev  int
		hasPrev   bool
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
