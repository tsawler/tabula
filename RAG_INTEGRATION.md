# RAG Integration Guide

This guide explains how to integrate the PDF library into RAG (Retrieval-Augmented Generation) pipelines.

## Overview

RAG systems require:
1. **Document chunking** - Break documents into semantically meaningful pieces
2. **Metadata extraction** - Capture structure, hierarchy, and context
3. **Embeddings** - Convert chunks to vector representations
4. **Retrieval** - Find relevant chunks for queries

This library provides the **parsing and structuring** layer optimized for RAG.

## Key Features for RAG

### 1. Semantic Element Detection

Unlike raw text extraction, we provide typed elements:

```go
for _, page := range doc.Pages {
    for _, elem := range page.Elements {
        switch e := elem.(type) {
        case *model.Heading:
            // Headings provide document hierarchy
            chunk := RAGChunk{
                Type:     "heading",
                Content:  e.Text,
                Level:    e.Level,
                Metadata: map[string]interface{}{
                    "font_size": e.FontSize,
                    "page":      page.Number,
                },
            }

        case *model.Paragraph:
            // Paragraphs are natural text chunks
            chunk := RAGChunk{
                Type:    "paragraph",
                Content: e.Text,
                Metadata: map[string]interface{}{
                    "page": page.Number,
                    "bbox": e.BBox,
                },
            }

        case *model.Table:
            // Tables need special serialization
            chunk := RAGChunk{
                Type:    "table",
                Content: e.ToMarkdown(), // Or JSON
                Metadata: map[string]interface{}{
                    "rows":       e.RowCount(),
                    "cols":       e.ColCount(),
                    "confidence": e.Confidence,
                },
            }
        }
    }
}
```

### 2. Hierarchical Structure

Build document hierarchy for context:

```go
type DocumentNode struct {
    Element  model.Element
    Children []*DocumentNode
    Parent   *DocumentNode
    Level    int
}

func buildHierarchy(page *model.Page) *DocumentNode {
    root := &DocumentNode{Level: 0}
    current := root

    for _, elem := range page.Elements {
        node := &DocumentNode{Element: elem}

        if heading, ok := elem.(*model.Heading); ok {
            // Navigate up to appropriate parent level
            for current.Level >= heading.Level {
                current = current.Parent
            }
            node.Parent = current
            node.Level = heading.Level
            current.Children = append(current.Children, node)
            current = node
        } else {
            // Add as child of current heading
            node.Parent = current
            node.Level = current.Level + 1
            current.Children = append(current.Children, node)
        }
    }

    return root
}
```

### 3. Table Serialization

Tables require special handling for LLMs:

#### Option 1: Markdown (Recommended)

```go
func serializeTableMarkdown(table *model.Table) string {
    return table.ToMarkdown()
}

// Output:
// | Name | Age | City |
// |------|-----|------|
// | Alice | 30 | NYC |
// | Bob | 25 | SF |
```

#### Option 2: Linearized Text

```go
func serializeTableLinear(table *model.Table) string {
    var lines []string

    // Add header
    if len(table.Rows) > 0 {
        var headers []string
        for _, cell := range table.Rows[0] {
            headers = append(headers, cell.Text)
        }
        lines = append(lines, "Table columns: "+strings.Join(headers, ", "))
    }

    // Add rows
    for i := 1; i < len(table.Rows); i++ {
        var values []string
        for j, cell := range table.Rows[i] {
            if j < len(table.Rows[0]) {
                headerText := table.Rows[0][j].Text
                values = append(values, fmt.Sprintf("%s: %s", headerText, cell.Text))
            }
        }
        lines = append(lines, strings.Join(values, ", "))
    }

    return strings.Join(lines, "\n")
}

// Output:
// Table columns: Name, Age, City
// Name: Alice, Age: 30, City: NYC
// Name: Bob, Age: 25, City: SF
```

#### Option 3: Structured JSON

```go
func serializeTableJSON(table *model.Table) ([]byte, error) {
    var data []map[string]string

    if len(table.Rows) < 2 {
        return nil, errors.New("no data rows")
    }

    // Extract headers
    var headers []string
    for _, cell := range table.Rows[0] {
        headers = append(headers, cell.Text)
    }

    // Build row objects
    for i := 1; i < len(table.Rows); i++ {
        row := make(map[string]string)
        for j, cell := range table.Rows[i] {
            if j < len(headers) {
                row[headers[j]] = cell.Text
            }
        }
        data = append(data, row)
    }

    return json.Marshal(data)
}

// Output:
// [
//   {"Name": "Alice", "Age": "30", "City": "NYC"},
//   {"Name": "Bob", "Age": "25", "City": "SF"}
// ]
```

## Complete RAG Pipeline Example

