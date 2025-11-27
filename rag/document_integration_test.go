package rag

import (
	"testing"

	"github.com/tsawler/tabula/model"
)

func createTestModelDocument() *model.Document {
	doc := model.NewDocument()
	doc.Metadata.Title = "Test Document"

	// Page 1 with heading and paragraphs
	page1 := &model.Page{
		Number: 1,
		Width:  612,
		Height: 792,
		Elements: []model.Element{
			&model.Heading{
				Text:     "Introduction",
				Level:    1,
				FontSize: 24,
			},
			&model.Paragraph{
				Text:     "This is the first paragraph of the introduction section.",
				FontSize: 12,
			},
			&model.Paragraph{
				Text:     "This is the second paragraph with more content.",
				FontSize: 12,
			},
		},
		Layout: &model.PageLayout{
			Headings: []model.HeadingInfo{
				{Text: "Introduction", Level: 1, FontSize: 24},
			},
		},
	}
	doc.AddPage(page1)

	// Page 2 with list
	page2 := &model.Page{
		Number: 2,
		Width:  612,
		Height: 792,
		Elements: []model.Element{
			&model.Heading{
				Text:     "Features",
				Level:    2,
				FontSize: 18,
			},
			&model.List{
				Ordered: false,
				Items: []model.ListItem{
					{Text: "First feature"},
					{Text: "Second feature"},
					{Text: "Third feature"},
				},
			},
		},
		Layout: &model.PageLayout{
			Headings: []model.HeadingInfo{
				{Text: "Features", Level: 2, FontSize: 18},
			},
		},
	}
	doc.AddPage(page2)

	// Page 3 with table
	page3 := &model.Page{
		Number: 3,
		Width:  612,
		Height: 792,
		Elements: []model.Element{
			&model.Heading{
				Text:     "Data",
				Level:    2,
				FontSize: 18,
			},
			&model.Table{
				Rows: [][]model.Cell{
					{{Text: "Name"}, {Text: "Value"}},
					{{Text: "Item 1"}, {Text: "100"}},
					{{Text: "Item 2"}, {Text: "200"}},
				},
			},
		},
		Layout: &model.PageLayout{
			Headings: []model.HeadingInfo{
				{Text: "Data", Level: 2, FontSize: 18},
			},
		},
	}
	doc.AddPage(page3)

	return doc
}

func TestNewDocumentChunker(t *testing.T) {
	chunker := NewDocumentChunker()
	if chunker == nil {
		t.Error("NewDocumentChunker returned nil")
	}
}

func TestNewDocumentChunkerWithConfig(t *testing.T) {
	config := DefaultChunkerConfig()
	sizeConfig := DefaultSizeConfig()
	chunker := NewDocumentChunkerWithConfig(config, sizeConfig)

	if chunker == nil {
		t.Error("NewDocumentChunkerWithConfig returned nil")
	}
}

func TestDocumentChunker_ChunkDocument(t *testing.T) {
	doc := createTestModelDocument()
	chunker := NewDocumentChunker()

	collection := chunker.ChunkDocument(doc)

	if collection == nil {
		t.Fatal("ChunkDocument returned nil")
	}

	if len(collection.Chunks) == 0 {
		t.Error("Expected chunks, got none")
	}

	// Verify all chunks have document title
	for _, chunk := range collection.Chunks {
		if chunk.Metadata.DocumentTitle != "Test Document" {
			t.Errorf("Expected document title 'Test Document', got '%s'", chunk.Metadata.DocumentTitle)
		}
	}
}

func TestDocumentChunker_ChunkDocument_NilDoc(t *testing.T) {
	chunker := NewDocumentChunker()
	collection := chunker.ChunkDocument(nil)

	if collection == nil {
		t.Fatal("Expected empty collection, got nil")
	}

	if len(collection.Chunks) != 0 {
		t.Errorf("Expected 0 chunks for nil document, got %d", len(collection.Chunks))
	}
}

func TestDocumentChunker_ChunkDocument_EmptyDoc(t *testing.T) {
	doc := model.NewDocument()
	chunker := NewDocumentChunker()

	collection := chunker.ChunkDocument(doc)

	if collection == nil {
		t.Fatal("Expected collection, got nil")
	}
}

func TestDocumentChunker_Headings(t *testing.T) {
	doc := createTestModelDocument()
	chunker := NewDocumentChunker()

	collection := chunker.ChunkDocument(doc)

	// Find heading chunks
	var headingChunks []*Chunk
	for _, chunk := range collection.Chunks {
		for _, et := range chunk.Metadata.ElementTypes {
			if et == "heading" {
				headingChunks = append(headingChunks, chunk)
				break
			}
		}
	}

	if len(headingChunks) == 0 {
		t.Error("Expected heading chunks")
	}

	// Verify heading levels are set
	for _, chunk := range headingChunks {
		if chunk.Metadata.HeadingLevel == 0 {
			t.Errorf("Expected heading level to be set for chunk: %s", chunk.Text)
		}
	}
}

