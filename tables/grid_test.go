package tables

import (
	"testing"

	"github.com/tsawler/tabula/graphicsstate"
	"github.com/tsawler/tabula/model"
)

func TestNewGridDetector(t *testing.T) {
	gd := NewGridDetector()
	if gd == nil {
		t.Fatal("NewGridDetector returned nil")
	}
	if gd.AlignmentTolerance != 3.0 {
		t.Errorf("Expected AlignmentTolerance 3.0, got %f", gd.AlignmentTolerance)
	}
	if gd.MinAlignedLines != 2 {
		t.Errorf("Expected MinAlignedLines 2, got %d", gd.MinAlignedLines)
	}
}

// Helper to create horizontal lines
func makeHLine(y, x1, x2 float64) graphicsstate.ExtractedLine {
	return graphicsstate.ExtractedLine{
		Start:        model.Point{X: x1, Y: y},
		End:          model.Point{X: x2, Y: y},
		IsHorizontal: true,
		IsVertical:   false,
	}
}

// Helper to create vertical lines
func makeVLine(x, y1, y2 float64) graphicsstate.ExtractedLine {
	return graphicsstate.ExtractedLine{
		Start:        model.Point{X: x, Y: y1},
		End:          model.Point{X: x, Y: y2},
		IsHorizontal: false,
		IsVertical:   true,
	}
}

func TestGridDetector_SimpleGrid(t *testing.T) {
	gd := NewGridDetector()

	// Create a simple 2x2 grid (3 horizontal lines, 3 vertical lines)
	horizontals := []graphicsstate.ExtractedLine{
		makeHLine(100, 0, 200), // Top
		makeHLine(50, 0, 200),  // Middle
		makeHLine(0, 0, 200),   // Bottom
	}

	verticals := []graphicsstate.ExtractedLine{
		makeVLine(0, 0, 100),   // Left
		makeVLine(100, 0, 100), // Middle
		makeVLine(200, 0, 100), // Right
	}

	hypotheses := gd.DetectFromLines(horizontals, verticals)

	if len(hypotheses) == 0 {
		t.Fatal("Expected at least one grid hypothesis")
	}

	h := hypotheses[0]
	if h.Rows != 2 {
		t.Errorf("Expected 2 rows, got %d", h.Rows)
	}
	if h.Cols != 2 {
		t.Errorf("Expected 2 cols, got %d", h.Cols)
	}
}

func TestGridDetector_LargerGrid(t *testing.T) {
	gd := NewGridDetector()

	// Create a 3x4 grid (4 horizontal lines, 5 vertical lines)
	horizontals := []graphicsstate.ExtractedLine{
		makeHLine(300, 0, 400), // Top
		makeHLine(200, 0, 400),
		makeHLine(100, 0, 400),
		makeHLine(0, 0, 400), // Bottom
	}

	verticals := []graphicsstate.ExtractedLine{
		makeVLine(0, 0, 300), // Left
		makeVLine(100, 0, 300),
		makeVLine(200, 0, 300),
		makeVLine(300, 0, 300),
		makeVLine(400, 0, 300), // Right
	}

	hypotheses := gd.DetectFromLines(horizontals, verticals)

	if len(hypotheses) == 0 {
		t.Fatal("Expected at least one grid hypothesis")
	}

	h := hypotheses[0]
	if h.Rows != 3 {
		t.Errorf("Expected 3 rows, got %d", h.Rows)
	}
	if h.Cols != 4 {
		t.Errorf("Expected 4 cols, got %d", h.Cols)
	}
}

