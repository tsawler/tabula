package tabula

import "github.com/tsawler/tabula/ocr"

// ExtractOptions holds configuration for text extraction.
type ExtractOptions struct {
	// Page selection (1-indexed in API, stored as-is)
	pages []int

	// Layout filtering
	excludeHeaders bool
	excludeFooters bool

	// Processing options
	byColumn       bool
	preserveLayout bool
	joinParagraphs bool // Join lines within paragraphs with spaces instead of newlines

	// OCR options (scanned-PDF fallback only; effective with -tags ocr)
	ocrLanguage string          // Tesseract language(s), e.g. "eng" or "eng+fra"
	ocrPSM      ocr.PageSegMode // page segmentation mode
	ocrPSMSet   bool            // whether ocrPSM was explicitly set
}

// defaultOptions returns the default extraction options.
func defaultOptions() ExtractOptions {
	return ExtractOptions{
		pages:          nil, // nil means all pages
		excludeHeaders: false,
		excludeFooters: false,
		byColumn:       false,
		preserveLayout: false,
		joinParagraphs: false,
	}
}

// clone creates a deep copy of ExtractOptions.
func (o ExtractOptions) clone() ExtractOptions {
	newOpts := ExtractOptions{
		excludeHeaders: o.excludeHeaders,
		excludeFooters: o.excludeFooters,
		byColumn:       o.byColumn,
		preserveLayout: o.preserveLayout,
		joinParagraphs: o.joinParagraphs,
		ocrLanguage:    o.ocrLanguage,
		ocrPSM:         o.ocrPSM,
		ocrPSMSet:      o.ocrPSMSet,
	}

	// Deep copy pages slice
	if o.pages != nil {
		newOpts.pages = make([]int, len(o.pages))
		copy(newOpts.pages, o.pages)
	}

	return newOpts
}
