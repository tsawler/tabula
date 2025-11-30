// Package graphicsstate provides PDF graphics state management.
//
// The PDF graphics state controls how content is rendered, including
// transformation matrices, colors, line properties, and text state.
// This package implements the state stack used during content stream
// processing.
//
// # Graphics State
//
// The main type is GraphicsState, which tracks:
//   - CTM (Current Transformation Matrix) for coordinate transformations
//   - Line properties (width, cap, join)
//   - Colors (stroke and fill)
//   - Text state (font, size, spacing, matrices)
//
// Example usage:
//
//	gs := graphicsstate.NewGraphicsState()
//	gs.Save()              // Push state (q operator)
//	gs.Transform(matrix)   // Modify CTM (cm operator)
//	gs.SetFont("F1", 12)   // Set font (Tf operator)
//	gs.Restore()           // Pop state (Q operator)
//
// # Text State
//
// Text rendering uses a separate TextState structure that tracks:
//   - Font name and size (Tf operator)
//   - Character and word spacing (Tc, Tw operators)
//   - Horizontal scaling (Tz operator)
//   - Leading for line spacing (TL operator)
//   - Text and text line matrices (Tm, Td operators)
//
// # Path Operations
//
// The package also includes path construction and painting support
// for extracting line graphics used in table detection:
//   - MoveTo, LineTo, CurveTo for path construction
//   - Rectangle for rect operator
//   - Stroke, Fill for path painting
package graphicsstate
