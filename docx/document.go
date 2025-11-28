package docx

import "encoding/xml"

// XML namespaces used in DOCX files
const (
	nsW  = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	nsR  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	nsWP = "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
	nsDC = "http://purl.org/dc/elements/1.1/"
	nsCP = "http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
)

// documentXML represents the structure of word/document.xml
type documentXML struct {
	XMLName xml.Name `xml:"document"`
	Body    *bodyXML `xml:"body"`
}

// bodyXML represents the document body.
// Note: Paragraphs and Tables are collected separately by xml.Unmarshal.
// Use Elements for ordered access (populated by custom parsing).
type bodyXML struct {
	Paragraphs []paragraphXML `xml:"p"`
	Tables     []tableXML     `xml:"tbl"`
	Elements   []bodyElement  `xml:"-"` // Populated manually to preserve order
}

// bodyElement represents an element in the document body (paragraph or table).
type bodyElement struct {
	Type      string // "paragraph" or "table"
	Paragraph *paragraphXML
	Table     *tableXML
}

// paragraphXML represents a paragraph element (<w:p>).
type paragraphXML struct {
	XMLName       xml.Name          `xml:"p"`
	Properties    paragraphPropsXML `xml:"pPr"`
	Runs          []runXML          `xml:"r"`
	Hyperlinks    []hyperlinkXML    `xml:"hyperlink"`
	BookmarkStart []bookmarkXML     `xml:"bookmarkStart"`
}

// paragraphPropsXML represents paragraph properties (<w:pPr>).
type paragraphPropsXML struct {
	Style         styleRefXML       `xml:"pStyle"`
	NumPr         numberingPropsXML `xml:"numPr"`
	Justification justificationXML  `xml:"jc"`
	Spacing       spacingXML        `xml:"spacing"`
	Indent        indentXML         `xml:"ind"`
	OutlineLvl    outlineLvlXML     `xml:"outlineLvl"`
}

// styleRefXML represents a style reference.
type styleRefXML struct {
	Val string `xml:"val,attr"`
}

// numberingPropsXML represents numbering properties for lists.
type numberingPropsXML struct {
	ILvl  ilvlXML  `xml:"ilvl"`
	NumID numIDXML `xml:"numId"`
}

// ilvlXML represents indentation level.
type ilvlXML struct {
	Val string `xml:"val,attr"`
}

// numIDXML represents numbering ID.
type numIDXML struct {
	Val string `xml:"val,attr"`
}

// justificationXML represents text justification.
type justificationXML struct {
	Val string `xml:"val,attr"` // left, center, right, both
}

// spacingXML represents paragraph spacing.
type spacingXML struct {
	Before string `xml:"before,attr"` // Space before in twips
	After  string `xml:"after,attr"`  // Space after in twips
	Line   string `xml:"line,attr"`   // Line spacing
}

// indentXML represents paragraph indentation.
type indentXML struct {
	Left      string `xml:"left,attr"`
	Right     string `xml:"right,attr"`
	FirstLine string `xml:"firstLine,attr"`
	Hanging   string `xml:"hanging,attr"`
}

// outlineLvlXML represents outline level.
type outlineLvlXML struct {
	Val string `xml:"val,attr"`
}

// runXML represents a text run (<w:r>).
type runXML struct {
	XMLName          xml.Name              `xml:"r"`
	Properties       runPropsXML           `xml:"rPr"`
	Text             []textXML             `xml:"t"`
	Tabs             []tabXML              `xml:"tab"`
	Breaks           []breakXML            `xml:"br"`
	Drawing          []drawingXML          `xml:"drawing"`
	Symbols          []symXML              `xml:"sym"`
	AlternateContent []alternateContentXML `xml:"AlternateContent"`
}

// symXML represents a symbol character (<w:sym>).
type symXML struct {
	Font string `xml:"font,attr"` // Font name (e.g., "Segoe UI Emoji")
	Char string `xml:"char,attr"` // Hex character code
}

