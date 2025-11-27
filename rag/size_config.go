package rag

import (
	"fmt"
	"strings"
)

// SizeUnit defines the unit of measurement for chunk sizes
type SizeUnit int

const (
	// SizeUnitCharacters measures size in characters
	SizeUnitCharacters SizeUnit = iota
	// SizeUnitTokens measures size in estimated tokens (chars/4)
	SizeUnitTokens
	// SizeUnitWords measures size in words
	SizeUnitWords
	// SizeUnitSentences measures size in sentences
	SizeUnitSentences
	// SizeUnitParagraphs measures size in paragraphs
	SizeUnitParagraphs
)

// String returns a human-readable representation of the size unit
func (su SizeUnit) String() string {
	switch su {
	case SizeUnitCharacters:
		return "characters"
	case SizeUnitTokens:
		return "tokens"
	case SizeUnitWords:
		return "words"
	case SizeUnitSentences:
		return "sentences"
	case SizeUnitParagraphs:
		return "paragraphs"
	default:
		return "unknown"
	}
}

// LimitType defines whether a limit is soft or hard
type LimitType int

const (
	// LimitTypeSoft is a preference - try not to exceed but allow if necessary
	LimitTypeSoft LimitType = iota
	// LimitTypeHard is a strict limit - must not exceed
	LimitTypeHard
)

// String returns a human-readable representation of the limit type
func (lt LimitType) String() string {
	switch lt {
	case LimitTypeSoft:
		return "soft"
	case LimitTypeHard:
		return "hard"
	default:
		return "unknown"
	}
}

// SizeLimit represents a size limit with its type and value
type SizeLimit struct {
	// Value is the limit value
	Value int

	// Unit is the unit of measurement
	Unit SizeUnit

	// Type determines if this is a soft or hard limit
	Type LimitType
}

// String returns a human-readable representation of the size limit
func (sl SizeLimit) String() string {
	return fmt.Sprintf("%d %s (%s)", sl.Value, sl.Unit.String(), sl.Type.String())
}

// SizeConfig holds comprehensive size configuration for chunking
type SizeConfig struct {
	// Target is the ideal chunk size to aim for
	Target SizeLimit

	// Min is the minimum chunk size
	Min SizeLimit

	// Max is the maximum chunk size
	Max SizeLimit

	// TokensPerChar is the ratio of tokens to characters (default: 0.25)
	// Used for token estimation
	TokensPerChar float64

	// AllowExceedForAtomicContent allows exceeding max for tables/lists
	AllowExceedForAtomicContent bool

	// MergeSmallChunks merges chunks below min with neighbors
	MergeSmallChunks bool

	// SplitAtSemanticBoundaries prefers semantic boundaries over exact sizes
	SplitAtSemanticBoundaries bool
}

// DefaultSizeConfig returns sensible defaults for size configuration
func DefaultSizeConfig() SizeConfig {
	return SizeConfig{
		Target: SizeLimit{
			Value: 1000,
			Unit:  SizeUnitCharacters,
			Type:  LimitTypeSoft,
		},
		Min: SizeLimit{
			Value: 100,
			Unit:  SizeUnitCharacters,
			Type:  LimitTypeSoft,
		},
		Max: SizeLimit{
			Value: 2000,
			Unit:  SizeUnitCharacters,
			Type:  LimitTypeHard,
		},
		TokensPerChar:               0.25,
		AllowExceedForAtomicContent: true,
		MergeSmallChunks:            true,
		SplitAtSemanticBoundaries:   true,
	}
}

// TokenBasedSizeConfig returns configuration optimized for token-based chunking
func TokenBasedSizeConfig(targetTokens, maxTokens int) SizeConfig {
	return SizeConfig{
		Target: SizeLimit{
			Value: targetTokens,
			Unit:  SizeUnitTokens,
			Type:  LimitTypeSoft,
		},
		Min: SizeLimit{
			Value: targetTokens / 10,
			Unit:  SizeUnitTokens,
			Type:  LimitTypeSoft,
		},
		Max: SizeLimit{
			Value: maxTokens,
			Unit:  SizeUnitTokens,
			Type:  LimitTypeHard,
		},
		TokensPerChar:               0.25,
		AllowExceedForAtomicContent: true,
		MergeSmallChunks:            true,
		SplitAtSemanticBoundaries:   true,
	}
}

