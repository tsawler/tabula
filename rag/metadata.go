package rag

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ContextFormat defines how context is injected into chunk text
type ContextFormat int

const (
	// ContextFormatNone adds no context
	ContextFormatNone ContextFormat = iota
	// ContextFormatBracket adds context in brackets: [Section Title]
	ContextFormatBracket
	// ContextFormatMarkdown adds context as markdown heading
	ContextFormatMarkdown
	// ContextFormatBreadcrumb adds full path as breadcrumb
	ContextFormatBreadcrumb
	// ContextFormatXML adds context in XML-style tags
	ContextFormatXML
)

// String returns a human-readable representation of the context format
func (cf ContextFormat) String() string {
	switch cf {
	case ContextFormatNone:
		return "none"
	case ContextFormatBracket:
		return "bracket"
	case ContextFormatMarkdown:
		return "markdown"
	case ContextFormatBreadcrumb:
		return "breadcrumb"
	case ContextFormatXML:
		return "xml"
	default:
		return "unknown"
	}
}

// MetadataConfig holds configuration for metadata handling
type MetadataConfig struct {
	// ContextFormat determines how context is added to chunk text
	ContextFormat ContextFormat

	// IncludeDocumentTitle includes document title in context
	IncludeDocumentTitle bool

	// IncludePageNumbers includes page numbers in context
	IncludePageNumbers bool

	// IncludeSectionPath includes full section path (not just title)
	IncludeSectionPath bool

	// WordsPerMinute for reading time estimation (default: 200)
	WordsPerMinute int
}

// DefaultMetadataConfig returns sensible defaults
func DefaultMetadataConfig() MetadataConfig {
	return MetadataConfig{
		ContextFormat:        ContextFormatBracket,
		IncludeDocumentTitle: false,
		IncludePageNumbers:   false,
		IncludeSectionPath:   false,
		WordsPerMinute:       200,
	}
}

// ChunkMetadata methods

// ToJSON serializes metadata to JSON
func (m *ChunkMetadata) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// ToJSONIndent serializes metadata to indented JSON
func (m *ChunkMetadata) ToJSONIndent() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// ToMap converts metadata to a map for flexible access
func (m *ChunkMetadata) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"document_title":   m.DocumentTitle,
		"section_path":     m.SectionPath,
		"section_title":    m.SectionTitle,
		"heading_level":    m.HeadingLevel,
		"page_start":       m.PageStart,
		"page_end":         m.PageEnd,
		"chunk_index":      m.ChunkIndex,
		"total_chunks":     m.TotalChunks,
		"level":            m.Level.String(),
		"parent_id":        m.ParentID,
		"child_ids":        m.ChildIDs,
		"element_types":    m.ElementTypes,
		"has_table":        m.HasTable,
		"has_list":         m.HasList,
		"has_image":        m.HasImage,
		"char_count":       m.CharCount,
		"word_count":       m.WordCount,
		"estimated_tokens": m.EstimatedTokens,
	}
}

// GetSectionPathString returns the section path as a formatted string
func (m *ChunkMetadata) GetSectionPathString(separator string) string {
	if len(m.SectionPath) == 0 {
		return ""
	}
	if separator == "" {
		separator = " > "
	}
	return strings.Join(m.SectionPath, separator)
}

// GetPageRange returns a formatted page range string
func (m *ChunkMetadata) GetPageRange() string {
	if m.PageStart == m.PageEnd {
		return fmt.Sprintf("p. %d", m.PageStart)
	}
	return fmt.Sprintf("pp. %d-%d", m.PageStart, m.PageEnd)
}

// GetReadingTimeMinutes estimates reading time in minutes
func (m *ChunkMetadata) GetReadingTimeMinutes(wordsPerMinute int) float64 {
	if wordsPerMinute <= 0 {
		wordsPerMinute = 200
	}
	return float64(m.WordCount) / float64(wordsPerMinute)
}

