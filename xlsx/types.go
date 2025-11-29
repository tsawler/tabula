// Package xlsx provides XLSX (Office Open XML Spreadsheet) document parsing.
package xlsx

import "encoding/xml"

// XML namespaces used in XLSX files.
const (
	nsSpreadsheetML = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	nsRelationships = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	nsPackageRels   = "http://schemas.openxmlformats.org/package/2006/relationships"
)

// workbookXML represents the xl/workbook.xml file structure.
type workbookXML struct {
	XMLName xml.Name  `xml:"workbook"`
	Sheets  sheetsXML `xml:"sheets"`
}

type sheetsXML struct {
	Sheet []sheetRefXML `xml:"sheet"`
}

type sheetRefXML struct {
	Name    string `xml:"name,attr"`
	SheetID string `xml:"sheetId,attr"`
	RID     string `xml:"id,attr"` // r:id attribute for relationship
}

// worksheetXML represents a xl/worksheets/sheet*.xml file structure.
type worksheetXML struct {
	XMLName    xml.Name       `xml:"worksheet"`
	Dimension  dimensionXML   `xml:"dimension"`
	SheetData  sheetDataXML   `xml:"sheetData"`
	MergeCells *mergeCellsXML `xml:"mergeCells"`
}

type dimensionXML struct {
	Ref string `xml:"ref,attr"` // e.g., "A1:D10"
}

type sheetDataXML struct {
	Rows []rowXML `xml:"row"`
}

type rowXML struct {
	R     int       `xml:"r,attr"` // Row number (1-indexed)
	Cells []cellXML `xml:"c"`
}

type cellXML struct {
	R  string        `xml:"r,attr"` // Cell reference (e.g., "A1")
	T  string        `xml:"t,attr"` // Type: s=shared string, n=number, b=bool, str=inline string, e=error
	S  int           `xml:"s,attr"` // Style index
	V  string        `xml:"v"`      // Value
	F  string        `xml:"f"`      // Formula (optional)
	Is *inlineStrXML `xml:"is"`     // Inline string (optional)
}

type inlineStrXML struct {
	T string `xml:"t"` // Text content
}

type mergeCellsXML struct {
	MergeCell []mergeCellXML `xml:"mergeCell"`
}

type mergeCellXML struct {
	Ref string `xml:"ref,attr"` // e.g., "A1:B2"
}

// sharedStringsXML represents the xl/sharedStrings.xml file structure.
type sharedStringsXML struct {
	XMLName xml.Name `xml:"sst"`
	Count   int      `xml:"count,attr"`
	Unique  int      `xml:"uniqueCount,attr"`
	SI      []siXML  `xml:"si"`
}

type siXML struct {
	T string `xml:"t"` // Simple text
	R []rXML `xml:"r"` // Rich text runs
}

type rXML struct {
	T string `xml:"t"` // Text in run
}

// stylesXML represents the xl/styles.xml file structure.
type stylesXML struct {
	XMLName xml.Name    `xml:"styleSheet"`
	NumFmts *numFmtsXML `xml:"numFmts"`
	CellXfs *cellXfsXML `xml:"cellXfs"`
}

type numFmtsXML struct {
	NumFmt []numFmtXML `xml:"numFmt"`
}

type numFmtXML struct {
	NumFmtID   int    `xml:"numFmtId,attr"`
	FormatCode string `xml:"formatCode,attr"`
}

type cellXfsXML struct {
	Xf []xfXML `xml:"xf"`
}

type xfXML struct {
	NumFmtID int `xml:"numFmtId,attr"`
	FontID   int `xml:"fontId,attr"`
	FillID   int `xml:"fillId,attr"`
	BorderID int `xml:"borderId,attr"`
}

// relationshipsXML represents .rels files.
type relationshipsXML struct {
	XMLName      xml.Name          `xml:"Relationships"`
	Relationship []relationshipXML `xml:"Relationship"`
}

type relationshipXML struct {
	ID     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

// corePropertiesXML represents docProps/core.xml.
type corePropertiesXML struct {
	XMLName     xml.Name `xml:"coreProperties"`
	Title       string   `xml:"title"`
	Subject     string   `xml:"subject"`
	Creator     string   `xml:"creator"`
	Keywords    string   `xml:"keywords"`
	Description string   `xml:"description"`
	LastModBy   string   `xml:"lastModifiedBy"`
}

// appPropertiesXML represents docProps/app.xml.
type appPropertiesXML struct {
	XMLName     xml.Name `xml:"Properties"`
	Application string   `xml:"Application"`
	Company     string   `xml:"Company"`
}
