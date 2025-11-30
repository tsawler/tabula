package text

import (
	"unicode"
)

// Direction represents the writing direction of text.
// It is used to detect and handle bidirectional text (bidi) in documents.
type Direction int

const (
	// LTR (Left-to-Right) for Latin, Cyrillic, etc.
	LTR Direction = iota
	// RTL (Right-to-Left) for Arabic, Hebrew, etc.
	RTL
	// Neutral for numbers, punctuation, etc.
	Neutral
)

// String returns a string representation of the direction ("LTR", "RTL", or "Neutral").
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

// DetectDirection analyzes a string and returns its dominant text direction
// based on Unicode character properties. It counts strong directional characters
// and returns the direction with the higher count, or Neutral if no strong
// directional characters are present.
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

// GetCharDirection returns the inherent direction of a single Unicode character.
// Digits, punctuation, whitespace, and symbols are Neutral; RTL scripts (Arabic,
// Hebrew, Syriac, Thaana, N'Ko) return RTL; all other scripts return LTR.
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

// isArabic reports whether r is in an Arabic Unicode block.
// This includes:
//   - Arabic: U+0600–U+06FF
//   - Arabic Supplement: U+0750–U+077F
//   - Arabic Extended-A: U+08A0–U+08FF
//   - Arabic Presentation Forms-A: U+FB50–U+FDFF
//   - Arabic Presentation Forms-B: U+FE70–U+FEFF
func isArabic(r rune) bool {
	return (r >= 0x0600 && r <= 0x06FF) ||
		(r >= 0x0750 && r <= 0x077F) ||
		(r >= 0x08A0 && r <= 0x08FF) ||
		(r >= 0xFB50 && r <= 0xFDFF) ||
		(r >= 0xFE70 && r <= 0xFEFF)
}

// isHebrew reports whether r is in a Hebrew Unicode block.
// This includes:
//   - Hebrew: U+0590–U+05FF
//   - Hebrew Presentation Forms: U+FB1D–U+FB4F
func isHebrew(r rune) bool {
	return (r >= 0x0590 && r <= 0x05FF) ||
		(r >= 0xFB1D && r <= 0xFB4F)
}

// isSyriac reports whether r is in the Syriac Unicode block (U+0700–U+074F).
func isSyriac(r rune) bool {
	return r >= 0x0700 && r <= 0x074F
}

// isThaana reports whether r is in the Thaana Unicode block (U+0780–U+07BF).
// Thaana is the script used to write Maldivian (Dhivehi).
func isThaana(r rune) bool {
	return r >= 0x0780 && r <= 0x07BF
}

// isNKo reports whether r is in the N'Ko Unicode block (U+07C0–U+07FF).
// N'Ko is a script used for Manding languages in West Africa.
func isNKo(r rune) bool {
	return r >= 0x07C0 && r <= 0x07FF
}

// isLatin reports whether r is in a Latin Unicode block.
// This includes:
//   - Basic Latin: U+0000–U+007F
//   - Latin-1 Supplement: U+0080–U+00FF
//   - Latin Extended-A: U+0100–U+017F
//   - Latin Extended-B: U+0180–U+024F
func isLatin(r rune) bool {
	return (r >= 0x0000 && r <= 0x007F) ||
		(r >= 0x0080 && r <= 0x00FF) ||
		(r >= 0x0100 && r <= 0x017F) ||
		(r >= 0x0180 && r <= 0x024F)
}

// isCyrillic reports whether r is in a Cyrillic Unicode block.
// This includes:
//   - Cyrillic: U+0400–U+04FF
//   - Cyrillic Supplement: U+0500–U+052F
func isCyrillic(r rune) bool {
	return (r >= 0x0400 && r <= 0x04FF) ||
		(r >= 0x0500 && r <= 0x052F)
}

// isGreek reports whether r is in a Greek Unicode block.
// This includes:
//   - Greek and Coptic: U+0370–U+03FF
//   - Greek Extended: U+1F00–U+1FFF
func isGreek(r rune) bool {
	return (r >= 0x0370 && r <= 0x03FF) ||
		(r >= 0x1F00 && r <= 0x1FFF)
}

// isArmenian reports whether r is in the Armenian Unicode block (U+0530–U+058F).
func isArmenian(r rune) bool {
	return r >= 0x0530 && r <= 0x058F
}

// isGeorgian reports whether r is in the Georgian Unicode block (U+10A0–U+10FF).
func isGeorgian(r rune) bool {
	return r >= 0x10A0 && r <= 0x10FF
}

// isThai reports whether r is in the Thai Unicode block (U+0E00–U+0E7F).
func isThai(r rune) bool {
	return r >= 0x0E00 && r <= 0x0E7F
}

// isCJK reports whether r is in a CJK (Chinese, Japanese, Korean) Unicode block.
// This includes:
//   - CJK Unified Ideographs: U+4E00–U+9FFF
//   - CJK Extension A: U+3400–U+4DBF
//   - Hiragana: U+3040–U+309F
//   - Katakana: U+30A0–U+30FF
//   - Hangul: U+AC00–U+D7AF
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x3040 && r <= 0x309F) ||
		(r >= 0x30A0 && r <= 0x30FF) ||
		(r >= 0xAC00 && r <= 0xD7AF)
}
