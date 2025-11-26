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
}

// DefaultColumnConfig returns sensible default configuration
func DefaultColumnConfig() ColumnConfig {
	return ColumnConfig{
		MinColumnWidth:    50.0,
		MinGapWidth:       20.0,
		MinGapHeightRatio: 0.5,
		MaxColumns:        6,
		MergeThreshold:    10.0,
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

	// Create columns based on gaps
	columns := d.createColumnsFromGaps(fragments, gaps, pageWidth, pageHeight)

	// Validate and possibly merge columns
	columns = d.validateColumns(columns)

	return &ColumnLayout{
		Columns:    columns,
		PageWidth:  pageWidth,
		PageHeight: pageHeight,
		Config:     d.config,
	}
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

// findVerticalGaps finds significant vertical whitespace gaps
func (d *ColumnDetector) findVerticalGaps(fragments []text.TextFragment, pageWidth, pageHeight float64) []Gap {
	if len(fragments) == 0 {
		return nil
	}

	// Build a list of horizontal "slabs" - the X ranges covered by text
	// at different Y levels

	// Collect all X ranges
	var slabs []slab
	for _, f := range fragments {
		slabs = append(slabs, slab{
			left:  f.X,
			right: f.X + f.Width,
		})
	}

	// Sort by left edge
	sort.Slice(slabs, func(i, j int) bool {
		return slabs[i].left < slabs[j].left
	})

	// Merge overlapping slabs to get covered regions
	merged := mergeSlabs(slabs)

	// Find gaps between merged regions
	var gaps []Gap
	for i := 0; i < len(merged)-1; i++ {
		gapLeft := merged[i].right
		gapRight := merged[i+1].left
		gapWidth := gapRight - gapLeft

		if gapWidth >= d.config.MinGapWidth {
			// Verify this gap extends vertically
			gapExtent := d.measureGapVerticalExtent(fragments, gapLeft, gapRight, pageHeight)

			if gapExtent >= d.config.MinGapHeightRatio {
				gaps = append(gaps, Gap{
					Left:   gapLeft,
					Right:  gapRight,
					Top:    pageHeight, // Full page for now
					Bottom: 0,
				})
			}
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

	// Update column bbox to actual content
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
	if l == nil || len(l.Columns) == 0 {
		return ""
	}

	var result string

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
	// We'll collect unique Y "bands" in order of appearance
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
