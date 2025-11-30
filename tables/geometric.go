package tables

import (
	"fmt"
	"math"
	"sort"

	"github.com/tsawler/tabula/model"
)

// GeometricDetector implements table detection using geometric heuristics.
// It analyzes spatial relationships between text fragments to identify tabular
// structures based on alignment patterns, grid regularity, and visible lines.
type GeometricDetector struct {
	config Config
}

// NewGeometricDetector creates a new geometric table detector with default configuration.
func NewGeometricDetector() *GeometricDetector {
	return &GeometricDetector{
		config: DefaultConfig(),
	}
}

// Name returns the detector's identifier ("geometric").
func (d *GeometricDetector) Name() string {
	return "geometric"
}

// Configure sets the detector configuration.
func (d *GeometricDetector) Configure(config Config) error {
	d.config = config
	return nil
}

// Detect finds tables on a page using geometric heuristics. It clusters text
// fragments by spatial proximity, then analyzes each cluster for tabular structure.
func (d *GeometricDetector) Detect(page *model.Page) ([]*model.Table, error) {
	if len(page.RawText) == 0 {
		return nil, nil
	}

	// Step 1: Cluster text fragments by spatial proximity
	clusters := d.clusterFragments(page.RawText)

	var tables []*model.Table

	// Step 2: For each cluster, try to detect table structure
	for _, cluster := range clusters {
		if table := d.detectTableInCluster(cluster, page.RawLines); table != nil {
			tables = append(tables, table)
		}
	}

	return tables, nil
}

// clusterFragments groups text fragments that are spatially close by vertical
// proximity. Fragments separated by more than 50 points vertically start new clusters.
func (d *GeometricDetector) clusterFragments(fragments []model.TextFragment) [][]model.TextFragment {
	if len(fragments) == 0 {
		return nil
	}

	// Simple clustering: group fragments that are vertically close
	sorted := make([]model.TextFragment, len(fragments))
	copy(sorted, fragments)

	// Sort by Y position (top to bottom)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].BBox.Y > sorted[j].BBox.Y
	})

	var clusters [][]model.TextFragment
	currentCluster := []model.TextFragment{sorted[0]}

	for i := 1; i < len(sorted); i++ {
		// If fragment is far from previous cluster, start new cluster
		lastBBox := currentCluster[len(currentCluster)-1].BBox
		currentBBox := sorted[i].BBox

		verticalGap := lastBBox.Y - (currentBBox.Y + currentBBox.Height)

		if verticalGap > 50 { // 50 points gap threshold
			clusters = append(clusters, currentCluster)
			currentCluster = []model.TextFragment{sorted[i]}
		} else {
			currentCluster = append(currentCluster, sorted[i])
		}
	}

	if len(currentCluster) > 0 {
		clusters = append(clusters, currentCluster)
	}

	return clusters
}

// detectTableInCluster attempts to find a table in a cluster of fragments.
// It builds a grid, calculates confidence score, assigns fragments to cells,
// and optionally detects merged cells.
func (d *GeometricDetector) detectTableInCluster(fragments []model.TextFragment, lines []model.Line) *model.Table {
	if len(fragments) < d.config.MinRows*d.config.MinCols {
		return nil
	}

	// Step 1: Build grid from fragments and lines
	grid := d.buildGrid(fragments, lines)

	if grid == nil || grid.RowCount() < d.config.MinRows || grid.ColCount() < d.config.MinCols {
		return nil
	}

	// Step 2: Calculate confidence score
	confidence := d.calculateConfidence(grid, fragments, lines)

	if confidence < d.config.MinConfidence {
		return nil
	}

	// Step 3: Assign fragments to cells
	table := model.NewTable(grid.RowCount(), grid.ColCount())
	d.assignFragmentsToCells(table, grid, fragments)

	// Step 4: Detect merged cells if enabled
	if d.config.DetectMergedCells {
		d.detectMergedCells(table, grid)
	}

	// Step 5: Set table properties
	table.BBox = d.calculateTableBBox(grid)
	table.Confidence = confidence
	table.HasGrid = d.hasVisibleGrid(grid, lines)

	return table
}

