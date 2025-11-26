package layout

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

func TestListTypeString(t *testing.T) {
	tests := []struct {
		listType ListType
		expected string
	}{
		{ListTypeUnknown, "unknown"},
		{ListTypeBullet, "bullet"},
		{ListTypeNumbered, "numbered"},
		{ListTypeLettered, "lettered"},
		{ListTypeRoman, "roman"},
		{ListTypeCheckbox, "checkbox"},
	}

	for _, tt := range tests {
		if got := tt.listType.String(); got != tt.expected {
			t.Errorf("ListType(%d).String() = %q, want %q", tt.listType, got, tt.expected)
		}
	}
}

func TestBulletStyleString(t *testing.T) {
	tests := []struct {
		style    BulletStyle
		expected string
	}{
		{BulletStyleUnknown, "unknown"},
		{BulletStyleDisc, "disc"},
		{BulletStyleCircle, "circle"},
		{BulletStyleSquare, "square"},
		{BulletStyleDash, "dash"},
		{BulletStyleAsterisk, "asterisk"},
		{BulletStyleArrow, "arrow"},
		{BulletStyleCheckEmpty, "checkbox-empty"},
		{BulletStyleCheckFilled, "checkbox-filled"},
	}

	for _, tt := range tests {
		if got := tt.style.String(); got != tt.expected {
			t.Errorf("BulletStyle(%d).String() = %q, want %q", tt.style, got, tt.expected)
		}
	}
}

func TestNewListDetector(t *testing.T) {
	detector := NewListDetector()
	if detector == nil {
		t.Fatal("NewListDetector returned nil")
	}
}

func TestNewListDetectorWithConfig(t *testing.T) {
	config := ListConfig{
		IndentThreshold:     20.0,
		MinConsecutiveItems: 3,
	}
	detector := NewListDetectorWithConfig(config)
	if detector == nil {
		t.Fatal("NewListDetectorWithConfig returned nil")
	}
	if detector.config.IndentThreshold != 20.0 {
		t.Errorf("Expected IndentThreshold=20.0, got %f", detector.config.IndentThreshold)
	}
}

func TestDefaultListConfig(t *testing.T) {
	config := DefaultListConfig()

	if len(config.BulletCharacters) == 0 {
		t.Error("Expected BulletCharacters to be populated")
	}
	if config.IndentThreshold != 15.0 {
		t.Errorf("Expected IndentThreshold=15.0, got %f", config.IndentThreshold)
	}
	if config.MinConsecutiveItems != 2 {
		t.Errorf("Expected MinConsecutiveItems=2, got %d", config.MinConsecutiveItems)
	}
}

func TestListDetector_DetectFromParagraphs_Empty(t *testing.T) {
	detector := NewListDetector()
	result := detector.DetectFromParagraphs([]Paragraph{}, 612, 792)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.ListCount() != 0 {
		t.Errorf("Expected 0 lists, got %d", result.ListCount())
	}
}

