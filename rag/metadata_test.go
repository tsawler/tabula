package rag

import (
	"encoding/json"
	"testing"
)

func TestContextFormat_String(t *testing.T) {
	tests := []struct {
		format ContextFormat
		want   string
	}{
		{ContextFormatNone, "none"},
		{ContextFormatBracket, "bracket"},
		{ContextFormatMarkdown, "markdown"},
		{ContextFormatBreadcrumb, "breadcrumb"},
		{ContextFormatXML, "xml"},
		{ContextFormat(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("ContextFormat.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultMetadataConfig(t *testing.T) {
	config := DefaultMetadataConfig()

	if config.ContextFormat != ContextFormatBracket {
		t.Errorf("Expected ContextFormat ContextFormatBracket, got %v", config.ContextFormat)
	}

	if config.IncludeDocumentTitle {
		t.Error("Expected IncludeDocumentTitle to be false")
	}

	if config.IncludePageNumbers {
		t.Error("Expected IncludePageNumbers to be false")
	}

	if config.IncludeSectionPath {
		t.Error("Expected IncludeSectionPath to be false")
	}

	if config.WordsPerMinute != 200 {
		t.Errorf("Expected WordsPerMinute 200, got %d", config.WordsPerMinute)
	}
}

func TestChunkMetadata_ToJSON(t *testing.T) {
	meta := &ChunkMetadata{
		DocumentTitle: "Test Document",
		SectionTitle:  "Introduction",
		PageStart:     1,
		PageEnd:       1,
		ChunkIndex:    0,
		Level:         ChunkLevelParagraph,
		CharCount:     100,
		WordCount:     20,
	}

	data, err := meta.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if result["document_title"] != "Test Document" {
		t.Error("Expected document_title in JSON output")
	}
}

func TestChunkMetadata_ToJSONIndent(t *testing.T) {
	meta := &ChunkMetadata{
		DocumentTitle: "Test Document",
		SectionTitle:  "Introduction",
		PageStart:     1,
		PageEnd:       2,
	}

	data, err := meta.ToJSONIndent()
	if err != nil {
		t.Fatalf("ToJSONIndent failed: %v", err)
	}

	// Check for indentation
	jsonStr := string(data)
	if jsonStr[0] != '{' {
		t.Error("JSON should start with {")
	}
	// Indented JSON should contain newlines
	if len(jsonStr) < 20 {
		t.Error("Indented JSON should be longer")
	}
}

func TestChunkMetadata_ToMap(t *testing.T) {
	meta := &ChunkMetadata{
		DocumentTitle:   "Test Doc",
		SectionPath:     []string{"Chapter 1", "Section A"},
		SectionTitle:    "Section A",
		HeadingLevel:    2,
		PageStart:       5,
		PageEnd:         6,
		ChunkIndex:      3,
		TotalChunks:     10,
		Level:           ChunkLevelSection,
		ParentID:        "parent_1",
		ChildIDs:        []string{"child_1", "child_2"},
		ElementTypes:    []string{"paragraph", "list"},
		HasTable:        false,
		HasList:         true,
		HasImage:        false,
		CharCount:       500,
		WordCount:       100,
		EstimatedTokens: 125,
	}

	m := meta.ToMap()

	if m["document_title"] != "Test Doc" {
		t.Errorf("Expected document_title 'Test Doc', got %v", m["document_title"])
	}

	if m["section_title"] != "Section A" {
		t.Errorf("Expected section_title 'Section A', got %v", m["section_title"])
	}

	if m["page_start"] != 5 {
		t.Errorf("Expected page_start 5, got %v", m["page_start"])
	}

	if m["has_list"] != true {
		t.Errorf("Expected has_list true, got %v", m["has_list"])
	}

	if m["level"] != "section" {
		t.Errorf("Expected level 'section', got %v", m["level"])
	}
}

func TestChunkMetadata_GetSectionPathString(t *testing.T) {
	tests := []struct {
		name      string
		path      []string
		separator string
		want      string
	}{
		{"empty", nil, "", ""},
		{"empty with separator", []string{}, " > ", ""},
		{"single", []string{"Chapter 1"}, "", "Chapter 1"},
		{"multiple default separator", []string{"Chapter 1", "Section A"}, "", "Chapter 1 > Section A"},
		{"multiple custom separator", []string{"A", "B", "C"}, " / ", "A / B / C"},
		{"three levels", []string{"Doc", "Part 1", "Chapter 1"}, " > ", "Doc > Part 1 > Chapter 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &ChunkMetadata{SectionPath: tt.path}
			got := meta.GetSectionPathString(tt.separator)
			if got != tt.want {
				t.Errorf("GetSectionPathString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChunkMetadata_GetPageRange(t *testing.T) {
	tests := []struct {
		name      string
		pageStart int
		pageEnd   int
		want      string
	}{
		{"single page", 1, 1, "p. 1"},
		{"multiple pages", 1, 5, "pp. 1-5"},
		{"two pages", 10, 11, "pp. 10-11"},
		{"page zero", 0, 0, "p. 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &ChunkMetadata{PageStart: tt.pageStart, PageEnd: tt.pageEnd}
			got := meta.GetPageRange()
			if got != tt.want {
				t.Errorf("GetPageRange() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChunkMetadata_GetReadingTimeMinutes(t *testing.T) {
	tests := []struct {
		name           string
		wordCount      int
		wordsPerMinute int
		wantMinutes    float64
	}{
		{"standard reading", 200, 200, 1.0},
		{"half minute", 100, 200, 0.5},
		{"default WPM", 400, 0, 2.0}, // 0 defaults to 200
		{"fast reader", 200, 400, 0.5},
		{"slow reader", 200, 100, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &ChunkMetadata{WordCount: tt.wordCount}
			got := meta.GetReadingTimeMinutes(tt.wordsPerMinute)
			if got != tt.wantMinutes {
				t.Errorf("GetReadingTimeMinutes() = %v, want %v", got, tt.wantMinutes)
			}
		})
	}
}

func TestChunkMetadata_GetReadingTimeString(t *testing.T) {
	tests := []struct {
		name      string
		wordCount int
		want      string
	}{
		{"less than minute", 50, "< 1 min read"},
		{"one minute", 200, "1 min read"},
		{"two minutes", 400, "2 min read"},
		{"five minutes", 1000, "5 min read"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &ChunkMetadata{WordCount: tt.wordCount}
			got := meta.GetReadingTimeString(200)
			if got != tt.want {
				t.Errorf("GetReadingTimeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChunkMetadata_IsInSection(t *testing.T) {
	meta := &ChunkMetadata{
		SectionTitle: "Current Section",
		SectionPath:  []string{"Chapter 1", "Part A", "Current Section"},
	}

	tests := []struct {
		name    string
		section string
		want    bool
	}{
		{"exact match", "Current Section", true},
		{"parent chapter", "Chapter 1", true},
		{"parent part", "Part A", true},
		{"not in path", "Chapter 2", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := meta.IsInSection(tt.section)
			if got != tt.want {
				t.Errorf("IsInSection(%q) = %v, want %v", tt.section, got, tt.want)
			}
		})
	}
}

func TestChunkMetadata_IsOnPage(t *testing.T) {
	meta := &ChunkMetadata{
		PageStart: 5,
		PageEnd:   10,
	}

	tests := []struct {
		page int
		want bool
	}{
		{4, false},
		{5, true},
		{7, true},
		{10, true},
		{11, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := meta.IsOnPage(tt.page)
			if got != tt.want {
				t.Errorf("IsOnPage(%d) = %v, want %v", tt.page, got, tt.want)
			}
		})
	}
}

func TestChunkMetadata_ContainsElementType(t *testing.T) {
	meta := &ChunkMetadata{
		ElementTypes: []string{"paragraph", "list", "heading"},
	}

	tests := []struct {
		elementType string
		want        bool
	}{
		{"paragraph", true},
		{"list", true},
		{"heading", true},
		{"PARAGRAPH", true}, // case insensitive
		{"Heading", true},
		{"table", false},
		{"image", false},
	}

	for _, tt := range tests {
		t.Run(tt.elementType, func(t *testing.T) {
			got := meta.ContainsElementType(tt.elementType)
			if got != tt.want {
				t.Errorf("ContainsElementType(%q) = %v, want %v", tt.elementType, got, tt.want)
			}
		})
	}
}

func TestChunk_GenerateContextText(t *testing.T) {
	chunk := &Chunk{
		Text: "This is the main content.",
		Metadata: ChunkMetadata{
			DocumentTitle: "My Document",
			SectionTitle:  "Introduction",
			SectionPath:   []string{"Chapter 1", "Introduction"},
			PageStart:     1,
			PageEnd:       1,
		},
	}

	tests := []struct {
		name     string
		config   MetadataConfig
		contains []string
	}{
		{
			"none format",
			MetadataConfig{ContextFormat: ContextFormatNone},
			[]string{"This is the main content."},
		},
		{
			"bracket with title",
			MetadataConfig{
				ContextFormat:        ContextFormatBracket,
				IncludeDocumentTitle: true,
			},
			[]string{"[My Document | Introduction]", "This is the main content."},
		},
		{
			"markdown format",
			MetadataConfig{
				ContextFormat:        ContextFormatMarkdown,
				IncludeDocumentTitle: true,
			},
			[]string{"# My Document | Introduction", "This is the main content."},
		},
		{
			"breadcrumb format",
			MetadataConfig{
				ContextFormat:        ContextFormatBreadcrumb,
				IncludeDocumentTitle: true,
			},
			[]string{"My Document | Introduction", "---", "This is the main content."},
		},
		{
			"XML format",
			MetadataConfig{
				ContextFormat:        ContextFormatXML,
				IncludeDocumentTitle: true,
			},
			[]string{"<context>My Document | Introduction</context>", "This is the main content."},
		},
		{
			"with section path",
			MetadataConfig{
				ContextFormat:      ContextFormatBracket,
				IncludeSectionPath: true,
			},
			[]string{"[Chapter 1 > Introduction]", "This is the main content."},
		},
		{
			"with page numbers",
			MetadataConfig{
				ContextFormat:      ContextFormatBracket,
				IncludePageNumbers: true,
			},
			[]string{"[Introduction | p. 1]", "This is the main content."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunk.GenerateContextText(tt.config)
			for _, expected := range tt.contains {
				if !containsString(result, expected) {
					t.Errorf("Expected result to contain %q, got %q", expected, result)
				}
			}
		})
	}
}

func TestChunk_ToEmbeddingFormat(t *testing.T) {
	chunk := &Chunk{
		Text: "Content for embedding.",
		Metadata: ChunkMetadata{
			SectionPath: []string{"Section A", "Subsection B"},
		},
	}

	result := chunk.ToEmbeddingFormat()

	// Should include section path context
	if !containsString(result, "Section A > Subsection B") {
		t.Error("Expected section path in embedding format")
	}
	if !containsString(result, "Content for embedding.") {
		t.Error("Expected content in embedding format")
	}
}

func TestChunk_ToSearchableText(t *testing.T) {
	chunk := &Chunk{
		Text: "Plain searchable content.",
		Metadata: ChunkMetadata{
			SectionTitle: "Should not appear",
		},
	}

	result := chunk.ToSearchableText()

	// Should return plain text without context
	if result != "Plain searchable content." {
		t.Errorf("Expected plain text, got %q", result)
	}
}

func TestChunk_Summary(t *testing.T) {
	chunk := &Chunk{
		Metadata: ChunkMetadata{
			SectionTitle: "Test Section",
			PageStart:    5,
			PageEnd:      5,
			WordCount:    100,
			HasTable:     true,
			HasList:      false,
			HasImage:     true,
		},
	}

	summary := chunk.Summary()

	if !containsString(summary, "Section: Test Section") {
		t.Error("Expected section title in summary")
	}
	if !containsString(summary, "p. 5") {
		t.Error("Expected page number in summary")
	}
	if !containsString(summary, "100 words") {
		t.Error("Expected word count in summary")
	}
	if !containsString(summary, "contains table") {
		t.Error("Expected table indicator in summary")
	}
	if !containsString(summary, "contains image") {
		t.Error("Expected image indicator in summary")
	}
	if containsString(summary, "contains list") {
		t.Error("Should not contain list indicator when HasList is false")
	}
}

func TestNewChunkCollection(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Text: "First"},
		{ID: "2", Text: "Second"},
	}

	cc := NewChunkCollection(chunks)

	if cc.Count() != 2 {
		t.Errorf("Expected 2 chunks, got %d", cc.Count())
	}
}

func TestChunkCollection_Filter(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Metadata: ChunkMetadata{WordCount: 50}},
		{ID: "2", Metadata: ChunkMetadata{WordCount: 100}},
		{ID: "3", Metadata: ChunkMetadata{WordCount: 150}},
	}

	cc := NewChunkCollection(chunks)

	filtered := cc.Filter(func(c *Chunk) bool {
		return c.Metadata.WordCount >= 100
	})

	if filtered.Count() != 2 {
		t.Errorf("Expected 2 filtered chunks, got %d", filtered.Count())
	}
}

func TestChunkCollection_FilterBySection(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Metadata: ChunkMetadata{SectionTitle: "Intro", SectionPath: []string{"Intro"}}},
		{ID: "2", Metadata: ChunkMetadata{SectionTitle: "Methods", SectionPath: []string{"Intro", "Methods"}}},
		{ID: "3", Metadata: ChunkMetadata{SectionTitle: "Results", SectionPath: []string{"Results"}}},
	}

	cc := NewChunkCollection(chunks)

	// Filter by Intro section - should get chunks 1 and 2 (2 is a child of Intro)
	filtered := cc.FilterBySection("Intro")

	if filtered.Count() != 2 {
		t.Errorf("Expected 2 chunks in Intro section, got %d", filtered.Count())
	}
}

func TestChunkCollection_FilterByPage(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Metadata: ChunkMetadata{PageStart: 1, PageEnd: 2}},
		{ID: "2", Metadata: ChunkMetadata{PageStart: 3, PageEnd: 3}},
		{ID: "3", Metadata: ChunkMetadata{PageStart: 4, PageEnd: 5}},
	}

	cc := NewChunkCollection(chunks)

	filtered := cc.FilterByPage(3)

	if filtered.Count() != 1 {
		t.Errorf("Expected 1 chunk on page 3, got %d", filtered.Count())
	}
	if filtered.First().ID != "2" {
		t.Errorf("Expected chunk 2, got %s", filtered.First().ID)
	}
}

func TestChunkCollection_FilterByPageRange(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Metadata: ChunkMetadata{PageStart: 1, PageEnd: 2}},
		{ID: "2", Metadata: ChunkMetadata{PageStart: 3, PageEnd: 4}},
		{ID: "3", Metadata: ChunkMetadata{PageStart: 5, PageEnd: 6}},
		{ID: "4", Metadata: ChunkMetadata{PageStart: 7, PageEnd: 8}},
	}

	cc := NewChunkCollection(chunks)

	// Filter pages 2-5
	filtered := cc.FilterByPageRange(2, 5)

	if filtered.Count() != 3 {
		t.Errorf("Expected 3 chunks in page range 2-5, got %d", filtered.Count())
	}
}

func TestChunkCollection_FilterByElementType(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Metadata: ChunkMetadata{ElementTypes: []string{"paragraph", "heading"}}},
		{ID: "2", Metadata: ChunkMetadata{ElementTypes: []string{"table"}}},
		{ID: "3", Metadata: ChunkMetadata{ElementTypes: []string{"paragraph", "list"}}},
	}

	cc := NewChunkCollection(chunks)

	filtered := cc.FilterByElementType("paragraph")

	if filtered.Count() != 2 {
		t.Errorf("Expected 2 chunks with paragraph, got %d", filtered.Count())
	}
}

func TestChunkCollection_FilterWithTables(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Metadata: ChunkMetadata{HasTable: true}},
		{ID: "2", Metadata: ChunkMetadata{HasTable: false}},
		{ID: "3", Metadata: ChunkMetadata{HasTable: true}},
	}

	cc := NewChunkCollection(chunks)

	filtered := cc.FilterWithTables()

	if filtered.Count() != 2 {
		t.Errorf("Expected 2 chunks with tables, got %d", filtered.Count())
	}
}

