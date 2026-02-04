package text

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/tsawler/tabula/contentstream"
	"github.com/tsawler/tabula/core"
	"github.com/tsawler/tabula/font"
	"github.com/tsawler/tabula/graphicsstate"
	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/pages"
)

// xTolerance is the tolerance for X position comparison as a fraction of font size.
// Handles PDF generators (Word, Quartz) that place fragments in correct stream
// order but with slightly overlapping or disordered X coordinates.
const xTolerance = 0.25

// TextFragment represents a piece of extracted text with position and font information.
type TextFragment struct {
	Text      string    // Decoded text content
	X, Y      float64   // Position in page coordinates
	Width     float64   // Width of the text in page units
	Height    float64   // Height (typically font size)
	FontName  string    // Name of the font used
	FontSize  float64   // Font size in page units
	Direction Direction // Text direction (LTR, RTL, Neutral)
}

// Extractor extracts text fragments from PDF content streams.
// It maintains graphics state and registered fonts to properly decode and position text.
type Extractor struct {
	gs    *graphicsstate.GraphicsState // Graphics state tracker
	fonts map[string]*font.Font        // Registered fonts by name

	fragments []TextFragment // Extracted text fragments

	// XObject support
	resources      core.Dict                                      // Current resources dictionary
	resolver       func(core.IndirectRef) (core.Object, error)    // Reference resolver
	xobjectDepth   int                                            // Current XObject nesting depth
	maxXObjectDepth int                                           // Maximum nesting depth (prevents infinite recursion)
}

// NewExtractor creates a new text extractor with initialized graphics state.
func NewExtractor() *Extractor {
	return &Extractor{
		gs:              graphicsstate.NewGraphicsState(),
		fonts:           make(map[string]*font.Font),
		fragments:       make([]TextFragment, 0),
		maxXObjectDepth: 10, // Reasonable limit for nested XObjects
	}
}

// SetResourceContext configures the extractor with resources and a resolver
// for XObject processing. This enables extraction of text from Form XObjects.
func (e *Extractor) SetResourceContext(resources core.Dict, resolver func(core.IndirectRef) (core.Object, error)) {
	e.resources = resources
	e.resolver = resolver
}

// RegisterFont registers a font by name for use during extraction.
// The baseFont and subtype are used to create a basic font with default metrics.
func (e *Extractor) RegisterFont(name, baseFont, subtype string) {
	e.fonts[name] = font.NewFont(name, baseFont, subtype)
}

// RegisterParsedFont registers a pre-parsed font for use during extraction.
// Use this when you have already parsed the font with its ToUnicode CMap and widths.
func (e *Extractor) RegisterParsedFont(name string, f *font.Font) {
	e.fonts[name] = f
}

// RegisterFontsFromPage parses and registers all fonts from a page's resources.
// This is the recommended way to prepare the extractor before extracting text from a page.
func (e *Extractor) RegisterFontsFromPage(page *pages.Page, resolver func(core.IndirectRef) (core.Object, error)) error {
	// Get page resources
	resources, err := page.Resources()
	if err != nil || resources == nil {
		return nil // Page has no resources
	}

	return e.RegisterFontsFromResources(resources, resolver)
}

// RegisterFontsFromResources parses and registers all fonts from a resources dictionary.
// Use this when working with page resources directly rather than through a Page object.
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

// resolveIfRef resolves an indirect reference, or returns the object unchanged if not a reference.
func resolveIfRef(obj core.Object, resolver func(core.IndirectRef) (core.Object, error)) (core.Object, error) {
	if ref, ok := obj.(core.IndirectRef); ok {
		return resolver(ref)
	}
	return obj, nil
}

// Extract extracts text fragments from parsed content stream operations.
func (e *Extractor) Extract(operations []contentstream.Operation) ([]TextFragment, error) {
	e.fragments = make([]TextFragment, 0)

	for i, op := range operations {
		if err := e.processOperation(op); err != nil {
			return nil, fmt.Errorf("operation %d (%s): %w", i, op.Operator, err)
		}
	}

	// Return deduplicated fragments to handle PDFs with multiple content layers
	return e.deduplicateFragments(), nil
}

