package docx

import (
	"strconv"
	"strings"
)

// ResolvedStyle contains the fully resolved properties for a style.
type ResolvedStyle struct {
	// Identity
	ID   string
	Name string
	Type string // paragraph, character, table

	// Heading info
	IsHeading    bool
	HeadingLevel int // 1-9, 0 if not a heading

	// Paragraph properties
	Alignment   string  // left, center, right, both (justify)
	SpaceBefore float64 // points
	SpaceAfter  float64 // points
	LineSpacing float64 // points (0 = auto)
	IndentLeft  float64 // points
	IndentRight float64 // points
	IndentFirst float64 // points (first line indent, can be negative for hanging)

	// Run/character properties
	FontName  string
	FontSize  float64 // points
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	SmallCaps bool
	AllCaps   bool
	Color     string // hex color like "FF0000"
	Highlight string // highlight color name
}

// StyleResolver resolves styles with inheritance support.
type StyleResolver struct {
	styles      map[string]*styleDefXML
	defaults    *docDefaultsXML
	resolved    map[string]*ResolvedStyle
	defaultFont string
	defaultSize float64
}

// NewStyleResolver creates a new style resolver from parsed styles.
func NewStyleResolver(styles *stylesXML) *StyleResolver {
	sr := &StyleResolver{
		styles:      make(map[string]*styleDefXML),
		resolved:    make(map[string]*ResolvedStyle),
		defaultFont: "Calibri", // Word default
		defaultSize: 11,        // Word default (11pt)
	}

	if styles == nil {
		return sr
	}

	// Build style map
	for i := range styles.Styles {
		style := &styles.Styles[i]
		sr.styles[style.StyleID] = style
	}

	// Extract defaults
	sr.defaults = &styles.DocDefaults

	// Parse default font and size from docDefaults
	if sr.defaults != nil {
		if sr.defaults.RPrDefault.RPr.Font.ASCII != "" {
			sr.defaultFont = sr.defaults.RPrDefault.RPr.Font.ASCII
		}
		if sr.defaults.RPrDefault.RPr.FontSize.Val != "" {
			if size := parseHalfPoints(sr.defaults.RPrDefault.RPr.FontSize.Val); size > 0 {
				sr.defaultSize = size
			}
		}
	}

	return sr
}

// Resolve returns the fully resolved style for the given style ID.
// If the style doesn't exist, returns a default style.
func (sr *StyleResolver) Resolve(styleID string) *ResolvedStyle {
	if styleID == "" {
		return sr.defaultStyle()
	}

	// Check cache
	if resolved, ok := sr.resolved[styleID]; ok {
		return resolved
	}

	// Start with default style
	resolved := sr.defaultStyle()
	resolved.ID = styleID

	// Find the style definition
	styleDef, ok := sr.styles[styleID]
	if !ok {
		// Style not found - check for built-in heading styles
		resolved.IsHeading, resolved.HeadingLevel = detectBuiltInHeading(styleID)
		sr.resolved[styleID] = resolved
		return resolved
	}

	resolved.Name = styleDef.Name.Val
	resolved.Type = styleDef.Type

	// Build inheritance chain (from base to derived)
	chain := sr.buildInheritanceChain(styleID)

	// Apply properties from base to derived
	for _, sid := range chain {
		if def, ok := sr.styles[sid]; ok {
			sr.applyStyleDef(resolved, def)
		}
	}

	// Detect heading
	resolved.IsHeading, resolved.HeadingLevel = sr.detectHeading(styleDef, resolved)

	// Cache and return
	sr.resolved[styleID] = resolved
	return resolved
}

// defaultStyle returns a style with default values.
func (sr *StyleResolver) defaultStyle() *ResolvedStyle {
	return &ResolvedStyle{
		FontName:    sr.defaultFont,
		FontSize:    sr.defaultSize,
		Alignment:   "left",
		SpaceAfter:  8, // Default paragraph spacing in Word
		LineSpacing: 0, // Auto
	}
}

// buildInheritanceChain returns style IDs from base to derived.
func (sr *StyleResolver) buildInheritanceChain(styleID string) []string {
	var chain []string
	visited := make(map[string]bool)

	current := styleID
	for current != "" && !visited[current] {
		visited[current] = true
		chain = append([]string{current}, chain...) // Prepend

		if def, ok := sr.styles[current]; ok {
			current = def.BasedOn.Val
		} else {
			break
		}
	}

	return chain
}

