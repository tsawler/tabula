// Package layout provides document layout analysis including column detection,
// reading order determination, and structural element identification.
package layout

import (
	"sort"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// Column represents a detected text column on a page
type Column struct {
	// Bounding box of the column
	BBox model.BBox

	// Fragments contained in this column (sorted top to bottom)
	Fragments []text.TextFragment

	// Index of the column (0-based, left to right)
	Index int
}

// ColumnLayout represents the detected column structure of a page
type ColumnLayout struct {
	// Detected columns (sorted left to right)
	Columns []Column

	// SpanningFragments are fragments that span across column gaps
	// (e.g., centered titles, full-width headers)
	// These are excluded from column content and stored separately
	SpanningFragments []text.TextFragment

	// Page dimensions
	PageWidth  float64
	PageHeight float64

	// Configuration used for detection
	Config ColumnConfig
}

// ColumnConfig holds configuration for column detection
type ColumnConfig struct {
	// MinColumnWidth is the minimum width for a region to be considered a column
	// Default: 50 points
	MinColumnWidth float64

	// MinGapWidth is the minimum whitespace gap to consider as column separator
	// Default: 20 points
	MinGapWidth float64

	// MinGapHeight is the minimum vertical extent of a gap to be significant
	// As a ratio of page height (0.0 to 1.0)
	// Default: 0.5 (50% of page height)
	MinGapHeightRatio float64

	// MaxColumns is the maximum number of columns to detect
	// Default: 6
	MaxColumns int

	// MergeThreshold is the maximum X distance between fragments to consider them same column
	// Default: 10 points
	MergeThreshold float64

	// SpanningThreshold is the minimum line width ratio (vs content width) for content
	// to be considered spanning. Lines with content in gaps but less than this width
	// are treated as column content (avoids false positives from stray fragments).
	// Default: 0.35 (line must span 35% of content width - allows centered titles)
	SpanningThreshold float64
}

// DefaultColumnConfig returns sensible default configuration
func DefaultColumnConfig() ColumnConfig {
	return ColumnConfig{
		MinColumnWidth:    50.0,
		MinGapWidth:       20.0,
		MinGapHeightRatio: 0.5,
		MaxColumns:        6,
		MergeThreshold:    10.0,
		SpanningThreshold: 0.35,
	}
}

// ColumnDetector detects multi-column layouts in text
type ColumnDetector struct {
	config ColumnConfig
}

// NewColumnDetector creates a new column detector with default configuration
func NewColumnDetector() *ColumnDetector {
	return &ColumnDetector{
		config: DefaultColumnConfig(),
	}
}

// NewColumnDetectorWithConfig creates a column detector with custom configuration
func NewColumnDetectorWithConfig(config ColumnConfig) *ColumnDetector {
	return &ColumnDetector{
		config: config,
	}
}

// Detect analyzes text fragments and detects column layout
func (d *ColumnDetector) Detect(fragments []text.TextFragment, pageWidth, pageHeight float64) *ColumnLayout {
	if len(fragments) == 0 {
		return &ColumnLayout{
			Columns:    nil,
			PageWidth:  pageWidth,
			PageHeight: pageHeight,
			Config:     d.config,
		}
	}

	// Find column boundaries using whitespace gap analysis
	gaps := d.findVerticalGaps(fragments, pageWidth, pageHeight)

	// If no significant gaps, treat as single column
	if len(gaps) == 0 {
		return d.singleColumnLayout(fragments, pageWidth, pageHeight)
	}

	// Separate spanning fragments (those that cross gaps) from regular fragments
	regularFragments, spanningFragments := d.separateSpanningFragments(fragments, gaps)

	// Create columns based on gaps using only regular fragments
	columns := d.createColumnsFromGaps(regularFragments, gaps, pageWidth, pageHeight)

	// Validate and possibly merge columns
	columns = d.validateColumns(columns)

	return &ColumnLayout{
		Columns:           columns,
		SpanningFragments: spanningFragments,
		PageWidth:         pageWidth,
		PageHeight:        pageHeight,
		Config:            d.config,
	}
}

// separateSpanningFragments separates fragments that span across column gaps
// from fragments that belong to a single column.
// A line is "spanning" if it has content INSIDE a gap region (e.g., centered title)
// AND the line spans a significant width (to avoid false positives from stray fragments).
// Multi-column lines have content on BOTH SIDES of gaps but NOT inside them.
func (d *ColumnDetector) separateSpanningFragments(fragments []text.TextFragment, gaps []Gap) (regular, spanning []text.TextFragment) {
	if len(fragments) == 0 || len(gaps) == 0 {
		return fragments, nil
	}

	// Group fragments by Y position (lines)
	lines := groupFragmentsIntoLines(fragments)

	// Calculate total width span of all fragments to estimate page content width
	pageLeft, pageRight := float64(1e9), float64(0)
	for _, frag := range fragments {
		if frag.X < pageLeft {
			pageLeft = frag.X
		}
		if frag.X+frag.Width > pageRight {
			pageRight = frag.X + frag.Width
		}
	}
	contentWidth := pageRight - pageLeft

	// Check each line to see if it has content in any gap region
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Calculate line extent (min X to max X+Width of non-whitespace fragments)
		lineLeft, lineRight := float64(1e9), float64(0)
		for _, frag := range line {
			if isWhitespaceOnly(frag.Text) {
				continue
			}
			if frag.X < lineLeft {
				lineLeft = frag.X
			}
			if frag.X+frag.Width > lineRight {
				lineRight = frag.X + frag.Width
			}
		}
		lineWidth := lineRight - lineLeft

		// Check if any non-whitespace fragment on this line has its CENTER in a gap region
		// (edge overlaps are not enough - column content at edges can slightly overlap)
		// Whitespace is ignored because trailing spaces can end up in gap regions
		lineSpansGap := false
		for _, gap := range gaps {
			for _, frag := range line {
				// Skip whitespace-only fragments
				if isWhitespaceOnly(frag.Text) {
					continue
				}
				fragCenter := frag.X + frag.Width/2
				// Fragment spans gap if its center is inside the gap region
				if fragCenter > gap.Left && fragCenter < gap.Right {
					lineSpansGap = true
					break
				}
			}
			if lineSpansGap {
				break
			}
		}

		// Only consider it spanning if:
		// 1. It has content in a gap region, AND
		// 2. The line width is at least SpanningThreshold of total content width
		//    This filters out stray fragments that happen to fall in gap regions
		isSpanning := lineSpansGap && lineWidth > contentWidth*d.config.SpanningThreshold

		if isSpanning {
			spanning = append(spanning, line...)
		} else {
			regular = append(regular, line...)
		}
	}

	// Additional filter: if spanning content looks like stray body text, move it to regular
	// Spanning content like titles should be at distinct Y positions from body content
	spanning = filterStraySpanningContent(spanning, regular)

	return regular, spanning
}

