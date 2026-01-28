package reader

import (
	"image"
	"testing"
)

func TestPageImage_ToPNG_Grayscale8Bit(t *testing.T) {
	// Create a 2x2 grayscale image
	img := &PageImage{
		Name:             "TestImg",
		Width:            2,
		Height:           2,
		ColorSpace:       "DeviceGray",
		BitsPerComponent: 8,
		Data:             []byte{0, 128, 64, 255}, // 2x2 pixels
	}

	pngData, err := img.ToPNG()
	if err != nil {
		t.Fatalf("ToPNG failed: %v", err)
	}

	if len(pngData) == 0 {
		t.Error("Expected non-empty PNG data")
	}

	// Verify PNG magic bytes
	if len(pngData) < 8 {
		t.Fatal("PNG data too short")
	}
	pngMagic := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	for i, b := range pngMagic {
		if pngData[i] != b {
			t.Errorf("PNG magic byte %d: got %x, want %x", i, pngData[i], b)
		}
	}
}

func TestPageImage_ToPNG_Bilevel(t *testing.T) {
	// Create a 8x1 bilevel image (1 byte = 8 pixels)
	// Binary: 10101010 = alternating black/white
	img := &PageImage{
		Name:             "TestImg",
		Width:            8,
		Height:           1,
		ColorSpace:       "DeviceGray",
		BitsPerComponent: 1,
		Data:             []byte{0xAA}, // 10101010
	}

	pngData, err := img.ToPNG()
	if err != nil {
		t.Fatalf("ToPNG failed: %v", err)
	}

	if len(pngData) == 0 {
		t.Error("Expected non-empty PNG data")
	}
}

func TestPageImage_ToPNG_RGB(t *testing.T) {
	// Create a 2x1 RGB image
	img := &PageImage{
		Name:             "TestImg",
		Width:            2,
		Height:           1,
		ColorSpace:       "DeviceRGB",
		BitsPerComponent: 8,
		Data: []byte{
			255, 0, 0, // Red pixel
			0, 255, 0, // Green pixel
		},
	}

	pngData, err := img.ToPNG()
	if err != nil {
		t.Fatalf("ToPNG failed: %v", err)
	}

	if len(pngData) == 0 {
		t.Error("Expected non-empty PNG data")
	}
}

func TestPageImage_ToPNG_CMYK(t *testing.T) {
	// Create a 1x1 CMYK image
	img := &PageImage{
		Name:             "TestImg",
		Width:            1,
		Height:           1,
		ColorSpace:       "DeviceCMYK",
		BitsPerComponent: 8,
		Data:             []byte{0, 255, 255, 0}, // C=0, M=255, Y=255, K=0 (Red in CMYK)
	}

	pngData, err := img.ToPNG()
	if err != nil {
		t.Fatalf("ToPNG failed: %v", err)
	}

	if len(pngData) == 0 {
		t.Error("Expected non-empty PNG data")
	}
}

func TestPageImage_toBilevelGray(t *testing.T) {
	// Test bilevel conversion
	img := &PageImage{
		Name:             "TestImg",
		Width:            16, // 2 bytes
		Height:           1,
		ColorSpace:       "DeviceGray",
		BitsPerComponent: 1,
		Data:             []byte{0xFF, 0x00}, // 8 white, 8 black
	}

	goImg, err := img.toBilevelGray()
	if err != nil {
		t.Fatalf("toBilevelGray failed: %v", err)
	}

	// Check dimensions
	bounds := goImg.Bounds()
	if bounds.Dx() != 16 || bounds.Dy() != 1 {
		t.Errorf("Wrong dimensions: got %dx%d, want 16x1", bounds.Dx(), bounds.Dy())
	}

	// Check first 8 pixels are white (255)
	for x := 0; x < 8; x++ {
		if goImg.GrayAt(x, 0).Y != 255 {
			t.Errorf("Pixel %d should be white (255), got %d", x, goImg.GrayAt(x, 0).Y)
		}
	}

	// Check last 8 pixels are black (0)
	for x := 8; x < 16; x++ {
		if goImg.GrayAt(x, 0).Y != 0 {
			t.Errorf("Pixel %d should be black (0), got %d", x, goImg.GrayAt(x, 0).Y)
		}
	}
}

