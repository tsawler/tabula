package font

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/tsawler/tabula/core"
)

// CMap represents a character map that maps character codes to Unicode
type CMap struct {
	// Single character mappings: charCode -> unicode string
	charMappings map[uint32]string

	// Range mappings for efficiency
	rangeMappings []CMapRange

	// Code space byte width (1, 2, 3, or 4 bytes per character code)
	// Determined from begincodespacerange/endcodespacerange
	// 0 means not specified (will try multiple widths)
	byteWidth int

	// Actual byte width observed in bfchar/bfrange source codes
	// This may differ from byteWidth when CMaps use shorter codes than codespacerange allows
	actualByteWidth int
}

// CMapRange represents a range of character code to Unicode mappings
type CMapRange struct {
	StartCode    uint32
	EndCode      uint32
	StartUnicode uint32
}

// NewCMap creates a new empty CMap
func NewCMap() *CMap {
	return &CMap{
		charMappings:  make(map[uint32]string),
		rangeMappings: make([]CMapRange, 0),
	}
}

// ParseToUnicodeCMap parses a ToUnicode CMap stream
func ParseToUnicodeCMap(stream *core.Stream) (*CMap, error) {
	if stream == nil {
		return nil, fmt.Errorf("stream is nil")
	}

	// Decode the stream
	data, err := stream.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode stream: %w", err)
	}

	return parseCMapData(data)
}

// parseCMapData parses the CMap data
func parseCMapData(data []byte) (*CMap, error) {
	cmap := NewCMap()

	// Convert to string for easier parsing
	content := string(data)

	// Parse begincodespacerange/endcodespacerange to determine byte width
	if err := cmap.parseCodeSpaceRange(content); err != nil {
		// Non-fatal - just log and continue
		_ = err
	}

	// Parse beginbfchar/endbfchar sections
	if err := cmap.parseBfChar(content); err != nil {
		// Non-fatal - just log and continue
		_ = err
	}

	// Parse beginbfrange/endbfrange sections
	if err := cmap.parseBfRange(content); err != nil {
		// Non-fatal - just log and continue
		_ = err
	}

	return cmap, nil
}

// parseCodeSpaceRange parses begincodespacerange/endcodespacerange sections
// to determine the byte width of character codes
func (cm *CMap) parseCodeSpaceRange(content string) error {
	// Find first begincodespacerange/endcodespacerange section
	beginIdx := strings.Index(content, "begincodespacerange")
	if beginIdx == -1 {
		return nil // No codespacerange section
	}

	endIdx := strings.Index(content[beginIdx:], "endcodespacerange")
	if endIdx == -1 {
		return nil
	}
	endIdx += beginIdx

	// Extract section content
	section := content[beginIdx+len("begincodespacerange") : endIdx]

	// Parse the first code range to determine byte width
	// Format: <low> <high>
	// Example: <0000> <FFFF> means 2-byte codes (4 hex digits = 2 bytes)
	lines := strings.Split(section, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Find hex strings
		var hexStrings []string
		startIdx := 0
		for {
			idx := strings.Index(line[startIdx:], "<")
			if idx == -1 {
				break
			}
			idx += startIdx
			endIdx := strings.Index(line[idx:], ">")
			if endIdx == -1 {
				break
			}
			endIdx += idx

			hexStr := line[idx+1 : endIdx]
			hexStrings = append(hexStrings, hexStr)
			startIdx = endIdx + 1
		}

		if len(hexStrings) >= 2 {
			// Determine byte width from first hex string length
			hexLen := len(hexStrings[0])
			// Each 2 hex digits = 1 byte
			cm.byteWidth = hexLen / 2
			if hexLen%2 != 0 {
				cm.byteWidth = (hexLen + 1) / 2
			}
			break // We got what we needed
		}
	}

	return nil
}

// parseBfChar parses beginbfchar/endbfchar sections
// Format: <srcCode> <dstUnicode>
func (cm *CMap) parseBfChar(content string) error {
	// Find all beginbfchar/endbfchar sections
	start := 0
	for {
		beginIdx := strings.Index(content[start:], "beginbfchar")
		if beginIdx == -1 {
			break
		}
		beginIdx += start

		endIdx := strings.Index(content[beginIdx:], "endbfchar")
		if endIdx == -1 {
			break
		}
		endIdx += beginIdx

		// Extract section content
		section := content[beginIdx+len("beginbfchar") : endIdx]

		// Parse mappings
		if err := cm.parseBfCharSection(section); err != nil {
			return err
		}

		start = endIdx + len("endbfchar")
	}

	return nil
}

