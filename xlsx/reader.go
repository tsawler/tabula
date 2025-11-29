// Package xlsx provides XLSX (Office Open XML Spreadsheet) document parsing.
package xlsx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/rag"
)

// Reader provides access to XLSX document content.
type Reader struct {
	zipReader     *zip.ReadCloser
	workbook      *workbookXML
	sharedStrings []string
	styles        *stylesXML
	rels          *relationshipsXML
	coreProps     *corePropertiesXML
	appProps      *appPropertiesXML
	sheets        []*Sheet
	sheetRels     map[string]string // RID -> target path
}

// Open opens an XLSX file for reading.
func Open(filename string) (*Reader, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, fmt.Errorf("opening ZIP archive: %w", err)
	}

	r := &Reader{
		zipReader: zr,
		sheetRels: make(map[string]string),
	}

	// Validate required files exist
	if err := r.validate(); err != nil {
		zr.Close()
		return nil, err
	}

	// Parse relationships first
	if err := r.parseRelationships(); err != nil {
		zr.Close()
		return nil, fmt.Errorf("parsing relationships: %w", err)
	}

	// Parse workbook to get sheet list
	if err := r.parseWorkbook(); err != nil {
		zr.Close()
		return nil, fmt.Errorf("parsing workbook: %w", err)
	}

	// Parse shared strings (optional but common)
	_ = r.parseSharedStrings()

	// Parse styles (optional)
	_ = r.parseStyles()

	// Parse all worksheets
	if err := r.parseWorksheets(); err != nil {
		zr.Close()
		return nil, fmt.Errorf("parsing worksheets: %w", err)
	}

	// Parse metadata (optional)
	r.parseCoreProperties()
	r.parseAppProperties()

	return r, nil
}

// Close releases resources associated with the Reader.
func (r *Reader) Close() error {
	if r.zipReader != nil {
		err := r.zipReader.Close()
		r.zipReader = nil
		return err
	}
	return nil
}

// validate checks that required XLSX files exist.
func (r *Reader) validate() error {
	required := []string{
		"[Content_Types].xml",
		"xl/workbook.xml",
	}

	fileMap := make(map[string]bool)
	for _, f := range r.zipReader.File {
		fileMap[f.Name] = true
	}

	for _, name := range required {
		if !fileMap[name] {
			return fmt.Errorf("missing required file: %s", name)
		}
	}

	return nil
}

