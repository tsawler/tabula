// Package layout provides document layout analysis including reading order detection,
// which determines the correct sequence for reading text in complex layouts.
package layout

import (
	"sort"

	"github.com/tsawler/tabula/text"
)

// ReadingDirection indicates the primary reading direction of a document
type ReadingDirection int

const (
	// LeftToRight is the default for most Western languages
	LeftToRight ReadingDirection = iota
	// RightToLeft is used for Arabic, Hebrew, etc.
	RightToLeft
	// TopToBottom is used for traditional Chinese/Japanese
	TopToBottom
)

// String returns a string representation of the reading direction
func (d ReadingDirection) String() string {
	switch d {
	case RightToLeft:
		return "rtl"
	case TopToBottom:
		return "ttb"
	default:
		return "ltr"
	}
}

// ReadingOrderConfig holds configuration for reading order detection
type ReadingOrderConfig struct {
	// Direction is the primary reading direction
	Direction ReadingDirection

	// ColumnConfig is the configuration for column detection
	ColumnConfig ColumnConfig

	// LineConfig is the configuration for line detection
	LineConfig LineConfig

	// PreferColumnOrder when true, reads entire columns before moving to next
	// When false, may interleave if content appears to flow across columns
	PreferColumnOrder bool

	// SpanningThreshold is the minimum width ratio for content to be considered spanning
	// Default: 0.7 (content spanning 70%+ of page width is considered spanning)
	SpanningThreshold float64

	// InvertedY indicates that Y coordinates increase downward (Y=0 at top)
	// rather than the standard PDF convention where Y increases upward (Y=0 at bottom)
	// When true, lower Y values are at the top of the page
	// Default: false (auto-detect based on content)
	InvertedY *bool
}

// DefaultReadingOrderConfig returns sensible default configuration
func DefaultReadingOrderConfig() ReadingOrderConfig {
	return ReadingOrderConfig{
		Direction:         LeftToRight,
		ColumnConfig:      DefaultColumnConfig(),
		LineConfig:        DefaultLineConfig(),
		PreferColumnOrder: true,
		SpanningThreshold: 0.7,
	}
}

// ReadingOrderResult holds the result of reading order analysis
type ReadingOrderResult struct {
	// Fragments in reading order
	Fragments []text.TextFragment

	// Lines in reading order
	Lines []Line

	// Sections represent logical sections of content (spanning + column content)
	Sections []ReadingSection

	// Direction is the detected or configured reading direction
	Direction ReadingDirection

	// ColumnCount is the number of columns detected
	ColumnCount int

	// PageWidth and PageHeight
	PageWidth  float64
	PageHeight float64
}

// ReadingSection represents a section of content in reading order
type ReadingSection struct {
	// Type indicates what kind of section this is
	Type SectionType

	// Lines in this section (in reading order)
	Lines []Line

	// Fragments in this section (in reading order)
	Fragments []text.TextFragment

	// ColumnIndex for column sections (-1 for spanning)
	ColumnIndex int

	// BBox is the bounding box of this section
	BBox struct {
		X, Y, Width, Height float64
	}
}

// SectionType indicates the type of reading section
type SectionType int

const (
	SectionSpanning SectionType = iota // Full-width content (titles, headers)
	SectionColumn                      // Column content
)

// String returns a string representation of the section type
func (t SectionType) String() string {
	if t == SectionSpanning {
		return "spanning"
	}
	return "column"
}

// ReadingOrderDetector determines the correct reading order for page content
type ReadingOrderDetector struct {
	config ReadingOrderConfig
}

// NewReadingOrderDetector creates a new reading order detector with default configuration
func NewReadingOrderDetector() *ReadingOrderDetector {
	return &ReadingOrderDetector{
		config: DefaultReadingOrderConfig(),
	}
}

// NewReadingOrderDetectorWithConfig creates a reading order detector with custom configuration
func NewReadingOrderDetectorWithConfig(config ReadingOrderConfig) *ReadingOrderDetector {
	return &ReadingOrderDetector{
		config: config,
	}
}

