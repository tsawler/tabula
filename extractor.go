package tabula

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tsawler/tabula/core"
	"github.com/tsawler/tabula/docx"
	"github.com/tsawler/tabula/format"
	"github.com/tsawler/tabula/layout"
	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/odt"
	"github.com/tsawler/tabula/pages"
	"github.com/tsawler/tabula/rag"
	"github.com/tsawler/tabula/reader"
	"github.com/tsawler/tabula/text"
)

// extractedPage holds the data extracted from a single page.
type extractedPage struct {
	index     int
	fragments []text.TextFragment
	page      *pages.Page
}

// Extractor provides a fluent interface for extracting content from PDFs, DOCX, and ODT files.
// Each configuration method returns a new Extractor instance, making it
// safe for concurrent use and allowing method chaining.
type Extractor struct {
	// Source
	filename string
	format   format.Format

	// Readers (only one will be used based on format)
	reader     *reader.Reader // PDF reader
	docxReader *docx.Reader   // DOCX reader
	odtReader  *odt.Reader    // ODT reader

	// Lifecycle
	ownsReader   bool // true if we opened the reader and should close it
	readerOpened bool // true if reader has been opened

	// Configuration
	options ExtractOptions

	// Accumulated error (fail-fast)
	err error

	// Warnings accumulated during processing
	warnings []Warning
}

// clone creates a shallow copy of the Extractor with a deep copy of options.
// This ensures immutability - each chain method returns a new instance.
func (e *Extractor) clone() *Extractor {
	newExt := &Extractor{
		filename:     e.filename,
		format:       e.format,
		reader:       e.reader,
		docxReader:   e.docxReader,
		odtReader:    e.odtReader,
		ownsReader:   e.ownsReader,
		readerOpened: e.readerOpened,
		options:      e.options.clone(),
		err:          e.err,
		warnings:     append([]Warning(nil), e.warnings...),
	}
	return newExt
}

// ensureReader opens the reader if not already open.
func (e *Extractor) ensureReader() error {
	if e.readerOpened {
		return nil
	}
	if e.filename == "" {
		return fmt.Errorf("no filename specified")
	}

	switch e.format {
	case format.DOCX:
		dr, err := docx.Open(e.filename)
		if err != nil {
			return fmt.Errorf("failed to open DOCX: %w", err)
		}
		e.docxReader = dr
		e.ownsReader = true
		e.readerOpened = true
		return nil

	case format.ODT:
		or, err := odt.Open(e.filename)
		if err != nil {
			return fmt.Errorf("failed to open ODT: %w", err)
		}
		e.odtReader = or
		e.ownsReader = true
		e.readerOpened = true
		return nil

	case format.PDF:
		r, err := reader.Open(e.filename)
		if err != nil {
			return fmt.Errorf("failed to open PDF: %w", err)
		}
		e.reader = r
		e.ownsReader = true
		e.readerOpened = true
		return nil

	default:
		return fmt.Errorf("unsupported file format: %s", e.format)
	}
}

// Close releases resources associated with the Extractor.
// It is safe to call Close multiple times.
func (e *Extractor) Close() error {
	if e.ownsReader {
		if e.reader != nil {
			err := e.reader.Close()
			e.reader = nil
			e.ownsReader = false
			return err
		}
		if e.docxReader != nil {
			err := e.docxReader.Close()
			e.docxReader = nil
			e.ownsReader = false
			return err
		}
		if e.odtReader != nil {
			err := e.odtReader.Close()
			e.odtReader = nil
			e.ownsReader = false
			return err
		}
	}
	return nil
}

// ============================================================================
// Configuration Methods (return new Extractor instance)
// ============================================================================

// Pages specifies which pages to extract from (1-indexed).
// Multiple calls are cumulative.
//
// Example:
//
//	text, _, err := tabula.Open("doc.pdf").Pages(1, 3, 5).Text()
func (e *Extractor) Pages(pages ...int) *Extractor {
	newExt := e.clone()
	newExt.options.pages = append(newExt.options.pages, pages...)
	return newExt
}

// PageRange specifies a range of pages to extract (1-indexed, inclusive).
//
// Example:
//
//	text, _, err := tabula.Open("doc.pdf").PageRange(5, 10).Text()
func (e *Extractor) PageRange(start, end int) *Extractor {
	newExt := e.clone()
	for i := start; i <= end; i++ {
		newExt.options.pages = append(newExt.options.pages, i)
	}
	return newExt
}

// ExcludeHeaders configures the extractor to exclude detected headers.
//
// Example:
//
//	text, _, err := tabula.Open("doc.pdf").ExcludeHeaders().Text()
func (e *Extractor) ExcludeHeaders() *Extractor {
	newExt := e.clone()
	newExt.options.excludeHeaders = true
	return newExt
}

// ExcludeFooters configures the extractor to exclude detected footers.
//
// Example:
//
//	text, _, err := tabula.Open("doc.pdf").ExcludeFooters().Text()
func (e *Extractor) ExcludeFooters() *Extractor {
	newExt := e.clone()
	newExt.options.excludeFooters = true
	return newExt
}

// ExcludeHeadersAndFooters configures the extractor to exclude both
// detected headers and footers. This is a convenience method equivalent
// to calling ExcludeHeaders().ExcludeFooters().
//
// Example:
//
//	text, _, err := tabula.Open("doc.pdf").ExcludeHeadersAndFooters().Text()
func (e *Extractor) ExcludeHeadersAndFooters() *Extractor {
	newExt := e.clone()
	newExt.options.excludeHeaders = true
	newExt.options.excludeFooters = true
	return newExt
}

