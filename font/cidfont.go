package font

import (
	"fmt"

	"github.com/tsawler/tabula/core"
)

// Type0Font represents a Type0 (composite) font in a PDF
// Type0 fonts are used for fonts with large character sets, especially CJK fonts
type Type0Font struct {
	*Font // Embed basic font

	// Type0-specific fields
	Encoding       string
	DescendantFont *CIDFont     // The actual CIDFont
	ToUnicode      *core.Stream // CMap for CID to Unicode mapping
	IsVertical     bool         // true for Identity-V, false for Identity-H
}

// CIDFont represents a CIDFont (Character ID keyed font)
// Used as descendant font in Type0 fonts
type CIDFont struct {
	BaseFont       string
	Subtype        string // CIDFontType0 or CIDFontType2
	CIDSystemInfo  *CIDSystemInfo
	FontDescriptor *FontDescriptor
	DW             float64           // Default width
	W              []WidthRange      // Width specifications
	DW2            [2]float64        // Default width for vertical writing [w1y w1]
	W2             []VerticalMetrics // Vertical metrics
	CIDToGIDMap    *core.Stream      // CID to GID mapping (for CIDFontType2)
}

// CIDSystemInfo identifies a character collection
type CIDSystemInfo struct {
	Registry   string // e.g., "Adobe"
	Ordering   string // e.g., "Japan1", "GB1", "CNS1", "Korea1"
	Supplement int    // Version of the character collection
}

// WidthRange represents a width specification in the W array
type WidthRange struct {
	StartCID int
	EndCID   int
	Width    float64   // Single width for range
	Widths   []float64 // Individual widths (if Width == 0)
}

// VerticalMetrics represents vertical writing metrics in the W2 array
type VerticalMetrics struct {
	StartCID int
	EndCID   int
	W1Y      float64  // Position vector y component
	W1       float64  // Vertical width
	Metrics  []Metric // Individual metrics (if W1Y == 0 && W1 == 0)
}

// Metric represents a single vertical metric
type Metric struct {
	W1Y float64
	W1  float64
}

// NewType0Font creates a Type0 font from a PDF font dictionary
func NewType0Font(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) (*Type0Font, error) {
	// Extract basic font properties
	name := extractName(fontDict.Get("Name"))
	baseFont := extractName(fontDict.Get("BaseFont"))
	subtype := extractName(fontDict.Get("Subtype"))

	if subtype != "Type0" {
		return nil, fmt.Errorf("not a Type0 font: %s", subtype)
	}

	// Create base font
	baseF := NewFont(name, baseFont, subtype)

	t0 := &Type0Font{
		Font: baseF,
	}

	// Parse encoding
	if encodingObj := fontDict.Get("Encoding"); encodingObj != nil {
		t0.Encoding = extractName(encodingObj)

		// Determine if vertical writing mode
		t0.IsVertical = (t0.Encoding == "Identity-V")
	} else {
		t0.Encoding = "Identity-H" // Default
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
			t0.ToUnicode = stream

			// Parse the ToUnicode CMap
			if cmap, err := ParseToUnicodeCMap(stream); err == nil {
				t0.Font.ToUnicodeCMap = cmap
			}
		}
	}

	// Parse descendant font
	if err := t0.parseDescendantFont(fontDict, resolver); err != nil {
		return nil, fmt.Errorf("failed to parse descendant font: %w", err)
	}

	return t0, nil
}

// parseDescendantFont parses the DescendantFonts array
func (t0 *Type0Font) parseDescendantFont(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
	descendantObj := fontDict.Get("DescendantFonts")
	if descendantObj == nil {
		return fmt.Errorf("missing DescendantFonts")
	}

	// Resolve indirect reference
	if ref, ok := descendantObj.(core.IndirectRef); ok {
		obj, err := resolver(ref)
		if err != nil {
			return err
		}
		descendantObj = obj
	}

	// Should be an array
	descendantArray, ok := descendantObj.(core.Array)
	if !ok {
		return fmt.Errorf("DescendantFonts is not an array: %T", descendantObj)
	}

	if len(descendantArray) == 0 {
		return fmt.Errorf("DescendantFonts array is empty")
	}

	// Get first descendant font (Type0 fonts typically have only one)
	descendantFontObj := descendantArray[0]

	// Resolve indirect reference
	if ref, ok := descendantFontObj.(core.IndirectRef); ok {
		obj, err := resolver(ref)
		if err != nil {
			return err
		}
		descendantFontObj = obj
	}

	descendantDict, ok := descendantFontObj.(core.Dict)
	if !ok {
		return fmt.Errorf("descendant font is not a dictionary: %T", descendantFontObj)
	}

	// Parse CIDFont
	cidFont, err := NewCIDFont(descendantDict, resolver)
	if err != nil {
		return fmt.Errorf("failed to parse CIDFont: %w", err)
	}

	t0.DescendantFont = cidFont

	return nil
}