// GetReadingTimeString returns a human-readable reading time
func (m *ChunkMetadata) GetReadingTimeString(wordsPerMinute int) string {
	minutes := m.GetReadingTimeMinutes(wordsPerMinute)
	if minutes < 1 {
		return "< 1 min read"
	}
	return fmt.Sprintf("%.0f min read", minutes)
}

// IsInSection checks if the chunk is within a given section path
func (m *ChunkMetadata) IsInSection(sectionTitle string) bool {
	if m.SectionTitle == sectionTitle {
		return true
	}
	for _, s := range m.SectionPath {
		if s == sectionTitle {
			return true
		}
	}
	return false
}

// IsOnPage checks if the chunk spans a given page
func (m *ChunkMetadata) IsOnPage(page int) bool {
	return page >= m.PageStart && page <= m.PageEnd
}

// ContainsElementType checks if the chunk contains a specific element type
func (m *ChunkMetadata) ContainsElementType(elementType string) bool {
	for _, et := range m.ElementTypes {
		if strings.EqualFold(et, elementType) {
			return true
		}
	}
	return false
}

// Chunk methods

// GenerateContextText generates context text based on configuration
func (c *Chunk) GenerateContextText(config MetadataConfig) string {
	if config.ContextFormat == ContextFormatNone {
		return c.Text
	}

	var contextParts []string

	// Add document title if configured
	if config.IncludeDocumentTitle && c.Metadata.DocumentTitle != "" {
		contextParts = append(contextParts, c.Metadata.DocumentTitle)
	}

	// Add section path or title
	if config.IncludeSectionPath && len(c.Metadata.SectionPath) > 0 {
		contextParts = append(contextParts, c.Metadata.GetSectionPathString(" > "))
	} else if c.Metadata.SectionTitle != "" {
		contextParts = append(contextParts, c.Metadata.SectionTitle)
	}

	// Add page numbers if configured
	if config.IncludePageNumbers && c.Metadata.PageStart > 0 {
		contextParts = append(contextParts, c.Metadata.GetPageRange())
	}

	if len(contextParts) == 0 {
		return c.Text
	}

	context := strings.Join(contextParts, " | ")

	switch config.ContextFormat {
	case ContextFormatBracket:
		return fmt.Sprintf("[%s]\n\n%s", context, c.Text)
	case ContextFormatMarkdown:
		return fmt.Sprintf("# %s\n\n%s", context, c.Text)
	case ContextFormatBreadcrumb:
		return fmt.Sprintf("%s\n---\n%s", context, c.Text)
	case ContextFormatXML:
		return fmt.Sprintf("<context>%s</context>\n\n%s", context, c.Text)
	default:
		return c.Text
	}
}

// ToEmbeddingFormat returns text optimized for embedding generation
func (c *Chunk) ToEmbeddingFormat() string {
	// Include context for better semantic representation
	config := MetadataConfig{
		ContextFormat:      ContextFormatBracket,
		IncludeSectionPath: true,
	}
	return c.GenerateContextText(config)
}

// ToSearchableText returns text optimized for keyword search
func (c *Chunk) ToSearchableText() string {
	// Plain text without formatting
	return c.Text
}

// Summary returns a brief summary of the chunk
func (c *Chunk) Summary() string {
	var parts []string

	if c.Metadata.SectionTitle != "" {
		parts = append(parts, fmt.Sprintf("Section: %s", c.Metadata.SectionTitle))
	}
	parts = append(parts, c.Metadata.GetPageRange())
	parts = append(parts, fmt.Sprintf("%d words", c.Metadata.WordCount))

	if c.Metadata.HasTable {
		parts = append(parts, "contains table")
	}
	if c.Metadata.HasList {
		parts = append(parts, "contains list")
	}
	if c.Metadata.HasImage {
		parts = append(parts, "contains image")
	}

	return strings.Join(parts, " | ")
}

// ChunkCollection provides filtering and search over chunks
type ChunkCollection struct {
	Chunks []*Chunk
}

