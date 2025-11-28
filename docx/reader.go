// Package docx provides DOCX (Office Open XML) document parsing.
package docx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/tsawler/tabula/model"
)

// Reader provides access to DOCX document content.
type Reader struct {
	file              *os.File
	zipReader         *zip.ReadCloser
	document          *documentXML
	styles            *stylesXML
	numbering         *numberingXML
	rels              *relationshipsXML
	coreProps         *corePropertiesXML
	appProps          *appPropertiesXML
	styleResolver     *StyleResolver
	numberingResolver *NumberingResolver
	tableParser       *TableParser
	listParser        *ListParser
	paragraphs        []parsedParagraph
	tables            []ParsedTable
	lists             []ParsedList
	elements          []parsedElement // Elements in document order
}

// parsedElement represents a parsed element (paragraph or table) with its type.
type parsedElement struct {
	Type      string           // "paragraph" or "table"
	Paragraph *parsedParagraph // Non-nil if Type == "paragraph"
	Table     *ParsedTable     // Non-nil if Type == "table"
}

// parsedParagraph holds a parsed paragraph with resolved styles.
type parsedParagraph struct {
	Text      string
	StyleID   string
	StyleName string
	IsHeading bool
	Level     int // heading level (1-9) or 0 for non-headings

	// List properties
	IsListItem bool
	NumID      string // Numbering ID (empty if not a list item)
	ListLevel  int    // List indentation level (0-based)

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
	_ = r.parseNumbering() // Numbering is optional

	// Create numbering resolver and list parser
	r.numberingResolver = NewNumberingResolver(r.numbering)
	r.listParser = NewListParser(r.numberingResolver)

	// Extract lists from paragraphs
	r.lists = r.listParser.ExtractLists(r.paragraphs)

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
// This includes text from paragraphs, lists, and tables in document order.
func (r *Reader) Text() (string, error) {
	if r.document == nil {
		return "", fmt.Errorf("document not parsed")
	}

	var result strings.Builder

	// Track list item counters for ordered lists (per numID and level)
	listCounters := make(map[string]map[int]int) // numID -> level -> count

	// If we have ordered elements, use them for correct document order
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

	// Fallback: output paragraphs then tables (old behavior)
	for i, para := range r.paragraphs {
		if i > 0 {
			result.WriteString("\n")
		}
		r.writeParagraphText(&result, &para, listCounters)
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

// Markdown returns the document content as a Markdown-formatted string.
// It converts:
//   - Headings to # notation (level 1-6)
//   - Lists to - (bullet) or 1. (numbered) notation
//   - Tables to markdown table format
//   - Paragraphs to plain text
func (r *Reader) Markdown() (string, error) {
	if len(r.elements) == 0 && len(r.paragraphs) == 0 {
		return "", nil
	}

	var result strings.Builder

	// Track list state for grouping
	var inList bool
	var lastNumID string
	listCounters := make(map[string]map[int]int)

	// Process elements in order
	for i, elem := range r.elements {
		switch elem.Type {
		case "paragraph":
			para := elem.Paragraph
			if para == nil {
				continue
			}

			// Add separator between elements (except first)
			if i > 0 && result.Len() > 0 {
				// Check if we're transitioning out of a list
				if inList && (!para.IsListItem || para.NumID != lastNumID) {
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
			} else if para.IsListItem && para.NumID != "" && para.NumID != "0" {
				// Render as markdown list item
				r.writeMarkdownListItem(&result, para, listCounters)
				inList = true
				lastNumID = para.NumID
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

	// Get or initialize counter for this list
	if listCounters[para.NumID] == nil {
		listCounters[para.NumID] = make(map[int]int)
	}

	// Reset child level counters when going back to a parent level
	lastLevel, hasLast := listCounters[para.NumID][-1] // Use -1 to store last level
	if hasLast && para.ListLevel <= lastLevel {
		// Clear counters for all levels deeper than current
		for lvl := range listCounters[para.NumID] {
			if lvl > para.ListLevel {
				delete(listCounters[para.NumID], lvl)
			}
		}
	}
	listCounters[para.NumID][-1] = para.ListLevel // Track last level

	// Determine list type
	listType, _, startAt := r.getListFormat(para.NumID, para.ListLevel)

	if listType == ListTypeOrdered {
		// Numbered list
		listCounters[para.NumID][para.ListLevel]++
		num := startAt + listCounters[para.NumID][para.ListLevel] - 1
		sb.WriteString(fmt.Sprintf("%d. ", num))
	} else {
		// Bullet list
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

		// Get or initialize counter for this list
		if listCounters[para.NumID] == nil {
			listCounters[para.NumID] = make(map[int]int)
		}

		// Determine list type and format bullet/number
		listType, bullet, startAt := r.getListFormat(para.NumID, para.ListLevel)

		if listType == ListTypeOrdered {
			// Increment counter for this level
			listCounters[para.NumID][para.ListLevel]++
			num := startAt + listCounters[para.NumID][para.ListLevel] - 1
			sb.WriteString(formatNumber(num, para.NumID, para.ListLevel, r.numberingResolver))
			sb.WriteString(". ")
		} else {
			// Bullet list
			if bullet == "" {
				bullet = getBulletChar("", para.ListLevel)
			}
			sb.WriteString(bullet)
			sb.WriteString(" ")
		}
	}

	sb.WriteString(para.Text)
}

// getListFormat returns list formatting info for a numID and level.
func (r *Reader) getListFormat(numID string, level int) (ListType, string, int) {
	if r.numberingResolver == nil {
		return ListTypeUnordered, "•", 1
	}
	return r.numberingResolver.ResolveLevel(numID, level)
}

// formatNumber formats a list number based on the numbering format.
func formatNumber(num int, numID string, level int, resolver *NumberingResolver) string {
	if resolver == nil {
		return fmt.Sprintf("%d", num)
	}

	// Get the format type from resolver
	abstractID, ok := resolver.numMappings[numID]
	if !ok {
		return fmt.Sprintf("%d", num)
	}

	abstractNum, ok := resolver.abstractNums[abstractID]
	if !ok {
		return fmt.Sprintf("%d", num)
	}

	levelStr := fmt.Sprintf("%d", level)
	for _, lvl := range abstractNum.Levels {
		if lvl.ILvl == levelStr {
			switch lvl.NumFmt.Val {
			case "lowerLetter":
				return toLowerLetter(num)
			case "upperLetter":
				return toUpperLetter(num)
			case "lowerRoman":
				return toLowerRoman(num)
			case "upperRoman":
				return toUpperRoman(num)
			default:
				return fmt.Sprintf("%d", num)
			}
		}
	}

	return fmt.Sprintf("%d", num)
}

// toLowerLetter converts a number to lowercase letter (1=a, 2=b, etc.)
func toLowerLetter(n int) string {
	if n < 1 {
		return "a"
	}
	result := ""
	for n > 0 {
		n-- // Make it 0-indexed
		result = string(rune('a'+n%26)) + result
		n /= 26
	}
	return result
}

// toUpperLetter converts a number to uppercase letter (1=A, 2=B, etc.)
func toUpperLetter(n int) string {
	return strings.ToUpper(toLowerLetter(n))
}

// toLowerRoman converts a number to lowercase Roman numerals.
func toLowerRoman(n int) string {
	return strings.ToLower(toUpperRoman(n))
}

// toUpperRoman converts a number to uppercase Roman numerals.
func toUpperRoman(n int) string {
	if n < 1 || n > 3999 {
		return fmt.Sprintf("%d", n)
	}

	romanNumerals := []struct {
		value  int
		symbol string
	}{
		{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"},
		{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"},
		{10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
	}

	result := ""
	for _, rn := range romanNumerals {
		for n >= rn.value {
			result += rn.symbol
			n -= rn.value
		}
	}
	return result
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

	// Process elements in document order, grouping list items
	yPos := 750.0 // Start near top of page

	// Track current list being built
	var currentList *model.List
	var currentListNumID string
	var currentListStartY float64
	var listLevelCounters map[int]int // Counter per level for ordered lists
	var lastListLevel int             // Track last item's level to reset child counters

	// Helper to finalize and add the current list
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
		currentListNumID = ""
		listLevelCounters = nil
		lastListLevel = 0
	}

	for _, elem := range r.elements {
		switch elem.Type {
		case "paragraph":
			para := elem.Paragraph
			if para == nil || para.Text == "" {
				continue
			}

			// Check if this is a list item
			if para.IsListItem && para.NumID != "" && para.NumID != "0" {
				// Determine if we need to start a new list
				if currentList == nil || para.NumID != currentListNumID {
					// Finalize previous list if any
					finalizeList()

					// Start new list
					listType, _, _ := r.numberingResolver.ResolveLevel(para.NumID, para.ListLevel)
					currentList = &model.List{
						Ordered: listType == ListTypeOrdered,
					}
					currentListNumID = para.NumID
					currentListStartY = yPos
					listLevelCounters = make(map[int]int)
				}

				// Reset child level counters when going back to a parent level
				if para.ListLevel <= lastListLevel {
					// Clear counters for all levels deeper than current
					for lvl := range listLevelCounters {
						if lvl > para.ListLevel {
							delete(listLevelCounters, lvl)
						}
					}
				}

				// Get bullet/number for this item
				_, bullet, startAt := r.numberingResolver.ResolveLevel(para.NumID, para.ListLevel)
				if currentList.Ordered && bullet == "" {
					// Increment counter for this level
					listLevelCounters[para.ListLevel]++
					itemNum := startAt + listLevelCounters[para.ListLevel] - 1
					bullet = formatNumber(itemNum, para.NumID, para.ListLevel, r.numberingResolver) + "."
				}
				if !currentList.Ordered && bullet == "" {
					bullet = getBulletChar("", para.ListLevel)
				}

				lastListLevel = para.ListLevel

				// Add item to current list
				currentList.Items = append(currentList.Items, model.ListItem{
					Text:   para.Text,
					Level:  para.ListLevel,
					Bullet: bullet,
				})
				continue
			}

			// Not a list item - finalize any pending list
			finalizeList()

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

		case "table":
			// Finalize any pending list before adding table
			finalizeList()

			if elem.Table == nil {
				continue
			}

			modelTable := elem.Table.ToModelTable()
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
	}

	// Finalize any remaining list
	finalizeList()

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

	// Parse elements in order using xml.Decoder
	if err := r.parseBodyElementsInOrder(data); err != nil {
		return fmt.Errorf("parsing body elements in order: %w", err)
	}

	// Process elements in document order
	r.processElementsInOrder()

	return nil
}

// parseBodyElementsInOrder parses body elements maintaining document order.
func (r *Reader) parseBodyElementsInOrder(data []byte) error {
	if r.document.Body == nil {
		return nil
	}

	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var inBody bool
	var paraIndex, tableIndex int

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Check if we're entering the body
			if t.Name.Local == "body" {
				inBody = true
				continue
			}

			if !inBody {
				continue
			}

			// Track elements in order
			switch t.Name.Local {
			case "p":
				if paraIndex < len(r.document.Body.Paragraphs) {
					r.document.Body.Elements = append(r.document.Body.Elements, bodyElement{
						Type:      "paragraph",
						Paragraph: &r.document.Body.Paragraphs[paraIndex],
					})
					paraIndex++
				}
			case "tbl":
				if tableIndex < len(r.document.Body.Tables) {
					r.document.Body.Elements = append(r.document.Body.Elements, bodyElement{
						Type:  "table",
						Table: &r.document.Body.Tables[tableIndex],
					})
					tableIndex++
				}
			}
		case xml.EndElement:
			if t.Name.Local == "body" {
				inBody = false
			}
		}
	}

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

// processElementsInOrder processes body elements in document order.
func (r *Reader) processElementsInOrder() {
	if r.document == nil || r.document.Body == nil {
		return
	}

	// If no Elements were parsed (fallback for simple parsing), use old method
	if len(r.document.Body.Elements) == 0 {
		r.processParagraphs()
		return
	}

	// Process elements in order
	r.elements = make([]parsedElement, 0, len(r.document.Body.Elements))
	r.paragraphs = make([]parsedParagraph, 0, len(r.document.Body.Paragraphs))
	r.tables = make([]ParsedTable, 0, len(r.document.Body.Tables))

	for _, elem := range r.document.Body.Elements {
		switch elem.Type {
		case "paragraph":
			if elem.Paragraph != nil {
				parsed := r.processParagraph(*elem.Paragraph)
				r.paragraphs = append(r.paragraphs, parsed)
				r.elements = append(r.elements, parsedElement{
					Type:      "paragraph",
					Paragraph: &r.paragraphs[len(r.paragraphs)-1],
				})
			}
		case "table":
			if elem.Table != nil && r.tableParser != nil {
				parsed := r.tableParser.ParseTable(*elem.Table)
				r.tables = append(r.tables, parsed)
				r.elements = append(r.elements, parsedElement{
					Type:  "table",
					Table: &r.tables[len(r.tables)-1],
				})
			}
		}
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

	// Check for list item (numbering properties)
	if ppr.NumPr.NumID.Val != "" && ppr.NumPr.NumID.Val != "0" {
		parsed.IsListItem = true
		parsed.NumID = ppr.NumPr.NumID.Val
		parsed.ListLevel = parseListLevel(ppr.NumPr.ILvl.Val)
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

	// Handle symbol characters (emoji and special symbols)
	for _, sym := range run.Symbols {
		if char := parseSymbolChar(sym.Char); char != "" {
			parts = append(parts, char)
		}
	}

	// Handle AlternateContent fallbacks (used for emoji in newer Word versions)
	for _, ac := range run.AlternateContent {
		for _, t := range ac.Fallback.Text {
			parts = append(parts, t.Value)
		}
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

// parseSymbolChar converts a hex character code to its Unicode character.
func parseSymbolChar(hexCode string) string {
	if hexCode == "" {
		return ""
	}

	// Parse the hex code
	code, err := strconv.ParseInt(hexCode, 16, 32)
	if err != nil {
		return ""
	}

	// Convert to rune and then to string
	// Handle surrogate pairs for emoji (codes > 0xFFFF)
	if code > 0 && code <= 0x10FFFF {
		return string(rune(code))
	}

	return ""
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

// parseListLevel parses a list level string to an integer (0-based).
func parseListLevel(s string) int {
	if s == "" {
		return 0
	}
	level := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			level = level*10 + int(c-'0')
		}
	}
	return level
}

// Lists returns all parsed lists from the document.
func (r *Reader) Lists() []ParsedList {
	return r.lists
}

// ModelLists returns lists converted to model.List format.
func (r *Reader) ModelLists() []*model.List {
	result := make([]*model.List, len(r.lists))
	for i := range r.lists {
		result[i] = r.lists[i].ToModelList()
	}
	return result
}
