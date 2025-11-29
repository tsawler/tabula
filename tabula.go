// Package tabula provides a fluent API for extracting text, tables, and other
// content from PDF, DOCX, ODT, XLSX, PPTX, and HTML files.
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
// DOCX files work the same way:
//
//	text, warnings, err := tabula.Open("document.docx").Text()
//
// With options:
//
//	text, _, err := tabula.Open("report.pdf").
//	    Pages(1, 2, 3).
//	    ExcludeHeaders().
//	    ExcludeFooters().
//	    Text()
//
// HTML content can be parsed from a string (useful for web scraping):
//
//	text, _, err := tabula.FromHTMLString(htmlContent).Text()
//
// For advanced use cases, the lower-level reader package is also available.
package tabula

import (
	"io"
	"strings"

	"github.com/tsawler/tabula/format"
	"github.com/tsawler/tabula/htmldoc"
	"github.com/tsawler/tabula/reader"
)

// Open opens a PDF or DOCX file and returns an Extractor for fluent configuration.
// The file format is automatically detected based on the file extension.
// The returned Extractor must be closed when done, either explicitly via Close()
// or implicitly when calling a terminal operation like Text().
//
// Supported formats:
//   - PDF (.pdf)
//   - DOCX (.docx)
//
// Example:
//
//	text, warnings, err := tabula.Open("document.pdf").Text()
//	text, warnings, err := tabula.Open("document.docx").Text()
func Open(filename string) *Extractor {
	return &Extractor{
		filename: filename,
		format:   format.Detect(filename),
		options:  defaultOptions(),
	}
}

// FromReader creates an Extractor from an already-opened PDF reader.Reader.
// This is useful when you need more control over the PDF reader lifecycle.
// Note: The caller is responsible for closing the reader.
// For DOCX files, use Open() instead which handles format detection automatically.
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
		format:       format.PDF,
		ownsReader:   false,
		readerOpened: true,
		options:      defaultOptions(),
	}
}

// FromHTMLReader creates an Extractor from an io.Reader containing HTML content.
// This is useful when you have HTML content that was fetched from a remote source
// (e.g., via HTTP) and want to extract text or convert it to markdown without
// saving it to a file first.
//
// Example:
//
//	resp, err := http.Get("https://example.com/page")
//	if err != nil {
//	    // handle error
//	}
//	defer resp.Body.Close()
//	text, warnings, err := tabula.FromHTMLReader(resp.Body).Text()
func FromHTMLReader(r io.Reader) *Extractor {
	htmlReader, err := htmldoc.OpenReader(r)
	if err != nil {
		return &Extractor{
			format:  format.HTML,
			options: defaultOptions(),
			err:     err,
		}
	}
	return &Extractor{
		htmlReader:   htmlReader,
		format:       format.HTML,
		ownsReader:   true,
		readerOpened: true,
		options:      defaultOptions(),
	}
}

// FromHTMLString creates an Extractor from a string containing HTML content.
// This is useful when you have HTML content as a string (e.g., fetched from a
// web API or embedded in your application) and want to extract text or convert
// it to markdown.
//
// Example:
//
//	html := `<html><body><h1>Hello</h1><p>World</p></body></html>`
//	text, warnings, err := tabula.FromHTMLString(html).Text()
//
// For web scraping:
//
//	resp, err := http.Get("https://example.com/page")
//	if err != nil {
//	    // handle error
//	}
//	body, _ := io.ReadAll(resp.Body)
//	resp.Body.Close()
//	text, _, _ := tabula.FromHTMLString(string(body)).Text()
func FromHTMLString(html string) *Extractor {
	return FromHTMLReader(strings.NewReader(html))
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
