// Package odt provides ODT (OpenDocument Text) document parsing.
package odt

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/tsawler/tabula/model"
)

// Reader provides access to ODT document content.
type Reader struct {
	zipReader     *zip.ReadCloser
	content       *documentXML
	contentStyles *contentStylesXML
	docStyles     *stylesXML
	meta          *metaXML
	styleResolver *StyleResolver
	tableParser   *TableParser
	listParser    *ListParser
	paragraphs    []parsedParagraph
	tables        []ParsedTable
	lists         []ParsedList
	elements      []parsedElement // Elements in document order
}

// Open opens an ODT file for reading.
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

	// Parse styles.xml first (optional but usually present)
	_ = r.parseStyles()

	// Parse content.xml automatic styles
	_ = r.parseContentStyles()

	// Create style resolver with both document styles and content automatic styles
	r.styleResolver = NewStyleResolver(r.contentStyles, r.docStyles)

	// Create parsers
	r.tableParser = NewTableParser(r.styleResolver)
	r.listParser = NewListParser(r.styleResolver)

	// Parse content.xml (main document content)
	if err := r.parseContent(); err != nil {
		zr.Close()
		return nil, fmt.Errorf("parsing content: %w", err)
	}

	// Parse metadata (optional)
	r.parseMetadata()

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

