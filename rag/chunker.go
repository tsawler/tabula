// Package rag provides semantic chunking for RAG (Retrieval-Augmented Generation) workflows.
// It implements hierarchical, context-aware chunking that respects document structure,
// ensuring chunks maintain complete thoughts rather than breaking mid-sentence or mid-list.
package rag

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/tsawler/tabula/model"
)

// ChunkLevel represents the hierarchical level of a chunk
type ChunkLevel int

const (
	// ChunkLevelDocument represents the entire document as one chunk
	ChunkLevelDocument ChunkLevel = iota
	// ChunkLevelSection represents a section defined by headings
	ChunkLevelSection
	// ChunkLevelParagraph represents a single paragraph
	ChunkLevelParagraph
	// ChunkLevelSentence represents a single sentence (used for oversized paragraphs)
	ChunkLevelSentence
)

// String returns a human-readable representation of the chunk level
func (cl ChunkLevel) String() string {
	switch cl {
	case ChunkLevelDocument:
		return "document"
	case ChunkLevelSection:
		return "section"
	case ChunkLevelParagraph:
		return "paragraph"
	case ChunkLevelSentence:
		return "sentence"
	default:
		return "unknown"
	}
}

// ChunkMetadata contains rich metadata about a chunk's context within the document
type ChunkMetadata struct {
	// DocumentTitle is the title of the source document
	DocumentTitle string `json:"document_title,omitempty"`

	// SectionPath is the hierarchical path of headings (e.g., ["Chapter 1", "Introduction", "Overview"])
	SectionPath []string `json:"section_path,omitempty"`

	// SectionTitle is the immediate section heading (last element of SectionPath)
	SectionTitle string `json:"section_title,omitempty"`

	// HeadingLevel is the level of the current section (1-6, 0 if no heading)
	HeadingLevel int `json:"heading_level,omitempty"`

	// PageStart is the starting page number (1-indexed)
	PageStart int `json:"page_start"`

	// PageEnd is the ending page number (1-indexed)
	PageEnd int `json:"page_end"`

	// ChunkIndex is the position of this chunk in the document (0-indexed)
	ChunkIndex int `json:"chunk_index"`

	// TotalChunks is the total number of chunks in the document
	TotalChunks int `json:"total_chunks,omitempty"`

	// Level is the hierarchical level of this chunk
	Level ChunkLevel `json:"level"`

	// ParentID is the ID of the parent chunk (empty for top-level chunks)
	ParentID string `json:"parent_id,omitempty"`

	// ChildIDs are the IDs of child chunks
	ChildIDs []string `json:"child_ids,omitempty"`

	// ElementTypes lists the types of elements contained (paragraph, list, table, etc.)
	ElementTypes []string `json:"element_types,omitempty"`

	// HasTable indicates if the chunk contains a table
	HasTable bool `json:"has_table,omitempty"`

	// HasList indicates if the chunk contains a list
	HasList bool `json:"has_list,omitempty"`

	// HasImage indicates if the chunk contains an image
	HasImage bool `json:"has_image,omitempty"`

	// CharCount is the number of characters in the chunk text
	CharCount int `json:"char_count"`

	// WordCount is the number of words in the chunk text
	WordCount int `json:"word_count"`

	// EstimatedTokens is an estimated token count (chars/4 as rough approximation)
	EstimatedTokens int `json:"estimated_tokens"`

	// BBox is the bounding box of the chunk content on the page
	BBox *model.BBox `json:"bbox,omitempty"`
}

// Chunk represents a semantic unit of text extracted from a document for RAG
type Chunk struct {
	// ID is a unique identifier for this chunk
	ID string `json:"id"`

	// Text is the chunk content
	Text string `json:"text"`

	// TextWithContext is the text with section heading prepended for better retrieval
	TextWithContext string `json:"text_with_context,omitempty"`

	// Metadata contains rich contextual information
	Metadata ChunkMetadata `json:"metadata"`
}

// NewChunk creates a new chunk with the given text and metadata
func NewChunk(id, text string, metadata ChunkMetadata) *Chunk {
	// Calculate text statistics
	metadata.CharCount = len(text)
	metadata.WordCount = countWords(text)
	metadata.EstimatedTokens = len(text) / 4 // Rough approximation

	chunk := &Chunk{
		ID:       id,
		Text:     text,
		Metadata: metadata,
	}

	// Generate text with context
	chunk.TextWithContext = chunk.generateContextualText()

	return chunk
}

