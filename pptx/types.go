// Package pptx provides PPTX (Office Open XML Presentation) document parsing.
package pptx

import "encoding/xml"

// XML namespaces used in PPTX files.
const (
	nsPresentationML = "http://schemas.openxmlformats.org/presentationml/2006/main"
	nsDrawingML      = "http://schemas.openxmlformats.org/drawingml/2006/main"
	nsRelationships  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	nsPackageRels    = "http://schemas.openxmlformats.org/package/2006/relationships"
)

// presentationXML represents the ppt/presentation.xml file structure.
type presentationXML struct {
	XMLName     xml.Name       `xml:"presentation"`
	SlideIdList *slideIdListXML `xml:"sldIdLst"`
	SlideSz     *slideSzXML    `xml:"sldSz"`
}

type slideIdListXML struct {
	SlideId []slideIdXML `xml:"sldId"`
}

type slideIdXML struct {
	ID  string `xml:"id,attr"`
	RID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"` // r:id attribute for relationship
}

type slideSzXML struct {
	Cx int `xml:"cx,attr"` // Width in EMUs
	Cy int `xml:"cy,attr"` // Height in EMUs
}

// slideXML represents a ppt/slides/slide*.xml file structure.
type slideXML struct {
	XMLName xml.Name `xml:"sld"`
	CSld    cSldXML  `xml:"cSld"`
}

type cSldXML struct {
	SpTree spTreeXML `xml:"spTree"`
}

// spTreeXML represents the shape tree containing all shapes on a slide.
type spTreeXML struct {
	NvGrpSpPr nvGrpSpPrXML `xml:"nvGrpSpPr"`
	Sp        []spXML      `xml:"sp"`        // Regular shapes
	Pic       []picXML     `xml:"pic"`       // Pictures
	GraphicFrame []graphicFrameXML `xml:"graphicFrame"` // Tables, charts, etc.
	GrpSp     []grpSpXML   `xml:"grpSp"`     // Grouped shapes
}

type nvGrpSpPrXML struct {
	CNvPr cNvPrXML `xml:"cNvPr"`
}

type cNvPrXML struct {
	ID    int    `xml:"id,attr"`
	Name  string `xml:"name,attr"`
	Title string `xml:"title,attr"`
}

// spXML represents a shape element.
type spXML struct {
	NvSpPr nvSpPrXML  `xml:"nvSpPr"`
	SpPr   spPrXML    `xml:"spPr"`
	TxBody *txBodyXML `xml:"txBody"`
}

type nvSpPrXML struct {
	CNvPr   cNvPrXML   `xml:"cNvPr"`
	NvPr    nvPrXML    `xml:"nvPr"`
}

type nvPrXML struct {
	Ph *phXML `xml:"ph"` // Placeholder info
}

type phXML struct {
	Type string `xml:"type,attr"` // title, body, subTitle, ctrTitle, etc.
	Idx  int    `xml:"idx,attr"`
}

type spPrXML struct {
	Xfrm *xfrmXML `xml:"xfrm"`
}

type xfrmXML struct {
	Off offXML `xml:"off"`
	Ext extXML `xml:"ext"`
}

type offXML struct {
	X int `xml:"x,attr"` // X position in EMUs
	Y int `xml:"y,attr"` // Y position in EMUs
}

type extXML struct {
	Cx int `xml:"cx,attr"` // Width in EMUs
	Cy int `xml:"cy,attr"` // Height in EMUs
}

// txBodyXML represents text body content.
type txBodyXML struct {
	BodyPr bodyPrXML `xml:"bodyPr"`
	P      []pXML    `xml:"p"` // Paragraphs
}

type bodyPrXML struct {
	Anchor string `xml:"anchor,attr"` // t, ctr, b (top, center, bottom)
}

// pXML represents a paragraph.
type pXML struct {
	PPr *pPrXML `xml:"pPr"` // Paragraph properties
	R   []rXML  `xml:"r"`   // Text runs
	Br  []brXML `xml:"br"`  // Line breaks
	Fld []fldXML `xml:"fld"` // Fields (like slide number)
	EndParaRPr *rPrXML `xml:"endParaRPr"` // End paragraph run properties
}

