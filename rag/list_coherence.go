package rag

import (
	"regexp"
	"strings"
	"unicode"
)

// ListType represents the type of list
type ListType int

const (
	// ListTypeUnordered is a bullet list
	ListTypeUnordered ListType = iota
	// ListTypeOrdered is a numbered list
	ListTypeOrdered
	// ListTypeDefinition is a definition list (term: definition)
	ListTypeDefinition
	// ListTypeChecklist is a checkbox list
	ListTypeChecklist
)

// String returns a human-readable representation of the list type
func (lt ListType) String() string {
	switch lt {
	case ListTypeUnordered:
		return "unordered"
	case ListTypeOrdered:
		return "ordered"
	case ListTypeDefinition:
		return "definition"
	case ListTypeChecklist:
		return "checklist"
	default:
		return "unknown"
	}
}

// ListItem represents a single item in a list
type ListItem struct {
	// Text is the item content
	Text string

	// Marker is the bullet/number (e.g., "•", "1.", "a)")
	Marker string

	// Level is the nesting level (0 = top level)
	Level int

	// Index is the position in the list
	Index int

	// Children are nested list items
	Children []*ListItem

	// IsComplete indicates if the item text is complete
	IsComplete bool
}

// ListBlock represents a complete list with its context
type ListBlock struct {
	// Type is the kind of list
	Type ListType

	// IntroText is the introductory paragraph (if any)
	IntroText string

	// HasIntro indicates if there's an introductory paragraph
	HasIntro bool

	// Items are the list items
	Items []*ListItem

	// MaxLevel is the deepest nesting level
	MaxLevel int

	// TotalItems is the total count including nested items
	TotalItems int

	// IsComplete indicates if the list is complete
	IsComplete bool
}

// ListCoherenceConfig holds configuration for list coherence
type ListCoherenceConfig struct {
	// KeepIntroWithList keeps introductory text with the list
	KeepIntroWithList bool

	// MaxIntroDistance is max chars between intro and list
	MaxIntroDistance int

	// PreserveNesting keeps nested lists together
	PreserveNesting bool

	// MaxListSize is max chars for a list before considering split
	MaxListSize int

	// MinItemsBeforeSplit is minimum items to have before splitting
	MinItemsBeforeSplit int

	// AllowSplitAtLevel allows splitting only at this nesting level or higher
	AllowSplitAtLevel int

	// IntroPatterns are patterns that detect list introductions
	IntroPatterns []*regexp.Regexp
}

// DefaultListCoherenceConfig returns sensible defaults
func DefaultListCoherenceConfig() ListCoherenceConfig {
	return ListCoherenceConfig{
		KeepIntroWithList:   true,
		MaxIntroDistance:    200,
		PreserveNesting:     true,
		MaxListSize:         3000,
		MinItemsBeforeSplit: 3,
		AllowSplitAtLevel:   0, // Only split at top level
		IntroPatterns: []*regexp.Regexp{
			// "The following X:" - requires colon at end or just the phrase
			regexp.MustCompile(`(?i)\b(the\s+following|here\s+are|these\s+(are|include)|below\s+(are|is)|as\s+follows)\s*:\s*$`),
			// "Steps/Features/etc:"
			regexp.MustCompile(`(?i)\b(steps?|features?|items?|points?|reasons?|benefits?|advantages?|disadvantages?|options?|examples?|requirements?|prerequisites?|instructions?|guidelines?|rules?|conditions?|criteria|objectives?|goals?|tasks?|actions?|methods?|approaches?|techniques?|strategies?|tips?|recommendations?|suggestions?|notes?|warnings?|cautions?|considerations?)\s*:?\s*$`),
			// "Include(s)/Such as:"
			regexp.MustCompile(`(?i)\b(include|includes|including|consist\s+of|consists\s+of|comprised\s+of|such\s+as|for\s+example|for\s+instance|e\.g\.|i\.e\.)\s*:?\s*$`),
			// "You can/should/must:"
			regexp.MustCompile(`(?i)\b(you\s+(can|should|must|need\s+to|will)|we\s+(can|should|must|will)|to\s+do\s+this)\s*:?\s*$`),
			// Ends with colon after noun phrase
			regexp.MustCompile(`(?i)\b\w+\s*:\s*$`),
		},
	}
}

