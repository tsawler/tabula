package graphicsstate

import (
	"github.com/tsawler/tabula/contentstream"
	"github.com/tsawler/tabula/core"
	"github.com/tsawler/tabula/model"
)

// GraphicsExtractor extracts lines and rectangles from content streams
type GraphicsExtractor struct {
	gs            *GraphicsState
	pathExtractor *PathExtractor

	// Minimum dimensions for filtering
	MinLineLength float64
	MinRectWidth  float64
	MinRectHeight float64
}

// NewGraphicsExtractor creates a new graphics extractor
func NewGraphicsExtractor() *GraphicsExtractor {
	gs := NewGraphicsState()
	return &GraphicsExtractor{
		gs:            gs,
		pathExtractor: NewPathExtractor(gs),
		MinLineLength: 1.0, // Minimum 1 point line
		MinRectWidth:  1.0, // Minimum 1 point wide rectangle
		MinRectHeight: 1.0, // Minimum 1 point tall rectangle
	}
}

// Extract extracts graphics from content stream operations
func (ge *GraphicsExtractor) Extract(operations []contentstream.Operation) error {
	for _, op := range operations {
		if err := ge.processOperation(op); err != nil {
			return err
		}
	}
	return nil
}

// ExtractFromBytes parses and extracts graphics from raw content stream data
func (ge *GraphicsExtractor) ExtractFromBytes(data []byte) error {
	parser := contentstream.NewParser(data)
	operations, err := parser.Parse()
	if err != nil {
		return err
	}
	return ge.Extract(operations)
}

// processOperation processes a single content stream operation
func (ge *GraphicsExtractor) processOperation(op contentstream.Operation) error {
	switch op.Operator {
	// Graphics state operators
	case "q":
		ge.gs.Save()
	case "Q":
		return ge.gs.Restore()
	case "cm":
		if len(op.Operands) == 6 {
			m := operandsToMatrix(op.Operands)
			ge.gs.Transform(m)
		}
	case "w":
		if len(op.Operands) == 1 {
			if w, ok := toFloat(op.Operands[0]); ok {
				ge.gs.SetLineWidth(w)
			}
		}

	// Color operators
	case "RG":
		if len(op.Operands) == 3 {
			r, _ := toFloat(op.Operands[0])
			g, _ := toFloat(op.Operands[1])
			b, _ := toFloat(op.Operands[2])
			ge.gs.SetStrokeColorRGB(r, g, b)
		}
	case "rg":
		if len(op.Operands) == 3 {
			r, _ := toFloat(op.Operands[0])
			g, _ := toFloat(op.Operands[1])
			b, _ := toFloat(op.Operands[2])
			ge.gs.SetFillColorRGB(r, g, b)
		}
	case "G":
		// Gray stroke color
		if len(op.Operands) == 1 {
			gray, _ := toFloat(op.Operands[0])
			ge.gs.SetStrokeColorRGB(gray, gray, gray)
		}
	case "g":
		// Gray fill color
		if len(op.Operands) == 1 {
			gray, _ := toFloat(op.Operands[0])
			ge.gs.SetFillColorRGB(gray, gray, gray)
		}
	case "K":
		// CMYK stroke color - convert to RGB approximation
		if len(op.Operands) == 4 {
			c, _ := toFloat(op.Operands[0])
			m, _ := toFloat(op.Operands[1])
			y, _ := toFloat(op.Operands[2])
			k, _ := toFloat(op.Operands[3])
			r, g, b := cmykToRGB(c, m, y, k)
			ge.gs.SetStrokeColorRGB(r, g, b)
		}
	case "k":
		// CMYK fill color
		if len(op.Operands) == 4 {
			c, _ := toFloat(op.Operands[0])
			m, _ := toFloat(op.Operands[1])
			y, _ := toFloat(op.Operands[2])
			k, _ := toFloat(op.Operands[3])
			r, g, b := cmykToRGB(c, m, y, k)
			ge.gs.SetFillColorRGB(r, g, b)
		}

	// Path construction operators
	case "m":
		if len(op.Operands) == 2 {
			x, _ := toFloat(op.Operands[0])
			y, _ := toFloat(op.Operands[1])
			ge.pathExtractor.MoveTo(x, y)
		}
	case "l":
		if len(op.Operands) == 2 {
			x, _ := toFloat(op.Operands[0])
			y, _ := toFloat(op.Operands[1])
			ge.pathExtractor.LineTo(x, y)
		}
	case "c":
		if len(op.Operands) == 6 {
			x1, _ := toFloat(op.Operands[0])
			y1, _ := toFloat(op.Operands[1])
			x2, _ := toFloat(op.Operands[2])
			y2, _ := toFloat(op.Operands[3])
			x3, _ := toFloat(op.Operands[4])
			y3, _ := toFloat(op.Operands[5])
			ge.pathExtractor.CurveTo(x1, y1, x2, y2, x3, y3)
		}
	case "v":
		if len(op.Operands) == 4 {
			x2, _ := toFloat(op.Operands[0])
			y2, _ := toFloat(op.Operands[1])
			x3, _ := toFloat(op.Operands[2])
			y3, _ := toFloat(op.Operands[3])
			ge.pathExtractor.CurveToV(x2, y2, x3, y3)
		}
	case "y":
		if len(op.Operands) == 4 {
			x1, _ := toFloat(op.Operands[0])
			y1, _ := toFloat(op.Operands[1])
			x3, _ := toFloat(op.Operands[2])
			y3, _ := toFloat(op.Operands[3])
			ge.pathExtractor.CurveToY(x1, y1, x3, y3)
		}
	case "h":
		ge.pathExtractor.ClosePath()
	case "re":
		if len(op.Operands) == 4 {
			x, _ := toFloat(op.Operands[0])
			y, _ := toFloat(op.Operands[1])
			w, _ := toFloat(op.Operands[2])
			h, _ := toFloat(op.Operands[3])
			ge.pathExtractor.Rectangle(x, y, w, h)
		}

	// Path painting operators
	case "S":
		ge.pathExtractor.Stroke()
	case "s":
		ge.pathExtractor.CloseAndStroke()
	case "f", "F":
		ge.pathExtractor.Fill()
	case "f*":
		ge.pathExtractor.FillEvenOdd()
	case "B":
		ge.pathExtractor.FillAndStroke()
	case "B*":
		ge.pathExtractor.FillAndStrokeEvenOdd()
	case "b":
		ge.pathExtractor.CloseFillAndStroke()
	case "b*":
		ge.pathExtractor.CloseFillAndStrokeEvenOdd()
	case "n":
		ge.pathExtractor.EndPath()
	}

	return nil
}

