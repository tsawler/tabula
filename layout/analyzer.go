// Package layout provides document layout analysis for extracting semantic
// structure from PDF pages. It includes the unified Layout Analyzer that
// orchestrates all detection components including column, line, block,
// paragraph, heading, list, and reading order detection.
package layout

import (
	"sort"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// AnalyzerConfig holds configuration options for the layout analyzer.
// Each detection component has its own sub-configuration, and there are
// flags to enable or disable optional analysis features.
type AnalyzerConfig struct {
	// Column detection configuration
	ColumnConfig ColumnConfig

	// Line detection configuration
	LineConfig LineConfig

	// Paragraph detection configuration
	ParagraphConfig ParagraphConfig

	// Block detection configuration
	BlockConfig BlockConfig

	// Heading detection configuration
	HeadingConfig HeadingConfig

	// List detection configuration
	ListConfig ListConfig

	// Reading order configuration
	ReadingOrderConfig ReadingOrderConfig

	// DetectHeadings enables heading detection
	DetectHeadings bool

	// DetectLists enables list detection
	DetectLists bool

	// UseReadingOrder uses reading order for element ordering
	UseReadingOrder bool
}

// DefaultAnalyzerConfig returns a configuration with sensible defaults for
// typical document layout analysis, with all detection features enabled.
func DefaultAnalyzerConfig() AnalyzerConfig {
	return AnalyzerConfig{
		ColumnConfig:       DefaultColumnConfig(),
		LineConfig:         DefaultLineConfig(),
		ParagraphConfig:    DefaultParagraphConfig(),
		BlockConfig:        DefaultBlockConfig(),
		HeadingConfig:      DefaultHeadingConfig(),
		ListConfig:         DefaultListConfig(),
		ReadingOrderConfig: DefaultReadingOrderConfig(),
		DetectHeadings:     true,
		DetectLists:        true,
		UseReadingOrder:    true,
	}
}

// LayoutElement represents a detected layout element such as a paragraph,
// heading, or list. It includes the element type, bounding box, text content,
// and type-specific metadata.
type LayoutElement struct {
	// Type is the element type (paragraph, heading, list, etc.)
	Type model.ElementType

	// BBox is the bounding box of the element
	BBox model.BBox

	// Text is the text content of the element
	Text string

	// Index is the element's position in reading order
	Index int

	// ZOrder is the visual stacking order (for overlapping elements)
	ZOrder int

	// Heading contains heading-specific data (if Type == ElementTypeHeading)
	Heading *Heading

	// List contains list-specific data (if Type == ElementTypeList)
	List *List

	// Paragraph contains paragraph-specific data (if Type == ElementTypeParagraph)
	Paragraph *Paragraph

	// Lines are the lines that make up this element
	Lines []Line

	// Children contains nested elements (for compound structures)
	Children []LayoutElement
}

// ToModelElement converts the layout element to the appropriate model.Element
// implementation (Heading, List, or Paragraph) based on the element type.
func (le *LayoutElement) ToModelElement() model.Element {
	switch le.Type {
	case model.ElementTypeHeading:
		level := 1
		if le.Heading != nil {
			level = int(le.Heading.Level)
		}
		return &model.Heading{
			Text:     le.Text,
			Level:    level,
			BBox:     le.BBox,
			FontSize: le.fontSize(),
			ZOrder:   le.ZOrder,
		}

	case model.ElementTypeList:
		var items []model.ListItem
		if le.List != nil {
			for _, item := range le.List.Items {
				items = append(items, model.ListItem{
					Text:  item.Text,
					Level: item.Level,
				})
			}
		}
		ordered := false
		if le.List != nil && (le.List.Type == ListTypeNumbered ||
			le.List.Type == ListTypeLettered ||
			le.List.Type == ListTypeRoman) {
			ordered = true
		}
		return &model.List{
			Items:   items,
			Ordered: ordered,
			BBox:    le.BBox,
			ZOrder:  le.ZOrder,
		}

	case model.ElementTypeParagraph:
		fallthrough
	default:
		alignment := model.AlignLeft
		if le.Paragraph != nil {
			alignment = toModelAlignment(le.Paragraph.Alignment)
		}
		return &model.Paragraph{
			Text:      le.Text,
			BBox:      le.BBox,
			FontSize:  le.fontSize(),
			Alignment: alignment,
			ZOrder:    le.ZOrder,
		}
	}
}

// fontSize returns the font size for the element, checking Heading, Paragraph,
// and Lines in that order, defaulting to 12.0 if no font size is available.
func (le *LayoutElement) fontSize() float64 {
	if le.Heading != nil {
		return le.Heading.FontSize
	}
	if le.Paragraph != nil {
		return le.Paragraph.AverageFontSize
	}
	if len(le.Lines) > 0 {
		return le.Lines[0].AverageFontSize
	}
	return 12.0 // default
}

// toModelAlignment converts a layout LineAlignment to a model.TextAlignment.
func toModelAlignment(align LineAlignment) model.TextAlignment {
	switch align {
	case AlignLeft:
		return model.AlignLeft
	case AlignCenter:
		return model.AlignCenter
	case AlignRight:
		return model.AlignRight
	case AlignJustified:
		return model.AlignJustify
	default:
		return model.AlignLeft
	}
}

// AnalysisResult holds the complete results from layout analysis, including
// detected elements, intermediate analysis structures (columns, lines, blocks,
// paragraphs, headings, lists), and statistics about the analysis.
type AnalysisResult struct {
	// Elements are all detected elements in reading order
	Elements []LayoutElement

	// Columns is the column layout analysis
	Columns *ColumnLayout

	// ReadingOrder is the reading order analysis
	ReadingOrder *ReadingOrderResult

	// Headings are all detected headings
	Headings *HeadingLayout

	// Lists are all detected lists
	Lists *ListLayout

	// Paragraphs are all detected paragraphs
	Paragraphs *ParagraphLayout

	// Blocks are all detected text blocks
	Blocks *BlockLayout

	// Lines are all detected lines
	Lines *LineLayout

	// PageWidth and PageHeight
	PageWidth  float64
	PageHeight float64

	// Statistics
	Stats AnalysisStats
}

// AnalysisStats contains counts of detected elements from the layout analysis.
type AnalysisStats struct {
	FragmentCount  int
	LineCount      int
	BlockCount     int
	ParagraphCount int
	HeadingCount   int
	ListCount      int
	ColumnCount    int
	ElementCount   int
}

// GetElements converts all layout elements to model.Element interfaces,
// returning them in reading order.
func (r *AnalysisResult) GetElements() []model.Element {
	elements := make([]model.Element, len(r.Elements))
	for i, le := range r.Elements {
		elements[i] = le.ToModelElement()
	}
	return elements
}

// GetText returns all extracted text concatenated in reading order.
// It prefers reading order text if available, falling back to paragraph text.
func (r *AnalysisResult) GetText() string {
	if r.ReadingOrder != nil {
		return r.ReadingOrder.GetText()
	}
	if r.Paragraphs != nil {
		return r.Paragraphs.GetText()
	}
	return ""
}

// GetMarkdown returns a Markdown representation of the document with headings,
// lists, and paragraphs formatted appropriately.
func (r *AnalysisResult) GetMarkdown() string {
	var result string

	for _, elem := range r.Elements {
		switch elem.Type {
		case model.ElementTypeHeading:
			if elem.Heading != nil {
				result += elem.Heading.ToMarkdown() + "\n\n"
			}
		case model.ElementTypeList:
			if elem.List != nil {
				result += elem.List.ToMarkdown() + "\n\n"
			}
		default:
			result += elem.Text + "\n\n"
		}
	}

	return result
}

// Analyzer orchestrates all layout detection components to extract semantic
// structure from PDF pages. It combines column, line, block, paragraph,
// heading, list, and reading order detection into a unified analysis pipeline.
type Analyzer struct {
	config AnalyzerConfig

	// Detectors
	columnDetector       *ColumnDetector
	lineDetector         *LineDetector
	blockDetector        *BlockDetector
	paragraphDetector    *ParagraphDetector
	headingDetector      *HeadingDetector
	listDetector         *ListDetector
	readingOrderDetector *ReadingOrderDetector
}

// NewAnalyzer creates a new layout analyzer with default configuration.
func NewAnalyzer() *Analyzer {
	return NewAnalyzerWithConfig(DefaultAnalyzerConfig())
}

// NewAnalyzerWithConfig creates a new layout analyzer with the specified configuration.
func NewAnalyzerWithConfig(config AnalyzerConfig) *Analyzer {
	return &Analyzer{
		config:               config,
		columnDetector:       NewColumnDetectorWithConfig(config.ColumnConfig),
		lineDetector:         NewLineDetectorWithConfig(config.LineConfig),
		blockDetector:        NewBlockDetectorWithConfig(config.BlockConfig),
		paragraphDetector:    NewParagraphDetectorWithConfig(config.ParagraphConfig),
		headingDetector:      NewHeadingDetectorWithConfig(config.HeadingConfig),
		listDetector:         NewListDetectorWithConfig(config.ListConfig),
		readingOrderDetector: NewReadingOrderDetectorWithConfig(config.ReadingOrderConfig),
	}
}

// Analyze performs complete layout analysis on the given text fragments.
// It runs through all detection phases: column detection, reading order analysis,
// line detection, block detection, paragraph detection, heading detection (if enabled),
// list detection (if enabled), and finally builds a unified element tree.
func (a *Analyzer) Analyze(fragments []text.TextFragment, pageWidth, pageHeight float64) *AnalysisResult {
	result := &AnalysisResult{
		PageWidth:  pageWidth,
		PageHeight: pageHeight,
		Stats: AnalysisStats{
			FragmentCount: len(fragments),
		},
	}

	if len(fragments) == 0 {
		return result
	}

	// Step 1: Column detection
	result.Columns = a.columnDetector.Detect(fragments, pageWidth, pageHeight)
	if result.Columns != nil {
		result.Stats.ColumnCount = len(result.Columns.Columns)
	}

	// Step 2: Reading order analysis (uses column detection internally)
	if a.config.UseReadingOrder {
		result.ReadingOrder = a.readingOrderDetector.Detect(fragments, pageWidth, pageHeight)
	}

	// Step 3: Line detection
	result.Lines = a.lineDetector.Detect(fragments, pageWidth, pageHeight)
	if result.Lines != nil {
		result.Stats.LineCount = len(result.Lines.Lines)
	}

	// Step 4: Block detection
	result.Blocks = a.blockDetector.Detect(fragments, pageWidth, pageHeight)
	if result.Blocks != nil {
		result.Stats.BlockCount = len(result.Blocks.Blocks)
	}

	// Step 5: Paragraph detection (uses reading order if available)
	if result.ReadingOrder != nil {
		result.Paragraphs = result.ReadingOrder.GetParagraphs()
	} else {
		result.Paragraphs = a.paragraphDetector.DetectFromFragments(fragments, pageWidth, pageHeight)
	}
	if result.Paragraphs != nil {
		result.Stats.ParagraphCount = len(result.Paragraphs.Paragraphs)
	}

	// Step 6: Heading detection
	if a.config.DetectHeadings {
		result.Headings = a.headingDetector.DetectFromFragments(fragments, pageWidth, pageHeight)
		if result.Headings != nil {
			result.Stats.HeadingCount = len(result.Headings.Headings)
		}
	}

	// Step 7: List detection
	if a.config.DetectLists {
		result.Lists = a.listDetector.DetectFromFragments(fragments, pageWidth, pageHeight)
		if result.Lists != nil {
			result.Stats.ListCount = len(result.Lists.Lists)
		}
	}

	// Step 8: Build unified element tree
	result.Elements = a.buildElementTree(result)
	result.Stats.ElementCount = len(result.Elements)

	return result
}

// buildElementTree creates a unified element tree from all detected components.
// It merges headings, lists, and paragraphs, avoiding duplicates where elements
// overlap, and sorts them into reading order.
func (a *Analyzer) buildElementTree(result *AnalysisResult) []LayoutElement {
	var elements []LayoutElement

	// Track which paragraphs have been consumed by headings or lists
	consumedParaIndices := make(map[int]bool)

	// Add headings
	if result.Headings != nil {
		for i, heading := range result.Headings.Headings {
			elem := LayoutElement{
				Type:    model.ElementTypeHeading,
				BBox:    heading.BBox,
				Text:    heading.Text,
				Index:   i,
				Heading: &heading,
				Lines:   heading.Lines,
			}
			elements = append(elements, elem)

			// Mark overlapping paragraphs as consumed
			if result.Paragraphs != nil {
				for j, para := range result.Paragraphs.Paragraphs {
					if bboxOverlaps(heading.BBox, para.BBox) {
						consumedParaIndices[j] = true
					}
				}
			}
		}
	}

	// Add lists
	if result.Lists != nil {
		for i, list := range result.Lists.Lists {
			elem := LayoutElement{
				Type:  model.ElementTypeList,
				BBox:  list.BBox,
				Text:  getListText(&list),
				Index: i,
				List:  &list,
			}
			elements = append(elements, elem)

			// Mark overlapping paragraphs as consumed
			if result.Paragraphs != nil {
				for j, para := range result.Paragraphs.Paragraphs {
					if bboxOverlaps(list.BBox, para.BBox) {
						consumedParaIndices[j] = true
					}
				}
			}
		}
	}

	// Add remaining paragraphs
	if result.Paragraphs != nil {
		for i, para := range result.Paragraphs.Paragraphs {
			if consumedParaIndices[i] {
				continue
			}
			elem := LayoutElement{
				Type:      model.ElementTypeParagraph,
				BBox:      para.BBox,
				Text:      para.Text,
				Index:     i,
				Paragraph: &para,
				Lines:     para.Lines,
			}
			elements = append(elements, elem)
		}
	}

	// Sort elements by reading order (top to bottom, respecting columns)
	sortElementsByReadingOrder(elements, result.ReadingOrder)

	// Reassign indices after sorting
	for i := range elements {
		elements[i].Index = i
		elements[i].ZOrder = i
	}

	return elements
}

// getListText extracts all text from a list by concatenating item prefixes and text.
func getListText(list *List) string {
	var text string
	for _, item := range list.Items {
		text += item.Prefix + " " + item.Text + "\n"
	}
	return text
}

// bboxOverlaps reports whether two bounding boxes overlap significantly
// (more than 50% of the smaller box's area is covered by the overlap).
func bboxOverlaps(a, b model.BBox) bool {
	// Check for no overlap
	if a.X+a.Width < b.X || b.X+b.Width < a.X {
		return false
	}
	if a.Y+a.Height < b.Y || b.Y+b.Height < a.Y {
		return false
	}

	// Calculate overlap area
	overlapX := minFloat(a.X+a.Width, b.X+b.Width) - maxFloat(a.X, b.X)
	overlapY := minFloat(a.Y+a.Height, b.Y+b.Height) - maxFloat(a.Y, b.Y)

	if overlapX <= 0 || overlapY <= 0 {
		return false
	}

	overlapArea := overlapX * overlapY
	smallerArea := minFloat(a.Width*a.Height, b.Width*b.Height)

	// Consider overlapping if >50% of smaller box is covered
	return overlapArea > smallerArea*0.5
}

// sortElementsByReadingOrder sorts elements according to reading order analysis.
// If no reading order is available, falls back to top-to-bottom, left-to-right sorting.
func sortElementsByReadingOrder(elements []LayoutElement, ro *ReadingOrderResult) {
	if ro == nil || len(ro.Sections) == 0 {
		// Fall back to simple top-to-bottom, left-to-right sorting
		sort.Slice(elements, func(i, j int) bool {
			// Group by Y with tolerance
			yDiff := elements[i].BBox.Y - elements[j].BBox.Y
			if absValue(yDiff) > 10 {
				return yDiff > 0 // Higher Y first (PDF coordinates: Y increases upward)
			}
			return elements[i].BBox.X < elements[j].BBox.X
		})
		return
	}

	// Build a position map based on reading order sections
	positionMap := make(map[*LayoutElement]int)
	position := 0

	for _, section := range ro.Sections {
		for _, line := range section.Lines {
			// Find elements that overlap with this line
			for i := range elements {
				elem := &elements[i]
				if _, exists := positionMap[elem]; exists {
					continue
				}
				if lineOverlapsElement(line, *elem) {
					positionMap[elem] = position
					position++
				}
			}
		}
	}

	// Assign positions to any remaining elements
	for i := range elements {
		elem := &elements[i]
		if _, exists := positionMap[elem]; !exists {
			positionMap[elem] = position
			position++
		}
	}

	// Sort by position
	sort.Slice(elements, func(i, j int) bool {
		return positionMap[&elements[i]] < positionMap[&elements[j]]
	})
}

// lineOverlapsElement reports whether a line's bounding box overlaps with an element's bounding box.
func lineOverlapsElement(line Line, elem LayoutElement) bool {
	// Check Y overlap with tolerance
	lineTop := line.BBox.Y + line.BBox.Height
	lineBottom := line.BBox.Y
	elemTop := elem.BBox.Y + elem.BBox.Height
	elemBottom := elem.BBox.Y

	// Y ranges overlap?
	if lineBottom > elemTop || elemBottom > lineTop {
		return false
	}

	// X ranges overlap?
	if line.BBox.X+line.BBox.Width < elem.BBox.X || elem.BBox.X+elem.BBox.Width < line.BBox.X {
		return false
	}

	return true
}

// minFloat returns the smaller of two float64 values.
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// maxFloat returns the larger of two float64 values.
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// absValue returns the absolute value of a float64.
func absValue(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// AnalyzeWithHeaderFooterFiltering performs layout analysis with automatic
// header and footer detection and removal. This requires multiple pages to
// identify repeated content at the top and bottom of pages.
func (a *Analyzer) AnalyzeWithHeaderFooterFiltering(
	pageFragments []PageFragments,
	pageIndex int,
) *AnalysisResult {
	if len(pageFragments) == 0 || pageIndex < 0 || pageIndex >= len(pageFragments) {
		return &AnalysisResult{}
	}

	// Detect headers/footers across all pages
	hfDetector := NewHeaderFooterDetector()
	hfResult := hfDetector.Detect(pageFragments)

	// Get the target page
	targetPage := pageFragments[pageIndex]

	// Filter fragments
	filteredFragments := targetPage.Fragments
	if hfResult != nil {
		filteredFragments = hfResult.FilterFragments(
			pageIndex,
			targetPage.Fragments,
			targetPage.PageHeight,
		)
	}

	// Perform analysis on filtered fragments
	return a.Analyze(filteredFragments, targetPage.PageWidth, targetPage.PageHeight)
}

// QuickAnalyze performs a fast analysis focusing on text structure without
// detailed heading or list detection. It only runs reading order and paragraph
// detection for better performance when fine-grained structure is not needed.
func (a *Analyzer) QuickAnalyze(fragments []text.TextFragment, pageWidth, pageHeight float64) *AnalysisResult {
	result := &AnalysisResult{
		PageWidth:  pageWidth,
		PageHeight: pageHeight,
		Stats: AnalysisStats{
			FragmentCount: len(fragments),
		},
	}

	if len(fragments) == 0 {
		return result
	}

	// Just do reading order and paragraph detection
	result.ReadingOrder = a.readingOrderDetector.Detect(fragments, pageWidth, pageHeight)
	if result.ReadingOrder != nil {
		result.Paragraphs = result.ReadingOrder.GetParagraphs()
		result.Stats.ColumnCount = result.ReadingOrder.ColumnCount
	}

	if result.Paragraphs != nil {
		result.Stats.ParagraphCount = len(result.Paragraphs.Paragraphs)

		// Convert paragraphs to elements
		for i, para := range result.Paragraphs.Paragraphs {
			elem := LayoutElement{
				Type:      model.ElementTypeParagraph,
				BBox:      para.BBox,
				Text:      para.Text,
				Index:     i,
				ZOrder:    i,
				Paragraph: &para,
				Lines:     para.Lines,
			}
			result.Elements = append(result.Elements, elem)
		}
		result.Stats.ElementCount = len(result.Elements)
	}

	return result
}
