package tabula

import (
	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/rag"
	"github.com/tsawler/tabula/text"
	"strings"
	"testing"
	"unicode"
)

func TestPDFSpatialMathPreservesFractionBaselines(t *testing.T) {
	fs := []text.TextFragment{
		{Text: "J", X: 20, Y: 30, FontSize: 10},
		{Text: "τ =", X: 0, Y: 25, FontSize: 10},
		{Text: "∆t", X: 20, Y: 20, FontSize: 10},
	}
	got := pdfSpatialText(fs)
	strip := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, s)
	}
	if strip(got) != "Jτ=∆t" {
		t.Fatalf("glyphs lost: %q", got)
	}
	if !strings.Contains(got, "\n") || strings.Index(got, "J") > strings.Index(got, "∆t") {
		t.Fatalf("fraction flattened: %q", got)
	}
	doc := &model.Document{Pages: []*model.Page{{Elements: []model.Element{&model.Paragraph{Text: got, Preformatted: true}}}}}
	md := pdfMarkdown(doc, rag.MarkdownOptions{})
	if !strings.Contains(md, "```text\n"+got+"\n```") {
		t.Fatalf("spatial layout lost: %q", md)
	}
}

func TestPDFElementsDoNotDuplicateHeading(t *testing.T) {
	fs := []text.TextFragment{
		{Text: "SECTION A", BaseFont: "Calibri-Bold", X: 20, Y: 150, Width: 90, Height: 16, FontSize: 16},
		{Text: "The contractor maintains the grounds.", BaseFont: "Calibri", X: 20, Y: 110, Width: 180, Height: 10, FontSize: 10},
	}
	elements := pdfElements(fs, 300, 200, nil)
	got := ""
	for _, e := range elements {
		if v, ok := e.(model.TextElement); ok {
			got += v.GetText() + "\n"
		}
	}
	if strings.Count(got, "SECTION A") != 1 || !strings.Contains(got, "contractor") {
		t.Fatalf("duplicate or lost content: %q", got)
	}
	stats := pdfPageLayout(fs, elements, 300, 200)
	if stats.Stats.HeadingCount != 1 || stats.Stats.ParagraphCount != 1 || stats.Stats.LineCount != 2 {
		t.Fatalf("incomplete stats: %+v", stats.Stats)
	}
}