// ListCoherenceAnalyzer analyzes and manages list coherence
type ListCoherenceAnalyzer struct {
	config ListCoherenceConfig
}

// NewListCoherenceAnalyzer creates a new analyzer with default config
func NewListCoherenceAnalyzer() *ListCoherenceAnalyzer {
	return &ListCoherenceAnalyzer{
		config: DefaultListCoherenceConfig(),
	}
}

// NewListCoherenceAnalyzerWithConfig creates an analyzer with custom config
func NewListCoherenceAnalyzerWithConfig(config ListCoherenceConfig) *ListCoherenceAnalyzer {
	return &ListCoherenceAnalyzer{
		config: config,
	}
}

// IsListIntro checks if text appears to introduce a list
func (a *ListCoherenceAnalyzer) IsListIntro(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}

	for _, pattern := range a.config.IntroPatterns {
		if pattern.MatchString(text) {
			return true
		}
	}

	return false
}

// DetectListType identifies the type of list from its content
func (a *ListCoherenceAnalyzer) DetectListType(text string) ListType {
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for ordered list markers
		if isOrderedListItem(line) {
			return ListTypeOrdered
		}

		// Check for checklist markers
		if isChecklistItem(line) {
			return ListTypeChecklist
		}

		// Check for definition list markers
		if isDefinitionItem(line) {
			return ListTypeDefinition
		}

		// Check for unordered list markers
		if isUnorderedListItem(line) {
			return ListTypeUnordered
		}
	}

	return ListTypeUnordered // Default
}

// ParseListItems extracts structured list items from text
func (a *ListCoherenceAnalyzer) ParseListItems(text string) []*ListItem {
	var items []*ListItem
	lines := strings.Split(text, "\n")
	var currentItem *ListItem
	itemIndex := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Determine nesting level from leading whitespace
		level := getIndentLevel(line)

		// Extract marker and content
		marker, content := extractListMarker(trimmed)

		if marker != "" {
			// This is a list item
			item := &ListItem{
				Text:       content,
				Marker:     marker,
				Level:      level,
				Index:      itemIndex,
				IsComplete: true,
			}

			if level == 0 {
				// Top-level item
				items = append(items, item)
				currentItem = item
			} else if currentItem != nil {
				// Nested item - add to appropriate parent
				addNestedItem(items, item, level)
			}
			itemIndex++
		} else if currentItem != nil {
			// Continuation of previous item
			currentItem.Text += " " + trimmed
		}
	}

	return items
}

// addNestedItem adds a nested item to the correct parent
func addNestedItem(items []*ListItem, item *ListItem, targetLevel int) {
	if len(items) == 0 {
		return
	}

	// Find the most recent item at level-1
	parent := findParentAtLevel(items, targetLevel-1)
	if parent != nil {
		parent.Children = append(parent.Children, item)
	} else if len(items) > 0 {
		// Add to last top-level item if no proper parent found
		items[len(items)-1].Children = append(items[len(items)-1].Children, item)
	}
}

// findParentAtLevel finds the most recent item at the specified level
func findParentAtLevel(items []*ListItem, level int) *ListItem {
	if level < 0 || len(items) == 0 {
		return nil
	}

	// Search from end to find most recent
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item.Level == level {
			return item
		}
		// Check children recursively
		if found := findParentInChildren(item.Children, level); found != nil {
			return found
		}
	}

	return nil
}

// findParentInChildren searches children for an item at the specified level
func findParentInChildren(children []*ListItem, level int) *ListItem {
	if len(children) == 0 {
		return nil
	}

	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]
		if child.Level == level {
			return child
		}
		if found := findParentInChildren(child.Children, level); found != nil {
			return found
		}
	}

	return nil
}