// GetLines returns all extracted lines
func (ge *GraphicsExtractor) GetLines() []ExtractedLine {
	return ge.pathExtractor.GetLines()
}

// GetRectangles returns all extracted rectangles
func (ge *GraphicsExtractor) GetRectangles() []ExtractedRectangle {
	return ge.pathExtractor.GetRectangles()
}

// GetHorizontalLines returns only horizontal lines
func (ge *GraphicsExtractor) GetHorizontalLines() []ExtractedLine {
	return ge.pathExtractor.GetHorizontalLines()
}

// GetVerticalLines returns only vertical lines
func (ge *GraphicsExtractor) GetVerticalLines() []ExtractedLine {
	return ge.pathExtractor.GetVerticalLines()
}

// GetFilteredLines returns lines meeting the minimum length requirement
func (ge *GraphicsExtractor) GetFilteredLines() []ExtractedLine {
	return ge.pathExtractor.FilterLinesByLength(ge.MinLineLength)
}

// GetFilteredRectangles returns rectangles meeting the minimum size requirements
func (ge *GraphicsExtractor) GetFilteredRectangles() []ExtractedRectangle {
	return ge.pathExtractor.FilterRectanglesBySize(ge.MinRectWidth, ge.MinRectHeight)
}

// Clear resets the extractor for reuse
func (ge *GraphicsExtractor) Clear() {
	ge.gs = NewGraphicsState()
	ge.pathExtractor = NewPathExtractor(ge.gs)
}

// ToModelLines converts extracted lines to model.Line objects
func (ge *GraphicsExtractor) ToModelLines() []model.Line {
	extractedLines := ge.GetFilteredLines()
	result := make([]model.Line, len(extractedLines))

	for i, el := range extractedLines {
		result[i] = model.Line{
			Start: el.Start,
			End:   el.End,
			Width: el.Width,
			Color: model.Color{
				R: floatToUint8(el.Color[0]),
				G: floatToUint8(el.Color[1]),
				B: floatToUint8(el.Color[2]),
			},
			IsRect:   false,
			RectFill: false,
		}
	}

	return result
}

