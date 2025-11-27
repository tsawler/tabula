package rag

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func createTestChunks() []*Chunk {
	return []*Chunk{
		{
			ID:   "chunk-1",
			Text: "This is the first chunk of text.",
			Metadata: ChunkMetadata{
				DocumentTitle: "Test Document",
				SectionPath:   []string{"Chapter 1", "Introduction"},
				SectionTitle:  "Introduction",
				HeadingLevel:  2,
				PageStart:     1,
				PageEnd:       1,
				ChunkIndex:    0,
				TotalChunks:   3,
				Level:         ChunkLevelParagraph,
				ElementTypes:  []string{"paragraph"},
				CharCount:     33,
				WordCount:     7,
			},
		},
		{
			ID:   "chunk-2",
			Text: "Second chunk with a table reference.",
			Metadata: ChunkMetadata{
				DocumentTitle: "Test Document",
				SectionPath:   []string{"Chapter 1", "Data"},
				SectionTitle:  "Data",
				HeadingLevel:  2,
				PageStart:     2,
				PageEnd:       2,
				ChunkIndex:    1,
				TotalChunks:   3,
				Level:         ChunkLevelParagraph,
				HasTable:      true,
				ElementTypes:  []string{"paragraph", "table"},
				CharCount:     36,
				WordCount:     6,
			},
		},
		{
			ID:   "chunk-3",
			Text: "Final chunk with list content.",
			Metadata: ChunkMetadata{
				DocumentTitle: "Test Document",
				SectionPath:   []string{"Chapter 1", "Summary"},
				SectionTitle:  "Summary",
				HeadingLevel:  2,
				PageStart:     3,
				PageEnd:       3,
				ChunkIndex:    2,
				TotalChunks:   3,
				Level:         ChunkLevelParagraph,
				HasList:       true,
				ElementTypes:  []string{"paragraph", "list"},
				CharCount:     30,
				WordCount:     5,
			},
		},
	}
}

