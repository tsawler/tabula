# Object Streams Implementation Plan

This document outlines the implementation of PDF 1.5+ Object Streams (ObjStm) support in tabula.

## Background

### What are Object Streams?

Object streams (PDF 1.5+) are a compression optimization that stores multiple PDF objects inside a single stream object. Instead of each object being written separately in the file, they are packed together and compressed.

**Traditional PDF (1.0-1.4):**
```
5 0 obj
<< /Type /Catalog /Pages 6 0 R >>
endobj

6 0 obj
<< /Type /Pages /Kids [7 0 R] /Count 1 >>
endobj
```

**With Object Streams (1.5+):**
```
10 0 obj
<< /Type /ObjStm /N 2 /First 10 /Filter /FlateDecode /Length 89 >>
stream
5 0 6 20 << /Type /Catalog /Pages 6 0 R >><< /Type /Pages /Kids [7 0 R] /Count 1 >>
endstream
endobj
```

Objects 5 and 6 are now stored inside object 10's stream.

### How XRef Streams Reference Object Streams

In an XRef stream, entries have three types:

| Type | Field 1 | Field 2 | Description |
|------|---------|---------|-------------|
| 0 | Next free obj | Gen number | Free entry |
| 1 | Byte offset | Gen number | Uncompressed object |
| 2 | ObjStm number | Index in stream | **Compressed object** |

Currently, tabula parses Type 2 entries but stores them incorrectly:
- `Offset` = Object stream number (should be a separate field)
- `Generation` = Index within stream (generation is always 0 for compressed objects)

## Current State

### What Works
- XRef stream detection and parsing (`core/xref.go:245-359`)
- Binary entry parsing with field widths (`core/xref.go:360-412`)
- Type 2 entries are recognized (`core/xref.go:401-407`)

### What's Missing
1. **XRefEntry lacks Type 2 distinction** - No way to know if an entry is compressed
2. **No object stream parser** - Cannot extract objects from ObjStm
3. **Reader doesn't handle compressed objects** - Will fail or return wrong data
4. **No tests with real compressed PDFs**

### Current Code (the TODO)

From `core/xref.go:401-407`:
```go
case 2:
    // Object in object stream (PDF 1.5+)
    // For now, treat as in-use with special offset encoding
    // TODO: Full object stream support in a future task
    entry.InUse = true
    entry.Offset = field1          // Object stream number
    entry.Generation = int(field2) // Index within stream
```

## Implementation Plan

### Phase 1: Data Structure Updates

#### 1.1 Update XRefEntry

**File:** `core/xref.go`

Add fields to distinguish compressed objects:

```go
type XRefEntry struct {
    Offset       int64 // Byte offset (Type 1) or object stream number (Type 2)
    Generation   int   // Generation number (Type 0/1) or index in stream (Type 2)
    InUse        bool  // true if object is in use
    Compressed   bool  // true if object is in an object stream (Type 2)
    ObjStmNum    int   // Object stream number (only valid if Compressed=true)
    ObjStmIndex  int   // Index within object stream (only valid if Compressed=true)
}
```

Alternative (cleaner): Use a type enum:

```go
type XRefEntryType int

const (
    XRefEntryFree       XRefEntryType = 0
    XRefEntryUncompressed XRefEntryType = 1
    XRefEntryCompressed   XRefEntryType = 2
)

type XRefEntry struct {
    Type         XRefEntryType
    Offset       int64 // Byte offset (uncompressed) or ObjStm number (compressed)
    Generation   int   // Generation (uncompressed) or index in stream (compressed)
    InUse        bool  // Derived: Type != XRefEntryFree
}
```

#### 1.2 Update XRef Stream Parsing

**File:** `core/xref.go`

Update `parseXRefStreamEntry()` to populate new fields correctly:

```go
case 2:
    entry.Type = XRefEntryCompressed
    entry.InUse = true
    entry.Offset = field1      // Object stream number
    entry.Generation = int(field2) // Index within stream
```

### Phase 2: Object Stream Parser

#### 2.1 Create ObjectStream Type

**File:** `core/objstm.go` (new file)

```go
// ObjectStream represents a PDF Object Stream (Type /ObjStm)
type ObjectStream struct {
    Stream  *Stream           // The underlying stream object
    N       int               // Number of objects in stream
    First   int               // Byte offset of first object in decoded data
    Extends *IndirectRef      // Optional reference to another ObjStm
    objects map[int]Object    // Cached parsed objects (index -> object)
    offsets []objectOffset    // Parsed offset pairs
}

type objectOffset struct {
    ObjNum int // Object number
    Offset int // Byte offset within decoded data (after First)
}
```

#### 2.2 Implement Object Stream Parsing

```go
// ParseObjectStream parses an object stream and returns an ObjectStream
func ParseObjectStream(stream *Stream) (*ObjectStream, error) {
    // 1. Verify /Type is /ObjStm
    // 2. Extract /N (number of objects)
    // 3. Extract /First (offset to first object data)
    // 4. Extract optional /Extends
    // 5. Decode the stream data
    // 6. Parse the header (N pairs of: objNum offset)
    // 7. Return ObjectStream ready to extract objects
}

// GetObject extracts an object by its index within the stream
func (os *ObjectStream) GetObject(index int) (Object, error) {
    // 1. Check cache
    // 2. Find offset for this index
    // 3. Parse object at that offset
    // 4. Cache and return
}
```

