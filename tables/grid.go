package tables

import (
	"math"
	"sort"

	"github.com/tsawler/tabula/graphicsstate"
	"github.com/tsawler/tabula/model"
)

// GridDetector detects table grids from extracted graphics lines
type GridDetector struct {
	// Tolerance for considering lines aligned (in points)
	AlignmentTolerance float64

	// Minimum number of aligned lines to form a grid axis
	MinAlignedLines int

	// Minimum line length to consider (in points)
	MinLineLength float64

	// Maximum gap between lines to be considered part of same grid
	MaxLineGap float64
}

// NewGridDetector creates a new grid detector with default settings
func NewGridDetector() *GridDetector {
	return &GridDetector{
		AlignmentTolerance: 3.0,  // 3 points tolerance
		MinAlignedLines:    2,    // At least 2 lines
		MinLineLength:      10.0, // At least 10 points long
		MaxLineGap:         50.0, // Max 50 points gap
	}
}

// GridHypothesis represents a potential table grid detected from lines
type GridHypothesis struct {
	// Bounding box of the grid
	BBox model.BBox

	// Horizontal line positions (Y coordinates, sorted descending)
	HorizontalLines []float64

	// Vertical line positions (X coordinates, sorted ascending)
	VerticalLines []float64

	// Confidence score (0-1)
	Confidence float64

	// Number of rows and columns
	Rows int
	Cols int

	// Whether the grid has complete borders
	HasTopBorder    bool
	HasBottomBorder bool
	HasLeftBorder   bool
	HasRightBorder  bool
}

// AlignedLineGroup represents a group of lines aligned on an axis
type AlignedLineGroup struct {
	// Position on the alignment axis (X for vertical lines, Y for horizontal)
	Position float64

	// Lines in this group
	Lines []graphicsstate.ExtractedLine

	// Total coverage (sum of line lengths)
	TotalLength float64

	// Span of the lines (min to max on the perpendicular axis)
	MinExtent float64
	MaxExtent float64
}

// DetectFromExtractor detects grid hypotheses from a graphics extractor
func (gd *GridDetector) DetectFromExtractor(ge *graphicsstate.GraphicsExtractor) []*GridHypothesis {
	// Get classified lines
	gridLines := ge.GetGridLines()

	return gd.DetectFromLines(gridLines.Horizontals, gridLines.Verticals)
}

// DetectFromLines detects grid hypotheses from horizontal and vertical lines
func (gd *GridDetector) DetectFromLines(horizontals, verticals []graphicsstate.ExtractedLine) []*GridHypothesis {
	// Filter lines by minimum length
	horizontals = gd.filterByLength(horizontals)
	verticals = gd.filterByLength(verticals)

	if len(horizontals) < gd.MinAlignedLines || len(verticals) < gd.MinAlignedLines {
		return nil
	}

	// Group aligned lines
	hGroups := gd.groupAlignedLines(horizontals, true)
	vGroups := gd.groupAlignedLines(verticals, false)

	if len(hGroups) < gd.MinAlignedLines || len(vGroups) < gd.MinAlignedLines {
		return nil
	}

	// Find intersecting line groups that could form grids
	return gd.findGrids(hGroups, vGroups)
}

// filterByLength filters lines by minimum length
func (gd *GridDetector) filterByLength(lines []graphicsstate.ExtractedLine) []graphicsstate.ExtractedLine {
	result := make([]graphicsstate.ExtractedLine, 0, len(lines))
	for _, line := range lines {
		length := gd.lineLength(line)
		if length >= gd.MinLineLength {
			result = append(result, line)
		}
	}
	return result
}

