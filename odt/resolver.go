package odt

import (
	"strconv"
	"strings"
)

// ResolvedStyle contains the fully resolved properties for a style.
type ResolvedStyle struct {
	// Identity
	Name   string
	Family string // paragraph, text, table, table-cell, etc.

	// Heading info
	IsHeading    bool
	HeadingLevel int // 1-9, 0 if not a heading

	// Paragraph properties
	Alignment   string  // left, center, right, justify
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
	Color     string // hex color like "#FF0000"
}

// StyleResolver resolves styles with inheritance support.
type StyleResolver struct {
	styles      map[string]*styleDefXML
	listStyles  map[string]*listStyleXML
	resolved    map[string]*ResolvedStyle
	defaultFont string
	defaultSize float64
}

// NewStyleResolver creates a new style resolver from parsed styles.
func NewStyleResolver(contentStyles *contentStylesXML, docStyles *stylesXML) *StyleResolver {
	sr := &StyleResolver{
		styles:      make(map[string]*styleDefXML),
		listStyles:  make(map[string]*listStyleXML),
		resolved:    make(map[string]*ResolvedStyle),
		defaultFont: "Liberation Sans", // LibreOffice default
		defaultSize: 12,                // Default 12pt
	}

	// Add styles from styles.xml (named styles)
	if docStyles != nil {
		if docStyles.Styles != nil {
			for i := range docStyles.Styles.Styles {
				style := &docStyles.Styles.Styles[i]
				sr.styles[style.Name] = style
			}
			for i := range docStyles.Styles.ListStyles {
				ls := &docStyles.Styles.ListStyles[i]
				sr.listStyles[ls.Name] = ls
			}
		}
		if docStyles.AutoStyles != nil {
			for i := range docStyles.AutoStyles.Styles {
				style := &docStyles.AutoStyles.Styles[i]
				sr.styles[style.Name] = style
			}
			for i := range docStyles.AutoStyles.ListStyles {
				ls := &docStyles.AutoStyles.ListStyles[i]
				sr.listStyles[ls.Name] = ls
			}
		}
	}

	// Add automatic styles from content.xml (override named styles)
	if contentStyles != nil {
		for i := range contentStyles.Styles {
			style := &contentStyles.Styles[i]
			sr.styles[style.Name] = style
		}
		for i := range contentStyles.ListStyles {
			ls := &contentStyles.ListStyles[i]
			sr.listStyles[ls.Name] = ls
		}
	}

	return sr
}

// Resolve returns the fully resolved style for the given style name.
// If the style doesn't exist, returns a default style.
func (sr *StyleResolver) Resolve(styleName string) *ResolvedStyle {
	if styleName == "" {
		return sr.defaultStyle()
	}

	// Check cache
	if resolved, ok := sr.resolved[styleName]; ok {
		return resolved
	}

	// Start with default style
	resolved := sr.defaultStyle()
	resolved.Name = styleName

	// Find the style definition
	styleDef, ok := sr.styles[styleName]
	if !ok {
		// Style not found - check for built-in heading styles
		resolved.IsHeading, resolved.HeadingLevel = detectBuiltInHeading(styleName)
		sr.resolved[styleName] = resolved
		return resolved
	}

	resolved.Family = styleDef.Family

	// Build inheritance chain (from base to derived)
	chain := sr.buildInheritanceChain(styleName)

	// Apply properties from base to derived
	for _, name := range chain {
		if def, ok := sr.styles[name]; ok {
			sr.applyStyleDef(resolved, def)
		}
	}

	// Detect heading from default-outline-level
	if styleDef.DefaultOutlineLevel != "" {
		if level, err := strconv.Atoi(styleDef.DefaultOutlineLevel); err == nil && level >= 1 && level <= 9 {
			resolved.IsHeading = true
			resolved.HeadingLevel = level
		}
	}

	// Fallback: detect heading from style name
	if !resolved.IsHeading {
		resolved.IsHeading, resolved.HeadingLevel = detectBuiltInHeading(styleName)
	}

	// Cache and return
	sr.resolved[styleName] = resolved
	return resolved
}

// defaultStyle returns a style with default values.
func (sr *StyleResolver) defaultStyle() *ResolvedStyle {
	return &ResolvedStyle{
		FontName:    sr.defaultFont,
		FontSize:    sr.defaultSize,
		Alignment:   "left",
		SpaceAfter:  0,
		LineSpacing: 0, // Auto
	}
}

// buildInheritanceChain returns style names from base to derived.
func (sr *StyleResolver) buildInheritanceChain(styleName string) []string {
	var chain []string
	visited := make(map[string]bool)

	current := styleName
	for current != "" && !visited[current] {
		visited[current] = true
		chain = append([]string{current}, chain...) // Prepend

		if def, ok := sr.styles[current]; ok {
			current = def.ParentStyleName
		} else {
			break
		}
	}

	return chain
}

