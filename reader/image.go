package reader

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/tsawler/tabula/core"
	"github.com/tsawler/tabula/pages"
)

// PageImage represents an extracted image from a PDF page.
type PageImage struct {
	Name             string // XObject name (e.g., "Im1")
	Width            int
	Height           int
	ColorSpace       string // DeviceGray, DeviceRGB, DeviceCMYK, etc.
	BitsPerComponent int
	Data             []byte // Decoded pixel data
	Filter           string // Original filter (for format detection)
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

// extractImage extracts a single image from a stream.
func (r *Reader) extractImage(name string, stream *core.Stream) (*PageImage, error) {
	dict := stream.Dict

	// Get image dimensions
	widthObj := dict.Get("Width")
	heightObj := dict.Get("Height")
	if widthObj == nil || heightObj == nil {
		return nil, fmt.Errorf("image missing Width or Height")
	}

	width, ok := widthObj.(core.Int)
	if !ok {
		return nil, fmt.Errorf("invalid Width type: %T", widthObj)
	}

	height, ok := heightObj.(core.Int)
	if !ok {
		return nil, fmt.Errorf("invalid Height type: %T", heightObj)
	}

	// Get bits per component (defaults to 1 for bi-level images like CCITT)
	bpc := 8
	if bpcObj := dict.Get("BitsPerComponent"); bpcObj != nil {
		if bpcInt, ok := bpcObj.(core.Int); ok {
			bpc = int(bpcInt)
		}
	}

	// Get color space
	colorSpace := "DeviceGray" // Default
	if csObj := dict.Get("ColorSpace"); csObj != nil {
		colorSpace = r.parseColorSpace(csObj)
	}

	// Get filter for format detection
	filter := ""
	if filterObj := dict.Get("Filter"); filterObj != nil {
		if filterName, ok := filterObj.(core.Name); ok {
			filter = string(filterName)
		} else if filterArr, ok := filterObj.(core.Array); ok && len(filterArr) > 0 {
			if filterName, ok := filterArr[0].(core.Name); ok {
				filter = string(filterName)
			}
		}
	}

	// Decode the stream data
	data, err := stream.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode image stream: %w", err)
	}

	return &PageImage{
		Name:             name,
		Width:            int(width),
		Height:           int(height),
		ColorSpace:       colorSpace,
		BitsPerComponent: bpc,
		Data:             data,
		Filter:           filter,
	}, nil
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

	// Handle different color spaces and bit depths
	switch img.ColorSpace {
	case "DeviceGray", "CalGray", "ICCBased":
		goImg, err = img.toGrayImage()
	case "DeviceRGB", "CalRGB":
		goImg, err = img.toRGBImage()
	case "DeviceCMYK":
		goImg, err = img.toCMYKImage()
	default:
		// Try grayscale as fallback
		goImg, err = img.toGrayImage()
	}

	if err != nil {
		return nil, err
	}

	// Encode as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, goImg); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
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
		copy(goImg.Pix, img.Data[:expectedSize])
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
			// In PDF, 0 typically means black and 1 means white
			// (unless BlackIs1 was set, which should be handled during decode)
			if bit == 0 {
				goImg.Pix[y*img.Width+x] = 0 // Black
			} else {
				goImg.Pix[y*img.Width+x] = 255 // White
			}
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
			// Scale 4-bit (0-15) to 8-bit (0-255)
			goImg.Pix[y*img.Width+x] = nibble * 17 // 17 = 255/15
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
