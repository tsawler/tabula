package rag

import (
	"fmt"
	"strings"

	"github.com/tsawler/tabula/model"
)

// DocumentChunker provides RAG chunking for Document objects
type DocumentChunker struct {
	config       ChunkerConfig
	sizeConfig   SizeConfig
	boundaries   *BoundaryDetector
	listHandler  *ListCoherenceAnalyzer
	tableHandler *TableFigureHandler
}

// NewDocumentChunker creates a new document chunker with default configuration
func NewDocumentChunker() *DocumentChunker {
	return &DocumentChunker{
		config:       DefaultChunkerConfig(),
		sizeConfig:   DefaultSizeConfig(),
		boundaries:   NewBoundaryDetector(),
		listHandler:  NewListCoherenceAnalyzer(),
		tableHandler: NewTableFigureHandler(),
	}
}

// NewDocumentChunkerWithConfig creates a document chunker with custom configuration
func NewDocumentChunkerWithConfig(config ChunkerConfig, sizeConfig SizeConfig) *DocumentChunker {
	return &DocumentChunker{
		config:       config,
		sizeConfig:   sizeConfig,
		boundaries:   NewBoundaryDetector(),
		listHandler:  NewListCoherenceAnalyzer(),
		tableHandler: NewTableFigureHandler(),
	}
}

// ChunkDocument chunks a Document into semantic units for RAG
func (dc *DocumentChunker) ChunkDocument(doc *model.Document) *ChunkCollection {
	if doc == nil {
		return NewChunkCollection([]*Chunk{})
	}

	var chunks []*Chunk
	chunkIndex := 0

	// Extract document title (leave empty if not set)
	docTitle := doc.Metadata.Title

	// Build section context from headings
	toc := doc.TableOfContents()
	currentSection := []string{}
	currentHeadingLevel := 0

	// Process each page
	for _, page := range doc.Pages {
		pageChunks := dc.chunkPage(page, docTitle, &currentSection, &currentHeadingLevel, toc, &chunkIndex)
		chunks = append(chunks, pageChunks...)
	}

	// Set total chunks count
	for _, chunk := range chunks {
		chunk.Metadata.TotalChunks = len(chunks)
	}

	return NewChunkCollection(chunks)
}

// chunkPage chunks a single page
func (dc *DocumentChunker) chunkPage(page *model.Page, docTitle string, currentSection *[]string, currentHeadingLevel *int, toc []model.TOCEntry, chunkIndex *int) []*Chunk {
	var chunks []*Chunk

	if page == nil {
		return chunks
	}

	// Process elements maintaining document order
	var currentBlock textBlock
	currentBlock.pageNum = page.Number

	// Helper to flush current text block as chunks
	flushTextBlock := func() {
		if currentBlock.text != "" {
			blockChunks := dc.textBlockToChunks(currentBlock, docTitle, chunkIndex)
			chunks = append(chunks, blockChunks...)
			currentBlock = textBlock{pageNum: page.Number}
		}
	}

	for _, elem := range page.Elements {
		switch e := elem.(type) {
		case *model.Paragraph:
			// Update section context if this is a heading-like paragraph
			if isHeadingElement(e.Text, toc, page.Number) {
				// Flush current block before heading
				flushTextBlock()

				// Update section path
				headingLevel := getHeadingLevel(e.Text, toc, page.Number)
				updateSectionPath(currentSection, currentHeadingLevel, headingLevel, e.Text)

				// Create heading chunk
				chunk := dc.createHeadingChunk(e.Text, docTitle, *currentSection, headingLevel, page.Number, chunkIndex)
				chunks = append(chunks, chunk)
			} else {
				// Accumulate text
				if currentBlock.text != "" {
					currentBlock.text += "\n\n"
				}
				currentBlock.text += e.Text
				currentBlock.sectionPath = append([]string{}, *currentSection...)
				currentBlock.elementTypes = appendUnique(currentBlock.elementTypes, "paragraph")
			}

		case *model.Heading:
			// Flush current block before heading
			flushTextBlock()

			// Update section path
			updateSectionPath(currentSection, currentHeadingLevel, e.Level, e.Text)

			// Create heading chunk
			chunk := dc.createChunkFromHeading(e, docTitle, *currentSection, page.Number, chunkIndex)
			chunks = append(chunks, chunk)

		case *model.List:
			// Flush current block before list
			flushTextBlock()

			// Create list chunk
			chunk := dc.createListChunk(e, docTitle, *currentSection, page.Number, chunkIndex)
			chunks = append(chunks, chunk)

		case *model.Table:
			// Flush current block before table
			flushTextBlock()

			// Create table chunk
			chunk := dc.createTableChunk(e, docTitle, *currentSection, page.Number, chunkIndex)
			chunks = append(chunks, chunk)

		case *model.Image:
			// Flush current block before image
			flushTextBlock()

			// Create image chunk if it has alt text
			if e.AltText != "" {
				chunk := dc.createImageChunk(e, docTitle, *currentSection, page.Number, chunkIndex)
				chunks = append(chunks, chunk)
			}
		}
	}

	// Flush final block
	flushTextBlock()

	return chunks
}