// lineLength calculates the length of a line
func (gd *GridDetector) lineLength(line graphicsstate.ExtractedLine) float64 {
	dx := line.End.X - line.Start.X
	dy := line.End.Y - line.Start.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// groupAlignedLines groups lines that are aligned on the same axis
func (gd *GridDetector) groupAlignedLines(lines []graphicsstate.ExtractedLine, isHorizontal bool) []AlignedLineGroup {
	if len(lines) == 0 {
		return nil
	}

	// Sort lines by position
	positions := make([]float64, len(lines))
	for i, line := range lines {
		if isHorizontal {
			positions[i] = (line.Start.Y + line.End.Y) / 2
		} else {
			positions[i] = (line.Start.X + line.End.X) / 2
		}
	}

	// Create index for sorting
	indices := make([]int, len(lines))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return positions[indices[i]] < positions[indices[j]]
	})

	// Group lines by position
	var groups []AlignedLineGroup
	currentGroup := AlignedLineGroup{
		Position: positions[indices[0]],
		Lines:    []graphicsstate.ExtractedLine{lines[indices[0]]},
	}

	for i := 1; i < len(indices); i++ {
		idx := indices[i]
		pos := positions[idx]

		if pos-currentGroup.Position <= gd.AlignmentTolerance {
			// Add to current group
			currentGroup.Lines = append(currentGroup.Lines, lines[idx])
			// Update position to average
			currentGroup.Position = (currentGroup.Position*float64(len(currentGroup.Lines)-1) + pos) / float64(len(currentGroup.Lines))
		} else {
			// Finish current group and start new one
			gd.finalizeGroup(&currentGroup, isHorizontal)
			groups = append(groups, currentGroup)

			currentGroup = AlignedLineGroup{
				Position: pos,
				Lines:    []graphicsstate.ExtractedLine{lines[idx]},
			}
		}
	}

	// Don't forget the last group
	gd.finalizeGroup(&currentGroup, isHorizontal)
	groups = append(groups, currentGroup)

	return groups
}

// finalizeGroup calculates final metrics for an aligned line group
func (gd *GridDetector) finalizeGroup(group *AlignedLineGroup, isHorizontal bool) {
	if len(group.Lines) == 0 {
		return
	}

	group.TotalLength = 0
	group.MinExtent = math.MaxFloat64
	group.MaxExtent = -math.MaxFloat64

	for _, line := range group.Lines {
		group.TotalLength += gd.lineLength(line)

		var minVal, maxVal float64
		if isHorizontal {
			// For horizontal lines, extent is X range
			minVal = math.Min(line.Start.X, line.End.X)
			maxVal = math.Max(line.Start.X, line.End.X)
		} else {
			// For vertical lines, extent is Y range
			minVal = math.Min(line.Start.Y, line.End.Y)
			maxVal = math.Max(line.Start.Y, line.End.Y)
		}

		if minVal < group.MinExtent {
			group.MinExtent = minVal
		}
		if maxVal > group.MaxExtent {
			group.MaxExtent = maxVal
		}
	}
}

// findGrids finds grid hypotheses from aligned line groups
func (gd *GridDetector) findGrids(hGroups, vGroups []AlignedLineGroup) []*GridHypothesis {
	// Find overlapping regions between horizontal and vertical line groups
	// A grid exists where horizontal and vertical lines intersect

	var hypotheses []*GridHypothesis

	// For horizontal lines: Position is Y, Extent (Min/Max) is X range
	// For vertical lines: Position is X, Extent (Min/Max) is Y range

	// Grid bounds:
	// - Left/Right come from vertical line positions (X coordinates)
	// - Top/Bottom come from horizontal line positions (Y coordinates)
	gridLeft := gd.minPosition(vGroups)
	gridRight := gd.maxPosition(vGroups)
	gridBottom := gd.minPosition(hGroups)
	gridTop := gd.maxPosition(hGroups)

	if gridRight <= gridLeft || gridTop <= gridBottom {
		return nil // No valid grid
	}

	// Now filter lines that actually span a significant portion of the grid
	// Horizontal lines should span from near gridLeft to near gridRight (in X)
	// Vertical lines should span from near gridBottom to near gridTop (in Y)
	relevantH := gd.filterGroupsByExtent(hGroups, gridLeft, gridRight, true)
	relevantV := gd.filterGroupsByExtent(vGroups, gridBottom, gridTop, false)

	if len(relevantH) < gd.MinAlignedLines || len(relevantV) < gd.MinAlignedLines {
		return nil
	}

	// Sort horizontal lines by Y (descending - top to bottom in PDF coords)
	sort.Slice(relevantH, func(i, j int) bool {
		return relevantH[i].Position > relevantH[j].Position
	})

	// Sort vertical lines by X (ascending - left to right)
	sort.Slice(relevantV, func(i, j int) bool {
		return relevantV[i].Position < relevantV[j].Position
	})

	// Create grid hypothesis
	hypothesis := &GridHypothesis{
		BBox: model.BBox{
			X:      gridLeft,
			Y:      gridBottom,
			Width:  gridRight - gridLeft,
			Height: gridTop - gridBottom,
		},
		HorizontalLines: make([]float64, len(relevantH)),
		VerticalLines:   make([]float64, len(relevantV)),
		Rows:            len(relevantH) - 1,
		Cols:            len(relevantV) - 1,
	}

	for i, g := range relevantH {
		hypothesis.HorizontalLines[i] = g.Position
	}
	for i, g := range relevantV {
		hypothesis.VerticalLines[i] = g.Position
	}

	// Check for complete borders using the actual min/max positions
	if len(relevantH) > 0 {
		hypothesis.HasTopBorder = math.Abs(relevantH[0].Position-gridTop) < gd.AlignmentTolerance
		hypothesis.HasBottomBorder = math.Abs(relevantH[len(relevantH)-1].Position-gridBottom) < gd.AlignmentTolerance
	}
	if len(relevantV) > 0 {
		hypothesis.HasLeftBorder = math.Abs(relevantV[0].Position-gridLeft) < gd.AlignmentTolerance
		hypothesis.HasRightBorder = math.Abs(relevantV[len(relevantV)-1].Position-gridRight) < gd.AlignmentTolerance
	}

	// Calculate confidence
	hypothesis.Confidence = gd.calculateConfidence(hypothesis, relevantH, relevantV)

	if hypothesis.Rows > 0 && hypothesis.Cols > 0 {
		hypotheses = append(hypotheses, hypothesis)
	}

	return hypotheses
}

