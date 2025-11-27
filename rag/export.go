package rag

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// ExportFormat defines the available export formats
type ExportFormat int

const (
	// ExportFormatJSONL exports as JSON Lines (one JSON object per line)
	ExportFormatJSONL ExportFormat = iota
	// ExportFormatJSON exports as a JSON array
	ExportFormatJSON
	// ExportFormatCSV exports as comma-separated values
	ExportFormatCSV
	// ExportFormatTSV exports as tab-separated values
	ExportFormatTSV
)

// String returns a human-readable representation of the export format
func (ef ExportFormat) String() string {
	switch ef {
	case ExportFormatJSONL:
		return "jsonl"
	case ExportFormatJSON:
		return "json"
	case ExportFormatCSV:
		return "csv"
	case ExportFormatTSV:
		return "tsv"
	default:
		return "unknown"
	}
}

// FileExtension returns the typical file extension for this format
func (ef ExportFormat) FileExtension() string {
	switch ef {
	case ExportFormatJSONL:
		return ".jsonl"
	case ExportFormatJSON:
		return ".json"
	case ExportFormatCSV:
		return ".csv"
	case ExportFormatTSV:
		return ".tsv"
	default:
		return ".txt"
	}
}

// ExportConfig holds configuration options for export
type ExportConfig struct {
	// Format specifies the export format
	Format ExportFormat

	// IncludeMetadata determines which metadata fields to include
	IncludeMetadata bool

	// MetadataFields specifies which metadata fields to include (nil = all)
	MetadataFields []string

	// IncludeText includes the chunk text content
	IncludeText bool

	// IncludeEmbeddings includes embedding vectors if present
	IncludeEmbeddings bool

	// FlattenMetadata flattens nested metadata into dot-notation keys
	FlattenMetadata bool

	// CSVDelimiter specifies the delimiter for CSV export (default: comma)
	CSVDelimiter rune

	// IncludeHeader includes header row in CSV/TSV exports
	IncludeHeader bool

	// PrettyPrint enables pretty printing for JSON formats
	PrettyPrint bool

	// TextColumnName specifies the column name for text content
	TextColumnName string

	// ChunkIDColumnName specifies the column name for chunk ID
	ChunkIDColumnName string
}

// DefaultExportConfig returns sensible defaults for export configuration
func DefaultExportConfig() ExportConfig {
	return ExportConfig{
		Format:            ExportFormatJSONL,
		IncludeMetadata:   true,
		MetadataFields:    nil, // all fields
		IncludeText:       true,
		IncludeEmbeddings: false,
		FlattenMetadata:   false,
		CSVDelimiter:      ',',
		IncludeHeader:     true,
		PrettyPrint:       false,
		TextColumnName:    "text",
		ChunkIDColumnName: "chunk_id",
	}
}

// JSONLExportConfig returns config optimized for JSON Lines export
func JSONLExportConfig() ExportConfig {
	config := DefaultExportConfig()
	config.Format = ExportFormatJSONL
	return config
}

// CSVExportConfig returns config optimized for CSV export
func CSVExportConfig() ExportConfig {
	config := DefaultExportConfig()
	config.Format = ExportFormatCSV
	config.FlattenMetadata = true
	return config
}

// TSVExportConfig returns config optimized for TSV export
func TSVExportConfig() ExportConfig {
	config := DefaultExportConfig()
	config.Format = ExportFormatTSV
	config.FlattenMetadata = true
	config.CSVDelimiter = '\t'
	return config
}

// VectorDBExportConfig returns config optimized for vector DB ingestion
func VectorDBExportConfig() ExportConfig {
	return ExportConfig{
		Format:          ExportFormatJSONL,
		IncludeMetadata: true,
		MetadataFields: []string{
			"document_title", "page_start", "chunk_index",
			"section_title", "section_path", "element_types",
		},
		IncludeText:       true,
		IncludeEmbeddings: true,
		FlattenMetadata:   false,
		PrettyPrint:       false,
		TextColumnName:    "text",
		ChunkIDColumnName: "id",
	}
}

// Exporter handles exporting chunks to various formats
type Exporter struct {
	config ExportConfig
}

// NewExporter creates a new exporter with default configuration
func NewExporter() *Exporter {
	return &Exporter{
		config: DefaultExportConfig(),
	}
}

// NewExporterWithConfig creates an exporter with custom configuration
func NewExporterWithConfig(config ExportConfig) *Exporter {
	return &Exporter{
		config: config,
	}
}