// generateContextualText creates text with section heading prepended
func (c *Chunk) generateContextualText() string {
	if c.Metadata.SectionTitle == "" {
		return c.Text
	}

	// Prepend section title for better retrieval
	return fmt.Sprintf("[%s]\n\n%s", c.Metadata.SectionTitle, c.Text)
}

// GetSectionPathString returns the section path as a formatted string
func (c *Chunk) GetSectionPathString() string {
	if len(c.Metadata.SectionPath) == 0 {
		return ""
	}
	return strings.Join(c.Metadata.SectionPath, " > ")
}

// ChunkerConfig holds configuration options for the chunker
type ChunkerConfig struct {
	// TargetChunkSize is the target size for chunks in characters
	// Default: 1000
	TargetChunkSize int

	// MaxChunkSize is the hard limit for chunk size in characters
	// Chunks will be split at sentence boundaries if they exceed this
	// Default: 2000
	MaxChunkSize int

	// MinChunkSize is the minimum size for a chunk in characters
	// Smaller chunks may be merged with adjacent content
	// Default: 100
	MinChunkSize int

	// OverlapSize is the number of characters to overlap between chunks
	// Default: 100
	OverlapSize int

	// OverlapSentences when true, uses sentence-based overlap instead of character-based
	// Default: true
	OverlapSentences bool

	// PreserveListCoherence keeps list intros with their items
	// Default: true
	PreserveListCoherence bool

	// PreserveTableCoherence keeps tables as atomic units
	// Default: true
	PreserveTableCoherence bool

	// IncludeSectionContext prepends section heading to chunk text
	// Default: true
	IncludeSectionContext bool

	// SplitOnHeadings creates new chunks at heading boundaries
	// Default: true
	SplitOnHeadings bool

	// MinHeadingLevel is the minimum heading level to split on (1-6)
	// Lower numbers = split on more headings
	// Default: 3 (split on H1, H2, H3)
	MinHeadingLevel int

	// PreserveParagraphs tries to keep paragraphs intact
	// Default: true
	PreserveParagraphs bool

	// IDPrefix is a prefix for generated chunk IDs
	// Default: "chunk"
	IDPrefix string
}

// DefaultChunkerConfig returns sensible default configuration
func DefaultChunkerConfig() ChunkerConfig {
	return ChunkerConfig{
		TargetChunkSize:        1000,
		MaxChunkSize:           2000,
		MinChunkSize:           100,
		OverlapSize:            100,
		OverlapSentences:       true,
		PreserveListCoherence:  true,
		PreserveTableCoherence: true,
		IncludeSectionContext:  true,
		SplitOnHeadings:        true,
		MinHeadingLevel:        3,
		PreserveParagraphs:     true,
		IDPrefix:               "chunk",
	}
}

// Chunker performs semantic chunking of documents
type Chunker struct {
	config ChunkerConfig
}

// NewChunker creates a new chunker with default configuration
func NewChunker() *Chunker {
	return &Chunker{
		config: DefaultChunkerConfig(),
	}
}

// NewChunkerWithConfig creates a chunker with custom configuration
func NewChunkerWithConfig(config ChunkerConfig) *Chunker {
	return &Chunker{
		config: config,
	}
}

// ChunkResult contains the chunking output
type ChunkResult struct {
	// Chunks are the generated chunks in reading order
	Chunks []*Chunk

	// DocumentTitle is the document title if available
	DocumentTitle string

	// TotalPages is the total number of pages processed
	TotalPages int

	// Statistics about the chunking process
	Stats ChunkStats
}

// ChunkStats contains statistics about the chunking process
type ChunkStats struct {
	TotalChunks     int
	TotalCharacters int
	TotalWords      int
	TotalTokensEst  int
	AvgChunkSize    int
	MinChunkSize    int
	MaxChunkSize    int
	SectionChunks   int
	ParagraphChunks int
	SentenceChunks  int
}

