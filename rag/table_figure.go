package rag

import (
	"fmt"
	"strings"

	"github.com/tsawler/tabula/model"
)

// TableFormat defines how tables are formatted in chunks
type TableFormat int

const (
	// TableFormatPlainText formats table as tab-separated text
	TableFormatPlainText TableFormat = iota
	// TableFormatMarkdown formats table as markdown
	TableFormatMarkdown
	// TableFormatCSV formats table as CSV
	TableFormatCSV
	// TableFormatHTML formats table as HTML
	TableFormatHTML
)

// String returns a human-readable representation of the table format
func (tf TableFormat) String() string {
	switch tf {
	case TableFormatPlainText:
		return "plaintext"
	case TableFormatMarkdown:
		return "markdown"
	case TableFormatCSV:
		return "csv"
	case TableFormatHTML:
		return "html"
	default:
		return "unknown"
	}
}

// TableFigureConfig holds configuration for table and figure chunking
type TableFigureConfig struct {
	// TableFormat determines how tables are rendered in chunks
	TableFormat TableFormat

	// MaxTableSize is the maximum characters for a table before considering split
	MaxTableSize int

	// MaxTableRows is the maximum rows before considering split
	MaxTableRows int

	// SplitLargeTables allows splitting tables that exceed limits
	SplitLargeTables bool

	// IncludeTableCaption includes detected captions with tables
	IncludeTableCaption bool

	// IncludeFigureCaption includes detected captions with figures
	IncludeFigureCaption bool

	// CaptionSearchDistance is max chars to search for caption
	CaptionSearchDistance int

	// IncludeTableSummary adds a brief summary of table dimensions
	IncludeTableSummary bool

	// IncludeFigureAltText includes alt text for figures
	IncludeFigureAltText bool

	// PreserveTableStructure keeps structural info for RAG
	PreserveTableStructure bool
}

// DefaultTableFigureConfig returns sensible defaults
func DefaultTableFigureConfig() TableFigureConfig {
	return TableFigureConfig{
		TableFormat:            TableFormatMarkdown,
		MaxTableSize:           5000,
		MaxTableRows:           50,
		SplitLargeTables:       false,
		IncludeTableCaption:    true,
		IncludeFigureCaption:   true,
		CaptionSearchDistance:  200,
		IncludeTableSummary:    true,
		IncludeFigureAltText:   true,
		PreserveTableStructure: true,
	}
}

// TableChunk represents a table as a chunk
type TableChunk struct {
	// Table is the source table
	Table *model.Table

	// Caption is the associated caption text
	Caption string

	// HasCaption indicates if a caption was found
	HasCaption bool

	// FormattedText is the table rendered as text
	FormattedText string

	// Summary is a brief description of the table
	Summary string

	// RowCount is the number of rows
	RowCount int

	// ColCount is the number of columns
	ColCount int

	// Headers are the column headers (if detected)
	Headers []string

	// IsSplit indicates if this is part of a split table
	IsSplit bool

	// SplitIndex is the index of this part (0-based)
	SplitIndex int

	// TotalSplits is the total number of parts
	TotalSplits int

	// PageNumber is the source page
	PageNumber int
}

// FigureChunk represents a figure/image as a chunk
type FigureChunk struct {
	// Image is the source image (if available)
	Image *model.Image

	// Caption is the associated caption text
	Caption string

	// HasCaption indicates if a caption was found
	HasCaption bool

	// AltText is alternative text for the image
	AltText string

	// Description is a generated description
	Description string

	// Format is the image format
	Format string

	// PageNumber is the source page
	PageNumber int
}

// TableFigureHandler handles table and figure chunking
type TableFigureHandler struct {
	config TableFigureConfig
}

// NewTableFigureHandler creates a new handler with default config
func NewTableFigureHandler() *TableFigureHandler {
	return &TableFigureHandler{
		config: DefaultTableFigureConfig(),
	}
}