func TestDocumentChunker_Lists(t *testing.T) {
	doc := createTestModelDocument()
	chunker := NewDocumentChunker()

	collection := chunker.ChunkDocument(doc)

	// Find list chunks
	var listChunks []*Chunk
	for _, chunk := range collection.Chunks {
		if chunk.Metadata.HasList {
			listChunks = append(listChunks, chunk)
		}
	}

	if len(listChunks) == 0 {
		t.Error("Expected list chunks")
	}

	// Verify list content
	for _, chunk := range listChunks {
		if chunk.Text == "" {
			t.Error("Expected list text content")
		}
	}
}

func TestDocumentChunker_Tables(t *testing.T) {
	doc := createTestModelDocument()
	chunker := NewDocumentChunker()

	collection := chunker.ChunkDocument(doc)

	// Find table chunks
	var tableChunks []*Chunk
	for _, chunk := range collection.Chunks {
		if chunk.Metadata.HasTable {
			tableChunks = append(tableChunks, chunk)
		}
	}

	if len(tableChunks) == 0 {
		t.Error("Expected table chunks")
	}

	// Verify table content (should be markdown formatted)
	for _, chunk := range tableChunks {
		if chunk.Text == "" {
			t.Error("Expected table text content")
		}
	}
}

func TestDocumentChunker_SectionPath(t *testing.T) {
	doc := createTestModelDocument()
	chunker := NewDocumentChunker()

	collection := chunker.ChunkDocument(doc)

	// Check that section paths are being tracked
	foundSectionPath := false
	for _, chunk := range collection.Chunks {
		if len(chunk.Metadata.SectionPath) > 0 {
			foundSectionPath = true
			break
		}
	}

	if !foundSectionPath {
		t.Error("Expected section paths to be tracked")
	}
}

func TestDocumentChunker_PageNumbers(t *testing.T) {
	doc := createTestModelDocument()
	chunker := NewDocumentChunker()

	collection := chunker.ChunkDocument(doc)

	// Verify page numbers are set
	for _, chunk := range collection.Chunks {
		if chunk.Metadata.PageStart == 0 {
			t.Errorf("Expected PageStart to be set for chunk: %s", chunk.ID)
		}
	}
}

func TestDocumentChunker_TotalChunks(t *testing.T) {
	doc := createTestModelDocument()
	chunker := NewDocumentChunker()

	collection := chunker.ChunkDocument(doc)

	// Verify TotalChunks is set on all chunks
	totalChunks := len(collection.Chunks)
	for _, chunk := range collection.Chunks {
		if chunk.Metadata.TotalChunks != totalChunks {
			t.Errorf("Expected TotalChunks %d, got %d for chunk %s",
				totalChunks, chunk.Metadata.TotalChunks, chunk.ID)
		}
	}
}

func TestDocumentChunker_Images(t *testing.T) {
	doc := model.NewDocument()
	page := &model.Page{
		Number: 1,
		Elements: []model.Element{
			&model.Image{
				AltText: "A test image",
			},
		},
	}
	doc.AddPage(page)

	chunker := NewDocumentChunker()
	collection := chunker.ChunkDocument(doc)

	// Find image chunks
	var imageChunks []*Chunk
	for _, chunk := range collection.Chunks {
		if chunk.Metadata.HasImage {
			imageChunks = append(imageChunks, chunk)
		}
	}

	if len(imageChunks) == 0 {
		t.Error("Expected image chunk for image with alt text")
	}

	// Verify image content
	for _, chunk := range imageChunks {
		if chunk.Text == "" {
			t.Error("Expected image text content")
		}
	}
}

func TestDocumentChunker_ImagesWithoutAltText(t *testing.T) {
	doc := model.NewDocument()
	page := &model.Page{
		Number: 1,
		Elements: []model.Element{
			&model.Image{
				AltText: "", // No alt text
			},
		},
	}
	doc.AddPage(page)

	chunker := NewDocumentChunker()
	collection := chunker.ChunkDocument(doc)

	// Should not create chunk for image without alt text
	for _, chunk := range collection.Chunks {
		if chunk.Metadata.HasImage {
			t.Error("Did not expect chunk for image without alt text")
		}
	}
}

func TestChunkDocument_Convenience(t *testing.T) {
	doc := createTestModelDocument()
	collection := ChunkDocument(doc)

	if collection == nil {
		t.Fatal("ChunkDocument returned nil")
	}

	if len(collection.Chunks) == 0 {
		t.Error("Expected chunks")
	}
}

func TestChunkDocumentWithConfig(t *testing.T) {
	doc := createTestModelDocument()

	config := ChunkerConfig{
		MinChunkSize:    50,
		MaxChunkSize:    500,
		TargetChunkSize: 200,
	}
	sizeConfig := SmallChunkConfig()

	collection := ChunkDocumentWithConfig(doc, config, sizeConfig)

	if collection == nil {
		t.Fatal("ChunkDocumentWithConfig returned nil")
	}

	if len(collection.Chunks) == 0 {
		t.Error("Expected chunks")
	}
}