// parseBfCharSection parses a single beginbfchar/endbfchar section
func (cm *CMap) parseBfCharSection(section string) error {
	// Split into lines
	lines := strings.Split(section, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse hex strings: <srcCode> <dstUnicode>
		// Split by < to handle <21><d83d dc4b> format
		var srcHex, dstHex string

		// Find all hex strings in the line
		hexStrings := make([]string, 0)
		startIdx := 0
		for {
			idx := strings.Index(line[startIdx:], "<")
			if idx == -1 {
				break
			}
			idx += startIdx
			endIdx := strings.Index(line[idx:], ">")
			if endIdx == -1 {
				break
			}
			endIdx += idx

			hexStr := line[idx+1 : endIdx]
			hexStrings = append(hexStrings, hexStr)
			startIdx = endIdx + 1
		}

		if len(hexStrings) < 2 {
			continue
		}

		srcHex = hexStrings[0]
		dstHex = hexStrings[1]

		if srcHex == "" || dstHex == "" {
			continue
		}

		// Track actual byte width from source code hex length
		// This helps detect when CMaps use shorter codes than codespacerange suggests
		srcHexLen := len(srcHex)
		if srcHexLen%2 != 0 {
			srcHexLen++ // Account for odd-length padding
		}
		srcByteWidth := srcHexLen / 2
		if srcByteWidth > cm.actualByteWidth {
			cm.actualByteWidth = srcByteWidth
		}

		// Convert source code to uint32
		srcCode, err := parseHexToUint32(srcHex)
		if err != nil {
			continue
		}

		// Convert destination to Unicode string
		unicode, err := hexToUnicode(dstHex)
		if err != nil {
			// Debug: show the error
			_ = err // Silently continue for now
			continue
		}

		cm.charMappings[srcCode] = unicode
	}

	return nil
}

// parseBfRange parses beginbfrange/endbfrange sections
// Format: <srcCodeStart> <srcCodeEnd> <dstUnicode>
// or: <srcCodeStart> <srcCodeEnd> [<u1> <u2> <u3> ...]
func (cm *CMap) parseBfRange(content string) error {
	// Find all beginbfrange/endbfrange sections
	start := 0
	for {
		beginIdx := strings.Index(content[start:], "beginbfrange")
		if beginIdx == -1 {
			break
		}
		beginIdx += start

		endIdx := strings.Index(content[beginIdx:], "endbfrange")
		if endIdx == -1 {
			break
		}
		endIdx += beginIdx

		// Extract section content
		section := content[beginIdx+len("beginbfrange") : endIdx]

		// Parse mappings
		if err := cm.parseBfRangeSection(section); err != nil {
			return err
		}

		start = endIdx + len("endbfrange")
	}

	return nil
}

// parseBfRangeSection parses a single beginbfrange/endbfrange section
func (cm *CMap) parseBfRangeSection(section string) error {
	// Split into lines
	lines := strings.Split(section, "\n")

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}

		// Check if this is an array format (multi-line)
		if strings.Contains(line, "[") {
			// Array format: <start> <end> [<u1> <u2> ...]
			// This may span multiple lines
			fullLine := line
			for !strings.Contains(fullLine, "]") && i+1 < len(lines) {
				i++
				fullLine += " " + strings.TrimSpace(lines[i])
			}
			cm.parseBfRangeArray(fullLine)
			i++
			continue
		}

		// Simple format: <start> <end> <unicode>
		// Find all hex strings in the line to handle tight packing like <21><21><0052>
		hexStrings := make([]string, 0)
		startIdx := 0
		for {
			idx := strings.Index(line[startIdx:], "<")
			if idx == -1 {
				break
			}
			idx += startIdx
			endIdx := strings.Index(line[idx:], ">")
			if endIdx == -1 {
				break
			}
			endIdx += idx

			hexStr := line[idx+1 : endIdx]
			hexStrings = append(hexStrings, hexStr)
			startIdx = endIdx + 1
		}

		if len(hexStrings) < 3 {
			i++
			continue
		}

		startHex := hexStrings[0]
		endHex := hexStrings[1]
		dstHex := hexStrings[2]

		if startHex == "" || endHex == "" || dstHex == "" {
			i++
			continue
		}

		// Track actual byte width from source code hex length
		srcHexLen := len(startHex)
		if srcHexLen%2 != 0 {
			srcHexLen++
		}
		srcByteWidth := srcHexLen / 2
		if srcByteWidth > cm.actualByteWidth {
			cm.actualByteWidth = srcByteWidth
		}

		startCode, err1 := parseHexToUint32(startHex)
		endCode, err2 := parseHexToUint32(endHex)
		dstUnicode, err3 := parseHexToUint32(dstHex)

		if err1 != nil || err2 != nil || err3 != nil {
			i++
			continue
		}

		// Add range mapping
		cm.rangeMappings = append(cm.rangeMappings, CMapRange{
			StartCode:    startCode,
			EndCode:      endCode,
			StartUnicode: dstUnicode,
		})

		i++
	}

	return nil
}

