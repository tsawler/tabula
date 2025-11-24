# Tabula Implementation Plan

This document provides a step-by-step implementation plan to build the Tabula library from the ground up.

## Overview

This plan breaks down the 6-phase roadmap into specific, actionable tasks with clear acceptance criteria and estimated effort.

## Phase 1: MVP - Core PDF Parsing (4 weeks)

**Goal**: Read basic PDFs and extract raw text

### Week 1: Foundation & Object Parsing

#### Task 1.1: Project Setup (2 hours)
- [x] Initialize git repository
- [x] Create package structure
- [x] Set up go.mod with dependencies
- [x] Create basic README

**Deliverable**: Working Go module structure

#### Task 1.2: PDF Object Implementation (8 hours)
- [ ] Implement all object types in `core/object.go`
  - Bool, Int, Real, String, Name
  - Array, Dict, Stream, IndirectRef
- [ ] Add helper methods on Dict (Get, GetName, GetInt, etc.)
- [ ] Write unit tests for each object type

**Deliverable**: Complete object model with tests
**Acceptance**: All object types parse correctly

#### Task 1.3: Lexer/Tokenizer (12 hours)
- [ ] Implement buffered PDF reader in `core/reader.go`
- [ ] Implement tokenization (skipWhitespace, readToken, etc.)
- [ ] Handle PDF comments (%)
- [ ] Handle different newline formats (\r, \n, \r\n)
- [ ] Write comprehensive tokenizer tests

**Deliverable**: Robust tokenizer
**Acceptance**: Can tokenize any valid PDF syntax

#### Task 1.4: Object Parser (16 hours)
- [ ] Complete `core/parser.go` implementation
- [ ] parseBool, parseNumber, parseString
- [ ] parseHexString, parseName
- [ ] parseArray, parseDict
- [ ] Handle indirect references (num gen R)
- [ ] Write parser tests for all object types
- [ ] Test with nested structures

**Deliverable**: Working PDF object parser
**Acceptance**: Can parse all 8 PDF object types

### Week 2: XRef & File Structure

#### Task 1.5: XRef Table Parsing (12 hours)
- [ ] Implement `core/xref.go`
- [ ] Parse traditional XRef table
- [ ] Parse trailer dictionary
- [ ] Handle multiple XRef sections
- [ ] Find XRef by scanning from EOF
- [ ] Write XRef parser tests

**Deliverable**: XRef table parser
**Acceptance**: Can parse XRef tables from real PDFs

#### Task 1.6: File Reader (8 hours)
- [ ] Implement `reader/reader.go`
- [ ] Parse PDF header (%PDF-x.y)
- [ ] Load XRef table
- [ ] Load trailer
- [ ] Resolve indirect references
- [ ] Write file reader tests

**Deliverable**: Basic PDF file reader
**Acceptance**: Can open and parse PDF structure

#### Task 1.7: Object Resolution (8 hours)
- [ ] Implement lazy object loading
- [ ] Cache resolved objects
- [ ] Handle circular references
- [ ] Add object resolver tests

**Deliverable**: Object graph traversal
**Acceptance**: Can resolve any object in PDF

#### Task 1.8: Catalog & Pages (8 hours)
- [ ] Parse document catalog
- [ ] Parse pages tree
- [ ] Implement page enumeration
- [ ] Get page by number
- [ ] Write page access tests

**Deliverable**: Page access API
**Acceptance**: Can enumerate all pages

### Week 3: Stream Decoding & Content Streams

#### Task 1.9: FlateDecode (12 hours)
- [ ] Implement `internal/compress/flate.go`
- [ ] Handle PNG predictors
- [ ] Support DecodeParms
- [ ] Write decompression tests
- [ ] Test with real PDF streams

**Deliverable**: FlateDecode support
**Acceptance**: Can decompress Flate-encoded streams

#### Task 1.10: Stream Decoder (8 hours)
- [ ] Implement `core/stream.go`
- [ ] Stream.Decoded() method
- [ ] Handle Filter array (multiple filters)
- [ ] Support ASCIIHexDecode, ASCII85Decode
- [ ] Write stream decoder tests

**Deliverable**: Multi-filter stream decoder
**Acceptance**: Can decode common PDF streams

#### Task 1.11: Content Stream Parser (16 hours)
- [ ] Implement `contentstream/parser.go`
- [ ] Parse operators and operands
- [ ] Handle all operator types
- [ ] Build operation list
- [ ] Write content stream parser tests