// textBlock represents accumulated text content
type textBlock struct {
	text         string
	sectionPath  []string
	elementTypes []string
	pageNum      int
}

// textBlockToChunks converts a text block to one or more chunks
func (dc *DocumentChunker) textBlockToChunks(block textBlock, docTitle string, chunkIndex *int) []*Chunk {
	var chunks []*Chunk

	sizeCalc := NewSizeCalculatorWithConfig(dc.sizeConfig)

	// If block fits within max size, create single chunk
	if !sizeCalc.IsAboveMax(block.text) {
		chunk := dc.createTextChunk(block, docTitle, chunkIndex)
		chunks = append(chunks, chunk)
		return chunks
	}

	// Split into multiple chunks
	texts := sizeCalc.SplitToSize(block.text, nil)
	for _, text := range texts {
		subBlock := textBlock{
			text:         text,
			sectionPath:  block.sectionPath,
			elementTypes: block.elementTypes,
			pageNum:      block.pageNum,
		}
		chunk := dc.createTextChunk(subBlock, docTitle, chunkIndex)
		chunks = append(chunks, chunk)
	}

	return chunks
}

// createTextChunk creates a chunk from a text block
func (dc *DocumentChunker) createTextChunk(block textBlock, docTitle string, chunkIndex *int) *Chunk {
	sectionTitle := ""
	if len(block.sectionPath) > 0 {
		sectionTitle = block.sectionPath[len(block.sectionPath)-1]
	}

	chunk := &Chunk{
		ID:   fmt.Sprintf("chunk-%d", *chunkIndex),
		Text: strings.TrimSpace(block.text),
		Metadata: ChunkMetadata{
			DocumentTitle: docTitle,
			SectionPath:   block.sectionPath,
			SectionTitle:  sectionTitle,
			PageStart:     block.pageNum,
			PageEnd:       block.pageNum,
			ChunkIndex:    *chunkIndex,
			Level:         ChunkLevelParagraph,
			ElementTypes:  block.elementTypes,
			CharCount:     len(block.text),
			WordCount:     countWords(block.text),
		},
	}

	*chunkIndex++
	return chunk
}

// createHeadingChunk creates a chunk from a heading-like paragraph
func (dc *DocumentChunker) createHeadingChunk(text string, docTitle string, sectionPath []string, level int, pageNum int, chunkIndex *int) *Chunk {
	sectionTitle := ""
	if len(sectionPath) > 0 {
		sectionTitle = sectionPath[len(sectionPath)-1]
	}

	chunk := &Chunk{
		ID:   fmt.Sprintf("chunk-%d", *chunkIndex),
		Text: text,
		Metadata: ChunkMetadata{
			DocumentTitle: docTitle,
			SectionPath:   sectionPath,
			SectionTitle:  sectionTitle,
			HeadingLevel:  level,
			PageStart:     pageNum,
			PageEnd:       pageNum,
			ChunkIndex:    *chunkIndex,
			Level:         ChunkLevelSection,
			ElementTypes:  []string{"heading"},
			CharCount:     len(text),
			WordCount:     countWords(text),
		},
	}

	*chunkIndex++
	return chunk
}