```go
package main

import (
    "context"
    "fmt"

    "github.com/tsawler/tabula/reader"
    "github.com/tsawler/tabula/model"
)

// RAGChunk represents a chunk for embedding
type RAGChunk struct {
    ID          string
    Type        string
    Content     string
    Metadata    map[string]interface{}
    Embedding   []float32
}

type RAGPipeline struct {
    embedder    Embedder
    vectorStore VectorStore
}

func (p *RAGPipeline) IngestPDF(filename string) error {
    // 1. Parse PDF
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    pdfReader, err := reader.New(file)
    if err != nil {
        return err
    }

    doc, err := pdfReader.Parse()
    if err != nil {
        return err
    }

    // 2. Extract chunks
    chunks := p.extractChunks(doc)

    // 3. Generate embeddings
    for i := range chunks {
        embedding, err := p.embedder.Embed(chunks[i].Content)
        if err != nil {
            return err
        }
        chunks[i].Embedding = embedding
    }

    // 4. Store in vector database
    return p.vectorStore.Insert(chunks)
}

func (p *RAGPipeline) extractChunks(doc *model.Document) []RAGChunk {
    var chunks []RAGChunk

    currentSection := ""
    currentSubsection := ""

    for _, page := range doc.Pages {
        for _, elem := range page.Elements {
            chunk := RAGChunk{
                ID:       fmt.Sprintf("page%d_%d", page.Number, len(chunks)),
                Metadata: make(map[string]interface{}),
            }

            // Add common metadata
            chunk.Metadata["page"] = page.Number
            chunk.Metadata["bbox"] = elem.BoundingBox()

            // Add hierarchical context
            if currentSection != "" {
                chunk.Metadata["section"] = currentSection
            }
            if currentSubsection != "" {
                chunk.Metadata["subsection"] = currentSubsection
            }

            switch e := elem.(type) {
            case *model.Heading:
                chunk.Type = "heading"
                chunk.Content = e.Text
                chunk.Metadata["level"] = e.Level
                chunk.Metadata["font_size"] = e.FontSize

                // Update section tracking
                if e.Level == 1 {
                    currentSection = e.Text
                    currentSubsection = ""
                } else if e.Level == 2 {
                    currentSubsection = e.Text
                }

            case *model.Paragraph:
                chunk.Type = "paragraph"
                chunk.Content = e.Text
                chunk.Metadata["font_size"] = e.FontSize

            case *model.Table:
                chunk.Type = "table"
                // Use markdown for LLM compatibility
                chunk.Content = e.ToMarkdown()
                chunk.Metadata["rows"] = e.RowCount()
                chunk.Metadata["cols"] = e.ColCount()
                chunk.Metadata["confidence"] = e.Confidence

                // Also store structured version
                chunk.Metadata["table_json"], _ = serializeTableJSON(e)

            case *model.List:
                chunk.Type = "list"
                chunk.Content = e.GetText()
                chunk.Metadata["item_count"] = len(e.Items)
                chunk.Metadata["ordered"] = e.Ordered

            case *model.Image:
                chunk.Type = "image"
                chunk.Content = e.AltText
                chunk.Metadata["format"] = e.Format
                chunk.Metadata["dpi"] = e.DPI
                // Optionally run image through OCR or vision model
            }

            chunks = append(chunks, chunk)
        }
    }

    return chunks
}

func (p *RAGPipeline) Query(ctx context.Context, query string, topK int) ([]RAGChunk, error) {
    // 1. Embed query
    queryEmbedding, err := p.embedder.Embed(query)
    if err != nil {
        return nil, err
    }

    // 2. Search vector store
    results, err := p.vectorStore.Search(queryEmbedding, topK)
    if err != nil {
        return nil, err
    }

    return results, nil
}
```

## Advanced Chunking Strategies

### 1. Semantic Chunking

Combine related elements into larger chunks:

```go
func semanticChunking(elements []model.Element, maxTokens int) []RAGChunk {
    var chunks []RAGChunk
    var currentChunk strings.Builder
    var currentElements []model.Element

    for _, elem := range elements {
        text := ""
        if te, ok := elem.(model.TextElement); ok {
            text = te.GetText()
        }

        // Estimate tokens (rough: 1 token ≈ 4 chars)
        tokens := (currentChunk.Len() + len(text)) / 4

        // Start new chunk if:
        // 1. Would exceed max tokens
        // 2. Hit a major heading
        // 3. Hit a table (tables standalone)
        shouldSplit := tokens > maxTokens

        if heading, ok := elem.(*model.Heading); ok && heading.Level == 1 {
            shouldSplit = true
        }

        if _, ok := elem.(*model.Table); ok {
            shouldSplit = true
        }

        if shouldSplit && currentChunk.Len() > 0 {
            // Finalize current chunk
            chunks = append(chunks, RAGChunk{
                Content:  currentChunk.String(),
                Metadata: combineMetadata(currentElements),
            })

            currentChunk.Reset()
            currentElements = nil
        }

        currentChunk.WriteString(text)
        currentChunk.WriteString("\n")
        currentElements = append(currentElements, elem)
    }

    if currentChunk.Len() > 0 {
        chunks = append(chunks, RAGChunk{
            Content:  currentChunk.String(),
            Metadata: combineMetadata(currentElements),
        })
    }

    return chunks
}
```

