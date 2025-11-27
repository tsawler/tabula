// Package layout provides document layout analysis including paragraph detection,
// line detection, block detection, and structural element identification.
package layout

import (
	"strings"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// ParagraphStyle represents the detected style of a paragraph
type ParagraphStyle int

const (
	StyleNormal ParagraphStyle = iota
	StyleHeading
	StyleBlockQuote
	StyleListItem
	StyleCode
	StyleCaption
)

// String returns a string representation of the paragraph style
func (s ParagraphStyle) String() string {
	switch s {
	case StyleHeading:
		return "heading"
	case StyleBlockQuote:
		return "blockquote"
	case StyleListItem:
		return "list-item"
	case StyleCode:
		return "code"
	case StyleCaption:
		return "caption"
	default:
		return "normal"
	}
}

// Paragraph represents a logical paragraph of text
type Paragraph struct {
	// BBox is the bounding box of the paragraph
	BBox model.BBox

	// Lines are the text lines in this paragraph (in reading order)
	Lines []Line

	// Text is the assembled text content
	Text string

	// Index is the paragraph's position in reading order (0-based)
	Index int

	// Style is the detected paragraph style
	Style ParagraphStyle

	// Alignment is the dominant alignment of the paragraph
	Alignment LineAlignment

	// FirstLineIndent is the indentation of the first line relative to subsequent lines
	FirstLineIndent float64

	// LeftMargin is the left margin of the paragraph body
	LeftMargin float64

	// AverageFontSize is the average font size across all lines
	AverageFontSize float64

	// LineSpacing is the average spacing between lines within this paragraph
	LineSpacing float64

	// SpacingBefore is the space before this paragraph
	SpacingBefore float64

	// SpacingAfter is the space after this paragraph
	SpacingAfter float64
}

// ParagraphLayout represents the detected paragraph structure of a page
type ParagraphLayout struct {
	// Paragraphs are the detected paragraphs (in reading order)
	Paragraphs []Paragraph

	// PageWidth is the width of the page
	PageWidth float64

	// PageHeight is the height of the page
	PageHeight float64

	// AverageParagraphSpacing is the average spacing between paragraphs
	AverageParagraphSpacing float64

	// Config is the configuration used for detection
	Config ParagraphConfig
}

// ParagraphConfig holds configuration for paragraph detection
type ParagraphConfig struct {
	// SpacingThreshold is the multiplier for line spacing to detect paragraph breaks
	// If spacing > avgLineSpacing * SpacingThreshold, it's a paragraph break
	// Default: 1.5
	SpacingThreshold float64

	// IndentThreshold is the minimum indentation to consider as first-line indent
	// Default: 15 points
	IndentThreshold float64

	// HeadingFontSizeRatio is the font size ratio to consider as heading
	// If fontSize > avgFontSize * HeadingFontSizeRatio, it's a heading
	// Default: 1.2 (20% larger)
	HeadingFontSizeRatio float64

	// MinParagraphLines is the minimum number of lines for a paragraph
	// Default: 1
	MinParagraphLines int

	// BlockQuoteIndent is the minimum indentation to consider as block quote
	// Default: 30 points
	BlockQuoteIndent float64

	// ListItemPatterns are regex patterns that indicate list items
	// Default: bullet points, numbers, letters
	ListItemPatterns []string
}

// DefaultParagraphConfig returns sensible default configuration
func DefaultParagraphConfig() ParagraphConfig {
	return ParagraphConfig{
		SpacingThreshold:     1.5,
		IndentThreshold:      15.0,
		HeadingFontSizeRatio: 1.2,
		MinParagraphLines:    1,
		BlockQuoteIndent:     30.0,
		ListItemPatterns: []string{
			`^[\•\-\*\→\►\◦\‣]\s`, // Bullet points
			`^\d+[\.\)]\s`,        // Numbered: 1. or 1)
			`^[a-zA-Z][\.\)]\s`,   // Lettered: a. or a)
			`^[ivxIVX]+[\.\)]\s`,  // Roman numerals
		},
	}
}

// ParagraphDetector detects paragraphs from lines
type ParagraphDetector struct {
	config ParagraphConfig
}

// NewParagraphDetector creates a new paragraph detector with default configuration
func NewParagraphDetector() *ParagraphDetector {
	return &ParagraphDetector{
		config: DefaultParagraphConfig(),
	}
}

// NewParagraphDetectorWithConfig creates a paragraph detector with custom configuration
func NewParagraphDetectorWithConfig(config ParagraphConfig) *ParagraphDetector {
	return &ParagraphDetector{
		config: config,
	}
}

// Detect analyzes lines and groups them into paragraphs
func (d *ParagraphDetector) Detect(lines []Line, pageWidth, pageHeight float64) *ParagraphLayout {
	if len(lines) == 0 {
		return &ParagraphLayout{
			Paragraphs: nil,
			PageWidth:  pageWidth,
			PageHeight: pageHeight,
			Config:     d.config,
		}
	}

	// Calculate baseline metrics
	avgLineSpacing := d.calculateAverageLineSpacing(lines)
	avgFontSize := d.calculateAverageFontSize(lines)
	leftMargin := d.detectLeftMargin(lines)

	// Group lines into paragraphs
	paragraphs := d.groupIntoParagraphs(lines, avgLineSpacing, avgFontSize, leftMargin)

	// Detect paragraph styles
	d.detectStyles(paragraphs, avgFontSize, leftMargin)

	// Calculate spacing between paragraphs
	d.calculateParagraphSpacing(paragraphs)

	// Calculate average paragraph spacing
	avgParagraphSpacing := d.calculateAverageParagraphSpacing(paragraphs)

	return &ParagraphLayout{
		Paragraphs:              paragraphs,
		PageWidth:               pageWidth,
		PageHeight:              pageHeight,
		AverageParagraphSpacing: avgParagraphSpacing,
		Config:                  d.config,
	}
}

// DetectFromFragments is a convenience method that first detects lines, then paragraphs
func (d *ParagraphDetector) DetectFromFragments(fragments []text.TextFragment, pageWidth, pageHeight float64) *ParagraphLayout {
	lineDetector := NewLineDetector()
	lineLayout := lineDetector.Detect(fragments, pageWidth, pageHeight)
	return d.Detect(lineLayout.Lines, pageWidth, pageHeight)
}

// calculateAverageLineSpacing calculates the average spacing between lines
func (d *ParagraphDetector) calculateAverageLineSpacing(lines []Line) float64 {
	if len(lines) < 2 {
		return 0
	}

	totalSpacing := 0.0
	count := 0

	for i := 1; i < len(lines); i++ {
		spacing := lines[i].SpacingBefore
		if spacing > 0 {
			totalSpacing += spacing
			count++
		}
	}

	if count == 0 {
		return 12.0 // Default line spacing
	}

	return totalSpacing / float64(count)
}

// calculateAverageFontSize calculates the average font size across all lines
func (d *ParagraphDetector) calculateAverageFontSize(lines []Line) float64 {
	if len(lines) == 0 {
		return 12.0
	}

	totalSize := 0.0
	for _, line := range lines {
		totalSize += line.AverageFontSize
	}

	return totalSize / float64(len(lines))
}

// detectLeftMargin detects the most common left margin
func (d *ParagraphDetector) detectLeftMargin(lines []Line) float64 {
	if len(lines) == 0 {
		return 0
	}

	// Count occurrences of each left position (with tolerance)
	marginCounts := make(map[int]int)
	tolerance := 5.0

	for _, line := range lines {
		bucket := int(line.BBox.X / tolerance)
		marginCounts[bucket]++
	}

	// Find most common
	maxCount := 0
	mostCommonBucket := 0
	for bucket, count := range marginCounts {
		if count > maxCount {
			maxCount = count
			mostCommonBucket = bucket
		}
	}

	return float64(mostCommonBucket) * tolerance
}

// groupIntoParagraphs groups lines into paragraphs
func (d *ParagraphDetector) groupIntoParagraphs(lines []Line, avgLineSpacing, avgFontSize, leftMargin float64) []Paragraph {
	if len(lines) == 0 {
		return nil
	}

	spacingThreshold := avgLineSpacing * d.config.SpacingThreshold
	var paragraphs []Paragraph
	var currentLines []Line

	for i, line := range lines {
		if len(currentLines) == 0 {
			currentLines = append(currentLines, line)
			continue
		}

		// Check if this line starts a new paragraph
		isNewParagraph := false

		// 1. Large vertical spacing
		if line.SpacingBefore > spacingThreshold {
			isNewParagraph = true
		}

		// 2. Significant font size change
		prevLine := currentLines[len(currentLines)-1]
		fontSizeRatio := line.AverageFontSize / prevLine.AverageFontSize
		if fontSizeRatio > d.config.HeadingFontSizeRatio || fontSizeRatio < 1/d.config.HeadingFontSizeRatio {
			isNewParagraph = true
		}

		// 3. Significant alignment change
		// Don't break on minor alignment differences (left/justified are often mixed in body text)
		if isSignificantAlignmentChange(prevLine.Alignment, line.Alignment) {
			isNewParagraph = true
		}

		// 4. First-line indentation pattern (new paragraph starts with indent)
		lineIndent := line.BBox.X - leftMargin
		prevIndent := prevLine.BBox.X - leftMargin
		if lineIndent > d.config.IndentThreshold && prevIndent <= d.config.IndentThreshold {
			isNewParagraph = true
		}

		// 5. List item pattern at start of line
		if d.isListItem(line.Text) && !d.isListItem(prevLine.Text) {
			isNewParagraph = true
		}

		// 6. Short previous line (likely end of paragraph) followed by normal indentation
		if prevLine.BBox.Width < currentLines[0].BBox.Width*0.7 && i < len(lines)-1 {
			// Previous line is significantly shorter - might be end of paragraph
			if lineIndent <= d.config.IndentThreshold || lineIndent > d.config.IndentThreshold {
				isNewParagraph = true
			}
		}

		if isNewParagraph {
			// Finalize current paragraph
			para := d.buildParagraph(currentLines, len(paragraphs), leftMargin)
			paragraphs = append(paragraphs, para)
			currentLines = []Line{line}
		} else {
			currentLines = append(currentLines, line)
		}
	}

	// Don't forget the last paragraph
	if len(currentLines) > 0 {
		para := d.buildParagraph(currentLines, len(paragraphs), leftMargin)
		paragraphs = append(paragraphs, para)
	}

	return paragraphs
}

// buildParagraph creates a Paragraph from a group of lines
func (d *ParagraphDetector) buildParagraph(lines []Line, index int, leftMargin float64) Paragraph {
	if len(lines) == 0 {
		return Paragraph{Index: index}
	}

	para := Paragraph{
		Lines: lines,
		Index: index,
	}

	// Calculate bounding box
	para.BBox = d.calculateParagraphBBox(lines)

	// Assemble text
	para.Text = d.assembleParagraphText(lines)

	// Calculate first line indent
	if len(lines) > 1 {
		firstLineX := lines[0].BBox.X
		// Find minimum X of subsequent lines
		minSubsequentX := lines[1].BBox.X
		for _, line := range lines[2:] {
			if line.BBox.X < minSubsequentX {
				minSubsequentX = line.BBox.X
			}
		}
		para.FirstLineIndent = firstLineX - minSubsequentX
	}

	// Set left margin
	para.LeftMargin = para.BBox.X

	// Calculate average font size
	totalFontSize := 0.0
	for _, line := range lines {
		totalFontSize += line.AverageFontSize
	}
	para.AverageFontSize = totalFontSize / float64(len(lines))

	// Detect dominant alignment
	para.Alignment = d.detectDominantAlignment(lines)

	// Calculate line spacing within paragraph
	if len(lines) > 1 {
		totalSpacing := 0.0
		count := 0
		for i := 1; i < len(lines); i++ {
			if lines[i].SpacingBefore > 0 {
				totalSpacing += lines[i].SpacingBefore
				count++
			}
		}
		if count > 0 {
			para.LineSpacing = totalSpacing / float64(count)
		}
	}

	return para
}

// calculateParagraphBBox calculates the bounding box of a paragraph
func (d *ParagraphDetector) calculateParagraphBBox(lines []Line) model.BBox {
	if len(lines) == 0 {
		return model.BBox{}
	}

	minX := lines[0].BBox.X
	minY := lines[0].BBox.Y
	maxX := lines[0].BBox.X + lines[0].BBox.Width
	maxY := lines[0].BBox.Y + lines[0].BBox.Height

	for _, line := range lines[1:] {
		if line.BBox.X < minX {
			minX = line.BBox.X
		}
		if line.BBox.Y < minY {
			minY = line.BBox.Y
		}
		if line.BBox.X+line.BBox.Width > maxX {
			maxX = line.BBox.X + line.BBox.Width
		}
		if line.BBox.Y+line.BBox.Height > maxY {
			maxY = line.BBox.Y + line.BBox.Height
		}
	}

	return model.BBox{
		X:      minX,
		Y:      minY,
		Width:  maxX - minX,
		Height: maxY - minY,
	}
}

// assembleParagraphText assembles text from paragraph lines
func (d *ParagraphDetector) assembleParagraphText(lines []Line) string {
	if len(lines) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, line := range lines {
		sb.WriteString(line.Text)
		if i < len(lines)-1 {
			// Add space between lines (paragraph text flows)
			if !strings.HasSuffix(line.Text, "-") {
				sb.WriteString(" ")
			}
		}
	}

	return sb.String()
}

// detectDominantAlignment finds the most common alignment in lines
func (d *ParagraphDetector) detectDominantAlignment(lines []Line) LineAlignment {
	if len(lines) == 0 {
		return AlignUnknown
	}

	counts := make(map[LineAlignment]int)
	for _, line := range lines {
		counts[line.Alignment]++
	}

	maxCount := 0
	dominant := AlignUnknown
	for align, count := range counts {
		if count > maxCount {
			maxCount = count
			dominant = align
		}
	}

	return dominant
}

// isSignificantAlignmentChange returns true if the alignment change is significant enough
// to indicate a paragraph break. Minor changes (left/justified) are ignored since they
// commonly occur within the same paragraph (e.g., short end-of-paragraph lines in justified text)
func isSignificantAlignmentChange(prev, curr LineAlignment) bool {
	// Unknown alignments don't trigger breaks
	if prev == AlignUnknown || curr == AlignUnknown {
		return false
	}

	// Same alignment - no change
	if prev == curr {
		return false
	}

	// Left and Justified are compatible (common in body text)
	if (prev == AlignLeft || prev == AlignJustified) && (curr == AlignLeft || curr == AlignJustified) {
		return false
	}

	// Center to/from anything else is significant
	if prev == AlignCenter || curr == AlignCenter {
		return true
	}

	// Right to/from anything else is significant
	if prev == AlignRight || curr == AlignRight {
		return true
	}

	return true
}

// isListItem checks if text starts with a list item pattern
func (d *ParagraphDetector) isListItem(text string) bool {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return false
	}

	// Check common bullet characters
	firstRune := []rune(text)[0]
	bullets := []rune{'•', '-', '*', '→', '►', '◦', '‣', '○', '●', '■', '□', '▪', '▫'}
	for _, b := range bullets {
		if firstRune == b {
			return true
		}
	}

	// Check numbered patterns: 1. 1) a. a) i. i)
	if len(text) >= 2 {
		// Digit followed by . or )
		if text[0] >= '0' && text[0] <= '9' {
			for i := 1; i < len(text) && i < 4; i++ {
				if text[i] == '.' || text[i] == ')' {
					return true
				}
				if text[i] < '0' || text[i] > '9' {
					break
				}
			}
		}
		// Letter followed by . or )
		if (text[0] >= 'a' && text[0] <= 'z') || (text[0] >= 'A' && text[0] <= 'Z') {
			if len(text) >= 2 && (text[1] == '.' || text[1] == ')') {
				return true
			}
		}
	}

	return false
}

