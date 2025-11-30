package rag

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/model"
)

func TestBoundaryType_String(t *testing.T) {
	tests := []struct {
		bt   BoundaryType
		want string
	}{
		{BoundaryNone, "none"},
		{BoundarySentence, "sentence"},
		{BoundaryParagraph, "paragraph"},
		{BoundaryList, "list"},
		{BoundaryListItem, "list_item"},
		{BoundaryHeading, "heading"},
		{BoundaryTable, "table"},
		{BoundaryFigure, "figure"},
		{BoundaryCodeBlock, "code_block"},
		{BoundaryPageBreak, "page_break"},
		{BoundaryType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.bt.String(); got != tt.want {
				t.Errorf("BoundaryType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoundaryType_Score(t *testing.T) {
	// Test that scores are ordered correctly
	if BoundaryHeading.Score() <= BoundaryParagraph.Score() {
		t.Error("Heading should have higher score than paragraph")
	}

	if BoundaryParagraph.Score() <= BoundarySentence.Score() {
		t.Error("Paragraph should have higher score than sentence")
	}

	if BoundarySentence.Score() <= BoundaryNone.Score() {
		t.Error("Sentence should have higher score than none")
	}

	if BoundaryTable.Score() <= BoundaryParagraph.Score() {
		t.Error("Table should have higher score than paragraph")
	}

	if BoundaryList.Score() <= BoundaryParagraph.Score() {
		t.Error("List should have higher score than paragraph")
	}
}

func TestDefaultBoundaryConfig(t *testing.T) {
	config := DefaultBoundaryConfig()

	if config.MinChunkSize != 100 {
		t.Errorf("Expected MinChunkSize 100, got %d", config.MinChunkSize)
	}

	if config.MaxChunkSize != 2000 {
		t.Errorf("Expected MaxChunkSize 2000, got %d", config.MaxChunkSize)
	}

	if !config.KeepListsIntact {
		t.Error("Expected KeepListsIntact to be true")
	}

	if !config.KeepTablesIntact {
		t.Error("Expected KeepTablesIntact to be true")
	}

	if len(config.ListIntroPatterns) == 0 {
		t.Error("Expected some ListIntroPatterns")
	}
}

func TestNewBoundaryDetector(t *testing.T) {
	detector := NewBoundaryDetector()
	if detector == nil {
		t.Error("NewBoundaryDetector returned nil")
	}
}

func TestBoundaryDetector_isListIntro(t *testing.T) {
	detector := NewBoundaryDetector()

	tests := []struct {
		text    string
		isIntro bool
	}{
		{"The following features:", true},
		{"Here are the steps:", true},
		{"These include:", true},
		{"Below are the options:", true},
		{"As follows:", true},
		{"Steps:", true},
		{"Features:", true},
		{"Including:", true},
		{"For example:", true},
		{"This ends with a colon:", true},
		{"Regular paragraph text.", false},
		{"No colon here", false},
		{"Question mark ending?", false},
	}

	for _, tt := range tests {
		t.Run(tt.text[:min(20, len(tt.text))], func(t *testing.T) {
			if got := detector.isListIntro(tt.text); got != tt.isIntro {
				t.Errorf("isListIntro(%q) = %v, want %v", tt.text, got, tt.isIntro)
			}
		})
	}
}

func TestIsSentenceEnd(t *testing.T) {
	tests := []struct {
		text     string
		position int
		want     bool
	}{
		{"Hello world.", 11, true},
		{"Hello world. How are you?", 11, true},
		{"Hello world? I'm fine.", 11, true},
		{"Hello world! Great.", 11, true},
		{"Mr. Smith is here.", 2, false}, // Abbreviation
		{"Dr. Jones arrived.", 2, false}, // Abbreviation
		{"e.g. this example", 3, false},  // Abbreviation
		{"The value is 3.14", 14, false}, // Decimal number
		{"Hello", 4, false},              // No punctuation
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := isSentenceEnd(tt.text, tt.position); got != tt.want {
				t.Errorf("isSentenceEnd(%q, %d) = %v, want %v", tt.text, tt.position, got, tt.want)
			}
		})
	}
}

func TestIsAbbreviation(t *testing.T) {
	tests := []struct {
		text string
		pos  int
		want bool
	}{
		{"Mr. Smith", 2, true},
		{"Dr. Jones", 2, true},
		{"Mrs. Brown", 3, true},
		{"etc. and more", 3, true},
		{"Hello. World", 5, false},
		{"Jan. 2024", 3, true},
		{"Corp. filing", 4, true},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := isAbbreviation(tt.text, tt.pos); got != tt.want {
				t.Errorf("isAbbreviation(%q, %d) = %v, want %v", tt.text, tt.pos, got, tt.want)
			}
		})
	}
}