// AnalyzeListBlock creates a complete ListBlock from text
func (a *ListCoherenceAnalyzer) AnalyzeListBlock(listText string, precedingText string) *ListBlock {
	block := &ListBlock{
		Type:       a.DetectListType(listText),
		Items:      a.ParseListItems(listText),
		IsComplete: true,
	}

	// Check for intro
	if precedingText != "" && a.IsListIntro(precedingText) {
		block.IntroText = strings.TrimSpace(precedingText)
		block.HasIntro = true
	}

	// Calculate stats
	block.TotalItems, block.MaxLevel = countItemsAndDepth(block.Items)

	return block
}

// countItemsAndDepth counts total items and maximum nesting depth
func countItemsAndDepth(items []*ListItem) (int, int) {
	count := 0
	maxDepth := 0

	for _, item := range items {
		count++
		if item.Level > maxDepth {
			maxDepth = item.Level
		}
		childCount, childDepth := countItemsAndDepth(item.Children)
		count += childCount
		if childDepth > maxDepth {
			maxDepth = childDepth
		}
	}

	return count, maxDepth
}

// FindListSplitPoints finds safe points to split a large list
func (a *ListCoherenceAnalyzer) FindListSplitPoints(block *ListBlock) []int {
	var splitPoints []int

	if len(block.Items) < a.config.MinItemsBeforeSplit {
		return splitPoints
	}

	// Only split at top-level item boundaries
	for i := a.config.MinItemsBeforeSplit; i < len(block.Items); i++ {
		item := block.Items[i]
		// Don't split if this item has nested children
		if a.config.PreserveNesting && len(item.Children) > 0 {
			continue
		}
		// Only split at allowed nesting levels
		if item.Level <= a.config.AllowSplitAtLevel {
			splitPoints = append(splitPoints, i)
		}
	}

	return splitPoints
}

// SplitListBlock splits a list at the specified item index
func (a *ListCoherenceAnalyzer) SplitListBlock(block *ListBlock, atIndex int) (*ListBlock, *ListBlock) {
	if atIndex <= 0 || atIndex >= len(block.Items) {
		return block, nil
	}

	first := &ListBlock{
		Type:       block.Type,
		IntroText:  block.IntroText,
		HasIntro:   block.HasIntro,
		Items:      block.Items[:atIndex],
		IsComplete: false,
	}
	first.TotalItems, first.MaxLevel = countItemsAndDepth(first.Items)

	second := &ListBlock{
		Type:       block.Type,
		Items:      block.Items[atIndex:],
		IsComplete: block.IsComplete,
	}
	second.TotalItems, second.MaxLevel = countItemsAndDepth(second.Items)

	return first, second
}

// FormatListBlock formats a list block back to text
func (a *ListCoherenceAnalyzer) FormatListBlock(block *ListBlock, preserveMarkers bool) string {
	var sb strings.Builder

	if block.HasIntro && block.IntroText != "" {
		sb.WriteString(block.IntroText)
		sb.WriteString("\n\n")
	}

	for _, item := range block.Items {
		formatListItem(&sb, item, preserveMarkers, "")
	}

	return strings.TrimSpace(sb.String())
}

// formatListItem recursively formats a list item
func formatListItem(sb *strings.Builder, item *ListItem, preserveMarkers bool, indent string) {
	if preserveMarkers && item.Marker != "" {
		sb.WriteString(indent)
		sb.WriteString(item.Marker)
		sb.WriteString(" ")
		sb.WriteString(item.Text)
	} else {
		sb.WriteString(indent)
		sb.WriteString(item.Text)
	}
	sb.WriteString("\n")

	// Format children
	for _, child := range item.Children {
		formatListItem(sb, child, preserveMarkers, indent+"  ")
	}
}

