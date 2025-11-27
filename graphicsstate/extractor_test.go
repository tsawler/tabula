package graphicsstate

import (
	"testing"

	"github.com/tsawler/tabula/contentstream"
	"github.com/tsawler/tabula/core"
)

func TestNewGraphicsExtractor(t *testing.T) {
	ge := NewGraphicsExtractor()
	if ge == nil {
		t.Fatal("NewGraphicsExtractor returned nil")
	}
	if ge.MinLineLength != 1.0 {
		t.Errorf("Expected MinLineLength 1.0, got %f", ge.MinLineLength)
	}
}

func TestGraphicsExtractor_SimpleHorizontalLine(t *testing.T) {
	ge := NewGraphicsExtractor()

	// Simulate: 0 100 m 200 100 l S
	ops := []contentstream.Operation{
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(100)}},
		{Operator: "l", Operands: []core.Object{core.Real(200), core.Real(100)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	lines := ge.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}

	line := lines[0]
	if !line.IsHorizontal {
		t.Error("Expected horizontal line")
	}
	if line.Start.X != 0 || line.Start.Y != 100 {
		t.Errorf("Expected start (0, 100), got (%f, %f)", line.Start.X, line.Start.Y)
	}
	if line.End.X != 200 || line.End.Y != 100 {
		t.Errorf("Expected end (200, 100), got (%f, %f)", line.End.X, line.End.Y)
	}
}

func TestGraphicsExtractor_VerticalLine(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		{Operator: "m", Operands: []core.Object{core.Real(100), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(200)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	lines := ge.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}

	if !lines[0].IsVertical {
		t.Error("Expected vertical line")
	}
}

func TestGraphicsExtractor_Rectangle(t *testing.T) {
	ge := NewGraphicsExtractor()

	// Simulate: 100 100 200 150 re S
	ops := []contentstream.Operation{
		{Operator: "re", Operands: []core.Object{core.Real(100), core.Real(100), core.Real(200), core.Real(150)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rects := ge.GetRectangles()
	if len(rects) != 1 {
		t.Fatalf("Expected 1 rectangle, got %d", len(rects))
	}

	rect := rects[0]
	if !rect.IsStroked {
		t.Error("Expected stroked rectangle")
	}
	if rect.BBox.X != 100 || rect.BBox.Y != 100 {
		t.Errorf("Expected BBox at (100, 100), got (%f, %f)", rect.BBox.X, rect.BBox.Y)
	}
	if rect.BBox.Width != 200 || rect.BBox.Height != 150 {
		t.Errorf("Expected size (200, 150), got (%f, %f)", rect.BBox.Width, rect.BBox.Height)
	}
}

func TestGraphicsExtractor_FilledRectangle(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		{Operator: "rg", Operands: []core.Object{core.Real(0), core.Real(1), core.Real(0)}}, // Green fill
		{Operator: "re", Operands: []core.Object{core.Real(0), core.Real(0), core.Real(100), core.Real(100)}},
		{Operator: "f", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rects := ge.GetRectangles()
	if len(rects) != 1 {
		t.Fatalf("Expected 1 rectangle, got %d", len(rects))
	}

	rect := rects[0]
	if !rect.IsFilled {
		t.Error("Expected filled rectangle")
	}
	if rect.FillColor[1] != 1.0 {
		t.Errorf("Expected green fill color, got %v", rect.FillColor)
	}
}

func TestGraphicsExtractor_GraphicsState(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		{Operator: "w", Operands: []core.Object{core.Real(2.5)}},                            // Line width
		{Operator: "RG", Operands: []core.Object{core.Real(1), core.Real(0), core.Real(0)}}, // Red stroke
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(0)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	lines := ge.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}

	line := lines[0]
	if line.Width != 2.5 {
		t.Errorf("Expected line width 2.5, got %f", line.Width)
	}
	if line.Color[0] != 1.0 {
		t.Errorf("Expected red stroke color, got %v", line.Color)
	}
}

func TestGraphicsExtractor_SaveRestore(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		{Operator: "w", Operands: []core.Object{core.Real(1)}}, // Line width = 1
		{Operator: "q", Operands: nil},                         // Save state
		{Operator: "w", Operands: []core.Object{core.Real(5)}}, // Line width = 5
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(0)}},
		{Operator: "S", Operands: nil},
		{Operator: "Q", Operands: nil}, // Restore state
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(50)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(50)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	lines := ge.GetLines()
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}

	// First line should have width 5
	if lines[0].Width != 5.0 {
		t.Errorf("First line: expected width 5.0, got %f", lines[0].Width)
	}

	// Second line should have width 1 (restored)
	if lines[1].Width != 1.0 {
		t.Errorf("Second line: expected width 1.0 (restored), got %f", lines[1].Width)
	}
}

func TestGraphicsExtractor_GrayColors(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		{Operator: "G", Operands: []core.Object{core.Real(0.5)}},  // Gray stroke
		{Operator: "g", Operands: []core.Object{core.Real(0.75)}}, // Gray fill
		{Operator: "re", Operands: []core.Object{core.Real(0), core.Real(0), core.Real(100), core.Real(100)}},
		{Operator: "B", Operands: nil}, // Fill and stroke
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rects := ge.GetRectangles()
	if len(rects) != 1 {
		t.Fatalf("Expected 1 rectangle, got %d", len(rects))
	}

	rect := rects[0]
	// Stroke color should be gray 0.5
	if rect.StrokeColor[0] != 0.5 || rect.StrokeColor[1] != 0.5 || rect.StrokeColor[2] != 0.5 {
		t.Errorf("Expected gray stroke (0.5), got %v", rect.StrokeColor)
	}
	// Fill color should be gray 0.75
	if rect.FillColor[0] != 0.75 || rect.FillColor[1] != 0.75 || rect.FillColor[2] != 0.75 {
		t.Errorf("Expected gray fill (0.75), got %v", rect.FillColor)
	}
}

func TestGraphicsExtractor_CMYKColors(t *testing.T) {
	ge := NewGraphicsExtractor()

	// Pure cyan in CMYK (1, 0, 0, 0) should give (0, 1, 1) in RGB
	ops := []contentstream.Operation{
		{Operator: "K", Operands: []core.Object{core.Real(1), core.Real(0), core.Real(0), core.Real(0)}},
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(0)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	lines := ge.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}

	// Cyan: C=1, M=0, Y=0, K=0 → R=0, G=1, B=1
	if lines[0].Color[0] != 0 || lines[0].Color[1] != 1 || lines[0].Color[2] != 1 {
		t.Errorf("Expected cyan RGB (0, 1, 1), got %v", lines[0].Color)
	}
}

func TestGraphicsExtractor_CurveTo(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "c", Operands: []core.Object{
			core.Real(50), core.Real(100),
			core.Real(100), core.Real(100),
			core.Real(150), core.Real(0),
		}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Curve should be approximated as a single line
	lines := ge.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line (curve approximation), got %d", len(lines))
	}
}

