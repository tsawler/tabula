package model

import (
	"fmt"
	"strings"
)

// Table represents a table with cells organized in rows and columns
type Table struct {
	Rows       [][]Cell
	BBox       BBox
	HasGrid    bool    // Whether table has visible gridlines
	Confidence float64 // Detection confidence (0-1)
	ZOrder     int
}

func (t *Table) Type() ElementType { return ElementTypeTable }
func (t *Table) BoundingBox() BBox { return t.BBox }
func (t *Table) ZIndex() int       { return t.ZOrder }
func (t *Table) GetText() string {
	var sb strings.Builder
	for _, row := range t.Rows {
		for j, cell := range row {
			sb.WriteString(cell.Text)
			if j < len(row)-1 {
				sb.WriteString("\t")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// NewTable creates a new table with given dimensions
func NewTable(rows, cols int) *Table {
	table := &Table{
		Rows:       make([][]Cell, rows),
		Confidence: 1.0,
	}
	for i := 0; i < rows; i++ {
		table.Rows[i] = make([]Cell, cols)
		for j := 0; j < cols; j++ {
			table.Rows[i][j] = Cell{
				RowSpan: 1,
				ColSpan: 1,
			}
		}
	}
	return table
}

// RowCount returns the number of rows
func (t *Table) RowCount() int {
	return len(t.Rows)
}

// ColCount returns the number of columns in the first row
func (t *Table) ColCount() int {
	if len(t.Rows) == 0 {
		return 0
	}
	return len(t.Rows[0])
}

// GetCell returns the cell at the given row and column (0-indexed)
func (t *Table) GetCell(row, col int) *Cell {
	if row < 0 || row >= len(t.Rows) {
		return nil
	}
	if col < 0 || col >= len(t.Rows[row]) {
		return nil
	}
	return &t.Rows[row][col]
}

// SetCell sets the cell at the given position
func (t *Table) SetCell(row, col int, cell Cell) error {
	if row < 0 || row >= len(t.Rows) {
		return fmt.Errorf("row index %d out of bounds", row)
	}
	if col < 0 || col >= len(t.Rows[row]) {
		return fmt.Errorf("col index %d out of bounds", col)
	}
	t.Rows[row][col] = cell
	return nil
}

// ToMarkdown converts the table to markdown format
func (t *Table) ToMarkdown() string {
	if len(t.Rows) == 0 {
		return ""
	}

	var sb strings.Builder

	// Header row
	for j, cell := range t.Rows[0] {
		sb.WriteString("| ")
		sb.WriteString(strings.ReplaceAll(cell.Text, "\n", " "))
		sb.WriteString(" ")
		if j == len(t.Rows[0])-1 {
			sb.WriteString("|")
		}
	}
	sb.WriteString("\n")

	// Separator
	for j := range t.Rows[0] {
		sb.WriteString("|---")
		if j == len(t.Rows[0])-1 {
			sb.WriteString("|")
		}
	}
	sb.WriteString("\n")

	// Data rows
	for i := 1; i < len(t.Rows); i++ {
		for j, cell := range t.Rows[i] {
			sb.WriteString("| ")
			sb.WriteString(strings.ReplaceAll(cell.Text, "\n", " "))
			sb.WriteString(" ")
			if j == len(t.Rows[i])-1 {
				sb.WriteString("|")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ToCSV converts the table to CSV format
func (t *Table) ToCSV() string {
	var sb strings.Builder
	for _, row := range t.Rows {
		for j, cell := range row {
			// Escape quotes and wrap in quotes if necessary
			text := cell.Text
			if strings.Contains(text, ",") || strings.Contains(text, "\"") || strings.Contains(text, "\n") {
				text = "\"" + strings.ReplaceAll(text, "\"", "\"\"") + "\""
			}
			sb.WriteString(text)
			if j < len(row)-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// Cell represents a table cell
type Cell struct {
	Text     string
	BBox     BBox
	RowSpan  int
	ColSpan  int
	IsHeader bool
	// Cell styling
	Style CellStyle
}

// CellStyle represents cell styling
type CellStyle struct {
	BackgroundColor Color
	BorderColor     Color
	BorderWidth     float64
	TextStyle       TextStyle
	Alignment       TextAlignment
	VerticalAlign   VerticalAlignment
}

// VerticalAlignment represents vertical alignment
type VerticalAlignment int

const (
	VAlignTop VerticalAlignment = iota
	VAlignMiddle
	VAlignBottom
)

// TableGrid represents the detected grid structure
type TableGrid struct {
	Rows      []float64 // Y-coordinates of row boundaries
	Cols      []float64 // X-coordinates of column boundaries
	HasHLines []bool    // Horizontal line presence
	HasVLines []bool    // Vertical line presence
}

// NewTableGrid creates a new empty grid
func NewTableGrid() *TableGrid {
	return &TableGrid{
		Rows:      make([]float64, 0),
		Cols:      make([]float64, 0),
		HasHLines: make([]bool, 0),
		HasVLines: make([]bool, 0),
	}
}

// RowCount returns the number of rows
func (g *TableGrid) RowCount() int {
	if len(g.Rows) <= 1 {
		return 0
	}
	return len(g.Rows) - 1
}

// ColCount returns the number of columns
func (g *TableGrid) ColCount() int {
	if len(g.Cols) <= 1 {
		return 0
	}
	return len(g.Cols) - 1
}

// GetCellBBox returns the bounding box for a cell
func (g *TableGrid) GetCellBBox(row, col int) BBox {
	if row < 0 || row >= g.RowCount() || col < 0 || col >= g.ColCount() {
		return BBox{}
	}
	return BBox{
		X:      g.Cols[col],
		Y:      g.Rows[row],
		Width:  g.Cols[col+1] - g.Cols[col],
		Height: g.Rows[row+1] - g.Rows[row],
	}
}