func TestGridDetector_AlignedLinesGrouping(t *testing.T) {
	gd := NewGridDetector()
	gd.AlignmentTolerance = 5.0

	// Create lines that are slightly misaligned but should group together
	horizontals := []graphicsstate.ExtractedLine{
		makeHLine(100, 0, 100),
		makeHLine(101, 100, 200), // Slightly off but within tolerance
		makeHLine(50, 0, 200),
		makeHLine(0, 0, 200),
	}

	verticals := []graphicsstate.ExtractedLine{
		makeVLine(0, 0, 100),
		makeVLine(100, 0, 100),
		makeVLine(200, 0, 100),
	}

	hypotheses := gd.DetectFromLines(horizontals, verticals)

	if len(hypotheses) == 0 {
		t.Fatal("Expected at least one grid hypothesis")
	}

	// The two slightly misaligned horizontal lines should group together
	h := hypotheses[0]
	if h.Rows != 2 {
		t.Errorf("Expected 2 rows (misaligned lines should group), got %d", h.Rows)
	}
}

func TestGridDetector_InsufficientLines(t *testing.T) {
	gd := NewGridDetector()

	// Only 1 horizontal line - not enough for a grid
	horizontals := []graphicsstate.ExtractedLine{
		makeHLine(100, 0, 200),
	}

	verticals := []graphicsstate.ExtractedLine{
		makeVLine(0, 0, 100),
		makeVLine(100, 0, 100),
		makeVLine(200, 0, 100),
	}

	hypotheses := gd.DetectFromLines(horizontals, verticals)

	if len(hypotheses) != 0 {
		t.Errorf("Expected no hypotheses with insufficient lines, got %d", len(hypotheses))
	}
}

func TestGridDetector_ShortLinesFiltered(t *testing.T) {
	gd := NewGridDetector()
	gd.MinLineLength = 50.0

	// Create some short lines that should be filtered
	horizontals := []graphicsstate.ExtractedLine{
		makeHLine(100, 0, 200),  // Long enough
		makeHLine(50, 0, 200),   // Long enough
		makeHLine(0, 0, 200),    // Long enough
		makeHLine(75, 100, 110), // Too short (10 points)
	}

	verticals := []graphicsstate.ExtractedLine{
		makeVLine(0, 0, 100),   // Long enough
		makeVLine(100, 0, 100), // Long enough
		makeVLine(200, 0, 100), // Long enough
		makeVLine(150, 50, 60), // Too short
	}

	hypotheses := gd.DetectFromLines(horizontals, verticals)

	if len(hypotheses) == 0 {
		t.Fatal("Expected at least one grid hypothesis")
	}

	// Grid should be formed from the long lines only
	h := hypotheses[0]
	if h.Rows != 2 {
		t.Errorf("Expected 2 rows, got %d", h.Rows)
	}
	if h.Cols != 2 {
		t.Errorf("Expected 2 cols, got %d", h.Cols)
	}
}

func TestGridDetector_BorderDetection(t *testing.T) {
	gd := NewGridDetector()

	// Create a complete bordered grid
	horizontals := []graphicsstate.ExtractedLine{
		makeHLine(100, 0, 200), // Top border
		makeHLine(50, 0, 200),
		makeHLine(0, 0, 200), // Bottom border
	}

	verticals := []graphicsstate.ExtractedLine{
		makeVLine(0, 0, 100), // Left border
		makeVLine(100, 0, 100),
		makeVLine(200, 0, 100), // Right border
	}

	hypotheses := gd.DetectFromLines(horizontals, verticals)

	if len(hypotheses) == 0 {
		t.Fatal("Expected at least one grid hypothesis")
	}

	h := hypotheses[0]
	if !h.HasTopBorder {
		t.Error("Expected HasTopBorder to be true")
	}
	if !h.HasBottomBorder {
		t.Error("Expected HasBottomBorder to be true")
	}
	if !h.HasLeftBorder {
		t.Error("Expected HasLeftBorder to be true")
	}
	if !h.HasRightBorder {
		t.Error("Expected HasRightBorder to be true")
	}
}