func TestExportFormat_String(t *testing.T) {
	tests := []struct {
		format ExportFormat
		want   string
	}{
		{ExportFormatJSONL, "jsonl"},
		{ExportFormatJSON, "json"},
		{ExportFormatCSV, "csv"},
		{ExportFormatTSV, "tsv"},
		{ExportFormat(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("ExportFormat.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExportFormat_FileExtension(t *testing.T) {
	tests := []struct {
		format ExportFormat
		want   string
	}{
		{ExportFormatJSONL, ".jsonl"},
		{ExportFormatJSON, ".json"},
		{ExportFormatCSV, ".csv"},
		{ExportFormatTSV, ".tsv"},
		{ExportFormat(99), ".txt"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.format.FileExtension(); got != tt.want {
				t.Errorf("ExportFormat.FileExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultExportConfig(t *testing.T) {
	config := DefaultExportConfig()

	if config.Format != ExportFormatJSONL {
		t.Errorf("Expected JSONL format, got %v", config.Format)
	}
	if !config.IncludeMetadata {
		t.Error("Expected IncludeMetadata to be true")
	}
	if !config.IncludeText {
		t.Error("Expected IncludeText to be true")
	}
	if config.IncludeEmbeddings {
		t.Error("Expected IncludeEmbeddings to be false")
	}
	if config.CSVDelimiter != ',' {
		t.Errorf("Expected comma delimiter, got %c", config.CSVDelimiter)
	}
}

func TestJSONLExportConfig(t *testing.T) {
	config := JSONLExportConfig()

	if config.Format != ExportFormatJSONL {
		t.Errorf("Expected JSONL format, got %v", config.Format)
	}
}

func TestCSVExportConfig(t *testing.T) {
	config := CSVExportConfig()

	if config.Format != ExportFormatCSV {
		t.Errorf("Expected CSV format, got %v", config.Format)
	}
	if !config.FlattenMetadata {
		t.Error("Expected FlattenMetadata to be true")
	}
}

func TestTSVExportConfig(t *testing.T) {
	config := TSVExportConfig()

	if config.Format != ExportFormatTSV {
		t.Errorf("Expected TSV format, got %v", config.Format)
	}
	if config.CSVDelimiter != '\t' {
		t.Errorf("Expected tab delimiter, got %c", config.CSVDelimiter)
	}
}

func TestVectorDBExportConfig(t *testing.T) {
	config := VectorDBExportConfig()

	if config.Format != ExportFormatJSONL {
		t.Errorf("Expected JSONL format, got %v", config.Format)
	}
	if !config.IncludeEmbeddings {
		t.Error("Expected IncludeEmbeddings to be true")
	}
	if len(config.MetadataFields) == 0 {
		t.Error("Expected specific metadata fields")
	}
}

func TestNewExporter(t *testing.T) {
	exporter := NewExporter()
	if exporter == nil {
		t.Error("NewExporter returned nil")
	}
}

func TestNewExporterWithConfig(t *testing.T) {
	config := CSVExportConfig()
	exporter := NewExporterWithConfig(config)

	if exporter == nil {
		t.Error("NewExporterWithConfig returned nil")
	}
}

func TestExporter_ExportJSONL(t *testing.T) {
	chunks := createTestChunks()
	exporter := NewExporterWithConfig(JSONLExportConfig())

	var buf bytes.Buffer
	err := exporter.Export(chunks, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	// Verify each chunk is on its own line
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var exported ExportedChunk
		if err := json.Unmarshal([]byte(line), &exported); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}

	// Verify first chunk content
	var first ExportedChunk
	json.Unmarshal([]byte(lines[0]), &first)
	if first.ID != "chunk-1" {
		t.Errorf("Expected ID chunk-1, got %s", first.ID)
	}
	if first.Text != "This is the first chunk of text." {
		t.Errorf("Unexpected text: %s", first.Text)
	}
}

func TestExporter_ExportJSON(t *testing.T) {
	chunks := createTestChunks()
	config := DefaultExportConfig()
	config.Format = ExportFormatJSON
	exporter := NewExporterWithConfig(config)

	var buf bytes.Buffer
	err := exporter.Export(chunks, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify it's a valid JSON array
	var exported []ExportedChunk
	if err := json.Unmarshal(buf.Bytes(), &exported); err != nil {
		t.Fatalf("Output is not valid JSON array: %v", err)
	}

	if len(exported) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(exported))
	}
}

func TestExporter_ExportCSV(t *testing.T) {
	chunks := createTestChunks()
	exporter := NewExporterWithConfig(CSVExportConfig())

	var buf bytes.Buffer
	err := exporter.Export(chunks, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header + 3 data rows
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines (header + 3 rows), got %d", len(lines))
	}

	// Verify header contains expected columns
	header := lines[0]
	expectedCols := []string{"chunk_id", "text", "chunk_index", "document_title"}
	for _, col := range expectedCols {
		if !strings.Contains(header, col) {
			t.Errorf("Header missing column: %s", col)
		}
	}
}

func TestExporter_ExportTSV(t *testing.T) {
	chunks := createTestChunks()
	exporter := NewExporterWithConfig(TSVExportConfig())

	var buf bytes.Buffer
	err := exporter.Export(chunks, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Verify tab separation
	if !strings.Contains(lines[0], "\t") {
		t.Error("Expected tab-separated values")
	}
}

func TestExporter_ExportToString(t *testing.T) {
	chunks := createTestChunks()
	exporter := NewExporter()

	output, err := exporter.ExportToString(chunks)
	if err != nil {
		t.Fatalf("ExportToString failed: %v", err)
	}

	if output == "" {
		t.Error("Expected non-empty output")
	}

	// Should be valid JSONL
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

func TestExporter_MetadataFiltering(t *testing.T) {
	chunks := createTestChunks()
	config := JSONLExportConfig()
	config.MetadataFields = []string{"document_title", "page_start"}
	exporter := NewExporterWithConfig(config)

	output, err := exporter.ExportToString(chunks)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Parse first chunk
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var first ExportedChunk
	json.Unmarshal([]byte(lines[0]), &first)

	// Should only have filtered fields
	if _, ok := first.Metadata["document_title"]; !ok {
		t.Error("Expected document_title in metadata")
	}
	if _, ok := first.Metadata["page_start"]; !ok {
		t.Error("Expected page_start in metadata")
	}
	if _, ok := first.Metadata["char_count"]; ok {
		t.Error("char_count should be filtered out")
	}
}

func TestExporter_NoText(t *testing.T) {
	chunks := createTestChunks()
	config := JSONLExportConfig()
	config.IncludeText = false
	exporter := NewExporterWithConfig(config)

	output, err := exporter.ExportToString(chunks)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Parse first chunk
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var first ExportedChunk
	json.Unmarshal([]byte(lines[0]), &first)

	if first.Text != "" {
		t.Error("Expected empty text when IncludeText is false")
	}
}

func TestExporter_NoMetadata(t *testing.T) {
	chunks := createTestChunks()
	config := JSONLExportConfig()
	config.IncludeMetadata = false
	exporter := NewExporterWithConfig(config)

	output, err := exporter.ExportToString(chunks)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Parse first chunk
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var first ExportedChunk
	json.Unmarshal([]byte(lines[0]), &first)

	if len(first.Metadata) != 0 {
		t.Error("Expected no metadata when IncludeMetadata is false")
	}
}

func TestExporter_PrettyPrint(t *testing.T) {
	chunks := createTestChunks()[:1]
	config := DefaultExportConfig()
	config.Format = ExportFormatJSON
	config.PrettyPrint = true
	exporter := NewExporterWithConfig(config)

	output, err := exporter.ExportToString(chunks)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Pretty print should have multiple lines and indentation
	if !strings.Contains(output, "\n  ") {
		t.Error("Expected indented output for pretty print")
	}
}

func TestChunkCollection_ToJSONL(t *testing.T) {
	chunks := createTestChunks()
	collection := NewChunkCollection(chunks)

	output, err := collection.ToJSONL()
	if err != nil {
		t.Fatalf("ToJSONL failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

func TestChunkCollection_ToJSON(t *testing.T) {
	chunks := createTestChunks()
	collection := NewChunkCollection(chunks)

	output, err := collection.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var exported []ExportedChunk
	if err := json.Unmarshal([]byte(output), &exported); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if len(exported) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(exported))
	}
}

func TestChunkCollection_ToCSV(t *testing.T) {
	chunks := createTestChunks()
	collection := NewChunkCollection(chunks)

	output, err := collection.ToCSV()
	if err != nil {
		t.Fatalf("ToCSV failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines, got %d", len(lines))
	}
}

func TestChunkCollection_ToTSV(t *testing.T) {
	chunks := createTestChunks()
	collection := NewChunkCollection(chunks)

	output, err := collection.ToTSV()
	if err != nil {
		t.Fatalf("ToTSV failed: %v", err)
	}

	if !strings.Contains(output, "\t") {
		t.Error("Expected tab-separated values")
	}
}

func TestBatchExporter_Export(t *testing.T) {
	chunks := createTestChunks()
	batchExporter := NewBatchExporter(2)

	var batches []ExportBatch
	err := batchExporter.Export(chunks, func(batch ExportBatch) error {
		batches = append(batches, batch)
		return nil
	})

	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// With batch size 2 and 3 chunks, should get 2 batches
	if len(batches) != 2 {
		t.Errorf("Expected 2 batches, got %d", len(batches))
	}

	// First batch should have 2 chunks
	if batches[0].ChunkCount != 2 {
		t.Errorf("Expected first batch to have 2 chunks, got %d", batches[0].ChunkCount)
	}

	// Second batch should have 1 chunk
	if batches[1].ChunkCount != 1 {
		t.Errorf("Expected second batch to have 1 chunk, got %d", batches[1].ChunkCount)
	}
}

func TestBatchExporter_BatchMetadata(t *testing.T) {
	chunks := createTestChunks()
	batchExporter := NewBatchExporter(2)

	var batches []ExportBatch
	batchExporter.Export(chunks, func(batch ExportBatch) error {
		batches = append(batches, batch)
		return nil
	})

	// Verify first batch metadata
	if batches[0].BatchNumber != 0 {
		t.Errorf("Expected batch 0, got %d", batches[0].BatchNumber)
	}
	if batches[0].StartIndex != 0 {
		t.Errorf("Expected start index 0, got %d", batches[0].StartIndex)
	}
	if batches[0].EndIndex != 2 {
		t.Errorf("Expected end index 2, got %d", batches[0].EndIndex)
	}

	// Verify second batch metadata
	if batches[1].BatchNumber != 1 {
		t.Errorf("Expected batch 1, got %d", batches[1].BatchNumber)
	}
	if batches[1].StartIndex != 2 {
		t.Errorf("Expected start index 2, got %d", batches[1].StartIndex)
	}
}

func TestStreamExporter_WriteChunk(t *testing.T) {
	chunks := createTestChunks()
	var buf bytes.Buffer
	streamExporter := NewStreamExporter(&buf)

	for i, chunk := range chunks {
		if err := streamExporter.WriteChunk(chunk, i); err != nil {
			t.Fatalf("WriteChunk failed: %v", err)
		}
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

func TestStreamExporter_Close(t *testing.T) {
	var buf bytes.Buffer
	streamExporter := NewStreamExporter(&buf)

	err := streamExporter.Close()
	if err != nil {
		t.Errorf("Close should not fail: %v", err)
	}
}

func TestStreamExporter_CSVNotSupported(t *testing.T) {
	chunks := createTestChunks()
	var buf bytes.Buffer
	config := CSVExportConfig()
	streamExporter := NewStreamExporterWithConfig(&buf, config)

	err := streamExporter.WriteChunk(chunks[0], 0)
	if err == nil {
		t.Error("Expected error for CSV streaming")
	}
}

func TestEmbeddingExporter_PrepareForVectorDB(t *testing.T) {
	chunks := createTestChunks()
	embeddingExporter := NewEmbeddingExporter()

	records := embeddingExporter.PrepareForVectorDB(chunks)

	if len(records) != 3 {
		t.Errorf("Expected 3 records, got %d", len(records))
	}

	// Verify first record
	if records[0].ID != "chunk-1" {
		t.Errorf("Expected ID chunk-1, got %s", records[0].ID)
	}
	if records[0].Text != "This is the first chunk of text." {
		t.Errorf("Unexpected text: %s", records[0].Text)
	}

	// Verify metadata
	if records[0].Metadata["document_title"] != "Test Document" {
		t.Error("Expected document_title in metadata")
	}
}

func TestEmbeddingExporter_ExportForPinecone(t *testing.T) {
	chunks := createTestChunks()
	embeddings := [][]float64{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
		{0.7, 0.8, 0.9},
	}
	embeddingExporter := NewEmbeddingExporter()

	var buf bytes.Buffer
	err := embeddingExporter.ExportForPinecone(chunks, embeddings, &buf)
	if err != nil {
		t.Fatalf("ExportForPinecone failed: %v", err)
	}

	// Parse output
	var result struct {
		Vectors []struct {
			ID       string                 `json:"id"`
			Values   []float64              `json:"values"`
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"vectors"`
	}

	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(result.Vectors) != 3 {
		t.Errorf("Expected 3 vectors, got %d", len(result.Vectors))
	}

	// Verify first vector
	if result.Vectors[0].ID != "chunk-1" {
		t.Errorf("Expected ID chunk-1, got %s", result.Vectors[0].ID)
	}
	if len(result.Vectors[0].Values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(result.Vectors[0].Values))
	}
}

func TestEmbeddingExporter_ExportForPinecone_SkipsNoEmbedding(t *testing.T) {
	chunks := createTestChunks()
	embeddings := [][]float64{
		{0.1, 0.2, 0.3},
		{}, // Empty embedding
		{0.7, 0.8, 0.9},
	}
	embeddingExporter := NewEmbeddingExporter()

	var buf bytes.Buffer
	embeddingExporter.ExportForPinecone(chunks, embeddings, &buf)

	var result struct {
		Vectors []interface{} `json:"vectors"`
	}
	json.Unmarshal(buf.Bytes(), &result)

	// Should only have 2 vectors (skipped chunk-2)
	if len(result.Vectors) != 2 {
		t.Errorf("Expected 2 vectors (skipped empty), got %d", len(result.Vectors))
	}
}

func TestEmbeddingExporter_ExportForChroma(t *testing.T) {
	chunks := createTestChunks()
	embeddings := [][]float64{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
		{0.7, 0.8, 0.9},
	}
	embeddingExporter := NewEmbeddingExporter()

	var buf bytes.Buffer
	err := embeddingExporter.ExportForChroma(chunks, embeddings, &buf)
	if err != nil {
		t.Fatalf("ExportForChroma failed: %v", err)
	}

	// Parse output
	var result struct {
		IDs        []string                 `json:"ids"`
		Documents  []string                 `json:"documents"`
		Embeddings [][]float64              `json:"embeddings"`
		Metadatas  []map[string]interface{} `json:"metadatas"`
	}

	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(result.IDs) != 3 {
		t.Errorf("Expected 3 IDs, got %d", len(result.IDs))
	}
	if len(result.Documents) != 3 {
		t.Errorf("Expected 3 documents, got %d", len(result.Documents))
	}
	if len(result.Embeddings) != 3 {
		t.Errorf("Expected 3 embeddings, got %d", len(result.Embeddings))
	}
}

func TestEmbeddingExporter_ExportForChroma_NoEmbeddings(t *testing.T) {
	chunks := createTestChunks()
	embeddingExporter := NewEmbeddingExporter()

	var buf bytes.Buffer
	embeddingExporter.ExportForChroma(chunks, nil, &buf)

	var result struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
	json.Unmarshal(buf.Bytes(), &result)

	if result.Embeddings != nil {
		t.Error("Expected nil embeddings when none provided")
	}
}

func TestEmbeddingExporter_ExportForWeaviate(t *testing.T) {
	chunks := createTestChunks()
	embeddings := [][]float64{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
		{0.7, 0.8, 0.9},
	}
	embeddingExporter := NewEmbeddingExporter()

	var buf bytes.Buffer
	err := embeddingExporter.ExportForWeaviate(chunks, embeddings, "Document", &buf)
	if err != nil {
		t.Fatalf("ExportForWeaviate failed: %v", err)
	}

	// Parse as JSONL
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Parse first object
	var first struct {
		Class      string                 `json:"class"`
		ID         string                 `json:"id"`
		Properties map[string]interface{} `json:"properties"`
		Vector     []float64              `json:"vector"`
	}
	json.Unmarshal([]byte(lines[0]), &first)

	if first.Class != "Document" {
		t.Errorf("Expected class Document, got %s", first.Class)
	}
	if first.ID != "chunk-1" {
		t.Errorf("Expected ID chunk-1, got %s", first.ID)
	}
	if len(first.Vector) != 3 {
		t.Errorf("Expected 3-dim vector, got %d", len(first.Vector))
	}
}

func TestChunkMetadataToMap(t *testing.T) {
	meta := ChunkMetadata{
		DocumentTitle:   "Test Doc",
		SectionPath:     []string{"A", "B"},
		SectionTitle:    "B",
		HeadingLevel:    2,
		PageStart:       1,
		PageEnd:         2,
		ChunkIndex:      5,
		TotalChunks:     10,
		Level:           ChunkLevelParagraph,
		HasTable:        true,
		HasList:         false,
		HasImage:        true,
		CharCount:       100,
		WordCount:       20,
		EstimatedTokens: 25,
	}

	m := chunkMetadataToMap(meta)

	if m["document_title"] != "Test Doc" {
		t.Error("Expected document_title")
	}
	if m["heading_level"] != 2 {
		t.Error("Expected heading_level")
	}
	if m["chunk_index"] != 5 {
		t.Error("Expected chunk_index")
	}
	if m["has_table"] != true {
		t.Error("Expected has_table")
	}
	if _, ok := m["has_list"]; ok {
		t.Error("has_list should not be set when false")
	}
	if m["has_image"] != true {
		t.Error("Expected has_image")
	}
}

func TestFlattenMetadata(t *testing.T) {
	data := map[string]interface{}{
		"level1": "value1",
		"nested": map[string]interface{}{
			"level2": "value2",
			"deeper": map[string]interface{}{
				"level3": "value3",
			},
		},
	}

	flattened := flattenMetadata(data, "")

	if flattened["level1"] != "value1" {
		t.Error("Expected level1")
	}
	if flattened["nested.level2"] != "value2" {
		t.Error("Expected nested.level2")
	}
	if flattened["nested.deeper.level3"] != "value3" {
		t.Error("Expected nested.deeper.level3")
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"int64", int64(100), "100"},
		{"float", 3.14, "3.140000"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"string slice", []string{"a", "b", "c"}, "[a,b,c]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.input)
			if got != tt.want {
				t.Errorf("formatValue(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEmbeddingsToString(t *testing.T) {
	embeddings := []float64{0.1, 0.2, 0.3}
	result := embeddingsToString(embeddings)

	if !strings.HasPrefix(result, "[") || !strings.HasSuffix(result, "]") {
		t.Error("Expected brackets around embeddings")
	}
	if !strings.Contains(result, "0.100000") {
		t.Error("Expected formatted float values")
	}
}

func TestIsStandardColumn(t *testing.T) {
	standardCols := []string{"document_title", "page_start", "chunk_index", "has_table"}
	for _, col := range standardCols {
		if !isStandardColumn(col) {
			t.Errorf("%s should be a standard column", col)
		}
	}

	if isStandardColumn("custom_field") {
		t.Error("custom_field should not be a standard column")
	}
}

func TestExporter_UnsupportedFormat(t *testing.T) {
	chunks := createTestChunks()
	config := DefaultExportConfig()
	config.Format = ExportFormat(99)
	exporter := NewExporterWithConfig(config)

	var buf bytes.Buffer
	err := exporter.Export(chunks, &buf)

	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

// Benchmarks

func BenchmarkExportJSONL(b *testing.B) {
	chunks := make([]*Chunk, 100)
	for i := 0; i < 100; i++ {
		chunks[i] = &Chunk{
			ID:   "chunk",
			Text: strings.Repeat("text ", 100),
			Metadata: ChunkMetadata{
				DocumentTitle: "Test",
				PageStart:     i,
			},
		}
	}
	exporter := NewExporter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		exporter.Export(chunks, &buf)
	}
}

func BenchmarkExportCSV(b *testing.B) {
	chunks := make([]*Chunk, 100)
	for i := 0; i < 100; i++ {
		chunks[i] = &Chunk{
			ID:   "chunk",
			Text: strings.Repeat("text ", 100),
			Metadata: ChunkMetadata{
				DocumentTitle: "Test",
				PageStart:     i,
			},
		}
	}
	exporter := NewExporterWithConfig(CSVExportConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		exporter.Export(chunks, &buf)
	}
}

func BenchmarkBatchExport(b *testing.B) {
	chunks := make([]*Chunk, 1000)
	for i := 0; i < 1000; i++ {
		chunks[i] = &Chunk{
			ID:   "chunk",
			Text: strings.Repeat("text ", 100),
		}
	}
	batchExporter := NewBatchExporter(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batchExporter.Export(chunks, func(batch ExportBatch) error {
			return nil
		})
	}
}
