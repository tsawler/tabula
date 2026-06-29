//go:build ocr

package tabula

import (
	"os/exec"
	"strings"
	"testing"
)

// TestOCRFullPageRenderRecoversVectorText verifies the full-page render path:
// crytopzoology.pdf is an illustrated book whose pages are vector artwork plus
// vector-outlined body text, with the artwork as the only embedded raster. The
// old embedded-image OCR path handed Tesseract just the illustration and missed
// the text; rasterizing the whole page via pdftoppm exposes the outlined text to
// OCR. Page 5 is the Chupacabra entry. Requires pdftoppm (poppler-utils).
func TestOCRFullPageRenderRecoversVectorText(t *testing.T) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm not installed; the full-page render path is unavailable")
	}

	text, _, err := Open("test-pdfs/crytopzoology.pdf").Pages(5).Text()
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	for _, want := range []string{"Chupacabra", "Puerto Rico", "goatsucker", "livestock"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in OCR output — vector text not recovered.\nGot: %q", want, text)
		}
	}
}