// JoinParagraphs configures the extractor to join lines within paragraphs
// using spaces instead of newlines. This produces cleaner text output where
// paragraph breaks are preserved but soft line breaks within paragraphs are removed.
//
// Example:
//
//	text, _, err := tabula.Open("doc.pdf").JoinParagraphs().Text()
//	text, _, err := tabula.Open("doc.pdf").ExcludeHeadersAndFooters().JoinParagraphs().Text()
func (e *Extractor) JoinParagraphs() *Extractor {
	newExt := e.clone()
	newExt.options.joinParagraphs = true
	return newExt
}

// ByColumn configures the extractor to process text column by column
// in reading order, rather than line by line across the full page width.
// This is useful for multi-column documents like newspapers or academic papers.
//
// Example:
//
//	text, _, err := tabula.Open("newspaper.pdf").ByColumn().Text()
func (e *Extractor) ByColumn() *Extractor {
	newExt := e.clone()
	newExt.options.byColumn = true
	return newExt
}

// PreserveLayout maintains spatial positioning by inserting spaces
// to approximate the visual layout of the original document.
//
// Example:
//
//	text, _, err := tabula.Open("form.pdf").PreserveLayout().Text()
func (e *Extractor) PreserveLayout() *Extractor {
	newExt := e.clone()
	newExt.options.preserveLayout = true
	return newExt
}

// IsCharacterLevel checks if the first page of the PDF uses character-level
// text fragments (one character per fragment). This requires special handling
// for proper text extraction.
// Note: This reads page 1 to make the determination. The reader remains open.
//
// Example:
//
//	ext := tabula.Open("document.pdf")
//	defer ext.Close()
//	isCharLevel, err := ext.IsCharacterLevel()
func (e *Extractor) IsCharacterLevel() (bool, error) {
	if e.err != nil {
		return false, e.err
	}

	if err := e.ensureReader(); err != nil {
		return false, err
	}

	page, err := e.reader.GetPage(0)
	if err != nil {
		return false, fmt.Errorf("reading page 1: %w", err)
	}

	fragments, err := e.reader.ExtractTextFragments(page)
	if err != nil {
		return false, fmt.Errorf("extracting fragments: %w", err)
	}

	return isCharacterLevel(fragments), nil
}

// IsMultiColumn checks if the first page of the PDF appears to have a
// multi-column layout.
// Note: This reads page 1 to make the determination. The reader remains open.
//
// Example:
//
//	ext := tabula.Open("newspaper.pdf")
//	defer ext.Close()
//	multiCol, err := ext.IsMultiColumn()
func (e *Extractor) IsMultiColumn() (bool, error) {
	if e.err != nil {
		return false, e.err
	}

	if err := e.ensureReader(); err != nil {
		return false, err
	}

	page, err := e.reader.GetPage(0)
	if err != nil {
		return false, fmt.Errorf("reading page 1: %w", err)
	}

	fragments, err := e.reader.ExtractTextFragments(page)
	if err != nil {
		return false, fmt.Errorf("extracting fragments: %w", err)
	}

	width, _ := page.Width()
	height, _ := page.Height()
	return detectMultiColumn(fragments, width, height), nil
}

// ============================================================================
// Terminal Operations (execute extraction and return results)
// ============================================================================

// Text extracts and returns the text content from the configured pages.
// This is a terminal operation that closes the underlying reader.
//
// Returns the extracted text, any warnings encountered during processing,
// and an error if extraction failed. Warnings indicate non-fatal issues
// (e.g., messy PDF detected) where extraction succeeded but results may
// be imperfect.
//
// Example:
//
//	text, warnings, err := tabula.Open("document.pdf").Text()
//	text, warnings, err := tabula.Open("document.docx").Text()
//	if len(warnings) > 0 {
//	    log.Println("Warnings:", tabula.FormatWarnings(warnings))
//	}
func (e *Extractor) Text() (string, []Warning, error) {
	if e.err != nil {
		return "", nil, e.err
	}

	if err := e.ensureReader(); err != nil {
		return "", nil, err
	}
	defer e.Close()

	// Handle DOCX files
	if e.format == format.DOCX {
		text, err := e.docxReader.Text()
		if err != nil {
			return "", e.warnings, err
		}
		return text, e.warnings, nil
	}

	// Handle ODT files
	if e.format == format.ODT {
		text, err := e.odtReader.Text()
		if err != nil {
			return "", e.warnings, err
		}
		return text, e.warnings, nil
	}

	// PDF processing
	pageIndices, err := e.resolvePages()
	if err != nil {
		return "", nil, err
	}

	// Collect requested page data
	requestedPages := make([]extractedPage, 0, len(pageIndices))

	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return "", nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return "", nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		requestedPages = append(requestedPages, extractedPage{
			index:     pageNum,
			fragments: fragments,
			page:      page,
		})
	}

	// Detect headers/footers if needed (requires ALL pages for pattern detection)
	var headerFooterResult *layout.HeaderFooterResult
	if e.options.excludeHeaders || e.options.excludeFooters {
		allPages, err := e.collectAllPages()
		if err == nil && len(allPages) > 0 {
			headerFooterResult = e.detectHeaderFooter(allPages)
		}
	}

	// Process each requested page
	var result strings.Builder
	for i, pd := range requestedPages {
		// Check for messy PDF traits on the first page processed
		if i == 0 {
			e.checkMessyPDF(pd.fragments)
		}

		fragments := pd.fragments

		// Filter headers/footers
		if headerFooterResult != nil {
			height, _ := pd.page.Height()
			fragments = headerFooterResult.FilterFragments(pd.index, fragments, height)
		}

		// Extract text
		var pageText string
		if e.options.joinParagraphs {
			pageText = e.extractWithParagraphs(fragments, pd.page)
		} else if e.options.byColumn {
			pageText = e.extractByColumn(fragments, pd.page)
		} else {
			// Auto-detect: use reading order if multi-column or character-level
			width, _ := pd.page.Width()
			height, _ := pd.page.Height()
			if isCharacterLevel(fragments) || detectMultiColumn(fragments, width, height) {
				pageText = e.extractByColumn(fragments, pd.page)
			} else {
				pageText = e.assembleText(fragments)
			}
		}

		if i > 0 && result.Len() > 0 && len(pageText) > 0 {
			result.WriteString("\n\n")
		}
		result.WriteString(pageText)
	}

	return result.String(), e.warnings, nil
}

