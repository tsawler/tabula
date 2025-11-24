# Performance Optimization Guide

## Overview

This document outlines strategies for achieving high performance and low memory usage when processing PDFs.

## Performance Targets

- **Parsing Speed**: 20-50 pages/second
- **Memory Usage**: < 100 MB for typical documents, < 1 GB for large documents
- **Concurrency**: Linear scaling up to CPU count
- **Table Detection**: < 100ms per page

## 1. Memory Management

### Streaming Architecture

Process PDFs page-by-page to avoid loading entire document into memory:

```go
type StreamingReader struct {
    file   *os.File
    xref   *XRefTable
    cached map[int]Object
}

func (r *StreamingReader) GetPage(pageNum int) (*model.Page, error) {
    // Lazy-load only the objects needed for this page
    pageRef := r.getPageReference(pageNum)
    pageDict := r.resolveObject(pageRef)

    // Parse only this page's content
    return r.parsePage(pageDict)
}
```

### Object Pooling

Reuse frequently allocated objects to reduce GC pressure:

```go
var (
    textFragmentPool = sync.Pool{
        New: func() interface{} {
            return &model.TextFragment{}
        },
    }

    bboxPool = sync.Pool{
        New: func() interface{} {
            return &model.BBox{}
        },
    }
)

func getTextFragment() *model.TextFragment {
    return textFragmentPool.Get().(*model.TextFragment)
}

func putTextFragment(tf *model.TextFragment) {
    *tf = model.TextFragment{} // Reset
    textFragmentPool.Put(tf)
}
```

### Lazy Decoding

Don't decode streams until needed:

```go
type Stream struct {
    dict    Dict
    rawData []byte
    decoded []byte
    once    sync.Once
}

func (s *Stream) Decoded() ([]byte, error) {
    var err error
    s.once.Do(func() {
        s.decoded, err = decodeStream(s.rawData, s.dict)
    })
    return s.decoded, err
}
```

### Memory Limits

Set hard limits to prevent OOM:

```go
type Config struct {
    MaxPageSize     int64  // Max bytes per page
    MaxImageSize    int64  // Max bytes per image
    MaxStreamSize   int64  // Max stream size
    MaxTotalMemory  int64  // Max total memory
}

func (r *Reader) parsePage(dict Dict) (*model.Page, error) {
    // Check memory usage
    if r.memoryUsage() > r.config.MaxTotalMemory {
        return nil, errors.New("memory limit exceeded")
    }

    // Parse page with limits
    page, err := r.parsePageWithLimits(dict)
    return page, err
}
```

## 2. Parsing Optimization

### Efficient Tokenization

Use buffered I/O and minimize allocations:

```go
type Parser struct {
    buf    []byte
    pos    int
    reader *bufio.Reader
}

func (p *Parser) nextToken() ([]byte, error) {
    // Skip whitespace without allocation
    for p.pos < len(p.buf) && isWhitespace(p.buf[p.pos]) {
        p.pos++
    }

    // Return slice of buffer (zero-copy)
    start := p.pos
    for p.pos < len(p.buf) && !isDelimiter(p.buf[p.pos]) {
        p.pos++
    }

    return p.buf[start:p.pos], nil
}
```

### Intern Strings

Deduplicate common strings (font names, operators):

```go
var internedStrings = make(map[string]string)
var internMutex sync.RWMutex

func intern(s string) string {
    internMutex.RLock()
    if existing, ok := internedStrings[s]; ok {
        internMutex.RUnlock()
        return existing
    }
    internMutex.RUnlock()

    internMutex.Lock()
    defer internMutex.Unlock()

    // Check again after acquiring write lock
    if existing, ok := internedStrings[s]; ok {
        return existing
    }

    internedStrings[s] = s
    return s
}
```

### Fast Number Parsing

Use optimized number parsing:

