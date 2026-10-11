package tabula

import (
	"github.com/tsawler/tabula/core"
	"github.com/tsawler/tabula/graphicsstate"
	"github.com/tsawler/tabula/layout"
	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/pages"
	"github.com/tsawler/tabula/tables"
	"github.com/tsawler/tabula/text"
	"math"
	"regexp"
	"sort"
	"strings"
)

var pdfListPrefix = regexp.MustCompile(`^([•◦▪●\-]|\d+[.)])\s+`)
var pdfSectionPrefix = regexp.MustCompile(`^(\d+(?:\.\d+)*|[IVX]+\.|[A-Z]\.)\s+`)

// Build each structural element once, within its reading-order section. The
// previous pipeline added overlapping headings, paragraphs and lists separately.
func pdfElements(fs []text.TextFragment, w, h float64, detectedTables []*model.Table) []model.Element {
	var remaining []text.TextFragment
	for _, f := range fs {
		inside := false
		for _, t := range detectedTables {
			b := t.BBox
			if f.X >= b.X-1 && f.X < b.X+b.Width+1 && f.Y >= b.Y-1 && f.Y <= b.Y+b.Height+1 {
				inside = true
				break
			}
		}
		if !inside {
			remaining = append(remaining, f)
		}
	}
	ro := layout.NewReadingOrderDetector().Detect(fs, w, h)
	var elements []model.Element
	for _, section := range ro.Sections {
		lines := section.Lines
		if len(detectedTables) > 0 {
			var keep []text.TextFragment
			for _, f := range section.Fragments {
				inside := false
				for _, table := range detectedTables {
					b := table.BBox
					if f.X >= b.X-1 && f.X < b.X+b.Width+1 && f.Y >= b.Y-1 && f.Y <= b.Y+b.Height+1 {
						inside = true
						break
					}
				}
				if !inside {
					keep = append(keep, f)
				}
			}
			lines = layout.NewLineDetector().Detect(keep, w, h).Lines
		}
		// Estimate body size by text volume, not by short headings and plot labels.
		counts := map[int]int{}
		body, best := 10.0, 0

		for _, l := range lines {
			for _, f := range l.Fragments {
				face := strings.ToLower(f.BaseFont)
				if strings.Contains(face, "bold") || strings.Contains(face, "medi") {
					continue
				}
				bucket := int(math.Round(f.FontSize))
				counts[bucket] += len([]rune(f.Text))
				if counts[bucket] > best {
					best = counts[bucket]
					body = float64(bucket)
				}
			}
		}

		var pending []layout.Line
		flush := func() {
			if len(pending) == 0 {
				return
			}
			pl := layout.NewParagraphDetector().Detect(pending, w, h)
			for _, p := range pl.Paragraphs {
				elements = append(elements, &model.Paragraph{Text: strings.TrimSpace(p.Text), BBox: p.BBox, FontSize: p.AverageFontSize})
			}
			pending = nil
		}
		for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
			line := lines[lineIndex]
			if pdfMathLine(line) {
				end := lineIndex + 1
				for end < len(lines) && pdfMathLine(lines[end]) && lines[end-1].BBox.Y-lines[end].BBox.Y < body*2 {
					end++
				}
				var mathFragments []text.TextFragment
				hasEquality := false
				for _, ml := range lines[lineIndex:end] {
					mathFragments = append(mathFragments, ml.Fragments...)
					hasEquality = hasEquality || strings.ContainsAny(ml.Text, "=≤≥")
				}
				if hasEquality {
					flush()
					elements = append(elements, &model.Paragraph{Text: pdfSpatialText(mathFragments), BBox: line.BBox, Preformatted: true})
					lineIndex = end - 1
					continue
				}
			}
			value := strings.TrimSpace(line.Text)
			if value == "" {
				continue
			}
			boldChars, total := 0, 0
			for _, f := range line.Fragments {
				n := len([]rune(strings.TrimSpace(f.Text)))
				total += n
				face := strings.ToLower(f.BaseFont)
				if strings.Contains(face, "bold") || strings.Contains(face, "medi") {
					boldChars += n
				}
			}
			bold := total > 0 && boldChars*5 >= total*4
			short := len([]rune(value)) <= 120
			if strings.Contains(value, "=") || strings.Contains(value, "τ") || (len(value) > 0 && !regexp.MustCompile(`[A-Za-z]{2}`).MatchString(value)) {
				short = false
			}
			heading := short && (line.AverageFontSize > body*1.15 || (bold && len(strings.Fields(value)) <= 12 && !strings.HasSuffix(value, ".") && line.AverageFontSize >= body*0.95) || (pdfSectionPrefix.MatchString(value) && len(strings.Fields(value)) <= 9 && !strings.HasSuffix(value, ".")) || (strings.ToUpper(value) == value && len(value) > 4 && len(value) < 90))
			if heading {
				flush()
				level := 2
				if line.AverageFontSize > body*1.6 {
					level = 1
				}
				prefix := pdfSectionPrefix.FindString(value)
				if strings.Count(prefix, ".") >= 1 {
					level = 3
				}
				elements = append(elements, &model.Heading{Text: value, Level: level, BBox: line.BBox})
				continue
			}
			if prefix := pdfListPrefix.FindString(value); prefix != "" {
				flush()
				elements = append(elements, &model.List{Items: []model.ListItem{{Text: strings.TrimSpace(strings.TrimPrefix(value, prefix)), Bullet: strings.TrimSpace(prefix)}}, Ordered: prefix[0] >= '0' && prefix[0] <= '9', BBox: line.BBox})
				continue
			}
			if len(elements) > 0 && len(pending) == 0 {
				if list, ok := elements[len(elements)-1].(*model.List); ok {
					gap := list.BBox.Y - (line.BBox.Y + line.BBox.Height)
					if gap < line.Height*0.8 && gap >= -1 && line.BBox.X > list.BBox.X+4 {
						list.Items[len(list.Items)-1].Text += " " + value
						list.BBox.Height += list.BBox.Y - line.BBox.Y
						list.BBox.Y = line.BBox.Y
						continue
					}
				}
			}
			pending = append(pending, line)
		}
		flush()
	}
	// Insert tables by position in their containing column without re-sorting text
	// across columns. Tables consume their fragments, so their text is never repeated.
	for _, table := range detectedTables {
		at := len(elements)
		for i, e := range elements {
			b := e.BoundingBox()
			if b.Y < table.BBox.Y+table.BBox.Height {
				at = i
				break
			}
		}
		elements = append(elements, nil)
		copy(elements[at+1:], elements[at:])
		elements[at] = table
	}
	for _, f := range remaining {
		if f.Vertical {
			elements = append(elements, &model.Paragraph{Text: f.Text, BBox: model.NewBBox(f.X, f.Y, f.Width, f.Height)})
		}
	}
	return elements
}

