package odt

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
	Items     []ParsedListItem
	Type      ListType
	StyleName string
	StartAt   int // Starting number for ordered lists
}

// ParsedListItem represents a single list item.
type ParsedListItem struct {
	Text   string
	Level  int    // Indentation level (0-based)
	Bullet string // The bullet character or number prefix
}

// ListParser handles parsing of ODT lists.
type ListParser struct {
	resolver *StyleResolver
}

// NewListParser creates a new list parser.
func NewListParser(resolver *StyleResolver) *ListParser {
	return &ListParser{
		resolver: resolver,
	}
}

// ParseList parses a list XML element into a ParsedList.
func (lp *ListParser) ParseList(list listXML, level int) ParsedList {
	parsed := ParsedList{
		StyleName: list.StyleName,
		StartAt:   1,
	}

	// Determine list type from style
	if lp.resolver != nil && list.StyleName != "" {
		ll := lp.resolver.ResolveListLevel(list.StyleName, level)
		if ll.IsBullet {
			parsed.Type = ListTypeUnordered
		} else {
			parsed.Type = ListTypeOrdered
			parsed.StartAt = ll.StartValue
		}
	}

	// Parse list items
	itemNum := parsed.StartAt
	for _, item := range list.Items {
		parsedItems := lp.parseListItem(item, level, list.StyleName, &itemNum)
		parsed.Items = append(parsed.Items, parsedItems...)
	}

	return parsed
}

// parseListItem parses a list item, including nested lists.
func (lp *ListParser) parseListItem(item listItemXML, level int, styleName string, itemNum *int) []ParsedListItem {
	var result []ParsedListItem

	// Get text from paragraphs
	var textParts []string
	for _, para := range item.Paragraphs {
		text := extractParagraphText(para)
		if text != "" {
			textParts = append(textParts, text)
		}
	}
	text := strings.Join(textParts, " ")

	// Get bullet/number for this item
	bullet := lp.getBullet(styleName, level, *itemNum)

	if text != "" {
		result = append(result, ParsedListItem{
			Text:   text,
			Level:  level,
			Bullet: bullet,
		})
		*itemNum++
	}

	// Process nested lists
	for _, subList := range item.SubLists {
		subItemNum := 1
		for _, subItem := range subList.Items {
			subItems := lp.parseListItem(subItem, level+1, subList.StyleName, &subItemNum)
			result = append(result, subItems...)
		}
	}

	return result
}

// getBullet returns the appropriate bullet or number for a list item.
func (lp *ListParser) getBullet(styleName string, level int, itemNum int) string {
	if lp.resolver == nil || styleName == "" {
		return getBulletChar(level)
	}

	ll := lp.resolver.ResolveListLevel(styleName, level)

	if ll.IsBullet {
		if ll.BulletChar != "" {
			return ll.BulletChar
		}
		return getBulletChar(level)
	}

	// Format number
	numStr := formatListNumber(itemNum, ll.NumFormat)
	return ll.NumPrefix + numStr + ll.NumSuffix
}

// extractParagraphText extracts text from a paragraph XML element.
func extractParagraphText(p paragraphXML) string {
	var parts []string

	// Direct text content
	if p.Text != "" {
		parts = append(parts, p.Text)
	}

	// Text from spans
	for _, span := range p.Spans {
		if span.Text != "" {
			parts = append(parts, span.Text)
		}
	}

	return strings.Join(parts, "")
}

// getBulletChar returns a bullet character based on nesting level.
func getBulletChar(level int) string {
	bullets := []string{"•", "○", "■", "□", "▪", "▫", "►", "◦"}
	if level < len(bullets) {
		return bullets[level]
	}
	return "•"
}

// formatListNumber formats a number according to the format type.
func formatListNumber(num int, format string) string {
	switch format {
	case "a":
		return toLowerLetter(num)
	case "A":
		return toUpperLetter(num)
	case "i":
		return toLowerRoman(num)
	case "I":
		return toUpperRoman(num)
	case "1", "":
		return strconv.Itoa(num)
	default:
		return strconv.Itoa(num)
	}
}

// toLowerLetter converts a number to lowercase letter (1=a, 2=b, etc.)
func toLowerLetter(n int) string {
	if n < 1 {
		return "a"
	}
	result := ""
	for n > 0 {
		n-- // Make it 0-indexed
		result = string(rune('a'+n%26)) + result
		n /= 26
	}
	return result
}

// toUpperLetter converts a number to uppercase letter (1=A, 2=B, etc.)
func toUpperLetter(n int) string {
	return strings.ToUpper(toLowerLetter(n))
}

// toLowerRoman converts a number to lowercase Roman numerals.
func toLowerRoman(n int) string {
	return strings.ToLower(toUpperRoman(n))
}

// toUpperRoman converts a number to uppercase Roman numerals.
func toUpperRoman(n int) string {
	if n < 1 || n > 3999 {
		return strconv.Itoa(n)
	}

	romanNumerals := []struct {
		value  int
		symbol string
	}{
		{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"},
		{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"},
		{10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
	}

	result := ""
	for _, rn := range romanNumerals {
		for n >= rn.value {
			result += rn.symbol
			n -= rn.value
		}
	}
	return result
}

// ToModelList converts a ParsedList to a model.List.
func (pl *ParsedList) ToModelList() *model.List {
	list := &model.List{
		Ordered: pl.Type == ListTypeOrdered,
		Items:   make([]model.ListItem, len(pl.Items)),
	}

	for i, item := range pl.Items {
		list.Items[i] = model.ListItem{
			Text:   item.Text,
			Level:  item.Level,
			Bullet: item.Bullet,
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
		sb.WriteString(item.Bullet)
		sb.WriteString(" ")
		sb.WriteString(item.Text)
	}

	return sb.String()
}