// NewChunkCollection creates a new collection from chunks
func NewChunkCollection(chunks []*Chunk) *ChunkCollection {
	return &ChunkCollection{Chunks: chunks}
}

// Filter returns chunks matching a predicate
func (cc *ChunkCollection) Filter(predicate func(*Chunk) bool) *ChunkCollection {
	var filtered []*Chunk
	for _, c := range cc.Chunks {
		if predicate(c) {
			filtered = append(filtered, c)
		}
	}
	return &ChunkCollection{Chunks: filtered}
}

// FilterBySection returns chunks in a specific section
func (cc *ChunkCollection) FilterBySection(sectionTitle string) *ChunkCollection {
	return cc.Filter(func(c *Chunk) bool {
		return c.Metadata.IsInSection(sectionTitle)
	})
}

// FilterByPage returns chunks on a specific page
func (cc *ChunkCollection) FilterByPage(page int) *ChunkCollection {
	return cc.Filter(func(c *Chunk) bool {
		return c.Metadata.IsOnPage(page)
	})
}

// FilterByPageRange returns chunks within a page range
func (cc *ChunkCollection) FilterByPageRange(startPage, endPage int) *ChunkCollection {
	return cc.Filter(func(c *Chunk) bool {
		return c.Metadata.PageEnd >= startPage && c.Metadata.PageStart <= endPage
	})
}

// FilterByElementType returns chunks containing a specific element type
func (cc *ChunkCollection) FilterByElementType(elementType string) *ChunkCollection {
	return cc.Filter(func(c *Chunk) bool {
		return c.Metadata.ContainsElementType(elementType)
	})
}

// FilterWithTables returns chunks containing tables
func (cc *ChunkCollection) FilterWithTables() *ChunkCollection {
	return cc.Filter(func(c *Chunk) bool {
		return c.Metadata.HasTable
	})
}

// FilterWithLists returns chunks containing lists
func (cc *ChunkCollection) FilterWithLists() *ChunkCollection {
	return cc.Filter(func(c *Chunk) bool {
		return c.Metadata.HasList
	})
}

// FilterWithImages returns chunks containing images
func (cc *ChunkCollection) FilterWithImages() *ChunkCollection {
	return cc.Filter(func(c *Chunk) bool {
		return c.Metadata.HasImage
	})
}

// FilterByMinTokens returns chunks with at least N estimated tokens
func (cc *ChunkCollection) FilterByMinTokens(minTokens int) *ChunkCollection {
	return cc.Filter(func(c *Chunk) bool {
		return c.Metadata.EstimatedTokens >= minTokens
	})
}

// FilterByMaxTokens returns chunks with at most N estimated tokens
func (cc *ChunkCollection) FilterByMaxTokens(maxTokens int) *ChunkCollection {
	return cc.Filter(func(c *Chunk) bool {
		return c.Metadata.EstimatedTokens <= maxTokens
	})
}

// Search returns chunks containing a keyword (case-insensitive)
func (cc *ChunkCollection) Search(keyword string) *ChunkCollection {
	keyword = strings.ToLower(keyword)
	return cc.Filter(func(c *Chunk) bool {
		return strings.Contains(strings.ToLower(c.Text), keyword)
	})
}

// Count returns the number of chunks in the collection
func (cc *ChunkCollection) Count() int {
	return len(cc.Chunks)
}

// First returns the first chunk or nil
func (cc *ChunkCollection) First() *Chunk {
	if len(cc.Chunks) == 0 {
		return nil
	}
	return cc.Chunks[0]
}

// Last returns the last chunk or nil
func (cc *ChunkCollection) Last() *Chunk {
	if len(cc.Chunks) == 0 {
		return nil
	}
	return cc.Chunks[len(cc.Chunks)-1]
}

// GetByIndex returns a chunk by index
func (cc *ChunkCollection) GetByIndex(index int) *Chunk {
	if index < 0 || index >= len(cc.Chunks) {
		return nil
	}
	return cc.Chunks[index]
}

