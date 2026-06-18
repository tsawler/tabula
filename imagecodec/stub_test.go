//go:build !ocr

package imagecodec

import (
	"errors"
	"testing"
)

// Without the ocr build tag the decoders report that they're unavailable.
func TestDecodersUnavailableWithoutOCR(t *testing.T) {
	if _, err := DecodeJBIG2([]byte{1}, nil, 1, 1); !errors.Is(err, ErrCodecUnavailable) {
		t.Errorf("DecodeJBIG2 err = %v, want ErrCodecUnavailable", err)
	}
	if _, err := DecodeJPEG2000([]byte{1}); !errors.Is(err, ErrCodecUnavailable) {
		t.Errorf("DecodeJPEG2000 err = %v, want ErrCodecUnavailable", err)
	}
}