// GetWidth returns the width for a character ID (CID)
func (t0 *Type0Font) GetWidth(r rune) float64 {
	if t0.DescendantFont == nil {
		return 500.0
	}

	// For CIDFonts, the rune is treated as a CID
	return t0.DescendantFont.GetWidthForCID(int(r))
}

// NewCIDFont creates a CIDFont from a PDF font dictionary
func NewCIDFont(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) (*CIDFont, error) {
	baseFont := extractName(fontDict.Get("BaseFont"))
	subtype := extractName(fontDict.Get("Subtype"))

	if subtype != "CIDFontType0" && subtype != "CIDFontType2" {
		return nil, fmt.Errorf("not a CIDFont: %s", subtype)
	}

	cid := &CIDFont{
		BaseFont: baseFont,
		Subtype:  subtype,
		DW:       1000.0, // Default width
	}

	// Parse CIDSystemInfo
	if err := cid.parseCIDSystemInfo(fontDict, resolver); err != nil {
		return nil, fmt.Errorf("failed to parse CIDSystemInfo: %w", err)
	}

	// Parse font descriptor
	if err := cid.parseFontDescriptor(fontDict, resolver); err != nil {
		// Font descriptor may be optional in some cases
		_ = err
	}

	// Parse default width (DW)
	if dwObj := fontDict.Get("DW"); dwObj != nil {
		cid.DW = getNumber(dwObj)
	}

	// Parse width array (W)
	if err := cid.parseWidthArray(fontDict, resolver); err != nil {
		// Non-fatal - we have default width
		_ = err
	}

	// Parse vertical metrics for vertical writing mode
	if dw2Obj := fontDict.Get("DW2"); dw2Obj != nil {
		if ref, ok := dw2Obj.(core.IndirectRef); ok {
			obj, err := resolver(ref)
			if err == nil {
				dw2Obj = obj
			}
		}
		if arr, ok := dw2Obj.(core.Array); ok && len(arr) >= 2 {
			cid.DW2[0] = getNumber(arr[0])
			cid.DW2[1] = getNumber(arr[1])
		}
	}

	// Parse W2 array for vertical metrics
	if err := cid.parseW2Array(fontDict, resolver); err != nil {
		// Non-fatal
		_ = err
	}

	// Parse CIDToGIDMap for CIDFontType2
	if subtype == "CIDFontType2" {
		if mapObj := fontDict.Get("CIDToGIDMap"); mapObj != nil {
			if ref, ok := mapObj.(core.IndirectRef); ok {
				obj, err := resolver(ref)
				if err == nil {
					if stream, ok := obj.(*core.Stream); ok {
						cid.CIDToGIDMap = stream
					}
				}
			} else if stream, ok := mapObj.(*core.Stream); ok {
				cid.CIDToGIDMap = stream
			}
		}
	}

	return cid, nil
}

// parseCIDSystemInfo parses the CIDSystemInfo dictionary
func (cid *CIDFont) parseCIDSystemInfo(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
	sysInfoObj := fontDict.Get("CIDSystemInfo")
	if sysInfoObj == nil {
		return fmt.Errorf("missing CIDSystemInfo")
	}

	// Resolve indirect reference
	if ref, ok := sysInfoObj.(core.IndirectRef); ok {
		obj, err := resolver(ref)
		if err != nil {
			return err
		}
		sysInfoObj = obj
	}

	sysInfoDict, ok := sysInfoObj.(core.Dict)
	if !ok {
		return fmt.Errorf("CIDSystemInfo is not a dictionary: %T", sysInfoObj)
	}

	cid.CIDSystemInfo = &CIDSystemInfo{
		Registry:   extractString(sysInfoDict.Get("Registry")),
		Ordering:   extractString(sysInfoDict.Get("Ordering")),
		Supplement: int(getNumber(sysInfoDict.Get("Supplement"))),
	}

	return nil
}

// parseFontDescriptor parses the font descriptor
func (cid *CIDFont) parseFontDescriptor(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
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

	cid.FontDescriptor = fd

	return nil
}

// parseWidthArray parses the W array for CIDFont widths
// Format: [c [w1 w2 ... wn]] or [cfirst clast w]
func (cid *CIDFont) parseWidthArray(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
	wObj := fontDict.Get("W")
	if wObj == nil {
		return nil // W is optional
	}

	// Resolve indirect reference
	if ref, ok := wObj.(core.IndirectRef); ok {
		obj, err := resolver(ref)
		if err != nil {
			return err
		}
		wObj = obj
	}

	wArray, ok := wObj.(core.Array)
	if !ok {
		return fmt.Errorf("W is not an array: %T", wObj)
	}

	// Parse W array
	for i := 0; i < len(wArray); {
		// First element is always a CID (start of range)
		startCID := int(getNumber(wArray[i]))
		i++

		if i >= len(wArray) {
			break
		}

		// Second element is either:
		// - An array of widths [w1 w2 ... wn]
		// - An end CID for a range
		if widthsArray, ok := wArray[i].(core.Array); ok {
			// Format: c [w1 w2 ... wn]
			widths := make([]float64, len(widthsArray))
			for j, w := range widthsArray {
				widths[j] = getNumber(w)
			}
			cid.W = append(cid.W, WidthRange{
				StartCID: startCID,
				EndCID:   startCID + len(widths) - 1,
				Widths:   widths,
			})
			i++
		} else {
			// Format: cfirst clast w
			endCID := int(getNumber(wArray[i]))
			i++

			if i >= len(wArray) {
				break
			}

			width := getNumber(wArray[i])
			i++

			cid.W = append(cid.W, WidthRange{
				StartCID: startCID,
				EndCID:   endCID,
				Width:    width,
			})
		}
	}

	return nil
}

