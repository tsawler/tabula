package text

import (
	"fmt"
	"strings"

	"github.com/tsawler/tabula/contentstream"
	"github.com/tsawler/tabula/core"
	"github.com/tsawler/tabula/font"
	"github.com/tsawler/tabula/graphicsstate"
	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/pages"
)

// TextFragment represents a piece of extracted text with position
type TextFragment struct {
	Text     string
	X, Y     float64
	Width    float64
	Height   float64
	FontName string
	FontSize float64
}

// Extractor extracts text from content streams
type Extractor struct {
	gs    *graphicsstate.GraphicsState
	fonts map[string]*font.Font

	fragments []TextFragment
}

// NewExtractor creates a new text extractor
func NewExtractor() *Extractor {
	return &Extractor{
		gs:        graphicsstate.NewGraphicsState(),
		fonts:     make(map[string]*font.Font),
		fragments: make([]TextFragment, 0),
	}
}

// RegisterFont registers a font for use during extraction
func (e *Extractor) RegisterFont(name, baseFont, subtype string) {
	e.fonts[name] = font.NewFont(name, baseFont, subtype)
}

// RegisterParsedFont registers a pre-parsed font for use during extraction
// This is useful when you have already parsed the font with its ToUnicode CMap
func (e *Extractor) RegisterParsedFont(name string, f *font.Font) {
	e.fonts[name] = f
}

// RegisterFontsFromPage parses and registers all fonts from a page's resources
// This is the recommended way to prepare the extractor for text extraction from a page
func (e *Extractor) RegisterFontsFromPage(page *pages.Page, resolver func(core.IndirectRef) (core.Object, error)) error {
	// Get page resources
	resources, err := page.Resources()
	if err != nil || resources == nil {
		return nil // Page has no resources
	}

	return e.RegisterFontsFromResources(resources, resolver)
}

// RegisterFontsFromResources parses and registers all fonts from a resources dictionary
// This is useful when working with page resources directly
func (e *Extractor) RegisterFontsFromResources(resources core.Dict, resolver func(core.IndirectRef) (core.Object, error)) error {
	// Get Font dictionary
	fontDictObj := resources.Get("Font")
	if fontDictObj == nil {
		return nil // No fonts in resources
	}

	// Resolve font dictionary if it's a reference
	fontDictResolved, err := resolveIfRef(fontDictObj, resolver)
	if err != nil {
		return fmt.Errorf("failed to resolve font dictionary: %w", err)
	}

	fonts, ok := fontDictResolved.(core.Dict)
	if !ok {
		return nil // Font object is not a dictionary
	}

	// Parse and register each font
	for name, fontObj := range fonts {
		// Resolve font object
		fontResolved, err := resolveIfRef(fontObj, resolver)
		if err != nil {
			continue // Skip fonts that can't be resolved
		}

		fontDict, ok := fontResolved.(core.Dict)
		if !ok {
			continue
		}

		// Get font subtype
		subtype := fontDict.Get("Subtype")
		if subtype == nil {
			continue
		}

		subtypeName, ok := subtype.(core.Name)
		if !ok {
			continue
		}

		// Parse font based on type
		var parsedFont *font.Font

		switch string(subtypeName) {
		case "Type1":
			if t1Font, err := font.NewType1Font(fontDict, resolver); err == nil {
				parsedFont = t1Font.Font
			}
		case "TrueType":
			if ttFont, err := font.NewTrueTypeFont(fontDict, resolver); err == nil {
				parsedFont = ttFont.Font
			}
		case "Type0":
			if t0Font, err := font.NewType0Font(fontDict, resolver); err == nil {
				parsedFont = t0Font.Font
			}
		}

		// Register parsed font
		if parsedFont != nil {
			e.RegisterParsedFont(name, parsedFont)
			// Also register with "/" prefix to handle both naming conventions
			if !strings.HasPrefix(name, "/") {
				e.RegisterParsedFont("/"+name, parsedFont)
			}
		}
	}

	return nil
}

// resolveIfRef resolves an object if it's an indirect reference, otherwise returns it as-is
func resolveIfRef(obj core.Object, resolver func(core.IndirectRef) (core.Object, error)) (core.Object, error) {
	if ref, ok := obj.(core.IndirectRef); ok {
		return resolver(ref)
	}
	return obj, nil
}

