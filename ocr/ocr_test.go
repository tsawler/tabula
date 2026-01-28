package ocr

import (
	"image"
	"image/color"
	"image/png"
	"bytes"
	"testing"
)

// createTestPNG creates a simple PNG image with text-like patterns for testing.
// This is a very basic image that OCR might or might not recognize.
func createTestPNG(width, height int) []byte {
	img := image.NewGray(image.Rect(0, 0, width, height))

	// Fill with white
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Draw some black pixels (simple pattern)
	for x := 10; x < 50; x++ {
		for y := 10; y < 30; y++ {
			img.Set(x, y, color.Black)
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestNew(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Skipf("Tesseract not available: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Error("Expected non-nil client")
	}
}

func TestRecognizeImage(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Skipf("Tesseract not available: %v", err)
	}
	defer client.Close()

	pngData := createTestPNG(100, 50)

	// We don't check the actual text since our test image is just a rectangle
	// We just verify the method doesn't crash
	_, err = client.RecognizeImage(pngData)
	if err != nil {
		t.Errorf("RecognizeImage failed: %v", err)
	}
}

func TestSetLanguage(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Skipf("Tesseract not available: %v", err)
	}
	defer client.Close()

	// English should always be available
	err = client.SetLanguage("eng")
	if err != nil {
		t.Errorf("SetLanguage failed: %v", err)
	}
}

func TestClose(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Skipf("Tesseract not available: %v", err)
	}

	// First close should succeed
	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Second close should also be safe (nil client)
	client.client = nil
	err = client.Close()
	if err != nil {
		t.Errorf("Close on nil client failed: %v", err)
	}
}
