# Tabula

A pure-Go text extraction library with a fluent API, designed for RAG (Retrieval-Augmented Generation) workflows. Supports PDF and docx files.

## Features

- **Fluent API** - Chain methods for clean, readable code
- **Multi-Format Support** - PDF (.pdf) and Word (.docx) files
- **Layout Analysis** - Detect headings, paragraphs, lists, and tables
- **Header/Footer Detection** - Automatically identify and exclude repeating content (PDF)
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
    // Works with both PDF and DOCX files
    text, warnings, err := tabula.Open("document.pdf").Text()
    // text, warnings, err := tabula.Open("document.docx").Text()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(text)

    for _, w := range warnings {
        fmt.Println("Warning:", w.Message)
    }
}
```

### Extract with Options (PDF)

```go
text, warnings, err := tabula.Open("document.pdf").
    Pages(1, 2, 3).              // Specific pages (PDF only)
    ExcludeHeadersAndFooters().  // Remove repeating headers/footers (PDF only)
    JoinParagraphs().            // Join text into paragraphs (PDF only)
    Text()
```

### Extract as Markdown

```go
// PDF with header/footer exclusion
markdown, warnings, err := tabula.Open("document.pdf").
    ExcludeHeadersAndFooters().
    ToMarkdown()

// DOCX (preserves headings, lists, tables)
markdown, warnings, err := tabula.Open("document.docx").ToMarkdown()
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
    // Works with both PDF and DOCX
    chunks, warnings, err := tabula.Open("document.pdf").Chunks()
    // chunks, warnings, err := tabula.Open("document.docx").Chunks()
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

// From existing PDF reader (PDF only)
r, _ := reader.Open("document.pdf")
ext := tabula.FromReader(r)
```

### Fluent Options

| Method | Description | Formats |
|--------|-------------|---------|
| `Pages(1, 2, 3)` | Extract specific pages (1-indexed) | PDF |
| `PageRange(1, 10)` | Extract page range (inclusive) | PDF |
| `ExcludeHeaders()` | Exclude detected headers | PDF |
| `ExcludeFooters()` | Exclude detected footers | PDF |
| `ExcludeHeadersAndFooters()` | Exclude both | PDF |
| `JoinParagraphs()` | Join text fragments into paragraphs | PDF |
| `ByColumn()` | Process multi-column layouts column by column | PDF |
| `PreserveLayout()` | Maintain spatial positioning | PDF |

### Terminal Operations

| Method | Returns | Description | Formats |
|--------|---------|-------------|---------|
| `Text()` | `string` | Plain text content | PDF, DOCX |
| `ToMarkdown()` | `string` | Markdown-formatted content | PDF, DOCX |
| `ToMarkdownWithOptions(opts)` | `string` | Markdown with custom options | PDF |
| `Document()` | `*model.Document` | Full document structure | PDF, DOCX |
| `Chunks()` | `*rag.ChunkCollection` | Semantic chunks for RAG | PDF, DOCX |
| `ChunksWithConfig(config, sizeConfig)` | `*rag.ChunkCollection` | Chunks with custom sizing | PDF, DOCX |
| `PageCount()` | `int` | Number of pages | PDF, DOCX |
| `Fragments()` | `[]text.TextFragment` | Raw text fragments with positions | PDF |
| `Lines()` | `[]layout.Line` | Detected text lines | PDF |
| `Paragraphs()` | `[]layout.Paragraph` | Detected paragraphs | PDF |
| `Headings()` | `[]layout.Heading` | Detected headings (H1-H6) | PDF |
| `Lists()` | `[]layout.List` | Detected lists | PDF |
| `Blocks()` | `[]layout.Block` | Text blocks | PDF |
| `Elements()` | `[]layout.LayoutElement` | All elements in reading order | PDF |
| `Analyze()` | `*layout.AnalysisResult` | Complete layout analysis | PDF |

### Inspection Methods (non-terminal, PDF only)

```go
ext := tabula.Open("document.pdf")
defer ext.Close()

isCharLevel, _ := ext.IsCharacterLevel()  // Detect character-level PDFs
isMultiCol, _ := ext.IsMultiColumn()      // Detect multi-column layouts
pageCount, _ := ext.PageCount()           // Get page count (works with DOCX too)
```

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

opts := rag.MarkdownOptions{
    IncludeMetadata:        true,   // YAML front matter
    IncludeTableOfContents: true,   // Generated TOC
    IncludeChunkSeparators: true,   // --- between chunks
    IncludePageNumbers:     true,   // Page references
    IncludeChunkIDs:        true,   // HTML comments with chunk IDs
}

markdown, _, _ := tabula.Open("doc.pdf").ToMarkdownWithOptions(opts)

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
