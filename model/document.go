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
