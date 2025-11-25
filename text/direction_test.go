package text

import (
	"testing"
)

func TestGetCharDirection(t *testing.T) {
	tests := []struct {
		name string
		char rune
		want Direction
	}{
		// Arabic
		{"Arabic alif", 'ا', RTL},       // U+0627
		{"Arabic baa", 'ب', RTL},        // U+0628
		{"Arabic seen", 'س', RTL},       // U+0633
		{"Arabic lam", 'ل', RTL},        // U+0644
		{"Arabic meem", 'م', RTL},       // U+0645

		// Hebrew
		{"Hebrew alef", 'א', RTL},       // U+05D0
		{"Hebrew bet", 'ב', RTL},        // U+05D1
		{"Hebrew shin", 'ש', RTL},       // U+05E9

		// Latin (LTR)
		{"Latin A", 'A', LTR},
		{"Latin a", 'a', LTR},
		{"Latin Z", 'Z', LTR},
		{"Latin é", 'é', LTR},           // U+00E9

		// Cyrillic (LTR)
		{"Cyrillic А", 'А', LTR},        // U+0410
		{"Cyrillic я", 'я', LTR},        // U+044F

		// Greek (LTR)
		{"Greek Alpha", 'Α', LTR},       // U+0391
		{"Greek Omega", 'Ω', LTR},       // U+03A9

		// CJK (LTR in modern usage)
		{"CJK 中", '中', LTR},            // U+4E2D
		{"CJK 文", '文', LTR},            // U+6587
		{"Hiragana あ", 'あ', LTR},       // U+3042
		{"Katakana ア", 'ア', LTR},       // U+30A2

		// Neutral characters
		{"Space", ' ', Neutral},
		{"Digit 0", '0', Neutral},
		{"Digit 5", '5', Neutral},
		{"Period", '.', Neutral},
		{"Comma", ',', Neutral},
		{"Exclamation", '!', Neutral},
		{"Question", '?', Neutral},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCharDirection(tt.char)
			if got != tt.want {
				t.Errorf("GetCharDirection(%q U+%04X) = %v, want %v",
					tt.char, tt.char, got, tt.want)
			}
		})
	}
}

