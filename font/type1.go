package font

import (
	"fmt"

	"github.com/tsawler/tabula/core"
)

// Type1Font represents a Type1 font in a PDF
// Type1 fonts are the original PostScript fonts and one of the most common font types in PDFs
type Type1Font struct {
	*Font // Embed basic font

	// Type1-specific fields
	FirstChar int
	LastChar  int
	Widths    []float64
	FontDescriptor *FontDescriptor
	ToUnicode *core.Stream // CMap for character code to Unicode mapping
}

// FontDescriptor contains font metrics and properties
type FontDescriptor struct {
	FontName   string
	Flags      int
	FontBBox   [4]float64 // [llx lly urx ury]
	ItalicAngle float64
	Ascent     float64
	Descent    float64
	CapHeight  float64
	StemV      float64
	StemH      float64
	AvgWidth   float64
	MaxWidth   float64
	MissingWidth float64
	FontFile   *core.Stream // Type1 font program
	FontFile2  *core.Stream // TrueType font program
	FontFile3  *core.Stream // Type1C or CIDFont program
}

// NewType1Font creates a Type1 font from a PDF font dictionary
func NewType1Font(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) (*Type1Font, error) {
	// Extract basic font properties
	name := extractName(fontDict.Get("Name"))
	baseFont := extractName(fontDict.Get("BaseFont"))
	subtype := extractName(fontDict.Get("Subtype"))

	if subtype != "Type1" {
		return nil, fmt.Errorf("not a Type1 font: %s", subtype)
	}

	// Create base font
	baseF := NewFont(name, baseFont, subtype)

	t1 := &Type1Font{
		Font:      baseF,
		FirstChar: 0,
		LastChar:  255,
	}

	// Parse encoding
	if err := t1.parseEncoding(fontDict, resolver); err != nil {
		return nil, fmt.Errorf("failed to parse encoding: %w", err)
	}

	// Parse widths
	if err := t1.parseWidths(fontDict, resolver); err != nil {
		return nil, fmt.Errorf("failed to parse widths: %w", err)
	}

	// Parse font descriptor (optional - if missing, widths and metrics will come from other sources)
	if err := t1.parseFontDescriptor(fontDict, resolver); err != nil {
		// Font descriptor is optional for Standard 14 fonts, and may be missing for custom fonts too
		// Just log the error but don't fail
		// In a real implementation, you might want to use a logger here
		_ = err // Suppress unused error
	}

	// Parse ToUnicode CMap if present
	if toUnicodeObj := fontDict.Get("ToUnicode"); toUnicodeObj != nil {
		if ref, ok := toUnicodeObj.(core.IndirectRef); ok {
			obj, err := resolver(ref)
			if err == nil {
				if stream, ok := obj.(*core.Stream); ok {
					t1.ToUnicode = stream
				}
			}
		} else if stream, ok := toUnicodeObj.(*core.Stream); ok {
			t1.ToUnicode = stream
		}
	}

	return t1, nil
}

