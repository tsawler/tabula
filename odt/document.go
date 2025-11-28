package odt

import "encoding/xml"

// ODF XML namespaces
const (
	nsOffice = "urn:oasis:names:tc:opendocument:xmlns:office:1.0"
	nsStyle  = "urn:oasis:names:tc:opendocument:xmlns:style:1.0"
	nsText   = "urn:oasis:names:tc:opendocument:xmlns:text:1.0"
	nsTable  = "urn:oasis:names:tc:opendocument:xmlns:table:1.0"
	nsDraw   = "urn:oasis:names:tc:opendocument:xmlns:drawing:1.0"
	nsFO     = "urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0"
	nsSVG    = "urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0"
	nsDC     = "http://purl.org/dc/elements/1.1/"
	nsMeta   = "urn:oasis:names:tc:opendocument:xmlns:meta:1.0"
	nsXLink  = "http://www.w3.org/1999/xlink"
)

// documentXML represents the structure of content.xml
type documentXML struct {
	XMLName xml.Name `xml:"document-content"`
	Body    *bodyXML `xml:"body"`
}

// bodyXML represents the document body.
type bodyXML struct {
	Text *textBodyXML `xml:"text"`
}

// textBodyXML represents the text body content.
type textBodyXML struct {
	Elements []bodyElement `xml:"-"` // Populated manually to preserve order
}

// bodyElement represents an element in the document body (paragraph, heading, list, or table).
type bodyElement struct {
	Type      string        // "paragraph", "heading", "list", or "table"
	Paragraph *paragraphXML // Non-nil if Type == "paragraph"
	Heading   *headingXML   // Non-nil if Type == "heading"
	List      *listXML      // Non-nil if Type == "list"
	Table     *tableXML     // Non-nil if Type == "table"
}

// paragraphXML represents a paragraph element (<text:p>).
type paragraphXML struct {
	XMLName   xml.Name  `xml:"p"`
	StyleName string    `xml:"style-name,attr"`
	Spans     []spanXML `xml:"span"`
	Text      string    `xml:",chardata"`
}

// headingXML represents a heading element (<text:h>).
type headingXML struct {
	XMLName      xml.Name  `xml:"h"`
	StyleName    string    `xml:"style-name,attr"`
	OutlineLevel string    `xml:"outline-level,attr"`
	Spans        []spanXML `xml:"span"`
	Text         string    `xml:",chardata"`
}

// spanXML represents a text span with formatting (<text:span>).
type spanXML struct {
	XMLName   xml.Name `xml:"span"`
	StyleName string   `xml:"style-name,attr"`
	Text      string   `xml:",chardata"`
}

// listXML represents a list (<text:list>).
type listXML struct {
	XMLName   xml.Name      `xml:"list"`
	StyleName string        `xml:"style-name,attr"`
	Items     []listItemXML `xml:"list-item"`
}

// listItemXML represents a list item (<text:list-item>).
type listItemXML struct {
	XMLName    xml.Name       `xml:"list-item"`
	Paragraphs []paragraphXML `xml:"p"`
	SubLists   []listXML      `xml:"list"` // Nested lists
}

// tableXML represents a table (<table:table>).
type tableXML struct {
	XMLName   xml.Name      `xml:"table"`
	Name      string        `xml:"name,attr"`
	StyleName string        `xml:"style-name,attr"`
	Columns   []tableColXML `xml:"table-column"`
	Rows      []tableRowXML `xml:"table-row"`
}

// tableColXML represents a table column definition.
type tableColXML struct {
	XMLName        xml.Name `xml:"table-column"`
	StyleName      string   `xml:"style-name,attr"`
	NumberRepeated string   `xml:"number-columns-repeated,attr"`
}

// tableRowXML represents a table row (<table:table-row>).
type tableRowXML struct {
	XMLName   xml.Name       `xml:"table-row"`
	StyleName string         `xml:"style-name,attr"`
	Cells     []tableCellXML `xml:"table-cell"`
}

// tableCellXML represents a table cell (<table:table-cell>).
type tableCellXML struct {
	XMLName              xml.Name       `xml:"table-cell"`
	StyleName            string         `xml:"style-name,attr"`
	NumberColumnsSpanned string         `xml:"number-columns-spanned,attr"`
	NumberRowsSpanned    string         `xml:"number-rows-spanned,attr"`
	Paragraphs           []paragraphXML `xml:"p"`
}

// coveredCellXML represents a covered (merged) cell (<table:covered-table-cell>).
type coveredCellXML struct {
	XMLName xml.Name `xml:"covered-table-cell"`
}

// parsedParagraph holds a parsed paragraph with resolved styles.
type parsedParagraph struct {
	Text      string
	StyleName string
	IsHeading bool
	Level     int // heading level (1-9) or 0 for non-headings

	// List properties
	IsListItem bool
	ListLevel  int // List indentation level (0-based)

	// Paragraph properties (resolved from style)
	Alignment   string  // left, center, right, justify
	SpaceBefore float64 // points
	SpaceAfter  float64 // points
	IndentLeft  float64 // points
	IndentRight float64 // points
	IndentFirst float64 // points

	// Text runs with formatting
	Runs []parsedRun
}

// parsedRun holds a parsed text run with formatting.
type parsedRun struct {
	Text      string
	FontName  string
	FontSize  float64 // points
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	Color     string // hex color
}

// parsedElement represents a parsed element (paragraph, heading, list, or table) with its type.
type parsedElement struct {
	Type      string           // "paragraph" or "table"
	Paragraph *parsedParagraph // Non-nil if Type == "paragraph"
	Table     *ParsedTable     // Non-nil if Type == "table"
}