// extractWithParagraphs uses paragraph detection to join lines within paragraphs
// with spaces instead of newlines, producing cleaner text output.
// It respects multi-column layouts by using reading order detection.
func (e *Extractor) extractWithParagraphs(fragments []text.TextFragment, page *pages.Page) string {
	if len(fragments) == 0 {
		return ""
	}

	width, _ := page.Width()
	height, _ := page.Height()

	// Use reading order detector to handle multi-column layouts
	roDetector := layout.NewReadingOrderDetector()
	roResult := roDetector.Detect(fragments, width, height)

	// Get lines - either from reading order result or directly from line detector
	var lines []layout.Line
	if roResult != nil && len(roResult.Lines) > 0 {
		lines = roResult.Lines
	} else {
		// Fallback to direct line detection for simple documents
		// (e.g., when column detection filters out all content)
		lineDetector := layout.NewLineDetector()
		lineLayout := lineDetector.Detect(fragments, width, height)
		lines = lineLayout.Lines
	}

	if len(lines) == 0 {
		// Last resort: use simple text assembly
		return e.assembleText(fragments)
	}

	// Detect paragraphs from lines
	paraDetector := layout.NewParagraphDetector()
	paraLayout := paraDetector.Detect(lines, width, height)

	// Build text by joining lines within paragraphs with spaces
	var result strings.Builder
	for i, para := range paraLayout.Paragraphs {
		if i > 0 {
			result.WriteString("\n\n")
		}

		// Join lines within the paragraph with spaces
		for j, line := range para.Lines {
			if j > 0 {
				result.WriteString(" ")
			}
			result.WriteString(strings.TrimSpace(line.Text))
		}
	}

	return result.String()
}

// ToMarkdown extracts content and returns it as a markdown-formatted string.
// This preserves document structure including headings, paragraphs, and lists.
// This is a terminal operation that closes the underlying reader.
//
// Returns the markdown text, any warnings encountered during processing,
// and an error if extraction failed.
//
// Example:
//
//	md, warnings, err := tabula.Open("document.pdf").
//	    ExcludeHeadersAndFooters().
//	    ToMarkdown()
func (e *Extractor) ToMarkdown() (string, []Warning, error) {
	return e.ToMarkdownWithOptions(rag.DefaultMarkdownOptions())
}

// ToMarkdownWithOptions extracts content and returns it as markdown with custom options.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	opts := rag.MarkdownOptions{
//	    IncludeTableOfContents: true,
//	    IncludePageNumbers:     true,
//	}
//	md, warnings, err := tabula.Open("document.pdf").ToMarkdownWithOptions(opts)
func (e *Extractor) ToMarkdownWithOptions(opts rag.MarkdownOptions) (string, []Warning, error) {
	// For DOCX files, use the native markdown method which preserves document order
	if e.format == format.DOCX {
		if err := e.ensureReader(); err != nil {
			return "", nil, err
		}
		md, err := e.docxReader.Markdown()
		if err != nil {
			return "", nil, err
		}
		return md, nil, nil
	}

	// For ODT files, use the native markdown method which preserves document order
	if e.format == format.ODT {
		if err := e.ensureReader(); err != nil {
			return "", nil, err
		}
		md, err := e.odtReader.Markdown()
		if err != nil {
			return "", nil, err
		}
		return md, nil, nil
	}

	// For PDF files, use the RAG chunking pipeline
	chunks, warnings, err := e.Chunks()
	if err != nil {
		return "", warnings, err
	}

	return chunks.ToMarkdownWithOptions(opts), warnings, nil
}

// Fragments extracts and returns text fragments with position information.
// This is a terminal operation that closes the underlying reader.
//
// Returns the fragments, any warnings encountered during processing,
// and an error if extraction failed.
//
// Example:
//
//	fragments, warnings, err := tabula.Open("document.pdf").Pages(1).Fragments()
func (e *Extractor) Fragments() ([]text.TextFragment, []Warning, error) {
	if e.err != nil {
		return nil, nil, e.err
	}

	if err := e.ensureReader(); err != nil {
		return nil, nil, err
	}
	defer e.Close()

	pageIndices, err := e.resolvePages()
	if err != nil {
		return nil, nil, err
	}

	var allFragments []text.TextFragment
	for i, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return nil, nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return nil, nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		// Check for messy PDF traits on the first page processed
		if i == 0 {
			e.checkMessyPDF(fragments)
		}

		allFragments = append(allFragments, fragments...)
	}

	return allFragments, e.warnings, nil
}

// PageCount returns the total number of pages in the document.
// For DOCX files, this returns 1 (the entire document is treated as a single page).
// Note: This does NOT close the reader, allowing further operations.
//
// Example:
//
//	ext := tabula.Open("document.pdf")
//	defer ext.Close()
//	count, err := ext.PageCount()
func (e *Extractor) PageCount() (int, error) {
	if e.err != nil {
		return 0, e.err
	}

	if err := e.ensureReader(); err != nil {
		return 0, err
	}

	if e.format == format.DOCX {
		return e.docxReader.PageCount()
	}

	if e.format == format.ODT {
		return e.odtReader.PageCount()
	}

	return e.reader.PageCount()
}

