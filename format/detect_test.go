package format

import (
	"bytes"
	"os"
	"testing"
)

func TestFormat_String(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{PDF, "PDF"},
		{DOCX, "DOCX"},
		{ODT, "ODT"},
		{XLSX, "XLSX"},
		{PPTX, "PPTX"},
		{HTML, "HTML"},
		{Unknown, "Unknown"},
		{Format(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.format.String(); got != tt.want {
			t.Errorf("Format(%d).String() = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestFormat_Extension(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{PDF, ".pdf"},
		{DOCX, ".docx"},
		{ODT, ".odt"},
		{XLSX, ".xlsx"},
		{PPTX, ".pptx"},
		{HTML, ".html"},
		{Unknown, ""},
	}

	for _, tt := range tests {
		if got := tt.format.Extension(); got != tt.want {
			t.Errorf("Format(%d).Extension() = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestDetect(t *testing.T) {
	tests := []struct {
		filename string
		want     Format
	}{
		{"document.pdf", PDF},
		{"document.PDF", PDF},
		{"document.Pdf", PDF},
		{"document.docx", DOCX},
		{"document.DOCX", DOCX},
		{"document.Docx", DOCX},
		{"document.odt", ODT},
		{"document.ODT", ODT},
		{"document.Odt", ODT},
		{"document.xlsx", XLSX},
		{"document.XLSX", XLSX},
		{"document.Xlsx", XLSX},
		{"document.pptx", PPTX},
		{"document.PPTX", PPTX},
		{"document.Pptx", PPTX},
		{"document.html", HTML},
		{"document.HTML", HTML},
		{"document.Html", HTML},
		{"document.htm", HTML},
		{"document.HTM", HTML},
		{"document.txt", Unknown},
		{"document", Unknown},
		{"", Unknown},
		{"/path/to/file.pdf", PDF},
		{"/path/to/file.docx", DOCX},
		{"/path/to/file.odt", ODT},
		{"/path/to/file.xlsx", XLSX},
		{"/path/to/file.pptx", PPTX},
		{"/path/to/file.html", HTML},
	}

	for _, tt := range tests {
		if got := Detect(tt.filename); got != tt.want {
			t.Errorf("Detect(%q) = %v, want %v", tt.filename, got, tt.want)
		}
	}
}

func TestDetectFromMagic(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want Format
	}{
		{
			name: "PDF magic bytes",
			data: []byte("%PDF-1.4"),
			want: PDF,
		},
		{
			name: "PDF minimal",
			data: []byte("%PDF"),
			want: PDF,
		},
		{
			name: "ZIP magic bytes (DOCX/XLSX/PPTX)",
			data: []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00},
			want: Unknown, // ZIP needs further inspection
		},
		{
			name: "HTML with DOCTYPE",
			data: []byte("<!DOCTYPE html>\n<html>"),
			want: HTML,
		},
		{
			name: "HTML with html tag",
			data: []byte("<html><head>"),
			want: HTML,
		},
		{
			name: "HTML with whitespace before DOCTYPE",
			data: []byte("  \n  <!DOCTYPE HTML PUBLIC"),
			want: HTML,
		},
		{
			name: "empty data",
			data: []byte{},
			want: Unknown,
		},
		{
			name: "short data",
			data: []byte{0x50, 0x4B},
			want: Unknown,
		},
		{
			name: "random data",
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			want: Unknown,
		},
		{
			name: "text file",
			data: []byte("Hello, World!"),
			want: Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectFromMagic(tt.data); got != tt.want {
				t.Errorf("DetectFromMagic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectFromReader_PDF(t *testing.T) {
	// Create a minimal PDF-like data
	data := []byte("%PDF-1.4\n%%EOF")
	r := bytes.NewReader(data)

	format, err := DetectFromReader(r, int64(len(data)))
	if err != nil {
		t.Fatalf("DetectFromReader() error = %v", err)
	}
	if format != PDF {
		t.Errorf("DetectFromReader() = %v, want PDF", format)
	}
}

func TestDetectFromReader_HTML(t *testing.T) {
	// Create minimal HTML data
	data := []byte("<!DOCTYPE html>\n<html><head><title>Test</title></head><body></body></html>")
	r := bytes.NewReader(data)

	format, err := DetectFromReader(r, int64(len(data)))
	if err != nil {
		t.Fatalf("DetectFromReader() error = %v", err)
	}
	if format != HTML {
		t.Errorf("DetectFromReader() = %v, want HTML", format)
	}
}

func TestDetectFromReader_Unknown(t *testing.T) {
	data := []byte("Hello, World! This is plain text.")
	r := bytes.NewReader(data)

	format, err := DetectFromReader(r, int64(len(data)))
	if err != nil {
		t.Fatalf("DetectFromReader() error = %v", err)
	}
	if format != Unknown {
		t.Errorf("DetectFromReader() = %v, want Unknown", format)
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 1, -1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		if got := min(tt.a, tt.b); got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestDetectFromReader_ZIP_DOCX(t *testing.T) {
	// Test with actual DOCX file if available
	docxPath := "../docx/testdata/hills.docx"
	f, err := os.Open(docxPath)
	if err != nil {
		t.Skip("DOCX test file not available:", docxPath)
	}
	defer f.Close()

	stat, _ := f.Stat()
	format, err := DetectFromReader(f, stat.Size())
	if err != nil {
		t.Fatalf("DetectFromReader() error = %v", err)
	}
	if format != DOCX {
		t.Errorf("DetectFromReader() = %v, want DOCX", format)
	}
}

func TestDetectFromReader_ZIP_XLSX(t *testing.T) {
	// Test with actual XLSX file if available
	xlsxPath := "../xlsx/testdata/basic.xlsx"
	f, err := os.Open(xlsxPath)
	if err != nil {
		t.Skip("XLSX test file not available:", xlsxPath)
	}
	defer f.Close()

	stat, _ := f.Stat()
	format, err := DetectFromReader(f, stat.Size())
	if err != nil {
		t.Fatalf("DetectFromReader() error = %v", err)
	}
	if format != XLSX {
		t.Errorf("DetectFromReader() = %v, want XLSX", format)
	}
}

func TestDetectFromReader_ZIP_PPTX(t *testing.T) {
	// Test with actual PPTX file if available
	pptxPath := "../pptx/testdata/basic.pptx"
	f, err := os.Open(pptxPath)
	if err != nil {
		t.Skip("PPTX test file not available:", pptxPath)
	}
	defer f.Close()

	stat, _ := f.Stat()
	format, err := DetectFromReader(f, stat.Size())
	if err != nil {
		t.Fatalf("DetectFromReader() error = %v", err)
	}
	if format != PPTX {
		t.Errorf("DetectFromReader() = %v, want PPTX", format)
	}
}

func TestDetectFromReader_ZIP_ODT(t *testing.T) {
	// Test with actual ODT file if available
	odtPath := "../odt/testdata/sample.odt"
	f, err := os.Open(odtPath)
	if err != nil {
		t.Skip("ODT test file not available:", odtPath)
	}
	defer f.Close()

	stat, _ := f.Stat()
	format, err := DetectFromReader(f, stat.Size())
	if err != nil {
		t.Fatalf("DetectFromReader() error = %v", err)
	}
	if format != ODT {
		t.Errorf("DetectFromReader() = %v, want ODT", format)
	}
}

func TestDetectFromReader_ZIP_EPUB(t *testing.T) {
	// Test with actual EPUB file if available
	epubPath := "../epubdoc/testdata/Frankenstein.epub"
	f, err := os.Open(epubPath)
	if err != nil {
		t.Skip("EPUB test file not available:", epubPath)
	}
	defer f.Close()

	stat, _ := f.Stat()
	format, err := DetectFromReader(f, stat.Size())
	if err != nil {
		t.Fatalf("DetectFromReader() error = %v", err)
	}
	if format != EPUB {
		t.Errorf("DetectFromReader() = %v, want EPUB", format)
	}
}
