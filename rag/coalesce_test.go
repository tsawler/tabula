package rag

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/model"
)

// makeChunk is a small helper to build a chunk with the fields the coalescing
// pass cares about.
func makeChunk(text string, sectionPath []string, elementTypes ...string) *Chunk {
	if len(elementTypes) == 0 {
		elementTypes = []string{"paragraph"}
	}
	sectionTitle := ""
	if len(sectionPath) > 0 {
		sectionTitle = sectionPath[len(sectionPath)-1]
	}
	return &Chunk{
		Text: text,
		Metadata: ChunkMetadata{
			SectionPath:  sectionPath,
			SectionTitle: sectionTitle,
			ElementTypes: elementTypes,
			CharCount:    len(text),
			WordCount:    countWords(text),
			PageStart:    1,
			PageEnd:      1,
		},
	}
}

func newTestChunker() *DocumentChunker {
	return NewDocumentChunker() // DefaultSizeConfig: Min 100, Max 2000 chars, MergeSmallChunks true
}

func TestCoalesce_MergesHeadingForwardIntoBody(t *testing.T) {
	dc := newTestChunker()
	body := "This is a reasonably sized paragraph of body content that comfortably exceeds the minimum chunk size threshold so it is not itself merged away by the undersized pass."

	chunks := []*Chunk{
		makeChunk("Chapter 1: The Beginning", []string{"Chapter 1: The Beginning"}, "heading"),
		makeChunk(body, []string{"Chapter 1: The Beginning"}, "paragraph"),
	}

	got := dc.coalesceSmallChunks(chunks)

	if len(got) != 1 {
		t.Fatalf("expected heading to merge into body (1 chunk), got %d", len(got))
	}
	if !strings.HasPrefix(got[0].Text, "Chapter 1: The Beginning") {
		t.Errorf("expected heading text prepended, got: %q", got[0].Text[:30])
	}
	if !strings.Contains(got[0].Text, "body content") {
		t.Errorf("expected body retained in merged chunk")
	}
	if !containsStr(got[0].Metadata.ElementTypes, "heading") || !containsStr(got[0].Metadata.ElementTypes, "paragraph") {
		t.Errorf("expected element types to union heading+paragraph, got %v", got[0].Metadata.ElementTypes)
	}
	if got[0].Metadata.CharCount != len(got[0].Text) {
		t.Errorf("CharCount not recomputed: %d vs %d", got[0].Metadata.CharCount, len(got[0].Text))
	}
}

func TestCoalesce_AccumulatesConsecutiveHeadings(t *testing.T) {
	dc := newTestChunker()
	body := strings.Repeat("word ", 60) // ~300 chars, above min

	chunks := []*Chunk{
		makeChunk("Part One", []string{"Part One"}, "heading"),
		makeChunk("Chapter 1", []string{"Part One", "Chapter 1"}, "heading"),
		makeChunk(body, []string{"Part One", "Chapter 1"}, "paragraph"),
	}

	got := dc.coalesceSmallChunks(chunks)
	if len(got) != 1 {
		t.Fatalf("expected both headings merged into body (1 chunk), got %d", len(got))
	}
	if !strings.Contains(got[0].Text, "Part One") || !strings.Contains(got[0].Text, "Chapter 1") {
		t.Errorf("expected both headings present, got: %q", got[0].Text[:40])
	}
}

func TestCoalesce_MergesUndersizedBodyIntoPrevSameSection(t *testing.T) {
	dc := newTestChunker()
	big := strings.Repeat("alpha ", 40) // ~240 chars
	tiny := "A stray fragment."          // < 100 chars

	chunks := []*Chunk{
		makeChunk(big, []string{"S1"}, "paragraph"),
		makeChunk(tiny, []string{"S1"}, "paragraph"),
	}

	got := dc.coalesceSmallChunks(chunks)
	if len(got) != 1 {
		t.Fatalf("expected tiny fragment merged into previous, got %d chunks", len(got))
	}
	if !strings.Contains(got[0].Text, "stray fragment") {
		t.Errorf("fragment text lost after merge")
	}
}