// Chunk processes a document and returns semantic chunks
func (c *Chunker) Chunk(doc *model.Document) (*ChunkResult, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	result := &ChunkResult{
		Chunks:        make([]*Chunk, 0),
		DocumentTitle: doc.Metadata.Title,
		TotalPages:    doc.PageCount(),
	}

	// Build document sections from headings
	sections := c.buildSections(doc)

	// Process each section into chunks
	chunkIndex := 0
	for _, section := range sections {
		sectionChunks := c.chunkSection(section, &chunkIndex, doc.Metadata.Title)
		result.Chunks = append(result.Chunks, sectionChunks...)
	}

	// If no sections were created, chunk by paragraphs
	if len(result.Chunks) == 0 {
		result.Chunks = c.chunkByParagraphs(doc, &chunkIndex)
	}

	// Calculate statistics
	result.Stats = c.calculateStats(result.Chunks)

	// Set total chunks in metadata
	for _, chunk := range result.Chunks {
		chunk.Metadata.TotalChunks = len(result.Chunks)
	}

	return result, nil
}

// Section represents a document section defined by a heading
type Section struct {
	// Heading is the section heading (nil for content before first heading)
	Heading *model.HeadingInfo

	// HeadingLevel is the heading level (0 if no heading)
	HeadingLevel int

	// Title is the section title
	Title string

	// Path is the hierarchical path of parent section titles
	Path []string

	// Content is the text content of this section
	Content []ContentElement

	// PageStart is the starting page (1-indexed)
	PageStart int

	// PageEnd is the ending page (1-indexed)
	PageEnd int

	// Children are nested subsections
	Children []*Section

	// Parent is the parent section (nil for top-level)
	Parent *Section
}

// ContentElement represents a piece of content within a section
type ContentElement struct {
	Type     model.ElementType
	Text     string
	Page     int
	BBox     model.BBox
	ListInfo *model.ListInfo
}

// buildSections constructs a hierarchical section tree from document headings
func (c *Chunker) buildSections(doc *model.Document) []*Section {
	sections := make([]*Section, 0)
	var currentPath []string
	var sectionStack []*Section

	// Track current position for content before first heading
	var preambleContent []ContentElement
	var preambleStartPage, preambleEndPage int

	for pageNum, page := range doc.Pages {
		pageIndex := pageNum + 1

		if page.Layout == nil {
			continue
		}

		// Process headings on this page
		for _, heading := range page.Layout.Headings {
			// If we have content before this heading, add it
			if len(preambleContent) > 0 && len(sectionStack) == 0 {
				// Content before first heading
				preambleSection := &Section{
					Title:     "",
					Path:      nil,
					Content:   preambleContent,
					PageStart: preambleStartPage,
					PageEnd:   preambleEndPage,
				}
				sections = append(sections, preambleSection)
				preambleContent = nil
			}

			// Create new section for this heading
			newSection := &Section{
				Heading:      &heading,
				HeadingLevel: heading.Level,
				Title:        heading.Text,
				PageStart:    pageIndex,
				PageEnd:      pageIndex,
				Content:      make([]ContentElement, 0),
			}

			// Handle section hierarchy
			if heading.Level <= c.config.MinHeadingLevel {
				// This is a major heading - update path
				// Pop stack until we find parent level
				for len(sectionStack) > 0 {
					last := sectionStack[len(sectionStack)-1]
					if last.HeadingLevel < heading.Level {
						break
					}
					sectionStack = sectionStack[:len(sectionStack)-1]
					if len(currentPath) > 0 {
						currentPath = currentPath[:len(currentPath)-1]
					}
				}

				// Update path
				currentPath = append(currentPath, heading.Text)
				newSection.Path = make([]string, len(currentPath))
				copy(newSection.Path, currentPath)

				// Set parent and add to tree
				if len(sectionStack) > 0 {
					parent := sectionStack[len(sectionStack)-1]
					newSection.Parent = parent
					parent.Children = append(parent.Children, newSection)
				} else {
					sections = append(sections, newSection)
				}

				sectionStack = append(sectionStack, newSection)
			} else {
				// Minor heading - include in current section's content
				if len(sectionStack) > 0 {
					currentSection := sectionStack[len(sectionStack)-1]
					currentSection.Content = append(currentSection.Content, ContentElement{
						Type: model.ElementTypeHeading,
						Text: heading.Text,
						Page: pageIndex,
						BBox: heading.BBox,
					})
				}
			}
		}

		// Add paragraphs to current section
		for _, para := range page.Layout.Paragraphs {
			elem := ContentElement{
				Type: model.ElementTypeParagraph,
				Text: para.Text,
				Page: pageIndex,
				BBox: para.BBox,
			}

			if len(sectionStack) > 0 {
				currentSection := sectionStack[len(sectionStack)-1]
				currentSection.Content = append(currentSection.Content, elem)
				currentSection.PageEnd = pageIndex
			} else {
				preambleContent = append(preambleContent, elem)
				if preambleStartPage == 0 {
					preambleStartPage = pageIndex
				}
				preambleEndPage = pageIndex
			}
		}

		// Add lists to current section
		for _, list := range page.Layout.Lists {
			elem := ContentElement{
				Type:     model.ElementTypeList,
				Text:     formatList(list),
				Page:     pageIndex,
				BBox:     list.BBox,
				ListInfo: &list,
			}

			if len(sectionStack) > 0 {
				currentSection := sectionStack[len(sectionStack)-1]
				currentSection.Content = append(currentSection.Content, elem)
				currentSection.PageEnd = pageIndex
			} else {
				preambleContent = append(preambleContent, elem)
				if preambleStartPage == 0 {
					preambleStartPage = pageIndex
				}
				preambleEndPage = pageIndex
			}
		}
	}

	// Handle any remaining preamble content
	if len(preambleContent) > 0 && len(sections) == 0 {
		preambleSection := &Section{
			Title:     "",
			Path:      nil,
			Content:   preambleContent,
			PageStart: preambleStartPage,
			PageEnd:   preambleEndPage,
		}
		sections = append(sections, preambleSection)
	}

	return sections
}

