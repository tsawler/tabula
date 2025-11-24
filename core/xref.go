package core

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// XRefEntry represents a single cross-reference table entry
type XRefEntry struct {
	Offset     int64 // Byte offset in file (for in-use objects) or next free object number (for free objects)
	Generation int   // Generation number
	InUse      bool  // true if object is in use, false if free
}

// XRefTable represents a PDF cross-reference table
type XRefTable struct {
	Entries map[int]*XRefEntry // Map from object number to XRef entry
	Trailer Dict               // Trailer dictionary
}

// NewXRefTable creates a new empty XRef table
func NewXRefTable() *XRefTable {
	return &XRefTable{
		Entries: make(map[int]*XRefEntry),
		Trailer: make(Dict),
	}
}

// Get retrieves an XRef entry by object number
func (x *XRefTable) Get(objNum int) (*XRefEntry, bool) {
	entry, ok := x.Entries[objNum]
	return entry, ok
}

// Set adds or updates an XRef entry
func (x *XRefTable) Set(objNum int, entry *XRefEntry) {
	x.Entries[objNum] = entry
}

// Size returns the number of entries in the table
func (x *XRefTable) Size() int {
	return len(x.Entries)
}

// XRefParser parses PDF cross-reference tables
type XRefParser struct {
	reader   io.ReadSeeker
	startPos int64 // Starting position for current parse
}

// NewXRefParser creates a new XRef parser
func NewXRefParser(r io.ReadSeeker) *XRefParser {
	return &XRefParser{
		reader: r,
	}
}

// FindXRef finds the byte offset of the XRef table by scanning from EOF
// PDFs end with "startxref\n<offset>\n%%EOF"
func (x *XRefParser) FindXRef() (int64, error) {
	// Seek to end to get file size
	fileSize, err := x.reader.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("failed to seek to end: %w", err)
	}

	// Read last 1024 bytes (should be enough for startxref section)
	readSize := int64(1024)
	if fileSize < readSize {
		readSize = fileSize
	}

	_, err = x.reader.Seek(fileSize-readSize, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("failed to seek to startxref area: %w", err)
	}

	buf := make([]byte, readSize)
	n, err := x.reader.Read(buf)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("failed to read startxref area: %w", err)
	}
	buf = buf[:n]

	// Find "startxref" keyword
	content := string(buf)
	idx := strings.LastIndex(content, "startxref")
	if idx == -1 {
		return 0, fmt.Errorf("startxref not found in PDF")
	}

	// Parse the offset after startxref
	afterStartXRef := content[idx+len("startxref"):]
	lines := strings.Split(afterStartXRef, "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("invalid startxref format")
	}

	// The offset should be on the next line
	offsetStr := strings.TrimSpace(lines[1])
	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid xref offset: %w", err)
	}

	return offset, nil
}

// ParseXRef parses the XRef table at the given byte offset
func (x *XRefParser) ParseXRef(offset int64) (*XRefTable, error) {
	// Seek to the XRef table
	_, err := x.reader.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to xref: %w", err)
	}

	x.startPos = offset

	scanner := bufio.NewScanner(x.reader)

	// Read "xref" keyword
	if !scanner.Scan() {
		return nil, fmt.Errorf("failed to read xref keyword")
	}
	line := strings.TrimSpace(scanner.Text())
	if line != "xref" {
		return nil, fmt.Errorf("expected 'xref' keyword, got '%s'", line)
	}

	table := NewXRefTable()
	foundTrailer := false

	// Parse subsections
	for scanner.Scan() {
		line = strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Check if we've reached the trailer
		if line == "trailer" {
			// Parse trailer dictionary
			trailer, err := x.parseTrailer(scanner)
			if err != nil {
				return nil, fmt.Errorf("failed to parse trailer: %w", err)
			}
			table.Trailer = trailer
			foundTrailer = true
			break
		}

		// Parse subsection header (firstObjNum count)
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid subsection header: %s", line)
		}

		firstObjNum, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid first object number: %w", err)
		}

		count, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid count: %w", err)
		}

		// Parse entries for this subsection
		for i := 0; i < count; i++ {
			if !scanner.Scan() {
				return nil, fmt.Errorf("unexpected end of xref subsection")
			}
			entryLine := scanner.Text()

			entry, err := x.parseEntry(entryLine)
			if err != nil {
				return nil, fmt.Errorf("failed to parse xref entry: %w", err)
			}

			objNum := firstObjNum + i
			table.Set(objNum, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	if !foundTrailer {
		return nil, fmt.Errorf("xref table missing trailer")
	}

	return table, nil
}

// parseEntry parses a single XRef entry line
// Format: "nnnnnnnnnn ggggg n" or "nnnnnnnnnn ggggg f"
// nnnnnnnnnn = 10-digit offset
// ggggg = 5-digit generation number
// n/f = in-use flag (n = in use, f = free)
func (x *XRefParser) parseEntry(line string) (*XRefEntry, error) {
	// XRef entries are exactly 20 bytes: "nnnnnnnnnn ggggg n \n"
	// But we might have trailing whitespace, so let's be flexible
	if len(line) < 18 {
		return nil, fmt.Errorf("xref entry too short: %q", line)
	}

	// Extract fields
	offsetStr := strings.TrimSpace(line[0:10])
	genStr := strings.TrimSpace(line[10:16])
	flag := strings.TrimSpace(line[16:18])

	// Parse offset
	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid offset %q: %w", offsetStr, err)
	}

	// Parse generation
	generation, err := strconv.Atoi(genStr)
	if err != nil {
		return nil, fmt.Errorf("invalid generation %q: %w", genStr, err)
	}

	// Parse in-use flag
	inUse := false
	if flag == "n" {
		inUse = true
	} else if flag == "f" {
		inUse = false
	} else {
		return nil, fmt.Errorf("invalid in-use flag: %q", flag)
	}

	return &XRefEntry{
		Offset:     offset,
		Generation: generation,
		InUse:      inUse,
	}, nil
}

