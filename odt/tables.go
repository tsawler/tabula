package odt

import (
	"strconv"
	"strings"

	"github.com/tsawler/tabula/model"
)

// ParsedTable represents a parsed table with resolved structure.
type ParsedTable struct {
	Rows       []ParsedTableRow
	ColWidths  []float64 // Column widths in points
	HasBorders bool
	StyleName  string
}

// ToText returns a plain text representation of the table.
func (pt *ParsedTable) ToText() string {
	var sb strings.Builder
	for i, row := range pt.Rows {
		if i > 0 {
			sb.WriteString("\n")
		}
		for j, cell := range row.Cells {
			if j > 0 {
				sb.WriteString("\t")
			}
			// Replace newlines within cells with spaces
			text := strings.ReplaceAll(cell.Text, "\n", " ")
			sb.WriteString(text)
		}
	}
	return sb.String()
}

// ToMarkdown returns a markdown table representation.
func (pt *ParsedTable) ToMarkdown() string {
	if len(pt.Rows) == 0 {
		return ""
	}

	var sb strings.Builder

	// Determine column count from first row
	colCount := 0
	for _, row := range pt.Rows {
		count := 0
		for _, cell := range row.Cells {
			span := cell.ColSpan
			if span < 1 {
				span = 1
			}
			count += span
		}
		if count > colCount {
			colCount = count
		}
	}

	if colCount == 0 {
		return ""
	}

	// Write each row
	for rowIdx, row := range pt.Rows {
		sb.WriteString("|")
		colIdx := 0
		for _, cell := range row.Cells {
			if cell.IsCovered {
				continue
			}
			// Replace newlines and pipes within cells
			text := strings.ReplaceAll(cell.Text, "\n", " ")
			text = strings.ReplaceAll(text, "|", "\\|")
			text = strings.TrimSpace(text)
			sb.WriteString(" ")
			sb.WriteString(text)
			sb.WriteString(" |")

			span := cell.ColSpan
			if span < 1 {
				span = 1
			}
			colIdx += span
		}
		// Pad remaining columns if needed
		for colIdx < colCount {
			sb.WriteString(" |")
			colIdx++
		}
		sb.WriteString("\n")

		// Add header separator after first row
		if rowIdx == 0 {
			sb.WriteString("|")
			for i := 0; i < colCount; i++ {
				sb.WriteString(" --- |")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// ColCount returns the number of columns in the table.
func (pt *ParsedTable) ColCount() int {
	if len(pt.Rows) == 0 {
		return 0
	}
	count := 0
	for _, cell := range pt.Rows[0].Cells {
		span := cell.ColSpan
		if span < 1 {
			span = 1
		}
		count += span
	}
	return count
}

// ParsedTableRow represents a parsed table row.
type ParsedTableRow struct {
	Cells     []ParsedTableCell
	Height    float64 // Row height in points (0 = auto)
	StyleName string
}

// ParsedTableCell represents a parsed table cell.
type ParsedTableCell struct {
	// Content
	Paragraphs []parsedParagraph
	Text       string // Combined text from all paragraphs

	// Structure
	ColSpan   int  // Number of columns spanned
	RowSpan   int  // Number of rows spanned
	IsCovered bool // True if this is a covered cell (part of a merge)

	// Dimensions
	Width float64 // Cell width in points

	// Styling
	VerticalAlign string // top, middle, bottom
	Background    string // Background color (hex)
	HasBorders    bool
	StyleName     string
}

// TableParser handles parsing of ODT tables.
type TableParser struct {
	styleResolver *StyleResolver
}

// NewTableParser creates a new table parser.
func NewTableParser(resolver *StyleResolver) *TableParser {
	return &TableParser{
		styleResolver: resolver,
	}
}

// ParseTable parses a table XML element into a ParsedTable.
func (tp *TableParser) ParseTable(tbl tableXML) ParsedTable {
	parsed := ParsedTable{
		StyleName: tbl.StyleName,
	}

	// Parse column widths from column definitions
	parsed.ColWidths = tp.parseTableColumns(tbl.Columns)

	// Parse rows
	for _, row := range tbl.Rows {
		parsedRow := tp.parseRow(row)
		parsed.Rows = append(parsed.Rows, parsedRow)
	}

	// Process row spans to mark covered cells
	tp.processRowSpans(&parsed)

	return parsed
}

// parseTableColumns extracts column widths from column definitions.
func (tp *TableParser) parseTableColumns(cols []tableColXML) []float64 {
	var widths []float64

	for _, col := range cols {
		width := 0.0

		// Get width from style if available
		if tp.styleResolver != nil && col.StyleName != "" {
			if style, ok := tp.styleResolver.styles[col.StyleName]; ok {
				if style.TableColumnProps != nil && style.TableColumnProps.ColumnWidth != "" {
					width = parseLength(style.TableColumnProps.ColumnWidth)
				}
			}
		}

		// Handle repeated columns
		repeat := 1
		if col.NumberRepeated != "" {
			if r, err := strconv.Atoi(col.NumberRepeated); err == nil && r > 0 {
				repeat = r
			}
		}

		for i := 0; i < repeat; i++ {
			widths = append(widths, width)
		}
	}

	return widths
}

// parseRow parses a table row.
func (tp *TableParser) parseRow(row tableRowXML) ParsedTableRow {
	parsed := ParsedTableRow{
		StyleName: row.StyleName,
	}

	// Get row height from style
	if tp.styleResolver != nil && row.StyleName != "" {
		if style, ok := tp.styleResolver.styles[row.StyleName]; ok {
			if style.TableRowProps != nil {
				if style.TableRowProps.RowHeight != "" {
					parsed.Height = parseLength(style.TableRowProps.RowHeight)
				} else if style.TableRowProps.MinRowHeight != "" {
					parsed.Height = parseLength(style.TableRowProps.MinRowHeight)
				}
			}
		}
	}

	// Parse cells
	for _, cell := range row.Cells {
		parsedCell := tp.parseCell(cell)
		parsed.Cells = append(parsed.Cells, parsedCell)
	}

	return parsed
}

// parseCell parses a table cell.
func (tp *TableParser) parseCell(cell tableCellXML) ParsedTableCell {
	parsed := ParsedTableCell{
		ColSpan:   1,
		RowSpan:   1,
		StyleName: cell.StyleName,
	}

	// Parse column span
	if cell.NumberColumnsSpanned != "" {
		if span, err := strconv.Atoi(cell.NumberColumnsSpanned); err == nil && span > 0 {
			parsed.ColSpan = span
		}
	}

	// Parse row span
	if cell.NumberRowsSpanned != "" {
		if span, err := strconv.Atoi(cell.NumberRowsSpanned); err == nil && span > 0 {
			parsed.RowSpan = span
		}
	}

	// Get cell properties from style
	if tp.styleResolver != nil && cell.StyleName != "" {
		if style, ok := tp.styleResolver.styles[cell.StyleName]; ok {
			if style.TableCellProps != nil {
				parsed.VerticalAlign = style.TableCellProps.VerticalAlign
				parsed.Background = style.TableCellProps.BackgroundColor
				// Check for borders
				parsed.HasBorders = style.TableCellProps.Border != "" ||
					style.TableCellProps.BorderTop != "" ||
					style.TableCellProps.BorderBottom != "" ||
					style.TableCellProps.BorderLeft != "" ||
					style.TableCellProps.BorderRight != ""
			}
		}
	}

	// Parse paragraphs within the cell
	for _, para := range cell.Paragraphs {
		parsedPara := tp.parseCellParagraph(para)
		parsed.Paragraphs = append(parsed.Paragraphs, parsedPara)
	}

	// Combine text from all paragraphs
	var textParts []string
	for _, para := range parsed.Paragraphs {
		if para.Text != "" {
			textParts = append(textParts, para.Text)
		}
	}
	parsed.Text = strings.Join(textParts, "\n")

	return parsed
}

// parseCellParagraph parses a paragraph within a table cell.
func (tp *TableParser) parseCellParagraph(p paragraphXML) parsedParagraph {
	parsed := parsedParagraph{
		StyleName: p.StyleName,
	}

	// Resolve style if available
	if tp.styleResolver != nil {
		resolved := tp.styleResolver.Resolve(p.StyleName)
		parsed.Alignment = resolved.Alignment
	}

	// Extract text
	var textParts []string

	// Direct text content
	if p.Text != "" {
		textParts = append(textParts, p.Text)
	}

	// Text from spans
	for _, span := range p.Spans {
		if span.Text != "" {
			textParts = append(textParts, span.Text)
		}
	}

	parsed.Text = strings.Join(textParts, "")

	return parsed
}

// processRowSpans marks cells that are covered by row spans.
func (tp *TableParser) processRowSpans(table *ParsedTable) {
	if len(table.Rows) == 0 {
		return
	}

	// Track row spans in progress for each column
	// rowSpansRemaining[col] = number of rows still to be covered
	colCount := 0
	for _, row := range table.Rows {
		count := 0
		for _, cell := range row.Cells {
			count += cell.ColSpan
		}
		if count > colCount {
			colCount = count
		}
	}

	rowSpansRemaining := make([]int, colCount)

	for rowIdx := range table.Rows {
		colIdx := 0
		newCells := make([]ParsedTableCell, 0)

		for cellIdx := range table.Rows[rowIdx].Cells {
			// Skip columns still covered by row spans from above
			for colIdx < colCount && rowSpansRemaining[colIdx] > 0 {
				// Insert a covered cell placeholder
				newCells = append(newCells, ParsedTableCell{
					IsCovered: true,
					ColSpan:   1,
					RowSpan:   1,
				})
				rowSpansRemaining[colIdx]--
				colIdx++
			}

			if colIdx >= colCount {
				break
			}

			cell := &table.Rows[rowIdx].Cells[cellIdx]
			newCells = append(newCells, *cell)

			// Track row spans for this cell's columns
			if cell.RowSpan > 1 {
				for c := 0; c < cell.ColSpan && colIdx+c < colCount; c++ {
					rowSpansRemaining[colIdx+c] = cell.RowSpan - 1
				}
			}

			colIdx += cell.ColSpan
		}

		// Fill any remaining columns covered by row spans
		for colIdx < colCount && rowSpansRemaining[colIdx] > 0 {
			newCells = append(newCells, ParsedTableCell{
				IsCovered: true,
				ColSpan:   1,
				RowSpan:   1,
			})
			rowSpansRemaining[colIdx]--
			colIdx++
		}

		table.Rows[rowIdx].Cells = newCells
	}
}

// ToModelTable converts a ParsedTable to a model.Table.
func (pt *ParsedTable) ToModelTable() *model.Table {
	if len(pt.Rows) == 0 {
		return model.NewTable(0, 0)
	}

	// Determine column count from grid or first row
	colCount := len(pt.ColWidths)
	if colCount == 0 {
		// Calculate from cells
		for _, row := range pt.Rows {
			count := 0
			for _, cell := range row.Cells {
				if !cell.IsCovered {
					count += cell.ColSpan
				} else {
					count++
				}
			}
			if count > colCount {
				colCount = count
			}
		}
	}

	rowCount := len(pt.Rows)
	table := model.NewTable(rowCount, colCount)
	table.HasGrid = pt.HasBorders
	table.Confidence = 1.0 // ODT tables are explicit

	// Fill cells
	for rowIdx, row := range pt.Rows {
		colIdx := 0
		for _, cell := range row.Cells {
			if colIdx >= colCount {
				break
			}

			// Skip covered cells
			if cell.IsCovered {
				colIdx++
				continue
			}

			modelCell := model.Cell{
				Text:    cell.Text,
				ColSpan: cell.ColSpan,
				RowSpan: cell.RowSpan,
			}

			// Set vertical alignment
			switch cell.VerticalAlign {
			case "middle":
				modelCell.Style.VerticalAlign = model.VAlignMiddle
			case "bottom":
				modelCell.Style.VerticalAlign = model.VAlignBottom
			default:
				modelCell.Style.VerticalAlign = model.VAlignTop
			}

			// Set cell in table
			if rowIdx < rowCount && colIdx < colCount {
				table.SetCell(rowIdx, colIdx, modelCell)
			}

			colIdx += cell.ColSpan
		}
	}

	return table
}
