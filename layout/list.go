// Package layout provides document layout analysis including list detection,
// which identifies and structures bulleted and numbered lists.
package layout

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// ListType represents the type of list
type ListType int

const (
	ListTypeUnknown  ListType = iota
	ListTypeBullet            // Bullet points (•, -, *, etc.)
	ListTypeNumbered          // Numbered (1., 2., 3.)
	ListTypeLettered          // Lettered (a., b., c. or A., B., C.)
	ListTypeRoman             // Roman numerals (i., ii., iii. or I., II., III.)
	ListTypeCheckbox          // Checkbox lists (☐, ☑, ✓)
)

// String returns a string representation of the list type
func (t ListType) String() string {
	switch t {
	case ListTypeBullet:
		return "bullet"
	case ListTypeNumbered:
		return "numbered"
	case ListTypeLettered:
		return "lettered"
	case ListTypeRoman:
		return "roman"
	case ListTypeCheckbox:
		return "checkbox"
	default:
		return "unknown"
	}
}

// BulletStyle represents the specific bullet character used
type BulletStyle int

const (
	BulletStyleUnknown     BulletStyle = iota
	BulletStyleDisc                    // • (filled circle)
	BulletStyleCircle                  // ○ (empty circle)
	BulletStyleSquare                  // ■ (filled square)
	BulletStyleDash                    // - (dash)
	BulletStyleAsterisk                // * (asterisk)
	BulletStyleArrow                   // → or ▶ (arrow)
	BulletStyleTriangle                // ▪ or ▸ (triangle)
	BulletStyleCheckEmpty              // ☐ (empty checkbox)
	BulletStyleCheckFilled             // ☑ or ✓ (checked)
)

// String returns a string representation of the bullet style
func (s BulletStyle) String() string {
	switch s {
	case BulletStyleDisc:
		return "disc"
	case BulletStyleCircle:
		return "circle"
	case BulletStyleSquare:
		return "square"
	case BulletStyleDash:
		return "dash"
	case BulletStyleAsterisk:
		return "asterisk"
	case BulletStyleArrow:
		return "arrow"
	case BulletStyleTriangle:
		return "triangle"
	case BulletStyleCheckEmpty:
		return "checkbox-empty"
	case BulletStyleCheckFilled:
		return "checkbox-filled"
	default:
		return "unknown"
	}
}

// ListItem represents a single item in a list
type ListItem struct {
	// Text is the item text (without the bullet/number prefix)
	Text string

	// RawText is the original text including prefix
	RawText string

	// Prefix is the bullet or number prefix (e.g., "•", "1.", "a)")
	Prefix string

	// BBox is the bounding box of this item
	BBox model.BBox

	// Lines are the lines that make up this item (may span multiple lines)
	Lines []Line

	// Index is the item's position within its parent list (0-based)
	Index int

	// Level is the nesting level (0 = top level, 1 = first nested, etc.)
	Level int

	// ListType is the type of list this item belongs to
	ListType ListType

	// BulletStyle is the bullet style (for bullet lists)
	BulletStyle BulletStyle

	// Number is the numeric value for numbered/lettered lists
	Number int

	// Children contains nested list items
	Children []ListItem
}

// List represents a complete list structure
type List struct {
	// Items are the list items (top level only; nested items are in Children)
	Items []ListItem

	// BBox is the bounding box of the entire list
	BBox model.BBox

	// Type is the primary list type
	Type ListType

	// BulletStyle is the bullet style (for bullet lists)
	BulletStyle BulletStyle

	// Index is the list's position in document order
	Index int

	// Level is the nesting level of this list
	Level int

	// IsMixed indicates if the list contains mixed types (bullet + numbered)
	IsMixed bool

	// ItemCount is the total number of items (including nested)
	ItemCount int
}

// ListLayout represents all detected lists on a page
type ListLayout struct {
	// Lists are all detected lists in document order
	Lists []List

	// AllItems are all list items in reading order (flattened)
	AllItems []ListItem

	// PageWidth and PageHeight
	PageWidth  float64
	PageHeight float64

	// Config is the configuration used for detection
	Config ListConfig
}

