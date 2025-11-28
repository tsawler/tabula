package docx

import (
	"strconv"
	"strings"

	"github.com/tsawler/tabula/model"
)

// ListType represents the type of list.
type ListType int

const (
	ListTypeUnordered ListType = iota // Bullet list
	ListTypeOrdered                   // Numbered list
)

// ParsedList represents a parsed list with its items.
type ParsedList struct {
	Items   []ParsedListItem
	Type    ListType
	NumID   string // Numbering ID from document
	StartAt int    // Starting number for ordered lists
}

// ParsedListItem represents a single list item.
type ParsedListItem struct {
	Text   string
	Level  int    // Indentation level (0-based)
	Bullet string // The bullet character or number prefix
	NumID  string
}

// NumberingResolver resolves numbering definitions from numbering.xml.
type NumberingResolver struct {
	abstractNums map[string]*abstractNumXML // abstractNumId -> definition
	numMappings  map[string]string          // numId -> abstractNumId
}

// NewNumberingResolver creates a resolver from parsed numbering.xml.
func NewNumberingResolver(numbering *numberingXML) *NumberingResolver {
	nr := &NumberingResolver{
		abstractNums: make(map[string]*abstractNumXML),
		numMappings:  make(map[string]string),
	}

	if numbering == nil {
		return nr
	}

	// Build abstract numbering map
	for i := range numbering.AbstractNums {
		an := &numbering.AbstractNums[i]
		nr.abstractNums[an.AbstractNumID] = an
	}

	// Build num -> abstractNum mapping
	for _, num := range numbering.Nums {
		nr.numMappings[num.NumID] = num.AbstractNumID.Val
	}

	return nr
}

// ResolveLevel returns the format info for a given numId and level.
func (nr *NumberingResolver) ResolveLevel(numID string, level int) (listType ListType, bullet string, startAt int) {
	// Default to bullet list
	listType = ListTypeUnordered
	bullet = "•"
	startAt = 1

	if numID == "" {
		return
	}

	// Find the abstract numbering
	abstractID, ok := nr.numMappings[numID]
	if !ok {
		return
	}

	abstractNum, ok := nr.abstractNums[abstractID]
	if !ok {
		return
	}

	// Find the level definition
	levelStr := strconv.Itoa(level)
	for _, lvl := range abstractNum.Levels {
		if lvl.ILvl == levelStr {
			// Determine list type from numFmt
			switch lvl.NumFmt.Val {
			case "bullet":
				listType = ListTypeUnordered
				bullet = getBulletChar(lvl.LvlText.Val, level)
			case "decimal":
				listType = ListTypeOrdered
				bullet = "" // Will be computed as "1.", "2.", etc.
			case "lowerLetter":
				listType = ListTypeOrdered
				bullet = "" // a., b., c.
			case "upperLetter":
				listType = ListTypeOrdered
				bullet = "" // A., B., C.
			case "lowerRoman":
				listType = ListTypeOrdered
				bullet = "" // i., ii., iii.
			case "upperRoman":
				listType = ListTypeOrdered
				bullet = "" // I., II., III.
			default:
				// Default to bullet
				listType = ListTypeUnordered
				bullet = "•"
			}

			// Get start value
			if lvl.Start.Val != "" {
				if s, err := strconv.Atoi(lvl.Start.Val); err == nil {
					startAt = s
				}
			}

			return
		}
	}

	return
}

// IsListParagraph returns true if the paragraph has numbering properties.
func (nr *NumberingResolver) IsListParagraph(numID, ilvl string) bool {
	return numID != "" && numID != "0"
}

// getBulletChar returns the appropriate bullet character for the level.
func getBulletChar(lvlText string, level int) string {
	// Common Word bullet characters (standard Unicode)
	bullets := []string{"•", "○", "■", "□", "▪", "▫", "►", "◦"}

	// If lvlText specifies a character, check if it's usable
	if lvlText != "" && !strings.Contains(lvlText, "%") {
		// Check if the character is renderable (not in Private Use Area)
		// Word often uses Symbol/Wingdings fonts with PUA characters (U+F000-U+F0FF)
		if isRenderableBullet(lvlText) {
			return lvlText
		}
	}

	// Default based on level
	if level < len(bullets) {
		return bullets[level]
	}
	return "•"
}

