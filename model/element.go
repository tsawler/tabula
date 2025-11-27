package model

// ElementType represents the type of page element
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

// Element is the interface for all page elements
type Element interface {
	Type() ElementType
	BoundingBox() BBox
	ZIndex() int
}

// TextElement is an interface for elements containing text
type TextElement interface {
	Element
	GetText() string
}

// Paragraph represents a paragraph of text
type Paragraph struct {
	Text      string
	BBox      BBox
	FontSize  float64
	FontName  string
	Style     TextStyle
	Alignment TextAlignment
	ZOrder    int
}

func (p *Paragraph) Type() ElementType { return ElementTypeParagraph }
func (p *Paragraph) BoundingBox() BBox { return p.BBox }
func (p *Paragraph) ZIndex() int       { return p.ZOrder }
func (p *Paragraph) GetText() string   { return p.Text }

// Heading represents a heading
type Heading struct {
	Text     string
	Level    int // 1-6
	BBox     BBox
	FontSize float64
	FontName string
	Style    TextStyle
	ZOrder   int
}

func (h *Heading) Type() ElementType { return ElementTypeHeading }
func (h *Heading) BoundingBox() BBox { return h.BBox }
func (h *Heading) ZIndex() int       { return h.ZOrder }
func (h *Heading) GetText() string   { return h.Text }

// List represents a list (ordered or unordered)
type List struct {
	Items   []ListItem
	Ordered bool
	BBox    BBox
	ZOrder  int
}

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

// ListItem represents a single list item
type ListItem struct {
	Text   string
	BBox   BBox
	Bullet string
	Level  int
}

// Image represents an embedded image
type Image struct {
	Data   []byte
	Format ImageFormat
	BBox   BBox
	DPI    float64
	ZOrder int
	// Alt text if available
	AltText string
}

func (i *Image) Type() ElementType { return ElementTypeImage }
func (i *Image) BoundingBox() BBox { return i.BBox }
func (i *Image) ZIndex() int       { return i.ZOrder }

// ImageFormat represents image format
type ImageFormat int

const (
	ImageFormatUnknown ImageFormat = iota
	ImageFormatJPEG
	ImageFormatPNG
	ImageFormatTIFF
	ImageFormatJPEG2000
	ImageFormatJBIG2
)

// TextStyle represents text styling
type TextStyle struct {
	Bold      bool
	Italic    bool
	Underline bool
	Color     Color
}

// TextAlignment represents text alignment
type TextAlignment int

const (
	AlignLeft TextAlignment = iota
	AlignCenter
	AlignRight
	AlignJustify
)

// Color represents an RGB color
type Color struct {
	R, G, B uint8
}

// TextFragment represents a positioned piece of text
type TextFragment struct {
	Text     string
	BBox     BBox
	FontSize float64
	FontName string
	Style    TextStyle
	Matrix   [6]float64 // Text transformation matrix
}

// Line represents a geometric line or rectangle
type Line struct {
	Start    Point
	End      Point
	Width    float64
	Color    Color
	IsRect   bool
	RectFill bool
}