// createChunkFromHeading creates a chunk from a Heading element
func (dc *DocumentChunker) createChunkFromHeading(h *model.Heading, docTitle string, sectionPath []string, pageNum int, chunkIndex *int) *Chunk {
	text := h.Text
	sectionTitle := ""
	if len(sectionPath) > 0 {
		sectionTitle = sectionPath[len(sectionPath)-1]
	}

	chunk := &Chunk{
		ID:   fmt.Sprintf("chunk-%d", *chunkIndex),
		Text: text,
		Metadata: ChunkMetadata{
			DocumentTitle: docTitle,
			SectionPath:   sectionPath,
			SectionTitle:  sectionTitle,
			HeadingLevel:  h.Level,
			PageStart:     pageNum,
			PageEnd:       pageNum,
			ChunkIndex:    *chunkIndex,
			Level:         ChunkLevelSection,
			ElementTypes:  []string{"heading"},
			CharCount:     len(text),
			WordCount:     countWords(text),
		},
	}

	*chunkIndex++
	return chunk
}

// createListChunk creates a chunk from a List element
func (dc *DocumentChunker) createListChunk(list *model.List, docTitle string, sectionPath []string, pageNum int, chunkIndex *int) *Chunk {
	// Format list as markdown with proper indentation for nesting
	var sb strings.Builder
	levelCounters := make(map[int]int) // Track counters per level for ordered lists
	lastLevel := -1

	for _, item := range list.Items {
		// Reset child counters when going back to parent level
		if item.Level <= lastLevel {
			for lvl := range levelCounters {
				if lvl > item.Level {
					delete(levelCounters, lvl)
				}
			}
		}
		lastLevel = item.Level

		// Add indentation (2 spaces per level)
		for j := 0; j < item.Level; j++ {
			sb.WriteString("  ")
		}

		if list.Ordered {
			levelCounters[item.Level]++
			sb.WriteString(fmt.Sprintf("%d. %s\n", levelCounters[item.Level], item.Text))
		} else {
			sb.WriteString(fmt.Sprintf("- %s\n", item.Text))
		}
	}
	text := strings.TrimSpace(sb.String())

	sectionTitle := ""
	if len(sectionPath) > 0 {
		sectionTitle = sectionPath[len(sectionPath)-1]
	}

	chunk := &Chunk{
		ID:   fmt.Sprintf("chunk-%d", *chunkIndex),
		Text: text,
		Metadata: ChunkMetadata{
			DocumentTitle: docTitle,
			SectionPath:   sectionPath,
			SectionTitle:  sectionTitle,
			PageStart:     pageNum,
			PageEnd:       pageNum,
			ChunkIndex:    *chunkIndex,
			Level:         ChunkLevelParagraph,
			HasList:       true,
			ElementTypes:  []string{"list"},
			CharCount:     len(text),
			WordCount:     countWords(text),
		},
	}

	*chunkIndex++
	return chunk
}

// createTableChunk creates a chunk from a Table element
func (dc *DocumentChunker) createTableChunk(table *model.Table, docTitle string, sectionPath []string, pageNum int, chunkIndex *int) *Chunk {
	// Convert table to markdown
	text := table.ToMarkdown()

	sectionTitle := ""
	if len(sectionPath) > 0 {
		sectionTitle = sectionPath[len(sectionPath)-1]
	}

	chunk := &Chunk{
		ID:   fmt.Sprintf("chunk-%d", *chunkIndex),
		Text: text,
		Metadata: ChunkMetadata{
			DocumentTitle: docTitle,
			SectionPath:   sectionPath,
			SectionTitle:  sectionTitle,
			PageStart:     pageNum,
			PageEnd:       pageNum,
			ChunkIndex:    *chunkIndex,
			Level:         ChunkLevelParagraph,
			HasTable:      true,
			ElementTypes:  []string{"table"},
			CharCount:     len(text),
			WordCount:     countWords(text),
		},
	}

	*chunkIndex++
	return chunk
}