func TestChunkCollection_FilterWithLists(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Metadata: ChunkMetadata{HasList: true}},
		{ID: "2", Metadata: ChunkMetadata{HasList: false}},
	}

	cc := NewChunkCollection(chunks)

	filtered := cc.FilterWithLists()

	if filtered.Count() != 1 {
		t.Errorf("Expected 1 chunk with list, got %d", filtered.Count())
	}
}

func TestChunkCollection_FilterWithImages(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Metadata: ChunkMetadata{HasImage: true}},
		{ID: "2", Metadata: ChunkMetadata{HasImage: true}},
		{ID: "3", Metadata: ChunkMetadata{HasImage: false}},
	}

	cc := NewChunkCollection(chunks)

	filtered := cc.FilterWithImages()

	if filtered.Count() != 2 {
		t.Errorf("Expected 2 chunks with images, got %d", filtered.Count())
	}
}

func TestChunkCollection_FilterByTokens(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Metadata: ChunkMetadata{EstimatedTokens: 50}},
		{ID: "2", Metadata: ChunkMetadata{EstimatedTokens: 100}},
		{ID: "3", Metadata: ChunkMetadata{EstimatedTokens: 200}},
	}

	cc := NewChunkCollection(chunks)

	t.Run("min tokens", func(t *testing.T) {
		filtered := cc.FilterByMinTokens(100)
		if filtered.Count() != 2 {
			t.Errorf("Expected 2 chunks with >= 100 tokens, got %d", filtered.Count())
		}
	})

	t.Run("max tokens", func(t *testing.T) {
		filtered := cc.FilterByMaxTokens(100)
		if filtered.Count() != 2 {
			t.Errorf("Expected 2 chunks with <= 100 tokens, got %d", filtered.Count())
		}
	})
}

