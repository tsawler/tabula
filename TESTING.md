# Testing Strategy

## Overview

Comprehensive testing strategy for the PDF library covering unit tests, integration tests, corpus testing, and benchmarks.

## Test Structure

```
pdf/
├── model/
│   ├── document_test.go
│   ├── table_test.go
│   └── geometry_test.go
├── core/
│   ├── parser_test.go
│   ├── object_test.go
│   └── xref_test.go
├── tables/
│   ├── geometric_test.go
│   └── detector_test.go
└── testdata/
    ├── simple.pdf
    ├── tables.pdf
    ├── complex.pdf
    ├── encrypted.pdf
    └── corpus/
        └── ... (large test corpus)
```

## Unit Tests

Test individual components in isolation.

### Example: Object Parser Tests

```go
package core

import (
    "bytes"
    "testing"
)

func TestParseBoolean(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    Bool
        wantErr bool
    }{
        {"true", "true", Bool(true), false},
        {"false", "false", Bool(false), false},
        {"invalid", "truee", nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := NewParser(bytes.NewReader([]byte(tt.input)))
            got, err := parser.ParseObject()

            if (err != nil) != tt.wantErr {
                t.Errorf("ParseObject() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !tt.wantErr && got != tt.want {
                t.Errorf("ParseObject() = %v, want %v", got, tt.want)
            }
        })
    }
}

func TestParseArray(t *testing.T) {
    input := "[1 2 /Name (string)]"
    parser := NewParser(bytes.NewReader([]byte(input)))

    obj, err := parser.ParseObject()
    if err != nil {
        t.Fatalf("ParseObject() error = %v", err)
    }

    arr, ok := obj.(Array)
    if !ok {
        t.Fatalf("Expected Array, got %T", obj)
    }

    if len(arr) != 4 {
        t.Errorf("Expected 4 elements, got %d", len(arr))
    }

    // Verify element types
    if _, ok := arr[0].(Int); !ok {
        t.Errorf("Element 0: expected Int, got %T", arr[0])
    }
    if _, ok := arr[2].(Name); !ok {
        t.Errorf("Element 2: expected Name, got %T", arr[2])
    }
}
```

### Example: Table Tests

```go
package model

import "testing"

func TestTableCreation(t *testing.T) {
    table := NewTable(3, 4)

    if table.RowCount() != 3 {
        t.Errorf("Expected 3 rows, got %d", table.RowCount())
    }

    if table.ColCount() != 4 {
        t.Errorf("Expected 4 columns, got %d", table.ColCount())
    }
}

func TestTableMarkdown(t *testing.T) {
    table := NewTable(2, 2)
    table.SetCell(0, 0, Cell{Text: "A"})
    table.SetCell(0, 1, Cell{Text: "B"})
    table.SetCell(1, 0, Cell{Text: "1"})
    table.SetCell(1, 1, Cell{Text: "2"})

    md := table.ToMarkdown()

    expected := "| A | B |\n|---|---|\n| 1 | 2 |\n"
    if md != expected {
        t.Errorf("ToMarkdown() = %q, want %q", md, expected)
    }
}

func TestTableCSV(t *testing.T) {
    table := NewTable(2, 2)
    table.SetCell(0, 0, Cell{Text: "Name"})
    table.SetCell(0, 1, Cell{Text: "Value"})
    table.SetCell(1, 0, Cell{Text: "Test,Item"})
    table.SetCell(1, 1, Cell{Text: "123"})

    csv := table.ToCSV()

    expected := "Name,Value\n\"Test,Item\",123\n"
    if csv != expected {
        t.Errorf("ToCSV() = %q, want %q", csv, expected)
    }
}
```

### Example: Geometry Tests