// NewTableFigureHandlerWithConfig creates a handler with custom config
func NewTableFigureHandlerWithConfig(config TableFigureConfig) *TableFigureHandler {
	return &TableFigureHandler{
		config: config,
	}
}

// ProcessTable converts a table to one or more chunks
func (h *TableFigureHandler) ProcessTable(table *model.Table, caption string, pageNumber int) []*TableChunk {
	if table == nil {
		return nil
	}

	// Format the table
	formattedText := h.formatTable(table)

	// Check if we need to split
	if h.config.SplitLargeTables && h.shouldSplitTable(table, formattedText) {
		return h.splitTable(table, caption, pageNumber)
	}

	// Create single chunk
	chunk := &TableChunk{
		Table:         table,
		Caption:       caption,
		HasCaption:    caption != "",
		FormattedText: formattedText,
		Summary:       h.generateTableSummary(table),
		RowCount:      table.RowCount(),
		ColCount:      table.ColCount(),
		Headers:       h.extractHeaders(table),
		PageNumber:    pageNumber,
	}

	return []*TableChunk{chunk}
}

// ProcessFigure converts a figure/image to a chunk
func (h *TableFigureHandler) ProcessFigure(image *model.Image, caption string, pageNumber int) *FigureChunk {
	chunk := &FigureChunk{
		Image:      image,
		Caption:    caption,
		HasCaption: caption != "",
		PageNumber: pageNumber,
	}

	if image != nil {
		chunk.AltText = image.AltText
		chunk.Format = imageFormatString(image.Format)
	}

	// Generate description
	chunk.Description = h.generateFigureDescription(chunk)

	return chunk
}

// formatTable renders a table in the configured format
func (h *TableFigureHandler) formatTable(table *model.Table) string {
	switch h.config.TableFormat {
	case TableFormatMarkdown:
		return table.ToMarkdown()
	case TableFormatCSV:
		return table.ToCSV()
	case TableFormatHTML:
		return h.tableToHTML(table)
	default:
		return table.GetText()
	}
}

// tableToHTML converts a table to HTML format
func (h *TableFigureHandler) tableToHTML(table *model.Table) string {
	var sb strings.Builder

	sb.WriteString("<table>\n")

	for i, row := range table.Rows {
		sb.WriteString("  <tr>\n")
		for _, cell := range row {
			tag := "td"
			if i == 0 || cell.IsHeader {
				tag = "th"
			}
			sb.WriteString(fmt.Sprintf("    <%s>%s</%s>\n", tag, escapeHTML(cell.Text), tag))
		}
		sb.WriteString("  </tr>\n")
	}

	sb.WriteString("</table>")
	return sb.String()
}

// shouldSplitTable determines if a table should be split
func (h *TableFigureHandler) shouldSplitTable(table *model.Table, formatted string) bool {
	if len(formatted) > h.config.MaxTableSize {
		return true
	}
	if table.RowCount() > h.config.MaxTableRows {
		return true
	}
	return false
}

// splitTable splits a large table into multiple chunks
func (h *TableFigureHandler) splitTable(table *model.Table, caption string, pageNumber int) []*TableChunk {
	var chunks []*TableChunk

	rowCount := table.RowCount()
	if rowCount == 0 {
		return chunks
	}

	// Determine split points
	rowsPerChunk := h.config.MaxTableRows
	if rowsPerChunk <= 0 {
		rowsPerChunk = 25
	}

	totalSplits := (rowCount + rowsPerChunk - 1) / rowsPerChunk
	headers := h.extractHeaders(table)

	for i := 0; i < totalSplits; i++ {
		startRow := i * rowsPerChunk
		endRow := startRow + rowsPerChunk
		if endRow > rowCount {
			endRow = rowCount
		}

		// Create partial table
		partialTable := h.createPartialTable(table, startRow, endRow, i > 0)

		chunk := &TableChunk{
			Table:         partialTable,
			FormattedText: h.formatTable(partialTable),
			Summary:       fmt.Sprintf("Table part %d of %d (rows %d-%d)", i+1, totalSplits, startRow+1, endRow),
			RowCount:      endRow - startRow,
			ColCount:      table.ColCount(),
			Headers:       headers,
			IsSplit:       true,
			SplitIndex:    i,
			TotalSplits:   totalSplits,
			PageNumber:    pageNumber,
		}

		// Only first chunk gets the caption
		if i == 0 {
			chunk.Caption = caption
			chunk.HasCaption = caption != ""
		}

		chunks = append(chunks, chunk)
	}

	return chunks
}

