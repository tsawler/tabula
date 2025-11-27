package font

// Font represents a PDF font
type Font struct {
	Name     string
	BaseFont string
	Subtype  string
	Encoding string

	// Character width information
	widths map[rune]float64

	// ToUnicode CMap for character code to Unicode mapping
	ToUnicodeCMap *CMap
}

// NewFont creates a new font
func NewFont(name, baseFont, subtype string) *Font {
	f := &Font{
		Name:     name,
		BaseFont: baseFont,
		Subtype:  subtype,
		Encoding: "WinAnsiEncoding", // Default
		widths:   make(map[rune]float64),
	}

	// Load default widths for Standard 14 fonts
	f.loadStandardWidths()

	return f
}

// GetWidth returns the width of a character (in 1000ths of em)
func (f *Font) GetWidth(r rune) float64 {
	if w, ok := f.widths[r]; ok {
		return w
	}

	// Default width if not found
	return 500.0
}

// GetStringWidth calculates the total width of a string
func (f *Font) GetStringWidth(s string) float64 {
	total := 0.0
	for _, r := range s {
		total += f.GetWidth(r)
	}
	return total
}

// IsStandardFont returns true if this is one of the Standard 14 fonts
func (f *Font) IsStandardFont() bool {
	_, ok := standardFonts[f.BaseFont]
	return ok
}

// DecodeString decodes a string of character codes to Unicode
// Priority order:
// 1. Use ToUnicode CMap if present (most accurate)
// 2. Check for UTF-16 Byte Order Mark (BOM) - FEFF or FFFE
// 3. Use font's Encoding property (standard encodings)
// 4. Fall back to raw bytes as string
// All decoded strings are normalized to NFC for consistent embeddings
func (f *Font) DecodeString(data []byte) string {
	var decoded string

	// Priority 1: ToUnicode CMap (most accurate)
	if f.ToUnicodeCMap != nil {
		decoded = f.ToUnicodeCMap.LookupString(data)
		return NormalizeUnicode(decoded)
	}

	// Priority 2: Check for UTF-16 Byte Order Mark (BOM)
	// PDF hex strings starting with FEFF or FFFE are UTF-16 encoded
	if len(data) >= 2 {
		if data[0] == 0xFE && data[1] == 0xFF {
			// UTF-16BE (Big Endian)
			decoded = DecodeUTF16BE(data[2:])
			return NormalizeUnicode(decoded)
		} else if data[0] == 0xFF && data[1] == 0xFE {
			// UTF-16LE (Little Endian)
			decoded = DecodeUTF16LE(data[2:])
			return NormalizeUnicode(decoded)
		}
	}

	// Priority 3: Use font's Encoding property
	if f.Encoding != "" {
		enc := GetEncoding(f.Encoding)
		decoded = enc.DecodeString(data)
		return NormalizeUnicode(decoded)
	}

	// Priority 4: Fall back to raw bytes as string
	decoded = string(data)
	return NormalizeUnicode(decoded)
}

// IsVertical returns true if this font uses vertical writing mode
// Vertical writing is indicated by the Identity-V encoding, commonly used for
// East Asian languages (Chinese, Japanese, Korean) where text flows top-to-bottom
func (f *Font) IsVertical() bool {
	return IsVerticalEncoding(f.Encoding)
}

// IsVerticalEncoding checks if an encoding name indicates vertical writing mode
// Identity-V is used for vertical text in CJK fonts
// Identity-H (or any other encoding) is horizontal
func IsVerticalEncoding(encoding string) bool {
	return encoding == "Identity-V"
}

// loadStandardWidths loads default widths for Standard 14 fonts
func (f *Font) loadStandardWidths() {
	// For Standard 14 fonts, use predefined widths
	if widths, ok := standardFonts[f.BaseFont]; ok {
		// Copy standard widths
		for r, w := range widths {
			f.widths[r] = w
		}
	} else {
		// For non-standard fonts, use default widths
		// Real implementation would parse font metrics from the PDF
		f.setDefaultWidths()
	}
}

// setDefaultWidths sets default widths for all printable ASCII characters
func (f *Font) setDefaultWidths() {
	// Use Helvetica widths as default
	for r := rune(32); r <= 126; r++ {
		if w, ok := helveticaWidths[r]; ok {
			f.widths[r] = w
		} else {
			f.widths[r] = 500.0 // Fallback
		}
	}
}

// Standard 14 font names
var standardFonts = map[string]map[rune]float64{
	"Helvetica":             helveticaWidths,
	"Helvetica-Bold":        helveticaBoldWidths,
	"Helvetica-Oblique":     helveticaWidths,
	"Helvetica-BoldOblique": helveticaBoldWidths,
	"Times-Roman":           timesWidths,
	"Times-Bold":            timesBoldWidths,
	"Times-Italic":          timesWidths,
	"Times-BoldItalic":      timesBoldWidths,
	"Courier":               courierWidths,
	"Courier-Bold":          courierWidths,
	"Courier-Oblique":       courierWidths,
	"Courier-BoldOblique":   courierWidths,
	"Symbol":                symbolWidths,
	"ZapfDingbats":          zapfDingbatsWidths,
}

