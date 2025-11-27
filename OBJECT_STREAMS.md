# Object Streams Implementation

This document describes the PDF 1.5+ Object Streams (ObjStm) support in tabula.

## Status: IMPLEMENTED

Object stream support was implemented on 2024-11-27. All phases are complete.

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

## Implementation Summary

### Phase 1: Data Structure Updates - COMPLETE

**File:** `core/xref.go`

Added `XRefEntryType` enum to distinguish entry types:

```go
type XRefEntryType int

const (
    XRefEntryFree         XRefEntryType = 0
    XRefEntryUncompressed XRefEntryType = 1
    XRefEntryCompressed   XRefEntryType = 2
)

type XRefEntry struct {
    Type       XRefEntryType // Entry type (free, uncompressed, or compressed)
    Offset     int64         // Byte offset (uncompressed) or ObjStm number (compressed)
    Generation int           // Generation (uncompressed) or index in stream (compressed)
    InUse      bool          // true if object is in use (Type != XRefEntryFree)
}
```

Updated `parseXRefStreamEntry()` and `parseEntry()` to set the `Type` field correctly.

### Phase 2: Object Stream Parser - COMPLETE

**File:** `core/objstm.go` (new file, ~250 lines)

Created `ObjectStream` type with the following API:

```go
// Create from a stream object
func NewObjectStream(stream *Stream) (*ObjectStream, error)

// Accessors
func (os *ObjectStream) N() int                    // Number of objects
func (os *ObjectStream) First() int                // Byte offset to first object
func (os *ObjectStream) Extends() *IndirectRef     // Optional extension reference

// Object extraction
func (os *ObjectStream) GetObjectByIndex(index int) (Object, int, error)
func (os *ObjectStream) GetObjectByNumber(objNum int) (Object, int, error)
func (os *ObjectStream) ObjectNumbers() ([]int, error)
func (os *ObjectStream) ContainsObject(objNum int) (bool, error)
```

Features:
- Validates /Type, /N, /First parameters
- Parses header (object number/offset pairs)
- Extracts objects by index or object number
- Caches decoded stream data and parsed objects
- Supports /Extends reference (for chained object streams)

**File:** `core/objstm_test.go` (new file, ~300 lines)

Comprehensive tests for:
- ObjectStream creation and validation
- Object extraction by index and number
- Caching behavior
- Error handling
- Edge cases

### Phase 3: Reader Integration - COMPLETE

**File:** `reader/reader.go`

Added `objStmCache` field to cache parsed object streams:

```go
type Reader struct {
    // ... existing fields
    objStmCache map[int]*core.ObjectStream
}
```

Refactored `GetObject()` to handle both entry types:

```go
func (r *Reader) GetObject(objNum int) (core.Object, error) {
    // Check cache, lookup XRef entry...

    switch entry.Type {
    case core.XRefEntryCompressed:
        return r.getCompressedObject(objNum, entry)
    case core.XRefEntryUncompressed:
        return r.getUncompressedObject(objNum, entry)
    }
}
```

Added helper methods:
- `getUncompressedObject()` - reads object directly from file
- `getCompressedObject()` - extracts from object stream
- `getObjectStream()` - loads and caches object streams
- `ObjectStreamCacheSize()` - returns number of cached streams

Updated `ClearCache()` to clear both object and object stream caches.

### Phase 4: Testing - COMPLETE

All unit tests pass:
- `core/objstm_test.go` - ObjectStream parsing tests
- `core/xref_test.go` - Updated to validate Type field
- `core/xref_stream_test.go` - Updated to validate Type field
- `reader/reader_test.go` - Existing tests still pass

## File Changes Summary

| File | Change | Lines |
|------|--------|-------|
| `core/xref.go` | Added `XRefEntryType` enum, updated `XRefEntry` struct | ~40 |
| `core/xref_test.go` | Added `TestXRefEntryType`, updated existing tests | ~50 |
| `core/xref_stream_test.go` | Updated to validate Type field | ~15 |
| `core/objstm.go` | **New file** - ObjectStream implementation | ~250 |
| `core/objstm_test.go` | **New file** - Unit tests | ~300 |
| `reader/reader.go` | Added object stream support to GetObject | ~100 |

**Total:** ~755 lines of code

## How It Works

1. **XRef Parsing**: When parsing XRef streams, Type 2 entries are marked with `XRefEntryCompressed`
2. **Object Loading**: `Reader.GetObject()` checks the entry type
3. **Compressed Objects**: For compressed entries:
   - `entry.Offset` = object stream number
   - `entry.Generation` = index within stream
4. **Object Stream Loading**: The reader loads and caches the object stream
5. **Extraction**: The object is extracted by index from the decoded stream
6. **Caching**: Both the object stream and extracted objects are cached

## Remaining Work

### Not Yet Implemented

1. **Integration tests with real PDFs** - The current tests use synthetic data. Testing with real-world PDFs from various generators (Adobe, Chrome, etc.) would increase confidence.

### Intentionally Not Implemented

1. **`/Extends` chain following** - The `/Extends` field is parsed and accessible via `ObjectStream.Extends()`, but chains are not automatically followed. This is intentional because:
   - The XRef table already contains the complete mapping of which object is in which stream
   - `Reader.GetObject()` uses XRef lookup, not stream traversal
   - Most PDF libraries (including qpdf) preserve but ignore `/Extends`
   - Real-world PDFs rarely use `/Extends` chains

### Future Enhancements

1. **PDF Writing** - Add support for writing object streams when creating PDFs
2. **Memory optimization** - Option to not cache large object streams

## References

- PDF 1.7 Specification (ISO 32000-1:2008)
  - Section 7.5.7: Object Streams
  - Section 7.5.8: Cross-Reference Streams
  - Table 16: Additional entries specific to an object stream dictionary

## Usage

Object stream support is transparent to users. Simply use `Reader.GetObject()` as usual:

```go
reader, err := reader.Open("document.pdf")
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

// Works for both compressed and uncompressed objects
obj, err := reader.GetObject(5)
if err != nil {
    log.Fatal(err)
}
```

The reader automatically handles the complexity of extracting objects from object streams.