// filterStraySpanningContent moves spanning lines back to regular if they appear to be
// stray body text rather than actual titles/headers. Body text that spans gaps is unusual
// and often indicates incorrect gap detection rather than true spanning content.
func filterStraySpanningContent(spanning, regular []text.TextFragment) []text.TextFragment {
	if len(spanning) == 0 {
		return spanning
	}

	// Group spanning fragments by Y (into lines)
	spanningLines := groupFragmentsIntoLines(spanning)

	// Find the Y position range of regular content
	if len(regular) == 0 {
		return spanning
	}

	regularMinY, regularMaxY := float64(1e9), float64(0)
	for _, frag := range regular {
		if frag.Y < regularMinY {
			regularMinY = frag.Y
		}
		if frag.Y > regularMaxY {
			regularMaxY = frag.Y
		}
	}

	// Keep only spanning lines that are clearly separate from body content
	// (at least 20 points away from the regular content Y range)
	var filteredSpanning []text.TextFragment
	tolerance := 20.0

	for _, line := range spanningLines {
		if len(line) == 0 {
			continue
		}

		lineY := line[0].Y

		// Line is valid spanning if it's above or below the main body content
		isAboveBody := lineY < regularMinY-tolerance
		isBelowBody := lineY > regularMaxY+tolerance

		// Also consider it spanning if it's near the top of regular content
		// (within first 5% of content Y range - header position)
		contentRange := regularMaxY - regularMinY
		isNearTop := contentRange > 0 && lineY < regularMinY+contentRange*0.05

		if isAboveBody || isBelowBody || isNearTop {
			filteredSpanning = append(filteredSpanning, line...)
		}
		// Lines in the middle of body content are moved back to regular implicitly
		// by not being added to filteredSpanning
	}

	return filteredSpanning
}

