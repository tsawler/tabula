package reader

import (
	"bytes"
	"fmt"
	"testing"
)

// buildPDF assembles a PDF from object bodies (object N is bodies[N-1]) with a
// correct classic xref table and trailer.
func buildPDF(bodies [][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(bodies))
	for i, body := range bodies {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n", i+1)
		buf.Write(body)
		buf.WriteString("\nendobj\n")
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(bodies)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(bodies)+1, xref)
	return buf.Bytes()
}

func grayImageObj() []byte {
	return []byte("<< /Type /XObject /Subtype /Image /Width 1 /Height 1 /BitsPerComponent 8 /ColorSpace /DeviceGray /Length 1 >>\nstream\n\x00\nendstream")
}

// TestExtractPageImages_FormRecursionAndOrder verifies images nested in a Form
// XObject are found (#4) and that extraction order is deterministic by XObject
// name (#3): the form "Frm" sorts before the image "Img", so its nested image
// comes first.
func TestExtractPageImages_FormRecursionAndOrder(t *testing.T) {
	bodies := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << /XObject << /Img 4 0 R /Frm 5 0 R >> >> >>"),
		grayImageObj(), // 4: top-level image "Img"
		[]byte("<< /Type /XObject /Subtype /Form /BBox [0 0 1 1] /Resources << /XObject << /Nested 6 0 R >> >> /Length 0 >>\nstream\nendstream"), // 5: form
		grayImageObj(), // 6: image nested in the form
	}
	path := createTempPDF(t, string(buildPDF(bodies)))

	r, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer r.Close()

	page, err := r.GetPage(0)
	if err != nil {
		t.Fatalf("GetPage(0): %v", err)
	}
	imgs, err := r.ExtractPageImages(page)
	if err != nil {
		t.Fatalf("ExtractPageImages: %v", err)
	}
	if len(imgs) != 2 {
		t.Fatalf("found %d images, want 2 (top-level + form-nested)", len(imgs))
	}
	// Deterministic order: "Frm" < "Img", so the form's "Nested" image is first.
	if imgs[0].Name != "Nested" || imgs[1].Name != "Img" {
		t.Errorf("order = [%s, %s], want [Nested, Img]", imgs[0].Name, imgs[1].Name)
	}
}

// TestRebuildXRefFromBrokenStartxref verifies recovery when the cross-reference
// table can't be parsed (here, a bogus startxref offset): the reader rebuilds
// the table by scanning for objects and recovering the catalog.
func TestRebuildXRefFromBrokenStartxref(t *testing.T) {
	bodies := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>"),
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n")
	for i, b := range bodies {
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, b)
	}
	buf.WriteString("startxref\n999999\n%%EOF") // deliberately wrong offset
	path := createTempPDF(t, buf.String())

	r, err := Open(path)
	if err != nil {
		t.Fatalf("open with broken xref: %v", err)
	}
	defer r.Close()
	if r.xrefTable == nil || len(r.xrefTable.Entries) < 3 {
		t.Fatalf("rebuild did not recover objects: %+v", r.xrefTable)
	}
	if _, err := r.GetPage(0); err != nil {
		t.Fatalf("GetPage(0) after rebuild: %v", err)
	}
}
