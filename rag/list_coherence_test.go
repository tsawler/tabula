package rag

import (
	"strings"
	"testing"
)

func TestListType_String(t *testing.T) {
	tests := []struct {
		listType ListType
		want     string
	}{
		{ListTypeUnordered, "unordered"},
		{ListTypeOrdered, "ordered"},
		{ListTypeDefinition, "definition"},
		{ListTypeChecklist, "checklist"},
		{ListType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.listType.String(); got != tt.want {
				t.Errorf("ListType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultListCoherenceConfig(t *testing.T) {
	config := DefaultListCoherenceConfig()

	if !config.KeepIntroWithList {
		t.Error("Expected KeepIntroWithList to be true")
	}

	if config.MaxIntroDistance != 200 {
		t.Errorf("Expected MaxIntroDistance 200, got %d", config.MaxIntroDistance)
	}

	if !config.PreserveNesting {
		t.Error("Expected PreserveNesting to be true")
	}

	if config.MinItemsBeforeSplit != 3 {
		t.Errorf("Expected MinItemsBeforeSplit 3, got %d", config.MinItemsBeforeSplit)
	}

	if len(config.IntroPatterns) == 0 {
		t.Error("Expected IntroPatterns to be populated")
	}
}

func TestNewListCoherenceAnalyzer(t *testing.T) {
	analyzer := NewListCoherenceAnalyzer()
	if analyzer == nil {
		t.Error("NewListCoherenceAnalyzer returned nil")
	}
}

func TestListCoherenceAnalyzer_IsListIntro(t *testing.T) {
	analyzer := NewListCoherenceAnalyzer()

	tests := []struct {
		name string
		text string
		want bool
	}{
		// Positive cases
		{"the following", "The following:", true},
		{"here are", "Here are:", true},
		{"these include", "These include:", true},
		{"below is", "Below is:", true},
		{"as follows", "As follows:", true},
		{"steps colon", "Steps:", true},
		{"features colon", "Features:", true},
		{"requirements colon", "Requirements:", true},
		{"including", "Including:", true},
		{"such as", "Such as:", true},
		{"for example", "For example:", true},
		{"you can", "You can:", true},
		{"we should", "We should:", true},
		{"ends with colon", "Important notes:", true},

		// Negative cases
		{"plain sentence", "This is a regular sentence.", false},
		{"no colon", "The following features are great", false},
		{"empty", "", false},
		{"just whitespace", "   ", false},
		{"question", "What are the features?", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.IsListIntro(tt.text)
			if got != tt.want {
				t.Errorf("IsListIntro(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestListCoherenceAnalyzer_DetectListType(t *testing.T) {
	analyzer := NewListCoherenceAnalyzer()

	tests := []struct {
		name string
		text string
		want ListType
	}{
		{
			"ordered numbers",
			"1. First item\n2. Second item\n3. Third item",
			ListTypeOrdered,
		},
		{
			"ordered letters",
			"a. First item\na. Second item",
			ListTypeOrdered,
		},
		{
			"unordered bullets",
			"• First item\n• Second item",
			ListTypeUnordered,
		},
		{
			"unordered dashes",
			"- First item\n- Second item",
			ListTypeUnordered,
		},
		{
			"unordered asterisks",
			"* First item\n* Second item",
			ListTypeUnordered,
		},
		{
			"checklist",
			"[x] Done task\n[ ] Pending task",
			ListTypeChecklist,
		},
		{
			"definition",
			"Term: Definition here\nAnother: Another definition",
			ListTypeDefinition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.DetectListType(tt.text)
			if got != tt.want {
				t.Errorf("DetectListType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListCoherenceAnalyzer_ParseListItems(t *testing.T) {
	analyzer := NewListCoherenceAnalyzer()

	t.Run("simple ordered list", func(t *testing.T) {
		text := "1. First item\n2. Second item\n3. Third item"
		items := analyzer.ParseListItems(text)

		if len(items) != 3 {
			t.Fatalf("Expected 3 items, got %d", len(items))
		}

		if items[0].Text != "First item" {
			t.Errorf("Expected 'First item', got %q", items[0].Text)
		}
		if items[0].Marker != "1." {
			t.Errorf("Expected marker '1.', got %q", items[0].Marker)
		}
	})

	t.Run("simple unordered list", func(t *testing.T) {
		text := "• First item\n• Second item"
		items := analyzer.ParseListItems(text)

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(items))
		}

		if items[0].Marker != "•" {
			t.Errorf("Expected marker '•', got %q", items[0].Marker)
		}
	})

	t.Run("nested list", func(t *testing.T) {
		text := "1. First item\n  a. Nested item\n  b. Another nested\n2. Second item"
		items := analyzer.ParseListItems(text)

		if len(items) != 2 {
			t.Fatalf("Expected 2 top-level items, got %d", len(items))
		}

		if len(items[0].Children) != 2 {
			t.Errorf("Expected 2 children in first item, got %d", len(items[0].Children))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		text := ""
		items := analyzer.ParseListItems(text)

		if len(items) != 0 {
			t.Errorf("Expected 0 items for empty text, got %d", len(items))
		}
	})

	t.Run("continuation lines", func(t *testing.T) {
		text := "1. First item that\ncontinues on next line\n2. Second item"
		items := analyzer.ParseListItems(text)

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(items))
		}

		if !strings.Contains(items[0].Text, "continues on next line") {
			t.Error("Expected continuation to be merged with first item")
		}
	})
}

func TestListCoherenceAnalyzer_AnalyzeListBlock(t *testing.T) {
	analyzer := NewListCoherenceAnalyzer()

	t.Run("list with intro", func(t *testing.T) {
		listText := "1. First\n2. Second\n3. Third"
		introText := "The following items:"

		block := analyzer.AnalyzeListBlock(listText, introText)

		if !block.HasIntro {
			t.Error("Expected HasIntro to be true")
		}
		if block.IntroText != introText {
			t.Errorf("Expected intro text %q, got %q", introText, block.IntroText)
		}
		if block.TotalItems != 3 {
			t.Errorf("Expected 3 total items, got %d", block.TotalItems)
		}
		if block.Type != ListTypeOrdered {
			t.Errorf("Expected ListTypeOrdered, got %v", block.Type)
		}
	})

	t.Run("list without intro", func(t *testing.T) {
		listText := "• Item one\n• Item two"
		introText := "This is not an intro."

		block := analyzer.AnalyzeListBlock(listText, introText)

		if block.HasIntro {
			t.Error("Expected HasIntro to be false")
		}
	})

	t.Run("nested list stats", func(t *testing.T) {
		listText := "1. Parent\n  a. Child 1\n  b. Child 2\n2. Another parent"
		block := analyzer.AnalyzeListBlock(listText, "")

		if block.MaxLevel != 1 {
			t.Errorf("Expected MaxLevel 1, got %d", block.MaxLevel)
		}
		if block.TotalItems != 4 {
			t.Errorf("Expected 4 total items, got %d", block.TotalItems)
		}
	})
}

func TestListCoherenceAnalyzer_FindListSplitPoints(t *testing.T) {
	analyzer := NewListCoherenceAnalyzer()

	t.Run("list with enough items", func(t *testing.T) {
		block := &ListBlock{
			Items: []*ListItem{
				{Text: "Item 1", Level: 0},
				{Text: "Item 2", Level: 0},
				{Text: "Item 3", Level: 0},
				{Text: "Item 4", Level: 0},
				{Text: "Item 5", Level: 0},
			},
			TotalItems: 5,
		}

		points := analyzer.FindListSplitPoints(block)

		if len(points) == 0 {
			t.Error("Expected at least one split point")
		}
	})

	t.Run("list too short", func(t *testing.T) {
		block := &ListBlock{
			Items: []*ListItem{
				{Text: "Item 1", Level: 0},
				{Text: "Item 2", Level: 0},
			},
			TotalItems: 2,
		}

		points := analyzer.FindListSplitPoints(block)

		if len(points) != 0 {
			t.Errorf("Expected no split points for short list, got %d", len(points))
		}
	})

	t.Run("skip items with children when preserving nesting", func(t *testing.T) {
		analyzer := NewListCoherenceAnalyzerWithConfig(ListCoherenceConfig{
			MinItemsBeforeSplit: 2,
			PreserveNesting:     true,
		})

		block := &ListBlock{
			Items: []*ListItem{
				{Text: "Item 1", Level: 0},
				{Text: "Item 2", Level: 0},
				{Text: "Item 3", Level: 0, Children: []*ListItem{{Text: "Child"}}},
				{Text: "Item 4", Level: 0},
			},
			TotalItems: 5,
		}

		points := analyzer.FindListSplitPoints(block)

		// Should not include index 3 (Item 3 has children)
		for _, p := range points {
			if p == 2 { // Index of "Item 3"
				t.Error("Should not split at item with children when preserving nesting")
			}
		}
	})
}

func TestListCoherenceAnalyzer_SplitListBlock(t *testing.T) {
	analyzer := NewListCoherenceAnalyzer()

	block := &ListBlock{
		Type:      ListTypeOrdered,
		IntroText: "Introduction:",
		HasIntro:  true,
		Items: []*ListItem{
			{Text: "Item 1", Marker: "1."},
			{Text: "Item 2", Marker: "2."},
			{Text: "Item 3", Marker: "3."},
			{Text: "Item 4", Marker: "4."},
		},
		TotalItems: 4,
		IsComplete: true,
	}

	t.Run("split in middle", func(t *testing.T) {
		first, second := analyzer.SplitListBlock(block, 2)

		if first == nil || second == nil {
			t.Fatal("Expected two blocks")
		}

		if len(first.Items) != 2 {
			t.Errorf("Expected 2 items in first block, got %d", len(first.Items))
		}
		if len(second.Items) != 2 {
			t.Errorf("Expected 2 items in second block, got %d", len(second.Items))
		}

		if !first.HasIntro {
			t.Error("First block should have intro")
		}
		if second.HasIntro {
			t.Error("Second block should not have intro")
		}

		if first.IsComplete {
			t.Error("First block should not be complete")
		}
		if !second.IsComplete {
			t.Error("Second block should be complete")
		}
	})

	t.Run("invalid split index", func(t *testing.T) {
		first, second := analyzer.SplitListBlock(block, 0)
		if second != nil {
			t.Error("Expected nil second block for invalid split")
		}
		if first != block {
			t.Error("Expected original block returned for invalid split")
		}

		first, second = analyzer.SplitListBlock(block, 10)
		if second != nil {
			t.Error("Expected nil second block for out of range split")
		}
	})
}

func TestListCoherenceAnalyzer_FormatListBlock(t *testing.T) {
	analyzer := NewListCoherenceAnalyzer()

	t.Run("with intro and markers", func(t *testing.T) {
		block := &ListBlock{
			IntroText: "The features:",
			HasIntro:  true,
			Items: []*ListItem{
				{Text: "Feature one", Marker: "1."},
				{Text: "Feature two", Marker: "2."},
			},
		}

		result := analyzer.FormatListBlock(block, true)

		if !strings.HasPrefix(result, "The features:") {
			t.Error("Expected result to start with intro")
		}
		if !strings.Contains(result, "1. Feature one") {
			t.Error("Expected marker preserved")
		}
	})

	t.Run("without markers", func(t *testing.T) {
		block := &ListBlock{
			Items: []*ListItem{
				{Text: "Item one", Marker: "•"},
				{Text: "Item two", Marker: "•"},
			},
		}

		result := analyzer.FormatListBlock(block, false)

		if strings.Contains(result, "•") {
			t.Error("Expected markers to be removed")
		}
		if !strings.Contains(result, "Item one") {
			t.Error("Expected item text preserved")
		}
	})

	t.Run("with nested items", func(t *testing.T) {
		block := &ListBlock{
			Items: []*ListItem{
				{
					Text:   "Parent",
					Marker: "1.",
					Children: []*ListItem{
						{Text: "Child", Marker: "a."},
					},
				},
			},
		}

		result := analyzer.FormatListBlock(block, true)

		if !strings.Contains(result, "Parent") {
			t.Error("Expected parent item")
		}
		if !strings.Contains(result, "Child") {
			t.Error("Expected child item")
		}
	})
}

func TestListCoherenceAnalyzer_ShouldKeepListTogether(t *testing.T) {
	t.Run("small list", func(t *testing.T) {
		analyzer := NewListCoherenceAnalyzer()
		block := &ListBlock{
			Items: []*ListItem{
				{Text: "Short item"},
				{Text: "Another short"},
			},
			TotalItems: 2,
		}

		if !analyzer.ShouldKeepListTogether(block) {
			t.Error("Small list should be kept together")
		}
	})

	t.Run("large list", func(t *testing.T) {
		config := DefaultListCoherenceConfig()
		config.MaxListSize = 50
		config.MinItemsBeforeSplit = 2
		analyzer := NewListCoherenceAnalyzerWithConfig(config)

		block := &ListBlock{
			Items: []*ListItem{
				{Text: "This is a very long item that exceeds the max size"},
				{Text: "Another very long item that also exceeds"},
				{Text: "Third very long item here"},
				{Text: "Fourth item"},
			},
			TotalItems: 4,
		}

		if analyzer.ShouldKeepListTogether(block) {
			t.Error("Large list should be split")
		}
	})

	t.Run("nested list with preserve nesting", func(t *testing.T) {
		config := DefaultListCoherenceConfig()
		config.MaxListSize = 50
		config.PreserveNesting = true
		analyzer := NewListCoherenceAnalyzerWithConfig(config)

		block := &ListBlock{
			Items: []*ListItem{
				{Text: "Parent with long text here", Children: []*ListItem{{Text: "Child"}}},
				{Text: "Another parent", Children: []*ListItem{{Text: "Child"}}},
				{Text: "Third parent", Children: []*ListItem{{Text: "Child"}}},
			},
			TotalItems: 6,
			MaxLevel:   1,
		}

		// Should keep together because all items have children
		if !analyzer.ShouldKeepListTogether(block) {
			t.Error("Nested list with no safe split points should be kept together")
		}
	})
}

func TestIsOrderedListItem(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"1. First item", true},
		{"2) Second item", true},
		{"a. Lettered item", true},
		{"b) Another letter", true},
		{"i. Roman numeral", true},
		{"(1) Parenthesized", true},
		{"(a) Also valid", true},
		{"• Bullet item", false},
		{"- Dash item", false},
		{"Regular text", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isOrderedListItem(tt.line); got != tt.want {
				t.Errorf("isOrderedListItem(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsUnorderedListItem(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"• Bullet", true},
		{"- Dash", true},
		{"* Asterisk", true},
		{"● Filled circle", true},
		{"○ Empty circle", true},
		{"■ Square", true},
		{"1. Numbered", false},
		{"Regular text", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isUnorderedListItem(tt.line); got != tt.want {
				t.Errorf("isUnorderedListItem(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsChecklistItem(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"[x] Checked", true},
		{"[ ] Unchecked", true},
		{"[X] Also checked", true},
		{"☐ Empty checkbox", true},
		{"☑ Checked checkbox", true},
		{"• Regular bullet", false},
		{"1. Numbered", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isChecklistItem(tt.line); got != tt.want {
				t.Errorf("isChecklistItem(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsDefinitionItem(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"Term: Definition here", true},
		{"Short: Also a definition", true},
		{"This is a sentence. Not a definition.", false},
		{"URL: http://example.com", true},
		{"Very long term that exceeds the maximum allowed length for a term: definition", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isDefinitionItem(tt.line); got != tt.want {
				t.Errorf("isDefinitionItem(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestGetIndentLevel(t *testing.T) {
	tests := []struct {
		line string
		want int
	}{
		{"No indent", 0},
		{"  Two spaces", 1},
		{"    Four spaces", 2},
		{"\tTab", 2},
		{"      Six spaces", 3},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := getIndentLevel(tt.line); got != tt.want {
				t.Errorf("getIndentLevel(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestExtractListMarker(t *testing.T) {
	tests := []struct {
		line       string
		wantMarker string
		wantText   string
	}{
		{"1. First item", "1.", "First item"},
		{"a) Lettered", "a)", "Lettered"},
		{"• Bullet item", "•", "Bullet item"},
		{"- Dash item", "-", "Dash item"},
		{"[x] Checked", "[x]", "Checked"},
		{"Regular text", "", "Regular text"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			marker, text := extractListMarker(tt.line)
			if marker != tt.wantMarker {
				t.Errorf("marker = %q, want %q", marker, tt.wantMarker)
			}
			if text != tt.wantText {
				t.Errorf("text = %q, want %q", text, tt.wantText)
			}
		})
	}
}

func TestIsListMarker(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"1. Item", true},
		{"• Item", true},
		{"- Item", true},
		{"[x] Item", true},
		{"Just text", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := IsListMarker(tt.text); got != tt.want {
				t.Errorf("IsListMarker(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestGetListMarkerType(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"1. Ordered", "ordered"},
		{"• Unordered", "unordered"},
		{"[x] Checklist", "checklist"},
		{"Term: Definition", "definition"},
		{"Plain text", ""},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := GetListMarkerType(tt.line); got != tt.want {
				t.Errorf("GetListMarkerType(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestNormalizeListMarkers(t *testing.T) {
	t.Run("to numbers", func(t *testing.T) {
		text := "• First\n• Second\n• Third"
		result := NormalizeListMarkers(text, true)

		if !strings.Contains(result, "1. First") {
			t.Error("Expected numbered markers")
		}
		if !strings.Contains(result, "2. Second") {
			t.Error("Expected numbered markers")
		}
	})

	t.Run("to bullets", func(t *testing.T) {
		text := "1. First\n2. Second"
		result := NormalizeListMarkers(text, false)

		if !strings.Contains(result, "• First") {
			t.Error("Expected bullet markers")
		}
	})

	t.Run("preserve non-list lines", func(t *testing.T) {
		text := "Intro text\n1. Item\nConclusion"
		result := NormalizeListMarkers(text, false)

		if !strings.Contains(result, "Intro text") {
			t.Error("Expected non-list lines preserved")
		}
		if !strings.Contains(result, "Conclusion") {
			t.Error("Expected non-list lines preserved")
		}
	})
}

func TestListCoherenceAnalyzer_AnalyzeListCoherence(t *testing.T) {
	analyzer := NewListCoherenceAnalyzer()

	t.Run("list with intro", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: 1, Text: "The following features:"},  // Paragraph (intro)
			{Type: 5, Text: "• Feature 1\n• Feature 2"}, // List
		}

		result := analyzer.AnalyzeListCoherence(blocks)

		if result.TotalLists != 1 {
			t.Errorf("Expected 1 list, got %d", result.TotalLists)
		}
		if result.ListsWithIntros != 1 {
			t.Errorf("Expected 1 list with intro, got %d", result.ListsWithIntros)
		}
		if len(result.IntroOrphans) != 0 {
			t.Errorf("Expected no orphaned intros, got %d", len(result.IntroOrphans))
		}
	})

	t.Run("orphaned intro", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: 1, Text: "The following features:"}, // Paragraph (intro)
			{Type: 1, Text: "Regular paragraph"},       // Another paragraph (not a list)
		}

		result := analyzer.AnalyzeListCoherence(blocks)

		if len(result.IntroOrphans) != 1 {
			t.Errorf("Expected 1 orphaned intro, got %d", len(result.IntroOrphans))
		}
	})

	t.Run("multiple lists", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: 5, Text: "• Item 1\n• Item 2"},
			{Type: 1, Text: "Some text"},
			{Type: 5, Text: "1. First\n2. Second"},
		}

		result := analyzer.AnalyzeListCoherence(blocks)

		if result.TotalLists != 2 {
			t.Errorf("Expected 2 lists, got %d", result.TotalLists)
		}
	})
}

// Benchmarks

func BenchmarkIsListIntro(b *testing.B) {
	analyzer := NewListCoherenceAnalyzer()
	text := "The following features are available:"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.IsListIntro(text)
	}
}

func BenchmarkParseListItems(b *testing.B) {
	analyzer := NewListCoherenceAnalyzer()
	text := `1. First item with some content
2. Second item with more content
3. Third item
  a. Nested item one
  b. Nested item two
4. Fourth item
5. Fifth item`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.ParseListItems(text)
	}
}

func BenchmarkDetectListType(b *testing.B) {
	analyzer := NewListCoherenceAnalyzer()
	text := "1. First\n2. Second\n3. Third"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.DetectListType(text)
	}
}

func BenchmarkNormalizeListMarkers(b *testing.B) {
	text := "• First item\n• Second item\n• Third item\n• Fourth item\n• Fifth item"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NormalizeListMarkers(text, true)
	}
}
