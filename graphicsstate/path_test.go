package graphicsstate

import (
	"math"
	"testing"

	"github.com/tsawler/tabula/model"
)

// Path tests

func TestNewPath(t *testing.T) {
	p := NewPath()
	if p == nil {
		t.Fatal("NewPath returned nil")
	}
	if len(p.Segments) != 0 {
		t.Errorf("Expected empty segments, got %d", len(p.Segments))
	}
	if p.HasCurrentPoint {
		t.Error("Expected HasCurrentPoint to be false")
	}
}

func TestPath_MoveTo(t *testing.T) {
	p := NewPath()
	p.MoveTo(100, 200)

	if len(p.Segments) != 1 {
		t.Fatalf("Expected 1 segment, got %d", len(p.Segments))
	}
	if p.Segments[0].Type != PathMoveTo {
		t.Error("Expected PathMoveTo type")
	}
	if !p.HasCurrentPoint {
		t.Error("Expected HasCurrentPoint to be true")
	}
	if p.CurrentPoint.X != 100 || p.CurrentPoint.Y != 200 {
		t.Errorf("Expected current point (100, 200), got (%f, %f)", p.CurrentPoint.X, p.CurrentPoint.Y)
	}
	if p.SubpathStart.X != 100 || p.SubpathStart.Y != 200 {
		t.Errorf("Expected subpath start (100, 200), got (%f, %f)", p.SubpathStart.X, p.SubpathStart.Y)
	}
}

func TestPath_LineTo(t *testing.T) {
	t.Run("with current point", func(t *testing.T) {
		p := NewPath()
		p.MoveTo(0, 0)
		p.LineTo(100, 0)

		if len(p.Segments) != 2 {
			t.Fatalf("Expected 2 segments, got %d", len(p.Segments))
		}
		if p.Segments[1].Type != PathLineTo {
			t.Error("Expected PathLineTo type")
		}
		if p.CurrentPoint.X != 100 || p.CurrentPoint.Y != 0 {
			t.Errorf("Expected current point (100, 0), got (%f, %f)", p.CurrentPoint.X, p.CurrentPoint.Y)
		}
	})

	t.Run("without current point becomes moveto", func(t *testing.T) {
		p := NewPath()
		p.LineTo(100, 200)

		if len(p.Segments) != 1 {
			t.Fatalf("Expected 1 segment, got %d", len(p.Segments))
		}
		if p.Segments[0].Type != PathMoveTo {
			t.Error("Expected PathMoveTo type (lineto should become moveto)")
		}
		if !p.HasCurrentPoint {
			t.Error("Expected HasCurrentPoint to be true")
		}
	})
}

func TestPath_CurveTo(t *testing.T) {
	p := NewPath()
	p.MoveTo(0, 0)
	p.CurveTo(10, 20, 30, 40, 50, 60)

	if len(p.Segments) != 2 {
		t.Fatalf("Expected 2 segments, got %d", len(p.Segments))
	}
	if p.Segments[1].Type != PathCurveTo {
		t.Error("Expected PathCurveTo type")
	}
	if len(p.Segments[1].Points) != 3 {
		t.Errorf("Expected 3 control points, got %d", len(p.Segments[1].Points))
	}
	if p.CurrentPoint.X != 50 || p.CurrentPoint.Y != 60 {
		t.Errorf("Expected current point (50, 60), got (%f, %f)", p.CurrentPoint.X, p.CurrentPoint.Y)
	}
}

func TestPath_CurveToV(t *testing.T) {
	p := NewPath()
	p.MoveTo(0, 0)
	p.CurveToV(20, 30, 40, 50)

	if len(p.Segments) != 2 {
		t.Fatalf("Expected 2 segments, got %d", len(p.Segments))
	}
	// CurveToV uses current point as first control point
	if p.Segments[1].Points[0].X != 0 || p.Segments[1].Points[0].Y != 0 {
		t.Error("First control point should be current point")
	}
	if p.CurrentPoint.X != 40 || p.CurrentPoint.Y != 50 {
		t.Errorf("Expected current point (40, 50), got (%f, %f)", p.CurrentPoint.X, p.CurrentPoint.Y)
	}
}