// applyStyleDef applies a style definition's properties to a resolved style.
func (sr *StyleResolver) applyStyleDef(resolved *ResolvedStyle, def *styleDefXML) {
	// Paragraph properties
	if ppr := def.ParagraphProps; ppr != nil {
		if ppr.TextAlign != "" {
			resolved.Alignment = ppr.TextAlign
		}
		if ppr.MarginTop != "" {
			resolved.SpaceBefore = parseLength(ppr.MarginTop)
		}
		if ppr.MarginBottom != "" {
			resolved.SpaceAfter = parseLength(ppr.MarginBottom)
		}
		if ppr.LineHeight != "" {
			resolved.LineSpacing = parseLength(ppr.LineHeight)
		}
		if ppr.MarginLeft != "" {
			resolved.IndentLeft = parseLength(ppr.MarginLeft)
		}
		if ppr.MarginRight != "" {
			resolved.IndentRight = parseLength(ppr.MarginRight)
		}
		if ppr.TextIndent != "" {
			resolved.IndentFirst = parseLength(ppr.TextIndent)
		}
	}

	// Text properties
	if tpr := def.TextProps; tpr != nil {
		if tpr.FontName != "" {
			resolved.FontName = tpr.FontName
		} else if tpr.FontFamily != "" {
			resolved.FontName = cleanFontFamily(tpr.FontFamily)
		}
		if tpr.FontSize != "" {
			if size := parseLength(tpr.FontSize); size > 0 {
				resolved.FontSize = size
			}
		}
		if tpr.FontWeight == "bold" {
			resolved.Bold = true
		}
		if tpr.FontStyle == "italic" {
			resolved.Italic = true
		}
		if tpr.TextUnderline != "" && tpr.TextUnderline != "none" {
			resolved.Underline = true
		}
		if tpr.TextLineThrough != "" && tpr.TextLineThrough != "none" {
			resolved.Strike = true
		}
		if tpr.Color != "" {
			resolved.Color = tpr.Color
		}
	}
}

// detectBuiltInHeading checks for common heading style names.
func detectBuiltInHeading(styleName string) (bool, int) {
	name := strings.ToLower(styleName)

	// Standard heading patterns
	headingPatterns := map[string]int{
		"heading_1": 1, "heading_2": 2, "heading_3": 3,
		"heading_4": 4, "heading_5": 5, "heading_6": 6,
		"heading_7": 7, "heading_8": 8, "heading_9": 9,
		"heading1": 1, "heading2": 2, "heading3": 3,
		"heading4": 4, "heading5": 5, "heading6": 6,
		"heading7": 7, "heading8": 8, "heading9": 9,
		"title": 1, "subtitle": 2,
	}

	// Direct match
	if level, ok := headingPatterns[name]; ok {
		return true, level
	}

	// Pattern match: "Heading 1", "Heading 2", etc.
	if strings.HasPrefix(name, "heading") {
		for i := 1; i <= 9; i++ {
			if strings.Contains(name, strconv.Itoa(i)) {
				return true, i
			}
		}
		return true, 1 // Default to H1
	}

	return false, 0
}

// parseLength parses an ODF length value to points.
// Supports: pt, in, cm, mm, px
func parseLength(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// Try parsing with units
	var value float64
	var unit string

	// Find where digits end and unit begins
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && c != '.' && c != '-' {
			break
		}
	}

	if i == 0 {
		return 0
	}

	val, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0
	}
	value = val
	unit = strings.ToLower(strings.TrimSpace(s[i:]))

	// Convert to points
	switch unit {
	case "pt":
		return value
	case "in":
		return value * 72 // 72 points per inch
	case "cm":
		return value * 28.3465 // ~28.35 points per cm
	case "mm":
		return value * 2.83465 // ~2.835 points per mm
	case "px":
		return value * 0.75 // Assume 96 DPI -> 72/96
	case "%":
		return 0 // Percentages not converted to points
	case "":
		return value // Assume points if no unit
	default:
		return value // Unknown unit, assume points
	}
}

// cleanFontFamily removes quotes from font family names.
func cleanFontFamily(family string) string {
	family = strings.TrimSpace(family)
	family = strings.Trim(family, "'\"")
	return family
}

// ResolvedListLevel contains resolved list level properties.
type ResolvedListLevel struct {
	Level      int
	IsBullet   bool
	BulletChar string
	NumFormat  string // "1", "a", "A", "i", "I"
	NumPrefix  string
	NumSuffix  string
	StartValue int
}

// ResolveListLevel returns the resolved list level for a given list style and level.
func (sr *StyleResolver) ResolveListLevel(listStyleName string, level int) *ResolvedListLevel {
	result := &ResolvedListLevel{
		Level:      level,
		IsBullet:   true,
		BulletChar: "•",
		StartValue: 1,
	}

	if listStyleName == "" {
		return result
	}

	ls, ok := sr.listStyles[listStyleName]
	if !ok {
		return result
	}

	levelStr := strconv.Itoa(level + 1) // ODF levels are 1-based

	// Check bullet levels
	for _, bl := range ls.BulletLevels {
		if bl.Level == levelStr {
			result.IsBullet = true
			if bl.BulletChar != "" {
				result.BulletChar = bl.BulletChar
			}
			result.NumPrefix = bl.NumPrefix
			result.NumSuffix = bl.NumSuffix
			return result
		}
	}

	// Check number levels
	for _, nl := range ls.NumberLevels {
		if nl.Level == levelStr {
			result.IsBullet = false
			result.NumFormat = nl.NumFormat
			result.NumPrefix = nl.NumPrefix
			result.NumSuffix = nl.NumSuffix
			if nl.StartValue != "" {
				if sv, err := strconv.Atoi(nl.StartValue); err == nil {
					result.StartValue = sv
				}
			}
			return result
		}
	}

	return result
}
