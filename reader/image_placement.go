package reader

import (
	"math"

	"github.com/tsawler/tabula/contentstream"
	"github.com/tsawler/tabula/core"
	"github.com/tsawler/tabula/graphicsstate"
	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/pages"
)

// PlacedPageImage is one raster image XObject as actually painted on a page,
// with its placement derived from the current transformation matrix (CTM) at
// the Do operator. Coordinates are in PDF user space (points, origin
// bottom-left) in the page's content-stream coordinate system.
type PlacedPageImage struct {
	Name        string  // XObject name (e.g. "Im1")
	PixelWidth  int     // intrinsic image width in pixels
	PixelHeight int     // intrinsic image height in pixels
	ColorSpace  string  // base color-space name (DeviceRGB, DeviceGray, ...)
	X, Y        float64 // bottom-left of the drawn bounding box, in points
	Width       float64 // drawn width in points
	Height      float64 // drawn height in points
}

// ExtractPlacedImages walks a page's content stream, tracking graphics state,
// and reports every image XObject painted via Do together with the axis-aligned
// bounding box it occupies on the page. Images nested inside Form XObjects are
// reported with the composed CTM (including the form's /Matrix). Images are
// returned in draw order. Inline images (BI/ID/EI) are not reported.
func (r *Reader) ExtractPlacedImages(page *pages.Page) ([]PlacedPageImage, error) {
	// Decode and concatenate the page's content streams (same approach as
	// extractTextWithFragments).
	contents, err := page.Contents()
	if err != nil || contents == nil {
		return nil, nil // No content / no resources => no images
	}
	var data []byte
	for _, contentObj := range contents {
		stream, ok := contentObj.(*core.Stream)
		if !ok {
			continue
		}
		decoded, err := stream.Decode()
		if err != nil {
			continue // Skip undecodable streams rather than failing the page
		}
		data = append(data, decoded...)
	}
	if len(data) == 0 {
		return nil, nil
	}

	resources, err := page.Resources()
	if err != nil {
		resources = nil
	}

	gs := graphicsstate.NewGraphicsState()
	var out []PlacedPageImage
	r.walkImageContent(data, resources, gs, 0, &out)
	return out, nil
}

// walkImageContent processes a content stream's operations, tracking the CTM via
// the graphics state and emitting a PlacedPageImage for each image XObject drawn
// by Do. Form XObjects are recursed into with their /Matrix composed onto the CTM.
func (r *Reader) walkImageContent(data []byte, resources core.Dict, gs *graphicsstate.GraphicsState, depth int, out *[]PlacedPageImage) {
	if depth > maxFormDepth {
		return
	}
	ops, err := contentstream.NewParser(data).Parse()
	if err != nil {
		return
	}
	for _, op := range ops {
		switch op.Operator {
		case "q":
			gs.Save()
		case "Q":
			_ = gs.Restore()
		case "cm":
			if len(op.Operands) == 6 {
				gs.Transform(operandsToMatrixCS(op.Operands))
			}
		case "Do":
			if len(op.Operands) == 1 {
				if name, ok := op.Operands[0].(core.Name); ok {
					r.doImageXObject(string(name), resources, gs, depth, out)
				}
			}
		}
	}
}

