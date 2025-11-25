package text

import (
	"unicode"
)

// Direction represents text direction
type Direction int

const (
	// LTR (Left-to-Right) for Latin, Cyrillic, etc.
	LTR Direction = iota
	// RTL (Right-to-Left) for Arabic, Hebrew, etc.
	RTL
	// Neutral for numbers, punctuation, etc.
	Neutral
)

// String returns string representation of direction
func (d Direction) String() string {
	switch d {
	case LTR:
		return "LTR"
	case RTL:
		return "RTL"
	case Neutral:
		return "Neutral"
	default:
		return "Unknown"
	}
}

// DetectDirection detects the dominant text direction of a string
// based on Unicode character properties
func DetectDirection(text string) Direction {
	if text == "" {
		return Neutral
	}

	ltrCount := 0
	rtlCount := 0

	for _, r := range text {
		dir := GetCharDirection(r)
		switch dir {
		case LTR:
			ltrCount++
		case RTL:
			rtlCount++
		}
	}

	// If no strong directional characters, it's neutral
	if ltrCount == 0 && rtlCount == 0 {
		return Neutral
	}

	// Return the dominant direction
	if rtlCount > ltrCount {
		return RTL
	}
	return LTR
}

// GetCharDirection returns the direction of a single character
// based on Unicode character properties
func GetCharDirection(r rune) Direction {
	// Numbers and neutral characters (check first, before script checks)
	if unicode.IsDigit(r) || unicode.IsPunct(r) || unicode.IsSpace(r) || unicode.IsSymbol(r) {
		return Neutral
	}

	// RTL scripts (primary RTL languages)
	if isArabic(r) || isHebrew(r) || isSyriac(r) || isThaana(r) || isNKo(r) {
		return RTL
	}

	// LTR scripts (most common)
	if isLatin(r) || isCyrillic(r) || isGreek(r) || isArmenian(r) || isGeorgian(r) || isThai(r) {
		return LTR
	}

	// CJK scripts are LTR in modern usage (though historically top-to-bottom)
	if isCJK(r) {
		return LTR
	}

	// Default to LTR for unknown scripts
	return LTR
}

// isArabic checks if rune is in Arabic Unicode block
// Arabic: U+0600–U+06FF
// Arabic Supplement: U+0750–U+077F
// Arabic Extended-A: U+08A0–U+08FF
// Arabic Presentation Forms-A: U+FB50–U+FDFF
// Arabic Presentation Forms-B: U+FE70–U+FEFF
func isArabic(r rune) bool {
	return (r >= 0x0600 && r <= 0x06FF) ||
		(r >= 0x0750 && r <= 0x077F) ||
		(r >= 0x08A0 && r <= 0x08FF) ||
		(r >= 0xFB50 && r <= 0xFDFF) ||
		(r >= 0xFE70 && r <= 0xFEFF)
}

// isHebrew checks if rune is in Hebrew Unicode block
// Hebrew: U+0590–U+05FF
// Hebrew Presentation Forms: U+FB1D–U+FB4F
func isHebrew(r rune) bool {
	return (r >= 0x0590 && r <= 0x05FF) ||
		(r >= 0xFB1D && r <= 0xFB4F)
}

// isSyriac checks if rune is in Syriac Unicode block
// Syriac: U+0700–U+074F
func isSyriac(r rune) bool {
	return r >= 0x0700 && r <= 0x074F
}

// isThaana checks if rune is in Thaana Unicode block (Maldivian)
// Thaana: U+0780–U+07BF
func isThaana(r rune) bool {
	return r >= 0x0780 && r <= 0x07BF
}

// isNKo checks if rune is in N'Ko Unicode block (West African)
// N'Ko: U+07C0–U+07FF
func isNKo(r rune) bool {
	return r >= 0x07C0 && r <= 0x07FF
}

// isLatin checks if rune is in Latin Unicode blocks
// Basic Latin: U+0000–U+007F
// Latin-1 Supplement: U+0080–U+00FF
// Latin Extended-A: U+0100–U+017F
// Latin Extended-B: U+0180–U+024F
func isLatin(r rune) bool {
	return (r >= 0x0000 && r <= 0x007F) ||
		(r >= 0x0080 && r <= 0x00FF) ||
		(r >= 0x0100 && r <= 0x017F) ||
		(r >= 0x0180 && r <= 0x024F)
}

// isCyrillic checks if rune is in Cyrillic Unicode block
// Cyrillic: U+0400–U+04FF
// Cyrillic Supplement: U+0500–U+052F
func isCyrillic(r rune) bool {
	return (r >= 0x0400 && r <= 0x04FF) ||
		(r >= 0x0500 && r <= 0x052F)
}

// isGreek checks if rune is in Greek Unicode block
// Greek and Coptic: U+0370–U+03FF
// Greek Extended: U+1F00–U+1FFF
func isGreek(r rune) bool {
	return (r >= 0x0370 && r <= 0x03FF) ||
		(r >= 0x1F00 && r <= 0x1FFF)
}

// isArmenian checks if rune is in Armenian Unicode block
// Armenian: U+0530–U+058F
func isArmenian(r rune) bool {
	return r >= 0x0530 && r <= 0x058F
}

// isGeorgian checks if rune is in Georgian Unicode block
// Georgian: U+10A0–U+10FF
func isGeorgian(r rune) bool {
	return r >= 0x10A0 && r <= 0x10FF
}

// isThai checks if rune is in Thai Unicode block
// Thai: U+0E00–U+0E7F
func isThai(r rune) bool {
	return r >= 0x0E00 && r <= 0x0E7F
}

// isCJK checks if rune is in CJK (Chinese, Japanese, Korean) Unicode blocks
// CJK Unified Ideographs: U+4E00–U+9FFF
// CJK Extension A: U+3400–U+4DBF
// Hiragana: U+3040–U+309F
// Katakana: U+30A0–U+30FF
// Hangul: U+AC00–U+D7AF
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x3040 && r <= 0x309F) ||
		(r >= 0x30A0 && r <= 0x30FF) ||
		(r >= 0xAC00 && r <= 0xD7AF)
}