#### 2.3 Object Stream Header Format

The decoded stream data has this structure:
```
objNum1 offset1 objNum2 offset2 ... objNumN offsetN [object1][object2]...[objectN]
```

Example (decoded):
```
5 0 6 15
<< /Type /Catalog /Pages 6 0 R >><< /Type /Pages /Kids [7 0 R] /Count 1 >>
```

- Object 5 is at offset 0 from `/First`
- Object 6 is at offset 15 from `/First`
- `/First` points to where `<< /Type /Catalog...` begins

### Phase 3: Reader Integration

#### 3.1 Update Object Resolution

**File:** `reader/reader.go` (or `core/resolver.go`)

The object resolver must handle compressed objects:

```go
func (r *Reader) ResolveObject(objNum, gen int) (Object, error) {
    entry := r.xref.GetEntry(objNum)
    if entry == nil {
        return nil, fmt.Errorf("object %d not found", objNum)
    }

    if !entry.InUse {
        return nil, fmt.Errorf("object %d is free", objNum)
    }

    if entry.Type == XRefEntryCompressed {
        return r.resolveCompressedObject(objNum, entry)
    }

    return r.resolveUncompressedObject(objNum, entry)
}

func (r *Reader) resolveCompressedObject(objNum int, entry *XRefEntry) (Object, error) {
    objStmNum := int(entry.Offset)
    index := entry.Generation

    // Get or parse the object stream
    objStm, err := r.getObjectStream(objStmNum)
    if err != nil {
        return nil, fmt.Errorf("failed to get object stream %d: %w", objStmNum, err)
    }

    // Extract the object at the given index
    return objStm.GetObject(index)
}
```

#### 3.2 Object Stream Cache

Cache parsed object streams to avoid re-parsing:

```go
type Reader struct {
    // ... existing fields
    objStmCache map[int]*ObjectStream // Cache of parsed object streams
}
```

### Phase 4: Testing

#### 4.1 Unit Tests

**File:** `core/objstm_test.go`

```go
func TestParseObjectStreamHeader(t *testing.T) {
    // Test parsing "5 0 6 15" header
}

func TestObjectStreamGetObject(t *testing.T) {
    // Test extracting objects by index
}

func TestObjectStreamWithExtends(t *testing.T) {
    // Test /Extends chaining
}
```

#### 4.2 Integration Tests

**File:** `reader/reader_objstm_test.go`

```go
func TestReadCompressedObject(t *testing.T) {
    // Test reading an object that's in an object stream
}

func TestMixedCompressedUncompressed(t *testing.T) {
    // Test PDF with both compressed and uncompressed objects
}
```

#### 4.3 Test PDFs

Create or obtain test PDFs with object streams:
- Simple PDF with all objects in one ObjStm
- PDF with multiple ObjStms
- PDF with /Extends chains
- PDF with mixed compressed/uncompressed objects
- Real-world PDFs from various generators

### Phase 5: Edge Cases

#### 5.1 Recursive Object Streams

An object stream cannot contain:
- Stream objects (including other object streams)
- Objects with generation number > 0
- The document's encryption dictionary
- The document's catalog (usually)

Validate these constraints.

#### 5.2 /Extends Handling

Object streams can extend others:
```
<< /Type /ObjStm /N 5 /First 20 /Extends 15 0 R >>
```

Object 15 is another ObjStm that this one extends. Must handle chains.

#### 5.3 Hybrid PDFs

PDFs can mix:
- Traditional XRef tables with XRef streams
- Compressed and uncompressed objects
- Multiple object streams

Ensure all combinations work.

## File Changes Summary

| File | Change |
|------|--------|
| `core/xref.go` | Add `Type`/`Compressed` fields to XRefEntry |
| `core/objstm.go` | **New file** - ObjectStream parsing |
| `core/objstm_test.go` | **New file** - Unit tests |
| `reader/reader.go` | Update object resolution for compressed objects |
| `reader/reader_test.go` | Integration tests |

## Dependencies

This implementation requires:
- Working stream decoding (`core/stream.go`) - **Already implemented**
- Working object parser (`core/parser.go`) - **Already implemented**
- Working XRef stream parsing (`core/xref.go`) - **Already implemented**

## Success Criteria

1. PDFs with object streams parse correctly
2. All objects (compressed and uncompressed) are accessible
3. No performance regression for PDFs without object streams
4. Test coverage > 80% for new code
5. Works with real-world PDFs from various generators (Adobe, Chrome print-to-PDF, etc.)

## Estimated Scope

- **XRefEntry updates:** ~20 lines changed
- **ObjectStream parser:** ~150-200 lines new code
- **Reader integration:** ~50-100 lines changed
- **Tests:** ~200-300 lines
- **Total:** ~400-600 lines of code

## References

- PDF 1.7 Specification (ISO 32000-1:2008)
  - Section 7.5.7: Object Streams
  - Section 7.5.8: Cross-Reference Streams
  - Table 16: Additional entries specific to an object stream dictionary
- Existing code:
  - `core/xref.go:401-407` - Current TODO
  - `core/stream.go` - Stream decoding
  - `core/parser.go` - Object parsing

## Post-Implementation

After implementing object streams:

1. Update README claim to be fully accurate
2. Add object stream support to feature list
3. Consider adding `/ObjStm` to PDF writing (future enhancement)
4. Document any limitations discovered during implementation