// parseW2Array parses the W2 array for vertical metrics
func (cid *CIDFont) parseW2Array(fontDict core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
	w2Obj := fontDict.Get("W2")
	if w2Obj == nil {
		return nil // W2 is optional
	}

	// Resolve indirect reference
	if ref, ok := w2Obj.(core.IndirectRef); ok {
		obj, err := resolver(ref)
		if err != nil {
			return err
		}
		w2Obj = obj
	}

	w2Array, ok := w2Obj.(core.Array)
	if !ok {
		return fmt.Errorf("W2 is not an array: %T", w2Obj)
	}

	// Parse W2 array - similar to W but with vertical metrics
	for i := 0; i < len(w2Array); {
		startCID := int(getNumber(w2Array[i]))
		i++

		if i >= len(w2Array) {
			break
		}

		// Check if next element is an array
		if metricsArray, ok := w2Array[i].(core.Array); ok {
			// Format: c [[w1y w1] [w2y w2] ...]
			metrics := make([]Metric, 0)
			for j := 0; j < len(metricsArray); j += 2 {
				if j+1 < len(metricsArray) {
					metrics = append(metrics, Metric{
						W1Y: getNumber(metricsArray[j]),
						W1:  getNumber(metricsArray[j+1]),
					})
				}
			}
			cid.W2 = append(cid.W2, VerticalMetrics{
				StartCID: startCID,
				EndCID:   startCID + len(metrics) - 1,
				Metrics:  metrics,
			})
			i++
		} else {
			// Format: cfirst clast w1y w1
			endCID := int(getNumber(w2Array[i]))
			i++

			if i+1 >= len(w2Array) {
				break
			}

			w1y := getNumber(w2Array[i])
			i++
			w1 := getNumber(w2Array[i])
			i++

			cid.W2 = append(cid.W2, VerticalMetrics{
				StartCID: startCID,
				EndCID:   endCID,
				W1Y:      w1y,
				W1:       w1,
			})
		}
	}

	return nil
}

// GetWidthForCID returns the width for a specific CID
func (cid *CIDFont) GetWidthForCID(cidValue int) float64 {
	// Search in W array
	for _, wr := range cid.W {
		if cidValue >= wr.StartCID && cidValue <= wr.EndCID {
			if wr.Widths != nil {
				// Individual widths
				idx := cidValue - wr.StartCID
				if idx < len(wr.Widths) {
					return wr.Widths[idx]
				}
			} else {
				// Single width for range
				return wr.Width
			}
		}
	}

	// Return default width
	return cid.DW
}

// IsJapanese returns true if this is a Japanese font
func (cid *CIDFont) IsJapanese() bool {
	if cid.CIDSystemInfo == nil {
		return false
	}
	return cid.CIDSystemInfo.Ordering == "Japan1"
}

// IsChinese returns true if this is a Chinese font
func (cid *CIDFont) IsChinese() bool {
	if cid.CIDSystemInfo == nil {
		return false
	}
	return cid.CIDSystemInfo.Ordering == "GB1" || cid.CIDSystemInfo.Ordering == "CNS1"
}

// IsKorean returns true if this is a Korean font
func (cid *CIDFont) IsKorean() bool {
	if cid.CIDSystemInfo == nil {
		return false
	}
	return cid.CIDSystemInfo.Ordering == "Korea1"
}

// IsCJK returns true if this is a CJK (Chinese, Japanese, Korean) font
func (cid *CIDFont) IsCJK() bool {
	return cid.IsJapanese() || cid.IsChinese() || cid.IsKorean()
}

// GetCharacterCollection returns a string identifying the character collection
func (cid *CIDFont) GetCharacterCollection() string {
	if cid.CIDSystemInfo == nil {
		return "Unknown"
	}
	return fmt.Sprintf("%s-%s-%d",
		cid.CIDSystemInfo.Registry,
		cid.CIDSystemInfo.Ordering,
		cid.CIDSystemInfo.Supplement)
}

// extractString extracts a string from a PDF object
func extractString(obj core.Object) string {
	if obj == nil {
		return ""
	}
	if str, ok := obj.(core.String); ok {
		return string(str)
	}
	if name, ok := obj.(core.Name); ok {
		return string(name)
	}
	return ""
}
