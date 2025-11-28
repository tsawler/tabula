package docx

import (
	"encoding/xml"
	"testing"
)

func TestNewStyleResolver_Nil(t *testing.T) {
	sr := NewStyleResolver(nil)
	if sr == nil {
		t.Fatal("NewStyleResolver(nil) returned nil")
	}

	// Should return default style
	style := sr.Resolve("")
	if style.FontSize != 11 {
		t.Errorf("default FontSize = %v, want 11", style.FontSize)
	}
	if style.FontName != "Calibri" {
		t.Errorf("default FontName = %v, want Calibri", style.FontName)
	}
}

func TestStyleResolver_DefaultStyle(t *testing.T) {
	sr := NewStyleResolver(nil)
	style := sr.defaultStyle()

	if style.FontSize != 11 {
		t.Errorf("FontSize = %v, want 11", style.FontSize)
	}
	if style.FontName != "Calibri" {
		t.Errorf("FontName = %v, want Calibri", style.FontName)
	}
	if style.Alignment != "left" {
		t.Errorf("Alignment = %v, want left", style.Alignment)
	}
}

func TestStyleResolver_ResolveBuiltInHeading(t *testing.T) {
	sr := NewStyleResolver(nil)

	tests := []struct {
		styleID       string
		wantIsHeading bool
		wantLevel     int
	}{
		{"Heading1", true, 1},
		{"Heading2", true, 2},
		{"heading1", true, 1}, // case insensitive
		{"Title", true, 1},
		{"Normal", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.styleID, func(t *testing.T) {
			style := sr.Resolve(tt.styleID)
			if style.IsHeading != tt.wantIsHeading {
				t.Errorf("IsHeading = %v, want %v", style.IsHeading, tt.wantIsHeading)
			}
			if style.HeadingLevel != tt.wantLevel {
				t.Errorf("HeadingLevel = %v, want %v", style.HeadingLevel, tt.wantLevel)
			}
		})
	}
}

func TestStyleResolver_WithStyles(t *testing.T) {
	styles := &stylesXML{
		Styles: []styleDefXML{
			{
				StyleID: "CustomHeading",
				Type:    "paragraph",
				Name:    styleNameXML{Val: "My Custom Heading"},
				PPr: paragraphPropsXML{
					OutlineLvl: outlineLvlXML{Val: "1"}, // Level 2 heading
				},
				RPr: runPropsXML{
					Bold:     boolXML{XMLName: xml.Name{Local: "b"}, Val: ""},
					FontSize: sizeXML{Val: "28"}, // 14pt
				},
			},
			{
				StyleID: "CenteredPara",
				Type:    "paragraph",
				Name:    styleNameXML{Val: "Centered Paragraph"},
				PPr: paragraphPropsXML{
					Justification: justificationXML{Val: "center"},
					Spacing:       spacingXML{Before: "240", After: "120"}, // 12pt before, 6pt after
				},
			},
		},
	}

	sr := NewStyleResolver(styles)

	t.Run("custom heading", func(t *testing.T) {
		style := sr.Resolve("CustomHeading")
		if !style.IsHeading {
			t.Error("expected IsHeading = true")
		}
		if style.HeadingLevel != 2 {
			t.Errorf("HeadingLevel = %v, want 2", style.HeadingLevel)
		}
		if style.FontSize != 14 {
			t.Errorf("FontSize = %v, want 14", style.FontSize)
		}
	})

	t.Run("centered paragraph", func(t *testing.T) {
		style := sr.Resolve("CenteredPara")
		if style.Alignment != "center" {
			t.Errorf("Alignment = %v, want center", style.Alignment)
		}
		if style.SpaceBefore != 12 {
			t.Errorf("SpaceBefore = %v, want 12", style.SpaceBefore)
		}
		if style.SpaceAfter != 6 {
			t.Errorf("SpaceAfter = %v, want 6", style.SpaceAfter)
		}
	})
}

