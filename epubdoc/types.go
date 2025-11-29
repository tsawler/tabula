// Package epubdoc provides EPUB document parsing.
package epubdoc

import (
	"time"
)

// Package represents the parsed OPF document.
type Package struct {
	Metadata Metadata
	Manifest map[string]ManifestItem // keyed by ID
	Spine    []SpineItem
	Version  string // "2.0" or "3.0"
}

// Metadata contains EPUB metadata (Dublin Core).
type Metadata struct {
	Title       string
	Creator     []string // Multiple authors possible
	Language    string
	Identifier  string // ISBN, UUID, etc.
	Publisher   string
	Date        string
	Description string
	Subjects    []string
	Rights      string
	Modified    time.Time
}

// ManifestItem represents a file in the EPUB.
type ManifestItem struct {
	ID         string
	Href       string
	MediaType  string
	Properties []string // "nav", "cover-image", etc.
}

// SpineItem represents a content document in reading order.
type SpineItem struct {
	IDRef  string
	Linear bool // true if part of main reading order
}

// Chapter represents extracted content from one spine item.
type Chapter struct {
	ID      string
	Title   string
	Index   int
	Href    string
	Content []byte // Raw XHTML content
}

// TableOfContents represents the navigation structure.
type TableOfContents struct {
	Title   string
	Entries []TOCEntry
}

// TOCEntry represents a single navigation entry.
type TOCEntry struct {
	Title    string
	Href     string
	Children []TOCEntry
}

// ExtractOptions configures content extraction.
type ExtractOptions struct {
	// NavigationExclusion controls filtering of nav/header/footer elements.
	// Uses htmldoc.NavigationExclusionMode values.
	NavigationExclusion int
}