func TestChunkCollection_Search(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Text: "Introduction to machine learning algorithms"},
		{ID: "2", Text: "Deep learning neural networks"},
		{ID: "3", Text: "Algorithms for data processing"},
	}

	cc := NewChunkCollection(chunks)

	t.Run("case insensitive search", func(t *testing.T) {
		filtered := cc.Search("LEARNING")
		if filtered.Count() != 2 {
			t.Errorf("Expected 2 chunks matching 'learning', got %d", filtered.Count())
		}
	})

	t.Run("specific search", func(t *testing.T) {
		filtered := cc.Search("neural")
		if filtered.Count() != 1 {
			t.Errorf("Expected 1 chunk matching 'neural', got %d", filtered.Count())
		}
	})

	t.Run("no matches", func(t *testing.T) {
		filtered := cc.Search("quantum")
		if filtered.Count() != 0 {
			t.Errorf("Expected 0 chunks matching 'quantum', got %d", filtered.Count())
		}
	})
}

func TestChunkCollection_FirstLastGet(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Text: "First"},
		{ID: "2", Text: "Second"},
		{ID: "3", Text: "Third"},
	}

	cc := NewChunkCollection(chunks)

	t.Run("First", func(t *testing.T) {
		first := cc.First()
		if first == nil || first.ID != "1" {
			t.Error("Expected first chunk to be '1'")
		}
	})

	t.Run("Last", func(t *testing.T) {
		last := cc.Last()
		if last == nil || last.ID != "3" {
			t.Error("Expected last chunk to be '3'")
		}
	})

	t.Run("GetByIndex", func(t *testing.T) {
		chunk := cc.GetByIndex(1)
		if chunk == nil || chunk.ID != "2" {
			t.Error("Expected chunk at index 1 to be '2'")
		}
	})

	t.Run("GetByIndex out of bounds", func(t *testing.T) {
		chunk := cc.GetByIndex(10)
		if chunk != nil {
			t.Error("Expected nil for out of bounds index")
		}
		chunk = cc.GetByIndex(-1)
		if chunk != nil {
			t.Error("Expected nil for negative index")
		}
	})

	t.Run("GetByID", func(t *testing.T) {
		chunk := cc.GetByID("2")
		if chunk == nil || chunk.Text != "Second" {
			t.Error("Expected to find chunk with ID '2'")
		}
	})

	t.Run("GetByID not found", func(t *testing.T) {
		chunk := cc.GetByID("99")
		if chunk != nil {
			t.Error("Expected nil for non-existent ID")
		}
	})

	t.Run("empty collection", func(t *testing.T) {
		empty := NewChunkCollection(nil)
		if empty.First() != nil {
			t.Error("First() on empty should return nil")
		}
		if empty.Last() != nil {
			t.Error("Last() on empty should return nil")
		}
	})
}