// ExtractFromBytes parses raw content stream data and extracts text fragments.
func (e *Extractor) ExtractFromBytes(data []byte) ([]TextFragment, error) {
	parser := contentstream.NewParser(data)
	operations, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse content stream: %w", err)
	}

	return e.Extract(operations)
}

// processOperation processes a single content stream operation, updating graphics
// state and extracting text as appropriate.
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

	// XObject invocation
	case "Do":
		if len(op.Operands) == 1 {
			if name, ok := op.Operands[0].(core.Name); ok {
				if err := e.invokeXObject(string(name)); err != nil {
					// Log but don't fail - XObject errors shouldn't stop extraction
					// The error is silently ignored as the PDF may still have
					// extractable text in other parts
				}
			}
		}
	}

	return nil
}

// invokeXObject handles the Do operator by processing Form XObjects.
// It extracts text from nested Form XObjects, handling their own resources and graphics state.
func (e *Extractor) invokeXObject(name string) error {
	// Check if we have resources and resolver configured
	if e.resources == nil || e.resolver == nil {
		return nil // No XObject support configured
	}

	// Check recursion depth
	if e.xobjectDepth >= e.maxXObjectDepth {
		return fmt.Errorf("XObject nesting too deep (max %d)", e.maxXObjectDepth)
	}

	// Get XObject dictionary from resources
	xobjectDictObj := e.resources.Get("XObject")
	if xobjectDictObj == nil {
		return nil // No XObjects in resources
	}

	// Resolve XObject dictionary if it's a reference
	xobjectDictResolved, err := resolveIfRef(xobjectDictObj, e.resolver)
	if err != nil {
		return fmt.Errorf("failed to resolve XObject dictionary: %w", err)
	}

	xobjectDict, ok := xobjectDictResolved.(core.Dict)
	if !ok {
		return nil // XObject is not a dictionary
	}

	// Look up the specific XObject by name
	xobjRef := xobjectDict.Get(name)
	if xobjRef == nil {
		// Try without leading slash
		xobjRef = xobjectDict.Get(strings.TrimPrefix(name, "/"))
	}
	if xobjRef == nil {
		return nil // XObject not found
	}

	// Resolve the XObject
	xobjResolved, err := resolveIfRef(xobjRef, e.resolver)
	if err != nil {
		return fmt.Errorf("failed to resolve XObject %s: %w", name, err)
	}

	// Must be a stream (Form XObjects are streams)
	xobjStream, ok := xobjResolved.(*core.Stream)
	if !ok {
		return nil // Not a stream, might be an image XObject
	}

	// Check if it's a Form XObject
	subtype := xobjStream.Dict.Get("Subtype")
	if subtype == nil {
		return nil
	}
	subtypeName, ok := subtype.(core.Name)
	if !ok || string(subtypeName) != "Form" {
		return nil // Not a Form XObject (might be Image)
	}

	// Decode the XObject content stream
	data, err := xobjStream.Decode()
	if err != nil {
		return fmt.Errorf("failed to decode XObject stream: %w", err)
	}

	if len(data) == 0 {
		return nil // Empty content
	}

	// Get XObject's own resources (if any)
	var xobjResources core.Dict
	if resObj := xobjStream.Dict.Get("Resources"); resObj != nil {
		resResolved, err := resolveIfRef(resObj, e.resolver)
		if err == nil {
			if resDict, ok := resResolved.(core.Dict); ok {
				xobjResources = resDict
			}
		}
	}

	// Register fonts from XObject's resources
	if xobjResources != nil {
		if err := e.RegisterFontsFromResources(xobjResources, e.resolver); err != nil {
			// Non-fatal - continue with existing fonts
		}
	}

	// Save current state
	e.gs.Save()
	e.xobjectDepth++

	// Save current resources and set XObject's resources (or merged)
	oldResources := e.resources
	if xobjResources != nil {
		// Use XObject's resources, but keep parent resources as fallback
		// by merging them (XObject resources take precedence)
		e.resources = e.mergeResources(oldResources, xobjResources)
	}

	// Apply XObject's transformation matrix if present
	if matrixObj := xobjStream.Dict.Get("Matrix"); matrixObj != nil {
		if matrixArr, ok := matrixObj.(core.Array); ok && len(matrixArr) == 6 {
			matrix := operandsToMatrix([]core.Object(matrixArr))
			e.gs.Transform(matrix)
		}
	}

	// Parse and process the XObject content stream
	parser := contentstream.NewParser(data)
	operations, err := parser.Parse()
	if err != nil {
		// Restore state and return error
		e.resources = oldResources
		e.xobjectDepth--
		e.gs.Restore()
		return fmt.Errorf("failed to parse XObject content: %w", err)
	}

	// Process operations
	for _, op := range operations {
		if err := e.processOperation(op); err != nil {
			// Continue processing despite errors
		}
	}

	// Restore state
	e.resources = oldResources
	e.xobjectDepth--
	e.gs.Restore()

	return nil
}