// SemanticSizeConfig returns configuration for semantic unit-based chunking
func SemanticSizeConfig(targetParagraphs, maxParagraphs int) SizeConfig {
	return SizeConfig{
		Target: SizeLimit{
			Value: targetParagraphs,
			Unit:  SizeUnitParagraphs,
			Type:  LimitTypeSoft,
		},
		Min: SizeLimit{
			Value: 1,
			Unit:  SizeUnitParagraphs,
			Type:  LimitTypeSoft,
		},
		Max: SizeLimit{
			Value: maxParagraphs,
			Unit:  SizeUnitParagraphs,
			Type:  LimitTypeHard,
		},
		TokensPerChar:               0.25,
		AllowExceedForAtomicContent: true,
		MergeSmallChunks:            false, // Don't merge semantic units
		SplitAtSemanticBoundaries:   true,
	}
}

// SizeCalculator calculates various size metrics for text
type SizeCalculator struct {
	config SizeConfig
}

// NewSizeCalculator creates a new size calculator with default config
func NewSizeCalculator() *SizeCalculator {
	return &SizeCalculator{
		config: DefaultSizeConfig(),
	}
}

// NewSizeCalculatorWithConfig creates a size calculator with custom config
func NewSizeCalculatorWithConfig(config SizeConfig) *SizeCalculator {
	return &SizeCalculator{
		config: config,
	}
}

// SizeMetrics holds all size measurements for a piece of text
type SizeMetrics struct {
	Characters int
	Tokens     int
	Words      int
	Sentences  int
	Paragraphs int
}

// Calculate computes all size metrics for the given text
func (sc *SizeCalculator) Calculate(text string) SizeMetrics {
	return SizeMetrics{
		Characters: len(text),
		Tokens:     sc.EstimateTokens(text),
		Words:      countWords(text),
		Sentences:  countSentences(text),
		Paragraphs: countParagraphs(text),
	}
}

// GetSize returns the size in the specified unit
func (sc *SizeCalculator) GetSize(text string, unit SizeUnit) int {
	metrics := sc.Calculate(text)
	return metrics.GetByUnit(unit)
}

// GetByUnit returns the metric value for the specified unit
func (m SizeMetrics) GetByUnit(unit SizeUnit) int {
	switch unit {
	case SizeUnitCharacters:
		return m.Characters
	case SizeUnitTokens:
		return m.Tokens
	case SizeUnitWords:
		return m.Words
	case SizeUnitSentences:
		return m.Sentences
	case SizeUnitParagraphs:
		return m.Paragraphs
	default:
		return m.Characters
	}
}

// EstimateTokens estimates token count using the configured ratio
func (sc *SizeCalculator) EstimateTokens(text string) int {
	ratio := sc.config.TokensPerChar
	if ratio <= 0 {
		ratio = 0.25
	}
	return int(float64(len(text)) * ratio)
}

// IsWithinTarget checks if size is within target range
func (sc *SizeCalculator) IsWithinTarget(text string) bool {
	size := sc.GetSize(text, sc.config.Target.Unit)
	target := sc.config.Target.Value

	// Allow 20% variance for soft targets
	if sc.config.Target.Type == LimitTypeSoft {
		minTarget := int(float64(target) * 0.8)
		maxTarget := int(float64(target) * 1.2)
		return size >= minTarget && size <= maxTarget
	}

	return size == target
}

// IsBelowMin checks if text is below minimum size
func (sc *SizeCalculator) IsBelowMin(text string) bool {
	size := sc.GetSize(text, sc.config.Min.Unit)
	return size < sc.config.Min.Value
}

// IsAboveMax checks if text exceeds maximum size
func (sc *SizeCalculator) IsAboveMax(text string) bool {
	size := sc.GetSize(text, sc.config.Max.Unit)
	return size > sc.config.Max.Value
}

// ExceedsLimit checks if text exceeds a specific limit
func (sc *SizeCalculator) ExceedsLimit(text string, limit SizeLimit) bool {
	size := sc.GetSize(text, limit.Unit)
	return size > limit.Value
}

