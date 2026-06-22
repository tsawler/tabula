package rag

import (
	"strings"
	"testing"
)

// buildText returns a sentence-rich string of approximately n characters.
func buildText(n int) string {
	var sb strings.Builder
	i := 0
	for sb.Len() < n {
		i++
		sb.WriteString("This is sentence number ")
		sb.WriteString(strings.Repeat("x", 3))
		sb.WriteString(". ")
	}
	return strings.TrimSpace(sb.String())
}

func TestSplitToSize_NoOverMax(t *testing.T) {
	cfg := DefaultSizeConfig() // Min 100, Max 2000
	calc := NewSizeCalculatorWithConfig(cfg)

	for _, n := range []int{2050, 2100, 3999, 4001, 6050, 8123} {
		text := buildText(n)
		chunks := calc.SplitToSize(text, nil)
		for i, c := range chunks {
			if len(c) > 2000 {
				t.Errorf("n=%d chunk %d exceeds max: %d chars", n, i, len(c))
			}
		}
	}
}

func TestSplitToSize_NoOrphanTail(t *testing.T) {
	cfg := DefaultSizeConfig() // Min 100, Max 2000
	calc := NewSizeCalculatorWithConfig(cfg)

	// Sizes chosen to land just above a max multiple, which previously stranded
	// a tiny tail (e.g. 2050 -> [2000, 50]).
	for _, n := range []int{2010, 2050, 2099, 4040, 6020} {
		text := buildText(n)
		chunks := calc.SplitToSize(text, nil)
		if len(chunks) < 2 {
			t.Fatalf("n=%d expected a split, got %d chunk(s)", n, len(chunks))
		}
		for i, c := range chunks {
			if len(c) < 100 {
				t.Errorf("n=%d chunk %d is an orphan: %d chars (%q...)", n, i, len(c), c[:min(20, len(c))])
			}
		}
	}
}

func TestSplitToSize_RebalancesBorderline(t *testing.T) {
	cfg := DefaultSizeConfig()
	calc := NewSizeCalculatorWithConfig(cfg)

	// 2050 chars: cutting at max would leave a 50-char tail; expect two balanced
	// pieces instead.
	text := buildText(2050)
	chunks := calc.SplitToSize(text, nil)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 balanced chunks, got %d", len(chunks))
	}
	// Both halves should be comfortably above min and below max.
	for i, c := range chunks {
		if len(c) < 100 || len(c) > 2000 {
			t.Errorf("chunk %d out of range: %d chars", i, len(c))
		}
	}
}

func TestSplitToSize_LargeTailNotRebalanced(t *testing.T) {
	cfg := DefaultSizeConfig()
	calc := NewSizeCalculatorWithConfig(cfg)

	// 2300 chars: cutting at max leaves a healthy ~300-char tail, so the first
	// chunk should still pack near the max (no unnecessary rebalancing).
	text := buildText(2300)
	chunks := calc.SplitToSize(text, nil)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) < 1500 {
		t.Errorf("expected first chunk to pack near max, got %d chars", len(chunks[0]))
	}
}

func TestSplitToSize_ContentPreserved(t *testing.T) {
	cfg := DefaultSizeConfig()
	calc := NewSizeCalculatorWithConfig(cfg)

	text := buildText(5000)
	chunks := calc.SplitToSize(text, nil)

	// Rejoining should preserve all non-whitespace content.
	strip := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	got := strip(strings.Join(chunks, " "))
	want := strip(text)
	if got != want {
		t.Errorf("content not preserved across split\nwant len=%d\ngot  len=%d", len(want), len(got))
	}
}