// ShouldKeepListTogether determines if a list should be kept as one chunk
func (a *ListCoherenceAnalyzer) ShouldKeepListTogether(block *ListBlock) bool {
	// Calculate total size
	totalSize := 0
	if block.HasIntro {
		totalSize += len(block.IntroText)
	}

	for _, item := range block.Items {
		totalSize += countListItemSize(item)
	}

	// Keep together if under max size
	if totalSize <= a.config.MaxListSize {
		return true
	}

	// Keep together if too few items to split
	if block.TotalItems < a.config.MinItemsBeforeSplit {
		return true
	}

	// Keep together if has nested content and preserving nesting
	if a.config.PreserveNesting && block.MaxLevel > 0 {
		// Check if any safe split points exist
		splitPoints := a.FindListSplitPoints(block)
		if len(splitPoints) == 0 {
			return true
		}
	}

	return false
}

// countListItemSize calculates the character size of an item and its children
func countListItemSize(item *ListItem) int {
	size := len(item.Text) + len(item.Marker) + 2 // +2 for space and newline
	for _, child := range item.Children {
		size += countListItemSize(child)
	}
	return size
}

// ListCoherenceResult holds the result of list coherence analysis
type ListCoherenceResult struct {
	// Blocks are the identified list blocks
	Blocks []*ListBlock

	// IntroOrphans are introductions without following lists
	IntroOrphans []string

	// TotalLists is the number of lists found
	TotalLists int

	// ListsWithIntros is the number of lists with introductions
	ListsWithIntros int

	// NestedLists is the number of lists with nesting
	NestedLists int
}

// AnalyzeListCoherence analyzes list coherence in a sequence of text blocks
func (a *ListCoherenceAnalyzer) AnalyzeListCoherence(blocks []ContentBlock) *ListCoherenceResult {
	result := &ListCoherenceResult{}

	for i := 0; i < len(blocks); i++ {
		block := blocks[i]

		// Check if this is a list
		if block.Type == 5 { // model.ElementTypeList
			var introText string

			// Check for preceding intro
			if i > 0 && blocks[i-1].Type == 1 { // model.ElementTypeParagraph
				if a.IsListIntro(blocks[i-1].Text) {
					introText = blocks[i-1].Text
				}
			}

			listBlock := a.AnalyzeListBlock(block.Text, introText)
			result.Blocks = append(result.Blocks, listBlock)
			result.TotalLists++

			if listBlock.HasIntro {
				result.ListsWithIntros++
			}
			if listBlock.MaxLevel > 0 {
				result.NestedLists++
			}
		} else if block.Type == 1 { // model.ElementTypeParagraph
			// Check for orphaned intro
			if a.IsListIntro(block.Text) {
				// Check if next block is NOT a list
				if i+1 >= len(blocks) || blocks[i+1].Type != 5 {
					result.IntroOrphans = append(result.IntroOrphans, block.Text)
				}
			}
		}
	}

	return result
}

// Helper functions for list detection

// isOrderedListItem checks if a line starts with an ordered list marker
func isOrderedListItem(line string) bool {
	// Patterns: "1.", "1)", "a.", "a)", "i.", "i)", "(1)", "(a)"
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^\d+[.\)]\s`),        // 1. or 1)
		regexp.MustCompile(`^[a-z][.\)]\s`),      // a. or a)
		regexp.MustCompile(`^[ivxlcdm]+[.\)]\s`), // Roman numerals
		regexp.MustCompile(`^\(\d+\)\s`),         // (1)
		regexp.MustCompile(`^\([a-z]\)\s`),       // (a)
	}

	for _, p := range patterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

// isUnorderedListItem checks if a line starts with an unordered list marker
func isUnorderedListItem(line string) bool {
	markers := []string{"•", "●", "○", "■", "□", "▪", "▫", "-", "*", "–", "—", "·"}
	for _, m := range markers {
		if strings.HasPrefix(line, m+" ") || strings.HasPrefix(line, m+"\t") {
			return true
		}
	}
	return false
}

// isChecklistItem checks if a line starts with a checkbox marker
func isChecklistItem(line string) bool {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^\[[ xX✓✗]\]\s`), // [x] or [ ]
		regexp.MustCompile(`^☐\s`),           // Empty checkbox
		regexp.MustCompile(`^☑\s`),           // Checked checkbox
		regexp.MustCompile(`^☒\s`),           // X-ed checkbox
	}

	for _, p := range patterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