// ListConfig holds configuration for list detection
type ListConfig struct {
	// BulletCharacters are characters recognized as bullets
	BulletCharacters []rune

	// IndentThreshold is the minimum indentation increase to consider nested
	// Default: 15 points
	IndentThreshold float64

	// MaxListGap is the maximum vertical gap between items to consider same list
	// as a ratio of line height (default: 2.0)
	MaxListGap float64

	// NumberedPatterns are regex patterns for numbered list items
	NumberedPatterns []*regexp.Regexp

	// LetterPatterns are regex patterns for lettered list items
	LetterPatterns []*regexp.Regexp

	// RomanPatterns are regex patterns for roman numeral list items
	RomanPatterns []*regexp.Regexp

	// MinConsecutiveItems is minimum items to consider a list (default: 2)
	MinConsecutiveItems int
}

// DefaultListConfig returns sensible default configuration
func DefaultListConfig() ListConfig {
	return ListConfig{
		BulletCharacters: []rune{
			'•', '●', '○', '◦', '◉', // Circles
			'■', '□', '▪', '▫', // Squares
			'-', '–', '—', // Dashes
			'*', '✱', '✲', // Asterisks
			'→', '▶', '►', '▸', '➤', '➜', // Arrows
			'‣', '⁃', // Other bullets
			'☐', '☑', '✓', '✔', '✗', '✘', // Checkboxes
		},
		IndentThreshold:     15.0,
		MaxListGap:          2.0,
		MinConsecutiveItems: 2,
		NumberedPatterns: []*regexp.Regexp{
			regexp.MustCompile(`^(\d+)[.\)]\s*`),
			regexp.MustCompile(`^(\d+)\s*$`), // Just number at start
		},
		LetterPatterns: []*regexp.Regexp{
			regexp.MustCompile(`^([a-zA-Z])[.\)]\s*`),
		},
		RomanPatterns: []*regexp.Regexp{
			regexp.MustCompile(`^([ivxlcdmIVXLCDM]+)[.\)]\s*`),
		},
	}
}

// ListDetector detects and structures lists in document content
type ListDetector struct {
	config ListConfig
}

// NewListDetector creates a new list detector with default configuration
func NewListDetector() *ListDetector {
	return &ListDetector{
		config: DefaultListConfig(),
	}
}

// NewListDetectorWithConfig creates a list detector with custom configuration
func NewListDetectorWithConfig(config ListConfig) *ListDetector {
	return &ListDetector{
		config: config,
	}
}

// DetectFromParagraphs analyzes paragraphs and detects lists
func (d *ListDetector) DetectFromParagraphs(paragraphs []Paragraph, pageWidth, pageHeight float64) *ListLayout {
	if len(paragraphs) == 0 {
		return &ListLayout{
			PageWidth:  pageWidth,
			PageHeight: pageHeight,
			Config:     d.config,
		}
	}

	// Step 1: Identify potential list items
	candidates := d.identifyListCandidates(paragraphs)

	// Step 2: Group consecutive items into lists
	lists := d.groupIntoLists(candidates, paragraphs)

	// Step 3: Detect nesting based on indentation
	d.detectNesting(lists)

	// Step 4: Flatten all items for reading order
	allItems := d.flattenItems(lists)

	return &ListLayout{
		Lists:      lists,
		AllItems:   allItems,
		PageWidth:  pageWidth,
		PageHeight: pageHeight,
		Config:     d.config,
	}
}

// DetectFromLines analyzes lines directly and detects lists
func (d *ListDetector) DetectFromLines(lines []Line, pageWidth, pageHeight float64) *ListLayout {
	paraDetector := NewParagraphDetector()
	paraLayout := paraDetector.Detect(lines, pageWidth, pageHeight)
	return d.DetectFromParagraphs(paraLayout.Paragraphs, pageWidth, pageHeight)
}

// DetectFromFragments analyzes fragments directly and detects lists
func (d *ListDetector) DetectFromFragments(fragments []text.TextFragment, pageWidth, pageHeight float64) *ListLayout {
	lineDetector := NewLineDetector()
	lineLayout := lineDetector.Detect(fragments, pageWidth, pageHeight)
	return d.DetectFromLines(lineLayout.Lines, pageWidth, pageHeight)
}

