package filters

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

// Params represents decode parameters (map of key-value pairs)
type Params map[string]interface{}

// FlateDecode decodes Flate (zlib/deflate) compressed data
// This is the most common compression filter in PDFs
func FlateDecode(data []byte, params Params) ([]byte, error) {
	// Decompress using zlib
	decompressed, err := zlibDecompress(data)
	if err != nil {
		return nil, fmt.Errorf("zlib decompression failed: %w", err)
	}

	// Apply predictor if specified
	if params != nil {
		if predictorObj, ok := params["Predictor"]; ok && predictorObj != nil {
			predictor := getIntParam(params, "Predictor", 1)
			if predictor != 1 {
				decompressed, err = applyPredictor(decompressed, predictor, params)
				if err != nil {
					return nil, fmt.Errorf("predictor failed: %w", err)
				}
			}
		}
	}

	return decompressed, nil
}

// zlibDecompress decompresses zlib-compressed data
func zlibDecompress(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib reader: %w", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}

	return buf.Bytes(), nil
}

// applyPredictor applies PNG predictor algorithms
// PDF supports PNG predictors for better compression of image data
func applyPredictor(data []byte, predictor int, params Params) ([]byte, error) {
	// Predictor values:
	// 1 = No prediction (identity)
	// 2 = TIFF Predictor 2
	// 10-15 = PNG predictors

	if predictor == 1 {
		// No prediction - return as-is
		return data, nil
	}

	if predictor == 2 {
		// TIFF Predictor 2 - not commonly used in PDFs
		return applyTIFFPredictor2(data, params)
	}

	if predictor >= 10 && predictor <= 15 {
		// PNG predictors
		return applyPNGPredictor(data, predictor, params)
	}

	return nil, fmt.Errorf("unsupported predictor: %d", predictor)
}

// applyTIFFPredictor2 applies TIFF Predictor 2
func applyTIFFPredictor2(data []byte, params Params) ([]byte, error) {
	columns := getIntParam(params, "Columns", 1)
	colors := getIntParam(params, "Colors", 1)
	bpc := getIntParam(params, "BitsPerComponent", 8)

	if bpc != 8 {
		return nil, fmt.Errorf("TIFF Predictor 2 only supports 8 bits per component, got %d", bpc)
	}

	rowSize := columns * colors
	if len(data)%rowSize != 0 {
		return nil, fmt.Errorf("data size %d is not a multiple of row size %d", len(data), rowSize)
	}

	result := make([]byte, len(data))

	for row := 0; row < len(data)/rowSize; row++ {
		rowStart := row * rowSize
		for col := 0; col < rowSize; col++ {
			idx := rowStart + col
			if col < colors {
				// First pixel in row - no prediction
				result[idx] = data[idx]
			} else {
				// Predict from left pixel
				result[idx] = data[idx] + result[idx-colors]
			}
		}
	}

	return result, nil
}

// applyPNGPredictor applies PNG predictor algorithms
func applyPNGPredictor(data []byte, predictor int, params Params) ([]byte, error) {
	columns := getIntParam(params, "Columns", 1)
	colors := getIntParam(params, "Colors", 1)
	bpc := getIntParam(params, "BitsPerComponent", 8)

	if bpc != 8 {
		return nil, fmt.Errorf("PNG predictor only supports 8 bits per component, got %d", bpc)
	}

	// PNG predictors work on rows with a predictor byte at the start of each row
	bytesPerPixel := colors
	rowSize := columns*colors + 1 // +1 for predictor byte

	if len(data)%rowSize != 0 {
		return nil, fmt.Errorf("data size %d is not a multiple of row size %d", len(data), rowSize)
	}

	numRows := len(data) / rowSize
	result := make([]byte, numRows*columns*colors) // Output without predictor bytes

	for row := 0; row < numRows; row++ {
		rowStart := row * rowSize
		predictorByte := data[rowStart]
		rowData := data[rowStart+1 : rowStart+rowSize]

		// Decode this row
		decodedRow, err := decodePNGRow(rowData, predictorByte, bytesPerPixel, row, result, columns*colors)
		if err != nil {
			return nil, fmt.Errorf("failed to decode row %d: %w", row, err)
		}

		// Copy to result
		copy(result[row*columns*colors:(row+1)*columns*colors], decodedRow)
	}

	return result, nil
}

// decodePNGRow decodes a single PNG-predicted row
func decodePNGRow(rowData []byte, predictor byte, bytesPerPixel int, rowNum int, prevRows []byte, rowLength int) ([]byte, error) {
	result := make([]byte, len(rowData))

	for i := 0; i < len(rowData); i++ {
		var predicted byte

		switch predictor {
		case 0: // None
			predicted = 0

		case 1: // Sub (predict from left)
			if i >= bytesPerPixel {
				predicted = result[i-bytesPerPixel]
			}

		case 2: // Up (predict from above)
			if rowNum > 0 {
				predicted = prevRows[(rowNum-1)*rowLength+i]
			}

		case 3: // Average (average of left and up)
			var left, up byte
			if i >= bytesPerPixel {
				left = result[i-bytesPerPixel]
			}
			if rowNum > 0 {
				up = prevRows[(rowNum-1)*rowLength+i]
			}
			predicted = byte((int(left) + int(up)) / 2)

		case 4: // Paeth (Paeth predictor)
			var left, up, upLeft byte
			if i >= bytesPerPixel {
				left = result[i-bytesPerPixel]
			}
			if rowNum > 0 {
				up = prevRows[(rowNum-1)*rowLength+i]
				if i >= bytesPerPixel {
					upLeft = prevRows[(rowNum-1)*rowLength+i-bytesPerPixel]
				}
			}
			predicted = paethPredictor(left, up, upLeft)

		default:
			return nil, fmt.Errorf("unknown PNG predictor: %d", predictor)
		}

		result[i] = rowData[i] + predicted
	}

	return result, nil
}

// paethPredictor implements the Paeth predictor algorithm
// This is used by PNG to predict pixel values
func paethPredictor(a, b, c byte) byte {
	// a = left, b = above, c = upper left
	p := int(a) + int(b) - int(c)
	pa := abs(p - int(a))
	pb := abs(p - int(b))
	pc := abs(p - int(c))

	if pa <= pb && pa <= pc {
		return a
	} else if pb <= pc {
		return b
	}
	return c
}

// getIntParam gets an integer parameter from DecodeParms, with default
func getIntParam(params Params, key string, defaultValue int) int {
	if params == nil {
		return defaultValue
	}

	obj, ok := params[key]
	if !ok {
		return defaultValue
	}

	// Handle various integer types
	switch v := obj.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case float64:
		return int(v)
	default:
		return defaultValue
	}
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
