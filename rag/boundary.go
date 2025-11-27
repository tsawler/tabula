package rag

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/tsawler/tabula/model"
)

// BoundaryType represents the type of semantic boundary
type BoundaryType int

const (
	// BoundaryNone indicates no boundary (middle of content)
	BoundaryNone BoundaryType = iota
	// BoundarySentence indicates a sentence ending
	BoundarySentence
	// BoundaryParagraph indicates a paragraph break
	BoundaryParagraph
	// BoundaryList indicates end of a list
	BoundaryList
	// BoundaryListItem indicates end of a list item
	BoundaryListItem
	// BoundaryHeading indicates a heading (section break)
	BoundaryHeading
	// BoundaryTable indicates end of a table
	BoundaryTable
	// BoundaryFigure indicates end of a figure/image
	BoundaryFigure
	// BoundaryCodeBlock indicates end of a code block
	BoundaryCodeBlock
	// BoundaryPageBreak indicates a page break
	BoundaryPageBreak
)

// String returns a human-readable representation of the boundary type
func (bt BoundaryType) String() string {
	switch bt {
	case BoundaryNone:
		return "none"
	case BoundarySentence:
		return "sentence"
	case BoundaryParagraph:
		return "paragraph"
	case BoundaryList:
		return "list"
	case BoundaryListItem:
		return "list_item"
	case BoundaryHeading:
		return "heading"
	case BoundaryTable:
		return "table"
	case BoundaryFigure:
		return "figure"
	case BoundaryCodeBlock:
		return "code_block"
	case BoundaryPageBreak:
		return "page_break"
	default:
		return "unknown"
	}
}

// Score returns a priority score for this boundary type (higher = better split point)
func (bt BoundaryType) Score() int {
	switch bt {
	case BoundaryHeading:
		return 100 // Best split point - new section
	case BoundaryPageBreak:
		return 90 // Good split point - page boundary
	case BoundaryTable:
		return 85 // After table is good
	case BoundaryFigure:
		return 85 // After figure is good
	case BoundaryList:
		return 80 // After list is good
	case BoundaryCodeBlock:
		return 80 // After code block is good
	case BoundaryParagraph:
		return 70 // Paragraph break is acceptable
	case BoundaryListItem:
		return 30 // Avoid splitting within lists
	case BoundarySentence:
		return 20 // Only if necessary
	case BoundaryNone:
		return 0 // Never split here
	default:
		return 0
	}
}

// Boundary represents a potential chunk boundary in the content
type Boundary struct {
	// Type is the kind of boundary
	Type BoundaryType

	// Position is the character offset in the text
	Position int

	// Score is the priority score for splitting here
	Score int

	// ElementIndex is the index of the element this boundary follows
	ElementIndex int

	// Context provides additional information about the boundary
	Context string
}

// BoundaryDetector detects semantic boundaries in content
type BoundaryDetector struct {
	config BoundaryConfig
}

// BoundaryConfig holds configuration for boundary detection
type BoundaryConfig struct {
	// MinChunkSize is the minimum characters before considering a boundary
	MinChunkSize int

	// MaxChunkSize is the maximum characters before forcing a boundary
	MaxChunkSize int

	// PreferParagraphBreaks prefers paragraph boundaries over sentence boundaries
	PreferParagraphBreaks bool

	// KeepListsIntact tries to keep lists with their introductory text
	KeepListsIntact bool

	// KeepTablesIntact treats tables as atomic units
	KeepTablesIntact bool

	// KeepFiguresIntact keeps figures with their captions
	KeepFiguresIntact bool

	// LookAheadChars is how far to look ahead for better boundaries
	LookAheadChars int

	// ListIntroPatterns are patterns that indicate list introductions
	ListIntroPatterns []*regexp.Regexp
}