func TestDefaultDocumentChunkOptions(t *testing.T) {
	opts := DefaultDocumentChunkOptions()

	if opts.ChunkerConfig.TargetChunkSize == 0 {
		t.Error("Expected TargetChunkSize to be set")
	}
	if opts.SizeConfig.Target.Value == 0 {
		t.Error("Expected SizeConfig Target to be set")
	}
}

func TestRAGOptimizedOptions(t *testing.T) {
	opts := RAGOptimizedOptions()

	if opts.ChunkerConfig.TargetChunkSize != 500 {
		t.Errorf("Expected TargetChunkSize 500, got %d", opts.ChunkerConfig.TargetChunkSize)
	}
	if opts.SizeConfig.Target.Unit != SizeUnitTokens {
		t.Error("Expected token-based size config")
	}
}

func TestUpdateSectionPath(t *testing.T) {
	t.Run("add section", func(t *testing.T) {
		path := []string{}
		level := 0
		updateSectionPath(&path, &level, 1, "Chapter 1")

		if len(path) != 1 {
			t.Errorf("Expected 1 section, got %d", len(path))
		}
		if path[0] != "Chapter 1" {
			t.Errorf("Expected 'Chapter 1', got '%s'", path[0])
		}
	})

	t.Run("nested section", func(t *testing.T) {
		path := []string{"Chapter 1"}
		level := 1
		updateSectionPath(&path, &level, 2, "Section 1.1")

		if len(path) != 2 {
			t.Errorf("Expected 2 sections, got %d", len(path))
		}
		if path[1] != "Section 1.1" {
			t.Errorf("Expected 'Section 1.1', got '%s'", path[1])
		}
	})

	t.Run("pop section", func(t *testing.T) {
		path := []string{"Chapter 1", "Section 1.1", "Subsection"}
		level := 3
		updateSectionPath(&path, &level, 2, "Section 1.2")

		if len(path) != 2 {
			t.Errorf("Expected 2 sections after pop, got %d", len(path))
		}
		if path[1] != "Section 1.2" {
			t.Errorf("Expected 'Section 1.2', got '%s'", path[1])
		}
	})

	t.Run("same level replacement", func(t *testing.T) {
		path := []string{"Chapter 1"}
		level := 1
		updateSectionPath(&path, &level, 1, "Chapter 2")

		if len(path) != 1 {
			t.Errorf("Expected 1 section after replacement, got %d", len(path))
		}
		if path[0] != "Chapter 2" {
			t.Errorf("Expected 'Chapter 2', got '%s'", path[0])
		}
	})
}

func TestAppendUnique(t *testing.T) {
	t.Run("add new item", func(t *testing.T) {
		slice := []string{"a", "b"}
		result := appendUnique(slice, "c")

		if len(result) != 3 {
			t.Errorf("Expected 3 items, got %d", len(result))
		}
	})

	t.Run("duplicate item", func(t *testing.T) {
		slice := []string{"a", "b"}
		result := appendUnique(slice, "b")

		if len(result) != 2 {
			t.Errorf("Expected 2 items (no duplicate), got %d", len(result))
		}
	})
}

func TestIsHeadingElement(t *testing.T) {
	toc := []model.TOCEntry{
		{Page: 1, Text: "Introduction", Level: 1},
		{Page: 2, Text: "Features", Level: 2},
	}

	t.Run("matches TOC entry", func(t *testing.T) {
		if !isHeadingElement("Introduction", toc, 1) {
			t.Error("Expected to match TOC entry")
		}
	})

	t.Run("wrong page", func(t *testing.T) {
		if isHeadingElement("Introduction", toc, 2) {
			t.Error("Should not match on wrong page")
		}
	})

	t.Run("not in TOC", func(t *testing.T) {
		if isHeadingElement("Random Text", toc, 1) {
			t.Error("Should not match non-TOC text")
		}
	})
}

func TestGetHeadingLevel(t *testing.T) {
	toc := []model.TOCEntry{
		{Page: 1, Text: "Introduction", Level: 1},
		{Page: 2, Text: "Features", Level: 2},
	}

	t.Run("returns correct level", func(t *testing.T) {
		level := getHeadingLevel("Features", toc, 2)
		if level != 2 {
			t.Errorf("Expected level 2, got %d", level)
		}
	})

	t.Run("defaults to 1", func(t *testing.T) {
		level := getHeadingLevel("Unknown", toc, 1)
		if level != 1 {
			t.Errorf("Expected default level 1, got %d", level)
		}
	})
}

// Benchmark

func BenchmarkDocumentChunker(b *testing.B) {
	doc := createTestModelDocument()
	chunker := NewDocumentChunker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.ChunkDocument(doc)
	}
}

func BenchmarkChunkDocument(b *testing.B) {
	doc := createTestModelDocument()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ChunkDocument(doc)
	}
}
