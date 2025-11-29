// Package font provides PDF font handling including Type1, TrueType, and CID fonts.
//
// This package handles font parsing, character encoding, and text width calculation
// for accurate text extraction from PDFs.
//
// # Font Types
//
// The package supports multiple PDF font types:
//
//   - [Type1Font] - PostScript Type 1 fonts (including Standard 14)
//   - [TrueTypeFont] - TrueType outline fonts
//   - [Type0Font] - Composite fonts for CJK text
//
// # Font Creation
//
// Fonts are created from PDF font dictionaries:
//
//	font, err := font.NewType1Font(fontDict, resolver)
//	font, err := font.NewTrueTypeFont(fontDict, resolver)
//	font, err := font.NewType0Font(fontDict, resolver)
//
// # Text Decoding
//
// The [Font] type provides text decoding using ToUnicode CMaps:
//
//	text := font.DecodeString(rawBytes)
//
// # Character Widths
//
// Width information is used for text positioning:
//
//	width := font.GetWidth(charCode)         // Single character
//	width := font.GetStringWidth(text)       // String width in font units
//
// # Encodings
//
// Character encodings map character codes to glyph names:
//
//   - Standard PDF encodings (WinAnsiEncoding, MacRomanEncoding, etc.)
//   - Custom encodings from /Encoding dictionary
//   - ToUnicode CMaps for Unicode conversion
//
// # CMap Support
//
// CMaps (Character Maps) handle character code to Unicode mapping:
//
//   - Embedded ToUnicode CMaps
//   - Predefined CJK CMaps
//   - CID-to-Unicode mapping
package font
