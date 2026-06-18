//go:build ocr

package imagecodec

import (
	"image"
	"os"
	"testing"
)

// mono.jbig2: a 16x16 image, left half black, right half white (jbig2enc).
func TestDecodeJBIG2(t *testing.T) {
	data, err := os.ReadFile("testdata/mono.jbig2")
	if err != nil {
		t.Fatal(err)
	}
	img, err := DecodeJBIG2(data, nil, 16, 16)
	if err != nil {
		t.Fatalf("DecodeJBIG2: %v", err)
	}
	g, ok := img.(*image.Gray)
	if !ok {
		t.Fatalf("expected *image.Gray, got %T", img)
	}
	if b := g.Bounds(); b.Dx() != 16 || b.Dy() != 16 {
		t.Fatalf("dims = %dx%d, want 16x16", b.Dx(), b.Dy())
	}
	if v := g.GrayAt(0, 0).Y; v != 0 {
		t.Errorf("left pixel = %d, want 0 (black)", v)
	}
	if v := g.GrayAt(15, 0).Y; v != 255 {
		t.Errorf("right pixel = %d, want 255 (white)", v)
	}
}

// red.jp2: a 64x64 solid-red image (lossless, openjpeg).
func TestDecodeJPEG2000(t *testing.T) {
	data, err := os.ReadFile("testdata/red.jp2")
	if err != nil {
		t.Fatal(err)
	}
	img, err := DecodeJPEG2000(data)
	if err != nil {
		t.Fatalf("DecodeJPEG2000: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 64 || b.Dy() != 64 {
		t.Fatalf("dims = %dx%d, want 64x64", b.Dx(), b.Dy())
	}
	r, g, b, _ := img.At(0, 0).RGBA()
	if r>>8 < 200 || g>>8 > 60 || b>>8 > 60 {
		t.Errorf("pixel = (%d,%d,%d), want ~red", r>>8, g>>8, b>>8)
	}
}