// validate checks that required ODT files exist.
func (r *Reader) validate() error {
	required := []string{
		"content.xml",
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

// PageCount returns the number of "pages" in the document.
// Since ODT doesn't have fixed pages, we return 1 (entire document as single page).
func (r *Reader) PageCount() (int, error) {
	return 1, nil
}

// Text extracts and returns all text content from the document.
func (r *Reader) Text() (string, error) {
	if len(r.elements) == 0 && len(r.paragraphs) == 0 {
		return "", nil
	}

	var result strings.Builder
	listCounters := make(map[string]map[int]int)

	// Use elements if available for correct document order
	if len(r.elements) > 0 {
		for i, elem := range r.elements {
			if i > 0 {
				result.WriteString("\n")
			}

			switch elem.Type {
			case "paragraph":
				if elem.Paragraph != nil {
					r.writeParagraphText(&result, elem.Paragraph, listCounters)
				}
			case "table":
				if elem.Table != nil {
					result.WriteString(elem.Table.ToText())
				}
			}
		}
		return result.String(), nil
	}

	// Fallback: output paragraphs then tables
	for i, para := range r.paragraphs {
		if i > 0 {
			result.WriteString("\n")
		}
		r.writeParagraphText(&result, &para, listCounters)
	}

	for _, tbl := range r.tables {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(tbl.ToText())
	}

	return result.String(), nil
}

// Markdown returns the document content as a Markdown-formatted string.
func (r *Reader) Markdown() (string, error) {
	if len(r.elements) == 0 && len(r.paragraphs) == 0 {
		return "", nil
	}

	var result strings.Builder
	var inList bool
	listCounters := make(map[string]map[int]int)

	for i, elem := range r.elements {
		switch elem.Type {
		case "paragraph":
			para := elem.Paragraph
			if para == nil {
				continue
			}

			// Add separator between elements (except first)
			if i > 0 && result.Len() > 0 {
				if inList && !para.IsListItem {
					result.WriteString("\n")
					inList = false
				}
			}

			if para.IsHeading {
				// Render as markdown heading
				level := para.Level
				if level < 1 {
					level = 1
				}
				if level > 6 {
					level = 6
				}
				result.WriteString(strings.Repeat("#", level))
				result.WriteString(" ")
				result.WriteString(para.Text)
				result.WriteString("\n\n")
				inList = false
			} else if para.IsListItem {
				// Render as markdown list item
				r.writeMarkdownListItem(&result, para, listCounters)
				inList = true
			} else if para.Text != "" {
				// Regular paragraph
				result.WriteString(para.Text)
				result.WriteString("\n\n")
				inList = false
			}

		case "table":
			if inList {
				result.WriteString("\n")
				inList = false
			}
			if elem.Table != nil {
				result.WriteString(elem.Table.ToMarkdown())
				result.WriteString("\n")
			}
		}
	}

	return strings.TrimSpace(result.String()), nil
}

// writeMarkdownListItem writes a list item in markdown format.
func (r *Reader) writeMarkdownListItem(sb *strings.Builder, para *parsedParagraph, listCounters map[string]map[int]int) {
	// Add indentation for nested lists (2 spaces per level)
	for j := 0; j < para.ListLevel; j++ {
		sb.WriteString("  ")
	}

	// Determine if ordered or unordered from style
	isOrdered := false
	if r.styleResolver != nil && para.StyleName != "" {
		ll := r.styleResolver.ResolveListLevel(para.StyleName, para.ListLevel)
		isOrdered = !ll.IsBullet
	}

	if isOrdered {
		// Initialize counter for this level if needed
		key := para.StyleName
		if listCounters[key] == nil {
			listCounters[key] = make(map[int]int)
		}
		listCounters[key][para.ListLevel]++
		num := listCounters[key][para.ListLevel]
		sb.WriteString(fmt.Sprintf("%d. ", num))
	} else {
		sb.WriteString("- ")
	}

	sb.WriteString(para.Text)
	sb.WriteString("\n")
}

// writeParagraphText writes a paragraph's text to the builder with list formatting.
func (r *Reader) writeParagraphText(sb *strings.Builder, para *parsedParagraph, listCounters map[string]map[int]int) {
	if para.IsListItem {
		// Add indentation for nested lists
		for j := 0; j < para.ListLevel; j++ {
			sb.WriteString("  ")
		}

		// Determine if ordered or unordered
		isOrdered := false
		bullet := "•"
		if r.styleResolver != nil && para.StyleName != "" {
			ll := r.styleResolver.ResolveListLevel(para.StyleName, para.ListLevel)
			isOrdered = !ll.IsBullet
			if ll.BulletChar != "" {
				bullet = ll.BulletChar
			}
		}

		if isOrdered {
			key := para.StyleName
			if listCounters[key] == nil {
				listCounters[key] = make(map[int]int)
			}
			listCounters[key][para.ListLevel]++
			num := listCounters[key][para.ListLevel]
			sb.WriteString(fmt.Sprintf("%d. ", num))
		} else {
			sb.WriteString(bullet)
			sb.WriteString(" ")
		}
	}

	sb.WriteString(para.Text)
}

// Document returns a model.Document representation of the ODT content.
func (r *Reader) Document() (*model.Document, error) {
	doc := model.NewDocument()

	// Set metadata
	if r.meta != nil && r.meta.Meta != nil {
		doc.Metadata.Title = r.meta.Meta.Title
		doc.Metadata.Author = r.meta.Meta.Creator
		if doc.Metadata.Author == "" {
			doc.Metadata.Author = r.meta.Meta.InitialCreator
		}
		doc.Metadata.Subject = r.meta.Meta.Subject
		doc.Metadata.Creator = r.meta.Meta.Generator
	}

	// Create single page for entire document (ODT doesn't have fixed pages)
	page := model.NewPage(612, 792) // Standard US Letter dimensions
	page.Number = 1

	// Process elements in document order
	yPos := 750.0 // Start near top of page

	// Track current list being built
	var currentList *model.List
	var currentListStartY float64

	finalizeList := func() {
		if currentList != nil && len(currentList.Items) > 0 {
			listHeight := float64(len(currentList.Items)) * 14.0
			currentList.BBox = model.BBox{
				X:      72,
				Y:      currentListStartY,
				Width:  468,
				Height: listHeight,
			}
			page.AddElement(currentList)
			yPos -= listHeight + 4
		}
		currentList = nil
	}

	for _, elem := range r.elements {
		switch elem.Type {
		case "paragraph":
			para := elem.Paragraph
			if para == nil || para.Text == "" {
				continue
			}

			// Check if this is a list item
			if para.IsListItem {
				if currentList == nil {
					isOrdered := false
					if r.styleResolver != nil && para.StyleName != "" {
						ll := r.styleResolver.ResolveListLevel(para.StyleName, para.ListLevel)
						isOrdered = !ll.IsBullet
					}
					currentList = &model.List{
						Ordered: isOrdered,
					}
					currentListStartY = yPos
				}

				bullet := "•"
				if r.styleResolver != nil && para.StyleName != "" {
					ll := r.styleResolver.ResolveListLevel(para.StyleName, para.ListLevel)
					if ll.BulletChar != "" {
						bullet = ll.BulletChar
					}
				}

				currentList.Items = append(currentList.Items, model.ListItem{
					Text:   para.Text,
					Level:  para.ListLevel,
					Bullet: bullet,
				})
				continue
			}

			// Not a list item - finalize any pending list
			finalizeList()

			estimatedHeight := 14.0
			if para.IsHeading {
				estimatedHeight = float64(24 - para.Level*2)
			}

			bbox := model.BBox{
				X:      72,
				Y:      yPos,
				Width:  468,
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

			yPos -= estimatedHeight + 4

		case "table":
			finalizeList()

			if elem.Table == nil {
				continue
			}

			modelTable := elem.Table.ToModelTable()
			if modelTable.RowCount() > 0 {
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
	}

	// Finalize any remaining list
	finalizeList()

	doc.AddPage(page)
	return doc, nil
}

// Metadata returns document metadata.
func (r *Reader) Metadata() model.Metadata {
	meta := model.Metadata{}
	if r.meta != nil && r.meta.Meta != nil {
		meta.Title = r.meta.Meta.Title
		meta.Author = r.meta.Meta.Creator
		if meta.Author == "" {
			meta.Author = r.meta.Meta.InitialCreator
		}
		meta.Subject = r.meta.Meta.Subject
		meta.Creator = r.meta.Meta.Generator
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

// Lists returns all parsed lists from the document.
func (r *Reader) Lists() []ParsedList {
	return r.lists
}

// parseStyles parses the styles.xml file.
func (r *Reader) parseStyles() error {
	data, err := r.getFileContent("styles.xml")
	if err != nil {
		return err
	}

	r.docStyles = &stylesXML{}
	return xml.Unmarshal(data, r.docStyles)
}

// parseContentStyles parses automatic styles from content.xml.
func (r *Reader) parseContentStyles() error {
	data, err := r.getFileContent("content.xml")
	if err != nil {
		return err
	}

	// Parse just the automatic-styles section
	type contentDoc struct {
		AutoStyles *contentStylesXML `xml:"automatic-styles"`
	}

	var doc contentDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return err
	}

	r.contentStyles = doc.AutoStyles
	return nil
}

// parseContent parses the content.xml file.
func (r *Reader) parseContent() error {
	data, err := r.getFileContent("content.xml")
	if err != nil {
		return err
	}

	// Parse body elements in order using streaming decoder
	return r.parseBodyElements(data)
}

// parseBodyElements parses body elements maintaining document order.
func (r *Reader) parseBodyElements(data []byte) error {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var inBody bool
	var currentListStyle string
	var currentListLevel int

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Check if we're entering the text body
			if t.Name.Local == "text" && t.Name.Space == nsOffice {
				inBody = true
				continue
			}

			if !inBody {
				continue
			}

			switch t.Name.Local {
			case "p":
				// Paragraph
				var para paragraphXML
				if err := decoder.DecodeElement(&para, &t); err != nil {
					continue
				}
				parsed := r.processParagraph(para)
				r.paragraphs = append(r.paragraphs, parsed)
				r.elements = append(r.elements, parsedElement{
					Type:      "paragraph",
					Paragraph: &r.paragraphs[len(r.paragraphs)-1],
				})

			case "h":
				// Heading
				var heading headingXML
				if err := decoder.DecodeElement(&heading, &t); err != nil {
					continue
				}
				parsed := r.processHeading(heading)
				r.paragraphs = append(r.paragraphs, parsed)
				r.elements = append(r.elements, parsedElement{
					Type:      "paragraph",
					Paragraph: &r.paragraphs[len(r.paragraphs)-1],
				})

			case "list":
				// List
				var list listXML
				// Get style name from attributes
				for _, attr := range t.Attr {
					if attr.Name.Local == "style-name" {
						currentListStyle = attr.Value
						break
					}
				}
				if err := decoder.DecodeElement(&list, &t); err != nil {
					continue
				}
				list.StyleName = currentListStyle
				parsedList := r.listParser.ParseList(list, currentListLevel)
				r.lists = append(r.lists, parsedList)
				// Add list items as paragraphs with list formatting
				for _, item := range parsedList.Items {
					para := parsedParagraph{
						Text:       item.Text,
						IsListItem: true,
						ListLevel:  item.Level,
						StyleName:  currentListStyle,
					}
					r.paragraphs = append(r.paragraphs, para)
					r.elements = append(r.elements, parsedElement{
						Type:      "paragraph",
						Paragraph: &r.paragraphs[len(r.paragraphs)-1],
					})
				}

			case "table":
				// Table
				var tbl tableXML
				if err := decoder.DecodeElement(&tbl, &t); err != nil {
					continue
				}
				parsed := r.tableParser.ParseTable(tbl)
				r.tables = append(r.tables, parsed)
				r.elements = append(r.elements, parsedElement{
					Type:  "table",
					Table: &r.tables[len(r.tables)-1],
				})
			}

		case xml.EndElement:
			if t.Name.Local == "text" && t.Name.Space == nsOffice {
				inBody = false
			}
		}
	}

	return nil
}

// processParagraph processes a single paragraph.
func (r *Reader) processParagraph(p paragraphXML) parsedParagraph {
	parsed := parsedParagraph{
		StyleName: p.StyleName,
	}

	// Resolve style
	if r.styleResolver != nil {
		resolved := r.styleResolver.Resolve(p.StyleName)
		parsed.Alignment = resolved.Alignment
		parsed.SpaceBefore = resolved.SpaceBefore
		parsed.SpaceAfter = resolved.SpaceAfter
		parsed.IndentLeft = resolved.IndentLeft
		parsed.IndentRight = resolved.IndentRight
		parsed.IndentFirst = resolved.IndentFirst
	}

	// Extract text
	var textParts []string

	// Direct text content
	if p.Text != "" {
		textParts = append(textParts, p.Text)
	}

	// Text from spans
	for _, span := range p.Spans {
		if span.Text != "" {
			textParts = append(textParts, span.Text)

			// Create run for formatting
			pr := parsedRun{Text: span.Text}
			if r.styleResolver != nil {
				resolved := r.styleResolver.Resolve(span.StyleName)
				pr.FontName = resolved.FontName
				pr.FontSize = resolved.FontSize
				pr.Bold = resolved.Bold
				pr.Italic = resolved.Italic
				pr.Underline = resolved.Underline
				pr.Strike = resolved.Strike
				pr.Color = resolved.Color
			}
			parsed.Runs = append(parsed.Runs, pr)
		}
	}

	parsed.Text = strings.Join(textParts, "")

	return parsed
}

// processHeading processes a heading element.
func (r *Reader) processHeading(h headingXML) parsedParagraph {
	parsed := parsedParagraph{
		StyleName: h.StyleName,
		IsHeading: true,
		Level:     1, // Default level
	}

	// Parse outline level
	if h.OutlineLevel != "" {
		if level, err := strconv.Atoi(h.OutlineLevel); err == nil && level >= 1 && level <= 9 {
			parsed.Level = level
		}
	}

	// Resolve style
	if r.styleResolver != nil {
		resolved := r.styleResolver.Resolve(h.StyleName)
		parsed.Alignment = resolved.Alignment
		// If style has heading level, prefer that
		if resolved.IsHeading && resolved.HeadingLevel > 0 {
			parsed.Level = resolved.HeadingLevel
		}
	}

	// Extract text
	var textParts []string

	// Direct text content
	if h.Text != "" {
		textParts = append(textParts, h.Text)
	}

	// Text from spans
	for _, span := range h.Spans {
		if span.Text != "" {
			textParts = append(textParts, span.Text)
		}
	}

	parsed.Text = strings.Join(textParts, "")

	return parsed
}

// parseMetadata parses the meta.xml file.
func (r *Reader) parseMetadata() {
	data, err := r.getFileContent("meta.xml")
	if err != nil {
		return
	}

	r.meta = &metaXML{}
	xml.Unmarshal(data, r.meta)
}
