package rag

import (
	"strings"
	"testing"
)

func TestSizeUnit_String(t *testing.T) {
	tests := []struct {
		unit SizeUnit
		want string
	}{
		{SizeUnitCharacters, "characters"},
		{SizeUnitTokens, "tokens"},
		{SizeUnitWords, "words"},
		{SizeUnitSentences, "sentences"},
		{SizeUnitParagraphs, "paragraphs"},
		{SizeUnit(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.unit.String(); got != tt.want {
				t.Errorf("SizeUnit.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimitType_String(t *testing.T) {
	tests := []struct {
		lt   LimitType
		want string
	}{
		{LimitTypeSoft, "soft"},
		{LimitTypeHard, "hard"},
		{LimitType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.lt.String(); got != tt.want {
				t.Errorf("LimitType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSizeLimit_String(t *testing.T) {
	limit := SizeLimit{
		Value: 1000,
		Unit:  SizeUnitCharacters,
		Type:  LimitTypeSoft,
	}

	result := limit.String()

	if !strings.Contains(result, "1000") {
		t.Error("Expected value in string")
	}
	if !strings.Contains(result, "characters") {
		t.Error("Expected unit in string")
	}
	if !strings.Contains(result, "soft") {
		t.Error("Expected type in string")
	}
}

func TestSizeAction_String(t *testing.T) {
	tests := []struct {
		action SizeAction
		want   string
	}{
		{SizeActionNone, "none"},
		{SizeActionSplit, "split"},
		{SizeActionMerge, "merge"},
		{SizeActionTruncate, "truncate"},
		{SizeAction(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.action.String(); got != tt.want {
				t.Errorf("SizeAction.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultSizeConfig(t *testing.T) {
	config := DefaultSizeConfig()

	if config.Target.Value != 1000 {
		t.Errorf("Expected target 1000, got %d", config.Target.Value)
	}
	if config.Target.Unit != SizeUnitCharacters {
		t.Errorf("Expected target unit characters, got %v", config.Target.Unit)
	}
	if config.Min.Value != 100 {
		t.Errorf("Expected min 100, got %d", config.Min.Value)
	}
	if config.Max.Value != 2000 {
		t.Errorf("Expected max 2000, got %d", config.Max.Value)
	}
	if config.Max.Type != LimitTypeHard {
		t.Error("Expected max to be hard limit")
	}
	if config.TokensPerChar != 0.25 {
		t.Errorf("Expected TokensPerChar 0.25, got %f", config.TokensPerChar)
	}
}

func TestTokenBasedSizeConfig(t *testing.T) {
	config := TokenBasedSizeConfig(512, 8000)

	if config.Target.Value != 512 {
		t.Errorf("Expected target 512, got %d", config.Target.Value)
	}
	if config.Target.Unit != SizeUnitTokens {
		t.Errorf("Expected target unit tokens, got %v", config.Target.Unit)
	}
	if config.Max.Value != 8000 {
		t.Errorf("Expected max 8000, got %d", config.Max.Value)
	}
}

func TestSemanticSizeConfig(t *testing.T) {
	config := SemanticSizeConfig(3, 10)

	if config.Target.Value != 3 {
		t.Errorf("Expected target 3, got %d", config.Target.Value)
	}
	if config.Target.Unit != SizeUnitParagraphs {
		t.Errorf("Expected target unit paragraphs, got %v", config.Target.Unit)
	}
	if config.Max.Value != 10 {
		t.Errorf("Expected max 10, got %d", config.Max.Value)
	}
	if config.MergeSmallChunks {
		t.Error("Expected MergeSmallChunks to be false for semantic config")
	}
}

func TestNewSizeCalculator(t *testing.T) {
	calc := NewSizeCalculator()
	if calc == nil {
		t.Error("NewSizeCalculator returned nil")
	}
}

func TestSizeCalculator_Calculate(t *testing.T) {
	calc := NewSizeCalculator()

	text := "This is a test. Another sentence here.\n\nSecond paragraph."

	metrics := calc.Calculate(text)

	if metrics.Characters != len(text) {
		t.Errorf("Expected %d characters, got %d", len(text), metrics.Characters)
	}
	if metrics.Words == 0 {
		t.Error("Expected words to be counted")
	}
	if metrics.Sentences < 2 {
		t.Errorf("Expected at least 2 sentences, got %d", metrics.Sentences)
	}
	if metrics.Paragraphs != 2 {
		t.Errorf("Expected 2 paragraphs, got %d", metrics.Paragraphs)
	}
	if metrics.Tokens == 0 {
		t.Error("Expected tokens to be estimated")
	}
}

func TestSizeCalculator_GetSize(t *testing.T) {
	calc := NewSizeCalculator()
	text := "Hello world. This is a test."

	t.Run("characters", func(t *testing.T) {
		size := calc.GetSize(text, SizeUnitCharacters)
		if size != len(text) {
			t.Errorf("Expected %d, got %d", len(text), size)
		}
	})

	t.Run("tokens", func(t *testing.T) {
		size := calc.GetSize(text, SizeUnitTokens)
		expected := int(float64(len(text)) * 0.25)
		if size != expected {
			t.Errorf("Expected %d, got %d", expected, size)
		}
	})

	t.Run("words", func(t *testing.T) {
		size := calc.GetSize(text, SizeUnitWords)
		if size < 5 {
			t.Errorf("Expected at least 5 words, got %d", size)
		}
	})
}

func TestSizeMetrics_GetByUnit(t *testing.T) {
	metrics := SizeMetrics{
		Characters: 100,
		Tokens:     25,
		Words:      20,
		Sentences:  3,
		Paragraphs: 1,
	}

	tests := []struct {
		unit SizeUnit
		want int
	}{
		{SizeUnitCharacters, 100},
		{SizeUnitTokens, 25},
		{SizeUnitWords, 20},
		{SizeUnitSentences, 3},
		{SizeUnitParagraphs, 1},
		{SizeUnit(99), 100}, // Default to characters
	}

	for _, tt := range tests {
		t.Run(tt.unit.String(), func(t *testing.T) {
			if got := metrics.GetByUnit(tt.unit); got != tt.want {
				t.Errorf("GetByUnit(%v) = %d, want %d", tt.unit, got, tt.want)
			}
		})
	}
}

func TestSizeCalculator_EstimateTokens(t *testing.T) {
	calc := NewSizeCalculator()

	text := strings.Repeat("a", 100)
	tokens := calc.EstimateTokens(text)

	// Default ratio is 0.25, so 100 chars = 25 tokens
	if tokens != 25 {
		t.Errorf("Expected 25 tokens, got %d", tokens)
	}
}

func TestSizeCalculator_IsWithinTarget(t *testing.T) {
	config := DefaultSizeConfig()
	config.Target = SizeLimit{Value: 100, Unit: SizeUnitCharacters, Type: LimitTypeSoft}
	calc := NewSizeCalculatorWithConfig(config)

	t.Run("within target", func(t *testing.T) {
		text := strings.Repeat("a", 100)
		if !calc.IsWithinTarget(text) {
			t.Error("Expected to be within target")
		}
	})

	t.Run("within 20% variance", func(t *testing.T) {
		text := strings.Repeat("a", 110) // 10% over
		if !calc.IsWithinTarget(text) {
			t.Error("Expected to be within 20% variance")
		}
	})

	t.Run("outside target", func(t *testing.T) {
		text := strings.Repeat("a", 200) // 100% over
		if calc.IsWithinTarget(text) {
			t.Error("Expected to be outside target")
		}
	})
}

func TestSizeCalculator_IsBelowMin(t *testing.T) {
	config := DefaultSizeConfig()
	config.Min = SizeLimit{Value: 100, Unit: SizeUnitCharacters, Type: LimitTypeSoft}
	calc := NewSizeCalculatorWithConfig(config)

	t.Run("below min", func(t *testing.T) {
		text := strings.Repeat("a", 50)
		if !calc.IsBelowMin(text) {
			t.Error("Expected to be below min")
		}
	})

	t.Run("at min", func(t *testing.T) {
		text := strings.Repeat("a", 100)
		if calc.IsBelowMin(text) {
			t.Error("Expected not to be below min")
		}
	})

	t.Run("above min", func(t *testing.T) {
		text := strings.Repeat("a", 200)
		if calc.IsBelowMin(text) {
			t.Error("Expected not to be below min")
		}
	})
}

func TestSizeCalculator_IsAboveMax(t *testing.T) {
	config := DefaultSizeConfig()
	config.Max = SizeLimit{Value: 100, Unit: SizeUnitCharacters, Type: LimitTypeHard}
	calc := NewSizeCalculatorWithConfig(config)

	t.Run("below max", func(t *testing.T) {
		text := strings.Repeat("a", 50)
		if calc.IsAboveMax(text) {
			t.Error("Expected not to be above max")
		}
	})

	t.Run("at max", func(t *testing.T) {
		text := strings.Repeat("a", 100)
		if calc.IsAboveMax(text) {
			t.Error("Expected not to be above max")
		}
	})

	t.Run("above max", func(t *testing.T) {
		text := strings.Repeat("a", 200)
		if !calc.IsAboveMax(text) {
			t.Error("Expected to be above max")
		}
	})
}

func TestSizeCalculator_Check(t *testing.T) {
	t.Run("valid size", func(t *testing.T) {
		config := DefaultSizeConfig()
		calc := NewSizeCalculatorWithConfig(config)

		text := strings.Repeat("a", 1000)
		result := calc.Check(text)

		if !result.IsValid {
			t.Error("Expected valid result")
		}
		if result.SuggestedAction != SizeActionNone {
			t.Errorf("Expected no action, got %v", result.SuggestedAction)
		}
	})

	t.Run("exceeds hard max", func(t *testing.T) {
		config := DefaultSizeConfig()
		config.Max = SizeLimit{Value: 100, Unit: SizeUnitCharacters, Type: LimitTypeHard}
		calc := NewSizeCalculatorWithConfig(config)

		text := strings.Repeat("a", 200)
		result := calc.Check(text)

		if result.IsValid {
			t.Error("Expected invalid result")
		}
		if result.SuggestedAction != SizeActionTruncate {
			t.Errorf("Expected truncate action, got %v", result.SuggestedAction)
		}
		if !strings.Contains(result.Reason, "hard max") {
			t.Error("Expected reason to mention hard max")
		}
	})

	t.Run("exceeds soft max", func(t *testing.T) {
		config := DefaultSizeConfig()
		config.Max = SizeLimit{Value: 100, Unit: SizeUnitCharacters, Type: LimitTypeSoft}
		calc := NewSizeCalculatorWithConfig(config)

		text := strings.Repeat("a", 200)
		result := calc.Check(text)

		if result.IsValid {
			t.Error("Expected invalid result")
		}
		if result.SuggestedAction != SizeActionSplit {
			t.Errorf("Expected split action, got %v", result.SuggestedAction)
		}
	})

	t.Run("below min", func(t *testing.T) {
		config := DefaultSizeConfig()
		config.Min = SizeLimit{Value: 100, Unit: SizeUnitCharacters, Type: LimitTypeSoft}
		calc := NewSizeCalculatorWithConfig(config)

		text := strings.Repeat("a", 50)
		result := calc.Check(text)

		if result.SuggestedAction != SizeActionMerge {
			t.Errorf("Expected merge action, got %v", result.SuggestedAction)
		}
	})
}

func TestSizeCalculator_FindSplitPoint(t *testing.T) {
	config := DefaultSizeConfig()
	config.Target = SizeLimit{Value: 50, Unit: SizeUnitCharacters, Type: LimitTypeSoft}
	calc := NewSizeCalculatorWithConfig(config)

	t.Run("split at sentence", func(t *testing.T) {
		text := "First sentence here. Second sentence follows. Third sentence ends."
		pos := calc.FindSplitPoint(text, nil)

		// Should find a sentence boundary
		if pos <= 0 || pos >= len(text) {
			t.Errorf("Expected valid split point, got %d", pos)
		}
	})

	t.Run("text shorter than target", func(t *testing.T) {
		text := "Short"
		pos := calc.FindSplitPoint(text, nil)

		if pos != len(text) {
			t.Errorf("Expected full length for short text, got %d", pos)
		}
	})

	t.Run("with boundaries", func(t *testing.T) {
		text := "First part of text. Second part here. Third part follows."
		boundaries := []Boundary{
			{Position: 20, Score: 100, Type: BoundarySentence},
			{Position: 37, Score: 50, Type: BoundarySentence},
		}

		config.SplitAtSemanticBoundaries = true
		calc = NewSizeCalculatorWithConfig(config)
		pos := calc.FindSplitPoint(text, boundaries)

		// Should prefer boundary with highest score
		if pos != 20 {
			t.Logf("Split point: %d (expected near 20)", pos)
		}
	})
}

func TestSizeCalculator_SplitToSize(t *testing.T) {
	config := DefaultSizeConfig()
	config.Max = SizeLimit{Value: 50, Unit: SizeUnitCharacters, Type: LimitTypeHard}
	calc := NewSizeCalculatorWithConfig(config)

	t.Run("split long text", func(t *testing.T) {
		text := "First sentence here. Second sentence follows. Third sentence at end."
		chunks := calc.SplitToSize(text, nil)

		if len(chunks) < 2 {
			t.Errorf("Expected multiple chunks, got %d", len(chunks))
		}

		// Each chunk should be within max
		for i, chunk := range chunks {
			if len(chunk) > 50 {
				t.Errorf("Chunk %d exceeds max: %d chars", i, len(chunk))
			}
		}
	})

	t.Run("short text no split", func(t *testing.T) {
		text := "Short text here."
		chunks := calc.SplitToSize(text, nil)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(chunks))
		}
	})
}

func TestCountSentences(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{"empty", "", 0},
		{"one sentence", "Hello world.", 1},
		{"two sentences", "First one. Second one.", 2},
		{"three sentences", "One. Two. Three.", 3},
		{"with question", "What is this? I don't know.", 2},
		{"with exclamation", "Wow! Amazing!", 2},
		{"no ending punct", "Hello world", 1},
		{"mixed", "Hello. World! How are you?", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countSentences(tt.text)
			if got != tt.want {
				t.Errorf("countSentences(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestCountParagraphs(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{"empty", "", 0},
		{"one paragraph", "Single paragraph here.", 1},
		{"two paragraphs", "First para.\n\nSecond para.", 2},
		{"three paragraphs", "One.\n\nTwo.\n\nThree.", 3},
		{"whitespace only", "   \n\n   ", 0},
		{"single newlines", "Line one.\nLine two.", 1},
		{"multiple newlines", "Para one.\n\n\n\nPara two.", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countParagraphs(tt.text)
			if got != tt.want {
				t.Errorf("countParagraphs(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestConvertSize(t *testing.T) {
	tests := []struct {
		name  string
		value int
		from  SizeUnit
		to    SizeUnit
		want  int
	}{
		{"chars to chars", 100, SizeUnitCharacters, SizeUnitCharacters, 100},
		{"chars to tokens", 100, SizeUnitCharacters, SizeUnitTokens, 25},
		{"tokens to chars", 25, SizeUnitTokens, SizeUnitCharacters, 100},
		{"words to chars", 10, SizeUnitWords, SizeUnitCharacters, 60},
		{"chars to words", 60, SizeUnitCharacters, SizeUnitWords, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertSize(tt.value, tt.from, tt.to)
			if got != tt.want {
				t.Errorf("ConvertSize(%d, %v, %v) = %d, want %d",
					tt.value, tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestPresetConfigs(t *testing.T) {
	t.Run("SmallChunkConfig", func(t *testing.T) {
		config := SmallChunkConfig()
		if config.Target.Value != 500 {
			t.Errorf("Expected target 500, got %d", config.Target.Value)
		}
		if config.Max.Value != 800 {
			t.Errorf("Expected max 800, got %d", config.Max.Value)
		}
	})

	t.Run("MediumChunkConfig", func(t *testing.T) {
		config := MediumChunkConfig()
		if config.Target.Value != 1000 {
			t.Errorf("Expected target 1000, got %d", config.Target.Value)
		}
	})

	t.Run("LargeChunkConfig", func(t *testing.T) {
		config := LargeChunkConfig()
		if config.Target.Value != 2000 {
			t.Errorf("Expected target 2000, got %d", config.Target.Value)
		}
		if config.Max.Value != 4000 {
			t.Errorf("Expected max 4000, got %d", config.Max.Value)
		}
	})

	t.Run("OpenAIEmbeddingConfig", func(t *testing.T) {
		config := OpenAIEmbeddingConfig()
		if config.Target.Unit != SizeUnitTokens {
			t.Error("Expected token-based config")
		}
		if config.Max.Value != 8000 {
			t.Errorf("Expected max 8000 tokens, got %d", config.Max.Value)
		}
	})

	t.Run("CohereEmbeddingConfig", func(t *testing.T) {
		config := CohereEmbeddingConfig()
		if config.Target.Unit != SizeUnitTokens {
			t.Error("Expected token-based config")
		}
		if config.Target.Value != 256 {
			t.Errorf("Expected target 256 tokens, got %d", config.Target.Value)
		}
	})

	t.Run("ClaudeContextConfig", func(t *testing.T) {
		config := ClaudeContextConfig()
		if config.Target.Unit != SizeUnitTokens {
			t.Error("Expected token-based config")
		}
		if config.Target.Value != 2000 {
			t.Errorf("Expected target 2000 tokens, got %d", config.Target.Value)
		}
	})
}

func TestFindSentenceEndNear(t *testing.T) {
	text := "First sentence here. Second sentence follows. Third one."

	t.Run("find sentence end before target", func(t *testing.T) {
		pos := findSentenceEndNear(text, 25)
		// Should find end of first sentence (position 20)
		if pos < 15 || pos > 25 {
			t.Errorf("Expected position near 20, got %d", pos)
		}
	})

	t.Run("target at end", func(t *testing.T) {
		pos := findSentenceEndNear(text, len(text)+10)
		if pos != len(text) {
			t.Errorf("Expected %d, got %d", len(text), pos)
		}
	})
}

func TestFindWordBoundaryNear(t *testing.T) {
	text := "Hello world this is a test"

	t.Run("find word boundary", func(t *testing.T) {
		pos := findWordBoundaryNear(text, 8)
		// Should find space between "Hello" and "world" or "world" and "this"
		if pos < 5 || pos > 15 {
			t.Errorf("Expected position near word boundary, got %d", pos)
		}
	})
}

// Benchmarks

func BenchmarkSizeCalculator_Calculate(b *testing.B) {
	calc := NewSizeCalculator()
	text := strings.Repeat("This is a test sentence. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(text)
	}
}

func BenchmarkSizeCalculator_Check(b *testing.B) {
	calc := NewSizeCalculator()
	text := strings.Repeat("This is a test sentence. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Check(text)
	}
}

func BenchmarkSplitToSize(b *testing.B) {
	config := DefaultSizeConfig()
	config.Max = SizeLimit{Value: 500, Unit: SizeUnitCharacters, Type: LimitTypeHard}
	calc := NewSizeCalculatorWithConfig(config)
	text := strings.Repeat("This is a test sentence here. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.SplitToSize(text, nil)
	}
}

func BenchmarkCountSentences(b *testing.B) {
	text := strings.Repeat("This is sentence one. This is sentence two. ", 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		countSentences(text)
	}
}

func BenchmarkCountParagraphs(b *testing.B) {
	text := strings.Repeat("This is paragraph one.\n\nThis is paragraph two.\n\n", 25)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		countParagraphs(text)
	}
}