// listCandidate holds info about a potential list item
type listCandidate struct {
	paragraphIndex int
	paragraph      Paragraph
	listType       ListType
	bulletStyle    BulletStyle
	prefix         string
	cleanText      string
	number         int
	indentation    float64
}

// identifyListCandidates finds paragraphs that look like list items
func (d *ListDetector) identifyListCandidates(paragraphs []Paragraph) []listCandidate {
	var candidates []listCandidate

	for i, para := range paragraphs {
		if candidate, ok := d.analyzeAsListItem(para, i); ok {
			candidates = append(candidates, candidate)
		}
	}

	return candidates
}

// analyzeAsListItem checks if a paragraph is a list item
func (d *ListDetector) analyzeAsListItem(para Paragraph, index int) (listCandidate, bool) {
	text := strings.TrimSpace(para.Text)
	if len(text) == 0 {
		return listCandidate{}, false
	}

	candidate := listCandidate{
		paragraphIndex: index,
		paragraph:      para,
		indentation:    para.LeftMargin,
	}

	// Check for bullet
	if listType, bulletStyle, prefix, cleanText := d.detectBullet(text); listType != ListTypeUnknown {
		candidate.listType = listType
		candidate.bulletStyle = bulletStyle
		candidate.prefix = prefix
		candidate.cleanText = cleanText
		return candidate, true
	}

	// Check for numbered
	if listType, prefix, cleanText, number := d.detectNumbered(text); listType != ListTypeUnknown {
		candidate.listType = listType
		candidate.prefix = prefix
		candidate.cleanText = cleanText
		candidate.number = number
		return candidate, true
	}

	return listCandidate{}, false
}

// detectBullet checks if text starts with a bullet character
func (d *ListDetector) detectBullet(text string) (ListType, BulletStyle, string, string) {
	if len(text) == 0 {
		return ListTypeUnknown, BulletStyleUnknown, "", ""
	}

	runes := []rune(text)
	firstRune := runes[0]

	// Check if first character is a bullet
	for _, bullet := range d.config.BulletCharacters {
		if firstRune == bullet {
			style := d.getBulletStyle(bullet)
			listType := ListTypeBullet
			if style == BulletStyleCheckEmpty || style == BulletStyleCheckFilled {
				listType = ListTypeCheckbox
			}

			// Get clean text (after bullet and whitespace)
			cleanText := strings.TrimSpace(string(runes[1:]))
			prefix := string(firstRune)

			return listType, style, prefix, cleanText
		}
	}

	return ListTypeUnknown, BulletStyleUnknown, "", ""
}

// getBulletStyle returns the bullet style for a character
func (d *ListDetector) getBulletStyle(r rune) BulletStyle {
	switch r {
	case '•', '●':
		return BulletStyleDisc
	case '○', '◦', '◉':
		return BulletStyleCircle
	case '■', '□', '▪', '▫':
		return BulletStyleSquare
	case '-', '–', '—':
		return BulletStyleDash
	case '*', '✱', '✲':
		return BulletStyleAsterisk
	case '→', '▶', '►', '▸', '➤', '➜':
		return BulletStyleArrow
	case '‣', '⁃':
		return BulletStyleTriangle
	case '☐':
		return BulletStyleCheckEmpty
	case '☑', '✓', '✔':
		return BulletStyleCheckFilled
	default:
		return BulletStyleUnknown
	}
}

