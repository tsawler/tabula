# Tabula - Context for Claude

This document provides context for Claude (AI assistant) to effectively help with the Tabula project.

## Project Overview

**Name**: Tabula
**Module**: `github.com/tsawler/tabula`
**Language**: Go 1.21+
**Purpose**: Production-grade PDF parsing and writing library with advanced table extraction for RAG workflows

**Directory Structure**: The project uses Go workspaces. The library code is in the `tabula/` subdirectory, and test applications can be created at the workspace root.

### Key Differentiators
- **Table-preserving**: Detects and extracts table structure (not just text)
- **RAG-optimized**: Intermediate representation designed for LLM ingestion
- **Pure Go**: No CGO dependencies
- **Advanced parsing**: Goes beyond simple PDF generation libraries

## Project Structure

The project uses **Go workspaces** for development:

```
pdf/                                  (workspace root)
├── go.work                           (workspace configuration)
├── README.md                         (workspace readme)
├── .gitignore                        (git ignore)
├── tabula/                           (the library - all code here)
│   ├── go.mod                        (module: github.com/tsawler/tabula)
│   ├── model/                        (document IR)
│   ├── core/                         (PDF primitives)
│   ├── tables/                       (table detection)
│   ├── [other packages...]
│   └── *.md                          (all documentation)
└── example-app/                      (example test app)
    ├── go.mod
    └── main.go
```

**Important**: All library code and documentation is in `tabula/`. Test applications are at the workspace root.

## Current Project State

### What Exists ✅

#### Complete Documentation (12 files in tabula/, ~5,500 lines)
- `GETTING_STARTED.md` - Developer quick start
- `IMPLEMENTATION_PLAN.md` - 200+ tasks, week-by-week breakdown
- `ARCHITECTURE.md` - Complete system architecture
- `SUMMARY.md` - Executive overview
- `DIAGRAMS.md` - Visual architecture diagrams
- `PDF_PARSING_GUIDE.md` - PDF internals deep dive
- `PERFORMANCE.md` - Optimization strategies
- `TESTING.md` - Testing strategy
- `RAG_INTEGRATION.md` - RAG pipeline integration
- `README.md` - User documentation
- `INDEX.md` - Documentation navigation

#### Working Code (10 files in tabula/, ~2,700 lines)
- `tabula/model/document.go` - Complete ✅
- `tabula/model/page.go` - Complete ✅
- `tabula/model/element.go` - Complete (Paragraph, Heading, List, Image, etc.) ✅
- `tabula/model/table.go` - Complete (with ToMarkdown, ToCSV, cell structure) ✅
- `tabula/model/geometry.go` - Complete (BBox, Point, Matrix with operations) ✅
- `tabula/core/object.go` - PDF object types defined ✅
- `tabula/core/parser.go` - Skeleton exists, needs implementation 🚧
- `tabula/tables/detector.go` - Detector interface complete ✅
- `tabula/tables/geometric.go` - **900+ line geometric detector, COMPLETE** ✅
- `tabula/examples/basic_usage.go` - Usage examples ✅

#### Configuration
- `tabula/go.mod` - Module configured for `github.com/tsawler/tabula` ✅
- `go.work` - Workspace configuration at root ✅

### What Needs Building 🚧

See `IMPLEMENTATION_PLAN.md` for the full breakdown. Summary:

**Phase 1 (Weeks 1-4)**: Core PDF parsing
- Complete object parser
- XRef table handling
- Stream decoding (FlateDecode)
- Content stream processing
- Basic text extraction

**Phase 2 (Weeks 5-8)**: Text & layout
- Font handling (Type1, TrueType)
- Text encoding/decoding
- Layout analysis
- Heading/paragraph/list detection

**Phase 3 (Weeks 9-12)**: Tables
- Line/rectangle detection from graphics
- Integration of geometric detector (already implemented!)
- Cell processing refinements
- Export format polishing

**Phase 4 (Weeks 13-16)**: Advanced features
- Image extraction
- Form fields
- Encryption
- Metadata

**Phase 5 (Weeks 17-20)**: Optimization
- Performance profiling
- Memory optimization
- Parallel processing
- Production hardening

**Phase 6 (Ongoing)**: Extensions
- ML-based detection
- OCR integration
- PDF/A compliance

## Architecture Overview

### Layer Structure
```
Application Layer (RAG, CLI, API)
         ↓
High-Level API (model/ - Document IR)
         ↓
Processing Layer (text/, tables/, layout/)
         ↓
Content Layer (contentstream/, font/)
         ↓
PDF Core (core/ - objects, XRef, streams)
         ↓
I/O Layer (file, buffer, compression)
```

### Key Packages

