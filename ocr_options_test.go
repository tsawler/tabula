package tabula

import "testing"

func TestEstimateImageDPI(t *testing.T) {
	// 1700x2200 px on a US-Letter page (612x792 pt) = 200 DPI.
	if got := estimateImageDPI(1700, 2200, 612, 792); got != 200 {
		t.Errorf("estimateImageDPI = %d, want 200", got)
	}
	// Unknown page size → 0 (no estimate).
	if got := estimateImageDPI(1000, 1000, 0, 0); got != 0 {
		t.Errorf("estimateImageDPI(unknown page) = %d, want 0", got)
	}
}

func TestOCRUpscale(t *testing.T) {
	cases := []struct {
		dpi  int
		want float64
	}{
		{0, 1.0},   // unknown
		{40, 1.0},  // implausibly low → untouched
		{100, 3.0}, // 300/100 = 3 (at cap)
		{150, 2.0}, // 300/150
		{200, 1.5}, // 300/200
		{250, 1.0}, // already near target
		{400, 1.0}, // above target
	}
	for _, c := range cases {
		if got := ocrUpscale(c.dpi); got != c.want {
			t.Errorf("ocrUpscale(%d) = %v, want %v", c.dpi, got, c.want)
		}
	}
}