// Lines extracts and returns detected text lines with position and alignment info.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	lines, err := tabula.Open("document.pdf").Lines()
//	for _, line := range lines {
//	    fmt.Printf("%s (align: %s)\n", line.Text, line.Alignment)
//	}
func (e *Extractor) Lines() ([]layout.Line, error) {
	if e.err != nil {
		return nil, e.err
	}

	if err := e.ensureReader(); err != nil {
		return nil, err
	}
	defer e.Close()

	pageIndices, err := e.resolvePages()
	if err != nil {
		return nil, err
	}

	// Collect requested page data
	requestedPages := make([]extractedPage, 0, len(pageIndices))
	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		requestedPages = append(requestedPages, extractedPage{
			index:     pageNum,
			fragments: fragments,
			page:      page,
		})
	}

	// Detect headers/footers if needed (requires ALL pages for pattern detection)
	var headerFooterResult *layout.HeaderFooterResult
	if e.options.excludeHeaders || e.options.excludeFooters {
		allPages, err := e.collectAllPages()
		if err == nil && len(allPages) > 0 {
			headerFooterResult = e.detectHeaderFooter(allPages)
		}
	}

	// Process each requested page and collect lines
	var allLines []layout.Line
	lineDetector := layout.NewLineDetector()

	for _, pd := range requestedPages {
		fragments := pd.fragments

		// Filter headers/footers
		if headerFooterResult != nil {
			height, _ := pd.page.Height()
			fragments = headerFooterResult.FilterFragments(pd.index, fragments, height)
		}

		width, _ := pd.page.Width()
		height, _ := pd.page.Height()

		lineLayout := lineDetector.Detect(fragments, width, height)
		allLines = append(allLines, lineLayout.Lines...)
	}

	return allLines, nil
}

// Paragraphs extracts and returns detected paragraphs with style information.
// This uses reading order detection to handle multi-column layouts correctly.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	paragraphs, err := tabula.Open("document.pdf").
//	    ExcludeHeaders().
//	    ExcludeFooters().
//	    Paragraphs()
//	for _, para := range paragraphs {
//	    fmt.Printf("[%s] %s\n", para.Style, para.Text)
//	}
func (e *Extractor) Paragraphs() ([]layout.Paragraph, error) {
	if e.err != nil {
		return nil, e.err
	}

	if err := e.ensureReader(); err != nil {
		return nil, err
	}
	defer e.Close()

	pageIndices, err := e.resolvePages()
	if err != nil {
		return nil, err
	}

	// Collect requested page data
	requestedPages := make([]extractedPage, 0, len(pageIndices))
	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		requestedPages = append(requestedPages, extractedPage{
			index:     pageNum,
			fragments: fragments,
			page:      page,
		})
	}

	// Detect headers/footers if needed (requires ALL pages for pattern detection)
	var headerFooterResult *layout.HeaderFooterResult
	if e.options.excludeHeaders || e.options.excludeFooters {
		allPages, err := e.collectAllPages()
		if err == nil && len(allPages) > 0 {
			headerFooterResult = e.detectHeaderFooter(allPages)
		}
	}

	// Process each requested page and collect paragraphs
	var allParagraphs []layout.Paragraph
	roDetector := layout.NewReadingOrderDetector()

	for _, pd := range requestedPages {
		fragments := pd.fragments

		// Filter headers/footers
		if headerFooterResult != nil {
			height, _ := pd.page.Height()
			fragments = headerFooterResult.FilterFragments(pd.index, fragments, height)
		}

		width, _ := pd.page.Width()
		height, _ := pd.page.Height()

		// Use reading order detection for multi-column support
		roResult := roDetector.Detect(fragments, width, height)
		paraLayout := roResult.GetParagraphs()
		allParagraphs = append(allParagraphs, paraLayout.Paragraphs...)
	}

	return allParagraphs, nil
}

// ReadingOrder extracts and returns detailed reading order analysis.
// This includes column detection, section boundaries, and proper text ordering
// for multi-column documents.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	ro, err := tabula.Open("newspaper.pdf").Pages(1).ReadingOrder()
//	fmt.Printf("Columns: %d\n", ro.ColumnCount)
//	for _, section := range ro.Sections {
//	    fmt.Printf("Section: %s\n", section.Type)
//	}
func (e *Extractor) ReadingOrder() (*layout.ReadingOrderResult, error) {
	if e.err != nil {
		return nil, e.err
	}

	if err := e.ensureReader(); err != nil {
		return nil, err
	}
	defer e.Close()

	pageIndices, err := e.resolvePages()
	if err != nil {
		return nil, err
	}

	if len(pageIndices) == 0 {
		return nil, fmt.Errorf("no pages to process")
	}

	// Detect headers/footers if requested (needs all pages)
	var headerFooterResult *layout.HeaderFooterResult
	if e.options.excludeHeaders || e.options.excludeFooters {
		allPages, err := e.collectAllPages()
		if err == nil && len(allPages) > 0 {
			headerFooterResult = e.detectHeaderFooter(allPages)
		}
	}

	roDetector := layout.NewReadingOrderDetector()

	// Combined result across all pages
	combined := &layout.ReadingOrderResult{}

	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		// Filter headers/footers if requested
		if headerFooterResult != nil {
			height, _ := page.Height()
			fragments = headerFooterResult.FilterFragments(pageNum, fragments, height)
		}

		width, _ := page.Width()
		height, _ := page.Height()

		pageResult := roDetector.Detect(fragments, width, height)

		// Combine results
		combined.Fragments = append(combined.Fragments, pageResult.Fragments...)
		combined.Lines = append(combined.Lines, pageResult.Lines...)
		combined.Sections = append(combined.Sections, pageResult.Sections...)

		// Track max column count
		if pageResult.ColumnCount > combined.ColumnCount {
			combined.ColumnCount = pageResult.ColumnCount
		}

		// Keep page dimensions from first page
		if combined.PageWidth == 0 {
			combined.PageWidth = pageResult.PageWidth
			combined.PageHeight = pageResult.PageHeight
			combined.Direction = pageResult.Direction
		}
	}

	return combined, nil
}

