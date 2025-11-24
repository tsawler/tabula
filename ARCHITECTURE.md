# PDF Library Architecture

## Overview

This document describes the architecture of a Go-based PDF parsing and writing library optimized for RAG (Retrieval-Augmented Generation) pipelines, with advanced table extraction capabilities.

## Design Principles

1. **Pure Go** - No CGO unless absolutely necessary
2. **Streaming Architecture** - Minimize memory footprint for large PDFs
3. **Layered Abstraction** - Clear separation between PDF primitives, layout analysis, and semantic extraction
4. **Extensible** - Plugin architecture for ML-based enhancements
5. **Idiomatic Go** - Follow Go best practices and conventions

## Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│  (RAG Integration, CLI Tools, API Servers)                  │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │
┌─────────────────────────────────────────────────────────────┐
│                  High-Level API Layer                        │
│  - Document Model (IR)                                       │
│  - Table Extractor                                           │
│  - Layout Analyzer                                           │
│  - Text Extractor                                            │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │
┌─────────────────────────────────────────────────────────────┐
│              Content Processing Layer                        │
│  - Content Stream Parser                                     │
│  - Text Rendering                                            │
│  - Graphics State Machine                                    │
│  - Font Handler                                              │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │
┌─────────────────────────────────────────────────────────────┐
│                PDF Primitive Layer                           │
│  - Object Parser                                             │
│  - Stream Decoder                                            │
│  - XRef Handler                                              │
│  - Encryption Handler                                        │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │
┌─────────────────────────────────────────────────────────────┐
│                   I/O Layer                                  │
│  - File Reader/Writer                                        │
│  - Buffer Management                                         │
│  - Compression (Flate, LZW, etc.)                           │
└─────────────────────────────────────────────────────────────┘
```

## Package Structure

```
github.com/tsawler/tabula/
├── pdf.go                    # Main package API
├── model/                    # Intermediate Representation
│   ├── document.go
│   ├── page.go
│   ├── element.go
│   ├── table.go
│   └── geometry.go
├── core/                     # PDF Primitives
│   ├── object.go            # PDF objects (dict, array, stream, etc.)
│   ├── parser.go            # Low-level PDF syntax parser
│   ├── xref.go              # Cross-reference table handling
│   ├── stream.go            # Stream encoding/decoding
│   └── encrypt.go           # Encryption support
├── reader/                   # PDF Reading
│   ├── reader.go            # Main reader
│   ├── page.go              # Page extraction
│   └── catalog.go           # Document catalog
├── writer/                   # PDF Writing
│   ├── writer.go            # Main writer
│   ├── composer.go          # Document composition
│   └── optimizer.go         # PDF optimization
├── contentstream/            # Content Stream Processing
│   ├── parser.go            # Content stream parser
│   ├── operators.go         # PDF operators
│   └── graphics.go          # Graphics state machine
├── text/                     # Text Extraction
│   ├── extractor.go         # Text extraction engine
│   ├── encoding.go          # Text encoding/decoding
│   └── unicode.go           # Unicode mapping
├── font/                     # Font Handling
│   ├── font.go              # Font interface
│   ├── type1.go             # Type 1 fonts
│   ├── truetype.go          # TrueType fonts
│   ├── cmap.go              # CMap support
│   └── metrics.go           # Font metrics
├── layout/                   # Layout Analysis
│   ├── analyzer.go          # Layout analyzer
│   ├── segment.go           # Text segmentation
│   ├── block.go             # Block detection
│   └── reading_order.go     # Reading order determination
├── tables/                   # Table Detection & Extraction
│   ├── detector.go          # Table detector interface
│   ├── geometric.go         # Geometric heuristics detector
│   ├── ml.go                # ML-based detector (future)
│   ├── cell.go              # Cell extraction
│   └── grid.go              # Grid reconstruction
├── image/                    # Image Handling
│   ├── extractor.go         # Image extraction
│   └── decoder.go           # Image decoding
└── internal/                 # Internal utilities
    ├── buffer/              # Buffered I/O
    ├── compress/            # Compression algorithms
    └── pool/                # Object pooling
```

## Core Data Structures

### Intermediate Representation (IR)

```go
// Document represents the complete extracted structure
type Document struct {
    Metadata Metadata
    Pages    []Page
}

// Page represents a single page
type Page struct {
    Number   int
    Width    float64
    Height   float64
    Elements []Element
    // Raw geometric info for debugging
    RawText  []TextFragment
}

// Element is the interface for all page elements
type Element interface {
    Type() ElementType
    BoundingBox() BBox
    ZIndex() int
}

// ElementType enum
type ElementType int
const (
    ElementTypeParagraph ElementType = iota
    ElementTypeHeading
    ElementTypeList
    ElementTypeTable
    ElementTypeImage
    ElementTypeFigure
)