// applyStyleDef applies a style definition's properties to a resolved style.
func (sr *StyleResolver) applyStyleDef(resolved *ResolvedStyle, def *styleDefXML) {
	// Paragraph properties
	ppr := def.PPr
	if ppr.Justification.Val != "" {
		resolved.Alignment = ppr.Justification.Val
	}
	if ppr.Spacing.Before != "" {
		resolved.SpaceBefore = parseTwips(ppr.Spacing.Before)
	}
	if ppr.Spacing.After != "" {
		resolved.SpaceAfter = parseTwips(ppr.Spacing.After)
	}
	if ppr.Spacing.Line != "" {
		resolved.LineSpacing = parseTwips(ppr.Spacing.Line)
	}
	if ppr.Indent.Left != "" {
		resolved.IndentLeft = parseTwips(ppr.Indent.Left)
	}
	if ppr.Indent.Right != "" {
		resolved.IndentRight = parseTwips(ppr.Indent.Right)
	}
	if ppr.Indent.FirstLine != "" {
		resolved.IndentFirst = parseTwips(ppr.Indent.FirstLine)
	}
	if ppr.Indent.Hanging != "" {
		resolved.IndentFirst = -parseTwips(ppr.Indent.Hanging)
	}

	// Run properties
	rpr := def.RPr
	if rpr.Font.ASCII != "" {
		resolved.FontName = rpr.Font.ASCII
	}
	if rpr.FontSize.Val != "" {
		if size := parseHalfPoints(rpr.FontSize.Val); size > 0 {
			resolved.FontSize = size
		}
	}
	// Bold - present means true (unless val="false" or val="0")
	if rpr.Bold.XMLName.Local != "" {
		resolved.Bold = rpr.Bold.Val != "false" && rpr.Bold.Val != "0"
	}
	if rpr.Italic.XMLName.Local != "" {
		resolved.Italic = rpr.Italic.Val != "false" && rpr.Italic.Val != "0"
	}
	if rpr.Strike.XMLName.Local != "" {
		resolved.Strike = rpr.Strike.Val != "false" && rpr.Strike.Val != "0"
	}
	if rpr.Underline.Val != "" && rpr.Underline.Val != "none" {
		resolved.Underline = true
	}
	if rpr.Color.Val != "" && rpr.Color.Val != "auto" {
		resolved.Color = rpr.Color.Val
	}
	if rpr.Highlight.Val != "" {
		resolved.Highlight = rpr.Highlight.Val
	}
}

// detectHeading determines if a style represents a heading.
func (sr *StyleResolver) detectHeading(def *styleDefXML, resolved *ResolvedStyle) (bool, int) {
	// Check for built-in heading style ID
	if isHeading, level := detectBuiltInHeading(def.StyleID); isHeading {
		return true, level
	}

	// Check style name for heading patterns
	name := strings.ToLower(def.Name.Val)
	if strings.HasPrefix(name, "heading") || strings.HasPrefix(name, "heading ") {
		// Try to extract level from name
		for i := 1; i <= 9; i++ {
			if strings.Contains(name, strconv.Itoa(i)) {
				return true, i
			}
		}
		return true, 1 // Default to H1
	}

	// Check outline level
	if def.PPr.OutlineLvl.Val != "" {
		level := parseOutlineLevel(def.PPr.OutlineLvl.Val)
		if level >= 0 && level <= 8 {
			return true, level + 1 // OutlineLvl is 0-based
		}
	}

	// Heuristic: large, bold text at start of document section might be heading
	// (This is a fallback for documents without proper heading styles)
	if resolved.Bold && resolved.FontSize >= 14 {
		return true, estimateHeadingLevel(resolved.FontSize)
	}

	return false, 0
}

// detectBuiltInHeading checks for Word's built-in heading style IDs.
func detectBuiltInHeading(styleID string) (bool, int) {
	id := strings.ToLower(styleID)

	headingMap := map[string]int{
		"heading1": 1, "heading2": 2, "heading3": 3,
		"heading4": 4, "heading5": 5, "heading6": 6,
		"heading7": 7, "heading8": 8, "heading9": 9,
		"title": 1, "subtitle": 2,
	}

	if level, ok := headingMap[id]; ok {
		return true, level
	}

	return false, 0
}

// estimateHeadingLevel estimates heading level from font size.
func estimateHeadingLevel(fontSize float64) int {
	switch {
	case fontSize >= 24:
		return 1
	case fontSize >= 18:
		return 2
	case fontSize >= 14:
		return 3
	case fontSize >= 12:
		return 4
	default:
		return 5
	}
}

// parseHalfPoints parses a size in half-points to points.
// Word uses half-points for font sizes (e.g., "24" = 12pt).
func parseHalfPoints(s string) float64 {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val / 2
}

// parseTwips parses a size in twips to points.
// 1 point = 20 twips.
func parseTwips(s string) float64 {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val / 20
}

// ResolvedRun contains resolved properties for a text run.
type ResolvedRun struct {
	Text      string
	FontName  string
	FontSize  float64
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	Color     string
	Highlight string
}

// ResolveRun resolves run properties, combining paragraph style with direct formatting.
func (sr *StyleResolver) ResolveRun(paragraphStyle string, runProps runPropsXML) *ResolvedRun {
	// Start with paragraph style
	baseStyle := sr.Resolve(paragraphStyle)

	resolved := &ResolvedRun{
		FontName: baseStyle.FontName,
		FontSize: baseStyle.FontSize,
		Bold:     baseStyle.Bold,
		Italic:   baseStyle.Italic,
	}

	// Apply direct run formatting (overrides style)
	if runProps.Font.ASCII != "" {
		resolved.FontName = runProps.Font.ASCII
	}
	if runProps.FontSize.Val != "" {
		if size := parseHalfPoints(runProps.FontSize.Val); size > 0 {
			resolved.FontSize = size
		}
	}
	if runProps.Bold.XMLName.Local != "" {
		resolved.Bold = runProps.Bold.Val != "false" && runProps.Bold.Val != "0"
	}
	if runProps.Italic.XMLName.Local != "" {
		resolved.Italic = runProps.Italic.Val != "false" && runProps.Italic.Val != "0"
	}
	if runProps.Strike.XMLName.Local != "" {
		resolved.Strike = runProps.Strike.Val != "false" && runProps.Strike.Val != "0"
	}
	if runProps.Underline.Val != "" && runProps.Underline.Val != "none" {
		resolved.Underline = true
	}
	if runProps.Color.Val != "" && runProps.Color.Val != "auto" {
		resolved.Color = runProps.Color.Val
	}
	if runProps.Highlight.Val != "" {
		resolved.Highlight = runProps.Highlight.Val
	}

	return resolved
}
