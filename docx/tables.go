package docx

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
	StyleID    string
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
			if cell.IsMergedContinuation {
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
	Cells    []ParsedTableCell
	Height   float64 // Row height in points (0 = auto)
	IsHeader bool
}

// ParsedTableCell represents a parsed table cell.
type ParsedTableCell struct {
	// Content
	Paragraphs []parsedParagraph
	Text       string // Combined text from all paragraphs

	// Structure
	ColSpan int // Number of columns spanned (gridSpan)
	RowSpan int // Number of rows spanned (vMerge)
	IsMergedContinuation bool // True if this is a continuation of a vertical merge

	// Dimensions
	Width float64 // Cell width in points

	// Styling
	VerticalAlign string // top, center, bottom
	Shading       string // Background color (hex)
	HasBorders    bool

	// Nested tables
	NestedTables []ParsedTable
}

// TableParser handles parsing of DOCX tables.
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
		StyleID: tbl.Properties.Style.Val,
	}

	// Parse column widths from grid
	parsed.ColWidths = tp.parseTableGrid(tbl.Grid)

	// Check for borders
	parsed.HasBorders = tp.hasBorders(tbl.Properties.Borders)

	// Parse rows
	for _, row := range tbl.Rows {
		parsedRow := tp.parseRow(row)
		parsed.Rows = append(parsed.Rows, parsedRow)
	}

	// Process vertical merges
	tp.processVerticalMerges(&parsed)

	return parsed
}

// parseTableGrid extracts column widths from the table grid.
func (tp *TableParser) parseTableGrid(grid tableGridXML) []float64 {
	widths := make([]float64, len(grid.Cols))
	for i, col := range grid.Cols {
		widths[i] = parseTwips(col.W)
	}
	return widths
}

// hasBorders checks if the table has visible borders.
func (tp *TableParser) hasBorders(borders tableBordersXML) bool {
	return borders.Top.Val != "" && borders.Top.Val != "nil" ||
		borders.Bottom.Val != "" && borders.Bottom.Val != "nil" ||
		borders.Left.Val != "" && borders.Left.Val != "nil" ||
		borders.Right.Val != "" && borders.Right.Val != "nil" ||
		borders.InsideH.Val != "" && borders.InsideH.Val != "nil" ||
		borders.InsideV.Val != "" && borders.InsideV.Val != "nil"
}

// parseRow parses a table row.
func (tp *TableParser) parseRow(row tableRowXML) ParsedTableRow {
	parsed := ParsedTableRow{
		IsHeader: row.Properties.Header.XMLName.Local != "",
	}

	// Parse row height
	if row.Properties.Height.Val != "" {
		parsed.Height = parseTwips(row.Properties.Height.Val)
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
		ColSpan: 1,
		RowSpan: 1,
	}

	props := cell.Properties

	// Parse column span (gridSpan)
	if props.GridSpan.Val != "" {
		if span, err := strconv.Atoi(props.GridSpan.Val); err == nil && span > 0 {
			parsed.ColSpan = span
		}
	}

	// Parse vertical merge
	if props.VMerge.Val == "restart" {
		// This cell starts a vertical merge
		parsed.RowSpan = 1 // Will be calculated in processVerticalMerges
	} else if props.VMerge.Val == "" && props.VMerge.XMLName.Local == "vMerge" {
		// This cell continues a vertical merge (empty val means continue)
		parsed.IsMergedContinuation = true
	}

	// Parse width
	if props.Width.W != "" {
		parsed.Width = parseTwipsOrPercent(props.Width.W, props.Width.Type)
	}

	// Parse vertical alignment
	parsed.VerticalAlign = props.VAlign.Val
	if parsed.VerticalAlign == "" {
		parsed.VerticalAlign = "top"
	}

	// Parse shading (background color)
	if props.Shading.Fill != "" && props.Shading.Fill != "auto" {
		parsed.Shading = props.Shading.Fill
	}

	// Check for borders
	parsed.HasBorders = tp.hasBorders(props.Borders)

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
		StyleID: p.Properties.Style.Val,
	}

	// Resolve style if available
	if tp.styleResolver != nil {
		resolved := tp.styleResolver.Resolve(parsed.StyleID)
		parsed.StyleName = resolved.Name
		parsed.Alignment = resolved.Alignment
	}

	// Apply direct formatting
	if p.Properties.Justification.Val != "" {
		parsed.Alignment = p.Properties.Justification.Val
	}

	// Extract text from runs
	var textParts []string
	for _, run := range p.Runs {
		for _, t := range run.Text {
			textParts = append(textParts, t.Value)
		}
	}
	parsed.Text = strings.Join(textParts, "")

	return parsed
}

