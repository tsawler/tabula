package reader

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"

	"github.com/tsawler/tabula/core"
	"github.com/tsawler/tabula/imagecodec"
	"github.com/tsawler/tabula/pages"
)

// PageImage represents an extracted image from a PDF page.
type PageImage struct {
	Name             string // XObject name (e.g., "Im1")
	Width            int
	Height           int
	ColorSpace       string // DeviceGray, DeviceRGB, DeviceCMYK, Indexed, etc.
	BitsPerComponent int
	Data             []byte // Decoded pixel data
	Filter           string // Original filter (for format detection)

	// Decode is the /Decode array (sample-value remapping), empty when absent.
	// Honored for grayscale/bilevel/indexed images; [1 0] inverts black/white.
	Decode []float64

	// ImageMask reports a 1-bit stencil mask (/ImageMask true): samples select
	// paint vs. transparent rather than a color. Rendered as black-on-white.
	ImageMask bool

	// Indexed (palette) color: Palette holds PaletteComps bytes per entry in the
	// PaletteBase color space (DeviceRGB/DeviceGray/DeviceCMYK). Set only when
	// ColorSpace == "Indexed".
	Palette      []byte
	PaletteBase  string
	PaletteComps int

	// Globals is the JBIG2Globals segment data (from /DecodeParms), needed to
	// decode a JBIG2 image stream. Empty when absent or not a JBIG2 image.
	Globals []byte
}

// ExtractPageImages extracts all image XObjects from a page.
// It returns a slice of PageImage containing decoded image data.
func (r *Reader) ExtractPageImages(page *pages.Page) ([]PageImage, error) {
	resources, err := page.Resources()
	if err != nil {
		return nil, nil // Page has no resources
	}

	xobjectObj := resources.Get("XObject")
	if xobjectObj == nil {
		return nil, nil // No XObjects in resources
	}

	// Resolve XObject dict if it's a reference
	xobjectResolved, err := r.Resolve(xobjectObj)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve XObject dictionary: %w", err)
	}

	xobjects, ok := xobjectResolved.(core.Dict)
	if !ok {
		return nil, nil // XObject is not a dictionary
	}

	var images []PageImage

	for name, xobj := range xobjects {
		// Resolve XObject
		resolved, err := r.Resolve(xobj)
		if err != nil {
			continue // Skip XObjects that can't be resolved
		}

		stream, ok := resolved.(*core.Stream)
		if !ok {
			continue // Not a stream
		}

		// Check if it's an image
		subtype := stream.Dict.Get("Subtype")
		if subtype == nil {
			continue
		}

		subtypeName, ok := subtype.(core.Name)
		if !ok || string(subtypeName) != "Image" {
			continue // Not an image
		}

		// Extract image properties
		img, err := r.extractImage(name, stream)
		if err != nil {
			continue // Skip images that fail to extract
		}

		images = append(images, *img)
	}

	return images, nil
}

// resolveInt resolves obj (which may be an indirect reference) and returns its
// integer value. ok is false when obj is nil, can't be resolved, or isn't an Int.
func (r *Reader) resolveInt(obj core.Object) (int, bool) {
	if obj == nil {
		return 0, false
	}
	resolved, err := r.Resolve(obj)
	if err != nil {
		return 0, false
	}
	if n, ok := resolved.(core.Int); ok {
		return int(n), true
	}
	return 0, false
}

