package pptx

// Slide represents a parsed slide.
type Slide struct {
	Index   int         // 0-indexed slide number
	Title   string      // Slide title (from title placeholder)
	Content []TextBlock // Text content in reading order
	Tables  []Table     // Tables on the slide
	Notes   string      // Speaker notes
}

// TextBlock represents a block of text on a slide.
type TextBlock struct {
	Text        string
	Paragraphs  []Paragraph
	IsTitle     bool   // Is this the slide title?
	IsSubtitle  bool   // Is this a subtitle?
	Placeholder string // Placeholder type (title, body, etc.)
	X, Y        int    // Position in EMUs
	Width       int    // Width in EMUs
	Height      int    // Height in EMUs
}

// Paragraph represents a paragraph within a text block.
type Paragraph struct {
	Text       string
	Level      int    // Bullet/indent level (0 = top level)
	IsBullet   bool   // Has bullet point
	IsNumbered bool   // Is numbered list
	BulletChar string // Bullet character (if custom)
	Alignment  string // l, ctr, r, just
	Runs       []Run  // Text runs with formatting
}

// Run represents a text run with consistent formatting.
type Run struct {
	Text     string
	Bold     bool
	Italic   bool
	FontSize int // In hundredths of a point
}

// Table represents a table on a slide.
type Table struct {
	Rows    [][]TableCell
	Columns int
	X, Y    int // Position in EMUs
	Width   int // Width in EMUs
	Height  int // Height in EMUs
}

// TableCell represents a cell in a table.
type TableCell struct {
	Text     string
	RowSpan  int
	ColSpan  int
	IsMerged bool // Part of a merged cell (not the origin)
}

// GetText returns all text from the slide as a single string.
func (s *Slide) GetText() string {
	var result string

	// Title first
	if s.Title != "" {
		result = s.Title + "\n\n"
	}

	// Then content
	for _, block := range s.Content {
		if block.IsTitle {
			continue // Already added
		}
		for _, para := range block.Paragraphs {
			if para.Text != "" {
				if para.IsBullet || para.IsNumbered {
					// Add indentation for bullet levels
					for i := 0; i < para.Level; i++ {
						result += "  "
					}
					if para.IsNumbered {
						result += "• " // Use bullet for now, could track numbering
					} else if para.BulletChar != "" {
						result += para.BulletChar + " "
					} else {
						result += "• "
					}
				}
				result += para.Text + "\n"
			}
		}
		result += "\n"
	}

	return result
}

// GetMarkdown returns the slide content as markdown.
func (s *Slide) GetMarkdown() string {
	var result string

	// Title as H1
	if s.Title != "" {
		result = "# " + s.Title + "\n\n"
	}

	// Content
	for _, block := range s.Content {
		if block.IsTitle {
			continue // Already added
		}

		for _, para := range block.Paragraphs {
			if para.Text == "" {
				continue
			}

			if para.IsBullet || para.IsNumbered {
				// Add indentation for bullet levels
				indent := ""
				for i := 0; i < para.Level; i++ {
					indent += "  "
				}
				if para.IsNumbered {
					result += indent + "1. " + para.Text + "\n"
				} else {
					result += indent + "- " + para.Text + "\n"
				}
			} else {
				result += para.Text + "\n\n"
			}
		}
	}

	// Tables
	for _, table := range s.Tables {
		result += "\n" + table.ToMarkdown() + "\n"
	}

	return result
}

// ToMarkdown converts a table to markdown format.
func (t *Table) ToMarkdown() string {
	if len(t.Rows) == 0 {
		return ""
	}

	var result string

	// Header row
	result += "|"
	for _, cell := range t.Rows[0] {
		result += " " + escapeMarkdown(cell.Text) + " |"
	}
	result += "\n"

	// Separator
	result += "|"
	for range t.Rows[0] {
		result += "---|"
	}
	result += "\n"

	// Data rows
	for i := 1; i < len(t.Rows); i++ {
		result += "|"
		for _, cell := range t.Rows[i] {
			result += " " + escapeMarkdown(cell.Text) + " |"
		}
		result += "\n"
	}

	return result
}

// escapeMarkdown escapes special markdown characters.
func escapeMarkdown(s string) string {
	// Replace pipe and newlines for table cells
	result := s
	for _, old := range []string{"|", "\n", "\r"} {
		if old == "|" {
			result = replaceAll(result, old, "\\|")
		} else {
			result = replaceAll(result, old, " ")
		}
	}
	return result
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i <= len(s)-len(old) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}
