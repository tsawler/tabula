package model

import "time"

// Document represents a complete PDF document with extracted structure
type Document struct {
	Metadata Metadata
	Pages    []*Page
}

// Metadata contains document-level information
type Metadata struct {
	Title        string
	Author       string
	Subject      string
	Keywords     []string
	Creator      string
	Producer     string
	CreationDate time.Time
	ModDate      time.Time
	// Custom metadata
	Custom map[string]string
}

// NewDocument creates a new empty document
func NewDocument() *Document {
	return &Document{
		Metadata: Metadata{
			Custom: make(map[string]string),
		},
		Pages: make([]*Page, 0),
	}
}

// AddPage adds a page to the document
func (d *Document) AddPage(page *Page) {
	page.Number = len(d.Pages) + 1
	d.Pages = append(d.Pages, page)
}

// GetPage returns a page by number (1-indexed)
func (d *Document) GetPage(number int) *Page {
	if number < 1 || number > len(d.Pages) {
		return nil
	}
	return d.Pages[number-1]
}

// PageCount returns the total number of pages
func (d *Document) PageCount() int {
	return len(d.Pages)
}

// ExtractText returns all text content concatenated
func (d *Document) ExtractText() string {
	var text string
	for _, page := range d.Pages {
		text += page.ExtractText() + "\n\n"
	}
	return text
}

// ExtractTables returns all tables from all pages
func (d *Document) ExtractTables() []*Table {
	var tables []*Table
	for _, page := range d.Pages {
		tables = append(tables, page.ExtractTables()...)
	}
	return tables
}

// HasLayout returns true if layout analysis has been performed on any page
func (d *Document) HasLayout() bool {
	for _, page := range d.Pages {
		if page.Layout != nil {
			return true
		}
	}
	return false
}

// AllHeadings returns all detected headings across all pages
func (d *Document) AllHeadings() []HeadingInfo {
	var headings []HeadingInfo
	for _, page := range d.Pages {
		if page.Layout != nil {
			headings = append(headings, page.Layout.Headings...)
		}
	}
	return headings
}

// AllLists returns all detected lists across all pages
func (d *Document) AllLists() []ListInfo {
	var lists []ListInfo
	for _, page := range d.Pages {
		if page.Layout != nil {
			lists = append(lists, page.Layout.Lists...)
		}
	}
	return lists
}

// AllParagraphs returns all detected paragraphs across all pages
func (d *Document) AllParagraphs() []ParagraphInfo {
	var paragraphs []ParagraphInfo
	for _, page := range d.Pages {
		if page.Layout != nil {
			paragraphs = append(paragraphs, page.Layout.Paragraphs...)
		}
	}
	return paragraphs
}

// LayoutStats returns combined layout statistics for the entire document
func (d *Document) LayoutStats() LayoutStats {
	var stats LayoutStats
	for _, page := range d.Pages {
		if page.Layout != nil {
			stats.FragmentCount += page.Layout.Stats.FragmentCount
			stats.LineCount += page.Layout.Stats.LineCount
			stats.BlockCount += page.Layout.Stats.BlockCount
			stats.ParagraphCount += page.Layout.Stats.ParagraphCount
			stats.HeadingCount += page.Layout.Stats.HeadingCount
			stats.ListCount += page.Layout.Stats.ListCount
		}
	}
	return stats
}

// TableOfContents returns headings organized as a document outline
func (d *Document) TableOfContents() []TOCEntry {
	var toc []TOCEntry
	for _, page := range d.Pages {
		if page.Layout == nil {
			continue
		}
		for _, h := range page.Layout.Headings {
			toc = append(toc, TOCEntry{
				Level:    h.Level,
				Text:     h.Text,
				Page:     page.Number,
				BBox:     h.BBox,
				FontSize: h.FontSize,
			})
		}
	}
	return toc
}

// TOCEntry represents an entry in the table of contents
type TOCEntry struct {
	Level    int     // Heading level (1-6)
	Text     string  // Heading text
	Page     int     // Page number (1-indexed)
	BBox     BBox    // Position on page
	FontSize float64 // Font size of heading
}