// Analyze performs complete layout analysis and returns all detected elements.
// This is the most comprehensive extraction method, combining columns, lines,
// paragraphs, headings, lists, and reading order into a unified result.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	result, err := tabula.Open("document.pdf").Pages(1).Analyze()
//	for _, elem := range result.Elements {
//	    fmt.Printf("[%s] %s\n", elem.Type, elem.Text)
//	}
func (e *Extractor) Analyze() (*layout.AnalysisResult, error) {
	if e.err != nil {
		return nil, e.err
	}

	if err := e.ensureReader(); err != nil {
		return nil, err
	}
	defer e.Close()

	pageIndices, err := e.resolvePages()
	if err != nil {
		return nil, err
	}

	if len(pageIndices) == 0 {
		return nil, fmt.Errorf("no pages to process")
	}

	// Detect headers/footers if requested (needs all pages)
	var headerFooterResult *layout.HeaderFooterResult
	if e.options.excludeHeaders || e.options.excludeFooters {
		allPages, err := e.collectAllPages()
		if err == nil && len(allPages) > 0 {
			headerFooterResult = e.detectHeaderFooter(allPages)
		}
	}

	analyzer := layout.NewAnalyzer()

	// Combined result across all pages
	combined := &layout.AnalysisResult{}

	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		// Filter headers/footers if requested
		if headerFooterResult != nil {
			height, _ := page.Height()
			fragments = headerFooterResult.FilterFragments(pageNum, fragments, height)
		}

		width, _ := page.Width()
		height, _ := page.Height()

		pageResult := analyzer.Analyze(fragments, width, height)

		// Update element page indices and combine
		for i := range pageResult.Elements {
			pageResult.Elements[i].Index = len(combined.Elements) + i
			pageResult.Elements[i].ZOrder = len(combined.Elements) + i
		}

		combined.Elements = append(combined.Elements, pageResult.Elements...)
		combined.Stats.FragmentCount += pageResult.Stats.FragmentCount
		combined.Stats.LineCount += pageResult.Stats.LineCount
		combined.Stats.BlockCount += pageResult.Stats.BlockCount
		combined.Stats.ParagraphCount += pageResult.Stats.ParagraphCount
		combined.Stats.HeadingCount += pageResult.Stats.HeadingCount
		combined.Stats.ListCount += pageResult.Stats.ListCount
		combined.Stats.ElementCount += pageResult.Stats.ElementCount

		// Keep page dimensions from first page (or could use max)
		if combined.PageWidth == 0 {
			combined.PageWidth = pageResult.PageWidth
			combined.PageHeight = pageResult.PageHeight
		}
	}

	return combined, nil
}

// Headings extracts and returns detected headings (H1-H6) from the document.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	headings, err := tabula.Open("document.pdf").Headings()
//	for _, h := range headings {
//	    fmt.Printf("[%s] %s\n", h.Level, h.Text)
//	}
func (e *Extractor) Headings() ([]layout.Heading, error) {
	if e.err != nil {
		return nil, e.err
	}

	if err := e.ensureReader(); err != nil {
		return nil, err
	}
	defer e.Close()

	pageIndices, err := e.resolvePages()
	if err != nil {
		return nil, err
	}

	// Detect headers/footers if requested (once, outside the loop)
	var headerFooterResult *layout.HeaderFooterResult
	if e.options.excludeHeaders || e.options.excludeFooters {
		allPages, err := e.collectAllPages()
		if err == nil && len(allPages) > 0 {
			headerFooterResult = e.detectHeaderFooter(allPages)
		}
	}

	var allHeadings []layout.Heading
	detector := layout.NewHeadingDetector()

	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		// Filter headers/footers if requested
		if headerFooterResult != nil {
			height, _ := page.Height()
			fragments = headerFooterResult.FilterFragments(pageNum, fragments, height)
		}

		width, _ := page.Width()
		height, _ := page.Height()

		result := detector.DetectFromFragments(fragments, width, height)
		if result != nil {
			for i := range result.Headings {
				result.Headings[i].PageIndex = pageNum
			}
			allHeadings = append(allHeadings, result.Headings...)
		}
	}

	return allHeadings, nil
}

// Lists extracts and returns detected lists (bulleted, numbered, etc.) from the document.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	lists, err := tabula.Open("document.pdf").Lists()
//	for _, list := range lists {
//	    fmt.Printf("List type: %s, items: %d\n", list.Type, len(list.Items))
//	}
func (e *Extractor) Lists() ([]layout.List, error) {
	if e.err != nil {
		return nil, e.err
	}

	if err := e.ensureReader(); err != nil {
		return nil, err
	}
	defer e.Close()

	pageIndices, err := e.resolvePages()
	if err != nil {
		return nil, err
	}

	// Detect headers/footers if requested (once, outside the loop)
	var headerFooterResult *layout.HeaderFooterResult
	if e.options.excludeHeaders || e.options.excludeFooters {
		allPages, err := e.collectAllPages()
		if err == nil && len(allPages) > 0 {
			headerFooterResult = e.detectHeaderFooter(allPages)
		}
	}

	var allLists []layout.List
	detector := layout.NewListDetector()

	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		// Filter headers/footers if requested
		if headerFooterResult != nil {
			height, _ := page.Height()
			fragments = headerFooterResult.FilterFragments(pageNum, fragments, height)
		}

		width, _ := page.Width()
		height, _ := page.Height()

		result := detector.DetectFromFragments(fragments, width, height)
		if result != nil {
			allLists = append(allLists, result.Lists...)
		}
	}

	return allLists, nil
}

