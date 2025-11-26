// Package layout provides document layout analysis including block detection,
// column detection, reading order determination, and structural element identification.
package layout

import (
	"sort"

	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/text"
)

// Block represents a contiguous rectangular region of text on a page.
// Blocks are spatially coherent groups of fragments separated by whitespace.
type Block struct {
	// BBox is the bounding box of the block
	BBox model.BBox

	// Fragments are the text fragments contained in this block (in reading order)
	Fragments []text.TextFragment

	// Lines are the fragments grouped into horizontal lines
	Lines [][]text.TextFragment

	// Index is the block's position in reading order (0-based)
	Index int

	// Level indicates nesting depth (0 = top level)
	Level int
}

// BlockLayout represents the detected block structure of a page
type BlockLayout struct {
	// Blocks are the detected text blocks (in reading order)
	Blocks []Block

	// PageWidth is the width of the page
	PageWidth float64

	// PageHeight is the height of the page
	PageHeight float64

	// Config is the configuration used for detection
	Config BlockConfig
}

// BlockConfig holds configuration for block detection
type BlockConfig struct {
	// LineHeightTolerance is the Y-distance tolerance for grouping fragments into lines
	// as a fraction of fragment height (default: 0.5)
	LineHeightTolerance float64

	// HorizontalGapThreshold is the minimum horizontal gap to consider fragments separate
	// as a fraction of average font size (default: 3.0)
	HorizontalGapThreshold float64

	// VerticalGapThreshold is the minimum vertical gap to start a new block
	// as a fraction of average line height (default: 1.5)
	VerticalGapThreshold float64

	// MinBlockWidth is the minimum width for a valid block (default: 10 points)
	MinBlockWidth float64

	// MinBlockHeight is the minimum height for a valid block (default: 5 points)
	MinBlockHeight float64

	// MergeOverlappingBlocks controls whether overlapping blocks should be merged
	MergeOverlappingBlocks bool
}

// DefaultBlockConfig returns sensible default configuration
func DefaultBlockConfig() BlockConfig {
	return BlockConfig{
		LineHeightTolerance:    0.5,
		HorizontalGapThreshold: 3.0,
		VerticalGapThreshold:   1.5,
		MinBlockWidth:          10.0,
		MinBlockHeight:         5.0,
		MergeOverlappingBlocks: true,
	}
}

// BlockDetector detects text blocks on a page
type BlockDetector struct {
	config BlockConfig
}

// NewBlockDetector creates a new block detector with default configuration
func NewBlockDetector() *BlockDetector {
	return &BlockDetector{
		config: DefaultBlockConfig(),
	}
}

// NewBlockDetectorWithConfig creates a block detector with custom configuration
func NewBlockDetectorWithConfig(config BlockConfig) *BlockDetector {
	return &BlockDetector{
		config: config,
	}
}

// Detect analyzes text fragments and detects block layout
func (d *BlockDetector) Detect(fragments []text.TextFragment, pageWidth, pageHeight float64) *BlockLayout {
	if len(fragments) == 0 {
		return &BlockLayout{
			Blocks:     nil,
			PageWidth:  pageWidth,
			PageHeight: pageHeight,
			Config:     d.config,
		}
	}

	// Step 1: Group fragments into lines
	lines := d.groupIntoLines(fragments)

	// Step 2: Group lines into blocks based on vertical gaps
	blocks := d.groupLinesIntoBlocks(lines)

	// Step 3: Optionally merge overlapping blocks
	if d.config.MergeOverlappingBlocks {
		blocks = d.mergeOverlappingBlocks(blocks)
	}

	// Step 4: Sort blocks in reading order and assign indices
	blocks = d.sortBlocksInReadingOrder(blocks)

	// Step 5: Validate blocks
	blocks = d.validateBlocks(blocks)

	return &BlockLayout{
		Blocks:     blocks,
		PageWidth:  pageWidth,
		PageHeight: pageHeight,
		Config:     d.config,
	}
}