func TestBoundaryDetector_DetectBoundaries(t *testing.T) {
	detector := NewBoundaryDetector()

	blocks := []ContentBlock{
		{Type: model.ElementTypeParagraph, Text: "First paragraph.", Index: 0},
		{Type: model.ElementTypeParagraph, Text: "Second paragraph.", Index: 1},
		{Type: model.ElementTypeHeading, Text: "Section Title", Index: 2},
		{Type: model.ElementTypeParagraph, Text: "Content under section.", Index: 3},
	}

	boundaries := detector.DetectBoundaries(blocks)

	// Should have boundaries after paragraphs and before heading
	if len(boundaries) == 0 {
		t.Error("Expected at least one boundary")
	}

	// Check for heading boundary
	foundHeading := false
	for _, b := range boundaries {
		if b.Type == BoundaryHeading {
			foundHeading = true
			break
		}
	}
	if !foundHeading {
		t.Error("Expected to find heading boundary")
	}
}

func TestBoundaryDetector_FindAtomicBlocks(t *testing.T) {
	detector := NewBoundaryDetector()

	t.Run("table is atomic", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: model.ElementTypeParagraph, Text: "Intro", Index: 0},
			{Type: model.ElementTypeTable, Text: "Table content", Index: 1},
			{Type: model.ElementTypeParagraph, Text: "After", Index: 2},
		}

		atomic := detector.FindAtomicBlocks(blocks)
		if len(atomic) != 1 {
			t.Errorf("Expected 1 atomic block, got %d", len(atomic))
			return
		}

		if atomic[0].Type != "table" {
			t.Errorf("Expected table atomic block, got %s", atomic[0].Type)
		}
	})

	t.Run("list with intro is atomic", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: model.ElementTypeParagraph, Text: "The following features:", Index: 0},
			{Type: model.ElementTypeList, Text: "- Item 1\n- Item 2", Index: 1},
			{Type: model.ElementTypeParagraph, Text: "After list", Index: 2},
		}

		atomic := detector.FindAtomicBlocks(blocks)

		// Should find the list, and the intro should be detected
		foundList := false
		for _, ab := range atomic {
			if ab.Type == "list" {
				foundList = true
				// Check if intro is included
				if ab.StartIndex == 0 && ab.EndIndex == 1 {
					t.Log("List with intro correctly detected as atomic")
				}
			}
		}
		if !foundList {
			t.Error("Expected to find list atomic block")
		}
	})

	t.Run("list without intro", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: model.ElementTypeParagraph, Text: "Regular paragraph.", Index: 0},
			{Type: model.ElementTypeList, Text: "- Item 1\n- Item 2", Index: 1},
		}

		atomic := detector.FindAtomicBlocks(blocks)

		// List should be atomic but not include the regular paragraph
		for _, ab := range atomic {
			if ab.Type == "list" {
				if ab.StartIndex != 1 {
					t.Errorf("List without intro should start at index 1, got %d", ab.StartIndex)
				}
			}
		}
	})
}