// SizeCheckResult contains the result of a size check
type SizeCheckResult struct {
	// Metrics are the calculated size metrics
	Metrics SizeMetrics

	// IsValid indicates if the size is acceptable
	IsValid bool

	// Reason explains why the size is not valid (if applicable)
	Reason string

	// SuggestedAction suggests what to do if size is not valid
	SuggestedAction SizeAction

	// TargetDiff is the difference from target size
	TargetDiff int
}

// SizeAction suggests what action to take for size issues
type SizeAction int

const (
	// SizeActionNone - no action needed
	SizeActionNone SizeAction = iota
	// SizeActionSplit - chunk should be split
	SizeActionSplit
	// SizeActionMerge - chunk should be merged with neighbor
	SizeActionMerge
	// SizeActionTruncate - chunk must be truncated (hard limit exceeded)
	SizeActionTruncate
)

// String returns a human-readable representation of the size action
func (sa SizeAction) String() string {
	switch sa {
	case SizeActionNone:
		return "none"
	case SizeActionSplit:
		return "split"
	case SizeActionMerge:
		return "merge"
	case SizeActionTruncate:
		return "truncate"
	default:
		return "unknown"
	}
}

// Check performs a comprehensive size check on the text
func (sc *SizeCalculator) Check(text string) SizeCheckResult {
	metrics := sc.Calculate(text)
	result := SizeCheckResult{
		Metrics: metrics,
		IsValid: true,
	}

	targetSize := metrics.GetByUnit(sc.config.Target.Unit)
	result.TargetDiff = targetSize - sc.config.Target.Value

	// Check hard max limit
	maxSize := metrics.GetByUnit(sc.config.Max.Unit)
	if maxSize > sc.config.Max.Value && sc.config.Max.Type == LimitTypeHard {
		result.IsValid = false
		result.Reason = fmt.Sprintf("exceeds hard max limit of %d %s (actual: %d)",
			sc.config.Max.Value, sc.config.Max.Unit.String(), maxSize)
		result.SuggestedAction = SizeActionTruncate
		return result
	}

	// Check soft max limit
	if maxSize > sc.config.Max.Value && sc.config.Max.Type == LimitTypeSoft {
		result.IsValid = false
		result.Reason = fmt.Sprintf("exceeds soft max limit of %d %s (actual: %d)",
			sc.config.Max.Value, sc.config.Max.Unit.String(), maxSize)
		result.SuggestedAction = SizeActionSplit
		return result
	}

	// Check min limit
	minSize := metrics.GetByUnit(sc.config.Min.Unit)
	if minSize < sc.config.Min.Value {
		if sc.config.Min.Type == LimitTypeHard {
			result.IsValid = false
			result.Reason = fmt.Sprintf("below hard min limit of %d %s (actual: %d)",
				sc.config.Min.Value, sc.config.Min.Unit.String(), minSize)
		} else {
			result.Reason = fmt.Sprintf("below soft min limit of %d %s (actual: %d)",
				sc.config.Min.Value, sc.config.Min.Unit.String(), minSize)
		}
		result.SuggestedAction = SizeActionMerge
		return result
	}

	return result
}

// FindSplitPoint finds the best position to split text to meet size constraints
func (sc *SizeCalculator) FindSplitPoint(text string, boundaries []Boundary) int {
	return sc.FindSplitPointAt(text, boundaries, sc.config.Target.Value, sc.config.Target.Unit)
}

