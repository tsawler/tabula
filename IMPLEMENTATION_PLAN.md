# Tabula Implementation Plan

This document provides a step-by-step implementation plan to build the Tabula library from the ground up.

## Overview

This plan breaks down the 6-phase roadmap into specific, actionable tasks with clear acceptance criteria and estimated effort.

## RAG-First Philosophy 🎯

Tabula is designed specifically for RAG (Retrieval-Augmented Generation) workflows. Unlike general PDF libraries, we prioritize:

1. **Semantic Chunking** (Phase 2.5) - Never break sentences, lists, or tables mid-thought
2. **Multi-Column Detection** (Task 2.6) - Academic papers and reports read correctly
3. **Header/Footer Filtering** (Task 2.7) - Remove repetitive noise that pollutes embeddings
4. **Table Structure Preservation** (Phase 3) - Tables as structured data, not garbled text
5. **Figure-Caption Association** (Task 4.4) - Capture visual context in text chunks
6. **Heading-Aware Chunking** - Every chunk knows its section context
7. **Math Content Detection** (Task 4.6) - Flag and preserve mathematical notation
8. **List Coherence** - Keep list intros with items

Tasks marked 🎯 are **RAG CRITICAL** - they directly impact embedding quality and retrieval accuracy.

---

## Phase 1: MVP - Core PDF Parsing (4 weeks)

**Goal**: Read basic PDFs and extract raw text

### Week 1: Foundation & Object Parsing

#### Task 1.1: Project Setup (2 hours) ✅ COMPLETE
- [x] Initialize git repository
- [x] Create package structure
- [x] Set up go.mod with dependencies
- [x] Create basic README

**Deliverable**: Working Go module structure ✅
**Completed**: November 24, 2024

#### Task 1.2: PDF Object Implementation (8 hours) ✅ COMPLETE
- [x] Implement all object types in `core/object.go`
  - Bool, Int, Real, String, Name
  - Array, Dict, Stream, IndirectRef
- [x] Add helper methods on Dict (Get, GetName, GetInt, etc.)
- [x] Write unit tests for each object type

**Deliverable**: Complete object model with tests ✅
**Acceptance**: All object types parse correctly ✅
**Completed**: November 24, 2024
**Tests**: 60+ test cases, all passing
**Coverage**: 80-100% on object.go

#### Task 1.3: Lexer/Tokenizer (12 hours) ✅ COMPLETE
- [x] Implement buffered PDF reader in `core/lexer.go`
- [x] Implement tokenization (skipWhitespace, NextToken, etc.)
- [x] Handle PDF comments (%)
- [x] Handle different newline formats (\r, \n, \r\n)
- [x] Write comprehensive tokenizer tests

**Deliverable**: Robust tokenizer ✅
**Acceptance**: Can tokenize any valid PDF syntax ✅
**Completed**: November 24, 2024
**Tests**: 90+ test cases, all passing
**Coverage**: 80-100% on lexer.go
**Performance**: 2.4M ops/sec for simple tokens, 700K ops/sec for realistic PDF

#### Task 1.4: Object Parser (16 hours) ✅ COMPLETE
- [x] Complete `core/parser.go` implementation
- [x] parseBool, parseNumber, parseString
- [x] parseHexString, parseName
- [x] parseArray, parseDict
- [x] Handle indirect references (num gen R)
- [x] Write parser tests for all object types
- [x] Test with nested structures

**Deliverable**: Working PDF object parser ✅
**Acceptance**: Can parse all 8 PDF object types ✅
**Completed**: November 24, 2024
**Tests**: 60+ test cases, all passing
**Coverage**: 70-90% on parser.go, 82.4% overall
**Performance**: 2.8M simple objects/sec, 1M arrays/sec, 870K dicts/sec

### Week 2: XRef & File Structure

#### Task 1.5: XRef Table Parsing (12 hours) ✅ COMPLETE
- [x] Implement `core/xref.go`
- [x] Parse traditional XRef table
- [x] Parse trailer dictionary
- [x] Handle multiple XRef sections
- [x] Find XRef by scanning from EOF
- [x] Write XRef parser tests

**Deliverable**: XRef table parser ✅
**Acceptance**: Can parse XRef tables from real PDFs ✅
**Completed**: November 24, 2024
**Tests**: 15 test functions, 25+ test cases, all passing
**Coverage**: 75-90% on xref.go, 79.5% overall
**Performance**: 120K XRef tables/sec, 4.9M FindXRef ops/sec

#### Task 1.6: File Reader (8 hours) ✅ COMPLETE
- [x] Implement `reader/reader.go`
- [x] Parse PDF header (%PDF-x.y)
- [x] Load XRef table
- [x] Load trailer
- [x] Resolve indirect references
- [x] Write file reader tests

**Deliverable**: Basic PDF file reader ✅
**Acceptance**: Can open and parse PDF structure ✅
**Completed**: November 24, 2024
**Tests**: 18 test functions, 25+ test cases, all passing
**Coverage**: 79.1%

#### Task 1.7: Object Resolution (8 hours) ✅ COMPLETE
- [x] Implement lazy object loading
- [x] Cache resolved objects
- [x] Handle circular references
- [x] Add object resolver tests

**Deliverable**: Object graph traversal ✅
**Acceptance**: Can resolve any object in PDF ✅
**Completed**: November 24, 2024
**Tests**: 16 test functions, 20+ test cases, all passing
**Coverage**: 86.6%

#### Task 1.8: Catalog & Pages (8 hours) ✅ COMPLETE
- [x] Parse document catalog
- [x] Parse pages tree
- [x] Implement page enumeration
- [x] Get page by number
- [x] Write page access tests

