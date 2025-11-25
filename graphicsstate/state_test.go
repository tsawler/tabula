package graphicsstate

import (
	"math"
	"testing"

	"github.com/tsawler/tabula/model"
)

// TestNewGraphicsState tests initial state
func TestNewGraphicsState(t *testing.T) {
	gs := NewGraphicsState()

	if gs.LineWidth != 1.0 {
		t.Errorf("expected line width 1.0, got %f", gs.LineWidth)
	}

	if gs.Text.FontSize != 12.0 {
		t.Errorf("expected font size 12.0, got %f", gs.Text.FontSize)
	}

	if gs.Text.HorizontalScaling != 100.0 {
		t.Errorf("expected horizontal scaling 100.0, got %f", gs.Text.HorizontalScaling)
	}

	// Check CTM is identity
	if !gs.CTM.IsIdentity() {
		t.Error("expected CTM to be identity matrix")
	}
}

// TestSaveRestore tests q/Q operators
func TestSaveRestore(t *testing.T) {
	gs := NewGraphicsState()

	// Modify state
	gs.SetLineWidth(2.5)
	gs.SetFont("Helvetica", 14)

	// Save
	gs.Save()

	// Modify again
	gs.SetLineWidth(5.0)
	gs.SetFont("Times", 18)

	if gs.LineWidth != 5.0 {
		t.Errorf("expected line width 5.0, got %f", gs.LineWidth)
	}

	// Restore
	err := gs.Restore()
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Check restored values
	if gs.LineWidth != 2.5 {
		t.Errorf("expected restored line width 2.5, got %f", gs.LineWidth)
	}

	if gs.Text.FontName != "Helvetica" {
		t.Errorf("expected restored font Helvetica, got %s", gs.Text.FontName)
	}

	if gs.Text.FontSize != 14 {
		t.Errorf("expected restored font size 14, got %f", gs.Text.FontSize)
	}
}

// TestRestoreUnderflow tests restore without save
func TestRestoreUnderflow(t *testing.T) {
	gs := NewGraphicsState()

	err := gs.Restore()
	if err == nil {
		t.Error("expected error on restore without save")
	}
}

// TestNestedSaveRestore tests nested q/Q
func TestNestedSaveRestore(t *testing.T) {
	gs := NewGraphicsState()

	gs.SetLineWidth(1.0)
	gs.Save() // Level 1

	gs.SetLineWidth(2.0)
	gs.Save() // Level 2

	gs.SetLineWidth(3.0)

	// Restore to level 2
	gs.Restore()
	if gs.LineWidth != 2.0 {
		t.Errorf("expected line width 2.0, got %f", gs.LineWidth)
	}

	// Restore to level 1
	gs.Restore()
	if gs.LineWidth != 1.0 {
		t.Errorf("expected line width 1.0, got %f", gs.LineWidth)
	}
}

// TestTransform tests cm operator
func TestTransform(t *testing.T) {
	gs := NewGraphicsState()

	// Apply translation
	translation := model.Translate(100, 200)
	gs.Transform(translation)

	if gs.CTM[4] != 100 || gs.CTM[5] != 200 {
		t.Errorf("expected translation (100, 200), got (%f, %f)", gs.CTM[4], gs.CTM[5])
	}
}

// TestSetFont tests Tf operator
func TestSetFont(t *testing.T) {
	gs := NewGraphicsState()

	gs.SetFont("Helvetica-Bold", 24.0)

	if gs.Text.FontName != "Helvetica-Bold" {
		t.Errorf("expected font Helvetica-Bold, got %s", gs.Text.FontName)
	}

	if gs.Text.FontSize != 24.0 {
		t.Errorf("expected font size 24.0, got %f", gs.Text.FontSize)
	}
}

// TestTextSpacing tests Tc and Tw operators
func TestTextSpacing(t *testing.T) {
	gs := NewGraphicsState()

	gs.SetCharSpacing(0.5)
	gs.SetWordSpacing(1.0)

	if gs.Text.CharSpacing != 0.5 {
		t.Errorf("expected char spacing 0.5, got %f", gs.Text.CharSpacing)
	}

	if gs.Text.WordSpacing != 1.0 {
		t.Errorf("expected word spacing 1.0, got %f", gs.Text.WordSpacing)
	}
}

// TestHorizontalScaling tests Tz operator
func TestHorizontalScaling(t *testing.T) {
	gs := NewGraphicsState()

	gs.SetHorizontalScaling(80.0)

	if gs.Text.HorizontalScaling != 80.0 {
		t.Errorf("expected horizontal scaling 80.0, got %f", gs.Text.HorizontalScaling)
	}
}