// groupIntoLines groups fragments into horizontal lines based on Y position
func (d *BlockDetector) groupIntoLines(fragments []text.TextFragment) [][]text.TextFragment {
	if len(fragments) == 0 {
		return nil
	}

	// Sort fragments by Y (descending, top to bottom in PDF coords) then X
	sorted := make([]text.TextFragment, len(fragments))
	copy(sorted, fragments)
	sort.Slice(sorted, func(i, j int) bool {
		yDiff := sorted[i].Y - sorted[j].Y
		avgHeight := (sorted[i].Height + sorted[j].Height) / 2
		tolerance := avgHeight * d.config.LineHeightTolerance
		if absFloat64(yDiff) > tolerance {
			return yDiff > 0 // Higher Y first (top of page)
		}
		return sorted[i].X < sorted[j].X // Left to right
	})

	var lines [][]text.TextFragment
	var currentLine []text.TextFragment

	for _, frag := range sorted {
		if len(currentLine) == 0 {
			currentLine = append(currentLine, frag)
			continue
		}

		// Check if fragment is on same line as previous
		lastFrag := currentLine[len(currentLine)-1]
		avgHeight := (frag.Height + lastFrag.Height) / 2
		tolerance := avgHeight * d.config.LineHeightTolerance

		if absFloat64(frag.Y-lastFrag.Y) <= tolerance {
			// Same line
			currentLine = append(currentLine, frag)
		} else {
			// New line
			lines = append(lines, currentLine)
			currentLine = []text.TextFragment{frag}
		}
	}

	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	// Sort fragments within each line by X
	for i := range lines {
		sort.Slice(lines[i], func(a, b int) bool {
			return lines[i][a].X < lines[i][b].X
		})
	}

	return lines
}

// groupLinesIntoBlocks groups lines into blocks based on vertical gaps
func (d *BlockDetector) groupLinesIntoBlocks(lines [][]text.TextFragment) []Block {
	if len(lines) == 0 {
		return nil
	}

	var blocks []Block
	var currentBlock Block
	currentBlock.Lines = [][]text.TextFragment{lines[0]}

	for i := 1; i < len(lines); i++ {
		prevLine := lines[i-1]
		currLine := lines[i]

		// Calculate vertical gap between lines
		prevLineY := lineMinY(prevLine)
		currLineY := lineMaxY(currLine)
		gap := prevLineY - currLineY // Distance between bottom of prev and top of curr

		// Calculate average line height for threshold
		avgHeight := (lineHeight(prevLine) + lineHeight(currLine)) / 2
		threshold := avgHeight * d.config.VerticalGapThreshold

		// Check horizontal overlap - lines must overlap horizontally to be in same block
		prevLeft, prevRight := lineXRange(prevLine)
		currLeft, currRight := lineXRange(currLine)
		hasHorizontalOverlap := prevRight > currLeft && currRight > prevLeft

		// Also check if there's a large horizontal gap (indentation change, etc.)
		horizontalGap := d.calculateHorizontalGap(prevLine, currLine)
		avgFontSize := averageLineHeight(prevLine)
		largeHorizontalGap := horizontalGap > avgFontSize*d.config.HorizontalGapThreshold

		// Start new block if:
		// 1. Vertical gap is larger than threshold, OR
		// 2. No horizontal overlap between lines, OR
		// 3. Large horizontal gap (suggesting different alignment)
		if gap > threshold || !hasHorizontalOverlap || largeHorizontalGap {
			// Finish current block
			currentBlock = d.finalizeBlock(currentBlock)
			blocks = append(blocks, currentBlock)

			// Start new block
			currentBlock = Block{
				Lines: [][]text.TextFragment{currLine},
			}
		} else {
			// Continue current block
			currentBlock.Lines = append(currentBlock.Lines, currLine)
		}
	}

	// Don't forget the last block
	currentBlock = d.finalizeBlock(currentBlock)
	blocks = append(blocks, currentBlock)

	return blocks
}

// finalizeBlock computes the bounding box and collects fragments for a block
func (d *BlockDetector) finalizeBlock(block Block) Block {
	if len(block.Lines) == 0 {
		return block
	}

	// Collect all fragments
	for _, line := range block.Lines {
		block.Fragments = append(block.Fragments, line...)
	}

	// Compute bounding box
	block.BBox = fragmentsBBox(block.Fragments)

	return block
}