func TestCoalesce_MergesUndersizedIntoNextWhenNoPrev(t *testing.T) {
	dc := newTestChunker()
	tiny := "Tiny lead."
	big := strings.Repeat("beta ", 40)

	chunks := []*Chunk{
		makeChunk(tiny, []string{"S1"}, "paragraph"),
		makeChunk(big, []string{"S1"}, "paragraph"),
	}

	got := dc.coalesceSmallChunks(chunks)
	if len(got) != 1 {
		t.Fatalf("expected tiny lead merged into next, got %d chunks", len(got))
	}
	if !strings.HasPrefix(got[0].Text, "Tiny lead.") {
		t.Errorf("expected tiny text prepended to next chunk, got: %q", got[0].Text[:20])
	}
}

func TestCoalesce_RespectsMaxChunkSize(t *testing.T) {
	dc := newTestChunker()
	// big is 1999 chars; the trailing fragment cannot merge into it without
	// exceeding the 2000-char max (1999 + 2 + 1 = 2002), and there is no next
	// chunk to absorb it, so it must be left as its own chunk.
	big := strings.Repeat("c", 1999)
	frag := "x"

	chunks := []*Chunk{
		makeChunk(big, []string{"S1"}, "paragraph"),
		makeChunk(frag, []string{"S1"}, "paragraph"),
	}

	got := dc.coalesceSmallChunks(chunks)
	for _, c := range got {
		if len(c.Text) > 2000 {
			t.Errorf("merge produced chunk over max: %d chars", len(c.Text))
		}
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks (frag unmergeable), got %d", len(got))
	}
}

func TestCoalesce_TrailingHeadingKept(t *testing.T) {
	dc := newTestChunker()
	body := strings.Repeat("gamma ", 40)

	chunks := []*Chunk{
		makeChunk(body, []string{"S1"}, "paragraph"),
		makeChunk("Appendix", []string{"Appendix"}, "heading"),
	}

	got := dc.coalesceSmallChunks(chunks)
	// Trailing heading has no following content; it is undersized so the
	// undersized pass folds it back into the previous body chunk.
	if len(got) != 1 {
		t.Fatalf("expected trailing heading folded back, got %d chunks", len(got))
	}
	if !strings.Contains(got[0].Text, "Appendix") {
		t.Errorf("trailing heading text lost")
	}
}

func TestCoalesce_Disabled(t *testing.T) {
	cfg := DefaultSizeConfig()
	cfg.MergeSmallChunks = false
	dc := NewDocumentChunkerWithConfig(DefaultChunkerConfig(), cfg)

	doc := model.NewDocument()
	doc.Metadata.Title = "T"
	doc.AddPage(&model.Page{
		Number: 1,
		Elements: []model.Element{
			&model.Heading{Text: "H", Level: 1},
			&model.Paragraph{Text: strings.Repeat("word ", 60)},
		},
	})

	got := dc.ChunkDocument(doc)
	// With merging off, the heading remains its own chunk.
	if got.Count() < 2 {
		t.Fatalf("expected heading kept separate when merging disabled, got %d chunks", got.Count())
	}
}

func TestCoalesce_ReindexesChunks(t *testing.T) {
	dc := newTestChunker()
	body := strings.Repeat("delta ", 40)

	chunks := []*Chunk{
		makeChunk("Heading A", []string{"Heading A"}, "heading"),
		makeChunk(body, []string{"Heading A"}, "paragraph"),
		makeChunk(body, []string{"Heading A"}, "paragraph"),
	}

	got := dc.coalesceSmallChunks(chunks)
	for i, c := range got {
		if c.Metadata.ChunkIndex != i {
			t.Errorf("chunk %d has ChunkIndex %d", i, c.Metadata.ChunkIndex)
		}
	}
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