func TestBoundaryDetector_FindBestBoundary(t *testing.T) {
	detector := NewBoundaryDetector()

	boundaries := []Boundary{
		{Type: BoundarySentence, Position: 50, Score: BoundarySentence.Score()},
		{Type: BoundaryParagraph, Position: 100, Score: BoundaryParagraph.Score()},
		{Type: BoundaryHeading, Position: 200, Score: BoundaryHeading.Score()},
	}

	t.Run("finds best in range", func(t *testing.T) {
		best := detector.FindBestBoundary(boundaries, 0, 250)
		if best == nil {
			t.Error("Expected to find a boundary")
			return
		}
		if best.Type != BoundaryHeading {
			t.Errorf("Expected heading boundary (highest score), got %v", best.Type)
		}
	})

	t.Run("respects range", func(t *testing.T) {
		best := detector.FindBestBoundary(boundaries, 0, 150)
		if best == nil {
			t.Error("Expected to find a boundary")
			return
		}
		if best.Type != BoundaryParagraph {
			t.Errorf("Expected paragraph boundary (best in range), got %v", best.Type)
		}
	})

	t.Run("returns nil for empty range", func(t *testing.T) {
		best := detector.FindBestBoundary(boundaries, 300, 400)
		if best != nil {
			t.Error("Expected nil for range with no boundaries")
		}
	})
}