// parseTrailer parses the trailer dictionary after the "trailer" keyword
func (x *XRefParser) parseTrailer(scanner *bufio.Scanner) (Dict, error) {
	// Collect all remaining lines until we find a dictionary
	var dictText strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		dictText.WriteString(line)
		dictText.WriteString("\n")

		// Check if we've seen the complete dictionary
		// (Simple heuristic: look for ">>" which ends the dict)
		if strings.Contains(line, ">>") {
			break
		}
	}

	// Parse the dictionary using our existing parser
	parser := NewParser(strings.NewReader(dictText.String()))
	obj, err := parser.ParseObject()
	if err != nil {
		return nil, fmt.Errorf("failed to parse trailer dictionary: %w", err)
	}

	dict, ok := obj.(Dict)
	if !ok {
		return nil, fmt.Errorf("trailer is not a dictionary, got %T", obj)
	}

	return dict, nil
}

// ParseXRefFromEOF finds and parses the XRef table by scanning from EOF
func (x *XRefParser) ParseXRefFromEOF() (*XRefTable, error) {
	offset, err := x.FindXRef()
	if err != nil {
		return nil, fmt.Errorf("failed to find xref: %w", err)
	}

	table, err := x.ParseXRef(offset)
	if err != nil {
		return nil, fmt.Errorf("failed to parse xref: %w", err)
	}

	return table, nil
}

// ParsePrevXRef checks if the trailer has a /Prev entry and parses that XRef table
// This handles incremental updates in PDFs
func (x *XRefParser) ParsePrevXRef(table *XRefTable) (*XRefTable, error) {
	prevObj := table.Trailer.Get("Prev")
	if prevObj == nil {
		return nil, nil // No previous XRef
	}

	prevInt, ok := prevObj.(Int)
	if !ok {
		return nil, fmt.Errorf("invalid /Prev entry type: %T", prevObj)
	}

	prevOffset := int64(prevInt)
	prevTable, err := x.ParseXRef(prevOffset)
	if err != nil {
		return nil, fmt.Errorf("failed to parse previous xref: %w", err)
	}

	return prevTable, nil
}

// MergeXRefTables merges multiple XRef tables (from incremental updates)
// Later entries override earlier ones
func MergeXRefTables(tables ...*XRefTable) *XRefTable {
	if len(tables) == 0 {
		return NewXRefTable()
	}

	merged := NewXRefTable()

	// Process tables in order (earliest first)
	// Later entries will override earlier ones
	for _, table := range tables {
		for objNum, entry := range table.Entries {
			merged.Set(objNum, entry)
		}
		// Keep the last trailer
		merged.Trailer = table.Trailer
	}

	return merged
}

// ParseAllXRefs parses the main XRef table and all previous ones (incremental updates)
// Returns them in order from oldest to newest
func (x *XRefParser) ParseAllXRefs() ([]*XRefTable, error) {
	// Parse main XRef
	mainTable, err := x.ParseXRefFromEOF()
	if err != nil {
		return nil, err
	}

	tables := []*XRefTable{mainTable}

	// Parse previous XRefs
	currentTable := mainTable
	for {
		prevTable, err := x.ParsePrevXRef(currentTable)
		if err != nil {
			return nil, fmt.Errorf("failed to parse prev xref: %w", err)
		}
		if prevTable == nil {
			break // No more previous XRefs
		}

		// Prepend (we want oldest first)
		tables = append([]*XRefTable{prevTable}, tables...)
		currentTable = prevTable
	}

	return tables, nil
}
