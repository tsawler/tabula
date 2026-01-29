//go:build ocr

// Package ocr provides OCR (Optical Character Recognition) capabilities
// for extracting text from images in scanned PDFs.
//
// This package wraps the Tesseract OCR engine via gosseract. It requires
// Tesseract to be installed on the system. On macOS, install via:
//
//	brew install tesseract
//
// On Ubuntu/Debian:
//
//	apt-get install tesseract-ocr
//
// OCR support is optional and requires the "ocr" build tag:
//
//	go build -tags ocr
//
// Without the build tag, OCR functions return ErrOCRNotEnabled.
package ocr

import (
	"fmt"
	"strings"

	"github.com/otiai10/gosseract/v2"
)

// PageSegMode represents page segmentation modes for OCR.
// These control how Tesseract analyzes the page layout.
type PageSegMode int

// Page segmentation modes (matching gosseract constants).
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

// Client wraps Tesseract for OCR operations.
type Client struct {
	client *gosseract.Client
}

// New creates a new OCR client.
// The client should be closed when no longer needed to release resources.
func New() (*Client, error) {
	client := gosseract.NewClient()
	return &Client{client: client}, nil
}

// Close releases OCR resources.
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// RecognizeImage performs OCR on image data (PNG, TIFF, JPEG, etc.).
// Returns the recognized text with leading/trailing whitespace trimmed.
func (c *Client) RecognizeImage(imageData []byte) (string, error) {
	if err := c.client.SetImageFromBytes(imageData); err != nil {
		return "", fmt.Errorf("failed to set image: %w", err)
	}

	text, err := c.client.Text()
	if err != nil {
		return "", fmt.Errorf("OCR failed: %w", err)
	}

	return strings.TrimSpace(text), nil
}

// SetLanguage sets the language(s) for OCR recognition.
// Multiple languages can be specified as a "+" separated string (e.g., "eng+fra").
// Default is "eng" (English).
func (c *Client) SetLanguage(lang string) error {
	return c.client.SetLanguage(lang)
}

// SetPageSegMode sets the page segmentation mode.
// This affects how Tesseract analyzes the page layout.
// See PageSegMode constants for available modes.
func (c *Client) SetPageSegMode(mode PageSegMode) error {
	return c.client.SetPageSegMode(gosseract.PageSegMode(mode))
}
