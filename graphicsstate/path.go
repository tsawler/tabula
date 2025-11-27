package graphicsstate

import (
	"math"

	"github.com/tsawler/tabula/model"
)

// PathSegmentType defines the type of path segment
type PathSegmentType int

const (
	// PathMoveTo starts a new subpath
	PathMoveTo PathSegmentType = iota
	// PathLineTo draws a line to a point
	PathLineTo
	// PathCurveTo draws a cubic Bézier curve
	PathCurveTo
	// PathClosePath closes the current subpath
	PathClosePath
)

// PathSegment represents a single segment of a path
type PathSegment struct {
	Type PathSegmentType

	// For MoveTo and LineTo: single point
	// For CurveTo: control point 1, control point 2, end point
	Points []model.Point
}

// Path represents a graphics path being constructed
type Path struct {
	// Segments contains all the path segments
	Segments []PathSegment

	// CurrentPoint is the current point in user space
	CurrentPoint model.Point

	// SubpathStart is the start of the current subpath (for closepath)
	SubpathStart model.Point

	// HasCurrentPoint indicates if a current point has been set
	HasCurrentPoint bool
}

// NewPath creates a new empty path
func NewPath() *Path {
	return &Path{
		Segments: make([]PathSegment, 0),
	}
}

// MoveTo starts a new subpath at the specified point (m operator)
func (p *Path) MoveTo(x, y float64) {
	pt := model.Point{X: x, Y: y}
	p.Segments = append(p.Segments, PathSegment{
		Type:   PathMoveTo,
		Points: []model.Point{pt},
	})
	p.CurrentPoint = pt
	p.SubpathStart = pt
	p.HasCurrentPoint = true
}

// LineTo appends a line segment from current point to (x, y) (l operator)
func (p *Path) LineTo(x, y float64) {
	if !p.HasCurrentPoint {
		// Treat as moveto if no current point
		p.MoveTo(x, y)
		return
	}

	pt := model.Point{X: x, Y: y}
	p.Segments = append(p.Segments, PathSegment{
		Type:   PathLineTo,
		Points: []model.Point{pt},
	})
	p.CurrentPoint = pt
}

// CurveTo appends a cubic Bézier curve (c operator)
// Control points (x1, y1) and (x2, y2), end point (x3, y3)
func (p *Path) CurveTo(x1, y1, x2, y2, x3, y3 float64) {
	if !p.HasCurrentPoint {
		p.MoveTo(x1, y1)
	}

	p.Segments = append(p.Segments, PathSegment{
		Type: PathCurveTo,
		Points: []model.Point{
			{X: x1, Y: y1},
			{X: x2, Y: y2},
			{X: x3, Y: y3},
		},
	})
	p.CurrentPoint = model.Point{X: x3, Y: y3}
}

// CurveToV appends a cubic Bézier curve with first control point = current point (v operator)
func (p *Path) CurveToV(x2, y2, x3, y3 float64) {
	if !p.HasCurrentPoint {
		return
	}
	p.CurveTo(p.CurrentPoint.X, p.CurrentPoint.Y, x2, y2, x3, y3)
}

// CurveToY appends a cubic Bézier curve with second control point = end point (y operator)
func (p *Path) CurveToY(x1, y1, x3, y3 float64) {
	if !p.HasCurrentPoint {
		return
	}
	p.CurveTo(x1, y1, x3, y3, x3, y3)
}

// ClosePath closes the current subpath (h operator)
func (p *Path) ClosePath() {
	if !p.HasCurrentPoint {
		return
	}

	p.Segments = append(p.Segments, PathSegment{
		Type: PathClosePath,
	})

	// Move current point back to subpath start
	p.CurrentPoint = p.SubpathStart
}

// Rectangle appends a rectangle as a complete subpath (re operator)
func (p *Path) Rectangle(x, y, width, height float64) {
	p.MoveTo(x, y)
	p.LineTo(x+width, y)
	p.LineTo(x+width, y+height)
	p.LineTo(x, y+height)
	p.ClosePath()
}

// Clear resets the path
func (p *Path) Clear() {
	p.Segments = p.Segments[:0]
	p.HasCurrentPoint = false
}

// IsEmpty returns true if the path has no segments
func (p *Path) IsEmpty() bool {
	return len(p.Segments) == 0
}

