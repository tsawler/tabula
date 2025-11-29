// Package text provides text extraction from PDF content streams.
//
// This package handles the extraction of text fragments from PDF pages,
// including position calculation, font handling, and text direction detection.
//
// # Text Extraction
//
// The [Extractor] type processes PDF content stream operations to extract
// positioned text:
//
//	extractor := text.NewExtractor()
//	extractor.RegisterFontsFromPage(page, resolver)
//	fragments, err := extractor.ExtractFromBytes(contentData)
//
// Each [TextFragment] contains the text along with position (X, Y), dimensions
// (Width, Height), font information, and text direction.
//
// # Font Registration
//
// For accurate text extraction, fonts should be registered before extraction:
//
//   - RegisterFont - register a font by name
//   - RegisterParsedFont - register a pre-parsed font with ToUnicode CMap
//   - RegisterFontsFromPage - automatically register all fonts from a page
//
// # Text Direction
//
// The package supports bidirectional text with the [Direction] type:
//
//   - LTR - left-to-right (Latin, CJK, etc.)
//   - RTL - right-to-left (Arabic, Hebrew, etc.)
//   - Neutral - direction-neutral characters (numbers, punctuation)
//
// The [DetectDirection] function analyzes text to determine its direction.
//
// # Smart Spacing
//
// The extractor intelligently handles spacing between fragments:
//
//   - Word-level PDFs: Uses font space width metrics
//   - Character-level PDFs: Uses adaptive gap detection
//   - Explicit spaces: Respects space characters in the stream
package text