// doImageXObject resolves the named XObject in resources. If it is an image, it
// records its placement using the current CTM. If it is a form, it recurses into
// the form's content stream with the form's /Matrix composed onto the CTM.
func (r *Reader) doImageXObject(name string, resources core.Dict, gs *graphicsstate.GraphicsState, depth int, out *[]PlacedPageImage) {
	if resources == nil {
		return
	}
	xobjects, ok := r.resolveDict(resources.Get("XObject"))
	if !ok {
		return
	}
	ref := xobjects.Get(name)
	if ref == nil {
		return
	}
	resolved, err := r.Resolve(ref)
	if err != nil {
		return
	}
	stream, ok := resolved.(*core.Stream)
	if !ok {
		return
	}

	switch subtype, _ := stream.Dict.Get("Subtype").(core.Name); string(subtype) {
	case "Image":
		x, y, w, h := imageBBoxFromCTM(gs.CTM)
		pw, _ := r.resolveInt(stream.Dict.Get("Width"))
		ph, _ := r.resolveInt(stream.Dict.Get("Height"))
		cs := "DeviceGray"
		if csObj := stream.Dict.Get("ColorSpace"); csObj != nil {
			cs = r.parseColorSpace(csObj)
		}
		if b, ok := r.resolveBool(stream.Dict.Get("ImageMask")); ok && b {
			cs = "DeviceGray"
		}
		*out = append(*out, PlacedPageImage{
			Name:        name,
			PixelWidth:  pw,
			PixelHeight: ph,
			ColorSpace:  cs,
			X:           x,
			Y:           y,
			Width:       w,
			Height:      h,
		})

	case "Form":
		data, err := stream.Decode()
		if err != nil || len(data) == 0 {
			return
		}
		// A form's own resources take precedence, falling back to the parent's.
		formRes := resources
		if fr, ok := r.resolveDict(stream.Dict.Get("Resources")); ok {
			formRes = mergeResourceDicts(resources, fr)
		}
		gs.Save()
		// Compose the form's /Matrix onto the CTM before processing its content.
		if matrixArr, ok := r.resolveArray(stream.Dict.Get("Matrix")); ok && len(matrixArr) == 6 {
			gs.Transform(operandsToMatrixCS(matrixArr))
		}
		r.walkImageContent(data, formRes, gs, depth+1, out)
		_ = gs.Restore()
	}
}

// imageBBoxFromCTM maps the image's unit square [0,1]x[0,1] through the CTM and
// returns the axis-aligned bounding box (x, y, width, height) in user space.
func imageBBoxFromCTM(ctm model.Matrix) (x, y, w, h float64) {
	corners := [4]model.Point{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, c := range corners {
		p := ctm.Transform(c)
		minX = math.Min(minX, p.X)
		minY = math.Min(minY, p.Y)
		maxX = math.Max(maxX, p.X)
		maxY = math.Max(maxY, p.Y)
	}
	return minX, minY, maxX - minX, maxY - minY
}

// resolveArray resolves obj (possibly indirect) to a core.Array.
func (r *Reader) resolveArray(obj core.Object) (core.Array, bool) {
	if obj == nil {
		return nil, false
	}
	resolved, err := r.Resolve(obj)
	if err != nil {
		return nil, false
	}
	arr, ok := resolved.(core.Array)
	return arr, ok
}

// operandsToMatrixCS converts six numeric operands to a model.Matrix.
func operandsToMatrixCS(operands []core.Object) model.Matrix {
	if len(operands) != 6 {
		return model.Identity()
	}
	var m model.Matrix
	for i := 0; i < 6; i++ {
		switch v := operands[i].(type) {
		case core.Int:
			m[i] = float64(v)
		case core.Real:
			m[i] = float64(v)
		}
	}
	return m
}

// mergeResourceDicts returns a shallow merge of parent and child resource
// dictionaries where child entries take precedence. The XObject sub-dictionary
// is merged so a form can reference both its own and inherited XObjects.
func mergeResourceDicts(parent, child core.Dict) core.Dict {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}
	merged := make(core.Dict, len(parent)+len(child))
	for k, v := range parent {
		merged[k] = v
	}
	for k, v := range child {
		if pSub, ok := parent[k].(core.Dict); ok {
			if cSub, ok := v.(core.Dict); ok {
				sub := make(core.Dict, len(pSub)+len(cSub))
				for sk, sv := range pSub {
					sub[sk] = sv
				}
				for sk, sv := range cSub {
					sub[sk] = sv
				}
				merged[k] = sub
				continue
			}
		}
		merged[k] = v
	}
	return merged
}
