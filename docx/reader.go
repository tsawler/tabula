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
	file       *os.File
	zipReader  *zip.ReadCloser
	document   *documentXML
	styles     *stylesXML
	numbering  *numberingXML
	rels       *relationshipsXML
	coreProps  *corePropertiesXML
	appProps   *appPropertiesXML
	paragraphs []parsedParagraph
}

// parsedParagraph holds a parsed paragraph with resolved styles.
type parsedParagraph struct {
	Text      string
	StyleID   string
	StyleName string
	IsHeading bool
	Level     int // heading level (1-9) or 0 for non-headings
	Runs      []parsedRun
}

// parsedRun holds a parsed text run.
type parsedRun struct {
	Text   string
	Bold   bool
	Italic bool
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

	// Parse document.xml
	if err := r.parseDocument(); err != nil {
		zr.Close()
		return nil, fmt.Errorf("parsing document: %w", err)
	}

	// Parse styles.xml (optional but usually present)
	if err := r.parseStyles(); err != nil {
		// Styles are optional - just continue without them
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
func (r *Reader) Text() (string, error) {
	if r.document == nil {
		return "", fmt.Errorf("document not parsed")
	}

	var result strings.Builder
	for i, para := range r.paragraphs {
		if i > 0 {
			result.WriteString("\n")
			if para.IsHeading {
				result.WriteString("\n") // Extra blank line before headings
			}
		}
		result.WriteString(para.Text)
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

// parseStyles parses the styles definition file.
func (r *Reader) parseStyles() error {
	data, err := r.getFileContent("word/styles.xml")
	if err != nil {
		return err
	}

	r.styles = &stylesXML{}
	return xml.Unmarshal(data, r.styles)
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
}

// processParagraph processes a single paragraph.
func (r *Reader) processParagraph(p paragraphXML) parsedParagraph {
	parsed := parsedParagraph{
		StyleID: p.Properties.Style.Val,
	}

	// Extract text from runs
	var textParts []string
	for _, run := range p.Runs {
		runText := r.extractRunText(run)
		if runText != "" {
			textParts = append(textParts, runText)
			parsed.Runs = append(parsed.Runs, parsedRun{
				Text:   runText,
				Bold:   run.Properties.Bold.Val != "false" && run.Properties.Bold.XMLName.Local != "",
				Italic: run.Properties.Italic.Val != "false" && run.Properties.Italic.XMLName.Local != "",
			})
		}
	}
	parsed.Text = strings.Join(textParts, "")

	// Detect heading from style
	if parsed.StyleID != "" {
		parsed.IsHeading, parsed.Level = r.isHeadingStyle(parsed.StyleID)
		if r.styles != nil {
			for _, style := range r.styles.Styles {
				if style.StyleID == parsed.StyleID {
					parsed.StyleName = style.Name.Val
					break
				}
			}
		}
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