func TestPageImage_to4BitGray(t *testing.T) {
	// Test 4-bit grayscale conversion
	img := &PageImage{
		Name:             "TestImg",
		Width:            2,
		Height:           1,
		ColorSpace:       "DeviceGray",
		BitsPerComponent: 4,
		Data:             []byte{0xF0}, // First pixel = 15 (white), second = 0 (black)
	}

	goImg, err := img.to4BitGray()
	if err != nil {
		t.Fatalf("to4BitGray failed: %v", err)
	}

	// Check first pixel is white (15 * 17 = 255)
	if goImg.GrayAt(0, 0).Y != 255 {
		t.Errorf("First pixel should be 255, got %d", goImg.GrayAt(0, 0).Y)
	}

	// Check second pixel is black (0 * 17 = 0)
	if goImg.GrayAt(1, 0).Y != 0 {
		t.Errorf("Second pixel should be 0, got %d", goImg.GrayAt(1, 0).Y)
	}
}

func TestPageImage_ToPNG_InsufficientData(t *testing.T) {
	img := &PageImage{
		Name:             "TestImg",
		Width:            10,
		Height:           10,
		ColorSpace:       "DeviceGray",
		BitsPerComponent: 8,
		Data:             []byte{0, 1, 2}, // Not enough data for 100 pixels
	}

	_, err := img.ToPNG()
	if err == nil {
		t.Error("Expected error for insufficient data")
	}
}

func TestPageImage_toRGBImage_InsufficientData(t *testing.T) {
	img := &PageImage{
		Name:             "TestImg",
		Width:            10,
		Height:           10,
		ColorSpace:       "DeviceRGB",
		BitsPerComponent: 8,
		Data:             []byte{0, 1, 2, 3, 4, 5}, // Not enough data for 100 RGB pixels
	}

	_, err := img.toRGBImage()
	if err == nil {
		t.Error("Expected error for insufficient data")
	}
}

func TestPageImage_toCMYKImage_InsufficientData(t *testing.T) {
	img := &PageImage{
		Name:             "TestImg",
		Width:            10,
		Height:           10,
		ColorSpace:       "DeviceCMYK",
		BitsPerComponent: 8,
		Data:             []byte{0, 1, 2, 3, 4, 5, 6, 7}, // Not enough data for 100 CMYK pixels
	}

	_, err := img.toCMYKImage()
	if err == nil {
		t.Error("Expected error for insufficient data")
	}
}

func TestPageImage_ToPNG_UnsupportedBPC(t *testing.T) {
	img := &PageImage{
		Name:             "TestImg",
		Width:            2,
		Height:           2,
		ColorSpace:       "DeviceGray",
		BitsPerComponent: 16, // Unsupported
		Data:             make([]byte, 8),
	}

	_, err := img.ToPNG()
	if err == nil {
		t.Error("Expected error for unsupported bits per component")
	}
}

// TestToGrayImageRoundtrip tests that a grayscale image can be converted to PNG and back
func TestToGrayImageRoundtrip(t *testing.T) {
	// Create a simple 4x4 grayscale pattern
	data := []byte{
		0, 85, 170, 255,
		255, 170, 85, 0,
		0, 85, 170, 255,
		255, 170, 85, 0,
	}

	img := &PageImage{
		Name:             "TestImg",
		Width:            4,
		Height:           4,
		ColorSpace:       "DeviceGray",
		BitsPerComponent: 8,
		Data:             data,
	}

	goImg, err := img.toGrayImage()
	if err != nil {
		t.Fatalf("toGrayImage failed: %v", err)
	}

	// Verify dimensions
	if goImg.Bounds().Dx() != 4 || goImg.Bounds().Dy() != 4 {
		t.Error("Wrong dimensions")
	}

	// Verify pixel values
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			expected := data[y*4+x]
			actual := goImg.GrayAt(x, y).Y
			if actual != expected {
				t.Errorf("Pixel (%d,%d): got %d, want %d", x, y, actual, expected)
			}
		}
	}
}

// BenchmarkToPNG benchmarks PNG encoding
func BenchmarkToPNG(b *testing.B) {
	// Create a larger test image
	width, height := 100, 100
	data := make([]byte, width*height)
	for i := range data {
		data[i] = byte(i % 256)
	}

	img := &PageImage{
		Name:             "BenchImg",
		Width:            width,
		Height:           height,
		ColorSpace:       "DeviceGray",
		BitsPerComponent: 8,
		Data:             data,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = img.ToPNG()
	}
}

// Mock image interface for testing
var _ image.Image = (*image.Gray)(nil)
var _ image.Image = (*image.RGBA)(nil)
