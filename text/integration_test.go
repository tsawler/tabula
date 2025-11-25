package text

import (
	"bytes"
	"compress/zlib"
	"strings"
	"testing"
)

// TestFullPipeline tests the complete text extraction pipeline:
// Content Stream → Parser → Graphics State → Extractor
func TestFullPipeline(t *testing.T) {
	// Realistic PDF content stream
	contentStream := []byte(`BT
/F1 12 Tf
1 0 0 1 72 720 Tm
(Hello, World!) Tj
0 -14 Td
(This is a test PDF.) Tj
0 -14 Td
(It has multiple lines.) Tj
ET`)

	extractor := NewExtractor()
	extractor.RegisterFont("/F1", "Helvetica", "Type1")

	fragments, err := extractor.ExtractFromBytes(contentStream)
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	if len(fragments) != 3 {
		t.Fatalf("Expected 3 text fragments, got %d", len(fragments))
	}

	// Check first fragment
	if fragments[0].Text != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got %q", fragments[0].Text)
	}

	if fragments[0].FontSize != 12 {
		t.Errorf("Expected font size 12, got %f", fragments[0].FontSize)
	}

	// Check positions
	if fragments[0].X != 72 || fragments[0].Y != 720 {
		t.Errorf("Expected position (72, 720), got (%f, %f)",
			fragments[0].X, fragments[0].Y)
	}

	// Second fragment should be below first
	if fragments[1].Y >= fragments[0].Y {
		t.Errorf("Second fragment should be below first")
	}

	// Get full text
	fullText := extractor.GetText()
	if !strings.Contains(fullText, "Hello, World!") {
		t.Errorf("Full text should contain 'Hello, World!', got: %q", fullText)
	}

	t.Logf("Extracted text: %q", fullText)
}

// TestCompressedContentStream tests extraction from zlib-compressed stream
func TestCompressedContentStream(t *testing.T) {
	// Original content stream
	contentStream := []byte(`BT
/F1 16 Tf
100 200 Td
(Compressed Text) Tj
ET`)

	// Compress with zlib
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(contentStream)
	w.Close()
	compressed := buf.Bytes()

	t.Logf("Original: %d bytes, Compressed: %d bytes", len(contentStream), len(compressed))

	// Decompress
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("Failed to create zlib reader: %v", err)
	}
	defer r.Close()

	var decompressed bytes.Buffer
	_, err = decompressed.ReadFrom(r)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	// Extract text from decompressed stream
	extractor := NewExtractor()
	extractor.RegisterFont("/F1", "Helvetica", "Type1")

	fragments, err := extractor.ExtractFromBytes(decompressed.Bytes())
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	if len(fragments) != 1 {
		t.Fatalf("Expected 1 fragment, got %d", len(fragments))
	}

	if fragments[0].Text != "Compressed Text" {
		t.Errorf("Expected 'Compressed Text', got %q", fragments[0].Text)
	}
}

// TestComplexLayout tests extraction from complex layout
func TestComplexLayout(t *testing.T) {
	contentStream := []byte(`q
1 0 0 1 50 50 cm
BT
/F1 24 Tf
1 0 0 1 0 700 Tm
(Heading) Tj
ET
Q

BT
/F1 12 Tf
72 650 Td
(Paragraph line 1) Tj
0 -14 Td
(Paragraph line 2) Tj
0 -14 Td
(Paragraph line 3) Tj
ET

BT
/F1 12 Tf
72 580 Td
[(Adjusted ) -200 (spacing) -300 (text)] TJ
ET`)

	extractor := NewExtractor()
	extractor.RegisterFont("/F1", "Helvetica", "Type1")

	fragments, err := extractor.ExtractFromBytes(contentStream)
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	if len(fragments) < 5 {
		t.Errorf("Expected at least 5 fragments, got %d", len(fragments))
	}

	// Check that we got the heading
	foundHeading := false
	for _, frag := range fragments {
		if frag.Text == "Heading" {
			foundHeading = true
			if frag.FontSize != 24 {
				t.Errorf("Heading should have font size 24, got %f", frag.FontSize)
			}
			// Position should include CTM transformation
			if frag.X != 50 {
				t.Errorf("Heading X should include CTM translation (50), got %f", frag.X)
			}
		}
	}

	if !foundHeading {
		t.Error("Should have found 'Heading' fragment")
	}

	// Check full text includes all parts
	fullText := extractor.GetText()
	requiredStrings := []string{"Heading", "Paragraph line 1", "Adjusted", "spacing"}
	for _, s := range requiredStrings {
		if !strings.Contains(fullText, s) {
			t.Errorf("Full text should contain %q, got: %q", s, fullText)
		}
	}

	t.Logf("Extracted %d fragments, full text length: %d", len(fragments), len(fullText))
}

