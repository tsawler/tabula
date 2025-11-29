// Package htmldoc provides HTML document parsing.
package htmldoc

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/net/html"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/rag"
)

// Reader provides access to HTML document content.
type Reader struct {
	doc      *html.Node
	title    string
	metadata map[string]string
	elements []parsedElement
}

// Open opens an HTML file for reading.
func Open(filename string) (*Reader, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	return OpenReader(f)
}

// OpenReader parses HTML from an io.Reader.
func OpenReader(r io.Reader) (*Reader, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	reader := &Reader{
		doc:      doc,
		metadata: make(map[string]string),
		elements: make([]parsedElement, 0),
	}

	// Extract title and metadata from head
	reader.extractHead(doc)

	// Extract content from body
	reader.extractBody(doc)

	return reader, nil
}

// Close releases resources associated with the Reader.
func (r *Reader) Close() error {
	// Nothing to close for HTML (no file handles kept)
	return nil
}

// extractHead extracts title and meta tags from the head element.
func (r *Reader) extractHead(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "head" {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				switch c.Data {
				case "title":
					r.title = getTextContent(c)
				case "meta":
					name, content := "", ""
					for _, attr := range c.Attr {
						switch attr.Key {
						case "name", "property":
							name = attr.Val
						case "content":
							content = attr.Val
						}
					}
					if name != "" && content != "" {
						r.metadata[name] = content
					}
				}
			}
		}
		return
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r.extractHead(c)
	}
}

// extractBody extracts content from the body element.
func (r *Reader) extractBody(n *html.Node) {
	body := findElement(n, "body")
	if body == nil {
		// No body tag, try to extract from root
		body = n
	}

	ctx := &parseContext{
		inList:       false,
		listOrdered:  false,
		listLevel:    0,
		listItems:    nil,
	}

	r.traverseNode(body, ctx)

	// Flush any remaining list
	if ctx.inList && len(ctx.listItems) > 0 {
		r.elements = append(r.elements, parsedElement{
			Type:    ElementList,
			Items:   ctx.listItems,
			Ordered: ctx.listOrdered,
		})
	}
}

// parseContext tracks the current parsing state.
type parseContext struct {
	inList       bool
	listOrdered  bool
	listLevel    int
	listItems    []listItem
}