// alternateContentXML represents mc:AlternateContent for emoji fallbacks.
type alternateContentXML struct {
	Fallback fallbackXML `xml:"Fallback"`
}

// fallbackXML represents mc:Fallback containing text.
type fallbackXML struct {
	Text []textXML `xml:"t"`
}

// runPropsXML represents run properties (<w:rPr>).
type runPropsXML struct {
	Bold      boolXML      `xml:"b"`
	Italic    boolXML      `xml:"i"`
	Underline underlineXML `xml:"u"`
	Strike    boolXML      `xml:"strike"`
	FontSize  sizeXML      `xml:"sz"`
	Font      fontXML      `xml:"rFonts"`
	Color     colorXML     `xml:"color"`
	Highlight highlightXML `xml:"highlight"`
}

// boolXML represents a boolean attribute.
type boolXML struct {
	XMLName xml.Name
	Val     string `xml:"val,attr"`
}

// underlineXML represents underline style.
type underlineXML struct {
	Val string `xml:"val,attr"` // single, double, etc.
}

// sizeXML represents font size (in half-points).
type sizeXML struct {
	Val string `xml:"val,attr"`
}

// fontXML represents font settings.
type fontXML struct {
	ASCII    string `xml:"ascii,attr"`
	HAnsi    string `xml:"hAnsi,attr"`
	CS       string `xml:"cs,attr"`
	EastAsia string `xml:"eastAsia,attr"`
}

// colorXML represents text color.
type colorXML struct {
	Val string `xml:"val,attr"` // Hex color or "auto"
}

// highlightXML represents highlight color.
type highlightXML struct {
	Val string `xml:"val,attr"` // Color name like "yellow"
}

// textXML represents text content (<w:t>).
type textXML struct {
	XMLName xml.Name `xml:"t"`
	Space   string   `xml:"space,attr"` // preserve
	Value   string   `xml:",chardata"`
}

// tabXML represents a tab character.
type tabXML struct {
	XMLName xml.Name `xml:"tab"`
}

// breakXML represents a break (line or page).
type breakXML struct {
	XMLName xml.Name `xml:"br"`
	Type    string   `xml:"type,attr"` // page, column, textWrapping
}

// drawingXML represents an embedded drawing/image.
type drawingXML struct {
	XMLName xml.Name   `xml:"drawing"`
	Inline  *inlineXML `xml:"inline"`
	Anchor  *anchorXML `xml:"anchor"`
}

// inlineXML represents an inline image.
type inlineXML struct {
	Extent extentXML `xml:"extent"`
	DocPr  docPrXML  `xml:"docPr"`
	Blip   *blipXML  `xml:"graphic>graphicData>pic>blipFill>blip"`
}

// anchorXML represents an anchored image.
type anchorXML struct {
	Extent extentXML `xml:"extent"`
	DocPr  docPrXML  `xml:"docPr"`
	Blip   *blipXML  `xml:"graphic>graphicData>pic>blipFill>blip"`
}

// extentXML represents image dimensions.
type extentXML struct {
	CX string `xml:"cx,attr"` // Width in EMUs
	CY string `xml:"cy,attr"` // Height in EMUs
}

// docPrXML represents document properties of an image.
type docPrXML struct {
	ID    string `xml:"id,attr"`
	Name  string `xml:"name,attr"`
	Descr string `xml:"descr,attr"` // Alt text
}

// blipXML represents an image reference.
type blipXML struct {
	Embed string `xml:"embed,attr"` // Relationship ID
}

// hyperlinkXML represents a hyperlink.
type hyperlinkXML struct {
	ID   string   `xml:"id,attr"`
	Runs []runXML `xml:"r"`
}

// bookmarkXML represents a bookmark.
type bookmarkXML struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// tableXML represents a table (<w:tbl>).
type tableXML struct {
	XMLName    xml.Name      `xml:"tbl"`
	Properties tablePropsXML `xml:"tblPr"`
	Grid       tableGridXML  `xml:"tblGrid"`
	Rows       []tableRowXML `xml:"tr"`
}

