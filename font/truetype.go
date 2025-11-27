package font

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/tsawler/tabula/core"
)

// TrueTypeFont represents a TrueType font in a PDF
// TrueType fonts contain glyph outlines as quadratic Bézier curves
type TrueTypeFont struct {
	*Font // Embed basic font

	// TrueType-specific fields
	FirstChar      int
	LastChar       int
	Widths         []float64
	FontDescriptor *FontDescriptor
	ToUnicode      *core.Stream // CMap for character code to Unicode mapping

	// TrueType font program data
	FontProgram []byte            // Raw font program from FontFile2
	Tables      map[string][]byte // Parsed TrueType tables

	// Parsed table data
	unitsPerEm  uint16
	glyphWidths map[uint16]uint16 // Glyph ID -> width
	cmapTable   *CMapTable        // Character to glyph mapping
	isSubset    bool              // Whether this is a subset font
}

// CMapTable represents a TrueType cmap table
type CMapTable struct {
	format   uint16
	encoding map[rune]uint16 // Character code -> Glyph ID
}

// NewTrueTypeFont creates a TrueType font from a PDF font dictionary
func NewTrueTypeFont(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) (*TrueTypeFont, error) {
	// Extract basic font properties
	name := extractName(fontDict.Get("Name"))
	baseFont := extractName(fontDict.Get("BaseFont"))
	subtype := extractName(fontDict.Get("Subtype"))

	if subtype != "TrueType" {
		return nil, fmt.Errorf("not a TrueType font: %s", subtype)
	}

	// Create base font
	baseF := NewFont(name, baseFont, subtype)

	tt := &TrueTypeFont{
		Font:        baseF,
		FirstChar:   0,
		LastChar:    255,
		Tables:      make(map[string][]byte),
		glyphWidths: make(map[uint16]uint16),
	}

	// Check if this is a subset font (name like "ABCDEF+FontName")
	tt.isSubset = isSubsetFont(baseFont)

	// Parse encoding
	if err := tt.parseEncoding(fontDict, resolver); err != nil {
		return nil, fmt.Errorf("failed to parse encoding: %w", err)
	}

	// Parse widths from PDF (these override font program widths)
	if err := tt.parseWidths(fontDict, resolver); err != nil {
		return nil, fmt.Errorf("failed to parse widths: %w", err)
	}

	// Parse font descriptor
	if err := tt.parseFontDescriptor(fontDict, resolver); err != nil {
		// Font descriptor is optional for standard fonts
		_ = err // Suppress unused error
	}

	// Parse ToUnicode CMap if present
	if toUnicodeObj := fontDict.Get("ToUnicode"); toUnicodeObj != nil {
		var stream *core.Stream

		if ref, ok := toUnicodeObj.(core.IndirectRef); ok {
			obj, err := resolver(ref)
			if err == nil {
				if s, ok := obj.(*core.Stream); ok {
					stream = s
				}
			}
		} else if s, ok := toUnicodeObj.(*core.Stream); ok {
			stream = s
		}

		// Store stream and parse CMap
		if stream != nil {
			tt.ToUnicode = stream

			// Parse the ToUnicode CMap
			if cmap, err := ParseToUnicodeCMap(stream); err == nil {
				tt.Font.ToUnicodeCMap = cmap
			}
		}
	}

	// Parse TrueType font program if embedded
	if tt.FontDescriptor != nil && tt.FontDescriptor.FontFile2 != nil {
		if err := tt.parseFontProgram(); err != nil {
			// Non-fatal - we can still use the widths from the PDF
			_ = err
		}
	}

	return tt, nil
}

// isSubsetFont checks if a font is a subset (has a prefix like "ABCDEF+")
func isSubsetFont(baseFontName string) bool {
	// Subset fonts have names like "ABCDEF+FontName"
	// The prefix is 6 uppercase letters followed by +
	if len(baseFontName) < 8 {
		return false
	}

	for i := 0; i < 6; i++ {
		if baseFontName[i] < 'A' || baseFontName[i] > 'Z' {
			return false
		}
	}

	return baseFontName[6] == '+'
}