// traverseNode recursively processes DOM nodes.
func (r *Reader) traverseNode(n *html.Node, ctx *parseContext) {
	if n.Type == html.ElementNode {
		// Skip non-content elements
		if shouldSkipElement(n.Data) {
			return
		}

		switch n.Data {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			// Flush list before heading
			if ctx.inList && len(ctx.listItems) > 0 {
				r.elements = append(r.elements, parsedElement{
					Type:    ElementList,
					Items:   ctx.listItems,
					Ordered: ctx.listOrdered,
				})
				ctx.inList = false
				ctx.listItems = nil
			}

			level := int(n.Data[1] - '0')
			text := strings.TrimSpace(getTextContent(n))
			if text != "" {
				r.elements = append(r.elements, parsedElement{
					Type:  ElementHeading,
					Text:  text,
					Level: level,
				})
			}
			return

		case "p", "div":
			// Flush list before paragraph
			if ctx.inList && len(ctx.listItems) > 0 && n.Data == "p" {
				r.elements = append(r.elements, parsedElement{
					Type:    ElementList,
					Items:   ctx.listItems,
					Ordered: ctx.listOrdered,
				})
				ctx.inList = false
				ctx.listItems = nil
			}

			text := strings.TrimSpace(getTextContent(n))
			if text != "" && !isBlockContainer(n) {
				r.elements = append(r.elements, parsedElement{
					Type: ElementParagraph,
					Text: text,
				})
				return
			}
			// If it's a block container (div with children), traverse children
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				r.traverseNode(c, ctx)
			}
			return

		case "ul", "ol":
			// Flush previous list if different type or top-level
			if ctx.inList && ctx.listLevel == 0 && len(ctx.listItems) > 0 {
				r.elements = append(r.elements, parsedElement{
					Type:    ElementList,
					Items:   ctx.listItems,
					Ordered: ctx.listOrdered,
				})
				ctx.listItems = nil
			}

			prevInList := ctx.inList
			prevOrdered := ctx.listOrdered
			prevLevel := ctx.listLevel

			ctx.inList = true
			ctx.listOrdered = n.Data == "ol"
			if !prevInList {
				ctx.listItems = make([]listItem, 0)
				ctx.listLevel = 0
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				r.traverseNode(c, ctx)
			}

			if !prevInList {
				// Flush the list we just built
				if len(ctx.listItems) > 0 {
					r.elements = append(r.elements, parsedElement{
						Type:    ElementList,
						Items:   ctx.listItems,
						Ordered: ctx.listOrdered,
					})
				}
				ctx.inList = false
				ctx.listItems = nil
			}

			ctx.listOrdered = prevOrdered
			ctx.listLevel = prevLevel
			return

		case "li":
			if ctx.inList {
				// Get direct text content, not nested lists
				text := getDirectTextContent(n)
				if text != "" {
					ctx.listItems = append(ctx.listItems, listItem{
						Text:  text,
						Level: ctx.listLevel,
					})
				}
				// Check for nested lists
				ctx.listLevel++
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && (c.Data == "ul" || c.Data == "ol") {
						r.traverseNode(c, ctx)
					}
				}
				ctx.listLevel--
			}
			return

		case "table":
			// Flush list before table
			if ctx.inList && len(ctx.listItems) > 0 {
				r.elements = append(r.elements, parsedElement{
					Type:    ElementList,
					Items:   ctx.listItems,
					Ordered: ctx.listOrdered,
				})
				ctx.inList = false
				ctx.listItems = nil
			}

			table := r.parseTable(n)
			if table != nil && len(table.Rows) > 0 {
				r.elements = append(r.elements, parsedElement{
					Type:  ElementTable,
					Table: table,
				})
			}
			return

		case "pre", "code":
			text := getTextContent(n)
			if text != "" {
				r.elements = append(r.elements, parsedElement{
					Type:   ElementCode,
					Text:   text,
					IsCode: true,
				})
			}
			return

		case "blockquote":
			text := strings.TrimSpace(getTextContent(n))
			if text != "" {
				r.elements = append(r.elements, parsedElement{
					Type: ElementBlockquote,
					Text: text,
				})
			}
			return

		case "a":
			// Links are handled inline in text extraction
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				r.traverseNode(c, ctx)
			}
			return

		case "br":
			// Line breaks handled in text extraction
			return

		case "hr":
			// Horizontal rules - could be handled as separators
			return

		case "article", "section", "main", "header", "footer", "nav", "aside":
			// Semantic containers - traverse children
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				r.traverseNode(c, ctx)
			}
			return
		}
	}

	// Default: traverse children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r.traverseNode(c, ctx)
	}
}

// parseTable extracts a table from an HTML table element.
func (r *Reader) parseTable(tableNode *html.Node) *ParsedTable {
	table := &ParsedTable{
		Rows: make([][]TableCell, 0),
	}

	// Find thead, tbody, or direct tr children
	for c := tableNode.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			switch c.Data {
			case "thead":
				table.HasHeader = true
				r.parseTableRows(c, table, true)
			case "tbody":
				r.parseTableRows(c, table, false)
			case "tr":
				row := r.parseTableRow(c, false)
				if len(row) > 0 {
					table.Rows = append(table.Rows, row)
				}
			}
		}
	}

	// If no explicit header but first row has th elements, mark as header
	if !table.HasHeader && len(table.Rows) > 0 {
		hasHeader := false
		for _, cell := range table.Rows[0] {
			if cell.IsHeader {
				hasHeader = true
				break
			}
		}
		table.HasHeader = hasHeader
	}

	return table
}