// Extract extracts text from content stream operations
func (e *Extractor) Extract(operations []contentstream.Operation) ([]TextFragment, error) {
	e.fragments = make([]TextFragment, 0)

	for i, op := range operations {
		if err := e.processOperation(op); err != nil {
			return nil, fmt.Errorf("operation %d (%s): %w", i, op.Operator, err)
		}
	}

	return e.fragments, nil
}

// ExtractFromBytes parses and extracts text from raw content stream data
func (e *Extractor) ExtractFromBytes(data []byte) ([]TextFragment, error) {
	parser := contentstream.NewParser(data)
	operations, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse content stream: %w", err)
	}

	return e.Extract(operations)
}

// processOperation processes a single content stream operation
func (e *Extractor) processOperation(op contentstream.Operation) error {
	switch op.Operator {
	// Graphics state
	case "q":
		e.gs.Save()
	case "Q":
		return e.gs.Restore()
	case "cm":
		if len(op.Operands) == 6 {
			m := operandsToMatrix(op.Operands)
			e.gs.Transform(m)
		}
	case "w":
		if len(op.Operands) == 1 {
			if w, ok := toFloat(op.Operands[0]); ok {
				e.gs.SetLineWidth(w)
			}
		}
	case "RG":
		if len(op.Operands) == 3 {
			r, _ := toFloat(op.Operands[0])
			g, _ := toFloat(op.Operands[1])
			b, _ := toFloat(op.Operands[2])
			e.gs.SetStrokeColorRGB(r, g, b)
		}
	case "rg":
		if len(op.Operands) == 3 {
			r, _ := toFloat(op.Operands[0])
			g, _ := toFloat(op.Operands[1])
			b, _ := toFloat(op.Operands[2])
			e.gs.SetFillColorRGB(r, g, b)
		}

	// Text state
	case "BT":
		e.gs.BeginText()
	case "ET":
		e.gs.EndText()
	case "Tf":
		if len(op.Operands) == 2 {
			if name, ok := op.Operands[0].(core.Name); ok {
				if size, ok := toFloat(op.Operands[1]); ok {
					fontName := string(name)
					if !strings.HasPrefix(fontName, "/") {
						fontName = "/" + fontName
					}
					e.gs.SetFont(fontName, size)

					// Auto-register font if not already registered
					if _, exists := e.fonts[fontName]; !exists {
						// Default to Helvetica for unknown fonts
						e.RegisterFont(fontName, "Helvetica", "Type1")
					}
				}
			}
		}
	case "Tc":
		if len(op.Operands) == 1 {
			if spacing, ok := toFloat(op.Operands[0]); ok {
				e.gs.SetCharSpacing(spacing)
			}
		}
	case "Tw":
		if len(op.Operands) == 1 {
			if spacing, ok := toFloat(op.Operands[0]); ok {
				e.gs.SetWordSpacing(spacing)
			}
		}
	case "Tz":
		if len(op.Operands) == 1 {
			if scale, ok := toFloat(op.Operands[0]); ok {
				e.gs.SetHorizontalScaling(scale)
			}
		}
	case "TL":
		if len(op.Operands) == 1 {
			if leading, ok := toFloat(op.Operands[0]); ok {
				e.gs.SetLeading(leading)
			}
		}
	case "Tr":
		if len(op.Operands) == 1 {
			if mode, ok := toInt(op.Operands[0]); ok {
				e.gs.SetRenderingMode(mode)
			}
		}
	case "Ts":
		if len(op.Operands) == 1 {
			if rise, ok := toFloat(op.Operands[0]); ok {
				e.gs.SetTextRise(rise)
			}
		}

	// Text positioning
	case "Tm":
		if len(op.Operands) == 6 {
			m := operandsToMatrix(op.Operands)
			e.gs.SetTextMatrix(m)
		}
	case "Td":
		if len(op.Operands) == 2 {
			tx, _ := toFloat(op.Operands[0])
			ty, _ := toFloat(op.Operands[1])
			e.gs.TranslateText(tx, ty)
		}
	case "TD":
		if len(op.Operands) == 2 {
			tx, _ := toFloat(op.Operands[0])
			ty, _ := toFloat(op.Operands[1])
			e.gs.TranslateTextSetLeading(tx, ty)
		}
	case "T*":
		e.gs.NextLine()

	// Text showing
	case "Tj":
		if len(op.Operands) == 1 {
			if str, ok := op.Operands[0].(core.String); ok {
				e.showText([]byte(str))
			}
		}
	case "TJ":
		if len(op.Operands) == 1 {
			if arr, ok := op.Operands[0].(core.Array); ok {
				e.showTextArray(arr)
			}
		}
	case "'":
		// Move to next line and show text
		e.gs.NextLine()
		if len(op.Operands) == 1 {
			if str, ok := op.Operands[0].(core.String); ok {
				e.showText([]byte(str))
			}
		}
	case "\"":
		// Set word/char spacing, move to next line, show text
		if len(op.Operands) == 3 {
			if wordSpacing, ok := toFloat(op.Operands[0]); ok {
				e.gs.SetWordSpacing(wordSpacing)
			}
			if charSpacing, ok := toFloat(op.Operands[1]); ok {
				e.gs.SetCharSpacing(charSpacing)
			}
			e.gs.NextLine()
			if str, ok := op.Operands[2].(core.String); ok {
				e.showText([]byte(str))
			}
		}
	}

	return nil
}