func (e *Extractor) pdfTables(page *pages.Page, fs []text.TextFragment) []*model.Table {
	contents, err := page.Contents()
	if err != nil {
		return nil
	}
	ge := graphicsstate.NewGraphicsExtractor()
	ge.MinRectWidth = 0.1
	ge.MinRectHeight = 0.1
	for _, obj := range contents {
		if stream, ok := obj.(*core.Stream); ok {
			data, err := stream.Decode()
			if err == nil {
				ge.ExtractFromBytes(data)
			}
		}
	}
	var result []*model.Table
	axes := ge.GetGridLines()
	for _, r := range ge.GetFilteredRectangles() {
		b := r.BBox
		if b.Height < 2 && b.Width > 20 {
			axes.Horizontals = append(axes.Horizontals, graphicsstate.ExtractedLine{Start: model.Point{X: b.X, Y: b.Y}, End: model.Point{X: b.X + b.Width, Y: b.Y}})
		}
		if b.Width < 2 && b.Height > 10 {
			axes.Verticals = append(axes.Verticals, graphicsstate.ExtractedLine{Start: model.Point{X: b.X, Y: b.Y}, End: model.Point{X: b.X, Y: b.Y + b.Height}})
		}
	}
	for _, grid := range pdfGridGroups(axes.Horizontals, axes.Verticals) {
		xs, ys := grid.VerticalLines, grid.HorizontalLines
		if len(xs) < 3 || len(ys) < 3 {
			continue
		}
		sort.Float64s(xs)
		sort.Sort(sort.Reverse(sort.Float64Slice(ys)))
		table := model.NewTable(len(ys)-1, len(xs)-1)
		table.BBox = grid.BBox
		used, words := 0, 0
		for row := range table.Rows {
			for col := range table.Rows[row] {
				var cell []text.TextFragment
				for _, f := range fs {
					if f.X >= xs[col]-1 && f.X < xs[col+1]-1 && f.Y <= ys[row]+1 && f.Y > ys[row+1] {
						cell = append(cell, f)
					}
				}
				ll := layout.NewLineDetector().Detect(cell, xs[col+1]-xs[col], ys[row]-ys[row+1])
				var parts []string
				for _, l := range ll.Lines {
					parts = append(parts, strings.TrimSpace(l.Text))
				}
				value := strings.Join(parts, " ")
				table.Rows[row][col].Text = value
				if value != "" {
					used++
				}
				if len(strings.Fields(value)) >= 2 {
					words++
				}
			}
		}
		// Axis labels around charts do not form a populated text grid.
		if words >= 2 && used >= len(table.Rows) {
			result = append(result, table)
		}
	}
	return result
}

