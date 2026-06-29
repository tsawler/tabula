package tabula

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/tsawler/tabula/format"
	"github.com/tsawler/tabula/ocr"
)

// ocrRenderDPI is the rasterization resolution for a full-page OCR render. 300
// DPI is the standard floor for reliable Tesseract accuracy on body text.
const ocrRenderDPI = 300

// ocrRenderTimeout bounds a single page render so a pathological PDF can't hang
// ingestion on the pdftoppm subprocess.
const ocrRenderTimeout = 60 * time.Second

var (
	pdftoppmOnce sync.Once
	pdftoppmPath string

	ocrAvailOnce sync.Once
	ocrAvail     bool
)

// pdftoppmBin returns the path to pdftoppm (poppler-utils) if it's installed, or
// "" otherwise. Looked up once and cached.
func pdftoppmBin() string {
	pdftoppmOnce.Do(func() {
		if p, err := exec.LookPath("pdftoppm"); err == nil {
			pdftoppmPath = p
		}
	})
	return pdftoppmPath
}

// ocrCompiledIn reports whether OCR support was built in (-tags ocr) with a
// usable Tesseract engine. Cached so we don't rasterize pages we couldn't OCR.
func ocrCompiledIn() bool {
	ocrAvailOnce.Do(func() {
		c, err := ocr.New()
		if err != nil {
			return
		}
		c.Close()
		ocrAvail = true
	})
	return ocrAvail
}

// renderPageForOCR rasterizes one page (1-based) of the source PDF to a single
// full-page image, via pdftoppm, ready for OCR. Unlike extracting a page's
// embedded images, a full-page render captures vector-outlined text and vector
// artwork — in many illustrated or design-heavy PDFs the body text is drawn as
// path outlines, not fonts, so it's invisible to both native text extraction and
// image extraction, and only a rasterized render exposes it to OCR.
//
// Returns nil (so the caller falls back to embedded-image OCR) when the source
// isn't a file-backed PDF, OCR isn't compiled in, pdftoppm isn't installed, or
// the render fails.
func (e *Extractor) renderPageForOCR(pageNum int) []preparedImage {
	if e.format != format.PDF || e.filename == "" || pageNum < 1 {
		return nil
	}
	if !ocrCompiledIn() {
		return nil
	}
	bin := pdftoppmBin()
	if bin == "" {
		return nil
	}

	tmp, err := os.MkdirTemp("", "tabula-ocr-")
	if err != nil {
		return nil
	}
	defer os.RemoveAll(tmp)
	prefix := filepath.Join(tmp, "page")

	ctx, cancel := context.WithTimeout(context.Background(), ocrRenderTimeout)
	defer cancel()
	// -singlefile writes exactly <prefix>.png (no page-number suffix); -r sets the
	// DPI; pdftoppm renders the page upright (honoring /Rotate) and flattens
	// vector text, vector art, and images into one bitmap.
	cmd := exec.CommandContext(ctx, bin, "-png", "-r", strconv.Itoa(ocrRenderDPI),
		"-f", strconv.Itoa(pageNum), "-l", strconv.Itoa(pageNum),
		"-singlefile", e.filename, prefix)
	if err := cmd.Run(); err != nil {
		return nil
	}

	png, err := os.ReadFile(prefix + ".png")
	if err != nil || len(png) == 0 {
		return nil
	}
	return []preparedImage{{png: png, dpi: ocrRenderDPI}}
}