// showText processes text showing operation
func (e *Extractor) showText(data []byte) {
	x, y := e.gs.GetTextPosition()
	fontSize := e.gs.GetFontSize()
	fontName := e.gs.GetFontName()

	// Decode text using font's ToUnicode CMap if available
	var decodedText string
	if f, ok := e.fonts[fontName]; ok {
		decodedText = f.DecodeString(data)
	} else {
		// No font registered - use raw bytes as string (fallback)
		decodedText = string(data)
	}

	// Calculate text width
	width := 0.0
	if f, ok := e.fonts[fontName]; ok {
		width = f.GetStringWidth(decodedText) * fontSize / 1000.0
	} else {
		// Estimate width if font not available
		width = float64(len(decodedText)) * fontSize * 0.5
	}

	fragment := TextFragment{
		Text:     decodedText,
		X:        x,
		Y:        y,
		Width:    width,
		Height:   fontSize,
		FontName: fontName,
		FontSize: fontSize,
	}

	e.fragments = append(e.fragments, fragment)

	// Update text position (use original byte length)
	e.gs.ShowText(string(data))
}

// showTextArray processes text array showing operation
func (e *Extractor) showTextArray(arr core.Array) {
	for _, item := range arr {
		switch v := item.(type) {
		case core.String:
			e.showText([]byte(v))
		case core.Int:
			// Position adjustment
			adjustment := -float64(v) * e.gs.GetFontSize() / 1000.0
			// Update text matrix
			tm := e.gs.GetTextMatrix()
			tm[4] += adjustment
		case core.Real:
			adjustment := -float64(v) * e.gs.GetFontSize() / 1000.0
			tm := e.gs.GetTextMatrix()
			tm[4] += adjustment
		}
	}
}

// GetText returns all extracted text concatenated with smart spacing
func (e *Extractor) GetText() string {
	if len(e.fragments) == 0 {
		return ""
	}

	var sb strings.Builder

	for i, frag := range e.fragments {
		sb.WriteString(frag.Text)

		// Determine spacing to next fragment
		if i < len(e.fragments)-1 {
			nextFrag := e.fragments[i+1]

			// Calculate vertical distance
			verticalDist := abs(nextFrag.Y - frag.Y)

			// Calculate horizontal distance
			horizontalDist := nextFrag.X - (frag.X + frag.Width)

			// Detect line break (significant vertical movement)
			if verticalDist > frag.Height*0.5 {
				// Check if it's a paragraph break (extra vertical space)
				if verticalDist > frag.Height*1.5 {
					sb.WriteString("\n\n")
				} else {
					sb.WriteString("\n")
				}
			} else if horizontalDist > frag.Height*0.2 {
				// Horizontal gap suggests a space
				sb.WriteString(" ")
			}
			// If fragments are adjacent, don't add anything
		}
	}

	return sb.String()
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetFragments returns all text fragments
func (e *Extractor) GetFragments() []TextFragment {
	return e.fragments
}

// Helper functions

func toFloat(obj core.Object) (float64, bool) {
	switch v := obj.(type) {
	case core.Int:
		return float64(v), true
	case core.Real:
		return float64(v), true
	default:
		return 0, false
	}
}

func toInt(obj core.Object) (int, bool) {
	if i, ok := obj.(core.Int); ok {
		return int(i), true
	}
	return 0, false
}

func operandsToMatrix(operands []core.Object) model.Matrix {
	if len(operands) != 6 {
		return model.Identity()
	}

	vals := make([]float64, 6)
	for i, op := range operands {
		vals[i], _ = toFloat(op)
	}

	return model.Matrix(vals)
}