// TestLeading tests TL operator
func TestLeading(t *testing.T) {
	gs := NewGraphicsState()

	gs.SetLeading(14.0)

	if gs.Text.Leading != 14.0 {
		t.Errorf("expected leading 14.0, got %f", gs.Text.Leading)
	}
}

// TestRenderingMode tests Tr operator
func TestRenderingMode(t *testing.T) {
	gs := NewGraphicsState()

	gs.SetRenderingMode(2)

	if gs.Text.RenderingMode != 2 {
		t.Errorf("expected rendering mode 2, got %d", gs.Text.RenderingMode)
	}
}

// TestTextRise tests Ts operator
func TestTextRise(t *testing.T) {
	gs := NewGraphicsState()

	gs.SetTextRise(5.0)

	if gs.Text.Rise != 5.0 {
		t.Errorf("expected text rise 5.0, got %f", gs.Text.Rise)
	}
}

// TestBeginText tests BT operator
func TestBeginText(t *testing.T) {
	gs := NewGraphicsState()

	// Modify text matrix
	gs.Text.TextMatrix = model.Matrix{1, 0, 0, 1, 100, 200}

	// Begin text should reset to identity
	gs.BeginText()

	if !gs.Text.TextMatrix.IsIdentity() {
		t.Error("expected text matrix to be identity after BT")
	}

	if !gs.Text.TextLineMatrix.IsIdentity() {
		t.Error("expected text line matrix to be identity after BT")
	}
}

// TestSetTextMatrix tests Tm operator
func TestSetTextMatrix(t *testing.T) {
	gs := NewGraphicsState()

	m := model.Matrix{1, 0, 0, 1, 72, 720}

	gs.SetTextMatrix(m)

	if gs.Text.TextMatrix != m {
		t.Error("text matrix not set correctly")
	}

	if gs.Text.TextLineMatrix != m {
		t.Error("text line matrix not set correctly")
	}
}

// TestTranslateText tests Td operator
func TestTranslateText(t *testing.T) {
	gs := NewGraphicsState()
	gs.BeginText()

	gs.TranslateText(10, 20)

	if gs.Text.TextMatrix[4] != 10 || gs.Text.TextMatrix[5] != 20 {
		t.Errorf("expected translation (10, 20), got (%f, %f)",
			gs.Text.TextMatrix[4], gs.Text.TextMatrix[5])
	}

	// Translate again
	gs.TranslateText(5, 10)

	if gs.Text.TextMatrix[4] != 15 || gs.Text.TextMatrix[5] != 30 {
		t.Errorf("expected cumulative translation (15, 30), got (%f, %f)",
			gs.Text.TextMatrix[4], gs.Text.TextMatrix[5])
	}
}

// TestTranslateTextSetLeading tests TD operator
func TestTranslateTextSetLeading(t *testing.T) {
	gs := NewGraphicsState()
	gs.BeginText()

	gs.TranslateTextSetLeading(0, -14)

	if gs.Text.Leading != 14 {
		t.Errorf("expected leading 14, got %f", gs.Text.Leading)
	}

	if gs.Text.TextMatrix[5] != -14 {
		t.Errorf("expected Y translation -14, got %f", gs.Text.TextMatrix[5])
	}
}

// TestNextLine tests T* operator
func TestNextLine(t *testing.T) {
	gs := NewGraphicsState()
	gs.BeginText()
	gs.SetLeading(14)

	initialY := gs.Text.TextMatrix[5]

	gs.NextLine()

	expectedY := initialY - 14
	if math.Abs(gs.Text.TextMatrix[5]-expectedY) > 0.001 {
		t.Errorf("expected Y %f, got %f", expectedY, gs.Text.TextMatrix[5])
	}
}

// TestShowText tests Tj operator
func TestShowText(t *testing.T) {
	gs := NewGraphicsState()
	gs.BeginText()
	gs.SetFont("Helvetica", 12)

	initialX := gs.Text.TextMatrix[4]

	dx, _ := gs.ShowText("Hello")

	// Text matrix should have advanced
	if gs.Text.TextMatrix[4] <= initialX {
		t.Error("text matrix should advance after showing text")
	}

	// dx should be positive
	if dx <= 0 {
		t.Errorf("expected positive dx, got %f", dx)
	}
}

// TestShowTextWithSpacing tests Tj with spacing
func TestShowTextWithSpacing(t *testing.T) {
	gs := NewGraphicsState()
	gs.BeginText()
	gs.SetFont("Helvetica", 12)
	gs.SetCharSpacing(0.5)
	gs.SetWordSpacing(2.0)

	initialX := gs.Text.TextMatrix[4]

	gs.ShowText("A B")

	// Should advance more than without spacing
	advancement := gs.Text.TextMatrix[4] - initialX

	if advancement <= 0 {
		t.Errorf("expected positive advancement, got %f", advancement)
	}
}