// detectStyles detects the style of each paragraph
func (d *ParagraphDetector) detectStyles(paragraphs []Paragraph, avgFontSize, leftMargin float64) {
	for i := range paragraphs {
		para := &paragraphs[i]

		// Check for heading (larger font, short, centered or left-aligned)
		if para.AverageFontSize > avgFontSize*d.config.HeadingFontSizeRatio {
			if len(para.Lines) <= 3 {
				para.Style = StyleHeading
				continue
			}
		}

		// Check for list item
		if len(para.Lines) > 0 && d.isListItem(para.Lines[0].Text) {
			para.Style = StyleListItem
			continue
		}

		// Check for block quote (indented from both margins)
		if para.LeftMargin > leftMargin+d.config.BlockQuoteIndent {
			para.Style = StyleBlockQuote
			continue
		}

		// Check for caption (smaller font, short, often centered)
		if para.AverageFontSize < avgFontSize*0.9 && len(para.Lines) <= 2 {
			if para.Alignment == AlignCenter {
				para.Style = StyleCaption
				continue
			}
		}

		// Default to normal
		para.Style = StyleNormal
	}
}

// calculateParagraphSpacing calculates spacing between paragraphs
func (d *ParagraphDetector) calculateParagraphSpacing(paragraphs []Paragraph) {
	for i := range paragraphs {
		if i > 0 {
			// Space from previous paragraph bottom to this paragraph top
			prevBottom := paragraphs[i-1].BBox.Y
			thisTop := paragraphs[i].BBox.Y + paragraphs[i].BBox.Height
			spacing := prevBottom - thisTop
			paragraphs[i].SpacingBefore = spacing
			paragraphs[i-1].SpacingAfter = spacing
		}
	}
}

