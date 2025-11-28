// Package docx provides DOCX (Office Open XML) document parsing.
package docx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tsawler/tabula/model"
)

// Reader provides access to DOCX document content.
type Reader struct {
	file          *os.File
	zipReader     *zip.ReadCloser
	document      *documentXML
	styles        *stylesXML
	numbering     *numberingXML
	rels          *relationshipsXML
	coreProps     *corePropertiesXML
	appProps      *appPropertiesXML
	styleResolver *StyleResolver
	tableParser   *TableParser
	paragraphs    []parsedParagraph
	tables        []ParsedTable
}

// parsedParagraph holds a parsed paragraph with resolved styles.
type parsedParagraph struct {
	Text      string
	StyleID   string
	StyleName string
	IsHeading bool
	Level     int // heading level (1-9) or 0 for non-headings

	// Paragraph properties
	Alignment   string  // left, center, right, both
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
	Highlight string // highlight color name
}

// Open opens a DOCX file for reading.
func Open(filename string) (*Reader, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, fmt.Errorf("opening ZIP archive: %w", err)
	}

	r := &Reader{
		zipReader: zr,
	}

	// Validate required files exist
	if err := r.validate(); err != nil {
		zr.Close()
		return nil, err
	}

	// Parse relationships first (needed for other parts)
	if err := r.parseRelationships(); err != nil {
		zr.Close()
		return nil, fmt.Errorf("parsing relationships: %w", err)
	}

	// Parse styles.xml first (optional but usually present)
	_ = r.parseStyles() // Creates styleResolver even on error

	// Ensure styleResolver exists
	if r.styleResolver == nil {
		r.styleResolver = NewStyleResolver(nil)
	}

	// Create table parser
	r.tableParser = NewTableParser(r.styleResolver)

	// Parse document.xml (now that styleResolver and tableParser are ready)
	if err := r.parseDocument(); err != nil {
		zr.Close()
		return nil, fmt.Errorf("parsing document: %w", err)
	}

	// Parse numbering.xml (optional)
	if err := r.parseNumbering(); err != nil {
		// Numbering is optional - just continue without it
	}

	// Parse metadata (optional)
	r.parseCoreProperties()
	r.parseAppProperties()

	return r, nil
}

// Close releases resources associated with the Reader.
func (r *Reader) Close() error {
	if r.zipReader != nil {
		err := r.zipReader.Close()
		r.zipReader = nil
		return err
	}
	return nil
}

// validate checks that required DOCX files exist.
func (r *Reader) validate() error {
	required := []string{
		"[Content_Types].xml",
		"word/document.xml",
	}

	fileMap := make(map[string]bool)
	for _, f := range r.zipReader.File {
		fileMap[f.Name] = true
	}

	for _, name := range required {
		if !fileMap[name] {
			return fmt.Errorf("missing required file: %s", name)
		}
	}

	return nil
}

