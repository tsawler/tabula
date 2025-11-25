package graphicsstate

import (
	"fmt"

	"github.com/tsawler/tabula/model"
)

// GraphicsState represents the PDF graphics state
type GraphicsState struct {
	// Current Transformation Matrix
	CTM model.Matrix

	// Text state
	Text TextState

	// Graphics state stack (for q/Q operators)
	stack []*GraphicsState

	// Line attributes
	LineWidth float64

	// Color (simplified - just RGB for now)
	StrokeColor [3]float64
	FillColor   [3]float64
}

// TextState represents text-specific state
type TextState struct {
	// Font and size
	FontName string
	FontSize float64

	// Character and word spacing
	CharSpacing float64
	WordSpacing float64

	// Horizontal scaling (percentage)
	HorizontalScaling float64

	// Leading (line spacing)
	Leading float64

	// Text rendering mode
	RenderingMode int

	// Text rise
	Rise float64

	// Text matrices
	TextMatrix     model.Matrix
	TextLineMatrix model.Matrix
}

// NewGraphicsState creates a new graphics state with default values
func NewGraphicsState() *GraphicsState {
	return &GraphicsState{
		CTM:         model.Identity(),
		LineWidth:   1.0,
		StrokeColor: [3]float64{0, 0, 0}, // Black
		FillColor:   [3]float64{0, 0, 0}, // Black
		Text: TextState{
			FontSize:          12.0,
			HorizontalScaling: 100.0,
			TextMatrix:        model.Identity(),
			TextLineMatrix:    model.Identity(),
		},
	}
}

// Clone creates a deep copy of the graphics state
func (gs *GraphicsState) Clone() *GraphicsState {
	clone := &GraphicsState{
		CTM:         gs.CTM,
		LineWidth:   gs.LineWidth,
		StrokeColor: gs.StrokeColor,
		FillColor:   gs.FillColor,
		Text:        gs.Text,
	}
	return clone
}

// Save pushes the current graphics state onto the stack (q operator)
func (gs *GraphicsState) Save() {
	gs.stack = append(gs.stack, gs.Clone())
}

// Restore pops a graphics state from the stack (Q operator)
func (gs *GraphicsState) Restore() error {
	if len(gs.stack) == 0 {
		return fmt.Errorf("graphics state stack underflow")
	}

	// Pop from stack
	saved := gs.stack[len(gs.stack)-1]
	gs.stack = gs.stack[:len(gs.stack)-1]

	// Restore state
	gs.CTM = saved.CTM
	gs.LineWidth = saved.LineWidth
	gs.StrokeColor = saved.StrokeColor
	gs.FillColor = saved.FillColor
	gs.Text = saved.Text

	return nil
}

// Transform applies a transformation matrix to CTM (cm operator)
func (gs *GraphicsState) Transform(m model.Matrix) {
	gs.CTM = gs.CTM.Multiply(m)
}

// SetLineWidth sets the line width (w operator)
func (gs *GraphicsState) SetLineWidth(width float64) {
	gs.LineWidth = width
}

// SetStrokeColorRGB sets the stroke color (RG operator)
func (gs *GraphicsState) SetStrokeColorRGB(r, g, b float64) {
	gs.StrokeColor = [3]float64{r, g, b}
}

// SetFillColorRGB sets the fill color (rg operator)
func (gs *GraphicsState) SetFillColorRGB(r, g, b float64) {
	gs.FillColor = [3]float64{r, g, b}
}

// SetFont sets the current font (Tf operator)
func (gs *GraphicsState) SetFont(name string, size float64) {
	gs.Text.FontName = name
	gs.Text.FontSize = size
}

// SetCharSpacing sets character spacing (Tc operator)
func (gs *GraphicsState) SetCharSpacing(spacing float64) {
	gs.Text.CharSpacing = spacing
}

// SetWordSpacing sets word spacing (Tw operator)
func (gs *GraphicsState) SetWordSpacing(spacing float64) {
	gs.Text.WordSpacing = spacing
}

// SetHorizontalScaling sets horizontal scaling (Tz operator)
func (gs *GraphicsState) SetHorizontalScaling(scale float64) {
	gs.Text.HorizontalScaling = scale
}

// SetLeading sets text leading (TL operator)
func (gs *GraphicsState) SetLeading(leading float64) {
	gs.Text.Leading = leading
}