```go
package model

import (
    "math"
    "testing"
)

func TestBBoxIntersection(t *testing.T) {
    b1 := BBox{X: 0, Y: 0, Width: 10, Height: 10}
    b2 := BBox{X: 5, Y: 5, Width: 10, Height: 10}

    if !b1.Intersects(b2) {
        t.Error("Expected boxes to intersect")
    }

    intersection := b1.Intersection(b2)
    expected := BBox{X: 5, Y: 5, Width: 5, Height: 5}

    if !bboxEqual(intersection, expected) {
        t.Errorf("Intersection = %+v, want %+v", intersection, expected)
    }
}

func TestBBoxUnion(t *testing.T) {
    b1 := BBox{X: 0, Y: 0, Width: 10, Height: 10}
    b2 := BBox{X: 5, Y: 5, Width: 10, Height: 10}

    union := b1.Union(b2)
    expected := BBox{X: 0, Y: 0, Width: 15, Height: 15}

    if !bboxEqual(union, expected) {
        t.Errorf("Union = %+v, want %+v", union, expected)
    }
}

func bboxEqual(a, b BBox) bool {
    return math.Abs(a.X-b.X) < 0.001 &&
        math.Abs(a.Y-b.Y) < 0.001 &&
        math.Abs(a.Width-b.Width) < 0.001 &&
        math.Abs(a.Height-b.Height) < 0.001
}
```

## Integration Tests

Test complete workflows end-to-end.

```go
package pdf_test

import (
    "os"
    "testing"

    "github.com/tsawler/tabula/reader"
)

func TestReadSimplePDF(t *testing.T) {
    file, err := os.Open("testdata/simple.pdf")
    if err != nil {
        t.Fatalf("Failed to open test file: %v", err)
    }
    defer file.Close()

    r, err := reader.New(file)
    if err != nil {
        t.Fatalf("Failed to create reader: %v", err)
    }

    doc, err := r.Parse()
    if err != nil {
        t.Fatalf("Failed to parse PDF: %v", err)
    }

    if doc.PageCount() != 1 {
        t.Errorf("Expected 1 page, got %d", doc.PageCount())
    }

    text := doc.ExtractText()
    if text == "" {
        t.Error("Expected non-empty text")
    }
}

func TestTableExtraction(t *testing.T) {
    file, err := os.Open("testdata/tables.pdf")
    if err != nil {
        t.Fatalf("Failed to open test file: %v", err)
    }
    defer file.Close()

    r, err := reader.New(file)
    if err != nil {
        t.Fatalf("Failed to create reader: %v", err)
    }

    doc, err := r.Parse()
    if err != nil {
        t.Fatalf("Failed to parse PDF: %v", err)
    }

    tables := doc.ExtractTables()
    if len(tables) == 0 {
        t.Error("Expected at least one table")
    }

    for i, table := range tables {
        if table.RowCount() < 2 {
            t.Errorf("Table %d: expected at least 2 rows, got %d", i, table.RowCount())
        }
        if table.ColCount() < 2 {
            t.Errorf("Table %d: expected at least 2 columns, got %d", i, table.ColCount())
        }
    }
}
```

## Corpus Testing

Test against a diverse corpus of real-world PDFs.

```go
package pdf_test

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/tsawler/tabula/reader"
)

func TestCorpus(t *testing.T) {
    corpusDir := "testdata/corpus"

    err := filepath.Walk(corpusDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if info.IsDir() || filepath.Ext(path) != ".pdf" {
            return nil
        }

        t.Run(filepath.Base(path), func(t *testing.T) {
            testPDF(t, path)
        })

        return nil
    })

    if err != nil {
        t.Fatalf("Failed to walk corpus: %v", err)
    }
}

func testPDF(t *testing.T, path string) {
    file, err := os.Open(path)
    if err != nil {
        t.Fatalf("Failed to open: %v", err)
    }
    defer file.Close()

    r, err := reader.New(file)
    if err != nil {
        t.Fatalf("Failed to create reader: %v", err)
    }

    doc, err := r.Parse()
    if err != nil {
        t.Fatalf("Failed to parse: %v", err)
    }

    // Basic sanity checks
    if doc.PageCount() == 0 {
        t.Error("Expected at least one page")
    }

    // Try to extract text from each page
    for i, page := range doc.Pages {
        text := page.ExtractText()
        if len(text) > 1000000 {
            t.Errorf("Page %d: text too long (%d bytes)", i+1, len(text))
        }
    }
}
```

