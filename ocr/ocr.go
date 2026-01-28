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
package ocr

import (
	"fmt"
	"strings"

	"github.com/otiai10/gosseract/v2"
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
// See gosseract.PageSegMode constants for available modes.
func (c *Client) SetPageSegMode(mode gosseract.PageSegMode) error {
	return c.client.SetPageSegMode(mode)
}