// chunkSection processes a section into chunks
func (c *Chunker) chunkSection(section *Section, chunkIndex *int, docTitle string) []*Chunk {
	chunks := make([]*Chunk, 0)

	// Gather all text content
	var textBuilder strings.Builder
	var elementTypes []string
	hasTable, hasList, hasImage := false, false, false
	var bbox *model.BBox

	for _, elem := range section.Content {
		if textBuilder.Len() > 0 {
			textBuilder.WriteString("\n\n")
		}
		textBuilder.WriteString(elem.Text)

		// Track element types
		elemType := elem.Type.String()
		found := false
		for _, et := range elementTypes {
			if et == elemType {
				found = true
				break
			}
		}
		if !found {
			elementTypes = append(elementTypes, elemType)
		}

		// Track content types
		switch elem.Type {
		case model.ElementTypeTable:
			hasTable = true
		case model.ElementTypeList:
			hasList = true
		case model.ElementTypeImage:
			hasImage = true
		}

		// Update bounding box
		if bbox == nil {
			bbox = &model.BBox{
				X:      elem.BBox.X,
				Y:      elem.BBox.Y,
				Width:  elem.BBox.Width,
				Height: elem.BBox.Height,
			}
		} else {
			bbox = mergeBBox(bbox, &elem.BBox)
		}
	}

	text := textBuilder.String()
	if strings.TrimSpace(text) == "" {
		return chunks
	}

	// Check if section fits in one chunk
	if len(text) <= c.config.MaxChunkSize {
		chunk := c.createChunk(text, section, *chunkIndex, docTitle, elementTypes, hasTable, hasList, hasImage, bbox)
		chunks = append(chunks, chunk)
		*chunkIndex++
		return chunks
	}

	// Section is too large - split by paragraphs
	paragraphChunks := c.splitSectionByParagraphs(section, chunkIndex, docTitle)
	chunks = append(chunks, paragraphChunks...)

	return chunks
}

