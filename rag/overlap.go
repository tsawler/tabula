package rag

import (
	"strings"
	"unicode"
)

// OverlapStrategy defines how overlap between chunks is computed
type OverlapStrategy int

const (
	// OverlapNone disables overlap between chunks
	OverlapNone OverlapStrategy = iota
	// OverlapCharacter uses character-based overlap (simple but can break words/sentences)
	OverlapCharacter
	// OverlapSentence uses sentence-based overlap (preserves complete sentences)
	OverlapSentence
	// OverlapParagraph uses paragraph-based overlap (preserves complete paragraphs)
	OverlapParagraph
)

// String returns a human-readable representation of the overlap strategy
func (os OverlapStrategy) String() string {
	switch os {
	case OverlapNone:
		return "none"
	case OverlapCharacter:
		return "character"
	case OverlapSentence:
		return "sentence"
	case OverlapParagraph:
		return "paragraph"
	default:
		return "unknown"
	}
}

// OverlapConfig holds configuration for chunk overlap
type OverlapConfig struct {
	// Strategy determines how overlap is computed
	Strategy OverlapStrategy

	// Size is the target overlap size in characters (for character-based)
	// or number of sentences/paragraphs (for sentence/paragraph-based)
	Size int

	// MinOverlap is the minimum overlap to include (avoids tiny overlaps)
	MinOverlap int

	// MaxOverlap is the maximum overlap allowed (prevents excessive duplication)
	MaxOverlap int

	// PreserveWords ensures character overlap doesn't break words
	PreserveWords bool

	// IncludeHeadingContext includes section heading in overlap for context
	IncludeHeadingContext bool
}

// DefaultOverlapConfig returns sensible defaults for overlap
func DefaultOverlapConfig() OverlapConfig {
	return OverlapConfig{
		Strategy:              OverlapSentence,
		Size:                  2, // 2 sentences for sentence-based
		MinOverlap:            20,
		MaxOverlap:            500,
		PreserveWords:         true,
		IncludeHeadingContext: false,
	}
}

// OverlapGenerator generates overlap content between chunks
type OverlapGenerator struct {
	config OverlapConfig
}

// NewOverlapGenerator creates a new overlap generator with default configuration
func NewOverlapGenerator() *OverlapGenerator {
	return &OverlapGenerator{
		config: DefaultOverlapConfig(),
	}
}

// NewOverlapGeneratorWithConfig creates an overlap generator with custom configuration
func NewOverlapGeneratorWithConfig(config OverlapConfig) *OverlapGenerator {
	return &OverlapGenerator{
		config: config,
	}
}

// OverlapResult contains the computed overlap text and metadata
type OverlapResult struct {
	// Text is the overlap content to prepend to the next chunk
	Text string

	// CharCount is the number of characters in the overlap
	CharCount int

	// SentenceCount is the number of complete sentences in the overlap
	SentenceCount int

	// Strategy is the strategy that was used
	Strategy OverlapStrategy
}

// GenerateOverlap extracts overlap content from the end of a chunk
func (og *OverlapGenerator) GenerateOverlap(chunkText string) *OverlapResult {
	if og.config.Strategy == OverlapNone || og.config.Size <= 0 {
		return &OverlapResult{Strategy: OverlapNone}
	}

	var overlap string
	var sentenceCount int

	switch og.config.Strategy {
	case OverlapCharacter:
		overlap = og.generateCharacterOverlap(chunkText)
	case OverlapSentence:
		overlap, sentenceCount = og.generateSentenceOverlap(chunkText)
	case OverlapParagraph:
		overlap, sentenceCount = og.generateParagraphOverlap(chunkText)
	default:
		return &OverlapResult{Strategy: og.config.Strategy}
	}

	// Apply min/max constraints
	if len(overlap) < og.config.MinOverlap {
		// Overlap too small, try to get more or skip
		if og.config.Strategy == OverlapSentence && sentenceCount == 0 {
			// No complete sentences, fall back to character overlap
			overlap = og.generateCharacterOverlap(chunkText)
		}
	}

	if len(overlap) > og.config.MaxOverlap {
		overlap = og.truncateOverlap(overlap)
	}

	return &OverlapResult{
		Text:          overlap,
		CharCount:     len(overlap),
		SentenceCount: sentenceCount,
		Strategy:      og.config.Strategy,
	}
}

// generateCharacterOverlap extracts character-based overlap from the end of text
func (og *OverlapGenerator) generateCharacterOverlap(text string) string {
	if len(text) <= og.config.Size {
		return text
	}

	// Start from target position
	start := len(text) - og.config.Size

	// If preserving words, find the next word boundary
	if og.config.PreserveWords {
		// Move forward to find start of a word
		for start < len(text) && !unicode.IsSpace(rune(text[start])) {
			start++
		}
		// Skip whitespace
		for start < len(text) && unicode.IsSpace(rune(text[start])) {
			start++
		}
	}

	if start >= len(text) {
		return ""
	}

	return strings.TrimSpace(text[start:])
}

