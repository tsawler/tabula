package docx

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestDOCXWithList creates a DOCX file with list content.
func createTestDOCXWithList(t *testing.T, documentXML, numberingXML string) string {
	t.Helper()

	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "test.docx")

	f, err := os.Create(docxPath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	zw := zip.NewWriter(f)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>
</Types>`
	w, _ := zw.Create("[Content_Types].xml")
	w.Write([]byte(contentTypes))

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
	w, _ = zw.Create("_rels/.rels")
	w.Write([]byte(rels))

	// word/document.xml
	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>` + documentXML + `</w:body>
</w:document>`
	w, _ = zw.Create("word/document.xml")
	w.Write([]byte(document))

	// word/numbering.xml (if provided)
	if numberingXML != "" {
		numbering := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` + numberingXML + `</w:numbering>`
		w, _ = zw.Create("word/numbering.xml")
		w.Write([]byte(numbering))
	}

	zw.Close()
	f.Close()

	return docxPath
}

func TestListParsing_BulletList(t *testing.T) {
	// Document with bullet list items (attributes don't have namespace prefix)
	documentXML := `
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="0"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>First item</w:t></w:r>
</w:p>
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="0"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>Second item</w:t></w:r>
</w:p>
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="0"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>Third item</w:t></w:r>
</w:p>`

	// Numbering definition for bullet list (attributes don't have namespace prefix)
	numberingXML := `
<w:abstractNum abstractNumId="0">
  <w:lvl ilvl="0">
    <w:start val="1"/>
    <w:numFmt val="bullet"/>
    <w:lvlText val="•"/>
  </w:lvl>
</w:abstractNum>
<w:num numId="1">
  <w:abstractNumId val="0"/>
</w:num>`

	docxPath := createTestDOCXWithList(t, documentXML, numberingXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	// Check that list items are detected
	lists := r.Lists()
	if len(lists) != 1 {
		t.Fatalf("expected 1 list, got %d", len(lists))
	}

	list := lists[0]
	if len(list.Items) != 3 {
		t.Errorf("expected 3 list items, got %d", len(list.Items))
	}

	if list.Type != ListTypeUnordered {
		t.Errorf("expected unordered list, got ordered")
	}

	// Check text extraction
	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}

	if !strings.Contains(text, "First item") {
		t.Error("Text() should contain 'First item'")
	}
	if !strings.Contains(text, "•") {
		t.Error("Text() should contain bullet character '•'")
	}
}

func TestListParsing_NumberedList(t *testing.T) {
	// Document with numbered list items (attributes don't have namespace prefix)
	documentXML := `
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="0"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>Step one</w:t></w:r>
</w:p>
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="0"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>Step two</w:t></w:r>
</w:p>
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="0"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>Step three</w:t></w:r>
</w:p>`

	// Numbering definition for decimal list (attributes don't have namespace prefix)
	numberingXML := `
<w:abstractNum abstractNumId="0">
  <w:lvl ilvl="0">
    <w:start val="1"/>
    <w:numFmt val="decimal"/>
    <w:lvlText val="%1."/>
  </w:lvl>
</w:abstractNum>
<w:num numId="1">
  <w:abstractNumId val="0"/>
</w:num>`

	docxPath := createTestDOCXWithList(t, documentXML, numberingXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	lists := r.Lists()
	if len(lists) != 1 {
		t.Fatalf("expected 1 list, got %d", len(lists))
	}

	list := lists[0]
	if list.Type != ListTypeOrdered {
		t.Errorf("expected ordered list, got unordered")
	}

	// Check text extraction includes numbers
	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}

	if !strings.Contains(text, "1. Step one") {
		t.Errorf("Text() should contain '1. Step one', got: %s", text)
	}
	if !strings.Contains(text, "2. Step two") {
		t.Errorf("Text() should contain '2. Step two', got: %s", text)
	}
}

func TestListParsing_NestedList(t *testing.T) {
	// Document with nested list items (attributes don't have namespace prefix)
	documentXML := `
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="0"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>Level 0 item</w:t></w:r>
</w:p>
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="1"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>Level 1 item</w:t></w:r>
</w:p>
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="2"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>Level 2 item</w:t></w:r>
</w:p>`

	// Numbering definition for bullet list with multiple levels
	numberingXML := `
<w:abstractNum abstractNumId="0">
  <w:lvl ilvl="0">
    <w:start val="1"/>
    <w:numFmt val="bullet"/>
    <w:lvlText val="•"/>
  </w:lvl>
  <w:lvl ilvl="1">
    <w:start val="1"/>
    <w:numFmt val="bullet"/>
    <w:lvlText val="○"/>
  </w:lvl>
  <w:lvl ilvl="2">
    <w:start val="1"/>
    <w:numFmt val="bullet"/>
    <w:lvlText val="■"/>
  </w:lvl>
</w:abstractNum>
<w:num numId="1">
  <w:abstractNumId val="0"/>
</w:num>`

	docxPath := createTestDOCXWithList(t, documentXML, numberingXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	lists := r.Lists()
	if len(lists) != 1 {
		t.Fatalf("expected 1 list, got %d", len(lists))
	}

	// Check different levels
	list := lists[0]
	if len(list.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(list.Items))
	}

	if list.Items[0].Level != 0 {
		t.Errorf("item 0 level = %d, want 0", list.Items[0].Level)
	}
	if list.Items[1].Level != 1 {
		t.Errorf("item 1 level = %d, want 1", list.Items[1].Level)
	}
	if list.Items[2].Level != 2 {
		t.Errorf("item 2 level = %d, want 2", list.Items[2].Level)
	}

	// Check indentation in text output
	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}

	// Level 1 should have 2 spaces indentation
	if !strings.Contains(text, "  ") {
		t.Error("Text() should contain indentation for nested items")
	}
}

func TestListParsing_MixedContent(t *testing.T) {
	// Document with paragraphs, list, and more paragraphs (attributes don't have namespace prefix)
	documentXML := `
<w:p><w:r><w:t>Introduction paragraph</w:t></w:r></w:p>
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="0"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>List item 1</w:t></w:r>
</w:p>
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="0"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>List item 2</w:t></w:r>
</w:p>
<w:p><w:r><w:t>Conclusion paragraph</w:t></w:r></w:p>`

	numberingXML := `
<w:abstractNum abstractNumId="0">
  <w:lvl ilvl="0">
    <w:start val="1"/>
    <w:numFmt val="bullet"/>
    <w:lvlText val="•"/>
  </w:lvl>
</w:abstractNum>
<w:num numId="1">
  <w:abstractNumId val="0"/>
</w:num>`

	docxPath := createTestDOCXWithList(t, documentXML, numberingXML)

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}

	// Check order is preserved
	introIdx := strings.Index(text, "Introduction")
	item1Idx := strings.Index(text, "List item 1")
	item2Idx := strings.Index(text, "List item 2")
	conclusionIdx := strings.Index(text, "Conclusion")

	if introIdx >= item1Idx || item1Idx >= item2Idx || item2Idx >= conclusionIdx {
		t.Error("Document order not preserved in text output")
	}
}

func TestListParsing_NoNumbering(t *testing.T) {
	// Document with list-like paragraphs but no numbering.xml (attributes don't have namespace prefix)
	documentXML := `
<w:p>
  <w:pPr>
    <w:numPr>
      <w:ilvl val="0"/>
      <w:numId val="1"/>
    </w:numPr>
  </w:pPr>
  <w:r><w:t>Item without numbering def</w:t></w:r>
</w:p>`

	// No numbering.xml
	docxPath := createTestDOCXWithList(t, documentXML, "")

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer r.Close()

	// Should still detect as list item with default bullet
	text, err := r.Text()
	if err != nil {
		t.Fatalf("Text() error = %v", err)
	}

	// Should have bullet (default when no numbering def)
	if !strings.Contains(text, "•") {
		t.Error("Text() should contain default bullet when no numbering.xml")
	}
}

func TestNumberingResolver_ResolveLevel(t *testing.T) {
	tests := []struct {
		name      string
		numID     string
		level     int
		wantType  ListType
		wantStart int
	}{
		{"empty numID", "", 0, ListTypeUnordered, 1},
		{"unknown numID", "999", 0, ListTypeUnordered, 1},
	}

	resolver := NewNumberingResolver(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listType, _, startAt := resolver.ResolveLevel(tt.numID, tt.level)
			if listType != tt.wantType {
				t.Errorf("listType = %v, want %v", listType, tt.wantType)
			}
			if startAt != tt.wantStart {
				t.Errorf("startAt = %d, want %d", startAt, tt.wantStart)
			}
		})
	}
}

func TestListParser_ExtractLists(t *testing.T) {
	resolver := NewNumberingResolver(nil)
	parser := NewListParser(resolver)

	paragraphs := []parsedParagraph{
		{Text: "Normal paragraph", IsListItem: false},
		{Text: "Item 1", IsListItem: true, NumID: "1", ListLevel: 0},
		{Text: "Item 2", IsListItem: true, NumID: "1", ListLevel: 0},
		{Text: "Another paragraph", IsListItem: false},
		{Text: "Item A", IsListItem: true, NumID: "2", ListLevel: 0},
	}

	lists := parser.ExtractLists(paragraphs)

	if len(lists) != 2 {
		t.Fatalf("expected 2 lists, got %d", len(lists))
	}

	if len(lists[0].Items) != 2 {
		t.Errorf("first list should have 2 items, got %d", len(lists[0].Items))
	}
	if len(lists[1].Items) != 1 {
		t.Errorf("second list should have 1 item, got %d", len(lists[1].Items))
	}
}

func TestModelList_Conversion(t *testing.T) {
	list := ParsedList{
		Type:    ListTypeOrdered,
		NumID:   "1",
		StartAt: 1,
		Items: []ParsedListItem{
			{Text: "First", Level: 0, Bullet: ""},
			{Text: "Second", Level: 0, Bullet: ""},
		},
	}

	modelList := list.ToModelList()

	if !modelList.Ordered {
		t.Error("model list should be ordered")
	}
	if len(modelList.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(modelList.Items))
	}
	if modelList.Items[0].Bullet != "1." {
		t.Errorf("first item bullet = %q, want '1.'", modelList.Items[0].Bullet)
	}
	if modelList.Items[1].Bullet != "2." {
		t.Errorf("second item bullet = %q, want '2.'", modelList.Items[1].Bullet)
	}
}

func TestToText_BulletList(t *testing.T) {
	list := ParsedList{
		Type:    ListTypeUnordered,
		NumID:   "1",
		StartAt: 1,
		Items: []ParsedListItem{
			{Text: "Item one", Level: 0, Bullet: "•"},
			{Text: "Item two", Level: 0, Bullet: "•"},
		},
	}

	text := list.ToText()

	if !strings.Contains(text, "• Item one") {
		t.Errorf("ToText() should contain '• Item one', got: %s", text)
	}
	if !strings.Contains(text, "• Item two") {
		t.Errorf("ToText() should contain '• Item two', got: %s", text)
	}
}

func TestToText_NumberedList(t *testing.T) {
	list := ParsedList{
		Type:    ListTypeOrdered,
		NumID:   "1",
		StartAt: 1,
		Items: []ParsedListItem{
			{Text: "First", Level: 0, Bullet: ""},
			{Text: "Second", Level: 0, Bullet: ""},
		},
	}

	text := list.ToText()

	if !strings.Contains(text, "1. First") {
		t.Errorf("ToText() should contain '1. First', got: %s", text)
	}
	if !strings.Contains(text, "2. Second") {
		t.Errorf("ToText() should contain '2. Second', got: %s", text)
	}
}

func TestRomanNumerals(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{1, "I"},
		{4, "IV"},
		{5, "V"},
		{9, "IX"},
		{10, "X"},
		{40, "XL"},
		{50, "L"},
		{90, "XC"},
		{100, "C"},
		{400, "CD"},
		{500, "D"},
		{900, "CM"},
		{1000, "M"},
		{1994, "MCMXCIV"},
		{2024, "MMXXIV"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := toUpperRoman(tt.input)
			if got != tt.want {
				t.Errorf("toUpperRoman(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLetterConversion(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{1, "a"},
		{2, "b"},
		{26, "z"},
		{27, "aa"},
		{28, "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := toLowerLetter(tt.input)
			if got != tt.want {
				t.Errorf("toLowerLetter(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
