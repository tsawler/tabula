// Package rag provides RAG (Retrieval-Augmented Generation) chunking and export
// functionality for LLM integration.
//
// This package prepares extracted document content for use with large language
// models by providing semantic chunking and various export formats.
//
// # Chunking
//
// The [Chunker] splits documents into semantically meaningful chunks:
//
//	chunker := rag.NewChunker(config)
//	chunks := chunker.ChunkDocument(document)
//
// Chunking respects document structure, avoiding splits in the middle of:
//   - Tables
//   - Lists
//   - Paragraphs
//   - Headings with their following content
//
// # Chunk Configuration
//
// Use [ChunkerConfig] to control chunking behavior:
//
//   - MaxChunkSize - maximum tokens/characters per chunk
//   - MinChunkSize - minimum chunk size (avoids tiny chunks)
//   - Overlap - overlap between consecutive chunks
//   - PreserveStructure - keep tables and lists intact
//
// # Chunk Metadata
//
// Each [Chunk] includes metadata for retrieval:
//
//   - Page numbers and positions
//   - Section headings
//   - Content type (paragraph, table, list, etc.)
//   - Relationships to other chunks
//
// # Export Formats
//
// Export chunks in various formats:
//
//   - ToMarkdown() - Markdown with preserved structure
//   - ToPlainText() - Plain text extraction
//   - ToJSON() - Structured JSON output
//
// # Markdown Export
//
// The [MarkdownOptions] control markdown generation:
//
//   - IncludeMetadata - add front matter
//   - PreserveTables - use markdown table syntax
//   - HeadingStyle - ATX (#) or Setext (===) headings
package rag