func TestGraphicsExtractor_ClosePath(t *testing.T) {
	ge := NewGraphicsExtractor()

	// Draw a triangle and close it
	ops := []contentstream.Operation{
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(50), core.Real(100)}},
		{Operator: "s", Operands: nil}, // Close and stroke
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Triangle has 3 sides
	lines := ge.GetLines()
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines for triangle, got %d", len(lines))
	}
}

func TestGraphicsExtractor_Transform(t *testing.T) {
	ge := NewGraphicsExtractor()

	// Apply 2x scale transform
	ops := []contentstream.Operation{
		{Operator: "cm", Operands: []core.Object{
			core.Real(2), core.Real(0),
			core.Real(0), core.Real(2),
			core.Real(0), core.Real(0),
		}},
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(0)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	lines := ge.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}

	// Line should be scaled to (0,0) to (200,0)
	if lines[0].End.X != 200 {
		t.Errorf("Expected end.X = 200 (2x scaled), got %f", lines[0].End.X)
	}
}

func TestGraphicsExtractor_ExtractFromBytes(t *testing.T) {
	ge := NewGraphicsExtractor()

	// Raw content stream
	data := []byte("0 100 m 200 100 l S")

	err := ge.ExtractFromBytes(data)
	if err != nil {
		t.Fatalf("ExtractFromBytes failed: %v", err)
	}

	lines := ge.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}

	if !lines[0].IsHorizontal {
		t.Error("Expected horizontal line")
	}
}

func TestGraphicsExtractor_ToModelLines(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		{Operator: "RG", Operands: []core.Object{core.Real(1), core.Real(0), core.Real(0)}},
		{Operator: "w", Operands: []core.Object{core.Real(2)}},
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(0)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	modelLines := ge.ToModelLines()
	if len(modelLines) != 1 {
		t.Fatalf("Expected 1 model line, got %d", len(modelLines))
	}

	ml := modelLines[0]
	if ml.Width != 2 {
		t.Errorf("Expected width 2, got %f", ml.Width)
	}
	if ml.Color.R != 255 {
		t.Errorf("Expected red color (R=255), got R=%d", ml.Color.R)
	}
	if ml.IsRect {
		t.Error("Expected IsRect=false for line")
	}
}