## Benchmarks

Measure performance.

```go
package core

import (
    "bytes"
    "testing"
)

func BenchmarkParseInt(b *testing.B) {
    input := []byte("12345")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        parser := NewParser(bytes.NewReader(input))
        parser.ParseObject()
    }
}

func BenchmarkParseDict(b *testing.B) {
    input := []byte("<</Type /Page /MediaBox [0 0 612 792] /Contents 5 0 R>>")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        parser := NewParser(bytes.NewReader(input))
        parser.ParseObject()
    }
}
```

```go
package tables

import (
    "testing"

    "github.com/tsawler/tabula/model"
)

func BenchmarkTableDetection(b *testing.B) {
    // Load test page with table
    page := loadTestPage("table_page.pdf")
    detector := NewGeometricDetector()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        detector.Detect(page)
    }
}

func BenchmarkGridConstruction(b *testing.B) {
    fragments := generateTestFragments(100)
    detector := NewGeometricDetector()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        detector.buildGrid(fragments, nil)
    }
}
```

## Fuzzing

Test parser robustness with random inputs.

```go
package core

import "testing"

func FuzzParser(f *testing.F) {
    // Seed corpus
    f.Add([]byte("true"))
    f.Add([]byte("123"))
    f.Add([]byte("/Name"))
    f.Add([]byte("[1 2 3]"))
    f.Add([]byte("<</Key /Value>>"))

    f.Fuzz(func(t *testing.T, data []byte) {
        parser := NewParser(bytes.NewReader(data))
        _, _ = parser.ParseObject() // Should not crash
    })
}
```

## Test Coverage

Aim for > 80% code coverage:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Continuous Integration

GitHub Actions workflow:

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Run benchmarks
        run: go test -bench=. -benchmem ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out

  corpus:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Download corpus
        run: ./scripts/download_corpus.sh

      - name: Run corpus tests
        run: go test -v ./... -tags corpus
```

## Test Data

Maintain a diverse test corpus:

- **Simple PDFs** - Basic text, single page
- **Complex PDFs** - Multiple pages, fonts, images
- **Tables** - Various table layouts
- **Encrypted PDFs** - Password-protected
- **Compressed PDFs** - Object streams, compressed XRef
- **Malformed PDFs** - Edge cases, spec violations

## Golden Tests

Compare output against known-good results:

```go
func TestTableExtractionGolden(t *testing.T) {
    file, _ := os.Open("testdata/tables.pdf")
    defer file.Close()

    r, _ := reader.New(file)
    doc, _ := r.Parse()
    tables := doc.ExtractTables()

    // Compare with golden file
    golden, _ := os.ReadFile("testdata/tables.golden.json")
    actual, _ := json.Marshal(tables)

    if !bytes.Equal(actual, golden) {
        // Update golden file if GOLDEN_UPDATE env var is set
        if os.Getenv("GOLDEN_UPDATE") == "1" {
            os.WriteFile("testdata/tables.golden.json", actual, 0644)
            t.Log("Golden file updated")
        } else {
            t.Errorf("Output differs from golden file")
        }
    }
}
```

## Testing Checklist

- [ ] Unit tests for all core components
- [ ] Integration tests for main workflows
- [ ] Corpus testing with diverse PDFs
- [ ] Benchmarks for performance-critical code
- [ ] Fuzzing for parser robustness
- [ ] > 80% code coverage
- [ ] CI/CD pipeline
- [ ] Golden tests for regression detection
- [ ] Memory leak testing
- [ ] Concurrent access testing
