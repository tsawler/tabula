// Package imagecodec decodes image formats that have no pure-Go decoder —
// JBIG2 and JPEG2000 — which appear in scanned PDFs. The real decoders bind to
// the system jbig2dec and openjpeg libraries via CGO and are only compiled with
// the "ocr" build tag (the same tag that enables Tesseract), since these images
// exist only to feed OCR. Without the tag, the decoders return
// ErrCodecUnavailable and callers skip the image.
package imagecodec

import "errors"

// ErrCodecUnavailable is returned by the decoders when the binary was built
// without the "ocr" tag (and therefore without jbig2dec / openjpeg).
var ErrCodecUnavailable = errors.New("image codec unavailable: rebuild with -tags ocr")
