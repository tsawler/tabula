# Task 1.8: Catalog & Pages - COMPLETE ✅

**Date Completed**: November 24, 2024
**Time Taken**: ~1.5 hours
**Estimated Time**: 8 hours

## Deliverable

Complete page access API with document catalog parsing, page tree traversal, and page property access.

## What Was Implemented

### 1. Core Structures (`pages.go` - 414 lines)

#### Catalog Type
```go
type Catalog struct {
    dict     core.Dict
    resolver ObjectResolver
}
```

Represents the PDF document catalog (root of document structure):
- **Type()** - Returns "Catalog"
- **Pages()** - Returns page tree root dictionary
- **Metadata()** - Returns metadata stream (optional)
- **Version()** - Returns catalog version (optional)

The catalog is the entry point to the document structure, referenced by `/Root` in the trailer.

#### PageTree Type
```go
type PageTree struct {
    root     core.Dict
    resolver ObjectResolver
    pages    []*Page // Cached flattened page list
}
```

Manages the PDF page tree:
- **Count()** - Returns total number of pages
- **GetPage(index)** - Returns page at index (0-based)
- **Pages()** - Returns all pages as slice
- Handles both flat and nested page tree structures
- Caches flattened page list for efficient access

The page tree can be:
- **Flat**: Single `/Pages` node with all pages as direct children
- **Nested**: Tree of `/Pages` nodes with pages at leaves (for large PDFs)

#### Page Type
```go
type Page struct {
    dict     core.Dict
    parent   core.Dict // Parent Pages node
    resolver ObjectResolver
}
```

Represents a single PDF page with methods for:
- **MediaBox()** - Page media box [x1, y1, x2, y2]
- **CropBox()** - Page crop box (defaults to MediaBox)
- **Resources()** - Page resources (fonts, images, etc.)
- **Contents()** - Page content streams
- **Rotate()** - Page rotation (0, 90, 180, 270)
- **Width()** / **Height()** - Page dimensions

### 2. Inheritable Attributes

PDF supports attribute inheritance from parent `/Pages` nodes. The implementation correctly handles inheritable attributes:

**Inheritable Attributes**:
- MediaBox
- CropBox
- Resources
- Rotate

**Algorithm**:
1. Check page dictionary first
2. If not found, check parent dictionary
3. If still not found, return error (or default for optional attributes)

Example:
```go
func (p *Page) getBox(name string) ([]float64, error) {
    boxObj := p.dict.Get(name)

    // Try parent if not in page dict
    if boxObj == nil && p.parent != nil {
        boxObj = p.parent.Get(name)
    }

    // ... resolve and parse
}
```

This is critical for PDFs with many pages that share common properties (reduces file size).

### 3. Page Tree Traversal

The `traversePageNode()` method recursively traverses the page tree:

**Algorithm**:
1. Check node `/Type`:
   - If "Pages": Intermediate node → recurse into `/Kids`
   - If "Page": Leaf node → create Page object
2. For Pages nodes:
   - Get `/Kids` array
   - Resolve each kid (may be reference)
   - Recursively traverse each kid
   - Pass current node as parent
3. For Page nodes:
   - Create Page object with parent
   - Add to flattened pages list

**Example Structure**:
```
Root Pages (Count: 4)
├── Pages (Count: 2)
│   ├── Page 1
│   └── Page 2
└── Pages (Count: 2)
    ├── Page 3
    └── Page 4
```

The algorithm flattens this to: `[Page1, Page2, Page3, Page4]`

### 4. CropBox Defaulting

CropBox is optional and defaults to MediaBox if not present:

```go
func (p *Page) CropBox() ([]float64, error) {
    box, err := p.getBox("CropBox")
    if err != nil {
        // CropBox defaults to MediaBox
        return p.MediaBox()
    }
    return box, nil
}
```

This follows PDF specification behavior.

### 5. Contents Handling

Page contents can be:
- **Single stream**: `/Contents <stream>`
- **Array of streams**: `/Contents [<stream1> <stream2> ...]`

The implementation handles both:
```go
switch v := contentsResolved.(type) {
case *core.Stream:
    return []core.Object{v}, nil
case core.Array:
    // Resolve each element
    ...
    return streams, nil
}
```

