package rag

import (
	"regexp"
	"strings"

	"github.com/tsawler/tabula/model"
)

// chapterMarkerRe matches strong, high-signal structural heading markers at the
// start of a block of text: CHAPTER / PART / BOOK / SECTION followed by an
// enumerator (digits, roman numerals, or a spelled-out number), and APPENDIX
// followed by a letter or number. Requiring the enumerator keeps precision high
// — it avoids promoting ordinary prose that merely begins with a word like
// "Part of the reason...". A small amount of leading OCR punctuation noise
// (e.g. "\", "|", ".") is tolerated.
//
// This is deliberately conservative: it recovers chapter-level structure that
// genuinely exists in the text (common in scanned/OCR books that carry no
// heading markup) without guessing at all-caps or title-case lines, which on
// noisy OCR are far more often page headers, captions, or garbage than real
// headings.
var chapterMarkerRe = regexp.MustCompile(`(?i)^[\s\\._|*]{0,4}(?:(?:CHAPTER|PART|BOOK|SECTION)\s+(?:\d{1,3}|[IVXLCDM]{1,7}|ONE|TWO|THREE|FOUR|FIVE|SIX|SEVEN|EIGHT|NINE|TEN|ELEVEN|TWELVE|THIRTEEN|FOURTEEN|FIFTEEN|SIXTEEN|SEVENTEEN|EIGHTEEN|NINETEEN|TWENTY|FIRST|SECOND|THIRD|FOURTH|FIFTH)|APPENDIX\s+(?:[A-Z]|\d{1,3}|[IVXLCDM]{1,7}))(?:\b|[:.\s])`)

// headingTitleMaxChars / headingTitleMaxWords bound how much text following a
// detected marker is treated as the heading title. Scanned books often glue the
// chapter title onto the body of the first paragraph with no clean separator,
// so we cap the title rather than risk swallowing the whole page.
const (
	headingTitleMaxChars = 90
	headingTitleMaxWords = 12
)

// detectChapterHeading reports whether text begins with a conservative chapter
// marker. When it does, it returns the heading portion (marker plus a bounded
// title) and the remaining body text, split so that no content is duplicated.
// The body may be empty when the whole block is a standalone heading.
func detectChapterHeading(text string) (heading, body string, ok bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", "", false
	}
	if !chapterMarkerRe.MatchString(trimmed) {
		return "", "", false
	}

	// Drop any leading OCR punctuation noise so it doesn't end up in the heading.
	trimmed = strings.TrimLeft(trimmed, " \t\\._|*")

	end := headingEnd(trimmed)
	heading = strings.TrimSpace(trimmed[:end])
	body = strings.TrimSpace(trimmed[end:])
	if heading == "" {
		return "", "", false
	}
	return heading, body, true
}

// headingEnd returns the byte index at which the heading title ends. It stops at
// the first sentence-ending punctuation or newline, otherwise at a word or
// character cap (backing up to a word boundary so it never cuts mid-word).
func headingEnd(s string) int {
	words := 0
	inWord := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c == '\n' || c == '\r' {
			return i
		}
		if isSentenceEndChar(c) {
			return i + 1 // include the punctuation
		}

		if c == ' ' || c == '\t' {
			if inWord {
				words++
				inWord = false
				if words >= headingTitleMaxWords {
					return i
				}
			}
		} else {
			inWord = true
		}

		if i+1 >= headingTitleMaxChars {
			// Reached the character cap; back up to the last word boundary.
			for j := i; j > 0; j-- {
				if s[j] == ' ' {
					return j
				}
			}
			return i
		}
	}

	return len(s)
}

// chapterHeadingLevel assigns a heading level so PART/BOOK outrank CHAPTER and
// the rest, giving a sensible two-level outline (e.g. PART ONE > CHAPTER 3).
func chapterHeadingLevel(heading string) int {
	u := strings.ToUpper(strings.TrimLeft(heading, " \\._|*"))
	if strings.HasPrefix(u, "PART") || strings.HasPrefix(u, "BOOK") {
		return 1
	}
	return 2
}

// documentHasHeadings reports whether the document already carries explicit
// heading structure, either as Heading elements or in page layout. Heuristic
// detection is skipped when it does, so structured documents are never
// second-guessed.
func documentHasHeadings(doc *model.Document) bool {
	for _, page := range doc.Pages {
		if page == nil {
			continue
		}
		for _, elem := range page.Elements {
			if _, ok := elem.(*model.Heading); ok {
				return true
			}
		}
		if page.Layout != nil && len(page.Layout.Headings) > 0 {
			return true
		}
	}
	return false
}