// splitSectionByParagraphs splits a large section into paragraph-based chunks
func (c *Chunker) splitSectionByParagraphs(section *Section, chunkIndex *int, docTitle string) []*Chunk {
	chunks := make([]*Chunk, 0)
	var currentText strings.Builder
	var currentElements []ContentElement
	var elementTypes []string
	hasTable, hasList, hasImage := false, false, false

	flushChunk := func() {
		text := currentText.String()
		if strings.TrimSpace(text) == "" {
			return
		}

		// Calculate bounding box from elements
		var bbox *model.BBox
		for _, elem := range currentElements {
			if bbox == nil {
				bbox = &model.BBox{
					X:      elem.BBox.X,
					Y:      elem.BBox.Y,
					Width:  elem.BBox.Width,
					Height: elem.BBox.Height,
				}
			} else {
				bbox = mergeBBox(bbox, &elem.BBox)
			}
		}

		chunk := c.createChunk(text, section, *chunkIndex, docTitle, elementTypes, hasTable, hasList, hasImage, bbox)
		chunk.Metadata.Level = ChunkLevelParagraph
		chunks = append(chunks, chunk)
		*chunkIndex++

		// Reset
		currentText.Reset()
		currentElements = nil
		elementTypes = nil
		hasTable, hasList, hasImage = false, false, false
	}

	for _, elem := range section.Content {
		elemText := elem.Text

		// Check if adding this element would exceed max size
		addedLen := len(elemText)
		if currentText.Len() > 0 {
			addedLen += 2 // "\n\n"
		}

		if currentText.Len()+addedLen > c.config.MaxChunkSize && currentText.Len() > 0 {
			// Would exceed max - flush current chunk
			flushChunk()
		}

		// Handle oversized single elements
		if len(elemText) > c.config.MaxChunkSize {
			// Flush any pending content
			if currentText.Len() > 0 {
				flushChunk()
			}

			// Split by sentences
			sentenceChunks := c.splitBySentences(elemText, section, chunkIndex, docTitle, elem)
			chunks = append(chunks, sentenceChunks...)
			continue
		}

		// Add to current chunk
		if currentText.Len() > 0 {
			currentText.WriteString("\n\n")
		}
		currentText.WriteString(elemText)
		currentElements = append(currentElements, elem)

		// Track element type
		elemType := elem.Type.String()
		found := false
		for _, et := range elementTypes {
			if et == elemType {
				found = true
				break
			}
		}
		if !found {
			elementTypes = append(elementTypes, elemType)
		}

		// Track content flags
		switch elem.Type {
		case model.ElementTypeTable:
			hasTable = true
		case model.ElementTypeList:
			hasList = true
		case model.ElementTypeImage:
			hasImage = true
		}
	}

	// Flush remaining content
	flushChunk()

	return chunks
}

// splitBySentences splits oversized text into sentence-based chunks
func (c *Chunker) splitBySentences(text string, section *Section, chunkIndex *int, docTitle string, elem ContentElement) []*Chunk {
	chunks := make([]*Chunk, 0)
	sentences := splitIntoSentences(text)

	var currentText strings.Builder
	for _, sentence := range sentences {
		addedLen := len(sentence)
		if currentText.Len() > 0 {
			addedLen++ // space
		}

		if currentText.Len()+addedLen > c.config.MaxChunkSize && currentText.Len() > 0 {
			// Create chunk from current sentences
			chunkText := currentText.String()
			chunk := c.createChunk(chunkText, section, *chunkIndex, docTitle,
				[]string{elem.Type.String()}, false, false, false, &elem.BBox)
			chunk.Metadata.Level = ChunkLevelSentence
			chunks = append(chunks, chunk)
			*chunkIndex++
			currentText.Reset()
		}

		if currentText.Len() > 0 {
			currentText.WriteString(" ")
		}
		currentText.WriteString(sentence)
	}

	// Flush remaining
	if currentText.Len() > 0 {
		chunkText := currentText.String()
		chunk := c.createChunk(chunkText, section, *chunkIndex, docTitle,
			[]string{elem.Type.String()}, false, false, false, &elem.BBox)
		chunk.Metadata.Level = ChunkLevelSentence
		chunks = append(chunks, chunk)
		*chunkIndex++
	}

	return chunks
}

// createChunk creates a new Chunk with the given parameters
func (c *Chunker) createChunk(text string, section *Section, index int, docTitle string,
	elementTypes []string, hasTable, hasList, hasImage bool, bbox *model.BBox) *Chunk {

	id := fmt.Sprintf("%s_%d", c.config.IDPrefix, index)

	metadata := ChunkMetadata{
		DocumentTitle: docTitle,
		SectionPath:   section.Path,
		SectionTitle:  section.Title,
		HeadingLevel:  section.HeadingLevel,
		PageStart:     section.PageStart,
		PageEnd:       section.PageEnd,
		ChunkIndex:    index,
		Level:         ChunkLevelSection,
		ElementTypes:  elementTypes,
		HasTable:      hasTable,
		HasList:       hasList,
		HasImage:      hasImage,
		BBox:          bbox,
	}

	return NewChunk(id, text, metadata)
}

