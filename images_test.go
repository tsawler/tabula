package tabula

import (
	"testing"
)

// TestImagesCryptozoologyNotFullPage is the key case: every page of
// crytopzoology.pdf carries one discrete illustration that is NOT a full-page
// scan. Images() must find them and their Coverage() must stay well below 1.0 so
// callers do not misclassify them as full-page.
func TestImagesCryptozoologyNotFullPage(t *testing.T) {
	images, err := Open("test-pdfs/crytopzoology.pdf").Images()
	if err != nil {
		t.Fatalf("Images() error: %v", err)
	}
	if len(images) == 0 {
		t.Fatal("expected images, got none")
	}

	for _, img := range images {
		cov := img.Coverage()
		if cov <= 0 {
			t.Errorf("page %d %s: coverage %.3f should be > 0", img.Page, img.Name, cov)
		}
		// The illustrations nearly fill the page but leave margins; they must
		// not read as full-page (measured values cluster around 0.83-0.84).
		if cov >= 0.85 {
			t.Errorf("page %d %s: coverage %.3f too high; should not be classified full-page", img.Page, img.Name, cov)
		}
	}
}

// TestImagesFullPageScans confirms full-page scanned PDFs report a single
// page-filling image per page with Coverage near 1.0.
func TestImagesFullPageScans(t *testing.T) {
	for _, f := range []string{"advil.pdf", "3773_eng.pdf"} {
		images, err := Open(f).Images()
		if err != nil {
			t.Fatalf("%s: Images() error: %v", f, err)
		}
		if len(images) == 0 {
			t.Fatalf("%s: expected images, got none", f)
		}
		for _, img := range images {
			assertPlausibleBBox(t, f, img)
			if cov := img.Coverage(); cov < 0.95 {
				t.Errorf("%s: page %d %s: coverage %.3f, expected near full-page", f, img.Page, img.Name, cov)
			}
		}
	}
}

// TestImagesDiscreteFigures confirms PDFs with small in-page figures report
// plausible, low-coverage bounding boxes inside the page bounds.
func TestImagesDiscreteFigures(t *testing.T) {
	images, err := Open("tylenol.pdf").Images()
	if err != nil {
		t.Fatalf("Images() error: %v", err)
	}
	if len(images) == 0 {
		t.Fatal("expected images, got none")
	}
	for _, img := range images {
		assertPlausibleBBox(t, "tylenol.pdf", img)
		if cov := img.Coverage(); cov >= 0.85 {
			t.Errorf("page %d %s: coverage %.3f, expected a discrete figure", img.Page, img.Name, cov)
		}
		if img.PixelWidth <= 0 || img.PixelHeight <= 0 {
			t.Errorf("page %d %s: missing pixel dimensions %dx%d", img.Page, img.Name, img.PixelWidth, img.PixelHeight)
		}
	}
}

// TestImagesNoneForTextPDF confirms a born-digital text PDF with no raster
// images returns an empty slice (and no error).
func TestImagesNoneForTextPDF(t *testing.T) {
	images, err := Open("406_fre.pdf").Images()
	if err != nil {
		t.Fatalf("Images() error: %v", err)
	}
	if len(images) != 0 {
		t.Fatalf("expected no images for a text PDF, got %d", len(images))
	}
}

// TestImagesDeterministicOrdering confirms Images() returns results in a stable
// order (by page, then draw order) across repeated calls.
func TestImagesDeterministicOrdering(t *testing.T) {
	first, err := Open("test-pdfs/crytopzoology.pdf").Images()
	if err != nil {
		t.Fatalf("Images() error: %v", err)
	}
	second, err := Open("test-pdfs/crytopzoology.pdf").Images()
	if err != nil {
		t.Fatalf("Images() error: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("non-deterministic count: %d vs %d", len(first), len(second))
	}
	prevPage := 0
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("index %d differs between runs: %+v vs %+v", i, first[i], second[i])
		}
		// Pages must be non-decreasing.
		if first[i].Page < prevPage {
			t.Errorf("index %d: page %d out of order (prev %d)", i, first[i].Page, prevPage)
		}
		prevPage = first[i].Page
	}
}

// TestImagesPageSelection confirms the Pages(...) chain method scopes Images().
func TestImagesPageSelection(t *testing.T) {
	images, err := Open("test-pdfs/crytopzoology.pdf").Pages(1).Images()
	if err != nil {
		t.Fatalf("Images() error: %v", err)
	}
	if len(images) == 0 {
		t.Fatal("expected images on page 1")
	}
	for _, img := range images {
		if img.Page != 1 {
			t.Errorf("expected only page 1 images, got page %d", img.Page)
		}
	}
}

// assertPlausibleBBox checks coverage is within [0,1] and the bounding box lies
// within the page bounds (with a small tolerance).
func assertPlausibleBBox(t *testing.T, file string, img PlacedImage) {
	t.Helper()
	if cov := img.Coverage(); cov < 0 || cov > 1 {
		t.Errorf("%s: page %d %s: coverage %.3f out of [0,1]", file, img.Page, img.Name, cov)
	}
	if img.Width <= 0 || img.Height <= 0 {
		t.Errorf("%s: page %d %s: non-positive size %.1fx%.1f", file, img.Page, img.Name, img.Width, img.Height)
	}
	const tol = 1.0
	if img.X < -tol || img.Y < -tol ||
		img.X+img.Width > img.PageWidth+tol || img.Y+img.Height > img.PageHeight+tol {
		t.Errorf("%s: page %d %s: bbox (%.1f,%.1f %.1fx%.1f) outside page %.1fx%.1f",
			file, img.Page, img.Name, img.X, img.Y, img.Width, img.Height, img.PageWidth, img.PageHeight)
	}
}
