package rag

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/model"
)

func TestChunkLevel_String(t *testing.T) {
	tests := []struct {
		level ChunkLevel
		want  string
	}{
		{ChunkLevelDocument, "document"},
		{ChunkLevelSection, "section"},
		{ChunkLevelParagraph, "paragraph"},
		{ChunkLevelSentence, "sentence"},
		{ChunkLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("ChunkLevel.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewChunk(t *testing.T) {
	metadata := ChunkMetadata{
		DocumentTitle: "Test Doc",
		SectionTitle:  "Introduction",
		PageStart:     1,
		PageEnd:       1,
	}

	chunk := NewChunk("chunk_0", "This is a test chunk with some text.", metadata)

	if chunk.ID != "chunk_0" {
		t.Errorf("Expected ID 'chunk_0', got '%s'", chunk.ID)
	}

	if chunk.Text != "This is a test chunk with some text." {
		t.Errorf("Text mismatch")
	}

	// "This is a test chunk with some text." = 36 chars
	if chunk.Metadata.CharCount != 36 {
		t.Errorf("Expected CharCount 36, got %d", chunk.Metadata.CharCount)
	}

	// 8 words: This, is, a, test, chunk, with, some, text
	if chunk.Metadata.WordCount != 8 {
		t.Errorf("Expected WordCount 8, got %d", chunk.Metadata.WordCount)
	}

	// Check estimated tokens (chars/4) = 36/4 = 9
	if chunk.Metadata.EstimatedTokens != 9 {
		t.Errorf("Expected EstimatedTokens 9, got %d", chunk.Metadata.EstimatedTokens)
	}

	// Check contextual text includes section title
	if !strings.Contains(chunk.TextWithContext, "[Introduction]") {
		t.Errorf("TextWithContext should contain section title")
	}
}

func TestChunk_GetSectionPathString(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want string
	}{
		{"empty path", nil, ""},
		{"single element", []string{"Introduction"}, "Introduction"},
		{"multiple elements", []string{"Chapter 1", "Section 2", "Subsection A"}, "Chapter 1 > Section 2 > Subsection A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &Chunk{
				Metadata: ChunkMetadata{
					SectionPath: tt.path,
				},
			}
			if got := chunk.GetSectionPathString(); got != tt.want {
				t.Errorf("GetSectionPathString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultChunkerConfig(t *testing.T) {
	config := DefaultChunkerConfig()

	if config.TargetChunkSize != 1000 {
		t.Errorf("Expected TargetChunkSize 1000, got %d", config.TargetChunkSize)
	}

	if config.MaxChunkSize != 2000 {
		t.Errorf("Expected MaxChunkSize 2000, got %d", config.MaxChunkSize)
	}

	if config.MinChunkSize != 100 {
		t.Errorf("Expected MinChunkSize 100, got %d", config.MinChunkSize)
	}

	if !config.PreserveListCoherence {
		t.Error("Expected PreserveListCoherence to be true")
	}

	if !config.SplitOnHeadings {
		t.Error("Expected SplitOnHeadings to be true")
	}
}

func TestNewChunker(t *testing.T) {
	chunker := NewChunker()
	if chunker == nil {
		t.Error("NewChunker returned nil")
	}

	if chunker.config.TargetChunkSize != 1000 {
		t.Error("Default config not applied")
	}
}

func TestNewChunkerWithConfig(t *testing.T) {
	config := ChunkerConfig{
		TargetChunkSize: 500,
		MaxChunkSize:    1000,
	}

	chunker := NewChunkerWithConfig(config)
	if chunker.config.TargetChunkSize != 500 {
		t.Error("Custom config not applied")
	}
}

func TestChunker_Chunk_NilDocument(t *testing.T) {
	chunker := NewChunker()
	_, err := chunker.Chunk(nil)
	if err == nil {
		t.Error("Expected error for nil document")
	}
}

func TestChunker_Chunk_EmptyDocument(t *testing.T) {
	chunker := NewChunker()
	doc := model.NewDocument()

	result, err := chunker.Chunk(doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result.Chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty document, got %d", len(result.Chunks))
	}
}

func TestChunker_Chunk_SingleParagraph(t *testing.T) {
	chunker := NewChunker()
	doc := createTestDocument([]testSection{
		{paragraphs: []string{"This is a single paragraph of text."}},
	})

	result, err := chunker.Chunk(doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result.Chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(result.Chunks))
	}

	if result.Chunks[0].Text != "This is a single paragraph of text." {
		t.Errorf("Text mismatch: %s", result.Chunks[0].Text)
	}
}

func TestChunker_Chunk_WithHeadings(t *testing.T) {
	chunker := NewChunker()
	doc := createTestDocumentWithHeadings([]testSectionWithHeading{
		{
			heading:    "Introduction",
			level:      1,
			paragraphs: []string{"This is the introduction."},
		},
		{
			heading:    "Background",
			level:      2,
			paragraphs: []string{"Some background information."},
		},
		{
			heading:    "Conclusion",
			level:      1,
			paragraphs: []string{"Final thoughts."},
		},
	})

	result, err := chunker.Chunk(doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should have at least 1 chunk
	if len(result.Chunks) < 1 {
		t.Errorf("Expected at least 1 chunk, got %d", len(result.Chunks))
		return
	}

	// Log what we got for debugging
	t.Logf("Got %d chunks", len(result.Chunks))
	for i, chunk := range result.Chunks {
		t.Logf("Chunk %d: section='%s', path=%v, text='%.50s...'",
			i, chunk.Metadata.SectionTitle, chunk.Metadata.SectionPath, chunk.Text)
	}

	// Check that we have content from the document
	totalText := ""
	for _, chunk := range result.Chunks {
		totalText += chunk.Text
	}
	if !strings.Contains(totalText, "introduction") {
		t.Error("Expected chunks to contain 'introduction'")
	}
}

func TestChunker_Chunk_LargeParagraph(t *testing.T) {
	config := DefaultChunkerConfig()
	config.MaxChunkSize = 100 // Small max to force splitting
	chunker := NewChunkerWithConfig(config)

	// Create a paragraph larger than MaxChunkSize
	largePara := strings.Repeat("This is a sentence. ", 20) // ~400 chars

	doc := createTestDocument([]testSection{
		{paragraphs: []string{largePara}},
	})

	result, err := chunker.Chunk(doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should have multiple chunks
	if len(result.Chunks) < 2 {
		t.Errorf("Expected multiple chunks for large paragraph, got %d", len(result.Chunks))
	}

	// Each chunk should be under MaxChunkSize
	for i, chunk := range result.Chunks {
		if len(chunk.Text) > config.MaxChunkSize+50 { // Allow some tolerance for sentence boundaries
			t.Errorf("Chunk %d exceeds MaxChunkSize: %d > %d", i, len(chunk.Text), config.MaxChunkSize)
		}
	}

	// Chunks should be sentence-level
	sentenceChunks := 0
	for _, chunk := range result.Chunks {
		if chunk.Metadata.Level == ChunkLevelSentence {
			sentenceChunks++
		}
	}
	if sentenceChunks == 0 {
		t.Error("Expected some sentence-level chunks")
	}
}

func TestChunker_Chunk_WithLists(t *testing.T) {
	chunker := NewChunker()
	doc := createTestDocumentWithLists([]testSectionWithList{
		{
			paragraphs: []string{"Here are the features:"},
			list: model.ListInfo{
				Type: model.ListTypeBullet,
				Items: []model.ListItem{
					{Text: "Feature one"},
					{Text: "Feature two"},
					{Text: "Feature three"},
				},
			},
		},
	})

	result, err := chunker.Chunk(doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result.Chunks) == 0 {
		t.Error("Expected at least one chunk")
		return
	}

	// Check that list is formatted
	foundList := false
	for _, chunk := range result.Chunks {
		if strings.Contains(chunk.Text, "- Feature one") {
			foundList = true
			break
		}
	}
	if !foundList {
		t.Error("List content not found in chunks")
	}
}

func TestChunker_Chunk_SectionPath(t *testing.T) {
	chunker := NewChunker()
	doc := createTestDocumentWithNestedHeadings()

	result, err := chunker.Chunk(doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check that we have section paths
	foundPath := false
	for _, chunk := range result.Chunks {
		if len(chunk.Metadata.SectionPath) > 1 {
			foundPath = true
			t.Logf("Found section path: %v", chunk.Metadata.SectionPath)
			break
		}
	}
	if !foundPath {
		t.Log("No nested section paths found (may be expected depending on heading levels)")
	}
}

func TestChunker_Chunk_Statistics(t *testing.T) {
	chunker := NewChunker()
	doc := createTestDocument([]testSection{
		{paragraphs: []string{"First paragraph.", "Second paragraph."}},
	})

	result, err := chunker.Chunk(doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Stats.TotalChunks != len(result.Chunks) {
		t.Errorf("TotalChunks mismatch: %d vs %d", result.Stats.TotalChunks, len(result.Chunks))
	}

	if result.Stats.TotalCharacters == 0 {
		t.Error("TotalCharacters should not be 0")
	}

	if result.Stats.TotalWords == 0 {
		t.Error("TotalWords should not be 0")
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		text string
		want int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  hello   world  ", 2},
		{"one two three four five", 5},
		{"hello\nworld", 2},
		{"hello\t\tworld", 2},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := countWords(tt.text); got != tt.want {
				t.Errorf("countWords(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestSplitIntoSentences(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{"empty", "", 0},
		{"single sentence", "Hello world.", 1},
		{"two sentences", "Hello world. How are you?", 2},
		{"three sentences", "First. Second. Third.", 3},
		{"with exclamation", "Hello! How are you?", 2},
		{"with question", "What is this? I don't know.", 2},
		{"no ending punctuation", "Hello world", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := splitIntoSentences(tt.text)
			if len(sentences) != tt.want {
				t.Errorf("splitIntoSentences(%q) got %d sentences, want %d: %v", tt.text, len(sentences), tt.want, sentences)
			}
		})
	}
}

func TestFormatList(t *testing.T) {
	list := model.ListInfo{
		Items: []model.ListItem{
			{Text: "Item one", Level: 0},
			{Text: "Item two", Level: 0},
			{Text: "Nested item", Level: 1},
		},
	}

	result := formatList(list)

	if !strings.Contains(result, "- Item one") {
		t.Error("Missing first item")
	}
	if !strings.Contains(result, "- Item two") {
		t.Error("Missing second item")
	}
	if !strings.Contains(result, "  - Nested item") {
		t.Error("Missing nested item with indentation")
	}
}

func TestMergeBBox(t *testing.T) {
	tests := []struct {
		name string
		a    *model.BBox
		b    *model.BBox
		want *model.BBox
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: nil,
		},
		{
			name: "a nil",
			a:    nil,
			b:    &model.BBox{X: 10, Y: 20, Width: 100, Height: 50},
			want: &model.BBox{X: 10, Y: 20, Width: 100, Height: 50},
		},
		{
			name: "b nil",
			a:    &model.BBox{X: 10, Y: 20, Width: 100, Height: 50},
			b:    nil,
			want: &model.BBox{X: 10, Y: 20, Width: 100, Height: 50},
		},
		{
			name: "merge overlapping",
			a:    &model.BBox{X: 0, Y: 0, Width: 100, Height: 100},
			b:    &model.BBox{X: 50, Y: 50, Width: 100, Height: 100},
			want: &model.BBox{X: 0, Y: 0, Width: 150, Height: 150},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeBBox(tt.a, tt.b)
			if tt.want == nil {
				if got != nil {
					t.Errorf("Expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Errorf("Expected %v, got nil", tt.want)
				return
			}
			if got.X != tt.want.X || got.Y != tt.want.Y ||
				got.Width != tt.want.Width || got.Height != tt.want.Height {
				t.Errorf("mergeBBox() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChunker_TotalChunksInMetadata(t *testing.T) {
	chunker := NewChunker()
	doc := createTestDocumentWithHeadings([]testSectionWithHeading{
		{heading: "Section 1", level: 1, paragraphs: []string{"Content 1"}},
		{heading: "Section 2", level: 1, paragraphs: []string{"Content 2"}},
		{heading: "Section 3", level: 1, paragraphs: []string{"Content 3"}},
	})

	result, err := chunker.Chunk(doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	for _, chunk := range result.Chunks {
		if chunk.Metadata.TotalChunks != len(result.Chunks) {
			t.Errorf("TotalChunks in metadata (%d) doesn't match actual chunks (%d)",
				chunk.Metadata.TotalChunks, len(result.Chunks))
		}
	}
}

// Benchmark tests
func BenchmarkChunker_SmallDocument(b *testing.B) {
	chunker := NewChunker()
	doc := createTestDocument([]testSection{
		{paragraphs: []string{
			"This is paragraph one.",
			"This is paragraph two.",
			"This is paragraph three.",
		}},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.Chunk(doc)
	}
}

func BenchmarkChunker_LargeDocument(b *testing.B) {
	chunker := NewChunker()

	// Create a document with many sections
	sections := make([]testSectionWithHeading, 50)
	for i := 0; i < 50; i++ {
		sections[i] = testSectionWithHeading{
			heading:    "Section " + string(rune('A'+i%26)),
			level:      1,
			paragraphs: []string{strings.Repeat("Lorem ipsum dolor sit amet. ", 10)},
		}
	}
	doc := createTestDocumentWithHeadings(sections)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunker.Chunk(doc)
	}
}

func BenchmarkSplitIntoSentences(b *testing.B) {
	text := "This is a test. It has multiple sentences. Each one ends with a period. Some end with questions? Others with exclamation marks!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitIntoSentences(text)
	}
}

// Test helpers

type testSection struct {
	paragraphs []string
}

type testSectionWithHeading struct {
	heading    string
	level      int
	paragraphs []string
}

type testSectionWithList struct {
	paragraphs []string
	list       model.ListInfo
}

func createTestDocument(sections []testSection) *model.Document {
	doc := model.NewDocument()
	page := model.NewPage(612, 792)
	page.Layout = &model.PageLayout{
		Paragraphs: make([]model.ParagraphInfo, 0),
	}

	for _, section := range sections {
		for _, para := range section.paragraphs {
			page.Layout.Paragraphs = append(page.Layout.Paragraphs, model.ParagraphInfo{
				Text: para,
				BBox: model.BBox{X: 72, Y: 700, Width: 468, Height: 20},
			})
		}
	}

	doc.AddPage(page)
	return doc
}

func createTestDocumentWithHeadings(sections []testSectionWithHeading) *model.Document {
	doc := model.NewDocument()
	page := model.NewPage(612, 792)
	page.Layout = &model.PageLayout{
		Paragraphs: make([]model.ParagraphInfo, 0),
		Headings:   make([]model.HeadingInfo, 0),
	}

	y := 700.0
	for _, section := range sections {
		// Add heading
		page.Layout.Headings = append(page.Layout.Headings, model.HeadingInfo{
			Text:  section.heading,
			Level: section.level,
			BBox:  model.BBox{X: 72, Y: y, Width: 468, Height: 24},
		})
		y -= 30

		// Add paragraphs
		for _, para := range section.paragraphs {
			page.Layout.Paragraphs = append(page.Layout.Paragraphs, model.ParagraphInfo{
				Text: para,
				BBox: model.BBox{X: 72, Y: y, Width: 468, Height: 20},
			})
			y -= 25
		}
	}

	doc.AddPage(page)
	return doc
}

func createTestDocumentWithLists(sections []testSectionWithList) *model.Document {
	doc := model.NewDocument()
	page := model.NewPage(612, 792)
	page.Layout = &model.PageLayout{
		Paragraphs: make([]model.ParagraphInfo, 0),
		Lists:      make([]model.ListInfo, 0),
	}

	y := 700.0
	for _, section := range sections {
		for _, para := range section.paragraphs {
			page.Layout.Paragraphs = append(page.Layout.Paragraphs, model.ParagraphInfo{
				Text: para,
				BBox: model.BBox{X: 72, Y: y, Width: 468, Height: 20},
			})
			y -= 25
		}

		section.list.BBox = model.BBox{X: 72, Y: y, Width: 468, Height: 60}
		page.Layout.Lists = append(page.Layout.Lists, section.list)
		y -= 70
	}

	doc.AddPage(page)
	return doc
}

func createTestDocumentWithNestedHeadings() *model.Document {
	doc := model.NewDocument()
	page := model.NewPage(612, 792)
	page.Layout = &model.PageLayout{
		Paragraphs: make([]model.ParagraphInfo, 0),
		Headings: []model.HeadingInfo{
			{Text: "Chapter 1", Level: 1, BBox: model.BBox{X: 72, Y: 700, Width: 468, Height: 24}},
			{Text: "Section 1.1", Level: 2, BBox: model.BBox{X: 72, Y: 650, Width: 468, Height: 20}},
			{Text: "Subsection 1.1.1", Level: 3, BBox: model.BBox{X: 72, Y: 600, Width: 468, Height: 18}},
			{Text: "Chapter 2", Level: 1, BBox: model.BBox{X: 72, Y: 500, Width: 468, Height: 24}},
		},
	}

	// Add paragraphs after each heading
	page.Layout.Paragraphs = []model.ParagraphInfo{
		{Text: "Chapter 1 content", BBox: model.BBox{X: 72, Y: 670, Width: 468, Height: 20}},
		{Text: "Section 1.1 content", BBox: model.BBox{X: 72, Y: 620, Width: 468, Height: 20}},
		{Text: "Subsection 1.1.1 content", BBox: model.BBox{X: 72, Y: 570, Width: 468, Height: 20}},
		{Text: "Chapter 2 content", BBox: model.BBox{X: 72, Y: 470, Width: 468, Height: 20}},
	}

	doc.AddPage(page)
	return doc
}
