package font

import (
	"testing"
)

// TestNewFont tests font creation
func TestNewFont(t *testing.T) {
	font := NewFont("F1", "Helvetica", "Type1")

	if font.Name != "F1" {
		t.Errorf("expected name F1, got %s", font.Name)
	}

	if font.BaseFont != "Helvetica" {
		t.Errorf("expected base font Helvetica, got %s", font.BaseFont)
	}

	if font.Subtype != "Type1" {
		t.Errorf("expected subtype Type1, got %s", font.Subtype)
	}
}

// TestGetWidth tests character width retrieval
func TestGetWidth(t *testing.T) {
	font := NewFont("F1", "Helvetica", "Type1")

	// Test known width
	width := font.GetWidth('A')
	if width != 667 {
		t.Errorf("expected width 667 for 'A', got %f", width)
	}

	// Test space
	width = font.GetWidth(' ')
	if width != 278 {
		t.Errorf("expected width 278 for space, got %f", width)
	}
}

// TestGetStringWidth tests string width calculation
func TestGetStringWidth(t *testing.T) {
	font := NewFont("F1", "Helvetica", "Type1")

	width := font.GetStringWidth("Hi")

	// H=722, i=222
	expected := 722.0 + 222.0
	if width != expected {
		t.Errorf("expected width %f for 'Hi', got %f", expected, width)
	}
}

// TestStandardFonts tests Standard 14 font detection
func TestStandardFonts(t *testing.T) {
	tests := []struct {
		baseFont  string
		isStandard bool
	}{
		{"Helvetica", true},
		{"Helvetica-Bold", true},
		{"Times-Roman", true},
		{"Courier", true},
		{"Arial", false},
		{"CustomFont", false},
	}

	for _, tt := range tests {
		t.Run(tt.baseFont, func(t *testing.T) {
			font := NewFont("F1", tt.baseFont, "Type1")

			if font.IsStandardFont() != tt.isStandard {
				t.Errorf("expected IsStandardFont() to be %v for %s",
					tt.isStandard, tt.baseFont)
			}
		})
	}
}

// TestCourierMonospaced tests Courier monospaced widths
func TestCourierMonospaced(t *testing.T) {
	font := NewFont("F1", "Courier", "Type1")

	// All characters should have same width in Courier
	width := font.GetWidth('A')
	expectedWidth := 600.0

	if width != expectedWidth {
		t.Errorf("expected width %f, got %f", expectedWidth, width)
	}

	// Check another character
	widthI := font.GetWidth('i')
	if widthI != expectedWidth {
		t.Errorf("expected width %f for 'i', got %f", expectedWidth, widthI)
	}
}

// TestHelveticaBold tests Helvetica-Bold widths
func TestHelveticaBold(t *testing.T) {
	font := NewFont("F1", "Helvetica-Bold", "Type1")

	width := font.GetWidth('A')
	expected := 722.0

	if width != expected {
		t.Errorf("expected width %f, got %f", expected, width)
	}
}

// TestTimesRoman tests Times-Roman widths
func TestTimesRoman(t *testing.T) {
	font := NewFont("F1", "Times-Roman", "Type1")

	width := font.GetWidth('A')
	expected := 722.0

	if width != expected {
		t.Errorf("expected width %f, got %f", expected, width)
	}
}

// TestNonStandardFont tests fallback for non-standard fonts
func TestNonStandardFont(t *testing.T) {
	font := NewFont("F1", "CustomFont", "Type1")

	// Should use Helvetica widths as default
	width := font.GetWidth('A')
	if width == 0 {
		t.Error("expected non-zero width for non-standard font")
	}
}

// TestUnknownCharacter tests fallback for unknown characters
func TestUnknownCharacter(t *testing.T) {
	font := NewFont("F1", "Helvetica", "Type1")

	// Test character not in width table
	width := font.GetWidth('\u2022') // Bullet point

	// Should return default width
	if width != 500.0 {
		t.Errorf("expected default width 500, got %f", width)
	}
}