// Gap represents a vertical whitespace gap
type Gap struct {
	Left   float64 // Left edge of gap
	Right  float64 // Right edge of gap
	Top    float64 // Top of gap region
	Bottom float64 // Bottom of gap region
}

// slab represents a horizontal range (for gap detection)
type slab struct {
	left, right float64
}

// Width returns the width of the gap
func (g Gap) Width() float64 {
	return g.Right - g.Left
}

// Height returns the height of the gap
func (g Gap) Height() float64 {
	return g.Top - g.Bottom
}

// Center returns the X center of the gap
func (g Gap) Center() float64 {
	return (g.Left + g.Right) / 2
}

// findVerticalGaps finds significant vertical whitespace gaps using density analysis
// This approach handles documents with spanning headers/titles that cross column boundaries
func (d *ColumnDetector) findVerticalGaps(fragments []text.TextFragment, pageWidth, pageHeight float64) []Gap {
	if len(fragments) == 0 {
		return nil
	}

	// Build histogram of fragment density across X axis
	// Use 5-point buckets for good resolution
	bucketSize := 5.0
	numBuckets := int(pageWidth/bucketSize) + 1
	histogram := make([]int, numBuckets)

	// Find X range of actual content
	minX, maxX := fragments[0].X, fragments[0].X+fragments[0].Width
	for _, f := range fragments {
		if f.X < minX {
			minX = f.X
		}
		if f.X+f.Width > maxX {
			maxX = f.X + f.Width
		}

		// Count fragment in buckets it spans
		startBucket := int(f.X / bucketSize)
		endBucket := int((f.X + f.Width) / bucketSize)
		if startBucket < 0 {
			startBucket = 0
		}
		if endBucket >= numBuckets {
			endBucket = numBuckets - 1
		}
		for b := startBucket; b <= endBucket; b++ {
			histogram[b]++
		}
	}

	// Find average density in content area
	startBucket := int(minX / bucketSize)
	endBucket := int(maxX / bucketSize)
	if startBucket < 0 {
		startBucket = 0
	}
	if endBucket >= numBuckets {
		endBucket = numBuckets - 1
	}

	totalDensity := 0
	contentBuckets := 0
	for b := startBucket; b <= endBucket; b++ {
		totalDensity += histogram[b]
		contentBuckets++
	}
	avgDensity := float64(totalDensity) / float64(contentBuckets)

	// Find valleys (low-density regions) that could be column gaps
	// A valley must have density < 20% of average AND span at least MinGapWidth
	densityThreshold := avgDensity * 0.2
	var gaps []Gap

	inValley := false
	valleyStart := 0

	for b := startBucket; b <= endBucket; b++ {
		isLow := float64(histogram[b]) < densityThreshold

		if isLow && !inValley {
			// Start of a valley
			inValley = true
			valleyStart = b
		} else if !isLow && inValley {
			// End of a valley
			inValley = false
			valleyEnd := b

			gapLeft := float64(valleyStart) * bucketSize
			gapRight := float64(valleyEnd) * bucketSize
			gapWidth := gapRight - gapLeft

			if gapWidth >= d.config.MinGapWidth {
				gaps = append(gaps, Gap{
					Left:   gapLeft,
					Right:  gapRight,
					Top:    pageHeight,
					Bottom: 0,
				})
			}
		}
	}

	// Handle valley at end
	if inValley {
		gapLeft := float64(valleyStart) * bucketSize
		gapRight := float64(endBucket) * bucketSize
		gapWidth := gapRight - gapLeft

		if gapWidth >= d.config.MinGapWidth {
			gaps = append(gaps, Gap{
				Left:   gapLeft,
				Right:  gapRight,
				Top:    pageHeight,
				Bottom: 0,
			})
		}
	}

	// Limit to max columns - 1 gaps
	if len(gaps) >= d.config.MaxColumns {
		gaps = gaps[:d.config.MaxColumns-1]
	}

	return gaps
}

