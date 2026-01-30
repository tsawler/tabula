package core

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// XRefEntryType identifies the type of a cross-reference table entry.
type XRefEntryType int

// XRef entry type constants.
const (
	// XRefEntryFree indicates a free (deleted) object entry.
	XRefEntryFree XRefEntryType = 0
	// XRefEntryUncompressed indicates an in-use object at a byte offset in the file.
	XRefEntryUncompressed XRefEntryType = 1
	// XRefEntryCompressed indicates an object stored in an object stream (PDF 1.5+).
	XRefEntryCompressed XRefEntryType = 2
)

// String returns a human-readable name for the entry type.
func (t XRefEntryType) String() string {
	switch t {
	case XRefEntryFree:
		return "free"
	case XRefEntryUncompressed:
		return "uncompressed"
	case XRefEntryCompressed:
		return "compressed"
	default:
		return "unknown"
	}
}

// XRefEntry represents a single entry in the cross-reference table,
// describing where an object is located in the PDF file.
type XRefEntry struct {
	Type       XRefEntryType // Entry type (free, uncompressed, or compressed)
	Offset     int64         // Byte offset (uncompressed) or object stream number (compressed)
	Generation int           // Generation number (uncompressed) or index within object stream (compressed)
	InUse      bool          // True if object is in use (Type != XRefEntryFree)
}

// XRefTable represents a PDF cross-reference table, which maps object numbers
// to their locations in the file. It includes the trailer dictionary containing
// document-level information.
type XRefTable struct {
	Entries  map[int]*XRefEntry // Map from object number to entry
	Trailer  Dict               // Trailer dictionary with /Root, /Info, /Size, etc.
	IsStream bool               // True if this XRef came from a stream (PDF 1.5+)
}

// NewXRefTable creates a new empty cross-reference table.
func NewXRefTable() *XRefTable {
	return &XRefTable{
		Entries: make(map[int]*XRefEntry),
		Trailer: make(Dict),
	}
}

// Get returns the entry for the given object number and a boolean indicating
// whether the entry exists.
func (x *XRefTable) Get(objNum int) (*XRefEntry, bool) {
	entry, ok := x.Entries[objNum]
	return entry, ok
}

// Set adds or updates an entry for the given object number.
func (x *XRefTable) Set(objNum int, entry *XRefEntry) {
	x.Entries[objNum] = entry
}

// Size returns the number of entries in the table.
func (x *XRefTable) Size() int {
	return len(x.Entries)
}

// XRefParser parses PDF cross-reference tables from a seekable reader.
// It supports both traditional xref tables (PDF 1.0-1.4) and xref streams (PDF 1.5+).
type XRefParser struct {
	reader   io.ReadSeeker
	startPos int64 // Starting position for current parse
}

// NewXRefParser creates a new XRef parser for the given reader.
func NewXRefParser(r io.ReadSeeker) *XRefParser {
	return &XRefParser{
		reader: r,
	}
}

// FindXRef finds the byte offset of the xref table by scanning from EOF.
// PDF files end with "startxref\n<offset>\n%%EOF", where offset points to the xref.
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

// ParseXRef parses the xref table at the given byte offset.
// It auto-detects and handles both traditional xref tables (PDF 1.0-1.4)
// and xref streams (PDF 1.5+).
func (x *XRefParser) ParseXRef(offset int64) (*XRefTable, error) {
	// Seek to the XRef table
	_, err := x.reader.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to xref: %w", err)
	}

	x.startPos = offset

	// Determine if this is a traditional XRef table or XRef stream
	isStream, err := x.isXRefStream()
	if err != nil {
		return nil, fmt.Errorf("failed to detect xref type: %w", err)
	}

	// Reset to start position
	_, err = x.reader.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek back to xref: %w", err)
	}

	if isStream {
		return x.parseXRefStream()
	}
	return x.parseTraditionalXRef()
}