// mergeResources creates a merged resources dictionary where child resources
// take precedence over parent resources. This is used for XObject processing.
func (e *Extractor) mergeResources(parent, child core.Dict) core.Dict {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}

	// Create a new dict with parent entries
	merged := make(core.Dict)
	for k, v := range parent {
		merged[k] = v
	}

	// Override/add with child entries
	for k, v := range child {
		// For sub-dictionaries like Font, XObject, etc., we should merge them
		if parentSub, ok := parent[k].(core.Dict); ok {
			if childSub, ok := v.(core.Dict); ok {
				// Merge sub-dictionaries
				mergedSub := make(core.Dict)
				for sk, sv := range parentSub {
					mergedSub[sk] = sv
				}
				for sk, sv := range childSub {
					mergedSub[sk] = sv
				}
				merged[k] = mergedSub
				continue
			}
		}
		merged[k] = v
	}

	return merged
}

// showText processes a text showing operation (Tj), decoding and positioning the text.
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

	// Apply CTM scaling to font size to get device-space size
	// This ensures that line grouping logic works correctly even if the page is scaled
	ctm := e.gs.CTM
	// Calculate Y scaling factor from CTM (magnitude of vertical vector)
	// CTM is [a b c d e f]
	// Vertical vector (0, 1) transforms to (c, d)
	// So vertical scaling factor is sqrt(c^2 + d^2)
	ctmScale := math.Sqrt(ctm[2]*ctm[2] + ctm[3]*ctm[3])

	// If CTM is identity (or close to it), scale is 1
	if ctmScale == 0 {
		ctmScale = 1.0
	}

	deviceFontSize := fontSize * ctmScale

	fragment := TextFragment{
		Text:      decodedText,
		X:         x,
		Y:         y,
		Width:     width * ctmScale, // Width should also be scaled? X is already transformed.
		Height:    deviceFontSize,
		FontName:  fontName,
		FontSize:  deviceFontSize, // Use device font size for layout calculations
		Direction: direction,
	}

	e.fragments = append(e.fragments, fragment)

	// Update text position (use original byte length)
	// Use the calculated width to update the graphics state
	// Note: width is already scaled by font size, but we need to check if it includes horizontal scaling
	// The GetStringWidth returns width in 1000ths of em.
	// width = f.GetStringWidth(decodedText) * fontSize / 1000.0
	// Horizontal scaling is applied in ShowTextWithWidth if we pass the raw width?
	// No, ShowTextWithWidth expects the width in user space.
	// Our 'width' variable is: GetStringWidth * fontSize / 1000.0
	// We should apply horizontal scaling to it before passing, or let ShowTextWithWidth handle it?
	// ShowTextWithWidth adds Tc and Tw scaled by Th.
	// It assumes 'width' is the glyph width.
	// We should apply horizontal scaling to 'width' here because GetStringWidth doesn't know about Th.

	hScale := e.gs.Text.HorizontalScaling / 100.0
	scaledWidth := width * hScale

	e.gs.ShowTextWithWidth(string(data), scaledWidth)
}

