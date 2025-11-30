package model

// ElementType identifies the type of a page element.
type ElementType int

const (
	ElementTypeUnknown ElementType = iota
	ElementTypeParagraph
	ElementTypeHeading
	ElementTypeList
	ElementTypeTable
	ElementTypeImage
	ElementTypeFigure
	ElementTypeCaption
)

// String returns the name of the element type.
func (et ElementType) String() string {
	switch et {
	case ElementTypeParagraph:
		return "Paragraph"
	case ElementTypeHeading:
		return "Heading"
	case ElementTypeList:
		return "List"
	case ElementTypeTable:
		return "Table"
	case ElementTypeImage:
		return "Image"
	case ElementTypeFigure:
		return "Figure"
	case ElementTypeCaption:
		return "Caption"
	default:
		return "Unknown"
	}
}

// Element is the interface implemented by all page elements such as
// paragraphs, headings, lists, tables, and images.
type Element interface {
	Type() ElementType
	BoundingBox() BBox
	ZIndex() int
}

// TextElement is the interface for elements that contain text content.
type TextElement interface {
	Element
	GetText() string
}

// Paragraph represents a paragraph of text with position, style, and alignment.
type Paragraph struct {
	Text      string
	BBox      BBox
	FontSize  float64
	FontName  string
	Style     TextStyle
	Alignment TextAlignment
	ZOrder    int
}

// Type returns ElementTypeParagraph.
func (p *Paragraph) Type() ElementType { return ElementTypeParagraph }
func (p *Paragraph) BoundingBox() BBox { return p.BBox }
func (p *Paragraph) ZIndex() int       { return p.ZOrder }
func (p *Paragraph) GetText() string   { return p.Text }

// Heading represents a heading with level (1-6), position, and style information.
type Heading struct {
	Text     string
	Level    int // 1-6
	BBox     BBox
	FontSize float64
	FontName string
	Style    TextStyle
	ZOrder   int
}

// Type returns ElementTypeHeading.
func (h *Heading) Type() ElementType { return ElementTypeHeading }
func (h *Heading) BoundingBox() BBox { return h.BBox }
func (h *Heading) ZIndex() int       { return h.ZOrder }
func (h *Heading) GetText() string   { return h.Text }

// List represents an ordered or unordered list with items.
type List struct {
	Items   []ListItem
	Ordered bool
	BBox    BBox
	ZOrder  int
}

// Type returns ElementTypeList.
func (l *List) Type() ElementType { return ElementTypeList }
func (l *List) BoundingBox() BBox { return l.BBox }
func (l *List) ZIndex() int       { return l.ZOrder }
func (l *List) GetText() string {
	var text string
	for _, item := range l.Items {
		text += item.Text + "\n"
	}
	return text
}

// ListItem represents a single item within a list.
type ListItem struct {
	Text   string
	BBox   BBox
	Bullet string
	Level  int
}

// Image represents an embedded image with its binary data and format.
type Image struct {
	Data   []byte
	Format ImageFormat
	BBox   BBox
	DPI    float64
	ZOrder int
	// Alt text if available
	AltText string
}

// Type returns ElementTypeImage.
func (i *Image) Type() ElementType { return ElementTypeImage }
func (i *Image) BoundingBox() BBox { return i.BBox }
func (i *Image) ZIndex() int       { return i.ZOrder }

// ImageFormat identifies the format of an embedded image.
type ImageFormat int

const (
	ImageFormatUnknown ImageFormat = iota
	ImageFormatJPEG
	ImageFormatPNG
	ImageFormatTIFF
	ImageFormatJPEG2000
	ImageFormatJBIG2
)

// TextStyle represents text styling attributes.
type TextStyle struct {
	Bold      bool
	Italic    bool
	Underline bool
	Color     Color
}

// TextAlignment represents horizontal text alignment.
type TextAlignment int

const (
	AlignLeft TextAlignment = iota
	AlignCenter
	AlignRight
	AlignJustify
)

// Color represents an RGB color with 8-bit components.
type Color struct {
	R, G, B uint8
}

// TextFragment represents a positioned piece of text extracted from a PDF page,
// including its position, font, and transformation matrix.
type TextFragment struct {
	Text     string
	BBox     BBox
	FontSize float64
	FontName string
	Style    TextStyle
	Matrix   [6]float64 // Text transformation matrix
}

// Line represents a geometric line or rectangle from PDF graphics operations.
type Line struct {
	Start    Point
	End      Point
	Width    float64
	Color    Color
	IsRect   bool
	RectFill bool
}