func TestGraphicsExtractor_ToModelRectangles(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		{Operator: "rg", Operands: []core.Object{core.Real(0), core.Real(1), core.Real(0)}},
		{Operator: "re", Operands: []core.Object{core.Real(10), core.Real(20), core.Real(100), core.Real(50)}},
		{Operator: "f", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	modelRects := ge.ToModelRectangles()
	if len(modelRects) != 1 {
		t.Fatalf("Expected 1 model rectangle, got %d", len(modelRects))
	}

	mr := modelRects[0]
	if !mr.IsRect {
		t.Error("Expected IsRect=true for rectangle")
	}
	if !mr.RectFill {
		t.Error("Expected RectFill=true for filled rectangle")
	}
	if mr.Color.G != 255 {
		t.Errorf("Expected green fill color (G=255), got G=%d", mr.Color.G)
	}
}

func TestGraphicsExtractor_ClassifyLines(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		// Horizontal line
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(100)}},
		{Operator: "l", Operands: []core.Object{core.Real(200), core.Real(100)}},
		{Operator: "S", Operands: nil},
		// Vertical line
		{Operator: "m", Operands: []core.Object{core.Real(100), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(200)}},
		{Operator: "S", Operands: nil},
		// Diagonal line
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(100)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	classification := ge.ClassifyLines()

	if len(classification.HorizontalLines) != 1 {
		t.Errorf("Expected 1 horizontal line, got %d", len(classification.HorizontalLines))
	}
	if len(classification.VerticalLines) != 1 {
		t.Errorf("Expected 1 vertical line, got %d", len(classification.VerticalLines))
	}
	if len(classification.DiagonalLines) != 1 {
		t.Errorf("Expected 1 diagonal line, got %d", len(classification.DiagonalLines))
	}
}

func TestGraphicsExtractor_GetGridLines(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		// 2 horizontal lines
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(100)}},
		{Operator: "l", Operands: []core.Object{core.Real(200), core.Real(100)}},
		{Operator: "S", Operands: nil},
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(200)}},
		{Operator: "l", Operands: []core.Object{core.Real(200), core.Real(200)}},
		{Operator: "S", Operands: nil},
		// 2 vertical lines
		{Operator: "m", Operands: []core.Object{core.Real(50), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(50), core.Real(300)}},
		{Operator: "S", Operands: nil},
		{Operator: "m", Operands: []core.Object{core.Real(150), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(150), core.Real(300)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	grid := ge.GetGridLines()

	if len(grid.Horizontals) != 2 {
		t.Errorf("Expected 2 horizontal grid lines, got %d", len(grid.Horizontals))
	}
	if len(grid.Verticals) != 2 {
		t.Errorf("Expected 2 vertical grid lines, got %d", len(grid.Verticals))
	}
}

func TestGraphicsExtractor_GetStatistics(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		// Line
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(0)}},
		{Operator: "S", Operands: nil},
		// Stroked rectangle
		{Operator: "re", Operands: []core.Object{core.Real(0), core.Real(100), core.Real(50), core.Real(50)}},
		{Operator: "S", Operands: nil},
		// Filled rectangle
		{Operator: "re", Operands: []core.Object{core.Real(100), core.Real(100), core.Real(50), core.Real(50)}},
		{Operator: "f", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	stats := ge.GetStatistics()

	if stats.TotalLines != 1 {
		t.Errorf("Expected 1 total line, got %d", stats.TotalLines)
	}
	if stats.HorizontalLines != 1 {
		t.Errorf("Expected 1 horizontal line, got %d", stats.HorizontalLines)
	}
	if stats.TotalRectangles != 2 {
		t.Errorf("Expected 2 total rectangles, got %d", stats.TotalRectangles)
	}
	if stats.StrokedRectangles != 1 {
		t.Errorf("Expected 1 stroked rectangle, got %d", stats.StrokedRectangles)
	}
	if stats.FilledRectangles != 1 {
		t.Errorf("Expected 1 filled rectangle, got %d", stats.FilledRectangles)
	}
}

