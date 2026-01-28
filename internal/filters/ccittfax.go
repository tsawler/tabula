package filters

import (
	"bytes"
	"io"

	"golang.org/x/image/ccitt"
)

// CCITTFaxDecode decodes CCITT Group 3/4 fax compressed data.
// This is commonly used for bi-level (black and white) images in PDFs,
// particularly for scanned documents.
//
// Parameters from the PDF decode parameters dictionary:
//   - K: Group selector (-1=Group4, 0=Group3 1D, >0=Group3 2D)
//   - Columns: Image width in pixels (default 1728)
//   - Rows: Image height in pixels (default 0, uses AutoDetectHeight)
//   - BlackIs1: Bit interpretation (default false, maps to ccitt.Options.Invert)
func CCITTFaxDecode(data []byte, params Params) ([]byte, error) {
	columns := getIntParam(params, "Columns", 1728)
	rows := getIntParam(params, "Rows", 0)
	k := getIntParam(params, "K", 0)
	blackIs1 := getBoolParam(params, "BlackIs1", false)

	// Determine subformat from K parameter
	// K < 0: pure Group 4
	// K = 0: pure Group 3 (1-dimensional)
	// K > 0: mixed Group 3 (2-dimensional)
	var sf ccitt.SubFormat
	if k < 0 {
		sf = ccitt.Group4
	} else {
		sf = ccitt.Group3
	}

	// PDF uses MSB order, blackIs1 maps to Invert option
	opts := &ccitt.Options{Invert: blackIs1}

	// Use AutoDetectHeight if rows not specified
	if rows == 0 {
		rows = ccitt.AutoDetectHeight
	}

	reader := ccitt.NewReader(bytes.NewReader(data), ccitt.MSB, sf, columns, rows, opts)
	return io.ReadAll(reader)
}

// getBoolParam extracts a boolean parameter from Params, returning defaultValue
// if the parameter is missing or cannot be converted to a boolean.
func getBoolParam(params Params, key string, defaultValue bool) bool {
	if params == nil {
		return defaultValue
	}

	obj, ok := params[key]
	if !ok {
		return defaultValue
	}

	switch v := obj.(type) {
	case bool:
		return v
	default:
		return defaultValue
	}
}