// extractImage extracts a single image from a stream.
func (r *Reader) extractImage(name string, stream *core.Stream) (*PageImage, error) {
	dict := stream.Dict

	// Get image dimensions. Width/Height (like Length) may be indirect
	// references — common in scanner output — so resolve before reading.
	width, ok := r.resolveInt(dict.Get("Width"))
	if !ok {
		return nil, fmt.Errorf("image missing or invalid Width")
	}
	height, ok := r.resolveInt(dict.Get("Height"))
	if !ok {
		return nil, fmt.Errorf("image missing or invalid Height")
	}

	// Get bits per component (defaults to 8); may also be indirect.
	bpc := 8
	if n, ok := r.resolveInt(dict.Get("BitsPerComponent")); ok {
		bpc = n
	}

	// Color space. Indexed (palette) images are captured specially so the
	// palette can be applied at render time; everything else collapses to a
	// base color-space name.
	colorSpace := "DeviceGray" // Default
	var palette []byte
	var paletteBase string
	var paletteComps int
	if csObj := dict.Get("ColorSpace"); csObj != nil {
		if base, pal, comps, ok := r.parseIndexedColorSpace(csObj); ok {
			colorSpace, paletteBase, palette, paletteComps = "Indexed", base, pal, comps
		} else {
			colorSpace = r.parseColorSpace(csObj)
		}
	}

	// Stencil masks (/ImageMask true) have no color space and are always 1 bit
	// per component; render them as black-on-white via the bilevel path.
	imageMask := false
	if b, ok := r.resolveBool(dict.Get("ImageMask")); ok && b {
		imageMask, bpc, colorSpace = true, 1, "DeviceGray"
	}

	// /Decode sample remapping (e.g. [1 0] inverts black/white).
	decode := r.parseNumberArray(dict.Get("Decode"))

	// Get filter for format detection (resolve in case it's an indirect ref).
	filter := ""
	if filterObj := dict.Get("Filter"); filterObj != nil {
		if resolved, err := r.Resolve(filterObj); err == nil {
			filterObj = resolved
		}
		if filterName, ok := filterObj.(core.Name); ok {
			filter = string(filterName)
		} else if filterArr, ok := filterObj.(core.Array); ok && len(filterArr) > 0 {
			if filterName, ok := filterArr[0].(core.Name); ok {
				filter = string(filterName)
			}
		}
	}

	// JBIG2 images carry shared segments in /DecodeParms /JBIG2Globals, needed
	// by the decoder.
	var globals []byte
	if filter == "JBIG2Decode" {
		if globals = r.jbig2Globals(dict.Get("DecodeParms")); globals == nil {
			globals = r.jbig2Globals(dict.Get("DP"))
		}
	}

	// Decode the stream data
	data, err := stream.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode image stream: %w", err)
	}

	return &PageImage{
		Globals:          globals,
		Name:             name,
		Width:            int(width),
		Height:           int(height),
		ColorSpace:       colorSpace,
		BitsPerComponent: bpc,
		Data:             data,
		Filter:           filter,
		Decode:           decode,
		ImageMask:        imageMask,
		Palette:          palette,
		PaletteBase:      paletteBase,
		PaletteComps:     paletteComps,
	}, nil
}

// resolveBool resolves obj (possibly indirect) to a boolean.
func (r *Reader) resolveBool(obj core.Object) (bool, bool) {
	if obj == nil {
		return false, false
	}
	resolved, err := r.Resolve(obj)
	if err != nil {
		return false, false
	}
	if b, ok := resolved.(core.Bool); ok {
		return bool(b), true
	}
	return false, false
}

// parseNumberArray resolves obj to a numeric array ([]float64), or nil.
func (r *Reader) parseNumberArray(obj core.Object) []float64 {
	if obj == nil {
		return nil
	}
	resolved, err := r.Resolve(obj)
	if err != nil {
		return nil
	}
	arr, ok := resolved.(core.Array)
	if !ok {
		return nil
	}
	out := make([]float64, 0, len(arr))
	for _, e := range arr {
		switch n := e.(type) {
		case core.Int:
			out = append(out, float64(n))
		case core.Real:
			out = append(out, float64(n))
		default:
			return nil // malformed — ignore the whole array
		}
	}
	return out
}

// componentsForColorSpace returns the samples-per-pixel for a base color space.
// ICCBased is assumed RGB (its real component count needs the profile).
func componentsForColorSpace(cs string) int {
	switch cs {
	case "DeviceGray", "CalGray":
		return 1
	case "DeviceRGB", "CalRGB", "Lab", "ICCBased":
		return 3
	case "DeviceCMYK":
		return 4
	}
	return 0
}

// parseIndexedColorSpace recognizes [/Indexed base hival lookup] and returns the
// base color-space name, the palette bytes, and the base's component count.
func (r *Reader) parseIndexedColorSpace(obj core.Object) (base string, palette []byte, comps int, ok bool) {
	resolved, err := r.Resolve(obj)
	if err != nil {
		return "", nil, 0, false
	}
	arr, isArr := resolved.(core.Array)
	if !isArr || len(arr) < 4 {
		return "", nil, 0, false
	}
	name, isName := arr[0].(core.Name)
	if !isName || (string(name) != "Indexed" && string(name) != "I") {
		return "", nil, 0, false
	}
	base = r.parseColorSpace(arr[1])
	comps = componentsForColorSpace(base)
	if comps == 0 {
		return "", nil, 0, false
	}
	// The lookup table is a string literal or a stream of comps*(hival+1) bytes.
	lookupObj, err := r.Resolve(arr[3])
	if err != nil {
		return "", nil, 0, false
	}
	switch v := lookupObj.(type) {
	case core.String:
		palette = []byte(v)
	case *core.Stream:
		palette, err = v.Decode()
		if err != nil {
			return "", nil, 0, false
		}
	default:
		return "", nil, 0, false
	}
	return base, palette, comps, true
}