// Separate physically disconnected grids before asking the grid detector for
// row/column coordinates. Decorative rules and neighbouring tables are excluded.
func pdfGridGroups(hs, vs []graphicsstate.ExtractedLine) []*tables.GridHypothesis {
	all := append(append([]graphicsstate.ExtractedLine(nil), hs...), vs...)
	parent := make([]int, len(all))
	for i := range parent {
		parent[i] = i
	}
	var root func(int) int
	root = func(i int) int {
		if parent[i] != i {
			parent[i] = root(parent[i])
		}
		return parent[i]
	}
	for i, a := range all {
		for j := 0; j < i; j++ {
			b := all[j]
			ax0, ax1 := math.Min(a.Start.X, a.End.X), math.Max(a.Start.X, a.End.X)
			ay0, ay1 := math.Min(a.Start.Y, a.End.Y), math.Max(a.Start.Y, a.End.Y)
			bx0, bx1 := math.Min(b.Start.X, b.End.X), math.Max(b.Start.X, b.End.X)
			by0, by1 := math.Min(b.Start.Y, b.End.Y), math.Max(b.Start.Y, b.End.Y)
			if ax0 <= bx1+2 && bx0 <= ax1+2 && ay0 <= by1+2 && by0 <= ay1+2 {
				parent[root(i)] = root(j)
			}
		}
	}
	groups := map[int][]int{}
	for i := range all {
		r := root(i)
		groups[r] = append(groups[r], i)
	}
	var grids []*tables.GridHypothesis
	for _, indices := range groups {
		var h, v []graphicsstate.ExtractedLine
		for _, i := range indices {
			if i < len(hs) {
				h = append(h, all[i])
			} else {
				v = append(v, all[i])
			}
		}
		grids = append(grids, tables.NewGridDetector().DetectFromLines(h, v)...)
	}
	sort.Slice(grids, func(i, j int) bool { return grids[i].BBox.Y > grids[j].BBox.Y })
	return grids
}