// generateSentenceOverlap extracts sentence-based overlap from the end of text
func (og *OverlapGenerator) generateSentenceOverlap(text string) (string, int) {
	sentences := splitIntoSentencesWithPositions(text)

	if len(sentences) == 0 {
		return "", 0
	}

	// Get the last N sentences
	numSentences := og.config.Size
	if numSentences > len(sentences) {
		numSentences = len(sentences)
	}

	startIdx := len(sentences) - numSentences

	// Build overlap from selected sentences
	var overlap strings.Builder
	for i := startIdx; i < len(sentences); i++ {
		if overlap.Len() > 0 {
			overlap.WriteString(" ")
		}
		overlap.WriteString(sentences[i].text)
	}

	return strings.TrimSpace(overlap.String()), numSentences
}

// generateParagraphOverlap extracts paragraph-based overlap from the end of text
func (og *OverlapGenerator) generateParagraphOverlap(text string) (string, int) {
	paragraphs := splitIntoParagraphs(text)

	if len(paragraphs) == 0 {
		return "", 0
	}

	// Get the last N paragraphs
	numParagraphs := og.config.Size
	if numParagraphs > len(paragraphs) {
		numParagraphs = len(paragraphs)
	}

	startIdx := len(paragraphs) - numParagraphs

	// Build overlap from selected paragraphs
	var overlap strings.Builder
	sentenceCount := 0
	for i := startIdx; i < len(paragraphs); i++ {
		if overlap.Len() > 0 {
			overlap.WriteString("\n\n")
		}
		overlap.WriteString(paragraphs[i])
		// Count sentences in paragraph
		sentenceCount += len(splitIntoSentencesWithPositions(paragraphs[i]))
	}

	return strings.TrimSpace(overlap.String()), sentenceCount
}

// truncateOverlap reduces overlap to fit within MaxOverlap while preserving sentences
func (og *OverlapGenerator) truncateOverlap(overlap string) string {
	if len(overlap) <= og.config.MaxOverlap {
		return overlap
	}

	// Try to truncate at a sentence boundary
	sentences := splitIntoSentencesWithPositions(overlap)
	if len(sentences) == 0 {
		// No sentences, truncate at word boundary
		return og.generateCharacterOverlap(overlap[:og.config.MaxOverlap])
	}

	// Find how many sentences fit within MaxOverlap
	var result strings.Builder
	for _, s := range sentences {
		test := result.String()
		if result.Len() > 0 {
			test += " "
		}
		test += s.text

		if len(test) > og.config.MaxOverlap {
			break
		}

		if result.Len() > 0 {
			result.WriteString(" ")
		}
		result.WriteString(s.text)
	}

	if result.Len() == 0 {
		// First sentence exceeds max, truncate it
		return og.generateCharacterOverlap(overlap[:og.config.MaxOverlap])
	}

	return result.String()
}

// sentenceWithPosition holds a sentence and its position in the original text
type sentenceWithPosition struct {
	text  string
	start int
	end   int
}

// splitIntoSentencesWithPositions splits text into sentences with position tracking
func splitIntoSentencesWithPositions(text string) []sentenceWithPosition {
	var sentences []sentenceWithPosition
	var current strings.Builder
	start := 0

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		current.WriteRune(r)

		// Check for sentence ending
		if r == '.' || r == '!' || r == '?' {
			// Look ahead to verify this is a sentence end
			if isSentenceEndRune(runes, i) {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentenceWithPosition{
						text:  sentence,
						start: start,
						end:   i + 1,
					})
				}
				current.Reset()
				// Skip whitespace for next sentence start
				for i+1 < len(runes) && unicode.IsSpace(runes[i+1]) {
					i++
				}
				start = i + 1
			}
		}
	}

	// Add any remaining text as a sentence
	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		sentences = append(sentences, sentenceWithPosition{
			text:  remaining,
			start: start,
			end:   len(runes),
		})
	}

	return sentences
}

// isSentenceEndRune checks if the punctuation at position i is a sentence end
func isSentenceEndRune(runes []rune, i int) bool {
	if i >= len(runes) {
		return false
	}

	r := runes[i]
	if r != '.' && r != '!' && r != '?' {
		return false
	}

	// Check for abbreviations
	if r == '.' && i > 0 {
		// Single capital letter before period (like "Mr." or initials)
		if unicode.IsUpper(runes[i-1]) {
			if i < 2 || !unicode.IsLetter(runes[i-2]) {
				return false
			}
		}

		// Check for common abbreviations
		if isAbbreviationRune(runes, i) {
			return false
		}

		// Check for decimal numbers
		if unicode.IsDigit(runes[i-1]) && i+1 < len(runes) && unicode.IsDigit(runes[i+1]) {
			return false
		}
	}

	// End of text is a sentence end
	if i+1 >= len(runes) {
		return true
	}

	// Check if followed by space and capital letter
	if i+2 < len(runes) && unicode.IsSpace(runes[i+1]) {
		next := runes[i+2]
		if unicode.IsUpper(next) || next == '"' || next == '\'' {
			return true
		}
	}

	return false
}