// parseBfRangeArray parses array format: <start> <end> [<u1> <u2> ...]
func (cm *CMap) parseBfRangeArray(line string) {
	// Extract start and end codes
	// Find hex strings for start/end
	hexStrings := make([]string, 0)
	startIdx := 0
	// Only look before the '['
	bracketIdx := strings.Index(line, "[")
	if bracketIdx == -1 {
		return
	}

	preBracket := line[:bracketIdx]
	for {
		idx := strings.Index(preBracket[startIdx:], "<")
		if idx == -1 {
			break
		}
		idx += startIdx
		endIdx := strings.Index(preBracket[idx:], ">")
		if endIdx == -1 {
			break
		}
		endIdx += idx

		hexStr := preBracket[idx+1 : endIdx]
		hexStrings = append(hexStrings, hexStr)
		startIdx = endIdx + 1
	}

	if len(hexStrings) < 2 {
		return
	}

	startHex := hexStrings[0]
	endHex := hexStrings[1]

	startCode, err1 := parseHexToUint32(startHex)
	endCode, err2 := parseHexToUint32(endHex)

	if err1 != nil || err2 != nil {
		return
	}

	// Extract array content
	arrayStart := strings.Index(line, "[")
	arrayEnd := strings.Index(line, "]")
	if arrayStart == -1 || arrayEnd == -1 {
		return
	}

	arrayContent := line[arrayStart+1 : arrayEnd]

	// Parse hex strings in array content
	arrayHexStrings := make([]string, 0)
	startIdx = 0
	for {
		idx := strings.Index(arrayContent[startIdx:], "<")
		if idx == -1 {
			break
		}
		idx += startIdx
		endIdx := strings.Index(arrayContent[idx:], ">")
		if endIdx == -1 {
			break
		}
		endIdx += idx

		hexStr := arrayContent[idx+1 : endIdx]
		arrayHexStrings = append(arrayHexStrings, hexStr)
		startIdx = endIdx + 1
	}

	// Map each character code to its Unicode value
	currentCode := startCode
	for _, hex := range arrayHexStrings {
		if hex == "" {
			continue
		}

		unicode, err := hexToUnicode(hex)
		if err == nil && currentCode <= endCode {
			cm.charMappings[currentCode] = unicode
		}

		currentCode++
	}
}

// Lookup looks up a character code and returns the Unicode string
// Returns empty string if no mapping is found (caller should handle fallback)
func (cm *CMap) Lookup(charCode uint32) string {
	// Check direct mappings first
	if unicode, ok := cm.charMappings[charCode]; ok {
		return unicode
	}

	// Check range mappings
	for _, r := range cm.rangeMappings {
		if charCode >= r.StartCode && charCode <= r.EndCode {
			// Calculate Unicode value
			offset := charCode - r.StartCode
			unicodeValue := r.StartUnicode + offset
			return string(rune(unicodeValue))
		}
	}

	// No mapping found - return empty string
	// Let the caller decide how to handle unmapped codes
	return ""
}

// LookupString decodes a string of character codes to Unicode
func (cm *CMap) LookupString(data []byte) string {
	if cm == nil {
		return string(data)
	}

	// Determine the effective byte width to use
	// Some CMaps declare a wide codespacerange (e.g., <0000><FFFF> = 2 bytes)
	// but only have bfchar entries with shorter codes (e.g., <20> = 1 byte)
	// In such cases, prefer the actual byte width from bfchar entries
	effectiveWidth := cm.byteWidth
	if cm.actualByteWidth > 0 && cm.actualByteWidth < cm.byteWidth {
		// The actual bfchar entries use shorter codes than codespacerange suggests
		effectiveWidth = cm.actualByteWidth
	}

	// If we have a known byte width, use it
	if effectiveWidth > 0 {
		return cm.lookupStringWithWidth(data, effectiveWidth)
	}

	// Otherwise, try different widths (fallback for CMaps without codespacerange)
	var result strings.Builder
	i := 0
	for i < len(data) {
		// Try 1-byte code first (most common for Latin and subset fonts)
		code1 := uint32(data[i])
		if unicode := cm.Lookup(code1); unicode != "" {
			result.WriteString(unicode)
			i++
			continue
		}

		// Try 2-byte code (common for CJK and some complex fonts)
		if i+1 < len(data) {
			code2 := uint32(data[i])<<8 | uint32(data[i+1])
			if unicode := cm.Lookup(code2); unicode != "" {
				result.WriteString(unicode)
				i += 2
				continue
			}
		}

		// No mapping found - try to interpret as direct Unicode (fallback)
		// This handles cases where the PDF uses character codes as Unicode
		if code1 < 0x110000 { // Valid Unicode range
			result.WriteRune(rune(code1))
		}
		i++
	}

	return result.String()
}