func TestGridDetector_Confidence(t *testing.T) {
	gd := NewGridDetector()

	// Create a regular grid (should have high confidence)
	horizontals := []graphicsstate.ExtractedLine{
		makeHLine(100, 0, 200),
		makeHLine(50, 0, 200),
		makeHLine(0, 0, 200),
	}

	verticals := []graphicsstate.ExtractedLine{
		makeVLine(0, 0, 100),
		makeVLine(100, 0, 100),
		makeVLine(200, 0, 100),
	}

	hypotheses := gd.DetectFromLines(horizontals, verticals)

	if len(hypotheses) == 0 {
		t.Fatal("Expected at least one grid hypothesis")
	}

	h := hypotheses[0]
	if h.Confidence < 0.3 {
		t.Errorf("Expected reasonable confidence for regular grid, got %f", h.Confidence)
	}
	if h.Confidence > 1.0 {
		t.Errorf("Confidence should not exceed 1.0, got %f", h.Confidence)
	}
}

func TestGridDetector_BoundingBox(t *testing.T) {
	gd := NewGridDetector()

	horizontals := []graphicsstate.ExtractedLine{
		makeHLine(100, 10, 210),
		makeHLine(50, 10, 210),
		makeHLine(0, 10, 210),
	}

	verticals := []graphicsstate.ExtractedLine{
		makeVLine(10, 0, 100),
		makeVLine(110, 0, 100),
		makeVLine(210, 0, 100),
	}

	hypotheses := gd.DetectFromLines(horizontals, verticals)

	if len(hypotheses) == 0 {
		t.Fatal("Expected at least one grid hypothesis")
	}

	h := hypotheses[0]
	// BBox should encompass the grid
	if h.BBox.X < 0 || h.BBox.X > 20 {
		t.Errorf("Unexpected BBox.X: %f", h.BBox.X)
	}
	if h.BBox.Width < 180 || h.BBox.Width > 220 {
		t.Errorf("Unexpected BBox.Width: %f", h.BBox.Width)
	}
}

func TestGridDetector_ToTableGrid(t *testing.T) {
	gd := NewGridDetector()

	horizontals := []graphicsstate.ExtractedLine{
		makeHLine(100, 0, 200),
		makeHLine(50, 0, 200),
		makeHLine(0, 0, 200),
	}

	verticals := []graphicsstate.ExtractedLine{
		makeVLine(0, 0, 100),
		makeVLine(100, 0, 100),
		makeVLine(200, 0, 100),
	}

	hypotheses := gd.DetectFromLines(horizontals, verticals)

	if len(hypotheses) == 0 {
		t.Fatal("Expected at least one grid hypothesis")
	}

	tableGrid := hypotheses[0].ToTableGrid()
	if tableGrid == nil {
		t.Fatal("ToTableGrid returned nil")
	}

	if tableGrid.RowCount() != 2 {
		t.Errorf("Expected 2 rows in TableGrid, got %d", tableGrid.RowCount())
	}
	if tableGrid.ColCount() != 2 {
		t.Errorf("Expected 2 cols in TableGrid, got %d", tableGrid.ColCount())
	}
}

func TestGridDetector_DetectGridsConvenience(t *testing.T) {
	// Create a graphics extractor with some lines
	gs := graphicsstate.NewGraphicsState()
	ge := graphicsstate.NewPathExtractor(gs)

	// Draw a simple grid
	// Horizontal lines
	ge.MoveTo(0, 100)
	ge.LineTo(200, 100)
	ge.Stroke()

	ge.MoveTo(0, 50)
	ge.LineTo(200, 50)
	ge.Stroke()

	ge.MoveTo(0, 0)
	ge.LineTo(200, 0)
	ge.Stroke()

	// Vertical lines
	ge.MoveTo(0, 0)
	ge.LineTo(0, 100)
	ge.Stroke()

	ge.MoveTo(100, 0)
	ge.LineTo(100, 100)
	ge.Stroke()

	ge.MoveTo(200, 0)
	ge.LineTo(200, 100)
	ge.Stroke()

	// For this test, we'll use DetectFromLines directly
	// In real usage, you would use DetectFromExtractor with a GraphicsExtractor

	gd := NewGridDetector()
	hypotheses := gd.DetectFromLines(ge.GetHorizontalLines(), ge.GetVerticalLines())

	if len(hypotheses) == 0 {
		t.Fatal("Expected at least one grid hypothesis from extractor")
	}
}