// Helvetica widths (in 1000ths of em) - simplified version
// Only includes common ASCII characters
var helveticaWidths = map[rune]float64{
	' ':  278,
	'!':  278,
	'"':  355,
	'#':  556,
	'$':  556,
	'%':  889,
	'&':  667,
	'\'': 191,
	'(':  333,
	')':  333,
	'*':  389,
	'+':  584,
	',':  278,
	'-':  333,
	'.':  278,
	'/':  278,
	'0':  556,
	'1':  556,
	'2':  556,
	'3':  556,
	'4':  556,
	'5':  556,
	'6':  556,
	'7':  556,
	'8':  556,
	'9':  556,
	':':  278,
	';':  278,
	'<':  584,
	'=':  584,
	'>':  584,
	'?':  556,
	'@':  1015,
	'A':  667,
	'B':  667,
	'C':  722,
	'D':  722,
	'E':  667,
	'F':  611,
	'G':  778,
	'H':  722,
	'I':  278,
	'J':  500,
	'K':  667,
	'L':  556,
	'M':  833,
	'N':  722,
	'O':  778,
	'P':  667,
	'Q':  778,
	'R':  722,
	'S':  667,
	'T':  611,
	'U':  722,
	'V':  667,
	'W':  944,
	'X':  667,
	'Y':  667,
	'Z':  611,
	'[':  278,
	'\\': 278,
	']':  278,
	'^':  469,
	'_':  556,
	'`':  333,
	'a':  556,
	'b':  556,
	'c':  500,
	'd':  556,
	'e':  556,
	'f':  278,
	'g':  556,
	'h':  556,
	'i':  222,
	'j':  222,
	'k':  500,
	'l':  222,
	'm':  833,
	'n':  556,
	'o':  556,
	'p':  556,
	'q':  556,
	'r':  333,
	's':  500,
	't':  278,
	'u':  556,
	'v':  500,
	'w':  722,
	'x':  500,
	'y':  500,
	'z':  500,
	'{':  334,
	'|':  260,
	'}':  334,
	'~':  584,
}

// Helvetica-Bold widths (simplified)
var helveticaBoldWidths = map[rune]float64{
	' ': 278,
	'A': 722,
	'B': 722,
	'C': 722,
	'D': 722,
	'E': 667,
	'F': 611,
	'G': 778,
	'H': 722,
	'I': 278,
	'J': 556,
	'K': 722,
	'L': 611,
	'M': 833,
	'N': 722,
	'O': 778,
	'P': 667,
	'Q': 778,
	'R': 722,
	'S': 667,
	'T': 611,
	'U': 722,
	'V': 667,
	'W': 944,
	'X': 667,
	'Y': 667,
	'Z': 611,
	'a': 556,
	'b': 611,
	'c': 556,
	'd': 611,
	'e': 556,
	'f': 333,
	'g': 611,
	'h': 611,
	'i': 278,
	'j': 278,
	'k': 556,
	'l': 278,
	'm': 889,
	'n': 611,
	'o': 611,
	'p': 611,
	'q': 611,
	'r': 389,
	's': 556,
	't': 333,
	'u': 611,
	'v': 556,
	'w': 778,
	'x': 556,
	'y': 556,
	'z': 500,
}

// Times-Roman widths (simplified)
var timesWidths = map[rune]float64{
	' ': 250,
	'A': 722,
	'B': 667,
	'C': 667,
	'D': 722,
	'E': 611,
	'F': 556,
	'G': 722,
	'H': 722,
	'I': 333,
	'J': 389,
	'K': 722,
	'L': 611,
	'M': 889,
	'N': 722,
	'O': 722,
	'P': 556,
	'Q': 722,
	'R': 667,
	'S': 556,
	'T': 611,
	'U': 722,
	'V': 722,
	'W': 944,
	'X': 722,
	'Y': 722,
	'Z': 611,
	'a': 444,
	'b': 500,
	'c': 444,
	'd': 500,
	'e': 444,
	'f': 333,
	'g': 500,
	'h': 500,
	'i': 278,
	'j': 278,
	'k': 500,
	'l': 278,
	'm': 778,
	'n': 500,
	'o': 500,
	'p': 500,
	'q': 500,
	'r': 333,
	's': 389,
	't': 278,
	'u': 500,
	'v': 500,
	'w': 722,
	'x': 500,
	'y': 500,
	'z': 444,
}

// Times-Bold widths (simplified)
var timesBoldWidths = map[rune]float64{
	' ': 250,
	'A': 722,
	'B': 667,
	'C': 722,
	'D': 722,
	'E': 667,
	'F': 611,
	'G': 778,
	'H': 778,
	'I': 389,
	'J': 500,
	'K': 778,
	'L': 667,
	'M': 944,
	'N': 722,
	'O': 778,
	'P': 611,
	'Q': 778,
	'R': 722,
	'S': 556,
	'T': 667,
	'U': 722,
	'V': 722,
	'W': 1000,
	'X': 722,
	'Y': 722,
	'Z': 667,
	'a': 500,
	'b': 556,
	'c': 444,
	'd': 556,
	'e': 444,
	'f': 333,
	'g': 500,
	'h': 556,
	'i': 278,
	'j': 333,
	'k': 556,
	'l': 278,
	'm': 833,
	'n': 556,
	'o': 500,
	'p': 556,
	'q': 556,
	'r': 444,
	's': 389,
	't': 333,
	'u': 556,
	'v': 500,
	'w': 722,
	'x': 500,
	'y': 500,
	'z': 444,
}

// Courier widths (monospaced)
var courierWidths = map[rune]float64{}

// Symbol widths
var symbolWidths = map[rune]float64{}

// ZapfDingbats widths
var zapfDingbatsWidths = map[rune]float64{}

func init() {
	// Courier is monospaced - all characters have same width
	for r := rune(32); r <= 126; r++ {
		courierWidths[r] = 600
	}

	// Symbol and ZapfDingbats - use default width for now
	for r := rune(32); r <= 126; r++ {
		symbolWidths[r] = 500
		zapfDingbatsWidths[r] = 500
	}
}