**Deliverable**: Page access API ✅
**Acceptance**: Can enumerate all pages ✅
**Completed**: November 24, 2024
**Tests**: 18 test functions, 20+ test cases, all passing
**Coverage**: 72.1%

### Week 3: Stream Decoding & Content Streams

#### Task 1.9: FlateDecode (12 hours) ✅ COMPLETE
- [x] Implement `internal/filters/flate.go`
- [x] Handle PNG predictors (None, Sub, Up, Average, Paeth)
- [x] Handle TIFF Predictor 2
- [x] Support DecodeParms
- [x] Write decompression tests
- [x] Test with real PDF streams

**Deliverable**: FlateDecode support ✅
**Acceptance**: Can decompress Flate-encoded streams ✅
**Completed**: November 24, 2024
**Tests**: 26 test cases, all passing
**Coverage**: 94.5%

#### Task 1.10: Stream Decoder (8 hours) ✅ COMPLETE
- [x] Implement `core/stream.go` Decode() method
- [x] Handle Filter array (multiple filters)
- [x] Support ASCIIHexDecode, ASCII85Decode
- [x] Support filter chains with DecodeParms
- [x] Write stream decoder tests

**Deliverable**: Multi-filter stream decoder ✅
**Acceptance**: Can decode common PDF streams ✅
**Completed**: November 24, 2024
**Tests**: All stream decoder tests passing

#### Task 1.11: Content Stream Parser (16 hours) ✅ COMPLETE
- [x] Implement `contentstream/parser.go`
- [x] Parse operators and operands
- [x] Handle all operator types (text, graphics, path)
- [x] Build operation list
- [x] Write content stream parser tests

**Deliverable**: Content stream parser ✅
**Acceptance**: Can parse page content streams ✅
**Completed**: November 24, 2024
**Tests**: 19 test cases, all passing (100%)
**Lines**: 571 lines implementation, 457 lines tests

### Week 4: Text Extraction

#### Task 1.12: Graphics State Machine (12 hours) ✅ COMPLETE
- [x] Implement `graphicsstate/state.go`
- [x] GraphicsState struct
- [x] State stack (q/Q operators)
- [x] CTM tracking (cm operator)
- [x] Text state tracking (Tf, Tm, Td, etc.)
- [x] Write state machine tests

**Deliverable**: Graphics state machine ✅
**Acceptance**: Correctly tracks all state changes ✅
**Completed**: November 24, 2024
**Tests**: 25 test cases, all passing (100%)
**Lines**: 284 lines implementation, 472 lines tests

#### Task 1.13: Simple Font Support (8 hours) ✅ COMPLETE
- [x] Implement `font/font.go`
- [x] Basic font metrics
- [x] Standard 14 fonts (hardcoded accurate metrics)
- [x] Character width calculations
- [x] Write font tests

**Deliverable**: Basic font support ✅
**Acceptance**: Can handle standard fonts ✅
**Completed**: November 24, 2024
**Tests**: 10 test cases, all passing (100%)
**Lines**: 461 lines implementation, 159 lines tests