func TestGridDetector_IrregularGrid(t *testing.T) {
	gd := NewGridDetector()

	// Create an irregular grid with varying row heights
	horizontals := []graphicsstate.ExtractedLine{
		makeHLine(100, 0, 200),
		makeHLine(80, 0, 200), // Small row
		makeHLine(30, 0, 200), // Large row
		makeHLine(0, 0, 200),
	}

	verticals := []graphicsstate.ExtractedLine{
		makeVLine(0, 0, 100),
		makeVLine(100, 0, 100),
		makeVLine(200, 0, 100),
	}

	hypotheses := gd.DetectFromLines(horizontals, verticals)

	if len(hypotheses) == 0 {
		t.Fatal("Expected at least one grid hypothesis")
	}

	h := hypotheses[0]
	// Irregular grid should have lower confidence due to non-uniform spacing
	// But should still be detected
	if h.Rows != 3 {
		t.Errorf("Expected 3 rows, got %d", h.Rows)
	}
}

func TestGridDetector_EmptyInput(t *testing.T) {
	gd := NewGridDetector()

	hypotheses := gd.DetectFromLines(nil, nil)
	if hypotheses != nil {
		t.Errorf("Expected nil for empty input, got %v", hypotheses)
	}

	hypotheses = gd.DetectFromLines([]graphicsstate.ExtractedLine{}, []graphicsstate.ExtractedLine{})
	if hypotheses != nil {
		t.Errorf("Expected nil for empty slices, got %v", hypotheses)
	}
}

func TestCoefficientOfVariation(t *testing.T) {
	// Uniform values should have CV of 0
	uniform := []float64{10, 10, 10, 10}
	cv := coefficientOfVariation(uniform)
	if cv != 0 {
		t.Errorf("Expected CV of 0 for uniform values, got %f", cv)
	}

	// Single value should have CV of 0
	single := []float64{10}
	cv = coefficientOfVariation(single)
	if cv != 0 {
		t.Errorf("Expected CV of 0 for single value, got %f", cv)
	}

	// Variable values should have non-zero CV
	variable := []float64{10, 20, 30, 40}
	cv = coefficientOfVariation(variable)
	if cv <= 0 {
		t.Errorf("Expected positive CV for variable values, got %f", cv)
	}
}

func TestAlignedLineGroup_Extent(t *testing.T) {
	gd := NewGridDetector()

	lines := []graphicsstate.ExtractedLine{
		makeHLine(100, 10, 50),
		makeHLine(100, 40, 90),
		makeHLine(100, 150, 200),
	}

	groups := gd.groupAlignedLines(lines, true)

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}

	g := groups[0]
	if g.MinExtent != 10 {
		t.Errorf("Expected MinExtent 10, got %f", g.MinExtent)
	}
	if g.MaxExtent != 200 {
		t.Errorf("Expected MaxExtent 200, got %f", g.MaxExtent)
	}
}

// Benchmark

func BenchmarkGridDetector(b *testing.B) {
	gd := NewGridDetector()

	// Create a moderately complex grid
	horizontals := make([]graphicsstate.ExtractedLine, 10)
	for i := 0; i < 10; i++ {
		horizontals[i] = makeHLine(float64(i*50), 0, 500)
	}

	verticals := make([]graphicsstate.ExtractedLine, 10)
	for i := 0; i < 10; i++ {
		verticals[i] = makeVLine(float64(i*50), 0, 450)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gd.DetectFromLines(horizontals, verticals)
	}
}
