//go:build !ocr

// Package ocr provides OCR (Optical Character Recognition) capabilities
// for extracting text from images in scanned PDFs.
//
// This is the stub implementation used when the "ocr" build tag is not set.
// All functions return ErrOCRNotEnabled.
//
// To enable OCR, rebuild with the "ocr" build tag:
//
//	go build -tags ocr
//
// This requires Tesseract to be installed. On macOS:
//
//	brew install tesseract
//
// On Ubuntu/Debian:
//
//	apt-get install tesseract-ocr
package ocr

import "errors"

// ErrOCRNotEnabled is returned when OCR functions are called but OCR support
// was not compiled in. Rebuild with -tags ocr to enable OCR support.
var ErrOCRNotEnabled = errors.New("OCR support not enabled; rebuild with -tags ocr")

// PageSegMode represents page segmentation modes for OCR.
// These control how Tesseract analyzes the page layout.
type PageSegMode int

// Page segmentation modes (matching the OCR-enabled implementation).
const (
	PSM_OSD_ONLY               PageSegMode = 0  // Orientation and script detection only
	PSM_AUTO_OSD               PageSegMode = 1  // Automatic with OSD
	PSM_AUTO_ONLY              PageSegMode = 2  // Automatic, no OSD or OCR
	PSM_AUTO                   PageSegMode = 3  // Fully automatic (default)
	PSM_SINGLE_COLUMN          PageSegMode = 4  // Single column of variable sizes
	PSM_SINGLE_BLOCK_VERT_TEXT PageSegMode = 5  // Single uniform block of vertically aligned text
	PSM_SINGLE_BLOCK           PageSegMode = 6  // Single uniform block of text
	PSM_SINGLE_LINE            PageSegMode = 7  // Single text line
	PSM_SINGLE_WORD            PageSegMode = 8  // Single word
	PSM_CIRCLE_WORD            PageSegMode = 9  // Single word in a circle
	PSM_SINGLE_CHAR            PageSegMode = 10 // Single character
	PSM_SPARSE_TEXT            PageSegMode = 11 // Find as much text as possible
	PSM_SPARSE_TEXT_OSD        PageSegMode = 12 // Sparse text with OSD
	PSM_RAW_LINE               PageSegMode = 13 // Treat image as single text line
)

// Client is a stub OCR client that returns errors for all operations.
type Client struct{}

// New returns an error indicating OCR support is not enabled.
// To enable OCR, rebuild with: go build -tags ocr
func New() (*Client, error) {
	return nil, ErrOCRNotEnabled
}

// Close is a no-op for the stub client.
// It is safe to call on a nil client.
func (c *Client) Close() error {
	return nil
}

// RecognizeImage returns an error indicating OCR support is not enabled.
func (c *Client) RecognizeImage(imageData []byte) (string, error) {
	return "", ErrOCRNotEnabled
}

// SetLanguage returns an error indicating OCR support is not enabled.
func (c *Client) SetLanguage(lang string) error {
	return ErrOCRNotEnabled
}

// SetPageSegMode returns an error indicating OCR support is not enabled.
func (c *Client) SetPageSegMode(mode PageSegMode) error {
	return ErrOCRNotEnabled
}