// jbig2Globals extracts the /JBIG2Globals segment data from a /DecodeParms
// object (a dict, or an array of dicts when there is a filter chain).
func (r *Reader) jbig2Globals(dp core.Object) []byte {
	if dp == nil {
		return nil
	}
	resolved, err := r.Resolve(dp)
	if err != nil {
		return nil
	}
	switch v := resolved.(type) {
	case core.Dict:
		return r.streamBytes(v.Get("JBIG2Globals"))
	case core.Array:
		for _, e := range v {
			er, err := r.Resolve(e)
			if err != nil {
				continue
			}
			if d, ok := er.(core.Dict); ok {
				if b := r.streamBytes(d.Get("JBIG2Globals")); b != nil {
					return b
				}
			}
		}
	}
	return nil
}

// streamBytes resolves obj to a stream and returns its decoded data, or nil.
func (r *Reader) streamBytes(obj core.Object) []byte {
	if obj == nil {
		return nil
	}
	resolved, err := r.Resolve(obj)
	if err != nil {
		return nil
	}
	if s, ok := resolved.(*core.Stream); ok {
		if data, err := s.Decode(); err == nil {
			return data
		}
	}
	return nil
}

// parseColorSpace parses a color space object and returns its name.
func (r *Reader) parseColorSpace(obj core.Object) string {
	// Resolve if reference
	resolved, err := r.Resolve(obj)
	if err != nil {
		return "DeviceGray"
	}

	switch v := resolved.(type) {
	case core.Name:
		return string(v)
	case core.Array:
		// For array color spaces like [/ICCBased ref] or [/Indexed /DeviceGray 255 ref]
		if len(v) > 0 {
			if name, ok := v[0].(core.Name); ok {
				csName := string(name)
				// For Indexed, get the base color space
				if csName == "Indexed" && len(v) > 1 {
					return r.parseColorSpace(v[1])
				}
				// For ICCBased, try to determine the number of components
				if csName == "ICCBased" && len(v) > 1 {
					// ICCBased profiles can be Gray, RGB, or CMYK
					// We'd need to parse the ICC profile to know for sure
					// For now, return a generic name
					return "ICCBased"
				}
				return csName
			}
		}
	}

	return "DeviceGray"
}

// ToPNG converts the decoded pixel data to PNG format.
// This is suitable for use with OCR engines like Tesseract.
func (img *PageImage) ToPNG() ([]byte, error) {
	var goImg image.Image
	var err error

	switch {
	case img.Filter == "JBIG2Decode":
		// Decoded via jbig2dec (CGO; available only with -tags ocr).
		goImg, err = imagecodec.DecodeJBIG2(img.Data, img.Globals, img.Width, img.Height)
	case img.Filter == "JPXDecode":
		// Decoded via openjpeg (CGO; available only with -tags ocr).
		goImg, err = imagecodec.DecodeJPEG2000(img.Data)
	case len(img.Data) >= 2 && img.Data[0] == 0xFF && img.Data[1] == 0xD8:
		// DCTDecode returns raw JPEG.
		goImg, err = jpeg.Decode(bytes.NewReader(img.Data))
		if err != nil {
			err = fmt.Errorf("failed to decode JPEG: %w", err)
		}
	default:
		// Raw pixel data, interpreted by color space.
		switch img.ColorSpace {
		case "Indexed":
			goImg, err = img.toIndexedImage()
		case "DeviceGray", "CalGray", "ICCBased":
			goImg, err = img.toGrayImage()
		case "DeviceRGB", "CalRGB":
			goImg, err = img.toRGBImage()
		case "DeviceCMYK":
			goImg, err = img.toCMYKImage()
		default:
			goImg, err = img.toGrayImage()
		}
	}
	if err != nil {
		return nil, err
	}

	// Separation colorspace represents ink density (high = more ink = darker)
	// For OCR, we need to invert so text appears dark on light background
	if img.ColorSpace == "Separation" {
		goImg = invertImage(goImg)
	}

	// Encode as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, goImg); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// invertImage inverts the colors of an image (for Separation colorspace).
func invertImage(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			// Invert RGB values (RGBA returns 16-bit values, scale to 8-bit)
			dst.Set(x, y, color.RGBA{
				R: uint8(255 - r>>8),
				G: uint8(255 - g>>8),
				B: uint8(255 - b>>8),
				A: uint8(a >> 8),
			})
		}
	}

	return dst
}