type pPrXML struct {
	Lvl     int      `xml:"lvl,attr"`     // Bullet level (0-8)
	Algn    string   `xml:"algn,attr"`    // Alignment: l, ctr, r, just
	MarL    int      `xml:"marL,attr"`    // Left margin in EMUs
	Indent  int      `xml:"indent,attr"`  // First line indent in EMUs
	BuNone  *struct{} `xml:"buNone"`      // No bullet
	BuChar  *buCharXML `xml:"buChar"`     // Character bullet
	BuAutoNum *buAutoNumXML `xml:"buAutoNum"` // Numbered list
}

type buCharXML struct {
	Char string `xml:"char,attr"` // Bullet character
}

type buAutoNumXML struct {
	Type    string `xml:"type,attr"`    // arabicPeriod, alphaLcParenR, etc.
	StartAt int    `xml:"startAt,attr"` // Starting number
}

// rXML represents a text run.
type rXML struct {
	RPr *rPrXML `xml:"rPr"` // Run properties
	T   string  `xml:"t"`   // Text content
}

type rPrXML struct {
	Lang   string `xml:"lang,attr"`
	Sz     int    `xml:"sz,attr"`    // Font size in hundredths of a point
	B      *int   `xml:"b,attr"`     // Bold (1 = true)
	I      *int   `xml:"i,attr"`     // Italic (1 = true)
	U      string `xml:"u,attr"`     // Underline type
}

type brXML struct{} // Line break

type fldXML struct {
	Type string `xml:"type,attr"` // slidenum, datetime, etc.
	T    string `xml:"t"`         // Field value
}

// picXML represents a picture element.
type picXML struct {
	NvPicPr nvPicPrXML `xml:"nvPicPr"`
	BlipFill blipFillXML `xml:"blipFill"`
}

type nvPicPrXML struct {
	CNvPr cNvPrXML `xml:"cNvPr"`
}

type blipFillXML struct {
	Blip blipXML `xml:"blip"`
}

type blipXML struct {
	Embed string `xml:"embed,attr"` // r:embed relationship ID
}

// graphicFrameXML represents a graphic frame (tables, charts).
type graphicFrameXML struct {
	NvGraphicFramePr nvGraphicFramePrXML `xml:"nvGraphicFramePr"`
	Graphic          graphicXML          `xml:"graphic"`
}

type nvGraphicFramePrXML struct {
	CNvPr cNvPrXML `xml:"cNvPr"`
}

type graphicXML struct {
	GraphicData graphicDataXML `xml:"graphicData"`
}

type graphicDataXML struct {
	URI string   `xml:"uri,attr"`
	Tbl *tblXML  `xml:"tbl"` // Table
}

// tblXML represents a table.
type tblXML struct {
	TblGrid tblGridXML `xml:"tblGrid"`
	Tr      []trXML    `xml:"tr"` // Table rows
}

type tblGridXML struct {
	GridCol []gridColXML `xml:"gridCol"`
}

type gridColXML struct {
	W int `xml:"w,attr"` // Width in EMUs
}

type trXML struct {
	H  int     `xml:"h,attr"` // Row height in EMUs
	Tc []tcXML `xml:"tc"`     // Table cells
}

type tcXML struct {
	TxBody   *txBodyXML `xml:"txBody"`
	RowSpan  int        `xml:"rowSpan,attr"`
	GridSpan int        `xml:"gridSpan,attr"`
	VMerge   *int       `xml:"vMerge,attr"` // Vertical merge
	HMerge   *int       `xml:"hMerge,attr"` // Horizontal merge
}

// grpSpXML represents a group of shapes.
type grpSpXML struct {
	NvGrpSpPr nvGrpSpPrXML `xml:"nvGrpSpPr"`
	GrpSpPr   grpSpPrXML   `xml:"grpSpPr"`
	Sp        []spXML      `xml:"sp"`
	Pic       []picXML     `xml:"pic"`
	GrpSp     []grpSpXML   `xml:"grpSp"` // Nested groups
}

type grpSpPrXML struct {
	Xfrm *xfrmXML `xml:"xfrm"`
}

// notesSlideXML represents a ppt/notesSlides/notesSlide*.xml file.
type notesSlideXML struct {
	XMLName xml.Name `xml:"notes"`
	CSld    cSldXML  `xml:"cSld"`
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
	Slides      int      `xml:"Slides"`
	Notes       int      `xml:"Notes"`
}
