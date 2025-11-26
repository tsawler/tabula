package tabula

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tsawler/tabula/layout"
	"github.com/tsawler/tabula/pages"
	"github.com/tsawler/tabula/reader"
	"github.com/tsawler/tabula/text"
)

// extractedPage holds the data extracted from a single page.
type extractedPage struct {
	index     int
	fragments []text.TextFragment
	page      *pages.Page
}

// Extractor provides a fluent interface for extracting content from PDFs.
// Each configuration method returns a new Extractor instance, making it
// safe for concurrent use and allowing method chaining.
type Extractor struct {
	// Source
	filename string
	reader   *reader.Reader

	// Lifecycle
	ownsReader   bool // true if we opened the reader and should close it
	readerOpened bool // true if reader has been opened

	// Configuration
	options ExtractOptions

	// Accumulated error (fail-fast)
	err error
}

// clone creates a shallow copy of the Extractor with a deep copy of options.
// This ensures immutability - each chain method returns a new instance.
func (e *Extractor) clone() *Extractor {
	newExt := &Extractor{
		filename:     e.filename,
		reader:       e.reader,
		ownsReader:   e.ownsReader,
		readerOpened: e.readerOpened,
		options:      e.options.clone(),
		err:          e.err,
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

	r, err := reader.Open(e.filename)
	if err != nil {
		return fmt.Errorf("failed to open PDF: %w", err)
	}

	e.reader = r
	e.ownsReader = true
	e.readerOpened = true
	return nil
}

// Close releases resources associated with the Extractor.
// It is safe to call Close multiple times.
func (e *Extractor) Close() error {
	if e.reader != nil && e.ownsReader {
		err := e.reader.Close()
		e.reader = nil
		e.ownsReader = false
		return err
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
//	text, err := tabula.Open("doc.pdf").Pages(1, 3, 5).Text()
func (e *Extractor) Pages(pages ...int) *Extractor {
	newExt := e.clone()
	newExt.options.pages = append(newExt.options.pages, pages...)
	return newExt
}

// PageRange specifies a range of pages to extract (1-indexed, inclusive).
//
// Example:
//
//	text, err := tabula.Open("doc.pdf").PageRange(5, 10).Text()
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
//	text, err := tabula.Open("doc.pdf").ExcludeHeaders().Text()
func (e *Extractor) ExcludeHeaders() *Extractor {
	newExt := e.clone()
	newExt.options.excludeHeaders = true
	return newExt
}

// ExcludeFooters configures the extractor to exclude detected footers.
//
// Example:
//
//	text, err := tabula.Open("doc.pdf").ExcludeFooters().Text()
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
//	text, err := tabula.Open("doc.pdf").ExcludeHeadersAndFooters().Text()
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
//	text, err := tabula.Open("doc.pdf").JoinParagraphs().Text()
//	text, err := tabula.Open("doc.pdf").ExcludeHeadersAndFooters().JoinParagraphs().Text()
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
//	text, err := tabula.Open("newspaper.pdf").ByColumn().Text()
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
//	text, err := tabula.Open("form.pdf").PreserveLayout().Text()
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
// Example:
//
//	text, err := tabula.Open("document.pdf").Text()
func (e *Extractor) Text() (string, error) {
	if e.err != nil {
		return "", e.err
	}

	if err := e.ensureReader(); err != nil {
		return "", err
	}
	defer e.Close()

	pageIndices, err := e.resolvePages()
	if err != nil {
		return "", err
	}

	// Collect requested page data
	requestedPages := make([]extractedPage, 0, len(pageIndices))

	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return "", fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return "", fmt.Errorf("page %d: %w", pageNum+1, err)
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

	return result.String(), nil
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

// Fragments extracts and returns text fragments with position information.
// This is a terminal operation that closes the underlying reader.
//
// Example:
//
//	fragments, err := tabula.Open("document.pdf").Pages(1).Fragments()
func (e *Extractor) Fragments() ([]text.TextFragment, error) {
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

	var allFragments []text.TextFragment
	for _, pageNum := range pageIndices {
		page, err := e.reader.GetPage(pageNum)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		fragments, err := e.reader.ExtractTextFragments(page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum+1, err)
		}

		allFragments = append(allFragments, fragments...)
	}

	return allFragments, nil
}

// PageCount returns the total number of pages in the PDF.
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
	sorted := make([]text.TextFragment, len(fragments))
	copy(sorted, fragments)
	sort.Slice(sorted, func(i, j int) bool {
		// Group by Y (with tolerance for same line)
		yDiff := sorted[i].Y - sorted[j].Y
		if abs(yDiff) > sorted[i].Height*0.5 {
			return yDiff > 0 // Higher Y first (PDF coordinates)
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