// parseTableRows parses rows within thead or tbody.
func (r *Reader) parseTableRows(section *html.Node, table *ParsedTable, isHeader bool) {
	for c := section.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tr" {
			row := r.parseTableRow(c, isHeader)
			if len(row) > 0 {
				table.Rows = append(table.Rows, row)
			}
		}
	}
}

// parseTableRow parses a single table row.
func (r *Reader) parseTableRow(tr *html.Node, isHeader bool) []TableCell {
	row := make([]TableCell, 0)

	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cell := TableCell{
				Text:     strings.TrimSpace(getTextContent(c)),
				IsHeader: isHeader || c.Data == "th",
				RowSpan:  1,
				ColSpan:  1,
			}

			// Parse rowspan and colspan
			for _, attr := range c.Attr {
				switch attr.Key {
				case "rowspan":
					fmt.Sscanf(attr.Val, "%d", &cell.RowSpan)
				case "colspan":
					fmt.Sscanf(attr.Val, "%d", &cell.ColSpan)
				}
			}

			row = append(row, cell)
		}
	}

	return row
}

// shouldSkipElement returns true if the element should be skipped during content extraction.
func shouldSkipElement(tagName string) bool {
	switch tagName {
	case "script", "style", "noscript", "template", "svg", "math", "iframe", "object", "embed":
		return true
	}
	return false
}

// isBlockContainer returns true if the element is a block container with block-level children.
func isBlockContainer(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			switch c.Data {
			case "div", "p", "ul", "ol", "table", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "pre", "article", "section":
				return true
			}
		}
	}
	return false
}

// findElement finds the first element with the given tag name.
func findElement(n *html.Node, tagName string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tagName {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findElement(c, tagName); result != nil {
			return result
		}
	}
	return nil
}

// getTextContent extracts all text content from a node and its descendants.
func getTextContent(n *html.Node) string {
	var result strings.Builder
	getTextContentRecursive(n, &result)
	return strings.TrimSpace(result.String())
}

func getTextContentRecursive(n *html.Node, result *strings.Builder) {
	if n.Type == html.TextNode {
		result.WriteString(n.Data)
	}
	if n.Type == html.ElementNode {
		// Skip script/style content
		if shouldSkipElement(n.Data) {
			return
		}
		// Add space after block elements
		if n.Data == "br" {
			result.WriteString("\n")
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		getTextContentRecursive(c, result)
	}
	// Add space after certain block elements
	if n.Type == html.ElementNode {
		switch n.Data {
		case "p", "div", "li", "h1", "h2", "h3", "h4", "h5", "h6", "tr":
			result.WriteString(" ")
		}
	}
}

// getDirectTextContent gets text content from a node, excluding nested block elements.
func getDirectTextContent(n *html.Node) string {
	var result strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			result.WriteString(c.Data)
		} else if c.Type == html.ElementNode {
			// Include inline elements, skip block elements
			switch c.Data {
			case "ul", "ol", "div", "p", "table", "blockquote":
				// Skip these - they're block elements
			default:
				result.WriteString(getTextContent(c))
			}
		}
	}
	return strings.TrimSpace(result.String())
}

// PageCount returns 1 (HTML documents are single-page).
func (r *Reader) PageCount() (int, error) {
	return 1, nil
}

// ExtractOptions holds options for text extraction.
type ExtractOptions struct {
	IncludeLinks      bool // Preserve link URLs in output
	IncludeMetadata   bool // Include meta tags
	StripNavigation   bool // Skip <nav>, <header>, <footer> elements (not yet implemented)
	ExcludeHeaders    bool // Exclude headers (not applicable for HTML)
	ExcludeFooters    bool // Exclude footers (not applicable for HTML)
}

// Text extracts and returns all text content from the HTML document.
func (r *Reader) Text() (string, error) {
	return r.TextWithOptions(ExtractOptions{})
}