// isXRefStream checks if the xref at the current position is a stream (PDF 1.5+)
// rather than a traditional table. Traditional tables start with "xref", while
// streams start with an object definition like "5 0 obj".
func (x *XRefParser) isXRefStream() (bool, error) {
	scanner := bufio.NewScanner(x.reader)
	if !scanner.Scan() {
		return false, fmt.Errorf("failed to read first line")
	}
	line := strings.TrimSpace(scanner.Text())

	// Traditional XRef starts with "xref" keyword
	if line == "xref" {
		return false, nil
	}

	// XRef stream starts with an object definition: "num gen obj"
	// e.g., "5 0 obj"
	// Note: Some PDFs use only CR as line terminator, causing Scanner to read
	// past the object definition into the dictionary. We check for >= 3 parts
	// to handle this case.
	// Also, some PDFs have no whitespace between "obj" and the dictionary,
	// e.g., "530 0 obj<<..." - we check if the third part starts with "obj".
	parts := strings.Fields(line)
	if len(parts) >= 3 && (parts[2] == "obj" || strings.HasPrefix(parts[2], "obj<<") || strings.HasPrefix(parts[2], "obj<")) {
		return true, nil
	}

	return false, fmt.Errorf("unrecognized xref format: %s", line)
}

// parseTraditionalXRef parses a traditional xref table (PDF 1.0-1.4).
// The format is: "xref\n<subsections>\ntrailer\n<dict>\nstartxref\n<offset>\n%%EOF"
func (x *XRefParser) parseTraditionalXRef() (*XRefTable, error) {
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

	table.IsStream = false
	return table, nil
}

// parseXRefStream parses an xref stream (PDF 1.5+).
// XRef streams store cross-reference data in a compressed stream object.
func (x *XRefParser) parseXRefStream() (*XRefTable, error) {
	// Parse the entire indirect object "num gen obj << ... >> stream ... endstream endobj"
	// Don't use Scanner here - it buffers ahead and corrupts the reader position
	parser := NewParser(x.reader)
	indObj, err := parser.ParseIndirectObject()
	if err != nil {
		return nil, fmt.Errorf("failed to parse xref stream object: %w", err)
	}

	stream, ok := indObj.Object.(*Stream)
	if !ok {
		return nil, fmt.Errorf("xref is not a stream object, got %T", indObj.Object)
	}

	// Verify this is an XRef stream
	typeObj := stream.Dict.Get("Type")
	if typeObj == nil {
		return nil, fmt.Errorf("xref stream missing /Type")
	}
	typeName, ok := typeObj.(Name)
	if !ok || string(typeName) != "XRef" {
		return nil, fmt.Errorf("stream is not an XRef stream, got type: %v", typeObj)
	}

	// Decode the stream data
	data, err := stream.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode xref stream: %w", err)
	}

	// Parse /Size - total number of entries
	sizeObj := stream.Dict.Get("Size")
	if sizeObj == nil {
		return nil, fmt.Errorf("xref stream missing /Size")
	}
	size, ok := sizeObj.(Int)
	if !ok {
		return nil, fmt.Errorf("invalid /Size type: %T", sizeObj)
	}

	// Parse /Index array (optional, default is [0 Size])
	index := []int{0, int(size)}
	if indexObj := stream.Dict.Get("Index"); indexObj != nil {
		indexArr, ok := indexObj.(Array)
		if !ok {
			return nil, fmt.Errorf("invalid /Index type: %T", indexObj)
		}
		index = make([]int, len(indexArr))
		for i, val := range indexArr {
			intVal, ok := val.(Int)
			if !ok {
				return nil, fmt.Errorf("invalid /Index element type: %T", val)
			}
			index[i] = int(intVal)
		}
	}

	// Parse /W array - field widths [type field1 field2]
	wObj := stream.Dict.Get("W")
	if wObj == nil {
		return nil, fmt.Errorf("xref stream missing /W")
	}
	wArr, ok := wObj.(Array)
	if !ok {
		return nil, fmt.Errorf("invalid /W type: %T", wObj)
	}
	if len(wArr) != 3 {
		return nil, fmt.Errorf("invalid /W array length: %d (expected 3)", len(wArr))
	}

	w := make([]int, 3)
	for i, val := range wArr {
		intVal, ok := val.(Int)
		if !ok {
			return nil, fmt.Errorf("invalid /W element type: %T", val)
		}
		w[i] = int(intVal)
	}

	// Parse entries from binary data
	table := NewXRefTable()
	table.IsStream = true

	// The stream dictionary itself serves as the trailer
	table.Trailer = stream.Dict

	// Process subsections defined by /Index
	dataOffset := 0
	for i := 0; i < len(index); i += 2 {
		firstObjNum := index[i]
		count := index[i+1]

		for j := 0; j < count; j++ {
			objNum := firstObjNum + j

			// Read fields according to /W array
			entry, bytesRead, err := x.parseXRefStreamEntry(data[dataOffset:], w)
			if err != nil {
				return nil, fmt.Errorf("failed to parse xref stream entry %d: %w", objNum, err)
			}
			dataOffset += bytesRead

			table.Set(objNum, entry)
		}
	}

	return table, nil
}