// Detect analyzes fragments and returns them in proper reading order
func (d *ReadingOrderDetector) Detect(fragments []text.TextFragment, pageWidth, pageHeight float64) *ReadingOrderResult {
	if len(fragments) == 0 {
		return &ReadingOrderResult{
			PageWidth:  pageWidth,
			PageHeight: pageHeight,
			Direction:  d.config.Direction,
		}
	}

	// Step 1: Detect columns
	columnDetector := NewColumnDetectorWithConfig(d.config.ColumnConfig)
	columnLayout := columnDetector.Detect(fragments, pageWidth, pageHeight)

	// Step 2: Auto-detect reading direction if needed
	direction := d.config.Direction
	if direction == LeftToRight {
		direction = d.detectReadingDirection(fragments)
	}

	// Step 2b: Detect if Y coordinates are inverted
	invertedY := d.detectInvertedY(fragments, pageHeight)

	// Step 3: Build sections from column layout
	sections := d.buildSections(columnLayout, pageWidth, pageHeight, direction, invertedY)

	// Step 4: Order sections by reading direction
	d.orderSections(sections, direction, invertedY)

	// Step 5: Build final ordered lists
	var orderedFragments []text.TextFragment
	var orderedLines []Line

	for i := range sections {
		orderedFragments = append(orderedFragments, sections[i].Fragments...)
		orderedLines = append(orderedLines, sections[i].Lines...)
	}

	return &ReadingOrderResult{
		Fragments:   orderedFragments,
		Lines:       orderedLines,
		Sections:    sections,
		Direction:   direction,
		ColumnCount: columnLayout.ColumnCount(),
		PageWidth:   pageWidth,
		PageHeight:  pageHeight,
	}
}

// detectInvertedY determines if the PDF uses inverted Y coordinates
// Returns true if Y=0 is at the top and Y increases downward
func (d *ReadingOrderDetector) detectInvertedY(fragments []text.TextFragment, pageHeight float64) bool {
	// If explicitly configured, use that
	if d.config.InvertedY != nil {
		return *d.config.InvertedY
	}

	if len(fragments) == 0 {
		return false
	}

	// Find actual Y coordinate range
	minY, maxY := fragments[0].Y, fragments[0].Y
	for _, f := range fragments {
		if f.Y < minY {
			minY = f.Y
		}
		if f.Y > maxY {
			maxY = f.Y
		}
	}

	// Use the relationship between Y coordinates and page height to determine
	// the coordinate system. This is more reliable than looking at stream order
	// since PDFs often have content rendered in arbitrary order (e.g., footers first).
	//
	// In standard PDF coordinates (Y=0 at bottom):
	//   - Content near the top of the page has Y values close to pageHeight
	//   - maxY should be a significant fraction of pageHeight (typically 70-95%)
	//
	// In inverted coordinates (Y=0 at top):
	//   - Content near the top of the page has Y values close to 0
	//   - maxY would be much smaller relative to typical page heights
	//
	// The key insight: in standard coordinates, maxY approaches pageHeight,
	// while in inverted coordinates, maxY is bounded by content position.

	if pageHeight > 0 {
		// If maxY is more than half of pageHeight, assume standard coordinates.
		// This works because page content typically spans most of the page height,
		// so in standard coords (Y=0 at bottom), maxY will be close to pageHeight.
		if maxY > pageHeight*0.5 {
			return false // Standard coordinates
		}

		// If maxY is less than 30% of pageHeight and minY is close to 0,
		// this strongly suggests inverted coordinates where content starts
		// at Y=0 (top) and increases downward.
		if maxY < pageHeight*0.3 && minY < pageHeight*0.1 {
			return true // Inverted coordinates
		}
	}

	// Fallback: use content range heuristic
	// If Y values span a range starting near 0, it's likely inverted
	yRange := maxY - minY
	if yRange > 0 && minY < yRange*0.1 {
		// Content starts very close to Y=0, suggesting inverted coords
		// where the top of the page is at Y=0
		return maxY < pageHeight*0.5
	}

	// Default to standard PDF coordinates per PDF specification
	return false
}

// detectReadingDirection analyzes fragments to detect the primary reading direction
func (d *ReadingOrderDetector) detectReadingDirection(fragments []text.TextFragment) ReadingDirection {
	rtlCount := 0
	ltrCount := 0

	for _, frag := range fragments {
		switch frag.Direction {
		case text.RTL:
			rtlCount++
		case text.LTR:
			ltrCount++
		}
	}

	// If majority is RTL, use RTL direction
	if rtlCount > ltrCount {
		return RightToLeft
	}
	return LeftToRight
}