// ExportedChunk represents a chunk prepared for export
type ExportedChunk struct {
	// ID is the unique identifier for the chunk
	ID string `json:"id,omitempty"`

	// Text is the chunk content
	Text string `json:"text,omitempty"`

	// Metadata holds all metadata fields as a map
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Embeddings holds the embedding vector(s) if present
	Embeddings []float64 `json:"embeddings,omitempty"`

	// Source document information
	DocumentTitle string `json:"document_title,omitempty"`
	PageStart     int    `json:"page_start,omitempty"`
	PageEnd       int    `json:"page_end,omitempty"`

	// Position within the document
	ChunkIndex int `json:"chunk_index,omitempty"`

	// Section information
	SectionTitle string   `json:"section_title,omitempty"`
	SectionPath  []string `json:"section_path,omitempty"`

	// Content indicators
	HasTable bool `json:"has_table,omitempty"`
	HasList  bool `json:"has_list,omitempty"`
	HasImage bool `json:"has_image,omitempty"`
}

// Export exports chunks to the specified writer
func (e *Exporter) Export(chunks []*Chunk, w io.Writer) error {
	switch e.config.Format {
	case ExportFormatJSONL:
		return e.exportJSONL(chunks, w)
	case ExportFormatJSON:
		return e.exportJSON(chunks, w)
	case ExportFormatCSV, ExportFormatTSV:
		return e.exportCSV(chunks, w)
	default:
		return fmt.Errorf("unsupported export format: %v", e.config.Format)
	}
}

// ExportToFile exports chunks to a file
func (e *Exporter) ExportToFile(chunks []*Chunk, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating export file: %w", err)
	}
	defer f.Close()

	return e.Export(chunks, f)
}

// ExportToString exports chunks to a string
func (e *Exporter) ExportToString(chunks []*Chunk) (string, error) {
	var buf bytes.Buffer
	if err := e.Export(chunks, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// chunkMetadataToMap converts ChunkMetadata struct to map for flexible export
func chunkMetadataToMap(meta ChunkMetadata) map[string]interface{} {
	m := make(map[string]interface{})

	if meta.DocumentTitle != "" {
		m["document_title"] = meta.DocumentTitle
	}
	if len(meta.SectionPath) > 0 {
		m["section_path"] = meta.SectionPath
	}
	if meta.SectionTitle != "" {
		m["section_title"] = meta.SectionTitle
	}
	if meta.HeadingLevel > 0 {
		m["heading_level"] = meta.HeadingLevel
	}
	if meta.PageStart > 0 {
		m["page_start"] = meta.PageStart
	}
	if meta.PageEnd > 0 {
		m["page_end"] = meta.PageEnd
	}
	m["chunk_index"] = meta.ChunkIndex
	if meta.TotalChunks > 0 {
		m["total_chunks"] = meta.TotalChunks
	}
	m["level"] = meta.Level.String()
	if meta.ParentID != "" {
		m["parent_id"] = meta.ParentID
	}
	if len(meta.ChildIDs) > 0 {
		m["child_ids"] = meta.ChildIDs
	}
	if len(meta.ElementTypes) > 0 {
		m["element_types"] = meta.ElementTypes
	}
	if meta.HasTable {
		m["has_table"] = true
	}
	if meta.HasList {
		m["has_list"] = true
	}
	if meta.HasImage {
		m["has_image"] = true
	}
	if meta.CharCount > 0 {
		m["char_count"] = meta.CharCount
	}
	if meta.WordCount > 0 {
		m["word_count"] = meta.WordCount
	}
	if meta.EstimatedTokens > 0 {
		m["estimated_tokens"] = meta.EstimatedTokens
	}

	return m
}

// prepareChunkForExport converts a Chunk to an ExportedChunk
func (e *Exporter) prepareChunkForExport(chunk *Chunk, index int) ExportedChunk {
	exported := ExportedChunk{
		ID:            chunk.ID,
		ChunkIndex:    chunk.Metadata.ChunkIndex,
		DocumentTitle: chunk.Metadata.DocumentTitle,
		PageStart:     chunk.Metadata.PageStart,
		PageEnd:       chunk.Metadata.PageEnd,
		SectionTitle:  chunk.Metadata.SectionTitle,
		SectionPath:   chunk.Metadata.SectionPath,
		HasTable:      chunk.Metadata.HasTable,
		HasList:       chunk.Metadata.HasList,
		HasImage:      chunk.Metadata.HasImage,
	}

	if e.config.IncludeText {
		exported.Text = chunk.Text
	}

	if e.config.IncludeMetadata {
		metaMap := chunkMetadataToMap(chunk.Metadata)
		exported.Metadata = e.filterMetadata(metaMap)
	}

	return exported
}

// filterMetadata filters metadata based on configuration
func (e *Exporter) filterMetadata(metadata map[string]interface{}) map[string]interface{} {
	if e.config.MetadataFields == nil {
		if e.config.FlattenMetadata {
			return flattenMetadata(metadata, "")
		}
		return metadata
	}

	filtered := make(map[string]interface{})
	for _, field := range e.config.MetadataFields {
		if val, ok := metadata[field]; ok {
			filtered[field] = val
		}
	}

	if e.config.FlattenMetadata {
		return flattenMetadata(filtered, "")
	}
	return filtered
}

// flattenMetadata flattens nested maps into dot-notation keys
func flattenMetadata(data map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// Recursively flatten nested maps
			nested := flattenMetadata(v, fullKey)
			for nk, nv := range nested {
				result[nk] = nv
			}
		default:
			result[fullKey] = value
		}
	}

	return result
}