// calculateAverageParagraphSpacing calculates average spacing between paragraphs
func (d *ParagraphDetector) calculateAverageParagraphSpacing(paragraphs []Paragraph) float64 {
	if len(paragraphs) < 2 {
		return 0
	}

	totalSpacing := 0.0
	count := 0
	for _, para := range paragraphs {
		if para.SpacingBefore > 0 {
			totalSpacing += para.SpacingBefore
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalSpacing / float64(count)
}

// ParagraphLayout methods

// ParagraphCount returns the number of detected paragraphs
func (l *ParagraphLayout) ParagraphCount() int {
	if l == nil {
		return 0
	}
	return len(l.Paragraphs)
}

// GetParagraph returns a specific paragraph by index
func (l *ParagraphLayout) GetParagraph(index int) *Paragraph {
	if l == nil || index < 0 || index >= len(l.Paragraphs) {
		return nil
	}
	return &l.Paragraphs[index]
}

// GetText returns all text with paragraph breaks
func (l *ParagraphLayout) GetText() string {
	if l == nil || len(l.Paragraphs) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, para := range l.Paragraphs {
		sb.WriteString(para.Text)
		if i < len(l.Paragraphs)-1 {
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

// GetParagraphsByStyle returns paragraphs with a specific style
func (l *ParagraphLayout) GetParagraphsByStyle(style ParagraphStyle) []Paragraph {
	if l == nil {
		return nil
	}

	var result []Paragraph
	for _, para := range l.Paragraphs {
		if para.Style == style {
			result = append(result, para)
		}
	}
	return result
}

// GetHeadings returns all paragraphs detected as headings
func (l *ParagraphLayout) GetHeadings() []Paragraph {
	return l.GetParagraphsByStyle(StyleHeading)
}

// GetListItems returns all paragraphs detected as list items
func (l *ParagraphLayout) GetListItems() []Paragraph {
	return l.GetParagraphsByStyle(StyleListItem)
}

// FindParagraphsInRegion returns paragraphs that fall within a bounding box
func (l *ParagraphLayout) FindParagraphsInRegion(bbox model.BBox) []Paragraph {
	if l == nil {
		return nil
	}

	var result []Paragraph
	for _, para := range l.Paragraphs {
		// Check if paragraph overlaps with region
		if para.BBox.X+para.BBox.Width > bbox.X &&
			para.BBox.X < bbox.X+bbox.Width &&
			para.BBox.Y+para.BBox.Height > bbox.Y &&
			para.BBox.Y < bbox.Y+bbox.Height {
			result = append(result, para)
		}
	}
	return result
}

// Paragraph methods

// LineCount returns the number of lines in this paragraph
func (p *Paragraph) LineCount() int {
	if p == nil {
		return 0
	}
	return len(p.Lines)
}

// WordCount returns an approximate word count for the paragraph
func (p *Paragraph) WordCount() int {
	if p == nil || p.Text == "" {
		return 0
	}
	return len(strings.Fields(p.Text))
}

// IsHeading returns true if this paragraph is styled as a heading
func (p *Paragraph) IsHeading() bool {
	if p == nil {
		return false
	}
	return p.Style == StyleHeading
}

// IsListItem returns true if this paragraph is styled as a list item
func (p *Paragraph) IsListItem() bool {
	if p == nil {
		return false
	}
	return p.Style == StyleListItem
}

// IsBlockQuote returns true if this paragraph is styled as a block quote
func (p *Paragraph) IsBlockQuote() bool {
	if p == nil {
		return false
	}
	return p.Style == StyleBlockQuote
}

// HasFirstLineIndent returns true if the paragraph has first-line indentation
func (p *Paragraph) HasFirstLineIndent() bool {
	if p == nil {
		return false
	}
	return p.FirstLineIndent > 5.0 // Small tolerance
}

// ContainsPoint returns true if the point is within the paragraph's bounding box
func (p *Paragraph) ContainsPoint(x, y float64) bool {
	if p == nil {
		return false
	}
	return x >= p.BBox.X && x <= p.BBox.X+p.BBox.Width &&
		y >= p.BBox.Y && y <= p.BBox.Y+p.BBox.Height
}

// GetFirstLine returns the first line of the paragraph
func (p *Paragraph) GetFirstLine() *Line {
	if p == nil || len(p.Lines) == 0 {
		return nil
	}
	return &p.Lines[0]
}

// GetLastLine returns the last line of the paragraph
func (p *Paragraph) GetLastLine() *Line {
	if p == nil || len(p.Lines) == 0 {
		return nil
	}
	return &p.Lines[len(p.Lines)-1]
}