// parseEncoding extracts the encoding from the font dictionary
func (tt *TrueTypeFont) parseEncoding(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
	encodingObj := fontDict.Get("Encoding")
	if encodingObj == nil {
		// Use default encoding
		tt.Encoding = "WinAnsiEncoding"
		return nil
	}

	// Resolve indirect reference
	if ref, ok := encodingObj.(core.IndirectRef); ok {
		obj, err := resolver(ref)
		if err != nil {
			return err
		}
		encodingObj = obj
	}

	// Check if it's a name (predefined encoding)
	if name, ok := encodingObj.(core.Name); ok {
		tt.Encoding = string(name)
		return nil
	}

	// Check if it's a dictionary (custom encoding with Differences)
	if dict, ok := encodingObj.(core.Dict); ok {
		// Get base encoding
		if baseEnc := dict.Get("BaseEncoding"); baseEnc != nil {
			if name, ok := baseEnc.(core.Name); ok {
				tt.Encoding = string(name)
			}
		} else {
			tt.Encoding = "WinAnsiEncoding"
		}
		return nil
	}

	return fmt.Errorf("invalid encoding type: %T", encodingObj)
}

// parseWidths extracts character width information from the font dictionary
func (tt *TrueTypeFont) parseWidths(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
	// Get FirstChar
	if firstCharObj := fontDict.Get("FirstChar"); firstCharObj != nil {
		if i, ok := firstCharObj.(core.Int); ok {
			tt.FirstChar = int(i)
		}
	}

	// Get LastChar
	if lastCharObj := fontDict.Get("LastChar"); lastCharObj != nil {
		if i, ok := lastCharObj.(core.Int); ok {
			tt.LastChar = int(i)
		}
	}

	// Get Widths array
	widthsObj := fontDict.Get("Widths")
	if widthsObj == nil {
		// No widths array - will use font program widths
		return nil
	}

	// Resolve indirect reference
	if ref, ok := widthsObj.(core.IndirectRef); ok {
		obj, err := resolver(ref)
		if err != nil {
			return err
		}
		widthsObj = obj
	}

	// Parse widths array
	widthsArray, ok := widthsObj.(core.Array)
	if !ok {
		return fmt.Errorf("widths is not an array: %T", widthsObj)
	}

	// Extract width values
	tt.Widths = make([]float64, len(widthsArray))
	for i, w := range widthsArray {
		switch v := w.(type) {
		case core.Int:
			tt.Widths[i] = float64(v)
		case core.Real:
			tt.Widths[i] = float64(v)
		default:
			return fmt.Errorf("invalid width type at index %d: %T", i, w)
		}
	}

	// Update the font's width map
	for i, width := range tt.Widths {
		charCode := tt.FirstChar + i
		if charCode <= tt.LastChar {
			// Map character code to rune
			// TODO: Use proper encoding to map character code to Unicode
			// For now, assume direct mapping for ASCII range
			if charCode < 256 {
				tt.widths[rune(charCode)] = width
			}
		}
	}

	return nil
}

// parseFontDescriptor extracts font descriptor information
func (tt *TrueTypeFont) parseFontDescriptor(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
	fdObj := fontDict.Get("FontDescriptor")
	if fdObj == nil {
		return fmt.Errorf("no font descriptor")
	}

	// Resolve indirect reference
	if ref, ok := fdObj.(core.IndirectRef); ok {
		obj, err := resolver(ref)
		if err != nil {
			return err
		}
		fdObj = obj
	}

	fdDict, ok := fdObj.(core.Dict)
	if !ok {
		return fmt.Errorf("font descriptor is not a dictionary: %T", fdObj)
	}

	fd := &FontDescriptor{}

	// Extract font descriptor fields
	fd.FontName = extractName(fdDict.Get("FontName"))

	if flags := fdDict.Get("Flags"); flags != nil {
		if i, ok := flags.(core.Int); ok {
			fd.Flags = int(i)
		}
	}

	// FontBBox
	if bboxObj := fdDict.Get("FontBBox"); bboxObj != nil {
		if ref, ok := bboxObj.(core.IndirectRef); ok {
			obj, err := resolver(ref)
			if err == nil {
				bboxObj = obj
			}
		}
		if bbox, ok := bboxObj.(core.Array); ok && len(bbox) >= 4 {
			fd.FontBBox[0] = getNumber(bbox[0])
			fd.FontBBox[1] = getNumber(bbox[1])
			fd.FontBBox[2] = getNumber(bbox[2])
			fd.FontBBox[3] = getNumber(bbox[3])
		}
	}

	// Font metrics
	fd.ItalicAngle = getNumber(fdDict.Get("ItalicAngle"))
	fd.Ascent = getNumber(fdDict.Get("Ascent"))
	fd.Descent = getNumber(fdDict.Get("Descent"))
	fd.CapHeight = getNumber(fdDict.Get("CapHeight"))
	fd.StemV = getNumber(fdDict.Get("StemV"))
	fd.StemH = getNumber(fdDict.Get("StemH"))
	fd.AvgWidth = getNumber(fdDict.Get("AvgWidth"))
	fd.MaxWidth = getNumber(fdDict.Get("MaxWidth"))
	fd.MissingWidth = getNumber(fdDict.Get("MissingWidth"))

	// Font programs - FontFile2 for TrueType
	if fontFile2 := fdDict.Get("FontFile2"); fontFile2 != nil {
		if ref, ok := fontFile2.(core.IndirectRef); ok {
			obj, err := resolver(ref)
			if err == nil {
				if stream, ok := obj.(*core.Stream); ok {
					fd.FontFile2 = stream
				}
			}
		} else if stream, ok := fontFile2.(*core.Stream); ok {
			fd.FontFile2 = stream
		}
	}

	tt.FontDescriptor = fd

	return nil
}

