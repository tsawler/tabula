//go:build ocr

package tabula

import (
	"strings"
	"testing"
)

// TestParallelOCRPreservesPageOrder verifies that multi-page OCR (run
// concurrently across pages) assembles results in correct page order. The
// fixture is a 3-page scanned PDF whose pages read "PAGE ONE/TWO/THREE".
func TestParallelOCRPreservesPageOrder(t *testing.T) {
	text, _, err := Open("testdata/multipage_scan.pdf").Text()
	if err != nil {
		t.Fatalf("Text: %v", err)
	}
	one := strings.Index(text, "ONE")
	two := strings.Index(text, "TWO")
	three := strings.Index(text, "THREE")
	if one < 0 || two < 0 || three < 0 {
		t.Fatalf("missing OCR'd page text in %q", text)
	}
	if !(one < two && two < three) {
		t.Errorf("pages out of order: ONE@%d TWO@%d THREE@%d\n%q", one, two, three, text)
	}

	// Chunks() shares the same parallel OCR path; check order there too.
	col, _, err := Open("testdata/multipage_scan.pdf").Chunks()
	if err != nil {
		t.Fatalf("Chunks: %v", err)
	}
	var joined strings.Builder
	for _, c := range col.Chunks {
		joined.WriteString(c.Text)
		joined.WriteString("\n")
	}
	cs := joined.String()
	if a, b, c := strings.Index(cs, "ONE"), strings.Index(cs, "TWO"), strings.Index(cs, "THREE"); !(a >= 0 && a < b && b < c) {
		t.Errorf("chunk order wrong: ONE@%d TWO@%d THREE@%d", a, b, c)
	}
}
