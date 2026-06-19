//go:build ocr

package imagecodec

/*
#cgo pkg-config: jbig2dec
// jbig2dec's <jbig2.h> uses uint32_t/uint8_t but doesn't include <stdint.h>
// itself; include it first so the build works on toolchains (e.g. musl/Alpine)
// whose <stdlib.h> doesn't pull in <stdint.h> transitively.
#include <stdint.h>
#include <stdlib.h>
#include <jbig2.h>

// jbig2_ctx_new is a function-like macro, which cgo can't call through the C.
// pseudo-package, so wrap it. Embedded mode, default allocator/error handler.
static Jbig2Ctx *jb_ctx_new(Jbig2GlobalCtx *global) {
	return jbig2_ctx_new(NULL, JBIG2_OPTIONS_EMBEDDED, global, NULL, NULL);
}
*/
import "C"

import (
	"fmt"
	"image"
	"unsafe"
)

// DecodeJBIG2 decodes an embedded-PDF JBIG2 image stream (with optional shared
// JBIG2Globals) into a grayscale image with the foreground rendered black on a
// white background. width/height are the PDF-declared dimensions, used only as a
// fallback; the decoder's own page dimensions are authoritative.
func DecodeJBIG2(data, globals []byte, width, height int) (image.Image, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("jbig2: empty image data")
	}

	// A shared global context is built from the JBIG2Globals segment, if any.
	var gctx *C.Jbig2GlobalCtx
	if len(globals) > 0 {
		gc := C.jb_ctx_new(nil)
		if gc == nil {
			return nil, fmt.Errorf("jbig2: failed to create global context")
		}
		C.jbig2_data_in(gc, (*C.uchar)(unsafe.Pointer(&globals[0])), C.size_t(len(globals)))
		gctx = C.jbig2_make_global_ctx(gc) // consumes gc
	}

	ctx := C.jb_ctx_new(gctx)
	if ctx == nil {
		if gctx != nil {
			C.jbig2_global_ctx_free(gctx)
		}
		return nil, fmt.Errorf("jbig2: failed to create context")
	}
	defer func() {
		C.jbig2_ctx_free(ctx)
		if gctx != nil {
			C.jbig2_global_ctx_free(gctx)
		}
	}()

	C.jbig2_data_in(ctx, (*C.uchar)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
	// Embedded streams often lack an end-of-page segment; force completion.
	C.jbig2_complete_page(ctx)

	page := C.jbig2_page_out(ctx)
	if page == nil {
		return nil, fmt.Errorf("jbig2: no page produced")
	}
	defer C.jbig2_release_page(ctx, page)

	w, h, stride := int(page.width), int(page.height), int(page.stride)
	if w <= 0 || h <= 0 || stride <= 0 {
		return nil, fmt.Errorf("jbig2: invalid page dimensions %dx%d", w, h)
	}
	src := C.GoBytes(unsafe.Pointer(page.data), C.int(stride*h))

	// 1 bpp, packed MSB-first, 1 = black foreground.
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		rowStart := y * stride
		for x := 0; x < w; x++ {
			bit := (src[rowStart+x/8] >> uint(7-(x%8))) & 1
			if bit == 1 {
				img.Pix[y*w+x] = 0 // black
			} else {
				img.Pix[y*w+x] = 255 // white
			}
		}
	}
	return img, nil
}