### 2. Sliding Window Chunking

Add context overlap between chunks:

```go
func slidingWindowChunking(elements []model.Element, windowSize, overlap int) []RAGChunk {
    var chunks []RAGChunk

    for i := 0; i < len(elements); i += (windowSize - overlap) {
        end := i + windowSize
        if end > len(elements) {
            end = len(elements)
        }

        window := elements[i:end]
        chunk := combineElements(window)
        chunks = append(chunks, chunk)

        if end >= len(elements) {
            break
        }
    }

    return chunks
}
```

### 3. Hierarchical Chunking

Maintain parent-child relationships:

```go
type HierarchicalChunk struct {
    RAGChunk
    ParentID  string
    ChildIDs  []string
}

func hierarchicalChunking(root *DocumentNode) []HierarchicalChunk {
    var chunks []HierarchicalChunk

    var traverse func(node *DocumentNode, parentID string) string
    traverse = func(node *DocumentNode, parentID string) string {
        if node.Element == nil {
            return ""
        }

        chunkID := generateID()
        chunk := HierarchicalChunk{
            RAGChunk: RAGChunk{
                ID:      chunkID,
                Content: extractContent(node.Element),
            },
            ParentID: parentID,
        }

        // Process children
        for _, child := range node.Children {
            childID := traverse(child, chunkID)
            chunk.ChildIDs = append(chunk.ChildIDs, childID)
        }

        chunks = append(chunks, chunk)
        return chunkID
    }

    traverse(root, "")
    return chunks
}
```

## Metadata Enhancement

Add rich metadata for better retrieval:

```go
func enrichMetadata(chunk *RAGChunk, elem model.Element, page *model.Page, doc *model.Document) {
    // Document-level
    chunk.Metadata["doc_title"] = doc.Metadata.Title
    chunk.Metadata["doc_author"] = doc.Metadata.Author

    // Page-level
    chunk.Metadata["page_number"] = page.Number
    chunk.Metadata["total_pages"] = len(doc.Pages)

    // Element-level
    chunk.Metadata["element_type"] = elem.Type().String()
    bbox := elem.BoundingBox()
    chunk.Metadata["position"] = map[string]float64{
        "x": bbox.X,
        "y": bbox.Y,
        "width": bbox.Width,
        "height": bbox.Height,
    }

    // Derived features
    chunk.Metadata["char_count"] = len(chunk.Content)
    chunk.Metadata["word_count"] = len(strings.Fields(chunk.Content))

    // Semantic features
    if te, ok := elem.(model.TextElement); ok {
        text := te.GetText()
        chunk.Metadata["has_numbers"] = containsNumbers(text)
        chunk.Metadata["has_dates"] = containsDates(text)
        chunk.Metadata["language"] = detectLanguage(text)
    }

    // Table-specific
    if table, ok := elem.(*model.Table); ok {
        chunk.Metadata["is_financial"] = detectFinancialTable(table)
        chunk.Metadata["column_names"] = extractColumnNames(table)
    }
}
```

## Best Practices

1. **Preserve Structure** - Keep headings, tables, and lists as distinct chunks
2. **Add Context** - Include section/subsection in metadata
3. **Optimize Chunk Size** - Target 256-512 tokens per chunk
4. **Serialize Tables** - Use markdown for best LLM compatibility
5. **Enrich Metadata** - More metadata = better retrieval
6. **Handle Images** - Use OCR or vision models for image content
7. **Test Retrieval** - Validate that relevant chunks are retrieved

## Integration with Vector Stores

### Pinecone

```go
func uploadToPinecone(chunks []RAGChunk, apiKey, index string) error {
    client := pinecone.NewClient(apiKey)

    vectors := make([]pinecone.Vector, len(chunks))
    for i, chunk := range chunks {
        vectors[i] = pinecone.Vector{
            ID:       chunk.ID,
            Values:   chunk.Embedding,
            Metadata: chunk.Metadata,
        }
    }

    return client.Upsert(index, vectors)
}
```

### Weaviate

```go
func uploadToWeaviate(chunks []RAGChunk, url string) error {
    client := weaviate.New(weaviate.Config{Host: url})

    for _, chunk := range chunks {
        obj := &models.Object{
            Class: "PDFChunk",
            Properties: map[string]interface{}{
                "content":  chunk.Content,
                "type":     chunk.Type,
                "metadata": chunk.Metadata,
            },
            Vector: chunk.Embedding,
        }

        _, err := client.Data().Creator().
            WithClassName("PDFChunk").
            WithObject(obj).
            Do(context.Background())

        if err != nil {
            return err
        }
    }

    return nil
}
```

## Conclusion

This library provides a solid foundation for PDF ingestion in RAG pipelines by:
- Extracting semantic structure (not just raw text)
- Preserving document hierarchy
- Detecting and serializing tables
- Providing rich metadata

The key is treating PDFs as structured documents rather than flat text.