// detectNumbered checks if text starts with a number pattern
func (d *ListDetector) detectNumbered(text string) (ListType, string, string, int) {
	// Check numbered patterns (1., 2., etc.)
	for _, pattern := range d.config.NumberedPatterns {
		if match := pattern.FindStringSubmatch(text); len(match) > 1 {
			prefix := match[0]
			cleanText := strings.TrimSpace(text[len(prefix):])
			number := d.parseNumber(match[1])
			return ListTypeNumbered, strings.TrimSpace(prefix), cleanText, number
		}
	}

	// Check lettered patterns (a., b., etc.)
	for _, pattern := range d.config.LetterPatterns {
		if match := pattern.FindStringSubmatch(text); len(match) > 1 {
			prefix := match[0]
			cleanText := strings.TrimSpace(text[len(prefix):])
			number := d.letterToNumber(match[1])
			return ListTypeLettered, strings.TrimSpace(prefix), cleanText, number
		}
	}

	// Check roman numeral patterns
	for _, pattern := range d.config.RomanPatterns {
		if match := pattern.FindStringSubmatch(text); len(match) > 1 {
			// Validate it's actually a roman numeral
			if d.isValidRoman(match[1]) {
				prefix := match[0]
				cleanText := strings.TrimSpace(text[len(prefix):])
				number := d.romanToNumber(match[1])
				return ListTypeRoman, strings.TrimSpace(prefix), cleanText, number
			}
		}
	}

	return ListTypeUnknown, "", "", 0
}

// parseNumber converts a string number to int
func (d *ListDetector) parseNumber(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	return n
}

// letterToNumber converts a letter to its numeric position (a=1, b=2, etc.)
func (d *ListDetector) letterToNumber(s string) int {
	if len(s) == 0 {
		return 0
	}
	r := rune(strings.ToLower(s)[0])
	if r >= 'a' && r <= 'z' {
		return int(r - 'a' + 1)
	}
	return 0
}

// isValidRoman checks if a string is a valid roman numeral
func (d *ListDetector) isValidRoman(s string) bool {
	s = strings.ToUpper(s)
	validChars := "IVXLCDM"
	for _, r := range s {
		if !strings.ContainsRune(validChars, r) {
			return false
		}
	}
	// Must be at least one character and not too long
	return len(s) >= 1 && len(s) <= 15
}

// romanToNumber converts a roman numeral to integer
func (d *ListDetector) romanToNumber(s string) int {
	s = strings.ToUpper(s)
	values := map[rune]int{
		'I': 1, 'V': 5, 'X': 10, 'L': 50,
		'C': 100, 'D': 500, 'M': 1000,
	}

	result := 0
	prev := 0
	for i := len(s) - 1; i >= 0; i-- {
		val := values[rune(s[i])]
		if val < prev {
			result -= val
		} else {
			result += val
		}
		prev = val
	}
	return result
}

// groupIntoLists groups consecutive list candidates into lists
func (d *ListDetector) groupIntoLists(candidates []listCandidate, paragraphs []Paragraph) []List {
	if len(candidates) == 0 {
		return nil
	}

	var lists []List
	var currentItems []ListItem
	var currentType ListType
	lastIndex := -2

	for _, candidate := range candidates {
		// Check if this continues the current list
		isConsecutive := candidate.paragraphIndex == lastIndex+1
		sameType := candidate.listType == currentType || currentType == ListTypeUnknown

		// Check vertical gap if not directly consecutive
		if !isConsecutive && lastIndex >= 0 && candidate.paragraphIndex < len(paragraphs) {
			prevPara := paragraphs[lastIndex]
			currPara := candidate.paragraph
			gap := d.calculateGap(prevPara, currPara)
			avgHeight := (prevPara.AverageFontSize + currPara.AverageFontSize) / 2
			if gap <= avgHeight*d.config.MaxListGap {
				isConsecutive = true
			}
		}

		if len(currentItems) > 0 && (!isConsecutive || !sameType) {
			// Finish current list
			if len(currentItems) >= d.config.MinConsecutiveItems {
				list := d.createList(currentItems, len(lists))
				lists = append(lists, list)
			}
			currentItems = nil
			currentType = ListTypeUnknown
		}

		// Add item to current list
		item := d.createListItem(candidate, len(currentItems))
		currentItems = append(currentItems, item)
		currentType = candidate.listType
		lastIndex = candidate.paragraphIndex
	}

	// Don't forget the last list
	if len(currentItems) >= d.config.MinConsecutiveItems {
		list := d.createList(currentItems, len(lists))
		lists = append(lists, list)
	}

	return lists
}