// Blocks extracts and returns detected text blocks from the document.
// Blocks are spatially grouped regions of text, useful for understanding
// document layout structure.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	blocks, err := tabula.Open("document.pdf").Blocks()
//	for _, block := range blocks {
//	    fmt.Printf("Block at (%.1f, %.1f): %s\n", block.BBox.X, block.BBox.Y, block.GetText())
//	}
func (e *Extractor) Blocks() ([]layout.Block, error) {
	if e.err != nil {
		return nil, e.err
	}

	if err := e.ensureReader(); err != nil {
		return nil, err
	}
	defer e.Close()

	pageIndices, err := e.resolvePages()
	if err != nil {
		return nil, err
	}

	// Detect headers/footers if requested (once, outside the loop)
	var headerFooterResult *layout.HeaderFooterResult
	if e.options.excludeHeaders || e.options.excludeFooters {
		allPages, err := e.collectAllPages()
		if err == nil && len(allPages) > 0 {
			headerFooterResult = e.detectHeaderFooter(allPages)
		}
	}

	var allBlocks []layout.Block
	detector := layout.NewBlockDetector()

	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		// Filter headers/footers if requested
		if headerFooterResult != nil {
			height, _ := page.Height()
			fragments = headerFooterResult.FilterFragments(pageNum, fragments, height)
		}

		width, _ := page.Width()
		height, _ := page.Height()

		result := detector.Detect(fragments, width, height)
		if result != nil {
			allBlocks = append(allBlocks, result.Blocks...)
		}
	}

	return allBlocks, nil
}

// Elements extracts and returns all detected elements in reading order.
// Elements include paragraphs, headings, and lists, unified into a single
// ordered list. This is useful for document reconstruction or RAG workflows.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	elements, err := tabula.Open("document.pdf").Elements()
//	for _, elem := range elements {
//	    fmt.Printf("[%s] %s\n", elem.Type, elem.Text)
//	}
func (e *Extractor) Elements() ([]layout.LayoutElement, error) {
	result, err := e.Analyze()
	if err != nil {
		return nil, err
	}
	return result.Elements, nil
}

// Document extracts content and returns a model.Document structure
// suitable for RAG chunking and other document processing workflows.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	doc, warnings, err := tabula.Open("document.pdf").
//	    ExcludeHeadersAndFooters().
//	    Document()
//	doc, warnings, err := tabula.Open("document.docx").Document()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Use doc for chunking or other processing
func (e *Extractor) Document() (*model.Document, []Warning, error) {
	if e.err != nil {
		return nil, nil, e.err
	}

	if err := e.ensureReader(); err != nil {
		return nil, nil, err
	}
	defer e.Close()

	// Handle DOCX files
	if e.format == format.DOCX {
		doc, err := e.docxReader.Document()
		if err != nil {
			return nil, e.warnings, err
		}
		return doc, e.warnings, nil
	}

	// Handle ODT files
	if e.format == format.ODT {
		doc, err := e.odtReader.Document()
		if err != nil {
			return nil, e.warnings, err
		}
		return doc, e.warnings, nil
	}

	// PDF processing
	pageIndices, err := e.resolvePages()
	if err != nil {
		return nil, nil, err
	}

	if len(pageIndices) == 0 {
		return nil, nil, fmt.Errorf("no pages to process")
	}

	// Create new document
	doc := model.NewDocument()

	// Try to get metadata from the PDF info dictionary
	if e.reader != nil {
		if info, err := e.reader.GetInfo(); err == nil && info != nil {
			if title := info.Get("Title"); title != nil {
				if s, ok := title.(core.String); ok {
					doc.Metadata.Title = string(s)
				}
			}
			if author := info.Get("Author"); author != nil {
				if s, ok := author.(core.String); ok {
					doc.Metadata.Author = string(s)
				}
			}
			if subject := info.Get("Subject"); subject != nil {
				if s, ok := subject.(core.String); ok {
					doc.Metadata.Subject = string(s)
				}
			}
			if creator := info.Get("Creator"); creator != nil {
				if s, ok := creator.(core.String); ok {
					doc.Metadata.Creator = string(s)
				}
			}
			if producer := info.Get("Producer"); producer != nil {
				if s, ok := producer.(core.String); ok {
					doc.Metadata.Producer = string(s)
				}
			}
			if keywords := info.Get("Keywords"); keywords != nil {
				if s, ok := keywords.(core.String); ok {
					doc.Metadata.Keywords = strings.Split(string(s), ",")
					for i, kw := range doc.Metadata.Keywords {
						doc.Metadata.Keywords[i] = strings.TrimSpace(kw)
					}
				}
			}
		}
	}

	// Detect headers/footers if needed (requires ALL pages for pattern detection)
	var headerFooterResult *layout.HeaderFooterResult
	if e.options.excludeHeaders || e.options.excludeFooters {
		allPages, err := e.collectAllPages()
		if err == nil && len(allPages) > 0 {
			headerFooterResult = e.detectHeaderFooter(allPages)
		}
	}

	roDetector := layout.NewReadingOrderDetector()
	paraDetector := layout.NewParagraphDetector()
	headingDetector := layout.NewHeadingDetector()
	listDetector := layout.NewListDetector()

	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return nil, nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return nil, nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		// Check for messy PDF traits on the first page processed
		if pageNum == pageIndices[0] {
			e.checkMessyPDF(fragments)
		}

		// Filter headers/footers if requested
		if headerFooterResult != nil {
			height, _ := page.Height()
			fragments = headerFooterResult.FilterFragments(pageNum, fragments, height)
		}

		width, _ := page.Width()
		height, _ := page.Height()

		// Create model page
		modelPage := model.NewPage(width, height)
		modelPage.Number = pageNum + 1

		// Perform layout analysis
		roResult := roDetector.Detect(fragments, width, height)

		// Get lines for paragraph detection
		var lines []layout.Line
		if roResult != nil && len(roResult.Lines) > 0 {
			lines = roResult.Lines
		}

		// Detect paragraphs
		var paragraphs []model.ParagraphInfo
		if len(lines) > 0 {
			paraLayout := paraDetector.Detect(lines, width, height)
			for _, para := range paraLayout.Paragraphs {
				paragraphs = append(paragraphs, model.ParagraphInfo{
					BBox:      model.BBox{X: para.BBox.X, Y: para.BBox.Y, Width: para.BBox.Width, Height: para.BBox.Height},
					Text:      para.Text,
					LineCount: len(para.Lines),
				})
			}
		}

		// Detect headings
		var headings []model.HeadingInfo
		headingResult := headingDetector.DetectFromFragments(fragments, width, height)
		if headingResult != nil {
			for _, h := range headingResult.Headings {
				headings = append(headings, model.HeadingInfo{
					Level:      int(h.Level),
					Text:       h.Text,
					BBox:       model.BBox{X: h.BBox.X, Y: h.BBox.Y, Width: h.BBox.Width, Height: h.BBox.Height},
					FontSize:   h.FontSize,
					Confidence: h.Confidence,
				})
			}
		}

		// Detect lists
		var lists []model.ListInfo
		listResult := listDetector.DetectFromFragments(fragments, width, height)
		if listResult != nil {
			for _, l := range listResult.Lists {
				listInfo := model.ListInfo{
					Type:   convertListType(l.Type),
					BBox:   model.BBox{X: l.BBox.X, Y: l.BBox.Y, Width: l.BBox.Width, Height: l.BBox.Height},
					Nested: l.Level > 0, // Consider nested if level > 0
				}
				for _, item := range l.Items {
					listInfo.Items = append(listInfo.Items, model.ListItem{
						Text:   item.Text,
						Level:  item.Level,
						Bullet: item.Prefix,
					})
				}
				lists = append(lists, listInfo)
			}
		}

		// Create layout info
		modelPage.Layout = &model.PageLayout{
			Paragraphs: paragraphs,
			Headings:   headings,
			Lists:      lists,
			Stats: model.LayoutStats{
				FragmentCount:  len(fragments),
				ParagraphCount: len(paragraphs),
				HeadingCount:   len(headings),
				ListCount:      len(lists),
			},
		}

		// Add elements to page
		for _, h := range headings {
			modelPage.AddElement(&model.Heading{
				Level: h.Level,
				Text:  h.Text,
				BBox:  h.BBox,
			})
		}
		for _, p := range paragraphs {
			modelPage.AddElement(&model.Paragraph{
				Text: p.Text,
				BBox: p.BBox,
			})
		}
		for _, l := range lists {
			modelPage.AddElement(&model.List{
				Items:   l.Items,
				Ordered: l.Type == model.ListTypeNumbered || l.Type == model.ListTypeLettered || l.Type == model.ListTypeRoman,
				BBox:    l.BBox,
			})
		}

		doc.AddPage(modelPage)
	}

	return doc, e.warnings, nil
}