// isAbbreviationRune checks if the period is part of an abbreviation
func isAbbreviationRune(runes []rune, i int) bool {
	// Get the word before the period
	start := i
	for start > 0 && unicode.IsLetter(runes[start-1]) {
		start--
	}

	if start >= i {
		return false
	}

	word := strings.ToLower(string(runes[start : i+1]))

	abbreviations := []string{
		"mr.", "mrs.", "ms.", "dr.", "prof.",
		"sr.", "jr.", "vs.", "etc.", "e.g.", "i.e.",
		"inc.", "ltd.", "co.", "corp.",
		"jan.", "feb.", "mar.", "apr.", "jun.", "jul.", "aug.", "sep.", "oct.", "nov.", "dec.",
		"st.", "rd.", "ave.", "blvd.",
		"no.", "vol.", "pp.", "pg.",
	}

	for _, abbr := range abbreviations {
		if word == abbr {
			return true
		}
	}

	return false
}

// splitIntoParagraphs splits text into paragraphs (separated by blank lines)
func splitIntoParagraphs(text string) []string {
	var paragraphs []string
	var current strings.Builder

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			// Blank line - end current paragraph
			if current.Len() > 0 {
				paragraphs = append(paragraphs, strings.TrimSpace(current.String()))
				current.Reset()
			}
		} else {
			if current.Len() > 0 {
				current.WriteString(" ")
			}
			current.WriteString(trimmed)
		}
	}

	// Add final paragraph
	if current.Len() > 0 {
		paragraphs = append(paragraphs, strings.TrimSpace(current.String()))
	}

	return paragraphs
}

// ApplyOverlap applies overlap from the previous chunk to the current chunk
func ApplyOverlap(currentText string, overlap *OverlapResult, sectionTitle string, includeContext bool) string {
	if overlap == nil || overlap.Text == "" {
		return currentText
	}

	var result strings.Builder

	// Optionally include section context
	if includeContext && sectionTitle != "" {
		result.WriteString("[")
		result.WriteString(sectionTitle)
		result.WriteString("]\n\n")
	}

	// Add overlap marker for clarity (optional)
	result.WriteString(overlap.Text)

	// Add separator between overlap and main content
	result.WriteString("\n\n")

	// Add main content
	result.WriteString(currentText)

	return result.String()
}

// ChunkWithOverlap represents a chunk with its overlap information
type ChunkWithOverlap struct {
	*Chunk

	// OverlapPrefix is the overlap content prepended from previous chunk
	OverlapPrefix string

	// OverlapSuffix is the overlap content that will be prepended to next chunk
	OverlapSuffix string

	// HasOverlapPrefix indicates if this chunk has overlap from previous
	HasOverlapPrefix bool

	// HasOverlapSuffix indicates if this chunk provides overlap to next
	HasOverlapSuffix bool
}

// ApplyOverlapToChunks adds overlap between consecutive chunks
func ApplyOverlapToChunks(chunks []*Chunk, config OverlapConfig) []*ChunkWithOverlap {
	if len(chunks) == 0 {
		return nil
	}

	generator := NewOverlapGeneratorWithConfig(config)
	result := make([]*ChunkWithOverlap, len(chunks))

	for i, chunk := range chunks {
		result[i] = &ChunkWithOverlap{
			Chunk: chunk,
		}

		if i > 0 && config.Strategy != OverlapNone {
			// Generate overlap from previous chunk
			prevChunk := chunks[i-1]
			overlap := generator.GenerateOverlap(prevChunk.Text)

			if overlap.Text != "" {
				result[i].OverlapPrefix = overlap.Text
				result[i].HasOverlapPrefix = true
				result[i-1].OverlapSuffix = overlap.Text
				result[i-1].HasOverlapSuffix = true

				// Update the chunk text to include overlap
				if config.IncludeHeadingContext {
					result[i].Chunk.Text = ApplyOverlap(chunk.Text, overlap, chunk.Metadata.SectionTitle, true)
				} else {
					result[i].Chunk.Text = ApplyOverlap(chunk.Text, overlap, "", false)
				}

				// Update metadata
				result[i].Chunk.Metadata.CharCount = len(result[i].Chunk.Text)
				result[i].Chunk.Metadata.WordCount = countWords(result[i].Chunk.Text)
				result[i].Chunk.Metadata.EstimatedTokens = len(result[i].Chunk.Text) / 4
			}
		}
	}

	return result
}

// GetOverlapText returns just the overlap portion of a chunk (for analysis)
func (c *ChunkWithOverlap) GetOverlapText() string {
	return c.OverlapPrefix
}

// GetOriginalText returns the chunk text without overlap prefix
func (c *ChunkWithOverlap) GetOriginalText() string {
	if !c.HasOverlapPrefix || c.OverlapPrefix == "" {
		return c.Text
	}

	// Remove overlap prefix from text
	idx := strings.Index(c.Text, c.OverlapPrefix)
	if idx == -1 {
		return c.Text
	}

	// Skip overlap and any separator
	remaining := c.Text[idx+len(c.OverlapPrefix):]
	return strings.TrimSpace(remaining)
}
