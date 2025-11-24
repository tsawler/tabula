# Documentation Index

Welcome to the Go PDF Library documentation. This index helps you navigate all available documentation.

## Getting Started

Start here if you're new to the library:

1. **[GETTING_STARTED.md](GETTING_STARTED.md)** - ⭐ START HERE - Quick start for developers
2. **[IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)** - Step-by-step implementation guide (200+ tasks)
3. **[README.md](README.md)** - Overview, quick start, installation
4. **[examples/basic_usage.go](examples/basic_usage.go)** - Working code examples
5. **[SUMMARY.md](SUMMARY.md)** - Complete design summary

## Architecture & Design

Deep dive into the library architecture:

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Complete system architecture
  - Design principles
  - Package structure
  - Core abstractions
  - Data flow
  - Development roadmap

- **[IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)** - Detailed implementation plan
  - 6 phases broken into weekly tasks
  - 200+ specific tasks with estimates
  - Acceptance criteria for each task
  - Testing and documentation strategy
  - Success metrics and risk management

- **[DIAGRAMS.md](DIAGRAMS.md)** - Visual architecture diagrams
  - System architecture
  - PDF file structure
  - Object graph
  - Content stream processing
  - Table detection pipeline
  - Memory architecture
  - Parallel processing

## Technical Guides

### PDF Internals

- **[PDF_PARSING_GUIDE.md](PDF_PARSING_GUIDE.md)** - Deep dive into PDF parsing
  - PDF file structure
  - Object parsing algorithms
  - XRef table handling
  - Content stream processing
  - Font handling
  - Stream decoding
  - Code examples with implementations

### Performance

- **[PERFORMANCE.md](PERFORMANCE.md)** - Optimization strategies
  - Memory management
  - Streaming architecture
  - Object pooling
  - Lazy loading
  - Parallel processing
  - Caching strategies
  - Benchmarking

### Testing

- **[TESTING.md](TESTING.md)** - Testing strategy
  - Unit tests
  - Integration tests
  - Corpus testing
  - Benchmarks
  - Fuzzing
  - Test coverage
  - CI/CD

### RAG Integration

- **[RAG_INTEGRATION.md](RAG_INTEGRATION.md)** - RAG pipeline integration
  - Semantic element detection
  - Hierarchical structure
  - Table serialization strategies
  - Chunking strategies
  - Metadata enhancement
  - Vector store integration
  - Complete pipeline example

## Code Reference

### Core Model (model/)

- **[model/document.go](model/document.go)** - Document structure
  - Document, Metadata
  - Methods: AddPage, GetPage, ExtractText, ExtractTables

- **[model/page.go](model/page.go)** - Page structure
  - Page with elements
  - Methods: AddElement, ExtractText, ExtractTables, GetElementsInRegion

- **[model/element.go](model/element.go)** - Element types
  - Element interface
  - Paragraph, Heading, List, Image
  - TextStyle, Color

- **[model/table.go](model/table.go)** - Table structure
  - Table with cells
  - Cell with styling
  - TableGrid for detection
  - Methods: ToMarkdown, ToCSV, GetCell, SetCell

- **[model/geometry.go](model/geometry.go)** - Geometric primitives
  - Point, BBox, Matrix
  - Methods: Intersects, Union, Contains, Transform

### PDF Core (core/)

- **[core/object.go](core/object.go)** - PDF object types
  - Object interface
  - Null, Bool, Int, Real, String, Name
  - Array, Dict, Stream, IndirectRef

- **[core/parser.go](core/parser.go)** - PDF syntax parser
  - Parser implementation
  - ParseObject method
  - Type-specific parsing methods

### Table Detection (tables/)

- **[tables/detector.go](tables/detector.go)** - Detector interface
  - Detector interface
  - Config
  - DetectorRegistry

- **[tables/geometric.go](tables/geometric.go)** - Geometric detector
  - GeometricDetector implementation
  - Complete algorithm (900+ lines)
  - Grid construction
  - Confidence scoring

### Examples (examples/)

- **[examples/basic_usage.go](examples/basic_usage.go)** - Usage examples
  - Reading and extracting text
  - Extracting tables
  - Creating PDFs
  - RAG integration

## Quick Reference

### Common Tasks

