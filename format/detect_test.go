package format

import (
	"bytes"
	"testing"
)

func TestFormat_String(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{PDF, "PDF"},
		{DOCX, "DOCX"},
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
		{"document.txt", Unknown},
		{"document.xlsx", Unknown},
		{"document.pptx", Unknown},
		{"document", Unknown},
		{"", Unknown},
		{"/path/to/file.pdf", PDF},
		{"/path/to/file.docx", DOCX},
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
