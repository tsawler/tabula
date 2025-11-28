package docx

import "encoding/xml"

// stylesXML represents the structure of word/styles.xml
type stylesXML struct {
	XMLName     xml.Name       `xml:"styles"`
	DocDefaults docDefaultsXML `xml:"docDefaults"`
	Styles      []styleDefXML  `xml:"style"`
}

// docDefaultsXML represents document default styles.
type docDefaultsXML struct {
	RPrDefault rPrDefaultXML `xml:"rPrDefault"`
	PPrDefault pPrDefaultXML `xml:"pPrDefault"`
}

// rPrDefaultXML represents default run properties.
type rPrDefaultXML struct {
	RPr runPropsXML `xml:"rPr"`
}

// pPrDefaultXML represents default paragraph properties.
type pPrDefaultXML struct {
	PPr paragraphPropsXML `xml:"pPr"`
}

// styleDefXML represents a style definition.
type styleDefXML struct {
	XMLName     xml.Name          `xml:"style"`
	Type        string            `xml:"type,attr"` // paragraph, character, table, numbering
	StyleID     string            `xml:"styleId,attr"`
	Default     string            `xml:"default,attr"` // "1" if default style
	CustomStyle string            `xml:"customStyle,attr"`
	Name        styleNameXML      `xml:"name"`
	BasedOn     basedOnXML        `xml:"basedOn"`
	Next        nextXML           `xml:"next"`
	Link        linkXML           `xml:"link"`
	PPr         paragraphPropsXML `xml:"pPr"`
	RPr         runPropsXML       `xml:"rPr"`
	TblPr       tablePropsXML     `xml:"tblPr"`
}

// styleNameXML represents a style name.
type styleNameXML struct {
	Val string `xml:"val,attr"`
}

// basedOnXML represents parent style reference.
type basedOnXML struct {
	Val string `xml:"val,attr"`
}

// nextXML represents next paragraph style.
type nextXML struct {
	Val string `xml:"val,attr"`
}

// linkXML represents linked style.
type linkXML struct {
	Val string `xml:"val,attr"`
}

// numberingXML represents word/numbering.xml
type numberingXML struct {
	XMLName      xml.Name         `xml:"numbering"`
	AbstractNums []abstractNumXML `xml:"abstractNum"`
	Nums         []numXML         `xml:"num"`
}

// abstractNumXML represents an abstract numbering definition.
type abstractNumXML struct {
	AbstractNumID string          `xml:"abstractNumId,attr"`
	Levels        []lvlXML        `xml:"lvl"`
	NumStyleLink  numStyleLinkXML `xml:"numStyleLink"`
}

// numStyleLinkXML represents a style link for numbering.
type numStyleLinkXML struct {
	Val string `xml:"val,attr"`
}

// lvlXML represents a numbering level.
type lvlXML struct {
	ILvl    string            `xml:"ilvl,attr"`
	Start   startXML          `xml:"start"`
	NumFmt  numFmtXML         `xml:"numFmt"`
	LvlText lvlTextXML        `xml:"lvlText"`
	LvlJc   lvlJcXML          `xml:"lvlJc"`
	PPr     paragraphPropsXML `xml:"pPr"`
}

// startXML represents numbering start value.
type startXML struct {
	Val string `xml:"val,attr"`
}

// numFmtXML represents number format.
type numFmtXML struct {
	Val string `xml:"val,attr"` // decimal, bullet, lowerLetter, upperLetter, lowerRoman, upperRoman
}

// lvlTextXML represents level text pattern.
type lvlTextXML struct {
	Val string `xml:"val,attr"` // e.g., "%1.", "%1.%2"
}

// lvlJcXML represents level justification.
type lvlJcXML struct {
	Val string `xml:"val,attr"` // left, center, right
}

// numXML represents a numbering instance.
type numXML struct {
	NumID         string         `xml:"numId,attr"`
	AbstractNumID abstractRefXML `xml:"abstractNumId"`
}

// abstractRefXML represents reference to abstract numbering.
type abstractRefXML struct {
	Val string `xml:"val,attr"`
}

// relationshipsXML represents _rels/*.rels files
type relationshipsXML struct {
	XMLName       xml.Name          `xml:"Relationships"`
	Relationships []relationshipXML `xml:"Relationship"`
}

// relationshipXML represents a single relationship.
type relationshipXML struct {
	ID         string `xml:"Id,attr"`
	Type       string `xml:"Type,attr"`
	Target     string `xml:"Target,attr"`
	TargetMode string `xml:"TargetMode,attr"` // External or empty (internal)
}

// corePropertiesXML represents docProps/core.xml (Dublin Core metadata)
type corePropertiesXML struct {
	XMLName        xml.Name `xml:"coreProperties"`
	Title          string   `xml:"title"`
	Subject        string   `xml:"subject"`
	Creator        string   `xml:"creator"`
	Keywords       string   `xml:"keywords"`
	Description    string   `xml:"description"`
	LastModifiedBy string   `xml:"lastModifiedBy"`
	Revision       string   `xml:"revision"`
	Created        string   `xml:"created"`
	Modified       string   `xml:"modified"`
	Category       string   `xml:"category"`
}

// appPropertiesXML represents docProps/app.xml
type appPropertiesXML struct {
	XMLName     xml.Name `xml:"Properties"`
	Template    string   `xml:"Template"`
	TotalTime   string   `xml:"TotalTime"`
	Pages       string   `xml:"Pages"`
	Words       string   `xml:"Words"`
	Characters  string   `xml:"Characters"`
	Application string   `xml:"Application"`
	DocSecurity string   `xml:"DocSecurity"`
	Lines       string   `xml:"Lines"`
	Paragraphs  string   `xml:"Paragraphs"`
	Company     string   `xml:"Company"`
}