// parseXRefStreamEntry parses a single entry from xref stream binary data.
// The w array specifies the byte widths of the three fields (type, field1, field2).
// Returns the entry, number of bytes consumed, and any error.
func (x *XRefParser) parseXRefStreamEntry(data []byte, w []int) (*XRefEntry, int, error) {
	totalWidth := w[0] + w[1] + w[2]
	if len(data) < totalWidth {
		return nil, 0, fmt.Errorf("insufficient data for xref entry (need %d, have %d)", totalWidth, len(data))
	}

	// Read fields as big-endian integers
	offset := 0

	// Field 0: Type (0=free, 1=in-use uncompressed, 2=in object stream)
	// Default to 1 if width is 0
	entryType := int64(1)
	if w[0] > 0 {
		entryType = readBigEndianInt(data[offset:offset+w[0]], w[0])
		offset += w[0]
	}

	// Field 1: For type 1: byte offset; for type 2: object stream number
	field1 := readBigEndianInt(data[offset:offset+w[1]], w[1])
	offset += w[1]

	// Field 2: For type 1: generation; for type 2: index within object stream
	field2 := readBigEndianInt(data[offset:offset+w[2]], w[2])

	// Create entry based on type
	entry := &XRefEntry{}

	switch entryType {
	case 0:
		// Free entry
		entry.Type = XRefEntryFree
		entry.InUse = false
		entry.Offset = field1          // Next free object number
		entry.Generation = int(field2) // Generation if reused
	case 1:
		// In-use uncompressed entry
		entry.Type = XRefEntryUncompressed
		entry.InUse = true
		entry.Offset = field1
		entry.Generation = int(field2)
	case 2:
		// Object in object stream (PDF 1.5+)
		// Offset stores the object stream number
		// Generation stores the index within the object stream
		entry.Type = XRefEntryCompressed
		entry.InUse = true
		entry.Offset = field1          // Object stream number
		entry.Generation = int(field2) // Index within object stream
	default:
		return nil, 0, fmt.Errorf("invalid xref entry type: %d", entryType)
	}

	return entry, totalWidth, nil
}

// readBigEndianInt reads a big-endian integer of the specified byte width from data.
func readBigEndianInt(data []byte, width int) int64 {
	if width == 0 {
		return 0
	}
	if width > 8 {
		width = 8 // Limit to 64-bit
	}

	var result int64
	for i := 0; i < width; i++ {
		result = (result << 8) | int64(data[i])
	}
	return result
}

// parseEntry parses a single traditional xref entry line.
// Format: "nnnnnnnnnn ggggg n" or "nnnnnnnnnn ggggg f", where:
//   - nnnnnnnnnn = 10-digit byte offset
//   - ggggg = 5-digit generation number
//   - n/f = in-use flag (n = in use, f = free)
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

	// Parse in-use flag and determine entry type
	// Traditional XRef tables only have free and uncompressed entries
	// (Compressed entries only exist in XRef streams, PDF 1.5+)
	var entryType XRefEntryType
	var inUse bool
	if flag == "n" {
		entryType = XRefEntryUncompressed
		inUse = true
	} else if flag == "f" {
		entryType = XRefEntryFree
		inUse = false
	} else {
		return nil, fmt.Errorf("invalid in-use flag: %q", flag)
	}

	return &XRefEntry{
		Type:       entryType,
		Offset:     offset,
		Generation: generation,
		InUse:      inUse,
	}, nil
}

// parseTrailer parses the trailer dictionary after the "trailer" keyword.
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

// ParseXRefFromEOF locates and parses the xref table by scanning from the end
// of the file to find the startxref offset.
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

// ParsePrevXRef checks if the trailer has a /Prev entry and parses that xref table.
// This handles incremental updates in PDFs, where each update adds a new xref
// table that points to the previous one.
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

// MergeXRefTables merges multiple xref tables from incremental updates.
// Tables should be provided in chronological order (oldest first); later entries
// override earlier ones for the same object number.
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

// ParseAllXRefs parses the main xref table and all previous ones from incremental
// updates, following /Prev links. Returns tables in chronological order (oldest first).
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