func pdfPageLayout(fs []text.TextFragment, elements []model.Element, w, h float64) *model.PageLayout {
	result := &model.PageLayout{Stats: model.LayoutStats{FragmentCount: len(fs)}}
	ro := layout.NewReadingOrderDetector().Detect(fs, w, h)
	for i, section := range ro.Sections {
		result.Columns = append(result.Columns, model.ColumnInfo{Index: i, BBox: section.BBox, Left: section.BBox.X, Right: section.BBox.X + section.BBox.Width, Width: section.BBox.Width})
	}
	result.ColumnCount = ro.ColumnCount
	for _, line := range ro.Lines {
		result.Lines = append(result.Lines, model.LineInfo{Index: len(result.Lines), Text: line.Text, BBox: line.BBox, FontSize: line.AverageFontSize})
	}
	result.Stats.LineCount = len(result.Lines)
	for i, element := range elements {
		result.ReadingOrder = append(result.ReadingOrder, i)
		switch v := element.(type) {
		case *model.Paragraph:
			result.Paragraphs = append(result.Paragraphs, model.ParagraphInfo{Index: len(result.Paragraphs), Text: v.Text, BBox: v.BBox, FontSize: v.FontSize})
		case *model.Heading:
			result.Headings = append(result.Headings, model.HeadingInfo{Text: v.Text, BBox: v.BBox, Level: v.Level})
		case *model.List:
			kind := model.ListTypeBullet
			if v.Ordered {
				kind = model.ListTypeNumbered
			}
			result.Lists = append(result.Lists, model.ListInfo{Type: kind, Items: v.Items, BBox: v.BBox})
		}
	}
	result.Stats.ParagraphCount = len(result.Paragraphs)
	result.Stats.HeadingCount = len(result.Headings)
	result.Stats.ListCount = len(result.Lists)
	return result
}

var pdfProseWord = regexp.MustCompile(`[A-Za-z]{3,}`)

func pdfMathLine(line layout.Line) bool {
	mathChars, total := 0, 0
	for _, f := range line.Fragments {
		n := len([]rune(strings.TrimSpace(f.Text)))
		total += n
		face := strings.ToLower(f.BaseFont)
		if strings.Contains(face, "cmmi") || strings.Contains(face, "cmsy") || strings.Contains(face, "cmex") {
			mathChars += n
		}
	}
	return total > 0 && ((mathChars*3 >= total && len(pdfProseWord.FindAllString(line.Text, -1)) <= 1) || (len(line.Text) < 20 && strings.Trim(line.Text, "0123456789()[]{} .") == ""))
}

// Preserve baseline and horizontal position without inventing LaTeX semantics.
func pdfSpatialText(fs []text.TextFragment) string {
	if len(fs) == 0 {
		return ""
	}
	fs = append([]text.TextFragment(nil), fs...)
	sort.Slice(fs, func(i, j int) bool {
		if math.Abs(fs[i].Y-fs[j].Y) > 1 {
			return fs[i].Y > fs[j].Y
		}
		return fs[i].X < fs[j].X
	})
	left, top, size := fs[0].X, fs[0].Y, 0.0
	for _, f := range fs {
		left = math.Min(left, f.X)
		top = math.Max(top, f.Y)
		size = math.Max(size, f.FontSize)
	}
	cell, rowHeight := math.Max(size*0.25, 1), math.Max(size*0.2, 1)
	rows := map[int][]rune{}
	last := 0
	for _, f := range fs {
		row := int(math.Round((top - f.Y) / rowHeight))
		col := int(math.Round((f.X - left) / cell))
		value := []rune(f.Text)
		if strings.TrimSpace(f.Text) == "" {
			continue
		}
		buf := rows[row]
		for len(buf) < col+len(value) {
			buf = append(buf, ' ')
		}
		// Glyph widths differ; avoid overwriting text that occupies the same cell.
		for {
			occupied := false
			for k := 0; k < len(value) && col+k < len(buf); k++ {
				if value[k] != ' ' && buf[col+k] != ' ' {
					occupied = true
					break
				}
			}
			if !occupied {
				break
			}
			col++
		}
		for len(buf) < col+len(value) {
			buf = append(buf, ' ')
		}
		for k, r := range value {
			if r != ' ' {
				buf[col+k] = r
			}
		}
		rows[row] = buf
		if row > last {
			last = row
		}
	}
	var lines []string
	for row := 0; row <= last; row++ {
		lines = append(lines, strings.TrimRight(string(rows[row]), " "))
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}
