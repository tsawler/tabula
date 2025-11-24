package model

// Page represents a single page in a PDF document
type Page struct {
	Number   int       // 1-indexed page number
	Width    float64   // Page width in points
	Height   float64   // Page height in points
	Rotation int       // Rotation angle (0, 90, 180, 270)
	Elements []Element // Ordered list of page elements

	// Raw data for debugging/advanced use
	RawText  []TextFragment // All text fragments with positions
	RawLines []Line         // All detected lines/rectangles
}

// NewPage creates a new page with given dimensions
func NewPage(width, height float64) *Page {
	return &Page{
		Width:    width,
		Height:   height,
		Elements: make([]Element, 0),
		RawText:  make([]TextFragment, 0),
		RawLines: make([]Line, 0),
	}
}

// AddElement adds an element to the page
func (p *Page) AddElement(elem Element) {
	p.Elements = append(p.Elements, elem)
}

// ExtractText concatenates all text elements
func (p *Page) ExtractText() string {
	var text string
	for _, elem := range p.Elements {
		if te, ok := elem.(TextElement); ok {
			text += te.GetText() + "\n"
		}
	}
	return text
}

// ExtractTables returns all table elements on the page
func (p *Page) ExtractTables() []*Table {
	var tables []*Table
	for _, elem := range p.Elements {
		if table, ok := elem.(*Table); ok {
			tables = append(tables, table)
		}
	}
	return tables
}

// GetElementsInRegion returns elements within a bounding box
func (p *Page) GetElementsInRegion(bbox BBox) []Element {
	var elements []Element
	for _, elem := range p.Elements {
		if bbox.Intersects(elem.BoundingBox()) {
			elements = append(elements, elem)
		}
	}
	return elements
}