// TextWithOptions extracts text content with the specified options.
func (r *Reader) TextWithOptions(opts ExtractOptions) (string, error) {
	var result strings.Builder

	for _, elem := range r.elements {
		switch elem.Type {
		case ElementHeading:
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			result.WriteString(elem.Text)

		case ElementParagraph:
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			result.WriteString(elem.Text)

		case ElementList:
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			for i, item := range elem.Items {
				if i > 0 {
					result.WriteString("\n")
				}
				for j := 0; j < item.Level; j++ {
					result.WriteString("  ")
				}
				result.WriteString("• ")
				result.WriteString(item.Text)
			}

		case ElementTable:
			if elem.Table != nil && len(elem.Table.Rows) > 0 {
				if result.Len() > 0 {
					result.WriteString("\n\n")
				}
				for _, row := range elem.Table.Rows {
					for j, cell := range row {
						if j > 0 {
							result.WriteString("\t")
						}
						result.WriteString(cell.Text)
					}
					result.WriteString("\n")
				}
			}

		case ElementCode:
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			result.WriteString(elem.Text)

		case ElementBlockquote:
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			result.WriteString(elem.Text)
		}
	}

	return result.String(), nil
}

// Markdown returns the HTML content as Markdown.
func (r *Reader) Markdown() (string, error) {
	return r.MarkdownWithOptions(ExtractOptions{})
}

// MarkdownWithOptions returns HTML content as Markdown with options.
func (r *Reader) MarkdownWithOptions(opts ExtractOptions) (string, error) {
	var result strings.Builder

	for _, elem := range r.elements {
		switch elem.Type {
		case ElementHeading:
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			for i := 0; i < elem.Level; i++ {
				result.WriteString("#")
			}
			result.WriteString(" ")
			result.WriteString(elem.Text)

		case ElementParagraph:
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			result.WriteString(elem.Text)

		case ElementList:
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			for i, item := range elem.Items {
				if i > 0 {
					result.WriteString("\n")
				}
				for j := 0; j < item.Level; j++ {
					result.WriteString("  ")
				}
				if elem.Ordered {
					result.WriteString("1. ")
				} else {
					result.WriteString("- ")
				}
				result.WriteString(item.Text)
			}

		case ElementTable:
			if elem.Table != nil && len(elem.Table.Rows) > 0 {
				if result.Len() > 0 {
					result.WriteString("\n\n")
				}
				result.WriteString(elem.Table.ToMarkdown())
			}

		case ElementCode:
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			result.WriteString("```\n")
			result.WriteString(elem.Text)
			result.WriteString("\n```")

		case ElementBlockquote:
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			lines := strings.Split(elem.Text, "\n")
			for i, line := range lines {
				if i > 0 {
					result.WriteString("\n")
				}
				result.WriteString("> ")
				result.WriteString(line)
			}
		}
	}

	return result.String(), nil
}

