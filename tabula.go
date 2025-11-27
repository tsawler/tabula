// Package tabula provides a fluent API for extracting text, tables, and other
// content from PDF files.
//
// Basic usage:
//
//	text, warnings, err := tabula.Open("document.pdf").Text()
//	if err != nil {
//	    // handle error
//	}
//	if len(warnings) > 0 {
//	    log.Println("Warnings:", tabula.FormatWarnings(warnings))
//	}
//
// With options:
//
//	text, _, err := tabula.Open("report.pdf").
//	    Pages(1, 2, 3).
//	    ExcludeHeaders().
//	    ExcludeFooters().
//	    Text()
//
// For advanced use cases, the lower-level reader package is also available.
package tabula

import (
	"github.com/tsawler/tabula/reader"
)

// Open opens a PDF file and returns an Extractor for fluent configuration.
// The returned Extractor must be closed when done, either explicitly via Close()
// or implicitly when calling a terminal operation like Text().
//
// Example:
//
//	text, warnings, err := tabula.Open("document.pdf").Text()
func Open(filename string) *Extractor {
	return &Extractor{
		filename: filename,
		options:  defaultOptions(),
	}
}

// FromReader creates an Extractor from an already-opened reader.Reader.
// This is useful when you need more control over the reader lifecycle.
// Note: The caller is responsible for closing the reader.
//
// Example:
//
//	r, err := reader.Open("document.pdf")
//	if err != nil {
//	    // handle error
//	}
//	defer r.Close()
//	text, warnings, err := tabula.FromReader(r).Text()
func FromReader(r *reader.Reader) *Extractor {
	return &Extractor{
		reader:       r,
		ownsReader:   false,
		readerOpened: true,
		options:      defaultOptions(),
	}
}

// Must is a helper that wraps a call to a function returning (T, error)
// and panics if the error is non-nil. It is intended for use in scripts
// or tests where error handling would be cumbersome.
//
// Example:
//
//	count := tabula.Must(tabula.Open("document.pdf").PageCount())
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

// MustText is a helper that wraps a call to Text() or Fragments() and panics
// if the error is non-nil. It discards warnings and returns just the value.
// It is intended for use in scripts or tests where error handling would be cumbersome.
//
// Example:
//
//	text := tabula.MustText(tabula.Open("document.pdf").Text())
func MustText[T any](val T, _ []Warning, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
