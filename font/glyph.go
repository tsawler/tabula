package font

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

//go:generate go run ./internal/generateagl

// TeX aliases retained from the extraction regressions. These are separate
// from the authoritative Adobe table; scalable delimiter names identify shapes,
// not distinct Unicode characters. See data/agl/README.md.
var texGlyphAliases = map[string]string{
	"epsilon1": "ϵ", "follows": "≻", "lscript": "ℓ",
	"parenleftbig": "(", "parenrightbig": ")",
	"bracketleftbig": "[", "bracketrightbig": "]",
	"parenleftbigg": "(", "parenrightbigg": ")", "braceleftBigg": "{",
	"summationdisplay": "∑", "summationtext": "∑", "vextendsingle": "|",
}

// glyphToUnicode follows the Adobe Glyph List specification. The boolean
// distinguishes an intentional empty mapping (.notdef) from an unknown name,
// for which custom encodings retain their historical base-encoding fallback.
func glyphToUnicode(name string, zapf bool) (string, bool) {
	if dot := strings.IndexByte(name, '.'); dot >= 0 {
		name = name[:dot]
	}
	if name == "" {
		return "", true
	}
	var result strings.Builder
	known := false
	for _, component := range strings.Split(name, "_") {
		value, ok := glyphComponent(component, zapf)
		if ok {
			known = true
			result.WriteString(value)
		}
	}
	return result.String(), known
}

func glyphComponent(name string, zapf bool) (string, bool) {
	if zapf {
		if value, ok := zapfGlyphNameToUnicode[name]; ok {
			return value, true
		}
	}
	if value, ok := glyphNameToUnicode[name]; ok {
		return value, true
	}
	if value, ok := texGlyphAliases[name]; ok {
		return value, true
	}
	if strings.HasPrefix(name, "uni") {
		digits := name[3:]
		if len(digits) == 0 || len(digits)%4 != 0 {
			return "", false
		}
		var result strings.Builder
		for i := 0; i < len(digits); i += 4 {
			r, ok := glyphHexRune(digits[i : i+4])
			if !ok {
				return "", false
			}
			result.WriteRune(r)
		}
		return result.String(), true
	}
	if strings.HasPrefix(name, "u") && len(name) >= 5 && len(name) <= 7 {
		if r, ok := glyphHexRune(name[1:]); ok {
			return string(r), true
		}
	}
	return "", false
}

func glyphHexRune(digits string) (rune, bool) {
	for _, c := range digits {
		if !(c >= '0' && c <= '9' || c >= 'A' && c <= 'F') {
			return 0, false
		}
	}
	n, err := strconv.ParseUint(digits, 16, 32)
	if err != nil || n > utf8.MaxRune || !utf8.ValidRune(rune(n)) {
		return 0, false
	}
	return rune(n), true
}