// exportJSONL exports chunks as JSON Lines (one JSON object per line)
func (e *Exporter) exportJSONL(chunks []*Chunk, w io.Writer) error {
	encoder := json.NewEncoder(w)
	if e.config.PrettyPrint {
		encoder.SetIndent("", "  ")
	}

	for i, chunk := range chunks {
		exported := e.prepareChunkForExport(chunk, i)
		if err := encoder.Encode(exported); err != nil {
			return fmt.Errorf("encoding chunk %d: %w", i, err)
		}
	}

	return nil
}

// exportJSON exports chunks as a JSON array
func (e *Exporter) exportJSON(chunks []*Chunk, w io.Writer) error {
	exported := make([]ExportedChunk, len(chunks))
	for i, chunk := range chunks {
		exported[i] = e.prepareChunkForExport(chunk, i)
	}

	encoder := json.NewEncoder(w)
	if e.config.PrettyPrint {
		encoder.SetIndent("", "  ")
	}

	return encoder.Encode(exported)
}

// exportCSV exports chunks as CSV or TSV
func (e *Exporter) exportCSV(chunks []*Chunk, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = e.config.CSVDelimiter

	// Collect all possible columns from all chunks
	columns := e.collectCSVColumns(chunks)

	// Write header
	if e.config.IncludeHeader {
		if err := csvWriter.Write(columns); err != nil {
			return fmt.Errorf("writing CSV header: %w", err)
		}
	}

	// Write data rows
	for i, chunk := range chunks {
		exported := e.prepareChunkForExport(chunk, i)
		row := e.chunkToCSVRow(exported, columns)
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("writing CSV row %d: %w", i, err)
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

// collectCSVColumns determines all columns for CSV export
func (e *Exporter) collectCSVColumns(chunks []*Chunk) []string {
	// Standard columns
	columns := []string{e.config.ChunkIDColumnName}

	if e.config.IncludeText {
		columns = append(columns, e.config.TextColumnName)
	}

	// Always include these positional columns
	columns = append(columns, "chunk_index", "document_title", "page_start", "page_end",
		"section_title", "has_table", "has_list", "has_image")

	// Collect metadata columns from all chunks
	metadataKeys := make(map[string]bool)
	for _, chunk := range chunks {
		metaMap := chunkMetadataToMap(chunk.Metadata)

		var meta map[string]interface{}
		if e.config.FlattenMetadata {
			meta = flattenMetadata(metaMap, "")
		} else {
			meta = metaMap
		}

		filtered := e.filterMetadata(meta)
		for key := range filtered {
			// Skip keys already in standard columns
			if !isStandardColumn(key) {
				metadataKeys[key] = true
			}
		}
	}

	// Sort metadata keys for consistent output
	sortedKeys := make([]string, 0, len(metadataKeys))
	for key := range metadataKeys {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	// Add metadata columns with "meta_" prefix
	for _, key := range sortedKeys {
		columns = append(columns, "meta_"+key)
	}

	// Add embeddings column if requested
	if e.config.IncludeEmbeddings {
		columns = append(columns, "embeddings")
	}

	return columns
}

// isStandardColumn checks if a key is a standard column
func isStandardColumn(key string) bool {
	standard := map[string]bool{
		"document_title": true, "page_start": true, "page_end": true,
		"chunk_index": true, "section_title": true,
		"has_table": true, "has_list": true, "has_image": true,
		"id": true, "text": true,
	}
	return standard[key]
}

// chunkToCSVRow converts an exported chunk to a CSV row
func (e *Exporter) chunkToCSVRow(chunk ExportedChunk, columns []string) []string {
	row := make([]string, len(columns))

	for i, col := range columns {
		row[i] = e.getColumnValue(chunk, col)
	}

	return row
}

// getColumnValue gets the value for a specific column
func (e *Exporter) getColumnValue(chunk ExportedChunk, column string) string {
	switch column {
	case e.config.ChunkIDColumnName:
		return chunk.ID
	case e.config.TextColumnName:
		return chunk.Text
	case "chunk_index":
		return fmt.Sprintf("%d", chunk.ChunkIndex)
	case "document_title":
		return chunk.DocumentTitle
	case "page_start":
		return fmt.Sprintf("%d", chunk.PageStart)
	case "page_end":
		return fmt.Sprintf("%d", chunk.PageEnd)
	case "section_title":
		return chunk.SectionTitle
	case "has_table":
		return fmt.Sprintf("%t", chunk.HasTable)
	case "has_list":
		return fmt.Sprintf("%t", chunk.HasList)
	case "has_image":
		return fmt.Sprintf("%t", chunk.HasImage)
	case "embeddings":
		if len(chunk.Embeddings) > 0 {
			return embeddingsToString(chunk.Embeddings)
		}
		return ""
	default:
		// Check for metadata fields (with "meta_" prefix)
		if strings.HasPrefix(column, "meta_") {
			key := strings.TrimPrefix(column, "meta_")
			if chunk.Metadata != nil {
				if val, ok := chunk.Metadata[key]; ok {
					return formatValue(val)
				}
			}
		}
		return ""
	}
}

// embeddingsToString converts embeddings to a string representation
func embeddingsToString(embeddings []float64) string {
	strs := make([]string, len(embeddings))
	for i, v := range embeddings {
		strs[i] = fmt.Sprintf("%.6f", v)
	}
	return "[" + strings.Join(strs, ",") + "]"
}

// formatValue formats a value for CSV output
func formatValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.6f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []interface{}:
		strs := make([]string, len(v))
		for i, item := range v {
			strs[i] = formatValue(item)
		}
		return "[" + strings.Join(strs, ",") + "]"
	case []string:
		return "[" + strings.Join(v, ",") + "]"
	case map[string]interface{}:
		// For nested objects in CSV, serialize as JSON
		b, _ := json.Marshal(v)
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ChunkCollection extension for export

// ToJSONL exports the collection as JSON Lines
func (cc *ChunkCollection) ToJSONL() (string, error) {
	exporter := NewExporterWithConfig(JSONLExportConfig())
	return exporter.ExportToString(cc.Chunks)
}

// ToJSON exports the collection as JSON array
func (cc *ChunkCollection) ToJSON() (string, error) {
	config := DefaultExportConfig()
	config.Format = ExportFormatJSON
	config.PrettyPrint = true
	exporter := NewExporterWithConfig(config)
	return exporter.ExportToString(cc.Chunks)
}

// ToCSV exports the collection as CSV
func (cc *ChunkCollection) ToCSV() (string, error) {
	exporter := NewExporterWithConfig(CSVExportConfig())
	return exporter.ExportToString(cc.Chunks)
}

// ToTSV exports the collection as TSV
func (cc *ChunkCollection) ToTSV() (string, error) {
	exporter := NewExporterWithConfig(TSVExportConfig())
	return exporter.ExportToString(cc.Chunks)
}

// ExportToFile exports the collection to a file
func (cc *ChunkCollection) ExportToFile(filename string, config ExportConfig) error {
	exporter := NewExporterWithConfig(config)
	return exporter.ExportToFile(cc.Chunks, filename)
}

// BatchExporter handles exporting large collections in batches
type BatchExporter struct {
	config    ExportConfig
	batchSize int
}

// NewBatchExporter creates a new batch exporter
func NewBatchExporter(batchSize int) *BatchExporter {
	return &BatchExporter{
		config:    DefaultExportConfig(),
		batchSize: batchSize,
	}
}

// NewBatchExporterWithConfig creates a batch exporter with custom config
func NewBatchExporterWithConfig(batchSize int, config ExportConfig) *BatchExporter {
	return &BatchExporter{
		config:    config,
		batchSize: batchSize,
	}
}

// ExportBatch represents a single exported batch
type ExportBatch struct {
	// BatchNumber is the zero-indexed batch number
	BatchNumber int

	// StartIndex is the starting chunk index in the original collection
	StartIndex int

	// EndIndex is the ending chunk index (exclusive)
	EndIndex int

	// ChunkCount is the number of chunks in this batch
	ChunkCount int

	// Data contains the exported data
	Data string
}

// Export exports chunks in batches, calling the callback for each batch
func (be *BatchExporter) Export(chunks []*Chunk, callback func(ExportBatch) error) error {
	exporter := NewExporterWithConfig(be.config)

	for i := 0; i < len(chunks); i += be.batchSize {
		end := i + be.batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]
		data, err := exporter.ExportToString(batch)
		if err != nil {
			return fmt.Errorf("exporting batch starting at %d: %w", i, err)
		}

		exportBatch := ExportBatch{
			BatchNumber: i / be.batchSize,
			StartIndex:  i,
			EndIndex:    end,
			ChunkCount:  len(batch),
			Data:        data,
		}

		if err := callback(exportBatch); err != nil {
			return fmt.Errorf("processing batch %d: %w", exportBatch.BatchNumber, err)
		}
	}

	return nil
}

// ExportToFiles exports chunks to numbered files
func (be *BatchExporter) ExportToFiles(chunks []*Chunk, filenamePattern string) error {
	return be.Export(chunks, func(batch ExportBatch) error {
		filename := fmt.Sprintf(filenamePattern, batch.BatchNumber)
		return os.WriteFile(filename, []byte(batch.Data), 0644)
	})
}

// StreamExporter handles streaming export for very large collections
type StreamExporter struct {
	config ExportConfig
	writer io.Writer
}

// NewStreamExporter creates a new stream exporter
func NewStreamExporter(w io.Writer) *StreamExporter {
	return &StreamExporter{
		config: DefaultExportConfig(),
		writer: w,
	}
}

// NewStreamExporterWithConfig creates a stream exporter with custom config
func NewStreamExporterWithConfig(w io.Writer, config ExportConfig) *StreamExporter {
	return &StreamExporter{
		config: config,
		writer: w,
	}
}

// WriteChunk writes a single chunk to the stream
func (se *StreamExporter) WriteChunk(chunk *Chunk, index int) error {
	exporter := NewExporterWithConfig(se.config)
	exported := exporter.prepareChunkForExport(chunk, index)

	switch se.config.Format {
	case ExportFormatJSONL:
		encoder := json.NewEncoder(se.writer)
		return encoder.Encode(exported)
	case ExportFormatJSON:
		// For streaming JSON, write as JSONL (one object per line)
		encoder := json.NewEncoder(se.writer)
		return encoder.Encode(exported)
	case ExportFormatCSV, ExportFormatTSV:
		// For streaming CSV, we need all columns upfront
		// This is a limitation - recommend using BatchExporter for CSV
		return fmt.Errorf("streaming CSV export not recommended; use BatchExporter instead")
	default:
		return fmt.Errorf("unsupported format for streaming: %v", se.config.Format)
	}
}

// Close finalizes the stream export
func (se *StreamExporter) Close() error {
	// Nothing to close for JSONL
	return nil
}

// EmbeddingExporter exports chunks with embeddings for vector databases
type EmbeddingExporter struct {
	config ExportConfig
}

// NewEmbeddingExporter creates an exporter optimized for embedding export
func NewEmbeddingExporter() *EmbeddingExporter {
	return &EmbeddingExporter{
		config: VectorDBExportConfig(),
	}
}

// EmbeddingRecord represents a single record for vector DB ingestion
type EmbeddingRecord struct {
	ID        string                 `json:"id"`
	Text      string                 `json:"text"`
	Embedding []float64              `json:"embedding,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PrepareForVectorDB prepares chunks for vector database ingestion
func (ee *EmbeddingExporter) PrepareForVectorDB(chunks []*Chunk) []EmbeddingRecord {
	records := make([]EmbeddingRecord, len(chunks))

	for i, chunk := range chunks {
		record := EmbeddingRecord{
			ID:   chunk.ID,
			Text: chunk.Text,
		}

		// Filter to essential metadata fields
		record.Metadata = make(map[string]interface{})
		record.Metadata["document_title"] = chunk.Metadata.DocumentTitle
		record.Metadata["page_start"] = chunk.Metadata.PageStart
		record.Metadata["chunk_index"] = chunk.Metadata.ChunkIndex
		record.Metadata["section_title"] = chunk.Metadata.SectionTitle
		if len(chunk.Metadata.SectionPath) > 0 {
			record.Metadata["section_path"] = chunk.Metadata.SectionPath
		}
		if len(chunk.Metadata.ElementTypes) > 0 {
			record.Metadata["element_types"] = chunk.Metadata.ElementTypes
		}

		records[i] = record
	}

	return records
}

// ExportForPinecone exports in Pinecone-compatible format
func (ee *EmbeddingExporter) ExportForPinecone(chunks []*Chunk, embeddings [][]float64, w io.Writer) error {
	type PineconeRecord struct {
		ID       string                 `json:"id"`
		Values   []float64              `json:"values"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}

	type PineconeUpsert struct {
		Vectors []PineconeRecord `json:"vectors"`
	}

	vectors := make([]PineconeRecord, 0, len(chunks))
	for i, chunk := range chunks {
		var embedding []float64
		if i < len(embeddings) {
			embedding = embeddings[i]
		}
		if len(embedding) == 0 {
			continue // Skip chunks without embeddings
		}

		metadata := make(map[string]interface{})
		metadata["text"] = chunk.Text
		metadata["document_title"] = chunk.Metadata.DocumentTitle
		metadata["page_start"] = chunk.Metadata.PageStart
		metadata["section_title"] = chunk.Metadata.SectionTitle

		vectors = append(vectors, PineconeRecord{
			ID:       chunk.ID,
			Values:   embedding,
			Metadata: metadata,
		})
	}

	upsert := PineconeUpsert{Vectors: vectors}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(upsert)
}

// ExportForChroma exports in Chroma-compatible format
func (ee *EmbeddingExporter) ExportForChroma(chunks []*Chunk, embeddings [][]float64, w io.Writer) error {
	type ChromaRecord struct {
		IDs        []string                 `json:"ids"`
		Documents  []string                 `json:"documents"`
		Embeddings [][]float64              `json:"embeddings,omitempty"`
		Metadatas  []map[string]interface{} `json:"metadatas,omitempty"`
	}

	record := ChromaRecord{
		IDs:       make([]string, len(chunks)),
		Documents: make([]string, len(chunks)),
		Metadatas: make([]map[string]interface{}, len(chunks)),
	}

	hasEmbeddings := len(embeddings) > 0
	if hasEmbeddings {
		record.Embeddings = embeddings
	}

	for i, chunk := range chunks {
		record.IDs[i] = chunk.ID
		record.Documents[i] = chunk.Text

		record.Metadatas[i] = map[string]interface{}{
			"document_title": chunk.Metadata.DocumentTitle,
			"page_start":     chunk.Metadata.PageStart,
			"section_title":  chunk.Metadata.SectionTitle,
			"chunk_index":    chunk.Metadata.ChunkIndex,
		}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(record)
}

// ExportForWeaviate exports in Weaviate-compatible format
func (ee *EmbeddingExporter) ExportForWeaviate(chunks []*Chunk, embeddings [][]float64, className string, w io.Writer) error {
	type WeaviateObject struct {
		Class      string                 `json:"class"`
		ID         string                 `json:"id,omitempty"`
		Properties map[string]interface{} `json:"properties"`
		Vector     []float64              `json:"vector,omitempty"`
	}

	encoder := json.NewEncoder(w)

	for i, chunk := range chunks {
		props := map[string]interface{}{
			"content":       chunk.Text,
			"documentTitle": chunk.Metadata.DocumentTitle,
			"pageStart":     chunk.Metadata.PageStart,
			"sectionTitle":  chunk.Metadata.SectionTitle,
			"chunkIndex":    chunk.Metadata.ChunkIndex,
		}

		obj := WeaviateObject{
			Class:      className,
			ID:         chunk.ID,
			Properties: props,
		}

		if i < len(embeddings) && len(embeddings[i]) > 0 {
			obj.Vector = embeddings[i]
		}

		if err := encoder.Encode(obj); err != nil {
			return err
		}
	}

	return nil
}