#### `model/` - Intermediate Representation (IR)
The semantic document model that represents the extracted structure.

**Key Types:**
- `Document` - Top-level with pages and metadata
- `Page` - Contains ordered Elements
- `Element` interface - Implemented by Paragraph, Heading, List, Table, Image
- `Table` - Full table structure with cells, rows, columns
- `Cell` - Table cell with text, bbox, row/col span
- `BBox` - Bounding box with geometry operations
- `Point`, `Matrix` - Geometric primitives

**Important:** This is the user-facing API. Keep it clean and well-documented.

#### `core/` - PDF Primitives
Low-level PDF object handling.

**Key Types:**
- `Object` interface - Base for all PDF objects
- `Bool`, `Int`, `Real`, `String`, `Name` - Primitive types
- `Array`, `Dict` - Container types
- `Stream` - Stream objects with compression
- `IndirectRef` - Indirect object references
- `Parser` - Parses PDF syntax into objects
- `XRefTable` - Maps object numbers to file positions

**Current State:** Objects defined, parser skeleton exists, needs implementation.

#### `tables/` - Table Detection
**IMPORTANT:** The geometric detector is **already fully implemented** in `tables/geometric.go` (900+ lines).

**Algorithm (already done!):**
1. Spatial clustering of text fragments
2. Alignment analysis (row/column detection)
3. Grid construction
4. Cell assignment
5. Confidence scoring (regularity + alignment + lines + occupancy)

**What's needed:** Testing, integration, and refinements.

#### `reader/` - PDF Reading
Not yet implemented. Will orchestrate core/ components to read PDFs.

#### `contentstream/` - Content Stream Processing
Not yet implemented. Will parse page content streams to extract text/graphics.

#### `text/` - Text Extraction
Not yet implemented. Will extract and order text from content streams.

#### `font/` - Font Handling
Not yet implemented. Will handle Type1, TrueType, CJK fonts.

#### `layout/` - Layout Analysis
Not yet implemented. Will detect paragraphs, headings, lists, reading order.

## Key Design Decisions

### 1. Intermediate Representation (IR) First
The `model/` package defines the desired output format. All parsing produces this IR. This makes the library easy to use regardless of PDF complexity.

### 2. Layered Architecture
Clear separation between PDF primitives (core/), content processing (contentstream/, text/, font/), and semantic extraction (layout/, tables/).

### 3. Table Detection Strategy
Using geometric heuristics (already implemented!) as primary method. ML-based detection is Phase 6.

**Confidence scoring weights:**
- Grid regularity: 30%
- Alignment quality: 30%
- Line presence: 20%
- Cell occupancy: 20%

### 4. Streaming & Performance
- Lazy-load objects
- Process pages one at a time
- Object pooling for frequent allocations
- Target: 20-50 pages/sec, <100MB memory

### 5. Pure Go
No CGO dependencies. All compression/decompression in pure Go (use stdlib where possible).

## Important Conventions

### Code Style
- Follow standard Go conventions
- Use `go fmt`
- Write godoc comments for all public APIs
- Keep functions small and focused
- Prefer explicit over clever

### Testing
- **TDD**: Write tests before implementation
- Test file naming: `*_test.go` in same package
- Coverage target: >80%
- Benchmark critical paths: `Benchmark*` functions
- Use table-driven tests

### Error Handling
- Return errors, don't panic (except for programmer errors)
- Wrap errors with context: `fmt.Errorf("parsing XRef: %w", err)`
- Provide clear error messages
- Lenient mode vs strict mode for malformed PDFs

### Naming
- PDF terms: Use PDF spec terminology (XRef not Xref, BBox not Bbox)
- Keep acronyms uppercase: `PDF`, `IR`, `RAG`
- Interfaces: Don't use "I" prefix (use `Reader` not `IReader`)
- Constructors: `New*` functions

### File Organization
- One primary type per file
- Group related functions with their type
- Keep files under 1000 lines (split if needed)
- Tests in separate `_test.go` files

## Common Tasks & How to Help

### Task: Implement a new feature

**What I need from you:**
1. Tell me which task from `IMPLEMENTATION_PLAN.md` (e.g., "Task 1.2")
2. Or describe the feature

**What I'll do:**
1. Check current implementation state
2. Review relevant architecture docs
3. Write the code following the plan
4. Include tests
5. Update any affected documentation

**Example request:**
> "Let's implement Task 1.3 - the PDF lexer/tokenizer"

### Task: Debug an issue

**What I need from you:**
1. Error message or unexpected behavior
2. Input that causes the issue (if applicable)
3. Expected vs actual behavior

**What I'll do:**
1. Analyze the code
2. Identify the root cause
3. Propose and implement a fix
4. Add test to prevent regression

