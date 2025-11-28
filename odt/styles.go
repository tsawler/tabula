package odt

import "encoding/xml"

// stylesXML represents the structure of styles.xml
type stylesXML struct {
	XMLName      xml.Name         `xml:"document-styles"`
	Styles       *officeStylesXML `xml:"styles"`
	AutoStyles   *autoStylesXML   `xml:"automatic-styles"`
	MasterStyles *masterStylesXML `xml:"master-styles"`
}

// contentStylesXML represents automatic styles in content.xml
type contentStylesXML struct {
	XMLName    xml.Name       `xml:"automatic-styles"`
	Styles     []styleDefXML  `xml:"style"`
	ListStyles []listStyleXML `xml:"list-style"`
}

// officeStylesXML represents the office:styles element (named styles).
type officeStylesXML struct {
	Styles       []styleDefXML    `xml:"style"`
	ListStyles   []listStyleXML   `xml:"list-style"`
	OutlineStyle *outlineStyleXML `xml:"outline-style"`
}

// autoStylesXML represents the office:automatic-styles element.
type autoStylesXML struct {
	Styles      []styleDefXML   `xml:"style"`
	ListStyles  []listStyleXML  `xml:"list-style"`
	PageLayouts []pageLayoutXML `xml:"page-layout"`
}

// masterStylesXML represents the office:master-styles element.
type masterStylesXML struct {
	MasterPages []masterPageXML `xml:"master-page"`
}

// styleDefXML represents a style definition (<style:style>).
type styleDefXML struct {
	XMLName             xml.Name             `xml:"style"`
	Name                string               `xml:"name,attr"`
	Family              string               `xml:"family,attr"` // paragraph, text, table, table-cell, etc.
	ParentStyleName     string               `xml:"parent-style-name,attr"`
	DisplayName         string               `xml:"display-name,attr"`
	Class               string               `xml:"class,attr"`
	DefaultOutlineLevel string               `xml:"default-outline-level,attr"`
	ParagraphProps      *paragraphPropsXML   `xml:"paragraph-properties"`
	TextProps           *textPropsXML        `xml:"text-properties"`
	TableProps          *tablePropsXML       `xml:"table-properties"`
	TableColumnProps    *tableColumnPropsXML `xml:"table-column-properties"`
	TableRowProps       *tableRowPropsXML    `xml:"table-row-properties"`
	TableCellProps      *tableCellPropsXML   `xml:"table-cell-properties"`
}

// paragraphPropsXML represents paragraph properties (<style:paragraph-properties>).
type paragraphPropsXML struct {
	XMLName         xml.Name `xml:"paragraph-properties"`
	TextAlign       string   `xml:"text-align,attr"` // left, right, center, justify
	MarginTop       string   `xml:"margin-top,attr"`
	MarginBottom    string   `xml:"margin-bottom,attr"`
	MarginLeft      string   `xml:"margin-left,attr"`
	MarginRight     string   `xml:"margin-right,attr"`
	TextIndent      string   `xml:"text-indent,attr"`
	LineHeight      string   `xml:"line-height,attr"`
	BackgroundColor string   `xml:"background-color,attr"`
}

// textPropsXML represents text properties (<style:text-properties>).
type textPropsXML struct {
	XMLName         xml.Name `xml:"text-properties"`
	FontName        string   `xml:"font-name,attr"`
	FontFamily      string   `xml:"font-family,attr"`
	FontSize        string   `xml:"font-size,attr"`
	FontStyle       string   `xml:"font-style,attr"`           // normal, italic
	FontWeight      string   `xml:"font-weight,attr"`          // normal, bold
	TextUnderline   string   `xml:"text-underline-style,attr"` // none, solid
	TextLineThrough string   `xml:"text-line-through-style,attr"`
	Color           string   `xml:"color,attr"`
	BackgroundColor string   `xml:"background-color,attr"`
}