**Deliverable**: Content stream parser
**Acceptance**: Can parse page content streams

### Week 4: Text Extraction

#### Task 1.12: Graphics State Machine (12 hours)
- [ ] Implement `contentstream/graphics.go`
- [ ] GraphicsState struct
- [ ] State stack (q/Q operators)
- [ ] CTM tracking (cm operator)
- [ ] Text state tracking (Tf, Tm, etc.)
- [ ] Write state machine tests

**Deliverable**: Graphics state machine
**Acceptance**: Correctly tracks all state changes

#### Task 1.13: Simple Font Support (8 hours)
- [ ] Implement `font/font.go` interface
- [ ] Basic font metrics
- [ ] Standard 14 fonts (hardcoded metrics)
- [ ] Simple encoding support
- [ ] Write font tests

**Deliverable**: Basic font support
**Acceptance**: Can handle standard fonts

#### Task 1.14: Text Extraction (16 hours)
- [ ] Implement `text/extractor.go`
- [ ] Process text operators (Tj, TJ, ')
- [ ] Calculate text positions
- [ ] Create TextFragment list
- [ ] Extract text in reading order
- [ ] Write text extraction tests

**Deliverable**: Text extraction engine
**Acceptance**: Can extract text from simple PDFs

#### Task 1.15: Integration & Testing (8 hours)
- [ ] End-to-end test: open PDF, extract text
- [ ] Test with 10+ real PDFs
- [ ] Fix bugs found during testing
- [ ] Document Phase 1 API

**Deliverable**: MVP release
**Acceptance**: Can extract text from 90%+ of simple PDFs

---

## Phase 2: Text & Layout (4 weeks)

**Goal**: Advanced text extraction with layout preservation

### Week 5: Font Handling

#### Task 2.1: Type1 Font Parser (16 hours)
- [ ] Implement `font/type1.go`
- [ ] Parse Type1 font dictionary
- [ ] Load font metrics
- [ ] Handle font encoding
- [ ] Character width calculation
- [ ] Write Type1 tests

#### Task 2.2: TrueType Font Parser (16 hours)
- [ ] Implement `font/truetype.go`
- [ ] Parse TrueType font program
- [ ] Extract glyph metrics
- [ ] Handle Unicode mapping
- [ ] Write TrueType tests

#### Task 2.3: Encoding Support (8 hours)
- [ ] Implement `text/encoding.go`
- [ ] WinAnsiEncoding
- [ ] MacRomanEncoding
- [ ] PDFDocEncoding
- [ ] Custom encodings
- [ ] Write encoding tests

### Week 6: Advanced Text Extraction

#### Task 2.4: Enhanced Text Extractor (12 hours)
- [ ] Improve text positioning accuracy
- [ ] Handle character spacing
- [ ] Handle word spacing
- [ ] Handle text rise
- [ ] Write positioning tests

#### Task 2.5: Text Fragment Ordering (8 hours)
- [ ] Sort fragments by position
- [ ] Detect reading order
- [ ] Handle multi-column layouts
- [ ] Write ordering tests

#### Task 2.6: Unicode Support (8 hours)
- [ ] Implement `text/unicode.go`
- [ ] ToUnicode CMap parsing
- [ ] Unicode normalization
- [ ] Write Unicode tests

#### Task 2.7: CJK Font Support (8 hours)
- [ ] Implement `font/cmap.go`
- [ ] CMap parsing for CJK fonts
- [ ] Character code mapping
- [ ] Write CJK tests

### Week 7: Layout Analysis

#### Task 2.8: Block Detection (12 hours)
- [ ] Implement `layout/block.go`
- [ ] Group fragments into blocks
- [ ] Detect block boundaries
- [ ] Write block detection tests

#### Task 2.9: Line Detection (8 hours)
- [ ] Implement `layout/line.go`
- [ ] Group fragments into lines
- [ ] Detect line spacing
- [ ] Write line detection tests

#### Task 2.10: Paragraph Detection (12 hours)
- [ ] Implement `layout/paragraph.go`
- [ ] Group lines into paragraphs
- [ ] Detect paragraph breaks
- [ ] Handle indentation
- [ ] Write paragraph tests

#### Task 2.11: Reading Order (8 hours)
- [ ] Implement `layout/reading_order.go`
- [ ] Column detection
- [ ] Z-order sorting
- [ ] Write reading order tests

### Week 8: Heading & List Detection

#### Task 2.12: Heading Detection (12 hours)
- [ ] Implement `layout/heading.go`
- [ ] Detect by font size
- [ ] Detect by font weight
- [ ] Detect by position
- [ ] Assign heading levels
- [ ] Write heading tests

#### Task 2.13: List Detection (8 hours)
- [ ] Implement `layout/list.go`
- [ ] Detect bullet points
- [ ] Detect numbering
- [ ] Handle nested lists
- [ ] Write list tests

#### Task 2.14: Layout Analyzer (16 hours)
- [ ] Implement `layout/analyzer.go`
- [ ] Orchestrate all detection
- [ ] Build element tree
- [ ] Assign element types
- [ ] Write integration tests

#### Task 2.15: Phase 2 Integration (8 hours)
- [ ] Update Document model
- [ ] Update Page model
- [ ] Add Elements to pages
- [ ] Test with complex PDFs
- [ ] Document Phase 2 API

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

#### Task 4.4: Image Integration (8 hours)
- [ ] Add Image element type
- [ ] Store binary data
- [ ] Add to document model
- [ ] Write integration tests

### Week 14: Form Fields & Annotations

#### Task 4.5: Form Field Parsing (12 hours)
- [ ] Parse AcroForm dictionary
- [ ] Parse field hierarchy
- [ ] Extract field values
- [ ] Write form tests

#### Task 4.6: Annotation Parsing (8 hours)
- [ ] Parse annotation dictionaries
- [ ] Support text annotations
- [ ] Support link annotations
- [ ] Write annotation tests

#### Task 4.7: Form/Annotation Integration (8 hours)
- [ ] Add to document model
- [ ] Provide access API
- [ ] Write integration tests

#### Task 4.8: Interactive Elements (8 hours)
- [ ] Detect buttons
- [ ] Detect checkboxes
- [ ] Detect text fields
- [ ] Write interactive tests

### Week 15: Encryption

#### Task 4.9: Encryption Detection (8 hours)
- [ ] Implement `core/encrypt.go`
- [ ] Parse Encrypt dictionary
- [ ] Detect encryption algorithm
- [ ] Write detection tests

#### Task 4.10: Standard Security (12 hours)
- [ ] Implement Standard security handler
- [ ] RC4 decryption
- [ ] AES decryption
- [ ] Password verification
- [ ] Write encryption tests

#### Task 4.11: Decryption Integration (8 hours)
- [ ] Decrypt objects on load
- [ ] Decrypt streams
- [ ] Handle permission flags
- [ ] Write integration tests

#### Task 4.12: Error Handling (8 hours)
- [ ] Handle wrong passwords
- [ ] Handle unsupported encryption
- [ ] Provide clear error messages
- [ ] Write error handling tests

### Week 16: Metadata & Polish

#### Task 4.13: Metadata Extraction (8 hours)
- [ ] Parse Info dictionary
- [ ] Parse XMP metadata
- [ ] Extract all standard fields
- [ ] Write metadata tests

#### Task 4.14: PDF/A Detection (8 hours)
- [ ] Detect PDF/A compliance
- [ ] Parse PDF/A metadata
- [ ] Validate PDF/A features
- [ ] Write PDF/A tests

#### Task 4.15: Error Recovery (12 hours)
- [ ] Handle malformed PDFs
- [ ] Recover from parse errors
- [ ] Lenient mode vs strict mode
- [ ] Write recovery tests

#### Task 4.16: Phase 4 Documentation (8 hours)
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

### Phase 3 (Tables)
- [ ] Detect tables with 80%+ precision
- [ ] Detect tables with 70%+ recall
- [ ] Handle merged cells correctly

### Phase 4 (Advanced)
- [ ] Extract images successfully
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
- **Phase 2**: 160 hours (4 weeks)
- **Phase 3**: 160 hours (4 weeks)
- **Phase 4**: 160 hours (4 weeks)
- **Phase 5**: 160 hours (4 weeks)
- **Phase 6**: Ongoing (as needed)

**Total Core Development**: 800 hours (~5 months full-time)

This assumes one developer working full-time. With multiple developers or part-time work, adjust accordingly.

---

## Conclusion

This implementation plan provides a clear path from the current blueprint to a production-ready library. Each task is specific, measurable, and achievable. Follow this plan step-by-step, and you'll build a world-class PDF library for Go.

**Start with Phase 1, Task 1.2**, and work systematically through each task. Good luck! 🚀