// parseEncoding extracts the encoding from the font dictionary
func (t1 *Type1Font) parseEncoding(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
	encodingObj := fontDict.Get("Encoding")
	if encodingObj == nil {
		// Use default encoding
		t1.Encoding = "StandardEncoding"
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
		t1.Encoding = string(name)
		return nil
	}

	// Check if it's a dictionary (custom encoding with Differences)
	if dict, ok := encodingObj.(core.Dict); ok {
		// Get base encoding
		if baseEnc := dict.Get("BaseEncoding"); baseEnc != nil {
			if name, ok := baseEnc.(core.Name); ok {
				t1.Encoding = string(name)
			}
		} else {
			t1.Encoding = "StandardEncoding"
		}

		// Apply differences
		if diffsObj := dict.Get("Differences"); diffsObj != nil {
			// Resolve if indirect
			if ref, ok := diffsObj.(core.IndirectRef); ok {
				obj, err := resolver(ref)
				if err != nil {
					return err
				}
				diffsObj = obj
			}

			if diffs, ok := diffsObj.(core.Array); ok {
				if err := t1.applyEncodingDifferences(diffs); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return fmt.Errorf("invalid encoding type: %T", encodingObj)
}

// applyEncodingDifferences applies the Differences array to customize encoding
// Format: [code name1 name2 ... code name1 name2 ...]
func (t1 *Type1Font) applyEncodingDifferences(diffs core.Array) error {
	code := 0
	for _, item := range diffs {
		switch v := item.(type) {
		case core.Int:
			// This is a starting code
			code = int(v)
		case core.Name:
			// This is a glyph name mapped to current code
			// We would need a glyph name to Unicode mapping table here
			// For now, just increment the code
			// TODO: Implement proper glyph name to Unicode mapping
			code++
		default:
			return fmt.Errorf("invalid differences array item: %T", item)
		}
	}
	return nil
}

// parseWidths extracts character width information from the font dictionary
func (t1 *Type1Font) parseWidths(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
	// Get FirstChar
	if firstCharObj := fontDict.Get("FirstChar"); firstCharObj != nil {
		if i, ok := firstCharObj.(core.Int); ok {
			t1.FirstChar = int(i)
		}
	}

	// Get LastChar
	if lastCharObj := fontDict.Get("LastChar"); lastCharObj != nil {
		if i, ok := lastCharObj.(core.Int); ok {
			t1.LastChar = int(i)
		}
	}

	// Get Widths array
	widthsObj := fontDict.Get("Widths")
	if widthsObj == nil {
		// No widths array - use defaults
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
	t1.Widths = make([]float64, len(widthsArray))
	for i, w := range widthsArray {
		switch v := w.(type) {
		case core.Int:
			t1.Widths[i] = float64(v)
		case core.Real:
			t1.Widths[i] = float64(v)
		default:
			return fmt.Errorf("invalid width type at index %d: %T", i, w)
		}
	}

	// Update the font's width map
	for i, width := range t1.Widths {
		charCode := t1.FirstChar + i
		if charCode <= t1.LastChar {
			// Map character code to rune
			// TODO: Use proper encoding to map character code to Unicode
			// For now, assume direct mapping for ASCII range
			if charCode < 256 {
				t1.widths[rune(charCode)] = width
			}
		}
	}

	return nil
}

// parseFontDescriptor extracts font descriptor information
func (t1 *Type1Font) parseFontDescriptor(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
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

	// Font programs
	if fontFile := fdDict.Get("FontFile"); fontFile != nil {
		if ref, ok := fontFile.(core.IndirectRef); ok {
			obj, err := resolver(ref)
			if err == nil {
				if stream, ok := obj.(*core.Stream); ok {
					fd.FontFile = stream
				}
			}
		}
	}

	if fontFile2 := fdDict.Get("FontFile2"); fontFile2 != nil {
		if ref, ok := fontFile2.(core.IndirectRef); ok {
			obj, err := resolver(ref)
			if err == nil {
				if stream, ok := obj.(*core.Stream); ok {
					fd.FontFile2 = stream
				}
			}
		}
	}

	if fontFile3 := fdDict.Get("FontFile3"); fontFile3 != nil {
		if ref, ok := fontFile3.(core.IndirectRef); ok {
			obj, err := resolver(ref)
			if err == nil {
				if stream, ok := obj.(*core.Stream); ok {
					fd.FontFile3 = stream
				}
			}
		}
	}

	t1.FontDescriptor = fd

	// Use MissingWidth from font descriptor if available
	if fd.MissingWidth > 0 {
		// Update default width
		// This would be used for characters not in the Widths array
	}

	return nil
}

// Helper functions

// extractName extracts a name from a PDF object
func extractName(obj core.Object) string {
	if obj == nil {
		return ""
	}
	if name, ok := obj.(core.Name); ok {
		return string(name)
	}
	if str, ok := obj.(core.String); ok {
		return string(str)
	}
	return ""
}

// getNumber extracts a numeric value from a PDF object
func getNumber(obj core.Object) float64 {
	if obj == nil {
		return 0
	}
	switch v := obj.(type) {
	case core.Int:
		return float64(v)
	case core.Real:
		return float64(v)
	default:
		return 0
	}
}