// DefaultBoundaryConfig returns sensible defaults for boundary detection
func DefaultBoundaryConfig() BoundaryConfig {
	return BoundaryConfig{
		MinChunkSize:          100,
		MaxChunkSize:          2000,
		PreferParagraphBreaks: true,
		KeepListsIntact:       true,
		KeepTablesIntact:      true,
		KeepFiguresIntact:     true,
		LookAheadChars:        200,
		ListIntroPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(the\s+following|here\s+are|these\s+(are|include)|below\s+(are|is)|as\s+follows)\s*:?\s*$`),
			regexp.MustCompile(`(?i)(steps?|features?|items?|points?|reasons?|benefits?|advantages?|options?|examples?)\s*:?\s*$`),
			regexp.MustCompile(`(?i)(include|includes|including|such\s+as|for\s+example|e\.g\.|i\.e\.)\s*:?\s*$`),
			regexp.MustCompile(`:\s*$`), // Ends with colon
		},
	}
}

// NewBoundaryDetector creates a new boundary detector with default configuration
func NewBoundaryDetector() *BoundaryDetector {
	return &BoundaryDetector{
		config: DefaultBoundaryConfig(),
	}
}

// NewBoundaryDetectorWithConfig creates a boundary detector with custom configuration
func NewBoundaryDetectorWithConfig(config BoundaryConfig) *BoundaryDetector {
	return &BoundaryDetector{
		config: config,
	}
}

// ContentBlock represents a block of content for boundary detection
type ContentBlock struct {
	Type     model.ElementType
	Text     string
	Page     int
	Index    int
	ListInfo *model.ListInfo
	IsIntro  bool // True if this appears to introduce the next element
}

// DetectBoundaries finds all semantic boundaries in a sequence of content blocks
func (d *BoundaryDetector) DetectBoundaries(blocks []ContentBlock) []Boundary {
	var boundaries []Boundary
	position := 0

	for i, block := range blocks {
		// Add boundary before headings
		if block.Type == model.ElementTypeHeading && i > 0 {
			boundaries = append(boundaries, Boundary{
				Type:         BoundaryHeading,
				Position:     position,
				Score:        BoundaryHeading.Score(),
				ElementIndex: i - 1,
				Context:      "before heading",
			})
		}

		// Detect boundaries within the block
		internalBoundaries := d.detectInternalBoundaries(block, position, i)
		boundaries = append(boundaries, internalBoundaries...)

		// Add boundary after each block based on type
		position += len(block.Text)
		if i < len(blocks)-1 {
			position += 2 // Account for "\n\n" separator
		}

		boundaryType := d.getBoundaryTypeForBlock(block, blocks, i)
		if boundaryType != BoundaryNone {
			boundaries = append(boundaries, Boundary{
				Type:         boundaryType,
				Position:     position,
				Score:        boundaryType.Score(),
				ElementIndex: i,
				Context:      "after " + block.Type.String(),
			})
		}
	}

	return boundaries
}

// getBoundaryTypeForBlock determines the boundary type after a block
func (d *BoundaryDetector) getBoundaryTypeForBlock(block ContentBlock, blocks []ContentBlock, index int) BoundaryType {
	switch block.Type {
	case model.ElementTypeHeading:
		// Don't create boundary right after heading - content should follow
		return BoundaryNone

	case model.ElementTypeTable:
		return BoundaryTable

	case model.ElementTypeImage, model.ElementTypeFigure:
		return BoundaryFigure

	case model.ElementTypeList:
		return BoundaryList

	case model.ElementTypeParagraph:
		// Check if this paragraph introduces the next element
		if d.isListIntro(block.Text) && index+1 < len(blocks) {
			nextBlock := blocks[index+1]
			if nextBlock.Type == model.ElementTypeList {
				// Don't create boundary - keep intro with list
				return BoundaryNone
			}
		}
		return BoundaryParagraph

	default:
		return BoundaryParagraph
	}
}

// detectInternalBoundaries finds sentence boundaries within a block of text
func (d *BoundaryDetector) detectInternalBoundaries(block ContentBlock, startPosition int, blockIndex int) []Boundary {
	var boundaries []Boundary

	// Only detect sentence boundaries in paragraphs
	if block.Type != model.ElementTypeParagraph {
		return boundaries
	}

	text := block.Text
	position := startPosition

	for i := 0; i < len(text); i++ {
		if isSentenceEnd(text, i) {
			boundaries = append(boundaries, Boundary{
				Type:         BoundarySentence,
				Position:     position + i + 1,
				Score:        BoundarySentence.Score(),
				ElementIndex: blockIndex,
				Context:      "sentence end",
			})
		}
	}

	return boundaries
}

// isListIntro checks if text appears to introduce a list
func (d *BoundaryDetector) isListIntro(text string) bool {
	text = strings.TrimSpace(text)

	for _, pattern := range d.config.ListIntroPatterns {
		if pattern.MatchString(text) {
			return true
		}
	}

	return false
}

// isSentenceEnd checks if position i in text is a sentence ending
func isSentenceEnd(text string, i int) bool {
	if i >= len(text) {
		return false
	}

	r := rune(text[i])

	// Check for sentence-ending punctuation
	if r != '.' && r != '!' && r != '?' {
		return false
	}

	// Check it's not an abbreviation (single capital letter before period)
	if r == '.' && i >= 1 {
		prev := rune(text[i-1])
		if unicode.IsUpper(prev) {
			// Check if it's a single letter (like "Mr." or "Dr.")
			if i < 2 || !unicode.IsLetter(rune(text[i-2])) {
				return false
			}
		}

		// Check for common abbreviations
		if isAbbreviation(text, i) {
			return false
		}

		// Check for decimal numbers (e.g., "3.14")
		if unicode.IsDigit(prev) && i+1 < len(text) && unicode.IsDigit(rune(text[i+1])) {
			return false
		}
	}

	// Check if followed by space and capital letter (or end of text)
	if i+1 >= len(text) {
		return true
	}

	// Look for space followed by content
	if i+2 < len(text) && unicode.IsSpace(rune(text[i+1])) {
		next := rune(text[i+2])
		// Sentence end if followed by capital letter or quote
		if unicode.IsUpper(next) || next == '"' || next == '\'' {
			return true
		}
	}

	return false
}

// isAbbreviation checks if the period at position i is part of an abbreviation
func isAbbreviation(text string, i int) bool {
	// Common abbreviations that end with period
	abbreviations := []string{
		"mr.", "mrs.", "ms.", "dr.", "prof.",
		"sr.", "jr.", "vs.", "etc.", "e.g.", "i.e.",
		"inc.", "ltd.", "co.", "corp.",
		"jan.", "feb.", "mar.", "apr.", "jun.", "jul.", "aug.", "sep.", "oct.", "nov.", "dec.",
		"st.", "rd.", "ave.", "blvd.",
		"no.", "vol.", "pp.", "pg.",
	}

	// Get the word before the period
	start := i
	for start > 0 && unicode.IsLetter(rune(text[start-1])) {
		start--
	}

	if start >= i {
		return false
	}

	word := strings.ToLower(text[start : i+1])
	for _, abbr := range abbreviations {
		if word == abbr {
			return true
		}
	}

	return false
}

// FindBestBoundary finds the best boundary within a range for splitting
func (d *BoundaryDetector) FindBestBoundary(boundaries []Boundary, minPos, maxPos int) *Boundary {
	var best *Boundary
	bestScore := -1

	for i := range boundaries {
		b := &boundaries[i]
		if b.Position >= minPos && b.Position <= maxPos {
			if b.Score > bestScore {
				best = b
				bestScore = b.Score
			}
		}
	}

	return best
}

// FindBoundaryWithLookAhead finds a boundary, looking ahead for better options
func (d *BoundaryDetector) FindBoundaryWithLookAhead(boundaries []Boundary, targetPos int) *Boundary {
	minPos := targetPos - d.config.LookAheadChars/2
	maxPos := targetPos + d.config.LookAheadChars/2

	if minPos < 0 {
		minPos = 0
	}

	return d.FindBestBoundary(boundaries, minPos, maxPos)
}

// ShouldKeepTogether determines if two blocks should be kept in the same chunk
func (d *BoundaryDetector) ShouldKeepTogether(block1, block2 ContentBlock) bool {
	// Keep list intro with list
	if d.config.KeepListsIntact {
		if block1.Type == model.ElementTypeParagraph && block2.Type == model.ElementTypeList {
			if d.isListIntro(block1.Text) {
				return true
			}
		}
	}

	// Keep figure caption with figure
	if d.config.KeepFiguresIntact {
		if block1.Type == model.ElementTypeCaption && block2.Type == model.ElementTypeImage {
			return true
		}
		if block1.Type == model.ElementTypeImage && block2.Type == model.ElementTypeCaption {
			return true
		}
	}

	// Keep heading with following content
	if block1.Type == model.ElementTypeHeading {
		return true
	}

	return false
}

// AtomicBlocks identifies blocks that should not be split
type AtomicBlock struct {
	StartIndex int
	EndIndex   int
	Type       string
	Reason     string
}

// FindAtomicBlocks identifies sequences of blocks that should stay together
func (d *BoundaryDetector) FindAtomicBlocks(blocks []ContentBlock) []AtomicBlock {
	var atomic []AtomicBlock

	for i := 0; i < len(blocks); i++ {
		block := blocks[i]

		// Tables are atomic
		if d.config.KeepTablesIntact && block.Type == model.ElementTypeTable {
			atomic = append(atomic, AtomicBlock{
				StartIndex: i,
				EndIndex:   i,
				Type:       "table",
				Reason:     "tables should not be split",
			})
			continue
		}

		// Lists with their intros are atomic
		if d.config.KeepListsIntact && block.Type == model.ElementTypeList {
			// Check if previous block is an intro
			startIdx := i
			if i > 0 && blocks[i-1].Type == model.ElementTypeParagraph {
				if d.isListIntro(blocks[i-1].Text) {
					startIdx = i - 1
				}
			}
			atomic = append(atomic, AtomicBlock{
				StartIndex: startIdx,
				EndIndex:   i,
				Type:       "list",
				Reason:     "list with introduction",
			})
			continue
		}

		// Figures with captions are atomic
		if d.config.KeepFiguresIntact {
			if block.Type == model.ElementTypeImage || block.Type == model.ElementTypeFigure {
				startIdx, endIdx := i, i

				// Check for caption before
				if i > 0 && blocks[i-1].Type == model.ElementTypeCaption {
					startIdx = i - 1
				}
				// Check for caption after
				if i+1 < len(blocks) && blocks[i+1].Type == model.ElementTypeCaption {
					endIdx = i + 1
				}

				atomic = append(atomic, AtomicBlock{
					StartIndex: startIdx,
					EndIndex:   endIdx,
					Type:       "figure",
					Reason:     "figure with caption",
				})
				continue
			}
		}
	}

	return atomic
}

// IsWithinAtomicBlock checks if an index is within any atomic block
func IsWithinAtomicBlock(index int, atomicBlocks []AtomicBlock) bool {
	for _, ab := range atomicBlocks {
		if index >= ab.StartIndex && index <= ab.EndIndex {
			return true
		}
	}
	return false
}

// GetAtomicBlockAt returns the atomic block containing the given index, if any
func GetAtomicBlockAt(index int, atomicBlocks []AtomicBlock) *AtomicBlock {
	for i := range atomicBlocks {
		if index >= atomicBlocks[i].StartIndex && index <= atomicBlocks[i].EndIndex {
			return &atomicBlocks[i]
		}
	}
	return nil
}

// OrphanedContentDetector helps avoid creating orphaned content at chunk boundaries
type OrphanedContentDetector struct {
	// MinOrphanSize is the minimum size for standalone content
	MinOrphanSize int
}

// NewOrphanedContentDetector creates a new orphan detector
func NewOrphanedContentDetector(minSize int) *OrphanedContentDetector {
	return &OrphanedContentDetector{
		MinOrphanSize: minSize,
	}
}

// WouldCreateOrphan checks if splitting at position would create orphaned content
func (o *OrphanedContentDetector) WouldCreateOrphan(text string, position int) bool {
	// Check content before split point
	before := strings.TrimSpace(text[:position])
	if len(before) > 0 && len(before) < o.MinOrphanSize {
		return true
	}

	// Check content after split point
	if position < len(text) {
		after := strings.TrimSpace(text[position:])
		if len(after) > 0 && len(after) < o.MinOrphanSize {
			return true
		}
	}

	return false
}

// AdjustForOrphans adjusts a split position to avoid orphaned content
func (o *OrphanedContentDetector) AdjustForOrphans(text string, position int, boundaries []Boundary) int {
	// If position would create orphan, find nearby boundary that doesn't
	if !o.WouldCreateOrphan(text, position) {
		return position
	}

	// Look for alternative boundaries
	for _, b := range boundaries {
		if b.Position > position-o.MinOrphanSize && b.Position < position+o.MinOrphanSize {
			if !o.WouldCreateOrphan(text, b.Position) {
				return b.Position
			}
		}
	}

	// No good alternative found - return original
	return position
}