// mergeSlabs merges overlapping horizontal slabs
func mergeSlabs(slabs []slab) []slab {
	if len(slabs) == 0 {
		return nil
	}

	merged := []slab{slabs[0]}

	for i := 1; i < len(slabs); i++ {
		current := slabs[i]
		last := &merged[len(merged)-1]

		// Check for overlap or adjacency (with small tolerance)
		if current.left <= last.right+5.0 {
			// Merge: extend the last slab
			if current.right > last.right {
				last.right = current.right
			}
		} else {
			// No overlap: add new slab
			merged = append(merged, current)
		}
	}

	return merged
}

// measureGapVerticalExtent measures what fraction of page height a gap spans
func (d *ColumnDetector) measureGapVerticalExtent(fragments []text.TextFragment, gapLeft, gapRight float64, pageHeight float64) float64 {
	// Find the Y range where the gap exists
	// The gap exists at a Y level if no fragment crosses it at that level

	// Collect Y ranges of fragments that cross the gap region
	var crossingYRanges []struct{ top, bottom float64 }

	for _, f := range fragments {
		fragLeft := f.X
		fragRight := f.X + f.Width

		// Does this fragment cross the gap horizontally?
		if fragRight > gapLeft && fragLeft < gapRight {
			// This fragment blocks the gap at its Y position
			crossingYRanges = append(crossingYRanges, struct{ top, bottom float64 }{
				top:    f.Y + f.Height,
				bottom: f.Y,
			})
		}
	}

	if len(crossingYRanges) == 0 {
		// No fragments cross this gap - it spans full page
		return 1.0
	}

	// Sort crossing ranges by bottom
	sort.Slice(crossingYRanges, func(i, j int) bool {
		return crossingYRanges[i].bottom < crossingYRanges[j].bottom
	})

	// Merge overlapping Y ranges
	merged := []struct{ top, bottom float64 }{crossingYRanges[0]}
	for i := 1; i < len(crossingYRanges); i++ {
		current := crossingYRanges[i]
		last := &merged[len(merged)-1]

		if current.bottom <= last.top {
			// Overlaps
			if current.top > last.top {
				last.top = current.top
			}
		} else {
			merged = append(merged, current)
		}
	}

	// Calculate total blocked height
	blockedHeight := 0.0
	for _, r := range merged {
		blockedHeight += r.top - r.bottom
	}

	// Gap extent is the unblocked fraction
	unblocked := pageHeight - blockedHeight
	if pageHeight <= 0 {
		return 0
	}

	return unblocked / pageHeight
}

// singleColumnLayout creates a layout with all fragments in one column
func (d *ColumnDetector) singleColumnLayout(fragments []text.TextFragment, pageWidth, pageHeight float64) *ColumnLayout {
	if len(fragments) == 0 {
		return &ColumnLayout{
			Columns:    nil,
			PageWidth:  pageWidth,
			PageHeight: pageHeight,
			Config:     d.config,
		}
	}

	// Calculate bounding box of all fragments
	bbox := fragmentsBBox(fragments)

	// Sort fragments top to bottom
	sorted := make([]text.TextFragment, len(fragments))
	copy(sorted, fragments)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Y > sorted[j].Y // Higher Y = higher on page
	})

	return &ColumnLayout{
		Columns: []Column{
			{
				BBox:      bbox,
				Fragments: sorted,
				Index:     0,
			},
		},
		PageWidth:  pageWidth,
		PageHeight: pageHeight,
		Config:     d.config,
	}
}