```go
func parseIntFast(b []byte) (int64, error) {
    if len(b) == 0 {
        return 0, errors.New("empty")
    }

    negative := b[0] == '-'
    if negative {
        b = b[1:]
    }

    var n int64
    for _, c := range b {
        if c < '0' || c > '9' {
            return 0, errors.New("invalid digit")
        }
        n = n*10 + int64(c-'0')
    }

    if negative {
        n = -n
    }
    return n, nil
}
```

## 3. Parallel Processing

### Page-Level Parallelism

Process pages concurrently:

```go
func (r *Reader) ParseParallel() (*model.Document, error) {
    pageCount := r.getPageCount()
    pages := make([]*model.Page, pageCount)

    // Worker pool
    workers := runtime.NumCPU()
    jobs := make(chan int, pageCount)
    results := make(chan pageResult, pageCount)

    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for pageNum := range jobs {
                page, err := r.GetPage(pageNum)
                results <- pageResult{pageNum, page, err}
            }
        }()
    }

    // Send jobs
    go func() {
        for i := 1; i <= pageCount; i++ {
            jobs <- i
        }
        close(jobs)
    }()

    // Collect results
    go func() {
        wg.Wait()
        close(results)
    }()

    for result := range results {
        if result.err != nil {
            return nil, result.err
        }
        pages[result.pageNum-1] = result.page
    }

    return &model.Document{Pages: pages}, nil
}
```

### Table Detection Parallelism

Detect tables on multiple pages concurrently:

```go
func (d *Detector) DetectParallel(pages []*model.Page) ([][]*model.Table, error) {
    results := make([][]*model.Table, len(pages))

    var wg sync.WaitGroup
    sem := make(chan struct{}, runtime.NumCPU())

    for i, page := range pages {
        wg.Add(1)
        go func(idx int, p *model.Page) {
            defer wg.Done()

            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release

            tables, _ := d.Detect(p)
            results[idx] = tables
        }(i, page)
    }

    wg.Wait()
    return results, nil
}
```

## 4. Table Detection Optimization

### Spatial Indexing

Use spatial data structures for fast lookups:

```go
type SpatialIndex struct {
    grid     [][]*model.TextFragment
    cellSize float64
    bounds   model.BBox
}

func NewSpatialIndex(fragments []model.TextFragment, cellSize float64) *SpatialIndex {
    // Calculate bounds
    bounds := calculateBounds(fragments)

    // Create grid
    cols := int(bounds.Width/cellSize) + 1
    rows := int(bounds.Height/cellSize) + 1
    grid := make([][]*model.TextFragment, rows*cols)

    // Insert fragments
    for i := range fragments {
        cell := getCellIndex(&fragments[i], cellSize, bounds)
        grid[cell] = append(grid[cell], &fragments[i])
    }

    return &SpatialIndex{grid, cellSize, bounds}
}

func (si *SpatialIndex) Query(bbox model.BBox) []*model.TextFragment {
    // Get cells that intersect bbox
    cells := si.getCells(bbox)

    var results []*model.TextFragment
    seen := make(map[*model.TextFragment]bool)

    for _, cell := range cells {
        for _, frag := range si.grid[cell] {
            if !seen[frag] && bbox.Intersects(frag.BBox) {
                results = append(results, frag)
                seen[frag] = true
            }
        }
    }

    return results
}
```

### Early Termination

Skip unlikely table candidates early:

```go
func (d *GeometricDetector) detectTableInCluster(fragments []model.TextFragment) *model.Table {
    // Quick rejection tests
    if len(fragments) < d.config.MinRows * d.config.MinCols {
        return nil
    }

    // Check if fragments show any alignment
    if !d.hasAlignment(fragments) {
        return nil
    }

    // Check if there's enough whitespace/structure
    if !d.hasStructure(fragments) {
        return nil
    }

    // Now do expensive grid construction
    grid := d.buildGrid(fragments)
    // ... rest of detection
}
```

### Incremental Processing

Build grid incrementally:

```go
type GridBuilder struct {
    rowCandidates map[float64]int
    colCandidates map[float64]int
    tolerance     float64
}

func (gb *GridBuilder) AddFragment(frag model.TextFragment) {
    // Merge with existing candidates
    gb.addYCoordinate(frag.BBox.Top())
    gb.addYCoordinate(frag.BBox.Bottom())
    gb.addXCoordinate(frag.BBox.Left())
    gb.addXCoordinate(frag.BBox.Right())
}

func (gb *GridBuilder) addYCoordinate(y float64) {
    // Find if there's a nearby candidate
    for existing := range gb.rowCandidates {
        if math.Abs(existing-y) < gb.tolerance {
            // Merge into existing
            gb.rowCandidates[existing]++
            return
        }
    }
    gb.rowCandidates[y] = 1
}
```

## 5. Caching Strategies

### Font Caching

Cache decoded fonts:

```go
type FontCache struct {
    fonts sync.Map // map[string]*Font
}

func (fc *FontCache) Get(name string, dict Dict) (*Font, error) {
    // Try cache first
    if cached, ok := fc.fonts.Load(name); ok {
        return cached.(*Font), nil
    }

    // Parse font
    font, err := parseFont(dict)
    if err != nil {
        return nil, err
    }

    // Store in cache
    fc.fonts.Store(name, font)
    return font, nil
}
```

### Content Stream Caching

Cache parsed content streams:

```go
type ContentCache struct {
    cache *lru.Cache
}

func (cc *ContentCache) GetOperations(stream *Stream) ([]Operation, error) {
    // Use stream data hash as key
    key := hash(stream.Data)

    if ops, ok := cc.cache.Get(key); ok {
        return ops.([]Operation), nil
    }

    ops, err := parseContentStream(stream.Data)
    if err != nil {
        return nil, err
    }

    cc.cache.Add(key, ops)
    return ops, nil
}
```

## 6. Benchmarking

### Benchmark Suite

```go
func BenchmarkParsePage(b *testing.B) {
    data := loadTestPDF("sample.pdf")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        reader := NewReader(bytes.NewReader(data))
        reader.GetPage(1)
    }
}

func BenchmarkTableDetection(b *testing.B) {
    page := loadTestPage("table_page.pdf")
    detector := NewGeometricDetector()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        detector.Detect(page)
    }
}

func BenchmarkTextExtraction(b *testing.B) {
    stream := loadContentStream("text_page.pdf")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        extractor := NewTextExtractor()
        extractor.Extract(stream)
    }
}
```

### Memory Profiling

```go
func TestMemoryUsage(t *testing.T) {
    var m1, m2 runtime.MemStats

    runtime.GC()
    runtime.ReadMemStats(&m1)

    // Process PDF
    reader := NewReader(openFile("large.pdf"))
    doc, _ := reader.Parse()

    runtime.ReadMemStats(&m2)

    allocated := m2.Alloc - m1.Alloc
    if allocated > 100*1024*1024 { // 100 MB limit
        t.Errorf("Memory usage too high: %d bytes", allocated)
    }
}
```

## 7. Profile-Guided Optimization

Use Go's profiler to find bottlenecks:

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Trace
go test -trace=trace.out -bench=.
go tool trace trace.out
```

## Performance Checklist

- [ ] Use buffered I/O
- [ ] Implement object pooling for frequent allocations
- [ ] Lazy-load objects and decode streams
- [ ] Process pages in parallel
- [ ] Use spatial indexing for layout analysis
- [ ] Cache fonts and parsed objects
- [ ] Intern common strings
- [ ] Set memory limits
- [ ] Profile and benchmark regularly
- [ ] Use efficient data structures (slices over maps when possible)
- [ ] Avoid reflection in hot paths
- [ ] Reuse buffers
- [ ] Minimize string concatenation (use strings.Builder)

## Expected Performance

With these optimizations:

- **Small PDFs (< 10 pages)**: < 100ms total
- **Medium PDFs (10-100 pages)**: 2-5 seconds
- **Large PDFs (100-1000 pages)**: 20-50 seconds
- **Memory**: 50-100 MB for typical documents

The key is balancing between memory usage and speed, and using parallel processing effectively.
