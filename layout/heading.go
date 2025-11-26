// Package layout provides document layout analysis including heading detection,
// which identifies and classifies document headings by level (H1-H6).
package layout

import (
	"regexp"
	"sort"
	"strings"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// HeadingLevel represents the hierarchical level of a heading (H1-H6)
type HeadingLevel int

const (
	HeadingLevelUnknown HeadingLevel = iota
	HeadingLevel1                    // H1 - Main title/chapter
	HeadingLevel2                    // H2 - Major section
	HeadingLevel3                    // H3 - Subsection
	HeadingLevel4                    // H4 - Sub-subsection
	HeadingLevel5                    // H5 - Minor heading
	HeadingLevel6                    // H6 - Lowest level heading
)

// String returns a string representation of the heading level
func (l HeadingLevel) String() string {
	switch l {
	case HeadingLevel1:
		return "h1"
	case HeadingLevel2:
		return "h2"
	case HeadingLevel3:
		return "h3"
	case HeadingLevel4:
		return "h4"
	case HeadingLevel5:
		return "h5"
	case HeadingLevel6:
		return "h6"
	default:
		return "unknown"
	}
}

// HTMLTag returns the HTML tag for this heading level
func (l HeadingLevel) HTMLTag() string {
	if l >= HeadingLevel1 && l <= HeadingLevel6 {
		return l.String()
	}
	return "p"
}

// Heading represents a detected heading in a document
type Heading struct {
	// Level is the heading level (H1-H6)
	Level HeadingLevel

	// Text is the heading text content
	Text string

	// BBox is the bounding box of the heading
	BBox model.BBox

	// Lines are the lines that make up this heading
	Lines []Line

	// Fragments are the text fragments in this heading
	Fragments []text.TextFragment

	// Index is the heading's position in document order (0-based)
	Index int

	// PageIndex is the page number where this heading appears (0-based)
	PageIndex int

	// FontSize is the average font size of the heading
	FontSize float64

	// IsBold indicates if the heading appears to be bold
	IsBold bool

	// IsItalic indicates if the heading appears to be italic
	IsItalic bool

	// IsAllCaps indicates if the heading is in all capital letters
	IsAllCaps bool

	// IsNumbered indicates if the heading has a number prefix (e.g., "1.2.3")
	IsNumbered bool

	// NumberPrefix is the number prefix if present (e.g., "1.2.3")
	NumberPrefix string

	// Confidence is a score from 0-1 indicating detection confidence
	Confidence float64

	// Alignment is the text alignment of the heading
	Alignment LineAlignment

	// SpacingBefore is the vertical space before this heading
	SpacingBefore float64

	// SpacingAfter is the vertical space after this heading
	SpacingAfter float64
}

// HeadingLayout represents all detected headings in a document or page
type HeadingLayout struct {
	// Headings are all detected headings in document order
	Headings []Heading

	// PageWidth and PageHeight of the analyzed page/document
	PageWidth  float64
	PageHeight float64

	// BodyFontSize is the detected average body text font size
	BodyFontSize float64

	// Config is the configuration used for detection
	Config HeadingConfig
}

// HeadingConfig holds configuration for heading detection
type HeadingConfig struct {
	// FontSizeRatios maps heading levels to minimum font size ratios relative to body text
	// Default: H1=1.8, H2=1.5, H3=1.3, H4=1.15, H5=1.1, H6=1.05
	FontSizeRatios map[HeadingLevel]float64

	// MaxHeadingLines is the maximum number of lines for a heading
	// Default: 3
	MaxHeadingLines int

	// MinSpacingRatio is the minimum spacing ratio (vs avg) to consider a heading
	// Default: 1.2 (20% more spacing before heading)
	MinSpacingRatio float64

	// BoldIndicatesHeading when true, bold text is more likely a heading
	// Default: true
	BoldIndicatesHeading bool

	// AllCapsIndicatesHeading when true, ALL CAPS text is more likely a heading
	// Default: true
	AllCapsIndicatesHeading bool

	// CenterAlignedBoost is the confidence boost for centered headings
	// Default: 0.1
	CenterAlignedBoost float64

	// NumberedPatterns are regex patterns for numbered headings
	// Default: "1.", "1.1", "1.1.1", "Chapter 1", etc.
	NumberedPatterns []*regexp.Regexp

	// MinConfidence is the minimum confidence to consider something a heading
	// Default: 0.5
	MinConfidence float64
}

// DefaultHeadingConfig returns sensible default configuration
func DefaultHeadingConfig() HeadingConfig {
	return HeadingConfig{
		FontSizeRatios: map[HeadingLevel]float64{
			HeadingLevel1: 1.8,  // 80% larger than body
			HeadingLevel2: 1.5,  // 50% larger
			HeadingLevel3: 1.3,  // 30% larger
			HeadingLevel4: 1.15, // 15% larger
			HeadingLevel5: 1.1,  // 10% larger
			HeadingLevel6: 1.05, // 5% larger
		},
		MaxHeadingLines:         3,
		MinSpacingRatio:         1.2,
		BoldIndicatesHeading:    true,
		AllCapsIndicatesHeading: true,
		CenterAlignedBoost:      0.1,
		NumberedPatterns: []*regexp.Regexp{
			regexp.MustCompile(`^(?i)(chapter|section|part)\s+\d+`),
			regexp.MustCompile(`^\d+\.\s`),
			regexp.MustCompile(`^\d+\.\d+\s`),
			regexp.MustCompile(`^\d+\.\d+\.\d+\s`),
			regexp.MustCompile(`^[IVXLCDM]+\.\s`), // Roman numerals
			regexp.MustCompile(`^[A-Z]\.\s`),      // Letter prefixes
		},
		MinConfidence: 0.5,
	}
}

// HeadingDetector detects and classifies headings in document content
type HeadingDetector struct {
	config HeadingConfig
}

// NewHeadingDetector creates a new heading detector with default configuration
func NewHeadingDetector() *HeadingDetector {
	return &HeadingDetector{
		config: DefaultHeadingConfig(),
	}
}

// NewHeadingDetectorWithConfig creates a heading detector with custom configuration
func NewHeadingDetectorWithConfig(config HeadingConfig) *HeadingDetector {
	return &HeadingDetector{
		config: config,
	}
}

// DetectFromParagraphs analyzes paragraphs and detects headings
func (d *HeadingDetector) DetectFromParagraphs(paragraphs []Paragraph, pageWidth, pageHeight float64) *HeadingLayout {
	if len(paragraphs) == 0 {
		return &HeadingLayout{
			PageWidth:  pageWidth,
			PageHeight: pageHeight,
			Config:     d.config,
		}
	}

	// Calculate body font size (most common font size, excluding outliers)
	bodyFontSize := d.detectBodyFontSize(paragraphs)

	// Analyze each paragraph as potential heading
	var headings []Heading
	for i, para := range paragraphs {
		if heading, isHeading := d.analyzeAsHeading(para, i, bodyFontSize); isHeading {
			headings = append(headings, heading)
		}
	}

	// Refine heading levels based on document structure
	d.refineHeadingLevels(headings, bodyFontSize)

	// Calculate spacing
	d.calculateHeadingSpacing(headings, paragraphs)

	return &HeadingLayout{
		Headings:     headings,
		PageWidth:    pageWidth,
		PageHeight:   pageHeight,
		BodyFontSize: bodyFontSize,
		Config:       d.config,
	}
}

// DetectFromLines analyzes lines directly and detects headings
func (d *HeadingDetector) DetectFromLines(lines []Line, pageWidth, pageHeight float64) *HeadingLayout {
	// First detect paragraphs, then headings
	paraDetector := NewParagraphDetector()
	paraLayout := paraDetector.Detect(lines, pageWidth, pageHeight)
	return d.DetectFromParagraphs(paraLayout.Paragraphs, pageWidth, pageHeight)
}

// DetectFromFragments analyzes fragments directly and detects headings
func (d *HeadingDetector) DetectFromFragments(fragments []text.TextFragment, pageWidth, pageHeight float64) *HeadingLayout {
	// First detect lines, then paragraphs, then headings
	lineDetector := NewLineDetector()
	lineLayout := lineDetector.Detect(fragments, pageWidth, pageHeight)
	return d.DetectFromLines(lineLayout.Lines, pageWidth, pageHeight)
}

// detectBodyFontSize determines the most common (body) font size
func (d *HeadingDetector) detectBodyFontSize(paragraphs []Paragraph) float64 {
	if len(paragraphs) == 0 {
		return 12.0 // Default
	}

	// Count font sizes with bucketing
	fontCounts := make(map[int]int)
	tolerance := 0.5

	for _, para := range paragraphs {
		bucket := int(para.AverageFontSize / tolerance)
		// Weight by number of lines (body text usually has more lines)
		fontCounts[bucket] += len(para.Lines)
	}

	// Find most common font size bucket
	maxCount := 0
	mostCommonBucket := 0
	for bucket, count := range fontCounts {
		if count > maxCount {
			maxCount = count
			mostCommonBucket = bucket
		}
	}

	return float64(mostCommonBucket) * tolerance
}

// analyzeAsHeading analyzes a paragraph to determine if it's a heading
func (d *HeadingDetector) analyzeAsHeading(para Paragraph, index int, bodyFontSize float64) (Heading, bool) {
	// Already marked as heading by paragraph detector
	if para.Style == StyleHeading {
		heading := d.createHeading(para, index, bodyFontSize)
		return heading, heading.Confidence >= d.config.MinConfidence
	}

	// Too many lines to be a heading
	if len(para.Lines) > d.config.MaxHeadingLines {
		return Heading{}, false
	}

	// Analyze characteristics
	heading := d.createHeading(para, index, bodyFontSize)

	// Must meet minimum confidence
	return heading, heading.Confidence >= d.config.MinConfidence
}

// createHeading creates a Heading from a paragraph with confidence scoring
func (d *HeadingDetector) createHeading(para Paragraph, index int, bodyFontSize float64) Heading {
	heading := Heading{
		Text:      para.Text,
		BBox:      para.BBox,
		Lines:     para.Lines,
		Index:     index,
		FontSize:  para.AverageFontSize,
		Alignment: para.Alignment,
	}

	// Collect fragments
	for _, line := range para.Lines {
		heading.Fragments = append(heading.Fragments, line.Fragments...)
	}

	// Detect characteristics
	heading.IsBold = d.detectBold(para)
	heading.IsItalic = d.detectItalic(para)
	heading.IsAllCaps = d.detectAllCaps(para.Text)
	heading.IsNumbered, heading.NumberPrefix = d.detectNumbered(para.Text)

	// Calculate confidence score
	heading.Confidence = d.calculateConfidence(para, bodyFontSize, heading)

	// Determine heading level
	heading.Level = d.determineLevel(para.AverageFontSize, bodyFontSize, heading)

	return heading
}

// detectBold checks if the paragraph appears to be bold
func (d *HeadingDetector) detectBold(para Paragraph) bool {
	// Check font name for bold indicators
	for _, line := range para.Lines {
		for _, frag := range line.Fragments {
			fontLower := strings.ToLower(frag.FontName)
			if strings.Contains(fontLower, "bold") ||
				strings.Contains(fontLower, "black") ||
				strings.Contains(fontLower, "heavy") ||
				strings.Contains(fontLower, "semibold") ||
				strings.Contains(fontLower, "demibold") {
				return true
			}
		}
	}
	return false
}

// detectItalic checks if the paragraph appears to be italic
func (d *HeadingDetector) detectItalic(para Paragraph) bool {
	for _, line := range para.Lines {
		for _, frag := range line.Fragments {
			fontLower := strings.ToLower(frag.FontName)
			if strings.Contains(fontLower, "italic") ||
				strings.Contains(fontLower, "oblique") {
				return true
			}
		}
	}
	return false
}

// detectAllCaps checks if text is in all capital letters
func (d *HeadingDetector) detectAllCaps(text string) bool {
	text = strings.TrimSpace(text)
	if len(text) < 3 {
		return false
	}

	// Count letters
	upperCount := 0
	lowerCount := 0
	for _, r := range text {
		if r >= 'A' && r <= 'Z' {
			upperCount++
		} else if r >= 'a' && r <= 'z' {
			lowerCount++
		}
	}

	// Must have some letters
	if upperCount+lowerCount < 3 {
		return false
	}

	// All caps if 90%+ uppercase
	return lowerCount == 0 || float64(upperCount)/float64(upperCount+lowerCount) > 0.9
}

// detectNumbered checks if text starts with a numbering pattern
func (d *HeadingDetector) detectNumbered(text string) (bool, string) {
	text = strings.TrimSpace(text)

	for _, pattern := range d.config.NumberedPatterns {
		if match := pattern.FindString(text); match != "" {
			return true, strings.TrimSpace(match)
		}
	}

	return false, ""
}

// calculateConfidence calculates a confidence score for heading detection
func (d *HeadingDetector) calculateConfidence(para Paragraph, bodyFontSize float64, heading Heading) float64 {
	confidence := 0.0

	// Font size is the strongest indicator
	if bodyFontSize > 0 {
		fontRatio := para.AverageFontSize / bodyFontSize
		if fontRatio >= 1.5 {
			confidence += 0.5 // Very large font - strong heading indicator
		} else if fontRatio >= 1.2 {
			confidence += 0.35
		} else if fontRatio >= 1.1 {
			confidence += 0.2
		} else if fontRatio >= 1.05 {
			confidence += 0.1
		}
	}

	// Bold text
	if heading.IsBold && d.config.BoldIndicatesHeading {
		confidence += 0.2
	}

	// All caps
	if heading.IsAllCaps && d.config.AllCapsIndicatesHeading {
		confidence += 0.15
	}

	// Numbered heading pattern
	if heading.IsNumbered {
		confidence += 0.2
	}

	// Center alignment
	if para.Alignment == AlignCenter {
		confidence += d.config.CenterAlignedBoost
	}

	// Short text (headings are typically short)
	wordCount := len(strings.Fields(para.Text))
	if wordCount <= 10 {
		confidence += 0.1
	} else if wordCount <= 20 {
		confidence += 0.05
	}

	// Few lines
	if len(para.Lines) == 1 {
		confidence += 0.1
	} else if len(para.Lines) <= 2 {
		confidence += 0.05
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// determineLevel determines the heading level based on font size and characteristics
func (d *HeadingDetector) determineLevel(fontSize, bodyFontSize float64, heading Heading) HeadingLevel {
	if bodyFontSize <= 0 {
		bodyFontSize = 12.0
	}

	fontRatio := fontSize / bodyFontSize

	// Check numbered patterns for level hints
	if heading.IsNumbered {
		// Count dots in number prefix to infer level
		dots := strings.Count(heading.NumberPrefix, ".")
		if dots == 0 {
			// "Chapter 1" or "1" style
			if strings.Contains(strings.ToLower(heading.NumberPrefix), "chapter") ||
				strings.Contains(strings.ToLower(heading.NumberPrefix), "part") {
				return HeadingLevel1
			}
		}
		// "1.2" = level 2, "1.2.3" = level 3, etc.
		level := HeadingLevel(dots + 1)
		if level > HeadingLevel6 {
			level = HeadingLevel6
		}
		if level >= HeadingLevel1 {
			return level
		}
	}

	// Determine by font size ratio
	for level := HeadingLevel1; level <= HeadingLevel6; level++ {
		ratio, ok := d.config.FontSizeRatios[level]
		if ok && fontRatio >= ratio {
			return level
		}
	}

	// Default to H6 if it's a heading but doesn't meet other thresholds
	if heading.Confidence >= d.config.MinConfidence {
		return HeadingLevel6
	}

	return HeadingLevelUnknown
}

// refineHeadingLevels adjusts heading levels based on document structure
func (d *HeadingDetector) refineHeadingLevels(headings []Heading, bodyFontSize float64) {
	if len(headings) == 0 {
		return
	}

	// Find all unique font sizes used by headings
	fontSizes := make([]float64, 0)
	fontSizeSet := make(map[int]bool)

	for _, h := range headings {
		bucket := int(h.FontSize * 10) // 0.1pt precision
		if !fontSizeSet[bucket] {
			fontSizeSet[bucket] = true
			fontSizes = append(fontSizes, h.FontSize)
		}
	}

	// Sort font sizes descending
	sort.Slice(fontSizes, func(i, j int) bool {
		return fontSizes[i] > fontSizes[j]
	})

	// Map font sizes to heading levels
	fontSizeToLevel := make(map[int]HeadingLevel)
	for i, size := range fontSizes {
		level := HeadingLevel(i + 1)
		if level > HeadingLevel6 {
			level = HeadingLevel6
		}
		bucket := int(size * 10)
		fontSizeToLevel[bucket] = level
	}

	// Update heading levels based on relative font sizes
	for i := range headings {
		bucket := int(headings[i].FontSize * 10)
		if level, ok := fontSizeToLevel[bucket]; ok {
			headings[i].Level = level
		}
	}
}

// calculateHeadingSpacing calculates spacing before and after headings
func (d *HeadingDetector) calculateHeadingSpacing(headings []Heading, paragraphs []Paragraph) {
	for i := range headings {
		headingIdx := headings[i].Index

		// Find spacing from paragraphs
		if headingIdx < len(paragraphs) {
			headings[i].SpacingBefore = paragraphs[headingIdx].SpacingBefore
			headings[i].SpacingAfter = paragraphs[headingIdx].SpacingAfter
		}
	}
}

// HeadingLayout methods

// HeadingCount returns the number of detected headings
func (l *HeadingLayout) HeadingCount() int {
	if l == nil {
		return 0
	}
	return len(l.Headings)
}

// GetHeading returns a specific heading by index
func (l *HeadingLayout) GetHeading(index int) *Heading {
	if l == nil || index < 0 || index >= len(l.Headings) {
		return nil
	}
	return &l.Headings[index]
}

// GetHeadingsAtLevel returns all headings at a specific level
func (l *HeadingLayout) GetHeadingsAtLevel(level HeadingLevel) []Heading {
	if l == nil {
		return nil
	}

	var result []Heading
	for _, h := range l.Headings {
		if h.Level == level {
			result = append(result, h)
		}
	}
	return result
}

// GetHeadingsInRange returns headings within a specific level range (inclusive)
func (l *HeadingLayout) GetHeadingsInRange(minLevel, maxLevel HeadingLevel) []Heading {
	if l == nil {
		return nil
	}

	var result []Heading
	for _, h := range l.Headings {
		if h.Level >= minLevel && h.Level <= maxLevel {
			result = append(result, h)
		}
	}
	return result
}

// GetH1 returns all H1 (top-level) headings
func (l *HeadingLayout) GetH1() []Heading {
	return l.GetHeadingsAtLevel(HeadingLevel1)
}

// GetH2 returns all H2 headings
func (l *HeadingLayout) GetH2() []Heading {
	return l.GetHeadingsAtLevel(HeadingLevel2)
}

// GetH3 returns all H3 headings
func (l *HeadingLayout) GetH3() []Heading {
	return l.GetHeadingsAtLevel(HeadingLevel3)
}

// OutlineEntry represents an entry in a document outline
type OutlineEntry struct {
	// Heading is the heading for this entry
	Heading Heading

	// Children are nested outline entries
	Children []OutlineEntry

	// Depth is the nesting depth (0 = top level)
	Depth int
}

// GetOutline returns a hierarchical outline of the document
func (l *HeadingLayout) GetOutline() []OutlineEntry {
	if l == nil || len(l.Headings) == 0 {
		return nil
	}

	var outline []OutlineEntry
	var stack []*OutlineEntry

	for _, h := range l.Headings {
		entry := OutlineEntry{
			Heading: h,
			Depth:   int(h.Level) - 1,
		}

		// Find parent for this entry
		for len(stack) > 0 && stack[len(stack)-1].Heading.Level >= h.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			// Top-level entry
			outline = append(outline, entry)
			stack = append(stack, &outline[len(outline)-1])
		} else {
			// Nested entry
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, entry)
			stack = append(stack, &parent.Children[len(parent.Children)-1])
		}
	}

	return outline
}

// GetTableOfContents returns a formatted table of contents string
func (l *HeadingLayout) GetTableOfContents() string {
	if l == nil || len(l.Headings) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, h := range l.Headings {
		// Indent based on level
		indent := strings.Repeat("  ", int(h.Level)-1)
		sb.WriteString(indent)
		sb.WriteString(h.Text)
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetMarkdownTOC returns a markdown-formatted table of contents
func (l *HeadingLayout) GetMarkdownTOC() string {
	if l == nil || len(l.Headings) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, h := range l.Headings {
		// Indent based on level
		indent := strings.Repeat("  ", int(h.Level)-1)
		sb.WriteString(indent)
		sb.WriteString("- ")
		sb.WriteString(h.Text)
		sb.WriteString("\n")
	}

	return sb.String()
}

// FindHeadingBefore returns the most recent heading that appears before the given
// Y position in reading order. In standard PDF coordinates (Y increases upward),
// this returns the last heading whose Y coordinate is greater than the given Y.
// For example, if querying Y=450 with headings at Y=700, 500, 300, it returns
// the heading at Y=500 (the closest heading above Y=450).
func (l *HeadingLayout) FindHeadingBefore(y float64) *Heading {
	if l == nil {
		return nil
	}

	var result *Heading
	for i := range l.Headings {
		h := &l.Headings[i]
		// A heading is "before" y in reading order if it's above y (Y > y)
		if h.BBox.Y > y {
			result = h
		}
	}
	return result
}

// FindHeadingsInRegion returns headings within a bounding box
func (l *HeadingLayout) FindHeadingsInRegion(bbox model.BBox) []Heading {
	if l == nil {
		return nil
	}

	var result []Heading
	for _, h := range l.Headings {
		if h.BBox.X+h.BBox.Width > bbox.X &&
			h.BBox.X < bbox.X+bbox.Width &&
			h.BBox.Y+h.BBox.Height > bbox.Y &&
			h.BBox.Y < bbox.Y+bbox.Height {
			result = append(result, h)
		}
	}
	return result
}

// Heading methods

// IsTopLevel returns true if this is an H1 heading
func (h *Heading) IsTopLevel() bool {
	if h == nil {
		return false
	}
	return h.Level == HeadingLevel1
}

// WordCount returns the word count of the heading text
func (h *Heading) WordCount() int {
	if h == nil {
		return 0
	}
	return len(strings.Fields(h.Text))
}

// GetCleanText returns the heading text without number prefix
func (h *Heading) GetCleanText() string {
	if h == nil {
		return ""
	}

	text := h.Text
	if h.IsNumbered && h.NumberPrefix != "" {
		text = strings.TrimPrefix(text, h.NumberPrefix)
		text = strings.TrimSpace(text)
	}
	return text
}

// GetAnchorID returns a URL-safe anchor ID for this heading
func (h *Heading) GetAnchorID() string {
	if h == nil {
		return ""
	}

	text := h.GetCleanText()
	text = strings.ToLower(text)

	// Replace spaces with hyphens
	text = strings.ReplaceAll(text, " ", "-")

	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	// Clean up multiple hyphens
	id := result.String()
	for strings.Contains(id, "--") {
		id = strings.ReplaceAll(id, "--", "-")
	}
	id = strings.Trim(id, "-")

	return id
}

// ToMarkdown returns the heading as a markdown heading
func (h *Heading) ToMarkdown() string {
	if h == nil {
		return ""
	}

	prefix := strings.Repeat("#", int(h.Level))
	return prefix + " " + h.Text
}

// ContainsPoint returns true if the point is within the heading's bounding box
func (h *Heading) ContainsPoint(x, y float64) bool {
	if h == nil {
		return false
	}
	return x >= h.BBox.X && x <= h.BBox.X+h.BBox.Width &&
		y >= h.BBox.Y && y <= h.BBox.Y+h.BBox.Height
}