| Task | Documentation | Code Example |
|------|--------------|--------------|
| Read a PDF | [README.md](README.md#extract-text-from-pdf) | [basic_usage.go](examples/basic_usage.go) |
| Extract tables | [README.md](README.md#extract-tables) | [basic_usage.go](examples/basic_usage.go) |
| Create PDF | [README.md](README.md#create-pdf-from-scratch) | [basic_usage.go](examples/basic_usage.go) |
| RAG integration | [RAG_INTEGRATION.md](RAG_INTEGRATION.md) | [basic_usage.go](examples/basic_usage.go) |
| Table detection | [ARCHITECTURE.md](ARCHITECTURE.md#3-table-detection-geometric-heuristics) | [tables/geometric.go](tables/geometric.go) |
| Optimize performance | [PERFORMANCE.md](PERFORMANCE.md) | - |
| Write tests | [TESTING.md](TESTING.md) | - |

### API Cheat Sheet

```go
// Reading
file, _ := os.Open("doc.pdf")
reader, _ := reader.New(file)
doc, _ := reader.Parse()

// Extracting
text := doc.ExtractText()
tables := doc.ExtractTables()

// Creating
doc := model.NewDocument()
page := model.NewPage(612, 792)
page.AddElement(&model.Heading{Text: "Title"})
doc.AddPage(page)

// Writing
writer.New(file).Write(doc)

// Table operations
for _, table := range tables {
    markdown := table.ToMarkdown()
    csv := table.ToCSV()
    rows := table.RowCount()
    cols := table.ColCount()
}
```

## Documentation by Role

### For Users

If you want to **use the library**:
1. Read [README.md](README.md)
2. Review [examples/basic_usage.go](examples/basic_usage.go)
3. Check [RAG_INTEGRATION.md](RAG_INTEGRATION.md) for RAG workflows

### For Contributors

If you want to **contribute**:
1. Read [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for task breakdown
2. Understand [ARCHITECTURE.md](ARCHITECTURE.md)
3. Study [PDF_PARSING_GUIDE.md](PDF_PARSING_GUIDE.md)
4. Follow [TESTING.md](TESTING.md) for tests
5. Review [PERFORMANCE.md](PERFORMANCE.md) for optimization

### For AI Assistants (Claude)

If you're **helping with implementation**:
1. **Read [CLAUDE.md](CLAUDE.md) first** - Complete project context
2. Check current task in [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)
3. Reference [ARCHITECTURE.md](ARCHITECTURE.md) for design decisions
4. Follow conventions in [CLAUDE.md](CLAUDE.md)

### For Researchers

If you want to **understand the algorithms**:
1. Read [SUMMARY.md](SUMMARY.md) for overview
2. Study [DIAGRAMS.md](DIAGRAMS.md) for visualizations
3. Review [tables/geometric.go](tables/geometric.go) for implementation
4. Check [PDF_PARSING_GUIDE.md](PDF_PARSING_GUIDE.md) for PDF internals

## Project Structure

```
pdf/
├── INDEX.md                  ← You are here
├── CLAUDE.md                 ← Context for AI assistant
├── GETTING_STARTED.md        ← ⭐ START HERE (NEW!)
├── IMPLEMENTATION_PLAN.md    ← Step-by-step plan
├── README.md                 ← User documentation
├── ARCHITECTURE.md           ← System design
├── SUMMARY.md                ← Design summary
├── DIAGRAMS.md               ← Visual diagrams
├── PDF_PARSING_GUIDE.md      ← PDF internals
├── PERFORMANCE.md            ← Optimization
├── TESTING.md                ← Testing strategy
├── RAG_INTEGRATION.md        ← RAG workflows
├── go.mod                    ← Go module
│
├── model/                    ← Intermediate representation
│   ├── document.go
│   ├── page.go
│   ├── element.go
│   ├── table.go
│   └── geometry.go
│
├── core/                     ← PDF primitives
│   ├── object.go
│   └── parser.go
│
├── tables/                   ← Table detection
│   ├── detector.go
│   └── geometric.go
│
└── examples/                 ← Code examples
    └── basic_usage.go
```

## Additional Resources

### External References

- [PDF 1.7 Specification (ISO 32000-1:2008)](https://www.adobe.com/content/dam/acom/en/devnet/pdf/pdfs/PDF32000_2008.pdf)
- [PDF 2.0 Specification (ISO 32000-2:2020)](https://pdfa.org/resource/iso-32000-pdf/)

### Related Projects

- [pdfcpu](https://github.com/pdfcpu/pdfcpu) - PDF processor in Go
- [gofpdf](https://github.com/jung-kurt/gofpdf) - PDF generation library
- [unipdf](https://github.com/unidoc/unipdf) - Commercial PDF library

## FAQ

**Q: Where do I start?**
A: Read [README.md](README.md), then try [examples/basic_usage.go](examples/basic_usage.go)

**Q: How does table detection work?**
A: See [ARCHITECTURE.md](ARCHITECTURE.md#3-table-detection-geometric-heuristics) and [tables/geometric.go](tables/geometric.go)

**Q: How do I integrate with RAG?**
A: Check [RAG_INTEGRATION.md](RAG_INTEGRATION.md)

**Q: What's the performance?**
A: See [PERFORMANCE.md](PERFORMANCE.md) for targets and optimization

**Q: How do I contribute?**
A: Read [ARCHITECTURE.md](ARCHITECTURE.md) and [TESTING.md](TESTING.md)

**Q: How complete is the implementation?**
A: This is a design document with example implementations. See [SUMMARY.md](SUMMARY.md) for roadmap.

## License

MIT License - see LICENSE file for details.

## Support

- GitHub Issues: Report bugs and request features
- GitHub Discussions: Ask questions and share ideas
