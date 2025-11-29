// Package tables provides table detection and extraction from PDF pages.
//
// This package implements algorithms for detecting tabular data in PDFs,
// even when tables lack explicit gridlines.
//
// # Detectors
//
// Table detection is performed by types implementing the [Detector] interface.
// The package provides:
//
//   - [GeometricDetector] - uses spatial analysis of text positions
//
// Detectors are registered globally and can be retrieved by name:
//
//	detector := tables.GetDetector("geometric")
//	tables, err := detector.Detect(page)
//
// # Geometric Detection
//
// The [GeometricDetector] uses a multi-step algorithm:
//
//  1. Spatial clustering of text fragments
//  2. Alignment analysis (row/column detection)
//  3. Grid construction from text positions and drawn lines
//  4. Cell assignment based on grid positions
//  5. Confidence scoring
//
// # Configuration
//
// Detector behavior is controlled by [Config]:
//
//	config := tables.DefaultConfig()
//	config.MinRows = 3
//	config.MinConfidence = 0.7
//	detector.Configure(config)
//
// Configuration options include:
//
//   - MinRows, MinCols - minimum table dimensions
//   - MinConfidence - confidence threshold (0-1)
//   - UseLines - whether to use drawn lines for detection
//   - UseWhitespace - whether to use whitespace patterns
//   - AlignmentTolerance - tolerance for row/column alignment
//
// # Confidence Scoring
//
// Detection confidence (0-1) is based on:
//
//   - Grid regularity (30%)
//   - Alignment quality (30%)
//   - Line presence (20%)
//   - Cell occupancy (20%)
package tables
