package rag

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/model"
)

func TestOverlapStrategy_String(t *testing.T) {
	tests := []struct {
		strategy OverlapStrategy
		want     string
	}{
		{OverlapNone, "none"},
		{OverlapCharacter, "character"},
		{OverlapSentence, "sentence"},
		{OverlapParagraph, "paragraph"},
		{OverlapStrategy(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.want {
				t.Errorf("OverlapStrategy.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultOverlapConfig(t *testing.T) {
	config := DefaultOverlapConfig()

	if config.Strategy != OverlapSentence {
		t.Errorf("Expected Strategy OverlapSentence, got %v", config.Strategy)
	}

	if config.Size != 2 {
		t.Errorf("Expected Size 2, got %d", config.Size)
	}

	if !config.PreserveWords {
		t.Error("Expected PreserveWords to be true")
	}
}

func TestNewOverlapGenerator(t *testing.T) {
	generator := NewOverlapGenerator()
	if generator == nil {
		t.Error("NewOverlapGenerator returned nil")
	}
}

func TestOverlapGenerator_GenerateOverlap_None(t *testing.T) {
	config := OverlapConfig{
		Strategy: OverlapNone,
	}
	generator := NewOverlapGeneratorWithConfig(config)

	result := generator.GenerateOverlap("This is some text.")

	if result.Strategy != OverlapNone {
		t.Errorf("Expected strategy OverlapNone, got %v", result.Strategy)
	}

	if result.Text != "" {
		t.Errorf("Expected empty overlap, got %q", result.Text)
	}
}

func TestOverlapGenerator_GenerateOverlap_Character(t *testing.T) {
	config := OverlapConfig{
		Strategy:      OverlapCharacter,
		Size:          20,
		PreserveWords: true,
	}
	generator := NewOverlapGeneratorWithConfig(config)

	text := "This is the first sentence. This is the second sentence."
	result := generator.GenerateOverlap(text)

	if result.Strategy != OverlapCharacter {
		t.Errorf("Expected strategy OverlapCharacter, got %v", result.Strategy)
	}

	// With PreserveWords, should start at a word boundary
	if strings.HasPrefix(result.Text, " ") {
		t.Error("Overlap should not start with space")
	}

	t.Logf("Character overlap: %q", result.Text)
}

func TestOverlapGenerator_GenerateOverlap_Sentence(t *testing.T) {
	config := OverlapConfig{
		Strategy:   OverlapSentence,
		Size:       2, // Get last 2 sentences
		MaxOverlap: 500,
	}
	generator := NewOverlapGeneratorWithConfig(config)

	text := "First sentence. Second sentence. Third sentence. Fourth sentence."
	result := generator.GenerateOverlap(text)

	if result.Strategy != OverlapSentence {
		t.Errorf("Expected strategy OverlapSentence, got %v", result.Strategy)
	}

	if result.SentenceCount != 2 {
		t.Errorf("Expected 2 sentences, got %d", result.SentenceCount)
	}

	// Should contain third and fourth sentences
	if !strings.Contains(result.Text, "Third sentence") {
		t.Error("Expected overlap to contain 'Third sentence'")
	}
	if !strings.Contains(result.Text, "Fourth sentence") {
		t.Error("Expected overlap to contain 'Fourth sentence'")
	}

	t.Logf("Sentence overlap: %q", result.Text)
}

func TestOverlapGenerator_GenerateOverlap_Paragraph(t *testing.T) {
	config := OverlapConfig{
		Strategy:   OverlapParagraph,
		Size:       1, // Get last paragraph
		MaxOverlap: 500,
	}
	generator := NewOverlapGeneratorWithConfig(config)

	text := "First paragraph content here.\n\nSecond paragraph with more content.\n\nThird paragraph at the end."
	result := generator.GenerateOverlap(text)

	if result.Strategy != OverlapParagraph {
		t.Errorf("Expected strategy OverlapParagraph, got %v", result.Strategy)
	}

	// Should contain third paragraph
	if !strings.Contains(result.Text, "Third paragraph") {
		t.Errorf("Expected overlap to contain 'Third paragraph', got %q", result.Text)
	}

	t.Logf("Paragraph overlap: %q", result.Text)
}

func TestOverlapGenerator_GenerateOverlap_MaxOverlap(t *testing.T) {
	config := OverlapConfig{
		Strategy:   OverlapSentence,
		Size:       10, // Request many sentences
		MaxOverlap: 50, // But limit to 50 chars
	}
	generator := NewOverlapGeneratorWithConfig(config)

	text := "First sentence here. Second sentence here. Third sentence here. Fourth sentence here."
	result := generator.GenerateOverlap(text)

	if len(result.Text) > 50 {
		t.Errorf("Overlap exceeds MaxOverlap: %d > 50", len(result.Text))
	}

	t.Logf("Max-limited overlap (%d chars): %q", len(result.Text), result.Text)
}

func TestOverlapGenerator_GenerateOverlap_ShortText(t *testing.T) {
	config := OverlapConfig{
		Strategy:   OverlapSentence,
		Size:       5, // More sentences than exist
		MaxOverlap: 500,
	}
	generator := NewOverlapGeneratorWithConfig(config)

	text := "Only one sentence."
	result := generator.GenerateOverlap(text)

	// Should return all available content
	if result.Text != "Only one sentence." {
		t.Errorf("Expected full text for short input, got %q", result.Text)
	}
}

func TestSplitIntoSentencesWithPositions(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		count int
	}{
		{"empty", "", 0},
		{"single sentence", "Hello world.", 1},
		{"two sentences", "First one. Second one.", 2},
		{"three sentences", "One. Two. Three.", 3},
		{"with abbreviation", "Dr. Smith is here. Hello.", 2},
		{"with question", "What is this? I don't know.", 2},
		{"no ending punctuation", "Hello world", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := splitIntoSentencesWithPositions(tt.text)
			if len(sentences) != tt.count {
				t.Errorf("splitIntoSentencesWithPositions(%q) got %d sentences, want %d: %v",
					tt.text, len(sentences), tt.count, sentences)
			}
		})
	}
}

func TestSplitIntoParagraphs(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		count int
	}{
		{"empty", "", 0},
		{"single paragraph", "Hello world.", 1},
		{"two paragraphs", "First para.\n\nSecond para.", 2},
		{"three paragraphs", "One.\n\nTwo.\n\nThree.", 3},
		{"multiple newlines", "One.\n\n\n\nTwo.", 2},
		{"single newlines", "One.\nTwo.\nThree.", 1}, // Single newlines don't split
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paragraphs := splitIntoParagraphs(tt.text)
			if len(paragraphs) != tt.count {
				t.Errorf("splitIntoParagraphs(%q) got %d paragraphs, want %d: %v",
					tt.text, len(paragraphs), tt.count, paragraphs)
			}
		})
	}
}

