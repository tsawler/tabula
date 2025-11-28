package tabula_test

import (
	"fmt"
	"log"

	"github.com/tsawler/tabula"
	"github.com/tsawler/tabula/rag"
	"github.com/tsawler/tabula/reader"
)

// These examples verify the README code samples compile correctly.
// They are not meant to be run as actual tests since they require files.

func Example_extractText() {
	// Works with both PDF and DOCX files
	text, warnings, err := tabula.Open("document.pdf").Text()
	// text, warnings, err := tabula.Open("document.docx").Text()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(text)

	for _, w := range warnings {
		fmt.Println("Warning:", w.Message)
	}
}

func Example_extractWithOptions() {
	text, warnings, err := tabula.Open("document.pdf").
		Pages(1, 2, 3).             // Specific pages (PDF only)
		ExcludeHeadersAndFooters(). // Remove repeating headers/footers (PDF only)
		JoinParagraphs().           // Join text into paragraphs (PDF only)
		Text()
	_ = text
	_ = warnings
	_ = err
}

func Example_extractMarkdown() {
	// PDF with header/footer exclusion
	markdown, warnings, err := tabula.Open("document.pdf").
		ExcludeHeadersAndFooters().
		ToMarkdown()
	_ = markdown
	_ = warnings
	_ = err

	// DOCX (preserves headings, lists, tables)
	markdown, warnings, err = tabula.Open("document.docx").ToMarkdown()
	_ = markdown
	_ = warnings
	_ = err
}

func Example_ragChunking() {
	// Works with both PDF and DOCX
	chunks, warnings, err := tabula.Open("document.pdf").Chunks()
	// chunks, warnings, err := tabula.Open("document.docx").Chunks()
	if err != nil {
		log.Fatal(err)
	}

	for i, chunk := range chunks.Chunks {
		fmt.Printf("Chunk %d: %s (p.%d-%d, ~%d tokens)\n",
			i+1,
			chunk.Metadata.SectionTitle,
			chunk.Metadata.PageStart,
			chunk.Metadata.PageEnd,
			chunk.Metadata.EstimatedTokens)
		fmt.Println(chunk.Text)
		fmt.Println("---")
	}

	// Warnings are non-fatal issues
	for _, w := range warnings {
		fmt.Println("Warning:", w.Message)
	}
}

func Example_chunksAsMarkdown() {
	chunks, _, err := tabula.Open("document.pdf").
		ExcludeHeadersAndFooters().
		Chunks()
	if err != nil {
		log.Fatal(err)
	}

	// Get each chunk as separate markdown strings
	mdChunks := chunks.ToMarkdownChunks()

	for i, md := range mdChunks {
		// Example: store each chunk in your vector database
		_ = chunks.Chunks[i].ID
		_ = md
	}
}

func Example_openDocuments() {
	// From file path (format auto-detected by extension)
	ext := tabula.Open("document.pdf")
	_ = ext
	ext = tabula.Open("document.docx")
	_ = ext

	// From existing PDF reader (PDF only)
	r, _ := reader.Open("document.pdf")
	ext = tabula.FromReader(r)
	_ = ext
}

func Example_chunkFiltering() {
	chunks, _, _ := tabula.Open("doc.pdf").Chunks()

	// Filter by content type
	tablesOnly := chunks.FilterWithTables()
	listsOnly := chunks.FilterWithLists()
	_ = tablesOnly
	_ = listsOnly

	// Filter by location
	section := chunks.FilterBySection("Introduction")
	page5 := chunks.FilterByPage(5)
	pages1to10 := chunks.FilterByPageRange(1, 10)
	_ = section
	_ = page5
	_ = pages1to10

	// Filter by size
	smallChunks := chunks.FilterByMaxTokens(500)
	largeChunks := chunks.FilterByMinTokens(100)
	_ = smallChunks
	_ = largeChunks

	// Search
	matches := chunks.Search("keyword")
	_ = matches

	// Chain filters
	result := chunks.
		FilterBySection("Methods").
		FilterByMinTokens(100).
		Search("algorithm")
	_ = result
}

func Example_markdownOptions() {
	opts := rag.MarkdownOptions{
		IncludeMetadata:        true, // YAML front matter
		IncludeTableOfContents: true, // Generated TOC
		IncludeChunkSeparators: true, // --- between chunks
		IncludePageNumbers:     true, // Page references
		IncludeChunkIDs:        true, // HTML comments with chunk IDs
	}

	markdown, _, _ := tabula.Open("doc.pdf").ToMarkdownWithOptions(opts)
	_ = markdown

	// Or use preset for RAG
	opts = rag.RAGOptimizedMarkdownOptions()
	_ = opts
}

func Example_customChunkSizing() {
	config := rag.ChunkerConfig{
		TargetChunkSize: 500,  // Target characters per chunk
		MaxChunkSize:    1000, // Maximum characters
		MinChunkSize:    100,  // Minimum characters
		OverlapSize:     50,   // Overlap between chunks
	}
	sizeConfig := rag.DefaultSizeConfig()

	chunks, _, _ := tabula.Open("doc.pdf").ChunksWithConfig(config, sizeConfig)
	_ = chunks
}

func Example_chunkMetadata() {
	chunks, _, _ := tabula.Open("doc.pdf").Chunks()

	for _, chunk := range chunks.Chunks {
		fmt.Println("ID:", chunk.ID)
		fmt.Println("Section:", chunk.Metadata.SectionTitle)
		fmt.Println("Pages:", chunk.Metadata.PageStart, "-", chunk.Metadata.PageEnd)
		fmt.Println("Words:", chunk.Metadata.WordCount)
		fmt.Println("Tokens:", chunk.Metadata.EstimatedTokens)
		fmt.Println("Has Table:", chunk.Metadata.HasTable)
		fmt.Println("Has List:", chunk.Metadata.HasList)
	}
}

func Example_collectionStatistics() {
	chunks, _, _ := tabula.Open("doc.pdf").Chunks()

	stats := chunks.Statistics()
	fmt.Println("Total chunks:", stats.TotalChunks)
	fmt.Println("Total words:", stats.TotalWords)
	fmt.Println("Average tokens:", stats.AvgTokens)
	fmt.Println("Chunks with tables:", stats.ChunksWithTables)
}

func Example_warnings() {
	text, warnings, err := tabula.Open("document.pdf").Text()
	if err != nil {
		log.Fatal(err) // Fatal error
	}
	_ = text

	for _, w := range warnings {
		log.Println("Warning:", w.Message) // Non-fatal issues
	}

	// Format all warnings
	formatted := tabula.FormatWarnings(warnings)
	_ = formatted
}

func Example_errorHandling() {
	// Panic on error (for scripts/tests)
	text := tabula.MustText(tabula.Open("doc.pdf").Text())
	count := tabula.Must(tabula.Open("doc.pdf").PageCount())
	_ = text
	_ = count
}

func Example_inspectionMethods() {
	ext := tabula.Open("document.pdf")
	defer ext.Close()

	isCharLevel, _ := ext.IsCharacterLevel() // Detect character-level PDFs
	isMultiCol, _ := ext.IsMultiColumn()     // Detect multi-column layouts
	pageCount, _ := ext.PageCount()          // Get page count (works with DOCX too)
	_ = isCharLevel
	_ = isMultiCol
	_ = pageCount
}