// toGrayImage converts grayscale pixel data to an image.Gray.
func (img *PageImage) toGrayImage() (*image.Gray, error) {
	goImg := image.NewGray(image.Rect(0, 0, img.Width, img.Height))

	switch img.BitsPerComponent {
	case 1:
		// 1-bit bi-level image (from CCITT, etc.)
		return img.toBilevelGray()
	case 8:
		// 8-bit grayscale
		expectedSize := img.Width * img.Height
		if len(img.Data) < expectedSize {
			return nil, fmt.Errorf("insufficient data: got %d, expected %d", len(img.Data), expectedSize)
		}
		if len(img.Decode) >= 2 {
			for i := 0; i < expectedSize; i++ {
				goImg.Pix[i] = img.grayFromSample(int(img.Data[i]), 255)
			}
		} else {
			copy(goImg.Pix, img.Data[:expectedSize])
		}
		return goImg, nil
	case 4:
		// 4-bit grayscale
		return img.to4BitGray()
	default:
		return nil, fmt.Errorf("unsupported bits per component: %d", img.BitsPerComponent)
	}
}

// toBilevelGray converts 1-bit bi-level data to 8-bit grayscale.
func (img *PageImage) toBilevelGray() (*image.Gray, error) {
	goImg := image.NewGray(image.Rect(0, 0, img.Width, img.Height))

	// Calculate bytes per row (rounded up to nearest byte)
	bytesPerRow := (img.Width + 7) / 8
	expectedSize := bytesPerRow * img.Height

	if len(img.Data) < expectedSize {
		return nil, fmt.Errorf("insufficient data for 1-bit image: got %d, expected %d", len(img.Data), expectedSize)
	}

	for y := 0; y < img.Height; y++ {
		rowStart := y * bytesPerRow
		for x := 0; x < img.Width; x++ {
			byteIdx := rowStart + x/8
			bitIdx := 7 - (x % 8) // MSB first
			bit := (img.Data[byteIdx] >> bitIdx) & 1
			// 0 = black, 1 = white by default; /Decode [1 0] inverts (common on
			// CCITT scans and image masks).
			goImg.Pix[y*img.Width+x] = img.grayFromSample(int(bit), 1)
		}
	}

	return goImg, nil
}

// to4BitGray converts 4-bit grayscale data to 8-bit grayscale.
func (img *PageImage) to4BitGray() (*image.Gray, error) {
	goImg := image.NewGray(image.Rect(0, 0, img.Width, img.Height))

	// Calculate bytes per row (two pixels per byte)
	bytesPerRow := (img.Width + 1) / 2
	expectedSize := bytesPerRow * img.Height

	if len(img.Data) < expectedSize {
		return nil, fmt.Errorf("insufficient data for 4-bit image: got %d, expected %d", len(img.Data), expectedSize)
	}

	for y := 0; y < img.Height; y++ {
		rowStart := y * bytesPerRow
		for x := 0; x < img.Width; x++ {
			byteIdx := rowStart + x/2
			var nibble byte
			if x%2 == 0 {
				nibble = (img.Data[byteIdx] >> 4) & 0x0F // High nibble first
			} else {
				nibble = img.Data[byteIdx] & 0x0F // Low nibble
			}
			// Scale 4-bit (0-15) to 8-bit, honoring /Decode when present.
			goImg.Pix[y*img.Width+x] = img.grayFromSample(int(nibble), 15)
		}
	}

	return goImg, nil
}

// grayFromSample maps a raw sample in [0, maxSample] to an 8-bit gray value,
// applying the /Decode endpoints when present (default [0 1]; [1 0] inverts).
func (img *PageImage) grayFromSample(sample, maxSample int) uint8 {
	d0, d1 := 0.0, 1.0
	if len(img.Decode) >= 2 {
		d0, d1 = img.Decode[0], img.Decode[1]
	}
	v := d0
	if maxSample > 0 {
		v = d0 + float64(sample)*(d1-d0)/float64(maxSample)
	}
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	return uint8(v*255 + 0.5)
}