func TestGraphicsExtractor_Clear(t *testing.T) {
	ge := NewGraphicsExtractor()

	ops := []contentstream.Operation{
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(0)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(ge.GetLines()) != 1 {
		t.Fatal("Expected 1 line before clear")
	}

	ge.Clear()

	if len(ge.GetLines()) != 0 {
		t.Error("Expected 0 lines after clear")
	}
}

func TestGraphicsExtractor_Filtering(t *testing.T) {
	ge := NewGraphicsExtractor()
	ge.MinLineLength = 50 // Only lines >= 50 points
	ge.MinRectWidth = 30
	ge.MinRectHeight = 30

	ops := []contentstream.Operation{
		// Short line (length 10)
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(10), core.Real(0)}},
		{Operator: "S", Operands: nil},
		// Long line (length 100)
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(50)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(50)}},
		{Operator: "S", Operands: nil},
		// Small rectangle (20x20)
		{Operator: "re", Operands: []core.Object{core.Real(0), core.Real(100), core.Real(20), core.Real(20)}},
		{Operator: "S", Operands: nil},
		// Large rectangle (50x50)
		{Operator: "re", Operands: []core.Object{core.Real(100), core.Real(100), core.Real(50), core.Real(50)}},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Unfiltered should have all
	if len(ge.GetLines()) != 2 {
		t.Errorf("Expected 2 unfiltered lines, got %d", len(ge.GetLines()))
	}
	if len(ge.GetRectangles()) != 2 {
		t.Errorf("Expected 2 unfiltered rectangles, got %d", len(ge.GetRectangles()))
	}

	// Filtered should respect minimums
	if len(ge.GetFilteredLines()) != 1 {
		t.Errorf("Expected 1 filtered line (>=50), got %d", len(ge.GetFilteredLines()))
	}
	if len(ge.GetFilteredRectangles()) != 1 {
		t.Errorf("Expected 1 filtered rectangle (>=30x30), got %d", len(ge.GetFilteredRectangles()))
	}
}

func TestGraphicsExtractor_AllPathOperators(t *testing.T) {
	ge := NewGraphicsExtractor()

	// Test all path-related operators
	ops := []contentstream.Operation{
		// v operator (curve with first control = current)
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "v", Operands: []core.Object{core.Real(50), core.Real(50), core.Real(100), core.Real(0)}},
		{Operator: "S", Operands: nil},
		// y operator (curve with second control = end)
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(100)}},
		{Operator: "y", Operands: []core.Object{core.Real(50), core.Real(50), core.Real(100), core.Real(100)}},
		{Operator: "S", Operands: nil},
		// h operator (close path)
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(200)}},
		{Operator: "l", Operands: []core.Object{core.Real(50), core.Real(200)}},
		{Operator: "l", Operands: []core.Object{core.Real(25), core.Real(250)}},
		{Operator: "h", Operands: nil},
		{Operator: "S", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should have lines from all operations
	lines := ge.GetLines()
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines from different operators, got %d", len(lines))
	}
}

func TestGraphicsExtractor_AllPaintOperators(t *testing.T) {
	ge := NewGraphicsExtractor()

	// Test all paint operators
	ops := []contentstream.Operation{
		// f* (fill even-odd)
		{Operator: "re", Operands: []core.Object{core.Real(0), core.Real(0), core.Real(20), core.Real(20)}},
		{Operator: "f*", Operands: nil},
		// B* (fill and stroke even-odd)
		{Operator: "re", Operands: []core.Object{core.Real(30), core.Real(0), core.Real(20), core.Real(20)}},
		{Operator: "B*", Operands: nil},
		// b (close, fill and stroke)
		{Operator: "m", Operands: []core.Object{core.Real(60), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(80), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(70), core.Real(20)}},
		{Operator: "b", Operands: nil},
		// b* (close, fill and stroke even-odd)
		{Operator: "m", Operands: []core.Object{core.Real(90), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(110), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(20)}},
		{Operator: "b*", Operands: nil},
		// n (end path)
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(50)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(50)}},
		{Operator: "n", Operands: nil},
	}

	err := ge.Extract(ops)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should have 2 rectangles (f* and B*)
	rects := ge.GetRectangles()
	if len(rects) != 2 {
		t.Errorf("Expected 2 rectangles, got %d", len(rects))
	}

	// Should have 6 lines from the two triangles (3 lines each)
	lines := ge.GetLines()
	if len(lines) != 6 {
		t.Errorf("Expected 6 lines (2 triangles), got %d", len(lines))
	}
}

// Benchmark

func BenchmarkGraphicsExtractor(b *testing.B) {
	ops := []contentstream.Operation{
		{Operator: "m", Operands: []core.Object{core.Real(0), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(0)}},
		{Operator: "l", Operands: []core.Object{core.Real(100), core.Real(100)}},
		{Operator: "l", Operands: []core.Object{core.Real(0), core.Real(100)}},
		{Operator: "h", Operands: nil},
		{Operator: "S", Operands: nil},
		{Operator: "re", Operands: []core.Object{core.Real(10), core.Real(10), core.Real(80), core.Real(80)}},
		{Operator: "f", Operands: nil},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ge := NewGraphicsExtractor()
		ge.Extract(ops)
	}
}