// processVerticalMerges calculates row spans for vertically merged cells.
func (tp *TableParser) processVerticalMerges(table *ParsedTable) {
	if len(table.Rows) == 0 {
		return
	}

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

	// Track merge starts for each column
	mergeStarts := make([]int, colCount) // Row index where merge started
	for i := range mergeStarts {
		mergeStarts[i] = -1
	}

	for rowIdx, row := range table.Rows {
		colIdx := 0
		for cellIdx := range row.Cells {
			cell := &table.Rows[rowIdx].Cells[cellIdx]

			// Check if this cell starts a merge
			if !cell.IsMergedContinuation && mergeStarts[colIdx] == -1 {
				// Check if vMerge restart
				// We need to look at the raw vMerge value
				// For now, assume any cell that's not a continuation could start a merge
				mergeStarts[colIdx] = rowIdx
			}

			if cell.IsMergedContinuation && mergeStarts[colIdx] >= 0 {
				// Increment the row span of the merge start cell
				startRow := mergeStarts[colIdx]
				startColIdx := tp.findCellAtColumn(table.Rows[startRow], colIdx)
				if startColIdx >= 0 {
					table.Rows[startRow].Cells[startColIdx].RowSpan++
				}
			} else if !cell.IsMergedContinuation {
				// Reset merge tracking for this column
				mergeStarts[colIdx] = -1
			}

			colIdx += cell.ColSpan
		}
	}
}

// findCellAtColumn finds the cell index that covers the given column.
func (tp *TableParser) findCellAtColumn(row ParsedTableRow, targetCol int) int {
	col := 0
	for i, cell := range row.Cells {
		if col == targetCol {
			return i
		}
		col += cell.ColSpan
		if col > targetCol {
			return i
		}
	}
	return -1
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
				count += cell.ColSpan
			}
			if count > colCount {
				colCount = count
			}
		}
	}

	rowCount := len(pt.Rows)
	table := model.NewTable(rowCount, colCount)
	table.HasGrid = pt.HasBorders
	table.Confidence = 1.0 // DOCX tables are explicit

	// Fill cells
	for rowIdx, row := range pt.Rows {
		colIdx := 0
		for _, cell := range row.Cells {
			if colIdx >= colCount {
				break
			}

			// Skip merged continuation cells
			if cell.IsMergedContinuation {
				colIdx += cell.ColSpan
				continue
			}

			modelCell := model.Cell{
				Text:     cell.Text,
				ColSpan:  cell.ColSpan,
				RowSpan:  cell.RowSpan,
				IsHeader: row.IsHeader,
			}

			// Set vertical alignment
			switch cell.VerticalAlign {
			case "center":
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

// parseTwipsOrPercent parses a width value that could be in twips or percent.
func parseTwipsOrPercent(value, widthType string) float64 {
	switch widthType {
	case "pct":
		// Percent value (stored as 50ths of a percent)
		if pct, err := strconv.ParseFloat(value, 64); err == nil {
			return pct / 50 // Convert to percentage
		}
	case "dxa":
		// Twips
		return parseTwips(value)
	case "auto":
		return 0
	default:
		// Default to twips
		return parseTwips(value)
	}
	return 0
}