// TestMultipleFonts tests handling multiple fonts in same document
func TestMultipleFonts(t *testing.T) {
	contentStream := []byte(`BT
/Helvetica 12 Tf
(Normal text) Tj
/Times-Bold 14 Tf
0 -16 Td
(Bold text) Tj
/Courier 10 Tf
0 -16 Td
(Monospace text) Tj
ET`)

	extractor := NewExtractor()
	// Auto-registration should handle these

	fragments, err := extractor.ExtractFromBytes(contentStream)
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	if len(fragments) != 3 {
		t.Fatalf("Expected 3 fragments, got %d", len(fragments))
	}

	// Check that different fonts were used
	fonts := make(map[string]bool)
	for _, frag := range fragments {
		fonts[frag.FontName] = true
	}

	if len(fonts) != 3 {
		t.Errorf("Expected 3 different fonts, got %d: %v", len(fonts), fonts)
	}
}

// TestGraphicsStateStack tests q/Q state save/restore
func TestGraphicsStateStack(t *testing.T) {
	contentStream := []byte(`BT
/F1 10 Tf
(Size 10) Tj
q
/F1 20 Tf
0 -22 Td
(Size 20) Tj
Q
0 -12 Td
(Size 10 again) Tj
ET`)

	extractor := NewExtractor()
	extractor.RegisterFont("/F1", "Helvetica", "Type1")

	fragments, err := extractor.ExtractFromBytes(contentStream)
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	if len(fragments) != 3 {
		t.Fatalf("Expected 3 fragments, got %d", len(fragments))
	}

	// First fragment: size 10
	if fragments[0].FontSize != 10 {
		t.Errorf("First fragment should be size 10, got %f", fragments[0].FontSize)
	}

	// Second fragment: size 20 (within q/Q)
	if fragments[1].FontSize != 20 {
		t.Errorf("Second fragment should be size 20, got %f", fragments[1].FontSize)
	}

	// Third fragment: size 10 (restored after Q)
	if fragments[2].FontSize != 10 {
		t.Errorf("Third fragment should be size 10 (restored), got %f", fragments[2].FontSize)
	}
}

// TestTextPositioning tests various text positioning operators
func TestTextPositioning(t *testing.T) {
	contentStream := []byte(`BT
/F1 12 Tf
100 700 Td
(Td operator) Tj
0 -14 TD
(TD operator with leading) Tj
T*
(T* next line) Tj
ET`)

	extractor := NewExtractor()
	extractor.RegisterFont("/F1", "Helvetica", "Type1")

	fragments, err := extractor.ExtractFromBytes(contentStream)
	if err != nil {
		t.Fatalf("Failed to extract text: %v", err)
	}

	if len(fragments) != 3 {
		t.Fatalf("Expected 3 fragments, got %d", len(fragments))
	}

	// All should be extracted
	texts := []string{}
	for _, frag := range fragments {
		texts = append(texts, frag.Text)
	}

	expectedTexts := []string{"Td operator", "TD operator with leading", "T* next line"}
	for i, expected := range expectedTexts {
		if i >= len(texts) || texts[i] != expected {
			t.Errorf("Fragment %d: expected %q, got %q", i, expected, texts[i])
		}
	}
}

// TestEmptyStreams tests handling of empty or whitespace-only streams
func TestEmptyStreams(t *testing.T) {
	tests := []struct {
		name   string
		stream []byte
	}{
		{"empty", []byte{}},
		{"whitespace", []byte("   \n\t\r  ")},
		{"just BT/ET", []byte("BT ET")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewExtractor()
			fragments, err := extractor.ExtractFromBytes(tt.stream)
			if err != nil {
				t.Fatalf("Should handle %s gracefully: %v", tt.name, err)
			}

			if len(fragments) != 0 {
				t.Errorf("Expected 0 fragments for %s, got %d", tt.name, len(fragments))
			}
		})
	}
}

// BenchmarkTextExtraction benchmarks the full extraction pipeline
func BenchmarkTextExtraction(b *testing.B) {
	contentStream := []byte(`BT
/F1 12 Tf
72 720 Td
(The quick brown fox jumps over the lazy dog.) Tj
0 -14 Td
(Pack my box with five dozen liquor jugs.) Tj
0 -14 Td
(How vexingly quick daft zebras jump!) Tj
ET`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor := NewExtractor()
		extractor.RegisterFont("/F1", "Helvetica", "Type1")
		_, _ = extractor.ExtractFromBytes(contentStream)
	}
}