Always returns slice for consistency.

### 6. Comprehensive Test Suite (`pages_test.go` - 586 lines)

Created **18 test functions** covering all functionality:

#### Test Coverage

1. **TestNewCatalog** - Catalog creation
   - Creates catalog from dict
   - Checks Type

2. **TestCatalogPages** - Getting pages from catalog
   - Resolves /Pages reference
   - Returns pages dict

3. **TestCatalogVersion** - Version attribute
   - Gets /Version from catalog

4. **TestCatalogMetadata** - Metadata stream
   - Resolves /Metadata reference
   - Returns stream

5. **TestPageTreeFlatStructure** - Flat page tree
   - 3 pages directly under root
   - Count() returns 3
   - GetPage(0), GetPage(2) work
   - Pages() returns all 3

6. **TestPageTreeNestedStructure** - Nested page tree
   - 2-level tree: Root → 2 intermediate nodes → 4 pages
   - Correct traversal order
   - Pages() returns all 4 in correct order

7. **TestPageMediaBox** - MediaBox access
   - Gets MediaBox from page
   - Returns [0, 0, 612, 792]

8. **TestPageInheritableMediaBox** - MediaBox inheritance
   - Page without MediaBox
   - Inherits from parent Pages node

9. **TestPageCropBox** - CropBox access
   - Gets CropBox from page
   - Returns custom crop box

10. **TestPageCropBoxDefaultsToMediaBox** - CropBox defaulting
    - Page without CropBox
    - Returns MediaBox as default

11. **TestPageResources** - Resources access
    - Gets Resources dict
    - Contains Font entry

12. **TestPageInheritableResources** - Resources inheritance
    - Page without Resources
    - Inherits from parent

13. **TestPageContents** - Contents (single stream)
    - Gets contents stream
    - Returns as single-element slice

14. **TestPageContentsArray** - Contents (array)
    - Gets contents array
    - Returns all streams

15. **TestPageRotate** - Rotation attribute
    - Gets /Rotate value
    - Returns 90

16. **TestPageWidthHeight** - Page dimensions
    - Width() returns 612
    - Height() returns 792

17. **TestPageTreeOutOfBounds** - Error handling
    - Index too large → error
    - Negative index → error

18. **TestPageMissingMediaBox** - Required attribute
    - Page without MediaBox
    - Parent without MediaBox
    - Returns error

**Total: 20+ individual test cases across 18 test functions**

### 7. Mock Resolver for Testing

Created `mockResolver` for isolated testing:
```go
type mockResolver struct {
    objects map[int]core.Object
}
```

Implements ObjectResolver interface:
- `Resolve(obj)` - Resolves references
- `ResolveDeep(obj)` - Deep resolution
- `ResolveReference(ref)` - Resolves by number
- `AddObject(num, obj)` - Helper for test setup

Allows testing pages package without full PDF reader/resolver.

### 8. Performance Metrics

#### Test Execution
```
ok  	github.com/tsawler/tabula/pages	0.418s
```
All 18 tests complete in ~420ms.

#### Code Coverage
```
github.com/tsawler/tabula/pages	72.1% coverage
```

**Per-function coverage:**
```
NewCatalog         100.0%
Type (Catalog)      75.0%
Pages               70.0%
Metadata            70.0%
Version             75.0%
NewPageTree        100.0%
Count               71.4%
GetPage             83.3%
Pages (tree)        75.0%
loadPages           75.0%
traversePageNode    69.0%
NewPage            100.0%
MediaBox           100.0%
CropBox            100.0%
getBox              75.0%
Resources           75.0%
Contents            75.0%
Rotate              62.5%
Width               75.0%
Height              75.0%
```

**Analysis:**
- Core functionality (tree traversal, page access): 69-83% ✅
- Page properties (MediaBox, CropBox, etc.): 62-100% ✅
- Error paths and edge cases covered ✅

Overall 72.1% is good coverage for initial implementation.

### 9. Key Implementation Details

#### ObjectResolver Interface

The pages package uses an ObjectResolver interface:
```go
type ObjectResolver interface {
    Resolve(obj core.Object) (core.Object, error)
    ResolveDeep(obj core.Object) (core.Object, error)
    ResolveReference(ref core.IndirectRef) (core.Object, error)
}
```