// SetRenderingMode sets text rendering mode (Tr operator)
func (gs *GraphicsState) SetRenderingMode(mode int) {
	gs.Text.RenderingMode = mode
}

// SetTextRise sets text rise (Ts operator)
func (gs *GraphicsState) SetTextRise(rise float64) {
	gs.Text.Rise = rise
}

// BeginText initializes text state (BT operator)
func (gs *GraphicsState) BeginText() {
	gs.Text.TextMatrix = model.Identity()
	gs.Text.TextLineMatrix = model.Identity()
}

// EndText does nothing for now (ET operator)
func (gs *GraphicsState) EndText() {
	// No-op in basic implementation
}

// SetTextMatrix sets the text matrix (Tm operator)
func (gs *GraphicsState) SetTextMatrix(m model.Matrix) {
	gs.Text.TextMatrix = m
	gs.Text.TextLineMatrix = m
}

// TranslateText translates the text matrix (Td operator)
func (gs *GraphicsState) TranslateText(tx, ty float64) {
	// Td is equivalent to: Tm = Tlm * T(tx, ty)
	translation := model.Translate(tx, ty)
	gs.Text.TextLineMatrix = gs.Text.TextLineMatrix.Multiply(translation)
	gs.Text.TextMatrix = gs.Text.TextLineMatrix
}

// TranslateTextSetLeading translates text and sets leading (TD operator)
func (gs *GraphicsState) TranslateTextSetLeading(tx, ty float64) {
	gs.SetLeading(-ty)
	gs.TranslateText(tx, ty)
}

// NextLine moves to next line (T* operator)
func (gs *GraphicsState) NextLine() {
	gs.TranslateText(0, -gs.Text.Leading)
}

// ShowText updates position after showing text (Tj operator)
// Returns the displacement caused by the text
func (gs *GraphicsState) ShowText(text string) (dx, dy float64) {
	// Calculate text displacement
	// Simplified: assumes each character advances by fontSize * horizontalScaling / 100
	// Real implementation would need font metrics

	numChars := float64(len(text))
	numSpaces := float64(0)
	for _, c := range text {
		if c == ' ' {
			numSpaces++
		}
	}

	// Calculate total displacement
	charAdvance := gs.Text.FontSize * gs.Text.HorizontalScaling / 100.0
	totalAdvance := numChars * charAdvance
	totalAdvance += numSpaces * gs.Text.WordSpacing
	totalAdvance += numChars * gs.Text.CharSpacing

	// Update text matrix (E component = tx)
	gs.Text.TextMatrix[4] += totalAdvance

	return totalAdvance, 0
}

// ShowTextArray shows text with positioning adjustments (TJ operator)
// Returns the displacement caused by the text
func (gs *GraphicsState) ShowTextArray(array []interface{}) (dx, dy float64) {
	totalDisplacement := 0.0

	for _, item := range array {
		switch v := item.(type) {
		case string:
			// Show text
			adv, _ := gs.ShowText(v)
			totalDisplacement += adv

		case int:
			// Position adjustment (in thousandths of em)
			adjustment := -float64(v) * gs.Text.FontSize / 1000.0
			gs.Text.TextMatrix[4] += adjustment
			totalDisplacement += adjustment

		case int64:
			adjustment := -float64(v) * gs.Text.FontSize / 1000.0
			gs.Text.TextMatrix[4] += adjustment
			totalDisplacement += adjustment

		case float64:
			adjustment := -v * gs.Text.FontSize / 1000.0
			gs.Text.TextMatrix[4] += adjustment
			totalDisplacement += adjustment
		}
	}

	return totalDisplacement, 0
}

// GetTextPosition returns the current text position in device space
func (gs *GraphicsState) GetTextPosition() (x, y float64) {
	// Transform text position by text matrix and CTM
	tm := gs.Text.TextMatrix
	x = tm[4] // E component
	y = tm[5] + gs.Text.Rise // F component + rise

	// Apply CTM
	p := gs.CTM.Transform(model.Point{X: x, Y: y})
	return p.X, p.Y
}

// GetTextMatrix returns the current text matrix
func (gs *GraphicsState) GetTextMatrix() model.Matrix {
	return gs.Text.TextMatrix
}

// GetFontSize returns the current font size
func (gs *GraphicsState) GetFontSize() float64 {
	return gs.Text.FontSize
}

// GetFontName returns the current font name
func (gs *GraphicsState) GetFontName() string {
	return gs.Text.FontName
}