// calculateGap calculates the vertical gap between two paragraphs
func (d *ListDetector) calculateGap(p1, p2 Paragraph) float64 {
	// Assuming standard PDF coordinates (higher Y = higher on page)
	gap := p1.BBox.Y - (p2.BBox.Y + p2.BBox.Height)
	if gap < 0 {
		gap = -gap
	}
	return gap
}

// createListItem creates a ListItem from a candidate
func (d *ListDetector) createListItem(candidate listCandidate, index int) ListItem {
	return ListItem{
		Text:        candidate.cleanText,
		RawText:     candidate.paragraph.Text,
		Prefix:      candidate.prefix,
		BBox:        candidate.paragraph.BBox,
		Lines:       candidate.paragraph.Lines,
		Index:       index,
		Level:       0, // Will be set during nesting detection
		ListType:    candidate.listType,
		BulletStyle: candidate.bulletStyle,
		Number:      candidate.number,
	}
}

// createList creates a List from items
func (d *ListDetector) createList(items []ListItem, index int) List {
	list := List{
		Items: items,
		Index: index,
		Level: 0,
	}

	// Determine list type (use first item's type)
	if len(items) > 0 {
		list.Type = items[0].ListType
		list.BulletStyle = items[0].BulletStyle
	}

	// Check for mixed types
	for _, item := range items {
		if item.ListType != list.Type {
			list.IsMixed = true
			break
		}
	}

	// Calculate bounding box
	list.BBox = d.calculateListBBox(items)

	// Count total items
	list.ItemCount = len(items)

	return list
}

// calculateListBBox calculates the bounding box for a list
func (d *ListDetector) calculateListBBox(items []ListItem) model.BBox {
	if len(items) == 0 {
		return model.BBox{}
	}

	bbox := items[0].BBox
	for _, item := range items[1:] {
		if item.BBox.X < bbox.X {
			bbox.Width += bbox.X - item.BBox.X
			bbox.X = item.BBox.X
		}
		if item.BBox.X+item.BBox.Width > bbox.X+bbox.Width {
			bbox.Width = item.BBox.X + item.BBox.Width - bbox.X
		}
		if item.BBox.Y < bbox.Y {
			bbox.Height += bbox.Y - item.BBox.Y
			bbox.Y = item.BBox.Y
		}
		if item.BBox.Y+item.BBox.Height > bbox.Y+bbox.Height {
			bbox.Height = item.BBox.Y + item.BBox.Height - bbox.Y
		}
	}

	return bbox
}

// detectNesting analyzes indentation to detect nested lists
func (d *ListDetector) detectNesting(lists []List) {
	for i := range lists {
		d.detectListNesting(&lists[i])
	}
}

// detectListNesting detects nesting within a single list
func (d *ListDetector) detectListNesting(list *List) {
	if len(list.Items) == 0 {
		return
	}

	// Find base indentation (minimum)
	baseIndent := list.Items[0].BBox.X
	for _, item := range list.Items {
		if item.BBox.X < baseIndent {
			baseIndent = item.BBox.X
		}
	}

	// Assign levels based on indentation
	for i := range list.Items {
		indent := list.Items[i].BBox.X - baseIndent
		level := int(indent / d.config.IndentThreshold)
		list.Items[i].Level = level
	}

	// Build hierarchy (move nested items to parent's Children)
	list.Items = d.buildHierarchy(list.Items)

	// Recount items including nested
	list.ItemCount = d.countAllItems(list.Items)
}

// buildHierarchy converts flat items with levels into nested structure
func (d *ListDetector) buildHierarchy(items []ListItem) []ListItem {
	if len(items) == 0 {
		return nil
	}

	var result []ListItem
	var stack []*ListItem

	for i := range items {
		item := items[i]

		// Pop items from stack until we find a parent at lower level
		for len(stack) > 0 && stack[len(stack)-1].Level >= item.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			// Top-level item
			result = append(result, item)
			stack = append(stack, &result[len(result)-1])
		} else {
			// Nested item - add to parent's children
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, item)
			stack = append(stack, &parent.Children[len(parent.Children)-1])
		}
	}

	return result
}