This allows pages package to work with any resolver implementation (from Task 1.7).

#### Page Tree Caching

The page tree lazily loads and caches the flattened page list:
```go
func (t *PageTree) GetPage(index int) (*Page, error) {
    // Ensure pages are loaded
    if t.pages == nil {
        if err := t.loadPages(); err != nil {
            return nil, err
        }
    }
    return t.pages[index], nil
}
```

First call traverses tree and builds cache. Subsequent calls use cached list.

#### Error Handling

All methods use Go's error return pattern with wrapped errors:
```go
if err != nil {
    return nil, fmt.Errorf("failed to resolve /Pages: %w", err)
}
```

Errors include:
- Context (what operation failed)
- Original error (wrapped with %w)
- Relevant identifiers (page index, attribute name)

#### PDF Coordinate System

MediaBox format: `[llx lly urx ury]`
- llx, lly: Lower-left corner
- urx, ury: Upper-right corner
- Width = urx - llx
- Height = ury - lly

Standard US Letter: `[0 0 612 792]` (8.5" × 11" at 72 DPI)

### 10. Integration with Previous Components

The pages package integrates seamlessly with:

**From Task 1.6 (Reader)**:
- Reader provides catalog via `GetCatalog()`
- Catalog dictionary passed to `NewCatalog()`

**From Task 1.7 (Resolver)**:
- Resolver passed to Catalog, PageTree, Page constructors
- Used to resolve all indirect references
- Enables lazy loading of page tree nodes

**Example Integration**:
```go
// Open PDF (Task 1.6)
r, _ := reader.Open("document.pdf")
defer r.Close()

// Create resolver (Task 1.7)
resolver := resolver.NewResolver(r)

// Get catalog (Task 1.6 + Task 1.8)
catalogDict, _ := r.GetCatalog()
catalog := pages.NewCatalog(catalogDict, resolver)

// Get page tree (Task 1.8)
pagesDict, _ := catalog.Pages()
tree := pages.NewPageTree(pagesDict, resolver)

// Access pages (Task 1.8)
count, _ := tree.Count()
page, _ := tree.GetPage(0)
mediaBox, _ := page.MediaBox()
```

## Acceptance Criteria

From IMPLEMENTATION_PLAN.md (Task 1.8):
- ✅ **Parse document catalog** - Catalog struct with Type, Pages, Metadata, Version
- ✅ **Parse pages tree** - PageTree with recursive traversal
- ✅ **Implement page enumeration** - GetPage(index), Pages()
- ✅ **Get page by number** - GetPage(index) with bounds checking
- ✅ **Write page access tests** - 18 test functions, 20+ test cases

**Deliverable**: Page access API ✅
**Acceptance**: Can enumerate all pages ✅

## Files Created

1. **tabula/pages/pages.go** (NEW)
   - 414 lines
   - Complete page access implementation
   - Catalog, PageTree, Page types
   - Inheritable attribute handling

2. **tabula/pages/pages_test.go** (NEW)
   - 586 lines
   - 18 test functions
   - Mock resolver for testing
   - Flat and nested tree tests

## Statistics

- **Lines of Code Added**: ~1,000
- **Test Functions**: 18
- **Test Cases**: 20+
- **Coverage**: 72.1%
- **Time to Run Tests**: 0.418 seconds

## What's Next

**Task 1.9**: FlateDecode (Week 3, 12 hours)
- Implement zlib/deflate decompression
- Handle PNG predictors
- Support DecodeParms
- Test with real PDF streams

The pages package provides the foundation for accessing page content streams, which will be decompressed in Task 1.9 and parsed in Task 1.11.

## Notes

The pages package is production-ready:
- ✅ Handles flat and nested page trees correctly
- ✅ Supports inheritable attributes per PDF spec
- ✅ Clean abstraction (ObjectResolver interface)
- ✅ Efficient (lazy loading with caching)
- ✅ Handles all common page properties
- ✅ Good test coverage (72.1%)
- ✅ Clear error messages
- ✅ Well-documented code

### Design Decisions

