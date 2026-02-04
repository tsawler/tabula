// Package layout provides document layout analysis including line detection,
// block detection, column detection, and structural element identification.
package layout

import (
	"sort"
	"strings"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// LineAlignment represents the horizontal alignment of a line
type LineAlignment int

const (
	AlignUnknown LineAlignment = iota
	AlignLeft
	AlignCenter
	AlignRight
	AlignJustified
)

// String returns a string representation of the alignment
func (a LineAlignment) String() string {
	switch a {
	case AlignLeft:
		return "left"
	case AlignCenter:
		return "center"
	case AlignRight:
		return "right"
	case AlignJustified:
		return "justified"
	default:
		return "unknown"
	}
}

// Line represents a single line of text on a page
type Line struct {
	// BBox is the bounding box of the line
	BBox model.BBox

	// Fragments are the text fragments that make up this line (sorted left to right)
	Fragments []text.TextFragment

	// Text is the assembled text content of the line
	Text string

	// Index is the line's position on the page (0-based, top to bottom)
	Index int

	// Baseline is the Y coordinate of the text baseline
	Baseline float64

	// Height is the line height (max fragment height)
	Height float64

	// SpacingBefore is the vertical space from the previous line (0 for first line)
	SpacingBefore float64

	// SpacingAfter is the vertical space to the next line (0 for last line)
	SpacingAfter float64

	// Alignment is the detected horizontal alignment
	Alignment LineAlignment

	// Indentation is the left indentation relative to the page/column margin
	Indentation float64

	// AverageFontSize is the average font size of fragments in this line
	AverageFontSize float64

	// Direction is the dominant text direction (LTR/RTL)
	Direction text.Direction
}

// LineLayout represents the detected line structure of a page or region
type LineLayout struct {
	// Lines are the detected text lines (sorted top to bottom)
	Lines []Line

	// PageWidth is the width of the page/region
	PageWidth float64

	// PageHeight is the height of the page/region
	PageHeight float64

	// AverageLineSpacing is the average spacing between lines
	AverageLineSpacing float64

	// AverageLineHeight is the average line height
	AverageLineHeight float64

	// Config is the configuration used for detection
	Config LineConfig
}

// LineConfig holds configuration for line detection
type LineConfig struct {
	// LineHeightTolerance is the Y-distance tolerance for grouping fragments into lines
	// as a fraction of fragment height (default: 0.5)
	LineHeightTolerance float64

	// MinLineWidth is the minimum width for a valid line (default: 5 points)
	MinLineWidth float64

	// AlignmentTolerance is the tolerance for alignment detection (default: 10 points)
	AlignmentTolerance float64

	// JustificationThreshold is the minimum line width ratio to consider justified
	// (default: 0.9 = line must be 90% of max width)
	JustificationThreshold float64
}

// DefaultLineConfig returns sensible default configuration
func DefaultLineConfig() LineConfig {
	return LineConfig{
		LineHeightTolerance:    0.5,
		MinLineWidth:           5.0,
		AlignmentTolerance:     10.0,
		JustificationThreshold: 0.9,
	}
}

// LineDetector detects text lines on a page
type LineDetector struct {
	config LineConfig
}

// NewLineDetector creates a new line detector with default configuration
func NewLineDetector() *LineDetector {
	return &LineDetector{
		config: DefaultLineConfig(),
	}
}

// NewLineDetectorWithConfig creates a line detector with custom configuration
func NewLineDetectorWithConfig(config LineConfig) *LineDetector {
	return &LineDetector{
		config: config,
	}
}

// Detect analyzes text fragments and detects lines
func (d *LineDetector) Detect(fragments []text.TextFragment, pageWidth, pageHeight float64) *LineLayout {
	if len(fragments) == 0 {
		return &LineLayout{
			Lines:      nil,
			PageWidth:  pageWidth,
			PageHeight: pageHeight,
			Config:     d.config,
		}
	}

	// Step 1: Group fragments into lines by Y position
	lineGroups := d.groupIntoLines(fragments)

	// Step 2: Build Line objects with metadata
	lines := d.buildLines(lineGroups, pageWidth)

	// Step 3: Calculate spacing between lines
	d.calculateSpacing(lines)

	// Step 4: Detect alignment
	d.detectAlignment(lines, pageWidth)

	// Step 5: Calculate layout statistics
	avgSpacing, avgHeight := d.calculateStatistics(lines)

	return &LineLayout{
		Lines:              lines,
		PageWidth:          pageWidth,
		PageHeight:         pageHeight,
		AverageLineSpacing: avgSpacing,
		AverageLineHeight:  avgHeight,
		Config:             d.config,
	}
}

// groupIntoLines groups fragments into horizontal lines based on Y position
func (d *LineDetector) groupIntoLines(fragments []text.TextFragment) [][]text.TextFragment {
	if len(fragments) == 0 {
		return nil
	}

	// Calculate adaptive tolerance based on actual content characteristics
	// This handles PDFs where CTM scaling compresses coordinates
	adaptiveTolerance := d.calculateAdaptiveTolerance(fragments)

	// Sort fragments by Y (descending, top to bottom in PDF coords) only
	// Preserve stream order for same-Y fragments - X sorting happens per-line later
	// (if shouldPreserveStreamOrder returns false for that line)
	sorted := make([]text.TextFragment, len(fragments))
	copy(sorted, fragments)
	sort.SliceStable(sorted, func(i, j int) bool {
		yDiff := sorted[i].Y - sorted[j].Y
		if absFloat64(yDiff) > adaptiveTolerance {
			return yDiff > 0 // Higher Y first (top of page)
		}
		// Same line - preserve stream order (return false means don't swap)
		return false
	})

	var lines [][]text.TextFragment
	var currentLine []text.TextFragment

	for _, frag := range sorted {
		if len(currentLine) == 0 {
			currentLine = append(currentLine, frag)
			continue
		}

		// Check if fragment is on same line as previous
		// Use the average Y of the current line for better accuracy
		avgY := d.averageLineY(currentLine)

		if absFloat64(frag.Y-avgY) <= adaptiveTolerance {
			// Same line
			currentLine = append(currentLine, frag)
		} else {
			// New line - sort current line by X if stream order shouldn't be preserved
			// Use stable sort with tolerance to handle overlapping fragments (Word/Quartz issue)
			if !shouldPreserveStreamOrder(currentLine) {
				sort.SliceStable(currentLine, func(i, j int) bool {
					xTol := currentLine[i].FontSize * xTolerance
					if absFloat64(currentLine[i].X-currentLine[j].X) < xTol {
						return false // Treat as equal, preserve stream order
					}
					return currentLine[i].X < currentLine[j].X
				})
			}
			lines = append(lines, currentLine)
			currentLine = []text.TextFragment{frag}
		}
	}

	// Don't forget the last line
	if len(currentLine) > 0 {
		if !shouldPreserveStreamOrder(currentLine) {
			sort.SliceStable(currentLine, func(i, j int) bool {
				xTol := currentLine[i].FontSize * xTolerance
				if absFloat64(currentLine[i].X-currentLine[j].X) < xTol {
					return false // Treat as equal, preserve stream order
				}
				return currentLine[i].X < currentLine[j].X
			})
		}
		lines = append(lines, currentLine)
	}

	return lines
}

// calculateAdaptiveTolerance determines the Y tolerance for line grouping based on
// actual content characteristics. This handles PDFs where CTM scaling compresses
// coordinates, making standard font-height-based tolerance too large.
func (d *LineDetector) calculateAdaptiveTolerance(fragments []text.TextFragment) float64 {
	if len(fragments) == 0 {
		return 2.0 // Default minimum
	}

	// Calculate average font height
	totalHeight := 0.0
	for _, f := range fragments {
		totalHeight += f.Height
	}
	avgHeight := totalHeight / float64(len(fragments))

	// Standard tolerance based on font height
	standardTolerance := avgHeight * d.config.LineHeightTolerance

	// Analyze actual Y gaps in the content to detect if coordinates are compressed
	// Sort unique Y positions to find typical line spacing
	yPositions := make(map[float64]bool)
	for _, f := range fragments {
		// Round Y to avoid floating point noise
		roundedY := float64(int(f.Y*10)) / 10
		yPositions[roundedY] = true
	}

	if len(yPositions) < 3 {
		return standardTolerance // Not enough data
	}

	// Convert to sorted slice
	uniqueYs := make([]float64, 0, len(yPositions))
	for y := range yPositions {
		uniqueYs = append(uniqueYs, y)
	}
	sort.Float64s(uniqueYs)

	// Calculate gaps between adjacent Y positions
	gaps := make([]float64, 0, len(uniqueYs)-1)
	for i := 1; i < len(uniqueYs); i++ {
		gap := absFloat64(uniqueYs[i] - uniqueYs[i-1])
		if gap > 0.1 { // Ignore tiny gaps (same line with baseline variation)
			gaps = append(gaps, gap)
		}
	}

	if len(gaps) == 0 {
		return standardTolerance
	}

	// Sort gaps to find percentiles
	sort.Float64s(gaps)

	// Use the 10th percentile gap as the tolerance threshold
	// This is more conservative - ensures we don't merge distinct lines
	// The 10th percentile represents the smallest inter-line gaps (excluding noise)
	p10Index := len(gaps) / 10
	if p10Index < 0 {
		p10Index = 0
	}
	minInterLineGap := gaps[p10Index]

	// If the min inter-line gap is much smaller than the font height,
	// the coordinates are likely scaled/compressed.
	// Use a very conservative tolerance to avoid merging separate lines.
	if minInterLineGap < avgHeight*0.5 && minInterLineGap > 0.1 {
		// Coordinates are compressed - use gap-based tolerance
		// Set tolerance to about 20% of the minimum gap to be very conservative
		adaptiveTolerance := minInterLineGap * 0.2
		if adaptiveTolerance < 0.15 {
			adaptiveTolerance = 0.15 // Very small minimum for compressed coordinates
		}
		return adaptiveTolerance
	}

	// Standard case - use font-height-based tolerance
	return standardTolerance
}

// averageLineY returns the average Y coordinate of fragments in a line
func (d *LineDetector) averageLineY(fragments []text.TextFragment) float64 {
	if len(fragments) == 0 {
		return 0
	}
	total := 0.0
	for _, f := range fragments {
		total += f.Y
	}
	return total / float64(len(fragments))
}

// averageFragmentHeight returns the average height of fragments
func (d *LineDetector) averageFragmentHeight(fragments []text.TextFragment) float64 {
	if len(fragments) == 0 {
		return 12.0 // Default
	}
	total := 0.0
	for _, f := range fragments {
		total += f.Height
	}
	return total / float64(len(fragments))
}

// buildLines creates Line objects from fragment groups
func (d *LineDetector) buildLines(lineGroups [][]text.TextFragment, pageWidth float64) []Line {
	lines := make([]Line, 0, len(lineGroups))

	for i, fragments := range lineGroups {
		if len(fragments) == 0 {
			continue
		}

		line := Line{
			Index:     i,
			Fragments: fragments,
		}

		// Calculate bounding box
		line.BBox = fragmentsBBox(fragments)

		// Calculate baseline (minimum Y in the line)
		line.Baseline = fragments[0].Y
		for _, f := range fragments[1:] {
			if f.Y < line.Baseline {
				line.Baseline = f.Y
			}
		}

		// Calculate height (maximum fragment height)
		line.Height = fragments[0].Height
		for _, f := range fragments[1:] {
			if f.Height > line.Height {
				line.Height = f.Height
			}
		}

		// Calculate average font size
		totalFontSize := 0.0
		for _, f := range fragments {
			totalFontSize += f.FontSize
		}
		line.AverageFontSize = totalFontSize / float64(len(fragments))

		// Assemble text
		line.Text = d.assembleLineText(fragments)

		// Detect text direction
		line.Direction = d.detectLineDirection(fragments)

		// Calculate indentation (distance from left margin)
		line.Indentation = line.BBox.X

		// Skip lines that are too narrow
		if line.BBox.Width < d.config.MinLineWidth {
			continue
		}

		lines = append(lines, line)
	}

	// Re-index lines
	for i := range lines {
		lines[i].Index = i
	}

	return lines
}

// assembleLineText assembles text from fragments with appropriate spacing
func (d *LineDetector) assembleLineText(fragments []text.TextFragment) string {
	if len(fragments) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, frag := range fragments {
		if i > 0 {
			prevFrag := fragments[i-1]
			gap := frag.X - (prevFrag.X + prevFrag.Width)
			// Add space if there's a significant gap
			if gap > frag.Height*0.1 {
				sb.WriteString(" ")
			}
		}
		sb.WriteString(frag.Text)
	}

	return sb.String()
}

// detectLineDirection determines the dominant text direction of a line
func (d *LineDetector) detectLineDirection(fragments []text.TextFragment) text.Direction {
	ltrCount := 0
	rtlCount := 0

	for _, frag := range fragments {
		switch frag.Direction {
		case text.LTR:
			ltrCount++
		case text.RTL:
			rtlCount++
		}
	}

	if rtlCount > ltrCount {
		return text.RTL
	}
	if ltrCount > 0 {
		return text.LTR
	}
	return text.Neutral
}

// calculateSpacing calculates spacing between consecutive lines
func (d *LineDetector) calculateSpacing(lines []Line) {
	for i := range lines {
		if i > 0 {
			// Spacing from previous line's bottom to this line's top
			prevBottom := lines[i-1].Baseline
			thisTop := lines[i].Baseline + lines[i].Height
			lines[i].SpacingBefore = prevBottom - thisTop
			lines[i-1].SpacingAfter = lines[i].SpacingBefore
		}
	}
}

// detectAlignment detects horizontal alignment for each line
func (d *LineDetector) detectAlignment(lines []Line, pageWidth float64) {
	if len(lines) == 0 {
		return
	}

	// Find content boundaries (leftmost and rightmost positions)
	leftMargin := lines[0].BBox.X
	rightMargin := lines[0].BBox.X + lines[0].BBox.Width
	maxWidth := lines[0].BBox.Width

	for _, line := range lines[1:] {
		if line.BBox.X < leftMargin {
			leftMargin = line.BBox.X
		}
		lineRight := line.BBox.X + line.BBox.Width
		if lineRight > rightMargin {
			rightMargin = lineRight
		}
		if line.BBox.Width > maxWidth {
			maxWidth = line.BBox.Width
		}
	}

	contentWidth := rightMargin - leftMargin
	tolerance := d.config.AlignmentTolerance

	for i := range lines {
		line := &lines[i]
		lineLeft := line.BBox.X
		lineRight := line.BBox.X + line.BBox.Width
		lineCenter := lineLeft + line.BBox.Width/2
		contentCenter := leftMargin + contentWidth/2

		// Check for justified (line spans almost full width)
		widthRatio := line.BBox.Width / maxWidth
		if widthRatio >= d.config.JustificationThreshold {
			line.Alignment = AlignJustified
			continue
		}

		// Check alignment based on position
		leftAligned := absFloat64(lineLeft-leftMargin) <= tolerance
		rightAligned := absFloat64(lineRight-rightMargin) <= tolerance
		centerAligned := absFloat64(lineCenter-contentCenter) <= tolerance

		if centerAligned && !leftAligned && !rightAligned {
			line.Alignment = AlignCenter
		} else if rightAligned && !leftAligned {
			line.Alignment = AlignRight
		} else if leftAligned {
			line.Alignment = AlignLeft
		} else {
			line.Alignment = AlignUnknown
		}
	}
}

// calculateStatistics calculates average line spacing and height
func (d *LineDetector) calculateStatistics(lines []Line) (avgSpacing, avgHeight float64) {
	if len(lines) == 0 {
		return 0, 0
	}

	totalHeight := 0.0
	for _, line := range lines {
		totalHeight += line.Height
	}
	avgHeight = totalHeight / float64(len(lines))

	if len(lines) < 2 {
		return 0, avgHeight
	}

	totalSpacing := 0.0
	spacingCount := 0
	for _, line := range lines {
		if line.SpacingBefore > 0 {
			totalSpacing += line.SpacingBefore
			spacingCount++
		}
	}

	if spacingCount > 0 {
		avgSpacing = totalSpacing / float64(spacingCount)
	}

	return avgSpacing, avgHeight
}

// LineLayout methods

// LineCount returns the number of detected lines
func (l *LineLayout) LineCount() int {
	if l == nil {
		return 0
	}
	return len(l.Lines)
}

// GetLine returns a specific line by index
func (l *LineLayout) GetLine(index int) *Line {
	if l == nil || index < 0 || index >= len(l.Lines) {
		return nil
	}
	return &l.Lines[index]
}

// GetText returns all text in line order
func (l *LineLayout) GetText() string {
	if l == nil || len(l.Lines) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, line := range l.Lines {
		sb.WriteString(line.Text)
		if i < len(l.Lines)-1 {
			// Add appropriate line break based on spacing
			if line.SpacingAfter > l.AverageLineSpacing*1.5 {
				sb.WriteString("\n\n") // Paragraph break
			} else {
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

// GetAllFragments returns all fragments in reading order
func (l *LineLayout) GetAllFragments() []text.TextFragment {
	if l == nil {
		return nil
	}

	var result []text.TextFragment
	for _, line := range l.Lines {
		result = append(result, line.Fragments...)
	}
	return result
}

// FindLinesInRegion returns lines that fall within a bounding box
func (l *LineLayout) FindLinesInRegion(bbox model.BBox) []Line {
	if l == nil {
		return nil
	}

	var result []Line
	for _, line := range l.Lines {
		// Check if line overlaps with region
		if line.BBox.X+line.BBox.Width > bbox.X &&
			line.BBox.X < bbox.X+bbox.Width &&
			line.BBox.Y+line.BBox.Height > bbox.Y &&
			line.BBox.Y < bbox.Y+bbox.Height {
			result = append(result, line)
		}
	}
	return result
}

// GetLinesByAlignment returns lines with a specific alignment
func (l *LineLayout) GetLinesByAlignment(alignment LineAlignment) []Line {
	if l == nil {
		return nil
	}

	var result []Line
	for _, line := range l.Lines {
		if line.Alignment == alignment {
			result = append(result, line)
		}
	}
	return result
}

// IsParagraphBreak returns true if there's a paragraph break after the given line index
func (l *LineLayout) IsParagraphBreak(lineIndex int) bool {
	if l == nil || lineIndex < 0 || lineIndex >= len(l.Lines)-1 {
		return false
	}

	line := l.Lines[lineIndex]
	// Consider it a paragraph break if spacing is significantly more than average
	return line.SpacingAfter > l.AverageLineSpacing*1.5
}

// Line methods

// IsIndented returns true if the line is indented relative to the margin
func (line *Line) IsIndented(margin, tolerance float64) bool {
	if line == nil {
		return false
	}
	return line.Indentation > margin+tolerance
}

// ContainsPoint returns true if the point is within the line's bounding box
func (line *Line) ContainsPoint(x, y float64) bool {
	if line == nil {
		return false
	}
	return x >= line.BBox.X && x <= line.BBox.X+line.BBox.Width &&
		y >= line.BBox.Y && y <= line.BBox.Y+line.BBox.Height
}

// WordCount returns an approximate word count for the line
func (line *Line) WordCount() int {
	if line == nil || line.Text == "" {
		return 0
	}
	// Simple word count by splitting on whitespace
	words := strings.Fields(line.Text)
	return len(words)
}

// IsEmpty returns true if the line has no text content
func (line *Line) IsEmpty() bool {
	if line == nil {
		return true
	}
	return strings.TrimSpace(line.Text) == ""
}

// HasLargerFont returns true if this line's font is larger than the given size
func (line *Line) HasLargerFont(size float64) bool {
	if line == nil {
		return false
	}
	return line.AverageFontSize > size
}