#### Task 1.14: Text Extraction (16 hours) ✅ COMPLETE
- [x] Implement `text/extractor.go`
- [x] Process text operators (Tj, TJ, ', ")
- [x] Calculate text positions with CTM
- [x] Create TextFragment list
- [x] Extract text with font and position info
- [x] Write text extraction tests

**Deliverable**: Text extraction engine ✅
**Acceptance**: Can extract text from simple PDFs ✅
**Completed**: November 24, 2024
**Tests**: 13 test cases, all passing (100%)
**Lines**: 334 lines implementation, 330 lines tests

#### Task 1.15: Integration & Testing (8 hours) ✅ COMPLETE
- [x] End-to-end test: content stream → text extraction
- [x] Comprehensive integration tests (8 test scenarios)
- [x] Test compressed streams (FlateDecode)
- [x] Test complex layouts with multiple fonts
- [x] Test graphics state management
- [x] All tests passing (227 total tests)

**Deliverable**: MVP release ✅
**Acceptance**: Can extract text from content streams ✅
**Completed**: November 24, 2024
**Tests**: 227 tests passing across all packages
**Integration Tests**: 8 comprehensive scenarios in text/integration_test.go

---

## Phase 1 Complete! 🎉

**Status**: ✅ All 15 tasks completed
**Duration**: Approximately 4 weeks as planned
**Completion Date**: November 24, 2024

### Phase 1 Summary

**What We Built:**
- Complete PDF file reading infrastructure (lexer, parser, XRef, resolver)
- Stream decoding with FlateDecode and PNG/TIFF predictors
- Content stream parser (all operators)
- Graphics state machine (CTM, text state, state stack)
- Basic font support (Standard 14 fonts with accurate metrics)
- Text extraction engine with position tracking

**Test Coverage:**
- **227 tests** passing across all packages
- **8 comprehensive integration tests**
- Coverage ranges from 72% to 95% across modules
- All components tested in isolation and integration

**Lines of Code:**
- Implementation: ~5,000 lines
- Tests: ~4,000 lines
- Documentation: ~8,000 lines
- **Total**: ~17,000 lines

**Key Achievements:**
- ✅ Can parse PDF file structure
- ✅ Can decode compressed streams (FlateDecode + predictors)
- ✅ Can parse content stream operations
- ✅ Can track graphics and text state accurately
- ✅ Can extract text with positions and font information
- ✅ Production-ready foundation for Phase 2

**What's Next:**
Phase 2 will add advanced text extraction with layout preservation, including:
- Advanced font support (Type1, TrueType, CIDFont)
- Text encoding and Unicode handling
- Layout analysis (paragraphs, columns, reading order)
- Header/footer detection

---

## Phase 2: Text & Layout (4 weeks)

**Goal**: Advanced text extraction with layout preservation

### Week 5: Font Handling & International Text

#### Task 2.1: Type1 Font Parser (16 hours) ✅
- [x] Implement `font/type1.go`
- [x] Parse Type1 font dictionary
- [x] Load font metrics
- [x] Handle font encoding
- [x] Character width calculation
- [x] Write Type1 tests

#### Task 2.2: TrueType Font Parser (16 hours) ✅
- [x] Implement `font/truetype.go`
- [x] Parse TrueType font program
- [x] Extract glyph metrics
- [x] Handle Unicode mapping
- [x] Write TrueType tests

#### Task 2.3: CIDFont/Type0 Support (16 hours) 🎯 RAG CRITICAL
- [ ] Implement `font/cidfont.go`
- [ ] Parse Type0 (composite) fonts
- [ ] Parse CIDFont dictionaries
- [ ] Support Identity-H encoding (horizontal CJK)
- [ ] Support Identity-V encoding (vertical CJK)
- [ ] Handle CJK character collections (Adobe-GB1, Adobe-Japan1, etc.)
- [ ] Write CIDFont tests with Chinese, Japanese, Korean samples

**RAG Impact**: CJK languages are critical for international RAG applications. Type0/CIDFonts are the standard way PDFs encode Chinese, Japanese, and Korean text.

#### Task 2.4: ToUnicode CMap Parsing (16 hours) 🎯 RAG CRITICAL
- [ ] Implement `font/tounicode.go`
- [ ] Parse ToUnicode CMap streams
- [ ] Handle bfchar and bfrange mappings
- [ ] Support multi-byte character codes
- [ ] **Fallback strategies when ToUnicode missing:**
  - [ ] Use font name heuristics (e.g., "Arial-Unicode")
  - [ ] Common encoding inference
  - [ ] Symbol font mapping tables
- [ ] **Emoji sequence handling:**
  - [ ] Multi-codepoint emoji (👨‍👩‍👧‍👦)
  - [ ] Skin tone modifiers (👋🏽)
  - [ ] ZWJ (Zero Width Joiner) sequences
- [ ] Write ToUnicode tests
- [ ] Test with emoji-heavy PDFs

**RAG Impact**: ToUnicode CMaps are essential for correct character mapping. Fallback strategies ensure we extract readable text even from poorly-formed PDFs. Emoji support is critical for modern documents.

#### Task 2.5: Text Encoding/Decoding (12 hours) 🎯 RAG CRITICAL
- [ ] Implement `text/encoding.go`
- [ ] WinAnsiEncoding (Western European)
- [ ] MacRomanEncoding (Mac legacy)
- [ ] PDFDocEncoding (PDF default)
- [ ] StandardEncoding (Type1 default)
- [ ] Custom encodings via /Differences
- [ ] UTF-16BE for PDF string objects
- [ ] **Unicode normalization to NFC:**
  - [ ] Use Go's `unicode/norm` package
  - [ ] Normalize after extraction
  - [ ] Ensure é (U+00E9) vs e+́ (U+0065+U+0301) → always U+00E9
- [ ] **Vertical text support:**
  - [ ] Detect Identity-V encoding
  - [ ] Handle vertical writing mode
- [ ] Write encoding tests
- [ ] Test normalization with accented characters

**RAG Impact**: Unicode normalization (NFC) ensures embedding consistency. Without it, "café" might be encoded two different ways (precomposed vs combining), causing identical text to have different embeddings and breaking semantic search.

#### Task 2.5b: RTL and Bidirectional Text (8 hours) 🎯 RAG CRITICAL
- [ ] Implement `text/bidi.go`
- [ ] Detect RTL text runs (Arabic, Hebrew)
- [ ] Preserve logical character order (for RAG embedding)
- [ ] Mark text direction in TextFragment (LTR/RTL/Vertical)
- [ ] Handle mixed LTR/RTL paragraphs
- [ ] Reference Unicode BiDi Algorithm (UAX #9) - simplified implementation
- [ ] Write RTL tests with Arabic and Hebrew samples
- [ ] Test mixed-direction text

**RAG Impact**: Arabic and Hebrew documents read right-to-left. Preserving logical order ensures embeddings capture correct meaning, and direction markers allow downstream processing to reconstruct proper reading order if needed.

### Week 6: Advanced Text Extraction & Layout

#### Task 2.6: Enhanced Text Extractor (12 hours)
- [ ] Improve text positioning accuracy
- [ ] Handle character spacing
- [ ] Handle word spacing
- [ ] Handle text rise
- [ ] Write positioning tests

#### Task 2.7: Text Fragment Ordering (8 hours)
- [ ] Sort fragments by position
- [ ] Detect basic reading order
- [ ] Handle RTL text ordering
- [ ] Handle vertical text ordering
- [ ] Write ordering tests

#### Task 2.8: Symbol and Emoji Font Handling (8 hours) 🎯 RAG CRITICAL
- [ ] Implement `font/symbol.go`
- [ ] Symbol font mapping (Wingdings → Unicode, Symbol → Unicode, Dingbats → Unicode)
- [ ] Emoji font detection and extraction
- [ ] Font fallback when emoji not embedded
- [ ] PUA (Private Use Area) character handling
- [ ] ActualText override support (PDF tagged content)
- [ ] Write symbol/emoji tests
- [ ] Test with Wingdings, emoji, and special character PDFs

**RAG Impact**: Modern documents increasingly use emoji and symbol fonts. Without proper mapping, these appear as garbled characters or missing content in embeddings, losing important semantic information.

#### Task 2.9: Multi-Column Layout Detection (12 hours) 🎯 RAG CRITICAL
- [ ] Implement `layout/columns.go`
- [ ] Detect column boundaries via X-coordinate clustering
- [ ] Detect column boundaries via whitespace gaps
- [ ] Handle 2, 3, and N-column layouts
- [ ] Preserve reading order within columns
- [ ] Write column detection tests
- [ ] Test with academic papers and reports

**RAG Impact**: Multi-column PDFs are common in academic papers. Without proper column detection, text extraction jumbles columns together, creating incoherent chunks that destroy RAG quality.

#### Task 2.10: Header/Footer Detection (12 hours) 🎯 RAG CRITICAL
- [ ] Implement `layout/header_footer.go`
- [ ] Detect repeating text at same positions across pages
- [ ] Detect page numbers (numeric patterns at consistent positions)
- [ ] Mark header/footer regions for exclusion
- [ ] Provide option to filter headers/footers from output
- [ ] Write header/footer detection tests
- [ ] Test with real documents (reports, books, papers)

**RAG Impact**: Headers, footers, and page numbers pollute embeddings with repetitive noise. Every chunk containing "Page 1", "Page 2", etc. or the same header text introduces irrelevant data that degrades semantic search quality.

### Week 7: Layout Analysis

#### Task 2.11: Block Detection (12 hours)
- [ ] Implement `layout/block.go`
- [ ] Group fragments into blocks
- [ ] Detect block boundaries
- [ ] Write block detection tests

#### Task 2.12: Line Detection (8 hours)
- [ ] Implement `layout/line.go`
- [ ] Group fragments into lines
- [ ] Detect line spacing
- [ ] Write line detection tests

#### Task 2.13: Paragraph Detection (12 hours)
- [ ] Implement `layout/paragraph.go`
- [ ] Group lines into paragraphs
- [ ] Detect paragraph breaks
- [ ] Handle indentation
- [ ] Write paragraph tests

#### Task 2.14: Reading Order (12 hours)
- [ ] Implement `layout/reading_order.go`
- [ ] Integrate multi-column detection
- [ ] Z-order sorting within columns
- [ ] Cross-column ordering
- [ ] Handle RTL and vertical text ordering
- [ ] Write reading order tests

### Week 8: Heading & List Detection

#### Task 2.15: Heading Detection (12 hours) 🎯 RAG IMPORTANT
- [ ] Implement `layout/heading.go`
- [ ] Detect by font size (larger than body text)
- [ ] Detect by font weight (bold)
- [ ] Detect by position (start of section)
- [ ] Assign heading levels (H1, H2, H3, etc.)
- [ ] Write heading tests

**RAG Impact**: Headings define semantic sections. Detecting them enables hierarchical chunking where each chunk knows its section context, dramatically improving retrieval relevance.

#### Task 2.16: List Detection (12 hours) 🎯 RAG IMPORTANT
- [ ] Implement `layout/list.go`
- [ ] Detect bullet points (•, ◦, ▪, -, *)
- [ ] Detect numbering (1., a., i., etc.)
- [ ] Handle nested lists
- [ ] Preserve list structure in chunks
- [ ] Ensure entire list stays in one chunk when possible
- [ ] Write list tests

**RAG Impact**: Breaking list items across chunks loses the logical grouping. A bulleted list of features or steps must stay together for context preservation.

#### Task 2.17: Layout Analyzer (16 hours)
- [ ] Implement `layout/analyzer.go`
- [ ] Orchestrate all detection
- [ ] Build element tree
- [ ] Assign element types
- [ ] Write integration tests

#### Task 2.18: Phase 2 Integration (8 hours)
- [ ] Update Document model
- [ ] Update Page model
- [ ] Add Elements to pages
- [ ] Test with complex PDFs
- [ ] Document Phase 2 API

---

## Phase 2.5: RAG Optimization & Semantic Chunking (2 weeks) 🎯 RAG CRITICAL

**Goal**: Implement intelligent, context-aware chunking specifically for RAG workflows

**Why this matters**: Fixed-size character chunking destroys semantic meaning. Breaking a sentence mid-thought, separating a list from its intro, or splitting a table caption from its table creates useless embeddings. This phase makes chunking RAG-native.

### Week 8.5-9: Semantic Chunking Strategy

#### Task 2.5.1: Hierarchical Chunking Framework (16 hours) 🎯 RAG CRITICAL
- [ ] Implement `rag/chunker.go`
- [ ] Define chunking hierarchy:
  - Level 1: Document (entire PDF)
  - Level 2: Section (by headings)
  - Level 3: Paragraph
  - Level 4: Sentence (only if paragraph too large)
- [ ] Implement chunk boundary detection at each level
- [ ] Preserve parent-child relationships (section → paragraphs)
- [ ] Add metadata to chunks (section title, page number, position)
- [ ] Write chunking framework tests

**RAG Impact**: This is THE most important feature for RAG quality. Hierarchical chunking ensures chunks have complete thoughts, not sentence fragments.

#### Task 2.5.2: Context-Aware Chunk Boundaries (12 hours) 🎯 RAG CRITICAL
- [ ] Implement smart boundary detection
- [ ] Never break within:
  - A sentence
  - A list (keep intro + items together)
  - A table
  - A figure caption
  - A code block
- [ ] Prefer boundaries at:
  - Paragraph breaks
  - Section breaks (headings)
  - List endings
- [ ] Implement "look ahead" to avoid orphaned content
- [ ] Write boundary detection tests

**RAG Impact**: Avoids the #1 chunking mistake - breaking semantic units mid-thought.

#### Task 2.5.3: Chunk Overlap Strategy (8 hours)
- [ ] Implement configurable overlap
- [ ] Sentence-level overlap (not character-level)
- [ ] Preserve complete sentences in overlap regions
- [ ] Ensure overlaps don't break semantic boundaries
- [ ] Write overlap tests

#### Task 2.5.4: Chunk Metadata & Context (12 hours) 🎯 RAG CRITICAL
- [ ] Add rich metadata to each chunk:
  - Document title
  - Section heading (full path: H1 → H2 → H3)
  - Page number(s)
  - Chunk position in document
  - Element types contained (text, table, list, etc.)
  - Estimated token count
- [ ] Implement context injection (prepend section heading to chunk text)
- [ ] Write metadata tests

**RAG Impact**: Metadata enables filtering ("only search in section X") and context injection improves retrieval by including the section heading in the chunk.

#### Task 2.5.5: List & Enumeration Coherence (8 hours) 🎯 RAG IMPORTANT
- [ ] Detect list intros ("The following features:", "Steps:")
- [ ] Keep list intro with list items in same chunk
- [ ] Preserve list numbering/bullets
- [ ] Handle nested lists correctly
- [ ] Avoid breaking lists mid-item
- [ ] Write list coherence tests

**RAG Impact**: A list without its intro is meaningless. "1. Item one 2. Item two" without "Features:" loses all context.

#### Task 2.5.6: Table & Figure Chunk Handling (8 hours)
- [ ] Treat tables as atomic chunks (don't split)
- [ ] Include table caption in table chunk
- [ ] Include figure caption in figure chunk
- [ ] Option to create separate chunks for large tables
- [ ] Write table/figure chunking tests

#### Task 2.5.7: Chunk Size Configuration (8 hours)
- [ ] Implement flexible size targets:
  - By character count
  - By token count (estimated)
  - By semantic units (paragraphs/sections)
- [ ] Soft limits (prefer not to exceed)
- [ ] Hard limits (must not exceed)
- [ ] Write configuration tests

#### Task 2.5.8: RAG Export Formats (12 hours) 🎯 RAG CRITICAL
- [ ] Implement `rag/export.go`
- [ ] Export to JSON Lines (one chunk per line)
- [ ] Export to CSV (chunk text + metadata columns)
- [ ] Export to Parquet (efficient columnar storage)
- [ ] Include all metadata fields
- [ ] Write export tests
- [ ] Create example notebooks (how to load into vector DB)

**RAG Impact**: Users need chunks in formats ready for embedding and vector DB ingestion.

#### Task 2.5.9: Integration & Testing (8 hours)
- [ ] Integrate with Document model
- [ ] Add Chunk() method to Document
- [ ] Test with diverse PDFs:
  - Academic papers (multi-column, equations)
  - Technical reports (tables, lists, code)
  - Books (chapters, long text)
  - Forms (structured data)
- [ ] Document chunking API
- [ ] Create usage examples

---

## Phase 3: Table Detection (4 weeks)

**Goal**: Detect and extract tables with high accuracy

### Week 9: Line & Rectangle Detection

#### Task 3.1: Graphics Operator Processing (12 hours)
- [ ] Extend graphics state machine
- [ ] Track path construction (m, l, c, v, y)
- [ ] Track rectangles (re)
- [ ] Track stroke/fill operators (S, s, f, F, B)
- [ ] Write graphics tests

#### Task 3.2: Line Extraction (8 hours)
- [ ] Implement line detection
- [ ] Store Line objects
- [ ] Classify horizontal/vertical
- [ ] Write line extraction tests

#### Task 3.3: Rectangle Extraction (8 hours)
- [ ] Detect rectangles from paths
- [ ] Classify filled/stroked
- [ ] Write rectangle tests

#### Task 3.4: Grid Line Detection (8 hours)
- [ ] Detect aligned lines
- [ ] Build grid hypothesis
- [ ] Write grid detection tests

### Week 10: Geometric Table Detector (Already Implemented!)

#### Task 3.5: Code Review & Refinement (8 hours)
- [ ] Review `tables/geometric.go` (already written!)
- [ ] Add missing edge cases
- [ ] Optimize performance
- [ ] Add inline documentation

#### Task 3.6: Testing (16 hours)
- [ ] Create test fixtures (PDFs with tables)
- [ ] Write unit tests for each method
- [ ] Test grid construction
- [ ] Test confidence scoring
- [ ] Test merged cell detection

#### Task 3.7: Validation (8 hours)
- [ ] Test on diverse table types
- [ ] Measure precision/recall
- [ ] Tune parameters
- [ ] Document accuracy

#### Task 3.8: Configuration (8 hours)
- [ ] Expose configuration options
- [ ] Add detector registry tests
- [ ] Write configuration docs

### Week 11: Cell Processing

#### Task 3.9: Cell Text Assembly (12 hours)
- [ ] Improve fragment-to-cell assignment
- [ ] Handle multi-line cells
- [ ] Preserve text order in cells
- [ ] Write cell assembly tests

#### Task 3.10: Cell Styling (8 hours)
- [ ] Detect cell borders
- [ ] Detect cell backgrounds
- [ ] Detect text alignment
- [ ] Write styling tests

#### Task 3.11: Header Detection (8 hours)
- [ ] Detect header rows
- [ ] Detect header columns
- [ ] Use font weight/style
- [ ] Write header tests

#### Task 3.12: Merged Cell Refinement (8 hours)
- [ ] Improve merged cell detection
- [ ] Handle complex spans
- [ ] Write merged cell tests

### Week 12: Table Export & Integration

#### Task 3.13: Export Formats (12 hours)
- [ ] Refine ToMarkdown()
- [ ] Refine ToCSV()
- [ ] Implement ToJSON()
- [ ] Implement ToHTML()
- [ ] Write export tests

#### Task 3.14: Table Quality Metrics (8 hours)
- [ ] Calculate precision metrics
- [ ] Add quality scores
- [ ] Add validation checks
- [ ] Write metrics tests

#### Task 3.15: Integration (16 hours)
- [ ] Integrate with Document model
- [ ] Add tables to Elements
- [ ] Test end-to-end workflow
- [ ] Benchmark performance

#### Task 3.16: Phase 3 Documentation (8 hours)
- [ ] Document table detection algorithm
- [ ] Write usage guide
- [ ] Create examples
- [ ] Update README

---

## Phase 4: Advanced Features (4 weeks)

**Goal**: Images, forms, encryption, metadata

### Week 13: Image Extraction

#### Task 4.1: Image Detection (8 hours)
- [ ] Detect Do (XObject) operators
- [ ] Parse Image XObjects
- [ ] Store image references
- [ ] Write image detection tests

#### Task 4.2: Image Decoding (12 hours)
- [ ] Implement `image/decoder.go`
- [ ] Support JPEG (DCTDecode)
- [ ] Support PNG (FlateDecode with predictors)
- [ ] Support TIFF
- [ ] Write image decoder tests

#### Task 4.3: Image Metadata (8 hours)
- [ ] Extract image dimensions
- [ ] Extract color space
- [ ] Calculate DPI
- [ ] Write metadata tests

#### Task 4.4: Figure-Caption Association (12 hours) 🎯 RAG IMPORTANT
- [ ] Implement `layout/figure_caption.go`
- [ ] Detect text near images (spatial analysis)
- [ ] Identify "Figure X:" patterns
- [ ] Associate caption with figure via proximity
- [ ] Handle captions above/below/beside figures
- [ ] Store caption text with Image element
- [ ] Write figure-caption tests

**RAG Impact**: Figure captions often contain critical information (key findings, summaries). Associating captions with figures ensures this context is captured in chunks.

#### Task 4.5: Image Integration (12 hours)
- [ ] Add Image element type
- [ ] Store binary data
- [ ] Store associated caption
- [ ] Add to document model
- [ ] Include image placeholder + caption in text chunks
- [ ] Write integration tests

### Week 14: Form Fields, Annotations & Math Content

#### Task 4.6: Math Content Detection (12 hours) 🎯 RAG IMPORTANT
- [ ] Implement `text/math.go`
- [ ] Detect mathematical symbols (∫, Σ, ∂, ≤, ≥, ±, etc.)
- [ ] Identify equation-like patterns
- [ ] Flag content as "contains math"
- [ ] Preserve math notation in extraction
- [ ] Attempt basic descriptive conversion ("x^2" → "x squared")
- [ ] Add metadata flag for math-heavy chunks
- [ ] Write math detection tests

**RAG Impact**: Mathematical content is often rendered as images or symbols. Detecting and flagging math ensures these critical sections aren't lost. While full LaTeX conversion is Phase 6, basic detection is essential.

#### Task 4.7: Form Field Parsing (12 hours)
- [ ] Parse AcroForm dictionary
- [ ] Parse field hierarchy
- [ ] Extract field values
- [ ] Write form tests

#### Task 4.8: Annotation Parsing (8 hours)
- [ ] Parse annotation dictionaries
- [ ] Support text annotations
- [ ] Support link annotations
- [ ] Write annotation tests

#### Task 4.9: Form/Annotation Integration (8 hours)
- [ ] Add to document model
- [ ] Provide access API
- [ ] Write integration tests

#### Task 4.10: Interactive Elements (8 hours)
- [ ] Detect buttons
- [ ] Detect checkboxes
- [ ] Detect text fields
- [ ] Write interactive tests

### Week 15: Encryption

#### Task 4.11: Encryption Detection (8 hours)
- [ ] Implement `core/encrypt.go`
- [ ] Parse Encrypt dictionary
- [ ] Detect encryption algorithm
- [ ] Write detection tests

#### Task 4.12: Standard Security (12 hours)
- [ ] Implement Standard security handler
- [ ] RC4 decryption
- [ ] AES decryption
- [ ] Password verification
- [ ] Write encryption tests

#### Task 4.13: Decryption Integration (8 hours)
- [ ] Decrypt objects on load
- [ ] Decrypt streams
- [ ] Handle permission flags
- [ ] Write integration tests

#### Task 4.14: Error Handling (8 hours)
- [ ] Handle wrong passwords
- [ ] Handle unsupported encryption
- [ ] Provide clear error messages
- [ ] Write error handling tests

### Week 16: Metadata & Polish

#### Task 4.15: Metadata Extraction (8 hours)
- [ ] Parse Info dictionary
- [ ] Parse XMP metadata
- [ ] Extract all standard fields
- [ ] Write metadata tests

#### Task 4.16: PDF/A Detection (8 hours)
- [ ] Detect PDF/A compliance
- [ ] Parse PDF/A metadata
- [ ] Validate PDF/A features
- [ ] Write PDF/A tests

#### Task 4.17: Error Recovery (12 hours)
- [ ] Handle malformed PDFs
- [ ] Recover from parse errors
- [ ] Lenient mode vs strict mode
- [ ] Write recovery tests

#### Task 4.18: Phase 4 Documentation (8 hours)
- [ ] Document all new features
- [ ] Update examples
- [ ] Update README
- [ ] Write migration guide

---

## Phase 5: Optimization (4 weeks)

**Goal**: Production-grade performance

### Week 17: Performance Profiling

#### Task 5.1: Benchmark Suite (12 hours)
- [ ] Create benchmark PDFs
- [ ] Write parsing benchmarks
- [ ] Write extraction benchmarks
- [ ] Write detection benchmarks
- [ ] Establish baselines

#### Task 5.2: CPU Profiling (8 hours)
- [ ] Profile with pprof
- [ ] Identify hot paths
- [ ] Document bottlenecks
- [ ] Create optimization plan

#### Task 5.3: Memory Profiling (8 hours)
- [ ] Profile memory usage
- [ ] Identify allocations
- [ ] Find memory leaks
- [ ] Create optimization plan

#### Task 5.4: Trace Analysis (8 hours)
- [ ] Run execution traces
- [ ] Analyze goroutine behavior
- [ ] Identify blocking operations
- [ ] Document findings

### Week 18: Memory Optimization

#### Task 5.5: Object Pooling (12 hours)
- [ ] Implement `internal/pool/pool.go`
- [ ] Pool TextFragments
- [ ] Pool BBoxes
- [ ] Pool common objects
- [ ] Measure improvement

#### Task 5.6: Streaming Improvements (12 hours)
- [ ] Optimize buffered I/O
- [ ] Minimize allocations in parser
- [ ] Reuse byte slices
- [ ] Measure improvement

#### Task 5.7: String Interning (8 hours)
- [ ] Implement string interning
- [ ] Intern operator names
- [ ] Intern font names
- [ ] Measure improvement

#### Task 5.8: Cache Optimization (8 hours)
- [ ] Implement LRU cache
- [ ] Tune cache sizes
- [ ] Add cache metrics
- [ ] Measure improvement

### Week 19: Performance Optimization

#### Task 5.9: Parser Optimization (12 hours)
- [ ] Optimize tokenization
- [ ] Fast path for common types
- [ ] Reduce branching
- [ ] Measure improvement

#### Task 5.10: Table Detection Optimization (8 hours)
- [ ] Add spatial indexing
- [ ] Optimize clustering
- [ ] Optimize grid construction
- [ ] Measure improvement

#### Task 5.11: Parallel Processing (12 hours)
- [ ] Implement parallel page processing
- [ ] Worker pool
- [ ] Load balancing
- [ ] Measure improvement

#### Task 5.12: Layout Optimization (8 hours)
- [ ] Optimize block detection
- [ ] Optimize reading order
- [ ] Reduce allocations
- [ ] Measure improvement

### Week 20: Hardening

#### Task 5.13: Resource Limits (8 hours)
- [ ] Add memory limits
- [ ] Add timeout limits
- [ ] Add size limits
- [ ] Write limit tests

#### Task 5.14: Stress Testing (12 hours)
- [ ] Test with huge PDFs (1000+ pages)
- [ ] Test with complex PDFs
- [ ] Test with malformed PDFs
- [ ] Fix discovered issues

#### Task 5.15: Fuzzing (12 hours)
- [ ] Set up go-fuzz
- [ ] Fuzz PDF parser
- [ ] Fuzz content stream parser
- [ ] Fix crashes

#### Task 5.16: Production Readiness (8 hours)
- [ ] Security audit
- [ ] Performance documentation
- [ ] Optimization guide
- [ ] Production checklist

---

## Phase 6: Extensions (Ongoing)

**Goal**: ML features, OCR, advanced capabilities

### ML-Based Table Detection

#### Task 6.1: ML Infrastructure (16 hours)
- [ ] Design ML detector interface
- [ ] Create training data format
- [ ] Implement model loader
- [ ] Write ML tests

#### Task 6.2: Model Integration (16 hours)
- [ ] Integrate TensorFlow/ONNX
- [ ] Load pre-trained model
- [ ] Inference pipeline
- [ ] Fallback to geometric

#### Task 6.3: Hybrid Detector (12 hours)
- [ ] Combine ML + geometric
- [ ] Confidence blending
- [ ] Best-of-both approach
- [ ] Performance testing

### OCR Integration

#### Task 6.4: OCR Interface (8 hours)
- [ ] Design OCR interface
- [ ] Tesseract integration
- [ ] Image preprocessing
- [ ] Write OCR tests

#### Task 6.5: Scanned PDF Detection (8 hours)
- [ ] Detect scanned PDFs
- [ ] Auto-trigger OCR
- [ ] Merge OCR with native text
- [ ] Write detection tests

### Advanced Features

#### Task 6.6: PDF/A Compliance (12 hours)
- [ ] Full PDF/A validation
- [ ] Compliance checking
- [ ] Compliance reporting
- [ ] Write compliance tests

#### Task 6.7: Digital Signatures (12 hours)
- [ ] Parse signature dictionaries
- [ ] Verify signatures
- [ ] Extract signer info
- [ ] Write signature tests

#### Task 6.8: Multimedia Support (8 hours)
- [ ] Support embedded audio
- [ ] Support embedded video
- [ ] Extract multimedia
- [ ] Write multimedia tests

---

## Testing Strategy

### Continuous Testing
- Run tests after each task
- Maintain > 80% coverage
- Fix bugs immediately

### Integration Testing
- Test at end of each week
- Test at end of each phase
- Full regression suite

### Corpus Testing
- Build diverse test corpus
- Test against 100+ real PDFs
- Track accuracy metrics

### Performance Testing
- Benchmark after each phase
- Track performance trends
- No regressions allowed

---

## Documentation Strategy

### Code Documentation
- Godoc for all public APIs
- Examples in doc comments
- Clear error messages

### User Documentation
- Update README after each phase
- Keep examples current
- Write migration guides

### Developer Documentation
- Update ARCHITECTURE.md as needed
- Document design decisions
- Keep diagrams current

---

## Release Strategy

### Version Scheme
- v0.1.0: Phase 1 (MVP)
- v0.2.0: Phase 2 (Layout)
- v0.3.0: Phase 3 (Tables)
- v0.4.0: Phase 4 (Advanced)
- v1.0.0: Phase 5 (Production-ready)
- v1.x.x: Phase 6 (Extensions)

### Release Checklist
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples working
- [ ] CHANGELOG updated
- [ ] GitHub release created
- [ ] Announcement post

---

## Success Metrics

### Phase 1 (MVP)
- [ ] Parse 90%+ of simple PDFs
- [ ] Extract text from 90%+ of PDFs
- [ ] < 5 seconds for 100 pages

### Phase 2 (Layout)
- [ ] Detect paragraphs with 80%+ accuracy
- [ ] Detect headings with 70%+ accuracy
- [ ] Preserve reading order correctly
- [ ] Multi-column layout detection: 85%+ accuracy
- [ ] Header/footer detection: 90%+ accuracy

### Phase 2.5 (RAG Optimization) 🎯
- [ ] Semantic chunks maintain complete thoughts (no mid-sentence breaks)
- [ ] List coherence: 95%+ of lists stay intact with intros
- [ ] Chunk metadata includes section context
- [ ] Export formats ready for vector DB ingestion
- [ ] User testing: Chunks improve RAG retrieval quality vs character chunking

### Phase 3 (Tables)
- [ ] Detect tables with 80%+ precision
- [ ] Detect tables with 70%+ recall
- [ ] Handle merged cells correctly

### Phase 4 (Advanced)
- [ ] Extract images successfully
- [ ] Figure-caption association: 80%+ accuracy
- [ ] Math content detection: 75%+ identification rate
- [ ] Decrypt encrypted PDFs
- [ ] Handle forms correctly

### Phase 5 (Optimization)
- [ ] 20-50 pages/second
- [ ] < 100MB for typical PDFs
- [ ] Linear scaling with cores

### Phase 6 (Extensions)
- [ ] ML detector improves accuracy by 10%+
- [ ] OCR works on scanned PDFs
- [ ] Full PDF/A validation

---

## Risk Management

### Technical Risks
- **PDF spec complexity**: Mitigate with thorough testing
- **Performance**: Profile early and often
- **Edge cases**: Build large test corpus

### Schedule Risks
- **Scope creep**: Stick to phases
- **Underestimation**: Add 20% buffer to estimates
- **Dependencies**: Minimize external dependencies

### Quality Risks
- **Bugs**: Comprehensive testing at every step
- **Security**: Regular security audits
- **Compatibility**: Test with diverse PDFs

---

## Next Steps

1. **Start Phase 1, Task 1.2**: Implement PDF objects
2. **Set up CI/CD**: GitHub Actions for automated testing
3. **Create test corpus**: Gather diverse PDF samples
4. **Daily progress**: Track completed tasks
5. **Weekly reviews**: Assess progress, adjust plan

---

## Estimated Total Effort

- **Phase 1**: 160 hours (4 weeks @ 40 hrs/week)
- **Phase 2**: 228 hours (5.5 weeks - expanded for international text support: CJK, RTL, emoji, Unicode normalization)
- **Phase 2.5**: 100 hours (2.5 weeks - RAG optimization)
- **Phase 3**: 160 hours (4 weeks)
- **Phase 4**: 180 hours (4.5 weeks - expanded for figure-caption + math detection)
- **Phase 5**: 160 hours (4 weeks)
- **Phase 6**: Ongoing (as needed)

**Total Core Development**: 988 hours (~6 months full-time with comprehensive international text support)

This assumes one developer working full-time. With multiple developers or part-time work, adjust accordingly.

---

## Conclusion

This implementation plan provides a clear path from the current blueprint to a production-ready, RAG-optimized PDF library. Each task is specific, measurable, and achievable.

### What Makes Tabula Different

Unlike traditional PDF libraries that focus on text extraction, Tabula is purpose-built for RAG:

- **Semantic Chunking**: Phase 2.5 implements hierarchical, context-aware chunking that respects document structure
- **Noise Removal**: Automatic header/footer detection ensures clean embeddings
- **Structure Preservation**: Multi-column layouts, tables, and lists maintain their logical organization
- **Context Enrichment**: Every chunk includes metadata (section headings, page numbers, element types)
- **RAG-Ready Export**: JSON Lines, CSV, and Parquet formats designed for vector database ingestion

### Implementation Priority

If you need RAG functionality sooner:
1. Complete Phase 1 (core parsing)
2. Implement Phase 2 international text support (Tasks 2.3-2.5b: CJK, RTL, emoji, normalization)
3. Implement Phase 2 RAG-critical tasks (Tasks 2.9, 2.10, 2.15, 2.16: multi-column, headers/footers, headings, lists)
4. Jump to Phase 2.5 (semantic chunking) - **this is the killer feature**
5. Return to Phase 3 (tables) for structured data extraction

Follow this plan step-by-step, and you'll build not just a PDF library, but a **RAG-first document intelligence system**.

**Current Progress**: Phase 1, Week 2 - Tasks 1.1 through 1.8 complete ✅
- ✅ Task 1.1-1.2: Project setup & objects
- ✅ Task 1.3: Lexer (2.4M ops/sec)
- ✅ Task 1.4: Parser (2.8M ops/sec, 82.4% coverage)
- ✅ Task 1.5: XRef parsing (120K tables/sec, 79.5% coverage)
- ✅ Task 1.6: File reader (79.1% coverage)
- ✅ Task 1.7: Object resolver (86.6% coverage)
- ✅ Task 1.8: Catalog & pages (72.1% coverage)

**Package Coverage**:
- Core: 79.5%
- Reader: 79.1%
- Resolver: 86.6%
- Pages: 72.1%
- **Average: ~79%**

**Phase 1 Progress**: 8 of 15 tasks complete (53%)

**Next Up**: Task 1.9 - FlateDecode (Stream Decompression)

Good luck! 🚀