func TestChunkCollection_ToSlice(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1"},
		{ID: "2"},
	}

	cc := NewChunkCollection(chunks)
	slice := cc.ToSlice()

	if len(slice) != 2 {
		t.Errorf("Expected slice of 2, got %d", len(slice))
	}
}

func TestChunkCollection_GetAllSections(t *testing.T) {
	chunks := []*Chunk{
		{Metadata: ChunkMetadata{SectionTitle: "Intro"}},
		{Metadata: ChunkMetadata{SectionTitle: "Methods"}},
		{Metadata: ChunkMetadata{SectionTitle: "Intro"}}, // duplicate
		{Metadata: ChunkMetadata{SectionTitle: "Results"}},
		{Metadata: ChunkMetadata{SectionTitle: ""}}, // empty
	}

	cc := NewChunkCollection(chunks)
	sections := cc.GetAllSections()

	if len(sections) != 3 {
		t.Errorf("Expected 3 unique sections, got %d: %v", len(sections), sections)
	}
}

func TestChunkCollection_GetPageRange(t *testing.T) {
	t.Run("normal range", func(t *testing.T) {
		chunks := []*Chunk{
			{Metadata: ChunkMetadata{PageStart: 5, PageEnd: 7}},
			{Metadata: ChunkMetadata{PageStart: 1, PageEnd: 3}},
			{Metadata: ChunkMetadata{PageStart: 10, PageEnd: 15}},
		}

		cc := NewChunkCollection(chunks)
		min, max := cc.GetPageRange()

		if min != 1 {
			t.Errorf("Expected min page 1, got %d", min)
		}
		if max != 15 {
			t.Errorf("Expected max page 15, got %d", max)
		}
	})

	t.Run("empty collection", func(t *testing.T) {
		cc := NewChunkCollection(nil)
		min, max := cc.GetPageRange()

		if min != 0 || max != 0 {
			t.Errorf("Expected 0,0 for empty collection, got %d,%d", min, max)
		}
	})
}