// tablePropsXML represents table properties (<style:table-properties>).
type tablePropsXML struct {
	XMLName         xml.Name `xml:"table-properties"`
	Width           string   `xml:"width,attr"`
	Align           string   `xml:"align,attr"`
	MarginLeft      string   `xml:"margin-left,attr"`
	MarginRight     string   `xml:"margin-right,attr"`
	BackgroundColor string   `xml:"background-color,attr"`
}

// tableColumnPropsXML represents table column properties.
type tableColumnPropsXML struct {
	XMLName     xml.Name `xml:"table-column-properties"`
	ColumnWidth string   `xml:"column-width,attr"`
}

// tableRowPropsXML represents table row properties.
type tableRowPropsXML struct {
	XMLName         xml.Name `xml:"table-row-properties"`
	RowHeight       string   `xml:"row-height,attr"`
	MinRowHeight    string   `xml:"min-row-height,attr"`
	BackgroundColor string   `xml:"background-color,attr"`
}

// tableCellPropsXML represents table cell properties.
type tableCellPropsXML struct {
	XMLName         xml.Name `xml:"table-cell-properties"`
	VerticalAlign   string   `xml:"vertical-align,attr"` // top, middle, bottom
	BackgroundColor string   `xml:"background-color,attr"`
	Padding         string   `xml:"padding,attr"`
	PaddingTop      string   `xml:"padding-top,attr"`
	PaddingBottom   string   `xml:"padding-bottom,attr"`
	PaddingLeft     string   `xml:"padding-left,attr"`
	PaddingRight    string   `xml:"padding-right,attr"`
	Border          string   `xml:"border,attr"`
	BorderTop       string   `xml:"border-top,attr"`
	BorderBottom    string   `xml:"border-bottom,attr"`
	BorderLeft      string   `xml:"border-left,attr"`
	BorderRight     string   `xml:"border-right,attr"`
}

// listStyleXML represents a list style definition (<text:list-style>).
type listStyleXML struct {
	XMLName      xml.Name             `xml:"list-style"`
	Name         string               `xml:"name,attr"`
	DisplayName  string               `xml:"display-name,attr"`
	Levels       []listLevelXML       `xml:"-"` // Populated by custom parsing
	BulletLevels []listLevelBulletXML `xml:"list-level-style-bullet"`
	NumberLevels []listLevelNumberXML `xml:"list-level-style-number"`
}

// listLevelXML is a generic interface for list levels.
type listLevelXML struct {
	Level      int
	IsBullet   bool
	BulletChar string
	NumFormat  string // "1", "a", "A", "i", "I"
	NumPrefix  string
	NumSuffix  string
	StartValue int
}

// listLevelBulletXML represents a bullet list level (<text:list-level-style-bullet>).
type listLevelBulletXML struct {
	XMLName    xml.Name `xml:"list-level-style-bullet"`
	Level      string   `xml:"level,attr"`
	BulletChar string   `xml:"bullet-char,attr"`
	StyleName  string   `xml:"style-name,attr"`
	NumPrefix  string   `xml:"num-prefix,attr"`
	NumSuffix  string   `xml:"num-suffix,attr"`
}

// listLevelNumberXML represents a numbered list level (<text:list-level-style-number>).
type listLevelNumberXML struct {
	XMLName       xml.Name `xml:"list-level-style-number"`
	Level         string   `xml:"level,attr"`
	NumFormat     string   `xml:"num-format,attr"` // "1", "a", "A", "i", "I"
	NumPrefix     string   `xml:"num-prefix,attr"`
	NumSuffix     string   `xml:"num-suffix,attr"`
	StartValue    string   `xml:"start-value,attr"`
	DisplayLevels string   `xml:"display-levels,attr"`
	StyleName     string   `xml:"style-name,attr"`
}

// outlineStyleXML represents outline numbering style.
type outlineStyleXML struct {
	XMLName xml.Name          `xml:"outline-style"`
	Name    string            `xml:"name,attr"`
	Levels  []outlineLevelXML `xml:"outline-level-style"`
}

