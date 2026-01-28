package core

import (
	"fmt"

	"github.com/tsawler/tabula/internal/filters"
)

// Decode decodes the stream data according to the Filter(s) specified in the
// stream dictionary. It supports FlateDecode, ASCIIHexDecode, ASCII85Decode,
// and filter chains. Returns the decoded data or an error.
func (s *Stream) Decode() ([]byte, error) {
	// Check if there's a filter
	filterObj := s.Dict.Get("Filter")
	if filterObj == nil {
		// No filter - return raw data
		return s.Data, nil
	}

	// Get decode parameters
	paramsObj := s.Dict.Get("DecodeParms")

	// Handle single filter
	if filterName, ok := filterObj.(Name); ok {
		return decodeWithFilter(s.Data, string(filterName), paramsObjToDict(paramsObj))
	}

	// Handle filter array (chain of filters)
	if filterArray, ok := filterObj.(Array); ok {
		data := s.Data

		// Apply each filter in sequence
		for i, filter := range filterArray {
			filterName, ok := filter.(Name)
			if !ok {
				return nil, fmt.Errorf("filter %d is not a name: %T", i, filter)
			}

			// Get corresponding decode params if array
			var params Dict
			if paramsArray, ok := paramsObj.(Array); ok {
				if i < len(paramsArray) {
					params = paramsObjToDict(paramsArray[i])
				}
			} else {
				// Single params for all filters
				params = paramsObjToDict(paramsObj)
			}

			var err error
			data, err = decodeWithFilter(data, string(filterName), params)
			if err != nil {
				return nil, fmt.Errorf("filter %d (%s) failed: %w", i, filterName, err)
			}
		}

		return data, nil
	}

	return nil, fmt.Errorf("invalid Filter type: %T", filterObj)
}

// decodeWithFilter applies a single decompression filter to data.
// The filterName should be a PDF filter name (e.g., "FlateDecode", "ASCIIHexDecode").
func decodeWithFilter(data []byte, filterName string, params Dict) ([]byte, error) {
	switch filterName {
	case "FlateDecode", "Fl":
		return filters.FlateDecode(data, dictToParams(params))

	case "ASCIIHexDecode", "AHx":
		return filters.ASCIIHexDecode(data)

	case "ASCII85Decode", "A85":
		return filters.ASCII85Decode(data)

	case "LZWDecode", "LZW":
		return nil, fmt.Errorf("LZWDecode not yet implemented")

	case "RunLengthDecode", "RL":
		return nil, fmt.Errorf("RunLengthDecode not yet implemented")

	case "CCITTFaxDecode", "CCF":
		return filters.CCITTFaxDecode(data, dictToParams(params))

	case "JBIG2Decode":
		return nil, fmt.Errorf("JBIG2Decode not yet implemented")

	case "DCTDecode", "DCT":
		// JPEG - return as-is for now (will handle in image extraction)
		return data, nil

	case "JPXDecode":
		// JPEG2000 - return as-is for now
		return data, nil

	case "Crypt":
		return nil, fmt.Errorf("Crypt filter not yet implemented")

	default:
		return nil, fmt.Errorf("unknown filter: %s", filterName)
	}
}

// paramsObjToDict converts a DecodeParms object to a Dict.
// Returns nil if the object is nil, Null, or not a Dict.
func paramsObjToDict(obj Object) Dict {
	if obj == nil {
		return nil
	}

	if dict, ok := obj.(Dict); ok {
		return dict
	}

	// Null is treated as no params
	if _, ok := obj.(Null); ok {
		return nil
	}

	return nil
}

// dictToParams converts a core.Dict to filters.Params, translating PDF object
// types to Go primitive types (Int->int, Real->float64, Bool->bool, etc.).
func dictToParams(dict Dict) filters.Params {
	if dict == nil {
		return nil
	}

	params := make(filters.Params)
	for k, v := range dict {
		// Convert PDF objects to Go primitives
		switch obj := v.(type) {
		case Int:
			params[k] = int(obj)
		case Real:
			params[k] = float64(obj)
		case Bool:
			params[k] = bool(obj)
		case String:
			params[k] = string(obj)
		case Name:
			params[k] = string(obj)
		default:
			// Keep other types as-is
			params[k] = v
		}
	}
	return params
}