// groupExtentRange returns min and max extents across all groups
func (gd *GridDetector) groupExtentRange(groups []AlignedLineGroup) (min, max float64) {
	if len(groups) == 0 {
		return 0, 0
	}

	min = groups[0].MinExtent
	max = groups[0].MaxExtent

	for _, g := range groups[1:] {
		if g.MinExtent < min {
			min = g.MinExtent
		}
		if g.MaxExtent > max {
			max = g.MaxExtent
		}
	}

	return
}

// minPosition returns the minimum position across all groups
func (gd *GridDetector) minPosition(groups []AlignedLineGroup) float64 {
	if len(groups) == 0 {
		return 0
	}

	min := groups[0].Position
	for _, g := range groups[1:] {
		if g.Position < min {
			min = g.Position
		}
	}
	return min
}

// maxPosition returns the maximum position across all groups
func (gd *GridDetector) maxPosition(groups []AlignedLineGroup) float64 {
	if len(groups) == 0 {
		return 0
	}

	max := groups[0].Position
	for _, g := range groups[1:] {
		if g.Position > max {
			max = g.Position
		}
	}
	return max
}

// filterGroupsByExtent filters groups that have lines spanning the given extent
func (gd *GridDetector) filterGroupsByExtent(groups []AlignedLineGroup, minExtent, maxExtent float64, useLineExtent bool) []AlignedLineGroup {
	var result []AlignedLineGroup

	for _, g := range groups {
		// Check if this group has lines covering a significant portion of the extent
		coverage := g.MaxExtent - g.MinExtent
		requiredCoverage := (maxExtent - minExtent) * 0.5 // At least 50% coverage

		if coverage >= requiredCoverage {
			// Also check overlap with the extent
			overlapMin := math.Max(g.MinExtent, minExtent)
			overlapMax := math.Min(g.MaxExtent, maxExtent)
			if overlapMax > overlapMin {
				result = append(result, g)
			}
		}
	}

	return result
}

// calculateConfidence calculates a confidence score for a grid hypothesis
func (gd *GridDetector) calculateConfidence(h *GridHypothesis, hGroups, vGroups []AlignedLineGroup) float64 {
	score := 0.0

	// Factor 1: Number of cells (more cells = higher confidence, up to a point)
	cellCount := h.Rows * h.Cols
	if cellCount >= 4 {
		score += 0.2
	}
	if cellCount >= 9 {
		score += 0.1
	}

	// Factor 2: Grid regularity (similar row heights and column widths)
	regularity := gd.calculateRegularity(h)
	score += regularity * 0.3

	// Factor 3: Border completeness
	borderScore := 0.0
	if h.HasTopBorder {
		borderScore += 0.25
	}
	if h.HasBottomBorder {
		borderScore += 0.25
	}
	if h.HasLeftBorder {
		borderScore += 0.25
	}
	if h.HasRightBorder {
		borderScore += 0.25
	}
	score += borderScore * 0.2

	// Factor 4: Line coverage (how much of the grid is covered by actual lines)
	coverage := gd.calculateLineCoverage(h, hGroups, vGroups)
	score += coverage * 0.2

	return math.Min(1.0, score)
}