// buildSections creates reading sections from column layout
func (d *ReadingOrderDetector) buildSections(columnLayout *ColumnLayout, pageWidth, pageHeight float64, direction ReadingDirection, invertedY bool) []ReadingSection {
	var sections []ReadingSection

	// Add spanning content as a section (if any)
	if len(columnLayout.SpanningFragments) > 0 {
		spanningSection := d.buildSpanningSection(columnLayout.SpanningFragments, pageWidth, pageHeight, invertedY)
		sections = append(sections, spanningSection)
	}

	// Add each column as a section
	for i, col := range columnLayout.Columns {
		if len(col.Fragments) == 0 {
			continue
		}

		colSection := d.buildColumnSection(col, i, pageHeight, invertedY)
		sections = append(sections, colSection)
	}

	return sections
}

// buildSpanningSection creates a section for spanning (full-width) content
func (d *ReadingOrderDetector) buildSpanningSection(fragments []text.TextFragment, pageWidth, pageHeight float64, invertedY bool) ReadingSection {
	// Detect lines in spanning content
	lineDetector := NewLineDetectorWithConfig(d.config.LineConfig)
	lineLayout := lineDetector.Detect(fragments, pageWidth, pageHeight)

	// Reorder lines based on Y coordinate direction
	lines := reorderLinesByY(lineLayout.Lines, invertedY)

	// Calculate bounding box
	bbox := fragmentsBBox(fragments)

	return ReadingSection{
		Type:        SectionSpanning,
		Lines:       lines,
		Fragments:   fragments,
		ColumnIndex: -1,
		BBox: struct {
			X, Y, Width, Height float64
		}{bbox.X, bbox.Y, bbox.Width, bbox.Height},
	}
}

// buildColumnSection creates a section for column content
func (d *ReadingOrderDetector) buildColumnSection(col Column, colIndex int, pageHeight float64, invertedY bool) ReadingSection {
	// Detect lines within this column
	lineDetector := NewLineDetectorWithConfig(d.config.LineConfig)
	lineLayout := lineDetector.Detect(col.Fragments, col.BBox.Width, pageHeight)

	// Reorder lines based on Y coordinate direction
	lines := reorderLinesByY(lineLayout.Lines, invertedY)

	// Normalize line X positions relative to column left edge for alignment calculation
	normalizeLineXPositions(lines, col.BBox.X)

	// Recalculate alignment based on column width
	recalculateLineAlignment(lines, col.BBox.Width)

	return ReadingSection{
		Type:        SectionColumn,
		Lines:       lines,
		Fragments:   col.Fragments,
		ColumnIndex: colIndex,
		BBox: struct {
			X, Y, Width, Height float64
		}{col.BBox.X, col.BBox.Y, col.BBox.Width, col.BBox.Height},
	}
}

// normalizeLineXPositions adjusts line X positions relative to section left edge
func normalizeLineXPositions(lines []Line, sectionLeft float64) {
	for i := range lines {
		lines[i].BBox.X -= sectionLeft
		// Also update indentation relative to section
		if lines[i].Indentation > 0 {
			lines[i].Indentation -= sectionLeft
			if lines[i].Indentation < 0 {
				lines[i].Indentation = 0
			}
		}
	}
}

// reorderLinesByY sorts lines based on vertical position accounting for coordinate system
// and recalculates spacing values based on the new order
func reorderLinesByY(lines []Line, invertedY bool) []Line {
	if len(lines) <= 1 {
		return lines
	}

	// Make a copy to avoid modifying original
	result := make([]Line, len(lines))
	copy(result, lines)

	sort.SliceStable(result, func(i, j int) bool {
		if invertedY {
			// Inverted: lower Y = top of page, so sort ascending
			return result[i].BBox.Y < result[j].BBox.Y
		}
		// Standard: higher Y = top of page, so sort descending
		return result[i].BBox.Y > result[j].BBox.Y
	})

	// Recalculate spacing based on new order
	for i := range result {
		if i == 0 {
			result[i].SpacingBefore = 0
		} else {
			// Calculate spacing from previous line
			prevLine := result[i-1]
			if invertedY {
				// Inverted: spacing = current Y - (prev Y + prev Height)
				result[i].SpacingBefore = result[i].BBox.Y - (prevLine.BBox.Y + prevLine.BBox.Height)
			} else {
				// Standard: spacing = prev Y - (current Y + current Height)
				result[i].SpacingBefore = prevLine.BBox.Y - (result[i].BBox.Y + result[i].BBox.Height)
			}
			// Clamp to non-negative
			if result[i].SpacingBefore < 0 {
				result[i].SpacingBefore = 0
			}
		}

		// Update SpacingAfter for previous line
		if i > 0 {
			result[i-1].SpacingAfter = result[i].SpacingBefore
		}
	}

	// Last line has no spacing after
	if len(result) > 0 {
		result[len(result)-1].SpacingAfter = 0
	}

	return result
}