func TestStyleResolver_Inheritance(t *testing.T) {
	styles := &stylesXML{
		Styles: []styleDefXML{
			{
				StyleID: "BaseStyle",
				Type:    "paragraph",
				Name:    styleNameXML{Val: "Base"},
				PPr: paragraphPropsXML{
					Justification: justificationXML{Val: "left"},
					Spacing:       spacingXML{After: "200"}, // 10pt
				},
				RPr: runPropsXML{
					FontSize: sizeXML{Val: "24"}, // 12pt
					Font:     fontXML{ASCII: "Arial"},
				},
			},
			{
				StyleID: "DerivedStyle",
				Type:    "paragraph",
				Name:    styleNameXML{Val: "Derived"},
				BasedOn: basedOnXML{Val: "BaseStyle"},
				PPr: paragraphPropsXML{
					Justification: justificationXML{Val: "center"}, // Override alignment
				},
				RPr: runPropsXML{
					Bold: boolXML{XMLName: xml.Name{Local: "b"}, Val: ""}, // Add bold
				},
			},
		},
	}

	sr := NewStyleResolver(styles)
	style := sr.Resolve("DerivedStyle")

	// Should inherit from BaseStyle
	if style.FontName != "Arial" {
		t.Errorf("FontName = %v, want Arial (inherited)", style.FontName)
	}
	if style.FontSize != 12 {
		t.Errorf("FontSize = %v, want 12 (inherited)", style.FontSize)
	}
	if style.SpaceAfter != 10 {
		t.Errorf("SpaceAfter = %v, want 10 (inherited)", style.SpaceAfter)
	}

	// Should override alignment
	if style.Alignment != "center" {
		t.Errorf("Alignment = %v, want center (overridden)", style.Alignment)
	}

	// Should add bold
	if !style.Bold {
		t.Error("Bold should be true")
	}
}

func TestStyleResolver_ResolveRun(t *testing.T) {
	styles := &stylesXML{
		Styles: []styleDefXML{
			{
				StyleID: "NormalStyle",
				Type:    "paragraph",
				Name:    styleNameXML{Val: "Normal"},
				RPr: runPropsXML{
					FontSize: sizeXML{Val: "22"}, // 11pt
					Font:     fontXML{ASCII: "Times New Roman"},
				},
			},
		},
	}

	sr := NewStyleResolver(styles)

	t.Run("inherit from paragraph style", func(t *testing.T) {
		runProps := runPropsXML{}
		resolved := sr.ResolveRun("NormalStyle", runProps)

		if resolved.FontName != "Times New Roman" {
			t.Errorf("FontName = %v, want Times New Roman", resolved.FontName)
		}
		if resolved.FontSize != 11 {
			t.Errorf("FontSize = %v, want 11", resolved.FontSize)
		}
	})

	t.Run("override with direct formatting", func(t *testing.T) {
		runProps := runPropsXML{
			Bold:     boolXML{XMLName: xml.Name{Local: "b"}, Val: ""}, // Simulates <w:b/>
			FontSize: sizeXML{Val: "28"},                               // 14pt
		}
		resolved := sr.ResolveRun("NormalStyle", runProps)

		// Inherited
		if resolved.FontName != "Times New Roman" {
			t.Errorf("FontName = %v, want Times New Roman", resolved.FontName)
		}

		// Overridden
		if resolved.FontSize != 14 {
			t.Errorf("FontSize = %v, want 14", resolved.FontSize)
		}
		if !resolved.Bold {
			t.Error("Bold should be true")
		}
	})
}

func TestParseHalfPoints(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"24", 12},    // 24 half-points = 12pt
		{"22", 11},    // 22 half-points = 11pt
		{"0", 0},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseHalfPoints(tt.input)
			if got != tt.want {
				t.Errorf("parseHalfPoints(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseTwips(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"240", 12}, // 240 twips = 12pt
		{"200", 10}, // 200 twips = 10pt
		{"20", 1},   // 20 twips = 1pt
		{"0", 0},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseTwips(tt.input)
			if got != tt.want {
				t.Errorf("parseTwips(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectBuiltInHeading(t *testing.T) {
	tests := []struct {
		styleID   string
		isHeading bool
		level     int
	}{
		{"Heading1", true, 1},
		{"heading1", true, 1},
		{"HEADING1", true, 1},
		{"Heading9", true, 9},
		{"Title", true, 1},
		{"Subtitle", true, 2},
		{"Normal", false, 0},
		{"BodyText", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.styleID, func(t *testing.T) {
			isHeading, level := detectBuiltInHeading(tt.styleID)
			if isHeading != tt.isHeading {
				t.Errorf("isHeading = %v, want %v", isHeading, tt.isHeading)
			}
			if level != tt.level {
				t.Errorf("level = %v, want %v", level, tt.level)
			}
		})
	}
}

func TestEstimateHeadingLevel(t *testing.T) {
	tests := []struct {
		fontSize float64
		want     int
	}{
		{24, 1},
		{18, 2},
		{14, 3},
		{12, 4},
		{10, 5},
	}

	for _, tt := range tests {
		got := estimateHeadingLevel(tt.fontSize)
		if got != tt.want {
			t.Errorf("estimateHeadingLevel(%v) = %v, want %v", tt.fontSize, got, tt.want)
		}
	}
}
