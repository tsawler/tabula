// Package layout provides document layout analysis for extracting semantic
// structure from PDF pages.
//
// This package analyzes text fragments to detect document structure including
// lines, paragraphs, headings, lists, columns, and reading order.
//
// # Layout Analysis
//
// The [Analyzer] orchestrates all detection components:
//
//	analyzer := layout.NewAnalyzer()
//	result := analyzer.Analyze(fragments, pageWidth, pageHeight)
//
// For faster analysis without heading/list detection:
//
//	result := analyzer.QuickAnalyze(fragments, pageWidth, pageHeight)
//
// # Analysis Results
//
// The [AnalysisResult] contains:
//
//   - Elements - all detected elements in reading order
//   - Columns - column layout information
//   - ReadingOrder - proper reading sequence for multi-column layouts
//   - Headings, Lists, Paragraphs - detected semantic elements
//   - Blocks, Lines - lower-level text structure
//
// # Detectors
//
// The package includes specialized detectors:
//
//   - [LineDetector] - groups fragments into text lines
//   - [ParagraphDetector] - groups lines into paragraphs
//   - [HeadingDetector] - identifies headings by font size and position
//   - [ListDetector] - detects bulleted and numbered lists
//   - [BlockDetector] - detects spatial text blocks
//   - [ColumnDetector] - detects multi-column layouts
//   - [ReadingOrderDetector] - determines proper reading sequence
//   - [HeaderFooterDetector] - identifies repeated headers/footers
//
// # Configuration
//
// Each detector can be configured independently:
//
//	config := layout.DefaultAnalyzerConfig()
//	config.DetectHeadings = true
//	config.DetectLists = true
//	config.HeadingConfig.MinFontSizeRatio = 1.2
//	analyzer := layout.NewAnalyzerWithConfig(config)
//
// # Header/Footer Filtering
//
// For multi-page documents, headers and footers can be detected and filtered:
//
//	result := analyzer.AnalyzeWithHeaderFooterFiltering(pageFragments, pageIndex)
package layout