// Paragraph text block
type Paragraph struct {
    Text     string
    BBox     BBox
    FontSize float64
    Style    TextStyle
}

// Table with full structure
type Table struct {
    Rows     [][]Cell
    BBox     BBox
    HasGrid  bool
    Confidence float64 // 0-1 confidence score
}

// Cell in a table
type Cell struct {
    Text     string
    BBox     BBox
    RowSpan  int
    ColSpan  int
    IsHeader bool
}

// Geometry types
type BBox struct {
    X, Y, Width, Height float64
}

type Point struct {
    X, Y float64
}
```

## Key Algorithms

### 1. PDF Object Graph Traversal

PDFs are object graphs with cross-references. The parser must:
1. Read XRef table/stream to build object location map
2. Lazy-load objects on demand
3. Resolve indirect references
4. Handle object streams (compressed object collections)

### 2. Content Stream Processing

Content streams contain drawing commands. The processor must:
1. Parse operators and operands
2. Maintain graphics state stack
3. Track text matrix transformations
4. Extract positioned text fragments

### 3. Table Detection (Geometric Heuristics)

```
Input: List of TextFragments with (x, y, width, height, text)

Step 1: Line Detection
  - Cluster fragments by Y-coordinate (rows)
  - Cluster fragments by X-coordinate (columns)
  - Detect alignment patterns

Step 2: Grid Construction
  - Find vertical separators (white space or lines)
  - Find horizontal separators
  - Build candidate grid

Step 3: Cell Assignment
  - Assign text fragments to cells
  - Detect merged cells (spanning)
  - Verify table consistency

Step 4: Validation
  - Check regularity score
  - Verify minimum rows/columns
  - Calculate confidence

Output: Table structure with cells
```

### 4. Reading Order Determination

```
Input: List of Elements with bounding boxes

Step 1: Column Detection
  - Detect multi-column layouts
  - Build column boundaries

Step 2: Block Ordering
  - Sort by Y-coordinate within columns
  - Handle overlapping elements
  - Respect Z-order

Step 3: Hierarchical Structure
  - Detect headings (font size, position)
  - Group paragraphs under headings
  - Build document tree

Output: Ordered element list
```

## Performance Considerations

### Memory Management

1. **Streaming**: Process pages one at a time
2. **Object Pooling**: Reuse frequently allocated objects
3. **Lazy Loading**: Load objects on demand
4. **Resource Limits**: Set maximum memory per page

### Optimization Strategies

1. **Incremental Parsing**: Don't parse entire PDF upfront
2. **Parallel Processing**: Process pages in parallel
3. **Caching**: Cache decoded fonts and images
4. **Index Building**: Build index for large documents

### Benchmarking Targets

- Parse 100-page PDF: < 5 seconds
- Extract tables from 1000-page PDF: < 30 seconds
- Memory usage: < 100MB for typical documents
- Concurrent processing: Linear scaling up to CPU count

## Development Roadmap

### Phase 1: MVP (Weeks 1-4)
- [ ] Core PDF object parser
- [ ] XRef table handling
- [ ] Basic stream decoding (Flate)
- [ ] Content stream parser
- [ ] Simple text extraction
- [ ] Basic PDF writer

### Phase 2: Text & Layout (Weeks 5-8)
- [ ] Font handling (Type1, TrueType)
- [ ] Text encoding/decoding
- [ ] Layout analysis
- [ ] Reading order determination
- [ ] Paragraph detection

### Phase 3: Tables (Weeks 9-12)
- [ ] Geometric table detector
- [ ] Line-based table detection
- [ ] Cell extraction
- [ ] Grid reconstruction
- [ ] Table validation

### Phase 4: Advanced Features (Weeks 13-16)
- [ ] Image extraction
- [ ] Form field support
- [ ] Encryption support
- [ ] Advanced PDF writing
- [ ] Metadata handling

### Phase 5: Optimization (Weeks 17-20)
- [ ] Performance optimization
- [ ] Memory profiling
- [ ] Parallel processing
- [ ] Benchmark suite
- [ ] Production hardening

### Phase 6: Extensions (Ongoing)
- [ ] ML-based table detection
- [ ] OCR integration
- [ ] Additional font types
- [ ] PDF/A compliance
- [ ] Annotation support

## Testing Strategy

1. **Unit Tests**: Each package has comprehensive tests
2. **Integration Tests**: End-to-end document processing
3. **Corpus Testing**: Test against diverse PDF corpus
4. **Regression Tests**: Track accuracy over time
5. **Benchmark Tests**: Performance regression detection
6. **Fuzzing**: PDF parser fuzzing for robustness
