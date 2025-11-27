package rag

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/model"
)

func TestTableFormat_String(t *testing.T) {
	tests := []struct {
		format TableFormat
		want   string
	}{
		{TableFormatPlainText, "plaintext"},
		{TableFormatMarkdown, "markdown"},
		{TableFormatCSV, "csv"},
		{TableFormatHTML, "html"},
		{TableFormat(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("TableFormat.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultTableFigureConfig(t *testing.T) {
	config := DefaultTableFigureConfig()

	if config.TableFormat != TableFormatMarkdown {
		t.Errorf("Expected TableFormat Markdown, got %v", config.TableFormat)
	}

	if config.MaxTableSize != 5000 {
		t.Errorf("Expected MaxTableSize 5000, got %d", config.MaxTableSize)
	}

	if config.MaxTableRows != 50 {
		t.Errorf("Expected MaxTableRows 50, got %d", config.MaxTableRows)
	}

	if !config.IncludeTableCaption {
		t.Error("Expected IncludeTableCaption to be true")
	}

	if !config.IncludeFigureCaption {
		t.Error("Expected IncludeFigureCaption to be true")
	}

	if !config.IncludeTableSummary {
		t.Error("Expected IncludeTableSummary to be true")
	}
}

func TestNewTableFigureHandler(t *testing.T) {
	handler := NewTableFigureHandler()
	if handler == nil {
		t.Error("NewTableFigureHandler returned nil")
	}
}

func TestTableFigureHandler_ProcessTable(t *testing.T) {
	handler := NewTableFigureHandler()

	t.Run("simple table", func(t *testing.T) {
		table := createTestTable(3, 3)
		chunks := handler.ProcessTable(table, "Table 1: Test Data", 1)

		if len(chunks) != 1 {
			t.Fatalf("Expected 1 chunk, got %d", len(chunks))
		}

		chunk := chunks[0]
		if !chunk.HasCaption {
			t.Error("Expected HasCaption to be true")
		}
		if chunk.Caption != "Table 1: Test Data" {
			t.Errorf("Expected caption 'Table 1: Test Data', got %q", chunk.Caption)
		}
		if chunk.RowCount != 3 {
			t.Errorf("Expected 3 rows, got %d", chunk.RowCount)
		}
		if chunk.ColCount != 3 {
			t.Errorf("Expected 3 cols, got %d", chunk.ColCount)
		}
		if chunk.IsSplit {
			t.Error("Expected IsSplit to be false")
		}
	})

	t.Run("table without caption", func(t *testing.T) {
		table := createTestTable(2, 2)
		chunks := handler.ProcessTable(table, "", 1)

		if len(chunks) != 1 {
			t.Fatalf("Expected 1 chunk, got %d", len(chunks))
		}

		if chunks[0].HasCaption {
			t.Error("Expected HasCaption to be false")
		}
	})

	t.Run("nil table", func(t *testing.T) {
		chunks := handler.ProcessTable(nil, "Caption", 1)

		if chunks != nil {
			t.Error("Expected nil for nil table")
		}
	})
}

func TestTableFigureHandler_ProcessTable_Split(t *testing.T) {
	config := DefaultTableFigureConfig()
	config.SplitLargeTables = true
	config.MaxTableRows = 5
	handler := NewTableFigureHandlerWithConfig(config)

	table := createTestTable(12, 3)
	chunks := handler.ProcessTable(table, "Large Table", 1)

	if len(chunks) < 2 {
		t.Fatalf("Expected multiple chunks for large table, got %d", len(chunks))
	}

	// First chunk should have caption
	if !chunks[0].HasCaption {
		t.Error("First chunk should have caption")
	}
	if chunks[0].Caption != "Large Table" {
		t.Errorf("Expected caption 'Large Table', got %q", chunks[0].Caption)
	}

	// All chunks should be marked as split
	for i, chunk := range chunks {
		if !chunk.IsSplit {
			t.Errorf("Chunk %d should be marked as split", i)
		}
		if chunk.TotalSplits != len(chunks) {
			t.Errorf("Chunk %d TotalSplits = %d, want %d", i, chunk.TotalSplits, len(chunks))
		}
		if chunk.SplitIndex != i {
			t.Errorf("Chunk %d SplitIndex = %d, want %d", i, chunk.SplitIndex, i)
		}
	}

	// Only first chunk should have caption
	for i := 1; i < len(chunks); i++ {
		if chunks[i].HasCaption {
			t.Errorf("Chunk %d should not have caption", i)
		}
	}
}

func TestTableFigureHandler_ProcessFigure(t *testing.T) {
	handler := NewTableFigureHandler()

	t.Run("figure with caption", func(t *testing.T) {
		image := &model.Image{
			Format:  model.ImageFormatPNG,
			AltText: "A test image",
		}
		chunk := handler.ProcessFigure(image, "Figure 1: Test Image", 1)

		if !chunk.HasCaption {
			t.Error("Expected HasCaption to be true")
		}
		if chunk.Caption != "Figure 1: Test Image" {
			t.Errorf("Expected caption 'Figure 1: Test Image', got %q", chunk.Caption)
		}
		if chunk.AltText != "A test image" {
			t.Errorf("Expected alt text 'A test image', got %q", chunk.AltText)
		}
		if chunk.Format != "png" {
			t.Errorf("Expected format 'png', got %q", chunk.Format)
		}
		if chunk.Description == "" {
			t.Error("Expected Description to be set")
		}
	})

	t.Run("figure without caption", func(t *testing.T) {
		image := &model.Image{
			Format: model.ImageFormatJPEG,
		}
		chunk := handler.ProcessFigure(image, "", 1)

		if chunk.HasCaption {
			t.Error("Expected HasCaption to be false")
		}
		if chunk.Format != "jpeg" {
			t.Errorf("Expected format 'jpeg', got %q", chunk.Format)
		}
	})

	t.Run("nil image", func(t *testing.T) {
		chunk := handler.ProcessFigure(nil, "Caption only", 1)

		if chunk == nil {
			t.Fatal("Expected non-nil chunk")
		}
		if !chunk.HasCaption {
			t.Error("Expected HasCaption to be true")
		}
	})
}

func TestTableFigureHandler_FormatTable(t *testing.T) {
	table := createTestTable(2, 2)

	t.Run("markdown format", func(t *testing.T) {
		config := DefaultTableFigureConfig()
		config.TableFormat = TableFormatMarkdown
		handler := NewTableFigureHandlerWithConfig(config)

		result := handler.formatTable(table)

		if !strings.Contains(result, "|") {
			t.Error("Markdown format should contain pipe characters")
		}
		if !strings.Contains(result, "---") {
			t.Error("Markdown format should contain separator")
		}
	})

	t.Run("csv format", func(t *testing.T) {
		config := DefaultTableFigureConfig()
		config.TableFormat = TableFormatCSV
		handler := NewTableFigureHandlerWithConfig(config)

		result := handler.formatTable(table)

		// CSV should not have pipes
		if strings.Contains(result, "|") {
			t.Error("CSV format should not contain pipe characters")
		}
	})

	t.Run("html format", func(t *testing.T) {
		config := DefaultTableFigureConfig()
		config.TableFormat = TableFormatHTML
		handler := NewTableFigureHandlerWithConfig(config)

		result := handler.formatTable(table)

		if !strings.Contains(result, "<table>") {
			t.Error("HTML format should contain <table> tag")
		}
		if !strings.Contains(result, "<tr>") {
			t.Error("HTML format should contain <tr> tags")
		}
		if !strings.Contains(result, "<th>") || !strings.Contains(result, "<td>") {
			t.Error("HTML format should contain cell tags")
		}
	})

	t.Run("plaintext format", func(t *testing.T) {
		config := DefaultTableFigureConfig()
		config.TableFormat = TableFormatPlainText
		handler := NewTableFigureHandlerWithConfig(config)

		result := handler.formatTable(table)

		// Plain text uses tabs
		if !strings.Contains(result, "\t") {
			t.Error("Plain text format should contain tabs")
		}
	})
}

func TestTableFigureHandler_GenerateTableSummary(t *testing.T) {
	handler := NewTableFigureHandler()

	t.Run("with summary enabled", func(t *testing.T) {
		table := createTestTableWithHeaders(3, 3, []string{"Name", "Age", "City"})
		summary := handler.generateTableSummary(table)

		if !strings.Contains(summary, "3 rows") {
			t.Error("Summary should mention row count")
		}
		if !strings.Contains(summary, "3 columns") {
			t.Error("Summary should mention column count")
		}
		if !strings.Contains(summary, "Name") {
			t.Error("Summary should mention column headers")
		}
	})

	t.Run("with summary disabled", func(t *testing.T) {
		config := DefaultTableFigureConfig()
		config.IncludeTableSummary = false
		handler := NewTableFigureHandlerWithConfig(config)

		table := createTestTable(3, 3)
		summary := handler.generateTableSummary(table)

		if summary != "" {
			t.Errorf("Expected empty summary when disabled, got %q", summary)
		}
	})

	t.Run("many columns truncated", func(t *testing.T) {
		headers := []string{"Col1", "Col2", "Col3", "Col4", "Col5", "Col6", "Col7"}
		table := createTestTableWithHeaders(2, 7, headers)
		summary := handler.generateTableSummary(table)

		if !strings.Contains(summary, "...") {
			t.Error("Summary should truncate many columns")
		}
	})
}

func TestTableFigureHandler_ExtractHeaders(t *testing.T) {
	handler := NewTableFigureHandler()

	t.Run("table with headers", func(t *testing.T) {
		table := createTestTableWithHeaders(3, 3, []string{"A", "B", "C"})
		headers := handler.extractHeaders(table)

		if len(headers) != 3 {
			t.Fatalf("Expected 3 headers, got %d", len(headers))
		}
		if headers[0] != "A" || headers[1] != "B" || headers[2] != "C" {
			t.Errorf("Headers mismatch: %v", headers)
		}
	})

	t.Run("empty table", func(t *testing.T) {
		table := model.NewTable(0, 0)
		headers := handler.extractHeaders(table)

		if headers != nil {
			t.Errorf("Expected nil headers for empty table, got %v", headers)
		}
	})
}

func TestTableChunk_ToChunk(t *testing.T) {
	tc := &TableChunk{
		Caption:       "Table 1: Data",
		HasCaption:    true,
		FormattedText: "| A | B |\n|---|---|\n| 1 | 2 |",
		Summary:       "Table with 2 rows and 2 columns",
		RowCount:      2,
		ColCount:      2,
		PageNumber:    5,
	}

	chunk := tc.ToChunk(0)

	if !strings.HasPrefix(chunk.ID, "table_5_0") {
		t.Errorf("Expected ID to start with 'table_5_0', got %q", chunk.ID)
	}
	if !strings.Contains(chunk.Text, "Table 1: Data") {
		t.Error("Chunk text should contain caption")
	}
	if !strings.Contains(chunk.Text, "| A | B |") {
		t.Error("Chunk text should contain table content")
	}
	if !chunk.Metadata.HasTable {
		t.Error("Metadata should have HasTable = true")
	}
	if chunk.Metadata.PageStart != 5 {
		t.Errorf("Expected PageStart 5, got %d", chunk.Metadata.PageStart)
	}
}

func TestFigureChunk_ToChunk(t *testing.T) {
	fc := &FigureChunk{
		Caption:     "Figure 1: Diagram",
		HasCaption:  true,
		AltText:     "A diagram showing data flow",
		Description: "Figure 1: Diagram - A diagram showing data flow - [PNG image]",
		Format:      "png",
		PageNumber:  3,
	}

	chunk := fc.ToChunk(1)

	if !strings.HasPrefix(chunk.ID, "figure_3_1") {
		t.Errorf("Expected ID to start with 'figure_3_1', got %q", chunk.ID)
	}
	if chunk.Text != fc.Description {
		t.Errorf("Expected text to be description, got %q", chunk.Text)
	}
	if !chunk.Metadata.HasImage {
		t.Error("Metadata should have HasImage = true")
	}
	if chunk.Metadata.PageStart != 3 {
		t.Errorf("Expected PageStart 3, got %d", chunk.Metadata.PageStart)
	}
}

func TestCaptionDetector_FindTableCaption(t *testing.T) {
	detector := NewCaptionDetector()

	t.Run("caption before table", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: model.ElementTypeParagraph, Text: "Table 1: Test Results"},
			{Type: model.ElementTypeTable, Text: "table content"},
		}

		caption := detector.FindTableCaption(blocks, 1)
		if caption != "Table 1: Test Results" {
			t.Errorf("Expected caption 'Table 1: Test Results', got %q", caption)
		}
	})

	t.Run("caption after table", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: model.ElementTypeTable, Text: "table content"},
			{Type: model.ElementTypeParagraph, Text: "Table 2: More Results"},
		}

		caption := detector.FindTableCaption(blocks, 0)
		if caption != "Table 2: More Results" {
			t.Errorf("Expected caption 'Table 2: More Results', got %q", caption)
		}
	})

	t.Run("no caption found", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: model.ElementTypeParagraph, Text: "Regular paragraph"},
			{Type: model.ElementTypeTable, Text: "table content"},
			{Type: model.ElementTypeParagraph, Text: "Another paragraph"},
		}

		caption := detector.FindTableCaption(blocks, 1)
		if caption != "" {
			t.Errorf("Expected empty caption, got %q", caption)
		}
	})

	t.Run("out of bounds index", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: model.ElementTypeTable, Text: "table"},
		}

		caption := detector.FindTableCaption(blocks, 5)
		if caption != "" {
			t.Errorf("Expected empty caption for out of bounds, got %q", caption)
		}
	})
}