// createColumnsFromGaps creates columns based on detected gaps
func (d *ColumnDetector) createColumnsFromGaps(fragments []text.TextFragment, gaps []Gap, pageWidth, pageHeight float64) []Column {
	if len(fragments) == 0 {
		return nil
	}

	// Sort gaps by X position
	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].Left < gaps[j].Left
	})

	// Create column boundaries from gaps
	// First column starts at leftmost fragment
	// Last column ends at rightmost fragment
	minX := fragments[0].X
	maxX := fragments[0].X + fragments[0].Width
	for _, f := range fragments {
		if f.X < minX {
			minX = f.X
		}
		if f.X+f.Width > maxX {
			maxX = f.X + f.Width
		}
	}

	// Build column boundaries
	type boundary struct {
		left, right float64
	}

	var boundaries []boundary

	// First column: from minX to first gap center
	boundaries = append(boundaries, boundary{
		left:  minX,
		right: gaps[0].Center(),
	})

	// Middle columns: between consecutive gap centers
	for i := 0; i < len(gaps)-1; i++ {
		boundaries = append(boundaries, boundary{
			left:  gaps[i].Center(),
			right: gaps[i+1].Center(),
		})
	}

	// Last column: from last gap center to maxX
	boundaries = append(boundaries, boundary{
		left:  gaps[len(gaps)-1].Center(),
		right: maxX,
	})

	// Assign fragments to columns
	columns := make([]Column, len(boundaries))
	for i, b := range boundaries {
		columns[i] = Column{
			BBox: model.BBox{
				X:      b.left,
				Y:      0,
				Width:  b.right - b.left,
				Height: pageHeight,
			},
			Index: i,
		}
	}

	// Assign each fragment to its column
	for _, f := range fragments {
		// Find which column this fragment belongs to
		fragCenter := f.X + f.Width/2

		for i := range columns {
			if fragCenter >= boundaries[i].left && fragCenter < boundaries[i].right {
				columns[i].Fragments = append(columns[i].Fragments, f)
				break
			}
		}
	}

	// Update column bounding boxes
	// Note: We preserve document order (order fragments appear in PDF)
	// rather than sorting by Y, since PDFs typically have content in visual order
	for i := range columns {
		if len(columns[i].Fragments) > 0 {
			columns[i].BBox = fragmentsBBox(columns[i].Fragments)
		}
	}

	return columns
}

// validateColumns validates and cleans up detected columns
func (d *ColumnDetector) validateColumns(columns []Column) []Column {
	var valid []Column

	for _, col := range columns {
		// Skip empty columns
		if len(col.Fragments) == 0 {
			continue
		}

		// Skip columns that are too narrow
		if col.BBox.Width < d.config.MinColumnWidth {
			continue
		}

		valid = append(valid, col)
	}

	// Re-index columns
	for i := range valid {
		valid[i].Index = i
	}

	return valid
}

// fragmentsBBox calculates the bounding box of a set of fragments
func fragmentsBBox(fragments []text.TextFragment) model.BBox {
	if len(fragments) == 0 {
		return model.BBox{}
	}

	minX := fragments[0].X
	minY := fragments[0].Y
	maxX := fragments[0].X + fragments[0].Width
	maxY := fragments[0].Y + fragments[0].Height

	for _, f := range fragments[1:] {
		if f.X < minX {
			minX = f.X
		}
		if f.Y < minY {
			minY = f.Y
		}
		if f.X+f.Width > maxX {
			maxX = f.X + f.Width
		}
		if f.Y+f.Height > maxY {
			maxY = f.Y + f.Height
		}
	}

	return model.BBox{
		X:      minX,
		Y:      minY,
		Width:  maxX - minX,
		Height: maxY - minY,
	}
}

// GetText returns the text content in reading order (column by column, top to bottom)
func (l *ColumnLayout) GetText() string {
	if l == nil {
		return ""
	}

	var result string

	// First, output spanning fragments (full-width headers, titles, etc.)
	// These are typically at the top of the page
	if len(l.SpanningFragments) > 0 {
		spanningText := l.getSpanningText()
		if len(spanningText) > 0 {
			result += spanningText + "\n\n"
		}
	}

	// Then output column content
	if len(l.Columns) == 0 {
		return result
	}

	for colIdx, col := range l.Columns {
		colText := l.getColumnText(col)
		result += colText

		// Add column separator (double newline between columns)
		if colIdx < len(l.Columns)-1 && len(colText) > 0 {
			result += "\n\n"
		}
	}

	return result
}

// getSpanningText returns text from spanning fragments
func (l *ColumnLayout) getSpanningText() string {
	if len(l.SpanningFragments) == 0 {
		return ""
	}

	// Group spanning fragments into lines
	lines := groupFragmentsIntoLines(l.SpanningFragments)

	var result string
	for lineIdx, line := range lines {
		// Sort fragments within line by X (left to right)
		sort.Slice(line, func(i, j int) bool {
			return line[i].X < line[j].X
		})

		// Assemble line text
		for i, frag := range line {
			if i > 0 {
				prevFrag := line[i-1]
				gap := frag.X - (prevFrag.X + prevFrag.Width)
				if gap > frag.Height*0.1 {
					result += " "
				}
			}
			result += frag.Text
		}

		// Add line break between lines
		if lineIdx < len(lines)-1 {
			result += "\n"
		}
	}

	return result
}