func TestChunkCollection_GetTotalTokens(t *testing.T) {
	chunks := []*Chunk{
		{Metadata: ChunkMetadata{EstimatedTokens: 100}},
		{Metadata: ChunkMetadata{EstimatedTokens: 200}},
		{Metadata: ChunkMetadata{EstimatedTokens: 150}},
	}

	cc := NewChunkCollection(chunks)
	total := cc.GetTotalTokens()

	if total != 450 {
		t.Errorf("Expected 450 total tokens, got %d", total)
	}
}

func TestChunkCollection_GetTotalWords(t *testing.T) {
	chunks := []*Chunk{
		{Metadata: ChunkMetadata{WordCount: 50}},
		{Metadata: ChunkMetadata{WordCount: 100}},
	}

	cc := NewChunkCollection(chunks)
	total := cc.GetTotalWords()

	if total != 150 {
		t.Errorf("Expected 150 total words, got %d", total)
	}
}

func TestChunkCollection_Statistics(t *testing.T) {
	chunks := []*Chunk{
		{Metadata: ChunkMetadata{
			SectionTitle:    "Section A",
			PageStart:       1,
			PageEnd:         2,
			EstimatedTokens: 100,
			WordCount:       50,
			CharCount:       200,
			HasTable:        true,
		}},
		{Metadata: ChunkMetadata{
			SectionTitle:    "Section B",
			PageStart:       3,
			PageEnd:         5,
			EstimatedTokens: 200,
			WordCount:       100,
			CharCount:       400,
			HasList:         true,
		}},
		{Metadata: ChunkMetadata{
			SectionTitle:    "Section A", // duplicate section
			PageStart:       6,
			PageEnd:         6,
			EstimatedTokens: 150,
			WordCount:       75,
			CharCount:       300,
			HasImage:        true,
		}},
	}

	cc := NewChunkCollection(chunks)
	stats := cc.Statistics()

	if stats.TotalChunks != 3 {
		t.Errorf("Expected 3 total chunks, got %d", stats.TotalChunks)
	}

	if stats.TotalTokens != 450 {
		t.Errorf("Expected 450 total tokens, got %d", stats.TotalTokens)
	}

	if stats.TotalWords != 225 {
		t.Errorf("Expected 225 total words, got %d", stats.TotalWords)
	}

	if stats.TotalChars != 900 {
		t.Errorf("Expected 900 total chars, got %d", stats.TotalChars)
	}

	if stats.AvgTokens != 150 {
		t.Errorf("Expected 150 avg tokens, got %d", stats.AvgTokens)
	}

	if stats.MinTokens != 100 {
		t.Errorf("Expected 100 min tokens, got %d", stats.MinTokens)
	}

	if stats.MaxTokens != 200 {
		t.Errorf("Expected 200 max tokens, got %d", stats.MaxTokens)
	}

	if stats.ChunksWithTables != 1 {
		t.Errorf("Expected 1 chunk with tables, got %d", stats.ChunksWithTables)
	}

	if stats.ChunksWithLists != 1 {
		t.Errorf("Expected 1 chunk with lists, got %d", stats.ChunksWithLists)
	}

	if stats.ChunksWithImages != 1 {
		t.Errorf("Expected 1 chunk with images, got %d", stats.ChunksWithImages)
	}

	if stats.UniqueSections != 2 {
		t.Errorf("Expected 2 unique sections, got %d", stats.UniqueSections)
	}

	if stats.PageStart != 1 {
		t.Errorf("Expected page start 1, got %d", stats.PageStart)
	}

	if stats.PageEnd != 6 {
		t.Errorf("Expected page end 6, got %d", stats.PageEnd)
	}
}

