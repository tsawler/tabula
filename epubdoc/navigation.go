package epubdoc

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"strings"

	"golang.org/x/net/html"
)

// ncxDocument represents an EPUB 2 NCX navigation document.
type ncxDocument struct {
	XMLName xml.Name  `xml:"ncx"`
	Title   string    `xml:"docTitle>text"`
	NavMap  ncxNavMap `xml:"navMap"`
}

type ncxNavMap struct {
	NavPoints []ncxNavPoint `xml:"navPoint"`
}

type ncxNavPoint struct {
	ID        string        `xml:"id,attr"`
	PlayOrder string        `xml:"playOrder,attr"`
	Label     string        `xml:"navLabel>text"`
	Content   ncxContent    `xml:"content"`
	Children  []ncxNavPoint `xml:"navPoint"`
}

type ncxContent struct {
	Src string `xml:"src,attr"`
}

// parseNavigation parses the table of contents from either NCX (EPUB 2) or nav document (EPUB 3).
func (r *Reader) parseNavigation(zr *zip.Reader) (*TableOfContents, error) {
	// Try EPUB 3 nav document first
	if navItem := r.findNavDocument(); navItem != nil {
		href := r.resolveHref(navItem.Href)
		content, err := r.readFile(zr, href)
		if err == nil {
			if toc, err := parseNavXHTML(content); err == nil {
				return toc, nil
			}
		}
	}

	// Fall back to EPUB 2 NCX
	if ncxItem := r.findNCX(); ncxItem != nil {
		href := r.resolveHref(ncxItem.Href)
		content, err := r.readFile(zr, href)
		if err == nil {
			if toc, err := parseNCX(content); err == nil {
				return toc, nil
			}
		}
	}

	// Generate from spine if no navigation found
	return r.generateTOCFromSpine(), nil
}

// findNavDocument finds the EPUB 3 nav document in the manifest.
func (r *Reader) findNavDocument() *ManifestItem {
	for _, item := range r.pkg.Manifest {
		for _, prop := range item.Properties {
			if prop == "nav" {
				return &item
			}
		}
	}
	return nil
}

// findNCX finds the NCX document in the manifest.
func (r *Reader) findNCX() *ManifestItem {
	for _, item := range r.pkg.Manifest {
		if item.MediaType == "application/x-dtbncx+xml" {
			return &item
		}
	}
	return nil
}

// parseNavXHTML parses an EPUB 3 nav document (XHTML with nav element).
func parseNavXHTML(content []byte) (*TableOfContents, error) {
	doc, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	toc := &TableOfContents{}

	// Find the <nav> element with epub:type="toc"
	var findNav func(*html.Node) *html.Node
	findNav = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode && n.Data == "nav" {
			for _, attr := range n.Attr {
				if (attr.Key == "epub:type" || attr.Key == "type") && strings.Contains(attr.Val, "toc") {
					return n
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if found := findNav(c); found != nil {
				return found
			}
		}
		return nil
	}

	nav := findNav(doc)
	if nav == nil {
		return nil, ErrMissingContent
	}

	// Find the title (usually in a heading or the first text)
	var findTitle func(*html.Node) string
	findTitle = func(n *html.Node) string {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h1", "h2", "h3", "h4", "h5", "h6":
				return extractText(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if title := findTitle(c); title != "" {
				return title
			}
		}
		return ""
	}
	toc.Title = findTitle(nav)

	// Find the <ol> element and parse entries
	var findOL func(*html.Node) *html.Node
	findOL = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode && n.Data == "ol" {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if found := findOL(c); found != nil {
				return found
			}
		}
		return nil
	}

	ol := findOL(nav)
	if ol != nil {
		toc.Entries = parseOLEntries(ol)
	}

	return toc, nil
}

// parseOLEntries parses TOC entries from an <ol> element.
func parseOLEntries(ol *html.Node) []TOCEntry {
	var entries []TOCEntry

	for c := ol.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "li" {
			entry := parseLIEntry(c)
			if entry.Title != "" || entry.Href != "" {
				entries = append(entries, entry)
			}
		}
	}

	return entries
}

// parseLIEntry parses a single TOC entry from an <li> element.
func parseLIEntry(li *html.Node) TOCEntry {
	entry := TOCEntry{}

	for c := li.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			switch c.Data {
			case "a":
				entry.Title = extractText(c)
				for _, attr := range c.Attr {
					if attr.Key == "href" {
						entry.Href = attr.Val
					}
				}
			case "span":
				if entry.Title == "" {
					entry.Title = extractText(c)
				}
			case "ol":
				entry.Children = parseOLEntries(c)
			}
		}
	}

	return entry
}

// parseNCX parses an EPUB 2 NCX document.
func parseNCX(content []byte) (*TableOfContents, error) {
	var ncx ncxDocument
	if err := xml.Unmarshal(content, &ncx); err != nil {
		return nil, err
	}

	toc := &TableOfContents{
		Title:   ncx.Title,
		Entries: convertNCXNavPoints(ncx.NavMap.NavPoints),
	}

	return toc, nil
}

// convertNCXNavPoints converts NCX navPoints to TOCEntries.
func convertNCXNavPoints(points []ncxNavPoint) []TOCEntry {
	entries := make([]TOCEntry, 0, len(points))

	for _, p := range points {
		entry := TOCEntry{
			Title:    strings.TrimSpace(p.Label),
			Href:     p.Content.Src,
			Children: convertNCXNavPoints(p.Children),
		}
		entries = append(entries, entry)
	}

	return entries
}

// generateTOCFromSpine creates a basic TOC from the spine when no navigation is present.
func (r *Reader) generateTOCFromSpine() *TableOfContents {
	toc := &TableOfContents{
		Title:   r.pkg.Metadata.Title,
		Entries: make([]TOCEntry, 0, len(r.chapters)),
	}

	for _, chapter := range r.chapters {
		title := chapter.Title
		if title == "" {
			title = chapter.ID
		}

		toc.Entries = append(toc.Entries, TOCEntry{
			Title: title,
			Href:  chapter.Href,
		})
	}

	return toc
}

// extractText extracts all text content from an HTML node.
func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(extractText(c))
	}
	return strings.TrimSpace(text.String())
}

// TableOfContents returns the parsed table of contents.
func (r *Reader) TableOfContents() *TableOfContents {
	if r.toc != nil {
		return r.toc
	}

	zr := r.getZipReader()
	toc, err := r.parseNavigation(zr)
	if err != nil {
		toc = r.generateTOCFromSpine()
	}

	r.toc = toc
	return toc
}