func TestPath_CurveToY(t *testing.T) {
	p := NewPath()
	p.MoveTo(0, 0)
	p.CurveToY(10, 20, 40, 50)

	if len(p.Segments) != 2 {
		t.Fatalf("Expected 2 segments, got %d", len(p.Segments))
	}
	// CurveToY uses end point as second control point
	if p.Segments[1].Points[1].X != 40 || p.Segments[1].Points[1].Y != 50 {
		t.Error("Second control point should be end point")
	}
	if p.CurrentPoint.X != 40 || p.CurrentPoint.Y != 50 {
		t.Errorf("Expected current point (40, 50), got (%f, %f)", p.CurrentPoint.X, p.CurrentPoint.Y)
	}
}

func TestPath_ClosePath(t *testing.T) {
	p := NewPath()
	p.MoveTo(0, 0)
	p.LineTo(100, 0)
	p.LineTo(100, 100)
	p.ClosePath()

	if len(p.Segments) != 4 {
		t.Fatalf("Expected 4 segments, got %d", len(p.Segments))
	}
	if p.Segments[3].Type != PathClosePath {
		t.Error("Expected PathClosePath type")
	}
	// Current point should return to subpath start
	if p.CurrentPoint.X != 0 || p.CurrentPoint.Y != 0 {
		t.Errorf("Expected current point (0, 0), got (%f, %f)", p.CurrentPoint.X, p.CurrentPoint.Y)
	}
}

func TestPath_Rectangle(t *testing.T) {
	p := NewPath()
	p.Rectangle(10, 20, 100, 50)

	// Rectangle creates: moveto + 3 lineto + closepath = 5 segments
	if len(p.Segments) != 5 {
		t.Fatalf("Expected 5 segments, got %d", len(p.Segments))
	}

	// Check the sequence
	if p.Segments[0].Type != PathMoveTo {
		t.Error("Expected PathMoveTo first")
	}
	for i := 1; i <= 3; i++ {
		if p.Segments[i].Type != PathLineTo {
			t.Errorf("Expected PathLineTo at index %d", i)
		}
	}
	if p.Segments[4].Type != PathClosePath {
		t.Error("Expected PathClosePath last")
	}
}

func TestPath_Clear(t *testing.T) {
	p := NewPath()
	p.MoveTo(0, 0)
	p.LineTo(100, 100)
	p.Clear()

	if len(p.Segments) != 0 {
		t.Errorf("Expected 0 segments after clear, got %d", len(p.Segments))
	}
	if p.HasCurrentPoint {
		t.Error("Expected HasCurrentPoint to be false after clear")
	}
}

func TestPath_IsEmpty(t *testing.T) {
	p := NewPath()
	if !p.IsEmpty() {
		t.Error("Expected new path to be empty")
	}

	p.MoveTo(0, 0)
	if p.IsEmpty() {
		t.Error("Expected path with segments to not be empty")
	}
}

// PathExtractor tests

func TestNewPathExtractor(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	if pe == nil {
		t.Fatal("NewPathExtractor returned nil")
	}
	if len(pe.Lines) != 0 {
		t.Error("Expected empty Lines")
	}
	if len(pe.Rectangles) != 0 {
		t.Error("Expected empty Rectangles")
	}
	if pe.AngleTolerance != 0.5 {
		t.Errorf("Expected AngleTolerance 0.5, got %f", pe.AngleTolerance)
	}
}

func TestPathExtractor_SimpleHorizontalLine(t *testing.T) {
	gs := NewGraphicsState()
	gs.SetLineWidth(1.0)
	pe := NewPathExtractor(gs)

	pe.MoveTo(0, 100)
	pe.LineTo(200, 100)
	pe.Stroke()

	lines := pe.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}

	line := lines[0]
	if !line.IsHorizontal {
		t.Error("Expected line to be horizontal")
	}
	if line.IsVertical {
		t.Error("Expected line to not be vertical")
	}
	if line.Width != 1.0 {
		t.Errorf("Expected line width 1.0, got %f", line.Width)
	}
}

func TestPathExtractor_SimpleVerticalLine(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	pe.MoveTo(100, 0)
	pe.LineTo(100, 200)
	pe.Stroke()

	lines := pe.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}

	line := lines[0]
	if line.IsHorizontal {
		t.Error("Expected line to not be horizontal")
	}
	if !line.IsVertical {
		t.Error("Expected line to be vertical")
	}
}