// getColumnText returns text from a single column
func (l *ColumnLayout) getColumnText(col Column) string {
	if len(col.Fragments) == 0 {
		return ""
	}

	// Group fragments into lines by Y position
	lines := groupFragmentsIntoLines(col.Fragments)

	var result string

	for lineIdx, line := range lines {
		// Sort fragments within line by X (left to right)
		sort.Slice(line, func(i, j int) bool {
			return line[i].X < line[j].X
		})

		// Assemble line text
		for i, frag := range line {
			if i > 0 {
				// Add space between fragments if there's a gap
				prevFrag := line[i-1]
				gap := frag.X - (prevFrag.X + prevFrag.Width)
				if gap > frag.Height*0.1 { // Small gap threshold
					result += " "
				}
			}
			result += frag.Text
		}

		// Add line/paragraph break
		if lineIdx < len(lines)-1 {
			nextLine := lines[lineIdx+1]
			currentY := line[0].Y
			nextY := nextLine[0].Y
			verticalGap := currentY - nextY // Positive going down the page

			if verticalGap > line[0].Height*1.5 {
				// Paragraph break
				result += "\n\n"
			} else {
				// Line break
				result += "\n"
			}
		}
	}

	return result
}

// groupFragmentsIntoLines groups fragments by Y position into lines
// Preserves document order (order fragments appear in PDF) for reading order,
// since PDFs are typically authored with content in visual order.
func groupFragmentsIntoLines(fragments []text.TextFragment) [][]text.TextFragment {
	if len(fragments) == 0 {
		return nil
	}

	// Group by Y similarity while preserving order
	type yBand struct {
		y         float64
		fragments []text.TextFragment
	}

	var bands []yBand

	for _, frag := range fragments {
		// Find existing band with similar Y
		found := false
		for i := range bands {
			yTolerance := frag.Height * 0.5
			if yTolerance < 2.0 {
				yTolerance = 2.0 // Minimum tolerance
			}
			if absFloat64(frag.Y-bands[i].y) <= yTolerance {
				bands[i].fragments = append(bands[i].fragments, frag)
				found = true
				break
			}
		}

		if !found {
			// Create new band
			bands = append(bands, yBand{
				y:         frag.Y,
				fragments: []text.TextFragment{frag},
			})
		}
	}

	// Sort bands by Y position (top to bottom in PDF coordinates = highest Y first)
	sort.Slice(bands, func(i, j int) bool {
		return bands[i].y > bands[j].y
	})

	// Convert bands to lines
	var lines [][]text.TextFragment
	for _, band := range bands {
		lines = append(lines, band.fragments)
	}

	return lines
}

// absFloat64 returns absolute value of float64
func absFloat64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// isWhitespaceOnly returns true if the string contains only whitespace
func isWhitespaceOnly(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

// ColumnCount returns the number of detected columns
func (l *ColumnLayout) ColumnCount() int {
	if l == nil {
		return 0
	}
	return len(l.Columns)
}

// IsSingleColumn returns true if only one column was detected
func (l *ColumnLayout) IsSingleColumn() bool {
	return l.ColumnCount() <= 1
}

// IsMultiColumn returns true if multiple columns were detected
func (l *ColumnLayout) IsMultiColumn() bool {
	return l.ColumnCount() > 1
}

// GetColumn returns a specific column by index
func (l *ColumnLayout) GetColumn(index int) *Column {
	if l == nil || index < 0 || index >= len(l.Columns) {
		return nil
	}
	return &l.Columns[index]
}

// GetFragmentsInReadingOrder returns all fragments ordered for reading
// (left column first, then right column, each column top-to-bottom)
func (l *ColumnLayout) GetFragmentsInReadingOrder() []text.TextFragment {
	if l == nil {
		return nil
	}

	var result []text.TextFragment
	for _, col := range l.Columns {
		result = append(result, col.Fragments...)
	}

	return result
}