// ToModelRectangles converts extracted rectangles to model.Line objects (with IsRect=true)
func (ge *GraphicsExtractor) ToModelRectangles() []model.Line {
	extractedRects := ge.GetFilteredRectangles()
	result := make([]model.Line, len(extractedRects))

	for i, er := range extractedRects {
		// Convert rectangle to Line with IsRect=true
		// Use top-left to bottom-right diagonal
		var color model.Color
		if er.IsFilled {
			color = model.Color{
				R: floatToUint8(er.FillColor[0]),
				G: floatToUint8(er.FillColor[1]),
				B: floatToUint8(er.FillColor[2]),
			}
		} else if er.IsStroked {
			color = model.Color{
				R: floatToUint8(er.StrokeColor[0]),
				G: floatToUint8(er.StrokeColor[1]),
				B: floatToUint8(er.StrokeColor[2]),
			}
		}

		result[i] = model.Line{
			Start: model.Point{
				X: er.BBox.X,
				Y: er.BBox.Y,
			},
			End: model.Point{
				X: er.BBox.X + er.BBox.Width,
				Y: er.BBox.Y + er.BBox.Height,
			},
			Width:    er.StrokeWidth,
			Color:    color,
			IsRect:   true,
			RectFill: er.IsFilled,
		}
	}

	return result
}

// GetGraphicsState returns the current graphics state (useful for debugging)
func (ge *GraphicsExtractor) GetGraphicsState() *GraphicsState {
	return ge.gs
}

// Helper functions

func toFloat(obj core.Object) (float64, bool) {
	switch v := obj.(type) {
	case core.Int:
		return float64(v), true
	case core.Real:
		return float64(v), true
	default:
		return 0, false
	}
}

func operandsToMatrix(operands []core.Object) model.Matrix {
	if len(operands) != 6 {
		return model.Identity()
	}

	vals := make([]float64, 6)
	for i, op := range operands {
		vals[i], _ = toFloat(op)
	}

	return model.Matrix(vals)
}

// cmykToRGB converts CMYK to RGB (approximate conversion)
func cmykToRGB(c, m, y, k float64) (r, g, b float64) {
	r = (1 - c) * (1 - k)
	g = (1 - m) * (1 - k)
	b = (1 - y) * (1 - k)
	return
}

// floatToUint8 converts a float64 color value (0.0-1.0) to uint8 (0-255)
func floatToUint8(f float64) uint8 {
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	return uint8(f * 255)
}

// LineClassification provides classification of extracted lines
type LineClassification struct {
	HorizontalLines []ExtractedLine
	VerticalLines   []ExtractedLine
	DiagonalLines   []ExtractedLine
}

// ClassifyLines classifies lines by orientation
func (ge *GraphicsExtractor) ClassifyLines() LineClassification {
	result := LineClassification{
		HorizontalLines: make([]ExtractedLine, 0),
		VerticalLines:   make([]ExtractedLine, 0),
		DiagonalLines:   make([]ExtractedLine, 0),
	}

	for _, line := range ge.GetFilteredLines() {
		if line.IsHorizontal {
			result.HorizontalLines = append(result.HorizontalLines, line)
		} else if line.IsVertical {
			result.VerticalLines = append(result.VerticalLines, line)
		} else {
			result.DiagonalLines = append(result.DiagonalLines, line)
		}
	}

	return result
}

// GridLines represents horizontal and vertical lines that could form a table grid
type GridLines struct {
	Horizontals []ExtractedLine
	Verticals   []ExtractedLine
}

// GetGridLines returns horizontal and vertical lines suitable for table detection
func (ge *GraphicsExtractor) GetGridLines() GridLines {
	classification := ge.ClassifyLines()
	return GridLines{
		Horizontals: classification.HorizontalLines,
		Verticals:   classification.VerticalLines,
	}
}

// Statistics provides statistics about extracted graphics
type GraphicsStatistics struct {
	TotalLines        int
	HorizontalLines   int
	VerticalLines     int
	DiagonalLines     int
	TotalRectangles   int
	FilledRectangles  int
	StrokedRectangles int
}

// GetStatistics returns statistics about extracted graphics
func (ge *GraphicsExtractor) GetStatistics() GraphicsStatistics {
	classification := ge.ClassifyLines()
	rects := ge.GetFilteredRectangles()

	stats := GraphicsStatistics{
		TotalLines:      len(ge.GetFilteredLines()),
		HorizontalLines: len(classification.HorizontalLines),
		VerticalLines:   len(classification.VerticalLines),
		DiagonalLines:   len(classification.DiagonalLines),
		TotalRectangles: len(rects),
	}

	for _, r := range rects {
		if r.IsFilled {
			stats.FilledRectangles++
		}
		if r.IsStroked {
			stats.StrokedRectangles++
		}
	}

	return stats
}