func TestApplyOverlap(t *testing.T) {
	overlap := &OverlapResult{
		Text:      "Previous sentence.",
		CharCount: 18,
	}

	currentText := "Current chunk content."

	t.Run("without context", func(t *testing.T) {
		result := ApplyOverlap(currentText, overlap, "", false)
		if !strings.HasPrefix(result, "Previous sentence.") {
			t.Error("Result should start with overlap")
		}
		if !strings.Contains(result, "Current chunk content.") {
			t.Error("Result should contain current text")
		}
	})

	t.Run("with context", func(t *testing.T) {
		result := ApplyOverlap(currentText, overlap, "Section Title", true)
		if !strings.HasPrefix(result, "[Section Title]") {
			t.Error("Result should start with section context")
		}
	})

	t.Run("nil overlap", func(t *testing.T) {
		result := ApplyOverlap(currentText, nil, "", false)
		if result != currentText {
			t.Error("Nil overlap should return original text")
		}
	})

	t.Run("empty overlap", func(t *testing.T) {
		emptyOverlap := &OverlapResult{Text: ""}
		result := ApplyOverlap(currentText, emptyOverlap, "", false)
		if result != currentText {
			t.Error("Empty overlap should return original text")
		}
	})
}

func TestApplyOverlapToChunks(t *testing.T) {
	chunks := []*Chunk{
		{
			ID:   "chunk_0",
			Text: "First chunk content. It has multiple sentences. Here is another one.",
		},
		{
			ID:   "chunk_1",
			Text: "Second chunk content here.",
		},
		{
			ID:   "chunk_2",
			Text: "Third chunk with more text.",
		},
	}

	config := OverlapConfig{
		Strategy:   OverlapSentence,
		Size:       1, // 1 sentence overlap
		MaxOverlap: 500,
	}

	result := ApplyOverlapToChunks(chunks, config)

	if len(result) != 3 {
		t.Fatalf("Expected 3 chunks, got %d", len(result))
	}

	// First chunk should have no overlap prefix
	if result[0].HasOverlapPrefix {
		t.Error("First chunk should not have overlap prefix")
	}

	// Second chunk should have overlap from first
	if !result[1].HasOverlapPrefix {
		t.Error("Second chunk should have overlap prefix")
	}

	// Check that overlap content is from end of previous chunk
	if result[1].HasOverlapPrefix {
		t.Logf("Chunk 1 overlap: %q", result[1].OverlapPrefix)
	}
}

func TestChunkWithOverlap_GetOriginalText(t *testing.T) {
	chunk := &ChunkWithOverlap{
		Chunk: &Chunk{
			Text: "Overlap text.\n\nOriginal content here.",
		},
		OverlapPrefix:    "Overlap text.",
		HasOverlapPrefix: true,
	}

	original := chunk.GetOriginalText()
	if !strings.Contains(original, "Original content") {
		t.Errorf("GetOriginalText should return content without overlap, got %q", original)
	}
}

