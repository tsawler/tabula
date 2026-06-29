package tabula

import (
	"fmt"
	"math"

	"github.com/tsawler/tabula/format"
)

// PlacedImage is one raster image as actually drawn on a page, with its
// placement in PDF user-space points (origin bottom-left).
type PlacedImage struct {
	Page                  int     // 1-based page number
	Name                  string  // XObject name (e.g. "Im1")
	PixelWidth            int     // intrinsic image width in pixels
	PixelHeight           int     // intrinsic image height in pixels
	ColorSpace            string  // base color-space name (DeviceRGB, DeviceGray, ...)
	X, Y, Width, Height   float64 // drawn bounding box on the page, in points
	PageWidth, PageHeight float64 // page dimensions in points
}

// Coverage is the fraction of the page area the image's bounding box covers (0..1).
// A value near 1 indicates a full-page image (e.g. a scanned page), while a small
// value indicates a discrete in-page figure. Returns 0 when page dimensions are
// unknown; the result is always clamped to [0, 1].
func (p PlacedImage) Coverage() float64 {
	pageArea := p.PageWidth * p.PageHeight
	if pageArea <= 0 {
		return 0
	}
	area := math.Abs(p.Width * p.Height)
	c := area / pageArea
	if c < 0 {
		return 0
	}
	if c > 1 {
		return 1
	}
	return c
}

// Images reports every raster image XObject drawn on the configured pages,
// together with the bounding box it occupies on the page (in points). This lets
// callers compute how much of a page each image covers (via Coverage) to tell a
// full-page scan from a discrete in-page figure.
//
// Placement is recovered by walking each page's content stream and tracking the
// current transformation matrix; images nested inside Form XObjects are included
// with the form's matrix composed in. Inline images (BI/ID/EI) are not reported.
//
// Results are ordered by page, then by draw order within a page. For non-PDF
// formats this returns (nil, nil). This is a terminal operation that closes the
// underlying reader.
//
// Example:
//
//	images, err := tabula.Open("document.pdf").Images()
//	for _, img := range images {
//	    fmt.Printf("page %d %s covers %.0f%%\n", img.Page, img.Name, img.Coverage()*100)
//	}
func (e *Extractor) Images() ([]PlacedImage, error) {
	if e.err != nil {
		return nil, e.err
	}

	if err := e.ensureReader(); err != nil {
		return nil, err
	}
	defer e.Close()

	// Image placement is a PDF-only concept here; other formats return no images.
	if e.format != format.PDF {
		return nil, nil
	}

	pageIndices, err := e.resolvePages()
	if err != nil {
		return nil, err
	}

	var result []PlacedImage
	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		pageWidth, _ := page.Width()
		pageHeight, _ := page.Height()

		placed, err := e.reader.ExtractPlacedImages(page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		for _, p := range placed {
			result = append(result, PlacedImage{
				Page:        pageNum + 1,
				Name:        p.Name,
				PixelWidth:  p.PixelWidth,
				PixelHeight: p.PixelHeight,
				ColorSpace:  p.ColorSpace,
				X:           p.X,
				Y:           p.Y,
				Width:       p.Width,
				Height:      p.Height,
				PageWidth:   pageWidth,
				PageHeight:  pageHeight,
			})
		}
	}

	return result, nil
}
