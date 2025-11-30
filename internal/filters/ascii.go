package filters

import (
	"bytes"
	"fmt"
)

// ASCIIHexDecode decodes ASCII hexadecimal encoded data.
// Each pair of hexadecimal digits (0-9, A-F, a-f) represents one byte.
// Whitespace is ignored, and > marks end of data.
func ASCIIHexDecode(data []byte) ([]byte, error) {
	var result bytes.Buffer

	i := 0
	for i < len(data) {
		// Skip whitespace
		if isWhitespace(data[i]) {
			i++
			continue
		}

		// Check for EOD marker
		if data[i] == '>' {
			break
		}

		// Read two hex digits
		if i+1 >= len(data) {
			// Odd number of digits - assume trailing 0
			b, err := hexDigitToByte(data[i])
			if err != nil {
				return nil, err
			}
			result.WriteByte(b << 4)
			break
		}

		// Get first hex digit
		b1, err := hexDigitToByte(data[i])
		if err != nil {
			return nil, err
		}
		i++

		// Skip whitespace before second digit
		for i < len(data) && isWhitespace(data[i]) {
			i++
		}

		if i >= len(data) || data[i] == '>' {
			// Odd number of digits
			result.WriteByte(b1 << 4)
			break
		}

		// Get second hex digit
		b2, err := hexDigitToByte(data[i])
		if err != nil {
			return nil, err
		}
		i++

		// Combine two hex digits into one byte
		result.WriteByte((b1 << 4) | b2)
	}

	return result.Bytes(), nil
}

// ASCII85Decode decodes ASCII base-85 (Ascii85) encoded data.
// Each group of 5 ASCII characters (! to u, values 33-117) represents 4 bytes.
// The special character 'z' represents four zero bytes. The sequence ~> marks
// end of data.
func ASCII85Decode(data []byte) ([]byte, error) {
	var result bytes.Buffer

	// Skip leading whitespace
	i := 0
	for i < len(data) && isWhitespace(data[i]) {
		i++
	}

	for i < len(data) {
		// Skip whitespace
		if isWhitespace(data[i]) {
			i++
			continue
		}

		// Check for EOD marker ~>
		if i+1 < len(data) && data[i] == '~' && data[i+1] == '>' {
			break
		}

		// Special case: 'z' represents 0x00000000
		if data[i] == 'z' {
			result.Write([]byte{0, 0, 0, 0})
			i++
			continue
		}

		// Read up to 5 base-85 digits
		digits := make([]byte, 0, 5)
		for len(digits) < 5 && i < len(data) {
			if isWhitespace(data[i]) {
				i++
				continue
			}

			if i+1 < len(data) && data[i] == '~' && data[i+1] == '>' {
				break
			}

			if data[i] < '!' || data[i] > 'u' {
				return nil, fmt.Errorf("invalid ASCII85 character: %c", data[i])
			}

			digits = append(digits, data[i]-'!')
			i++
		}

		if len(digits) == 0 {
			break
		}

		// Pad incomplete group with 'u' (84 = highest ASCII85 value)
		// This is required for correct decoding
		numBytes := len(digits) - 1
		if numBytes > 4 {
			numBytes = 4
		}

		for len(digits) < 5 {
			digits = append(digits, 84) // 'u' - '!' = 84
		}

		// Convert base-85 to binary
		// Each group of 5 digits represents 4 bytes
		value := uint32(0)
		for _, d := range digits {
			value = value*85 + uint32(d)
		}

		// Extract bytes (big-endian)
		for j := 0; j < numBytes; j++ {
			result.WriteByte(byte(value >> (24 - j*8)))
		}
	}

	return result.Bytes(), nil
}

// hexDigitToByte converts a hexadecimal character to its numeric value (0-15).
func hexDigitToByte(c byte) (byte, error) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', nil
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, nil
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, nil
	default:
		return 0, fmt.Errorf("invalid hex digit: %c", c)
	}
}

// isWhitespace reports whether c is a PDF whitespace character.
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == 0
}