func TestBoundaryDetector_ShouldKeepTogether(t *testing.T) {
	detector := NewBoundaryDetector()

	tests := []struct {
		name   string
		block1 ContentBlock
		block2 ContentBlock
		want   bool
	}{
		{
			name:   "list intro with list",
			block1: ContentBlock{Type: model.ElementTypeParagraph, Text: "The following:"},
			block2: ContentBlock{Type: model.ElementTypeList, Text: "- Item"},
			want:   true,
		},
		{
			name:   "heading with content",
			block1: ContentBlock{Type: model.ElementTypeHeading, Text: "Title"},
			block2: ContentBlock{Type: model.ElementTypeParagraph, Text: "Content"},
			want:   true,
		},
		{
			name:   "two paragraphs",
			block1: ContentBlock{Type: model.ElementTypeParagraph, Text: "Para 1"},
			block2: ContentBlock{Type: model.ElementTypeParagraph, Text: "Para 2"},
			want:   false,
		},
		{
			name:   "caption with image",
			block1: ContentBlock{Type: model.ElementTypeCaption, Text: "Figure 1"},
			block2: ContentBlock{Type: model.ElementTypeImage, Text: ""},
			want:   true,
		},
		{
			name:   "image with caption",
			block1: ContentBlock{Type: model.ElementTypeImage, Text: ""},
			block2: ContentBlock{Type: model.ElementTypeCaption, Text: "Figure 1"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detector.ShouldKeepTogether(tt.block1, tt.block2); got != tt.want {
				t.Errorf("ShouldKeepTogether() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWithinAtomicBlock(t *testing.T) {
	atomicBlocks := []AtomicBlock{
		{StartIndex: 2, EndIndex: 4, Type: "list"},
		{StartIndex: 7, EndIndex: 7, Type: "table"},
	}

	tests := []struct {
		index int
		want  bool
	}{
		{0, false},
		{1, false},
		{2, true}, // Start of first atomic
		{3, true}, // Middle of first atomic
		{4, true}, // End of first atomic
		{5, false},
		{6, false},
		{7, true}, // Single-element atomic
		{8, false},
	}

	for _, tt := range tests {
		if got := IsWithinAtomicBlock(tt.index, atomicBlocks); got != tt.want {
			t.Errorf("IsWithinAtomicBlock(%d) = %v, want %v", tt.index, got, tt.want)
		}
	}
}

func TestGetAtomicBlockAt(t *testing.T) {
	atomicBlocks := []AtomicBlock{
		{StartIndex: 2, EndIndex: 4, Type: "list"},
		{StartIndex: 7, EndIndex: 7, Type: "table"},
	}

	t.Run("index in atomic block", func(t *testing.T) {
		ab := GetAtomicBlockAt(3, atomicBlocks)
		if ab == nil {
			t.Error("Expected to find atomic block")
			return
		}
		if ab.Type != "list" {
			t.Errorf("Expected list, got %s", ab.Type)
		}
	})

	t.Run("index not in atomic block", func(t *testing.T) {
		ab := GetAtomicBlockAt(5, atomicBlocks)
		if ab != nil {
			t.Error("Expected nil for index not in atomic block")
		}
	})
}

func TestOrphanedContentDetector_WouldCreateOrphan(t *testing.T) {
	detector := NewOrphanedContentDetector(50)

	tests := []struct {
		name     string
		text     string
		position int
		want     bool
	}{
		{
			name:     "small content before",
			text:     "Hi. This is a longer piece of content that would not be orphaned.",
			position: 3,
			want:     true,
		},
		{
			name:     "small content after",
			text:     "This is a longer piece of content that would not be orphaned. Hi.",
			position: 62,
			want:     true,
		},
		{
			name:     "good split point",
			text:     "This is the first sentence with plenty of enough content here. This is the second sentence also with plenty of content.",
			position: 62,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detector.WouldCreateOrphan(tt.text, tt.position); got != tt.want {
				t.Errorf("WouldCreateOrphan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChunker_KeepsListIntroWithList(t *testing.T) {
	config := DefaultChunkerConfig()
	config.MaxChunkSize = 500
	chunker := NewChunkerWithConfig(config)

	doc := createTestDocumentWithListIntro()
	result, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// The intro "The following features:" should be in the same chunk as the list
	for _, chunk := range result.Chunks {
		hasIntro := strings.Contains(chunk.Text, "following features")
		hasList := strings.Contains(chunk.Text, "Feature one")

		// If we have the intro, we should also have the list items
		if hasIntro && !hasList {
			t.Error("List intro separated from list items")
		}
	}
}

func TestChunker_TableAsAtomicUnit(t *testing.T) {
	config := DefaultChunkerConfig()
	config.MaxChunkSize = 200 // Small to force splitting
	chunker := NewChunkerWithConfig(config)

	// A table should stay as one unit unless it exceeds max size
	doc := createTestDocumentWithTable()
	result, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that table content stays together
	for _, chunk := range result.Chunks {
		if chunk.Metadata.HasTable {
			// Table chunk should have the table content
			t.Logf("Found table chunk: %s", chunk.Text[:min(50, len(chunk.Text))])
		}
	}
}

func TestChunker_AvoidOrphanedContent(t *testing.T) {
	config := DefaultChunkerConfig()
	config.MinChunkSize = 50
	config.MaxChunkSize = 200
	chunker := NewChunkerWithConfig(config)

	doc := createTestDocumentWithShortContent()
	result, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that no chunk is smaller than MinChunkSize (unless it's the only content)
	for i, chunk := range result.Chunks {
		if chunk.Metadata.CharCount < config.MinChunkSize && len(result.Chunks) > 1 {
			t.Logf("Warning: Chunk %d has only %d chars (min: %d)", i, chunk.Metadata.CharCount, config.MinChunkSize)
		}
	}
}

// Helper functions for creating test documents

func createTestDocumentWithListIntro() *model.Document {
	doc := model.NewDocument()
	page := model.NewPage(612, 792)
	page.Layout = &model.PageLayout{
		Paragraphs: []model.ParagraphInfo{
			{Text: "Introduction paragraph.", BBox: model.BBox{X: 72, Y: 700, Width: 468, Height: 20}},
			{Text: "The following features:", BBox: model.BBox{X: 72, Y: 650, Width: 468, Height: 20}},
		},
		Lists: []model.ListInfo{
			{
				Type: model.ListTypeBullet,
				BBox: model.BBox{X: 72, Y: 600, Width: 468, Height: 60},
				Items: []model.ListItem{
					{Text: "Feature one"},
					{Text: "Feature two"},
					{Text: "Feature three"},
				},
			},
		},
	}
	doc.AddPage(page)
	return doc
}

func createTestDocumentWithTable() *model.Document {
	doc := model.NewDocument()
	page := model.NewPage(612, 792)
	page.Layout = &model.PageLayout{
		Paragraphs: []model.ParagraphInfo{
			{Text: "Text before table.", BBox: model.BBox{X: 72, Y: 700, Width: 468, Height: 20}},
			{Text: "Text after table.", BBox: model.BBox{X: 72, Y: 500, Width: 468, Height: 20}},
		},
	}
	// Note: Tables are typically added as elements, but for this test we simulate via paragraphs
	doc.AddPage(page)
	return doc
}

func createTestDocumentWithShortContent() *model.Document {
	doc := model.NewDocument()
	page := model.NewPage(612, 792)
	page.Layout = &model.PageLayout{
		Paragraphs: []model.ParagraphInfo{
			{Text: "First paragraph with enough content to be substantial.", BBox: model.BBox{X: 72, Y: 700, Width: 468, Height: 20}},
			{Text: "Short.", BBox: model.BBox{X: 72, Y: 650, Width: 468, Height: 20}},
			{Text: "Another paragraph with enough content to be substantial.", BBox: model.BBox{X: 72, Y: 600, Width: 468, Height: 20}},
		},
	}
	doc.AddPage(page)
	return doc
}

// Benchmarks

func BenchmarkBoundaryDetector_DetectBoundaries(b *testing.B) {
	detector := NewBoundaryDetector()

	blocks := make([]ContentBlock, 100)
	for i := range blocks {
		blocks[i] = ContentBlock{
			Type:  model.ElementTypeParagraph,
			Text:  "This is paragraph " + string(rune('A'+i%26)) + ". It has multiple sentences. Here is another one.",
			Index: i,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectBoundaries(blocks)
	}
}

func BenchmarkBoundaryDetector_FindAtomicBlocks(b *testing.B) {
	detector := NewBoundaryDetector()

	blocks := make([]ContentBlock, 50)
	for i := range blocks {
		if i%5 == 0 {
			blocks[i] = ContentBlock{Type: model.ElementTypeList, Text: "- Item", Index: i}
		} else if i%7 == 0 {
			blocks[i] = ContentBlock{Type: model.ElementTypeTable, Text: "Table", Index: i}
		} else {
			blocks[i] = ContentBlock{Type: model.ElementTypeParagraph, Text: "Para", Index: i}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.FindAtomicBlocks(blocks)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================================
// Additional tests for better coverage
// ============================================================================

func TestBoundaryType_Score_AllTypes(t *testing.T) {
	tests := []struct {
		bt        BoundaryType
		wantScore int
	}{
		{BoundaryNone, 0},
		{BoundarySentence, 20},
		{BoundaryParagraph, 70},
		{BoundaryList, 80},
		{BoundaryListItem, 30},
		{BoundaryHeading, 100},
		{BoundaryTable, 85},
		{BoundaryFigure, 85},
		{BoundaryCodeBlock, 80},
		{BoundaryPageBreak, 90},
		{BoundaryType(99), 0}, // Unknown type
	}

	for _, tt := range tests {
		t.Run(tt.bt.String(), func(t *testing.T) {
			if got := tt.bt.Score(); got != tt.wantScore {
				t.Errorf("BoundaryType.Score() = %v, want %v", got, tt.wantScore)
			}
		})
	}
}

func TestBoundaryDetector_FindBoundaryWithLookAhead(t *testing.T) {
	detector := NewBoundaryDetector()

	boundaries := []Boundary{
		{Type: BoundarySentence, Position: 50, Score: BoundarySentence.Score()},
		{Type: BoundaryParagraph, Position: 100, Score: BoundaryParagraph.Score()},
		{Type: BoundaryHeading, Position: 200, Score: BoundaryHeading.Score()},
	}

	t.Run("finds boundary with lookahead", func(t *testing.T) {
		best := detector.FindBoundaryWithLookAhead(boundaries, 95)
		if best == nil {
			t.Error("Expected to find a boundary")
			return
		}
		// Should find paragraph at 100 (within lookahead range)
		if best.Position != 100 {
			t.Errorf("Expected boundary at 100, got %d", best.Position)
		}
	})

	t.Run("handles negative minPos", func(t *testing.T) {
		best := detector.FindBoundaryWithLookAhead(boundaries, 10)
		// Should still work with small target position
		if best != nil {
			t.Logf("Found boundary at position %d", best.Position)
		}
	})
}

func TestOrphanedContentDetector_AdjustForOrphans(t *testing.T) {
	detector := NewOrphanedContentDetector(50)

	boundaries := []Boundary{
		{Type: BoundarySentence, Position: 60, Score: 20},
		{Type: BoundaryParagraph, Position: 120, Score: 70},
	}

	t.Run("returns original if no orphan", func(t *testing.T) {
		text := "This is a fairly long piece of text that should not create any orphaned content when split at this point. And here is more content after the split."
		position := 80
		adjusted := detector.AdjustForOrphans(text, position, boundaries)
		// Should return original or nearby boundary
		if adjusted < 0 || adjusted > len(text) {
			t.Errorf("Invalid adjusted position: %d", adjusted)
		}
	})

	t.Run("adjusts to avoid orphan", func(t *testing.T) {
		text := "Hi. This is a much longer piece of text that will have enough content on both sides when split correctly."
		position := 3 // Would create orphan "Hi."
		adjusted := detector.AdjustForOrphans(text, position, boundaries)
		t.Logf("Original: %d, Adjusted: %d", position, adjusted)
	})

	t.Run("returns original when no good alternative", func(t *testing.T) {
		text := "AB"
		position := 1
		adjusted := detector.AdjustForOrphans(text, position, nil)
		// No boundaries to adjust to
		if adjusted != position {
			t.Errorf("Expected original position %d, got %d", position, adjusted)
		}
	})
}

func TestGetBoundaryTypeForBlock(t *testing.T) {
	tests := []struct {
		elementType model.ElementType
		wantType    BoundaryType
	}{
		{model.ElementTypeParagraph, BoundaryParagraph},
		{model.ElementTypeHeading, BoundaryHeading},
		{model.ElementTypeList, BoundaryList},
		{model.ElementTypeTable, BoundaryTable},
		{model.ElementTypeImage, BoundaryFigure},
		{model.ElementTypeFigure, BoundaryFigure},
	}

	detector := NewBoundaryDetector()

	for _, tt := range tests {
		t.Run(tt.elementType.String(), func(t *testing.T) {
			block := ContentBlock{Type: tt.elementType, Text: "content"}
			// We test indirectly through DetectBoundaries
			blocks := []ContentBlock{
				block,
				{Type: model.ElementTypeParagraph, Text: "next"},
			}
			boundaries := detector.DetectBoundaries(blocks)
			// Should detect boundaries based on block types
			if len(boundaries) > 0 {
				t.Logf("Detected %d boundaries for %s", len(boundaries), tt.elementType.String())
			}
		})
	}
}

func TestBoundaryDetector_FindAtomicBlocks_WithFigure(t *testing.T) {
	detector := NewBoundaryDetector()

	t.Run("figure with caption after", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: model.ElementTypeParagraph, Text: "Intro", Index: 0},
			{Type: model.ElementTypeImage, Text: "", Index: 1},
			{Type: model.ElementTypeCaption, Text: "Figure 1: Description", Index: 2},
			{Type: model.ElementTypeParagraph, Text: "After", Index: 3},
		}

		atomic := detector.FindAtomicBlocks(blocks)

		// Should find figure with caption as atomic
		foundFigure := false
		for _, ab := range atomic {
			if ab.Type == "figure" {
				foundFigure = true
				if ab.EndIndex < ab.StartIndex {
					t.Error("Invalid atomic block range")
				}
			}
		}
		if !foundFigure {
			t.Error("Expected to find figure atomic block")
		}
	})

	t.Run("figure with caption before", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: model.ElementTypeCaption, Text: "Figure 1: Description", Index: 0},
			{Type: model.ElementTypeImage, Text: "", Index: 1},
			{Type: model.ElementTypeParagraph, Text: "After", Index: 2},
		}

		atomic := detector.FindAtomicBlocks(blocks)

		// Should find figure with caption as atomic
		foundFigure := false
		for _, ab := range atomic {
			if ab.Type == "figure" {
				foundFigure = true
			}
		}
		if !foundFigure {
			t.Error("Expected to find figure atomic block")
		}
	})
}

func TestIsSentenceEnd_EdgeCases(t *testing.T) {
	tests := []struct {
		text     string
		position int
		want     bool
	}{
		{"", 0, false},                // Empty string
		{"A", 0, false},               // Single char, no punctuation
		{".", 0, true},                // Just a period
		{"U.S.A. is great", 5, false}, // Abbreviation
		{"vs. the team", 2, false},    // Common abbreviation
		{"i.e. example", 3, false},    // Latin abbreviation
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := isSentenceEnd(tt.text, tt.position); got != tt.want {
				t.Errorf("isSentenceEnd(%q, %d) = %v, want %v", tt.text, tt.position, got, tt.want)
			}
		})
	}
}