func TestCaptionDetector_FindFigureCaption(t *testing.T) {
	detector := NewCaptionDetector()

	t.Run("figure caption before", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: model.ElementTypeParagraph, Text: "Figure 1: System Architecture"},
			{Type: model.ElementTypeImage, Text: ""},
		}

		caption := detector.FindFigureCaption(blocks, 1)
		if caption != "Figure 1: System Architecture" {
			t.Errorf("Expected figure caption, got %q", caption)
		}
	})

	t.Run("figure caption after", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: model.ElementTypeImage, Text: ""},
			{Type: model.ElementTypeParagraph, Text: "Fig. 2: Data Flow"},
		}

		caption := detector.FindFigureCaption(blocks, 0)
		if caption != "Fig. 2: Data Flow" {
			t.Errorf("Expected figure caption, got %q", caption)
		}
	})
}

func TestCaptionDetector_IsTableCaption(t *testing.T) {
	detector := NewCaptionDetector()

	tests := []struct {
		text string
		want bool
	}{
		{"Table 1: Results", true},
		{"Table 2.1: More Results", true},
		{"Tbl. 1 Summary", true},
		{"Tab 3: Data", true},
		{"This is a table", false},
		{"Regular paragraph", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := detector.isTableCaption(tt.text); got != tt.want {
				t.Errorf("isTableCaption(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestCaptionDetector_IsFigureCaption(t *testing.T) {
	detector := NewCaptionDetector()

	tests := []struct {
		text string
		want bool
	}{
		{"Figure 1: Architecture", true},
		{"Fig. 2: Diagram", true},
		{"Fig 3: Flow Chart", true},
		{"Image 1: Screenshot", true},
		{"Diagram 1: Overview", true},
		{"Chart 1: Sales Data", true},
		{"Graph 1: Trend", true},
		{"This is a figure", false},
		{"Regular paragraph", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := detector.isFigureCaption(tt.text); got != tt.want {
				t.Errorf("isFigureCaption(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestTableFigureHandler_ProcessBlocks(t *testing.T) {
	handler := NewTableFigureHandler()

	blocks := []ContentBlock{
		{Type: model.ElementTypeParagraph, Text: "Table 1: Data", Page: 1},
		{Type: model.ElementTypeTable, Text: "| A | B |\n| 1 | 2 |", Page: 1},
		{Type: model.ElementTypeParagraph, Text: "Some text", Page: 1},
		{Type: model.ElementTypeParagraph, Text: "Figure 1: Chart", Page: 2},
		{Type: model.ElementTypeImage, Text: "", Page: 2},
	}

	result := handler.ProcessBlocks(blocks)

	if result.Stats.TotalTables != 1 {
		t.Errorf("Expected 1 table, got %d", result.Stats.TotalTables)
	}
	if result.Stats.TotalFigures != 1 {
		t.Errorf("Expected 1 figure, got %d", result.Stats.TotalFigures)
	}
	if result.Stats.TablesWithCaption != 1 {
		t.Errorf("Expected 1 table with caption, got %d", result.Stats.TablesWithCaption)
	}
	if result.Stats.FiguresWithCaption != 1 {
		t.Errorf("Expected 1 figure with caption, got %d", result.Stats.FiguresWithCaption)
	}
	if len(result.TableChunks) != 1 {
		t.Errorf("Expected 1 table chunk, got %d", len(result.TableChunks))
	}
	if len(result.FigureChunks) != 1 {
		t.Errorf("Expected 1 figure chunk, got %d", len(result.FigureChunks))
	}
}

func TestImageFormatString(t *testing.T) {
	tests := []struct {
		format model.ImageFormat
		want   string
	}{
		{model.ImageFormatJPEG, "jpeg"},
		{model.ImageFormatPNG, "png"},
		{model.ImageFormatTIFF, "tiff"},
		{model.ImageFormatJPEG2000, "jpeg2000"},
		{model.ImageFormatJBIG2, "jbig2"},
		{model.ImageFormatUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := imageFormatString(tt.format); got != tt.want {
				t.Errorf("imageFormatString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<script>", "&lt;script&gt;"},
		{"a & b", "a &amp; b"},
		{"\"quoted\"", "&quot;quoted&quot;"},
		{"plain text", "plain text"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := escapeHTML(tt.input); got != tt.want {
				t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestContainsNumber(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Table 1", true},
		{"Figure 2.3", true},
		{"No numbers", false},
		{"", false},
		{"123", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := containsNumber(tt.input); got != tt.want {
				t.Errorf("containsNumber(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsTableElement(t *testing.T) {
	if !IsTableElement(model.ElementTypeTable) {
		t.Error("ElementTypeTable should be table element")
	}
	if IsTableElement(model.ElementTypeParagraph) {
		t.Error("ElementTypeParagraph should not be table element")
	}
}

func TestIsFigureElement(t *testing.T) {
	if !IsFigureElement(model.ElementTypeImage) {
		t.Error("ElementTypeImage should be figure element")
	}
	if !IsFigureElement(model.ElementTypeFigure) {
		t.Error("ElementTypeFigure should be figure element")
	}
	if IsFigureElement(model.ElementTypeParagraph) {
		t.Error("ElementTypeParagraph should not be figure element")
	}
}

func TestIsCaptionElement(t *testing.T) {
	if !IsCaptionElement(model.ElementTypeCaption) {
		t.Error("ElementTypeCaption should be caption element")
	}
	if IsCaptionElement(model.ElementTypeParagraph) {
		t.Error("ElementTypeParagraph should not be caption element")
	}
}

// Helper functions

func createTestTable(rows, cols int) *model.Table {
	table := model.NewTable(rows, cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			table.Rows[i][j].Text = cellText(i, j)
		}
	}
	return table
}

func createTestTableWithHeaders(rows, cols int, headers []string) *model.Table {
	table := model.NewTable(rows, cols)

	// Set headers in first row
	for j := 0; j < cols && j < len(headers); j++ {
		table.Rows[0][j].Text = headers[j]
		table.Rows[0][j].IsHeader = true
	}

	// Fill data rows
	for i := 1; i < rows; i++ {
		for j := 0; j < cols; j++ {
			table.Rows[i][j].Text = cellText(i, j)
		}
	}

	return table
}

func cellText(row, col int) string {
	return strings.Repeat("X", (row+1)*(col+1))
}

// Benchmarks

func BenchmarkProcessTable(b *testing.B) {
	handler := NewTableFigureHandler()
	table := createTestTable(10, 5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ProcessTable(table, "Test Caption", 1)
	}
}

func BenchmarkProcessBlocks(b *testing.B) {
	handler := NewTableFigureHandler()
	blocks := []ContentBlock{
		{Type: model.ElementTypeParagraph, Text: "Table 1: Data"},
		{Type: model.ElementTypeTable, Text: "| A | B |\n| 1 | 2 |"},
		{Type: model.ElementTypeParagraph, Text: "Figure 1: Chart"},
		{Type: model.ElementTypeImage, Text: ""},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ProcessBlocks(blocks)
	}
}

func BenchmarkFormatTableMarkdown(b *testing.B) {
	config := DefaultTableFigureConfig()
	config.TableFormat = TableFormatMarkdown
	handler := NewTableFigureHandlerWithConfig(config)
	table := createTestTable(20, 5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.formatTable(table)
	}
}

func BenchmarkFormatTableHTML(b *testing.B) {
	config := DefaultTableFigureConfig()
	config.TableFormat = TableFormatHTML
	handler := NewTableFigureHandlerWithConfig(config)
	table := createTestTable(20, 5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.formatTable(table)
	}
}