// calculateHorizontalGap calculates the horizontal gap between two lines
// Returns 0 if lines overlap horizontally, otherwise the gap distance
func (d *BlockDetector) calculateHorizontalGap(line1, line2 []text.TextFragment) float64 {
	left1, right1 := lineXRange(line1)
	left2, right2 := lineXRange(line2)

	// Check for overlap
	if right1 > left2 && right2 > left1 {
		return 0 // Lines overlap
	}

	// Calculate gap
	if left2 > right1 {
		return left2 - right1
	}
	return left1 - right2
}

// mergeOverlappingBlocks merges blocks that significantly overlap
func (d *BlockDetector) mergeOverlappingBlocks(blocks []Block) []Block {
	if len(blocks) <= 1 {
		return blocks
	}

	merged := make([]Block, 0, len(blocks))
	used := make([]bool, len(blocks))

	for i := 0; i < len(blocks); i++ {
		if used[i] {
			continue
		}

		current := blocks[i]

		// Find all overlapping blocks
		for j := i + 1; j < len(blocks); j++ {
			if used[j] {
				continue
			}

			if d.blocksOverlap(current, blocks[j]) {
				// Merge blocks
				current = d.mergeBlocks(current, blocks[j])
				used[j] = true
			}
		}

		merged = append(merged, current)
	}

	return merged
}

// blocksOverlap returns true if two blocks significantly overlap
func (d *BlockDetector) blocksOverlap(b1, b2 Block) bool {
	// Calculate intersection
	left := max(b1.BBox.X, b2.BBox.X)
	right := min(b1.BBox.X+b1.BBox.Width, b2.BBox.X+b2.BBox.Width)
	bottom := max(b1.BBox.Y, b2.BBox.Y)
	top := min(b1.BBox.Y+b1.BBox.Height, b2.BBox.Y+b2.BBox.Height)

	if left >= right || bottom >= top {
		return false // No intersection
	}

	intersectionArea := (right - left) * (top - bottom)
	b1Area := b1.BBox.Width * b1.BBox.Height
	b2Area := b2.BBox.Width * b2.BBox.Height
	smallerArea := min(b1Area, b2Area)

	// Consider overlapping if intersection is > 30% of smaller block
	return intersectionArea > smallerArea*0.3
}

// mergeBlocks merges two blocks into one
func (d *BlockDetector) mergeBlocks(b1, b2 Block) Block {
	merged := Block{
		Fragments: append(b1.Fragments, b2.Fragments...),
		Lines:     append(b1.Lines, b2.Lines...),
	}

	// Sort lines by Y position
	sort.Slice(merged.Lines, func(i, j int) bool {
		return lineMaxY(merged.Lines[i]) > lineMaxY(merged.Lines[j])
	})

	merged.BBox = fragmentsBBox(merged.Fragments)
	return merged
}

// sortBlocksInReadingOrder sorts blocks in top-to-bottom, left-to-right order
func (d *BlockDetector) sortBlocksInReadingOrder(blocks []Block) []Block {
	if len(blocks) <= 1 {
		return blocks
	}

	sort.Slice(blocks, func(i, j int) bool {
		// First sort by Y (top to bottom)
		yDiff := blocks[i].BBox.Y + blocks[i].BBox.Height - (blocks[j].BBox.Y + blocks[j].BBox.Height)
		if absFloat64(yDiff) > 10 { // Tolerance for "same row"
			return yDiff > 0 // Higher Y (top of page) first
		}
		// Then by X (left to right)
		return blocks[i].BBox.X < blocks[j].BBox.X
	})

	// Assign indices
	for i := range blocks {
		blocks[i].Index = i
	}

	return blocks
}

// validateBlocks filters out invalid blocks
func (d *BlockDetector) validateBlocks(blocks []Block) []Block {
	var valid []Block

	for _, block := range blocks {
		// Skip empty blocks
		if len(block.Fragments) == 0 {
			continue
		}

		// Skip blocks that are too small
		if block.BBox.Width < d.config.MinBlockWidth ||
			block.BBox.Height < d.config.MinBlockHeight {
			continue
		}

		valid = append(valid, block)
	}

	// Re-assign indices
	for i := range valid {
		valid[i].Index = i
	}

	return valid
}

// Helper functions

// lineMinY returns the minimum Y of all fragments in a line (bottom)
func lineMinY(line []text.TextFragment) float64 {
	if len(line) == 0 {
		return 0
	}
	minY := line[0].Y
	for _, f := range line[1:] {
		if f.Y < minY {
			minY = f.Y
		}
	}
	return minY
}