// FindSplitPointAt finds the best position to split text at a specific size limit
func (sc *SizeCalculator) FindSplitPointAt(text string, boundaries []Boundary, targetSize int, targetUnit SizeUnit) int {
	// Convert target to character position estimate
	var targetPos int
	switch targetUnit {
	case SizeUnitCharacters:
		targetPos = targetSize
	case SizeUnitTokens:
		targetPos = int(float64(targetSize) / sc.config.TokensPerChar)
	case SizeUnitWords:
		targetPos = targetSize * 6 // Rough estimate: 6 chars per word
	case SizeUnitSentences:
		targetPos = targetSize * 80 // Rough estimate: 80 chars per sentence
	case SizeUnitParagraphs:
		targetPos = targetSize * 400 // Rough estimate: 400 chars per paragraph
	default:
		targetPos = targetSize
	}

	if targetPos >= len(text) {
		return len(text)
	}

	// Find the best boundary near the target position
	if sc.config.SplitAtSemanticBoundaries && len(boundaries) > 0 {
		bestBoundary := findBestBoundaryNear(boundaries, targetPos, targetPos/4)
		if bestBoundary != nil {
			return bestBoundary.Position
		}
	}

	// Fall back to finding a sentence boundary
	return findSentenceEndNear(text, targetPos)
}

// findBestBoundaryNear finds the highest-scored boundary within tolerance of position
func findBestBoundaryNear(boundaries []Boundary, position, tolerance int) *Boundary {
	var best *Boundary
	bestScore := -1

	minPos := position - tolerance
	maxPos := position + tolerance

	if minPos < 0 {
		minPos = 0
	}

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

// findSentenceEndNear finds a sentence ending near the target position
func findSentenceEndNear(text string, targetPos int) int {
	if targetPos >= len(text) {
		return len(text)
	}

	// Look backwards for sentence end
	for i := targetPos; i >= 0 && i > targetPos-100; i-- {
		if i < len(text) && isSentenceEndChar(text[i]) {
			// Verify it's a real sentence end
			if i+1 < len(text) && (text[i+1] == ' ' || text[i+1] == '\n') {
				return i + 1
			}
		}
	}

	// Look forwards for sentence end
	for i := targetPos; i < len(text) && i < targetPos+100; i++ {
		if isSentenceEndChar(text[i]) {
			if i+1 < len(text) && (text[i+1] == ' ' || text[i+1] == '\n') {
				return i + 1
			}
			if i+1 >= len(text) {
				return i + 1
			}
		}
	}

	// Fall back to word boundary
	return findWordBoundaryNear(text, targetPos)
}

// findWordBoundaryNear finds a word boundary near the target position
func findWordBoundaryNear(text string, targetPos int) int {
	if targetPos >= len(text) {
		return len(text)
	}

	// Look for space before target
	for i := targetPos; i >= 0 && i > targetPos-50; i-- {
		if text[i] == ' ' || text[i] == '\n' {
			return i + 1
		}
	}

	// Look for space after target
	for i := targetPos; i < len(text) && i < targetPos+50; i++ {
		if text[i] == ' ' || text[i] == '\n' {
			return i + 1
		}
	}

	return targetPos
}

// isSentenceEndChar checks if a character typically ends a sentence
func isSentenceEndChar(c byte) bool {
	return c == '.' || c == '!' || c == '?'
}

// SplitToSize splits text into chunks that meet size constraints
func (sc *SizeCalculator) SplitToSize(text string, boundaries []Boundary) []string {
	var chunks []string

	remaining := text
	for len(remaining) > 0 {
		// Check if remaining text fits within max
		if !sc.IsAboveMax(remaining) {
			chunks = append(chunks, remaining)
			break
		}

		// Find split point using max limit (not target) to ensure chunks fit
		splitPos := sc.FindSplitPointAt(remaining, boundaries, sc.config.Max.Value, sc.config.Max.Unit)
		if splitPos <= 0 || splitPos >= len(remaining) {
			// Can't split further, add remaining as-is
			chunks = append(chunks, remaining)
			break
		}

		chunk := strings.TrimSpace(remaining[:splitPos])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		remaining = strings.TrimSpace(remaining[splitPos:])

		// Update boundary positions for remaining text
		boundaries = adjustBoundaryPositions(boundaries, splitPos)
	}

	return chunks
}

// adjustBoundaryPositions adjusts boundary positions after a split
func adjustBoundaryPositions(boundaries []Boundary, offset int) []Boundary {
	var adjusted []Boundary
	for _, b := range boundaries {
		if b.Position > offset {
			adjusted = append(adjusted, Boundary{
				Type:         b.Type,
				Position:     b.Position - offset,
				Score:        b.Score,
				ElementIndex: b.ElementIndex,
				Context:      b.Context,
			})
		}
	}
	return adjusted
}

// Helper functions for counting

// countSentences counts the number of sentences in text
func countSentences(text string) int {
	if text == "" {
		return 0
	}

	count := 0
	inSentence := false

	for i := 0; i < len(text); i++ {
		c := text[i]

		// Start of sentence
		if !inSentence && c != ' ' && c != '\n' && c != '\t' {
			inSentence = true
		}

		// End of sentence
		if inSentence && (c == '.' || c == '!' || c == '?') {
			// Check it's not an abbreviation
			if i > 0 && i+1 < len(text) {
				next := text[i+1]
				if next == ' ' || next == '\n' || next == '\t' {
					count++
					inSentence = false
				}
			} else if i+1 >= len(text) {
				count++
				inSentence = false
			}
		}
	}

	// Count incomplete sentence at end
	if inSentence {
		count++
	}

	return count
}

// countParagraphs counts the number of paragraphs in text
func countParagraphs(text string) int {
	if text == "" {
		return 0
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}

	// Split by double newlines
	parts := strings.Split(text, "\n\n")
	count := 0
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			count++
		}
	}

	if count == 0 {
		return 1 // At least one paragraph if there's text
	}

	return count
}