// parseFontProgram parses the embedded TrueType font program
func (tt *TrueTypeFont) parseFontProgram() error {
	if tt.FontDescriptor == nil || tt.FontDescriptor.FontFile2 == nil {
		return fmt.Errorf("no font program available")
	}

	// Decode the font program stream
	data, err := tt.FontDescriptor.FontFile2.Decode()
	if err != nil {
		return fmt.Errorf("failed to decode font program: %w", err)
	}

	tt.FontProgram = data

	// Parse TrueType tables
	if err := tt.parseTrueTypeTables(); err != nil {
		return fmt.Errorf("failed to parse TrueType tables: %w", err)
	}

	return nil
}

// parseTrueTypeTables parses the binary TrueType font tables
func (tt *TrueTypeFont) parseTrueTypeTables() error {
	if len(tt.FontProgram) < 12 {
		return fmt.Errorf("font program too short")
	}

	r := bytes.NewReader(tt.FontProgram)

	// Read offset table (font directory)
	var offsetTable struct {
		SfntVersion   uint32
		NumTables     uint16
		SearchRange   uint16
		EntrySelector uint16
		RangeShift    uint16
	}

	if err := binary.Read(r, binary.BigEndian, &offsetTable); err != nil {
		return fmt.Errorf("failed to read offset table: %w", err)
	}

	// Read table directory
	for i := 0; i < int(offsetTable.NumTables); i++ {
		var entry struct {
			Tag      [4]byte
			Checksum uint32
			Offset   uint32
			Length   uint32
		}

		if err := binary.Read(r, binary.BigEndian, &entry); err != nil {
			return fmt.Errorf("failed to read table entry %d: %w", i, err)
		}

		tag := string(entry.Tag[:])

		// Extract table data
		if int(entry.Offset)+int(entry.Length) <= len(tt.FontProgram) {
			tableData := tt.FontProgram[entry.Offset : entry.Offset+entry.Length]
			tt.Tables[tag] = tableData
		}
	}

	// Parse key tables
	if err := tt.parseHeadTable(); err != nil {
		return fmt.Errorf("failed to parse head table: %w", err)
	}

	if err := tt.parseHmtxTable(); err != nil {
		// Non-fatal - we have widths from PDF
		_ = err
	}

	if err := tt.parseCmapTable(); err != nil {
		// Non-fatal - we can use ToUnicode instead
		_ = err
	}

	return nil
}

// parseHeadTable parses the 'head' table for font metrics
func (tt *TrueTypeFont) parseHeadTable() error {
	headData, ok := tt.Tables["head"]
	if !ok {
		return fmt.Errorf("head table not found")
	}

	if len(headData) < 54 {
		return fmt.Errorf("head table too short")
	}

	r := bytes.NewReader(headData)

	// Skip to unitsPerEm (offset 18)
	r.Seek(18, 0)

	if err := binary.Read(r, binary.BigEndian, &tt.unitsPerEm); err != nil {
		return err
	}

	return nil
}

// parseHmtxTable parses the 'hmtx' table for glyph widths
func (tt *TrueTypeFont) parseHmtxTable() error {
	hmtxData, ok := tt.Tables["hmtx"]
	if !ok {
		return fmt.Errorf("hmtx table not found")
	}

	// We also need the hhea table to know numberOfHMetrics
	hheaData, ok := tt.Tables["hhea"]
	if !ok {
		return fmt.Errorf("hhea table not found")
	}

	if len(hheaData) < 36 {
		return fmt.Errorf("hhea table too short")
	}

	r := bytes.NewReader(hheaData)

	// Skip to numberOfHMetrics (offset 34)
	r.Seek(34, 0)

	var numberOfHMetrics uint16
	if err := binary.Read(r, binary.BigEndian, &numberOfHMetrics); err != nil {
		return err
	}

	// Parse hmtx entries
	r = bytes.NewReader(hmtxData)

	for i := uint16(0); i < numberOfHMetrics; i++ {
		var advanceWidth uint16
		var leftSideBearing int16

		if err := binary.Read(r, binary.BigEndian, &advanceWidth); err != nil {
			return err
		}
		if err := binary.Read(r, binary.BigEndian, &leftSideBearing); err != nil {
			return err
		}

		tt.glyphWidths[i] = advanceWidth
	}

	return nil
}

