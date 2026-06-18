//go:build !ocr

package imagecodec

import "image"

// DecodeJBIG2 is unavailable without the "ocr" build tag.
func DecodeJBIG2(data, globals []byte, width, height int) (image.Image, error) {
	return nil, ErrCodecUnavailable
}

// DecodeJPEG2000 is unavailable without the "ocr" build tag.
func DecodeJPEG2000(data []byte) (image.Image, error) {
	return nil, ErrCodecUnavailable
}