// createPartialTable creates a subset of a table
func (h *TableFigureHandler) createPartialTable(table *model.Table, startRow, endRow int, includeHeader bool) *model.Table {
	colCount := table.ColCount()

	// Calculate row count (with header if needed)
	rowCount := endRow - startRow
	if includeHeader && startRow > 0 {
		rowCount++ // Add header row
	}

	partial := model.NewTable(rowCount, colCount)

	rowIdx := 0

	// Copy header if this is a continuation
	if includeHeader && startRow > 0 && len(table.Rows) > 0 {
		for j, cell := range table.Rows[0] {
			if j < colCount {
				partial.Rows[rowIdx][j] = cell
			}
		}
		rowIdx++
	}

	// Copy data rows
	for i := startRow; i < endRow && i < len(table.Rows); i++ {
		for j, cell := range table.Rows[i] {
			if j < colCount {
				partial.Rows[rowIdx][j] = cell
			}
		}
		rowIdx++
	}

	return partial
}

// generateTableSummary creates a brief summary of the table
func (h *TableFigureHandler) generateTableSummary(table *model.Table) string {
	if !h.config.IncludeTableSummary {
		return ""
	}

	rows := table.RowCount()
	cols := table.ColCount()

	summary := fmt.Sprintf("Table with %d rows and %d columns", rows, cols)

	// Add header info if available
	headers := h.extractHeaders(table)
	if len(headers) > 0 && len(headers) <= 5 {
		summary += fmt.Sprintf(". Columns: %s", strings.Join(headers, ", "))
	} else if len(headers) > 5 {
		summary += fmt.Sprintf(". Columns include: %s, ...", strings.Join(headers[:5], ", "))
	}

	return summary
}

// extractHeaders extracts column headers from the first row
func (h *TableFigureHandler) extractHeaders(table *model.Table) []string {
	if table.RowCount() == 0 {
		return nil
	}

	var headers []string
	for _, cell := range table.Rows[0] {
		text := strings.TrimSpace(cell.Text)
		if text != "" {
			headers = append(headers, text)
		}
	}

	return headers
}

// generateFigureDescription creates a description for a figure
func (h *TableFigureHandler) generateFigureDescription(chunk *FigureChunk) string {
	var parts []string

	if chunk.HasCaption {
		parts = append(parts, chunk.Caption)
	}

	if h.config.IncludeFigureAltText && chunk.AltText != "" {
		if !chunk.HasCaption || chunk.AltText != chunk.Caption {
			parts = append(parts, chunk.AltText)
		}
	}

	if chunk.Format != "" && chunk.Format != "unknown" {
		parts = append(parts, fmt.Sprintf("[%s image]", strings.ToUpper(chunk.Format)))
	} else {
		parts = append(parts, "[Image]")
	}

	return strings.Join(parts, " - ")
}