// lookupStringWithWidth decodes using a specific byte width
func (cm *CMap) lookupStringWithWidth(data []byte, width int) string {
	var result strings.Builder

	i := 0
	for i < len(data) {
		// Check if we have enough bytes
		if i+width > len(data) {
			// Not enough bytes for a complete code
			// Handle remaining bytes as 1-byte codes
			for i < len(data) {
				code := uint32(data[i])
				if unicode := cm.Lookup(code); unicode != "" {
					result.WriteString(unicode)
				} else if code < 0x110000 {
					result.WriteRune(rune(code))
				}
				i++
			}
			break
		}

		// Build multi-byte code
		var code uint32
		for j := 0; j < width; j++ {
			code = (code << 8) | uint32(data[i+j])
		}

		// Lookup the code
		if unicode := cm.Lookup(code); unicode != "" {
			result.WriteString(unicode)
		} else if code < 0x110000 {
			// Fallback to direct Unicode interpretation
			result.WriteRune(rune(code))
		}

		i += width
	}

	return result.String()
}

// Helper functions

// extractHexString extracts hex content from <ABCD> format
func extractHexString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return ""
	}
	if s[0] == '<' && s[len(s)-1] == '>' {
		return s[1 : len(s)-1]
	}
	return ""
}

// parseHexToUint32 parses a hex string to uint32
func parseHexToUint32(hexStr string) (uint32, error) {
	// Pad to even length for malformed PDFs with odd-length hex
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}

	val, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return 0, err
	}

	return uint32(val), nil
}

// hexToUnicode converts hex string to Unicode string
func hexToUnicode(hexStr string) (string, error) {
	// Remove any whitespace from hex string
	hexStr = strings.ReplaceAll(hexStr, " ", "")
	hexStr = strings.ReplaceAll(hexStr, "\t", "")
	hexStr = strings.ReplaceAll(hexStr, "\n", "")
	hexStr = strings.ReplaceAll(hexStr, "\r", "")

	// Pad to even length for malformed PDFs with odd-length hex
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}

	// Decode hex to bytes
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}

	// Convert to Unicode
	// Handle UTF-16BE (common in PDFs)
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		// UTF-16BE with BOM
		return decodeUTF16BE(data[2:])
	}

	// If 2+ bytes, assume UTF-16BE
	if len(data) >= 2 {
		return decodeUTF16BE(data)
	}

	// Single byte - ASCII
	if len(data) == 1 {
		return string(rune(data[0])), nil
	}

	return "", fmt.Errorf("invalid unicode data")
}

// decodeUTF16BE decodes UTF-16BE bytes to string
func decodeUTF16BE(data []byte) (string, error) {
	if len(data)%2 != 0 {
		return "", fmt.Errorf("invalid UTF-16BE data length")
	}

	var result strings.Builder
	buf := bytes.NewReader(data)

	for buf.Len() > 0 {
		// Read 2 bytes for each UTF-16 code unit
		var b1, b2 byte
		var err error

		b1, err = buf.ReadByte()
		if err != nil {
			break
		}
		b2, err = buf.ReadByte()
		if err != nil {
			break
		}

		codeUnit := uint16(b1)<<8 | uint16(b2)

		// Handle surrogate pairs (for characters beyond BMP)
		if codeUnit >= 0xD800 && codeUnit <= 0xDBFF {
			// High surrogate - read low surrogate
			if buf.Len() >= 2 {
				b1, _ = buf.ReadByte()
				b2, _ = buf.ReadByte()
				lowSurrogate := uint16(b1)<<8 | uint16(b2)

				if lowSurrogate >= 0xDC00 && lowSurrogate <= 0xDFFF {
					// Combine surrogates
					codePoint := 0x10000 + ((uint32(codeUnit-0xD800) << 10) | uint32(lowSurrogate-0xDC00))
					result.WriteRune(rune(codePoint))
					continue
				}
			}
		}

		result.WriteRune(rune(codeUnit))
	}

	return result.String(), nil
}
