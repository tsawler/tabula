package rag

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/model"
)

func TestDetectChapterHeading_Matches(t *testing.T) {
	cases := []struct {
		name     string
		text     string
		wantHead string // prefix the heading should start with
	}{
		{"chapter digit", "CHAPTER 1 Reflections of a Traveling American Fortean Our thoughts often turn to moving.", "CHAPTER 1"},
		{"chapter roman", "CHAPTER IV The Dover Demon. People often ask me about it.", "CHAPTER IV"},
		{"part word", "PART ONE The Early Years", "PART ONE"},
		{"appendix letter", "APPENDIX A Sources and Notes", "APPENDIX A"},
		{"section number", "SECTION 3 Methods used in the field", "SECTION 3"},
		{"leading ocr noise", "| CHAPTER 2 A Couple of Side Trips into the Unknown", "CHAPTER 2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			head, body, ok := detectChapterHeading(tc.text)
			if !ok {
				t.Fatalf("expected match for %q", tc.text)
			}
			if !strings.Contains(head, strings.TrimPrefix(tc.wantHead, "")) {
				t.Errorf("heading %q does not contain marker %q", head, tc.wantHead)
			}
			// Heading + body together must preserve all words of the input.
			joined := strings.Join(strings.Fields(head+" "+body), " ")
			want := strings.Join(strings.Fields(strings.TrimLeft(tc.text, " |\\._*")), " ")
			if joined != want {
				t.Errorf("content not preserved:\n got=%q\nwant=%q", joined, want)
			}
		})
	}
}

func TestDetectChapterHeading_Rejects(t *testing.T) {
	cases := []string{
		"Part of the reason we went there was simple curiosity about the woods.",
		"The chapter house stood at the edge of the old monastery grounds.",
		"Section the log into four pieces before you stack it for winter.",
		"Books were stacked everywhere in the cramped little study room.",
		"This is an ordinary paragraph with no structural marker at all here.",
		"",
	}
	for _, text := range cases {
		if _, _, ok := detectChapterHeading(text); ok {
			t.Errorf("expected NO match for %q", text)
		}
	}
}

func TestDetectChapterHeading_TitleBounded(t *testing.T) {
	// A marker glued onto a long run-on body should yield a bounded heading and
	// a non-empty body (it must not swallow the whole page).
	body := strings.Repeat("and the investigation continued for many days ", 40)
	head, rest, ok := detectChapterHeading("CHAPTER 7 Alligators in the Sewers " + body)
	if !ok {
		t.Fatal("expected match")
	}
	if len(head) > headingTitleMaxChars {
		t.Errorf("heading exceeds cap: %d chars (%q)", len(head), head)
	}
	if strings.TrimSpace(rest) == "" {
		t.Error("expected non-empty body after bounded heading")
	}
}

func TestDetectChapterHeading_StandaloneHeading(t *testing.T) {
	head, body, ok := detectChapterHeading("CHAPTER 5")
	if !ok {
		t.Fatal("expected match")
	}
	if head != "CHAPTER 5" {
		t.Errorf("got heading %q", head)
	}
	if body != "" {
		t.Errorf("expected empty body, got %q", body)
	}
}

func TestChapterHeadingLevel(t *testing.T) {
	if got := chapterHeadingLevel("PART ONE"); got != 1 {
		t.Errorf("PART should be level 1, got %d", got)
	}
	if got := chapterHeadingLevel("BOOK II"); got != 1 {
		t.Errorf("BOOK should be level 1, got %d", got)
	}
	if got := chapterHeadingLevel("CHAPTER 3 Something"); got != 2 {
		t.Errorf("CHAPTER should be level 2, got %d", got)
	}
}

func TestDocumentHasHeadings(t *testing.T) {
	flat := model.NewDocument()
	flat.AddPage(&model.Page{Number: 1, Elements: []model.Element{
		&model.Paragraph{Text: "Just text."},
	}})
	if documentHasHeadings(flat) {
		t.Error("flat document should report no headings")
	}

	structured := model.NewDocument()
	structured.AddPage(&model.Page{Number: 1, Elements: []model.Element{
		&model.Heading{Text: "Intro", Level: 1},
		&model.Paragraph{Text: "Body."},
	}})
	if !documentHasHeadings(structured) {
		t.Error("structured document should report headings")
	}
}

func TestChunkDocument_PromotesHeadings_OnlyWhenFlat(t *testing.T) {
	body := strings.Repeat("word ", 80)

	// Flat doc: chapter marker is glued to the body paragraph.
	flat := model.NewDocument()
	flat.Metadata.Title = "Scanned Book"
	flat.AddPage(&model.Page{Number: 1, Elements: []model.Element{
		&model.Paragraph{Text: "CHAPTER 1 The Beginning " + body},
		&model.Paragraph{Text: body},
		&model.Paragraph{Text: "CHAPTER 2 The Middle " + body},
	}})

	got := NewDocumentChunker().ChunkDocument(flat)
	foundCh1, foundCh2 := false, false
	for _, c := range got.Chunks {
		for _, s := range c.Metadata.SectionPath {
			if strings.Contains(s, "CHAPTER 1") {
				foundCh1 = true
			}
			if strings.Contains(s, "CHAPTER 2") {
				foundCh2 = true
			}
		}
	}
	if !foundCh1 || !foundCh2 {
		t.Errorf("expected promoted chapter section paths (ch1=%v ch2=%v)", foundCh1, foundCh2)
	}

	// Disabling detection should leave the document flat (no section path).
	cfg := DefaultChunkerConfig()
	cfg.DetectHeadings = false
	gotOff := NewDocumentChunkerWithConfig(cfg, DefaultSizeConfig()).ChunkDocument(flat)
	for _, c := range gotOff.Chunks {
		for _, s := range c.Metadata.SectionPath {
			if strings.Contains(s, "CHAPTER") {
				t.Errorf("detection disabled but section path has heading: %q", s)
			}
		}
	}
}