func TestDetectFromParagraphs_BulletList(t *testing.T) {
	paragraphs := []Paragraph{
		{Text: "• First item", BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "• Second item", BBox: model.BBox{X: 72, Y: 680, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "• Third item", BBox: model.BBox{X: 72, Y: 660, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
	}

	detector := NewListDetector()
	result := detector.DetectFromParagraphs(paragraphs, 612, 792)

	if result.ListCount() != 1 {
		t.Fatalf("Expected 1 list, got %d", result.ListCount())
	}

	list := result.GetList(0)
	if list.Type != ListTypeBullet {
		t.Errorf("Expected bullet list, got %v", list.Type)
	}
	if len(list.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(list.Items))
	}
}

func TestDetectFromParagraphs_NumberedList(t *testing.T) {
	paragraphs := []Paragraph{
		{Text: "1. First step", BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "2. Second step", BBox: model.BBox{X: 72, Y: 680, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "3. Third step", BBox: model.BBox{X: 72, Y: 660, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
	}

	detector := NewListDetector()
	result := detector.DetectFromParagraphs(paragraphs, 612, 792)

	if result.ListCount() != 1 {
		t.Fatalf("Expected 1 list, got %d", result.ListCount())
	}

	list := result.GetList(0)
	if list.Type != ListTypeNumbered {
		t.Errorf("Expected numbered list, got %v", list.Type)
	}
	if len(list.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(list.Items))
	}

	// Check numbers
	for i, item := range list.Items {
		if item.Number != i+1 {
			t.Errorf("Item %d: expected number %d, got %d", i, i+1, item.Number)
		}
	}
}

func TestDetectFromParagraphs_LetteredList(t *testing.T) {
	paragraphs := []Paragraph{
		{Text: "a. Option A", BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "b. Option B", BBox: model.BBox{X: 72, Y: 680, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "c. Option C", BBox: model.BBox{X: 72, Y: 660, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
	}

	detector := NewListDetector()
	result := detector.DetectFromParagraphs(paragraphs, 612, 792)

	if result.ListCount() != 1 {
		t.Fatalf("Expected 1 list, got %d", result.ListCount())
	}

	list := result.GetList(0)
	if list.Type != ListTypeLettered {
		t.Errorf("Expected lettered list, got %v", list.Type)
	}
}

func TestDetectFromParagraphs_DashBullets(t *testing.T) {
	paragraphs := []Paragraph{
		{Text: "- First item", BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "- Second item", BBox: model.BBox{X: 72, Y: 680, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
	}

	detector := NewListDetector()
	result := detector.DetectFromParagraphs(paragraphs, 612, 792)

	if result.ListCount() != 1 {
		t.Fatalf("Expected 1 list, got %d", result.ListCount())
	}

	list := result.GetList(0)
	if list.BulletStyle != BulletStyleDash {
		t.Errorf("Expected dash bullet style, got %v", list.BulletStyle)
	}
}

func TestDetectFromParagraphs_AsteriskBullets(t *testing.T) {
	paragraphs := []Paragraph{
		{Text: "* Item one", BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "* Item two", BBox: model.BBox{X: 72, Y: 680, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
	}

	detector := NewListDetector()
	result := detector.DetectFromParagraphs(paragraphs, 612, 792)

	if result.ListCount() != 1 {
		t.Fatalf("Expected 1 list, got %d", result.ListCount())
	}

	list := result.GetList(0)
	if list.BulletStyle != BulletStyleAsterisk {
		t.Errorf("Expected asterisk bullet style, got %v", list.BulletStyle)
	}
}

func TestDetectFromParagraphs_NestedList(t *testing.T) {
	paragraphs := []Paragraph{
		{Text: "• Parent item", BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "• Child item 1", BBox: model.BBox{X: 92, Y: 680, Width: 180, Height: 14}, LeftMargin: 92, AverageFontSize: 12},
		{Text: "• Child item 2", BBox: model.BBox{X: 92, Y: 660, Width: 180, Height: 14}, LeftMargin: 92, AverageFontSize: 12},
		{Text: "• Another parent", BBox: model.BBox{X: 72, Y: 640, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
	}

	detector := NewListDetector()
	result := detector.DetectFromParagraphs(paragraphs, 612, 792)

	if result.ListCount() != 1 {
		t.Fatalf("Expected 1 list, got %d", result.ListCount())
	}

	list := result.GetList(0)
	if !list.HasNesting() {
		t.Error("Expected list to have nesting")
	}
}

func TestDetectFromParagraphs_MultipleLists(t *testing.T) {
	paragraphs := []Paragraph{
		// First list
		{Text: "• Bullet 1", BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "• Bullet 2", BBox: model.BBox{X: 72, Y: 680, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		// Gap - not a list item
		{Text: "Some regular text in between.", BBox: model.BBox{X: 72, Y: 600, Width: 300, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		// Second list
		{Text: "1. Step one", BBox: model.BBox{X: 72, Y: 550, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "2. Step two", BBox: model.BBox{X: 72, Y: 530, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
	}

	detector := NewListDetector()
	result := detector.DetectFromParagraphs(paragraphs, 612, 792)

	if result.ListCount() != 2 {
		t.Errorf("Expected 2 lists, got %d", result.ListCount())
	}
}

func TestDetectFromParagraphs_Checkbox(t *testing.T) {
	paragraphs := []Paragraph{
		{Text: "☐ Unchecked task", BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "☑ Checked task", BBox: model.BBox{X: 72, Y: 680, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
	}

	detector := NewListDetector()
	result := detector.DetectFromParagraphs(paragraphs, 612, 792)

	if result.ListCount() != 1 {
		t.Fatalf("Expected 1 list, got %d", result.ListCount())
	}

	list := result.GetList(0)
	if list.Type != ListTypeCheckbox {
		t.Errorf("Expected checkbox list, got %v", list.Type)
	}

	// Check first item is unchecked
	if list.Items[0].IsChecked() {
		t.Error("First item should be unchecked")
	}

	// Check second item is checked
	if !list.Items[1].IsChecked() {
		t.Error("Second item should be checked")
	}
}

func TestDetectFromParagraphs_RomanNumerals(t *testing.T) {
	// Note: Using multi-character roman numerals to avoid conflict with lettered list pattern
	// Single "i." matches lettered pattern (letter a-z followed by dot)
	paragraphs := []Paragraph{
		{Text: "II. Second point", BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "III. Third point", BBox: model.BBox{X: 72, Y: 680, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
		{Text: "IV. Fourth point", BBox: model.BBox{X: 72, Y: 660, Width: 200, Height: 14}, LeftMargin: 72, AverageFontSize: 12},
	}

	detector := NewListDetector()
	result := detector.DetectFromParagraphs(paragraphs, 612, 792)

	if result.ListCount() != 1 {
		t.Fatalf("Expected 1 list, got %d", result.ListCount())
	}

	list := result.GetList(0)
	if list.Type != ListTypeRoman {
		t.Errorf("Expected roman numeral list, got %v", list.Type)
	}

	// Check roman numeral conversion
	if len(list.Items) < 3 {
		t.Fatalf("Expected at least 3 items, got %d", len(list.Items))
	}
	if list.Items[0].Number != 2 {
		t.Errorf("Expected II=2, got %d", list.Items[0].Number)
	}
	if list.Items[1].Number != 3 {
		t.Errorf("Expected III=3, got %d", list.Items[1].Number)
	}
	if list.Items[2].Number != 4 {
		t.Errorf("Expected IV=4, got %d", list.Items[2].Number)
	}
}

func TestListDetector_DetectFromLines(t *testing.T) {
	lines := []Line{
		{Text: "• Item 1", BBox: model.BBox{X: 72, Y: 700, Width: 200, Height: 14}, AverageFontSize: 12},
		{Text: "• Item 2", BBox: model.BBox{X: 72, Y: 680, Width: 200, Height: 14}, AverageFontSize: 12},
	}

	detector := NewListDetector()
	result := detector.DetectFromLines(lines, 612, 792)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestListDetector_DetectFromFragments(t *testing.T) {
	fragments := []text.TextFragment{
		{Text: "• Item 1", X: 72, Y: 700, Width: 200, Height: 14, FontSize: 12},
		{Text: "• Item 2", X: 72, Y: 680, Width: 200, Height: 14, FontSize: 12},
	}

	detector := NewListDetector()
	result := detector.DetectFromFragments(fragments, 612, 792)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestDetectBullet(t *testing.T) {
	detector := NewListDetector()

	tests := []struct {
		name        string
		text        string
		expectType  ListType
		expectStyle BulletStyle
	}{
		{"disc bullet", "• Item text", ListTypeBullet, BulletStyleDisc},
		{"dash bullet", "- Item text", ListTypeBullet, BulletStyleDash},
		{"asterisk bullet", "* Item text", ListTypeBullet, BulletStyleAsterisk},
		{"arrow bullet", "→ Item text", ListTypeBullet, BulletStyleArrow},
		{"square bullet", "■ Item text", ListTypeBullet, BulletStyleSquare},
		{"checkbox empty", "☐ Task item", ListTypeCheckbox, BulletStyleCheckEmpty},
		{"checkbox filled", "☑ Done task", ListTypeCheckbox, BulletStyleCheckFilled},
		{"not a bullet", "Regular text", ListTypeUnknown, BulletStyleUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listType, bulletStyle, _, _ := detector.detectBullet(tt.text)
			if listType != tt.expectType {
				t.Errorf("detectBullet(%q) type = %v, want %v", tt.text, listType, tt.expectType)
			}
			if bulletStyle != tt.expectStyle {
				t.Errorf("detectBullet(%q) style = %v, want %v", tt.text, bulletStyle, tt.expectStyle)
			}
		})
	}
}

func TestListDetector_DetectNumbered(t *testing.T) {
	detector := NewListDetector()

	tests := []struct {
		name       string
		text       string
		expectType ListType
		expectNum  int
	}{
		{"number dot", "1. First item", ListTypeNumbered, 1},
		{"number paren", "2) Second item", ListTypeNumbered, 2},
		{"double digit", "10. Tenth item", ListTypeNumbered, 10},
		{"letter dot", "a. Letter item", ListTypeLettered, 1},
		{"letter paren", "b) Letter item", ListTypeLettered, 2},
		{"uppercase letter", "C. Upper letter", ListTypeLettered, 3},
		{"roman lower", "iv. Roman four", ListTypeRoman, 4},
		{"roman upper", "IX. Roman nine", ListTypeRoman, 9},
		{"not numbered", "Regular text", ListTypeUnknown, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listType, _, _, num := detector.detectNumbered(tt.text)
			if listType != tt.expectType {
				t.Errorf("detectNumbered(%q) type = %v, want %v", tt.text, listType, tt.expectType)
			}
			if num != tt.expectNum {
				t.Errorf("detectNumbered(%q) number = %d, want %d", tt.text, num, tt.expectNum)
			}
		})
	}
}

func TestRomanToNumber(t *testing.T) {
	detector := NewListDetector()

	tests := []struct {
		roman    string
		expected int
	}{
		{"I", 1},
		{"II", 2},
		{"III", 3},
		{"IV", 4},
		{"V", 5},
		{"VI", 6},
		{"VII", 7},
		{"VIII", 8},
		{"IX", 9},
		{"X", 10},
		{"XI", 11},
		{"XIV", 14},
		{"XV", 15},
		{"XIX", 19},
		{"XX", 20},
		{"XL", 40},
		{"L", 50},
		{"XC", 90},
		{"C", 100},
		{"CD", 400},
		{"D", 500},
		{"CM", 900},
		{"M", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.roman, func(t *testing.T) {
			got := detector.romanToNumber(tt.roman)
			if got != tt.expected {
				t.Errorf("romanToNumber(%q) = %d, want %d", tt.roman, got, tt.expected)
			}
		})
	}
}

func TestLetterToNumber(t *testing.T) {
	detector := NewListDetector()

	tests := []struct {
		letter   string
		expected int
	}{
		{"a", 1},
		{"b", 2},
		{"c", 3},
		{"z", 26},
		{"A", 1},
		{"B", 2},
		{"Z", 26},
	}

	for _, tt := range tests {
		t.Run(tt.letter, func(t *testing.T) {
			got := detector.letterToNumber(tt.letter)
			if got != tt.expected {
				t.Errorf("letterToNumber(%q) = %d, want %d", tt.letter, got, tt.expected)
			}
		})
	}
}

func TestIsValidRoman(t *testing.T) {
	detector := NewListDetector()

	tests := []struct {
		input    string
		expected bool
	}{
		{"I", true},
		{"IV", true},
		{"MCMXCIX", true},
		{"A", false},   // A is not a roman numeral
		{"ABC", false}, // B is not valid
		{"", false},    // Empty string
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := detector.isValidRoman(tt.input)
			if got != tt.expected {
				t.Errorf("isValidRoman(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestListLayout_Methods(t *testing.T) {
	layout := &ListLayout{
		Lists: []List{
			{Type: ListTypeBullet, Items: []ListItem{{Text: "Item 1"}, {Text: "Item 2"}}},
			{Type: ListTypeNumbered, Items: []ListItem{{Text: "Step 1"}}},
		},
		AllItems: []ListItem{{Text: "Item 1"}, {Text: "Item 2"}, {Text: "Step 1"}},
	}

	if layout.ListCount() != 2 {
		t.Errorf("ListCount() = %d, want 2", layout.ListCount())
	}

	if layout.TotalItemCount() != 3 {
		t.Errorf("TotalItemCount() = %d, want 3", layout.TotalItemCount())
	}

	bulletLists := layout.GetBulletLists()
	if len(bulletLists) != 1 {
		t.Errorf("GetBulletLists() count = %d, want 1", len(bulletLists))
	}

	numberedLists := layout.GetNumberedLists()
	if len(numberedLists) != 1 {
		t.Errorf("GetNumberedLists() count = %d, want 1", len(numberedLists))
	}
}

func TestListLayout_NilSafety(t *testing.T) {
	var layout *ListLayout

	if layout.ListCount() != 0 {
		t.Error("ListCount on nil should return 0")
	}
	if layout.GetList(0) != nil {
		t.Error("GetList on nil should return nil")
	}
	if layout.GetListsByType(ListTypeBullet) != nil {
		t.Error("GetListsByType on nil should return nil")
	}
	if layout.TotalItemCount() != 0 {
		t.Error("TotalItemCount on nil should return 0")
	}
}

func TestList_GetText(t *testing.T) {
	list := &List{
		Type: ListTypeBullet,
		Items: []ListItem{
			{Text: "First item", Prefix: "•"},
			{Text: "Second item", Prefix: "•"},
		},
	}

	text := list.GetText()
	if !strings.Contains(text, "First item") {
		t.Error("GetText should contain 'First item'")
	}
	if !strings.Contains(text, "•") {
		t.Error("GetText should contain bullet prefix")
	}
}

func TestList_ToMarkdown(t *testing.T) {
	bulletList := &List{
		Type: ListTypeBullet,
		Items: []ListItem{
			{Text: "Item A"},
			{Text: "Item B"},
		},
	}

	md := bulletList.ToMarkdown()
	if !strings.Contains(md, "- Item A") {
		t.Error("Bullet list markdown should use - prefix")
	}

	numberedList := &List{
		Type: ListTypeNumbered,
		Items: []ListItem{
			{Text: "Step one"},
			{Text: "Step two"},
		},
	}

	md = numberedList.ToMarkdown()
	if !strings.Contains(md, "1.") {
		t.Error("Numbered list markdown should use 1. prefix")
	}
}

func TestList_HasNesting(t *testing.T) {
	flatList := &List{
		Items: []ListItem{
			{Text: "Item 1"},
			{Text: "Item 2"},
		},
	}
	if flatList.HasNesting() {
		t.Error("Flat list should not have nesting")
	}

	nestedList := &List{
		Items: []ListItem{
			{
				Text: "Parent",
				Children: []ListItem{
					{Text: "Child"},
				},
			},
		},
	}
	if !nestedList.HasNesting() {
		t.Error("Nested list should have nesting")
	}
}

func TestList_MaxDepth(t *testing.T) {
	tests := []struct {
		name     string
		list     *List
		expected int
	}{
		{
			name:     "nil list",
			list:     nil,
			expected: 0,
		},
		{
			name: "flat list",
			list: &List{
				Items: []ListItem{{Text: "A"}, {Text: "B"}},
			},
			expected: 0,
		},
		{
			name: "one level nesting",
			list: &List{
				Items: []ListItem{
					{Text: "A", Children: []ListItem{{Text: "A1"}}},
				},
			},
			expected: 1,
		},
		{
			name: "two level nesting",
			list: &List{
				Items: []ListItem{
					{
						Text: "A",
						Children: []ListItem{
							{Text: "A1", Children: []ListItem{{Text: "A1a"}}},
						},
					},
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.list.MaxDepth()
			if got != tt.expected {
				t.Errorf("MaxDepth() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestList_NilSafety(t *testing.T) {
	var list *List

	if list.GetAllItems() != nil {
		t.Error("GetAllItems on nil should return nil")
	}
	if list.GetText() != "" {
		t.Error("GetText on nil should return empty string")
	}
	if list.ToMarkdown() != "" {
		t.Error("ToMarkdown on nil should return empty string")
	}
	if list.HasNesting() {
		t.Error("HasNesting on nil should return false")
	}
	if list.MaxDepth() != 0 {
		t.Error("MaxDepth on nil should return 0")
	}
}

func TestListItem_Methods(t *testing.T) {
	item := &ListItem{
		Text:        "Test item content",
		RawText:     "• Test item content",
		Prefix:      "•",
		ListType:    ListTypeBullet,
		BulletStyle: BulletStyleDisc,
		Index:       0,
		Number:      1,
		BBox:        model.BBox{X: 72, Y: 700, Width: 200, Height: 14},
	}

	if item.GetFullText() != "• Test item content" {
		t.Errorf("GetFullText() = %q, want '• Test item content'", item.GetFullText())
	}

	if !item.IsFirstInList() {
		t.Error("Item with Index=0 should be first in list")
	}

	if item.WordCount() != 3 {
		t.Errorf("WordCount() = %d, want 3", item.WordCount())
	}
}

func TestListItem_Checkbox(t *testing.T) {
	unchecked := &ListItem{
		ListType:    ListTypeCheckbox,
		BulletStyle: BulletStyleCheckEmpty,
	}
	if !unchecked.IsCheckbox() {
		t.Error("Should be checkbox")
	}
	if unchecked.IsChecked() {
		t.Error("Should not be checked")
	}

	checked := &ListItem{
		ListType:    ListTypeCheckbox,
		BulletStyle: BulletStyleCheckFilled,
	}
	if !checked.IsCheckbox() {
		t.Error("Should be checkbox")
	}
	if !checked.IsChecked() {
		t.Error("Should be checked")
	}
}

func TestListItem_ContainsPoint(t *testing.T) {
	item := &ListItem{
		BBox: model.BBox{X: 100, Y: 200, Width: 300, Height: 20},
	}

	if !item.ContainsPoint(200, 210) {
		t.Error("Point inside should return true")
	}
	if item.ContainsPoint(50, 210) {
		t.Error("Point outside should return false")
	}
}

func TestListItem_NilSafety(t *testing.T) {
	var item *ListItem

	if item.HasChildren() {
		t.Error("HasChildren on nil should return false")
	}
	if item.ChildCount() != 0 {
		t.Error("ChildCount on nil should return 0")
	}
	if item.GetFullText() != "" {
		t.Error("GetFullText on nil should return empty string")
	}
	if item.IsCheckbox() {
		t.Error("IsCheckbox on nil should return false")
	}
	if item.IsChecked() {
		t.Error("IsChecked on nil should return false")
	}
	if item.WordCount() != 0 {
		t.Error("WordCount on nil should return 0")
	}
	if item.ContainsPoint(0, 0) {
		t.Error("ContainsPoint on nil should return false")
	}
	if item.IsFirstInList() {
		t.Error("IsFirstInList on nil should return false")
	}
}

func TestIsListItemText(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		{"• Item", true},
		{"- Item", true},
		{"* Item", true},
		{"1. Item", true},
		{"2) Item", true},
		{"a. Item", true},
		{"b) Item", true},
		{"→ Arrow item", true},
		{"Regular text", false},
		{"", false},
		{"   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := IsListItemText(tt.text)
			if got != tt.expected {
				t.Errorf("IsListItemText(%q) = %v, want %v", tt.text, got, tt.expected)
			}
		})
	}
}

func TestFindListsInRegion(t *testing.T) {
	layout := &ListLayout{
		Lists: []List{
			{BBox: model.BBox{X: 50, Y: 700, Width: 200, Height: 50}},
			{BBox: model.BBox{X: 50, Y: 500, Width: 200, Height: 50}},
			{BBox: model.BBox{X: 50, Y: 300, Width: 200, Height: 50}},
		},
	}

	// Region covering middle list
	region := model.BBox{X: 0, Y: 480, Width: 300, Height: 100}
	found := layout.FindListsInRegion(region)

	if len(found) != 1 {
		t.Errorf("Expected 1 list in region, got %d", len(found))
	}
}

// Benchmark tests
func BenchmarkListDetection(b *testing.B) {
	var paragraphs []Paragraph
	// Create a document with 10 lists of 5 items each
	for i := 0; i < 10; i++ {
		for j := 0; j < 5; j++ {
			paragraphs = append(paragraphs, Paragraph{
				Text:            "• List item content",
				BBox:            model.BBox{X: 72, Y: float64(700 - i*100 - j*20), Width: 200, Height: 14},
				LeftMargin:      72,
				AverageFontSize: 12,
			})
		}
		// Add non-list paragraph between lists
		paragraphs = append(paragraphs, Paragraph{
			Text:            "Regular paragraph text.",
			BBox:            model.BBox{X: 72, Y: float64(700 - i*100 - 120), Width: 300, Height: 14},
			LeftMargin:      72,
			AverageFontSize: 12,
		})
	}

	detector := NewListDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectFromParagraphs(paragraphs, 612, 792)
	}
}

func BenchmarkNestedListDetection(b *testing.B) {
	var paragraphs []Paragraph
	// Create nested list structure
	for i := 0; i < 20; i++ {
		indent := float64(72 + (i%3)*20)
		paragraphs = append(paragraphs, Paragraph{
			Text:            "• Nested item",
			BBox:            model.BBox{X: indent, Y: float64(700 - i*15), Width: 200, Height: 14},
			LeftMargin:      indent,
			AverageFontSize: 12,
		})
	}

	detector := NewListDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectFromParagraphs(paragraphs, 612, 792)
	}
}