func TestPathExtractor_DiagonalLine(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	pe.MoveTo(0, 0)
	pe.LineTo(100, 100)
	pe.Stroke()

	lines := pe.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}

	line := lines[0]
	if line.IsHorizontal {
		t.Error("Expected diagonal line to not be horizontal")
	}
	if line.IsVertical {
		t.Error("Expected diagonal line to not be vertical")
	}
}

func TestPathExtractor_Rectangle(t *testing.T) {
	gs := NewGraphicsState()
	gs.SetLineWidth(2.0)
	gs.SetStrokeColorRGB(1, 0, 0)
	pe := NewPathExtractor(gs)

	pe.Rectangle(100, 100, 200, 150)
	pe.Stroke()

	rects := pe.GetRectangles()
	if len(rects) != 1 {
		t.Fatalf("Expected 1 rectangle, got %d", len(rects))
	}

	rect := rects[0]
	if !rect.IsStroked {
		t.Error("Expected rectangle to be stroked")
	}
	if rect.IsFilled {
		t.Error("Expected rectangle to not be filled")
	}
	if rect.StrokeWidth != 2.0 {
		t.Errorf("Expected stroke width 2.0, got %f", rect.StrokeWidth)
	}
	if rect.StrokeColor[0] != 1 {
		t.Errorf("Expected red stroke color, got %v", rect.StrokeColor)
	}

	// Check bbox (rectangle is at 100, 100 with size 200x150)
	if rect.BBox.X != 100 || rect.BBox.Y != 100 {
		t.Errorf("Expected BBox at (100, 100), got (%f, %f)", rect.BBox.X, rect.BBox.Y)
	}
	if rect.BBox.Width != 200 || rect.BBox.Height != 150 {
		t.Errorf("Expected BBox size (200, 150), got (%f, %f)", rect.BBox.Width, rect.BBox.Height)
	}
}

func TestPathExtractor_FilledRectangle(t *testing.T) {
	gs := NewGraphicsState()
	gs.SetFillColorRGB(0, 1, 0)
	pe := NewPathExtractor(gs)

	pe.Rectangle(0, 0, 100, 100)
	pe.Fill()

	rects := pe.GetRectangles()
	if len(rects) != 1 {
		t.Fatalf("Expected 1 rectangle, got %d", len(rects))
	}

	rect := rects[0]
	if rect.IsStroked {
		t.Error("Expected rectangle to not be stroked")
	}
	if !rect.IsFilled {
		t.Error("Expected rectangle to be filled")
	}
	if rect.FillColor[1] != 1 {
		t.Errorf("Expected green fill color, got %v", rect.FillColor)
	}
}

func TestPathExtractor_FillAndStroke(t *testing.T) {
	gs := NewGraphicsState()
	gs.SetLineWidth(1.5)
	gs.SetStrokeColorRGB(1, 0, 0)
	gs.SetFillColorRGB(0, 0, 1)
	pe := NewPathExtractor(gs)

	pe.Rectangle(50, 50, 100, 100)
	pe.FillAndStroke()

	rects := pe.GetRectangles()
	if len(rects) != 1 {
		t.Fatalf("Expected 1 rectangle, got %d", len(rects))
	}

	rect := rects[0]
	if !rect.IsStroked {
		t.Error("Expected rectangle to be stroked")
	}
	if !rect.IsFilled {
		t.Error("Expected rectangle to be filled")
	}
	if rect.StrokeColor[0] != 1 {
		t.Error("Expected red stroke")
	}
	if rect.FillColor[2] != 1 {
		t.Error("Expected blue fill")
	}
}

func TestPathExtractor_CloseAndStroke(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	pe.MoveTo(0, 0)
	pe.LineTo(100, 0)
	pe.LineTo(100, 100)
	pe.CloseAndStroke()

	lines := pe.GetLines()
	// Triangle: 3 lines (including the close path line back to start)
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines for closed triangle, got %d", len(lines))
	}
}

func TestPathExtractor_EndPath(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	pe.MoveTo(0, 0)
	pe.LineTo(100, 0)
	pe.EndPath()

	// EndPath clears without stroking/filling
	if len(pe.GetLines()) != 0 {
		t.Error("Expected no lines after EndPath")
	}
	if len(pe.GetRectangles()) != 0 {
		t.Error("Expected no rectangles after EndPath")
	}
}