// chunkByParagraphs chunks a document without heading structure
func (c *Chunker) chunkByParagraphs(doc *model.Document, chunkIndex *int) []*Chunk {
	chunks := make([]*Chunk, 0)

	// Create a single section from all content
	section := &Section{
		Title:     doc.Metadata.Title,
		Path:      nil,
		Content:   make([]ContentElement, 0),
		PageStart: 1,
		PageEnd:   doc.PageCount(),
	}

	// Collect all paragraphs
	for pageNum, page := range doc.Pages {
		pageIndex := pageNum + 1
		if page.Layout == nil {
			continue
		}

		for _, para := range page.Layout.Paragraphs {
			section.Content = append(section.Content, ContentElement{
				Type: model.ElementTypeParagraph,
				Text: para.Text,
				Page: pageIndex,
				BBox: para.BBox,
			})
		}

		for _, list := range page.Layout.Lists {
			section.Content = append(section.Content, ContentElement{
				Type:     model.ElementTypeList,
				Text:     formatList(list),
				Page:     pageIndex,
				BBox:     list.BBox,
				ListInfo: &list,
			})
		}
	}

	if len(section.Content) == 0 {
		return chunks
	}

	return c.splitSectionByParagraphs(section, chunkIndex, doc.Metadata.Title)
}

// calculateStats computes statistics about the chunks
func (c *Chunker) calculateStats(chunks []*Chunk) ChunkStats {
	stats := ChunkStats{
		TotalChunks:  len(chunks),
		MinChunkSize: -1,
	}

	for _, chunk := range chunks {
		stats.TotalCharacters += chunk.Metadata.CharCount
		stats.TotalWords += chunk.Metadata.WordCount
		stats.TotalTokensEst += chunk.Metadata.EstimatedTokens

		if stats.MinChunkSize < 0 || chunk.Metadata.CharCount < stats.MinChunkSize {
			stats.MinChunkSize = chunk.Metadata.CharCount
		}
		if chunk.Metadata.CharCount > stats.MaxChunkSize {
			stats.MaxChunkSize = chunk.Metadata.CharCount
		}

		switch chunk.Metadata.Level {
		case ChunkLevelSection:
			stats.SectionChunks++
		case ChunkLevelParagraph:
			stats.ParagraphChunks++
		case ChunkLevelSentence:
			stats.SentenceChunks++
		}
	}

	if len(chunks) > 0 {
		stats.AvgChunkSize = stats.TotalCharacters / len(chunks)
	}

	if stats.MinChunkSize < 0 {
		stats.MinChunkSize = 0
	}

	return stats
}

// Helper functions

// countWords counts the number of words in text
func countWords(text string) int {
	words := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			words++
		}
	}
	return words
}

// splitIntoSentences splits text into sentences
func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		current.WriteRune(r)

		// Check for sentence ending
		if r == '.' || r == '!' || r == '?' {
			// Look ahead to see if this is really a sentence end
			// Skip if followed by lowercase letter (abbreviation like "e.g.")
			if i+1 < len(runes) {
				next := runes[i+1]
				if unicode.IsLower(next) {
					continue
				}
			}

			// Skip if preceded by single capital letter (abbreviation like "Mr.")
			if i > 0 && current.Len() > 1 {
				str := current.String()
				if len(str) >= 2 {
					prevChar := rune(str[len(str)-2])
					if unicode.IsUpper(prevChar) && (i < 2 || unicode.IsSpace(rune(str[len(str)-3]))) {
						continue
					}
				}
			}

			// This looks like a sentence end
			sentence := strings.TrimSpace(current.String())
			if sentence != "" {
				sentences = append(sentences, sentence)
			}
			current.Reset()
		}
	}

	// Add any remaining text
	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		sentences = append(sentences, remaining)
	}

	return sentences
}

// formatList formats a list for text output
func formatList(list model.ListInfo) string {
	var sb strings.Builder
	for i, item := range list.Items {
		if i > 0 {
			sb.WriteString("\n")
		}
		// Add indentation for nested items
		for j := 0; j < item.Level; j++ {
			sb.WriteString("  ")
		}
		sb.WriteString("- ")
		sb.WriteString(item.Text)
	}
	return sb.String()
}

// mergeBBox merges two bounding boxes into one that contains both
func mergeBBox(a, b *model.BBox) *model.BBox {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	minX := a.X
	if b.X < minX {
		minX = b.X
	}

	minY := a.Y
	if b.Y < minY {
		minY = b.Y
	}

	maxX := a.X + a.Width
	if b.X+b.Width > maxX {
		maxX = b.X + b.Width
	}

	maxY := a.Y + a.Height
	if b.Y+b.Height > maxY {
		maxY = b.Y + b.Height
	}

	return &model.BBox{
		X:      minX,
		Y:      minY,
		Width:  maxX - minX,
		Height: maxY - minY,
	}
}