// getFileContent reads the content of a file from the ZIP archive.
func (r *Reader) getFileContent(name string) ([]byte, error) {
	for _, f := range r.zipReader.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

// parseRelationships parses the workbook relationships file.
func (r *Reader) parseRelationships() error {
	data, err := r.getFileContent("xl/_rels/workbook.xml.rels")
	if err != nil {
		// Try alternate location
		data, err = r.getFileContent("xl/_rels/workbook.rels")
		if err != nil {
			return nil // Relationships are optional
		}
	}

	r.rels = &relationshipsXML{}
	if err := xml.Unmarshal(data, r.rels); err != nil {
		return err
	}

	// Build map of RID to target
	for _, rel := range r.rels.Relationship {
		r.sheetRels[rel.ID] = rel.Target
	}

	return nil
}

// parseWorkbook parses the main workbook file.
func (r *Reader) parseWorkbook() error {
	data, err := r.getFileContent("xl/workbook.xml")
	if err != nil {
		return err
	}

	r.workbook = &workbookXML{}
	return xml.Unmarshal(data, r.workbook)
}

// parseSharedStrings parses the shared strings table.
func (r *Reader) parseSharedStrings() error {
	data, err := r.getFileContent("xl/sharedStrings.xml")
	if err != nil {
		return err // Shared strings are optional
	}

	var sst sharedStringsXML
	if err := xml.Unmarshal(data, &sst); err != nil {
		return err
	}

	r.sharedStrings = make([]string, len(sst.SI))
	for i, si := range sst.SI {
		if si.T != "" {
			r.sharedStrings[i] = si.T
		} else {
			// Rich text - concatenate all runs
			var text strings.Builder
			for _, run := range si.R {
				text.WriteString(run.T)
			}
			r.sharedStrings[i] = text.String()
		}
	}

	return nil
}

// parseStyles parses the styles file.
func (r *Reader) parseStyles() error {
	data, err := r.getFileContent("xl/styles.xml")
	if err != nil {
		return err // Styles are optional
	}

	r.styles = &stylesXML{}
	return xml.Unmarshal(data, r.styles)
}

// parseWorksheets parses all worksheet files.
func (r *Reader) parseWorksheets() error {
	if r.workbook == nil {
		return fmt.Errorf("workbook not parsed")
	}

	r.sheets = make([]*Sheet, 0, len(r.workbook.Sheets.Sheet))

	for i, sheetRef := range r.workbook.Sheets.Sheet {
		// Find the sheet file path from relationships
		target := r.sheetRels[sheetRef.RID]
		if target == "" {
			// Try default naming
			target = fmt.Sprintf("worksheets/sheet%d.xml", i+1)
		}

		// Normalize path
		if !strings.HasPrefix(target, "xl/") && !strings.HasPrefix(target, "/") {
			target = "xl/" + target
		}
		target = strings.TrimPrefix(target, "/")

		data, err := r.getFileContent(target)
		if err != nil {
			// Try without xl/ prefix
			target = strings.TrimPrefix(target, "xl/")
			data, err = r.getFileContent("xl/" + target)
			if err != nil {
				continue // Skip sheets we can't read
			}
		}

		sheet, err := r.parseWorksheet(data, sheetRef.Name, i)
		if err != nil {
			continue // Skip sheets that fail to parse
		}

		r.sheets = append(r.sheets, sheet)
	}

	if len(r.sheets) == 0 {
		return fmt.Errorf("no worksheets found")
	}

	return nil
}

// parseWorksheet parses a single worksheet.
func (r *Reader) parseWorksheet(data []byte, name string, index int) (*Sheet, error) {
	var ws worksheetXML
	if err := xml.Unmarshal(data, &ws); err != nil {
		return nil, err
	}

	sheet := &Sheet{
		Name:  name,
		Index: index,
	}

	// Parse merged regions first
	if ws.MergeCells != nil {
		for _, mc := range ws.MergeCells.MergeCell {
			startCol, startRow, endCol, endRow, err := ParseRangeRef(mc.Ref)
			if err != nil {
				continue
			}
			sheet.MergedRegions = append(sheet.MergedRegions, MergedRegion{
				StartRow: startRow,
				StartCol: startCol,
				EndRow:   endRow,
				EndCol:   endCol,
			})
		}
	}

	// Determine dimensions
	maxRow := 0
	maxCol := 0

	// First pass: find dimensions
	for _, row := range ws.SheetData.Rows {
		if row.R > maxRow {
			maxRow = row.R
		}
		for _, cell := range row.Cells {
			col, _, err := ParseCellRef(cell.R)
			if err != nil {
				continue
			}
			if col > maxCol {
				maxCol = col
			}
		}
	}

	sheet.MaxRow = maxRow - 1 // Convert to 0-indexed
	sheet.MaxCol = maxCol

	// Initialize rows
	sheet.Rows = make([][]Cell, maxRow)
	for i := range sheet.Rows {
		sheet.Rows[i] = make([]Cell, maxCol+1)
		for j := range sheet.Rows[i] {
			sheet.Rows[i][j] = Cell{
				Row:       i,
				Col:       j,
				Type:      CellTypeEmpty,
				MergeRows: 1,
				MergeCols: 1,
			}
		}
	}

	// Second pass: populate cells
	for _, row := range ws.SheetData.Rows {
		rowIdx := row.R - 1 // Convert to 0-indexed
		if rowIdx < 0 || rowIdx >= len(sheet.Rows) {
			continue
		}

		for _, cellXML := range row.Cells {
			col, _, err := ParseCellRef(cellXML.R)
			if err != nil {
				continue
			}
			if col < 0 || col >= len(sheet.Rows[rowIdx]) {
				continue
			}

			cell := &sheet.Rows[rowIdx][col]
			cell.RawValue = cellXML.V
			cell.StyleIndex = cellXML.S
			cell.Formula = cellXML.F

			// Determine cell type and value
			switch cellXML.T {
			case "s": // Shared string
				cell.Type = CellTypeString
				idx, err := strconv.Atoi(cellXML.V)
				if err == nil && idx >= 0 && idx < len(r.sharedStrings) {
					cell.Value = r.sharedStrings[idx]
				}
			case "b": // Boolean
				cell.Type = CellTypeBoolean
				if cellXML.V == "1" {
					cell.Value = "TRUE"
				} else {
					cell.Value = "FALSE"
				}
			case "e": // Error
				cell.Type = CellTypeError
				cell.Value = cellXML.V
			case "str": // Inline string formula result
				cell.Type = CellTypeString
				cell.Value = cellXML.V
			case "inlineStr": // Inline string
				cell.Type = CellTypeString
				if cellXML.Is != nil {
					cell.Value = cellXML.Is.T
				}
			default: // Number or empty
				if cellXML.V != "" {
					cell.Type = CellTypeNumber
					cell.Value = r.formatNumber(cellXML.V, cellXML.S)
				} else if cellXML.F != "" {
					cell.Type = CellTypeFormula
					cell.Value = "" // Formula without cached value
				}
			}
		}
	}

	// Apply merged region info to cells
	for _, mr := range sheet.MergedRegions {
		for row := mr.StartRow; row <= mr.EndRow && row < len(sheet.Rows); row++ {
			for col := mr.StartCol; col <= mr.EndCol && col < len(sheet.Rows[row]); col++ {
				cell := &sheet.Rows[row][col]
				cell.IsMerged = true
				if row == mr.StartRow && col == mr.StartCol {
					cell.IsMergeRoot = true
					cell.MergeRows = mr.EndRow - mr.StartRow + 1
					cell.MergeCols = mr.EndCol - mr.StartCol + 1
				}
			}
		}
	}

	return sheet, nil
}

// formatNumber applies number formatting to a value.
func (r *Reader) formatNumber(value string, styleIndex int) string {
	// For now, just return the raw value
	// TODO: Apply number format from styles
	return value
}

// parseCoreProperties parses Dublin Core metadata.
func (r *Reader) parseCoreProperties() {
	data, err := r.getFileContent("docProps/core.xml")
	if err != nil {
		return
	}

	r.coreProps = &corePropertiesXML{}
	xml.Unmarshal(data, r.coreProps)
}

// parseAppProperties parses application metadata.
func (r *Reader) parseAppProperties() {
	data, err := r.getFileContent("docProps/app.xml")
	if err != nil {
		return
	}

	r.appProps = &appPropertiesXML{}
	xml.Unmarshal(data, r.appProps)
}

// SheetCount returns the number of sheets in the workbook.
func (r *Reader) SheetCount() int {
	return len(r.sheets)
}

// SheetNames returns the names of all sheets.
func (r *Reader) SheetNames() []string {
	names := make([]string, len(r.sheets))
	for i, s := range r.sheets {
		names[i] = s.Name
	}
	return names
}

// Sheet returns the sheet at the given index (0-indexed).
func (r *Reader) Sheet(index int) (*Sheet, error) {
	if index < 0 || index >= len(r.sheets) {
		return nil, fmt.Errorf("sheet index %d out of range (0-%d)", index, len(r.sheets)-1)
	}
	return r.sheets[index], nil
}

// SheetByName returns the sheet with the given name.
func (r *Reader) SheetByName(name string) (*Sheet, error) {
	for _, s := range r.sheets {
		if s.Name == name {
			return s, nil
		}
	}
	return nil, fmt.Errorf("sheet not found: %s", name)
}

// PageCount returns the number of "pages" (sheets) in the workbook.
func (r *Reader) PageCount() (int, error) {
	return len(r.sheets), nil
}

// ExtractOptions holds options for text extraction.
type ExtractOptions struct {
	Sheets         []int  // Which sheets to include (0-indexed, empty = all)
	IncludeHeaders bool   // Include sheet names as headers
	Delimiter      string // Cell delimiter (default: tab)
	ExcludeHeaders bool   // For compatibility with other formats
	ExcludeFooters bool   // For compatibility with other formats
}

// Text extracts and returns all text content from the workbook.
func (r *Reader) Text() (string, error) {
	return r.TextWithOptions(ExtractOptions{})
}

// TextWithOptions extracts text content with the specified options.
func (r *Reader) TextWithOptions(opts ExtractOptions) (string, error) {
	delimiter := opts.Delimiter
	if delimiter == "" {
		delimiter = "\t"
	}

	sheets := r.sheets
	if len(opts.Sheets) > 0 {
		sheets = make([]*Sheet, 0, len(opts.Sheets))
		for _, idx := range opts.Sheets {
			if idx >= 0 && idx < len(r.sheets) {
				sheets = append(sheets, r.sheets[idx])
			}
		}
	}

	var result strings.Builder

	for i, sheet := range sheets {
		if i > 0 {
			result.WriteString("\n\n")
		}

		if opts.IncludeHeaders {
			result.WriteString("=== ")
			result.WriteString(sheet.Name)
			result.WriteString(" ===\n")
		}

		for rowIdx, row := range sheet.Rows {
			if rowIdx > 0 {
				result.WriteString("\n")
			}

			for colIdx, cell := range row {
				if colIdx > 0 {
					result.WriteString(delimiter)
				}
				// For merged cells, only output the root cell's value
				if cell.IsMerged && !cell.IsMergeRoot {
					continue
				}
				result.WriteString(cell.Value)
			}
		}
	}

	return result.String(), nil
}

// Markdown returns the workbook content as Markdown.
func (r *Reader) Markdown() (string, error) {
	return r.MarkdownWithOptions(ExtractOptions{})
}

// MarkdownWithOptions returns workbook content as Markdown with options.
func (r *Reader) MarkdownWithOptions(opts ExtractOptions) (string, error) {
	sheets := r.sheets
	if len(opts.Sheets) > 0 {
		sheets = make([]*Sheet, 0, len(opts.Sheets))
		for _, idx := range opts.Sheets {
			if idx >= 0 && idx < len(r.sheets) {
				sheets = append(sheets, r.sheets[idx])
			}
		}
	}

	var result strings.Builder

	for i, sheet := range sheets {
		if i > 0 {
			result.WriteString("\n\n")
		}

		// Sheet name as heading
		result.WriteString("## ")
		result.WriteString(sheet.Name)
		result.WriteString("\n\n")

		// Convert sheet to markdown table
		if len(sheet.Rows) == 0 {
			continue
		}

		// Find actual content bounds (skip empty rows/cols)
		minRow, maxRow, minCol, maxCol := r.findContentBounds(sheet)
		if minRow > maxRow || minCol > maxCol {
			continue // Empty sheet
		}

		// Write table header (first row)
		result.WriteString("|")
		for col := minCol; col <= maxCol; col++ {
			result.WriteString(" ")
			if minRow < len(sheet.Rows) && col < len(sheet.Rows[minRow]) {
				result.WriteString(escapeMarkdown(sheet.Rows[minRow][col].Value))
			}
			result.WriteString(" |")
		}
		result.WriteString("\n")

		// Write separator
		result.WriteString("|")
		for col := minCol; col <= maxCol; col++ {
			result.WriteString("---|")
		}
		result.WriteString("\n")

		// Write data rows
		for row := minRow + 1; row <= maxRow; row++ {
			result.WriteString("|")
			for col := minCol; col <= maxCol; col++ {
				result.WriteString(" ")
				if row < len(sheet.Rows) && col < len(sheet.Rows[row]) {
					cell := sheet.Rows[row][col]
					if !cell.IsMerged || cell.IsMergeRoot {
						result.WriteString(escapeMarkdown(cell.Value))
					}
				}
				result.WriteString(" |")
			}
			result.WriteString("\n")
		}
	}

	return strings.TrimSpace(result.String()), nil
}

// MarkdownWithRAGOptions returns workbook content as Markdown with RAG options.
func (r *Reader) MarkdownWithRAGOptions(extractOpts ExtractOptions, mdOpts rag.MarkdownOptions) (string, error) {
	var result strings.Builder

	// Add YAML front matter metadata if requested
	if mdOpts.IncludeMetadata {
		meta := r.Metadata()
		result.WriteString("---\n")
		if meta.Title != "" {
			result.WriteString(fmt.Sprintf("title: %q\n", meta.Title))
		}
		if meta.Author != "" {
			result.WriteString(fmt.Sprintf("author: %q\n", meta.Author))
		}
		if meta.Subject != "" {
			result.WriteString(fmt.Sprintf("subject: %q\n", meta.Subject))
		}
		if meta.Creator != "" {
			result.WriteString(fmt.Sprintf("generator: %q\n", meta.Creator))
		}
		result.WriteString(fmt.Sprintf("sheets: %d\n", len(r.sheets)))
		result.WriteString("---\n\n")
	}

	// Add table of contents if requested
	if mdOpts.IncludeTableOfContents && len(r.sheets) > 1 {
		result.WriteString("## Table of Contents\n\n")
		for _, sheet := range r.sheets {
			anchor := strings.ToLower(strings.ReplaceAll(sheet.Name, " ", "-"))
			result.WriteString(fmt.Sprintf("- [%s](#%s)\n", sheet.Name, anchor))
		}
		result.WriteString("\n---\n\n")
	}

	// Generate main content
	md, err := r.MarkdownWithOptions(extractOpts)
	if err != nil {
		return "", err
	}
	result.WriteString(md)

	return result.String(), nil
}

// findContentBounds finds the bounds of non-empty cells in a sheet.
func (r *Reader) findContentBounds(sheet *Sheet) (minRow, maxRow, minCol, maxCol int) {
	minRow = len(sheet.Rows)
	maxRow = -1
	minCol = sheet.MaxCol + 1
	maxCol = -1

	for rowIdx, row := range sheet.Rows {
		for colIdx, cell := range row {
			if !cell.IsEmpty() {
				if rowIdx < minRow {
					minRow = rowIdx
				}
				if rowIdx > maxRow {
					maxRow = rowIdx
				}
				if colIdx < minCol {
					minCol = colIdx
				}
				if colIdx > maxCol {
					maxCol = colIdx
				}
			}
		}
	}

	return minRow, maxRow, minCol, maxCol
}

// Metadata returns document metadata.
func (r *Reader) Metadata() model.Metadata {
	meta := model.Metadata{}
	if r.coreProps != nil {
		meta.Title = r.coreProps.Title
		meta.Author = r.coreProps.Creator
		meta.Subject = r.coreProps.Subject
		if r.coreProps.Keywords != "" {
			meta.Keywords = strings.Split(r.coreProps.Keywords, ",")
			for i, kw := range meta.Keywords {
				meta.Keywords[i] = strings.TrimSpace(kw)
			}
		}
	}
	if r.appProps != nil {
		meta.Creator = r.appProps.Application
	}
	return meta
}

// Document returns a model.Document representation of the XLSX content.
func (r *Reader) Document() (*model.Document, error) {
	doc := model.NewDocument()

	// Set metadata
	if r.coreProps != nil {
		doc.Metadata.Title = r.coreProps.Title
		doc.Metadata.Author = r.coreProps.Creator
		doc.Metadata.Subject = r.coreProps.Subject
		if r.coreProps.Keywords != "" {
			doc.Metadata.Keywords = strings.Split(r.coreProps.Keywords, ",")
			for i, kw := range doc.Metadata.Keywords {
				doc.Metadata.Keywords[i] = strings.TrimSpace(kw)
			}
		}
	}
	if r.appProps != nil {
		doc.Metadata.Creator = r.appProps.Application
	}

	// Each sheet becomes a page
	for _, sheet := range r.sheets {
		page := model.NewPage(612, 792) // Standard letter size
		page.Number = sheet.Index + 1

		// Find content bounds
		minRow, maxRow, minCol, maxCol := r.findContentBounds(sheet)
		if minRow > maxRow || minCol > maxCol {
			// Empty sheet - still add the page
			doc.AddPage(page)
			continue
		}

		// Create table from sheet data
		numRows := maxRow - minRow + 1
		numCols := maxCol - minCol + 1

		table := model.NewTable(numRows, numCols)
		table.BBox = model.BBox{
			X:      72,
			Y:      720,
			Width:  468,
			Height: float64(numRows*20 + 20),
		}

		// Build table structure
		for rowIdx := minRow; rowIdx <= maxRow; rowIdx++ {
			tableRow := rowIdx - minRow
			for colIdx := minCol; colIdx <= maxCol; colIdx++ {
				tableCol := colIdx - minCol
				cell := sheet.Rows[rowIdx][colIdx]

				modelCell := model.Cell{
					Text:    cell.Value,
					RowSpan: cell.MergeRows,
					ColSpan: cell.MergeCols,
				}

				// Mark first row as headers
				if tableRow == 0 {
					modelCell.IsHeader = true
				}

				table.Rows[tableRow][tableCol] = modelCell
			}
		}

		page.AddElement(table)
		doc.AddPage(page)
	}

	return doc, nil
}

// Tables returns all sheets as ParsedTable format (for compatibility).
func (r *Reader) Tables() []ParsedTable {
	tables := make([]ParsedTable, len(r.sheets))
	for i, sheet := range r.sheets {
		tables[i] = r.sheetToTable(sheet)
	}
	return tables
}

// ParsedTable represents a table extracted from a sheet.
type ParsedTable struct {
	Name    string
	Rows    [][]string
	Headers []string
}

// sheetToTable converts a sheet to ParsedTable format.
func (r *Reader) sheetToTable(sheet *Sheet) ParsedTable {
	minRow, maxRow, minCol, maxCol := r.findContentBounds(sheet)

	table := ParsedTable{
		Name: sheet.Name,
	}

	if minRow > maxRow || minCol > maxCol {
		return table // Empty
	}

	// First row as headers
	if minRow <= maxRow && minRow < len(sheet.Rows) {
		for col := minCol; col <= maxCol; col++ {
			if col < len(sheet.Rows[minRow]) {
				table.Headers = append(table.Headers, sheet.Rows[minRow][col].Value)
			} else {
				table.Headers = append(table.Headers, "")
			}
		}
	}

	// Remaining rows as data
	for row := minRow + 1; row <= maxRow; row++ {
		var rowData []string
		for col := minCol; col <= maxCol; col++ {
			if row < len(sheet.Rows) && col < len(sheet.Rows[row]) {
				rowData = append(rowData, sheet.Rows[row][col].Value)
			} else {
				rowData = append(rowData, "")
			}
		}
		table.Rows = append(table.Rows, rowData)
	}

	return table
}

// ToText converts a ParsedTable to text.
func (t ParsedTable) ToText() string {
	var result strings.Builder

	// Headers
	if len(t.Headers) > 0 {
		result.WriteString(strings.Join(t.Headers, "\t"))
		result.WriteString("\n")
	}

	// Rows
	for _, row := range t.Rows {
		result.WriteString(strings.Join(row, "\t"))
		result.WriteString("\n")
	}

	return result.String()
}

// ToMarkdown converts a ParsedTable to markdown.
func (t ParsedTable) ToMarkdown() string {
	var result strings.Builder

	if len(t.Headers) == 0 && len(t.Rows) == 0 {
		return ""
	}

	// Headers
	result.WriteString("|")
	for _, h := range t.Headers {
		result.WriteString(" ")
		result.WriteString(escapeMarkdown(h))
		result.WriteString(" |")
	}
	result.WriteString("\n")

	// Separator
	result.WriteString("|")
	for range t.Headers {
		result.WriteString("---|")
	}
	result.WriteString("\n")

	// Rows
	for _, row := range t.Rows {
		result.WriteString("|")
		for _, cell := range row {
			result.WriteString(" ")
			result.WriteString(escapeMarkdown(cell))
			result.WriteString(" |")
		}
		result.WriteString("\n")
	}

	return result.String()
}

// escapeMarkdown escapes special markdown characters in table cells.
func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