// ConvertSize converts a size value from one unit to another (approximate)
func ConvertSize(value int, from, to SizeUnit) int {
	// First convert to characters
	var chars int
	switch from {
	case SizeUnitCharacters:
		chars = value
	case SizeUnitTokens:
		chars = value * 4
	case SizeUnitWords:
		chars = value * 6
	case SizeUnitSentences:
		chars = value * 80
	case SizeUnitParagraphs:
		chars = value * 400
	default:
		chars = value
	}

	// Then convert to target unit
	switch to {
	case SizeUnitCharacters:
		return chars
	case SizeUnitTokens:
		return chars / 4
	case SizeUnitWords:
		return chars / 6
	case SizeUnitSentences:
		return chars / 80
	case SizeUnitParagraphs:
		return chars / 400
	default:
		return chars
	}
}

// PresetConfigs provides common preset configurations

// SmallChunkConfig returns config for small chunks (good for precise retrieval)
func SmallChunkConfig() SizeConfig {
	return SizeConfig{
		Target: SizeLimit{Value: 500, Unit: SizeUnitCharacters, Type: LimitTypeSoft},
		Min:    SizeLimit{Value: 100, Unit: SizeUnitCharacters, Type: LimitTypeSoft},
		Max:    SizeLimit{Value: 800, Unit: SizeUnitCharacters, Type: LimitTypeHard},
		TokensPerChar:               0.25,
		AllowExceedForAtomicContent: true,
		MergeSmallChunks:            true,
		SplitAtSemanticBoundaries:   true,
	}
}

// MediumChunkConfig returns config for medium chunks (balanced)
func MediumChunkConfig() SizeConfig {
	return DefaultSizeConfig()
}

// LargeChunkConfig returns config for large chunks (good for context)
func LargeChunkConfig() SizeConfig {
	return SizeConfig{
		Target: SizeLimit{Value: 2000, Unit: SizeUnitCharacters, Type: LimitTypeSoft},
		Min:    SizeLimit{Value: 500, Unit: SizeUnitCharacters, Type: LimitTypeSoft},
		Max:    SizeLimit{Value: 4000, Unit: SizeUnitCharacters, Type: LimitTypeHard},
		TokensPerChar:               0.25,
		AllowExceedForAtomicContent: true,
		MergeSmallChunks:            true,
		SplitAtSemanticBoundaries:   true,
	}
}

// OpenAIEmbeddingConfig returns config optimized for OpenAI embeddings (8191 tokens max)
func OpenAIEmbeddingConfig() SizeConfig {
	return TokenBasedSizeConfig(512, 8000)
}

// CohereEmbeddingConfig returns config optimized for Cohere embeddings
func CohereEmbeddingConfig() SizeConfig {
	return TokenBasedSizeConfig(256, 512)
}

// ClaudeContextConfig returns config for Claude's context window
func ClaudeContextConfig() SizeConfig {
	return TokenBasedSizeConfig(2000, 8000)
}