### Task: Review code

**What I need from you:**
1. File(s) to review
2. Specific concerns (performance, correctness, style)

**What I'll do:**
1. Review against our conventions
2. Check for bugs/edge cases
3. Suggest improvements
4. Verify tests exist

### Task: Write tests

**What I need from you:**
1. What to test (function/package/feature)

**What I'll do:**
1. Write comprehensive test cases
2. Include edge cases
3. Use table-driven tests
4. Add benchmarks if performance-critical

### Task: Update documentation

**What I need from you:**
1. What changed
2. Which docs to update

**What I'll do:**
1. Update relevant markdown files
2. Update code comments/godocs
3. Update examples if needed
4. Keep docs consistent

## Implementation Guidelines

### When implementing Phase 1 tasks:

**Parser (core/parser.go)**
- Reference `PDF_PARSING_GUIDE.md` for algorithms
- Handle all 8 object types
- Support nested structures
- Handle whitespace and comments correctly
- Test with real PDF snippets

**XRef (core/xref.go)**
- Support traditional XRef tables
- Support XRef streams (PDF 1.5+)
- Handle incremental updates
- Lazy-load objects

**Stream Decoding (core/stream.go)**
- Start with FlateDecode (most common)
- Support filter chains
- Handle DecodeParms
- Add other filters incrementally

**Content Streams (contentstream/parser.go)**
- Parse operators and operands
- Don't interpret yet (just parse)
- Build operation list
- Reference PDF_PARSING_GUIDE.md Section 3

### When implementing Phase 2 tasks:

**Font Handling (font/)**
- Start with Standard 14 fonts (hardcoded metrics)
- Then Type1 (more common)
- Then TrueType (more complex)
- CJK fonts last
- Reference PDF spec extensively

**Text Extraction (text/extractor.go)**
- Track graphics state (CTM, text matrix)
- Calculate accurate positions
- Build TextFragment list
- Don't worry about ordering yet

**Layout Analysis (layout/)**
- Start with simple block detection
- Use spatial clustering
- Then add paragraph detection
- Heading detection uses font size + position
- Reading order is last

### When working with tables:

**IMPORTANT:** `tables/geometric.go` is already complete!

**What's needed:**
1. Testing the existing implementation
2. Integration with page processing
3. Refinements based on real PDFs
4. Performance optimization

**Don't rewrite the detector** - it's a 900-line complete implementation.

## Important Files to Reference

### For implementation:
- `IMPLEMENTATION_PLAN.md` - Task-by-task breakdown
- `PDF_PARSING_GUIDE.md` - Algorithms and examples
- `ARCHITECTURE.md` - Design decisions

### For understanding:
- `SUMMARY.md` - Quick overview
- `DIAGRAMS.md` - Visual architecture

### For APIs:
- `model/*.go` - The user-facing types
- `examples/basic_usage.go` - How users will use it

### For patterns:
- `tables/geometric.go` - Example of complete implementation
- `model/geometry.go` - Example of math utilities

## PDF Specification References

When implementing PDF features, always reference:

**PDF 1.7 Specification (ISO 32000-1:2008)**
- Syntax: Section 7.2-7.3
- Objects: Section 7.3
- Streams: Section 7.3.8
- XRef: Section 7.5.4
- Content Streams: Section 7.8.2
- Text: Section 9.4
- Fonts: Section 9.6-9.9

**Key PDF Concepts:**
- **Object numbering**: Objects have number + generation (e.g., "5 0 obj")
- **Indirect references**: "5 0 R" refers to object 5
- **Streams**: Dict + binary data
- **Content streams**: Commands like "BT", "Tf", "Tj", "ET"
- **Coordinate system**: Origin at bottom-left, Y increases upward
- **Text matrix**: Transforms text positioning
- **Graphics state**: Stack-based (q/Q save/restore)

## Testing Strategy

### Unit Tests
- Test each function/method
- Use table-driven tests
- Test edge cases
- Mock dependencies

### Integration Tests
- Test complete workflows
- Use real PDF samples
- Test error handling
- Test with diverse PDFs

### Corpus Testing
- Maintain test PDF collection
- Track accuracy metrics
- Test against 100+ PDFs
- Include edge cases and malformed PDFs

### Benchmarks
- Benchmark critical paths
- Track performance over time
- No regressions allowed
- Target metrics in IMPLEMENTATION_PLAN.md

## How to Work Together Effectively

### When starting a new session:

**You should tell me:**
1. What task you want to work on
2. Any issues you're facing
3. What you've tried

**I'll:**
1. Read relevant context files
2. Check current implementation state
3. Propose a solution
4. Implement with tests

### Best practices:

✅ **Do:**
- Reference specific tasks from IMPLEMENTATION_PLAN.md
- Provide error messages and context
- Ask me to explain architecture decisions
- Request code reviews before moving on
- Ask me to write tests

❌ **Don't:**
- Skip tasks in the implementation plan
- Ask me to rewrite existing complete code (like geometric detector)
- Skip tests "for now"
- Forget to update documentation

### Questions I can answer:

- "How should I implement X?" (I'll check the plan and architecture)
- "What's the current state of package Y?" (I'll check the files)
- "Is this approach correct?" (I'll review against our architecture)
- "Why did we design it this way?" (I'll check ARCHITECTURE.md)
- "What does the PDF spec say about X?" (I'll reference PDF_PARSING_GUIDE.md)
- "How do I test this?" (I'll reference TESTING.md and write tests)

### What I need from you:

- **Be specific**: "Implement Task 1.2" vs "help with objects"
- **Show context**: Error messages, file names, what you tried
- **One task at a time**: Don't jump between phases
- **Tell me when done**: So I can mark tasks complete

## Current Priority

**Next task:** Phase 1, Task 1.2 - Implement PDF Object tests and helper methods

**Current phase:** Phase 1 - MVP (Core PDF parsing)

**Current milestone:** None yet (starting from scratch on implementation)

**Test coverage:** 0% (no implementation yet)

## Project Statistics

- **Total files**: 22 (11 docs, 10 code, 1 config)
- **Documentation**: ~5,000 lines
- **Code**: ~2,700 lines
- **Total**: ~7,700 lines
- **Implementation progress**: ~5% (models done, detector done, rest pending)

## Key Success Metrics

The implementation plan defines success metrics for each phase. Track:

- **Phase 1**: Parse 90%+ of simple PDFs, extract text
- **Phase 2**: Detect paragraphs/headings with 80%+ accuracy
- **Phase 3**: Table detection 80% precision, 70% recall
- **Phase 4**: Successfully handle images, forms, encryption
- **Phase 5**: 20-50 pages/sec, <100MB memory
- **Phase 6**: ML improves accuracy by 10%+

## Version Roadmap

- **v0.1.0**: Phase 1 complete (MVP - basic parsing)
- **v0.2.0**: Phase 2 complete (layout analysis)
- **v0.3.0**: Phase 3 complete (table extraction)
- **v0.4.0**: Phase 4 complete (advanced features)
- **v1.0.0**: Phase 5 complete (production-ready)
- **v1.x.x**: Phase 6 (extensions)

## Quick Command Reference

```bash
# Run tests
go test ./...

# Run specific package tests
go test ./core

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Format code
go fmt ./...

# Lint (if installed)
golangci-lint run

# Build examples
go build ./examples/...

# Check module
go mod tidy
go mod verify
```

## Communication Tips

When asking for help, include:

1. **Task reference**: "Working on Task 1.3 from IMPLEMENTATION_PLAN.md"
2. **Current state**: "I've implemented X, now stuck on Y"
3. **Question**: Specific question or request
4. **Context**: Error messages, code snippets if relevant

Example good request:
> "I'm working on Task 1.3 - implementing the lexer. I've written the NextToken() method but I'm not sure how to handle PDF comments correctly. Can you help implement comment skipping?"

Example less helpful request:
> "Help me with the parser"

## File Modification Guidelines

### Files you can freely modify:
- Any file in `tabula/core/`, `tabula/reader/`, `tabula/writer/`, etc.
- Test files (`*_test.go`)
- `tabula/examples/` (add new examples)
- Test applications at workspace root (e.g., `example-app/`)

### Files to modify carefully:
- `tabula/model/*.go` - This is the public API, changes affect users
- `tabula/tables/geometric.go` - Already complete, only add tests or refinements
- `tabula/go.mod` - Only add necessary dependencies

### Files to leave alone (unless explicitly updating):
- All `.md` documentation files in `tabula/`
- `tabula/examples/basic_usage.go` (reference implementation)
- `go.work` at workspace root (unless adding new modules)

## Remember

1. **The geometric table detector is done** - 900 lines, fully implemented
2. **Follow IMPLEMENTATION_PLAN.md** - It's detailed for a reason
3. **TDD always** - Write tests before implementation
4. **Document as you go** - Update docs when you change behavior
5. **One task at a time** - Complete before moving to next
6. **Ask questions** - I have full context via this file

## Final Note

This project is ambitious but well-planned. The architecture is solid, the plan is detailed, and key algorithms (like table detection) are already implemented. Success comes from systematically working through the implementation plan, writing comprehensive tests, and maintaining the high quality standards we've established in the design.

**Let's build something great together!** 🚀
