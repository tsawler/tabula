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
	Text      string
	X, Y      float64
	Width     float64
	Height    float64
	FontName  string
	FontSize  float64
	Direction Direction // Text direction (LTR, RTL, Neutral)
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
	fontSize := e.gs.GetEffectiveFontSize() // Use effective size (accounts for text matrix)
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

	// Detect text direction based on Unicode properties
	direction := DetectDirection(decodedText)

	fragment := TextFragment{
		Text:      decodedText,
		X:         x,
		Y:         y,
		Width:     width,
		Height:    fontSize,
		FontName:  fontName,
		FontSize:  fontSize,
		Direction: direction,
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

// GetText returns all extracted text concatenated with smart spacing and RTL support
func (e *Extractor) GetText() string {
	if len(e.fragments) == 0 {
		return ""
	}

	// Group fragments by lines (same Y coordinate within tolerance)
	lines := e.groupFragmentsByLine()

	var sb strings.Builder

	for lineIdx, line := range lines {
		// Determine line direction
		lineDir := e.detectLineDirection(line)

		// Reorder fragments in reading order based on direction
		orderedFrags := e.reorderFragmentsForReading(line, lineDir)

		// Calculate line metrics for smart spacing
		lineMetrics := e.calculateLineMetrics(orderedFrags, lineDir)

		// Assemble line text
		for i, frag := range orderedFrags {
			sb.WriteString(frag.Text)

			// Add space between fragments if needed
			if i < len(orderedFrags)-1 {
				nextFrag := orderedFrags[i+1]
				horizontalDist := calculateHorizontalDistance(frag, nextFrag, lineDir)

				if e.shouldInsertSpaceSmart(frag, nextFrag, horizontalDist, lineMetrics) {
					sb.WriteString(" ")
				}
			}
		}

		// Add line break between lines
		if lineIdx < len(lines)-1 {
			nextLine := lines[lineIdx+1]
			currentY := line[0].Y
			nextY := nextLine[0].Y
			verticalDist := abs(nextY - currentY)

			// Check if it's a paragraph break (extra vertical space)
			if verticalDist > line[0].Height*1.5 {
				sb.WriteString("\n\n")
			} else {
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

// lineMetrics holds computed metrics for a line to enable smart spacing decisions
type lineMetrics struct {
	isCharacterLevel       bool    // True if fragments are single/few characters each
	hasExplicitSpaces      bool    // True if line has explicit space character fragments
	avgFragmentLen         float64 // Average number of characters per fragment
	medianGap              float64 // Median gap between fragments (for character-level)
	avgGap                 float64 // Average gap between fragments
	maxNonSpaceGap         float64 // Maximum gap between non-space fragments
	typicalCharGap         float64 // Typical inter-character gap (25th percentile)
}

// calculateLineMetrics computes metrics for a line to enable smart spacing
func (e *Extractor) calculateLineMetrics(fragments []TextFragment, lineDir Direction) lineMetrics {
	metrics := lineMetrics{}

	if len(fragments) == 0 {
		return metrics
	}

	// Calculate average fragment length (in characters) and check for explicit spaces
	totalChars := 0
	for _, frag := range fragments {
		text := frag.Text
		totalChars += len([]rune(text))

		// Check if this fragment is or contains an explicit space
		if strings.TrimSpace(text) == "" || strings.Contains(text, " ") {
			metrics.hasExplicitSpaces = true
		}
	}
	metrics.avgFragmentLen = float64(totalChars) / float64(len(fragments))

	// Character-level if average fragment is <= 2 characters
	metrics.isCharacterLevel = metrics.avgFragmentLen <= 2.0

	// Calculate gaps between non-space fragments
	if len(fragments) > 1 {
		gaps := make([]float64, 0, len(fragments)-1)
		for i := 0; i < len(fragments)-1; i++ {
			// Skip gaps involving space-only fragments
			if strings.TrimSpace(fragments[i].Text) == "" || strings.TrimSpace(fragments[i+1].Text) == "" {
				continue
			}

			gap := calculateHorizontalDistance(fragments[i], fragments[i+1], lineDir)
			if gap > 0 { // Only consider positive gaps
				gaps = append(gaps, gap)
				if gap > metrics.maxNonSpaceGap {
					metrics.maxNonSpaceGap = gap
				}
			}
		}

		if len(gaps) > 0 {
			// Calculate average gap
			totalGap := 0.0
			for _, g := range gaps {
				totalGap += g
			}
			metrics.avgGap = totalGap / float64(len(gaps))

			// Sort gaps to find the typical inter-character gap
			sortedGaps := make([]float64, len(gaps))
			copy(sortedGaps, gaps)
			// Simple bubble sort for small arrays
			for i := 0; i < len(sortedGaps)-1; i++ {
				for j := i + 1; j < len(sortedGaps); j++ {
					if sortedGaps[i] > sortedGaps[j] {
						sortedGaps[i], sortedGaps[j] = sortedGaps[j], sortedGaps[i]
					}
				}
			}

			// Use 10th percentile for baseline inter-character gap
			p10Index := len(sortedGaps) / 10
			if p10Index < 0 {
				p10Index = 0
			}
			metrics.medianGap = sortedGaps[p10Index]

			// Use 25th percentile as typical character gap
			p25Index := len(sortedGaps) / 4
			if p25Index >= len(sortedGaps) {
				p25Index = len(sortedGaps) - 1
			}
			metrics.typicalCharGap = sortedGaps[p25Index]
		}
	}

	return metrics
}

// shouldInsertSpaceSmart determines if a space should be inserted between two fragments
// using line-level metrics to handle both word-level and character-level PDFs
func (e *Extractor) shouldInsertSpaceSmart(frag, nextFrag TextFragment, horizontalDist float64, metrics lineMetrics) bool {
	// If current fragment ends with whitespace or next fragment starts with whitespace,
	// don't insert additional space - the space is already in the text stream
	if len(frag.Text) > 0 && isWhitespace(frag.Text[len(frag.Text)-1]) {
		return false
	}
	if len(nextFrag.Text) > 0 && isWhitespace(nextFrag.Text[0]) {
		return false
	}

	// If fragments overlap or are very close, no space
	if horizontalDist < 0 || horizontalDist < frag.FontSize*0.05 {
		return false
	}

	// For character-level PDFs with explicit space fragments, be very conservative
	// Only add spaces if this gap is MUCH larger than typical gaps
	if metrics.isCharacterLevel && metrics.hasExplicitSpaces {
		// When the PDF has explicit space fragments, trust those for word boundaries
		// Only add a space if the gap is more than 5x the typical character gap
		// This handles edge cases where there's truly missing spacing
		if metrics.typicalCharGap > 0 {
			threshold := metrics.typicalCharGap * 5.0
			return horizontalDist >= threshold
		}
		// If we can't compute typical gap, don't add spaces - trust explicit ones
		return false
	}

	// For character-level PDFs without explicit spaces, use adaptive detection
	if metrics.isCharacterLevel {
		// Use the larger of:
		// 1. 3x the typical (10th percentile) inter-character gap
		// 2. 80% of font size (conservative fallback)
		threshold := frag.FontSize * 0.8

		// If we have gap data, use gap-based threshold if it's higher
		if metrics.medianGap > 0 {
			gapThreshold := metrics.medianGap * 3.0
			if gapThreshold > threshold {
				threshold = gapThreshold
			}
		}

		return horizontalDist >= threshold
	}

	// For word-level PDFs, use font space width
	spaceWidth := e.getSpaceWidth(frag.FontName, frag.FontSize)

	// Insert space if gap is >= 50% of a space character width
	threshold := spaceWidth * 0.5

	return horizontalDist >= threshold
}

// isWhitespace checks if a byte is a whitespace character
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// groupFragmentsByLine groups fragments into lines based on Y coordinate
func (e *Extractor) groupFragmentsByLine() [][]TextFragment {
	if len(e.fragments) == 0 {
		return nil
	}

	lines := make([][]TextFragment, 0)
	currentLine := []TextFragment{e.fragments[0]}

	for i := 1; i < len(e.fragments); i++ {
		frag := e.fragments[i]
		prevFrag := e.fragments[i-1]

		// Check if this fragment is on the same line (Y within tolerance)
		verticalDist := abs(frag.Y - prevFrag.Y)
		if verticalDist <= prevFrag.Height*0.5 {
			// Same line
			currentLine = append(currentLine, frag)
		} else {
			// New line - save current line and start new one
			lines = append(lines, currentLine)
			currentLine = []TextFragment{frag}
		}
	}

	// Don't forget the last line
	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	return lines
}

// detectLineDirection determines the dominant direction of a line
func (e *Extractor) detectLineDirection(fragments []TextFragment) Direction {
	ltrCount := 0
	rtlCount := 0

	for _, frag := range fragments {
		switch frag.Direction {
		case LTR:
			ltrCount++
		case RTL:
			rtlCount++
		}
	}

	// If no strong directional fragments, default to LTR
	if ltrCount == 0 && rtlCount == 0 {
		return LTR
	}

	// Return dominant direction
	if rtlCount > ltrCount {
		return RTL
	}
	return LTR
}

// reorderFragmentsForReading reorders fragments based on reading direction
func (e *Extractor) reorderFragmentsForReading(fragments []TextFragment, lineDir Direction) []TextFragment {
	if len(fragments) <= 1 {
		return fragments
	}

	// Make a copy to avoid modifying original
	ordered := make([]TextFragment, len(fragments))
	copy(ordered, fragments)

	// Sort by X coordinate
	// For LTR: left to right (ascending X)
	// For RTL: right to left (descending X)
	for i := 0; i < len(ordered)-1; i++ {
		for j := i + 1; j < len(ordered); j++ {
			shouldSwap := false
			if lineDir == RTL {
				// RTL: higher X comes first
				shouldSwap = ordered[i].X < ordered[j].X
			} else {
				// LTR: lower X comes first
				shouldSwap = ordered[i].X > ordered[j].X
			}

			if shouldSwap {
				ordered[i], ordered[j] = ordered[j], ordered[i]
			}
		}
	}

	return ordered
}

// calculateHorizontalDistance calculates the gap between two fragments
// accounting for text direction
func calculateHorizontalDistance(frag, nextFrag TextFragment, lineDir Direction) float64 {
	if lineDir == RTL {
		// For RTL, distance is from end of next fragment to start of current
		// (reading right-to-left)
		return frag.X - (nextFrag.X + nextFrag.Width)
	}
	// For LTR, distance is from end of current to start of next
	return nextFrag.X - (frag.X + frag.Width)
}

// shouldInsertSpace determines if a space should be inserted between two fragments
// based on the horizontal gap and font metrics
func (e *Extractor) shouldInsertSpace(frag, nextFrag TextFragment, horizontalDist float64) bool {
	// If fragments overlap or are very close, no space
	if horizontalDist < 0 || horizontalDist < frag.FontSize*0.05 {
		return false
	}

	// Get the expected space width from font metrics
	spaceWidth := e.getSpaceWidth(frag.FontName, frag.FontSize)

	// Insert space if gap is >= 50% of a space character width
	// This threshold accounts for kerning and tight spacing while still detecting word boundaries
	threshold := spaceWidth * 0.5

	// DEBUG: Uncomment to see spacing decisions
	// fmt.Printf("DEBUG: '%s' -> '%s': gap=%.2f, spaceWidth=%.2f, threshold=%.2f, insert=%v\n",
	//     frag.Text, nextFrag.Text, horizontalDist, spaceWidth, threshold, horizontalDist >= threshold)

	return horizontalDist >= threshold
}

// getSpaceWidth returns the expected width of a space character for the given font and size
func (e *Extractor) getSpaceWidth(fontName string, fontSize float64) float64 {
	// Get font from registered fonts
	if f, ok := e.fonts[fontName]; ok {
		// Get space character width (character code 0x20)
		spaceCharWidth := f.GetWidth(' ') // Width in 1000ths of em

		// Convert from font units to actual width
		// Font width is in 1000ths of em, fontSize is in points
		actualWidth := (spaceCharWidth * fontSize) / 1000.0

		return actualWidth
	}

	// Fallback: estimate space width as 25% of font size
	// This is a reasonable default for most proportional fonts
	return fontSize * 0.25
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

// GetFonts returns the fonts registered in this extractor
// Useful for debugging font loading and ToUnicode CMap issues
func (e *Extractor) GetFonts() map[string]*font.Font {
	return e.fonts
}