// isRenderableBullet checks if a bullet character will render properly.
// Returns false for Private Use Area characters that require special fonts.
func isRenderableBullet(s string) bool {
	for _, r := range s {
		// Check for Private Use Area (U+E000-U+F8FF)
		// Word commonly uses U+F0xx for Symbol/Wingdings characters
		if r >= 0xE000 && r <= 0xF8FF {
			return false
		}
		// Also reject control characters
		if r < 0x20 {
			return false
		}
	}
	return len(s) > 0
}

// ListParser handles parsing of lists from document paragraphs.
type ListParser struct {
	resolver *NumberingResolver
}

// NewListParser creates a new list parser.
func NewListParser(resolver *NumberingResolver) *ListParser {
	return &ListParser{
		resolver: resolver,
	}
}

// ExtractLists extracts lists from a sequence of parsed paragraphs.
// It groups consecutive list items into lists.
func (lp *ListParser) ExtractLists(paragraphs []parsedParagraph) []ParsedList {
	var lists []ParsedList
	var currentList *ParsedList
	var lastNumID string
	var lastLevel int

	for _, para := range paragraphs {
		numID := para.NumID
		level := para.ListLevel

		if numID == "" || numID == "0" {
			// Not a list item - finalize current list if any
			if currentList != nil && len(currentList.Items) > 0 {
				lists = append(lists, *currentList)
				currentList = nil
			}
			lastNumID = ""
			continue
		}

		// This is a list item
		listType, bullet, startAt := lp.resolver.ResolveLevel(numID, level)

		// Check if we need to start a new list
		if currentList == nil || numID != lastNumID {
			// Finalize previous list
			if currentList != nil && len(currentList.Items) > 0 {
				lists = append(lists, *currentList)
			}
			// Start new list
			currentList = &ParsedList{
				Type:    listType,
				NumID:   numID,
				StartAt: startAt,
			}
		}

		// Add item to current list
		item := ParsedListItem{
			Text:   para.Text,
			Level:  level,
			Bullet: bullet,
			NumID:  numID,
		}
		currentList.Items = append(currentList.Items, item)

		lastNumID = numID
		lastLevel = level
		_ = lastLevel // Suppress unused warning for now
	}

	// Finalize last list
	if currentList != nil && len(currentList.Items) > 0 {
		lists = append(lists, *currentList)
	}

	return lists
}

// ToModelList converts a ParsedList to a model.List.
func (pl *ParsedList) ToModelList() *model.List {
	list := &model.List{
		Ordered: pl.Type == ListTypeOrdered,
		Items:   make([]model.ListItem, len(pl.Items)),
	}

	for i, item := range pl.Items {
		bullet := item.Bullet
		if list.Ordered && bullet == "" {
			// Generate number prefix
			bullet = strconv.Itoa(pl.StartAt+i) + "."
		}

		list.Items[i] = model.ListItem{
			Text:   item.Text,
			Level:  item.Level,
			Bullet: bullet,
		}
	}

	return list
}

// ToText returns a plain text representation of the list.
func (pl *ParsedList) ToText() string {
	var sb strings.Builder

	for i, item := range pl.Items {
		if i > 0 {
			sb.WriteString("\n")
		}

		// Add indentation
		for j := 0; j < item.Level; j++ {
			sb.WriteString("  ")
		}

		// Add bullet/number
		if pl.Type == ListTypeOrdered {
			sb.WriteString(strconv.Itoa(pl.StartAt+i) + ". ")
		} else {
			if item.Bullet != "" {
				sb.WriteString(item.Bullet + " ")
			} else {
				sb.WriteString("• ")
			}
		}

		sb.WriteString(item.Text)
	}

	return sb.String()
}
