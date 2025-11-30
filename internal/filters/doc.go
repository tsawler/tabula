// Package filters provides PDF stream decompression filters.
//
// PDF streams can be compressed using various algorithms. This package
// implements the standard PDF decompression filters.
//
// # Supported Filters
//
// FlateDecode (zlib/deflate):
//
//	decoded, err := filters.FlateDecode(data, params)
//
// FlateDecode supports PNG predictors for improved compression of image data.
// The Predictor parameter specifies the algorithm:
//   - 1: No prediction (default)
//   - 2: TIFF Predictor 2
//   - 10-15: PNG predictors (None, Sub, Up, Average, Paeth)
//
// ASCIIHexDecode:
//
//	decoded, err := filters.ASCIIHexDecode(data)
//
// Decodes hexadecimal-encoded data. Whitespace is ignored.
//
// ASCII85Decode:
//
//	decoded, err := filters.ASCII85Decode(data)
//
// Decodes ASCII base-85 encoded data (also known as Ascii85).
//
// # Decode Parameters
//
// Filters accept a Params map for additional parameters:
//
//	params := filters.Params{
//	    "Predictor": 12,
//	    "Columns":   100,
//	    "Colors":    3,
//	}
//	decoded, err := filters.FlateDecode(data, params)
package filters
