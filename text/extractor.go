package text

import (
	"fmt"
	"strings"

	"github.com/tsawler/tabula/contentstream"
	"github.com/tsawler/tabula/core"
	"github.com/tsawler/tabula/font"
	"github.com/tsawler/tabula/graphicsstate"
	"github.com/tsawler/tabula/model"
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
				e.showText(string(str))
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
				e.showText(string(str))
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
				e.showText(string(str))
			}
		}
	}

	return nil
}

// showText processes text showing operation
func (e *Extractor) showText(text string) {
	x, y := e.gs.GetTextPosition()
	fontSize := e.gs.GetFontSize()
	fontName := e.gs.GetFontName()

	// Calculate text width
	width := 0.0
	if f, ok := e.fonts[fontName]; ok {
		width = f.GetStringWidth(text) * fontSize / 1000.0
	} else {
		// Estimate width if font not available
		width = float64(len(text)) * fontSize * 0.5
	}

	fragment := TextFragment{
		Text:     text,
		X:        x,
		Y:        y,
		Width:    width,
		Height:   fontSize,
		FontName: fontName,
		FontSize: fontSize,
	}

	e.fragments = append(e.fragments, fragment)

	// Update text position
	e.gs.ShowText(text)
}

// showTextArray processes text array showing operation
func (e *Extractor) showTextArray(arr core.Array) {
	for _, item := range arr {
		switch v := item.(type) {
		case core.String:
			e.showText(string(v))
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

// GetText returns all extracted text concatenated
func (e *Extractor) GetText() string {
	var sb strings.Builder
	for i, frag := range e.fragments {
		sb.WriteString(frag.Text)
		// Add space between fragments if they're not adjacent
		if i < len(e.fragments)-1 {
			sb.WriteString(" ")
		}
	}
	return sb.String()
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