// showTextArray processes a text array showing operation (TJ).
// Arrays contain strings interleaved with position adjustments.
func (e *Extractor) showTextArray(arr core.Array) {
	for _, item := range arr {
		switch v := item.(type) {
		case core.String:
			e.showText([]byte(v))
		case core.Int:
			// Position adjustment
			// The number is in thousands of a unit of text space
			// It is subtracted from the current x coordinate
			// adjustment = -v * fontSize / 1000
			// Also need to apply horizontal scaling
			hScale := e.gs.Text.HorizontalScaling / 100.0
			adjustment := -float64(v) * e.gs.GetFontSize() * hScale / 1000.0

			// Update text matrix
			tm := e.gs.GetTextMatrix()
			tm[4] += adjustment
			e.gs.SetTextMatrix(tm)
		case core.Real:
			hScale := e.gs.Text.HorizontalScaling / 100.0
			adjustment := -float64(v) * e.gs.GetFontSize() * hScale / 1000.0
			tm := e.gs.GetTextMatrix()
			tm[4] += adjustment
			e.gs.SetTextMatrix(tm)
		}
	}
}

// GetText returns all extracted text as a string with smart spacing.
// Handles both LTR and RTL text, grouping fragments into lines and adding
// appropriate word and line breaks. Duplicate fragments at the same position
// are automatically removed.
func (e *Extractor) GetText() string {
	if len(e.fragments) == 0 {
		return ""
	}

	// Deduplicate fragments first to handle PDFs with multiple content layers
	fragments := e.deduplicateFragments()

	// Group fragments by lines (same Y coordinate within tolerance)
	lines := groupFragments(fragments)

	// Sort lines by Y position (descending - top to bottom in page coordinates)
	// PDF coordinate system has origin at bottom-left, so higher Y = higher on page
	sort.SliceStable(lines, func(i, j int) bool {
		// Use the Y of the first fragment in each line
		return lines[i][0].Y > lines[j][0].Y
	})

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

// lineMetrics holds computed metrics for a line to enable smart spacing decisions.
// These metrics help distinguish word-level PDFs from character-level PDFs.
type lineMetrics struct {
	isCharacterLevel  bool    // True if fragments are single/few characters each
	hasExplicitSpaces bool    // True if line has explicit space character fragments
	avgFragmentLen    float64 // Average number of characters per fragment
	medianGap         float64 // 10th percentile gap (baseline inter-character gap)
	avgGap            float64 // Average gap between fragments
	maxNonSpaceGap    float64 // Maximum gap between non-space fragments
	typicalCharGap    float64 // Typical inter-character gap (25th percentile)
}

// calculateLineMetrics computes metrics for a line to enable smart spacing decisions.
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

// shouldInsertSpaceSmart determines if a space should be inserted between two fragments.
// It uses line-level metrics to handle both word-level and character-level PDFs correctly.
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

// isWhitespace reports whether b is a whitespace character.
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// groupFragmentsByLine groups fragments into lines based on Y coordinate proximity.
func (e *Extractor) groupFragmentsByLine() [][]TextFragment {
	return groupFragments(e.fragments)
}

// groupFragments groups text fragments into lines based on Y position and X continuity.
// Handles cases where wrapped text has similar Y coordinates but X jumps backwards,
// and detects multi-column layouts where content from different columns has similar Y.
func groupFragments(fragments []TextFragment) [][]TextFragment {
	if len(fragments) == 0 {
		return nil
	}

	lines := make([][]TextFragment, 0)
	currentLine := []TextFragment{fragments[0]}

	// Track the X extent of the current line to detect overlapping content
	lineMinX := fragments[0].X
	lineMaxX := fragments[0].X + fragments[0].Width

	for i := 1; i < len(fragments); i++ {
		frag := fragments[i]
		prevFrag := fragments[i-1]

		// Check vertical distance
		verticalDist := abs(frag.Y - prevFrag.Y)
		sameLineByY := verticalDist <= prevFrag.Height*0.5

		// Check for X position jump backwards from previous fragment
		prevEndX := prevFrag.X + prevFrag.Width
		xJumpBack := prevEndX - frag.X

		// Consider it a line wrap if X jumped backwards significantly
		isLineWrap := sameLineByY && xJumpBack > prevFrag.Height*1.5

		// Detect multi-column layouts where text from different columns
		// has similar Y positions but overlapping X ranges.
		// Key insight: text from a different column will have:
		// 1. Slightly different Y (even if within tolerance)
		// 2. X position that overlaps with existing line content
		fragEndX := frag.X + frag.Width

		// Check if Y differs significantly - this distinguishes column changes from
		// normal baseline variations (subscripts, superscripts, kerning adjustments).
		// Use 15% of font height as threshold to tolerate baseline variations while
		// still detecting true column overlaps.
		yDiffThreshold := prevFrag.Height * 0.15
		if yDiffThreshold < 1.0 {
			yDiffThreshold = 1.0 // Minimum 1 point threshold
		}
		yDiffers := verticalDist > yDiffThreshold

		// Check if X is within the line's existing range (would overlap)
		xWithinLineRange := frag.X >= lineMinX && frag.X <= lineMaxX

		// If Y differs notably AND X overlaps with line content AND fragment is
		// not just whitespace, it's likely a different column/stream of text.
		// Whitespace fragments often have slightly different baselines and shouldn't
		// trigger column detection.
		isWhitespaceFragment := len(frag.Text) > 0 && frag.Text == " " || frag.Text == ""
		isOverlappingColumn := sameLineByY && yDiffers && xWithinLineRange && !isWhitespaceFragment

		if sameLineByY && !isLineWrap && !isOverlappingColumn {
			// Same line - Y is close, X didn't jump back, and no significant overlap
			currentLine = append(currentLine, frag)
			// Update line extent
			if frag.X < lineMinX {
				lineMinX = frag.X
			}
			if fragEndX > lineMaxX {
				lineMaxX = fragEndX
			}
		} else {
			// New line - either Y changed, X jumped backwards, or overlapping column
			lines = append(lines, currentLine)
			currentLine = []TextFragment{frag}
			lineMinX = frag.X
			lineMaxX = fragEndX
		}
	}

	// Don't forget the last line
	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	return lines
}

// detectLineDirection determines the dominant text direction of a line.
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

// reorderFragmentsForReading reorders fragments for proper reading order based on direction.
// LTR text is sorted left-to-right, RTL text is sorted right-to-left.
// For character-level PDFs where stream order is already correct, it preserves stream order
// to avoid breaking kerning and intentional character positioning.
func (e *Extractor) reorderFragmentsForReading(fragments []TextFragment, lineDir Direction) []TextFragment {
	if len(fragments) <= 1 {
		return fragments
	}

	// Check if this is a character-level line (high fragmentation)
	// and if stream order is already mostly sorted by X
	isCharacterLevel := e.isCharacterLevelLine(fragments)
	streamOrderScore := e.calculateStreamOrderScore(fragments, lineDir)

	// If character-level and stream order is reasonably sorted (score > 0.7),
	// trust the stream order instead of X-sorting
	if isCharacterLevel && streamOrderScore > 0.7 {
		// Return fragments in original stream order
		return fragments
	}

	// Make a copy to avoid modifying original
	ordered := make([]TextFragment, len(fragments))
	copy(ordered, fragments)

	// Use stable sort to preserve stream order for overlapping fragments
	sort.SliceStable(ordered, func(i, j int) bool {
		tolerance := ordered[i].FontSize * xTolerance

		// For RTL: right to left (descending X)
		if lineDir == RTL {
			if abs(ordered[i].X-ordered[j].X) < tolerance {
				return false // Treat as equal, preserve order (i comes before j)
			}
			return ordered[i].X > ordered[j].X
		}

		// For LTR: left to right (ascending X)
		if abs(ordered[i].X-ordered[j].X) < tolerance {
			return false // Treat as equal, preserve order (i comes before j)
		}
		return ordered[i].X < ordered[j].X
	})

	return ordered
}

// isCharacterLevelLine checks if a line consists of highly fragmented text
// (mostly single characters per fragment), which indicates character-level positioning.
func (e *Extractor) isCharacterLevelLine(fragments []TextFragment) bool {
	if len(fragments) == 0 {
		return false
	}

	totalChars := 0
	for _, f := range fragments {
		totalChars += len([]rune(f.Text))
	}

	avgCharsPerFragment := float64(totalChars) / float64(len(fragments))
	return avgCharsPerFragment <= 2.0
}

// calculateStreamOrderScore calculates how well the fragments are already sorted
// in the stream order. Returns a score from 0 to 1, where 1 means perfectly sorted.
// This helps detect when stream order should be trusted over X-sorting.
func (e *Extractor) calculateStreamOrderScore(fragments []TextFragment, lineDir Direction) float64 {
	if len(fragments) <= 1 {
		return 1.0
	}

	// Count how many adjacent pairs are in correct order
	correctPairs := 0
	totalPairs := len(fragments) - 1

	for i := 0; i < totalPairs; i++ {
		curr := fragments[i]
		next := fragments[i+1]

		// Allow tolerance for kerning (characters can overlap slightly)
		tolerance := curr.FontSize * 0.5 // 50% of font size tolerance for kerning

		var inOrder bool
		if lineDir == RTL {
			// For RTL, X should decrease (or stay within tolerance)
			inOrder = next.X <= curr.X+tolerance
		} else {
			// For LTR, X should increase (or stay within tolerance)
			inOrder = next.X >= curr.X-tolerance
		}

		if inOrder {
			correctPairs++
		}
	}

	return float64(correctPairs) / float64(totalPairs)
}

// calculateHorizontalDistance calculates the gap between two fragments,
// accounting for text direction.
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
// based on the horizontal gap and font metrics. This is a simpler version of
// shouldInsertSpaceSmart used for word-level PDFs.
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

// getSpaceWidth returns the expected width of a space character for the given font and size.
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

// abs returns the absolute value of x.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetFragments returns all extracted text fragments with duplicates removed.
// Duplicate fragments are those at the same position with the same text,
// which can occur in PDFs with multiple content layers or tagged structure.
func (e *Extractor) GetFragments() []TextFragment {
	return e.deduplicateFragments()
}

// GetFragmentsRaw returns all extracted text fragments without deduplication.
// Use this when you need to see all fragments including duplicates.
func (e *Extractor) GetFragmentsRaw() []TextFragment {
	return e.fragments
}

// deduplicateFragments removes duplicate text fragments at the same position.
// Some PDFs render the same text multiple times at identical positions due to:
// - Multiple optional content layers (OCG)
// - Tagged PDF structure with repeated content
// - PDF generators creating redundant rendering passes
func (e *Extractor) deduplicateFragments() []TextFragment {
	if len(e.fragments) == 0 {
		return e.fragments
	}

	// Use a map to track unique fragments by position and text
	// Key: rounded position + text content
	type fragKey struct {
		x, y int    // Position rounded to integer (sub-pixel precision not needed)
		text string // Text content
	}

	seen := make(map[fragKey]bool)
	result := make([]TextFragment, 0, len(e.fragments))

	for _, frag := range e.fragments {
		// Round position to handle minor floating point differences
		key := fragKey{
			x:    int(frag.X + 0.5), // Round to nearest integer
			y:    int(frag.Y + 0.5),
			text: frag.Text,
		}

		if !seen[key] {
			seen[key] = true
			result = append(result, frag)
		}
	}

	return result
}

// toFloat converts a PDF numeric object to float64.
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

	var m model.Matrix
	for i, op := range operands {
		if i < 6 {
			m[i], _ = toFloat(op)
		}
	}

	return m
}

// GetFonts returns the fonts registered in this extractor
// Useful for debugging font loading and ToUnicode CMap issues
func (e *Extractor) GetFonts() map[string]*font.Font {
	return e.fonts
}
