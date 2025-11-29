// Package htmldoc provides HTML document parsing.
package htmldoc

// parsedElement represents a parsed element from the HTML document.
type parsedElement struct {
	Type    ElementType
	Text    string
	Level   int        // For headings (1-6)
	Items   []listItem // For lists
	Ordered bool       // For lists
	Table   *ParsedTable
	LinkURL string // For links
	IsCode  bool   // For code blocks
}

// ElementType represents the type of HTML element.
type ElementType int

const (
	ElementParagraph ElementType = iota
	ElementHeading
	ElementList
	ElementTable
	ElementCode
	ElementBlockquote
	ElementLink
)

// NavigationExclusionMode controls how navigation, headers, and footers are filtered.
type NavigationExclusionMode int

const (
	// NavigationExclusionNone includes all content without filtering.
	NavigationExclusionNone NavigationExclusionMode = iota

	// NavigationExclusionExplicit skips only explicit semantic HTML5 elements:
	// <nav>, <aside>, and ARIA roles (role="navigation", role="complementary").
	// <header> and <footer> are only skipped when they are direct children of <body>
	// or a single top-level wrapper element.
	NavigationExclusionExplicit

	// NavigationExclusionStandard (default) combines explicit element detection with
	// common class/id pattern matching. This catches navigation and boilerplate content
	// even when sites don't use semantic HTML5 elements.
	// Patterns matched include: nav, navbar, navigation, menu, footer, sidebar, etc.
	NavigationExclusionStandard

	// NavigationExclusionAggressive adds link-density heuristics to standard detection.
	// Sections with very high link-to-text ratios are excluded. This may occasionally
	// exclude legitimate content like link-heavy documentation or "related articles" sections.
	NavigationExclusionAggressive
)

// listItem represents an item in a list.
type listItem struct {
	Text  string
	Level int
}

// ParsedTable represents a table extracted from HTML.
type ParsedTable struct {
	Rows      [][]TableCell
	HasHeader bool
}

// TableCell represents a cell in an HTML table.
type TableCell struct {
	Text     string
	IsHeader bool
	RowSpan  int
	ColSpan  int
}

// ToMarkdown converts the table to markdown format.
func (t *ParsedTable) ToMarkdown() string {
	if len(t.Rows) == 0 {
		return ""
	}

	var result string

	// First row (header or first data row)
	firstRow := t.Rows[0]
	result += "|"
	for _, cell := range firstRow {
		result += " " + escapeMarkdown(cell.Text) + " |"
	}
	result += "\n"

	// Separator
	result += "|"
	for range firstRow {
		result += " --- |"
	}
	result += "\n"

	// Data rows (skip first if it was header)
	startRow := 1
	if !t.HasHeader && len(t.Rows) > 1 {
		startRow = 0
	}

	for i := startRow; i < len(t.Rows); i++ {
		result += "|"
		for _, cell := range t.Rows[i] {
			result += " " + escapeMarkdown(cell.Text) + " |"
		}
		result += "\n"
	}

	return result
}

// escapeMarkdown escapes special markdown characters in text.
func escapeMarkdown(text string) string {
	// Replace pipe characters which break markdown tables
	result := ""
	for _, r := range text {
		switch r {
		case '|':
			result += "\\|"
		case '\n':
			result += " "
		case '\r':
			// Skip
		default:
			result += string(r)
		}
	}
	return result
}