// countAllItems counts total items including nested
func (d *ListDetector) countAllItems(items []ListItem) int {
	count := len(items)
	for _, item := range items {
		count += d.countAllItems(item.Children)
	}
	return count
}

// flattenItems returns all items in reading order
func (d *ListDetector) flattenItems(lists []List) []ListItem {
	var result []ListItem
	for _, list := range lists {
		result = append(result, d.flattenListItems(list.Items)...)
	}
	return result
}

// flattenListItems flattens nested items
func (d *ListDetector) flattenListItems(items []ListItem) []ListItem {
	var result []ListItem
	for _, item := range items {
		result = append(result, item)
		if len(item.Children) > 0 {
			result = append(result, d.flattenListItems(item.Children)...)
		}
	}
	return result
}

// ListLayout methods

// ListCount returns the number of detected lists
func (l *ListLayout) ListCount() int {
	if l == nil {
		return 0
	}
	return len(l.Lists)
}

// GetList returns a specific list by index
func (l *ListLayout) GetList(index int) *List {
	if l == nil || index < 0 || index >= len(l.Lists) {
		return nil
	}
	return &l.Lists[index]
}

// GetListsByType returns lists of a specific type
func (l *ListLayout) GetListsByType(listType ListType) []List {
	if l == nil {
		return nil
	}

	var result []List
	for _, list := range l.Lists {
		if list.Type == listType {
			result = append(result, list)
		}
	}
	return result
}

// GetBulletLists returns all bullet lists
func (l *ListLayout) GetBulletLists() []List {
	return l.GetListsByType(ListTypeBullet)
}

// GetNumberedLists returns all numbered lists
func (l *ListLayout) GetNumberedLists() []List {
	return l.GetListsByType(ListTypeNumbered)
}

// TotalItemCount returns the total number of list items
func (l *ListLayout) TotalItemCount() int {
	if l == nil {
		return 0
	}
	return len(l.AllItems)
}

// FindListsInRegion returns lists within a bounding box
func (l *ListLayout) FindListsInRegion(bbox model.BBox) []List {
	if l == nil {
		return nil
	}

	var result []List
	for _, list := range l.Lists {
		if list.BBox.X+list.BBox.Width > bbox.X &&
			list.BBox.X < bbox.X+bbox.Width &&
			list.BBox.Y+list.BBox.Height > bbox.Y &&
			list.BBox.Y < bbox.Y+bbox.Height {
			result = append(result, list)
		}
	}
	return result
}

// List methods

// GetAllItems returns all items including nested (flattened)
func (list *List) GetAllItems() []ListItem {
	if list == nil {
		return nil
	}

	var result []ListItem
	var flatten func(items []ListItem)
	flatten = func(items []ListItem) {
		for _, item := range items {
			result = append(result, item)
			flatten(item.Children)
		}
	}
	flatten(list.Items)
	return result
}

// GetText returns all list text as a formatted string
func (list *List) GetText() string {
	if list == nil {
		return ""
	}

	var sb strings.Builder
	var writeItems func(items []ListItem, indent string)
	writeItems = func(items []ListItem, indent string) {
		for _, item := range items {
			sb.WriteString(indent)
			sb.WriteString(item.Prefix)
			sb.WriteString(" ")
			sb.WriteString(item.Text)
			sb.WriteString("\n")
			if len(item.Children) > 0 {
				writeItems(item.Children, indent+"  ")
			}
		}
	}
	writeItems(list.Items, "")
	return sb.String()
}

// ToMarkdown returns the list as markdown
func (list *List) ToMarkdown() string {
	if list == nil {
		return ""
	}

	var sb strings.Builder
	var writeItems func(items []ListItem, indent string, startNum int)
	writeItems = func(items []ListItem, indent string, startNum int) {
		for i, item := range items {
			sb.WriteString(indent)
			switch list.Type {
			case ListTypeNumbered:
				sb.WriteString(strings.Repeat(" ", len(indent)))
				num := startNum + i
				sb.WriteString(string(rune('0'+num%10)) + ". ")
			case ListTypeLettered:
				sb.WriteString(string(rune('a'+i%26)) + ". ")
			default:
				sb.WriteString("- ")
			}
			sb.WriteString(item.Text)
			sb.WriteString("\n")
			if len(item.Children) > 0 {
				writeItems(item.Children, indent+"  ", 1)
			}
		}
	}
	writeItems(list.Items, "", 1)
	return sb.String()
}