func TestChunker_ChunkWithOverlapEnabled(t *testing.T) {
	config := DefaultChunkerConfig()
	config.OverlapSize = 100
	config.OverlapSentences = true
	chunker := NewChunkerWithConfig(config)

	doc := createTestDocumentForOverlap()
	result, err := chunker.ChunkWithOverlapEnabled(doc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Chunks) == 0 {
		t.Error("Expected at least one chunk")
		return
	}

	t.Logf("Created %d chunks", len(result.Chunks))
	t.Logf("Overlap stats: %+v", result.OverlapStats)

	// Check that overlap strategy is correct
	if result.OverlapStats.OverlapStrategy != OverlapSentence {
		t.Errorf("Expected OverlapSentence strategy, got %v", result.OverlapStats.OverlapStrategy)
	}
}

func TestChunker_ChunkWithOverlapDisabled(t *testing.T) {
	config := DefaultChunkerConfig()
	config.OverlapSize = 0 // Disable overlap
	chunker := NewChunkerWithConfig(config)

	doc := createTestDocumentForOverlap()
	result, err := chunker.ChunkWithOverlapEnabled(doc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// No chunks should have overlap
	for i, chunk := range result.Chunks {
		if chunk.HasOverlapPrefix {
			t.Errorf("Chunk %d should not have overlap when disabled", i)
		}
	}

	if result.OverlapStats.ChunksWithOverlap != 0 {
		t.Errorf("Expected 0 chunks with overlap, got %d", result.OverlapStats.ChunksWithOverlap)
	}
}

func TestOverlapGenerator_PreserveWords(t *testing.T) {
	config := OverlapConfig{
		Strategy:      OverlapCharacter,
		Size:          15, // Would break in middle of "sentence"
		PreserveWords: true,
	}
	generator := NewOverlapGeneratorWithConfig(config)

	text := "This is a test sentence."
	result := generator.GenerateOverlap(text)

	// With PreserveWords, shouldn't break mid-word
	if strings.HasPrefix(result.Text, "entence") {
		t.Error("Should not break in middle of word 'sentence'")
	}

	t.Logf("Word-preserved overlap: %q", result.Text)
}

func TestIsSentenceEndRune(t *testing.T) {
	tests := []struct {
		text string
		pos  int
		want bool
	}{
		{"Hello.", 5, true},
		{"Hello. World", 5, true},
		{"Mr. Smith", 2, false},  // Abbreviation
		{"Dr. Jones", 2, false},  // Abbreviation
		{"3.14 is pi", 1, false}, // Decimal
		{"Hello", 4, false},      // No punctuation
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			runes := []rune(tt.text)
			if got := isSentenceEndRune(runes, tt.pos); got != tt.want {
				t.Errorf("isSentenceEndRune(%q, %d) = %v, want %v", tt.text, tt.pos, got, tt.want)
			}
		})
	}
}

// Benchmarks

func BenchmarkOverlapGenerator_Character(b *testing.B) {
	config := OverlapConfig{
		Strategy:      OverlapCharacter,
		Size:          100,
		PreserveWords: true,
	}
	generator := NewOverlapGeneratorWithConfig(config)
	text := strings.Repeat("This is a test sentence. ", 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.GenerateOverlap(text)
	}
}

func BenchmarkOverlapGenerator_Sentence(b *testing.B) {
	config := OverlapConfig{
		Strategy: OverlapSentence,
		Size:     2,
	}
	generator := NewOverlapGeneratorWithConfig(config)
	text := strings.Repeat("This is a test sentence. ", 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.GenerateOverlap(text)
	}
}

func BenchmarkApplyOverlapToChunks(b *testing.B) {
	chunks := make([]*Chunk, 50)
	for i := range chunks {
		chunks[i] = &Chunk{
			ID:   "chunk",
			Text: "This is chunk content. It has multiple sentences. Here is more content.",
		}
	}

	config := OverlapConfig{
		Strategy: OverlapSentence,
		Size:     1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ApplyOverlapToChunks(chunks, config)
	}
}

// Helper functions

func createTestDocumentForOverlap() *model.Document {
	doc := model.NewDocument()
	page := model.NewPage(612, 792)
	page.Layout = &model.PageLayout{
		Paragraphs: []model.ParagraphInfo{
			{Text: "First paragraph with content. It has multiple sentences. Here is another one.", BBox: model.BBox{X: 72, Y: 700, Width: 468, Height: 20}},
			{Text: "Second paragraph with different content. This also has sentences.", BBox: model.BBox{X: 72, Y: 650, Width: 468, Height: 20}},
			{Text: "Third paragraph continues the document. More content here.", BBox: model.BBox{X: 72, Y: 600, Width: 468, Height: 20}},
		},
	}
	doc.AddPage(page)
	return doc
}
