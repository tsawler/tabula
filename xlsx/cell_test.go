package xlsx

import "testing"

func TestParseCellRef(t *testing.T) {
	tests := []struct {
		ref     string
		wantCol int
		wantRow int
		wantErr bool
	}{
		{"A1", 0, 0, false},
		{"B1", 1, 0, false},
		{"Z1", 25, 0, false},
		{"AA1", 26, 0, false},
		{"AB1", 27, 0, false},
		{"AZ1", 51, 0, false},
		{"BA1", 52, 0, false},
		{"A10", 0, 9, false},
		{"C100", 2, 99, false},
		{"AA100", 26, 99, false},
		{"XFD1048576", 16383, 1048575, false}, // Max Excel cell
		{"", 0, 0, true},
		{"1", 0, 0, true},
		{"A", 0, 0, true},
		{"A0", 0, 0, true},
		{"A-1", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			col, row, err := ParseCellRef(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseCellRef(%q) expected error, got col=%d, row=%d", tt.ref, col, row)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseCellRef(%q) unexpected error: %v", tt.ref, err)
				return
			}
			if col != tt.wantCol {
				t.Errorf("ParseCellRef(%q) col = %d, want %d", tt.ref, col, tt.wantCol)
			}
			if row != tt.wantRow {
				t.Errorf("ParseCellRef(%q) row = %d, want %d", tt.ref, row, tt.wantRow)
			}
		})
	}
}

func TestColumnToIndex(t *testing.T) {
	tests := []struct {
		col  string
		want int
	}{
		{"A", 0},
		{"B", 1},
		{"Z", 25},
		{"AA", 26},
		{"AB", 27},
		{"AZ", 51},
		{"BA", 52},
		{"ZZ", 701},
		{"AAA", 702},
		{"XFD", 16383}, // Excel max column
		{"a", 0},       // Lowercase
		{"aa", 26},
	}

	for _, tt := range tests {
		t.Run(tt.col, func(t *testing.T) {
			got := ColumnToIndex(tt.col)
			if got != tt.want {
				t.Errorf("ColumnToIndex(%q) = %d, want %d", tt.col, got, tt.want)
			}
		})
	}
}

func TestIndexToColumn(t *testing.T) {
	tests := []struct {
		index int
		want  string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{51, "AZ"},
		{52, "BA"},
		{701, "ZZ"},
		{702, "AAA"},
		{16383, "XFD"},
		{-1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := IndexToColumn(tt.index)
			if got != tt.want {
				t.Errorf("IndexToColumn(%d) = %q, want %q", tt.index, got, tt.want)
			}
		})
	}
}

func TestCellRef(t *testing.T) {
	tests := []struct {
		col  int
		row  int
		want string
	}{
		{0, 0, "A1"},
		{1, 0, "B1"},
		{25, 0, "Z1"},
		{26, 0, "AA1"},
		{0, 9, "A10"},
		{2, 99, "C100"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := CellRef(tt.col, tt.row)
			if got != tt.want {
				t.Errorf("CellRef(%d, %d) = %q, want %q", tt.col, tt.row, got, tt.want)
			}
		})
	}
}

func TestParseRangeRef(t *testing.T) {
	tests := []struct {
		ref                                      string
		wantStartCol, wantStartRow               int
		wantEndCol, wantEndRow                   int
		wantErr                                  bool
	}{
		{"A1:B2", 0, 0, 1, 1, false},
		{"A1:D10", 0, 0, 3, 9, false},
		{"B5:F20", 1, 4, 5, 19, false},
		{"AA1:AB10", 26, 0, 27, 9, false},
		{"A1", 0, 0, 0, 0, true},  // No colon
		{"A1:B", 0, 0, 0, 0, true}, // Invalid end
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			startCol, startRow, endCol, endRow, err := ParseRangeRef(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRangeRef(%q) expected error", tt.ref)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseRangeRef(%q) unexpected error: %v", tt.ref, err)
				return
			}
			if startCol != tt.wantStartCol || startRow != tt.wantStartRow ||
				endCol != tt.wantEndCol || endRow != tt.wantEndRow {
				t.Errorf("ParseRangeRef(%q) = (%d,%d,%d,%d), want (%d,%d,%d,%d)",
					tt.ref, startCol, startRow, endCol, endRow,
					tt.wantStartCol, tt.wantStartRow, tt.wantEndCol, tt.wantEndRow)
			}
		})
	}
}

func TestCellType_String(t *testing.T) {
	tests := []struct {
		ct   CellType
		want string
	}{
		{CellTypeString, "string"},
		{CellTypeNumber, "number"},
		{CellTypeBoolean, "boolean"},
		{CellTypeFormula, "formula"},
		{CellTypeError, "error"},
		{CellTypeEmpty, "empty"},
		{CellType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.ct.String(); got != tt.want {
				t.Errorf("CellType(%d).String() = %q, want %q", tt.ct, got, tt.want)
			}
		})
	}
}