// sampleAt extracts the bpc-bit sample for pixel x from a packed row (MSB first).
func sampleAt(row []byte, x, bpc int) int {
	switch bpc {
	case 8:
		if x < len(row) {
			return int(row[x])
		}
	case 4:
		if bi := x / 2; bi < len(row) {
			if x%2 == 0 {
				return int(row[bi] >> 4)
			}
			return int(row[bi] & 0x0F)
		}
	case 2:
		if bi := x / 4; bi < len(row) {
			return int((row[bi] >> uint(6-2*(x%4))) & 0x03)
		}
	case 1:
		if bi := x / 8; bi < len(row) {
			return int((row[bi] >> uint(7-(x%8))) & 0x01)
		}
	}
	return 0
}

// paletteRGB resolves a palette index to an RGB triple using the base color space.
func (img *PageImage) paletteRGB(idx int) (uint8, uint8, uint8) {
	off := idx * img.PaletteComps
	if off+img.PaletteComps > len(img.Palette) {
		return 0, 0, 0
	}
	p := img.Palette[off:]
	switch img.PaletteBase {
	case "DeviceRGB", "CalRGB", "Lab", "ICCBased":
		return p[0], p[1], p[2]
	case "DeviceCMYK":
		return color.CMYKToRGB(p[0], p[1], p[2], p[3])
	default: // DeviceGray, CalGray
		return p[0], p[0], p[0]
	}
}

// toIndexedImage maps palette indices to RGB using the captured palette.
func (img *PageImage) toIndexedImage() (*image.RGBA, error) {
	if img.PaletteComps == 0 || len(img.Palette) < img.PaletteComps {
		return nil, fmt.Errorf("indexed image has no usable palette")
	}
	goImg := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))
	maxIndex := len(img.Palette)/img.PaletteComps - 1

	bytesPerRow := (img.Width*img.BitsPerComponent + 7) / 8
	if len(img.Data) < bytesPerRow*img.Height {
		return nil, fmt.Errorf("insufficient data for indexed image: got %d, expected %d", len(img.Data), bytesPerRow*img.Height)
	}
	for y := 0; y < img.Height; y++ {
		row := img.Data[y*bytesPerRow : (y+1)*bytesPerRow]
		for x := 0; x < img.Width; x++ {
			idx := sampleAt(row, x, img.BitsPerComponent)
			if idx > maxIndex {
				idx = maxIndex
			}
			r, g, b := img.paletteRGB(idx)
			d := (y*img.Width + x) * 4
			goImg.Pix[d+0], goImg.Pix[d+1], goImg.Pix[d+2], goImg.Pix[d+3] = r, g, b, 255
		}
	}
	return goImg, nil
}

// toRGBImage converts RGB pixel data to an image.RGBA.
func (img *PageImage) toRGBImage() (*image.RGBA, error) {
	if img.BitsPerComponent != 8 {
		return nil, fmt.Errorf("unsupported bits per component for RGB: %d", img.BitsPerComponent)
	}

	goImg := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))

	expectedSize := img.Width * img.Height * 3
	if len(img.Data) < expectedSize {
		return nil, fmt.Errorf("insufficient data for RGB image: got %d, expected %d", len(img.Data), expectedSize)
	}

	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			srcIdx := (y*img.Width + x) * 3
			dstIdx := (y*img.Width + x) * 4
			goImg.Pix[dstIdx+0] = img.Data[srcIdx+0] // R
			goImg.Pix[dstIdx+1] = img.Data[srcIdx+1] // G
			goImg.Pix[dstIdx+2] = img.Data[srcIdx+2] // B
			goImg.Pix[dstIdx+3] = 255                // A
		}
	}

	return goImg, nil
}

// toCMYKImage converts CMYK pixel data to an image.RGBA.
func (img *PageImage) toCMYKImage() (*image.RGBA, error) {
	if img.BitsPerComponent != 8 {
		return nil, fmt.Errorf("unsupported bits per component for CMYK: %d", img.BitsPerComponent)
	}

	goImg := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))

	expectedSize := img.Width * img.Height * 4
	if len(img.Data) < expectedSize {
		return nil, fmt.Errorf("insufficient data for CMYK image: got %d, expected %d", len(img.Data), expectedSize)
	}

	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			srcIdx := (y*img.Width + x) * 4
			c := img.Data[srcIdx+0]
			m := img.Data[srcIdx+1]
			yy := img.Data[srcIdx+2]
			k := img.Data[srcIdx+3]

			// Convert CMYK to RGB
			r, g, b := color.CMYKToRGB(c, m, yy, k)

			dstIdx := (y*img.Width + x) * 4
			goImg.Pix[dstIdx+0] = r
			goImg.Pix[dstIdx+1] = g
			goImg.Pix[dstIdx+2] = b
			goImg.Pix[dstIdx+3] = 255
		}
	}

	return goImg, nil
}