// Chunks extracts content and returns semantic chunks for RAG workflows.
// This method combines document extraction with RAG chunking in a single call.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	chunks, warnings, err := tabula.Open("document.pdf").
//	    ExcludeHeadersAndFooters().
//	    Chunks()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, chunk := range chunks.Chunks {
//	    fmt.Printf("[%s] %s\n", chunk.Metadata.SectionTitle, chunk.Text[:50])
//	}
func (e *Extractor) Chunks() (*rag.ChunkCollection, []Warning, error) {
	doc, warnings, err := e.Document()
	if err != nil {
		return nil, warnings, err
	}

	chunks := rag.ChunkDocument(doc)
	return chunks, warnings, nil
}

// ChunksWithConfig extracts content and returns semantic chunks using custom configuration.
// This allows fine-tuning of chunk sizes, overlap, and other parameters.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	config := rag.ChunkerConfig{
//	    TargetChunkSize: 500,
//	    MaxChunkSize:    1000,
//	    OverlapSize:     50,
//	}
//	sizeConfig := rag.DefaultSizeConfig()
//	chunks, warnings, err := tabula.Open("document.pdf").
//	    ExcludeHeadersAndFooters().
//	    ChunksWithConfig(config, sizeConfig)
func (e *Extractor) ChunksWithConfig(config rag.ChunkerConfig, sizeConfig rag.SizeConfig) (*rag.ChunkCollection, []Warning, error) {
	doc, warnings, err := e.Document()
	if err != nil {
		return nil, warnings, err
	}

	chunks := rag.ChunkDocumentWithConfig(doc, config, sizeConfig)
	return chunks, warnings, nil
}

// ============================================================================
// Internal helpers
// ============================================================================

// resolvePages converts 1-indexed page numbers to 0-indexed and validates them.
// If no pages specified, returns all pages.
func (e *Extractor) resolvePages() ([]int, error) {
	pageCount, err := e.reader.PageCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	// If no pages specified, use all pages
	if len(e.options.pages) == 0 {
		pageIndices := make([]int, pageCount)
		for i := 0; i < pageCount; i++ {
			pageIndices[i] = i
		}
		return pageIndices, nil
	}

	// Convert 1-indexed to 0-indexed and validate
	seen := make(map[int]bool)
	var pageIndices []int
	for _, p := range e.options.pages {
		if p < 1 || p > pageCount {
			return nil, fmt.Errorf("page %d out of range (1-%d)", p, pageCount)
		}
		zeroIndexed := p - 1
		if !seen[zeroIndexed] {
			seen[zeroIndexed] = true
			pageIndices = append(pageIndices, zeroIndexed)
		}
	}

	// Sort pages in order
	sort.Ints(pageIndices)
	return pageIndices, nil
}

