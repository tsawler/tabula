//go:build !ocr

package ocr

import (
	"errors"
	"testing"
)

func TestNewReturnsError(t *testing.T) {
	client, err := New()
	if err == nil {
		t.Error("Expected error from New() when OCR is disabled")
	}
	if !errors.Is(err, ErrOCRNotEnabled) {
		t.Errorf("Expected ErrOCRNotEnabled, got: %v", err)
	}
	if client != nil {
		t.Error("Expected nil client when OCR is disabled")
	}
}

func TestCloseOnNilClient(t *testing.T) {
	var client *Client
	err := client.Close()
	if err != nil {
		t.Errorf("Close on nil client should not error: %v", err)
	}
}
