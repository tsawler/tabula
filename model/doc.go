// Package model provides the intermediate representation (IR) for extracted
// document content.
//
// This package defines the user-facing data structures that represent the
// semantic structure of documents. All parsing and extraction operations
// ultimately produce these types, making them the primary API for consuming
// extracted content.
//
// # Document Structure
//
// The [Document] type represents a complete document with metadata and pages:
//
//	doc := model.NewDocument()
//	doc.Metadata.Title = "My Document"
//	doc.AddPage(page)
//
// Each [Page] contains dimensions, rotation, and a list of [Element] objects
// representing the page content.
//
// # Elements
//
// All page content implements the [Element] interface. The concrete types are:
//
//   - [Paragraph] - text paragraphs
//   - [Heading] - headings (levels 1-6)
//   - [List] - ordered or unordered lists
//   - [Table] - tables with cells, row/column spans
//   - [Image] - embedded images
//
// # Tables
//
// The [Table] type provides a complete table representation with:
//
//   - Rows and columns of [Cell] values
//   - Row and column spanning
//   - Export methods: ToMarkdown() and ToCSV()
//
// # Geometry
//
// Geometric primitives support position and layout calculations:
//
//   - [BBox] - bounding box with intersection, union, and overlap calculations
//   - [Point] - 2D point with distance calculation
//   - [Matrix] - 2D affine transformation matrix
//
// # Layout Information
//
// When layout analysis is performed, pages contain additional structure:
//
//   - [PageLayout] - column detection, paragraphs, headings, lists
//   - [HeadingInfo], [ParagraphInfo], [ListInfo] - detected elements
//   - Reading order for proper text extraction
package model
