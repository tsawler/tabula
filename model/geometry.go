package model

import "math"

// Point represents a 2D point
type Point struct {
	X, Y float64
}

// Distance calculates the Euclidean distance to another point
func (p Point) Distance(other Point) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// BBox represents a bounding box (rectangle)
type BBox struct {
	X      float64 // Left
	Y      float64 // Bottom (PDF coordinate system)
	Width  float64
	Height float64
}

// NewBBox creates a bounding box from coordinates
func NewBBox(x, y, width, height float64) BBox {
	return BBox{X: x, Y: y, Width: width, Height: height}
}

// NewBBoxFromPoints creates a bounding box from two points
func NewBBoxFromPoints(p1, p2 Point) BBox {
	x := math.Min(p1.X, p2.X)
	y := math.Min(p1.Y, p2.Y)
	width := math.Abs(p2.X - p1.X)
	height := math.Abs(p2.Y - p1.Y)
	return BBox{X: x, Y: y, Width: width, Height: height}
}

// Left returns the left edge X coordinate
func (b BBox) Left() float64 {
	return b.X
}

// Right returns the right edge X coordinate
func (b BBox) Right() float64 {
	return b.X + b.Width
}

// Bottom returns the bottom edge Y coordinate
func (b BBox) Bottom() float64 {
	return b.Y
}

// Top returns the top edge Y coordinate
func (b BBox) Top() float64 {
	return b.Y + b.Height
}

// Center returns the center point
func (b BBox) Center() Point {
	return Point{
		X: b.X + b.Width/2,
		Y: b.Y + b.Height/2,
	}
}

// Contains checks if a point is inside the bounding box
func (b BBox) Contains(p Point) bool {
	return p.X >= b.Left() && p.X <= b.Right() &&
		p.Y >= b.Bottom() && p.Y <= b.Top()
}

// Intersects checks if two bounding boxes intersect
func (b BBox) Intersects(other BBox) bool {
	return !(b.Right() < other.Left() ||
		b.Left() > other.Right() ||
		b.Top() < other.Bottom() ||
		b.Bottom() > other.Top())
}

// Intersection returns the intersection of two bounding boxes
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

// Union returns the union of two bounding boxes
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

// Area returns the area of the bounding box
func (b BBox) Area() float64 {
	return b.Width * b.Height
}

// Expand expands the bounding box by a margin on all sides
func (b BBox) Expand(margin float64) BBox {
	return BBox{
		X:      b.X - margin,
		Y:      b.Y - margin,
		Width:  b.Width + 2*margin,
		Height: b.Height + 2*margin,
	}
}

// OverlapRatio calculates the overlap ratio with another box
// Returns value between 0 and 1
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

// IsEmpty returns true if the bounding box has zero area
func (b BBox) IsEmpty() bool {
	return b.Width <= 0 || b.Height <= 0
}

// IsValid returns true if the bounding box has positive dimensions
func (b BBox) IsValid() bool {
	return b.Width > 0 && b.Height > 0
}

// Matrix represents a 2D affine transformation matrix
type Matrix [6]float64

// Identity returns an identity matrix
func Identity() Matrix {
	return Matrix{1, 0, 0, 1, 0, 0}
}

// Transform applies the matrix transformation to a point
func (m Matrix) Transform(p Point) Point {
	return Point{
		X: m[0]*p.X + m[2]*p.Y + m[4],
		Y: m[1]*p.X + m[3]*p.Y + m[5],
	}
}

// Multiply multiplies two matrices
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

// Translate creates a translation matrix
func Translate(tx, ty float64) Matrix {
	return Matrix{1, 0, 0, 1, tx, ty}
}

// Scale creates a scaling matrix
func Scale(sx, sy float64) Matrix {
	return Matrix{sx, 0, 0, sy, 0, 0}
}

// Rotate creates a rotation matrix (angle in radians)
func Rotate(angle float64) Matrix {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Matrix{cos, sin, -sin, cos, 0, 0}
}

// IsIdentity returns true if the matrix is an identity matrix
func (m Matrix) IsIdentity() bool {
	return m[0] == 1 && m[1] == 0 && m[2] == 0 && m[3] == 1 && m[4] == 0 && m[5] == 0
}
