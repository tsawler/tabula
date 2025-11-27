# DOCX Support Implementation Plan

This document outlines the implementation plan for adding Microsoft Word (.docx) file support to the tabula library.

## Executive Summary

**Estimated Effort**: 2-3 weeks for full implementation
**Code Reuse**: 80-90% of existing infrastructure
**New Code**: ~2,000-3,000 lines
**Risk Level**: Low-Medium (well-understood format, good Go libraries available)

### Why DOCX Support Makes Sense

1. **Architecture Ready**: The `model/` package is completely format-agnostic
2. **Shared Processing**: Layout analysis, table handling, and RAG integration work on the IR
3. **User Demand**: DOCX is ubiquitous in enterprise environments
4. **Simpler Parsing**: DOCX has explicit structure (unlike PDF's render-based model)

---

## DOCX Format Overview

### What is DOCX?

DOCX (Office Open XML) is a ZIP archive containing XML files:

```
document.docx (ZIP archive)
├── [Content_Types].xml          # MIME types for parts
├── _rels/
│   └── .rels                    # Root relationships
├── word/
│   ├── document.xml             # Main document content
│   ├── styles.xml               # Style definitions
│   ├── settings.xml             # Document settings
│   ├── fontTable.xml            # Font declarations
│   ├── numbering.xml            # List numbering definitions
│   ├── _rels/
│   │   └── document.xml.rels    # Document relationships (images, etc.)
│   └── media/                   # Embedded images
│       ├── image1.png
│       └── image2.jpeg
└── docProps/
    ├── core.xml                 # Dublin Core metadata
    └── app.xml                  # Application metadata
```

### Key XML Namespaces

```xml
xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
```

### Document Structure

```xml
<w:document>
  <w:body>
    <w:p>                        <!-- Paragraph -->
      <w:pPr>                    <!-- Paragraph properties -->
        <w:pStyle w:val="Heading1"/>
        <w:jc w:val="center"/>   <!-- Justification -->
      </w:pPr>
      <w:r>                      <!-- Run (text with same formatting) -->
        <w:rPr>                  <!-- Run properties -->
          <w:b/>                 <!-- Bold -->
          <w:sz w:val="24"/>     <!-- Font size (half-points) -->
        </w:rPr>
        <w:t>Hello World</w:t>   <!-- Text content -->
      </w:r>
    </w:p>
    <w:tbl>                      <!-- Table -->
      <w:tr>                     <!-- Table row -->
        <w:tc>                   <!-- Table cell -->
          <w:p>...</w:p>
        </w:tc>
      </w:tr>
    </w:tbl>
  </w:body>
</w:document>
```

### DOCX vs PDF: Key Differences

| Aspect | PDF | DOCX |
|--------|-----|------|
| **Coordinates** | Absolute (points) | Flow-based (no coordinates) |
| **Structure** | Implicit (must be detected) | Explicit (XML elements) |
| **Tables** | Lines/whitespace detection | `<w:tbl>` elements |
| **Headings** | Font size heuristics | Style references |
| **Lists** | Bullet detection | `<w:numPr>` elements |
| **Images** | Embedded streams | Relationship references |

---

## Architecture Design

### Integration Strategy

```
┌─────────────────────────────────────────────────────────┐
│              Application Layer (unchanged)               │
│   tabula.Open("file.pdf|docx") → Extractor              │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│           Format Detection & Routing (NEW)              │
│   DetectFormat() → pdf_reader | docx_reader             │
└─────────────────────────────────────────────────────────┘
          ↓                              ↓
┌──────────────────────┐    ┌──────────────────────────┐
│  PDF Reader          │    │  DOCX Reader (NEW)       │
│  (existing)          │    │  docx/reader.go          │
│  reader/reader.go    │    │  docx/parser.go          │
└──────────────────────┘    │  docx/styles.go          │
          ↓                 │  docx/tables.go          │
          ↓                 │  docx/images.go          │
          ↓                 └──────────────────────────┘
          ↓                              ↓
┌─────────────────────────────────────────────────────────┐
│              model.Document (unchanged)                  │
│   Pages, Elements, Tables, Paragraphs, Headings         │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│         Shared Processing (unchanged)                    │
│   layout/ (optional) | tables/ | rag/                   │
└─────────────────────────────────────────────────────────┘
```

### New Package Structure

```
tabula/
├── docx/                        # NEW: DOCX-specific code
│   ├── reader.go                # Main entry point, ZIP handling
│   ├── document.go              # document.xml parsing
│   ├── styles.go                # styles.xml parsing
│   ├── numbering.go             # numbering.xml (lists)
│   ├── relationships.go         # .rels file parsing
│   ├── tables.go                # Table extraction
│   ├── images.go                # Image extraction
│   ├── metadata.go              # docProps parsing
│   └── reader_test.go           # Tests
├── format/                      # NEW: Format detection
│   ├── detect.go                # File format detection
│   └── detect_test.go
└── tabula.go                    # MODIFIED: Multi-format support
```

### Interface Definitions

```go
// format/format.go
package format

type Format int

const (
    Unknown Format = iota
    PDF
    DOCX
    // Future: PPTX, XLSX, etc.
)

// Detect determines file format from extension and magic bytes
func Detect(filename string) (Format, error)
func DetectFromReader(r io.ReaderAt) (Format, error)
```

```go
// Internal interface for format-specific readers
// (Not exported - implementation detail)
type documentReader interface {
    PageCount() (int, error)
    ExtractPage(index int) (*model.Page, error)
    Metadata() (model.Metadata, error)
    Close() error
}
```

---

## Implementation Phases

### Phase 1: Foundation (3-4 days)

**Goal**: Basic DOCX reading with text extraction

#### Task 1.1: Format Detection
**File**: `format/detect.go`

```go
package format

import (
    "os"
    "path/filepath"
    "strings"
)

type Format int

const (
    Unknown Format = iota
    PDF
    DOCX
)

func (f Format) String() string {
    switch f {
    case PDF:
        return "PDF"
    case DOCX:
        return "DOCX"
    default:
        return "Unknown"
    }
}

// Detect determines file format from filename
func Detect(filename string) Format {
    ext := strings.ToLower(filepath.Ext(filename))
    switch ext {
    case ".pdf":
        return PDF
    case ".docx":
        return DOCX
    default:
        return Unknown
    }
}

// DetectFromMagic checks file magic bytes
func DetectFromMagic(data []byte) Format {
    if len(data) >= 4 {
        // ZIP magic (DOCX is a ZIP)
        if data[0] == 0x50 && data[1] == 0x4B &&
           data[2] == 0x03 && data[3] == 0x04 {
            return DOCX  // Could be DOCX, XLSX, PPTX - need further check
        }
        // PDF magic
        if string(data[:4]) == "%PDF" {
            return PDF
        }
    }
    return Unknown
}
```

#### Task 1.2: DOCX Reader Skeleton
**File**: `docx/reader.go`

- Open ZIP archive
- Validate required files exist ([Content_Types].xml, word/document.xml)
- Parse relationships
- Implement `Close()`

#### Task 1.3: Basic Document Parsing
**File**: `docx/document.go`

- Parse `<w:body>` structure
- Extract `<w:p>` paragraphs
- Extract text from `<w:t>` elements
- Handle `<w:r>` runs

#### Task 1.4: Integration with Extractor
**File**: `tabula.go` (modifications)

- Modify `Open()` to detect format
- Route to appropriate reader
- Return same `Extractor` type

**Deliverable**: `tabula.Open("file.docx").Text()` returns plain text

---

### Phase 2: Styles & Structure (3-4 days)

**Goal**: Heading detection, paragraph styles, font information

#### Task 2.1: Style Parsing
**File**: `docx/styles.go`

- Parse `word/styles.xml`
- Build style inheritance tree
- Resolve paragraph styles to properties
- Map built-in styles (Heading1-9, Title, Normal, etc.)

```go
type Style struct {
    ID       string
    Name     string
    BasedOn  string  // Parent style ID
    Type     string  // paragraph, character, table

    // Resolved properties
    FontName string
    FontSize float64  // Points (DOCX uses half-points)
    Bold     bool
    Italic   bool
    // ... etc
}

type StyleResolver struct {
    styles map[string]*Style
}

func (r *StyleResolver) Resolve(styleID string) *Style
```

#### Task 2.2: Heading Detection
**File**: `docx/document.go` (additions)

- Detect heading styles (Heading1-9, Title)
- Map to `model.Heading` with appropriate level
- Handle custom heading styles (based on outline level)

#### Task 2.3: Paragraph Properties
- Extract alignment (left, center, right, justify)
- Extract indentation
- Extract spacing (before/after)
- Map to `model.Paragraph`

#### Task 2.4: Run Properties
- Bold, italic, underline
- Font name and size
- Map to `model.TextStyle`

**Deliverable**: Headings and styled paragraphs in output

---

### Phase 3: Tables (2-3 days)

**Goal**: Full table extraction with structure

#### Task 3.1: Table Parsing
**File**: `docx/tables.go`

```go
func (p *Parser) parseTable(tbl *xmlquery.Node) (*model.Table, error) {
    // Parse <w:tbl>
    //   <w:tblPr> - table properties
    //   <w:tblGrid> - column definitions
    //   <w:tr> - rows
    //     <w:tc> - cells
}
```

#### Task 3.2: Cell Properties
- Row span (`<w:vMerge>`)
- Column span (`<w:gridSpan>`)
- Cell width
- Vertical alignment
- Borders

#### Task 3.3: Table Grid
- Parse `<w:tblGrid>` for column widths
- Handle merged cells correctly
- Build `model.Table` with proper structure

#### Task 3.4: Nested Tables
- Handle tables within cells
- Recursive parsing

**Deliverable**: `tabula.Open("file.docx").Tables()` returns structured tables

---

### Phase 4: Lists & Numbering (2 days)

**Goal**: Ordered and unordered list detection

#### Task 4.1: Numbering Definitions
**File**: `docx/numbering.go`

- Parse `word/numbering.xml`
- Build numbering format definitions
- Handle abstract numbering and numbering instances

```go
type NumberingDef struct {
    NumID        int
    AbstractNumID int
    Levels       []NumberingLevel
}

type NumberingLevel struct {
    Level   int
    Format  string  // decimal, bullet, lowerLetter, etc.
    Text    string  // Format string like "%1."
    Start   int
}
```

#### Task 4.2: List Detection
- Detect `<w:numPr>` in paragraphs
- Group consecutive numbered paragraphs into lists
- Determine ordered vs unordered
- Map to `model.List` and `model.ListItem`

#### Task 4.3: Nested Lists
- Handle multi-level lists
- Track indentation levels

**Deliverable**: Lists properly extracted and structured

---

### Phase 5: Images & Media (2 days)

**Goal**: Extract embedded images

#### Task 5.1: Relationship Parsing
**File**: `docx/relationships.go`

- Parse `word/_rels/document.xml.rels`
- Map relationship IDs to targets
- Handle different relationship types

#### Task 5.2: Inline Images
**File**: `docx/images.go`

- Parse `<w:drawing>` elements
- Extract image references (`<a:blip r:embed="rId5"/>`)
- Load image data from `word/media/`

#### Task 5.3: Image Properties
- Extract dimensions
- Calculate approximate position (based on paragraph order)
- Handle alt text (`<wp:docPr descr="..."/>`)

#### Task 5.4: Image Formats
- Support JPEG, PNG, GIF, TIFF
- Detect format from content type or extension
- Map to `model.Image`

**Deliverable**: Images extracted with metadata

---

### Phase 6: Metadata & Polish (1-2 days)

**Goal**: Document metadata, edge cases, testing

#### Task 6.1: Core Metadata
**File**: `docx/metadata.go`

- Parse `docProps/core.xml` (Dublin Core)
  - Title, Author, Subject, Keywords
  - Created, Modified dates
- Parse `docProps/app.xml`
  - Application name (Creator/Producer equivalent)
  - Page count, word count

#### Task 6.2: Custom Properties
- Parse custom document properties
- Map to `model.Metadata.Custom`

#### Task 6.3: Edge Cases
- Empty documents
- Documents with only tables
- Documents with only images
- Malformed XML handling
- Very large documents

#### Task 6.4: Comprehensive Testing
- Unit tests for each component
- Integration tests with real DOCX files
- Test corpus (10-20 diverse documents)
- Benchmark tests

**Deliverable**: Full DOCX support with robust error handling

---

## API Design

### User-Facing API (No Changes Required)

The existing fluent API works unchanged:

```go
// PDF (existing)
text, warnings, err := tabula.Open("document.pdf").Text()

// DOCX (new - same API!)
text, warnings, err := tabula.Open("document.docx").Text()

// With options (works for both)
text, _, err := tabula.Open("report.docx").
    Pages(1, 2, 3).
    ExcludeHeaders().
    Text()

// Tables (works for both)
tables, _, err := tabula.Open("data.docx").Tables()

// Full document (works for both)
doc, _, err := tabula.Open("report.docx").Document()
```

### Format-Specific Options (Future)

If needed, format-specific options can be added:

```go
// Potential future API
text, _, err := tabula.Open("document.docx").
    DOCXOptions(tabula.DOCXOpts{
        IncludeComments:  true,
        IncludeRevisions: false,
    }).
    Text()
```

---

## Page Model Considerations

### The "Page" Concept in DOCX

DOCX doesn't have fixed pages like PDF. Options:

#### Option A: Single Page (Recommended for Phase 1)
- Treat entire document as one "page"
- Simpler implementation
- Consistent with continuous document model

```go
// Implementation
func (r *Reader) ExtractDocument() (*model.Document, error) {
    doc := model.NewDocument()
    page := model.NewPage()
    page.Number = 1

    // All content goes on page 1
    for _, elem := range elements {
        page.Elements = append(page.Elements, elem)
    }

    doc.AddPage(page)
    return doc, nil
}
```

#### Option B: Section-Based Pages (Future Enhancement)
- Each `<w:sectPr>` creates a new page
- Better for documents with explicit page breaks
- More complex implementation

#### Option C: Calculated Pages (Future Enhancement)
- Estimate page breaks based on content height
- Requires layout calculation
- Most accurate but most complex

**Recommendation**: Start with Option A, add Option B later if needed.

---

## Position/BBox Handling

### Challenge

DOCX doesn't have absolute positions. The `model.BBox` fields need values.

### Solution: Synthetic Positions

Calculate synthetic positions based on:
- Document flow order
- Estimated line heights (from font size)
- Standard page dimensions (8.5" x 11")

```go
// Synthetic position calculator
type PositionCalculator struct {
    pageWidth  float64  // 612 points (8.5")
    pageHeight float64  // 792 points (11")
    marginX    float64  // 72 points (1")
    marginY    float64  // 72 points (1")
    currentY   float64  // Current Y position
}

func (c *PositionCalculator) NextPosition(height float64) model.BBox {
    bbox := model.BBox{
        X:      c.marginX,
        Y:      c.pageHeight - c.marginY - c.currentY - height,
        Width:  c.pageWidth - 2*c.marginX,
        Height: height,
    }
    c.currentY += height
    return bbox
}
```

### Alternative: Zero BBox with Flag

```go
// Mark elements as not having real positions
type Element interface {
    // ... existing methods
    HasPosition() bool  // false for DOCX elements
}
```

**Recommendation**: Use synthetic positions for compatibility with existing code.

---

## Dependencies

### Required (stdlib only)
- `archive/zip` - DOCX decompression
- `encoding/xml` - XML parsing

### Optional (for better XML handling)
- `github.com/antchfx/xmlquery` - XPath queries (makes parsing easier)

### Recommendation

Start with stdlib `encoding/xml`. Add xmlquery only if parsing becomes too complex.

```go
// Example with stdlib
type Document struct {
    XMLName xml.Name `xml:"document"`
    Body    Body     `xml:"body"`
}

type Body struct {
    Paragraphs []Paragraph `xml:"p"`
    Tables     []Table     `xml:"tbl"`
}

type Paragraph struct {
    Properties ParagraphProps `xml:"pPr"`
    Runs       []Run          `xml:"r"`
}
```

---

## Testing Strategy

### Unit Tests

```go
// docx/reader_test.go
func TestOpenValid(t *testing.T)
func TestOpenInvalid(t *testing.T)
func TestOpenNotZip(t *testing.T)

// docx/document_test.go
func TestParseParagraph(t *testing.T)
func TestParseRuns(t *testing.T)
func TestParseHeading(t *testing.T)

// docx/tables_test.go
func TestParseSimpleTable(t *testing.T)
func TestParseMergedCells(t *testing.T)
func TestParseNestedTable(t *testing.T)

// docx/styles_test.go
func TestResolveStyle(t *testing.T)
func TestStyleInheritance(t *testing.T)
```

### Integration Tests

```go
func TestExtractText(t *testing.T) {
    text, _, err := tabula.Open("testdata/simple.docx").Text()
    require.NoError(t, err)
    assert.Contains(t, text, "Expected content")
}

func TestExtractTables(t *testing.T) {
    tables, _, err := tabula.Open("testdata/tables.docx").Tables()
    require.NoError(t, err)
    assert.Len(t, tables, 2)
    assert.Equal(t, 3, tables[0].RowCount())
}
```

### Test Corpus

Create test documents covering:

| File | Purpose |
|------|---------|
| `simple.docx` | Plain paragraphs |
| `headings.docx` | H1-H6 headings |
| `styles.docx` | Various text styles |
| `tables.docx` | Simple tables |
| `merged_cells.docx` | Complex table with spans |
| `nested_tables.docx` | Tables within tables |
| `lists.docx` | Ordered and unordered lists |
| `images.docx` | Embedded images |
| `mixed.docx` | All element types |
| `large.docx` | 100+ pages for performance |
| `empty.docx` | Empty document |
| `metadata.docx` | Rich metadata |

---

## Risk Assessment

### Low Risk
- **XML Parsing**: Well-understood, good stdlib support
- **ZIP Handling**: Stdlib `archive/zip` is mature
- **Model Mapping**: Existing model is flexible

### Medium Risk
- **Style Resolution**: Complex inheritance chains
- **Table Merging**: Row/column span calculations
- **Position Calculation**: Synthetic positions may not match expectations

### Mitigation Strategies
1. **Style Resolution**: Build comprehensive test cases from real documents
2. **Table Merging**: Port tested algorithms from existing table code
3. **Positions**: Document that DOCX positions are approximate; offer flag to disable

---

## Success Criteria

### Phase 1 Complete When:
- [ ] `tabula.Open("file.docx").Text()` returns document text
- [ ] Basic paragraphs extracted
- [ ] Unit tests pass
- [ ] Works with 5+ test documents

### Phase 2 Complete When:
- [ ] Headings detected with correct levels
- [ ] Text styles (bold, italic) preserved
- [ ] Paragraph alignment extracted
- [ ] Works with styled documents

### Phase 3 Complete When:
- [ ] Tables extracted with structure
- [ ] Merged cells handled correctly
- [ ] `table.ToMarkdown()` produces valid output
- [ ] Works with complex table documents

### Phase 4 Complete When:
- [ ] Ordered lists detected
- [ ] Unordered lists detected
- [ ] Nested lists handled
- [ ] List markers preserved

### Phase 5 Complete When:
- [ ] Images extracted
- [ ] Image dimensions available
- [ ] Alt text preserved
- [ ] Various image formats supported

### Phase 6 Complete When:
- [ ] Document metadata extracted
- [ ] Edge cases handled gracefully
- [ ] Test coverage > 80%
- [ ] Documentation complete

---

## Implementation Schedule

| Phase | Tasks | Estimated Days |
|-------|-------|----------------|
| Phase 1 | Foundation | 3-4 |
| Phase 2 | Styles & Structure | 3-4 |
| Phase 3 | Tables | 2-3 |
| Phase 4 | Lists | 2 |
| Phase 5 | Images | 2 |
| Phase 6 | Metadata & Polish | 1-2 |
| **Total** | | **13-17 days** |

---

## Future Enhancements (Post-MVP)

### Near-term
- Comments and track changes
- Headers and footers
- Footnotes and endnotes
- Hyperlinks
- Bookmarks

### Long-term
- DOCX writing (round-trip)
- Template filling
- Mail merge support
- PPTX support (similar architecture)
- XLSX support (similar architecture)

---

## Getting Started

To begin implementation:

1. Create the `docx/` and `format/` directories
2. Start with Phase 1, Task 1.1 (format detection)
3. Write tests first (TDD)
4. Create test documents as needed
5. Integrate with existing `tabula.go` incrementally

```bash
# Create directory structure
mkdir -p tabula/docx tabula/format

# Create initial files
touch tabula/format/detect.go
touch tabula/format/detect_test.go
touch tabula/docx/reader.go
touch tabula/docx/reader_test.go
```