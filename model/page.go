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

	// Layout analysis results (populated by AnalyzeLayout)
	Layout *PageLayout // Layout analysis results, nil if not analyzed
}

// PageLayout contains the results of layout analysis for a page
type PageLayout struct {
	// Column structure
	Columns     []ColumnInfo // Detected columns
	ColumnCount int          // Number of columns detected

	// Text structure
	TextBlocks []BlockInfo     // Detected text blocks
	Paragraphs []ParagraphInfo // Detected paragraphs
	Lines      []LineInfo      // Detected text lines

	// Semantic elements
	Headings []HeadingInfo // Detected headings (H1-H6)
	Lists    []ListInfo    // Detected lists

	// Reading order
	ReadingOrder []int // Indices into Elements in reading order

	// Header/footer detection
	HasHeader    bool    // Whether this page has a detected header
	HasFooter    bool    // Whether this page has a detected footer
	HeaderHeight float64 // Height of header region
	FooterHeight float64 // Height of footer region

	// Statistics
	Stats LayoutStats
}

// LayoutStats contains statistics about the layout analysis
type LayoutStats struct {
	FragmentCount  int // Number of text fragments processed
	LineCount      int // Number of text lines detected
	BlockCount     int // Number of text blocks detected
	ParagraphCount int // Number of paragraphs detected
	HeadingCount   int // Number of headings detected
	ListCount      int // Number of lists detected
}

// ColumnInfo contains information about a detected column
type ColumnInfo struct {
	Index int     // Column index (0-based, left to right)
	Left  float64 // Left edge X coordinate
	Right float64 // Right edge X coordinate
	Width float64 // Column width
	BBox  BBox    // Bounding box of column content
}

// BlockInfo contains information about a detected text block
type BlockInfo struct {
	Index     int       // Block index
	BBox      BBox      // Bounding box
	LineCount int       // Number of lines in block
	Text      string    // Combined text content
	Column    int       // Column index this block belongs to (-1 if unknown)
	FontSize  float64   // Average font size
	Alignment Alignment // Text alignment
}

// Alignment represents text alignment within a block
type Alignment int

const (
	AlignmentUnknown Alignment = iota
	AlignmentLeft
	AlignmentCenter
	AlignmentRight
	AlignmentJustified
)

func (a Alignment) String() string {
	switch a {
	case AlignmentLeft:
		return "left"
	case AlignmentCenter:
		return "center"
	case AlignmentRight:
		return "right"
	case AlignmentJustified:
		return "justified"
	default:
		return "unknown"
	}
}

// ParagraphInfo contains information about a detected paragraph
type ParagraphInfo struct {
	Index      int       // Paragraph index
	BBox       BBox      // Bounding box
	Text       string    // Text content
	FontSize   float64   // Average font size
	FontName   string    // Primary font name
	LineCount  int       // Number of lines
	Alignment  Alignment // Text alignment
	FirstLine  float64   // First line indent (positive = indented)
	LineHeight float64   // Average line height
}

// LineInfo contains information about a detected text line
type LineInfo struct {
	Index     int       // Line index
	BBox      BBox      // Bounding box
	Text      string    // Text content
	FontSize  float64   // Average font size
	Alignment Alignment // Detected alignment
	IsIndent  bool      // Whether line appears indented
}

// HeadingInfo contains information about a detected heading
type HeadingInfo struct {
	Level      int     // Heading level (1-6)
	Text       string  // Heading text
	BBox       BBox    // Bounding box
	FontSize   float64 // Font size
	FontName   string  // Font name
	Confidence float64 // Detection confidence (0-1)
}

// ListInfo contains information about a detected list
type ListInfo struct {
	Type       ListType   // Type of list
	Items      []ListItem // List items
	BBox       BBox       // Bounding box
	Nested     bool       // Whether list contains nested items
	StartValue int        // Starting value for numbered lists
}

// ListType represents the type of list
type ListType int

const (
	ListTypeUnknown  ListType = iota
	ListTypeBullet            // Bullet points (•, -, *, etc.)
	ListTypeNumbered          // Numbered (1, 2, 3)
	ListTypeLettered          // Lettered (a, b, c or A, B, C)
	ListTypeRoman             // Roman numerals (i, ii, iii or I, II, III)
	ListTypeCheckbox          // Checkboxes (☐, ☑, ✓)
)

func (lt ListType) String() string {
	switch lt {
	case ListTypeBullet:
		return "bullet"
	case ListTypeNumbered:
		return "numbered"
	case ListTypeLettered:
		return "lettered"
	case ListTypeRoman:
		return "roman"
	case ListTypeCheckbox:
		return "checkbox"
	default:
		return "unknown"
	}
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

// HasLayout returns true if layout analysis has been performed on this page
func (p *Page) HasLayout() bool {
	return p.Layout != nil
}

// GetHeadings returns all headings on this page (requires layout analysis)
func (p *Page) GetHeadings() []HeadingInfo {
	if p.Layout == nil {
		return nil
	}
	return p.Layout.Headings
}

// GetLists returns all lists on this page (requires layout analysis)
func (p *Page) GetLists() []ListInfo {
	if p.Layout == nil {
		return nil
	}
	return p.Layout.Lists
}

// GetParagraphs returns all paragraphs on this page (requires layout analysis)
func (p *Page) GetParagraphs() []ParagraphInfo {
	if p.Layout == nil {
		return nil
	}
	return p.Layout.Paragraphs
}

// GetBlocks returns all text blocks on this page (requires layout analysis)
func (p *Page) GetBlocks() []BlockInfo {
	if p.Layout == nil {
		return nil
	}
	return p.Layout.TextBlocks
}

// ColumnCount returns the number of columns detected on this page
func (p *Page) ColumnCount() int {
	if p.Layout == nil {
		return 0
	}
	return p.Layout.ColumnCount
}

// IsMultiColumn returns true if the page has multiple columns
func (p *Page) IsMultiColumn() bool {
	return p.ColumnCount() > 1
}

// ContentBBox returns the bounding box of all content on the page,
// excluding headers and footers if detected
func (p *Page) ContentBBox() BBox {
	if p.Layout == nil {
		// No layout analysis, return full page
		return BBox{X: 0, Y: 0, Width: p.Width, Height: p.Height}
	}

	top := 0.0
	bottom := p.Height

	if p.Layout.HasHeader && p.Layout.HeaderHeight > 0 {
		top = p.Layout.HeaderHeight
	}
	if p.Layout.HasFooter && p.Layout.FooterHeight > 0 {
		bottom = p.Height - p.Layout.FooterHeight
	}

	return BBox{X: 0, Y: top, Width: p.Width, Height: bottom - top}
}

// ElementsInReadingOrder returns elements sorted by reading order
// If layout analysis hasn't been performed, returns elements in original order
func (p *Page) ElementsInReadingOrder() []Element {
	if p.Layout == nil || len(p.Layout.ReadingOrder) == 0 {
		return p.Elements
	}

	result := make([]Element, 0, len(p.Layout.ReadingOrder))
	for _, idx := range p.Layout.ReadingOrder {
		if idx >= 0 && idx < len(p.Elements) {
			result = append(result, p.Elements[idx])
		}
	}
	return result
}