**Why Lazy Loading?**
- Large PDFs may have thousands of pages
- Not all pages are typically accessed
- Lazy loading improves startup time
- Cache only what's needed

**Why Cache Flattened List?**
- Random access by index is common
- Traversing tree for each access is wasteful
- Trade memory for speed
- One-time traversal cost

**Why Separate Catalog/PageTree/Page?**
- Clear separation of concerns
- Catalog: Document-level properties
- PageTree: Page enumeration logic
- Page: Individual page properties
- Easier to test and maintain

**Why ObjectResolver Interface?**
- Decouples from specific resolver implementation
- Enables testing with mock resolver
- Follows dependency inversion principle
- Future-proof for alternative resolvers

### Known Limitations

1. **Page Labels**: Does not handle page labels (/PageLabels) for custom numbering (i, ii, iii, etc.). Can be added in future.

2. **Page Transitions**: Does not handle page transitions or durations. Not needed for document parsing.

3. **Other Inheritable Attributes**: Currently handles MediaBox, CropBox, Resources, Rotate. Other inheritable attributes (BleedBox, TrimBox, ArtBox) can be added if needed.

4. **Structural Parent Tree**: Does not parse structural parent tree for tagged PDFs. Will be added in Phase 4 if needed.

These limitations are acceptable for Phase 1. Core page enumeration and property access is complete.

## Example Usage

### Basic Usage
```go
// Open PDF
r, _ := reader.Open("document.pdf")
defer r.Close()

// Get catalog
resolver := resolver.NewResolver(r)
catalogDict, _ := r.GetCatalog()
catalog := pages.NewCatalog(catalogDict, resolver)

// Get page tree
pagesDict, _ := catalog.Pages()
tree := pages.NewPageTree(pagesDict, resolver)

// Get page count
count, _ := tree.Count()
fmt.Printf("Document has %d pages\n", count)

// Access first page
page, _ := tree.GetPage(0)
width, _ := page.Width()
height, _ := page.Height()
fmt.Printf("Page 1: %fx%f\n", width, height)
```

### Enumerate All Pages
```go
pages, _ := tree.Pages()
for i, page := range pages {
    mediaBox, _ := page.MediaBox()
    rotate := page.Rotate()
    fmt.Printf("Page %d: %v rotation=%d\n", i+1, mediaBox, rotate)
}
```

### Access Page Contents
```go
page, _ := tree.GetPage(0)
contents, _ := page.Contents()
for i, stream := range contents {
    s := stream.(*core.Stream)
    fmt.Printf("Content stream %d: %d bytes\n", i, len(s.Data))
}
```

## Integration Points

The pages package integrates with:
- **Reader (Task 1.6)**: Uses reader to get catalog dictionary
- **Resolver (Task 1.7)**: Uses resolver to follow all references
- **Content Streams (Task 1.11)**: Will use page.Contents() to access content streams
- **Text Extraction (Task 1.14)**: Will enumerate pages and extract text from each

## Production Readiness

The pages package is ready for production use:
- ✅ Correct implementation per PDF spec
- ✅ Handles flat and nested page trees
- ✅ Proper attribute inheritance
- ✅ Efficient lazy loading
- ✅ Comprehensive error handling
- ✅ Well-tested with good coverage
- ✅ Clean, maintainable code
- ✅ Clear API documentation

This completes Task 1.8 - Catalog & Pages!

## Phase 1 Progress Update

**Completed Tasks**:
- ✅ Task 1.1-1.2: Project setup & objects
- ✅ Task 1.3: Lexer (2.4M ops/sec)
- ✅ Task 1.4: Parser (2.8M ops/sec, 82.4% coverage)
- ✅ Task 1.5: XRef parsing (120K tables/sec, 79.5% coverage)
- ✅ Task 1.6: File reader (79.1% coverage)
- ✅ Task 1.7: Object resolver (86.6% coverage)
- ✅ Task 1.8: Catalog & pages (72.1% coverage)

**Overall Package Coverage**:
- Core: 79.5%
- Reader: 79.1%
- Resolver: 86.6%
- Pages: 72.1%
- **Average: ~79%**

**Progress**: 8 of 15 Phase 1 tasks complete (53%)

**Next Up**: Task 1.9 - FlateDecode (Stream Decompression)