// ToChunk converts a TableChunk to a generic Chunk
func (tc *TableChunk) ToChunk(chunkIndex int) *Chunk {
	var text strings.Builder

	// Add caption if present
	if tc.HasCaption {
		text.WriteString(tc.Caption)
		text.WriteString("\n\n")
	}

	// Add summary if present
	if tc.Summary != "" {
		text.WriteString("[")
		text.WriteString(tc.Summary)
		text.WriteString("]\n\n")
	}

	// Add table content
	text.WriteString(tc.FormattedText)

	chunk := &Chunk{
		ID:   fmt.Sprintf("table_%d_%d", tc.PageNumber, chunkIndex),
		Text: text.String(),
		Metadata: ChunkMetadata{
			PageStart:       tc.PageNumber,
			PageEnd:         tc.PageNumber,
			ChunkIndex:      chunkIndex,
			Level:           ChunkLevelParagraph,
			ElementTypes:    []string{"table"},
			HasTable:        true,
			CharCount:       len(text.String()),
			WordCount:       countWords(text.String()),
			EstimatedTokens: len(text.String()) / 4,
		},
	}

	if tc.HasCaption {
		chunk.Metadata.SectionTitle = tc.Caption
	}

	return chunk
}

// ToChunk converts a FigureChunk to a generic Chunk
func (fc *FigureChunk) ToChunk(chunkIndex int) *Chunk {
	text := fc.Description

	chunk := &Chunk{
		ID:   fmt.Sprintf("figure_%d_%d", fc.PageNumber, chunkIndex),
		Text: text,
		Metadata: ChunkMetadata{
			PageStart:       fc.PageNumber,
			PageEnd:         fc.PageNumber,
			ChunkIndex:      chunkIndex,
			Level:           ChunkLevelParagraph,
			ElementTypes:    []string{"image", "figure"},
			HasImage:        true,
			CharCount:       len(text),
			WordCount:       countWords(text),
			EstimatedTokens: len(text) / 4,
		},
	}

	if fc.HasCaption {
		chunk.Metadata.SectionTitle = fc.Caption
	}

	return chunk
}

// CaptionDetector helps find captions associated with tables and figures
type CaptionDetector struct {
	config TableFigureConfig
}

// NewCaptionDetector creates a new caption detector
func NewCaptionDetector() *CaptionDetector {
	return &CaptionDetector{
		config: DefaultTableFigureConfig(),
	}
}

// NewCaptionDetectorWithConfig creates a caption detector with custom config
func NewCaptionDetectorWithConfig(config TableFigureConfig) *CaptionDetector {
	return &CaptionDetector{
		config: config,
	}
}

// FindTableCaption searches for a caption near a table
func (d *CaptionDetector) FindTableCaption(blocks []ContentBlock, tableIndex int) string {
	if tableIndex < 0 || tableIndex >= len(blocks) {
		return ""
	}

	// Check block before table
	if tableIndex > 0 {
		prev := blocks[tableIndex-1]
		if d.isTableCaption(prev.Text) {
			return prev.Text
		}
	}

	// Check block after table
	if tableIndex+1 < len(blocks) {
		next := blocks[tableIndex+1]
		if d.isTableCaption(next.Text) {
			return next.Text
		}
	}

	return ""
}

// FindFigureCaption searches for a caption near a figure
func (d *CaptionDetector) FindFigureCaption(blocks []ContentBlock, figureIndex int) string {
	if figureIndex < 0 || figureIndex >= len(blocks) {
		return ""
	}

	// Check block before figure
	if figureIndex > 0 {
		prev := blocks[figureIndex-1]
		if d.isFigureCaption(prev.Text) {
			return prev.Text
		}
	}

	// Check block after figure
	if figureIndex+1 < len(blocks) {
		next := blocks[figureIndex+1]
		if d.isFigureCaption(next.Text) {
			return next.Text
		}
	}

	return ""
}

// isTableCaption checks if text appears to be a table caption
func (d *CaptionDetector) isTableCaption(text string) bool {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)

	// Check for common table caption patterns
	patterns := []string{
		"table ",
		"tbl ",
		"tbl.",
		"tab ",
		"tab.",
	}

	for _, p := range patterns {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}

	// Check for numbered table reference
	if strings.Contains(lower, "table") && containsNumber(text) {
		return true
	}

	return false
}