// buildGrid constructs a grid from text fragment positions and detects which
// grid lines have visible graphical lines.
func (d *GeometricDetector) buildGrid(fragments []model.TextFragment, lines []model.Line) *model.TableGrid {
	// Extract unique Y coordinates (row boundaries)
	yCoords := d.extractRowBoundaries(fragments)
	if len(yCoords) < d.config.MinRows+1 {
		return nil
	}

	// Extract unique X coordinates (column boundaries)
	xCoords := d.extractColumnBoundaries(fragments)
	if len(xCoords) < d.config.MinCols+1 {
		return nil
	}

	grid := model.NewTableGrid()
	grid.Rows = yCoords
	grid.Cols = xCoords

	// Detect which grid lines are visible
	grid.HasHLines = d.detectHorizontalLines(yCoords, lines)
	grid.HasVLines = d.detectVerticalLines(xCoords, lines)

	return grid
}

// extractRowBoundaries extracts unique Y coordinates for row boundaries by
// clustering the top and bottom edges of all text fragments.
func (d *GeometricDetector) extractRowBoundaries(fragments []model.TextFragment) []float64 {
	if len(fragments) == 0 {
		return nil
	}

	// Collect all Y positions (tops and bottoms)
	yValues := make([]float64, 0, len(fragments)*2)
	for _, frag := range fragments {
		yValues = append(yValues, frag.BBox.Top(), frag.BBox.Bottom())
	}

	// Sort and cluster
	sort.Float64s(yValues)

	clustered := d.clusterValues(yValues, d.config.AlignmentTolerance)

	// Sort descending (PDF coordinates: top is larger)
	sort.Sort(sort.Reverse(sort.Float64Slice(clustered)))

	return clustered
}

// extractColumnBoundaries extracts unique X coordinates for column boundaries by
// clustering the left and right edges of all text fragments.
func (d *GeometricDetector) extractColumnBoundaries(fragments []model.TextFragment) []float64 {
	if len(fragments) == 0 {
		return nil
	}

	// Collect all X positions (lefts and rights)
	xValues := make([]float64, 0, len(fragments)*2)
	for _, frag := range fragments {
		xValues = append(xValues, frag.BBox.Left(), frag.BBox.Right())
	}

	// Sort and cluster
	sort.Float64s(xValues)

	return d.clusterValues(xValues, d.config.AlignmentTolerance)
}

// clusterValues clusters nearby values within the given tolerance, averaging
// values that fall within the tolerance of the cluster center.
func (d *GeometricDetector) clusterValues(values []float64, tolerance float64) []float64 {
	if len(values) == 0 {
		return nil
	}

	clustered := []float64{values[0]}

	for i := 1; i < len(values); i++ {
		diff := values[i] - clustered[len(clustered)-1]
		if diff > tolerance {
			clustered = append(clustered, values[i])
		} else {
			// Update cluster center with average
			clustered[len(clustered)-1] = (clustered[len(clustered)-1] + values[i]) / 2
		}
	}

	return clustered
}

// detectHorizontalLines determines which row boundaries have visible horizontal
// graphical lines within the alignment tolerance.
func (d *GeometricDetector) detectHorizontalLines(yCoords []float64, lines []model.Line) []bool {
	hasLines := make([]bool, len(yCoords))

	for i, y := range yCoords {
		for _, line := range lines {
			// Check if line is horizontal and close to this Y coordinate
			if math.Abs(line.Start.Y-y) < d.config.AlignmentTolerance &&
				math.Abs(line.End.Y-y) < d.config.AlignmentTolerance {
				hasLines[i] = true
				break
			}
		}
	}

	return hasLines
}

// detectVerticalLines determines which column boundaries have visible vertical
// graphical lines within the alignment tolerance.
func (d *GeometricDetector) detectVerticalLines(xCoords []float64, lines []model.Line) []bool {
	hasLines := make([]bool, len(xCoords))

	for i, x := range xCoords {
		for _, line := range lines {
			// Check if line is vertical and close to this X coordinate
			if math.Abs(line.Start.X-x) < d.config.AlignmentTolerance &&
				math.Abs(line.End.X-x) < d.config.AlignmentTolerance {
				hasLines[i] = true
				break
			}
		}
	}

	return hasLines
}

// calculateConfidence computes a confidence score (0.0-1.0) for the detected table.
// The score combines grid regularity (30%), alignment quality (30%), line presence (20%),
// and cell occupancy (20%).
func (d *GeometricDetector) calculateConfidence(grid *model.TableGrid, fragments []model.TextFragment, lines []model.Line) float64 {
	score := 0.0
	maxScore := 0.0

	// Factor 1: Grid regularity (0-0.3)
	regularity := d.calculateGridRegularity(grid)
	score += regularity * 0.3
	maxScore += 0.3

	// Factor 2: Alignment quality (0-0.3)
	alignment := d.calculateAlignmentQuality(fragments, grid)
	score += alignment * 0.3
	maxScore += 0.3

	// Factor 3: Line presence (0-0.2)
	lineScore := d.calculateLineScore(grid)
	score += lineScore * 0.2
	maxScore += 0.2

	// Factor 4: Cell occupancy (0-0.2)
	occupancy := d.calculateCellOccupancy(fragments, grid)
	score += occupancy * 0.2
	maxScore += 0.2

	if maxScore == 0 {
		return 0
	}

	return score / maxScore
}