// TestShowTextArray tests TJ operator
func TestShowTextArray(t *testing.T) {
	gs := NewGraphicsState()
	gs.BeginText()
	gs.SetFont("Helvetica", 12)

	initialX := gs.Text.TextMatrix[4]

	// Array with text and positioning
	array := []interface{}{
		"Hello",
		-200, // Negative adjustment moves right
		"World",
	}

	dx, _ := gs.ShowTextArray(array)

	// Should have advanced
	if gs.Text.TextMatrix[4] <= initialX {
		t.Error("text matrix should advance")
	}

	if dx <= 0 {
		t.Errorf("expected positive dx, got %f", dx)
	}
}

// TestGetTextPosition tests position calculation
func TestGetTextPosition(t *testing.T) {
	gs := NewGraphicsState()
	gs.BeginText()
	gs.SetTextMatrix(model.Matrix{1, 0, 0, 1, 100, 200})

	x, y := gs.GetTextPosition()

	if x != 100 || y != 200 {
		t.Errorf("expected position (100, 200), got (%f, %f)", x, y)
	}
}

// TestGetTextPositionWithCTM tests position with CTM
func TestGetTextPositionWithCTM(t *testing.T) {
	gs := NewGraphicsState()

	// Apply CTM translation
	gs.Transform(model.Translate(50, 50))

	gs.BeginText()
	gs.SetTextMatrix(model.Matrix{1, 0, 0, 1, 100, 200})

	x, y := gs.GetTextPosition()

	// Should include CTM translation
	expectedX := 150.0
	expectedY := 250.0

	if math.Abs(x-expectedX) > 0.001 || math.Abs(y-expectedY) > 0.001 {
		t.Errorf("expected position (%f, %f), got (%f, %f)", expectedX, expectedY, x, y)
	}
}

// TestColors tests RG and rg operators
func TestColors(t *testing.T) {
	gs := NewGraphicsState()

	gs.SetStrokeColorRGB(1.0, 0.0, 0.0)
	gs.SetFillColorRGB(0.0, 1.0, 0.0)

	if gs.StrokeColor != [3]float64{1.0, 0.0, 0.0} {
		t.Errorf("stroke color not set correctly: %v", gs.StrokeColor)
	}

	if gs.FillColor != [3]float64{0.0, 1.0, 0.0} {
		t.Errorf("fill color not set correctly: %v", gs.FillColor)
	}
}

// TestLineWidth tests w operator
func TestLineWidth(t *testing.T) {
	gs := NewGraphicsState()

	gs.SetLineWidth(2.5)

	if gs.LineWidth != 2.5 {
		t.Errorf("expected line width 2.5, got %f", gs.LineWidth)
	}
}

// TestClone tests state cloning
func TestClone(t *testing.T) {
	gs := NewGraphicsState()
	gs.SetFont("Helvetica", 14)
	gs.SetLineWidth(2.0)

	clone := gs.Clone()

	// Modify original
	gs.SetFont("Times", 18)
	gs.SetLineWidth(3.0)

	// Clone should be unchanged
	if clone.Text.FontName != "Helvetica" {
		t.Errorf("clone font should be Helvetica, got %s", clone.Text.FontName)
	}

	if clone.Text.FontSize != 14 {
		t.Errorf("clone font size should be 14, got %f", clone.Text.FontSize)
	}

	if clone.LineWidth != 2.0 {
		t.Errorf("clone line width should be 2.0, got %f", clone.LineWidth)
	}
}

// TestComplexTextFlow tests realistic text flow
func TestComplexTextFlow(t *testing.T) {
	gs := NewGraphicsState()

	// BT
	gs.BeginText()

	// /F1 12 Tf
	gs.SetFont("F1", 12)

	// 72 720 Td
	gs.TranslateText(72, 720)

	// (Hello) Tj
	gs.ShowText("Hello")

	// 0 -14 Td
	gs.TranslateText(0, -14)

	// (World) Tj
	gs.ShowText("World")

	// ET
	gs.EndText()

	// Text matrix should have moved
	if gs.Text.TextMatrix[4] <= 72 {
		t.Error("text matrix X should have advanced")
	}

	if gs.Text.TextMatrix[5] != 706 {
		t.Errorf("expected Y position 706, got %f", gs.Text.TextMatrix[5])
	}
}