// isFigureCaption checks if text appears to be a figure caption
func (d *CaptionDetector) isFigureCaption(text string) bool {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)

	// Check for common figure caption patterns
	patterns := []string{
		"figure ",
		"fig ",
		"fig.",
		"image ",
		"img ",
		"diagram ",
		"illustration ",
		"chart ",
		"graph ",
		"plot ",
	}

	for _, p := range patterns {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}

	// Check for numbered figure reference
	if (strings.Contains(lower, "figure") || strings.Contains(lower, "fig")) && containsNumber(text) {
		return true
	}

	return false
}

// TableFigureResult holds the result of processing tables and figures
type TableFigureResult struct {
	// TableChunks are the processed table chunks
	TableChunks []*TableChunk

	// FigureChunks are the processed figure chunks
	FigureChunks []*FigureChunk

	// Stats contains processing statistics
	Stats TableFigureStats
}

// TableFigureStats contains statistics about table/figure processing
type TableFigureStats struct {
	TotalTables        int
	TotalFigures       int
	TablesWithCaption  int
	FiguresWithCaption int
	SplitTables        int
	TotalTableRows     int
	TotalTableCols     int
}

// ProcessBlocks processes content blocks to extract tables and figures
func (h *TableFigureHandler) ProcessBlocks(blocks []ContentBlock) *TableFigureResult {
	result := &TableFigureResult{}
	detector := NewCaptionDetectorWithConfig(h.config)

	for i, block := range blocks {
		switch block.Type {
		case model.ElementTypeTable:
			caption := ""
			if h.config.IncludeTableCaption {
				caption = detector.FindTableCaption(blocks, i)
			}

			// We need to handle when Table is not available in ContentBlock
			// For now, create chunks from text
			tableChunk := &TableChunk{
				Caption:       caption,
				HasCaption:    caption != "",
				FormattedText: block.Text,
				Summary:       "Table content",
				PageNumber:    block.Page,
			}

			result.TableChunks = append(result.TableChunks, tableChunk)
			result.Stats.TotalTables++
			if caption != "" {
				result.Stats.TablesWithCaption++
			}

		case model.ElementTypeImage, model.ElementTypeFigure:
			caption := ""
			if h.config.IncludeFigureCaption {
				caption = detector.FindFigureCaption(blocks, i)
			}

			figureChunk := &FigureChunk{
				Caption:    caption,
				HasCaption: caption != "",
				PageNumber: block.Page,
			}
			figureChunk.Description = h.generateFigureDescription(figureChunk)

			result.FigureChunks = append(result.FigureChunks, figureChunk)
			result.Stats.TotalFigures++
			if caption != "" {
				result.Stats.FiguresWithCaption++
			}
		}
	}

	return result
}

// Helper functions

// imageFormatString converts an ImageFormat to a string
func imageFormatString(format model.ImageFormat) string {
	switch format {
	case model.ImageFormatJPEG:
		return "jpeg"
	case model.ImageFormatPNG:
		return "png"
	case model.ImageFormatTIFF:
		return "tiff"
	case model.ImageFormatJPEG2000:
		return "jpeg2000"
	case model.ImageFormatJBIG2:
		return "jbig2"
	default:
		return "unknown"
	}
}

// escapeHTML escapes HTML special characters
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// containsNumber checks if a string contains any digit
func containsNumber(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// IsTableElement checks if an element type is a table
func IsTableElement(elementType model.ElementType) bool {
	return elementType == model.ElementTypeTable
}

// IsFigureElement checks if an element type is a figure or image
func IsFigureElement(elementType model.ElementType) bool {
	return elementType == model.ElementTypeImage || elementType == model.ElementTypeFigure
}

// IsCaptionElement checks if an element type is a caption
func IsCaptionElement(elementType model.ElementType) bool {
	return elementType == model.ElementTypeCaption
}