// calculateGridRegularity measures how regular the grid is by computing the
// coefficient of variation of row heights and column widths. Lower variance
// results in a higher score.
func (d *GeometricDetector) calculateGridRegularity(grid *model.TableGrid) float64 {
	if grid.RowCount() < 2 || grid.ColCount() < 2 {
		return 0
	}

	// Calculate variance in row heights
	rowHeights := make([]float64, grid.RowCount())
	for i := 0; i < grid.RowCount(); i++ {
		rowHeights[i] = grid.Rows[i] - grid.Rows[i+1]
	}
	rowVariance := variance(rowHeights)

	// Calculate variance in column widths
	colWidths := make([]float64, grid.ColCount())
	for i := 0; i < grid.ColCount(); i++ {
		colWidths[i] = grid.Cols[i+1] - grid.Cols[i]
	}
	colVariance := variance(colWidths)

	// Lower variance = more regular = higher score
	// Normalize by mean to get coefficient of variation
	rowCV := math.Sqrt(rowVariance) / mean(rowHeights)
	colCV := math.Sqrt(colVariance) / mean(colWidths)

	// Score decreases with CV (more irregular = lower score)
	rowScore := math.Max(0, 1-rowCV)
	colScore := math.Max(0, 1-colCV)

	return (rowScore + colScore) / 2
}

// calculateAlignmentQuality measures the fraction of text fragments whose edges
// align well with grid lines (at least 2 edges within tolerance).
func (d *GeometricDetector) calculateAlignmentQuality(fragments []model.TextFragment, grid *model.TableGrid) float64 {
	if len(fragments) == 0 {
		return 0
	}

	alignedCount := 0
	for _, frag := range fragments {
		if d.isAlignedToGrid(frag, grid) {
			alignedCount++
		}
	}

	return float64(alignedCount) / float64(len(fragments))
}

// isAlignedToGrid reports whether a fragment has at least 2 of its 4 edges
// aligned to grid lines within tolerance.
func (d *GeometricDetector) isAlignedToGrid(frag model.TextFragment, grid *model.TableGrid) bool {
	// Check if fragment edges are close to grid lines
	leftAligned := d.isNearGridLine(frag.BBox.Left(), grid.Cols)
	rightAligned := d.isNearGridLine(frag.BBox.Right(), grid.Cols)
	topAligned := d.isNearGridLine(frag.BBox.Top(), grid.Rows)
	bottomAligned := d.isNearGridLine(frag.BBox.Bottom(), grid.Rows)

	// At least 2 edges should align
	alignedCount := 0
	if leftAligned {
		alignedCount++
	}
	if rightAligned {
		alignedCount++
	}
	if topAligned {
		alignedCount++
	}
	if bottomAligned {
		alignedCount++
	}

	return alignedCount >= 2
}

// isNearGridLine reports whether a value is within 2x the alignment tolerance
// of any grid line.
func (d *GeometricDetector) isNearGridLine(value float64, gridLines []float64) bool {
	for _, line := range gridLines {
		if math.Abs(value-line) < d.config.AlignmentTolerance*2 {
			return true
		}
	}
	return false
}

// calculateLineScore measures the fraction of grid boundaries that have visible
// graphical lines, averaging horizontal and vertical line coverage.
func (d *GeometricDetector) calculateLineScore(grid *model.TableGrid) float64 {
	if len(grid.HasHLines) == 0 || len(grid.HasVLines) == 0 {
		return 0
	}

	hLineCount := 0
	for _, has := range grid.HasHLines {
		if has {
			hLineCount++
		}
	}

	vLineCount := 0
	for _, has := range grid.HasVLines {
		if has {
			vLineCount++
		}
	}

	hScore := float64(hLineCount) / float64(len(grid.HasHLines))
	vScore := float64(vLineCount) / float64(len(grid.HasVLines))

	return (hScore + vScore) / 2
}

