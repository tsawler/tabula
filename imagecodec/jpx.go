//go:build ocr

package imagecodec

/*
#cgo pkg-config: libopenjp2
#include <stdlib.h>
#include <string.h>
#include <openjpeg.h>

// In-memory openjpeg stream: openjpeg has no built-in memory stream, so we wire
// read/skip/seek callbacks over a byte buffer.
typedef struct {
	OPJ_BYTE   *buf;
	OPJ_SIZE_T  size;
	OPJ_SIZE_T  pos;
} memstream;

static OPJ_SIZE_T mem_read(void *out, OPJ_SIZE_T n, void *user) {
	memstream *m = (memstream *)user;
	if (m->pos >= m->size) return (OPJ_SIZE_T)-1; // EOF
	OPJ_SIZE_T avail = m->size - m->pos;
	if (n > avail) n = avail;
	memcpy(out, m->buf + m->pos, n);
	m->pos += n;
	return n;
}

static OPJ_OFF_T mem_skip(OPJ_OFF_T n, void *user) {
	memstream *m = (memstream *)user;
	OPJ_OFF_T newpos = (OPJ_OFF_T)m->pos + n;
	if (newpos < 0) newpos = 0;
	if ((OPJ_SIZE_T)newpos > m->size) newpos = (OPJ_OFF_T)m->size;
	m->pos = (OPJ_SIZE_T)newpos;
	return n;
}

static OPJ_BOOL mem_seek(OPJ_OFF_T n, void *user) {
	memstream *m = (memstream *)user;
	if (n < 0 || (OPJ_SIZE_T)n > m->size) return OPJ_FALSE;
	m->pos = (OPJ_SIZE_T)n;
	return OPJ_TRUE;
}

static void mem_free(void *user) { free(user); }

static void quiet(const char *msg, void *client) { (void)msg; (void)client; }

// jpx_decode sets up the memory stream and codec and returns the decoded image
// (caller frees with opj_image_destroy), or NULL on failure.
static opj_image_t *jpx_decode(OPJ_BYTE *buf, OPJ_SIZE_T size, OPJ_CODEC_FORMAT fmt) {
	memstream *m = (memstream *)malloc(sizeof(memstream));
	if (!m) return NULL;
	m->buf = buf; m->size = size; m->pos = 0;

	opj_stream_t *stream = opj_stream_default_create(OPJ_TRUE);
	if (!stream) { free(m); return NULL; }
	opj_stream_set_user_data(stream, m, mem_free);
	opj_stream_set_user_data_length(stream, size);
	opj_stream_set_read_function(stream, mem_read);
	opj_stream_set_skip_function(stream, mem_skip);
	opj_stream_set_seek_function(stream, mem_seek);

	opj_codec_t *codec = opj_create_decompress(fmt);
	if (!codec) { opj_stream_destroy(stream); return NULL; }
	opj_set_error_handler(codec, quiet, NULL);
	opj_set_warning_handler(codec, quiet, NULL);
	opj_set_info_handler(codec, quiet, NULL);

	opj_dparameters_t params;
	opj_set_default_decoder_parameters(&params);
	if (!opj_setup_decoder(codec, &params)) {
		opj_destroy_codec(codec); opj_stream_destroy(stream); return NULL;
	}

	opj_image_t *image = NULL;
	if (!opj_read_header(stream, codec, &image)) {
		opj_destroy_codec(codec); opj_stream_destroy(stream); return NULL;
	}
	if (!opj_decode(codec, stream, image) || !opj_end_decompress(codec, stream)) {
		opj_image_destroy(image); opj_destroy_codec(codec); opj_stream_destroy(stream); return NULL;
	}
	opj_destroy_codec(codec);
	opj_stream_destroy(stream);
	return image;
}
*/
import "C"

import (
	"fmt"
	"image"
	"unsafe"
)

// DecodeJPEG2000 decodes a JPEG2000 image (raw codestream or JP2) into an RGBA
// image. Grayscale and 3+ component (RGB) images are supported; component
// precisions other than 8 bits are scaled to 8.
func DecodeJPEG2000(data []byte) (image.Image, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("jpx: empty image data")
	}

	// Raw codestreams start with the SOC marker 0xFF4F; otherwise assume JP2.
	format := C.OPJ_CODEC_JP2
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0x4F {
		format = C.OPJ_CODEC_J2K
	}

	cimg := C.jpx_decode(
		(*C.OPJ_BYTE)(unsafe.Pointer(&data[0])),
		C.OPJ_SIZE_T(len(data)),
		C.OPJ_CODEC_FORMAT(format),
	)
	if cimg == nil {
		return nil, fmt.Errorf("jpx: decode failed")
	}
	defer C.opj_image_destroy(cimg)

	numcomps := int(cimg.numcomps)
	if numcomps == 0 {
		return nil, fmt.Errorf("jpx: image has no components")
	}
	comps := unsafe.Slice(cimg.comps, numcomps)
	w, h := int(comps[0].w), int(comps[0].h)
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("jpx: invalid dimensions %dx%d", w, h)
	}

	// Use RGB only when the first three components share comp[0]'s dimensions
	// (no chroma subsampling); otherwise render grayscale from the first.
	rgb := numcomps >= 3 &&
		int(comps[1].w) == w && int(comps[1].h) == h &&
		int(comps[2].w) == w && int(comps[2].h) == h

	planes := make([][]C.OPJ_INT32, numcomps)
	for i := 0; i < numcomps; i++ {
		planes[i] = unsafe.Slice(comps[i].data, int(comps[i].w)*int(comps[i].h))
	}
	sample := func(ci, idx int) uint8 {
		c := comps[ci]
		v := int(planes[ci][idx])
		if c.sgnd != 0 {
			v += 1 << (uint(c.prec) - 1)
		}
		if prec := int(c.prec); prec > 8 {
			v >>= uint(prec - 8)
		} else if prec < 8 {
			v <<= uint(8 - prec)
		}
		if v < 0 {
			v = 0
		} else if v > 255 {
			v = 255
		}
		return uint8(v)
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		var r, g, b uint8
		if rgb {
			r, g, b = sample(0, i), sample(1, i), sample(2, i)
		} else {
			v := sample(0, i)
			r, g, b = v, v, v
		}
		o := i * 4
		img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = r, g, b, 255
	}
	return img, nil
}