// getFileContent reads the content of a file from the ZIP archive.
func (r *Reader) getFileContent(name string) ([]byte, error) {
	for _, f := range r.zipReader.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

// getFile returns a zip.File by name.
func (r *Reader) getFile(name string) *zip.File {
	for _, f := range r.zipReader.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// PageCount returns the number of "pages" in the document.
// Since DOCX doesn't have fixed pages, we return 1 (entire document as single page).
func (r *Reader) PageCount() (int, error) {
	return 1, nil
}

// Text extracts and returns all text content from the document.
// This includes text from paragraphs and tables.
func (r *Reader) Text() (string, error) {
	if r.document == nil {
		return "", fmt.Errorf("document not parsed")
	}

	var result strings.Builder

	// Add paragraph text
	for i, para := range r.paragraphs {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(para.Text)
	}

	// Add table text
	for _, tbl := range r.tables {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(tbl.ToText())
	}

	return result.String(), nil
}

// Document returns a model.Document representation of the DOCX content.
func (r *Reader) Document() (*model.Document, error) {
	doc := model.NewDocument()

	// Set metadata
	if r.coreProps != nil {
		doc.Metadata.Title = r.coreProps.Title
		doc.Metadata.Author = r.coreProps.Creator
		doc.Metadata.Subject = r.coreProps.Subject
		if r.coreProps.Keywords != "" {
			doc.Metadata.Keywords = strings.Split(r.coreProps.Keywords, ",")
			for i, kw := range doc.Metadata.Keywords {
				doc.Metadata.Keywords[i] = strings.TrimSpace(kw)
			}
		}
	}
	if r.appProps != nil {
		doc.Metadata.Creator = r.appProps.Application
	}

	// Create single page for entire document (DOCX doesn't have fixed pages)
	page := model.NewPage(612, 792) // Standard US Letter dimensions
	page.Number = 1

	// Add elements
	yPos := 750.0 // Start near top of page
	for _, para := range r.paragraphs {
		if para.Text == "" {
			continue
		}

		estimatedHeight := 14.0 // Default line height
		if para.IsHeading {
			estimatedHeight = float64(24 - para.Level*2) // Larger for higher-level headings
		}

		bbox := model.BBox{
			X:      72,  // 1 inch margin
			Y:      yPos,
			Width:  468, // 6.5 inch text width
			Height: estimatedHeight,
		}

		if para.IsHeading {
			heading := &model.Heading{
				Level: para.Level,
				Text:  para.Text,
				BBox:  bbox,
			}
			page.AddElement(heading)
		} else {
			paragraph := &model.Paragraph{
				Text: para.Text,
				BBox: bbox,
			}
			page.AddElement(paragraph)
		}

		yPos -= estimatedHeight + 4 // Move down for next element
	}

	// Add tables
	for _, tbl := range r.tables {
		modelTable := tbl.ToModelTable()
		if modelTable.RowCount() > 0 {
			// Estimate table height
			tableHeight := float64(modelTable.RowCount()) * 20.0

			modelTable.BBox = model.BBox{
				X:      72,
				Y:      yPos,
				Width:  468,
				Height: tableHeight,
			}

			page.AddElement(modelTable)
			yPos -= tableHeight + 10
		}
	}

	doc.AddPage(page)
	return doc, nil
}

// Metadata returns document metadata.
func (r *Reader) Metadata() model.Metadata {
	meta := model.Metadata{}
	if r.coreProps != nil {
		meta.Title = r.coreProps.Title
		meta.Author = r.coreProps.Creator
		meta.Subject = r.coreProps.Subject
		if r.coreProps.Keywords != "" {
			meta.Keywords = strings.Split(r.coreProps.Keywords, ",")
			for i, kw := range meta.Keywords {
				meta.Keywords[i] = strings.TrimSpace(kw)
			}
		}
	}
	if r.appProps != nil {
		meta.Creator = r.appProps.Application
	}
	return meta
}

// Tables returns all parsed tables from the document.
func (r *Reader) Tables() []ParsedTable {
	return r.tables
}

// ModelTables returns tables converted to model.Table format.
func (r *Reader) ModelTables() []*model.Table {
	result := make([]*model.Table, len(r.tables))
	for i := range r.tables {
		result[i] = r.tables[i].ToModelTable()
	}
	return result
}

// parseRelationships parses the document relationships file.
func (r *Reader) parseRelationships() error {
	data, err := r.getFileContent("word/_rels/document.xml.rels")
	if err != nil {
		// Relationships file is optional
		return nil
	}

	r.rels = &relationshipsXML{}
	return xml.Unmarshal(data, r.rels)
}

// parseDocument parses the main document content.
func (r *Reader) parseDocument() error {
	data, err := r.getFileContent("word/document.xml")
	if err != nil {
		return err
	}

	r.document = &documentXML{}
	if err := xml.Unmarshal(data, r.document); err != nil {
		return fmt.Errorf("unmarshaling document.xml: %w", err)
	}

	// Process paragraphs
	r.processParagraphs()

	return nil
}

// parseStyles parses the styles definition file and creates the resolver.
func (r *Reader) parseStyles() error {
	data, err := r.getFileContent("word/styles.xml")
	if err != nil {
		// Create resolver with no styles (will use defaults)
		r.styleResolver = NewStyleResolver(nil)
		return err
	}

	r.styles = &stylesXML{}
	if err := xml.Unmarshal(data, r.styles); err != nil {
		r.styleResolver = NewStyleResolver(nil)
		return err
	}

	// Create style resolver
	r.styleResolver = NewStyleResolver(r.styles)
	return nil
}

// parseNumbering parses the numbering definitions file.
func (r *Reader) parseNumbering() error {
	data, err := r.getFileContent("word/numbering.xml")
	if err != nil {
		return err
	}

	r.numbering = &numberingXML{}
	return xml.Unmarshal(data, r.numbering)
}

// parseCoreProperties parses Dublin Core metadata.
func (r *Reader) parseCoreProperties() {
	data, err := r.getFileContent("docProps/core.xml")
	if err != nil {
		return
	}

	r.coreProps = &corePropertiesXML{}
	xml.Unmarshal(data, r.coreProps)
}

// parseAppProperties parses application metadata.
func (r *Reader) parseAppProperties() {
	data, err := r.getFileContent("docProps/app.xml")
	if err != nil {
		return
	}

	r.appProps = &appPropertiesXML{}
	xml.Unmarshal(data, r.appProps)
}

// processParagraphs processes all paragraphs in the document.
func (r *Reader) processParagraphs() {
	if r.document == nil || r.document.Body == nil {
		return
	}

	r.paragraphs = make([]parsedParagraph, 0, len(r.document.Body.Paragraphs))

	for _, p := range r.document.Body.Paragraphs {
		parsed := r.processParagraph(p)
		r.paragraphs = append(r.paragraphs, parsed)
	}

	// Process tables
	r.processTables()
}

// processTables processes all tables in the document.
func (r *Reader) processTables() {
	if r.document == nil || r.document.Body == nil || r.tableParser == nil {
		return
	}

	r.tables = make([]ParsedTable, 0, len(r.document.Body.Tables))

	for _, tbl := range r.document.Body.Tables {
		parsed := r.tableParser.ParseTable(tbl)
		r.tables = append(r.tables, parsed)
	}
}

// processParagraph processes a single paragraph.
func (r *Reader) processParagraph(p paragraphXML) parsedParagraph {
	parsed := parsedParagraph{
		StyleID: p.Properties.Style.Val,
	}

	// Resolve style (handles inheritance)
	var resolvedStyle *ResolvedStyle
	if r.styleResolver != nil {
		resolvedStyle = r.styleResolver.Resolve(parsed.StyleID)
		parsed.StyleName = resolvedStyle.Name
		parsed.IsHeading = resolvedStyle.IsHeading
		parsed.Level = resolvedStyle.HeadingLevel

		// Apply style's paragraph properties
		parsed.Alignment = resolvedStyle.Alignment
		parsed.SpaceBefore = resolvedStyle.SpaceBefore
		parsed.SpaceAfter = resolvedStyle.SpaceAfter
		parsed.IndentLeft = resolvedStyle.IndentLeft
		parsed.IndentRight = resolvedStyle.IndentRight
		parsed.IndentFirst = resolvedStyle.IndentFirst
	}

	// Apply direct paragraph formatting (overrides style)
	ppr := p.Properties
	if ppr.Justification.Val != "" {
		parsed.Alignment = ppr.Justification.Val
	}
	if ppr.Spacing.Before != "" {
		parsed.SpaceBefore = parseTwips(ppr.Spacing.Before)
	}
	if ppr.Spacing.After != "" {
		parsed.SpaceAfter = parseTwips(ppr.Spacing.After)
	}
	if ppr.Indent.Left != "" {
		parsed.IndentLeft = parseTwips(ppr.Indent.Left)
	}
	if ppr.Indent.Right != "" {
		parsed.IndentRight = parseTwips(ppr.Indent.Right)
	}
	if ppr.Indent.FirstLine != "" {
		parsed.IndentFirst = parseTwips(ppr.Indent.FirstLine)
	}
	if ppr.Indent.Hanging != "" {
		parsed.IndentFirst = -parseTwips(ppr.Indent.Hanging)
	}

	// Check outline level for heading detection (direct formatting)
	if !parsed.IsHeading && ppr.OutlineLvl.Val != "" {
		level := parseOutlineLevel(ppr.OutlineLvl.Val)
		if level >= 0 && level <= 8 {
			parsed.IsHeading = true
			parsed.Level = level + 1
		}
	}

	// Extract text from runs with resolved formatting
	var textParts []string
	for _, run := range p.Runs {
		runText := r.extractRunText(run)
		if runText != "" {
			textParts = append(textParts, runText)

			// Resolve run properties
			var resolvedRun *ResolvedRun
			if r.styleResolver != nil {
				resolvedRun = r.styleResolver.ResolveRun(parsed.StyleID, run.Properties)
			}

			pr := parsedRun{
				Text: runText,
			}

			if resolvedRun != nil {
				pr.FontName = resolvedRun.FontName
				pr.FontSize = resolvedRun.FontSize
				pr.Bold = resolvedRun.Bold
				pr.Italic = resolvedRun.Italic
				pr.Underline = resolvedRun.Underline
				pr.Strike = resolvedRun.Strike
				pr.Color = resolvedRun.Color
				pr.Highlight = resolvedRun.Highlight
			} else {
				// Fallback: direct property check
				pr.Bold = run.Properties.Bold.XMLName.Local != "" && run.Properties.Bold.Val != "false"
				pr.Italic = run.Properties.Italic.XMLName.Local != "" && run.Properties.Italic.Val != "false"
			}

			parsed.Runs = append(parsed.Runs, pr)
		}
	}
	parsed.Text = strings.Join(textParts, "")

	// Legacy fallback for heading detection if no style resolver
	if r.styleResolver == nil && parsed.StyleID != "" {
		parsed.IsHeading, parsed.Level = r.isHeadingStyle(parsed.StyleID)
	}

	return parsed
}

// extractRunText extracts text from a run element.
func (r *Reader) extractRunText(run runXML) string {
	var parts []string

	for _, t := range run.Text {
		parts = append(parts, t.Value)
	}

	// Handle tab characters
	for range run.Tabs {
		parts = append(parts, "\t")
	}

	// Handle breaks
	for _, br := range run.Breaks {
		if br.Type == "page" {
			parts = append(parts, "\n\n")
		} else {
			parts = append(parts, "\n")
		}
	}

	return strings.Join(parts, "")
}

// isHeadingStyle determines if a style ID represents a heading.
func (r *Reader) isHeadingStyle(styleID string) (bool, int) {
	// Check for built-in heading styles
	styleID = strings.ToLower(styleID)

	// Standard Word heading style IDs
	headingMap := map[string]int{
		"heading1": 1, "heading2": 2, "heading3": 3,
		"heading4": 4, "heading5": 5, "heading6": 6,
		"heading7": 7, "heading8": 8, "heading9": 9,
		"title":    1, // Title is typically H1 equivalent
	}

	if level, ok := headingMap[styleID]; ok {
		return true, level
	}

	// Check style definitions for outline level
	if r.styles != nil {
		for _, style := range r.styles.Styles {
			if strings.EqualFold(style.StyleID, styleID) {
				if style.PPr.OutlineLvl.Val != "" {
					// OutlineLvl is 0-based in OOXML
					if level := parseOutlineLevel(style.PPr.OutlineLvl.Val); level >= 0 {
						return true, level + 1
					}
				}
				// Check if style name contains "heading"
				if strings.Contains(strings.ToLower(style.Name.Val), "heading") {
					return true, 1 // Default to H1 if we can't determine level
				}
			}
		}
	}

	return false, 0
}

// parseOutlineLevel parses an outline level string to an integer.
func parseOutlineLevel(s string) int {
	level := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			level = level*10 + int(c-'0')
		}
	}
	if level >= 0 && level <= 8 {
		return level
	}
	return -1
}