// lineMaxY returns the maximum Y + height of all fragments in a line (top)
func lineMaxY(line []text.TextFragment) float64 {
	if len(line) == 0 {
		return 0
	}
	maxY := line[0].Y + line[0].Height
	for _, f := range line[1:] {
		top := f.Y + f.Height
		if top > maxY {
			maxY = top
		}
	}
	return maxY
}

// lineHeight returns the average height of fragments in a line
func lineHeight(line []text.TextFragment) float64 {
	if len(line) == 0 {
		return 0
	}
	total := 0.0
	for _, f := range line {
		total += f.Height
	}
	return total / float64(len(line))
}

// lineXRange returns the leftmost X and rightmost X of a line
func lineXRange(line []text.TextFragment) (float64, float64) {
	if len(line) == 0 {
		return 0, 0
	}
	left := line[0].X
	right := line[0].X + line[0].Width
	for _, f := range line[1:] {
		if f.X < left {
			left = f.X
		}
		if f.X+f.Width > right {
			right = f.X + f.Width
		}
	}
	return left, right
}

// averageLineHeight returns the average font size/height in a line
func averageLineHeight(line []text.TextFragment) float64 {
	if len(line) == 0 {
		return 12.0 // Default
	}
	total := 0.0
	for _, f := range line {
		total += f.FontSize
	}
	return total / float64(len(line))
}

// max returns the larger of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// min returns the smaller of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// BlockLayout methods

// BlockCount returns the number of detected blocks
func (l *BlockLayout) BlockCount() int {
	if l == nil {
		return 0
	}
	return len(l.Blocks)
}

// GetBlock returns a specific block by index
func (l *BlockLayout) GetBlock(index int) *Block {
	if l == nil || index < 0 || index >= len(l.Blocks) {
		return nil
	}
	return &l.Blocks[index]
}

// GetText returns all text in reading order (block by block)
func (l *BlockLayout) GetText() string {
	if l == nil || len(l.Blocks) == 0 {
		return ""
	}

	var result string
	for i, block := range l.Blocks {
		blockText := block.GetText()
		result += blockText

		// Add paragraph break between blocks
		if i < len(l.Blocks)-1 && len(blockText) > 0 {
			result += "\n\n"
		}
	}

	return result
}

// GetAllFragments returns all fragments in reading order
func (l *BlockLayout) GetAllFragments() []text.TextFragment {
	if l == nil {
		return nil
	}

	var result []text.TextFragment
	for _, block := range l.Blocks {
		result = append(result, block.Fragments...)
	}
	return result
}

// Block methods

// GetText returns the text content of this block
func (b *Block) GetText() string {
	if b == nil || len(b.Lines) == 0 {
		return ""
	}

	var result string
	for lineIdx, line := range b.Lines {
		// Assemble line text
		for i, frag := range line {
			if i > 0 {
				prevFrag := line[i-1]
				gap := frag.X - (prevFrag.X + prevFrag.Width)
				if gap > frag.Height*0.1 {
					result += " "
				}
			}
			result += frag.Text
		}

		// Add line break
		if lineIdx < len(b.Lines)-1 {
			result += "\n"
		}
	}

	return result
}

// LineCount returns the number of lines in this block
func (b *Block) LineCount() int {
	if b == nil {
		return 0
	}
	return len(b.Lines)
}

// FragmentCount returns the number of fragments in this block
func (b *Block) FragmentCount() int {
	if b == nil {
		return 0
	}
	return len(b.Fragments)
}

// AverageFontSize returns the average font size of fragments in this block
func (b *Block) AverageFontSize() float64 {
	if b == nil || len(b.Fragments) == 0 {
		return 0
	}

	total := 0.0
	for _, f := range b.Fragments {
		total += f.FontSize
	}
	return total / float64(len(b.Fragments))
}

// ContainsPoint returns true if the given point is within this block's bounding box
func (b *Block) ContainsPoint(x, y float64) bool {
	if b == nil {
		return false
	}
	return x >= b.BBox.X && x <= b.BBox.X+b.BBox.Width &&
		y >= b.BBox.Y && y <= b.BBox.Y+b.BBox.Height
}