// parseCmapTable parses the 'cmap' table for character to glyph mapping
func (tt *TrueTypeFont) parseCmapTable() error {
	cmapData, ok := tt.Tables["cmap"]
	if !ok {
		return fmt.Errorf("cmap table not found")
	}

	if len(cmapData) < 4 {
		return fmt.Errorf("cmap table too short")
	}

	r := bytes.NewReader(cmapData)

	// Read cmap header
	var version, numTables uint16
	binary.Read(r, binary.BigEndian, &version)
	binary.Read(r, binary.BigEndian, &numTables)

	// Find a suitable subtable (prefer Unicode)
	var bestOffset uint32
	var bestFormat uint16

	for i := uint16(0); i < numTables; i++ {
		var platformID, encodingID uint16
		var offset uint32

		binary.Read(r, binary.BigEndian, &platformID)
		binary.Read(r, binary.BigEndian, &encodingID)
		binary.Read(r, binary.BigEndian, &offset)

		// Prefer Unicode platform (3) with Unicode BMP encoding (1)
		if platformID == 3 && encodingID == 1 {
			bestOffset = offset
			break
		}

		// Fallback to first found table
		if bestOffset == 0 {
			bestOffset = offset
		}
	}

	if bestOffset == 0 {
		return fmt.Errorf("no suitable cmap subtable found")
	}

	// Read the subtable format
	r.Seek(int64(bestOffset), 0)
	binary.Read(r, binary.BigEndian, &bestFormat)

	tt.cmapTable = &CMapTable{
		format:   bestFormat,
		encoding: make(map[rune]uint16),
	}

	// Parse based on format
	// Format 4 is most common for Unicode BMP
	if bestFormat == 4 {
		return tt.parseCmapFormat4(r)
	}

	// Other formats would be implemented here
	return fmt.Errorf("cmap format %d not yet supported", bestFormat)
}

// parseCmapFormat4 parses a format 4 cmap subtable
func (tt *TrueTypeFont) parseCmapFormat4(r *bytes.Reader) error {
	// Format 4 structure:
	// uint16 format
	// uint16 length
	// uint16 language
	// uint16 segCountX2
	// ... (complex structure for segment mapping)

	var length, language, segCountX2 uint16
	binary.Read(r, binary.BigEndian, &length)
	binary.Read(r, binary.BigEndian, &language)
	binary.Read(r, binary.BigEndian, &segCountX2)

	segCount := segCountX2 / 2

	// Skip searchRange, entrySelector, rangeShift
	r.Seek(6, 1)

	// Read endCode array
	endCode := make([]uint16, segCount)
	for i := range endCode {
		binary.Read(r, binary.BigEndian, &endCode[i])
	}

	// Skip reservedPad
	r.Seek(2, 1)

	// Read startCode array
	startCode := make([]uint16, segCount)
	for i := range startCode {
		binary.Read(r, binary.BigEndian, &startCode[i])
	}

	// For simplicity, we'll just store a basic mapping
	// Full implementation would handle idDelta and idRangeOffset
	for i := range startCode {
		for c := startCode[i]; c <= endCode[i]; c++ {
			tt.cmapTable.encoding[rune(c)] = c // Simplified mapping
		}
	}

	return nil
}

// GetGlyphID returns the glyph ID for a character
func (tt *TrueTypeFont) GetGlyphID(r rune) uint16 {
	if tt.cmapTable != nil {
		if gid, ok := tt.cmapTable.encoding[r]; ok {
			return gid
		}
	}
	return 0 // .notdef glyph
}

// GetWidthFromGlyph gets the width for a glyph ID
func (tt *TrueTypeFont) GetWidthFromGlyph(glyphID uint16) float64 {
	if width, ok := tt.glyphWidths[glyphID]; ok {
		if tt.unitsPerEm > 0 {
			// Convert from font units to 1000ths of em
			return float64(width) * 1000.0 / float64(tt.unitsPerEm)
		}
		return float64(width)
	}
	return 500.0 // Default
}