// outlineLevelXML represents an outline level style.
type outlineLevelXML struct {
	XMLName    xml.Name `xml:"outline-level-style"`
	Level      string   `xml:"level,attr"`
	NumFormat  string   `xml:"num-format,attr"`
	NumPrefix  string   `xml:"num-prefix,attr"`
	NumSuffix  string   `xml:"num-suffix,attr"`
	StartValue string   `xml:"start-value,attr"`
}

// pageLayoutXML represents a page layout (<style:page-layout>).
type pageLayoutXML struct {
	XMLName     xml.Name              `xml:"page-layout"`
	Name        string                `xml:"name,attr"`
	PageProps   *pagePropsXML         `xml:"page-layout-properties"`
	HeaderStyle *headerFooterStyleXML `xml:"header-style"`
	FooterStyle *headerFooterStyleXML `xml:"footer-style"`
}

// pagePropsXML represents page layout properties.
type pagePropsXML struct {
	XMLName          xml.Name `xml:"page-layout-properties"`
	PageWidth        string   `xml:"page-width,attr"`
	PageHeight       string   `xml:"page-height,attr"`
	MarginTop        string   `xml:"margin-top,attr"`
	MarginBottom     string   `xml:"margin-bottom,attr"`
	MarginLeft       string   `xml:"margin-left,attr"`
	MarginRight      string   `xml:"margin-right,attr"`
	PrintOrientation string   `xml:"print-orientation,attr"`
}

// headerFooterStyleXML represents header/footer style.
type headerFooterStyleXML struct {
	Properties *headerFooterPropsXML `xml:"header-footer-properties"`
}

// headerFooterPropsXML represents header/footer properties.
type headerFooterPropsXML struct {
	MinHeight    string `xml:"min-height,attr"`
	MarginBottom string `xml:"margin-bottom,attr"`
}

// masterPageXML represents a master page (<style:master-page>).
type masterPageXML struct {
	XMLName        xml.Name           `xml:"master-page"`
	Name           string             `xml:"name,attr"`
	PageLayoutName string             `xml:"page-layout-name,attr"`
	DisplayName    string             `xml:"display-name,attr"`
	Header         *masterHeaderXML   `xml:"header"`
	Footer         *masterFooterXML   `xml:"footer"`
	HeaderLeft     *masterHeaderXML   `xml:"header-left"`
	FooterLeft     *masterFooterXML   `xml:"footer-left"`
	HeaderFirst    *masterHeaderXML   `xml:"header-first"`
	FooterFirst    *masterFooterXML   `xml:"footer-first"`
}

// masterHeaderXML represents header content in a master page (<style:header>).
type masterHeaderXML struct {
	Paragraphs []masterParagraphXML `xml:"p"`
}

// masterFooterXML represents footer content in a master page (<style:footer>).
type masterFooterXML struct {
	Paragraphs []masterParagraphXML `xml:"p"`
}

// masterParagraphXML represents a paragraph in header/footer.
type masterParagraphXML struct {
	StyleName string          `xml:"style-name,attr"`
	Text      string          `xml:",chardata"`
	Spans     []masterSpanXML `xml:"span"`
}

// masterSpanXML represents a text span in header/footer paragraph.
type masterSpanXML struct {
	StyleName string `xml:"style-name,attr"`
	Text      string `xml:",chardata"`
}

// metaXML represents document metadata from meta.xml.
type metaXML struct {
	XMLName xml.Name     `xml:"document-meta"`
	Meta    *metaInfoXML `xml:"meta"`
}

// metaInfoXML represents the office:meta element.
type metaInfoXML struct {
	Title          string `xml:"title"`
	Description    string `xml:"description"`
	Subject        string `xml:"subject"`
	Keyword        string `xml:"keyword"`
	InitialCreator string `xml:"initial-creator"`
	Creator        string `xml:"creator"`
	CreationDate   string `xml:"creation-date"`
	Date           string `xml:"date"` // Last modified
	Generator      string `xml:"generator"`
	Language       string `xml:"language"`
}
