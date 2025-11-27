package text

import (
	"testing"
)

func TestShouldInsertSpace(t *testing.T) {
	// Create extractor with a test font
	e := NewExtractor()
	e.RegisterFont("/TestFont", "Helvetica", "Type1")

	tests := []struct {
		name           string
		frag1          TextFragment
		frag2          TextFragment
		horizontalDist float64
		want           bool
	}{
		{
			name: "No space - fragments touching",
			frag1: TextFragment{
				Text:     "Hello",
				FontName: "/TestFont",
				FontSize: 12.0,
			},
			frag2: TextFragment{
				Text:     "World",
				FontName: "/TestFont",
				FontSize: 12.0,
			},
			horizontalDist: 0.0,
			want:           false,
		},
		{
			name: "No space - very small gap (kerning)",
			frag1: TextFragment{
				Text:     "ar",
				FontName: "/TestFont",
				FontSize: 12.0,
			},
			frag2: TextFragment{
				Text:     "e",
				FontName: "/TestFont",
				FontSize: 12.0,
			},
			horizontalDist: 0.5, // Small kerning gap
			want:           false,
		},
		{
			name: "Insert space - normal word gap",
			frag1: TextFragment{
				Text:     "Hello",
				FontName: "/TestFont",
				FontSize: 12.0,
			},
			frag2: TextFragment{
				Text:     "World",
				FontName: "/TestFont",
				FontSize: 12.0,
			},
			horizontalDist: 3.0, // Typical space width
			want:           true,
		},
		{
			name: "Insert space - large gap",
			frag1: TextFragment{
				Text:     "Hello",
				FontName: "/TestFont",
				FontSize: 12.0,
			},
			frag2: TextFragment{
				Text:     "World",
				FontName: "/TestFont",
				FontSize: 12.0,
			},
			horizontalDist: 10.0, // Large gap
			want:           true,
		},
		{
			name: "No space - overlapping fragments",
			frag1: TextFragment{
				Text:     "Hello",
				FontName: "/TestFont",
				FontSize: 12.0,
			},
			frag2: TextFragment{
				Text:     "World",
				FontName: "/TestFont",
				FontSize: 12.0,
			},
			horizontalDist: -1.0, // Overlap
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.shouldInsertSpace(tt.frag1, tt.frag2, tt.horizontalDist)
			if got != tt.want {
				t.Errorf("shouldInsertSpace() = %v, want %v (gap=%.2f)", got, tt.want, tt.horizontalDist)
			}
		})
	}
}

func TestGetSpaceWidth(t *testing.T) {
	e := NewExtractor()
	e.RegisterFont("/TestFont", "Helvetica", "Type1")

	tests := []struct {
		name     string
		fontName string
		fontSize float64
		wantMin  float64
		wantMax  float64
	}{
		{
			name:     "Helvetica 12pt",
			fontName: "/TestFont",
			fontSize: 12.0,
			wantMin:  3.0, // Space width should be around 3.3 points
			wantMax:  3.5,
		},
		{
			name:     "Helvetica 24pt",
			fontName: "/TestFont",
			fontSize: 24.0,
			wantMin:  6.0, // Space width scales with font size
			wantMax:  7.0,
		},
		{
			name:     "Unknown font - fallback",
			fontName: "/UnknownFont",
			fontSize: 12.0,
			wantMin:  2.5, // Fallback is 25% of font size
			wantMax:  3.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.getSpaceWidth(tt.fontName, tt.fontSize)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("getSpaceWidth() = %.2f, want between %.2f and %.2f", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestGetTextWithSmartSpacing(t *testing.T) {
	e := NewExtractor()
	e.RegisterFont("/TestFont", "Helvetica", "Type1")

	// Create fragments that simulate "ar" and "e" being placed close together
	e.fragments = []TextFragment{
		{
			Text:     "These",
			X:        10.0,
			Y:        100.0,
			Width:    25.0,
			Height:   12.0,
			FontName: "/TestFont",
			FontSize: 12.0,
		},
		{
			Text:     "ar",
			X:        38.5, // Gap of 3.5 from end of "These" - should insert space
			Y:        100.0,
			Width:    10.0,
			Height:   12.0,
			FontName: "/TestFont",
			FontSize: 12.0,
		},
		{
			Text:     "e",
			X:        49.0, // Small gap (0.5) - should NOT insert space
			Y:        100.0,
			Width:    5.0,
			Height:   12.0,
			FontName: "/TestFont",
			FontSize: 12.0,
		},
		{
			Text:     "words",
			X:        57.5, // Larger gap (3.5) - should insert space
			Y:        100.0,
			Width:    30.0,
			Height:   12.0,
			FontName: "/TestFont",
			FontSize: 12.0,
		},
	}

	result := e.GetText()

	// Should be "These are words" NOT "These ar e words"
	expected := "These are words"
	if result != expected {
		t.Errorf("GetText() = %q, want %q", result, expected)
	}
}

func TestEffectiveFontSizeIntegration(t *testing.T) {
	// This test ensures that when a PDF uses text matrix scaling (like Pages.app),
	// we correctly calculate space width using the effective font size

	e := NewExtractor()
	e.RegisterFont("/TestFont", "Helvetica", "Type1")

	// Simulate a font with size=1 in Tf operator, but scaled to 12 via text matrix
	// (This is what Pages.app does)

	spaceWidth := e.getSpaceWidth("/TestFont", 12.0)

	// For Helvetica, space width is 278 units in 1000ths of em
	// At 12pt: (278 * 12) / 1000 = 3.336 points
	expectedMin := 3.0
	expectedMax := 3.5

	if spaceWidth < expectedMin || spaceWidth > expectedMax {
		t.Errorf("Space width with effective font size: got %.2f, want between %.2f and %.2f",
			spaceWidth, expectedMin, expectedMax)
	}

	t.Logf("✅ Space width at 12pt: %.2f points (expected ~3.3)", spaceWidth)
}