// HasNesting returns true if the list contains nested items
func (list *List) HasNesting() bool {
	if list == nil {
		return false
	}
	for _, item := range list.Items {
		if len(item.Children) > 0 {
			return true
		}
	}
	return false
}

// MaxDepth returns the maximum nesting depth
func (list *List) MaxDepth() int {
	if list == nil {
		return 0
	}

	var maxDepth func(items []ListItem, depth int) int
	maxDepth = func(items []ListItem, depth int) int {
		max := depth
		for _, item := range items {
			if len(item.Children) > 0 {
				childMax := maxDepth(item.Children, depth+1)
				if childMax > max {
					max = childMax
				}
			}
		}
		return max
	}
	return maxDepth(list.Items, 0)
}

// ListItem methods

// HasChildren returns true if this item has nested items
func (item *ListItem) HasChildren() bool {
	if item == nil {
		return false
	}
	return len(item.Children) > 0
}

// ChildCount returns the number of direct children
func (item *ListItem) ChildCount() int {
	if item == nil {
		return 0
	}
	return len(item.Children)
}

// GetFullText returns the raw text including prefix
func (item *ListItem) GetFullText() string {
	if item == nil {
		return ""
	}
	return item.RawText
}

// IsCheckbox returns true if this is a checkbox item
func (item *ListItem) IsCheckbox() bool {
	if item == nil {
		return false
	}
	return item.ListType == ListTypeCheckbox
}

// IsChecked returns true if this is a checked checkbox
func (item *ListItem) IsChecked() bool {
	if item == nil {
		return false
	}
	return item.BulletStyle == BulletStyleCheckFilled
}

// WordCount returns the word count of the item text
func (item *ListItem) WordCount() int {
	if item == nil {
		return 0
	}
	return len(strings.Fields(item.Text))
}

// ContainsPoint returns true if the point is within the item's bounding box
func (item *ListItem) ContainsPoint(x, y float64) bool {
	if item == nil {
		return false
	}
	return x >= item.BBox.X && x <= item.BBox.X+item.BBox.Width &&
		y >= item.BBox.Y && y <= item.BBox.Y+item.BBox.Height
}

// IsFirstInList returns true if this item has number/index 1 or 0
func (item *ListItem) IsFirstInList() bool {
	if item == nil {
		return false
	}
	return item.Index == 0 || item.Number == 1
}

// Helper to check if a rune is a list bullet
func isBulletRune(r rune) bool {
	bullets := []rune{
		'•', '●', '○', '◦', '◉',
		'■', '□', '▪', '▫',
		'‣', '⁃',
		'→', '▶', '►', '▸', '➤', '➜',
		'☐', '☑', '✓', '✔', '✗', '✘',
	}
	for _, b := range bullets {
		if r == b {
			return true
		}
	}
	// Also check common ASCII bullets
	return r == '-' || r == '*' || r == '+'
}

// IsListItemText checks if text appears to be a list item
func IsListItemText(text string) bool {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return false
	}

	// Check for bullet character
	firstRune := []rune(text)[0]
	if isBulletRune(firstRune) {
		return true
	}

	// Check for numbered pattern
	if len(text) >= 2 {
		// 1. or 1) pattern
		if unicode.IsDigit(firstRune) {
			for i := 1; i < len(text) && i < 4; i++ {
				if text[i] == '.' || text[i] == ')' {
					return true
				}
				if !unicode.IsDigit(rune(text[i])) {
					break
				}
			}
		}
		// a. or a) pattern
		if unicode.IsLetter(firstRune) && len(text) >= 2 {
			if text[1] == '.' || text[1] == ')' {
				return true
			}
		}
	}

	return false
}
