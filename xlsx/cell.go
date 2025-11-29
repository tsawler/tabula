package xlsx

import (
	"fmt"
	"strconv"
	"strings"
)

// CellType represents the type of data in a cell.
type CellType int

const (
	// CellTypeString indicates a string value.
	CellTypeString CellType = iota
	// CellTypeNumber indicates a numeric value.
	CellTypeNumber
	// CellTypeBoolean indicates a boolean value.
	CellTypeBoolean
	// CellTypeFormula indicates a formula.
	CellTypeFormula
	// CellTypeError indicates an error value.
	CellTypeError
	// CellTypeEmpty indicates an empty cell.
	CellTypeEmpty
)

// String returns the string representation of the cell type.
func (t CellType) String() string {
	switch t {
	case CellTypeString:
		return "string"
	case CellTypeNumber:
		return "number"
	case CellTypeBoolean:
		return "boolean"
	case CellTypeFormula:
		return "formula"
	case CellTypeError:
		return "error"
	case CellTypeEmpty:
		return "empty"
	default:
		return "unknown"
	}
}

// Cell represents a cell in a worksheet.
type Cell struct {
	Value      string   // The cell's display value
	RawValue   string   // The raw value from XML
	Type       CellType // The type of data
	Row        int      // 0-indexed row
	Col        int      // 0-indexed column
	StyleIndex int      // Index into styles
	Formula    string   // Formula if present

	// Merge information
	IsMerged    bool // Is this cell part of a merged region?
	IsMergeRoot bool // Is this the top-left cell of a merged region?
	MergeRows   int  // Number of rows in merge (1 = no merge)
	MergeCols   int  // Number of columns in merge (1 = no merge)
}

// IsEmpty returns true if the cell has no value.
func (c *Cell) IsEmpty() bool {
	return c.Type == CellTypeEmpty || c.Value == ""
}

// Sheet represents a worksheet in the workbook.
type Sheet struct {
	Name   string
	Index  int
	Rows   [][]Cell
	MaxRow int // Maximum row index (0-indexed)
	MaxCol int // Maximum column index (0-indexed)

	// Merged cell regions
	MergedRegions []MergedRegion
}

// MergedRegion represents a merged cell region.
type MergedRegion struct {
	StartRow int
	StartCol int
	EndRow   int
	EndCol   int
}

// Cell returns the cell at the given row and column (0-indexed).
// Returns nil if the cell doesn't exist.
func (s *Sheet) Cell(row, col int) *Cell {
	if row < 0 || row >= len(s.Rows) {
		return nil
	}
	if col < 0 || col >= len(s.Rows[row]) {
		return nil
	}
	return &s.Rows[row][col]
}

// CellByRef returns the cell at the given reference (e.g., "A1").
// Returns nil if the cell doesn't exist.
func (s *Sheet) CellByRef(ref string) *Cell {
	col, row, err := ParseCellRef(ref)
	if err != nil {
		return nil
	}
	return s.Cell(row, col)
}

// RowCount returns the number of rows in the sheet.
func (s *Sheet) RowCount() int {
	return len(s.Rows)
}

// ColCount returns the maximum number of columns in any row.
func (s *Sheet) ColCount() int {
	return s.MaxCol + 1
}

// ParseCellRef parses a cell reference like "A1" or "AA100" into column and row indices (0-indexed).
func ParseCellRef(ref string) (col, row int, err error) {
	if ref == "" {
		return 0, 0, fmt.Errorf("empty cell reference")
	}

	// Find where letters end and numbers begin
	i := 0
	for i < len(ref) && isLetter(ref[i]) {
		i++
	}

	if i == 0 {
		return 0, 0, fmt.Errorf("invalid cell reference: no column letters")
	}
	if i == len(ref) {
		return 0, 0, fmt.Errorf("invalid cell reference: no row number")
	}

	colPart := ref[:i]
	rowPart := ref[i:]

	// Parse column (A=0, B=1, ..., Z=25, AA=26, etc.)
	col = ColumnToIndex(colPart)
	if col < 0 {
		return 0, 0, fmt.Errorf("invalid column: %s", colPart)
	}

	// Parse row (1-indexed in Excel, convert to 0-indexed)
	rowNum, err := strconv.Atoi(rowPart)
	if err != nil || rowNum < 1 {
		return 0, 0, fmt.Errorf("invalid row: %s", rowPart)
	}
	row = rowNum - 1

	return col, row, nil
}

// ColumnToIndex converts a column letter(s) to a 0-indexed column number.
// A=0, B=1, ..., Z=25, AA=26, AB=27, etc.
func ColumnToIndex(col string) int {
	col = strings.ToUpper(col)
	result := 0
	for _, c := range col {
		if c < 'A' || c > 'Z' {
			return -1
		}
		result = result*26 + int(c-'A') + 1
	}
	return result - 1
}

// IndexToColumn converts a 0-indexed column number to column letter(s).
// 0=A, 1=B, ..., 25=Z, 26=AA, 27=AB, etc.
func IndexToColumn(index int) string {
	if index < 0 {
		return ""
	}

	result := ""
	index++ // Convert to 1-indexed for calculation
	for index > 0 {
		index-- // Adjust for 0-based modulo
		result = string(rune('A'+index%26)) + result
		index /= 26
	}
	return result
}

// CellRef creates a cell reference string from column and row indices (0-indexed).
func CellRef(col, row int) string {
	return fmt.Sprintf("%s%d", IndexToColumn(col), row+1)
}

func isLetter(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// ParseRangeRef parses a range reference like "A1:D10" into start and end coordinates.
func ParseRangeRef(ref string) (startCol, startRow, endCol, endRow int, err error) {
	parts := strings.Split(ref, ":")
	if len(parts) != 2 {
		return 0, 0, 0, 0, fmt.Errorf("invalid range reference: %s", ref)
	}

	startCol, startRow, err = ParseCellRef(parts[0])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid start cell: %w", err)
	}

	endCol, endRow, err = ParseCellRef(parts[1])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid end cell: %w", err)
	}

	return startCol, startRow, endCol, endRow, nil
}