func TestChunkCollection_Statistics_Empty(t *testing.T) {
	cc := NewChunkCollection(nil)
	stats := cc.Statistics()

	if stats.TotalChunks != 0 {
		t.Errorf("Expected 0 total chunks, got %d", stats.TotalChunks)
	}

	if stats.TotalTokens != 0 {
		t.Errorf("Expected 0 total tokens, got %d", stats.TotalTokens)
	}
}

func TestCollectionStats_ToJSON(t *testing.T) {
	stats := &CollectionStats{
		TotalChunks: 10,
		TotalTokens: 1000,
		TotalWords:  500,
	}

	data, err := stats.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if result["TotalChunks"].(float64) != 10 {
		t.Error("Expected TotalChunks in JSON output")
	}
}

func TestChunkCollection_Chaining(t *testing.T) {
	chunks := []*Chunk{
		{ID: "1", Text: "Introduction to methods", Metadata: ChunkMetadata{
			SectionTitle:    "Intro",
			PageStart:       1,
			PageEnd:         1,
			EstimatedTokens: 100,
		}},
		{ID: "2", Text: "Advanced methods", Metadata: ChunkMetadata{
			SectionTitle:    "Methods",
			PageStart:       2,
			PageEnd:         2,
			EstimatedTokens: 200,
		}},
		{ID: "3", Text: "Results analysis", Metadata: ChunkMetadata{
			SectionTitle:    "Results",
			PageStart:       3,
			PageEnd:         3,
			EstimatedTokens: 150,
		}},
	}

	cc := NewChunkCollection(chunks)

	// Chain multiple filters
	result := cc.
		Search("methods").
		FilterByMinTokens(150)

	if result.Count() != 1 {
		t.Errorf("Expected 1 chunk after chained filters, got %d", result.Count())
	}
	if result.First().ID != "2" {
		t.Errorf("Expected chunk '2', got '%s'", result.First().ID)
	}
}