// ExtractedLine represents a line extracted from PDF graphics
type ExtractedLine struct {
	// Start and end points in device space
	Start model.Point
	End   model.Point

	// Line attributes
	Width float64
	Color [3]float64

	// Classification
	IsHorizontal bool
	IsVertical   bool

	// Original bounding box
	BBox model.BBox
}

// ExtractedRectangle represents a rectangle extracted from PDF graphics
type ExtractedRectangle struct {
	// Bounding box in device space
	BBox model.BBox

	// Rectangle attributes
	StrokeWidth float64
	StrokeColor [3]float64
	FillColor   [3]float64
	IsFilled    bool
	IsStroked   bool
}

// PathExtractor extracts lines and rectangles from paths
type PathExtractor struct {
	// Collected graphics elements
	Lines      []ExtractedLine
	Rectangles []ExtractedRectangle

	// Current path being constructed
	currentPath *Path

	// Graphics state reference (for CTM, line width, colors)
	gs *GraphicsState

	// Tolerance for horizontal/vertical classification (in points)
	AngleTolerance float64
}

// NewPathExtractor creates a new path extractor
func NewPathExtractor(gs *GraphicsState) *PathExtractor {
	return &PathExtractor{
		Lines:          make([]ExtractedLine, 0),
		Rectangles:     make([]ExtractedRectangle, 0),
		currentPath:    NewPath(),
		gs:             gs,
		AngleTolerance: 0.5, // Allow 0.5 point deviation for horizontal/vertical
	}
}

// MoveTo handles the m operator
func (pe *PathExtractor) MoveTo(x, y float64) {
	pe.currentPath.MoveTo(x, y)
}

// LineTo handles the l operator
func (pe *PathExtractor) LineTo(x, y float64) {
	pe.currentPath.LineTo(x, y)
}

// CurveTo handles the c operator
func (pe *PathExtractor) CurveTo(x1, y1, x2, y2, x3, y3 float64) {
	pe.currentPath.CurveTo(x1, y1, x2, y2, x3, y3)
}

// CurveToV handles the v operator
func (pe *PathExtractor) CurveToV(x2, y2, x3, y3 float64) {
	pe.currentPath.CurveToV(x2, y2, x3, y3)
}

// CurveToY handles the y operator
func (pe *PathExtractor) CurveToY(x1, y1, x3, y3 float64) {
	pe.currentPath.CurveToY(x1, y1, x3, y3)
}

// ClosePath handles the h operator
func (pe *PathExtractor) ClosePath() {
	pe.currentPath.ClosePath()
}

// Rectangle handles the re operator
func (pe *PathExtractor) Rectangle(x, y, width, height float64) {
	pe.currentPath.Rectangle(x, y, width, height)
}

// Stroke handles the S operator (stroke path)
func (pe *PathExtractor) Stroke() {
	pe.extractLinesFromPath(true, false)
	pe.currentPath.Clear()
}

// CloseAndStroke handles the s operator (close path and stroke)
func (pe *PathExtractor) CloseAndStroke() {
	pe.currentPath.ClosePath()
	pe.Stroke()
}

// Fill handles the f/F operator (fill path)
func (pe *PathExtractor) Fill() {
	pe.extractLinesFromPath(false, true)
	pe.currentPath.Clear()
}

// FillEvenOdd handles the f* operator (fill with even-odd rule)
func (pe *PathExtractor) FillEvenOdd() {
	pe.Fill()
}

// FillAndStroke handles the B operator (fill and stroke)
func (pe *PathExtractor) FillAndStroke() {
	pe.extractLinesFromPath(true, true)
	pe.currentPath.Clear()
}

// FillAndStrokeEvenOdd handles the B* operator
func (pe *PathExtractor) FillAndStrokeEvenOdd() {
	pe.FillAndStroke()
}

// CloseFillAndStroke handles the b operator
func (pe *PathExtractor) CloseFillAndStroke() {
	pe.currentPath.ClosePath()
	pe.FillAndStroke()
}

// CloseFillAndStrokeEvenOdd handles the b* operator
func (pe *PathExtractor) CloseFillAndStrokeEvenOdd() {
	pe.currentPath.ClosePath()
	pe.FillAndStrokeEvenOdd()
}

// EndPath handles the n operator (end path without filling or stroking)
func (pe *PathExtractor) EndPath() {
	pe.currentPath.Clear()
}

