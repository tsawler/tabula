package font

import "testing"

func TestIdentityWidthUsesCIDInsteadOfUnicode(t *testing.T) {
	f := NewFont("F1", "Subset", "Type0")
	f.Encoding = "Identity-H"
	f.cidWidths = &CIDFont{DW: 1000, W: []WidthRange{{StartCID: 7, EndCID: 7, Width: 250}}}
	if got := f.GetEncodedWidth([]byte{0, 7, 0, 9}); got != 1250 {
		t.Fatalf("width = %v, want 1250", got)
	}
}