// MarkdownWithRAGOptions returns HTML content as Markdown with RAG options.
func (r *Reader) MarkdownWithRAGOptions(extractOpts ExtractOptions, mdOpts rag.MarkdownOptions) (string, error) {
	var result strings.Builder

	// Add YAML front matter metadata if requested
	if mdOpts.IncludeMetadata {
		result.WriteString("---\n")
		if r.title != "" {
			result.WriteString(fmt.Sprintf("title: %q\n", r.title))
		}
		if author, ok := r.metadata["author"]; ok {
			result.WriteString(fmt.Sprintf("author: %q\n", author))
		}
		if desc, ok := r.metadata["description"]; ok {
			result.WriteString(fmt.Sprintf("description: %q\n", desc))
		}
		if keywords, ok := r.metadata["keywords"]; ok {
			result.WriteString(fmt.Sprintf("keywords: %q\n", keywords))
		}
		result.WriteString("---\n\n")
	}

	// Add table of contents if requested
	if mdOpts.IncludeTableOfContents {
		headings := make([]parsedElement, 0)
		for _, elem := range r.elements {
			if elem.Type == ElementHeading {
				headings = append(headings, elem)
			}
		}
		if len(headings) > 1 {
			result.WriteString("## Table of Contents\n\n")
			for i, h := range headings {
				anchor := strings.ToLower(strings.ReplaceAll(h.Text, " ", "-"))
				result.WriteString(fmt.Sprintf("%d. [%s](#%s)\n", i+1, h.Text, anchor))
			}
			result.WriteString("\n---\n\n")
		}
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
	meta := model.Metadata{
		Title: r.title,
	}

	if author, ok := r.metadata["author"]; ok {
		meta.Author = author
	}
	if desc, ok := r.metadata["description"]; ok {
		meta.Subject = desc
	}
	if keywords, ok := r.metadata["keywords"]; ok {
		meta.Keywords = strings.Split(keywords, ",")
		for i, kw := range meta.Keywords {
			meta.Keywords[i] = strings.TrimSpace(kw)
		}
	}

	return meta
}

// Document returns a model.Document representation of the HTML content.
func (r *Reader) Document() (*model.Document, error) {
	doc := model.NewDocument()

	// Set metadata
	doc.Metadata.Title = r.title
	if author, ok := r.metadata["author"]; ok {
		doc.Metadata.Author = author
	}
	if desc, ok := r.metadata["description"]; ok {
		doc.Metadata.Subject = desc
	}
	if keywords, ok := r.metadata["keywords"]; ok {
		doc.Metadata.Keywords = strings.Split(keywords, ",")
		for i, kw := range doc.Metadata.Keywords {
			doc.Metadata.Keywords[i] = strings.TrimSpace(kw)
		}
	}

	// HTML is treated as a single page
	page := model.NewPage(612, 792) // Letter size
	page.Number = 1

	yPos := 750.0

	for _, elem := range r.elements {
		switch elem.Type {
		case ElementHeading:
			heading := &model.Heading{
				Level: elem.Level,
				Text:  elem.Text,
				BBox:  model.BBox{X: 36, Y: yPos, Width: 540, Height: 20},
			}
			page.AddElement(heading)
			yPos -= 30

		case ElementParagraph:
			para := &model.Paragraph{
				Text: elem.Text,
				BBox: model.BBox{X: 36, Y: yPos, Width: 540, Height: 15},
			}
			page.AddElement(para)
			yPos -= 25

		case ElementList:
			list := &model.List{
				Ordered: elem.Ordered,
				BBox:    model.BBox{X: 36, Y: yPos, Width: 540, Height: 15},
			}
			for _, item := range elem.Items {
				list.Items = append(list.Items, model.ListItem{
					Text:  item.Text,
					Level: item.Level,
				})
			}
			page.AddElement(list)
			yPos -= float64(15 * len(elem.Items))

		case ElementTable:
			if elem.Table != nil && len(elem.Table.Rows) > 0 {
				numRows := len(elem.Table.Rows)
				numCols := 0
				if numRows > 0 {
					numCols = len(elem.Table.Rows[0])
				}

				modelTable := model.NewTable(numRows, numCols)
				modelTable.BBox = model.BBox{X: 36, Y: yPos, Width: 540, Height: float64(numRows * 15)}

				for i, row := range elem.Table.Rows {
					for j, cell := range row {
						if j < len(modelTable.Rows[i]) {
							modelTable.Rows[i][j] = model.Cell{
								Text:     cell.Text,
								RowSpan:  cell.RowSpan,
								ColSpan:  cell.ColSpan,
								IsHeader: cell.IsHeader,
							}
						}
					}
				}

				page.AddElement(modelTable)
				yPos -= float64(numRows*15 + 10)
			}

		case ElementCode:
			// Treat code blocks as paragraphs with special formatting
			para := &model.Paragraph{
				Text: elem.Text,
				BBox: model.BBox{X: 36, Y: yPos, Width: 540, Height: 15},
			}
			page.AddElement(para)
			yPos -= 25

		case ElementBlockquote:
			para := &model.Paragraph{
				Text: elem.Text,
				BBox: model.BBox{X: 50, Y: yPos, Width: 520, Height: 15},
			}
			page.AddElement(para)
			yPos -= 25
		}
	}

	doc.AddPage(page)

	return doc, nil
}