// detectHeaderFooter runs header/footer detection across multiple pages.
func (e *Extractor) detectHeaderFooter(allPages []extractedPage) *layout.HeaderFooterResult {
	// Convert to layout.PageFragments format
	pageFragments := make([]layout.PageFragments, len(allPages))
	for i, pd := range allPages {
		width, _ := pd.page.Width()
		height, _ := pd.page.Height()
		pageFragments[i] = layout.PageFragments{
			PageIndex:  pd.index, // Set the page index for filtering
			Fragments:  pd.fragments,
			PageWidth:  width,
			PageHeight: height,
		}
	}

	detector := layout.NewHeaderFooterDetector()
	return detector.Detect(pageFragments)
}

// collectAllPages collects fragment data from ALL pages in the document.
// This is needed for header/footer detection which requires multi-page patterns.
func (e *Extractor) collectAllPages() ([]extractedPage, error) {
	pageCount, err := e.reader.PageCount()
	if err != nil {
		return nil, err
	}

	allPages := make([]extractedPage, 0, pageCount)
	for i := 0; i < pageCount; i++ {
		page, err := e.reader.GetPage(i)
		if err != nil {
			continue // Skip pages that can't be read
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			continue // Skip pages that can't be extracted
		}

		allPages = append(allPages, extractedPage{
			index:     i,
			fragments: fragments,
			page:      page,
		})
	}

	return allPages, nil
}

// extractByColumn processes fragments column by column using reading order detection.
func (e *Extractor) extractByColumn(fragments []text.TextFragment, page *pages.Page) string {
	if len(fragments) == 0 {
		return ""
	}

	width, _ := page.Width()
	height, _ := page.Height()

	// Use reading order detector which handles multi-column layouts correctly
	roDetector := layout.NewReadingOrderDetector()
	roResult := roDetector.Detect(fragments, width, height)

	if roResult == nil || len(roResult.Sections) == 0 {
		return e.assembleText(fragments)
	}

	// Build text from sections in reading order
	var result strings.Builder

	for si, section := range roResult.Sections {
		if si > 0 && result.Len() > 0 {
			result.WriteString("\n\n")
		}

		// Output lines within each section
		for li, line := range section.Lines {
			if li > 0 {
				// Check vertical gap for paragraph breaks within section
				prevLine := section.Lines[li-1]
				gap := prevLine.BBox.Y - (line.BBox.Y + line.BBox.Height)
				if gap > line.BBox.Height*0.8 {
					result.WriteString("\n\n")
				} else {
					result.WriteString("\n")
				}
			}
			result.WriteString(strings.TrimSpace(line.Text))
		}
	}

	return result.String()
}

// assembleText combines fragments into text with appropriate spacing.
func (e *Extractor) assembleText(fragments []text.TextFragment) string {
	if len(fragments) == 0 {
		return ""
	}

	// Sort fragments by position (top to bottom, left to right)
	// Use stable sort to preserve stream order for overlapping fragments
	sorted := make([]text.TextFragment, len(fragments))
	copy(sorted, fragments)

	// Tolerance for X position comparison as fraction of font size.
	// Handles PDF generators (Word/Quartz) that place fragments in correct stream
	// order but with slightly overlapping or disordered X coordinates.
	const xTolerance = 0.25

	sort.SliceStable(sorted, func(i, j int) bool {
		// Group by Y (with tolerance for same line)
		yDiff := sorted[i].Y - sorted[j].Y
		if abs(yDiff) > sorted[i].Height*0.5 {
			return yDiff > 0 // Higher Y first (PDF coordinates)
		}

		// Same line - sort by X with tolerance
		// If X coordinates are very close, consider them equal (preserve stream order)
		tolerance := sorted[i].FontSize * xTolerance
		if abs(sorted[i].X-sorted[j].X) < tolerance {
			return false // Treat as equal, preserve order (i comes before j)
		}
		return sorted[i].X < sorted[j].X // Then left to right
	})

	var result strings.Builder
	var lastY float64
	var lastEndX float64
	firstFrag := true

	for _, frag := range sorted {
		if firstFrag {
			result.WriteString(frag.Text)
			lastY = frag.Y
			lastEndX = frag.X + frag.Width
			firstFrag = false
			continue
		}

		// Check if new line
		yDiff := abs(frag.Y - lastY)
		if yDiff > frag.Height*0.5 {
			// New line
			if yDiff > frag.Height*1.5 {
				result.WriteString("\n\n") // Paragraph break
			} else {
				result.WriteString("\n")
			}
			result.WriteString(frag.Text)
		} else {
			// Same line - check spacing
			gap := frag.X - lastEndX
			if gap > frag.FontSize*0.3 {
				result.WriteString(" ")
			}
			result.WriteString(frag.Text)
		}

		lastY = frag.Y
		lastEndX = frag.X + frag.Width
	}

	return result.String()
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// isCharacterLevel detects if fragments appear to be character-level
// (one character per fragment) which requires special handling.
// Returns true if more than 60% of fragments contain single characters.
func isCharacterLevel(fragments []text.TextFragment) bool {
	if len(fragments) < 10 {
		return false // Not enough data to determine
	}

	singleCharCount := 0
	for _, frag := range fragments {
		// Trim whitespace and check length
		text := strings.TrimSpace(frag.Text)
		if len(text) <= 1 {
			singleCharCount++
		}
	}

	// If more than 60% are single characters, it's character-level
	return float64(singleCharCount)/float64(len(fragments)) > 0.6
}

// detectMultiColumn checks if fragments appear to be laid out in multiple columns.
// Uses the ReadingOrderDetector to perform proper column analysis.
func detectMultiColumn(fragments []text.TextFragment, pageWidth, pageHeight float64) bool {
	if len(fragments) < 20 || pageWidth == 0 {
		return false
	}

	// Use the reading order detector which has sophisticated column detection
	roDetector := layout.NewReadingOrderDetector()
	result := roDetector.Detect(fragments, pageWidth, pageHeight)

	return result != nil && result.ColumnCount > 1
}