func TestPathExtractor_GetHorizontalLines(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	// Draw horizontal line
	pe.MoveTo(0, 100)
	pe.LineTo(200, 100)
	pe.Stroke()

	// Draw vertical line
	pe.MoveTo(100, 0)
	pe.LineTo(100, 200)
	pe.Stroke()

	// Draw diagonal line
	pe.MoveTo(0, 0)
	pe.LineTo(100, 100)
	pe.Stroke()

	horiz := pe.GetHorizontalLines()
	if len(horiz) != 1 {
		t.Errorf("Expected 1 horizontal line, got %d", len(horiz))
	}
}

func TestPathExtractor_GetVerticalLines(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	// Draw horizontal line
	pe.MoveTo(0, 100)
	pe.LineTo(200, 100)
	pe.Stroke()

	// Draw vertical line
	pe.MoveTo(100, 0)
	pe.LineTo(100, 200)
	pe.Stroke()

	vert := pe.GetVerticalLines()
	if len(vert) != 1 {
		t.Errorf("Expected 1 vertical line, got %d", len(vert))
	}
}

func TestPathExtractor_FilterLinesByLength(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	// Short line (length 50)
	pe.MoveTo(0, 0)
	pe.LineTo(50, 0)
	pe.Stroke()

	// Long line (length 200)
	pe.MoveTo(0, 100)
	pe.LineTo(200, 100)
	pe.Stroke()

	filtered := pe.FilterLinesByLength(100)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 line >= 100, got %d", len(filtered))
	}
}

func TestPathExtractor_FilterRectanglesBySize(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	// Small rectangle
	pe.Rectangle(0, 0, 10, 10)
	pe.Stroke()

	// Large rectangle
	pe.Rectangle(100, 100, 200, 150)
	pe.Stroke()

	filtered := pe.FilterRectanglesBySize(50, 50)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 rectangle >= 50x50, got %d", len(filtered))
	}
}

func TestPathExtractor_Clear(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	pe.MoveTo(0, 0)
	pe.LineTo(100, 0)
	pe.Stroke()

	pe.Rectangle(0, 0, 100, 100)
	pe.Stroke()

	pe.Clear()

	if len(pe.Lines) != 0 {
		t.Errorf("Expected 0 lines after clear, got %d", len(pe.Lines))
	}
	if len(pe.Rectangles) != 0 {
		t.Errorf("Expected 0 rectangles after clear, got %d", len(pe.Rectangles))
	}
}

func TestPathExtractor_WithTransform(t *testing.T) {
	gs := NewGraphicsState()
	// Apply a scale transform
	gs.Transform(model.Scale(2, 2))
	pe := NewPathExtractor(gs)

	// Draw a line from (0,0) to (100,0) in user space
	pe.MoveTo(0, 0)
	pe.LineTo(100, 0)
	pe.Stroke()

	lines := pe.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}

	// In device space, it should be from (0,0) to (200,0) due to 2x scale
	line := lines[0]
	if line.End.X != 200 {
		t.Errorf("Expected end.X = 200 (scaled), got %f", line.End.X)
	}
}

func TestPathExtractor_CurveApproximation(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	// Draw a curve
	pe.MoveTo(0, 0)
	pe.CurveTo(50, 100, 100, 100, 150, 0)
	pe.Stroke()

	lines := pe.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 approximated line from curve, got %d", len(lines))
	}

	// The approximated line should go from start to end of curve
	line := lines[0]
	if line.Start.X != 0 || line.Start.Y != 0 {
		t.Errorf("Expected start (0,0), got (%f, %f)", line.Start.X, line.Start.Y)
	}
	if line.End.X != 150 || line.End.Y != 0 {
		t.Errorf("Expected end (150, 0), got (%f, %f)", line.End.X, line.End.Y)
	}
}

func TestPathExtractor_ComplexPath(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	// A closed square is detected as a rectangle, not individual lines
	// So let's draw a non-rectangular closed path (pentagon)
	pe.MoveTo(50, 0)
	pe.LineTo(100, 40)
	pe.LineTo(80, 100)
	pe.LineTo(20, 100)
	pe.LineTo(0, 40)
	pe.ClosePath()
	pe.Stroke()

	lines := pe.GetLines()
	// 5 lines for a closed pentagon
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines for pentagon, got %d", len(lines))
	}
}