// createImageChunk creates a chunk from an Image element
func (dc *DocumentChunker) createImageChunk(img *model.Image, docTitle string, sectionPath []string, pageNum int, chunkIndex *int) *Chunk {
	// Format image reference as text
	text := "[Image: " + img.AltText + "]"

	sectionTitle := ""
	if len(sectionPath) > 0 {
		sectionTitle = sectionPath[len(sectionPath)-1]
	}

	chunk := &Chunk{
		ID:   fmt.Sprintf("chunk-%d", *chunkIndex),
		Text: text,
		Metadata: ChunkMetadata{
			DocumentTitle: docTitle,
			SectionPath:   sectionPath,
			SectionTitle:  sectionTitle,
			PageStart:     pageNum,
			PageEnd:       pageNum,
			ChunkIndex:    *chunkIndex,
			Level:         ChunkLevelParagraph,
			HasImage:      true,
			ElementTypes:  []string{"image"},
			CharCount:     len(text),
			WordCount:     countWords(text),
		},
	}

	*chunkIndex++
	return chunk
}

// Helper functions

// isHeadingElement checks if text matches a TOC entry (is a heading)
func isHeadingElement(text string, toc []model.TOCEntry, pageNum int) bool {
	text = strings.TrimSpace(text)

	// Check against TOC entries
	for _, entry := range toc {
		if entry.Page == pageNum && strings.TrimSpace(entry.Text) == text {
			return true
		}
	}

	return false
}

// getHeadingLevel returns the heading level for text matching a TOC entry
func getHeadingLevel(text string, toc []model.TOCEntry, pageNum int) int {
	text = strings.TrimSpace(text)

	for _, entry := range toc {
		if entry.Page == pageNum && strings.TrimSpace(entry.Text) == text {
			return entry.Level
		}
	}

	return 1 // Default to level 1
}

// updateSectionPath updates the section path based on heading level
func updateSectionPath(sectionPath *[]string, currentLevel *int, newLevel int, headingText string) {
	headingText = strings.TrimSpace(headingText)

	if newLevel <= *currentLevel {
		// Pop sections until we're at the right level
		for len(*sectionPath) >= newLevel {
			if len(*sectionPath) > 0 {
				*sectionPath = (*sectionPath)[:len(*sectionPath)-1]
			} else {
				break
			}
		}
	}

	// Add new section
	*sectionPath = append(*sectionPath, headingText)
	*currentLevel = newLevel
}

// appendUnique appends an item to a slice only if not already present
func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

// ChunkDocument is a convenience function to chunk a document with default settings
func ChunkDocument(doc *model.Document) *ChunkCollection {
	chunker := NewDocumentChunker()
	return chunker.ChunkDocument(doc)
}

// ChunkDocumentWithConfig chunks a document with custom configuration
func ChunkDocumentWithConfig(doc *model.Document, config ChunkerConfig, sizeConfig SizeConfig) *ChunkCollection {
	chunker := NewDocumentChunkerWithConfig(config, sizeConfig)
	return chunker.ChunkDocument(doc)
}

// Document extension methods - these can be called on model.Document

// DocumentChunkOptions holds options for document chunking
type DocumentChunkOptions struct {
	ChunkerConfig ChunkerConfig
	SizeConfig    SizeConfig
}

// DefaultDocumentChunkOptions returns default chunking options
func DefaultDocumentChunkOptions() DocumentChunkOptions {
	return DocumentChunkOptions{
		ChunkerConfig: DefaultChunkerConfig(),
		SizeConfig:    DefaultSizeConfig(),
	}
}

// RAGOptimizedOptions returns options optimized for RAG workflows
func RAGOptimizedOptions() DocumentChunkOptions {
	return DocumentChunkOptions{
		ChunkerConfig: ChunkerConfig{
			MinChunkSize:    100,
			MaxChunkSize:    1000,
			TargetChunkSize: 500,
			OverlapSize:     50,
		},
		SizeConfig: OpenAIEmbeddingConfig(),
	}
}
