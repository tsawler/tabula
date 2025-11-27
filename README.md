# Tabula

A pure-Go PDF text extraction library with a fluent API, designed for RAG (Retrieval-Augmented Generation) workflows.

## Features

- **Fluent API** - Chain methods for clean, readable code
- **Layout Analysis** - Detect headings, paragraphs, lists, and columns
- **Header/Footer Detection** - Automatically identify and exclude repeating content
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
    text, warnings, err := tabula.Open("document.pdf").Text()
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
text, warnings, err := tabula.Open("document.pdf").
    Pages(1, 2, 3).              // Specific pages
    ExcludeHeadersAndFooters().  // Remove repeating headers/footers
    JoinParagraphs().            // Join text into paragraphs
    Text()
```

### Extract as Markdown

```go
markdown, warnings, err := tabula.Open("document.pdf").
    ExcludeHeadersAndFooters().
    ToMarkdown()
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
    chunks, warnings, err := tabula.Open("document.pdf").
        ExcludeHeadersAndFooters().
        Chunks()
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

### Opening a PDF

```go
// From file path
ext := tabula.Open("document.pdf")

// From existing reader
r, _ := reader.Open("document.pdf")
ext := tabula.FromReader(r)
```

### Fluent Options

| Method | Description |
|--------|-------------|
| `Pages(1, 2, 3)` | Extract specific pages (1-indexed) |
| `PageRange(1, 10)` | Extract page range (inclusive) |
| `ExcludeHeaders()` | Exclude detected headers |
| `ExcludeFooters()` | Exclude detected footers |
| `ExcludeHeadersAndFooters()` | Exclude both |
| `JoinParagraphs()` | Join text fragments into paragraphs |
| `ByColumn()` | Process multi-column layouts column by column |
| `PreserveLayout()` | Maintain spatial positioning |

### Terminal Operations

| Method | Returns | Description |
|--------|---------|-------------|
| `Text()` | `string` | Plain text content |
| `ToMarkdown()` | `string` | Markdown-formatted content |
| `ToMarkdownWithOptions(opts)` | `string` | Markdown with custom options |
| `Fragments()` | `[]text.TextFragment` | Raw text fragments with positions |
| `Lines()` | `[]layout.Line` | Detected text lines |
| `Paragraphs()` | `[]layout.Paragraph` | Detected paragraphs |
| `Headings()` | `[]layout.Heading` | Detected headings (H1-H6) |
| `Lists()` | `[]layout.List` | Detected lists |
| `Blocks()` | `[]layout.Block` | Text blocks |
| `Elements()` | `[]layout.LayoutElement` | All elements in reading order |
| `Document()` | `*model.Document` | Full document structure |
| `Chunks()` | `*rag.ChunkCollection` | Semantic chunks for RAG |
| `ChunksWithConfig(config, sizeConfig)` | `*rag.ChunkCollection` | Chunks with custom sizing |
| `Analyze()` | `*layout.AnalysisResult` | Complete layout analysis |
| `PageCount()` | `int` | Number of pages |

### Inspection Methods (non-terminal)

```go
ext := tabula.Open("document.pdf")
defer ext.Close()

isCharLevel, _ := ext.IsCharacterLevel()  // Detect character-level PDFs
isMultiCol, _ := ext.IsMultiColumn()      // Detect multi-column layouts
pageCount, _ := ext.PageCount()           // Get page count
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
