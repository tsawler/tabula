// Package pptx provides PPTX (Office Open XML Presentation) document parsing.
package pptx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/rag"
)

// Reader provides access to PPTX document content.
type Reader struct {
	zipReader    *zip.ReadCloser
	presentation *presentationXML
	slides       []*Slide
	slideRels    map[int]*relationshipsXML // Slide index -> relationships
	coreProps    *corePropertiesXML
	appProps     *appPropertiesXML
	presRels     *relationshipsXML
}

// Open opens a PPTX file for reading.
func Open(filename string) (*Reader, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, fmt.Errorf("opening ZIP archive: %w", err)
	}

	r := &Reader{
		zipReader: zr,
		slideRels: make(map[int]*relationshipsXML),
	}

	// Validate required files exist
	if err := r.validate(); err != nil {
		zr.Close()
		return nil, err
	}

	// Parse presentation relationships first
	if err := r.parseRelationships(); err != nil {
		zr.Close()
		return nil, fmt.Errorf("parsing relationships: %w", err)
	}

	// Parse presentation to get slide order
	if err := r.parsePresentation(); err != nil {
		zr.Close()
		return nil, fmt.Errorf("parsing presentation: %w", err)
	}

	// Parse all slides
	if err := r.parseSlides(); err != nil {
		zr.Close()
		return nil, fmt.Errorf("parsing slides: %w", err)
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

// validate checks that required PPTX files exist.
func (r *Reader) validate() error {
	required := []string{
		"[Content_Types].xml",
		"ppt/presentation.xml",
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

	// Check for at least one slide
	hasSlide := false
	for name := range fileMap {
		if strings.HasPrefix(name, "ppt/slides/slide") && strings.HasSuffix(name, ".xml") {
			hasSlide = true
			break
		}
	}
	if !hasSlide {
		return fmt.Errorf("no slides found in presentation")
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

// parseRelationships parses the presentation relationships file.
func (r *Reader) parseRelationships() error {
	data, err := r.getFileContent("ppt/_rels/presentation.xml.rels")
	if err != nil {
		return nil // Relationships might be optional
	}

	r.presRels = &relationshipsXML{}
	return xml.Unmarshal(data, r.presRels)
}

// parsePresentation parses the main presentation file.
func (r *Reader) parsePresentation() error {
	data, err := r.getFileContent("ppt/presentation.xml")
	if err != nil {
		return err
	}

	r.presentation = &presentationXML{}
	return xml.Unmarshal(data, r.presentation)
}

// parseSlides parses all slide files.
func (r *Reader) parseSlides() error {
	// Find all slide files
	slideFiles := make([]string, 0)
	for _, f := range r.zipReader.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			// Exclude relationship files
			if !strings.Contains(f.Name, "_rels") {
				slideFiles = append(slideFiles, f.Name)
			}
		}
	}

	// Sort slides by number
	sort.Slice(slideFiles, func(i, j int) bool {
		return extractSlideNumber(slideFiles[i]) < extractSlideNumber(slideFiles[j])
	})

	r.slides = make([]*Slide, 0, len(slideFiles))

	for i, slidePath := range slideFiles {
		slide, err := r.parseSlide(slidePath, i)
		if err != nil {
			continue // Skip slides that fail to parse
		}

		// Parse slide relationships for notes
		r.parseSlideRelationships(slidePath, i)

		// Parse speaker notes if available
		r.parseSlideNotes(i, slide)

		r.slides = append(r.slides, slide)
	}

	if len(r.slides) == 0 {
		return fmt.Errorf("no slides could be parsed")
	}

	return nil
}

// extractSlideNumber extracts the slide number from a path like "ppt/slides/slide1.xml"
func extractSlideNumber(path string) int {
	// Extract number from slideN.xml
	name := strings.TrimPrefix(path, "ppt/slides/slide")
	name = strings.TrimSuffix(name, ".xml")
	var num int
	fmt.Sscanf(name, "%d", &num)
	return num
}

// parseSlide parses a single slide file.
func (r *Reader) parseSlide(slidePath string, index int) (*Slide, error) {
	data, err := r.getFileContent(slidePath)
	if err != nil {
		return nil, err
	}

	var slideXML slideXML
	if err := xml.Unmarshal(data, &slideXML); err != nil {
		return nil, err
	}

	slide := &Slide{
		Index:   index,
		Content: make([]TextBlock, 0),
		Tables:  make([]Table, 0),
	}

	// Extract shapes from shape tree
	r.extractShapes(&slideXML.CSld.SpTree, slide)

	return slide, nil
}

// extractShapes extracts text content from all shapes in the shape tree.
func (r *Reader) extractShapes(spTree *spTreeXML, slide *Slide) {
	// Process regular shapes
	for _, sp := range spTree.Sp {
		block := r.extractTextBlock(&sp)
		if block != nil {
			if block.IsTitle && slide.Title == "" {
				slide.Title = block.Text
			}
			slide.Content = append(slide.Content, *block)
		}
	}

	// Process graphic frames (tables)
	for _, gf := range spTree.GraphicFrame {
		if gf.Graphic.GraphicData.Tbl != nil {
			table := r.extractTable(gf.Graphic.GraphicData.Tbl)
			slide.Tables = append(slide.Tables, table)
		}
	}

	// Process grouped shapes (recursive)
	for _, grpSp := range spTree.GrpSp {
		r.extractGroupedShapes(&grpSp, slide)
	}
}

// extractGroupedShapes extracts shapes from a group.
func (r *Reader) extractGroupedShapes(grpSp *grpSpXML, slide *Slide) {
	for _, sp := range grpSp.Sp {
		block := r.extractTextBlock(&sp)
		if block != nil {
			slide.Content = append(slide.Content, *block)
		}
	}

	// Recursively process nested groups
	for _, nestedGrp := range grpSp.GrpSp {
		r.extractGroupedShapes(&nestedGrp, slide)
	}
}

// extractTextBlock extracts text from a shape.
func (r *Reader) extractTextBlock(sp *spXML) *TextBlock {
	if sp.TxBody == nil || len(sp.TxBody.P) == 0 {
		return nil
	}

	block := &TextBlock{
		Paragraphs: make([]Paragraph, 0),
	}

	// Check if this is a title placeholder
	if sp.NvSpPr.NvPr.Ph != nil {
		phType := sp.NvSpPr.NvPr.Ph.Type
		block.Placeholder = phType
		block.IsTitle = phType == "title" || phType == "ctrTitle"
		block.IsSubtitle = phType == "subTitle"
	}

	// Get position if available
	if sp.SpPr.Xfrm != nil {
		block.X = sp.SpPr.Xfrm.Off.X
		block.Y = sp.SpPr.Xfrm.Off.Y
		block.Width = sp.SpPr.Xfrm.Ext.Cx
		block.Height = sp.SpPr.Xfrm.Ext.Cy
	}

	// Extract paragraphs
	var allText strings.Builder
	for _, p := range sp.TxBody.P {
		para := r.extractParagraph(&p)
		if para.Text != "" {
			block.Paragraphs = append(block.Paragraphs, para)
			if allText.Len() > 0 {
				allText.WriteString("\n")
			}
			allText.WriteString(para.Text)
		}
	}

	block.Text = allText.String()

	if block.Text == "" {
		return nil
	}

	return block
}

// extractParagraph extracts text and formatting from a paragraph.
func (r *Reader) extractParagraph(p *pXML) Paragraph {
	para := Paragraph{
		Runs: make([]Run, 0),
	}

	// Get paragraph properties
	if p.PPr != nil {
		para.Level = p.PPr.Lvl
		para.Alignment = p.PPr.Algn

		// Check for bullets
		if p.PPr.BuNone == nil {
			// Has some kind of bullet unless explicitly none
			if p.PPr.BuAutoNum != nil {
				para.IsNumbered = true
			} else if p.PPr.BuChar != nil {
				para.IsBullet = true
				para.BulletChar = p.PPr.BuChar.Char
			} else if para.Level > 0 {
				// Default to bullet for indented items
				para.IsBullet = true
			}
		}
	}

	// Extract text from runs
	var text strings.Builder
	for _, run := range p.R {
		text.WriteString(run.T)

		runObj := Run{
			Text: run.T,
		}
		if run.RPr != nil {
			if run.RPr.B != nil && *run.RPr.B == 1 {
				runObj.Bold = true
			}
			if run.RPr.I != nil && *run.RPr.I == 1 {
				runObj.Italic = true
			}
			runObj.FontSize = run.RPr.Sz
		}
		para.Runs = append(para.Runs, runObj)
	}

	// Include field values (like slide numbers)
	for _, fld := range p.Fld {
		text.WriteString(fld.T)
	}

	para.Text = strings.TrimSpace(text.String())
	return para
}

// extractTable extracts a table from a graphic frame.
func (r *Reader) extractTable(tbl *tblXML) Table {
	table := Table{
		Columns: len(tbl.TblGrid.GridCol),
		Rows:    make([][]TableCell, 0, len(tbl.Tr)),
	}

	for _, tr := range tbl.Tr {
		row := make([]TableCell, 0, len(tr.Tc))
		for _, tc := range tr.Tc {
			cell := TableCell{
				RowSpan: tc.RowSpan,
				ColSpan: tc.GridSpan,
			}
			if cell.RowSpan == 0 {
				cell.RowSpan = 1
			}
			if cell.ColSpan == 0 {
				cell.ColSpan = 1
			}

			// Check if this is a merged cell (not the origin)
			if tc.VMerge != nil || tc.HMerge != nil {
				cell.IsMerged = true
			}

			// Extract text from cell
			if tc.TxBody != nil {
				var text strings.Builder
				for _, p := range tc.TxBody.P {
					para := r.extractParagraph(&p)
					if para.Text != "" {
						if text.Len() > 0 {
							text.WriteString(" ")
						}
						text.WriteString(para.Text)
					}
				}
				cell.Text = text.String()
			}

			row = append(row, cell)
		}
		table.Rows = append(table.Rows, row)
	}

	return table
}

// parseSlideRelationships parses the relationships for a slide.
func (r *Reader) parseSlideRelationships(slidePath string, index int) {
	// Construct the rels path
	dir := path.Dir(slidePath)
	base := path.Base(slidePath)
	relsPath := path.Join(dir, "_rels", base+".rels")

	data, err := r.getFileContent(relsPath)
	if err != nil {
		return // Relationships are optional
	}

	rels := &relationshipsXML{}
	if err := xml.Unmarshal(data, rels); err != nil {
		return
	}

	r.slideRels[index] = rels
}

// parseSlideNotes parses speaker notes for a slide.
func (r *Reader) parseSlideNotes(index int, slide *Slide) {
	rels := r.slideRels[index]
	if rels == nil {
		return
	}

	// Find notes relationship
	var notesPath string
	for _, rel := range rels.Relationship {
		if strings.Contains(rel.Type, "notesSlide") {
			notesPath = rel.Target
			break
		}
	}

	if notesPath == "" {
		return
	}

	// Normalize path
	if strings.HasPrefix(notesPath, "../") {
		notesPath = "ppt/" + strings.TrimPrefix(notesPath, "../")
	} else if !strings.HasPrefix(notesPath, "ppt/") {
		notesPath = "ppt/slides/" + notesPath
	}

	data, err := r.getFileContent(notesPath)
	if err != nil {
		return
	}

	var notes notesSlideXML
	if err := xml.Unmarshal(data, &notes); err != nil {
		return
	}

	// Extract text from notes
	var text strings.Builder
	for _, sp := range notes.CSld.SpTree.Sp {
		// Skip the slide image placeholder
		if sp.NvSpPr.NvPr.Ph != nil && sp.NvSpPr.NvPr.Ph.Type == "sldImg" {
			continue
		}

		if sp.TxBody != nil {
			for _, p := range sp.TxBody.P {
				para := r.extractParagraph(&p)
				if para.Text != "" {
					if text.Len() > 0 {
						text.WriteString("\n")
					}
					text.WriteString(para.Text)
				}
			}
		}
	}

	slide.Notes = strings.TrimSpace(text.String())
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

// SlideCount returns the number of slides.
func (r *Reader) SlideCount() int {
	return len(r.slides)
}

// Slide returns the slide at the given index (0-indexed).
func (r *Reader) Slide(index int) (*Slide, error) {
	if index < 0 || index >= len(r.slides) {
		return nil, fmt.Errorf("slide index %d out of range (0-%d)", index, len(r.slides)-1)
	}
	return r.slides[index], nil
}

// PageCount returns the number of slides (alias for SlideCount).
func (r *Reader) PageCount() (int, error) {
	return len(r.slides), nil
}

// ExtractOptions holds options for text extraction.
type ExtractOptions struct {
	IncludeNotes   bool  // Include speaker notes
	IncludeTitles  bool  // Include slide titles (default: true)
	SlideNumbers   []int // Which slides to include (0-indexed, empty = all)
	ExcludeHeaders bool  // Exclude header placeholders
	ExcludeFooters bool  // Exclude footer placeholders (footer, date, slide number)
}

// isFooterPlaceholder returns true if the placeholder type is a footer element.
// Footer elements include: ftr (footer), dt (date/time), sldNum (slide number).
func isFooterPlaceholder(phType string) bool {
	switch phType {
	case "ftr", "dt", "sldNum":
		return true
	}
	return false
}

// isHeaderPlaceholder returns true if the placeholder type is a header element.
func isHeaderPlaceholder(phType string) bool {
	switch phType {
	case "hdr":
		return true
	}
	return false
}

// Text extracts and returns all text content from the presentation.
func (r *Reader) Text() (string, error) {
	return r.TextWithOptions(ExtractOptions{IncludeTitles: true})
}

// TextWithOptions extracts text content with the specified options.
func (r *Reader) TextWithOptions(opts ExtractOptions) (string, error) {
	slides := r.slides
	if len(opts.SlideNumbers) > 0 {
		slides = make([]*Slide, 0, len(opts.SlideNumbers))
		for _, idx := range opts.SlideNumbers {
			if idx >= 0 && idx < len(r.slides) {
				slides = append(slides, r.slides[idx])
			}
		}
	}

	var result strings.Builder

	for i, slide := range slides {
		if i > 0 {
			result.WriteString("\n\n")
		}

		// Slide title
		if opts.IncludeTitles && slide.Title != "" {
			result.WriteString(slide.Title)
			result.WriteString("\n\n")
		}

		// Content
		for _, block := range slide.Content {
			if block.IsTitle && opts.IncludeTitles {
				continue // Already added
			}
			// Skip header/footer placeholders if requested
			if opts.ExcludeFooters && isFooterPlaceholder(block.Placeholder) {
				continue
			}
			if opts.ExcludeHeaders && isHeaderPlaceholder(block.Placeholder) {
				continue
			}
			for _, para := range block.Paragraphs {
				if para.Text != "" {
					// Add bullet/number prefix
					if para.IsBullet || para.IsNumbered {
						for j := 0; j < para.Level; j++ {
							result.WriteString("  ")
						}
						result.WriteString("• ")
					}
					result.WriteString(para.Text)
					result.WriteString("\n")
				}
			}
		}

		// Tables
		for _, table := range slide.Tables {
			result.WriteString("\n")
			for _, row := range table.Rows {
				for j, cell := range row {
					if j > 0 {
						result.WriteString("\t")
					}
					result.WriteString(cell.Text)
				}
				result.WriteString("\n")
			}
		}

		// Notes
		if opts.IncludeNotes && slide.Notes != "" {
			result.WriteString("\n[Notes: ")
			result.WriteString(slide.Notes)
			result.WriteString("]\n")
		}
	}

	return result.String(), nil
}

// Markdown returns the presentation content as Markdown.
func (r *Reader) Markdown() (string, error) {
	return r.MarkdownWithOptions(ExtractOptions{IncludeTitles: true})
}

// MarkdownWithOptions returns presentation content as Markdown with options.
func (r *Reader) MarkdownWithOptions(opts ExtractOptions) (string, error) {
	slides := r.slides
	if len(opts.SlideNumbers) > 0 {
		slides = make([]*Slide, 0, len(opts.SlideNumbers))
		for _, idx := range opts.SlideNumbers {
			if idx >= 0 && idx < len(r.slides) {
				slides = append(slides, r.slides[idx])
			}
		}
	}

	var result strings.Builder

	for i, slide := range slides {
		if i > 0 {
			result.WriteString("\n---\n\n")
		}

		// Slide title as H1
		if slide.Title != "" {
			result.WriteString("# ")
			result.WriteString(slide.Title)
			result.WriteString("\n\n")
		}

		// Content
		for _, block := range slide.Content {
			if block.IsTitle {
				continue // Already added
			}
			// Skip header/footer placeholders if requested
			if opts.ExcludeFooters && isFooterPlaceholder(block.Placeholder) {
				continue
			}
			if opts.ExcludeHeaders && isHeaderPlaceholder(block.Placeholder) {
				continue
			}

			for _, para := range block.Paragraphs {
				if para.Text == "" {
					continue
				}

				if para.IsBullet || para.IsNumbered {
					// Indentation for nested bullets
					for j := 0; j < para.Level; j++ {
						result.WriteString("  ")
					}
					if para.IsNumbered {
						result.WriteString("1. ")
					} else {
						result.WriteString("- ")
					}
					result.WriteString(para.Text)
					result.WriteString("\n")
				} else {
					result.WriteString(para.Text)
					result.WriteString("\n\n")
				}
			}
		}

		// Tables
		for _, table := range slide.Tables {
			result.WriteString("\n")
			result.WriteString(table.ToMarkdown())
		}

		// Notes as blockquote
		if opts.IncludeNotes && slide.Notes != "" {
			result.WriteString("\n> **Notes:** ")
			// Replace newlines with blockquote continuation
			notes := strings.ReplaceAll(slide.Notes, "\n", "\n> ")
			result.WriteString(notes)
			result.WriteString("\n")
		}
	}

	return strings.TrimSpace(result.String()), nil
}

// MarkdownWithRAGOptions returns presentation content as Markdown with RAG options.
func (r *Reader) MarkdownWithRAGOptions(extractOpts ExtractOptions, mdOpts rag.MarkdownOptions) (string, error) {
	var result strings.Builder

	// Add YAML front matter metadata if requested
	if mdOpts.IncludeMetadata {
		meta := r.Metadata()
		result.WriteString("---\n")
		if meta.Title != "" {
			result.WriteString(fmt.Sprintf("title: %q\n", meta.Title))
		}
		if meta.Author != "" {
			result.WriteString(fmt.Sprintf("author: %q\n", meta.Author))
		}
		if meta.Subject != "" {
			result.WriteString(fmt.Sprintf("subject: %q\n", meta.Subject))
		}
		if meta.Creator != "" {
			result.WriteString(fmt.Sprintf("generator: %q\n", meta.Creator))
		}
		result.WriteString(fmt.Sprintf("slides: %d\n", len(r.slides)))
		result.WriteString("---\n\n")
	}

	// Add table of contents if requested
	if mdOpts.IncludeTableOfContents && len(r.slides) > 1 {
		result.WriteString("## Table of Contents\n\n")
		for i, slide := range r.slides {
			title := slide.Title
			if title == "" {
				title = fmt.Sprintf("Slide %d", i+1)
			}
			anchor := strings.ToLower(strings.ReplaceAll(title, " ", "-"))
			result.WriteString(fmt.Sprintf("%d. [%s](#%s)\n", i+1, title, anchor))
		}
		result.WriteString("\n---\n\n")
	}

	// Generate main content
	md, err := r.MarkdownWithOptions(extractOpts)
	if err != nil {
		return "", err
	}
	result.WriteString(md)

	return result.String(), nil
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

// Document returns a model.Document representation of the PPTX content.
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

	// Each slide becomes a page
	for _, slide := range r.slides {
		// Use standard slide dimensions (in points, approximating 10" x 7.5")
		page := model.NewPage(720, 540)
		page.Number = slide.Index + 1

		// Add title as heading
		if slide.Title != "" {
			heading := &model.Heading{
				Level: 1,
				Text:  slide.Title,
				BBox:  model.BBox{X: 36, Y: 500, Width: 648, Height: 40},
			}
			page.AddElement(heading)
		}

		// Add content blocks
		yPos := 450.0
		for _, block := range slide.Content {
			if block.IsTitle {
				continue // Already added
			}

			// Check if this block has bullet points
			hasBullets := false
			for _, para := range block.Paragraphs {
				if para.IsBullet || para.IsNumbered {
					hasBullets = true
					break
				}
			}

			if hasBullets {
				// Create a list element
				list := &model.List{
					Ordered: false,
					BBox:    model.BBox{X: 36, Y: yPos, Width: 648, Height: 20},
				}
				for _, para := range block.Paragraphs {
					if para.Text != "" {
						list.Items = append(list.Items, model.ListItem{
							Text:  para.Text,
							Level: para.Level,
						})
						if para.IsNumbered {
							list.Ordered = true
						}
					}
				}
				if len(list.Items) > 0 {
					page.AddElement(list)
					yPos -= 20 * float64(len(list.Items))
				}
			} else {
				// Create paragraph elements
				for _, para := range block.Paragraphs {
					if para.Text != "" {
						p := &model.Paragraph{
							Text: para.Text,
							BBox: model.BBox{X: 36, Y: yPos, Width: 648, Height: 20},
						}
						page.AddElement(p)
						yPos -= 25
					}
				}
			}
		}

		// Add tables
		for _, table := range slide.Tables {
			numRows := len(table.Rows)
			numCols := 0
			if numRows > 0 {
				numCols = len(table.Rows[0])
			}

			modelTable := model.NewTable(numRows, numCols)
			modelTable.BBox = model.BBox{X: 36, Y: yPos, Width: 648, Height: float64(numRows * 20)}

			for i, row := range table.Rows {
				for j, cell := range row {
					if j < len(modelTable.Rows[i]) {
						modelTable.Rows[i][j] = model.Cell{
							Text:    cell.Text,
							RowSpan: cell.RowSpan,
							ColSpan: cell.ColSpan,
						}
						if i == 0 {
							modelTable.Rows[i][j].IsHeader = true
						}
					}
				}
			}

			page.AddElement(modelTable)
			yPos -= float64(numRows*20 + 10)
		}

		doc.AddPage(page)
	}

	return doc, nil
}