// Benchmarks

func BenchmarkChunkCollection_Search(b *testing.B) {
	chunks := make([]*Chunk, 1000)
	for i := range chunks {
		chunks[i] = &Chunk{
			ID:   "chunk",
			Text: "This is some sample text with various keywords for searching and filtering.",
		}
	}

	cc := NewChunkCollection(chunks)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cc.Search("keywords")
	}
}

func BenchmarkChunkCollection_Statistics(b *testing.B) {
	chunks := make([]*Chunk, 1000)
	for i := range chunks {
		chunks[i] = &Chunk{
			Metadata: ChunkMetadata{
				SectionTitle:    "Section",
				PageStart:       i + 1,
				PageEnd:         i + 1,
				EstimatedTokens: 100,
				WordCount:       50,
				CharCount:       200,
			},
		}
	}

	cc := NewChunkCollection(chunks)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cc.Statistics()
	}
}

func BenchmarkChunk_GenerateContextText(b *testing.B) {
	chunk := &Chunk{
		Text: "This is the main content of the chunk for testing context generation.",
		Metadata: ChunkMetadata{
			DocumentTitle: "Test Document",
			SectionTitle:  "Test Section",
			SectionPath:   []string{"Chapter 1", "Section A", "Test Section"},
			PageStart:     5,
			PageEnd:       5,
		},
	}

	config := MetadataConfig{
		ContextFormat:        ContextFormatBracket,
		IncludeDocumentTitle: true,
		IncludePageNumbers:   true,
		IncludeSectionPath:   true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunk.GenerateContextText(config)
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