// recalculateLineAlignment sets alignment for lines in a column section
// Since alignment detection is unreliable after column reordering, we use a simple approach:
// - Full-width lines are justified
// - Short lines are left-aligned
func recalculateLineAlignment(lines []Line, sectionWidth float64) {
	if len(lines) == 0 || sectionWidth <= 0 {
		return
	}

	// Find the maximum line width to use as reference
	maxWidth := 0.0
	for _, line := range lines {
		if line.BBox.Width > maxWidth {
			maxWidth = line.BBox.Width
		}
	}

	// Reference width is the max line width (more reliable than section width)
	referenceWidth := maxWidth
	if referenceWidth <= 0 {
		referenceWidth = sectionWidth
	}

	for i := range lines {
		line := &lines[i]
		// Simple rule: full-width lines are justified, short lines are left
		if line.BBox.Width >= referenceWidth*0.85 {
			line.Alignment = AlignJustified
		} else {
			line.Alignment = AlignLeft
		}
	}
}

// orderSections orders sections based on reading direction and vertical position
func (d *ReadingOrderDetector) orderSections(sections []ReadingSection, direction ReadingDirection, invertedY bool) {
	if len(sections) <= 1 {
		return
	}

	// Sort sections:
	// 1. First by vertical position (top content first)
	// 2. Then by horizontal position based on reading direction
	sort.SliceStable(sections, func(i, j int) bool {
		si, sj := sections[i], sections[j]

		// Get vertical position for comparison
		// For inverted Y: use BBox.Y directly (lower Y = top)
		// For standard Y: use BBox.Y + Height (higher Y = top)
		var siVertical, sjVertical float64
		if invertedY {
			siVertical = si.BBox.Y
			sjVertical = sj.BBox.Y
		} else {
			siVertical = si.BBox.Y + si.BBox.Height
			sjVertical = sj.BBox.Y + sj.BBox.Height
		}

		// Spanning sections at similar Y should come before column sections
		if si.Type == SectionSpanning && sj.Type == SectionColumn {
			if invertedY {
				// For inverted Y, spanning is above if its Y is smaller
				if siVertical <= sjVertical+10 {
					return true
				}
			} else {
				// For standard Y, spanning is above if its top is greater
				if siVertical >= sjVertical-10 {
					return true
				}
			}
		}
		if sj.Type == SectionSpanning && si.Type == SectionColumn {
			if invertedY {
				if sjVertical <= siVertical+10 {
					return false
				}
			} else {
				if sjVertical >= siVertical-10 {
					return false
				}
			}
		}

		// Check if sections are at similar vertical level by looking at Y overlap
		siBottom, siTopY := si.BBox.Y, si.BBox.Y+si.BBox.Height
		sjBottom, sjTopY := sj.BBox.Y, sj.BBox.Y+sj.BBox.Height

		// Calculate vertical overlap
		overlapBottom := maxFloat64(siBottom, sjBottom)
		overlapTop := minFloat64(siTopY, sjTopY)
		overlap := overlapTop - overlapBottom

		// If there's significant overlap (sections are at same level), sort by X
		minHeight := minFloat64(si.BBox.Height, sj.BBox.Height)
		if minHeight > 0 && overlap > minHeight*0.5 {
			// Same vertical level - sort by X based on reading direction
			if direction == RightToLeft {
				return si.BBox.X > sj.BBox.X // Right to left
			}
			return si.BBox.X < sj.BBox.X // Left to right
		}

		// Different vertical levels - sort by Y (top first)
		if invertedY {
			// Inverted: lower Y = top of page
			return siVertical < sjVertical
		}
		// Standard: higher Y = top of page
		return siVertical > sjVertical
	})
}

