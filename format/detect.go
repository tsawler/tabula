// Package format provides file format detection for the tabula library.
package format

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
)

// Format represents a supported document format.
type Format int

const (
	// Unknown indicates an unrecognized format.
	Unknown Format = iota
	// PDF indicates a PDF document.
	PDF
	// DOCX indicates a Microsoft Word (.docx) document.
	DOCX
	// ODT indicates an OpenDocument Text (.odt) document.
	ODT
	// XLSX indicates a Microsoft Excel (.xlsx) document.
	XLSX
)

// String returns the string representation of the format.
func (f Format) String() string {
	switch f {
	case PDF:
		return "PDF"
	case DOCX:
		return "DOCX"
	case ODT:
		return "ODT"
	case XLSX:
		return "XLSX"
	default:
		return "Unknown"
	}
}

// Extension returns the typical file extension for the format.
func (f Format) Extension() string {
	switch f {
	case PDF:
		return ".pdf"
	case DOCX:
		return ".docx"
	case ODT:
		return ".odt"
	case XLSX:
		return ".xlsx"
	default:
		return ""
	}
}

// Detect determines file format from filename extension.
func Detect(filename string) Format {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return PDF
	case ".docx":
		return DOCX
	case ".odt":
		return ODT
	case ".xlsx":
		return XLSX
	default:
		return Unknown
	}
}

// DetectFromMagic checks file magic bytes to determine format.
// This provides more reliable detection than extension-based detection.
// Returns Unknown if the format cannot be determined from magic bytes alone.
func DetectFromMagic(data []byte) Format {
	if len(data) < 4 {
		return Unknown
	}

	// PDF magic: %PDF
	if data[0] == '%' && data[1] == 'P' && data[2] == 'D' && data[3] == 'F' {
		return PDF
	}

	// ZIP magic (DOCX is a ZIP archive): PK\x03\x04
	if data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04 {
		// Could be DOCX, XLSX, PPTX, or other ZIP-based format
		// Return Unknown here - caller should use DetectFromReader for ZIP files
		return Unknown
	}

	return Unknown
}

// DetectFromReader inspects the content to determine format.
// This is more reliable than extension-based detection and can
// distinguish between different ZIP-based formats (DOCX, XLSX, PPTX).
func DetectFromReader(r io.ReaderAt, size int64) (Format, error) {
	// Read magic bytes first
	magic := make([]byte, 8)
	n, err := r.ReadAt(magic, 0)
	if err != nil && err != io.EOF {
		return Unknown, err
	}
	magic = magic[:n]

	// Check for PDF
	if len(magic) >= 4 && magic[0] == '%' && magic[1] == 'P' && magic[2] == 'D' && magic[3] == 'F' {
		return PDF, nil
	}

	// Check for ZIP-based format
	if len(magic) >= 4 && magic[0] == 0x50 && magic[1] == 0x4B && magic[2] == 0x03 && magic[3] == 0x04 {
		// It's a ZIP archive - check contents to determine specific format
		return detectZIPFormat(r, size)
	}

	return Unknown, nil
}

// detectZIPFormat inspects a ZIP archive to determine if it's DOCX, XLSX, PPTX, ODT, etc.
func detectZIPFormat(r io.ReaderAt, size int64) (Format, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return Unknown, err
	}

	// Check for OpenDocument Format first (has mimetype file at the start)
	for _, f := range zr.File {
		if f.Name == "mimetype" {
			rc, err := f.Open()
			if err == nil {
				data := make([]byte, 256)
				n, _ := rc.Read(data)
				rc.Close()
				mimeType := string(data[:n])
				if strings.Contains(mimeType, "application/vnd.oasis.opendocument.text") {
					return ODT, nil
				}
			}
		}
	}

	// Check for Office Open XML markers
	for _, f := range zr.File {
		switch {
		case f.Name == "[Content_Types].xml":
			// This is an OOXML file - check for specific format markers
			continue
		case strings.HasPrefix(f.Name, "word/"):
			return DOCX, nil
		case strings.HasPrefix(f.Name, "xl/"):
			return XLSX, nil
		case strings.HasPrefix(f.Name, "ppt/"):
			// Future: return PPTX
			return Unknown, nil
		}
	}

	return Unknown, nil
}
