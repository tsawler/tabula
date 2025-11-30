package model

import (
	"math"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Point Tests
// ============================================================================

func TestPointDistance(t *testing.T) {
	tests := []struct {
		name     string
		p1, p2   Point
		expected float64
	}{
		{"same point", Point{0, 0}, Point{0, 0}, 0},
		{"horizontal", Point{0, 0}, Point{3, 0}, 3},
		{"vertical", Point{0, 0}, Point{0, 4}, 4},
		{"diagonal 3-4-5", Point{0, 0}, Point{3, 4}, 5},
		{"negative coords", Point{-1, -1}, Point{2, 3}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.p1.Distance(tt.p2)
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("Distance() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ============================================================================
// BBox Tests
// ============================================================================

func TestNewBBox(t *testing.T) {
	bbox := NewBBox(10, 20, 100, 50)
	if bbox.X != 10 || bbox.Y != 20 || bbox.Width != 100 || bbox.Height != 50 {
		t.Errorf("NewBBox() = %+v, want {10, 20, 100, 50}", bbox)
	}
}

func TestNewBBoxFromPoints(t *testing.T) {
	tests := []struct {
		name   string
		p1, p2 Point
		want   BBox
	}{
		{"normal", Point{10, 20}, Point{50, 70}, BBox{10, 20, 40, 50}},
		{"reversed", Point{50, 70}, Point{10, 20}, BBox{10, 20, 40, 50}},
		{"same point", Point{10, 10}, Point{10, 10}, BBox{10, 10, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewBBoxFromPoints(tt.p1, tt.p2)
			if got != tt.want {
				t.Errorf("NewBBoxFromPoints() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBBoxEdges(t *testing.T) {
	bbox := NewBBox(10, 20, 100, 50)

	if bbox.Left() != 10 {
		t.Errorf("Left() = %v, want 10", bbox.Left())
	}
	if bbox.Right() != 110 {
		t.Errorf("Right() = %v, want 110", bbox.Right())
	}
	if bbox.Bottom() != 20 {
		t.Errorf("Bottom() = %v, want 20", bbox.Bottom())
	}
	if bbox.Top() != 70 {
		t.Errorf("Top() = %v, want 70", bbox.Top())
	}
}

func TestBBoxCenter(t *testing.T) {
	bbox := NewBBox(0, 0, 100, 50)
	center := bbox.Center()

	if center.X != 50 || center.Y != 25 {
		t.Errorf("Center() = %+v, want {50, 25}", center)
	}
}

func TestBBoxContains(t *testing.T) {
	bbox := NewBBox(0, 0, 100, 100)

	tests := []struct {
		name     string
		point    Point
		expected bool
	}{
		{"inside", Point{50, 50}, true},
		{"on left edge", Point{0, 50}, true},
		{"on right edge", Point{100, 50}, true},
		{"outside left", Point{-1, 50}, false},
		{"outside right", Point{101, 50}, false},
		{"outside top", Point{50, 101}, false},
		{"outside bottom", Point{50, -1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bbox.Contains(tt.point)
			if result != tt.expected {
				t.Errorf("Contains(%+v) = %v, want %v", tt.point, result, tt.expected)
			}
		})
	}
}

func TestBBoxIntersects(t *testing.T) {
	bbox := NewBBox(0, 0, 100, 100)

	tests := []struct {
		name     string
		other    BBox
		expected bool
	}{
		{"overlapping", NewBBox(50, 50, 100, 100), true},
		{"touching edge", NewBBox(100, 0, 50, 50), true},
		{"inside", NewBBox(25, 25, 50, 50), true},
		{"containing", NewBBox(-10, -10, 200, 200), true},
		{"no overlap right", NewBBox(150, 0, 50, 50), false},
		{"no overlap left", NewBBox(-100, 0, 50, 50), false},
		{"no overlap above", NewBBox(0, 150, 50, 50), false},
		{"no overlap below", NewBBox(0, -100, 50, 50), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bbox.Intersects(tt.other)
			if result != tt.expected {
				t.Errorf("Intersects(%+v) = %v, want %v", tt.other, result, tt.expected)
			}
		})
	}
}

func TestBBoxIntersection(t *testing.T) {
	bbox := NewBBox(0, 0, 100, 100)

	t.Run("overlapping boxes", func(t *testing.T) {
		other := NewBBox(50, 50, 100, 100)
		result := bbox.Intersection(other)

		if result.X != 50 || result.Y != 50 || result.Width != 50 || result.Height != 50 {
			t.Errorf("Intersection() = %+v, want {50, 50, 50, 50}", result)
		}
	})

	t.Run("non-overlapping boxes", func(t *testing.T) {
		other := NewBBox(200, 200, 50, 50)
		result := bbox.Intersection(other)

		if result != (BBox{}) {
			t.Errorf("Intersection() = %+v, want empty BBox", result)
		}
	})
}

func TestBBoxUnion(t *testing.T) {
	bbox1 := NewBBox(0, 0, 50, 50)
	bbox2 := NewBBox(25, 25, 75, 75)

	result := bbox1.Union(bbox2)

	if result.X != 0 || result.Y != 0 || result.Width != 100 || result.Height != 100 {
		t.Errorf("Union() = %+v, want {0, 0, 100, 100}", result)
	}
}

func TestBBoxArea(t *testing.T) {
	bbox := NewBBox(0, 0, 10, 20)
	if bbox.Area() != 200 {
		t.Errorf("Area() = %v, want 200", bbox.Area())
	}
}

func TestBBoxExpand(t *testing.T) {
	bbox := NewBBox(10, 10, 50, 50)
	expanded := bbox.Expand(5)

	if expanded.X != 5 || expanded.Y != 5 || expanded.Width != 60 || expanded.Height != 60 {
		t.Errorf("Expand(5) = %+v, want {5, 5, 60, 60}", expanded)
	}
}

func TestBBoxOverlapRatio(t *testing.T) {
	bbox := NewBBox(0, 0, 100, 100)

	t.Run("complete overlap", func(t *testing.T) {
		other := NewBBox(0, 0, 100, 100)
		ratio := bbox.OverlapRatio(other)
		if ratio != 1.0 {
			t.Errorf("OverlapRatio() = %v, want 1.0", ratio)
		}
	})

	t.Run("half overlap", func(t *testing.T) {
		other := NewBBox(50, 0, 100, 100)
		ratio := bbox.OverlapRatio(other)
		if ratio != 0.5 {
			t.Errorf("OverlapRatio() = %v, want 0.5", ratio)
		}
	})

	t.Run("no overlap", func(t *testing.T) {
		other := NewBBox(200, 200, 50, 50)
		ratio := bbox.OverlapRatio(other)
		if ratio != 0 {
			t.Errorf("OverlapRatio() = %v, want 0", ratio)
		}
	})

	t.Run("zero area box", func(t *testing.T) {
		other := NewBBox(0, 0, 0, 0)
		ratio := bbox.OverlapRatio(other)
		if ratio != 0 {
			t.Errorf("OverlapRatio() = %v, want 0", ratio)
		}
	})
}

func TestBBoxIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		bbox     BBox
		expected bool
	}{
		{"valid box", NewBBox(0, 0, 10, 10), false},
		{"zero width", NewBBox(0, 0, 0, 10), true},
		{"zero height", NewBBox(0, 0, 10, 0), true},
		{"negative width", NewBBox(0, 0, -10, 10), true},
		{"negative height", NewBBox(0, 0, 10, -10), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.bbox.IsEmpty() != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", tt.bbox.IsEmpty(), tt.expected)
			}
		})
	}
}

func TestBBoxIsValid(t *testing.T) {
	tests := []struct {
		name     string
		bbox     BBox
		expected bool
	}{
		{"valid box", NewBBox(0, 0, 10, 10), true},
		{"zero width", NewBBox(0, 0, 0, 10), false},
		{"zero height", NewBBox(0, 0, 10, 0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.bbox.IsValid() != tt.expected {
				t.Errorf("IsValid() = %v, want %v", tt.bbox.IsValid(), tt.expected)
			}
		})
	}
}

// ============================================================================
// Matrix Tests
// ============================================================================

func TestIdentity(t *testing.T) {
	m := Identity()
	expected := Matrix{1, 0, 0, 1, 0, 0}
	if m != expected {
		t.Errorf("Identity() = %v, want %v", m, expected)
	}
}

func TestMatrixTransform(t *testing.T) {
	t.Run("identity", func(t *testing.T) {
		m := Identity()
		p := Point{10, 20}
		result := m.Transform(p)
		if result != p {
			t.Errorf("Identity.Transform(%v) = %v, want %v", p, result, p)
		}
	})

	t.Run("translation", func(t *testing.T) {
		m := Translate(100, 50)
		p := Point{10, 20}
		result := m.Transform(p)
		expected := Point{110, 70}
		if result != expected {
			t.Errorf("Translate.Transform(%v) = %v, want %v", p, result, expected)
		}
	})

	t.Run("scale", func(t *testing.T) {
		m := Scale(2, 3)
		p := Point{10, 20}
		result := m.Transform(p)
		expected := Point{20, 60}
		if result != expected {
			t.Errorf("Scale.Transform(%v) = %v, want %v", p, result, expected)
		}
	})
}

func TestMatrixMultiply(t *testing.T) {
	// Test matrix multiplication
	// The Multiply method computes m * other
	// So translate.Multiply(scale) means apply translate first, then scale
	translate := Translate(10, 20)
	scale := Scale(2, 2)
	combined := translate.Multiply(scale)

	p := Point{5, 5}
	result := combined.Transform(p)

	// With translate.Multiply(scale):
	// First translate (5+10, 5+20) = (15, 25), then scale (15*2, 25*2) = (30, 50)
	expected := Point{30, 50}
	if result != expected {
		t.Errorf("Combined transform(%v) = %v, want %v", p, result, expected)
	}
}

func TestTranslate(t *testing.T) {
	m := Translate(100, 200)
	expected := Matrix{1, 0, 0, 1, 100, 200}
	if m != expected {
		t.Errorf("Translate(100, 200) = %v, want %v", m, expected)
	}
}

func TestScale(t *testing.T) {
	m := Scale(2, 3)
	expected := Matrix{2, 0, 0, 3, 0, 0}
	if m != expected {
		t.Errorf("Scale(2, 3) = %v, want %v", m, expected)
	}
}

func TestRotate(t *testing.T) {
	// Rotate 90 degrees
	m := Rotate(math.Pi / 2)
	p := Point{1, 0}
	result := m.Transform(p)

	// After 90 degree rotation, (1,0) -> (0,1)
	if math.Abs(result.X) > 0.0001 || math.Abs(result.Y-1) > 0.0001 {
		t.Errorf("Rotate(Pi/2).Transform(1,0) = %v, want ~(0,1)", result)
	}
}

func TestMatrixIsIdentity(t *testing.T) {
	tests := []struct {
		name     string
		matrix   Matrix
		expected bool
	}{
		{"identity", Identity(), true},
		{"translated", Translate(1, 0), false},
		{"scaled", Scale(2, 1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.matrix.IsIdentity() != tt.expected {
				t.Errorf("IsIdentity() = %v, want %v", tt.matrix.IsIdentity(), tt.expected)
			}
		})
	}
}

// ============================================================================
// Document Tests
// ============================================================================

func TestNewDocument(t *testing.T) {
	doc := NewDocument()

	if doc == nil {
		t.Fatal("NewDocument() returned nil")
	}
	if doc.Metadata.Custom == nil {
		t.Error("Metadata.Custom not initialized")
	}
	if doc.Pages == nil {
		t.Error("Pages not initialized")
	}
	if len(doc.Pages) != 0 {
		t.Errorf("Pages should be empty, got %d", len(doc.Pages))
	}
}

func TestDocumentAddPage(t *testing.T) {
	doc := NewDocument()
	page1 := NewPage(612, 792)
	page2 := NewPage(612, 792)

	doc.AddPage(page1)
	doc.AddPage(page2)

	if len(doc.Pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(doc.Pages))
	}
	if page1.Number != 1 {
		t.Errorf("page1.Number = %d, want 1", page1.Number)
	}
	if page2.Number != 2 {
		t.Errorf("page2.Number = %d, want 2", page2.Number)
	}
}

func TestDocumentGetPage(t *testing.T) {
	doc := NewDocument()
	page := NewPage(612, 792)
	doc.AddPage(page)

	t.Run("valid page", func(t *testing.T) {
		p := doc.GetPage(1)
		if p != page {
			t.Error("GetPage(1) didn't return the correct page")
		}
	})

	t.Run("page 0", func(t *testing.T) {
		p := doc.GetPage(0)
		if p != nil {
			t.Error("GetPage(0) should return nil")
		}
	})

	t.Run("out of range", func(t *testing.T) {
		p := doc.GetPage(10)
		if p != nil {
			t.Error("GetPage(10) should return nil")
		}
	})
}

func TestDocumentPageCount(t *testing.T) {
	doc := NewDocument()
	if doc.PageCount() != 0 {
		t.Errorf("empty doc PageCount() = %d, want 0", doc.PageCount())
	}

	doc.AddPage(NewPage(612, 792))
	doc.AddPage(NewPage(612, 792))

	if doc.PageCount() != 2 {
		t.Errorf("PageCount() = %d, want 2", doc.PageCount())
	}
}

func TestDocumentExtractText(t *testing.T) {
	doc := NewDocument()
	page1 := NewPage(612, 792)
	page1.AddElement(&Paragraph{Text: "Page 1 text"})

	page2 := NewPage(612, 792)
	page2.AddElement(&Paragraph{Text: "Page 2 text"})

	doc.AddPage(page1)
	doc.AddPage(page2)

	text := doc.ExtractText()
	if !strings.Contains(text, "Page 1 text") {
		t.Error("ExtractText() missing page 1 content")
	}
	if !strings.Contains(text, "Page 2 text") {
		t.Error("ExtractText() missing page 2 content")
	}
}

func TestDocumentExtractTables(t *testing.T) {
	doc := NewDocument()
	page := NewPage(612, 792)

	table := NewTable(2, 2)
	page.AddElement(table)
	doc.AddPage(page)

	tables := doc.ExtractTables()
	if len(tables) != 1 {
		t.Errorf("ExtractTables() returned %d tables, want 1", len(tables))
	}
}

func TestDocumentHasLayout(t *testing.T) {
	doc := NewDocument()
	page := NewPage(612, 792)
	doc.AddPage(page)

	if doc.HasLayout() {
		t.Error("HasLayout() should be false without layout analysis")
	}

	page.Layout = &PageLayout{}
	if !doc.HasLayout() {
		t.Error("HasLayout() should be true with layout analysis")
	}
}

func TestDocumentAllHeadings(t *testing.T) {
	doc := NewDocument()
	page := NewPage(612, 792)
	page.Layout = &PageLayout{
		Headings: []HeadingInfo{
			{Level: 1, Text: "Heading 1"},
			{Level: 2, Text: "Heading 2"},
		},
	}
	doc.AddPage(page)

	headings := doc.AllHeadings()
	if len(headings) != 2 {
		t.Errorf("AllHeadings() returned %d, want 2", len(headings))
	}
}

func TestDocumentAllLists(t *testing.T) {
	doc := NewDocument()
	page := NewPage(612, 792)
	page.Layout = &PageLayout{
		Lists: []ListInfo{{Type: ListTypeBullet}},
	}
	doc.AddPage(page)

	lists := doc.AllLists()
	if len(lists) != 1 {
		t.Errorf("AllLists() returned %d, want 1", len(lists))
	}
}

func TestDocumentAllParagraphs(t *testing.T) {
	doc := NewDocument()
	page := NewPage(612, 792)
	page.Layout = &PageLayout{
		Paragraphs: []ParagraphInfo{{Text: "Para 1"}, {Text: "Para 2"}},
	}
	doc.AddPage(page)

	paragraphs := doc.AllParagraphs()
	if len(paragraphs) != 2 {
		t.Errorf("AllParagraphs() returned %d, want 2", len(paragraphs))
	}
}

func TestDocumentLayoutStats(t *testing.T) {
	doc := NewDocument()
	page := NewPage(612, 792)
	page.Layout = &PageLayout{
		Stats: LayoutStats{
			FragmentCount:  10,
			LineCount:      5,
			BlockCount:     3,
			ParagraphCount: 2,
			HeadingCount:   1,
			ListCount:      1,
		},
	}
	doc.AddPage(page)

	stats := doc.LayoutStats()
	if stats.FragmentCount != 10 {
		t.Errorf("LayoutStats().FragmentCount = %d, want 10", stats.FragmentCount)
	}
}

func TestDocumentTableOfContents(t *testing.T) {
	doc := NewDocument()
	page := NewPage(612, 792)
	page.Layout = &PageLayout{
		Headings: []HeadingInfo{
			{Level: 1, Text: "Chapter 1", FontSize: 24},
		},
	}
	doc.AddPage(page)

	toc := doc.TableOfContents()
	if len(toc) != 1 {
		t.Fatalf("TableOfContents() returned %d entries, want 1", len(toc))
	}
	if toc[0].Text != "Chapter 1" || toc[0].Level != 1 || toc[0].Page != 1 {
		t.Errorf("TOC entry = %+v, unexpected values", toc[0])
	}
}

// ============================================================================
// Page Tests
// ============================================================================

func TestNewPage(t *testing.T) {
	page := NewPage(612, 792)

	if page.Width != 612 || page.Height != 792 {
		t.Errorf("page dimensions = (%v, %v), want (612, 792)", page.Width, page.Height)
	}
	if page.Elements == nil {
		t.Error("Elements not initialized")
	}
	if page.RawText == nil {
		t.Error("RawText not initialized")
	}
	if page.RawLines == nil {
		t.Error("RawLines not initialized")
	}
}

func TestPageAddElement(t *testing.T) {
	page := NewPage(612, 792)
	para := &Paragraph{Text: "Test"}
	page.AddElement(para)

	if len(page.Elements) != 1 {
		t.Errorf("expected 1 element, got %d", len(page.Elements))
	}
}

func TestPageExtractText(t *testing.T) {
	page := NewPage(612, 792)
	page.AddElement(&Paragraph{Text: "Para 1"})
	page.AddElement(&Heading{Text: "Heading"})
	page.AddElement(&Image{}) // Non-text element

	text := page.ExtractText()
	if !strings.Contains(text, "Para 1") {
		t.Error("missing paragraph text")
	}
	if !strings.Contains(text, "Heading") {
		t.Error("missing heading text")
	}
}

func TestPageExtractTables(t *testing.T) {
	page := NewPage(612, 792)
	page.AddElement(&Paragraph{Text: "Text"})
	page.AddElement(NewTable(2, 2))
	page.AddElement(NewTable(3, 3))

	tables := page.ExtractTables()
	if len(tables) != 2 {
		t.Errorf("ExtractTables() returned %d, want 2", len(tables))
	}
}

func TestPageGetElementsInRegion(t *testing.T) {
	page := NewPage(612, 792)
	page.AddElement(&Paragraph{Text: "Inside", BBox: NewBBox(50, 50, 100, 100)})
	page.AddElement(&Paragraph{Text: "Outside", BBox: NewBBox(500, 500, 50, 50)})

	region := NewBBox(0, 0, 200, 200)
	elements := page.GetElementsInRegion(region)

	if len(elements) != 1 {
		t.Errorf("GetElementsInRegion() returned %d elements, want 1", len(elements))
	}
}

func TestPageHasLayout(t *testing.T) {
	page := NewPage(612, 792)

	if page.HasLayout() {
		t.Error("HasLayout() should be false initially")
	}

	page.Layout = &PageLayout{}
	if !page.HasLayout() {
		t.Error("HasLayout() should be true after setting Layout")
	}
}

func TestPageGetHeadings(t *testing.T) {
	page := NewPage(612, 792)

	// Without layout
	if page.GetHeadings() != nil {
		t.Error("GetHeadings() should return nil without layout")
	}

	// With layout
	page.Layout = &PageLayout{
		Headings: []HeadingInfo{{Level: 1, Text: "Test"}},
	}
	headings := page.GetHeadings()
	if len(headings) != 1 {
		t.Errorf("GetHeadings() returned %d, want 1", len(headings))
	}
}

func TestPageGetLists(t *testing.T) {
	page := NewPage(612, 792)

	if page.GetLists() != nil {
		t.Error("GetLists() should return nil without layout")
	}

	page.Layout = &PageLayout{
		Lists: []ListInfo{{Type: ListTypeBullet}},
	}
	if len(page.GetLists()) != 1 {
		t.Error("GetLists() should return 1 list")
	}
}

func TestPageGetParagraphs(t *testing.T) {
	page := NewPage(612, 792)

	if page.GetParagraphs() != nil {
		t.Error("GetParagraphs() should return nil without layout")
	}

	page.Layout = &PageLayout{
		Paragraphs: []ParagraphInfo{{Text: "Test"}},
	}
	if len(page.GetParagraphs()) != 1 {
		t.Error("GetParagraphs() should return 1 paragraph")
	}
}

func TestPageGetBlocks(t *testing.T) {
	page := NewPage(612, 792)

	if page.GetBlocks() != nil {
		t.Error("GetBlocks() should return nil without layout")
	}

	page.Layout = &PageLayout{
		TextBlocks: []BlockInfo{{Text: "Block"}},
	}
	if len(page.GetBlocks()) != 1 {
		t.Error("GetBlocks() should return 1 block")
	}
}

func TestPageColumnCount(t *testing.T) {
	page := NewPage(612, 792)

	if page.ColumnCount() != 0 {
		t.Error("ColumnCount() should be 0 without layout")
	}

	page.Layout = &PageLayout{ColumnCount: 2}
	if page.ColumnCount() != 2 {
		t.Error("ColumnCount() should be 2")
	}
}

func TestPageIsMultiColumn(t *testing.T) {
	page := NewPage(612, 792)
	page.Layout = &PageLayout{ColumnCount: 1}

	if page.IsMultiColumn() {
		t.Error("IsMultiColumn() should be false for 1 column")
	}

	page.Layout.ColumnCount = 2
	if !page.IsMultiColumn() {
		t.Error("IsMultiColumn() should be true for 2 columns")
	}
}

func TestPageContentBBox(t *testing.T) {
	page := NewPage(612, 792)

	// Without layout
	bbox := page.ContentBBox()
	if bbox.Width != 612 || bbox.Height != 792 {
		t.Errorf("ContentBBox() without layout = %+v, want full page", bbox)
	}

	// With header/footer
	page.Layout = &PageLayout{
		HasHeader:    true,
		HeaderHeight: 50,
		HasFooter:    true,
		FooterHeight: 30,
	}
	bbox = page.ContentBBox()
	if bbox.Y != 50 || bbox.Height != 712 {
		t.Errorf("ContentBBox() with header/footer = %+v, unexpected", bbox)
	}
}

func TestPageElementsInReadingOrder(t *testing.T) {
	page := NewPage(612, 792)
	elem1 := &Paragraph{Text: "First"}
	elem2 := &Paragraph{Text: "Second"}
	elem3 := &Paragraph{Text: "Third"}

	page.AddElement(elem1)
	page.AddElement(elem2)
	page.AddElement(elem3)

	// Without layout - original order
	elements := page.ElementsInReadingOrder()
	if len(elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(elements))
	}

	// With reading order
	page.Layout = &PageLayout{
		ReadingOrder: []int{2, 0, 1}, // Third, First, Second
	}
	elements = page.ElementsInReadingOrder()
	if elements[0].(*Paragraph).Text != "Third" {
		t.Error("reading order not respected")
	}
}

// ============================================================================
// Element Type Tests
// ============================================================================

func TestElementTypeString(t *testing.T) {
	tests := []struct {
		et       ElementType
		expected string
	}{
		{ElementTypeUnknown, "Unknown"},
		{ElementTypeParagraph, "Paragraph"},
		{ElementTypeHeading, "Heading"},
		{ElementTypeList, "List"},
		{ElementTypeTable, "Table"},
		{ElementTypeImage, "Image"},
		{ElementTypeFigure, "Figure"},
		{ElementTypeCaption, "Caption"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.et.String() != tt.expected {
				t.Errorf("String() = %v, want %v", tt.et.String(), tt.expected)
			}
		})
	}
}

func TestParagraphInterface(t *testing.T) {
	p := &Paragraph{
		Text:   "Test paragraph",
		BBox:   NewBBox(0, 0, 100, 50),
		ZOrder: 5,
	}

	if p.Type() != ElementTypeParagraph {
		t.Error("Type() should return ElementTypeParagraph")
	}
	if p.BoundingBox() != p.BBox {
		t.Error("BoundingBox() should return BBox")
	}
	if p.ZIndex() != 5 {
		t.Error("ZIndex() should return ZOrder")
	}
	if p.GetText() != "Test paragraph" {
		t.Error("GetText() should return Text")
	}
}

func TestHeadingInterface(t *testing.T) {
	h := &Heading{
		Text:   "Test heading",
		Level:  2,
		BBox:   NewBBox(0, 0, 100, 30),
		ZOrder: 3,
	}

	if h.Type() != ElementTypeHeading {
		t.Error("Type() should return ElementTypeHeading")
	}
	if h.GetText() != "Test heading" {
		t.Error("GetText() should return Text")
	}
}

func TestListInterface(t *testing.T) {
	l := &List{
		Items: []ListItem{
			{Text: "Item 1"},
			{Text: "Item 2"},
		},
		BBox:   NewBBox(0, 0, 100, 100),
		ZOrder: 2,
	}

	if l.Type() != ElementTypeList {
		t.Error("Type() should return ElementTypeList")
	}

	text := l.GetText()
	if !strings.Contains(text, "Item 1") || !strings.Contains(text, "Item 2") {
		t.Error("GetText() should contain all items")
	}
}

func TestImageInterface(t *testing.T) {
	img := &Image{
		Data:   []byte{1, 2, 3},
		Format: ImageFormatJPEG,
		BBox:   NewBBox(0, 0, 200, 150),
		ZOrder: 1,
	}

	if img.Type() != ElementTypeImage {
		t.Error("Type() should return ElementTypeImage")
	}
	if img.BoundingBox() != img.BBox {
		t.Error("BoundingBox() mismatch")
	}
	if img.ZIndex() != 1 {
		t.Error("ZIndex() should return ZOrder")
	}
}

// ============================================================================
// Alignment Tests
// ============================================================================

func TestAlignmentString(t *testing.T) {
	tests := []struct {
		a        Alignment
		expected string
	}{
		{AlignmentUnknown, "unknown"},
		{AlignmentLeft, "left"},
		{AlignmentCenter, "center"},
		{AlignmentRight, "right"},
		{AlignmentJustified, "justified"},
	}

	for _, tt := range tests {
		if tt.a.String() != tt.expected {
			t.Errorf("Alignment(%d).String() = %v, want %v", tt.a, tt.a.String(), tt.expected)
		}
	}
}

func TestListTypeString(t *testing.T) {
	tests := []struct {
		lt       ListType
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
		if tt.lt.String() != tt.expected {
			t.Errorf("ListType(%d).String() = %v, want %v", tt.lt, tt.lt.String(), tt.expected)
		}
	}
}

// ============================================================================
// Table Tests
// ============================================================================

func TestNewTable(t *testing.T) {
	table := NewTable(3, 4)

	if table.RowCount() != 3 {
		t.Errorf("RowCount() = %d, want 3", table.RowCount())
	}
	if table.ColCount() != 4 {
		t.Errorf("ColCount() = %d, want 4", table.ColCount())
	}
	if table.Confidence != 1.0 {
		t.Errorf("Confidence = %v, want 1.0", table.Confidence)
	}

	// Check default cell values
	cell := table.GetCell(0, 0)
	if cell.RowSpan != 1 || cell.ColSpan != 1 {
		t.Error("default cell should have RowSpan=1, ColSpan=1")
	}
}

func TestTableInterface(t *testing.T) {
	table := NewTable(2, 2)
	table.BBox = NewBBox(0, 0, 200, 100)
	table.ZOrder = 10

	if table.Type() != ElementTypeTable {
		t.Error("Type() should return ElementTypeTable")
	}
	if table.BoundingBox() != table.BBox {
		t.Error("BoundingBox() mismatch")
	}
	if table.ZIndex() != 10 {
		t.Error("ZIndex() should return ZOrder")
	}
}

func TestTableGetText(t *testing.T) {
	table := NewTable(2, 2)
	table.SetCell(0, 0, Cell{Text: "A1"})
	table.SetCell(0, 1, Cell{Text: "B1"})
	table.SetCell(1, 0, Cell{Text: "A2"})
	table.SetCell(1, 1, Cell{Text: "B2"})

	text := table.GetText()
	if !strings.Contains(text, "A1") || !strings.Contains(text, "B2") {
		t.Error("GetText() should contain all cell text")
	}
}

func TestTableRowColCount(t *testing.T) {
	t.Run("normal table", func(t *testing.T) {
		table := NewTable(3, 4)
		if table.RowCount() != 3 {
			t.Errorf("RowCount() = %d, want 3", table.RowCount())
		}
		if table.ColCount() != 4 {
			t.Errorf("ColCount() = %d, want 4", table.ColCount())
		}
	})

	t.Run("empty table", func(t *testing.T) {
		table := &Table{}
		if table.RowCount() != 0 {
			t.Errorf("empty table RowCount() = %d, want 0", table.RowCount())
		}
		if table.ColCount() != 0 {
			t.Errorf("empty table ColCount() = %d, want 0", table.ColCount())
		}
	})
}

func TestTableGetCell(t *testing.T) {
	table := NewTable(2, 2)
	table.SetCell(0, 0, Cell{Text: "Test"})

	t.Run("valid cell", func(t *testing.T) {
		cell := table.GetCell(0, 0)
		if cell == nil || cell.Text != "Test" {
			t.Error("GetCell(0,0) should return the cell")
		}
	})

	t.Run("out of bounds row", func(t *testing.T) {
		cell := table.GetCell(10, 0)
		if cell != nil {
			t.Error("GetCell(10,0) should return nil")
		}
	})

	t.Run("out of bounds col", func(t *testing.T) {
		cell := table.GetCell(0, 10)
		if cell != nil {
			t.Error("GetCell(0,10) should return nil")
		}
	})

	t.Run("negative indices", func(t *testing.T) {
		if table.GetCell(-1, 0) != nil {
			t.Error("negative row should return nil")
		}
		if table.GetCell(0, -1) != nil {
			t.Error("negative col should return nil")
		}
	})
}

func TestTableSetCell(t *testing.T) {
	table := NewTable(2, 2)

	t.Run("valid set", func(t *testing.T) {
		err := table.SetCell(0, 0, Cell{Text: "New"})
		if err != nil {
			t.Errorf("SetCell() error = %v", err)
		}
		if table.GetCell(0, 0).Text != "New" {
			t.Error("cell text not updated")
		}
	})

	t.Run("invalid row", func(t *testing.T) {
		err := table.SetCell(10, 0, Cell{})
		if err == nil {
			t.Error("SetCell() should return error for invalid row")
		}
	})

	t.Run("invalid col", func(t *testing.T) {
		err := table.SetCell(0, 10, Cell{})
		if err == nil {
			t.Error("SetCell() should return error for invalid col")
		}
	})
}

func TestTableToMarkdown(t *testing.T) {
	table := NewTable(3, 2)
	table.SetCell(0, 0, Cell{Text: "Header1"})
	table.SetCell(0, 1, Cell{Text: "Header2"})
	table.SetCell(1, 0, Cell{Text: "Data1"})
	table.SetCell(1, 1, Cell{Text: "Data2"})
	table.SetCell(2, 0, Cell{Text: "Data3"})
	table.SetCell(2, 1, Cell{Text: "Data4"})

	md := table.ToMarkdown()

	if !strings.Contains(md, "| Header1 |") {
		t.Error("markdown should contain header row")
	}
	if !strings.Contains(md, "|---|") {
		t.Error("markdown should contain separator")
	}
	if !strings.Contains(md, "| Data1 |") {
		t.Error("markdown should contain data rows")
	}
}

func TestTableToMarkdown_Empty(t *testing.T) {
	table := &Table{}
	md := table.ToMarkdown()
	if md != "" {
		t.Error("empty table should produce empty markdown")
	}
}

func TestTableToCSV(t *testing.T) {
	table := NewTable(2, 2)
	table.SetCell(0, 0, Cell{Text: "A1"})
	table.SetCell(0, 1, Cell{Text: "B1"})
	table.SetCell(1, 0, Cell{Text: "A2"})
	table.SetCell(1, 1, Cell{Text: "B2"})

	csv := table.ToCSV()

	if !strings.Contains(csv, "A1,B1") {
		t.Error("CSV should contain first row")
	}
	if !strings.Contains(csv, "A2,B2") {
		t.Error("CSV should contain second row")
	}
}

func TestTableToCSV_SpecialChars(t *testing.T) {
	table := NewTable(1, 2)
	table.SetCell(0, 0, Cell{Text: "Hello, World"}) // Contains comma
	table.SetCell(0, 1, Cell{Text: `Say "Hi"`})     // Contains quotes

	csv := table.ToCSV()

	if !strings.Contains(csv, `"Hello, World"`) {
		t.Error("CSV should quote cells with commas")
	}
	if !strings.Contains(csv, `"Say ""Hi"""`) {
		t.Error("CSV should escape quotes")
	}
}

// ============================================================================
// TableGrid Tests
// ============================================================================

func TestNewTableGrid(t *testing.T) {
	grid := NewTableGrid()

	if grid.Rows == nil || grid.Cols == nil {
		t.Error("NewTableGrid() should initialize slices")
	}
	if grid.RowCount() != 0 || grid.ColCount() != 0 {
		t.Error("new grid should have 0 rows and cols")
	}
}

func TestTableGridRowColCount(t *testing.T) {
	grid := NewTableGrid()
	grid.Rows = []float64{0, 50, 100}
	grid.Cols = []float64{0, 100, 200, 300}

	if grid.RowCount() != 2 {
		t.Errorf("RowCount() = %d, want 2", grid.RowCount())
	}
	if grid.ColCount() != 3 {
		t.Errorf("ColCount() = %d, want 3", grid.ColCount())
	}
}

func TestTableGridGetCellBBox(t *testing.T) {
	grid := NewTableGrid()
	grid.Rows = []float64{0, 50, 100}
	grid.Cols = []float64{0, 100, 200}

	t.Run("valid cell", func(t *testing.T) {
		bbox := grid.GetCellBBox(0, 0)
		if bbox.X != 0 || bbox.Y != 0 || bbox.Width != 100 || bbox.Height != 50 {
			t.Errorf("GetCellBBox(0,0) = %+v, unexpected", bbox)
		}
	})

	t.Run("out of bounds", func(t *testing.T) {
		bbox := grid.GetCellBBox(10, 10)
		if bbox != (BBox{}) {
			t.Error("out of bounds should return empty BBox")
		}
	})
}

// ============================================================================
// Metadata Tests
// ============================================================================

func TestMetadata(t *testing.T) {
	now := time.Now()
	meta := Metadata{
		Title:        "Test Document",
		Author:       "Test Author",
		Subject:      "Testing",
		Keywords:     []string{"test", "go"},
		Creator:      "Test Creator",
		Producer:     "Test Producer",
		CreationDate: now,
		ModDate:      now,
		Custom:       map[string]string{"key": "value"},
	}

	if meta.Title != "Test Document" {
		t.Error("Title not set correctly")
	}
	if len(meta.Keywords) != 2 {
		t.Error("Keywords not set correctly")
	}
	if meta.Custom["key"] != "value" {
		t.Error("Custom metadata not set correctly")
	}
}