// isDefinitionItem checks if a line is a definition list item
func isDefinitionItem(line string) bool {
	// Pattern: "Term: Definition" or "Term - Definition"
	if strings.Contains(line, ": ") {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			term := strings.TrimSpace(parts[0])
			// Term should be relatively short
			if len(term) > 0 && len(term) < 50 && !strings.Contains(term, ".") {
				return true
			}
		}
	}
	return false
}

// getIndentLevel returns the nesting level based on leading whitespace
func getIndentLevel(line string) int {
	spaces := 0
	for _, r := range line {
		if r == ' ' {
			spaces++
		} else if r == '\t' {
			spaces += 4 // Treat tab as 4 spaces
		} else {
			break
		}
	}
	return spaces / 2 // 2 spaces = 1 level
}

// extractListMarker extracts the marker and content from a list item line
func extractListMarker(line string) (string, string) {
	// Try ordered patterns first
	orderedPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^(\d+[.\)])\s+(.*)$`),
		regexp.MustCompile(`^([a-z][.\)])\s+(.*)$`),
		regexp.MustCompile(`^([ivxlcdm]+[.\)])\s+(.*)$`),
		regexp.MustCompile(`^(\(\d+\))\s+(.*)$`),
		regexp.MustCompile(`^(\([a-z]\))\s+(.*)$`),
	}

	for _, p := range orderedPatterns {
		if matches := p.FindStringSubmatch(line); len(matches) == 3 {
			return matches[1], matches[2]
		}
	}

	// Try unordered markers
	unorderedMarkers := []string{"•", "●", "○", "■", "□", "▪", "▫", "-", "*", "–", "—", "·"}
	for _, m := range unorderedMarkers {
		if strings.HasPrefix(line, m+" ") {
			return m, strings.TrimPrefix(line, m+" ")
		}
		if strings.HasPrefix(line, m+"\t") {
			return m, strings.TrimPrefix(line, m+"\t")
		}
	}

	// Try checkbox patterns
	checkboxPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^(\[[ xX✓✗]\])\s+(.*)$`),
		regexp.MustCompile(`^(☐|☑|☒)\s+(.*)$`),
	}

	for _, p := range checkboxPatterns {
		if matches := p.FindStringSubmatch(line); len(matches) == 3 {
			return matches[1], matches[2]
		}
	}

	return "", line
}

// IsListMarker checks if text starts with any list marker
func IsListMarker(text string) bool {
	text = strings.TrimSpace(text)
	marker, _ := extractListMarker(text)
	return marker != ""
}

// GetListMarkerType returns the marker type for a line
func GetListMarkerType(line string) string {
	line = strings.TrimSpace(line)

	if isOrderedListItem(line) {
		return "ordered"
	}
	if isChecklistItem(line) {
		return "checklist"
	}
	if isDefinitionItem(line) {
		return "definition"
	}
	if isUnorderedListItem(line) {
		return "unordered"
	}

	return ""
}

// NormalizeListMarkers normalizes list markers to a consistent format
func NormalizeListMarkers(text string, useNumbers bool) string {
	lines := strings.Split(text, "\n")
	var result []string
	itemNum := 1

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result = append(result, line)
			continue
		}

		indent := ""
		for _, r := range line {
			if unicode.IsSpace(r) {
				indent += string(r)
			} else {
				break
			}
		}

		marker, content := extractListMarker(trimmed)
		if marker != "" {
			if useNumbers {
				result = append(result, indent+string(rune('0'+itemNum))+". "+content)
				itemNum++
			} else {
				result = append(result, indent+"• "+content)
			}
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