// ReadingOrderResult methods

// GetText returns all text in reading order
func (r *ReadingOrderResult) GetText() string {
	if r == nil || len(r.Lines) == 0 {
		return ""
	}

	lineLayout := &LineLayout{
		Lines:              r.Lines,
		PageWidth:          r.PageWidth,
		PageHeight:         r.PageHeight,
		AverageLineSpacing: calculateAverageSpacing(r.Lines),
	}

	return lineLayout.GetText()
}

// GetParagraphs detects paragraphs from the ordered lines
// For multi-column documents, paragraphs are detected within each section separately
// to maintain proper spacing context, then combined in reading order
func (r *ReadingOrderResult) GetParagraphs() *ParagraphLayout {
	if r == nil || len(r.Lines) == 0 {
		return &ParagraphLayout{
			PageWidth:  r.PageWidth,
			PageHeight: r.PageHeight,
		}
	}

	// For single-column or no sections, use simple detection
	if len(r.Sections) <= 1 {
		paraDetector := NewParagraphDetector()
		return paraDetector.Detect(r.Lines, r.PageWidth, r.PageHeight)
	}

	// For multi-column, detect paragraphs within each section separately
	paraDetector := NewParagraphDetector()
	var allParagraphs []Paragraph
	var totalSpacing float64
	spacingCount := 0

	for _, section := range r.Sections {
		if len(section.Lines) == 0 {
			continue
		}

		// Detect paragraphs within this section
		sectionLayout := paraDetector.Detect(section.Lines, section.BBox.Width, r.PageHeight)

		// Add paragraphs from this section
		for i := 0; i < sectionLayout.ParagraphCount(); i++ {
			para := sectionLayout.GetParagraph(i)
			allParagraphs = append(allParagraphs, *para)
		}

		// Track spacing
		if sectionLayout.AverageParagraphSpacing > 0 {
			totalSpacing += sectionLayout.AverageParagraphSpacing
			spacingCount++
		}
	}

	avgSpacing := 0.0
	if spacingCount > 0 {
		avgSpacing = totalSpacing / float64(spacingCount)
	}

	return &ParagraphLayout{
		Paragraphs:              allParagraphs,
		PageWidth:               r.PageWidth,
		PageHeight:              r.PageHeight,
		AverageParagraphSpacing: avgSpacing,
	}
}

// IsMultiColumn returns true if multiple columns were detected
func (r *ReadingOrderResult) IsMultiColumn() bool {
	if r == nil {
		return false
	}
	return r.ColumnCount > 1
}

// GetSectionCount returns the number of reading sections
func (r *ReadingOrderResult) GetSectionCount() int {
	if r == nil {
		return 0
	}
	return len(r.Sections)
}

// Helper function
func calculateAverageSpacing(lines []Line) float64 {
	if len(lines) < 2 {
		return 0
	}

	total := 0.0
	count := 0
	for _, line := range lines {
		if line.SpacingBefore > 0 {
			total += line.SpacingBefore
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// DetectFromLines is a convenience method when you already have lines
func (d *ReadingOrderDetector) DetectFromLines(lines []Line, pageWidth, pageHeight float64) *ReadingOrderResult {
	// Convert lines back to fragments for column detection
	var fragments []text.TextFragment
	for _, line := range lines {
		fragments = append(fragments, line.Fragments...)
	}

	return d.Detect(fragments, pageWidth, pageHeight)
}

// ReorderForReading takes fragments and returns them in proper reading order
// This is a convenience function for simple use cases
func ReorderForReading(fragments []text.TextFragment, pageWidth, pageHeight float64) []text.TextFragment {
	detector := NewReadingOrderDetector()
	result := detector.Detect(fragments, pageWidth, pageHeight)
	return result.Fragments
}

// ReorderLinesForReading takes lines and returns them in proper reading order
func ReorderLinesForReading(lines []Line, pageWidth, pageHeight float64) []Line {
	detector := NewReadingOrderDetector()
	result := detector.DetectFromLines(lines, pageWidth, pageHeight)
	return result.Lines
}

// minFloat64 returns the smaller of two float64 values
func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// maxFloat64 returns the larger of two float64 values
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