func TestPathExtractor_ClosedSquareAsRectangle(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	// Draw a closed square using moveto + lineto + closepath
	pe.MoveTo(0, 0)
	pe.LineTo(100, 0)
	pe.LineTo(100, 100)
	pe.LineTo(0, 100)
	pe.ClosePath()
	pe.Stroke()

	// Should be detected as rectangle, not individual lines
	rects := pe.GetRectangles()
	if len(rects) != 1 {
		t.Errorf("Expected closed square to be detected as 1 rectangle, got %d", len(rects))
	}
	if len(pe.GetLines()) != 0 {
		t.Errorf("Expected no individual lines (rectangle detected), got %d", len(pe.Lines))
	}
}

// Helper function tests

func TestPointsEqual(t *testing.T) {
	tests := []struct {
		name      string
		a, b      model.Point
		tolerance float64
		want      bool
	}{
		{
			name:      "equal points",
			a:         model.Point{X: 100, Y: 200},
			b:         model.Point{X: 100, Y: 200},
			tolerance: 0.1,
			want:      true,
		},
		{
			name:      "within tolerance",
			a:         model.Point{X: 100, Y: 200},
			b:         model.Point{X: 100.05, Y: 200.05},
			tolerance: 0.1,
			want:      true,
		},
		{
			name:      "outside tolerance",
			a:         model.Point{X: 100, Y: 200},
			b:         model.Point{X: 101, Y: 200},
			tolerance: 0.1,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pointsEqual(tt.a, tt.b, tt.tolerance)
			if got != tt.want {
				t.Errorf("pointsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRectangle(t *testing.T) {
	t.Run("valid rectangle", func(t *testing.T) {
		corners := []model.Point{
			{X: 0, Y: 0},
			{X: 100, Y: 0},
			{X: 100, Y: 50},
			{X: 0, Y: 50},
		}
		if !isRectangle(corners, 0.5) {
			t.Error("Expected valid rectangle")
		}
	})

	t.Run("rotated rectangle", func(t *testing.T) {
		// 45-degree rotated square
		s := math.Sqrt(2) / 2 * 100
		corners := []model.Point{
			{X: 0, Y: s},
			{X: s, Y: 0},
			{X: 2 * s, Y: s},
			{X: s, Y: 2 * s},
		}
		if !isRectangle(corners, 0.5) {
			t.Error("Expected rotated rectangle to be valid")
		}
	})

	t.Run("not a rectangle - parallelogram", func(t *testing.T) {
		corners := []model.Point{
			{X: 0, Y: 0},
			{X: 100, Y: 0},
			{X: 150, Y: 50},
			{X: 50, Y: 50},
		}
		if isRectangle(corners, 0.5) {
			t.Error("Parallelogram should not be detected as rectangle")
		}
	})

	t.Run("wrong number of corners", func(t *testing.T) {
		corners := []model.Point{
			{X: 0, Y: 0},
			{X: 100, Y: 0},
			{X: 50, Y: 50},
		}
		if isRectangle(corners, 0.5) {
			t.Error("Triangle should not be detected as rectangle")
		}
	})
}

func TestBoundingBoxFromPoints(t *testing.T) {
	t.Run("single point", func(t *testing.T) {
		points := []model.Point{{X: 100, Y: 200}}
		bbox := boundingBoxFromPoints(points)
		if bbox.X != 100 || bbox.Y != 200 {
			t.Errorf("Expected (100, 200), got (%f, %f)", bbox.X, bbox.Y)
		}
		if bbox.Width != 0 || bbox.Height != 0 {
			t.Errorf("Expected 0x0 size, got %fx%f", bbox.Width, bbox.Height)
		}
	})

	t.Run("multiple points", func(t *testing.T) {
		points := []model.Point{
			{X: 10, Y: 20},
			{X: 100, Y: 50},
			{X: 50, Y: 200},
		}
		bbox := boundingBoxFromPoints(points)
		if bbox.X != 10 {
			t.Errorf("Expected X=10, got %f", bbox.X)
		}
		if bbox.Y != 20 {
			t.Errorf("Expected Y=20, got %f", bbox.Y)
		}
		if bbox.Width != 90 {
			t.Errorf("Expected Width=90, got %f", bbox.Width)
		}
		if bbox.Height != 180 {
			t.Errorf("Expected Height=180, got %f", bbox.Height)
		}
	})

	t.Run("empty points", func(t *testing.T) {
		bbox := boundingBoxFromPoints([]model.Point{})
		if bbox.X != 0 || bbox.Y != 0 {
			t.Error("Expected empty bbox for empty points")
		}
	})
}

// Edge cases

func TestPathExtractor_EmptyPath(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	// Stroke without any path operations
	pe.Stroke()

	if len(pe.Lines) != 0 {
		t.Error("Expected no lines for empty path")
	}
	if len(pe.Rectangles) != 0 {
		t.Error("Expected no rectangles for empty path")
	}
}

func TestPathExtractor_NonRectanglePath(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	// Draw a triangle (not a rectangle)
	pe.MoveTo(0, 0)
	pe.LineTo(100, 0)
	pe.LineTo(50, 100)
	pe.ClosePath()
	pe.Stroke()

	// Should extract lines, not rectangle
	if len(pe.Rectangles) != 0 {
		t.Error("Triangle should not be detected as rectangle")
	}
	if len(pe.Lines) != 3 {
		t.Errorf("Expected 3 lines for triangle, got %d", len(pe.Lines))
	}
}

func TestPathExtractor_MultipleStrokes(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	// First stroke
	pe.MoveTo(0, 0)
	pe.LineTo(100, 0)
	pe.Stroke()

	// Second stroke
	pe.MoveTo(0, 50)
	pe.LineTo(100, 50)
	pe.Stroke()

	lines := pe.GetLines()
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines from multiple strokes, got %d", len(lines))
	}
}

func TestPath_CurveOperatorsWithoutCurrentPoint(t *testing.T) {
	p := NewPath()

	// CurveToV without current point should be no-op
	p.CurveToV(10, 20, 30, 40)
	if len(p.Segments) != 0 {
		t.Error("CurveToV without current point should be no-op")
	}

	// CurveToY without current point should be no-op
	p.CurveToY(10, 20, 30, 40)
	if len(p.Segments) != 0 {
		t.Error("CurveToY without current point should be no-op")
	}

	// ClosePath without current point should be no-op
	p.ClosePath()
	if len(p.Segments) != 0 {
		t.Error("ClosePath without current point should be no-op")
	}
}

func TestPathExtractor_FillEvenOdd(t *testing.T) {
	gs := NewGraphicsState()
	gs.SetFillColorRGB(1, 1, 0)
	pe := NewPathExtractor(gs)

	pe.Rectangle(0, 0, 100, 100)
	pe.FillEvenOdd()

	rects := pe.GetRectangles()
	if len(rects) != 1 {
		t.Fatalf("Expected 1 rectangle, got %d", len(rects))
	}
	if !rects[0].IsFilled {
		t.Error("Expected rectangle to be filled")
	}
}

func TestPathExtractor_CloseFillAndStroke(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	// Open triangle path
	pe.MoveTo(0, 0)
	pe.LineTo(100, 0)
	pe.LineTo(50, 100)
	pe.CloseFillAndStroke()

	// Should have 3 lines (triangle closed)
	if len(pe.Lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(pe.Lines))
	}
}

func TestPathExtractor_LineBoundingBox(t *testing.T) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	pe.MoveTo(50, 100)
	pe.LineTo(150, 200)
	pe.Stroke()

	lines := pe.GetLines()
	if len(lines) != 1 {
		t.Fatal("Expected 1 line")
	}

	bbox := lines[0].BBox
	if bbox.X != 50 {
		t.Errorf("Expected bbox.X = 50, got %f", bbox.X)
	}
	if bbox.Y != 100 {
		t.Errorf("Expected bbox.Y = 100, got %f", bbox.Y)
	}
	if bbox.Width != 100 {
		t.Errorf("Expected bbox.Width = 100, got %f", bbox.Width)
	}
	if bbox.Height != 100 {
		t.Errorf("Expected bbox.Height = 100, got %f", bbox.Height)
	}
}

// Benchmarks

func BenchmarkPathExtractor_Lines(b *testing.B) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pe.MoveTo(0, 0)
		pe.LineTo(100, 0)
		pe.LineTo(100, 100)
		pe.LineTo(0, 100)
		pe.ClosePath()
		pe.Stroke()
		pe.Clear()
	}
}

func BenchmarkPathExtractor_Rectangles(b *testing.B) {
	gs := NewGraphicsState()
	pe := NewPathExtractor(gs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pe.Rectangle(float64(i), float64(i), 100, 50)
		pe.Stroke()
	}
}
