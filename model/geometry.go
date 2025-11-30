package model

import "math"

// Point represents a 2D point with X and Y coordinates.
type Point struct {
	X, Y float64
}

// Distance returns the Euclidean distance between p and another point.
func (p Point) Distance(other Point) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// BBox represents an axis-aligned bounding box (rectangle).
// In PDF coordinates, Y increases upward, so Y is the bottom edge.
type BBox struct {
	X      float64 // Left
	Y      float64 // Bottom (PDF coordinate system)
	Width  float64
	Height float64
}

// NewBBox creates a bounding box from the given coordinates.
func NewBBox(x, y, width, height float64) BBox {
	return BBox{X: x, Y: y, Width: width, Height: height}
}

// NewBBoxFromPoints creates a bounding box that encloses the two given points.
func NewBBoxFromPoints(p1, p2 Point) BBox {
	x := math.Min(p1.X, p2.X)
	y := math.Min(p1.Y, p2.Y)
	width := math.Abs(p2.X - p1.X)
	height := math.Abs(p2.Y - p1.Y)
	return BBox{X: x, Y: y, Width: width, Height: height}
}

// Left returns the X coordinate of the left edge.
func (b BBox) Left() float64 {
	return b.X
}

// Right returns the X coordinate of the right edge.
func (b BBox) Right() float64 {
	return b.X + b.Width
}

// Bottom returns the Y coordinate of the bottom edge.
func (b BBox) Bottom() float64 {
	return b.Y
}

// Top returns the Y coordinate of the top edge.
func (b BBox) Top() float64 {
	return b.Y + b.Height
}

// Center returns the center point of the bounding box.
func (b BBox) Center() Point {
	return Point{
		X: b.X + b.Width/2,
		Y: b.Y + b.Height/2,
	}
}

// Contains reports whether the point p is inside the bounding box.
func (b BBox) Contains(p Point) bool {
	return p.X >= b.Left() && p.X <= b.Right() &&
		p.Y >= b.Bottom() && p.Y <= b.Top()
}

// Intersects reports whether b and other have any overlapping area.
func (b BBox) Intersects(other BBox) bool {
	return !(b.Right() < other.Left() ||
		b.Left() > other.Right() ||
		b.Top() < other.Bottom() ||
		b.Bottom() > other.Top())
}

// Intersection returns the bounding box of the overlapping region,
// or an empty BBox if the boxes do not intersect.
func (b BBox) Intersection(other BBox) BBox {
	if !b.Intersects(other) {
		return BBox{}
	}

	x := math.Max(b.Left(), other.Left())
	y := math.Max(b.Bottom(), other.Bottom())
	right := math.Min(b.Right(), other.Right())
	top := math.Min(b.Top(), other.Top())

	return BBox{
		X:      x,
		Y:      y,
		Width:  right - x,
		Height: top - y,
	}
}

// Union returns the smallest bounding box that contains both b and other.
func (b BBox) Union(other BBox) BBox {
	x := math.Min(b.Left(), other.Left())
	y := math.Min(b.Bottom(), other.Bottom())
	right := math.Max(b.Right(), other.Right())
	top := math.Max(b.Top(), other.Top())

	return BBox{
		X:      x,
		Y:      y,
		Width:  right - x,
		Height: top - y,
	}
}

// Area returns the area of the bounding box (Width * Height).
func (b BBox) Area() float64 {
	return b.Width * b.Height
}

// Expand returns a new bounding box expanded by margin on all four sides.
func (b BBox) Expand(margin float64) BBox {
	return BBox{
		X:      b.X - margin,
		Y:      b.Y - margin,
		Width:  b.Width + 2*margin,
		Height: b.Height + 2*margin,
	}
}

// OverlapRatio returns the ratio of intersection area to the smaller box's area.
// Returns a value between 0 (no overlap) and 1 (complete overlap).
func (b BBox) OverlapRatio(other BBox) float64 {
	if !b.Intersects(other) {
		return 0
	}

	intersection := b.Intersection(other)
	minArea := math.Min(b.Area(), other.Area())

	if minArea == 0 {
		return 0
	}

	return intersection.Area() / minArea
}

// IsEmpty reports whether the bounding box has zero or negative dimensions.
func (b BBox) IsEmpty() bool {
	return b.Width <= 0 || b.Height <= 0
}

// IsValid reports whether the bounding box has positive width and height.
func (b BBox) IsValid() bool {
	return b.Width > 0 && b.Height > 0
}

// Matrix represents a 2D affine transformation matrix [a, b, c, d, e, f].
// This is stored in row-major order as used by PDF: [a b c d e f].
type Matrix [6]float64

// Identity returns the identity transformation matrix.
func Identity() Matrix {
	return Matrix{1, 0, 0, 1, 0, 0}
}

// Transform applies the affine transformation to a point and returns the result.
func (m Matrix) Transform(p Point) Point {
	return Point{
		X: m[0]*p.X + m[2]*p.Y + m[4],
		Y: m[1]*p.X + m[3]*p.Y + m[5],
	}
}

// Multiply returns the product of m and other (m * other).
func (m Matrix) Multiply(other Matrix) Matrix {
	return Matrix{
		m[0]*other[0] + m[1]*other[2],
		m[0]*other[1] + m[1]*other[3],
		m[2]*other[0] + m[3]*other[2],
		m[2]*other[1] + m[3]*other[3],
		m[4]*other[0] + m[5]*other[2] + other[4],
		m[4]*other[1] + m[5]*other[3] + other[5],
	}
}

// Translate returns a translation matrix that moves by (tx, ty).
func Translate(tx, ty float64) Matrix {
	return Matrix{1, 0, 0, 1, tx, ty}
}

// Scale returns a scaling matrix with scale factors (sx, sy).
func Scale(sx, sy float64) Matrix {
	return Matrix{sx, 0, 0, sy, 0, 0}
}

// Rotate returns a rotation matrix for the given angle in radians.
func Rotate(angle float64) Matrix {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Matrix{cos, sin, -sin, cos, 0, 0}
}

// IsIdentity reports whether m is the identity matrix.
func (m Matrix) IsIdentity() bool {
	return m[0] == 1 && m[1] == 0 && m[2] == 0 && m[3] == 1 && m[4] == 0 && m[5] == 0
}