// calculateCellOccupancy measures the fraction of grid cells that contain at
// least one text fragment.
func (d *GeometricDetector) calculateCellOccupancy(fragments []model.TextFragment, grid *model.TableGrid) float64 {
	occupiedCells := make(map[string]bool)

	for _, frag := range fragments {
		row, col := d.findCell(frag.BBox.Center(), grid)
		if row >= 0 && col >= 0 {
			key := fmt.Sprintf("%d,%d", row, col)
			occupiedCells[key] = true
		}
	}

	totalCells := grid.RowCount() * grid.ColCount()
	if totalCells == 0 {
		return 0
	}

	return float64(len(occupiedCells)) / float64(totalCells)
}

// assignFragmentsToCells places each text fragment into the appropriate table
// cell based on its center position. Fragments in the same cell are concatenated.
func (d *GeometricDetector) assignFragmentsToCells(table *model.Table, grid *model.TableGrid, fragments []model.TextFragment) {
	for _, frag := range fragments {
		row, col := d.findCell(frag.BBox.Center(), grid)
		if row >= 0 && col >= 0 && row < table.RowCount() && col < table.ColCount() {
			cell := table.GetCell(row, col)
			if cell != nil {
				if cell.Text != "" {
					cell.Text += " "
				}
				cell.Text += frag.Text

				// Expand cell bounding box
				if cell.BBox.IsEmpty() {
					cell.BBox = frag.BBox
				} else {
					cell.BBox = cell.BBox.Union(frag.BBox)
				}
			}
		}
	}
}

// findCell returns the row and column indices of the cell containing the given
// point, or -1 for both if the point is outside the grid.
func (d *GeometricDetector) findCell(p model.Point, grid *model.TableGrid) (row, col int) {
	row = -1
	col = -1

	// Find row
	for i := 0; i < grid.RowCount(); i++ {
		if p.Y <= grid.Rows[i] && p.Y >= grid.Rows[i+1] {
			row = i
			break
		}
	}

	// Find column
	for i := 0; i < grid.ColCount(); i++ {
		if p.X >= grid.Cols[i] && p.X <= grid.Cols[i+1] {
			col = i
			break
		}
	}

	return row, col
}

// detectMergedCells detects cells that span multiple rows or columns by checking
// if a cell's content bounding box intersects adjacent grid cells.
func (d *GeometricDetector) detectMergedCells(table *model.Table, grid *model.TableGrid) {
	// Simple heuristic: if a cell's content bbox spans multiple grid cells,
	// it's likely a merged cell
	for i := 0; i < table.RowCount(); i++ {
		for j := 0; j < table.ColCount(); j++ {
			cell := table.GetCell(i, j)
			if cell == nil || cell.BBox.IsEmpty() {
				continue
			}

			// Check row span
			for k := i + 1; k < table.RowCount(); k++ {
				cellBBox := grid.GetCellBBox(k, j)
				if cell.BBox.Intersects(cellBBox) {
					cell.RowSpan = k - i + 1
				} else {
					break
				}
			}

			// Check column span
			for k := j + 1; k < table.ColCount(); k++ {
				cellBBox := grid.GetCellBBox(i, k)
				if cell.BBox.Intersects(cellBBox) {
					cell.ColSpan = k - j + 1
				} else {
					break
				}
			}
		}
	}
}

// calculateTableBBox computes the overall bounding box of the table from the grid.
func (d *GeometricDetector) calculateTableBBox(grid *model.TableGrid) model.BBox {
	if grid.RowCount() == 0 || grid.ColCount() == 0 {
		return model.BBox{}
	}

	return model.BBox{
		X:      grid.Cols[0],
		Y:      grid.Rows[len(grid.Rows)-1],
		Width:  grid.Cols[len(grid.Cols)-1] - grid.Cols[0],
		Height: grid.Rows[0] - grid.Rows[len(grid.Rows)-1],
	}
}

// hasVisibleGrid reports whether at least 50% of the table's grid boundaries
// have visible graphical lines.
func (d *GeometricDetector) hasVisibleGrid(grid *model.TableGrid, lines []model.Line) bool {
	// Consider table to have grid if at least 50% of lines are visible
	totalLines := len(grid.HasHLines) + len(grid.HasVLines)
	if totalLines == 0 {
		return false
	}

	visibleCount := 0
	for _, has := range grid.HasHLines {
		if has {
			visibleCount++
		}
	}
	for _, has := range grid.HasVLines {
		if has {
			visibleCount++
		}
	}

	return float64(visibleCount)/float64(totalLines) >= 0.5
}

// Utility functions

// mean computes the arithmetic mean of a slice of float64 values.
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// variance computes the population variance of a slice of float64 values.
func variance(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := mean(values)
	sum := 0.0
	for _, v := range values {
		diff := v - m
		sum += diff * diff
	}
	return sum / float64(len(values))
}
