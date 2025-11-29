<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with GoLang"></a>
[![Version](https://img.shields.io/badge/goversion-1.21.x-blue.svg)](https://golang.org)
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/tsawler/tabula/master/LICENSE.md)
<a href="https://pkg.go.dev/github.com/tsawler/tabula"><img src="https://img.shields.io/badge/godoc-reference-%23007d9c.svg"></a>
[![Go Report Card](https://goreportcard.com/badge/github.com/tsawler/tabula)](https://goreportcard.com/report/github.com/tsawler/tabula)

# Tabula

A pure-Go text extraction library with a fluent API, designed for RAG (Retrieval-Augmented Generation) workflows. Supports PDF, DOCX, ODT, XLSX, PPTX, HTML, and EPUB files.

## Features

- **Fluent API** - Chain methods for clean, readable code
- **Multi-Format Support** - PDF (.pdf), Word (.docx), OpenDocument (.odt), Excel (.xlsx), PowerPoint (.pptx), HTML (.html, .htm), and EPUB (.epub) files
- **Layout Analysis** - Detect headings, paragraphs, lists, and tables
- **Header/Footer Detection** - Automatically identify and exclude repeating content
- **HTML Navigation Filtering** - Remove headers, footers, nav, and sidebars from web pages with configurable exclusion modes
- **RAG-Ready Chunking** - Semantic document chunking with metadata
- **Markdown Export** - Convert extracted content to markdown
- **PDF 1.0-1.7 Support** - Including modern XRef streams (PDF 1.5+)
- **Pure Go** - No CGO dependencies

## Installation

```bash
go get github.com/tsawler/tabula
```

## Quick Start

### Extract Text

```go
package main

import (
    "fmt"
    "log"

    "github.com/tsawler/tabula"
)

func main() {
    // Works with PDF, DOCX, ODT, XLSX, PPTX, HTML, and EPUB files
    text, warnings, err := tabula.Open("document.pdf").Text()
    // text, warnings, err := tabula.Open("document.docx").Text()
    // text, warnings, err := tabula.Open("document.odt").Text()
    // text, warnings, err := tabula.Open("spreadsheet.xlsx").Text()
    // text, warnings, err := tabula.Open("presentation.pptx").Text()
    // text, warnings, err := tabula.Open("page.html").Text()
    // text, warnings, err := tabula.Open("book.epub").Text()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(text)

    for _, w := range warnings {
        fmt.Println("Warning:", w.Message)
    }
}
```

### Extract with Options

```go
// PDF with all options
text, warnings, err := tabula.Open("document.pdf").
    Pages(1, 2, 3).              // Specific pages (PDF only)
    ExcludeHeadersAndFooters().  // Remove headers/footers
    JoinParagraphs().            // Join text into paragraphs (PDF only)
    Text()

// DOCX/ODT with header/footer exclusion
text, warnings, err := tabula.Open("document.docx").
    ExcludeHeadersAndFooters().  // Remove headers/footers
    Text()

// XLSX (each sheet extracted as tab-separated values)
text, warnings, err := tabula.Open("spreadsheet.xlsx").Text()

// PPTX (each slide extracted with title and content)
text, warnings, err := tabula.Open("presentation.pptx").
    ExcludeHeadersAndFooters().  // Remove slide footers and numbers
    Text()

// HTML (extracts text from headings, paragraphs, lists, tables)
// Use navigation exclusion to remove headers, footers, and nav elements
text, warnings, err := tabula.Open("page.html").Text()

// HTML from URL with navigation filtering
import "github.com/tsawler/tabula/htmldoc"
resp, _ := http.Get("https://example.com")
reader, _ := htmldoc.OpenReader(resp.Body)
opts := htmldoc.ExtractOptions{NavigationExclusion: htmldoc.NavigationExclusionStandard}
text, _ := reader.TextWithOptions(opts)

// EPUB (e-books, supports EPUB 2 and EPUB 3)
text, warnings, err := tabula.Open("book.epub").Text()
```

### Extract as Markdown

```go
// PDF with header/footer exclusion
markdown, warnings, err := tabula.Open("document.pdf").
    ExcludeHeadersAndFooters().
    ToMarkdown()

// DOCX with header/footer exclusion (preserves headings, lists, tables)
markdown, warnings, err := tabula.Open("document.docx").
    ExcludeHeadersAndFooters().
    ToMarkdown()

// ODT with header/footer exclusion (preserves headings, lists, tables)
markdown, warnings, err := tabula.Open("document.odt").
    ExcludeHeadersAndFooters().
    ToMarkdown()

// XLSX (each sheet as a markdown table)
markdown, warnings, err := tabula.Open("spreadsheet.xlsx").ToMarkdown()

// PPTX (each slide with title as heading, content, and tables)
markdown, warnings, err := tabula.Open("presentation.pptx").
    ExcludeHeadersAndFooters().
    ToMarkdown()

// HTML (preserves headings, lists, tables, code blocks)
// Use navigation exclusion to remove headers, footers, and nav elements
markdown, warnings, err := tabula.Open("page.html").ToMarkdown()

// HTML with aggressive navigation filtering
import "github.com/tsawler/tabula/htmldoc"
reader, _ := htmldoc.OpenReader(resp.Body)
opts := htmldoc.ExtractOptions{NavigationExclusion: htmldoc.NavigationExclusionAggressive}
markdown, _ := reader.MarkdownWithOptions(opts)

// EPUB (preserves chapter structure, headings, lists)
markdown, warnings, err := tabula.Open("book.epub").ToMarkdown()
```

### RAG Chunking

```go
package main

import (
    "fmt"
    "log"

    "github.com/tsawler/tabula"
)

func main() {
    // Works with PDF, DOCX, ODT, XLSX, PPTX, HTML, and EPUB
    chunks, warnings, err := tabula.Open("document.pdf").Chunks()
    // chunks, warnings, err := tabula.Open("document.docx").Chunks()
    // chunks, warnings, err := tabula.Open("document.odt").Chunks()
    // chunks, warnings, err := tabula.Open("spreadsheet.xlsx").Chunks()
    // chunks, warnings, err := tabula.Open("presentation.pptx").Chunks()
    // chunks, warnings, err := tabula.Open("page.html").Chunks()
    // chunks, warnings, err := tabula.Open("book.epub").Chunks()
    if err != nil {
        log.Fatal(err)
    }

    for i, chunk := range chunks.Chunks {
        fmt.Printf("Chunk %d: %s (p.%d-%d, ~%d tokens)\n",
            i+1,
            chunk.Metadata.SectionTitle,
            chunk.Metadata.PageStart,
            chunk.Metadata.PageEnd,
            chunk.Metadata.EstimatedTokens)
        fmt.Println(chunk.Text)
        fmt.Println("---")
    }

    // Warnings are non-fatal issues
    for _, w := range warnings {
        fmt.Println("Warning:", w.Message)
    }
}
```

### Chunks as Markdown (for Vector DBs)

```go
chunks, _, err := tabula.Open("document.pdf").
    ExcludeHeadersAndFooters().
    Chunks()
if err != nil {
    log.Fatal(err)
}

// Get each chunk as separate markdown strings
mdChunks := chunks.ToMarkdownChunks()

for i, md := range mdChunks {
    // Store each chunk in your vector database
    embedding := embedModel.Embed(md)
    vectorDB.Store(chunks.Chunks[i].ID, embedding, md)
}
```

## API Reference

### Opening Documents

```go
// From file path (format auto-detected by extension)
ext := tabula.Open("document.pdf")
ext := tabula.Open("document.docx")
ext := tabula.Open("document.odt")
ext := tabula.Open("spreadsheet.xlsx")
ext := tabula.Open("presentation.pptx")
ext := tabula.Open("page.html")
ext := tabula.Open("book.epub")

// From existing PDF reader (PDF only)
r, _ := reader.Open("document.pdf")
ext := tabula.FromReader(r)

// From HTML string (useful for web scraping)
html := `<html><body><h1>Hello</h1><p>World</p></body></html>`
ext := tabula.FromHTMLString(html)

// From HTML io.Reader (useful for HTTP responses)
resp, _ := http.Get("https://example.com/page")
defer resp.Body.Close()
ext := tabula.FromHTMLReader(resp.Body)
```

### Fluent Options

| Method | Description | Formats |
|--------|-------------|---------|
| `Pages(1, 2, 3)` | Extract specific pages (1-indexed) | PDF |
| `PageRange(1, 10)` | Extract page range (inclusive) | PDF |
| `ExcludeHeaders()` | Exclude detected headers | PDF, DOCX, ODT, XLSX, PPTX, EPUB |
| `ExcludeFooters()` | Exclude detected footers | PDF, DOCX, ODT, XLSX, PPTX, EPUB |
| `ExcludeHeadersAndFooters()` | Exclude both | PDF, DOCX, ODT, XLSX, PPTX, EPUB |
| `JoinParagraphs()` | Join text fragments into paragraphs | PDF |
| `ByColumn()` | Process multi-column layouts column by column | PDF |
| `PreserveLayout()` | Maintain spatial positioning | PDF |

**Note:** HTML files are single-page documents, so page selection options don't apply. For HTML navigation/header/footer removal, use the `htmldoc` package directly with `NavigationExclusionMode` options (see below).

### Terminal Operations

| Method | Returns | Description | Formats |
|--------|---------|-------------|---------|
| `Text()` | `string` | Plain text content | PDF, DOCX, ODT, XLSX, PPTX, HTML, EPUB |
| `ToMarkdown()` | `string` | Markdown-formatted content | PDF, DOCX, ODT, XLSX, PPTX, HTML, EPUB |
| `ToMarkdownWithOptions(opts)` | `string` | Markdown with custom options | PDF, DOCX, ODT, XLSX, PPTX, HTML, EPUB |
| `Document()` | `*model.Document` | Full document structure | PDF, DOCX, ODT, XLSX, PPTX, HTML, EPUB |
| `Chunks()` | `*rag.ChunkCollection` | Semantic chunks for RAG | PDF, DOCX, ODT, XLSX, PPTX, HTML, EPUB |
| `ChunksWithConfig(config, sizeConfig)` | `*rag.ChunkCollection` | Chunks with custom sizing | PDF, DOCX, ODT, XLSX, PPTX, HTML, EPUB |
| `PageCount()` | `int` | Number of pages/sheets/slides/chapters | PDF, DOCX, ODT, XLSX, PPTX, HTML, EPUB |
| `Fragments()` | `[]text.TextFragment` | Raw text fragments with positions | PDF |
| `Lines()` | `[]layout.Line` | Detected text lines | PDF |
| `Paragraphs()` | `[]layout.Paragraph` | Detected paragraphs | PDF |
| `Headings()` | `[]layout.Heading` | Detected headings (H1-H6) | PDF |
| `Lists()` | `[]layout.List` | Detected lists | PDF |
| `Blocks()` | `[]layout.Block` | Text blocks | PDF |
| `Elements()` | `[]layout.LayoutElement` | All elements in reading order | PDF |
| `Analyze()` | `*layout.AnalysisResult` | Complete layout analysis | PDF |

**Note on PDF-only methods:** The methods marked "PDF" in the tables above (`Pages`, `PageRange`, `JoinParagraphs`, `ByColumn`, `PreserveLayout`, `Fragments`, `Lines`, `Paragraphs`, `Headings`, `Lists`, `Blocks`, `Elements`, `Analyze`) exist because PDFs lack semantic structure - they store raw text fragments at arbitrary positions, requiring layout analysis to reconstruct document structure. DOCX, ODT, XLSX, PPTX, HTML, and EPUB files already contain explicit semantic markup, so these detection methods aren't needed. Use `Document()` to access the semantic structure for all formats.

**Note on XLSX:** For Excel files, each sheet becomes a page, and the sheet data is represented as a table element. `PageCount()` returns the number of sheets. `Text()` returns tab-separated values, while `ToMarkdown()` formats each sheet as a markdown table.

**Note on PPTX:** For PowerPoint files, each slide becomes a page. `PageCount()` returns the number of slides. Slide titles are extracted as headings, bullet points as lists, and tables are preserved. Use `ExcludeHeadersAndFooters()` to remove slide footers, dates, and slide numbers.

**Note on HTML:** For HTML files, the entire document is treated as a single page. `PageCount()` returns 1. Semantic elements are preserved: headings (`<h1>`-`<h6>`), paragraphs (`<p>`), lists (`<ul>`, `<ol>`), tables (`<table>` with colspan/rowspan), code blocks (`<pre>`, `<code>`), and blockquotes (`<blockquote>`). Metadata is extracted from `<title>` and `<meta>` tags. For navigation/header/footer removal, use the `htmldoc` package with `NavigationExclusionMode` (see HTML Navigation Filtering section below).

**Note on EPUB:** For EPUB files (both EPUB 2 and EPUB 3), each chapter (spine item) becomes a page. `PageCount()` returns the number of chapters. Dublin Core metadata is extracted (title, author, language, identifier/ISBN, etc.). The table of contents is parsed from NCX (EPUB 2) or nav document (EPUB 3). DRM-protected EPUBs are rejected with an error. Content is extracted using the HTML parser, preserving headings, paragraphs, lists, and tables.

### Inspection Methods (non-terminal, PDF only)

```go
ext := tabula.Open("document.pdf")
defer ext.Close()

isCharLevel, _ := ext.IsCharacterLevel()  // Detect character-level PDFs
isMultiCol, _ := ext.IsMultiColumn()      // Detect multi-column layouts
pageCount, _ := ext.PageCount()           // Get page count (works with DOCX and ODT too)
```

### HTML Navigation Filtering

When processing HTML content (especially web pages), use the `htmldoc` package directly to filter out navigation, headers, footers, and sidebars:

```go
import "github.com/tsawler/tabula/htmldoc"

// From HTTP response
resp, _ := http.Get("https://example.com/article")
defer resp.Body.Close()
reader, _ := htmldoc.OpenReader(resp.Body)

// Choose exclusion mode
opts := htmldoc.ExtractOptions{
    NavigationExclusion: htmldoc.NavigationExclusionStandard,
}

// Extract clean text or markdown
text, _ := reader.TextWithOptions(opts)
markdown, _ := reader.MarkdownWithOptions(opts)
```

**Navigation Exclusion Modes:**

| Mode | Description |
|------|-------------|
| `NavigationExclusionNone` | Include all content without filtering |
| `NavigationExclusionExplicit` | Skip only semantic HTML5 elements: `<nav>`, `<aside>`, and ARIA roles. `<header>`/`<footer>` only when top-level |
| `NavigationExclusionStandard` | Explicit + class/id pattern matching (nav, navbar, menu, footer, sidebar, etc.) |
| `NavigationExclusionAggressive` | Standard + link-density heuristics (excludes sections with >60% link text) |

## RAG Integration

### Chunk Filtering

```go
chunks, _, _ := tabula.Open("doc.pdf").Chunks()

// Filter by content type
tablesOnly := chunks.FilterWithTables()
listsOnly := chunks.FilterWithLists()

// Filter by location
section := chunks.FilterBySection("Introduction")
page5 := chunks.FilterByPage(5)
pages1to10 := chunks.FilterByPageRange(1, 10)

// Filter by size
smallChunks := chunks.FilterByMaxTokens(500)
largeChunks := chunks.FilterByMinTokens(100)

// Search
matches := chunks.Search("keyword")

// Chain filters
result := chunks.
    FilterBySection("Methods").
    FilterByMinTokens(100).
    Search("algorithm")
```

### Markdown Options

```go
import "github.com/tsawler/tabula/rag"

// Options supported by all formats (PDF, DOCX, ODT, XLSX, PPTX, HTML, EPUB)
opts := rag.MarkdownOptions{
    IncludeMetadata:        true,   // YAML front matter with document metadata
    IncludeTableOfContents: true,   // Generated TOC from headings
    HeadingLevelOffset:     0,      // Adjust heading levels (1 makes H1 -> H2)
    MaxHeadingLevel:        6,      // Cap heading depth
}

// Works with all formats
markdown, _, _ := tabula.Open("doc.pdf").ToMarkdownWithOptions(opts)
markdown, _, _ := tabula.Open("doc.docx").ToMarkdownWithOptions(opts)
markdown, _, _ := tabula.Open("doc.odt").ToMarkdownWithOptions(opts)
markdown, _, _ := tabula.Open("spreadsheet.xlsx").ToMarkdownWithOptions(opts)
markdown, _, _ := tabula.Open("presentation.pptx").ToMarkdownWithOptions(opts)
markdown, _, _ := tabula.Open("page.html").ToMarkdownWithOptions(opts)
markdown, _, _ := tabula.Open("book.epub").ToMarkdownWithOptions(opts)

// PDF-only options (used via RAG chunking pipeline)
pdfOpts := rag.MarkdownOptions{
    IncludeMetadata:        true,
    IncludeChunkSeparators: true,   // --- between chunks (PDF only)
    IncludePageNumbers:     true,   // Page references (PDF only)
    IncludeChunkIDs:        true,   // HTML comments with chunk IDs (PDF only)
}

// Or use preset for RAG
opts := rag.RAGOptimizedMarkdownOptions()
```

### Custom Chunk Sizing

```go
import "github.com/tsawler/tabula/rag"

config := rag.ChunkerConfig{
    TargetChunkSize: 500,   // Target characters per chunk
    MaxChunkSize:    1000,  // Maximum characters
    MinChunkSize:    100,   // Minimum characters
    OverlapSize:     50,    // Overlap between chunks
}
sizeConfig := rag.DefaultSizeConfig()

chunks, _, _ := tabula.Open("doc.pdf").ChunksWithConfig(config, sizeConfig)
```

## Working with Results

### Chunk Metadata

```go
for _, chunk := range chunks.Chunks {
    fmt.Println("ID:", chunk.ID)
    fmt.Println("Section:", chunk.Metadata.SectionTitle)
    fmt.Println("Pages:", chunk.Metadata.PageStart, "-", chunk.Metadata.PageEnd)
    fmt.Println("Words:", chunk.Metadata.WordCount)
    fmt.Println("Tokens:", chunk.Metadata.EstimatedTokens)
    fmt.Println("Has Table:", chunk.Metadata.HasTable)
    fmt.Println("Has List:", chunk.Metadata.HasList)
}
```

### Collection Statistics

```go
stats := chunks.Statistics()
fmt.Println("Total chunks:", stats.TotalChunks)
fmt.Println("Total words:", stats.TotalWords)
fmt.Println("Average tokens:", stats.AvgTokens)
fmt.Println("Chunks with tables:", stats.ChunksWithTables)
```

## Warnings

The library returns warnings for non-fatal issues:

```go
text, warnings, err := tabula.Open("document.pdf").Text()
if err != nil {
    log.Fatal(err)  // Fatal error
}

for _, w := range warnings {
    log.Println("Warning:", w.Message)  // Non-fatal issues
}

// Format all warnings
formatted := tabula.FormatWarnings(warnings)
```

Common warnings:
- "Detected messy/display-oriented PDF traits" - PDF may have unusual text layout
- High fragmentation warnings - Text is split into many small fragments

## Error Handling Helpers

```go
// Panic on error (for scripts/tests)
text := tabula.MustText(tabula.Open("doc.pdf").Text())
count := tabula.Must(tabula.Open("doc.pdf").PageCount())
```

## Testing

```bash
go test ./...
```

## License

MIT License

## Related Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) - System architecture
- [PDF_PARSING_GUIDE.md](PDF_PARSING_GUIDE.md) - PDF internals
- [RAG_INTEGRATION.md](RAG_INTEGRATION.md) - RAG pipeline details