// extractLinesFromPath extracts lines and rectangles from the current path
func (pe *PathExtractor) extractLinesFromPath(stroked, filled bool) {
	if pe.currentPath.IsEmpty() {
		return
	}

	// Check if this is a rectangle
	if rect := pe.detectRectangle(); rect != nil {
		rect.IsStroked = stroked
		rect.IsFilled = filled
		if stroked {
			rect.StrokeWidth = pe.gs.LineWidth
			rect.StrokeColor = pe.gs.StrokeColor
		}
		if filled {
			rect.FillColor = pe.gs.FillColor
		}
		pe.Rectangles = append(pe.Rectangles, *rect)
		return
	}

	// Extract individual line segments (only if stroked)
	if stroked {
		pe.extractLineSegments()
	}
}

// detectRectangle checks if the current path is a rectangle
func (pe *PathExtractor) detectRectangle() *ExtractedRectangle {
	segments := pe.currentPath.Segments
	if len(segments) < 4 {
		return nil
	}

	// A rectangle should have: moveto, 3 lineto (or 4 lineto with close)
	// Pattern: m, l, l, l, (h or l back to start)
	if segments[0].Type != PathMoveTo {
		return nil
	}

	var corners []model.Point
	corners = append(corners, segments[0].Points[0])

	for i := 1; i < len(segments); i++ {
		seg := segments[i]
		switch seg.Type {
		case PathLineTo:
			corners = append(corners, seg.Points[0])
		case PathClosePath:
			// Close path completes the rectangle
		case PathMoveTo:
			// New subpath - not a simple rectangle
			return nil
		default:
			// Curves make this not a simple rectangle
			return nil
		}
	}

	// Need exactly 4 corners for a rectangle
	if len(corners) < 4 || len(corners) > 5 {
		return nil
	}

	// If 5 corners, the last should be same as first (closed path)
	if len(corners) == 5 {
		if !pointsEqual(corners[0], corners[4], 0.1) {
			return nil
		}
		corners = corners[:4]
	}

	// Check if it forms a rectangle (4 right angles)
	if !isRectangle(corners, pe.AngleTolerance) {
		return nil
	}

	// Transform corners to device space
	transformed := make([]model.Point, 4)
	for i, c := range corners {
		transformed[i] = pe.gs.CTM.Transform(c)
	}

	// Calculate bounding box
	bbox := boundingBoxFromPoints(transformed)

	return &ExtractedRectangle{
		BBox: bbox,
	}
}

// extractLineSegments extracts line segments from the path
func (pe *PathExtractor) extractLineSegments() {
	var currentPoint model.Point
	var subpathStart model.Point

	for _, seg := range pe.currentPath.Segments {
		switch seg.Type {
		case PathMoveTo:
			currentPoint = seg.Points[0]
			subpathStart = currentPoint

		case PathLineTo:
			endPoint := seg.Points[0]
			line := pe.createLine(currentPoint, endPoint)
			pe.Lines = append(pe.Lines, line)
			currentPoint = endPoint

		case PathCurveTo:
			// For curves, we approximate with a line from start to end
			// This is a simplification - for better accuracy, we'd sample the curve
			endPoint := seg.Points[2]
			line := pe.createLine(currentPoint, endPoint)
			pe.Lines = append(pe.Lines, line)
			currentPoint = endPoint

		case PathClosePath:
			if !pointsEqual(currentPoint, subpathStart, 0.1) {
				line := pe.createLine(currentPoint, subpathStart)
				pe.Lines = append(pe.Lines, line)
			}
			currentPoint = subpathStart
		}
	}
}

// createLine creates an ExtractedLine from two points
func (pe *PathExtractor) createLine(start, end model.Point) ExtractedLine {
	// Transform to device space
	startDevice := pe.gs.CTM.Transform(start)
	endDevice := pe.gs.CTM.Transform(end)

	// Calculate line attributes
	dx := endDevice.X - startDevice.X
	dy := endDevice.Y - startDevice.Y

	isHoriz := math.Abs(dy) < pe.AngleTolerance
	isVert := math.Abs(dx) < pe.AngleTolerance

	// Create bounding box
	minX := math.Min(startDevice.X, endDevice.X)
	maxX := math.Max(startDevice.X, endDevice.X)
	minY := math.Min(startDevice.Y, endDevice.Y)
	maxY := math.Max(startDevice.Y, endDevice.Y)

	return ExtractedLine{
		Start:        startDevice,
		End:          endDevice,
		Width:        pe.gs.LineWidth,
		Color:        pe.gs.StrokeColor,
		IsHorizontal: isHoriz,
		IsVertical:   isVert,
		BBox: model.BBox{
			X:      minX,
			Y:      minY,
			Width:  maxX - minX,
			Height: maxY - minY,
		},
	}
}