// calculateRegularity measures how regular the grid spacing is
func (gd *GridDetector) calculateRegularity(h *GridHypothesis) float64 {
	// Calculate variance in row heights
	rowScore := 1.0
	if h.Rows > 1 {
		rowHeights := make([]float64, h.Rows)
		for i := 0; i < h.Rows; i++ {
			rowHeights[i] = h.HorizontalLines[i] - h.HorizontalLines[i+1]
		}
		rowCV := coefficientOfVariation(rowHeights)
		rowScore = math.Max(0, 1-rowCV)
	}

	// Calculate variance in column widths
	colScore := 1.0
	if h.Cols > 1 {
		colWidths := make([]float64, h.Cols)
		for i := 0; i < h.Cols; i++ {
			colWidths[i] = h.VerticalLines[i+1] - h.VerticalLines[i]
		}
		colCV := coefficientOfVariation(colWidths)
		colScore = math.Max(0, 1-colCV)
	}

	return (rowScore + colScore) / 2
}

// calculateLineCoverage calculates what fraction of grid lines have actual drawn lines
func (gd *GridDetector) calculateLineCoverage(h *GridHypothesis, hGroups, vGroups []AlignedLineGroup) float64 {
	totalExpected := float64(len(h.HorizontalLines) + len(h.VerticalLines))
	if totalExpected == 0 {
		return 0
	}

	actualCount := float64(len(hGroups) + len(vGroups))
	return math.Min(1.0, actualCount/totalExpected)
}

// coefficientOfVariation calculates CV (std dev / mean)
func coefficientOfVariation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	m := 0.0
	for _, v := range values {
		m += v
	}
	m /= float64(len(values))

	if m == 0 {
		return 0
	}

	v := 0.0
	for _, val := range values {
		diff := val - m
		v += diff * diff
	}
	v /= float64(len(values))

	return math.Sqrt(v) / m
}

// ToTableGrid converts a grid hypothesis to a model.TableGrid
func (h *GridHypothesis) ToTableGrid() *model.TableGrid {
	grid := model.NewTableGrid()
	grid.Rows = make([]float64, len(h.HorizontalLines))
	copy(grid.Rows, h.HorizontalLines)

	grid.Cols = make([]float64, len(h.VerticalLines))
	copy(grid.Cols, h.VerticalLines)

	// Set line presence flags
	grid.HasHLines = make([]bool, len(h.HorizontalLines))
	for i := range grid.HasHLines {
		grid.HasHLines[i] = true // All horizontal lines in hypothesis are real
	}

	grid.HasVLines = make([]bool, len(h.VerticalLines))
	for i := range grid.HasVLines {
		grid.HasVLines[i] = true // All vertical lines in hypothesis are real
	}

	return grid
}

// GridDetectionResult contains the result of grid detection
type GridDetectionResult struct {
	// All detected grid hypotheses, sorted by confidence
	Hypotheses []*GridHypothesis

	// Statistics about the detection
	TotalHorizontalLines int
	TotalVerticalLines   int
	AlignedHGroups       int
	AlignedVGroups       int
}

// DetectGrids is a convenience function for grid detection
func DetectGrids(ge *graphicsstate.GraphicsExtractor) *GridDetectionResult {
	detector := NewGridDetector()
	gridLines := ge.GetGridLines()

	result := &GridDetectionResult{
		TotalHorizontalLines: len(gridLines.Horizontals),
		TotalVerticalLines:   len(gridLines.Verticals),
	}

	// Filter lines
	horizontals := detector.filterByLength(gridLines.Horizontals)
	verticals := detector.filterByLength(gridLines.Verticals)

	// Group lines
	hGroups := detector.groupAlignedLines(horizontals, true)
	vGroups := detector.groupAlignedLines(verticals, false)

	result.AlignedHGroups = len(hGroups)
	result.AlignedVGroups = len(vGroups)

	// Find grids
	result.Hypotheses = detector.findGrids(hGroups, vGroups)

	// Sort by confidence
	sort.Slice(result.Hypotheses, func(i, j int) bool {
		return result.Hypotheses[i].Confidence > result.Hypotheses[j].Confidence
	})

	return result
}