// tablePropsXML represents table properties.
type tablePropsXML struct {
	Style   styleRefXML     `xml:"tblStyle"`
	Width   tableSizeXML    `xml:"tblW"`
	Borders tableBordersXML `xml:"tblBorders"`
}

// tableSizeXML represents table/cell size.
type tableSizeXML struct {
	W    string `xml:"w,attr"`    // Width value
	Type string `xml:"type,attr"` // dxa (twips), pct, auto
}

// tableBordersXML represents table borders.
type tableBordersXML struct {
	Top     borderXML `xml:"top"`
	Bottom  borderXML `xml:"bottom"`
	Left    borderXML `xml:"left"`
	Right   borderXML `xml:"right"`
	InsideH borderXML `xml:"insideH"`
	InsideV borderXML `xml:"insideV"`
}

// borderXML represents a single border.
type borderXML struct {
	Val   string `xml:"val,attr"`   // Border style: single, double, etc.
	Sz    string `xml:"sz,attr"`    // Size in eighths of a point
	Space string `xml:"space,attr"` // Space from text
	Color string `xml:"color,attr"` // Color
}

// tableGridXML represents table grid definition.
type tableGridXML struct {
	Cols []gridColXML `xml:"gridCol"`
}

// gridColXML represents a grid column.
type gridColXML struct {
	W string `xml:"w,attr"` // Width in twips
}

// tableRowXML represents a table row (<w:tr>).
type tableRowXML struct {
	XMLName    xml.Name       `xml:"tr"`
	Properties rowPropsXML    `xml:"trPr"`
	Cells      []tableCellXML `xml:"tc"`
}

// rowPropsXML represents row properties.
type rowPropsXML struct {
	Height rowHeightXML `xml:"trHeight"`
	Header boolXML      `xml:"tblHeader"` // Is this a header row?
}

// rowHeightXML represents row height.
type rowHeightXML struct {
	Val  string `xml:"val,attr"`
	Rule string `xml:"hRule,attr"` // exact, atLeast, auto
}

// tableCellXML represents a table cell (<w:tc>).
type tableCellXML struct {
	XMLName    xml.Name       `xml:"tc"`
	Properties cellPropsXML   `xml:"tcPr"`
	Paragraphs []paragraphXML `xml:"p"`
}

// cellPropsXML represents cell properties.
type cellPropsXML struct {
	Width    tableSizeXML    `xml:"tcW"`
	GridSpan gridSpanXML     `xml:"gridSpan"`
	VMerge   vMergeXML       `xml:"vMerge"`
	Borders  tableBordersXML `xml:"tcBorders"`
	Shading  shadingXML      `xml:"shd"`
	VAlign   vAlignXML       `xml:"vAlign"`
}

// gridSpanXML represents column span.
type gridSpanXML struct {
	Val string `xml:"val,attr"` // Number of columns spanned
}

// vMergeXML represents vertical merge.
type vMergeXML struct {
	XMLName xml.Name `xml:"vMerge"`
	Val     string   `xml:"val,attr"` // "restart" or empty (continue)
}

// shadingXML represents cell shading.
type shadingXML struct {
	Val   string `xml:"val,attr"`   // Pattern
	Color string `xml:"color,attr"` // Pattern color
	Fill  string `xml:"fill,attr"`  // Background color
}

// vAlignXML represents vertical alignment.
type vAlignXML struct {
	Val string `xml:"val,attr"` // top, center, bottom
}

// headerXML represents the structure of word/header*.xml files (<w:hdr>).
type headerXML struct {
	XMLName    xml.Name       `xml:"hdr"`
	Paragraphs []paragraphXML `xml:"p"`
	Tables     []tableXML     `xml:"tbl"`
}

// footerXML represents the structure of word/footer*.xml files (<w:ftr>).
type footerXML struct {
	XMLName    xml.Name       `xml:"ftr"`
	Paragraphs []paragraphXML `xml:"p"`
	Tables     []tableXML     `xml:"tbl"`
}