// GetByID returns a chunk by ID
func (cc *ChunkCollection) GetByID(id string) *Chunk {
	for _, c := range cc.Chunks {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// ToSlice returns the underlying slice
func (cc *ChunkCollection) ToSlice() []*Chunk {
	return cc.Chunks
}

// GetAllSections returns unique section titles
func (cc *ChunkCollection) GetAllSections() []string {
	seen := make(map[string]bool)
	var sections []string

	for _, c := range cc.Chunks {
		if c.Metadata.SectionTitle != "" && !seen[c.Metadata.SectionTitle] {
			seen[c.Metadata.SectionTitle] = true
			sections = append(sections, c.Metadata.SectionTitle)
		}
	}

	return sections
}

// GetPageRange returns the min and max page numbers
func (cc *ChunkCollection) GetPageRange() (int, int) {
	if len(cc.Chunks) == 0 {
		return 0, 0
	}

	minPage := cc.Chunks[0].Metadata.PageStart
	maxPage := cc.Chunks[0].Metadata.PageEnd

	for _, c := range cc.Chunks[1:] {
		if c.Metadata.PageStart < minPage {
			minPage = c.Metadata.PageStart
		}
		if c.Metadata.PageEnd > maxPage {
			maxPage = c.Metadata.PageEnd
		}
	}

	return minPage, maxPage
}

// GetTotalTokens returns the sum of estimated tokens across all chunks
func (cc *ChunkCollection) GetTotalTokens() int {
	total := 0
	for _, c := range cc.Chunks {
		total += c.Metadata.EstimatedTokens
	}
	return total
}

// GetTotalWords returns the sum of words across all chunks
func (cc *ChunkCollection) GetTotalWords() int {
	total := 0
	for _, c := range cc.Chunks {
		total += c.Metadata.WordCount
	}
	return total
}

// Statistics returns aggregate statistics about the collection
func (cc *ChunkCollection) Statistics() CollectionStats {
	stats := CollectionStats{
		TotalChunks: len(cc.Chunks),
	}

	if len(cc.Chunks) == 0 {
		return stats
	}

	minTokens := cc.Chunks[0].Metadata.EstimatedTokens
	maxTokens := cc.Chunks[0].Metadata.EstimatedTokens

	for _, c := range cc.Chunks {
		stats.TotalTokens += c.Metadata.EstimatedTokens
		stats.TotalWords += c.Metadata.WordCount
		stats.TotalChars += c.Metadata.CharCount

		if c.Metadata.EstimatedTokens < minTokens {
			minTokens = c.Metadata.EstimatedTokens
		}
		if c.Metadata.EstimatedTokens > maxTokens {
			maxTokens = c.Metadata.EstimatedTokens
		}

		if c.Metadata.HasTable {
			stats.ChunksWithTables++
		}
		if c.Metadata.HasList {
			stats.ChunksWithLists++
		}
		if c.Metadata.HasImage {
			stats.ChunksWithImages++
		}
	}

	stats.AvgTokens = stats.TotalTokens / len(cc.Chunks)
	stats.MinTokens = minTokens
	stats.MaxTokens = maxTokens
	stats.UniqueSections = len(cc.GetAllSections())
	stats.PageStart, stats.PageEnd = cc.GetPageRange()

	return stats
}

// CollectionStats contains aggregate statistics about a chunk collection
type CollectionStats struct {
	TotalChunks      int
	TotalTokens      int
	TotalWords       int
	TotalChars       int
	AvgTokens        int
	MinTokens        int
	MaxTokens        int
	ChunksWithTables int
	ChunksWithLists  int
	ChunksWithImages int
	UniqueSections   int
	PageStart        int
	PageEnd          int
}

// ToJSON serializes stats to JSON
func (cs *CollectionStats) ToJSON() ([]byte, error) {
	return json.Marshal(cs)
}