func TestDetectDirection(t *testing.T) {
	tests := []struct {
		name string
		text string
		want Direction
	}{
		// Pure LTR
		{"English", "Hello World", LTR},
		{"Russian", "Привет мир", LTR},
		{"Greek", "Γεια σου κόσμε", LTR},
		{"Chinese", "你好世界", LTR},

		// Pure RTL
		{"Arabic مرحبا", "مرحبا", RTL},         // Hello
		{"Arabic السلام", "السلام عليكم", RTL}, // Peace be upon you
		{"Hebrew shalom", "שלום", RTL},         // Hello

		// Bidirectional (mixed)
		{"English with Arabic", "Hello مرحبا World", LTR},  // More English
		{"Arabic with English", "مرحبا Hello عليكم", RTL},  // More Arabic

		// Neutral only
		{"Numbers only", "12345", Neutral},
		{"Punctuation", "...", Neutral},
		{"Spaces", "   ", Neutral},

		// Empty
		{"Empty string", "", Neutral},

		// Mixed with numbers
		{"English + numbers", "Hello 123", LTR},
		{"Arabic + numbers", "مرحبا 123", RTL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectDirection(tt.text)
			if got != tt.want {
				t.Errorf("DetectDirection(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestGroupFragmentsByLine(t *testing.T) {
	e := NewExtractor()
	e.RegisterFont("/TestFont", "Helvetica", "Type1")

	// Create fragments on multiple lines
	e.fragments = []TextFragment{
		// Line 1 (Y=100)
		{Text: "Hello", X: 10, Y: 100, Height: 12, Direction: LTR},
		{Text: "World", X: 50, Y: 100, Height: 12, Direction: LTR},

		// Line 2 (Y=80) - different line (vertical gap > 0.5*height)
		{Text: "Second", X: 10, Y: 80, Height: 12, Direction: LTR},
		{Text: "Line", X: 60, Y: 80, Height: 12, Direction: LTR},

		// Line 3 (Y=60)
		{Text: "Third", X: 10, Y: 60, Height: 12, Direction: LTR},
	}

	lines := e.groupFragmentsByLine()

	// Should have 3 lines
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}

	// Line 1 should have 2 fragments
	if len(lines[0]) != 2 {
		t.Errorf("Line 1: expected 2 fragments, got %d", len(lines[0]))
	}
	if lines[0][0].Text != "Hello" || lines[0][1].Text != "World" {
		t.Errorf("Line 1: unexpected fragments")
	}

	// Line 2 should have 2 fragments
	if len(lines[1]) != 2 {
		t.Errorf("Line 2: expected 2 fragments, got %d", len(lines[1]))
	}

	// Line 3 should have 1 fragment
	if len(lines[2]) != 1 {
		t.Errorf("Line 3: expected 1 fragment, got %d", len(lines[2]))
	}
}

func TestDetectLineDirection(t *testing.T) {
	e := NewExtractor()

	tests := []struct {
		name      string
		fragments []TextFragment
		want      Direction
	}{
		{
			name: "Pure LTR line",
			fragments: []TextFragment{
				{Text: "Hello", Direction: LTR},
				{Text: "World", Direction: LTR},
			},
			want: LTR,
		},
		{
			name: "Pure RTL line",
			fragments: []TextFragment{
				{Text: "مرحبا", Direction: RTL},
				{Text: "العالم", Direction: RTL},
			},
			want: RTL,
		},
		{
			name: "Mixed line - LTR dominant",
			fragments: []TextFragment{
				{Text: "Hello", Direction: LTR},
				{Text: "مرحبا", Direction: RTL},
				{Text: "World", Direction: LTR},
			},
			want: LTR,
		},
		{
			name: "Mixed line - RTL dominant",
			fragments: []TextFragment{
				{Text: "مرحبا", Direction: RTL},
				{Text: "Hello", Direction: LTR},
				{Text: "العالم", Direction: RTL},
			},
			want: RTL,
		},
		{
			name: "Neutral only",
			fragments: []TextFragment{
				{Text: "123", Direction: Neutral},
				{Text: "...", Direction: Neutral},
			},
			want: LTR, // Default to LTR
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.detectLineDirection(tt.fragments)
			if got != tt.want {
				t.Errorf("detectLineDirection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReorderFragmentsForReading(t *testing.T) {
	e := NewExtractor()

	tests := []struct {
		name      string
		fragments []TextFragment
		direction Direction
		wantOrder []string // Expected text order after reordering
	}{
		{
			name: "LTR - already in order",
			fragments: []TextFragment{
				{Text: "Hello", X: 10},
				{Text: "World", X: 50},
			},
			direction: LTR,
			wantOrder: []string{"Hello", "World"},
		},
		{
			name: "LTR - needs reordering",
			fragments: []TextFragment{
				{Text: "World", X: 50},
				{Text: "Hello", X: 10},
			},
			direction: LTR,
			wantOrder: []string{"Hello", "World"},
		},
		{
			name: "RTL - visual right-to-left",
			fragments: []TextFragment{
				{Text: "العالم", X: 10, Width: 30},  // "world" on left visually
				{Text: "مرحبا", X: 50, Width: 30},   // "hello" on right visually
			},
			direction: RTL,
			wantOrder: []string{"مرحبا", "العالم"}, // Reading order: right to left
		},
		{
			name: "RTL - already in reading order",
			fragments: []TextFragment{
				{Text: "مرحبا", X: 50, Width: 30},   // "hello" on right
				{Text: "العالم", X: 10, Width: 30},  // "world" on left
			},
			direction: RTL,
			wantOrder: []string{"مرحبا", "العالم"}, // Reading order maintained
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.reorderFragmentsForReading(tt.fragments, tt.direction)

			if len(got) != len(tt.wantOrder) {
				t.Fatalf("Expected %d fragments, got %d", len(tt.wantOrder), len(got))
			}

			for i, wantText := range tt.wantOrder {
				if got[i].Text != wantText {
					t.Errorf("Fragment %d: got %q, want %q", i, got[i].Text, wantText)
				}
			}
		})
	}
}

func TestCalculateHorizontalDistance(t *testing.T) {
	tests := []struct {
		name     string
		frag     TextFragment
		nextFrag TextFragment
		lineDir  Direction
		want     float64
	}{
		{
			name:     "LTR - normal gap",
			frag:     TextFragment{X: 10, Width: 20},
			nextFrag: TextFragment{X: 35, Width: 15},
			lineDir:  LTR,
			want:     5.0, // 35 - (10 + 20) = 5
		},
		{
			name:     "LTR - no gap",
			frag:     TextFragment{X: 10, Width: 20},
			nextFrag: TextFragment{X: 30, Width: 15},
			lineDir:  LTR,
			want:     0.0, // 30 - (10 + 20) = 0
		},
		{
			name:     "RTL - normal gap",
			frag:     TextFragment{X: 50, Width: 20},       // Current fragment (reading from right)
			nextFrag: TextFragment{X: 20, Width: 15},       // Next fragment (to the left)
			lineDir:  RTL,
			want:     15.0, // 50 - (20 + 15) = 15
		},
		{
			name:     "RTL - no gap",
			frag:     TextFragment{X: 50, Width: 20},
			nextFrag: TextFragment{X: 30, Width: 20},
			lineDir:  RTL,
			want:     0.0, // 50 - (30 + 20) = 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateHorizontalDistance(tt.frag, tt.nextFrag, tt.lineDir)
			if got != tt.want {
				t.Errorf("calculateHorizontalDistance() = %.2f, want %.2f", got, tt.want)
			}
		})
	}
}

func TestGetTextWithRTL(t *testing.T) {
	tests := []struct {
		name      string
		fragments []TextFragment
		want      string
	}{
		{
			name: "Simple LTR",
			fragments: []TextFragment{
				{Text: "Hello", X: 10, Y: 100, Width: 25, Height: 12, FontName: "/TestFont", FontSize: 12, Direction: LTR},
				{Text: "World", X: 38, Y: 100, Width: 30, Height: 12, FontName: "/TestFont", FontSize: 12, Direction: LTR},
			},
			want: "Hello World",
		},
		{
			name: "Simple RTL",
			fragments: []TextFragment{
				// In PDF, RTL text is stored in visual order (right to left on page)
				// "مرحبا العالم" = "Hello World" in Arabic
				// Visually: [العالم on left] [مرحبا on right]
				// Reading order (RTL): مرحبا العالم
				{Text: "العالم", X: 10, Y: 100, Width: 30, Height: 12, FontName: "/TestFont", FontSize: 12, Direction: RTL},
				{Text: "مرحبا", X: 50, Y: 100, Width: 30, Height: 12, FontName: "/TestFont", FontSize: 12, Direction: RTL},
			},
			want: "مرحبا العالم",
		},
		{
			name: "Multiple lines - mixed directions",
			fragments: []TextFragment{
				// Line 1: English (LTR)
				{Text: "Hello", X: 10, Y: 100, Width: 25, Height: 12, FontName: "/TestFont", FontSize: 12, Direction: LTR},
				{Text: "World", X: 40, Y: 100, Width: 30, Height: 12, FontName: "/TestFont", FontSize: 12, Direction: LTR},

				// Line 2: Arabic (RTL) - Y=88 for normal line break (not paragraph break)
				{Text: "العالم", X: 10, Y: 88, Width: 30, Height: 12, FontName: "/TestFont", FontSize: 12, Direction: RTL},
				{Text: "مرحبا", X: 50, Y: 88, Width: 30, Height: 12, FontName: "/TestFont", FontSize: 12, Direction: RTL},
			},
			want: "Hello World\nمرحبا العالم",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewExtractor()
			e.RegisterFont("/TestFont", "Helvetica", "Type1")
			e.fragments = tt.fragments

			got := e.GetText()
			if got != tt.want {
				t.Errorf("GetText() = %q, want %q", got, tt.want)
			}
		})
	}
}