// Helper functions

// pointsEqual checks if two points are approximately equal
func pointsEqual(a, b model.Point, tolerance float64) bool {
	return math.Abs(a.X-b.X) < tolerance && math.Abs(a.Y-b.Y) < tolerance
}

// isRectangle checks if four points form a rectangle
func isRectangle(corners []model.Point, tolerance float64) bool {
	if len(corners) != 4 {
		return false
	}

	// Check if opposite sides are parallel and equal length
	// Side 0-1 should be parallel to side 3-2
	// Side 1-2 should be parallel to side 0-3

	// Also check for right angles
	for i := 0; i < 4; i++ {
		p0 := corners[i]
		p1 := corners[(i+1)%4]
		p2 := corners[(i+2)%4]

		// Vector from p0 to p1
		v1x := p1.X - p0.X
		v1y := p1.Y - p0.Y

		// Vector from p1 to p2
		v2x := p2.X - p1.X
		v2y := p2.Y - p1.Y

		// Dot product should be ~0 for perpendicular
		dot := v1x*v2x + v1y*v2y
		len1 := math.Sqrt(v1x*v1x + v1y*v1y)
		len2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if len1 < tolerance || len2 < tolerance {
			continue // Degenerate case
		}

		// Normalized dot product (cosine of angle)
		cosAngle := dot / (len1 * len2)

		// Should be close to 0 for 90 degrees
		if math.Abs(cosAngle) > 0.1 { // Allow ~6 degrees deviation
			return false
		}
	}

	return true
}

// boundingBoxFromPoints calculates the bounding box of a set of points
func boundingBoxFromPoints(points []model.Point) model.BBox {
	if len(points) == 0 {
		return model.BBox{}
	}

	minX, maxX := points[0].X, points[0].X
	minY, maxY := points[0].Y, points[0].Y

	for _, p := range points[1:] {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	return model.BBox{
		X:      minX,
		Y:      minY,
		Width:  maxX - minX,
		Height: maxY - minY,
	}
}

// GetLines returns all extracted lines
func (pe *PathExtractor) GetLines() []ExtractedLine {
	return pe.Lines
}

// GetRectangles returns all extracted rectangles
func (pe *PathExtractor) GetRectangles() []ExtractedRectangle {
	return pe.Rectangles
}

// GetHorizontalLines returns only horizontal lines
func (pe *PathExtractor) GetHorizontalLines() []ExtractedLine {
	var result []ExtractedLine
	for _, line := range pe.Lines {
		if line.IsHorizontal {
			result = append(result, line)
		}
	}
	return result
}

// GetVerticalLines returns only vertical lines
func (pe *PathExtractor) GetVerticalLines() []ExtractedLine {
	var result []ExtractedLine
	for _, line := range pe.Lines {
		if line.IsVertical {
			result = append(result, line)
		}
	}
	return result
}

// Clear clears all extracted elements and the current path
func (pe *PathExtractor) Clear() {
	pe.Lines = pe.Lines[:0]
	pe.Rectangles = pe.Rectangles[:0]
	pe.currentPath.Clear()
}

// FilterLinesByLength filters lines by minimum length
func (pe *PathExtractor) FilterLinesByLength(minLength float64) []ExtractedLine {
	var result []ExtractedLine
	for _, line := range pe.Lines {
		dx := line.End.X - line.Start.X
		dy := line.End.Y - line.Start.Y
		length := math.Sqrt(dx*dx + dy*dy)
		if length >= minLength {
			result = append(result, line)
		}
	}
	return result
}

// FilterRectanglesBySize filters rectangles by minimum dimensions
func (pe *PathExtractor) FilterRectanglesBySize(minWidth, minHeight float64) []ExtractedRectangle {
	var result []ExtractedRectangle
	for _, rect := range pe.Rectangles {
		if rect.BBox.Width >= minWidth && rect.BBox.Height >= minHeight {
			result = append(result, rect)
		}
	}
	return result
}
